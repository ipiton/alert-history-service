package publishing

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Mock PagerDuty Events Client
type MockPagerDutyEventsClient struct {
	mock.Mock
}

func (m *MockPagerDutyEventsClient) TriggerEvent(ctx context.Context, req *TriggerEventRequest) (*EventResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EventResponse), args.Error(1)
}

func (m *MockPagerDutyEventsClient) AcknowledgeEvent(ctx context.Context, req *AcknowledgeEventRequest) (*EventResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EventResponse), args.Error(1)
}

func (m *MockPagerDutyEventsClient) ResolveEvent(ctx context.Context, req *ResolveEventRequest) (*EventResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*EventResponse), args.Error(1)
}

func (m *MockPagerDutyEventsClient) SendChangeEvent(ctx context.Context, req *ChangeEventRequest) (*ChangeEventResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*ChangeEventResponse), args.Error(1)
}

func (m *MockPagerDutyEventsClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// Mock Alert Formatter
type MockAlertFormatter struct {
	mock.Mock
}

func (m *MockAlertFormatter) FormatAlert(ctx context.Context, enrichedAlert *core.EnrichedAlert, format core.PublishingFormat) (map[string]any, error) {
	args := m.Called(ctx, enrichedAlert, format)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]any), args.Error(1)
}

// Mock Event Key Cache
type MockEventKeyCache struct {
	mock.Mock
}

func (m *MockEventKeyCache) Set(fingerprint string, dedupKey string) {
	m.Called(fingerprint, dedupKey)
}

func (m *MockEventKeyCache) Get(fingerprint string) (string, bool) {
	args := m.Called(fingerprint)
	return args.String(0), args.Bool(1)
}

func (m *MockEventKeyCache) Delete(fingerprint string) {
	m.Called(fingerprint)
}

func (m *MockEventKeyCache) Cleanup() {
	m.Called()
}

func (m *MockEventKeyCache) Size() int {
	args := m.Called()
	return args.Int(0)
}

// Test Helpers
func createTestEnrichedAlert(status core.AlertStatus) *core.EnrichedAlert {
	return &core.EnrichedAlert{
		Alert: &core.Alert{
			Fingerprint: "test-fingerprint",
			AlertName:   "TestAlert",
			Status:      status,
			Labels: map[string]string{
				"severity": "critical",
			},
			Annotations: map[string]string{
				"grafana_url": "https://grafana.example.com/dashboard",
				"runbook_url": "https://runbook.example.com",
			},
			StartsAt: time.Now(),
		},
		Classification: &core.AlertClassification{
			Severity:   core.SeverityCritical,
			Confidence: 0.95,
		},
	}
}

func createTestTarget() *core.PublishingTarget {
	return &core.PublishingTarget{
		Name:   "pagerduty-prod",
		Type:   "pagerduty",
		URL:    "https://events.pagerduty.com/v2/events",
		Format: core.FormatPagerDuty,
		Headers: map[string]string{
			"routing_key": "test-routing-key",
		},
	}
}

// Tests

func TestEnhancedPagerDutyPublisher_Publish_FiringAlert(t *testing.T) {
	// Setup mocks
	mockClient := new(MockPagerDutyEventsClient)
	mockCache := new(MockEventKeyCache)
	mockFormatter := new(MockAlertFormatter)
	metrics := NewPagerDutyMetrics()
	logger := slog.Default()

	publisher := NewEnhancedPagerDutyPublisher(
		mockClient,
		mockCache,
		metrics,
		mockFormatter,
		logger,
	)

	// Test data
	enrichedAlert := createTestEnrichedAlert(core.StatusFiring)
	target := createTestTarget()
	ctx := context.Background()

	// Mock formatter response
	formattedPayload := map[string]any{
		"summary":  "Test alert",
		"severity": "critical",
		"source":   "alert-history",
		"payload": map[string]any{
			"custom_details": map[string]any{
				"fingerprint": "test-fingerprint",
			},
		},
	}

	mockFormatter.On("FormatAlert", ctx, enrichedAlert, core.FormatPagerDuty).
		Return(formattedPayload, nil)

	// Mock client response
	mockClient.On("TriggerEvent", ctx, mock.MatchedBy(func(req *TriggerEventRequest) bool {
		return req.RoutingKey == "test-routing-key" &&
			req.DedupKey == "test-fingerprint" &&
			req.EventAction == "trigger"
	})).Return(&EventResponse{
		Status:   "success",
		DedupKey: "pd-dedup-key",
	}, nil)

	// Mock cache
	mockCache.On("Set", "test-fingerprint", "pd-dedup-key")

	// Execute
	err := publisher.Publish(ctx, enrichedAlert, target)

	// Assert
	assert.NoError(t, err)
	mockFormatter.AssertExpectations(t)
	mockClient.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestEnhancedPagerDutyPublisher_Publish_ResolvedAlert(t *testing.T) {
	// Setup mocks
	mockClient := new(MockPagerDutyEventsClient)
	mockCache := new(MockEventKeyCache)
	mockFormatter := new(MockAlertFormatter)
	metrics := NewPagerDutyMetrics()
	logger := slog.Default()

	publisher := NewEnhancedPagerDutyPublisher(
		mockClient,
		mockCache,
		metrics,
		mockFormatter,
		logger,
	)

	// Test data
	enrichedAlert := createTestEnrichedAlert(core.StatusResolved)
	target := createTestTarget()
	ctx := context.Background()

	// Mock cache lookup
	mockCache.On("Get", "test-fingerprint").
		Return("pd-dedup-key", true)

	// Mock client response
	mockClient.On("ResolveEvent", ctx, mock.MatchedBy(func(req *ResolveEventRequest) bool {
		return req.RoutingKey == "test-routing-key" &&
			req.DedupKey == "pd-dedup-key" &&
			req.EventAction == "resolve"
	})).Return(&EventResponse{
		Status:   "success",
		DedupKey: "pd-dedup-key",
	}, nil)

	// Mock cache delete
	mockCache.On("Delete", "test-fingerprint")

	// Execute
	err := publisher.Publish(ctx, enrichedAlert, target)

	// Assert
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
	mockCache.AssertExpectations(t)
}

func TestEnhancedPagerDutyPublisher_Publish_ResolvedAlert_NotTracked(t *testing.T) {
	// Setup mocks
	mockClient := new(MockPagerDutyEventsClient)
	mockCache := new(MockEventKeyCache)
	mockFormatter := new(MockAlertFormatter)
	metrics := NewPagerDutyMetrics()
	logger := slog.Default()

	publisher := NewEnhancedPagerDutyPublisher(
		mockClient,
		mockCache,
		metrics,
		mockFormatter,
		logger,
	)

	// Test data
	enrichedAlert := createTestEnrichedAlert(core.StatusResolved)
	target := createTestTarget()
	ctx := context.Background()

	// Mock cache lookup - not found
	mockCache.On("Get", "test-fingerprint").
		Return("", false)

	// Execute
	err := publisher.Publish(ctx, enrichedAlert, target)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrEventNotTracked)
	mockCache.AssertExpectations(t)
}

func TestEnhancedPagerDutyPublisher_Publish_MissingRoutingKey(t *testing.T) {
	// Setup mocks
	mockClient := new(MockPagerDutyEventsClient)
	mockCache := new(MockEventKeyCache)
	mockFormatter := new(MockAlertFormatter)
	metrics := NewPagerDutyMetrics()
	logger := slog.Default()

	publisher := NewEnhancedPagerDutyPublisher(
		mockClient,
		mockCache,
		metrics,
		mockFormatter,
		logger,
	)

	// Test data
	enrichedAlert := createTestEnrichedAlert(core.StatusFiring)
	target := &core.PublishingTarget{
		Name:    "pagerduty-prod",
		Type:    "pagerduty",
		Headers: map[string]string{}, // No routing_key
	}
	ctx := context.Background()

	// Execute
	err := publisher.Publish(ctx, enrichedAlert, target)

	// Assert
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrMissingRoutingKey)
}

func TestEnhancedPagerDutyPublisher_Publish_FormatterError(t *testing.T) {
	// Setup mocks
	mockClient := new(MockPagerDutyEventsClient)
	mockCache := new(MockEventKeyCache)
	mockFormatter := new(MockAlertFormatter)
	metrics := NewPagerDutyMetrics()
	logger := slog.Default()

	publisher := NewEnhancedPagerDutyPublisher(
		mockClient,
		mockCache,
		metrics,
		mockFormatter,
		logger,
	)

	// Test data
	enrichedAlert := createTestEnrichedAlert(core.StatusFiring)
	target := createTestTarget()
	ctx := context.Background()

	// Mock formatter error
	mockFormatter.On("FormatAlert", ctx, enrichedAlert, core.FormatPagerDuty).
		Return(nil, errors.New("formatter error"))

	// Execute
	err := publisher.Publish(ctx, enrichedAlert, target)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "formatter error")
	mockFormatter.AssertExpectations(t)
}

func TestEnhancedPagerDutyPublisher_Publish_ChangeEvent(t *testing.T) {
	// Setup mocks
	mockClient := new(MockPagerDutyEventsClient)
	mockCache := new(MockEventKeyCache)
	mockFormatter := new(MockAlertFormatter)
	metrics := NewPagerDutyMetrics()
	logger := slog.Default()

	publisher := NewEnhancedPagerDutyPublisher(
		mockClient,
		mockCache,
		metrics,
		mockFormatter,
		logger,
	)

	// Test data - alert with change_event label
	enrichedAlert := createTestEnrichedAlert(core.StatusFiring)
	enrichedAlert.Alert.Labels["change_event"] = "true"
	target := createTestTarget()
	ctx := context.Background()

	// Mock client response
	mockClient.On("SendChangeEvent", ctx, mock.MatchedBy(func(req *ChangeEventRequest) bool {
		return req.RoutingKey == "test-routing-key"
	})).Return(&ChangeEventResponse{
		Status:  "success",
		Message: "Change event processed",
	}, nil)

	// Execute
	err := publisher.Publish(ctx, enrichedAlert, target)

	// Assert
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestEnhancedPagerDutyPublisher_ExtractLinks(t *testing.T) {
	publisher := &EnhancedPagerDutyPublisher{
		logger: slog.Default(),
	}

	alert := &core.Alert{
		Annotations: map[string]string{
			"grafana_url": "https://grafana.example.com/dashboard",
			"runbook_url": "https://runbook.example.com",
		},
	}

	links := publisher.extractLinks(alert)

	assert.Len(t, links, 2)
	assert.Equal(t, "https://grafana.example.com/dashboard", links[0].Href)
	assert.Equal(t, "Grafana Dashboard", links[0].Text)
	assert.Equal(t, "https://runbook.example.com", links[1].Href)
	assert.Equal(t, "Runbook", links[1].Text)
}

func TestEnhancedPagerDutyPublisher_ExtractImages(t *testing.T) {
	publisher := &EnhancedPagerDutyPublisher{
		logger: slog.Default(),
	}

	alert := &core.Alert{
		Annotations: map[string]string{
			"grafana_snapshot": "https://snapshot.grafana.com/image.png",
		},
	}

	images := publisher.extractImages(alert)

	assert.Len(t, images, 1)
	assert.Equal(t, "https://snapshot.grafana.com/image.png", images[0].Src)
	assert.Equal(t, "Grafana Snapshot", images[0].Alt)
}

func TestEnhancedPagerDutyPublisher_Name(t *testing.T) {
	publisher := &EnhancedPagerDutyPublisher{}

	assert.Equal(t, "PagerDuty", publisher.Name())
}
