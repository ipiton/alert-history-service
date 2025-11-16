package cache

import (
	"context"
	"log/slog"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Warmer pre-populates cache with popular queries
type Warmer struct {
	cacheManager *Manager
	repository   core.AlertHistoryRepository
	logger       *slog.Logger
	stopCh       chan struct{}
}

// NewWarmer creates a new cache warmer
func NewWarmer(
	cacheManager *Manager,
	repository core.AlertHistoryRepository,
	logger *slog.Logger,
) *Warmer {
	if logger == nil {
		logger = slog.Default()
	}

	return &Warmer{
		cacheManager: cacheManager,
		repository:   repository,
		logger:       logger,
		stopCh:       make(chan struct{}),
	}
}

// Start starts the cache warming background worker
func (cw *Warmer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Warm cache immediately on start
	cw.warmCache(ctx)

	for {
		select {
		case <-ticker.C:
			cw.warmCache(ctx)
		case <-cw.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the cache warming worker
func (cw *Warmer) Stop() {
	close(cw.stopCh)
}

// warmCache pre-populates cache with popular queries
func (cw *Warmer) warmCache(ctx context.Context) {
	cw.logger.Info("Starting cache warming")
	start := time.Now()

	// Define popular query patterns
	popularQueries := []struct {
		name string
		req  *core.HistoryRequest
	}{
		{
			name: "recent_firing_critical",
			req: &core.HistoryRequest{
				Filters: &core.AlertFilters{
					Status:   ptrStatus(core.StatusFiring),
					Severity: ptrString("critical"),
				},
				Pagination: &core.Pagination{
					Page:    1,
					PerPage: 50,
				},
			},
		},
		{
			name: "recent_all",
			req: &core.HistoryRequest{
				Filters: &core.AlertFilters{},
				Pagination: &core.Pagination{
					Page:    1,
					PerPage: 50,
				},
			},
		},
		{
			name: "recent_firing",
			req: &core.HistoryRequest{
				Filters: &core.AlertFilters{
					Status: ptrStatus(core.StatusFiring),
				},
				Pagination: &core.Pagination{
					Page:    1,
					PerPage: 50,
				},
			},
		},
	}

	// Warm cache for each popular query
	warmed := 0
	for _, pq := range popularQueries {
		// Check if already cached
		cacheKey := cw.cacheManager.GenerateCacheKey(pq.req)
		if _, found := cw.cacheManager.Get(ctx, cacheKey); found {
			continue // Already cached
		}

		// Query from database
		response, err := cw.repository.GetHistory(ctx, pq.req)
		if err != nil {
			cw.logger.Warn("Failed to warm cache for query",
				"query", pq.name,
				"error", err)
			continue
		}

		// Store in cache
		if err := cw.cacheManager.Set(ctx, cacheKey, response); err != nil {
			cw.logger.Warn("Failed to cache warmed query",
				"query", pq.name,
				"error", err)
			continue
		}

		warmed++
	}

	duration := time.Since(start)
	cw.logger.Info("Cache warming complete",
		"warmed_queries", warmed,
		"total_queries", len(popularQueries),
		"duration_ms", duration.Milliseconds())
}

func ptrString(s string) *string {
	return &s
}

func ptrStatus(s core.AlertStatus) *core.AlertStatus {
	return &s
}
