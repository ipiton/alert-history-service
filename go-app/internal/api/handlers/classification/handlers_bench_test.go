package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
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

// ===== Benchmarks for ClassifyAlert Endpoint =====

// BenchmarkClassifyAlert_CacheHit benchmarks cache hit scenario
func BenchmarkClassifyAlert_CacheHit(b *testing.B) {
	cachedResult := &core.ClassificationResult{
		Severity:       core.SeverityWarning,
		Confidence:     0.85,
		Reasoning:      "Cached",
		ProcessingTime: 0.01,
	}

	mockService := &MockClassificationService{
		getCachedFunc: func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
			return cachedResult, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "benchmark-123",
			AlertName:   "BenchmarkAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))
		w := httptest.NewRecorder()
		handlers.ClassifyAlert(w, req)
	}
}

// BenchmarkClassifyAlert_CacheMiss benchmarks cache miss scenario
func BenchmarkClassifyAlert_CacheMiss(b *testing.B) {
	mockService := &MockClassificationService{
		getCachedFunc: func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
			return nil, nil // Cache miss
		},
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityInfo,
				Confidence:     0.75,
				Reasoning:      "New classification",
				ProcessingTime: 0.2,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "benchmark-123",
			AlertName:   "BenchmarkAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))
		w := httptest.NewRecorder()
		handlers.ClassifyAlert(w, req)
	}
}

// BenchmarkClassifyAlert_ForceFlag benchmarks force flag scenario
func BenchmarkClassifyAlert_ForceFlag(b *testing.B) {
	mockService := &MockClassificationService{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityCritical,
				Confidence:     0.98,
				Reasoning:      "Forced",
				ProcessingTime: 0.15,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "benchmark-123",
			AlertName:   "BenchmarkAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))
		w := httptest.NewRecorder()
		handlers.ClassifyAlert(w, req)
	}
}

// BenchmarkClassifyAlert_Validation benchmarks validation overhead
func BenchmarkClassifyAlert_Validation(b *testing.B) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "benchmark-123",
			AlertName:   "BenchmarkAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))
		w := httptest.NewRecorder()
		handlers.ClassifyAlert(w, req)
	}
}

// BenchmarkClassifyAlert_WithoutService benchmarks handler without ClassificationService
func BenchmarkClassifyAlert_WithoutService(b *testing.B) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.95,
				Reasoning:      "Test",
				ProcessingTime: 0.05,
			}, nil
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, nil)

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "benchmark-123",
			AlertName:   "BenchmarkAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "benchmark"))
		w := httptest.NewRecorder()
		handlers.ClassifyAlert(w, req)
	}
}

// BenchmarkValidateAlert benchmarks validation function
func BenchmarkValidateAlert(b *testing.B) {
	handlers := NewClassificationHandlers(&MockClassifier{}, nil)
	alert := &core.Alert{
		Fingerprint: "benchmark-123",
		AlertName:   "BenchmarkAlert",
		Status:      core.StatusFiring,
		StartsAt:    time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = handlers.validateAlert(alert)
	}
}

// BenchmarkFormatDuration benchmarks duration formatting
func BenchmarkFormatDuration(b *testing.B) {
	durations := []time.Duration{
		500 * time.Microsecond,
		50 * time.Millisecond,
		2 * time.Second,
		1500 * time.Millisecond,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := durations[i%len(durations)]
		_ = formatDuration(d)
	}
}
