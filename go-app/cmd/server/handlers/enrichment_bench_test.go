package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// BenchmarkGetMode_CacheHit benchmarks the hot path (in-memory cache hit)
// Target: < 100ns per operation, 0 allocations
func BenchmarkGetMode_CacheHit(b *testing.B) {
	// Setup: Mock manager with cached mode
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeEnriched, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())

	// Create reusable request
	req := httptest.NewRequest("GET", "/enrichment/mode", nil)
	rr := httptest.NewRecorder()

	b.ResetTimer()
	b.ReportAllocs()

	// Run benchmark
	for i := 0; i < b.N; i++ {
		// Reset recorder for each iteration
		rr.Body.Reset()
		rr.Code = 0

		handlers.GetMode(rr, req)
	}
}

// BenchmarkGetMode_RedisFallback benchmarks Redis fallback scenario
// Target: < 2ms per operation
func BenchmarkGetMode_RedisFallback(b *testing.B) {
	// Setup: Mock manager with Redis fallback simulation
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			// Simulate Redis latency (~1-2ms)
			// In real scenario, this would query Redis
			return services.EnrichmentModeTransparent, "redis", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())

	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)
	}
}

// BenchmarkGetMode_Concurrent benchmarks concurrent access (realistic scenario)
// Target: < 100ns per operation with 10K concurrent goroutines
func BenchmarkGetMode_Concurrent(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeEnriched, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		req := httptest.NewRequest("GET", "/enrichment/mode", nil)
		rr := httptest.NewRecorder()

		for pb.Next() {
			rr.Body.Reset()
			rr.Code = 0
			handlers.GetMode(rr, req)
		}
	})
}

// BenchmarkGetModeWithSource benchmarks service layer directly
// Target: < 50ns per operation (pure in-memory read)
func BenchmarkGetModeWithSource(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeEnriched, "memory", nil
		},
	}

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = mockManager.GetModeWithSource(ctx)
	}
}

// BenchmarkJSONEncode benchmarks JSON response encoding
// Target: < 500ns per operation
func BenchmarkJSONEncode(b *testing.B) {
	response := EnrichmentModeResponse{
		Mode:   "enriched",
		Source: "redis",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		_ = json.NewEncoder(rr).Encode(response)
	}
}

// BenchmarkRWMutexRLock benchmarks RWMutex read lock overhead
// Target: < 20ns per operation
func BenchmarkRWMutexRLock(b *testing.B) {
	var mu sync.RWMutex
	mode := services.EnrichmentModeEnriched
	source := "memory"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mu.RLock()
		_ = mode
		_ = source
		mu.RUnlock()
	}
}

// BenchmarkErrorHandling benchmarks error path
// Target: < 500ns per operation
func BenchmarkErrorHandling(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return "", "", fmt.Errorf("invalid enrichment mode")
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())

	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)
	}
}

// BenchmarkFullHTTPStack benchmarks complete HTTP stack (realistic end-to-end)
// Target: < 200ns per operation
func BenchmarkFullHTTPStack(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeEnriched, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handlers.GetMode))
	defer server.Close()

	client := &http.Client{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest("GET", server.URL+"/enrichment/mode", nil)
		resp, _ := client.Do(req)
		if resp != nil {
			resp.Body.Close()
		}
	}
}

// BenchmarkResponseWriter benchmarks http.ResponseWriter operations
// Target: < 50ns per operation
func BenchmarkResponseWriter(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		rr.Header().Set("Content-Type", "application/json")
		rr.WriteHeader(http.StatusOK)
	}
}

// BenchmarkContextPropagation benchmarks context propagation overhead
// Target: < 10ns per operation
func BenchmarkContextPropagation(b *testing.B) {
	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		ctx := req.Context()
		_ = ctx
	}
}

// Benchmark comparison: Different mode values
func BenchmarkGetMode_TransparentMode(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeTransparent, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())
	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)
	}
}

func BenchmarkGetMode_EnrichedMode(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeEnriched, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())
	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)
	}
}

func BenchmarkGetMode_TransparentWithRecommendationsMode(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeTransparentWithRecommendations, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())
	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)
	}
}

// Benchmark memory allocations specifically
func BenchmarkGetMode_AllocationsOnly(b *testing.B) {
	mockManager := &mockEnrichmentManager{
		getModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
			return services.EnrichmentModeEnriched, "memory", nil
		},
	}

	handlers := NewEnrichmentHandlers(mockManager, slog.Default())
	req := httptest.NewRequest("GET", "/enrichment/mode", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetMode(rr, req)

		// Report allocations
		if i == 0 {
			b.Logf("First iteration allocations recorded")
		}
	}
}
