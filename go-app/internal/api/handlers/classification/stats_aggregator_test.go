package classification

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

func TestStatsAggregator_AggregateStats_Basic(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:   1000,
			CacheHitRate:    0.75,
			LLMSuccessRate:  0.95,
			FallbackRate:    0.05,
			AvgResponseTime: 100 * time.Millisecond,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Validate basic metrics
	assert.Equal(t, int64(1000), response.TotalRequests)
	assert.Equal(t, int64(1000), response.TotalClassified)
	assert.Equal(t, 0.95, response.ClassificationRate) // 1.0 - 0.05 fallback rate
	assert.Equal(t, 100.0, response.AvgProcessing)     // 100ms in milliseconds

	// Validate cache stats
	assert.Equal(t, 0.75, response.CacheStats.HitRate)
	assert.Equal(t, int64(250), response.CacheStats.Misses) // 25% of 1000

	// Validate LLM stats
	assert.Equal(t, 0.95, response.LLMStats.SuccessRate)
	assert.Equal(t, 0.25, response.LLMStats.UsageRate) // 25% cache miss rate
	assert.Equal(t, int64(250), response.LLMStats.Requests) // 25% of 1000

	// Validate fallback stats
	assert.Equal(t, 0.05, response.FallbackStats.Rate)
	assert.Equal(t, int64(50), response.FallbackStats.Used) // 5% of 1000

	// Validate error stats
	assert.GreaterOrEqual(t, response.ErrorStats.Total, int64(0))
	assert.GreaterOrEqual(t, response.ErrorStats.Rate, 0.0)

	// Validate severity stats are initialized
	assert.NotNil(t, response.BySeverity)
	assert.Contains(t, response.BySeverity, "critical")
	assert.Contains(t, response.BySeverity, "warning")
	assert.Contains(t, response.BySeverity, "info")
	assert.Contains(t, response.BySeverity, "noise")
}

func TestStatsAggregator_AggregateStats_ZeroRequests(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:   0,
			CacheHitRate:    0.0,
			LLMSuccessRate:  0.0,
			FallbackRate:    0.0,
			AvgResponseTime: 0,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Validate zero values
	assert.Equal(t, int64(0), response.TotalRequests)
	assert.Equal(t, int64(0), response.TotalClassified)
	assert.Equal(t, 0.0, response.ClassificationRate)
	assert.Equal(t, 0.0, response.AvgProcessing)

	// Validate cache stats
	assert.Equal(t, 0.0, response.CacheStats.HitRate)
	assert.Equal(t, int64(0), response.CacheStats.Misses)

	// Validate LLM stats
	assert.Equal(t, 0.0, response.LLMStats.SuccessRate)
	assert.Equal(t, 0.0, response.LLMStats.UsageRate)
	assert.Equal(t, int64(0), response.LLMStats.Requests)

	// Validate fallback stats
	assert.Equal(t, 0.0, response.FallbackStats.Rate)
	assert.Equal(t, int64(0), response.FallbackStats.Used)
}

func TestStatsAggregator_AggregateStats_AllCacheHits(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:   500,
			CacheHitRate:    1.0, // 100% cache hits
			LLMSuccessRate:  0.0, // No LLM calls
			FallbackRate:    0.0,
			AvgResponseTime: 5 * time.Millisecond,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Validate cache stats
	assert.Equal(t, 1.0, response.CacheStats.HitRate)
	assert.Equal(t, int64(0), response.CacheStats.Misses)

	// Validate LLM stats (should be zero)
	assert.Equal(t, 0.0, response.LLMStats.UsageRate)
	assert.Equal(t, int64(0), response.LLMStats.Requests)

	// Validate fallback stats (should be zero)
	assert.Equal(t, 0.0, response.FallbackStats.Rate)
	assert.Equal(t, int64(0), response.FallbackStats.Used)
}

func TestStatsAggregator_AggregateStats_AllFallback(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:   200,
			CacheHitRate:    0.0, // No cache hits
			LLMSuccessRate:  0.0, // All LLM calls failed
			FallbackRate:    1.0, // 100% fallback
			AvgResponseTime: 2 * time.Millisecond,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, response)

	// Validate fallback stats
	assert.Equal(t, 1.0, response.FallbackStats.Rate)
	assert.Equal(t, int64(200), response.FallbackStats.Used)

	// Validate LLM stats (all failed)
	assert.Equal(t, 0.0, response.LLMStats.SuccessRate)
	assert.Equal(t, int64(200), response.LLMStats.Requests) // All cache misses went to LLM
	assert.Equal(t, int64(200), response.LLMStats.Failures) // All failed (100% failure rate)

	// Validate cache stats
	assert.Equal(t, 0.0, response.CacheStats.HitRate)
	assert.Equal(t, int64(200), response.CacheStats.Misses)
}

func TestStatsAggregator_CalculateSeverityStats(t *testing.T) {
	// This test will be enhanced when Prometheus integration is added
	// For now, we just verify that severity stats are initialized
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests: 100,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)

	// Verify all severity levels are present
	expectedSeverities := []string{"critical", "warning", "info", "noise"}
	for _, severity := range expectedSeverities {
		stats, exists := response.BySeverity[severity]
		assert.True(t, exists, "Severity %s should exist", severity)
		assert.Equal(t, int64(0), stats.Count)
		assert.Equal(t, 0.0, stats.AvgConfidence)
	}
}

func TestStatsAggregator_CalculateCacheStats(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests: 1000,
			CacheHitRate:  0.65,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)

	cacheStats := response.CacheStats
	assert.Equal(t, 0.65, cacheStats.HitRate)
	assert.Equal(t, int64(350), cacheStats.Misses) // 35% of 1000
}

func TestStatsAggregator_CalculateLLMStats(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  1000,
			CacheHitRate:   0.5,  // 50% cache hits
			LLMSuccessRate: 0.9,  // 90% LLM success
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)

	llmStats := response.LLMStats
	assert.Equal(t, 0.9, llmStats.SuccessRate)
	assert.Equal(t, 0.5, llmStats.UsageRate) // 50% cache miss rate
	assert.Equal(t, int64(500), llmStats.Requests) // 50% of 1000
	// Allow for rounding: 10% of 500 = 50, but may be 49-51 due to float64 precision
	assert.InDelta(t, 50, llmStats.Failures, 1) // 10% of 500 = 50 ± 1
}

func TestStatsAggregator_CalculateFallbackStats(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests: 1000,
			FallbackRate:  0.1, // 10% fallback
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)

	fallbackStats := response.FallbackStats
	assert.Equal(t, 0.1, fallbackStats.Rate)
	assert.Equal(t, int64(100), fallbackStats.Used) // 10% of 1000
	assert.Greater(t, fallbackStats.AvgLatencyMs, 0.0) // Should have latency
}

func TestStatsAggregator_CalculateErrorStats(t *testing.T) {
	now := time.Now()
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests: 1000,
			CacheHitRate:  0.5,
			LLMSuccessRate: 0.8, // 20% LLM failures
			LastError:     "LLM timeout",
			LastErrorTime: &now,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	response, err := aggregator.AggregateStats(ctx)
	require.NoError(t, err)

	errorStats := response.ErrorStats
	assert.Equal(t, "LLM timeout", errorStats.LastError)
	assert.NotNil(t, errorStats.LastErrorTime)
	assert.Greater(t, errorStats.Total, int64(0)) // Should have some errors
	assert.Greater(t, errorStats.Rate, 0.0)
}

func TestStatsAggregator_ConcurrentAccess(t *testing.T) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests: 1000,
			CacheHitRate:  0.65,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			response, err := aggregator.AggregateStats(ctx)
			assert.NoError(t, err)
			assert.NotNil(t, response)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
