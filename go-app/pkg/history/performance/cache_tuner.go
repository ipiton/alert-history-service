package performance

import (
	"context"
	"log/slog"
	"time"
	
	"github.com/vitaliisemenov/alert-history/pkg/history/cache"
)

// CacheTuner optimizes cache configuration based on usage patterns
type CacheTuner struct {
	cacheManager *cache.Manager
	logger       *slog.Logger
	stats        *CacheStats
}

// CacheStats tracks cache performance statistics
type CacheStats struct {
	HitRate      float64
	MissRate     float64
	AvgLatency   time.Duration
	EvictionRate float64
	Size         int64
	MaxSize      int64
}

// NewCacheTuner creates a new cache tuner
func NewCacheTuner(cacheManager *cache.Manager, logger *slog.Logger) *CacheTuner {
	if logger == nil {
		logger = slog.Default()
	}
	
	return &CacheTuner{
		cacheManager: cacheManager,
		logger:       logger,
		stats:         &CacheStats{},
	}
}

// AnalyzeCachePerformance analyzes cache performance and suggests optimizations
func (ct *CacheTuner) AnalyzeCachePerformance(ctx context.Context) (*CacheStats, []string) {
	stats := ct.cacheManager.Stats()
	suggestions := []string{}
	
	// Get L1 cache stats
	if l1Stats, ok := stats["l1"].(map[string]interface{}); ok {
		entries := l1Stats["entries"].(int)
		maxEntries := l1Stats["max_entries"].(int64)
		utilization := l1Stats["utilization"].(float64)
		
		ct.stats.Size = int64(entries)
		ct.stats.MaxSize = maxEntries
		
		// Suggest optimizations based on utilization
		if utilization > 90 {
			suggestions = append(suggestions, "L1 cache utilization > 90%, consider increasing max_entries")
		} else if utilization < 20 {
			suggestions = append(suggestions, "L1 cache utilization < 20%, consider decreasing max_entries to save memory")
		}
	}
	
	// Analyze hit rate (would need metrics)
	// For now, provide placeholder suggestions
	if ct.stats.HitRate < 0.8 {
		suggestions = append(suggestions, "Cache hit rate < 80%, consider increasing TTL or cache warming")
	}
	
	return ct.stats, suggestions
}

// OptimizeCacheConfig optimizes cache configuration based on usage
func (ct *CacheTuner) OptimizeCacheConfig(current *cache.Config) *cache.Config {
	optimized := *current
	
	// Adjust L1 cache size based on memory pressure
	// This would be based on actual metrics
	if optimized.L1MaxEntries < 50000 {
		optimized.L1MaxEntries = 50000 // Increase for better hit rate
	}
	
	// Adjust TTL based on data freshness requirements
	if optimized.L1TTL < 5*time.Minute {
		optimized.L1TTL = 5 * time.Minute // Minimum TTL
	}
	
	// Adjust L2 TTL for better hit rate
	if optimized.L2TTL < 1*time.Hour {
		optimized.L2TTL = 1 * time.Hour // Minimum L2 TTL
	}
	
	return &optimized
}

// WarmCacheForPopularQueries warms cache with popular query patterns
func (ct *CacheTuner) WarmCacheForPopularQueries(ctx context.Context, repository interface{}) error {
	// This would identify popular queries and pre-populate cache
	// For now, just log
	ct.logger.Info("Cache warming for popular queries")
	return nil
}

// MonitorCacheHealth monitors cache health and alerts on issues
func (ct *CacheTuner) MonitorCacheHealth(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			stats, suggestions := ct.AnalyzeCachePerformance(ctx)
			
			// Log warnings for poor performance
			if stats.HitRate < 0.7 {
				ct.logger.Warn("Cache hit rate below threshold",
					"hit_rate", stats.HitRate,
					"threshold", 0.7)
			}
			
			// Log suggestions
			for _, suggestion := range suggestions {
				ct.logger.Info("Cache optimization suggestion", "suggestion", suggestion)
			}
			
		case <-ctx.Done():
			return
		}
	}
}

