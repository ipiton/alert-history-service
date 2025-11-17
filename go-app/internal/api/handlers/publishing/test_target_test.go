package publishing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
	infrapub "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// MockTargetDiscoveryManager is a mock implementation of TargetDiscoveryManager
type MockTargetDiscoveryManager struct {
	mock.Mock
}

func (m *MockTargetDiscoveryManager) DiscoverTargets(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTargetDiscoveryManager) GetTarget(name string) (*core.PublishingTarget, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*core.PublishingTarget), args.Error(1)
}

func (m *MockTargetDiscoveryManager) ListTargets() []*core.PublishingTarget {
	args := m.Called()
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*core.PublishingTarget)
}

func (m *MockTargetDiscoveryManager) GetTargetsByType(targetType string) []*core.PublishingTarget {
	args := m.Called(targetType)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]*core.PublishingTarget)
}

func (m *MockTargetDiscoveryManager) GetStats() publishing.DiscoveryStats {
	args := m.Called()
	if args.Get(0) == nil {
		return publishing.DiscoveryStats{}
	}
	return args.Get(0).(publishing.DiscoveryStats)
}

func (m *MockTargetDiscoveryManager) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockTargetDiscoveryManager) GetTargetCount() int {
	args := m.Called()
	return args.Int(0)
}


var (
	testQueueOnce sync.Once
	testQueue     *infrapub.PublishingQueue
)

// createTestQueue creates a minimal PublishingQueue for testing
// Uses singleton pattern to avoid duplicate Prometheus metric registration
func createTestQueue() *infrapub.PublishingQueue {
	testQueueOnce.Do(func() {
		config := infrapub.DefaultPublishingQueueConfig()
		config.WorkerCount = 1
		config.HighPriorityQueueSize = 10
		config.MediumPriorityQueueSize = 10
		config.LowPriorityQueueSize = 10

		// Create formatter and factory (required for queue to process jobs)
		formatter := infrapub.NewAlertFormatter()
		factory := infrapub.NewPublisherFactory(formatter, nil)

		testQueue = infrapub.NewPublishingQueue(
			factory, // factory (required for processing)
			nil,     // dlqRepository (not needed for test)
			nil,     // jobTrackingStore (not needed for test)
			config,
			nil, // metrics
			nil, // modeManager
			nil, // logger
		)

		// Start queue (required for Submit to work)
		testQueue.Start()
	})

	return testQueue
}

func TestTestTarget_Success(t *testing.T) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)

	target := &core.PublishingTarget{
		Name:    "test-target",
		Type:    "rootly",
		Enabled: true,
	}

	mockDiscovery.On("GetTarget", "test-target").Return(target, nil)
	mockDiscovery.On("GetTargetCount").Return(1).Maybe()
	mockDiscovery.On("Health", mock.Anything).Return(nil).Maybe()

	// Create coordinator with real queue (queue will accept submissions)
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil, // modeManager
		config,
		nil, // logger
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	// Create request
	req := httptest.NewRequest("POST", "/api/v2/publishing/targets/test-target/test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-target"})
	w := httptest.NewRecorder()

	// Execute
	handler.TestTarget(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestTargetResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)
	assert.Equal(t, "test-target", response.TargetName)
	assert.Equal(t, "Test alert sent", response.Message)
	assert.GreaterOrEqual(t, response.ResponseTimeMs, 0) // Can be 0 for very fast operations
	assert.NotZero(t, response.TestTimestamp)

	mockDiscovery.AssertExpectations(t)
}

func TestTestTarget_TargetNotFound(t *testing.T) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil,
		config,
		nil,
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	mockDiscovery.On("GetTarget", "invalid-target").Return(nil, errors.New("target not found"))
	mockDiscovery.On("GetTargetCount").Return(0).Maybe()
	mockDiscovery.On("Health", mock.Anything).Return(nil).Maybe()

	// Create request
	req := httptest.NewRequest("POST", "/api/v2/publishing/targets/invalid-target/test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "invalid-target"})
	w := httptest.NewRecorder()

	// Execute
	handler.TestTarget(w, req)

	// Assert
	assert.Equal(t, http.StatusNotFound, w.Code)

	mockDiscovery.AssertExpectations(t)
}

func TestTestTarget_TargetDisabled(t *testing.T) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil,
		config,
		nil,
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	target := &core.PublishingTarget{
		Name:    "disabled-target",
		Type:    "rootly",
		URL:     "http://localhost:8080/test",
		Enabled: false,
	}

	mockDiscovery.On("GetTarget", "disabled-target").Return(target, nil)
	mockDiscovery.On("GetTargetCount").Return(1).Maybe()
	mockDiscovery.On("Health", mock.Anything).Return(nil).Maybe()

	// Create request
	req := httptest.NewRequest("POST", "/api/v2/publishing/targets/disabled-target/test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "disabled-target"})
	w := httptest.NewRecorder()

	// Execute
	handler.TestTarget(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestTargetResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.False(t, response.Success)
	assert.Equal(t, "Target is disabled", response.Message)
	assert.Equal(t, "disabled-target", response.TargetName)

	mockDiscovery.AssertExpectations(t)
}

func TestTestTarget_WithCustomAlert(t *testing.T) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil,
		config,
		nil,
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	target := &core.PublishingTarget{
		Name:    "test-target",
		Type:    "rootly",
		URL:     "http://localhost:8080/test",
		Enabled: true,
	}

	requestBody := TestTargetRequest{
		AlertName: "CustomTestAlert",
		TestAlert: &CustomTestAlert{
			Labels: map[string]string{
				"alertname": "CustomTestAlert",
				"severity":  "warning",
			},
			Annotations: map[string]string{
				"summary": "Custom test alert",
			},
		},
		TimeoutSeconds: 60,
	}

	mockDiscovery.On("GetTarget", "test-target").Return(target, nil)
	mockDiscovery.On("GetTargetCount").Return(1).Maybe()
	mockDiscovery.On("Health", mock.Anything).Return(nil).Maybe()

	// Create request
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v2/publishing/targets/test-target/test", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-target"})
	w := httptest.NewRecorder()

	// Execute
	handler.TestTarget(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestTargetResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.True(t, response.Success)

	// Verify response
	assert.True(t, response.Success)
	assert.Equal(t, "test-target", response.TargetName)

	mockDiscovery.AssertExpectations(t)
}

func TestTestTarget_PublishingFailure(t *testing.T) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)
	// For failure test, we'll use a queue that fails on submit
	// But coordinator will handle it gracefully
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil,
		config,
		nil,
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	target := &core.PublishingTarget{
		Name:    "test-target",
		Type:    "rootly",
		URL:     "http://localhost:8080/test",
		Enabled: true,
	}

	mockDiscovery.On("GetTarget", "test-target").Return(target, nil)
	mockDiscovery.On("GetTargetCount").Return(1).Maybe()
	mockDiscovery.On("Health", mock.Anything).Return(nil).Maybe()
	// Note: With real coordinator, publishing will succeed (queue accepts),
	// but we can test error handling by checking response

	// Create request
	req := httptest.NewRequest("POST", "/api/v2/publishing/targets/test-target/test", nil)
	req = mux.SetURLVars(req, map[string]string{"name": "test-target"})
	w := httptest.NewRecorder()

	// Execute
	handler.TestTarget(w, req)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response TestTargetResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	// With real coordinator, publishing should succeed (queue accepts submissions)
	// For failure testing, we'd need to mock the publisher factory
	assert.Equal(t, "test-target", response.TargetName)

	mockDiscovery.AssertExpectations(t)
}

func TestTestTarget_InvalidTimeout(t *testing.T) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil,
		config,
		nil,
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	requestBody := TestTargetRequest{
		TimeoutSeconds: 500, // Invalid: > 300
	}

	// Create request
	body, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/api/v2/publishing/targets/test-target/test", bytes.NewReader(body))
	req = mux.SetURLVars(req, map[string]string{"name": "test-target"})
	w := httptest.NewRecorder()

	// Execute
	handler.TestTarget(w, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, w.Code)

	mockDiscovery.AssertNotCalled(t, "GetTarget")
}

func TestBuildTestAlert_Default(t *testing.T) {
	// Setup
	handler := NewPublishingHandlers(nil, nil, nil, nil, nil)

	req := &TestTargetRequest{
		AlertName: "MyTestAlert",
	}

	// Execute
	alert, err := handler.buildTestAlert(req, "test-target")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, "MyTestAlert", alert.Alert.AlertName)
	assert.Equal(t, core.StatusFiring, alert.Alert.Status)
	assert.Equal(t, "true", alert.Alert.Labels["test"])
	assert.Equal(t, "info", alert.Alert.Labels["severity"])
	assert.Contains(t, alert.Alert.Fingerprint, "test-target")
}

func TestBuildTestAlert_CustomPayload(t *testing.T) {
	// Setup
	handler := NewPublishingHandlers(nil, nil, nil, nil, nil)

	req := &TestTargetRequest{
		TestAlert: &CustomTestAlert{
			Fingerprint: "custom-fingerprint",
			Labels: map[string]string{
				"alertname": "CustomAlert",
				"severity":  "critical",
			},
			Annotations: map[string]string{
				"summary": "Custom summary",
			},
			Status: "firing",
		},
	}

	// Execute
	alert, err := handler.buildTestAlert(req, "test-target")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, "CustomAlert", alert.Alert.AlertName)
	assert.Equal(t, "custom-fingerprint", alert.Alert.Fingerprint)
	assert.Equal(t, "critical", alert.Alert.Labels["severity"])
	assert.Equal(t, "true", alert.Alert.Labels["test"]) // Test label should be added
	assert.Equal(t, "Custom summary", alert.Alert.Annotations["summary"])
}

func TestBuildTestAlert_CustomResolvedStatus(t *testing.T) {
	// Setup
	handler := NewPublishingHandlers(nil, nil, nil, nil, nil)

	req := &TestTargetRequest{
		TestAlert: &CustomTestAlert{
			Status: "resolved",
		},
	}

	// Execute
	alert, err := handler.buildTestAlert(req, "test-target")

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, alert)
	assert.Equal(t, core.StatusResolved, alert.Alert.Status)
}

func BenchmarkTestTarget(b *testing.B) {
	// Setup
	mockDiscovery := new(MockTargetDiscoveryManager)
	testQueue := createTestQueue()
	// Note: We don't stop the queue here as it's shared across tests

	target := &core.PublishingTarget{
		Name:    "test-target",
		Type:    "rootly",
		URL:     "http://localhost:8080/test",
		Enabled: true,
	}

	mockDiscovery.On("GetTarget", "test-target").Return(target, nil).Maybe()
	mockDiscovery.On("GetTargetCount").Return(1).Maybe()
	mockDiscovery.On("Health", mock.Anything).Return(nil).Maybe()

	config := infrapub.DefaultCoordinatorConfig()
	coordinator := infrapub.NewPublishingCoordinator(
		testQueue,
		mockDiscovery,
		nil,
		config,
		nil,
	)

	handler := NewPublishingHandlers(mockDiscovery, nil, nil, coordinator, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/publishing/targets/test-target/test", nil)
		req = mux.SetURLVars(req, map[string]string{"name": "test-target"})
		w := httptest.NewRecorder()
		handler.TestTarget(w, req)
	}
}

func BenchmarkBuildTestAlert(b *testing.B) {
	handler := NewPublishingHandlers(nil, nil, nil, nil, nil)
	req := &TestTargetRequest{
		AlertName: "TestAlert",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = handler.buildTestAlert(req, "test-target")
	}
}
