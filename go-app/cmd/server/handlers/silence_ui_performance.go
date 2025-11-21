// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"log/slog"
	"sync"
	"time"

	coresilencing "github.com/vitaliisemenov/alert-history/internal/core/silencing"
	infrasilencing "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
)

// PerformanceMonitor provides performance monitoring and optimization.
// Phase 10: Performance Optimization enhancement.
type PerformanceMonitor struct {
	renderTimes map[string][]time.Duration
	mu          sync.RWMutex
	logger      *slog.Logger
}

// NewPerformanceMonitor creates a new performance monitor.
func NewPerformanceMonitor(logger *slog.Logger) *PerformanceMonitor {
	return &PerformanceMonitor{
		renderTimes: make(map[string][]time.Duration),
		logger:      logger,
	}
}

// RecordRenderTime records a page render time.
func (pm *PerformanceMonitor) RecordRenderTime(page string, duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.renderTimes[page] == nil {
		pm.renderTimes[page] = make([]time.Duration, 0, 100)
	}

	pm.renderTimes[page] = append(pm.renderTimes[page], duration)

	// Keep only last 100 measurements
	if len(pm.renderTimes[page]) > 100 {
		pm.renderTimes[page] = pm.renderTimes[page][len(pm.renderTimes[page])-100:]
	}
}

// GetAverageRenderTime returns average render time for a page.
func (pm *PerformanceMonitor) GetAverageRenderTime(page string) time.Duration {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	times, exists := pm.renderTimes[page]
	if !exists || len(times) == 0 {
		return 0
	}

	var sum time.Duration
	for _, t := range times {
		sum += t
	}

	return sum / time.Duration(len(times))
}

// GetStats returns performance statistics.
func (pm *PerformanceMonitor) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	for page, times := range pm.renderTimes {
		if len(times) == 0 {
			continue
		}

		var sum, min, max time.Duration
		min = times[0]
		max = times[0]

		for _, t := range times {
			sum += t
			if t < min {
				min = t
			}
			if t > max {
				max = t
			}
		}

		avg := sum / time.Duration(len(times))

		stats[page] = map[string]interface{}{
			"count":    len(times),
			"avg_ms":   avg.Milliseconds(),
			"min_ms":   min.Milliseconds(),
			"max_ms":   max.Milliseconds(),
			"total_ms": sum.Milliseconds(),
		}
	}

	return stats
}

// DatabaseQueryOptimizer provides query optimization hints.
// Phase 10: Database Query Optimization enhancement.
type DatabaseQueryOptimizer struct {
	logger *slog.Logger
}

// NewDatabaseQueryOptimizer creates a new query optimizer.
func NewDatabaseQueryOptimizer(logger *slog.Logger) *DatabaseQueryOptimizer {
	return &DatabaseQueryOptimizer{
		logger: logger,
	}
}

// OptimizeFilter provides optimization hints for silence filters.
func (dqo *DatabaseQueryOptimizer) OptimizeFilter(filter infrasilencing.SilenceFilter) infrasilencing.SilenceFilter {
	// Ensure reasonable limits
	if filter.Limit <= 0 {
		filter.Limit = 100 // Default limit
	}
	if filter.Limit > 1000 {
		filter.Limit = 1000 // Max limit to prevent DoS
		dqo.logger.Warn("Filter limit capped at 1000",
			"requested_limit", filter.Limit,
		)
	}

	// Ensure reasonable offset
	if filter.Offset < 0 {
		filter.Offset = 0
	}

	return filter
}

// OptimizeQueryContext adds timeout to query context.
func (dqo *DatabaseQueryOptimizer) OptimizeQueryContext(ctx context.Context) (context.Context, context.CancelFunc) {
	// Add 5-second timeout for database queries
	return context.WithTimeout(ctx, 5*time.Second)
}

// QueryPerformanceHint provides performance hints for queries.
func (dqo *DatabaseQueryOptimizer) QueryPerformanceHint(filter infrasilencing.SilenceFilter) string {
	// Provide hints based on filter complexity
	if len(filter.Statuses) == 1 && filter.Statuses[0] == coresilencing.SilenceStatusActive {
		return "Use cache (fast path)"
	}
	if filter.Limit > 100 {
		return "Consider pagination"
	}
	return "Standard query"
}
