// Package handlers provides HTTP handlers for the Alert History Service.
// Performance benchmarks for webhook endpoint optimization.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// BenchmarkWebhookHandler_Baseline establishes performance baseline
func BenchmarkWebhookHandler_Baseline(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	processor := &mockAlertProcessor{}
	
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	
	handler := NewWebhookHTTPHandler(nil, config, logger)
	
	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkWebhookHandler_WithMiddleware benchmarks full stack
func BenchmarkWebhookHandler_WithMiddleware(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	processor := &mockAlertProcessor{}
	
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	
	handler := NewWebhookHTTPHandler(nil, config, logger)
	
	// Full middleware stack
	recovery := middleware.NewRecoveryMiddleware(logger)
	requestID := middleware.NewRequestIDMiddleware(logger)
	logging := middleware.LoggingMiddleware(logger)
	
	fullHandler := middleware.Chain(
		recovery.Middleware,
		requestID.Middleware,
		logging,
	)(handler)
	
	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		
		fullHandler.ServeHTTP(rr, req)
	}
}

// BenchmarkWebhookHandler_PayloadSizes benchmarks different payload sizes
func BenchmarkWebhookHandler_PayloadSizes(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	
	handler := NewWebhookHTTPHandler(nil, config, logger)
	
	payloadSizes := []struct {
		name   string
		alerts int
	}{
		{"1_alert", 1},
		{"5_alerts", 5},
		{"10_alerts", 10},
		{"50_alerts", 50},
		{"100_alerts", 100},
		{"500_alerts", 500},
	}
	
	for _, ps := range payloadSizes {
		b.Run(ps.name, func(b *testing.B) {
			// Generate payload
			alerts := make([]string, ps.alerts)
			for i := 0; i < ps.alerts; i++ {
				alerts[i] = `{"status":"firing","labels":{"alertname":"Test","instance":"server-` + 
					string(rune(i)) + `"}}`
			}
			payload := `{"alerts":[` + strings.Join(alerts, ",") + `]}`
			
			b.SetBytes(int64(len(payload)))
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
				req.Header.Set("Content-Type", "application/json")
				rr := httptest.NewRecorder()
				
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkWebhookHandler_Concurrent benchmarks concurrent performance
func BenchmarkWebhookHandler_Concurrent(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	
	handler := NewWebhookHTTPHandler(nil, config, logger)
	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
	
	concurrencyLevels := []int{1, 10, 50, 100}
	
	for _, conc := range concurrencyLevels {
		b.Run(string(rune(conc))+"_goroutines", func(b *testing.B) {
			b.SetParallelism(conc)
			b.ReportAllocs()
			b.ResetTimer()
			
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
					req.Header.Set("Content-Type", "application/json")
					rr := httptest.NewRecorder()
					
					handler.ServeHTTP(rr, req)
				}
			})
		})
	}
}

// BenchmarkWebhookHandler_MemoryProfile benchmarks memory allocations
func BenchmarkWebhookHandler_MemoryProfile(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	
	handler := NewWebhookHTTPHandler(nil, config, logger)
	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
	
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		
		handler.ServeHTTP(rr, req)
	}
	
	b.StopTimer()
	runtime.GC()
	runtime.ReadMemStats(&m2)
	
	allocPerOp := float64(m2.TotalAlloc-m1.TotalAlloc) / float64(b.N)
	b.ReportMetric(allocPerOp, "bytes/op")
}

// BenchmarkMiddleware_Individual benchmarks each middleware separately
func BenchmarkMiddleware_Individual(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	
	baseHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	middlewares := []struct {
		name string
		mw   middleware.Middleware
	}{
		{"recovery", middleware.NewRecoveryMiddleware(logger).Middleware},
		{"request_id", middleware.NewRequestIDMiddleware(logger).Middleware},
		{"logging", middleware.LoggingMiddleware(logger)},
	}
	
	for _, mw := range middlewares {
		b.Run(mw.name, func(b *testing.B) {
			handler := mw.mw(baseHandler)
			
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				rr := httptest.NewRecorder()
				
				handler.ServeHTTP(rr, req)
			}
		})
	}
}

// BenchmarkJSONParsing benchmarks JSON parsing performance
func BenchmarkJSONParsing(b *testing.B) {
	payloads := []struct {
		name string
		json string
	}{
		{
			"small",
			`{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`,
		},
		{
			"medium",
			`{"alerts":[` + strings.Repeat(`{"status":"firing","labels":{"alertname":"Test"}},`, 9) +
				`{"status":"firing","labels":{"alertname":"Test"}}]}`,
		},
		{
			"large",
			`{"alerts":[` + strings.Repeat(`{"status":"firing","labels":{"alertname":"Test"}},`, 99) +
				`{"status":"firing","labels":{"alertname":"Test"}}]}`,
		},
	}
	
	for _, payload := range payloads {
		b.Run(payload.name, func(b *testing.B) {
			b.SetBytes(int64(len(payload.json)))
			b.ReportAllocs()
			b.ResetTimer()
			
			for i := 0; i < b.N; i++ {
				var data map[string]interface{}
				if err := json.Unmarshal([]byte(payload.json), &data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRequestIDGeneration benchmarks UUID generation
func BenchmarkRequestIDGeneration(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := middleware.NewRequestIDMiddleware(logger)
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})
	
	fullHandler := requestID.Middleware(handler)
	
	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		
		fullHandler.ServeHTTP(rr, req)
	}
}

// BenchmarkResponseWriter benchmarks response writer wrapping
func BenchmarkResponseWriter(b *testing.B) {
	b.Run("direct_write", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			rr := httptest.NewRecorder()
			rr.WriteHeader(http.StatusOK)
			rr.Write([]byte("test"))
		}
	})
	
	b.Run("wrapped_write", func(b *testing.B) {
		type responseWriter struct {
			http.ResponseWriter
			statusCode int
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			rr := httptest.NewRecorder()
			wrapped := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}
			wrapped.WriteHeader(http.StatusOK)
			wrapped.Write([]byte("test"))
		}
	})
}

// BenchmarkBufferPooling benchmarks buffer reuse
func BenchmarkBufferPooling(b *testing.B) {
	payload := bytes.Repeat([]byte("x"), 1024) // 1KB
	
	b.Run("no_pooling", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			buf := new(bytes.Buffer)
			buf.Write(payload)
			_ = buf.Bytes()
		}
	})
	
	b.Run("with_pooling", func(b *testing.B) {
		pool := sync.Pool{
			New: func() interface{} {
				return new(bytes.Buffer)
			},
		}
		
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			buf := pool.Get().(*bytes.Buffer)
			buf.Reset()
			buf.Write(payload)
			_ = buf.Bytes()
			pool.Put(buf)
		}
	})
}

// BenchmarkContextOperations benchmarks context operations
func BenchmarkContextOperations(b *testing.B) {
	b.Run("context_with_value", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			ctx := context.Background()
			ctx = context.WithValue(ctx, "key", "value")
			_ = ctx.Value("key")
		}
	})
	
	b.Run("context_with_cancel", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			_ = ctx
		}
	})
	
	b.Run("context_with_timeout", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			cancel()
			_ = ctx
		}
	})
}

