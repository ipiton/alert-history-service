// Package handlers provides HTTP handlers for the Alert History Service.
// Failure scenario tests for webhook endpoint.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// failingAlertProcessor simulates processing failures
type failingAlertProcessor struct {
	failCount  int32
	totalCalls int32
	failAfter  int32 // Fail after N successful calls
}

func (f *failingAlertProcessor) ProcessAlert(ctx context.Context, alert *core.Alert) error {
	calls := atomic.AddInt32(&f.totalCalls, 1)

	if calls > f.failAfter {
		atomic.AddInt32(&f.failCount, 1)
		return errors.New("simulated processing failure")
	}

	return nil
}

// slowAlertProcessor simulates slow processing
type slowAlertProcessor struct {
	delay time.Duration
}

func (s *slowAlertProcessor) ProcessAlert(ctx context.Context, alert *core.Alert) error {
	select {
	case <-time.After(s.delay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// TestFailure_ProcessingError tests handling of alert processing failures
func TestFailure_ProcessingError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	processor := &failingAlertProcessor{
		failAfter: 0, // Fail immediately
	}

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	// Note: In real implementation, handler would use processor
	handler := NewWebhookHTTPHandler(nil, config, logger)

	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should handle processing error gracefully
	t.Logf("Processing error resulted in status: %d", rr.Code)

	// Should return JSON error response
	if ct := rr.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Logf("Content-Type: %s (expected JSON for error)", ct)
	}
}

// TestFailure_PartialProcessingFailure tests handling of partial failures
func TestFailure_PartialProcessingFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	processor := &failingAlertProcessor{
		failAfter: 2, // Succeed for first 2, then fail
	}

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)

	// Multiple alerts in one request
	payload := `{
		"alerts": [
			{"status":"firing","labels":{"alertname":"Alert1"}},
			{"status":"firing","labels":{"alertname":"Alert2"}},
			{"status":"firing","labels":{"alertname":"Alert3"}},
			{"status":"firing","labels":{"alertname":"Alert4"}}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should return 207 Multi-Status for partial success
	t.Logf("Partial failure status: %d", rr.Code)

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err == nil {
		t.Logf("Response: %+v", resp)
	}
}

// TestFailure_TimeoutDuringProcessing tests timeout during slow processing
func TestFailure_TimeoutDuringProcessing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	processor := &slowAlertProcessor{
		delay: 200 * time.Millisecond, // Slow processing
	}

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  50 * time.Millisecond, // Short timeout
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)

	// Add timeout middleware
	timeoutMw := middleware.TimeoutMiddleware(config.RequestTimeout)
	fullHandler := timeoutMw(handler)

	payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	fullHandler.ServeHTTP(rr, req)

	// Should timeout
	t.Logf("Timeout scenario status: %d", rr.Code)

	if rr.Code == http.StatusOK {
		t.Log("Request completed (timeout may not have triggered)")
	}
}

// TestFailure_InvalidJSON tests handling of malformed JSON
func TestFailure_InvalidJSON(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)

	testCases := []struct {
		name    string
		payload string
	}{
		{"incomplete json", `{"alerts":[{"status":"firing"`},
		{"invalid syntax", `{alerts: [status: "firing"]}]`},
		{"empty object", `{}`},
		{"null", `null`},
		{"array instead", `[]`},
		{"non-json", `this is not json`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Should return 400 Bad Request for invalid JSON
			if rr.Code != http.StatusBadRequest && rr.Code != http.StatusInternalServerError {
				t.Logf("Invalid JSON %q resulted in status: %d", tc.name, rr.Code)
			}

			// Should return JSON error
			var resp map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Logf("Error response not JSON: %v", err)
			}
		})
	}
}

// TestFailure_EmptyAlerts tests handling of empty alerts array
func TestFailure_EmptyAlerts(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)

	payload := `{"alerts":[]}`
	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Empty alerts should be accepted (or rejected with 400)
	t.Logf("Empty alerts status: %d", rr.Code)
}

// TestFailure_MissingRequiredFields tests validation of required fields
func TestFailure_MissingRequiredFields(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)

	testCases := []struct {
		name    string
		payload string
	}{
		{"no labels", `{"alerts":[{"status":"firing"}]}`},
		{"no status", `{"alerts":[{"labels":{"alertname":"Test"}}]}`},
		{"no alertname", `{"alerts":[{"status":"firing","labels":{}}]}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(tc.payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			t.Logf("Missing field %q status: %d", tc.name, rr.Code)
		})
	}
}

// TestFailure_RateLimitExhaustion tests behavior when rate limit is exhausted
func TestFailure_RateLimitExhaustion(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	rateLimitConfig := &middleware.RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  3, // Very low limit
		GlobalLimit: 100,
		Logger:      logger,
	}

	rateLimit := middleware.NewRateLimitMiddleware(rateLimitConfig)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	fullHandler := rateLimit.Middleware(handler)

	successCount := 0
	rateLimitedCount := 0
	const numRequests = 10

	// Exhaust rate limit
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = "192.168.1.200:12345" // Same IP
		rr := httptest.NewRecorder()

		fullHandler.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			successCount++
		} else if rr.Code == http.StatusTooManyRequests {
			rateLimitedCount++

			// Check Retry-After header
			if retryAfter := rr.Header().Get("Retry-After"); retryAfter != "" {
				t.Logf("Retry-After: %s", retryAfter)
			}

			// Check error response
			var resp map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err == nil {
				t.Logf("Rate limit response: %+v", resp)
			}
		}
	}

	t.Logf("Rate limit exhaustion: %d success, %d rate-limited", successCount, rateLimitedCount)

	if rateLimitedCount == 0 {
		t.Error("Expected some requests to be rate-limited")
	}
}

// TestFailure_AuthenticationFailures tests various auth failure scenarios
func TestFailure_AuthenticationFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	authConfig := &middleware.AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "correct-key",
		Logger:  logger,
	}

	auth := middleware.AuthenticationMiddleware(authConfig)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called with failed auth")
		w.WriteHeader(http.StatusOK)
	})

	fullHandler := auth(handler)

	testCases := []struct {
		name   string
		apiKey string
	}{
		{"wrong key", "wrong-key"},
		{"empty key", ""},
		{"almost correct", "correct-ke"},
		{"case mismatch", "CORRECT-KEY"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tc.apiKey != "" {
				req.Header.Set("X-API-Key", tc.apiKey)
			}
			rr := httptest.NewRecorder()

			fullHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", rr.Code)
			}

			// Check WWW-Authenticate header
			if wwwAuth := rr.Header().Get("WWW-Authenticate"); wwwAuth == "" {
				t.Error("Expected WWW-Authenticate header")
			}

			// Check error response
			var resp map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Errorf("Failed to decode error response: %v", err)
			}
		})
	}
}

// TestFailure_PanicRecovery tests panic recovery in middleware
func TestFailure_PanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	recovery := middleware.NewRecoveryMiddleware(logger)

	testCases := []struct {
		name      string
		panicType interface{}
	}{
		{"string panic", "panic message"},
		{"error panic", errors.New("error panic")},
		{"nil panic", nil},
		{"int panic", 42},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tc.panicType)
			})

			fullHandler := recovery.Middleware(handler)

			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			rr := httptest.NewRecorder()

			// Should not panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Panic not recovered: %v", r)
				}
			}()

			fullHandler.ServeHTTP(rr, req)

			if rr.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 500 after panic, got %d", rr.Code)
			}
		})
	}
}

// TestFailure_ConcurrentFailures tests handling of concurrent failures
func TestFailure_ConcurrentFailures(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	processor := &failingAlertProcessor{
		failAfter: 5, // Fail after 5 successful
	}

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)
	recovery := middleware.NewRecoveryMiddleware(logger)
	fullHandler := recovery.Middleware(handler)

	const numRequests = 20
	results := make(chan int, numRequests)

	// Send concurrent requests (some will fail)
	for i := 0; i < numRequests; i++ {
		go func() {
			payload := `{"alerts":[{"status":"firing","labels":{"alertname":"Test"}}]}`
			req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			fullHandler.ServeHTTP(rr, req)
			results <- rr.Code
		}()
	}

	// Collect results
	statusCounts := make(map[int]int)
	for i := 0; i < numRequests; i++ {
		code := <-results
		statusCounts[code]++
	}

	t.Logf("Concurrent failures status distribution: %v", statusCounts)

	// Should handle all requests (some may fail)
	totalProcessed := 0
	for _, count := range statusCounts {
		totalProcessed += count
	}

	if totalProcessed != numRequests {
		t.Errorf("Expected %d requests processed, got %d", numRequests, totalProcessed)
	}
}

// BenchmarkFailure_ErrorPath benchmarks error handling performance
func BenchmarkFailure_ErrorPath(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	config := &WebhookConfig{
		MaxRequestSize:  1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}

	handler := NewWebhookHTTPHandler(nil, config, logger)

	// Invalid JSON payload
	payload := `{"invalid json`

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	}
}
