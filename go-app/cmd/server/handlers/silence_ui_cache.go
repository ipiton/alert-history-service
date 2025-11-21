// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// TemplateCache provides caching for rendered templates.
type TemplateCache struct {
	cache    map[string]*cachedTemplate
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
	logger   *slog.Logger
	hits     int64
	misses   int64
	evictions int64
}

type cachedTemplate struct {
	content    []byte
	etag       string
	createdAt  time.Time
	accessCount int64
	lastAccess time.Time
}

// NewTemplateCache creates a new template cache.
func NewTemplateCache(maxSize int, ttl time.Duration, logger *slog.Logger) *TemplateCache {
	return &TemplateCache{
		cache:   make(map[string]*cachedTemplate),
		maxSize: maxSize,
		ttl:     ttl,
		logger:  logger,
	}
}

// Get retrieves a cached template or returns nil if not found.
func (tc *TemplateCache) Get(key string) ([]byte, string, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	cached, exists := tc.cache[key]
	if !exists {
		tc.misses++
		return nil, "", false
	}

	// Check TTL
	if time.Since(cached.createdAt) > tc.ttl {
		tc.mu.RUnlock()
		tc.mu.Lock()
		delete(tc.cache, key)
		tc.evictions++
		tc.misses++
		tc.mu.Unlock()
		tc.mu.RLock()
		return nil, "", false
	}

	// Update access stats
	cached.accessCount++
	cached.lastAccess = time.Now()
	tc.hits++

	return cached.content, cached.etag, true
}

// Set stores a rendered template in the cache.
func (tc *TemplateCache) Set(key string, content []byte) {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// Generate ETag from content
	hash := sha256.Sum256(content)
	etag := hex.EncodeToString(hash[:])[:16]

	// Check cache size and evict if needed
	if len(tc.cache) >= tc.maxSize {
		tc.evictLRU()
	}

	tc.cache[key] = &cachedTemplate{
		content:     content,
		etag:        fmt.Sprintf(`"%s"`, etag),
		createdAt:   time.Now(),
		accessCount: 1,
		lastAccess:  time.Now(),
	}

	tc.logger.Debug("Template cached",
		"key", key,
		"size_bytes", len(content),
		"cache_size", len(tc.cache),
	)
}

// evictLRU evicts the least recently used entry.
func (tc *TemplateCache) evictLRU() {
	if len(tc.cache) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time = time.Now()

	for key, cached := range tc.cache {
		if cached.lastAccess.Before(oldestTime) {
			oldestTime = cached.lastAccess
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(tc.cache, oldestKey)
		tc.evictions++
		tc.logger.Debug("Template evicted from cache", "key", oldestKey)
	}
}

// Clear clears all cached templates.
func (tc *TemplateCache) Clear() {
	tc.mu.Lock()
	defer tc.mu.Unlock()

	tc.cache = make(map[string]*cachedTemplate)
	tc.logger.Info("Template cache cleared")
}

// Stats returns cache statistics.
func (tc *TemplateCache) Stats() map[string]interface{} {
	tc.mu.RLock()
	defer tc.mu.RUnlock()

	total := tc.hits + tc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(tc.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"size":       len(tc.cache),
		"max_size":   tc.maxSize,
		"hits":       tc.hits,
		"misses":     tc.misses,
		"hit_rate":   hitRate,
		"evictions":  tc.evictions,
		"ttl_seconds": int(tc.ttl.Seconds()),
	}
}

// generateCacheKey generates a cache key from template name and data hash.
func (h *SilenceUIHandler) generateCacheKey(templateName string, data interface{}) string {
	// Create a hash from template name and data
	// For simplicity, we'll use a combination of template name and filters
	var keyParts []string
	keyParts = append(keyParts, templateName)

	// Add data-specific parts based on template type
	switch d := data.(type) {
	case DashboardData:
		keyParts = append(keyParts,
			fmt.Sprintf("status:%s", d.Filters.Status),
			fmt.Sprintf("page:%d", d.Page),
			fmt.Sprintf("limit:%d", d.Filters.Limit),
		)
	case DetailViewData:
		keyParts = append(keyParts, fmt.Sprintf("id:%s", d.Silence.ID))
	}

	// Create hash from key parts
	key := fmt.Sprintf("%v", keyParts)
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])[:16]
}

// renderWithCache renders a template with caching support.
func (h *SilenceUIHandler) renderWithCache(
	w http.ResponseWriter,
	r *http.Request,
	templateName string,
	data interface{},
) error {
	// Generate cache key
	cacheKey := h.generateCacheKey(templateName, data)

	// Check cache (only for GET requests)
	if r.Method == http.MethodGet {
		// Initialize cache if not exists
		if h.templateCache == nil {
			h.templateCache = NewTemplateCache(100, 5*time.Minute, h.logger)
		}

		// Check cache
		cached, etag, found := h.templateCache.Get(cacheKey)
		if found {
			// Record cache hit (Phase 14 enhancement)
			if h.metrics != nil {
				h.metrics.RecordTemplateCacheHit()
			}

			// Check If-None-Match header
			ifNoneMatch := r.Header.Get("If-None-Match")
			if ifNoneMatch == etag {
				w.WriteHeader(http.StatusNotModified)
				return nil
			}

			// Set cache headers
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "public, max-age=300") // 5 minutes

			// Write cached content
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, err := w.Write(cached)
			return err
		}

		// Record cache miss (Phase 14 enhancement)
		if h.metrics != nil {
			h.metrics.RecordTemplateCacheMiss()
		}
	}

	// Render template
	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, templateName, data); err != nil {
		return fmt.Errorf("failed to render template %s: %w", templateName, err)
	}

	content := buf.Bytes()

	// Cache rendered content (only for GET requests)
	if r.Method == http.MethodGet && h.templateCache != nil {
		h.templateCache.Set(cacheKey, content)

		// Update cache size metric (Phase 14 enhancement)
		if h.metrics != nil {
			stats := h.templateCache.Stats()
			if size, ok := stats["size"].(int); ok {
				h.metrics.UpdateTemplateCacheSize(size)
			}
		}
	}

	// Set headers
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if h.templateCache != nil {
		_, etag, found := h.templateCache.Get(cacheKey)
		if found {
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "public, max-age=300")
		}
	}

	// Write content
	_, err := w.Write(content)
	return err
}
