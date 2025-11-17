package classification

import (
	"sync"
	"time"
)

// StatsCache provides in-memory caching for classification statistics
// This improves performance by reducing load on ClassificationService and Prometheus
type StatsCache struct {
	mu        sync.RWMutex
	cached    *StatsResponse
	expiresAt time.Time
	ttl       time.Duration
}

// NewStatsCache creates a new stats cache with the specified TTL
func NewStatsCache(ttl time.Duration) *StatsCache {
	if ttl <= 0 {
		ttl = 5 * time.Second // Default TTL: 5 seconds
	}
	return &StatsCache{
		ttl: ttl,
	}
}

// Get retrieves cached stats if available and not expired
// Returns (stats, true) if cache hit, (nil, false) if cache miss
func (c *StatsCache) Get() (*StatsResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.cached == nil {
		return nil, false
	}

	if time.Now().After(c.expiresAt) {
		return nil, false // Cache expired
	}

	return c.cached, true
}

// Set stores stats in cache with expiration time
func (c *StatsCache) Set(stats *StatsResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cached = stats
	c.expiresAt = time.Now().Add(c.ttl)
}

// Invalidate clears the cache
func (c *StatsCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cached = nil
	c.expiresAt = time.Time{}
}

// IsExpired checks if cache is expired (without locking, for internal use)
func (c *StatsCache) IsExpired() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.cached == nil || time.Now().After(c.expiresAt)
}

// GetTTL returns the cache TTL
func (c *StatsCache) GetTTL() time.Duration {
	return c.ttl
}

// SetTTL updates the cache TTL
func (c *StatsCache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ttl = ttl
	if c.cached != nil {
		// Update expiration time based on new TTL
		c.expiresAt = time.Now().Add(c.ttl)
	}
}
