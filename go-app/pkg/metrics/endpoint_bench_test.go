package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// BenchmarkMetricsEndpointHandler_ServeHTTP benchmarks the basic ServeHTTP method.
func BenchmarkMetricsEndpointHandler_ServeHTTP(b *testing.B) {
	config := DefaultEndpointConfig()
	config.Path = "/metrics"
	registry := DefaultRegistry()

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkMetricsEndpointHandler_ServeHTTP_WithCache benchmarks with caching enabled.
func BenchmarkMetricsEndpointHandler_ServeHTTP_WithCache(b *testing.B) {
	config := DefaultEndpointConfig()
	config.Path = "/metrics"
	config.CacheEnabled = true
	config.CacheTTL = 5 * time.Second
	registry := DefaultRegistry()

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	// Warm up cache with first request
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkMetricsEndpointHandler_ServeHTTP_WithManyMetrics benchmarks with many metrics.
func BenchmarkMetricsEndpointHandler_ServeHTTP_WithManyMetrics(b *testing.B) {
	config := DefaultEndpointConfig()
	config.Path = "/metrics"
	config.EnableGoRuntime = true
	registry := DefaultRegistry()

	// Initialize all metrics to ensure they're registered
	business := registry.Business()
	technical := registry.Technical()
	infra := registry.Infra()

	// Use metrics to ensure they're created
	for i := 0; i < 100; i++ {
		business.AlertsProcessedTotal.WithLabelValues("test").Inc()
		// Ensure HTTP metrics are initialized
		if technical.HTTP != nil {
			_ = technical.HTTP
		}
		infra.DB.ConnectionsActive.Set(float64(i))
	}

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

// BenchmarkMetricsEndpointHandler_ServeHTTP_Concurrent benchmarks concurrent requests.
func BenchmarkMetricsEndpointHandler_ServeHTTP_Concurrent(b *testing.B) {
	config := DefaultEndpointConfig()
	config.Path = "/metrics"
	config.EnableSelfMetrics = true
	registry := DefaultRegistry()

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		for pb.Next() {
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
		}
	})
}

// BenchmarkMetricsEndpointHandler_GatherMetrics benchmarks metrics gathering.
func BenchmarkMetricsEndpointHandler_GatherMetrics(b *testing.B) {
	config := DefaultEndpointConfig()
	registry := DefaultRegistry()

	// Initialize metrics
	business := registry.Business()
	technical := registry.Technical()
	infra := registry.Infra()

	// Create many metrics
	for i := 0; i < 50; i++ {
		business.AlertsProcessedTotal.WithLabelValues("test").Inc()
		// Ensure HTTP metrics are initialized
		if technical.HTTP != nil {
			_ = technical.HTTP
		}
		infra.DB.ConnectionsActive.Set(float64(i))
	}

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := handler.gatherer.Gather()
		if err != nil {
			b.Fatalf("Failed to gather metrics: %v", err)
		}
	}
}

// BenchmarkMetricsEndpointHandler_WriteResponse benchmarks response writing.
func BenchmarkMetricsEndpointHandler_WriteResponse(b *testing.B) {
	config := DefaultEndpointConfig()
	registry := DefaultRegistry()

	// Initialize metrics
	business := registry.Business()
	technical := registry.Technical()
	infra := registry.Infra()

	// Create many metrics
	for i := 0; i < 50; i++ {
		business.AlertsProcessedTotal.WithLabelValues("test").Inc()
		// Ensure HTTP metrics are initialized
		if technical.HTTP != nil {
			_ = technical.HTTP
		}
		infra.DB.ConnectionsActive.Set(float64(i))
	}

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	// Gather metrics once
	families, err := handler.gatherer.Gather()
	if err != nil {
		b.Fatalf("Failed to gather metrics: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		_, err := handler.writeResponse(w, families)
		if err != nil {
			b.Fatalf("Failed to write response: %v", err)
		}
	}
}

// BenchmarkMetricsEndpointHandler_WithCustomGatherer benchmarks with custom gatherer.
func BenchmarkMetricsEndpointHandler_WithCustomGatherer(b *testing.B) {
	config := DefaultEndpointConfig()
	config.Path = "/metrics"

	// Simplified: create just a few metrics to avoid registration conflicts
	customRegistry := prometheus.NewRegistry()
	for i := 0; i < 10; i++ {
		counter := prometheus.NewCounter(prometheus.CounterOpts{
			Name: prometheus.BuildFQName("test", "bench", "metric_total"),
		})
		if err := customRegistry.Register(counter); err == nil {
			counter.Inc()
		}
	}

	config.CustomGatherer = customRegistry
	registry := DefaultRegistry()

	handler, err := NewMetricsEndpointHandler(config, registry)
	if err != nil {
		b.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
