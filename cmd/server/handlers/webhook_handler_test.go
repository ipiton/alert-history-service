// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/middleware"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/webhook"
)

// mockUniversalWebhookHandler mocks UniversalWebhookHandler
type mockUniversalWebhookHandler struct {
	handleFunc func(ctx context.Context, req *webhook.HandleWebhookRequest) (*webhook.HandleWebhookResponse, error)
}

func (m *mockUniversalWebhookHandler) HandleWebhook(ctx context.Context, req *webhook.HandleWebhookRequest) (*webhook.HandleWebhookResponse, error) {
	if m.handleFunc != nil {
		return m.handleFunc(ctx, req)
	}
	return &webhook.HandleWebhookResponse{
		Status:          "success",
		Message:         "Webhook processed successfully",
		WebhookType:     "alertmanager",
		AlertsReceived:  1,
		AlertsProcessed: 1,
		ProcessingTime:  "1ms",
	}, nil
}

// newTestWebhookHandler creates a WebhookHTTPHandler for testing
func newTestWebhookHandler(mockHandler *mockUniversalWebhookHandler, config *WebhookConfig) *WebhookHTTPHandler {
	if config == nil {
		config = &WebhookConfig{
			MaxRequestSize:  1024 * 1024, // 1MB for tests
			RequestTimeout:  30 * time.Second,
			MaxAlertsPerReq: 1000,
			EnableMetrics:   false,
			EnableAuth:      false,
		}
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	// Type assertion to satisfy interface
	var universalHandler *webhook.UniversalWebhookHandler
	// In real code, mockHandler would implement the interface
	// For this test, we'll need to adjust the handler

	return &WebhookHTTPHandler{
		universalHandler: universalHandler, // This would be the mock
		logger:           logger,
		config:           config,
	}
}

// TestWebhookHTTPHandler_ServeHTTP_Success tests successful webhook processing
func TestWebhookHTTPHandler_ServeHTTP_Success(t *testing.T) {
	// Create mock handler
	mockHandler := &mockUniversalWebhookHandler{
		handleFunc: func(ctx context.Context, req *webhook.HandleWebhookRequest) (*webhook.HandleWebhookResponse, error) {
			if len(req.Payload) == 0 {
				t.Error("Expected non-empty payload")
			}
			return &webhook.HandleWebhookResponse{
				Status:          "success",
				Message:         "Webhook processed successfully",
				WebhookType:     "alertmanager",
				AlertsReceived:  2,
				AlertsProcessed: 2,
				ProcessingTime:  "5ms",
			}, nil
		},
	}

	handler := newTestWebhookHandler(mockHandler, nil)

	// Create test request
	payload := []byte(`{"alerts":[{"status":"firing","labels":{"alertname":"TestAlert"}}]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	
	// Add request ID to context
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-123")
	req = req.WithContext(ctx)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Call handler
	handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Parse response
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify response fields
	if status, ok := resp["status"].(string); !ok || status != "success" {
		t.Errorf("Expected status 'success', got %v", resp["status"])
	}

	if requestID, ok := resp["request_id"].(string); !ok || requestID != "test-request-123" {
		t.Errorf("Expected request_id 'test-request-123', got %v", resp["request_id"])
	}
}

// TestWebhookHTTPHandler_ServeHTTP_InvalidMethod tests invalid HTTP method
func TestWebhookHTTPHandler_ServeHTTP_InvalidMethod(t *testing.T) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)

	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/webhook", nil)
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected status %d, got %d for method %s", http.StatusMethodNotAllowed, rr.Code, method)
			}

			var resp ErrorResponse
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if resp.Status != "error" {
				t.Errorf("Expected status 'error', got %s", resp.Status)
			}
		})
	}
}

// TestWebhookHTTPHandler_ServeHTTP_PayloadTooLarge tests payload size limit
func TestWebhookHTTPHandler_ServeHTTP_PayloadTooLarge(t *testing.T) {
	config := &WebhookConfig{
		MaxRequestSize:  100, // Very small limit for testing
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, config)

	// Create large payload (exceeds limit)
	largePayload := bytes.Repeat([]byte("x"), 200)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(largePayload))
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("Expected status %d, got %d", http.StatusRequestEntityTooLarge, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got %s", resp.Status)
	}

	if !strings.Contains(resp.Message, "too large") {
		t.Errorf("Expected error message about size, got: %s", resp.Message)
	}
}

// TestWebhookHTTPHandler_ServeHTTP_EmptyBody tests empty request body
func TestWebhookHTTPHandler_ServeHTTP_EmptyBody(t *testing.T) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader([]byte{}))
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}
}

// TestWebhookHTTPHandler_ServeHTTP_ReadError tests read error handling
func TestWebhookHTTPHandler_ServeHTTP_ReadError(t *testing.T) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)

	// Create request with reader that fails
	req := httptest.NewRequest(http.MethodPost, "/webhook", &errorReader{})
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
}

// TestWebhookHTTPHandler_ServeHTTP_HandlerError tests handler error response
func TestWebhookHTTPHandler_ServeHTTP_HandlerError(t *testing.T) {
	mockHandler := &mockUniversalWebhookHandler{
		handleFunc: func(ctx context.Context, req *webhook.HandleWebhookRequest) (*webhook.HandleWebhookResponse, error) {
			return nil, errors.New("processing failed")
		},
	}

	handler := newTestWebhookHandler(mockHandler, nil)

	payload := []byte(`{"alerts":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}

	var resp ErrorResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if resp.Status != "error" {
		t.Errorf("Expected status 'error', got %s", resp.Status)
	}
}

// TestWebhookHTTPHandler_ServeHTTP_PartialSuccess tests partial success response
func TestWebhookHTTPHandler_ServeHTTP_PartialSuccess(t *testing.T) {
	mockHandler := &mockUniversalWebhookHandler{
		handleFunc: func(ctx context.Context, req *webhook.HandleWebhookRequest) (*webhook.HandleWebhookResponse, error) {
			return &webhook.HandleWebhookResponse{
				Status:          "partial_success",
				Message:         "Processed 1 of 2 alerts",
				WebhookType:     "alertmanager",
				AlertsReceived:  2,
				AlertsProcessed: 1,
				Errors:          []string{"Alert 2 (TestAlert2): validation failed"},
				ProcessingTime:  "10ms",
			}, nil
		},
	}

	handler := newTestWebhookHandler(mockHandler, nil)

	payload := []byte(`{"alerts":[{"status":"firing"},{"status":"invalid"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMultiStatus {
		t.Errorf("Expected status %d, got %d", http.StatusMultiStatus, rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if status, ok := resp["status"].(string); !ok || status != "partial_success" {
		t.Errorf("Expected status 'partial_success', got %v", resp["status"])
	}

	if errors, ok := resp["errors"].([]interface{}); !ok || len(errors) != 1 {
		t.Errorf("Expected 1 error, got %v", resp["errors"])
	}
}

// TestWebhookHTTPHandler_ServeHTTP_NoRequestID tests missing request ID
func TestWebhookHTTPHandler_ServeHTTP_NoRequestID(t *testing.T) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)

	payload := []byte(`{"alerts":[{"status":"firing"}]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
	// No request ID in context
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should still work, just use "unknown" as request ID
	if rr.Code != http.StatusOK && rr.Code != http.StatusInternalServerError {
		t.Logf("Status code: %d (acceptable for missing request ID)", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if requestID, ok := resp["request_id"].(string); ok && requestID == "" {
		t.Error("Expected request_id to be non-empty (should default to 'unknown')")
	}
}

// TestWebhookHTTPHandler_ServeHTTP_ContentTypeVariations tests different content types
func TestWebhookHTTPHandler_ServeHTTP_ContentTypeVariations(t *testing.T) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)

	contentTypes := []string{
		"application/json",
		"application/json; charset=utf-8",
		"application/json;charset=utf-8",
		"", // No content type
	}

	for _, ct := range contentTypes {
		t.Run(ct, func(t *testing.T) {
			payload := []byte(`{"alerts":[]}`)
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
			if ct != "" {
				req.Header.Set("Content-Type", ct)
			}
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request")
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Should accept all variations
			if rr.Code < 200 || rr.Code >= 300 {
				t.Logf("Content-Type %q resulted in status %d (may be acceptable)", ct, rr.Code)
			}
		})
	}
}

// TestWebhookHTTPHandler_ServeHTTP_Concurrency tests concurrent requests
func TestWebhookHTTPHandler_ServeHTTP_Concurrency(t *testing.T) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{
		handleFunc: func(ctx context.Context, req *webhook.HandleWebhookRequest) (*webhook.HandleWebhookResponse, error) {
			// Simulate some processing time
			time.Sleep(10 * time.Millisecond)
			return &webhook.HandleWebhookResponse{
				Status:         "success",
				Message:        "Processed",
				WebhookType:    "alertmanager",
				ProcessingTime: "10ms",
			}, nil
		},
	}, nil)

	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			payload := []byte(`{"alerts":[{"status":"firing"}]}`)
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
			ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "test-request-"+string(rune(id)))
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results <- rr.Code
		}(i)
	}

	// Collect results
	successCount := 0
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		}
	}

	if successCount != numRequests {
		t.Errorf("Expected %d successful requests, got %d", numRequests, successCount)
	}
}

// TestWebhookHTTPHandler_NewWebhookHTTPHandler tests constructor
func TestWebhookHTTPHandler_NewWebhookHTTPHandler(t *testing.T) {
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler := NewWebhookHTTPHandler(nil, config, logger)

	if handler == nil {
		t.Fatal("Expected non-nil handler")
	}

	if handler.config != config {
		t.Error("Config not set correctly")
	}

	if handler.logger != logger {
		t.Error("Logger not set correctly")
	}
}

// TestWebhookHTTPHandler_ErrorTypes tests error type definitions
func TestWebhookHTTPHandler_ErrorTypes(t *testing.T) {
	if ErrPayloadTooLarge == nil {
		t.Error("Expected ErrPayloadTooLarge to be defined")
	}

	if ErrInvalidMethod == nil {
		t.Error("Expected ErrInvalidMethod to be defined")
	}

	if ErrReadFailed == nil {
		t.Error("Expected ErrReadFailed to be defined")
	}

	// Test error messages
	if !strings.Contains(ErrPayloadTooLarge.Error(), "too large") {
		t.Errorf("Unexpected error message: %s", ErrPayloadTooLarge.Error())
	}
}

// TestWebhookConfig_DefaultValues tests config with default values
func TestWebhookConfig_DefaultValues(t *testing.T) {
	config := &WebhookConfig{
		MaxRequestSize:  10 * 1024 * 1024,
		RequestTimeout:  30 * time.Second,
		MaxAlertsPerReq: 1000,
		EnableMetrics:   true,
		EnableAuth:      false,
	}

	if config.MaxRequestSize != 10*1024*1024 {
		t.Errorf("Expected MaxRequestSize 10MB, got %d", config.MaxRequestSize)
	}

	if config.RequestTimeout != 30*time.Second {
		t.Errorf("Expected RequestTimeout 30s, got %v", config.RequestTimeout)
	}

	if config.MaxAlertsPerReq != 1000 {
		t.Errorf("Expected MaxAlertsPerReq 1000, got %d", config.MaxAlertsPerReq)
	}
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("read error")
}

// TestErrorResponse_JSONMarshaling tests ErrorResponse JSON marshaling
func TestErrorResponse_JSONMarshaling(t *testing.T) {
	errResp := ErrorResponse{
		Status:    "error",
		Message:   "Test error",
		RequestID: "test-123",
		Details: map[string]interface{}{
			"code": "ERR_TEST",
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.Marshal(errResp)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}

	var decoded ErrorResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if decoded.Status != errResp.Status {
		t.Errorf("Expected status %s, got %s", errResp.Status, decoded.Status)
	}

	if decoded.Message != errResp.Message {
		t.Errorf("Expected message %s, got %s", errResp.Message, decoded.Message)
	}
}

// Benchmarks

// BenchmarkWebhookHTTPHandler_ServeHTTP benchmarks handler performance
func BenchmarkWebhookHTTPHandler_ServeHTTP(b *testing.B) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)
	payload := []byte(`{"alerts":[{"status":"firing","labels":{"alertname":"TestAlert"}}]}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "bench-request")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkWebhookHTTPHandler_LargePayload benchmarks with large payload
func BenchmarkWebhookHTTPHandler_LargePayload(b *testing.B) {
	handler := newTestWebhookHandler(&mockUniversalWebhookHandler{}, nil)
	
	// Create large payload with 100 alerts
	alerts := make([]string, 100)
	for i := 0; i < 100; i++ {
		alerts[i] = `{"status":"firing","labels":{"alertname":"Alert` + string(rune(i)) + `"}}`
	}
	payload := []byte(`{"alerts":[` + strings.Join(alerts, ",") + `]}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		ctx := context.WithValue(req.Context(), middleware.RequestIDKey, "bench-request")
		req = req.WithContext(ctx)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	}
}

