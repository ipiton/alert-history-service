package cache

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Manager manages 2-tier caching (L1: in-memory, L2: Redis)
type Manager struct {
	l1Cache      *L1Cache
	l2Cache      *L2Cache
	l1Enabled    bool
	l2Enabled    bool
	logger       *slog.Logger
	metrics      *Metrics
}

// Metrics contains Prometheus metrics for cache operations
type Metrics struct {
	Hits      *prometheus.CounterVec
	Misses    *prometheus.CounterVec
	Evictions *prometheus.CounterVec
	Errors    *prometheus.CounterVec
	Size      *prometheus.GaugeVec
	Latency   *prometheus.HistogramVec
}

// NewMetrics creates new cache metrics
func NewMetrics() *Metrics {
	return &Metrics{
		Hits: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "hits_total",
				Help:      "Total number of cache hits",
			},
			[]string{"cache_layer"},
		),
		Misses: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "misses_total",
				Help:      "Total number of cache misses",
			},
			[]string{"cache_layer"},
		),
		Evictions: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "evictions_total",
				Help:      "Total number of cache evictions",
			},
			[]string{"cache_layer"},
		),
		Errors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "errors_total",
				Help:      "Total number of cache errors",
			},
			[]string{"cache_layer", "error_type"},
		),
		Size: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "size_entries",
				Help:      "Current number of entries in cache",
			},
			[]string{"cache_layer"},
		),
		Latency: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "api_history_cache",
				Name:      "operation_duration_seconds",
				Help:      "Cache operation duration in seconds",
				Buckets:   []float64{.0001, .0005, .001, .005, .01, .025, .05, .1, .25, .5, 1},
			},
			[]string{"cache_layer", "operation", "status"},
		),
	}
}

// NewManager creates a new cache manager
func NewManager(cfg *Config, logger *slog.Logger) (*Manager, error) {
	if logger == nil {
		logger = slog.Default()
	}
	
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	
	manager := &Manager{
		l1Enabled: cfg.L1Enabled,
		l2Enabled: cfg.L2Enabled,
		logger:    logger,
		metrics:   NewMetrics(),
	}
	
	// Initialize L1 cache
	if cfg.L1Enabled {
		manager.l1Cache = NewL1Cache(cfg.L1MaxEntries, cfg.L1TTL)
		logger.Info("L1 cache initialized",
			"max_entries", cfg.L1MaxEntries,
			"max_size_mb", cfg.L1MaxSizeMB,
			"ttl", cfg.L1TTL)
	}
	
	// Initialize L2 cache
	if cfg.L2Enabled {
		l2Cache, err := NewL2Cache(
			cfg.RedisAddr,
			cfg.RedisPassword,
			cfg.RedisDB,
			cfg.RedisPoolSize,
			cfg.RedisMinIdle,
			cfg.L2TTL,
			cfg.L2Compression,
			logger,
		)
		if err != nil {
			logger.Warn("Failed to initialize L2 cache, continuing without it", "error", err)
			manager.l2Enabled = false
		} else {
			manager.l2Cache = l2Cache
		}
	}
	
	return manager, nil
}

// Get retrieves a value from cache (L1 first, then L2)
func (cm *Manager) Get(ctx context.Context, key string) (*core.HistoryResponse, bool) {
	start := time.Now()
	
	// Try L1 cache first
	if cm.l1Enabled && cm.l1Cache != nil {
		if value, found := cm.l1Cache.Get(key); found {
			cm.metrics.Hits.WithLabelValues("l1").Inc()
			cm.metrics.Latency.WithLabelValues("l1", "get", "hit").Observe(time.Since(start).Seconds())
			return value, true
		}
		cm.metrics.Misses.WithLabelValues("l1").Inc()
	}
	
	// Try L2 cache (Redis)
	if cm.l2Enabled && cm.l2Cache != nil {
		l2Start := time.Now()
		value, err := cm.l2Cache.Get(ctx, key)
		if err == nil {
			cm.metrics.Hits.WithLabelValues("l2").Inc()
			cm.metrics.Latency.WithLabelValues("l2", "get", "hit").Observe(time.Since(l2Start).Seconds())
			
			// Populate L1 cache for next time
			if cm.l1Enabled && cm.l1Cache != nil {
				cm.l1Cache.Set(key, value)
			}
			
			return value, true
		}
		
		if err != ErrNotFound {
			cm.metrics.Errors.WithLabelValues("l2", err.(*CacheError).Type).Inc()
			cm.logger.Warn("L2 cache error", "error", err, "key", key)
		}
		cm.metrics.Misses.WithLabelValues("l2").Inc()
		cm.metrics.Latency.WithLabelValues("l2", "get", "miss").Observe(time.Since(l2Start).Seconds())
	}
	
	cm.metrics.Latency.WithLabelValues("combined", "get", "miss").Observe(time.Since(start).Seconds())
	return nil, false
}

// Set stores a value in both L1 and L2 caches
func (cm *Manager) Set(ctx context.Context, key string, value *core.HistoryResponse) error {
	start := time.Now()
	
	// Store in L1 cache
	if cm.l1Enabled && cm.l1Cache != nil {
		cm.l1Cache.Set(key, value)
		cm.metrics.Latency.WithLabelValues("l1", "set", "success").Observe(time.Since(start).Seconds())
	}
	
	// Store in L2 cache
	if cm.l2Enabled && cm.l2Cache != nil {
		l2Start := time.Now()
		if err := cm.l2Cache.Set(ctx, key, value); err != nil {
			cm.metrics.Errors.WithLabelValues("l2", err.(*CacheError).Type).Inc()
			cm.metrics.Latency.WithLabelValues("l2", "set", "error").Observe(time.Since(l2Start).Seconds())
			return err
		}
		cm.metrics.Latency.WithLabelValues("l2", "set", "success").Observe(time.Since(l2Start).Seconds())
	}
	
	return nil
}

// Invalidate removes a key from both caches
func (cm *Manager) Invalidate(ctx context.Context, key string) error {
	// Remove from L1
	if cm.l1Enabled && cm.l1Cache != nil {
		cm.l1Cache.Delete(key)
	}
	
	// Remove from L2
	if cm.l2Enabled && cm.l2Cache != nil {
		return cm.l2Cache.Delete(ctx, key)
	}
	
	return nil
}

// InvalidatePattern removes all keys matching a pattern from L2 cache
// Note: L1 cache doesn't support pattern deletion, uses TTL instead
func (cm *Manager) InvalidatePattern(ctx context.Context, pattern string) error {
	if cm.l2Enabled && cm.l2Cache != nil {
		return cm.l2Cache.DeletePattern(ctx, pattern)
	}
	return nil
}

// GenerateCacheKey generates a cache key from request parameters
func (cm *Manager) GenerateCacheKey(req *core.HistoryRequest) string {
	// Serialize request to JSON
	data, err := json.Marshal(req)
	if err != nil {
		cm.logger.Error("Failed to marshal request for cache key", "error", err)
		return ""
	}
	
	// Generate SHA-256 hash
	hash := sha256.Sum256(data)
	hashStr := base64.URLEncoding.EncodeToString(hash[:])
	
	// Format: "history:v2:{hash}"
	return fmt.Sprintf("history:v2:%s", hashStr)
}

// Stats returns cache statistics
func (cm *Manager) Stats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	if cm.l1Enabled && cm.l1Cache != nil {
		stats["l1"] = cm.l1Cache.Stats()
	}
	
	if cm.l2Enabled && cm.l2Cache != nil {
		stats["l2"] = map[string]interface{}{
			"enabled": true,
			// Redis stats would require additional Redis commands
		}
	}
	
	return stats
}

// UpdateMetrics updates Prometheus metrics
func (cm *Manager) UpdateMetrics() {
	if cm.l1Enabled && cm.l1Cache != nil {
		l1Stats := cm.l1Cache.Stats()
		cm.metrics.Size.WithLabelValues("l1").Set(float64(l1Stats["entries"].(int)))
	}
}

// Close closes cache connections
func (cm *Manager) Close() error {
	if cm.l2Cache != nil {
		return cm.l2Cache.Close()
	}
	return nil
}

