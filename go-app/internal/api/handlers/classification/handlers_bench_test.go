package classification

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// BenchmarkGetClassificationStats_Basic benchmarks the basic handler without cache
func BenchmarkGetClassificationStats_Basic(b *testing.B) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  10000,
			CacheHitRate:   0.65,
			LLMSuccessRate: 0.98,
			FallbackRate:   0.02,
			AvgResponseTime: 50 * time.Millisecond,
		},
	}

	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)
	// Disable cache for this benchmark
	handlers.statsCache = nil

	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.GetClassificationStats(w, req)
	}
}

// BenchmarkGetClassificationStats_Cached benchmarks the handler with cache enabled
func BenchmarkGetClassificationStats_Cached(b *testing.B) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  10000,
			CacheHitRate:   0.65,
			LLMSuccessRate: 0.98,
			FallbackRate:   0.02,
			AvgResponseTime: 50 * time.Millisecond,
		},
	}

	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)
	handlers.statsCache.SetTTL(10 * time.Second)

	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "benchmark"))

	// Warm up cache
	w := httptest.NewRecorder()
	handlers.GetClassificationStats(w, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handlers.GetClassificationStats(w, req)
	}
}

// BenchmarkAggregateStats benchmarks the stats aggregation logic
func BenchmarkAggregateStats(b *testing.B) {
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:  10000,
			CacheHitRate:   0.65,
			LLMSuccessRate: 0.98,
			FallbackRate:   0.02,
			AvgResponseTime: 50 * time.Millisecond,
		},
	}

	aggregator := NewStatsAggregator(mockService, nil)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = aggregator.AggregateStats(ctx)
	}
}

// BenchmarkStatsCache_Get benchmarks cache get operation
func BenchmarkStatsCache_Get(b *testing.B) {
	cache := NewStatsCache(5 * time.Second)

	// Pre-populate cache
	stats := &StatsResponse{
		TotalRequests:  1000,
		TotalClassified: 1000,
		Timestamp:      time.Now(),
	}
	cache.Set(stats)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get()
	}
}

// BenchmarkStatsCache_Set benchmarks cache set operation
func BenchmarkStatsCache_Set(b *testing.B) {
	cache := NewStatsCache(5 * time.Second)
	stats := &StatsResponse{
		TotalRequests:  1000,
		TotalClassified: 1000,
		Timestamp:      time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(stats)
	}
}
