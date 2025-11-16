package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"log/slog"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
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

func (m *MockProxyWebhookService) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestNewProxyWebhookHTTPHandler tests handler creation
func TestNewProxyWebhookHTTPHandler(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()
	metricsRegistry := metrics.NewMetricsRegistry("test-proxy-webhook")

	handler := NewProxyWebhookHTTPHandler(mockService, config, logger, metricsRegistry)

	require.NotNil(t, handler)
	assert.NotNil(t, handler.service)
	assert.NotNil(t, handler.config)
	assert.NotNil(t, handler.logger)
}

// TestProxyWebhookHandler_MethodNotAllowed tests non-POST requests
func TestProxyWebhookHandler_MethodNotAllowed(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()
	metricsRegistry := metrics.NewMetricsRegistry("test-proxy-webhook")

	handler := NewProxyWebhookHTTPHandler(mockService, config, logger, metricsRegistry)

	req := httptest.NewRequest(http.MethodGet, "/webhook/proxy", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

// TestProxyWebhookHandler_InvalidContentType tests invalid Content-Type
func TestProxyWebhookHandler_InvalidContentType(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()
	metricsRegistry := metrics.NewMetricsRegistry("test-proxy-webhook")

	handler := NewProxyWebhookHTTPHandler(mockService, config, logger, metricsRegistry)

	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", nil)
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

// TestProxyWebhookHandler_Success tests successful request processing
func TestProxyWebhookHandler_Success(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()
	metricsRegistry := metrics.NewMetricsRegistry("test-proxy-webhook")

	handler := NewProxyWebhookHTTPHandler(mockService, config, logger, metricsRegistry)

	// Mock response
	mockResponse := &ProxyWebhookResponse{
		Status:         "success",
		Message:        "Processed successfully",
		Timestamp:      time.Now(),
		ProcessingTime: 100 * time.Millisecond,
		AlertsSummary: AlertsProcessingSummary{
			TotalReceived:   1,
			TotalProcessed:  1,
			TotalClassified: 1,
			TotalFiltered:   0,
			TotalPublished:  1,
		},
		PublishingSummary: PublishingSummary{
			TotalTargets:      1,
			SuccessfulTargets: 1,
			FailedTargets:     0,
		},
	}

	mockService.On("ProcessWebhook", mock.Anything, mock.Anything).Return(mockResponse, nil)

	reqBody := ProxyWebhookRequest{
		Receiver: "webhook-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status:   "firing",
				Labels:   map[string]string{"alertname": "TestAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ProxyWebhookResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, 1, resp.AlertsSummary.TotalProcessed)
}

// TestProxyWebhookHandler_EmptyBody tests empty request body
func TestProxyWebhookHandler_EmptyBody(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()
	metricsRegistry := metrics.NewMetricsRegistry("test-proxy-webhook")

	handler := NewProxyWebhookHTTPHandler(mockService, config, logger, metricsRegistry)

	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestProxyWebhookHandler_InvalidJSON tests invalid JSON payload
func TestProxyWebhookHandler_InvalidJSON(t *testing.T) {
	mockService := new(MockProxyWebhookService)
	config := DefaultProxyWebhookConfig()
	logger := slog.Default()
	metricsRegistry := metrics.NewMetricsRegistry("test-proxy-webhook")

	handler := NewProxyWebhookHTTPHandler(mockService, config, logger, metricsRegistry)

	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
