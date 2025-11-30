//go:build e2e
// +build e2e

package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// MockResponse configures mock target response behavior
type MockResponse struct {
	StatusCode int
	Body       interface{}
	Delay      time.Duration
	ErrorRate  float64 // 0.0-1.0 probability of returning error
}

// MockPublishingTarget simulates external publishing endpoints (Slack, PagerDuty, Rootly)
type MockPublishingTarget struct {
	Server    *httptest.Server
	Name      string
	Type      string // "slack", "pagerduty", "rootly", "webhook"
	Requests  []*RecordedRequest
	Responses []MockResponse
	mu        sync.Mutex
	callCount int
}

// RecordedRequest stores details of received request
type RecordedRequest struct {
	Method      string
	Path        string
	Headers     http.Header
	Body        map[string]interface{}
	ReceivedAt  time.Time
	ProcessedIn time.Duration
}

// NewMockPublishingTarget creates a new mock target
func NewMockPublishingTarget(name, targetType string) *MockPublishingTarget {
	mock := &MockPublishingTarget{
		Name:      name,
		Type:      targetType,
		Requests:  make([]*RecordedRequest, 0),
		Responses: make([]MockResponse, 0),
	}

	// Create HTTP server
	mock.Server = httptest.NewServer(http.HandlerFunc(mock.handler))

	return mock
}

// handler processes incoming requests
func (m *MockPublishingTarget) handler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	m.mu.Lock()
	defer m.mu.Unlock()

	// Record request
	body, _ := io.ReadAll(r.Body)
	var bodyJSON map[string]interface{}
	if len(body) > 0 {
		json.Unmarshal(body, &bodyJSON)
	}

	req := &RecordedRequest{
		Method:     r.Method,
		Path:       r.URL.Path,
		Headers:    r.Header.Clone(),
		Body:       bodyJSON,
		ReceivedAt: start,
	}

	// Determine response
	var resp MockResponse
	if m.callCount < len(m.Responses) {
		resp = m.Responses[m.callCount]
	} else {
		// Default success response
		resp = m.getDefaultResponse()
	}
	m.callCount++

	// Simulate delay
	if resp.Delay > 0 {
		time.Sleep(resp.Delay)
	}

	// Simulate error rate
	if resp.ErrorRate > 0 {
		// Use nanoseconds for pseudo-random
		if float64(time.Now().Nanosecond()%100)/100.0 < resp.ErrorRate {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "simulated error",
			})
			req.ProcessedIn = time.Since(start)
			m.Requests = append(m.Requests, req)
			return
		}
	}

	// Write response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if resp.Body != nil {
		json.NewEncoder(w).Encode(resp.Body)
	}

	req.ProcessedIn = time.Since(start)
	m.Requests = append(m.Requests, req)
}

// getDefaultResponse returns default success response based on type
func (m *MockPublishingTarget) getDefaultResponse() MockResponse {
	switch m.Type {
	case "slack":
		return MockResponse{
			StatusCode: http.StatusOK,
			Body:       map[string]interface{}{"ok": true},
		}
	case "pagerduty":
		return MockResponse{
			StatusCode: http.StatusAccepted,
			Body: map[string]interface{}{
				"status":    "success",
				"dedup_key": fmt.Sprintf("dedup_%d", time.Now().Unix()),
			},
		}
	case "rootly":
		return MockResponse{
			StatusCode: http.StatusCreated,
			Body: map[string]interface{}{
				"data": map[string]interface{}{
					"id":    fmt.Sprintf("inc_%d", time.Now().Unix()),
					"title": "Mock Incident",
				},
			},
		}
	default: // webhook
		return MockResponse{
			StatusCode: http.StatusOK,
			Body:       map[string]interface{}{"received": true},
		}
	}
}

// AddResponse adds a configured response for next request
func (m *MockPublishingTarget) AddResponse(resp MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = append(m.Responses, resp)
}

// SetResponses replaces all responses with new list
func (m *MockPublishingTarget) SetResponses(responses []MockResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Responses = responses
	m.callCount = 0
}

// GetRequests returns all recorded requests
func (m *MockPublishingTarget) GetRequests() []*RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Requests
}

// GetRequestCount returns number of requests received
func (m *MockPublishingTarget) GetRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Requests)
}

// GetLastRequest returns the most recent request
func (m *MockPublishingTarget) GetLastRequest() *RecordedRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.Requests) == 0 {
		return nil
	}
	return m.Requests[len(m.Requests)-1]
}

// ClearRequests clears all recorded requests
func (m *MockPublishingTarget) ClearRequests() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Requests = make([]*RecordedRequest, 0)
	m.callCount = 0
}

// Close stops the mock server
func (m *MockPublishingTarget) Close() {
	m.Server.Close()
}

// URL returns the mock server URL
func (m *MockPublishingTarget) URL() string {
	return m.Server.URL
}

// --- Mock Slack Target ---

// NewMockSlackTarget creates a Slack webhook mock
func NewMockSlackTarget(name string) *MockPublishingTarget {
	return NewMockPublishingTarget(name, "slack")
}

// VerifySlackMessage asserts Slack message was received with expected content
func (m *MockPublishingTarget) VerifySlackMessage(expectedText string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Requests) == 0 {
		return fmt.Errorf("no requests received")
	}

	lastReq := m.Requests[len(m.Requests)-1]
	if lastReq.Body == nil {
		return fmt.Errorf("request body is nil")
	}

	// Check for text in blocks or text field
	if text, ok := lastReq.Body["text"].(string); ok {
		if text == expectedText {
			return nil
		}
	}

	// Check in blocks
	if blocks, ok := lastReq.Body["blocks"].([]interface{}); ok {
		for _, block := range blocks {
			if blockMap, ok := block.(map[string]interface{}); ok {
				if text, ok := blockMap["text"].(map[string]interface{}); ok {
					if textVal, ok := text["text"].(string); ok && textVal == expectedText {
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("expected text '%s' not found in Slack message", expectedText)
}

// --- Mock PagerDuty Target ---

// NewMockPagerDutyTarget creates a PagerDuty Events API mock
func NewMockPagerDutyTarget(name string) *MockPublishingTarget {
	return NewMockPublishingTarget(name, "pagerduty")
}

// VerifyPagerDutyEvent asserts PagerDuty event was received with expected action
func (m *MockPublishingTarget) VerifyPagerDutyEvent(expectedAction string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Requests) == 0 {
		return fmt.Errorf("no requests received")
	}

	lastReq := m.Requests[len(m.Requests)-1]
	if lastReq.Body == nil {
		return fmt.Errorf("request body is nil")
	}

	action, ok := lastReq.Body["event_action"].(string)
	if !ok {
		return fmt.Errorf("event_action not found in request")
	}

	if action != expectedAction {
		return fmt.Errorf("expected event_action '%s', got '%s'", expectedAction, action)
	}

	return nil
}

// --- Mock Rootly Target ---

// NewMockRootlyTarget creates a Rootly Incidents API mock
func NewMockRootlyTarget(name string) *MockPublishingTarget {
	return NewMockPublishingTarget(name, "rootly")
}

// VerifyRootlyIncident asserts Rootly incident was created with expected title
func (m *MockPublishingTarget) VerifyRootlyIncident(expectedTitle string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Requests) == 0 {
		return fmt.Errorf("no requests received")
	}

	lastReq := m.Requests[len(m.Requests)-1]
	if lastReq.Body == nil {
		return fmt.Errorf("request body is nil")
	}

	// Check title in data object
	if data, ok := lastReq.Body["data"].(map[string]interface{}); ok {
		if attributes, ok := data["attributes"].(map[string]interface{}); ok {
			if title, ok := attributes["title"].(string); ok && title == expectedTitle {
				return nil
			}
		}
	}

	// Check title at root level
	if title, ok := lastReq.Body["title"].(string); ok && title == expectedTitle {
		return nil
	}

	return fmt.Errorf("expected title '%s' not found in Rootly incident", expectedTitle)
}

// --- Mock Generic Webhook ---

// NewMockGenericWebhook creates a generic webhook mock
func NewMockGenericWebhook(name string) *MockPublishingTarget {
	return NewMockPublishingTarget(name, "webhook")
}

// --- Helper Functions ---

// VerifyRequestField asserts a field exists in request body with expected value
func (r *RecordedRequest) VerifyField(field string, expectedValue interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is nil")
	}

	value, ok := r.Body[field]
	if !ok {
		return fmt.Errorf("field '%s' not found in request body", field)
	}

	if value != expectedValue {
		return fmt.Errorf("field '%s': expected '%v', got '%v'", field, expectedValue, value)
	}

	return nil
}

// HasHeader checks if request has a specific header
func (r *RecordedRequest) HasHeader(key, value string) bool {
	return r.Headers.Get(key) == value
}

// GetBodyField retrieves a field from request body
func (r *RecordedRequest) GetBodyField(field string) (interface{}, bool) {
	if r.Body == nil {
		return nil, false
	}
	value, ok := r.Body[field]
	return value, ok
}
