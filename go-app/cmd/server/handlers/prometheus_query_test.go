// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ============================================================================
// Test 1-8: Query Parameter Parsing Tests
// ============================================================================

func TestParseQueryParameters_NoParams(t *testing.T) {
	query := url.Values{}
	params, err := ParseQueryParameters(query)

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Check defaults
	if params.Page != 1 {
		t.Errorf("Expected default page 1, got %d", params.Page)
	}
	if params.Limit != 100 {
		t.Errorf("Expected default limit 100, got %d", params.Limit)
	}
	if params.SortBy != "startsAt" {
		t.Errorf("Expected default sortBy 'startsAt', got %s", params.SortBy)
	}
	if params.SortOrder != "desc" {
		t.Errorf("Expected default sortOrder 'desc', got %s", params.SortOrder)
	}
}

func TestParseQueryParameters_AllParams(t *testing.T) {
	query := url.Values{
		"filter":    []string{`{alertname="HighCPU"}`},
		"receiver":  []string{"team-ops"},
		"silenced":  []string{"false"},
		"inhibited": []string{"false"},
		"active":    []string{"true"},
		"status":    []string{"firing"},
		"severity":  []string{"critical"},
		"startTime": []string{"2025-11-19T10:00:00Z"},
		"endTime":   []string{"2025-11-19T11:00:00Z"},
		"page":      []string{"2"},
		"limit":     []string{"50"},
		"sort":      []string{"severity:asc"},
	}

	params, err := ParseQueryParameters(query)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify all parameters
	if params.Filter != `{alertname="HighCPU"}` {
		t.Errorf("Filter mismatch")
	}
	if params.Receiver != "team-ops" {
		t.Errorf("Receiver mismatch")
	}
	if params.Silenced == nil || *params.Silenced != false {
		t.Errorf("Silenced mismatch")
	}
	if params.Status != "firing" {
		t.Errorf("Status mismatch")
	}
	if params.Page != 2 {
		t.Errorf("Page mismatch")
	}
	if params.Limit != 50 {
		t.Errorf("Limit mismatch")
	}
	if params.SortBy != "severity" || params.SortOrder != "asc" {
		t.Errorf("Sort mismatch")
	}
}

func TestParseQueryParameters_InvalidStatus(t *testing.T) {
	query := url.Values{
		"status": []string{"invalid"},
	}

	_, err := ParseQueryParameters(query)
	if err == nil {
		t.Fatal("Expected error for invalid status")
	}
	if !containsAny(err.Error(), []string{"invalid status"}) {
		t.Errorf("Expected 'invalid status' error, got: %v", err)
	}
}

func TestParseQueryParameters_InvalidBool(t *testing.T) {
	query := url.Values{
		"silenced": []string{"maybe"},
	}

	_, err := ParseQueryParameters(query)
	if err == nil {
		t.Fatal("Expected error for invalid boolean")
	}
}

func TestParseQueryParameters_InvalidPage(t *testing.T) {
	query := url.Values{
		"page": []string{"-1"},
	}

	_, err := ParseQueryParameters(query)
	if err == nil {
		t.Fatal("Expected error for negative page")
	}
}

func TestParseQueryParameters_LimitTooLarge(t *testing.T) {
	query := url.Values{
		"limit": []string{"5000"},
	}

	_, err := ParseQueryParameters(query)
	if err == nil {
		t.Fatal("Expected error for limit exceeding max")
	}
	if !containsAny(err.Error(), []string{"exceeds maximum"}) {
		t.Errorf("Expected 'exceeds maximum' error, got: %v", err)
	}
}

func TestParseQueryParameters_InvalidTimeFormat(t *testing.T) {
	query := url.Values{
		"startTime": []string{"invalid-time"},
	}

	_, err := ParseQueryParameters(query)
	if err == nil {
		t.Fatal("Expected error for invalid time format")
	}
}

func TestParseQueryParameters_TimeRangeInvalid(t *testing.T) {
	query := url.Values{
		"startTime": []string{"2025-11-19T11:00:00Z"},
		"endTime":   []string{"2025-11-19T10:00:00Z"},
	}

	_, err := ParseQueryParameters(query)
	if err == nil {
		t.Fatal("Expected error for startTime after endTime")
	}
}

// ============================================================================
// Test 9-13: Label Matcher Parsing Tests
// ============================================================================

func TestParseLabelMatchers_Empty(t *testing.T) {
	matchers, err := ParseLabelMatchers("{}")
	if err != nil {
		t.Fatalf("Expected no error for empty matcher, got: %v", err)
	}
	if len(matchers) != 0 {
		t.Errorf("Expected 0 matchers, got %d", len(matchers))
	}
}

func TestParseLabelMatchers_SingleExact(t *testing.T) {
	matchers, err := ParseLabelMatchers(`{alertname="HighCPU"}`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(matchers) != 1 {
		t.Fatalf("Expected 1 matcher, got %d", len(matchers))
	}

	if matchers[0].Name != "alertname" || matchers[0].Operator != "=" || matchers[0].Value != "HighCPU" {
		t.Errorf("Matcher mismatch: %+v", matchers[0])
	}
}

func TestParseLabelMatchers_MultipleMatchers(t *testing.T) {
	matchers, err := ParseLabelMatchers(`{alertname="HighCPU",severity="critical",instance=~"node-.*"}`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(matchers) != 3 {
		t.Fatalf("Expected 3 matchers, got %d", len(matchers))
	}

	// Verify regex matcher
	if matchers[2].Operator != "=~" {
		t.Errorf("Expected regex operator =~, got %s", matchers[2].Operator)
	}
}

func TestParseLabelMatchers_NegativeMatchers(t *testing.T) {
	matchers, err := ParseLabelMatchers(`{severity!="info",alertname!~"Test.*"}`)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(matchers) != 2 {
		t.Fatalf("Expected 2 matchers, got %d", len(matchers))
	}

	if matchers[0].Operator != "!=" || matchers[1].Operator != "!~" {
		t.Errorf("Negative operator mismatch")
	}
}

func TestParseLabelMatchers_InvalidSyntax(t *testing.T) {
	testCases := []string{
		"alertname=HighCPU",      // Missing braces
		"{alertname=HighCPU}",    // Missing quotes
		"{alertname=\"HighCPU\"", // Unclosed brace
	}

	for _, tc := range testCases {
		_, err := ParseLabelMatchers(tc)
		if err == nil {
			t.Errorf("Expected error for invalid syntax: %s", tc)
		}
	}
}

// ============================================================================
// Test 14-16: Validation Tests
// ============================================================================

func TestValidateQueryParameters_Valid(t *testing.T) {
	params := &QueryParameters{
		Page:      1,
		Limit:     100,
		StartTime: time.Now().Add(-1 * time.Hour),
		EndTime:   time.Now(),
	}

	result := ValidateQueryParameters(params)
	if !result.Valid {
		t.Errorf("Expected valid parameters, got errors: %v", result.Errors)
	}
}

func TestValidateQueryParameters_InvalidPage(t *testing.T) {
	params := &QueryParameters{
		Page:  0,
		Limit: 100,
	}

	result := ValidateQueryParameters(params)
	if result.Valid {
		t.Error("Expected validation to fail for page=0")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected validation errors")
	}
}

func TestValidateQueryParameters_InvalidLimit(t *testing.T) {
	params := &QueryParameters{
		Page:  1,
		Limit: 2000, // Exceeds MaxAlertsPerPage
	}

	result := ValidateQueryParameters(params)
	if result.Valid {
		t.Error("Expected validation to fail for limit > max")
	}
}

// ============================================================================
// Test 17-20: Format Conversion Tests
// ============================================================================

func TestConvertToAlertmanagerFormat_Empty(t *testing.T) {
	ctx := context.Background()
	deps := &ConverterDependencies{}

	result, err := ConvertToAlertmanagerFormat(ctx, nil, deps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result for nil input, got %d alerts", len(result))
	}
}

func TestConvertToAlertmanagerFormat_SingleAlert(t *testing.T) {
	ctx := context.Background()
	deps := &ConverterDependencies{}

	startsAt := time.Now()
	alerts := []*core.Alert{
		{
			Fingerprint: "abc123",
			AlertName:   "HighCPU",
			Status:      core.StatusFiring,
			Labels: map[string]string{
				"alertname": "HighCPU",
				"severity":  "critical",
			},
			Annotations: map[string]string{
				"summary": "CPU usage high",
			},
			StartsAt: startsAt,
		},
	}

	result, err := ConvertToAlertmanagerFormat(ctx, alerts, deps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(result))
	}

	amAlert := result[0]
	if amAlert.Fingerprint != "abc123" {
		t.Errorf("Fingerprint mismatch")
	}
	if amAlert.Labels["alertname"] != "HighCPU" {
		t.Errorf("Label mismatch")
	}
	if amAlert.Status.State != "active" {
		t.Errorf("Expected state 'active', got %s", amAlert.Status.State)
	}
}

func TestConvertToAlertmanagerFormat_ResolvedAlert(t *testing.T) {
	ctx := context.Background()
	deps := &ConverterDependencies{}

	startsAt := time.Now().Add(-1 * time.Hour)
	endsAt := time.Now()

	alerts := []*core.Alert{
		{
			Fingerprint: "def456",
			AlertName:   "HighCPU",
			Status:      core.StatusResolved,
			Labels:      map[string]string{"alertname": "HighCPU"},
			Annotations: map[string]string{},
			StartsAt:    startsAt,
			EndsAt:      &endsAt,
		},
	}

	result, err := ConvertToAlertmanagerFormat(ctx, alerts, deps)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("Expected 1 alert, got %d", len(result))
	}

	if result[0].EndsAt == "" {
		t.Error("Expected endsAt to be set for resolved alert")
	}
}

func TestBuildAlertmanagerListResponse(t *testing.T) {
	alerts := []AlertmanagerAlert{
		{Fingerprint: "abc123"},
		{Fingerprint: "def456"},
	}

	response := BuildAlertmanagerListResponse(alerts, 42, 1, 100)

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got %s", response.Status)
	}
	if response.Data == nil {
		t.Fatal("Expected data to be non-nil")
	}
	if len(response.Data.Alerts) != 2 {
		t.Errorf("Expected 2 alerts, got %d", len(response.Data.Alerts))
	}
	if response.Data.Total != 42 {
		t.Errorf("Expected total 42, got %d", response.Data.Total)
	}
}

// ============================================================================
// Test 21-25: HTTP Handler Tests
// ============================================================================

// mockHistoryRepo implements AlertHistoryRepository for testing
type mockHistoryRepo struct {
	alerts []*core.Alert
	err    error
}

func (m *mockHistoryRepo) GetHistory(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
	if m.err != nil {
		return nil, m.err
	}

	return &core.HistoryResponse{
		Alerts:     m.alerts,
		Total:      int64(len(m.alerts)),
		Page:       req.Pagination.Page,
		PerPage:    req.Pagination.PerPage,
		TotalPages: 1,
		HasNext:    false,
		HasPrev:    false,
	}, nil
}

func TestHandlePrometheusQuery_MethodNotAllowed(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false // Disable metrics for testing

	handler, err := NewPrometheusQueryHandler(repo, nil, config, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v2/alerts", nil)
	w := httptest.NewRecorder()

	handler.HandlePrometheusQuery(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandlePrometheusQuery_Success(t *testing.T) {
	// Setup mock repository with sample alerts
	startsAt := time.Now()
	repo := &mockHistoryRepo{
		alerts: []*core.Alert{
			{
				Fingerprint: "test123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
				Labels:      map[string]string{"alertname": "TestAlert"},
				Annotations: map[string]string{"summary": "Test"},
				StartsAt:    startsAt,
			},
		},
	}

	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false // Disable metrics for testing

	handler, err := NewPrometheusQueryHandler(repo, nil, config, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/alerts?limit=10", nil)
	w := httptest.NewRecorder()

	handler.HandlePrometheusQuery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response AlertmanagerListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected status 'success', got %s", response.Status)
	}

	if len(response.Data.Alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(response.Data.Alerts))
	}
}

func TestHandlePrometheusQuery_InvalidParameters(t *testing.T) {
	repo := &mockHistoryRepo{}
	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, err := NewPrometheusQueryHandler(repo, nil, config, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	// Invalid limit (too large)
	req := httptest.NewRequest(http.MethodGet, "/api/v2/alerts?limit=5000", nil)
	w := httptest.NewRecorder()

	handler.HandlePrometheusQuery(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}

func TestHandlePrometheusQuery_DatabaseError(t *testing.T) {
	repo := &mockHistoryRepo{
		err: context.DeadlineExceeded,
	}

	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, err := NewPrometheusQueryHandler(repo, nil, config, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/alerts", nil)
	w := httptest.NewRecorder()

	handler.HandlePrometheusQuery(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandlePrometheusQuery_WithFilters(t *testing.T) {
	startsAt := time.Now()
	repo := &mockHistoryRepo{
		alerts: []*core.Alert{
			{
				Fingerprint: "test123",
				AlertName:   "HighCPU",
				Status:      core.StatusFiring,
				Labels: map[string]string{
					"alertname": "HighCPU",
					"severity":  "critical",
				},
				Annotations: map[string]string{},
				StartsAt:    startsAt,
			},
		},
	}

	config := DefaultPrometheusQueryConfig()
	config.EnableMetrics = false

	handler, err := NewPrometheusQueryHandler(repo, nil, config, nil)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v2/alerts?status=firing&severity=critical&page=1&limit=50&sort=startsAt:desc", nil)
	w := httptest.NewRecorder()

	handler.HandlePrometheusQuery(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d, body: %s", w.Code, w.Body.String())
	}

	var response AlertmanagerListResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Status != "success" {
		t.Errorf("Expected success status")
	}

	// Verify pagination metadata
	if response.Data.Page != 1 || response.Data.Limit != 50 {
		t.Errorf("Pagination metadata mismatch")
	}
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestParseBool(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
		wantErr  bool
	}{
		{"true", true, false},
		{"TRUE", true, false},
		{"1", true, false},
		{"yes", true, false},
		{"on", true, false},
		{"false", false, false},
		{"FALSE", false, false},
		{"0", false, false},
		{"no", false, false},
		{"off", false, false},
		{"invalid", false, true},
	}

	for _, tc := range testCases {
		result, err := parseBool(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("parseBool(%q): expected error", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("parseBool(%q): unexpected error: %v", tc.input, err)
		}
		if !tc.wantErr && result != tc.expected {
			t.Errorf("parseBool(%q): expected %v, got %v", tc.input, tc.expected, result)
		}
	}
}

func TestParseSortParameter(t *testing.T) {
	testCases := []struct {
		input         string
		expectedField string
		expectedOrder string
		wantErr       bool
	}{
		{"startsAt:desc", "startsAt", "desc", false},
		{"severity:asc", "severity", "asc", false},
		{"startsAt", "startsAt", "desc", false}, // Default order
		{"invalid:desc", "", "", true},
		{"startsAt:invalid", "", "", true},
	}

	for _, tc := range testCases {
		field, order, err := parseSortParameter(tc.input)
		if tc.wantErr && err == nil {
			t.Errorf("parseSortParameter(%q): expected error", tc.input)
		}
		if !tc.wantErr && err != nil {
			t.Errorf("parseSortParameter(%q): unexpected error: %v", tc.input, err)
		}
		if !tc.wantErr && (field != tc.expectedField || order != tc.expectedOrder) {
			t.Errorf("parseSortParameter(%q): expected (%s, %s), got (%s, %s)",
				tc.input, tc.expectedField, tc.expectedOrder, field, order)
		}
	}
}
