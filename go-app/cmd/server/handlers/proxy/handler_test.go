package proxy

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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProxyWebhookService is a mock implementation of the proxy webhook service
type MockProxyWebhookService struct {
	mock.Mock
}

func (m *MockProxyWebhookService) ProcessWebhook(ctx context.Context, req *ProxyWebhookRequest) (*ProxyWebhookResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ProxyWebhookResponse), args.Error(1)
}

// TestNewProxyWebhookHTTPHandler tests handler creation
func TestNewProxyWebhookHTTPHandler(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)

	require.NoError(t, err)
	require.NotNil(t, handler)
	assert.NotNil(t, handler.service)
	assert.NotNil(t, handler.config)
	assert.NotNil(t, handler.logger)
}

// TestNewProxyWebhookHTTPHandler_NilService tests handler creation with nil service
func TestNewProxyWebhookHTTPHandler_NilService(t *testing.T) {
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(nil, config, logger)

	assert.Error(t, err)
	assert.Nil(t, handler)
	assert.Contains(t, err.Error(), "service is required")
}

// TestProxyWebhookHTTPHandler_ServeHTTP_Success tests successful request processing
func TestProxyWebhookHTTPHandler_ServeHTTP_Success(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	// Create valid request
	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":  "warning",
				},
				Annotations: map[string]string{
					"summary": "Test alert",
				},
				StartsAt: time.Now(),
			},
		},
	}

	// Expected response
	expectedResp := &ProxyWebhookResponse{
		Status:    "success",
		Message:   "All alerts processed successfully",
		Timestamp: time.Now(),
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived:  1,
			TotalProcessed: 1,
		},
		ProcessingTime: 100 * time.Millisecond,
	}

	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).Return(expectedResp, nil)

	// Create HTTP request
	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response ProxyWebhookResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, 1, response.AlertsSummary.TotalReceived)

	mockService.AssertExpectations(t)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_PartialSuccess tests partial success scenario
func TestProxyWebhookHTTPHandler_ServeHTTP_PartialSuccess(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert1",
				},
				StartsAt: time.Now(),
			},
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert2",
				},
				StartsAt: time.Now(),
			},
		},
	}

	expectedResp := &ProxyWebhookResponse{
		Status:    "partial",
		Message:   "1 of 2 alerts failed",
		Timestamp: time.Now(),
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived:  2,
			TotalProcessed: 1,
			TotalFailed:    1,
		},
		ProcessingTime: 200 * time.Millisecond,
	}

	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).Return(expectedResp, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Partial success should return 207 Multi-Status
	assert.Equal(t, http.StatusMultiStatus, w.Code)

	var response ProxyWebhookResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "partial", response.Status)
	assert.Equal(t, 2, response.AlertsSummary.TotalReceived)
	assert.Equal(t, 1, response.AlertsSummary.TotalFailed)

	mockService.AssertExpectations(t)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_MethodNotAllowed tests invalid HTTP method
func TestProxyWebhookHTTPHandler_ServeHTTP_MethodNotAllowed(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	// Test GET method (should fail)
	req := httptest.NewRequest(http.MethodGet, "/webhook/proxy", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	var errResp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, string(ErrCodeValidation), errResp.Error)
	assert.Contains(t, errResp.Message, "POST")
}

// TestProxyWebhookHTTPHandler_ServeHTTP_InvalidContentType tests invalid content type
func TestProxyWebhookHTTPHandler_ServeHTTP_InvalidContentType(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", strings.NewReader("test"))
	req.Header.Set("Content-Type", "text/plain") // Invalid
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)

	var errResp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, string(ErrCodeUnsupportedMediaType), errResp.Error)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_InvalidJSON tests invalid JSON payload
func TestProxyWebhookHTTPHandler_ServeHTTP_InvalidJSON(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", strings.NewReader("{invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, string(ErrCodeValidation), errResp.Error)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_EmptyBody tests empty request body
func TestProxyWebhookHTTPHandler_ServeHTTP_EmptyBody(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", nil)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_PayloadTooLarge tests oversized payload
func TestProxyWebhookHTTPHandler_ServeHTTP_PayloadTooLarge(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	config.MaxRequestSize = 100 // 100 bytes limit
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	// Create large payload (> 100 bytes)
	largePayload := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts:   make([]AlertPayload, 100),
	}

	jsonBody, _ := json.Marshal(largePayload)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)

	var errResp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, string(ErrCodePayloadTooLarge), errResp.Error)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_ValidationError tests validation errors
func TestProxyWebhookHTTPHandler_ServeHTTP_ValidationError(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	tests := []struct {
		name    string
		request ProxyWebhookRequest
	}{
		{
			name: "missing receiver",
			request: ProxyWebhookRequest{
				Receiver: "", // Invalid
				Status:   "firing",
				Alerts: []AlertPayload{
					{
						Status: "firing",
						Labels: map[string]string{"alertname": "Test"},
						StartsAt: time.Now(),
					},
				},
			},
		},
		{
			name: "missing status",
			request: ProxyWebhookRequest{
				Receiver: "webhook-receiver",
				Status:   "", // Invalid
				Alerts: []AlertPayload{
					{
						Status: "firing",
						Labels: map[string]string{"alertname": "Test"},
						StartsAt: time.Now(),
					},
				},
			},
		},
		{
			name: "no alerts",
			request: ProxyWebhookRequest{
				Receiver: "webhook-receiver",
				Status:   "firing",
				Alerts:   []AlertPayload{}, // Invalid
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.request)
			req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusBadRequest, w.Code)

			var errResp ErrorResponse
			err = json.NewDecoder(w.Body).Decode(&errResp)
			require.NoError(t, err)
			assert.Equal(t, string(ErrCodeValidation), errResp.Error)
		})
	}
}

// TestProxyWebhookHTTPHandler_ServeHTTP_ServiceError tests service-level errors
func TestProxyWebhookHTTPHandler_ServeHTTP_ServiceError(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "TestAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	// Mock service returns error
	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).
		Return(nil, errors.New("internal service error"))

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errResp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, string(ErrCodeInternal), errResp.Error)

	mockService.AssertExpectations(t)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_AllFailed tests all alerts failed scenario
func TestProxyWebhookHTTPHandler_ServeHTTP_AllFailed(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "TestAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	expectedResp := &ProxyWebhookResponse{
		Status:    "failed",
		Message:   "All alerts failed processing",
		Timestamp: time.Now(),
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived: 1,
			TotalFailed:   1,
		},
		ProcessingTime: 50 * time.Millisecond,
	}

	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).Return(expectedResp, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ProxyWebhookResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	assert.Equal(t, "failed", response.Status)

	mockService.AssertExpectations(t)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_TooManyAlerts tests max alerts limit
func TestProxyWebhookHTTPHandler_ServeHTTP_TooManyAlerts(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	config.MaxAlertsPerRequest = 10 // Limit to 10 alerts
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	// Create request with 15 alerts (exceeds limit)
	alerts := make([]AlertPayload, 15)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{"alertname": "TestAlert"},
			StartsAt: time.Now(),
		}
	}

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts:   alerts,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var errResp ErrorResponse
	err = json.NewDecoder(w.Body).Decode(&errResp)
	require.NoError(t, err)
	assert.Equal(t, string(ErrCodeValidation), errResp.Error)
	assert.Contains(t, errResp.Message, "10")
}

// TestProxyWebhookHTTPHandler_ServeHTTP_RequestIDTracking tests request ID tracking
func TestProxyWebhookHTTPHandler_ServeHTTP_RequestIDTracking(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "TestAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	expectedResp := &ProxyWebhookResponse{
		Status:    "success",
		Timestamp: time.Now(),
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived:  1,
			TotalProcessed: 1,
		},
		ProcessingTime: 100 * time.Millisecond,
	}

	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).Return(expectedResp, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Request-ID", "test-request-id-123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Request ID should be preserved in response headers or logging
	// (Actual implementation may vary)

	mockService.AssertExpectations(t)
}

// TestProxyWebhookHTTPHandler_ServeHTTP_Timeout tests request timeout
func TestProxyWebhookHTTPHandler_ServeHTTP_Timeout(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	config.RequestTimeout = 100 * time.Millisecond // Short timeout
	logger := slog.Default()

	handler, err := NewProxyWebhookHTTPHandler(mockService, config, logger)
	require.NoError(t, err)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "TestAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	// Mock service simulates slow processing
	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			time.Sleep(200 * time.Millisecond) // Exceeds timeout
		}).
		Return(&ProxyWebhookResponse{}, nil)

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return timeout error
	assert.Equal(t, http.StatusGatewayTimeout, w.Code)
}

// BenchmarkProxyWebhookHTTPHandler_ServeHTTP benchmarks HTTP request handling
func BenchmarkProxyWebhookHTTPHandler_ServeHTTP(b *testing.B) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()

	handler, _ := NewProxyWebhookHTTPHandler(mockService, config, logger)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "TestAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	expectedResp := &ProxyWebhookResponse{
		Status:    "success",
		Timestamp: time.Now(),
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived:  1,
			TotalProcessed: 1,
		},
		ProcessingTime: 10 * time.Millisecond,
	}

	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).Return(expectedResp, nil)

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)
	}
}

// BenchmarkProxyWebhookHTTPHandler_ParseRequest benchmarks request parsing
func BenchmarkProxyWebhookHTTPHandler_ParseRequest(b *testing.B) {
	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "TestAlert", "severity": "warning"},
				Annotations: map[string]string{"summary": "Test"},
				StartsAt: time.Now(),
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var parsed ProxyWebhookRequest
		_ = json.Unmarshal(jsonBody, &parsed)
	}
}

