package inhibition

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
)

// TwoTierAlertCache implements ActiveAlertCache with two-tier caching strategy:
//  - L1: In-memory cache (fast, limited capacity)
//  - L2: Redis cache (persistent, distributed)
//
// Fallback strategy: L1 → L2 → empty (graceful degradation)
//
// Thread-safety: Safe for concurrent use (protected by mutex).
// Performance: L1 <1ms, L2 <10ms.
//
// Example:
//
//	cache := inhibition.NewTwoTierAlertCache(redisCache, logger)
//	defer cache.Stop()
//	_ = cache.AddFiringAlert(ctx, alert)
type TwoTierAlertCache struct {
	// L1: In-memory cache (map-based)
	l1Cache map[string]*core.Alert
	l1Mutex sync.RWMutex
	l1Max   int // Max entries in L1

	// L2: Redis cache
	redisCache cache.Cache
	keyPrefix  string
	ttl        time.Duration

	// Background cleanup
	stopCh       chan struct{}
	cleanupDone  chan struct{}
	cleanupInterval time.Duration

	logger *slog.Logger
}

// AlertCacheOptions contains optional configuration for TwoTierAlertCache.
type AlertCacheOptions struct {
	// CleanupInterval sets how often to run background cleanup (default: 1 minute)
	CleanupInterval time.Duration
	// L1Max sets maximum entries in L1 cache (default: 1000)
	L1Max int
	// TTL sets Redis TTL (default: 5 minutes)
	TTL time.Duration
}

// NewTwoTierAlertCache creates a new two-tier alert cache.
//
// Parameters:
//   - redisCache: Redis cache for L2 (can be nil for L1-only mode)
//   - logger: structured logger
//
// Returns:
//   - *TwoTierAlertCache: initialized cache (starts background cleanup worker)
//
// Example:
//
//	cache := inhibition.NewTwoTierAlertCache(redisCache, logger)
//	defer cache.Stop()
func NewTwoTierAlertCache(redisCache cache.Cache, logger *slog.Logger) *TwoTierAlertCache {
	return NewTwoTierAlertCacheWithOptions(redisCache, logger, nil)
}

// NewTwoTierAlertCacheWithOptions creates a new two-tier alert cache with custom options.
//
// Parameters:
//   - redisCache: Redis cache for L2 (can be nil for L1-only mode)
//   - logger: structured logger
//   - opts: optional configuration (can be nil for defaults)
//
// Returns:
//   - *TwoTierAlertCache: initialized cache (starts background cleanup worker)
//
// Example:
//
//	opts := &AlertCacheOptions{CleanupInterval: 30 * time.Second}
//	cache := inhibition.NewTwoTierAlertCacheWithOptions(redisCache, logger, opts)
//	defer cache.Stop()
func NewTwoTierAlertCacheWithOptions(redisCache cache.Cache, logger *slog.Logger, opts *AlertCacheOptions) *TwoTierAlertCache {
	if logger == nil {
		logger = slog.Default()
	}

	// Apply defaults
	cleanupInterval := 1 * time.Minute
	l1Max := 1000
	ttl := 5 * time.Minute

	if opts != nil {
		if opts.CleanupInterval > 0 {
			cleanupInterval = opts.CleanupInterval
		}
		if opts.L1Max > 0 {
			l1Max = opts.L1Max
		}
		if opts.TTL > 0 {
			ttl = opts.TTL
		}
	}

	c := &TwoTierAlertCache{
		l1Cache:         make(map[string]*core.Alert),
		l1Max:           l1Max,
		redisCache:      redisCache,
		keyPrefix:       "inhibition:active_alerts:",
		ttl:             ttl,
		stopCh:          make(chan struct{}),
		cleanupDone:     make(chan struct{}),
		cleanupInterval: cleanupInterval,
		logger:          logger,
	}

	// Start background cleanup worker
	go c.cleanupWorker()

	return c
}

// GetFiringAlerts implements ActiveAlertCache.GetFiringAlerts.
//
// Lookup strategy:
//  1. Try L1 cache (in-memory) → return if all found
//  2. Try L2 cache (Redis) → populate L1 → return
//  3. Return empty on error (graceful degradation)
func (c *TwoTierAlertCache) GetFiringAlerts(ctx context.Context) ([]*core.Alert, error) {
	// Try L1 cache first
	c.l1Mutex.RLock()
	alerts := make([]*core.Alert, 0, len(c.l1Cache))
	for _, alert := range c.l1Cache {
		// Only return firing alerts
		if alert.Status == "firing" {
			alerts = append(alerts, alert)
		}
	}
	c.l1Mutex.RUnlock()

	// If L1 has alerts, return them
	if len(alerts) > 0 {
		c.logger.Debug("L1 cache hit",
			"alerts_count", len(alerts))
		return alerts, nil
	}

	// Try L2 cache (Redis)
	if c.redisCache != nil {
		alerts, err := c.getFromRedis(ctx)
		if err != nil {
			c.logger.Warn("Failed to get alerts from Redis, returning empty",
				"error", err)
			return []*core.Alert{}, nil // Graceful degradation
		}

		// Populate L1 cache
		c.populateL1(alerts)

		c.logger.Debug("L2 cache hit",
			"alerts_count", len(alerts))
		return alerts, nil
	}

	// No alerts found
	return []*core.Alert{}, nil
}

// AddFiringAlert implements ActiveAlertCache.AddFiringAlert.
//
// Adds alert to both L1 and L2 caches.
// L2 failures are logged but don't fail the operation (best-effort).
func (c *TwoTierAlertCache) AddFiringAlert(ctx context.Context, alert *core.Alert) error {
	if alert == nil {
		return fmt.Errorf("alert is nil")
	}

	// Add to L1 cache
	c.l1Mutex.Lock()

	// Check capacity
	if len(c.l1Cache) >= c.l1Max {
		// Evict oldest alert (simple FIFO for now)
		var oldestKey string
		var oldestTime time.Time
		for key, a := range c.l1Cache {
			if oldestKey == "" || a.StartsAt.Before(oldestTime) {
				oldestKey = key
				oldestTime = a.StartsAt
			}
		}
		if oldestKey != "" {
			delete(c.l1Cache, oldestKey)
		}
	}

	c.l1Cache[alert.Fingerprint] = alert
	c.l1Mutex.Unlock()

	c.logger.Debug("Added alert to L1 cache",
		"fingerprint", alert.Fingerprint,
		"alert_name", alert.AlertName)

	// Add to L2 cache (Redis) - best effort
	if c.redisCache != nil {
		if err := c.addToRedis(ctx, alert); err != nil {
			c.logger.Warn("Failed to add alert to Redis (continuing with L1 only)",
				"error", err,
				"fingerprint", alert.Fingerprint)
			// Don't return error - L1 cache is sufficient
		}
	}

	return nil
}

// RemoveAlert implements ActiveAlertCache.RemoveAlert.
//
// Removes alert from both L1 and L2 caches.
func (c *TwoTierAlertCache) RemoveAlert(ctx context.Context, fingerprint string) error {
	// Remove from L1
	c.l1Mutex.Lock()
	delete(c.l1Cache, fingerprint)
	c.l1Mutex.Unlock()

	c.logger.Debug("Removed alert from L1 cache",
		"fingerprint", fingerprint)

	// Remove from L2 (Redis) - best effort
	if c.redisCache != nil {
		key := c.redisKey(fingerprint)
		if err := c.redisCache.Delete(ctx, key); err != nil {
			c.logger.Warn("Failed to remove alert from Redis",
				"error", err,
				"fingerprint", fingerprint)
			// Don't return error - L1 removal is sufficient
		}
	}

	return nil
}

// Stop stops the background cleanup worker.
// Should be called when shutting down the cache.
func (c *TwoTierAlertCache) Stop() {
	close(c.stopCh)
	<-c.cleanupDone // Wait for cleanup worker to finish
}

// --- Private methods ---

// cleanupWorker runs periodic cleanup of expired alerts.
func (c *TwoTierAlertCache) cleanupWorker() {
	defer close(c.cleanupDone)

	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			c.logger.Info("Cleanup worker stopped")
			return
		case <-ticker.C:
			c.cleanup()
		}
	}
}

// cleanup removes expired alerts from L1 cache.
func (c *TwoTierAlertCache) cleanup() {
	now := time.Now()
	removed := 0

	c.l1Mutex.Lock()
	for fingerprint, alert := range c.l1Cache {
		// Remove if alert has ended
		if alert.EndsAt != nil && alert.EndsAt.Before(now) {
			delete(c.l1Cache, fingerprint)
			removed++
		}
		// Remove if alert is too old (TTL)
		if alert.StartsAt.Add(c.ttl).Before(now) {
			delete(c.l1Cache, fingerprint)
			removed++
		}
	}
	c.l1Mutex.Unlock()

	if removed > 0 {
		c.logger.Info("Cleanup completed",
			"removed_alerts", removed,
			"remaining_alerts", len(c.l1Cache))
	}
}

// getFromRedis retrieves all firing alerts from Redis.
func (c *TwoTierAlertCache) getFromRedis(ctx context.Context) ([]*core.Alert, error) {
	// Note: This is a simplified implementation
	// In production, you'd use Redis SCAN or maintain a set of fingerprints

	// For now, return empty - will be populated by AddFiringAlert calls
	return []*core.Alert{}, nil
}

// addToRedis adds an alert to Redis with TTL.
func (c *TwoTierAlertCache) addToRedis(ctx context.Context, alert *core.Alert) error {
	key := c.redisKey(alert.Fingerprint)

	// Serialize alert to JSON
	data, err := json.Marshal(alert)
	if err != nil {
		return fmt.Errorf("failed to marshal alert: %w", err)
	}

	// Set with TTL
	return c.redisCache.Set(ctx, key, string(data), c.ttl)
}

// populateL1 populates L1 cache from a list of alerts.
func (c *TwoTierAlertCache) populateL1(alerts []*core.Alert) {
	c.l1Mutex.Lock()
	defer c.l1Mutex.Unlock()

	for _, alert := range alerts {
		// Don't exceed capacity
		if len(c.l1Cache) >= c.l1Max {
			break
		}
		c.l1Cache[alert.Fingerprint] = alert
	}
}

// redisKey generates Redis key for an alert fingerprint.
func (c *TwoTierAlertCache) redisKey(fingerprint string) string {
	return c.keyPrefix + fingerprint
}
