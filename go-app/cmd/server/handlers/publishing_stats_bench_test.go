package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// ============================================================================
// Benchmark: HTTP Endpoints
// Target: <10ms per request (p95)
// ============================================================================

// BenchmarkGetMetrics benchmarks the /metrics endpoint
func BenchmarkGetMetrics(b *testing.B) {
	// Setup: Mock collector with 50 metrics
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := make(map[string]float64, 50)
			for i := 0; i < 50; i++ {
				metrics["metric_"+string(rune('a'+i%26))] = float64(i)
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  25 * time.Microsecond,
				AvailableCollectors: []string{"test1", "test2"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/publishing/metrics", nil)
		w := httptest.NewRecorder()
		handler.GetMetrics(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetStats benchmarks the /stats endpoint
func BenchmarkGetStats(b *testing.B) {
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := map[string]float64{
				"publishing_jobs_submitted_total": 1000,
				"publishing_jobs_succeeded_total": 950,
				"publishing_jobs_failed_total":    50,
				"publishing_job_duration_seconds": 0.125,
				"queue_size_total":                45,
				"queue_capacity":                  1000,
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  30 * time.Microsecond,
				AvailableCollectors: []string{"queue"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()
		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetHealth benchmarks the /health endpoint
func BenchmarkGetHealth(b *testing.B) {
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := map[string]float64{
				"health_status_rootly":              1,
				"health_consecutive_failures_rootly": 0,
				"health_success_rate_rootly":        0.95,
				"health_status_pagerduty":           1,
				"health_success_rate_pagerduty":     0.98,
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  20 * time.Microsecond,
				AvailableCollectors: []string{"health"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/publishing/health", nil)
		w := httptest.NewRecorder()
		handler.GetHealth(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetTargetStats benchmarks the /stats/{target} endpoint
func BenchmarkGetTargetStats(b *testing.B) {
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := map[string]float64{
				"health_status_rootly":                1,
				"health_success_rate_rootly":          0.95,
				"publishing_jobs_succeeded_rootly":    100,
				"publishing_jobs_failed_rootly":       5,
				"publishing_job_duration_rootly":      0.150,
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  25 * time.Microsecond,
				AvailableCollectors: []string{"health", "queue"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/publishing/stats/rootly", nil)
		req.SetPathValue("target", "rootly")
		w := httptest.NewRecorder()
		handler.GetTargetStats(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// BenchmarkGetTrends benchmarks the /trends endpoint
func BenchmarkGetTrends(b *testing.B) {
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := map[string]float64{
				"publishing_jobs_succeeded_total": 950,
				"publishing_jobs_failed_total":    50,
				"publishing_job_duration_seconds": 0.125,
				"queue_size_total":                45,
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  30 * time.Microsecond,
				AvailableCollectors: []string{"queue"},
				Errors:              make(map[string]error),
			}
		},
	}

	// Create trend detector with historical data
	trendDetector := publishing.NewTrendDetector()

	handler := &PublishingStatsHandler{
		collector:     mockCollector,
		trendDetector: trendDetector,
		logger:        slog.Default(),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/publishing/trends", nil)
		w := httptest.NewRecorder()
		handler.GetTrends(w, req)

		if w.Code != http.StatusOK {
			b.Fatalf("Expected status 200, got %d", w.Code)
		}
	}
}

// ============================================================================
// Benchmark: Concurrent Request Handling
// ============================================================================

// BenchmarkConcurrentGetMetrics tests concurrent requests to /metrics
func BenchmarkConcurrentGetMetrics(b *testing.B) {
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := make(map[string]float64, 50)
			for i := 0; i < 50; i++ {
				metrics["metric_"+string(rune('a'+i%26))] = float64(i)
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  25 * time.Microsecond,
				AvailableCollectors: []string{"test1", "test2"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/v2/publishing/metrics", nil)
			w := httptest.NewRecorder()
			handler.GetMetrics(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// BenchmarkConcurrentGetStats tests concurrent requests to /stats
func BenchmarkConcurrentGetStats(b *testing.B) {
	mockCollector := &MockCollectorForHandler{
		CollectAllFunc: func(ctx context.Context) *publishing.MetricsSnapshot {
			metrics := map[string]float64{
				"publishing_jobs_submitted_total": 1000,
				"publishing_jobs_succeeded_total": 950,
				"publishing_jobs_failed_total":    50,
				"publishing_job_duration_seconds": 0.125,
			}
			return &publishing.MetricsSnapshot{
				Timestamp:           time.Now(),
				Metrics:             metrics,
				CollectionDuration:  30 * time.Microsecond,
				AvailableCollectors: []string{"queue"},
				Errors:              make(map[string]error),
			}
		},
	}

	handler := NewPublishingStatsHandlerWithCollector(mockCollector, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/v2/publishing/stats", nil)
			w := httptest.NewRecorder()
			handler.GetStats(w, req)

			if w.Code != http.StatusOK {
				b.Fatalf("Expected status 200, got %d", w.Code)
			}
		}
	})
}

// ============================================================================
// Benchmark: JSON Encoding
// ============================================================================

// BenchmarkJSONEncoding_MetricsResponse tests JSON encoding performance
func BenchmarkJSONEncoding_MetricsResponse(b *testing.B) {
	metrics := make(map[string]float64, 50)
	for i := 0; i < 50; i++ {
		metrics["metric_"+string(rune('a'+i%26))] = float64(i)
	}

	response := MetricsResponse{
		Timestamp:           time.Now(),
		Metrics:             metrics,
		MetricsCount:        50,
		CollectionDuration:  "25µs",
		AvailableCollectors: []string{"test1", "test2"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(response); err != nil {
			b.Fatalf("Failed to encode: %v", err)
		}
	}
}

// BenchmarkJSONEncoding_StatsResponse tests JSON encoding for stats response
func BenchmarkJSONEncoding_StatsResponse(b *testing.B) {
	response := StatsResponse{
		Timestamp: time.Now(),
		System: SystemStats{
			TotalTargets:     5,
			HealthyTargets:   4,
			UnhealthyTargets: 1,
			SuccessRate:      95.0,
			QueueSize:        45,
			QueueCapacity:    1000,
		},
		TargetStats: map[string]float64{
			"rootly_success_rate":    95.0,
			"pagerduty_success_rate": 98.0,
		},
		QueueStats: map[string]float64{
			"queue_size":     45.0,
			"queue_capacity": 1000.0,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		if err := json.NewEncoder(buf).Encode(response); err != nil {
			b.Fatalf("Failed to encode: %v", err)
		}
	}
}

// ============================================================================
// Benchmark: Helper Functions
// ============================================================================

// BenchmarkExtractTargetHealthStatus benchmarks health status extraction
func BenchmarkExtractTargetHealthStatus(b *testing.B) {
	metrics := map[string]float64{
		"health_status_rootly":    1,
		"health_status_pagerduty": 0,
		"health_status_slack":     1,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = extractTargetHealthStatus(metrics, "rootly")
		_ = extractTargetHealthStatus(metrics, "pagerduty")
		_ = extractTargetHealthStatus(metrics, "slack")
	}
}

// BenchmarkCalculateTargetJobSuccessRate benchmarks success rate calculation
func BenchmarkCalculateTargetJobSuccessRate(b *testing.B) {
	metrics := map[string]float64{
		"publishing_jobs_succeeded_rootly": 100,
		"publishing_jobs_failed_rootly":    5,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = calculateTargetJobSuccessRate(metrics, "rootly")
	}
}
