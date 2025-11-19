// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ============================================================================
// Additional Tests for 85%+ Coverage
// ============================================================================

// Test label matcher conversion (currently 0% coverage)
func TestConvertLabelMatchers(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	// Test exact match
	matchers := []LabelMatcher{
		{Name: "alertname", Operator: "=", Value: "HighCPU"},
		{Name: "severity", Operator: "=", Value: "critical"},
	}

	result := handler.convertLabelMatchers(matchers)
	if len(result) != 2 {
		t.Errorf("Expected 2 matchers, got %d", len(result))
	}
	if result["alertname"] != "HighCPU" {
		t.Errorf("alertname mismatch")
	}

	// Test non-exact match (should be ignored in simplified implementation)
	matchers2 := []LabelMatcher{
		{Name: "test", Operator: "=~", Value: ".*"},
	}
	result2 := handler.convertLabelMatchers(matchers2)
	if len(result2) != 0 {
		t.Errorf("Expected regex matchers to be ignored, got %d", len(result2))
	}
}

// Test sort field mapping (currently 25% coverage)
func TestMapSortField(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	testCases := []struct {
		input    string
		expected string
	}{
		{"startsAt", "starts_at"},
		{"endsAt", "ends_at"},
		{"severity", "severity"},
		{"alertname", "alert_name"},
		{"fingerprint", "fingerprint"},
		{"status", "status"},
		{"invalid", "starts_at"}, // Default
	}

	for _, tc := range testCases {
		result := handler.mapSortField(tc.input)
		if result != tc.expected {
			t.Errorf("mapSortField(%q): expected %q, got %q", tc.input, tc.expected, result)
		}
	}
}

// Test sort order mapping (currently 66.7% coverage)
func TestMapSortOrder(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	// Test asc
	if result := handler.mapSortOrder("asc"); result != core.SortOrderAsc {
		t.Errorf("Expected SortOrderAsc, got %v", result)
	}

	// Test desc
	if result := handler.mapSortOrder("desc"); result != core.SortOrderDesc {
		t.Errorf("Expected SortOrderDesc, got %v", result)
	}

	// Test default
	if result := handler.mapSortOrder("invalid"); result != core.SortOrderDesc {
		t.Errorf("Expected SortOrderDesc (default), got %v", result)
	}
}

// Test validation error response (currently 0% coverage)
func TestRespondValidationError(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	w := httptest.NewRecorder()
	result := &QueryValidationResult{
		Valid: false,
		Errors: []QueryValidationError{
			{Parameter: "limit", Message: "too large", Value: 5000},
			{Parameter: "page", Message: "must be positive", Value: -1},
		},
	}

	handler.respondValidationError(w, result)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected JSON content type")
	}
}

// Test buildHistoryRequest coverage (currently 55%)
func TestBuildHistoryRequest_AllFilters(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	params := &QueryParameters{
		Status:    "firing",
		Severity:  "critical",
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
		Filter:    `{alertname="HighCPU"}`,
		Page:      2,
		Limit:     50,
		SortBy:    "severity",
		SortOrder: "asc",
	}

	req, err := handler.buildHistoryRequest(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if req.Filters.Status == nil {
		t.Error("Expected status filter to be set")
	}
	if req.Filters.Severity == nil {
		t.Error("Expected severity filter to be set")
	}
	if req.Filters.TimeRange == nil {
		t.Error("Expected time range to be set")
	}
	if req.Pagination.Page != 2 {
		t.Errorf("Expected page 2, got %d", req.Pagination.Page)
	}
	if req.Pagination.PerPage != 50 {
		t.Errorf("Expected limit 50, got %d", req.Pagination.PerPage)
	}
}

func TestBuildHistoryRequest_MinimalParams(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	params := &QueryParameters{
		Page:      1,
		Limit:     100,
		SortBy:    "startsAt",
		SortOrder: "desc",
	}

	req, err := handler.buildHistoryRequest(params)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if req.Filters == nil {
		t.Error("Expected filters to be initialized")
	}
	if req.Pagination == nil {
		t.Error("Expected pagination to be initialized")
	}
	if req.Sorting == nil {
		t.Error("Expected sorting to be initialized")
	}
}

func TestBuildHistoryRequest_InvalidFilter(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	params := &QueryParameters{
		Filter: "{invalid syntax",
		Page:   1,
		Limit:  100,
	}

	_, err := handler.buildHistoryRequest(params)
	if err == nil {
		t.Error("Expected error for invalid filter")
	}
}

// Test buildAlertStatus coverage (currently 45%)
func TestBuildAlertStatus_WithSilences(t *testing.T) {
	// Mock silence checker
	silences := []string{"silence-1", "silence-2"}
	mockChecker := &mockSilenceChecker{silences: silences}

	deps := &ConverterDependencies{
		SilenceChecker: mockChecker,
	}

	alert := &core.Alert{
		Fingerprint: "test123",
		Status:      core.StatusFiring,
	}

	ctx := context.Background()
	status, err := buildAlertStatus(ctx, alert, deps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.State != "suppressed" {
		t.Errorf("Expected state 'suppressed', got %s", status.State)
	}
	if len(status.SilencedBy) != 2 {
		t.Errorf("Expected 2 silences, got %d", len(status.SilencedBy))
	}
}

func TestBuildAlertStatus_WithInhibitions(t *testing.T) {
	// Mock inhibition checker
	inhibitors := []string{"alert-abc", "alert-def"}
	mockChecker := &mockInhibitionChecker{inhibitors: inhibitors}

	deps := &ConverterDependencies{
		InhibitionChecker: mockChecker,
	}

	alert := &core.Alert{
		Fingerprint: "test123",
		Status:      core.StatusFiring,
	}

	ctx := context.Background()
	status, err := buildAlertStatus(ctx, alert, deps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.State != "suppressed" {
		t.Errorf("Expected state 'suppressed', got %s", status.State)
	}
	if len(status.InhibitedBy) != 2 {
		t.Errorf("Expected 2 inhibitors, got %d", len(status.InhibitedBy))
	}
}

func TestBuildAlertStatus_NoDependencies(t *testing.T) {
	deps := &ConverterDependencies{}

	alert := &core.Alert{
		Fingerprint: "test123",
		Status:      core.StatusFiring,
	}

	ctx := context.Background()
	status, err := buildAlertStatus(ctx, alert, deps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if status.State != "active" {
		t.Errorf("Expected state 'active', got %s", status.State)
	}
	if len(status.SilencedBy) != 0 {
		t.Errorf("Expected no silences, got %d", len(status.SilencedBy))
	}
	if len(status.InhibitedBy) != 0 {
		t.Errorf("Expected no inhibitors, got %d", len(status.InhibitedBy))
	}
}

// Test buildReceivers coverage (currently 66.7%)
func TestBuildReceivers_WithLabel(t *testing.T) {
	alert := &core.Alert{
		Labels: map[string]string{
			"receiver": "team-ops",
		},
	}

	receivers := buildReceivers(alert)
	if len(receivers) != 1 {
		t.Fatalf("Expected 1 receiver, got %d", len(receivers))
	}
	if receivers[0] != "team-ops" {
		t.Errorf("Expected 'team-ops', got %s", receivers[0])
	}
}

func TestBuildReceivers_NoLabel(t *testing.T) {
	alert := &core.Alert{
		Labels: map[string]string{},
	}

	receivers := buildReceivers(alert)
	if len(receivers) != 1 {
		t.Fatalf("Expected 1 receiver (default), got %d", len(receivers))
	}
	if receivers[0] != "default" {
		t.Errorf("Expected 'default', got %s", receivers[0])
	}
}

// Test copyLabels coverage (currently 83.3%)
func TestCopyLabels_Nil(t *testing.T) {
	result := copyLabels(nil)
	if result == nil {
		t.Error("Expected non-nil map for nil input")
	}
	if len(result) != 0 {
		t.Errorf("Expected empty map, got %d elements", len(result))
	}
}

func TestCopyLabels_WithData(t *testing.T) {
	input := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	result := copyLabels(input)
	if len(result) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(result))
	}

	// Verify it's a deep copy
	result["key1"] = "modified"
	if input["key1"] == "modified" {
		t.Error("copyLabels should create a deep copy")
	}
}

// Test respondSuccess and respondError (currently 75-80%)
func TestRespondSuccess(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	w := httptest.NewRecorder()
	response := BuildAlertmanagerListResponse([]AlertmanagerAlert{}, 0, 1, 100)

	handler.respondSuccess(w, response)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Expected JSON content type")
	}
}

func TestRespondError(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, _ := NewPrometheusQueryHandler(repo, nil, config, nil)

	w := httptest.NewRecorder()
	handler.respondError(w, http.StatusBadRequest, "test error")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

// Mock implementations for testing
type mockSilenceChecker struct {
	silences []string
	err      error
}

func (m *mockSilenceChecker) IsAlertSilenced(ctx context.Context, alert *core.Alert) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.silences, nil
}

type mockInhibitionChecker struct {
	inhibitors []string
	err        error
}

func (m *mockInhibitionChecker) IsAlertInhibited(ctx context.Context, alert *core.Alert) ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.inhibitors, nil
}

// Test context cancellation in converter
func TestConvertToAlertmanagerFormat_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	alerts := []*core.Alert{
		{Fingerprint: "test"},
	}

	_, err := ConvertToAlertmanagerFormat(ctx, alerts, nil)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// Test error responses
func TestBuildErrorResponse(t *testing.T) {
	response := BuildErrorResponse("test error")

	if response.Status != "error" {
		t.Errorf("Expected status 'error', got %s", response.Status)
	}
	if response.Error != "test error" {
		t.Errorf("Expected error message 'test error', got %s", response.Error)
	}
	if response.Data != nil {
		t.Error("Expected data to be nil for error response")
	}
}
