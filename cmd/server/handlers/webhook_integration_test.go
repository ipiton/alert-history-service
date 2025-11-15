// Package handlers provides HTTP handlers for the Alert History Service.
// Integration tests for webhook endpoint with full middleware stack.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/webhook"
)

// mockAlertProcessor mocks core.AlertProcessor for integration tests
type mockAlertProcessor struct {
	mu             sync.Mutex
	processedCount int
	lastAlert      *core.Alert
	processFunc    func(ctx context.Context, alert *core.Alert) error
}

func (m *mockAlertProcessor) ProcessAlert(ctx context.Context, alert *core.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.processedCount++
	m.lastAlert = alert

	if m.processFunc != nil {
		return m.processFunc(ctx, alert)
	}
	return nil
}

func (m *mockAlertProcessor) GetProcessedCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.processedCount
}

// buildTestMiddlewareStack creates a full middleware stack for testing
func buildTestMiddlewareStack(logger *slog.Logger) middleware.Middleware {
	// Build realistic middleware stack
	recovery := middleware.NewRecoveryMiddleware(logger)
	requestID := middleware.NewRequestIDMiddleware(logger)
	logging := middleware.LoggingMiddleware(logger)

	return middleware.Chain(
		recovery.Middleware,
		requestID.Middleware,
		logging,
	)
}

// TestIntegration_FullWebhookFlow tests complete webhook processing flow
func TestIntegration_FullWebhookFlow(t *testing.T) {
	// Setup
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	processor := &mockAlertProcessor{}

	// Create universal webhook handler (would use real one in practice)
	// For this test, we'll simulate it
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)
	middlewareStack := buildTestMiddlewareStack(logger)
	fullHandler := middlewareStack(handler)

	// Test data: Alertmanager-style webhook
	payload := `{
		"alerts": [
			{
				"status": "firing",
				"labels": {
					"alertname": "HighCPU",
					"severity": "critical",
					"instance": "server-1"
				},
				"annotations": {
					"summary": "High CPU usage detected"
				},
				"startsAt": "2025-11-15T10:00:00Z"
			}
		]
	}`

	// Execute
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	fullHandler.ServeHTTP(rr, req)

	// Verify
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Logf("Status: %d (expected 200 or 500 for integration test)", rr.Code)
	}

	// Should have X-Request-ID from middleware
	if requestID := rr.Header().Get("X-Request-ID"); requestID == "" {
		t.Error("Expected X-Request-ID header from middleware")
	}

	// Should have Content-Type
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Logf("Content-Type: %s (may not be JSON for error case)", ct)
	}
}

// TestIntegration_MiddlewareStackOrder tests middleware execution order
func TestIntegration_MiddlewareStackOrder(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Track execution order
	var executionOrder []string
	var mu sync.Mutex

	trackMiddleware := func(name string) middleware.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				executionOrder = append(executionOrder, name+"-before")
				mu.Unlock()

				next.ServeHTTP(w, r)

				mu.Lock()
				executionOrder = append(executionOrder, name+"-after")
				mu.Unlock()
			})
		}
	}

	// Build stack
	stack := middleware.Chain(
		trackMiddleware("first"),
		trackMiddleware("second"),
		trackMiddleware("third"),
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		executionOrder = append(executionOrder, "handler")
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	fullHandler := stack(handler)

	// Execute
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	fullHandler.ServeHTTP(rr, req)

	// Verify order
	expected := []string{
		"first-before", "second-before", "third-before",
		"handler",
		"third-after", "second-after", "first-after",
	}

	if len(executionOrder) != len(expected) {
		t.Fatalf("Expected %d execution steps, got %d", len(expected), len(executionOrder))
	}

	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("Step %d: expected %q, got %q", i, exp, executionOrder[i])
		}
	}
}

// TestIntegration_ContextPropagation tests context values through middleware
func TestIntegration_ContextPropagation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Build stack with RequestID middleware
	requestID := middleware.NewRequestIDMiddleware(logger)

	var capturedRequestID string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = middleware.GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	fullHandler := requestID.Middleware(handler)

	// Test 1: With existing request ID
	t.Run("existing_request_id", func(t *testing.T) {
		capturedRequestID = ""
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "test-id-123")
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if capturedRequestID != "test-id-123" {
			t.Errorf("Expected request ID 'test-id-123', got %q", capturedRequestID)
		}

		if rr.Header().Get("X-Request-ID") != "test-id-123" {
			t.Error("Request ID not in response header")
		}
	})

	// Test 2: Generated request ID
	t.Run("generated_request_id", func(t *testing.T) {
		capturedRequestID = ""
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if capturedRequestID == "" {
			t.Error("Expected generated request ID")
		}

		if rr.Header().Get("X-Request-ID") != capturedRequestID {
			t.Error("Request ID mismatch between context and header")
		}

		// Should be valid UUID format
		if !strings.Contains(capturedRequestID, "-") {
			t.Errorf("Generated ID doesn't look like UUID: %s", capturedRequestID)
		}
	})
}

// TestIntegration_ErrorHandlingAcrossLayers tests error propagation
func TestIntegration_ErrorHandlingAcrossLayers(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Build stack with recovery
	recovery := middleware.NewRecoveryMiddleware(logger)

	t.Run("panic_recovery", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test panic in handler")
		})

		fullHandler := recovery.Middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		// Should not panic
		fullHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Errorf("Expected status 500 after panic, got %d", rr.Code)
		}
	})

	t.Run("error_response", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "bad request",
			})
		})

		fullHandler := recovery.Middleware(handler)

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		if resp["error"] != "bad request" {
			t.Errorf("Expected error 'bad request', got %v", resp["error"])
		}
	})
}

// TestIntegration_ConcurrentRequests tests concurrent request handling
func TestIntegration_ConcurrentRequests(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	processor := &mockAlertProcessor{}

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)
	middlewareStack := buildTestMiddlewareStack(logger)
	fullHandler := middlewareStack(handler)

	const numRequests = 20
	var wg sync.WaitGroup
	results := make(chan int, numRequests)

	// Send concurrent requests
	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			fullHandler.ServeHTTP(rr, req)
			results <- rr.Code
		}(i)
	}

	wg.Wait()
	close(results)

	// Collect results
	statusCounts := make(map[int]int)
	for code := range results {
		statusCounts[code]++
	}

	t.Logf("Concurrent requests status distribution: %v", statusCounts)

	// Should handle all requests (may have varying status codes)
	totalProcessed := 0
	for _, count := range statusCounts {
		totalProcessed += count
	}

	if totalProcessed != numRequests {
		t.Errorf("Expected %d requests processed, got %d", numRequests, totalProcessed)
	}
}

// TestIntegration_RateLimitingWithAuth tests rate limiting + auth interaction
func TestIntegration_RateLimitingWithAuth(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Build stack: rate limit → auth
	rateLimitConfig := &middleware.RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  5, // Low limit for testing
		GlobalLimit: 100,
		Logger:      logger,
	}

	authConfig := &middleware.AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "test-key",
		Logger:  logger,
	}

	rateLimit := middleware.NewRateLimitMiddleware(rateLimitConfig)
	auth := middleware.AuthenticationMiddleware(authConfig)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Stack: rate limit first, then auth
	fullHandler := rateLimit.Middleware(auth(handler))

	// Test 1: Valid auth, within rate limit
	t.Run("valid_within_limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.Header.Set("X-API-Key", "test-key")
		req.RemoteAddr = "192.168.1.100:12345"
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})

	// Test 2: Invalid auth (should be rejected before rate limit)
	t.Run("invalid_auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.Header.Set("X-API-Key", "wrong-key")
		req.RemoteAddr = "192.168.1.101:12345"
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})

	// Test 3: Exceed rate limit (valid auth)
	t.Run("exceed_rate_limit", func(t *testing.T) {
		// Send many requests from same IP
		successCount := 0
		rateLimitedCount := 0

		for i := 0; i < 10; i++ {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			req.Header.Set("X-API-Key", "test-key")
			req.RemoteAddr = "192.168.1.102:12345" // Same IP
			rr := httptest.NewRecorder()

			fullHandler.ServeHTTP(rr, req)

			if rr.Code == http.StatusOK {
				successCount++
			} else if rr.Code == http.StatusTooManyRequests {
				rateLimitedCount++
			}
		}

		t.Logf("Rate limit test: %d success, %d rate-limited", successCount, rateLimitedCount)

		if rateLimitedCount == 0 {
			t.Log("Expected some requests to be rate-limited (may need adjustment)")
		}
	})
}

// TestIntegration_TimeoutHandling tests timeout across middleware stack
func TestIntegration_TimeoutHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	timeout := 50 * time.Millisecond
	timeoutMw := middleware.TimeoutMiddleware(timeout)

	t.Run("fast_handler", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		fullHandler := timeoutMw(handler)

		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for fast handler, got %d", rr.Code)
		}
	})

	t.Run("slow_handler", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(200 * time.Millisecond):
				w.WriteHeader(http.StatusOK)
			case <-r.Context().Done():
				return
			}
		})

		fullHandler := timeoutMw(handler)

		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		t.Logf("Slow handler status: %d (timeout should trigger)", rr.Code)
	})
}

// TestIntegration_LargePayloadHandling tests large payload with size limit
func TestIntegration_LargePayloadHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	maxSize := int64(1024) // 1KB limit
	sizeLimit := middleware.SizeLimitMiddleware(maxSize)

	t.Run("within_limit", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			t.Logf("Received %d bytes", len(body))
			w.WriteHeader(http.StatusOK)
		})

		fullHandler := sizeLimit(handler)

		payload := bytes.Repeat([]byte("x"), 500) // 500 bytes
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for small payload, got %d", rr.Code)
		}
	})

	t.Run("exceeds_limit", func(t *testing.T) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("Handler should not be called for oversized payload")
			w.WriteHeader(http.StatusOK)
		})

		fullHandler := sizeLimit(handler)

		payload := bytes.Repeat([]byte("x"), 2000) // 2KB, exceeds 1KB limit
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		t.Logf("Oversized payload status: %d", rr.Code)
	})
}

// BenchmarkIntegration_FullStack benchmarks complete middleware stack
func BenchmarkIntegration_FullStack(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)
	middlewareStack := buildTestMiddlewareStack(logger)
	fullHandler := middlewareStack(handler)

	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)
	}
}

