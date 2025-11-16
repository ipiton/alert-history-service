package cache

import (
	"sync"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// L1Cache is an in-memory cache implementation
// TODO: Replace with Ristretto for production (better eviction policies)
type L1Cache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
	maxSize int64
	ttl     time.Duration
}

type cacheEntry struct {
	value      *core.HistoryResponse
	expiresAt  time.Time
	accessTime time.Time
}

// NewL1Cache creates a new L1 cache
func NewL1Cache(maxEntries int64, ttl time.Duration) *L1Cache {
	cache := &L1Cache{
		entries: make(map[string]*cacheEntry),
		maxSize: maxEntries,
		ttl:     ttl,
	}
	
	// Start background cleanup goroutine
	go cache.cleanup()
	
	return cache
}

// Get retrieves a value from cache
func (c *L1Cache) Get(key string) (*core.HistoryResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	entry, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	
	// Check expiration
	if time.Now().After(entry.expiresAt) {
		// Expired, but don't delete here (cleanup goroutine will handle it)
		return nil, false
	}
	
	// Update access time
	entry.accessTime = time.Now()
	
	return entry.value, true
}

// Set stores a value in cache
func (c *L1Cache) Set(key string, value *core.HistoryResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Evict if cache is full (simple LRU: remove oldest accessed entry)
	if int64(len(c.entries)) >= c.maxSize {
		c.evictOldest()
	}
	
	c.entries[key] = &cacheEntry{
		value:      value,
		expiresAt:  time.Now().Add(c.ttl),
		accessTime: time.Now(),
	}
}

// Delete removes a key from cache
func (c *L1Cache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, key)
}

// Clear removes all entries from cache
func (c *L1Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[string]*cacheEntry)
}

// evictOldest removes the oldest accessed entry (simple LRU)
func (c *L1Cache) evictOldest() {
	if len(c.entries) == 0 {
		return
	}
	
	var oldestKey string
	var oldestTime time.Time
	first := true
	
	for key, entry := range c.entries {
		if first || entry.accessTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.accessTime
			first = false
		}
	}
	
	if oldestKey != "" {
		delete(c.entries, oldestKey)
	}
}

// cleanup periodically removes expired entries
func (c *L1Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.entries {
			if now.After(entry.expiresAt) {
				delete(c.entries, key)
			}
		}
		c.mu.Unlock()
	}
}

// Stats returns cache statistics
func (c *L1Cache) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	expired := 0
	now := time.Now()
	for _, entry := range c.entries {
		if now.After(entry.expiresAt) {
			expired++
		}
	}
	
	return map[string]interface{}{
		"entries":      len(c.entries),
		"max_entries":  c.maxSize,
		"expired":      expired,
		"utilization":  float64(len(c.entries)) / float64(c.maxSize) * 100,
	}
}

