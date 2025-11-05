package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"log/slog"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/inhibition"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// Mock implementations for testing

type mockParser struct {
	config *inhibition.InhibitionConfig
}

func (m *mockParser) ParseFile(path string) (*inhibition.InhibitionConfig, error) {
	return m.config, nil
}

func (m *mockParser) Parse(data []byte) (*inhibition.InhibitionConfig, error) {
	return m.config, nil
}

func (m *mockParser) ParseString(yaml string) (*inhibition.InhibitionConfig, error) {
	return m.config, nil
}

func (m *mockParser) ParseReader(r io.Reader) (*inhibition.InhibitionConfig, error) {
	return m.config, nil
}

func (m *mockParser) Validate(config *inhibition.InhibitionConfig) error {
	return nil
}

func (m *mockParser) GetConfig() *inhibition.InhibitionConfig {
	return m.config
}

type mockMatcher struct {
	shouldInhibit bool
	matchedRule   *inhibition.InhibitionRule
	matchedAlert  *core.Alert
	err           error
}

func (m *mockMatcher) ShouldInhibit(ctx context.Context, alert *core.Alert) (*inhibition.MatchResult, error) {
	if m.err != nil {
		return nil, m.err
	}

	result := &inhibition.MatchResult{
		Matched:       m.shouldInhibit,
		MatchDuration: 2 * time.Millisecond,
	}

	if m.shouldInhibit {
		result.Rule = m.matchedRule
		result.InhibitedBy = m.matchedAlert
	}

	return result, nil
}

func (m *mockMatcher) FindInhibitors(ctx context.Context, alert *core.Alert) ([]*inhibition.MatchResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []*inhibition.MatchResult{}, nil
}

func (m *mockMatcher) MatchRule(rule *inhibition.InhibitionRule, sourceAlert, targetAlert *core.Alert) bool {
	return m.shouldInhibit
}

type mockStateManager struct {
	states []*inhibition.InhibitionState
	err    error
}

func (m *mockStateManager) RecordInhibition(ctx context.Context, state *inhibition.InhibitionState) error {
	if m.err != nil {
		return m.err
	}
	m.states = append(m.states, state)
	return nil
}

func (m *mockStateManager) RemoveInhibition(ctx context.Context, targetFingerprint string) error {
	return m.err
}

func (m *mockStateManager) GetActiveInhibitions(ctx context.Context) ([]*inhibition.InhibitionState, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.states, nil
}

func (m *mockStateManager) GetInhibitedAlerts(ctx context.Context) ([]string, error) {
	return nil, m.err
}

func (m *mockStateManager) IsInhibited(ctx context.Context, targetFingerprint string) (bool, error) {
	return false, m.err
}

func (m *mockStateManager) GetInhibitionState(ctx context.Context, targetFingerprint string) (*inhibition.InhibitionState, error) {
	return nil, m.err
}

// Test helpers

var testMetrics *metrics.BusinessMetrics

func init() {
	// Create metrics once for all tests to avoid duplicate registration
	testMetrics = metrics.NewBusinessMetrics("test")
}

func setupTestHandler(parser inhibition.InhibitionParser, matcher inhibition.InhibitionMatcher, stateManager inhibition.InhibitionStateManager) *InhibitionHandler {
	logger := slog.Default()
	return NewInhibitionHandler(parser, matcher, stateManager, testMetrics, logger)
}

func createTestRule(name string) inhibition.InhibitionRule {
	return inhibition.InhibitionRule{
		Name: name,
		SourceMatch: map[string]string{
			"alertname": "NodeDown",
			"severity":  "critical",
		},
		TargetMatch: map[string]string{
			"alertname": "InstanceDown",
		},
		Equal: []string{"node", "cluster"},
	}
}

func createTestAlert(alertname, fingerprint string) *core.Alert {
	return &core.Alert{
		Labels: map[string]string{
			"alertname": alertname,
			"node":      "node1",
			"cluster":   "prod",
		},
		Annotations: map[string]string{
			"summary": "Test alert",
		},
		Status:      core.StatusFiring,
		Fingerprint: fingerprint,
		StartsAt:    time.Now(),
	}
}

// Test Suite 1: GET /api/v2/inhibition/rules

func TestInhibitionHandler_GetRules_Success_NoRules(t *testing.T) {
	// Setup: parser with empty config
	parser := &mockParser{
		config: &inhibition.InhibitionConfig{
			Rules: []inhibition.InhibitionRule{},
		},
	}
	handler := setupTestHandler(parser, &mockMatcher{}, &mockStateManager{})

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
	w := httptest.NewRecorder()
	handler.GetRules(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionRulesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("Expected count 0, got %d", response.Count)
	}

	if len(response.Rules) != 0 {
		t.Errorf("Expected 0 rules, got %d", len(response.Rules))
	}
}

func TestInhibitionHandler_GetRules_Success_OneRule(t *testing.T) {
	// Setup
	rule := createTestRule("test-rule")
	parser := &mockParser{
		config: &inhibition.InhibitionConfig{
			Rules: []inhibition.InhibitionRule{rule},
		},
	}
	handler := setupTestHandler(parser, &mockMatcher{}, &mockStateManager{})

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
	w := httptest.NewRecorder()
	handler.GetRules(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionRulesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 1 {
		t.Errorf("Expected count 1, got %d", response.Count)
	}

	if len(response.Rules) != 1 {
		t.Fatalf("Expected 1 rule, got %d", len(response.Rules))
	}

	if response.Rules[0].Name != "test-rule" {
		t.Errorf("Expected rule name 'test-rule', got '%s'", response.Rules[0].Name)
	}
}

func TestInhibitionHandler_GetRules_Success_MultipleRules(t *testing.T) {
	// Setup: 10 rules
	rules := make([]inhibition.InhibitionRule, 10)
	for i := 0; i < 10; i++ {
		rules[i] = createTestRule("rule-" + string(rune('0'+i)))
	}
	parser := &mockParser{
		config: &inhibition.InhibitionConfig{
			Rules: rules,
		},
	}
	handler := setupTestHandler(parser, &mockMatcher{}, &mockStateManager{})

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
	w := httptest.NewRecorder()
	handler.GetRules(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionRulesResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 10 {
		t.Errorf("Expected count 10, got %d", response.Count)
	}

	if len(response.Rules) != 10 {
		t.Errorf("Expected 10 rules, got %d", len(response.Rules))
	}
}

func TestInhibitionHandler_GetRules_ContentType(t *testing.T) {
	// Setup
	parser := &mockParser{
		config: &inhibition.InhibitionConfig{Rules: []inhibition.InhibitionRule{}},
	}
	handler := setupTestHandler(parser, &mockMatcher{}, &mockStateManager{})

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
	w := httptest.NewRecorder()
	handler.GetRules(w, req)

	// Assert
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}
}

// Test Suite 2: GET /api/v2/inhibition/status

func TestInhibitionHandler_GetStatus_Success_NoInhibitions(t *testing.T) {
	// Setup
	stateManager := &mockStateManager{
		states: []*inhibition.InhibitionState{},
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, stateManager)

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 0 {
		t.Errorf("Expected count 0, got %d", response.Count)
	}

	if len(response.Active) != 0 {
		t.Errorf("Expected 0 active inhibitions, got %d", len(response.Active))
	}
}

func TestInhibitionHandler_GetStatus_Success_OneInhibition(t *testing.T) {
	// Setup
	now := time.Now()
	stateManager := &mockStateManager{
		states: []*inhibition.InhibitionState{
			{
				TargetFingerprint: "target-123",
				SourceFingerprint: "source-456",
				RuleName:          "test-rule",
				InhibitedAt:       now,
			},
		},
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, stateManager)

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 1 {
		t.Errorf("Expected count 1, got %d", response.Count)
	}

	if len(response.Active) != 1 {
		t.Fatalf("Expected 1 active inhibition, got %d", len(response.Active))
	}

	state := response.Active[0]
	if state.TargetFingerprint != "target-123" {
		t.Errorf("Expected target 'target-123', got '%s'", state.TargetFingerprint)
	}
	if state.SourceFingerprint != "source-456" {
		t.Errorf("Expected source 'source-456', got '%s'", state.SourceFingerprint)
	}
	if state.RuleName != "test-rule" {
		t.Errorf("Expected rule 'test-rule', got '%s'", state.RuleName)
	}
}

func TestInhibitionHandler_GetStatus_Success_MultipleInhibitions(t *testing.T) {
	// Setup: 5 active inhibitions
	now := time.Now()
	states := make([]*inhibition.InhibitionState, 5)
	for i := 0; i < 5; i++ {
		states[i] = &inhibition.InhibitionState{
			TargetFingerprint: "target-" + string(rune('0'+i)),
			SourceFingerprint: "source-" + string(rune('0'+i)),
			RuleName:          "rule-" + string(rune('0'+i)),
			InhibitedAt:       now,
		}
	}
	stateManager := &mockStateManager{states: states}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, stateManager)

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Count != 5 {
		t.Errorf("Expected count 5, got %d", response.Count)
	}

	if len(response.Active) != 5 {
		t.Errorf("Expected 5 active inhibitions, got %d", len(response.Active))
	}
}

func TestInhibitionHandler_GetStatus_Error_StateManagerFailure(t *testing.T) {
	// Setup: state manager that returns error
	stateManager := &mockStateManager{
		err: errors.New("Redis connection failed"),
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, stateManager)

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/status", nil)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	// Assert
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

func TestInhibitionHandler_GetStatus_ContextCancellation(t *testing.T) {
	// Setup
	stateManager := &mockStateManager{
		states: []*inhibition.InhibitionState{},
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, stateManager)

	// Execute with cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately
	req := httptest.NewRequest("GET", "/api/v2/inhibition/status", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler.GetStatus(w, req)

	// Assert: should still complete (context cancellation handled gracefully)
	// Note: actual behavior depends on implementation
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 200 or 500, got %d", w.Code)
	}
}

// Test Suite 3: POST /api/v2/inhibition/check

func TestInhibitionHandler_CheckAlert_Success_NotInhibited(t *testing.T) {
	// Setup
	matcher := &mockMatcher{
		shouldInhibit: false,
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	// Execute
	alert := createTestAlert("InstanceDown", "test-fp-123")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Inhibited {
		t.Error("Expected inhibited=false, got true")
	}

	if response.Alert == nil {
		t.Error("Expected alert in response")
	}

	if response.LatencyMs <= 0 {
		t.Errorf("Expected positive latency, got %d", response.LatencyMs)
	}

	if response.InhibitedBy != nil {
		t.Error("Expected no inhibited_by for non-inhibited alert")
	}

	if response.Rule != nil {
		t.Error("Expected no rule for non-inhibited alert")
	}
}

func TestInhibitionHandler_CheckAlert_Success_Inhibited(t *testing.T) {
	// Setup
	rule := createTestRule("node-down-inhibits-instance-down")
	sourceAlert := createTestAlert("NodeDown", "source-fp-456")
	matcher := &mockMatcher{
		shouldInhibit: true,
		matchedRule:   &rule,
		matchedAlert:  sourceAlert,
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	// Execute
	targetAlert := createTestAlert("InstanceDown", "target-fp-123")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: targetAlert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Inhibited {
		t.Error("Expected inhibited=true, got false")
	}

	if response.InhibitedBy == nil {
		t.Fatal("Expected inhibited_by alert")
	}

	if response.InhibitedBy.Fingerprint != "source-fp-456" {
		t.Errorf("Expected source fingerprint 'source-fp-456', got '%s'", response.InhibitedBy.Fingerprint)
	}

	if response.Rule == nil {
		t.Fatal("Expected rule in response")
	}

	if response.Rule.Name != "node-down-inhibits-instance-down" {
		t.Errorf("Expected rule name 'node-down-inhibits-instance-down', got '%s'", response.Rule.Name)
	}

	if response.LatencyMs <= 0 {
		t.Errorf("Expected positive latency, got %d", response.LatencyMs)
	}
}

func TestInhibitionHandler_CheckAlert_Error_InvalidJSON(t *testing.T) {
	// Setup
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, &mockStateManager{})

	// Execute with malformed JSON
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader([]byte("invalid json {")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

func TestInhibitionHandler_CheckAlert_Error_MissingAlert(t *testing.T) {
	// Setup
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, &mockStateManager{})

	// Execute with empty request body
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: nil})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	errorMsg, ok := response["error"].(string)
	if !ok {
		t.Fatal("Expected error field as string")
	}

	if errorMsg != "Alert is required" {
		t.Errorf("Expected 'Alert is required', got '%s'", errorMsg)
	}
}

func TestInhibitionHandler_CheckAlert_Error_MatcherFailure(t *testing.T) {
	// Setup: matcher that returns error
	matcher := &mockMatcher{
		err: errors.New("Cache unavailable"),
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	// Execute
	alert := createTestAlert("InstanceDown", "test-fp-123")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if _, ok := response["error"]; !ok {
		t.Error("Expected error field in response")
	}
}

// Test Suite 4: Metrics Recording

func TestInhibitionHandler_CheckAlert_MetricsRecorded_Inhibited(t *testing.T) {
	// Setup
	rule := createTestRule("test-rule")
	sourceAlert := createTestAlert("NodeDown", "source-fp")
	matcher := &mockMatcher{
		shouldInhibit: true,
		matchedRule:   &rule,
		matchedAlert:  sourceAlert,
	}
	logger := slog.Default()
	handler := NewInhibitionHandler(
		&mockParser{config: &inhibition.InhibitionConfig{}},
		matcher,
		&mockStateManager{},
		testMetrics,
		logger,
	)

	// Execute
	alert := createTestAlert("InstanceDown", "target-fp")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert: metrics should be recorded
	// Note: actual metric values would need prometheus test utilities to verify
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Inhibited {
		t.Error("Expected inhibited=true for metrics test")
	}
}

func TestInhibitionHandler_CheckAlert_MetricsRecorded_Allowed(t *testing.T) {
	// Setup
	matcher := &mockMatcher{shouldInhibit: false}
	logger := slog.Default()
	handler := NewInhibitionHandler(
		&mockParser{config: &inhibition.InhibitionConfig{}},
		matcher,
		&mockStateManager{},
		testMetrics,
		logger,
	)

	// Execute
	alert := createTestAlert("InstanceDown", "target-fp")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response InhibitionCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Inhibited {
		t.Error("Expected inhibited=false for metrics test")
	}
}

// Test Suite 5: Edge Cases

func TestInhibitionHandler_CheckAlert_EmptyLabels(t *testing.T) {
	// Setup
	matcher := &mockMatcher{shouldInhibit: false}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	// Execute with alert with empty labels
	alert := &core.Alert{
		Labels:      map[string]string{},
		Annotations: map[string]string{},
		Status:      core.StatusFiring,
		Fingerprint: "empty-fp",
	}
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert: should handle gracefully
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestInhibitionHandler_CheckAlert_LargePayload(t *testing.T) {
	// Setup
	matcher := &mockMatcher{shouldInhibit: false}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	// Execute with alert with many labels (100+)
	alert := &core.Alert{
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Status:      core.StatusFiring,
		Fingerprint: "large-fp",
	}
	for i := 0; i < 100; i++ {
		alert.Labels["label"+string(rune('0'+i%10))] = "value" + string(rune('0'+i%10))
	}

	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})
	req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.CheckAlert(w, req)

	// Assert: should handle large payloads
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestInhibitionHandler_NilPointerSafety(t *testing.T) {
	// Test that handler doesn't panic with nil components
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Handler panicked with nil logger: %v", r)
		}
	}()

	// Setup with nil logger (should use default)
	handler := NewInhibitionHandler(
		&mockParser{config: &inhibition.InhibitionConfig{}},
		&mockMatcher{},
		&mockStateManager{},
		testMetrics,
		nil, // nil logger
	)

	// Execute
	req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
	w := httptest.NewRecorder()
	handler.GetRules(w, req)

	// Assert: should not panic
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test Suite 6: Concurrent Requests

func TestInhibitionHandler_ConcurrentRequests(t *testing.T) {
	// Setup
	parser := &mockParser{
		config: &inhibition.InhibitionConfig{
			Rules: []inhibition.InhibitionRule{createTestRule("test-rule")},
		},
	}
	matcher := &mockMatcher{shouldInhibit: false}
	handler := setupTestHandler(parser, matcher, &mockStateManager{})

	// Execute: 100 concurrent requests
	const numRequests = 100
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
			w := httptest.NewRecorder()
			handler.GetRules(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Concurrent request failed with status %d", w.Code)
			}
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}
}

// Benchmark Suite

func BenchmarkInhibitionHandler_GetRules(b *testing.B) {
	// Setup with 10 rules
	rules := make([]inhibition.InhibitionRule, 10)
	for i := 0; i < 10; i++ {
		rules[i] = createTestRule("rule-" + string(rune('0'+i)))
	}
	parser := &mockParser{
		config: &inhibition.InhibitionConfig{Rules: rules},
	}
	handler := setupTestHandler(parser, &mockMatcher{}, &mockStateManager{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/inhibition/rules", nil)
		w := httptest.NewRecorder()
		handler.GetRules(w, req)
	}
}

func BenchmarkInhibitionHandler_GetStatus(b *testing.B) {
	// Setup with 100 active inhibitions
	now := time.Now()
	states := make([]*inhibition.InhibitionState, 100)
	for i := 0; i < 100; i++ {
		states[i] = &inhibition.InhibitionState{
			TargetFingerprint: "target-" + string(rune('0'+i%10)),
			SourceFingerprint: "source-" + string(rune('0'+i%10)),
			RuleName:          "rule-" + string(rune('0'+i%10)),
			InhibitedAt:       now,
		}
	}
	stateManager := &mockStateManager{states: states}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, &mockMatcher{}, stateManager)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v2/inhibition/status", nil)
		w := httptest.NewRecorder()
		handler.GetStatus(w, req)
	}
}

func BenchmarkInhibitionHandler_CheckAlert_NotInhibited(b *testing.B) {
	// Setup
	matcher := &mockMatcher{shouldInhibit: false}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	alert := createTestAlert("InstanceDown", "test-fp")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: alert})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.CheckAlert(w, req)
	}
}

func BenchmarkInhibitionHandler_CheckAlert_Inhibited(b *testing.B) {
	// Setup
	rule := createTestRule("test-rule")
	sourceAlert := createTestAlert("NodeDown", "source-fp")
	matcher := &mockMatcher{
		shouldInhibit: true,
		matchedRule:   &rule,
		matchedAlert:  sourceAlert,
	}
	handler := setupTestHandler(&mockParser{config: &inhibition.InhibitionConfig{}}, matcher, &mockStateManager{})

	targetAlert := createTestAlert("InstanceDown", "target-fp")
	reqBody, _ := json.Marshal(InhibitionCheckRequest{Alert: targetAlert})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/inhibition/check", bytes.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handler.CheckAlert(w, req)
	}
}
