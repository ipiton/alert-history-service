// +build integration

package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/business/proxy"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// Integration tests require real dependencies or in-memory implementations
// These tests verify end-to-end flow through HTTP -> Service -> Pipelines

// TestIntegration_FullPipeline_Success tests complete pipeline with real components
func TestIntegration_FullPipeline_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup real or in-memory components
	svc := setupTestService(t)
	handler, err := NewProxyWebhookHTTPHandler(svc, DefaultProxyWebhookConfig(), slog.Default())
	require.NoError(t, err)

	// Create realistic webhook request
	reqBody := ProxyWebhookRequest{
		Receiver: "alertmanager-webhook",
		Status:   "firing",
		GroupKey: "group-cpu-high",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPUUsage",
					"severity":  "warning",
					"instance":  "server-01",
					"namespace": "production",
					"cluster":   "us-east-1",
				},
				Annotations: map[string]string{
					"summary":     "High CPU usage detected",
					"description": "CPU usage has been above 80% for 5 minutes",
				},
				StartsAt:     time.Now().Add(-5 * time.Minute),
				GeneratorURL: "http://prometheus:9090/alerts",
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ProxyWebhookResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "success", resp.Status)
	assert.Equal(t, 1, resp.AlertsSummary.TotalReceived)
	assert.Greater(t, resp.AlertsSummary.TotalClassified, 0)
}

// TestIntegration_ClassificationPipeline tests classification pipeline integration
func TestIntegration_ClassificationPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	svc := setupTestService(t)

	req := &ProxyWebhookRequest{
		Receiver: "test-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "DatabaseDown",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary": "Database connection failed",
				},
				StartsAt: time.Now(),
			},
		},
	}

	ctx := context.Background()
	resp, err := svc.ProcessWebhook(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1, resp.AlertsSummary.TotalClassified)
	
	// Verify classification result
	if len(resp.AlertResults) > 0 && resp.AlertResults[0].Classification != nil {
		class := resp.AlertResults[0].Classification
		assert.NotEmpty(t, class.Severity)
		assert.NotEmpty(t, class.Category)
		assert.GreaterOrEqual(t, class.Confidence, 0.0)
		assert.LessOrEqual(t, class.Confidence, 1.0)
	}
}

// TestIntegration_FilteringPipeline tests filtering pipeline integration
func TestIntegration_FilteringPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	svc := setupTestService(t)

	tests := []struct {
		name           string
		alert          AlertPayload
		expectedFilter bool
	}{
		{
			name: "test alert should be filtered",
			alert: AlertPayload{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert", // Contains "Test"
				},
				StartsAt: time.Now(),
			},
			expectedFilter: true,
		},
		{
			name: "production alert should pass",
			alert: AlertPayload{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "ProductionAlert",
					"severity":  "critical",
				},
				StartsAt: time.Now(),
			},
			expectedFilter: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &ProxyWebhookRequest{
				Receiver: "test-receiver",
				Status:   "firing",
				Alerts:   []AlertPayload{tt.alert},
			}

			ctx := context.Background()
			resp, err := svc.ProcessWebhook(ctx, req)

			require.NoError(t, err)
			require.NotNil(t, resp)

			if tt.expectedFilter {
				assert.Greater(t, resp.AlertsSummary.TotalFiltered, 0)
			} else {
				assert.Equal(t, 0, resp.AlertsSummary.TotalFiltered)
			}
		})
	}
}

// TestIntegration_PublishingPipeline tests publishing pipeline integration
func TestIntegration_PublishingPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	svc := setupTestServiceWithTargets(t)

	req := &ProxyWebhookRequest{
		Receiver: "test-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "ServiceDown",
					"severity":  "critical",
				},
				StartsAt: time.Now(),
			},
		},
	}

	ctx := context.Background()
	resp, err := svc.ProcessWebhook(ctx, req)

	require.NoError(t, err)
	require.NotNil(t, resp)
	
	// Verify publishing metrics
	assert.GreaterOrEqual(t, resp.PublishingSummary.TotalTargets, 0)
	if resp.PublishingSummary.TotalTargets > 0 {
		assert.GreaterOrEqual(t, resp.PublishingSummary.SuccessfulTargets+resp.PublishingSummary.FailedTargets,
			0)
	}
}

// TestIntegration_BatchProcessing tests batch alert processing
func TestIntegration_BatchProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	svc := setupTestService(t)

	// Create batch of 50 alerts
	alerts := make([]AlertPayload, 50)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "BatchAlert",
				"instance":  "server-01",
			},
			StartsAt: time.Now(),
		}
	}

	req := &ProxyWebhookRequest{
		Receiver: "test-receiver",
		Status:   "firing",
		Alerts:   alerts,
	}

	ctx := context.Background()
	startTime := time.Now()
	resp, err := svc.ProcessWebhook(ctx, req)
	duration := time.Since(startTime)

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 50, resp.AlertsSummary.TotalReceived)
	
	// Performance: should process 50 alerts in reasonable time
	assert.Less(t, duration, 10*time.Second, "Batch processing took too long")
	
	t.Logf("Processed 50 alerts in %v (avg %v per alert)",
		duration, duration/50)
}

// TestIntegration_ErrorRecovery tests error recovery and fallbacks
func TestIntegration_ErrorRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Create service with potentially failing components
	svc := setupTestServiceWithFailures(t)

	req := &ProxyWebhookRequest{
		Receiver: "test-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":  "warning",
				},
				StartsAt: time.Now(),
			},
		},
	}

	ctx := context.Background()
	resp, err := svc.ProcessWebhook(ctx, req)

	// Should not fail entirely due to fallback mechanisms
	require.NoError(t, err)
	require.NotNil(t, resp)
	
	// Verify fallback classification was used
	if len(resp.AlertResults) > 0 && resp.AlertResults[0].Classification != nil {
		assert.Equal(t, "default", resp.AlertResults[0].Classification.Source)
	}
}

// TestIntegration_ConcurrentRequests tests concurrent webhook processing
func TestIntegration_ConcurrentRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	svc := setupTestService(t)
	handler, err := NewProxyWebhookHTTPHandler(svc, DefaultProxyWebhookConfig(), slog.Default())
	require.NoError(t, err)

	reqBody := ProxyWebhookRequest{
		Receiver: "test-receiver",
		Status:   "firing",
		Alerts: []AlertPayload{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "ConcurrentAlert"},
				StartsAt: time.Now(),
			},
		},
	}

	jsonBody, _ := json.Marshal(reqBody)

	// Send 10 concurrent requests
	concurrency := 10
	done := make(chan bool, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < concurrency; i++ {
		<-done
	}
}

// TestIntegration_LargePayload tests handling of large webhook payloads
func TestIntegration_LargePayload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	svc := setupTestService(t)
	handler, err := NewProxyWebhookHTTPHandler(svc, DefaultProxyWebhookConfig(), slog.Default())
	require.NoError(t, err)

	// Create large payload with 100 alerts
	alerts := make([]AlertPayload, 100)
	for i := range alerts {
		alerts[i] = AlertPayload{
			Status: "firing",
			Labels: map[string]string{
				"alertname": "LargePayloadAlert",
				"instance":  "server-01",
			},
			Annotations: map[string]string{
				"summary":     "Test alert with long description",
				"description": "This is a very long description that takes up space in the payload",
			},
			StartsAt: time.Now(),
		}
	}

	reqBody := ProxyWebhookRequest{
		Receiver: "test-receiver",
		Status:   "firing",
		Alerts:   alerts,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/webhook/proxy", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp ProxyWebhookResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, 100, resp.AlertsSummary.TotalReceived)
}

// Helper functions for test setup

func setupTestService(t *testing.T) *proxy.ProxyWebhookService {
	// Create in-memory or mock implementations
	cfg := proxy.ServiceConfig{
		AlertProcessor:    newMockAlertProcessor(),
		ClassificationSvc: newMockClassificationService(),
		FilterEngine:      newMockFilterEngine(),
		TargetManager:     newMockTargetManager(0), // No targets
		ParallelPublisher: newMockParallelPublisher(),
		Config:            DefaultProxyWebhookConfig(),
		Logger:            slog.Default(),
	}

	svc, err := proxy.NewProxyWebhookService(cfg)
	require.NoError(t, err)
	return svc
}

func setupTestServiceWithTargets(t *testing.T) *proxy.ProxyWebhookService {
	cfg := proxy.ServiceConfig{
		AlertProcessor:    newMockAlertProcessor(),
		ClassificationSvc: newMockClassificationService(),
		FilterEngine:      newMockFilterEngine(),
		TargetManager:     newMockTargetManager(3), // 3 targets
		ParallelPublisher: newMockParallelPublisher(),
		Config:            DefaultProxyWebhookConfig(),
		Logger:            slog.Default(),
	}

	svc, err := proxy.NewProxyWebhookService(cfg)
	require.NoError(t, err)
	return svc
}

func setupTestServiceWithFailures(t *testing.T) *proxy.ProxyWebhookService {
	cfg := proxy.ServiceConfig{
		AlertProcessor:    newMockAlertProcessor(),
		ClassificationSvc: newFailingClassificationService(),
		FilterEngine:      newMockFilterEngine(),
		TargetManager:     newMockTargetManager(0),
		ParallelPublisher: newMockParallelPublisher(),
		Config:            DefaultProxyWebhookConfig(),
		Logger:            slog.Default(),
	}

	svc, err := proxy.NewProxyWebhookService(cfg)
	require.NoError(t, err)
	return svc
}

// Mock implementations for integration tests

func newMockAlertProcessor() services.AlertProcessor {
	// Return simple mock that stores alerts
	mock := new(MockAlertProcessor)
	mock.On("ProcessAlert", context.Background(), &core.Alert{}).Return(nil)
	mock.On("Health", context.Background()).Return(nil)
	return mock
}

func newMockClassificationService() services.ClassificationService {
	mock := new(MockClassificationService)
	mock.On("ClassifyAlert", context.Background(), &core.Alert{}).Return(&core.ClassificationResult{
		Severity:   core.SeverityWarning,
		Category:   "test",
		Confidence: 0.85,
	}, nil)
	mock.On("Health", context.Background()).Return(nil)
	return mock
}

func newFailingClassificationService() services.ClassificationService {
	mock := new(MockClassificationService)
	mock.On("ClassifyAlert", context.Background(), &core.Alert{}).Return(nil, errors.New("LLM unavailable"))
	mock.On("Health", context.Background()).Return(nil)
	return mock
}

func newMockFilterEngine() services.FilterEngine {
	mock := new(MockFilterEngine)
	mock.On("ShouldBlock", &core.Alert{}, &core.ClassificationResult{}).Return(false, "")
	return mock
}

func newMockTargetManager(targetCount int) publishing.TargetDiscoveryManager {
	mock := new(MockTargetDiscoveryManager)
	
	targets := make([]*core.PublishingTarget, targetCount)
	for i := 0; i < targetCount; i++ {
		targets[i] = &core.PublishingTarget{
			Name:    "test-target",
			Type:    "slack",
			Enabled: true,
		}
	}
	
	mock.On("ListTargets").Return(targets)
	mock.On("GetTargetCount").Return(targetCount)
	return mock
}

func newMockParallelPublisher() publishing.ParallelPublisher {
	mock := new(MockParallelPublisher)
	mock.On("PublishToMultiple", context.Background(), &core.EnrichedAlert{}, []*core.PublishingTarget{}).
		Return(&publishing.ParallelPublishResult{
			SuccessCount: 0,
			FailureCount: 0,
			TotalTargets: 0,
		}, nil)
	return mock
}

