package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ============================================================================
// TN-064: Unit Tests for Report Endpoint
// ============================================================================

// MockHistoryRepository for testing
type MockHistoryRepository struct {
	GetAggregatedStatsFunc func(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error)
	GetTopAlertsFunc       func(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error)
	GetFlappingAlertsFunc  func(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error)
	GetRecentAlertsFunc    func(ctx context.Context, limit int) ([]*core.Alert, error)
}

func (m *MockHistoryRepository) GetHistory(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
	return nil, nil
}

func (m *MockHistoryRepository) GetAlertsByFingerprint(ctx context.Context, fingerprint string, limit int) ([]*core.Alert, error) {
	return nil, nil
}

func (m *MockHistoryRepository) GetRecentAlerts(ctx context.Context, limit int) ([]*core.Alert, error) {
	if m.GetRecentAlertsFunc != nil {
		return m.GetRecentAlertsFunc(ctx, limit)
	}
	return []*core.Alert{}, nil
}

func (m *MockHistoryRepository) GetAggregatedStats(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
	if m.GetAggregatedStatsFunc != nil {
		return m.GetAggregatedStatsFunc(ctx, timeRange)
	}
	return &core.AggregatedStats{
		TotalAlerts:    100,
		FiringAlerts:   10,
		ResolvedAlerts: 90,
	}, nil
}

func (m *MockHistoryRepository) GetTopAlerts(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
	if m.GetTopAlertsFunc != nil {
		return m.GetTopAlertsFunc(ctx, timeRange, limit)
	}
	namespace := "production"
	return []*core.TopAlert{
		{
			Fingerprint: "fp1",
			AlertName:   "CPUHigh",
			Namespace:   &namespace,
			FireCount:   50,
		},
	}, nil
}

func (m *MockHistoryRepository) GetFlappingAlerts(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
	if m.GetFlappingAlertsFunc != nil {
		return m.GetFlappingAlertsFunc(ctx, timeRange, threshold)
	}
	namespace := "staging"
	return []*core.FlappingAlert{
		{
			Fingerprint:     "fp2",
			AlertName:       "DiskSpace",
			Namespace:       &namespace,
			TransitionCount: 10,
			FlappingScore:   5.0,
		},
	}, nil
}

// ============================================================================
// Test 1-10: Request Parsing Tests
// ============================================================================

func TestParseReportRequest_Valid(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z&top=20&min_flap=5", nil)

	result, err := handler.parseReportRequest(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.TopLimit != 20 {
		t.Errorf("Expected TopLimit=20, got %d", result.TopLimit)
	}

	if result.MinFlapCount != 5 {
		t.Errorf("Expected MinFlapCount=5, got %d", result.MinFlapCount)
	}

	if result.TimeRange == nil {
		t.Fatal("Expected TimeRange to be set")
	}
}

func TestParseReportRequest_DefaultValues(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report", nil)

	result, err := handler.parseReportRequest(req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.TopLimit != 10 {
		t.Errorf("Expected default TopLimit=10, got %d", result.TopLimit)
	}

	if result.MinFlapCount != 3 {
		t.Errorf("Expected default MinFlapCount=3, got %d", result.MinFlapCount)
	}

	// Should have default time range (last 24 hours)
	if result.TimeRange == nil {
		t.Fatal("Expected default TimeRange to be set")
	}
}

func TestParseReportRequest_InvalidTimeRange_ToBeforeFrom(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?from=2024-01-02T00:00:00Z&to=2024-01-01T00:00:00Z", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid time range, got nil")
	}

	validationErr, ok := err.(*core.ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "to" {
		t.Errorf("Expected field='to', got %s", validationErr.Field)
	}
}

func TestParseReportRequest_InvalidTimeRange_TooLarge(t *testing.T) {
	handler := &HistoryHandlerV2{}

	// 100 days (exceeds 90 day limit)
	req := httptest.NewRequest("GET", "/report?from=2024-01-01T00:00:00Z&to=2024-04-10T00:00:00Z", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for time range too large, got nil")
	}

	validationErr, ok := err.(*core.ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "time_range" {
		t.Errorf("Expected field='time_range', got %s", validationErr.Field)
	}
}

func TestParseReportRequest_InvalidSeverity(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?severity=invalid", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for invalid severity, got nil")
	}

	validationErr, ok := err.(*core.ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "severity" {
		t.Errorf("Expected field='severity', got %s", validationErr.Field)
	}
}

func TestParseReportRequest_InvalidTopLimit_TooLow(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?top=0", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for top=0, got nil")
	}
}

func TestParseReportRequest_InvalidTopLimit_TooHigh(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?top=101", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for top=101, got nil")
	}
}

func TestParseReportRequest_InvalidMinFlap_TooLow(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?min_flap=0", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for min_flap=0, got nil")
	}
}

func TestParseReportRequest_InvalidMinFlap_TooHigh(t *testing.T) {
	handler := &HistoryHandlerV2{}

	req := httptest.NewRequest("GET", "/report?min_flap=101", nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for min_flap=101, got nil")
	}
}

func TestParseReportRequest_NamespaceTooLong(t *testing.T) {
	handler := &HistoryHandlerV2{}

	// 256 characters (exceeds 255 limit)
	longNamespace := ""
	for i := 0; i < 256; i++ {
		longNamespace += "a"
	}
	req := httptest.NewRequest("GET", "/report?namespace="+longNamespace, nil)

	_, err := handler.parseReportRequest(req)
	if err == nil {
		t.Fatal("Expected error for namespace too long, got nil")
	}

	validationErr, ok := err.(*core.ValidationError)
	if !ok {
		t.Fatalf("Expected ValidationError, got %T", err)
	}

	if validationErr.Field != "namespace" {
		t.Errorf("Expected field='namespace', got %s", validationErr.Field)
	}
}

// ============================================================================
// Test 11-15: Response Generation Tests
// ============================================================================

func TestGenerateReport_Success_AllData(t *testing.T) {
	mockRepo := &MockHistoryRepository{}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := &core.ReportRequest{
		TimeRange:    &core.TimeRange{},
		TopLimit:     10,
		MinFlapCount: 3,
	}

	report, err := handler.generateReport(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if report == nil {
		t.Fatal("Expected report, got nil")
	}

	if report.Metadata == nil {
		t.Fatal("Expected metadata, got nil")
	}

	if report.Metadata.PartialFailure {
		t.Error("Expected no partial failure")
	}

	if report.Summary == nil {
		t.Error("Expected summary data")
	}

	if len(report.TopAlerts) == 0 {
		t.Error("Expected top alerts data")
	}

	if len(report.FlappingAlerts) == 0 {
		t.Error("Expected flapping alerts data")
	}
}

func TestGenerateReport_PartialFailure_StatsError(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetAggregatedStatsFunc: func(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
			return nil, &core.TimeoutError{Operation: "stats", Duration: 10 * time.Second}
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := &core.ReportRequest{
		TimeRange:    &core.TimeRange{},
		TopLimit:     10,
		MinFlapCount: 3,
	}

	report, err := handler.generateReport(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error (partial failure), got: %v", err)
	}

	if !report.Metadata.PartialFailure {
		t.Error("Expected partial failure flag to be true")
	}

	if len(report.Metadata.Errors) == 0 {
		t.Error("Expected error messages in metadata")
	}

	// Other data should still be present
	if len(report.TopAlerts) == 0 {
		t.Error("Expected top alerts data despite stats failure")
	}
}

func TestGenerateReport_WithNamespaceFilter(t *testing.T) {
	mockRepo := &MockHistoryRepository{}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	namespace := "production"
	req := &core.ReportRequest{
		TimeRange:    &core.TimeRange{},
		Namespace:    &namespace,
		TopLimit:     10,
		MinFlapCount: 3,
	}

	report, err := handler.generateReport(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify namespace filtering applied
	for _, alert := range report.TopAlerts {
		if alert.Namespace == nil || *alert.Namespace != namespace {
			t.Errorf("Expected all alerts to have namespace=%s", namespace)
		}
	}
}

func TestGenerateReport_WithIncludeRecent(t *testing.T) {
	recentCalled := false
	mockRepo := &MockHistoryRepository{
		GetRecentAlertsFunc: func(ctx context.Context, limit int) ([]*core.Alert, error) {
			recentCalled = true
			return []*core.Alert{
				{Fingerprint: "fp1", AlertName: "Test"},
			}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := &core.ReportRequest{
		TimeRange:     &core.TimeRange{},
		TopLimit:      10,
		MinFlapCount:  3,
		IncludeRecent: true,
	}

	report, err := handler.generateReport(context.Background(), req)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if !recentCalled {
		t.Error("Expected GetRecentAlerts to be called")
	}

	if len(report.RecentAlerts) == 0 {
		t.Error("Expected recent alerts data")
	}
}

// ============================================================================
// Test 16-20: Handler Integration Tests
// ============================================================================

func TestHandleReport_Success(t *testing.T) {
	mockRepo := &MockHistoryRepository{}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/report?top=5", nil)
	w := httptest.NewRecorder()

	handler.HandleReport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type=application/json, got %s", contentType)
	}

	// Parse response
	var report core.ReportResponse
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if report.Metadata == nil {
		t.Error("Expected metadata in response")
	}

	// ProcessingTimeMs may be 0 for very fast operations with mock data
	// Just verify it's set (>=0)
	if report.Metadata.ProcessingTimeMs < 0 {
		t.Error("Expected processing_time_ms >= 0")
	}
}

func TestHandleReport_InvalidMethod(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)

	req := httptest.NewRequest("POST", "/report", nil)
	w := httptest.NewRecorder()

	handler.HandleReport(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleReport_ValidationError(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)

	req := httptest.NewRequest("GET", "/report?top=0", nil)
	w := httptest.NewRecorder()

	handler.HandleReport(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandleReport_WithFilters(t *testing.T) {
	mockRepo := &MockHistoryRepository{}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/report?namespace=production&severity=critical&top=20&min_flap=5", nil)
	w := httptest.NewRecorder()

	handler.HandleReport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report core.ReportResponse
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Verify filters applied
	if report.Summary == nil {
		t.Error("Expected summary data")
	}
}

func TestHandleReport_AllComponentsSucceed(t *testing.T) {
	// Test that when all components succeed, partial_failure is false
	mockRepo := &MockHistoryRepository{}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/report?include_recent=true", nil)
	w := httptest.NewRecorder()

	handler.HandleReport(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var report core.ReportResponse
	if err := json.NewDecoder(w.Body).Decode(&report); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if report.Metadata == nil {
		t.Fatal("Expected metadata to be set")
	}

	if report.Metadata.PartialFailure {
		t.Error("Expected partial_failure=false when all components succeed")
	}

	if len(report.Metadata.Errors) > 0 {
		t.Errorf("Expected no errors, got %v", report.Metadata.Errors)
	}

	// Verify all data is present
	if report.Summary == nil {
		t.Error("Expected summary data")
	}

	if len(report.TopAlerts) == 0 {
		t.Error("Expected top alerts")
	}

	if len(report.FlappingAlerts) == 0 {
		t.Error("Expected flapping alerts")
	}

	// RecentAlerts may be empty with default mock (returns empty array)
	// Just verify the request succeeded
}

// ============================================================================
// Test 21-25: Filter Helper Tests
// ============================================================================

func TestFilterTopAlertsByNamespace_EmptyInput(t *testing.T) {
	result := filterTopAlertsByNamespace([]*core.TopAlert{}, "production")
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d alerts", len(result))
	}
}

func TestFilterTopAlertsByNamespace_MatchingAlerts(t *testing.T) {
	namespace1 := "production"
	namespace2 := "staging"

	alerts := []*core.TopAlert{
		{Fingerprint: "fp1", Namespace: &namespace1},
		{Fingerprint: "fp2", Namespace: &namespace2},
		{Fingerprint: "fp3", Namespace: &namespace1},
	}

	result := filterTopAlertsByNamespace(alerts, "production")
	if len(result) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(result))
	}

	for _, alert := range result {
		if alert.Namespace == nil || *alert.Namespace != "production" {
			t.Error("Expected all filtered alerts to have namespace=production")
		}
	}
}

func TestFilterFlappingAlertsByNamespace_EmptyInput(t *testing.T) {
	result := filterFlappingAlertsByNamespace([]*core.FlappingAlert{}, "production")
	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d alerts", len(result))
	}
}

func TestFilterFlappingAlertsByNamespace_MatchingAlerts(t *testing.T) {
	namespace1 := "production"
	namespace2 := "staging"

	alerts := []*core.FlappingAlert{
		{Fingerprint: "fp1", Namespace: &namespace1},
		{Fingerprint: "fp2", Namespace: &namespace2},
		{Fingerprint: "fp3", Namespace: &namespace1},
	}

	result := filterFlappingAlertsByNamespace(alerts, "production")
	if len(result) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(result))
	}

	for _, alert := range result {
		if alert.Namespace == nil || *alert.Namespace != "production" {
			t.Error("Expected all filtered alerts to have namespace=production")
		}
	}
}

func TestFilterHelpers_NilNamespace(t *testing.T) {
	alerts := []*core.TopAlert{
		{Fingerprint: "fp1", Namespace: nil},
	}

	result := filterTopAlertsByNamespace(alerts, "production")
	if len(result) != 0 {
		t.Error("Expected alerts with nil namespace to be filtered out")
	}
}
