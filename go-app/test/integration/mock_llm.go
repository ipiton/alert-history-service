//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// MockLLMServer simulates LLM API for testing
type MockLLMServer struct {
	server    *httptest.Server
	mu        sync.Mutex
	responses map[string]*ClassificationResponse
	errorResp *ErrorResponse
	latency   time.Duration
	requests  []*ClassificationRequest
}

// ClassificationRequest represents LLM classification request
type ClassificationRequest struct {
	AlertName  string                 `json:"alert_name"`
	Labels     map[string]string      `json:"labels"`
	Annotations map[string]string     `json:"annotations"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ClassificationResponse represents LLM classification response
type ClassificationResponse struct {
	Severity     string   `json:"severity"`
	Category     string   `json:"category"`
	Confidence   float64  `json:"confidence"`
	Reasoning    string   `json:"reasoning"`
	ActionItems  []string `json:"action_items"`
	Model        string   `json:"model"`
	ProcessingMS int      `json:"processing_ms"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}

// NewMockLLMServer creates mock LLM server
func NewMockLLMServer() *MockLLMServer {
	mock := &MockLLMServer{
		responses: make(map[string]*ClassificationResponse),
		requests:  make([]*ClassificationRequest, 0),
	}

	// Create HTTP test server
	mock.server = httptest.NewServer(http.HandlerFunc(mock.handleClassify))

	return mock
}

// handleClassify handles classification requests
func (m *MockLLMServer) handleClassify(w http.ResponseWriter, r *http.Request) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Simulate latency if configured
	if m.latency > 0 {
		time.Sleep(m.latency)
	}

	// Return error if configured
	if m.errorResp != nil {
		w.WriteHeader(m.errorResp.StatusCode)
		json.NewEncoder(w).Encode(map[string]string{
			"error": m.errorResp.Message,
		})
		return
	}

	// Parse request
	var req ClassificationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid request body",
		})
		return
	}

	// Track request
	m.requests = append(m.requests, &req)

	// Get configured response or return default
	resp, exists := m.responses[req.AlertName]
	if !exists {
		// Default response
		resp = &ClassificationResponse{
			Severity:     "warning",
			Category:     "infrastructure",
			Confidence:   0.85,
			Reasoning:    "Default mock classification",
			ActionItems:  []string{"Check logs", "Review metrics"},
			Model:        "mock-llm-v1",
			ProcessingMS: 50,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// SetResponse configures mock response for specific alert
func (m *MockLLMServer) SetResponse(alertName string, response *ClassificationResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses[alertName] = response
}

// SetError simulates LLM API error
func (m *MockLLMServer) SetError(statusCode int, errorMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorResp = &ErrorResponse{
		StatusCode: statusCode,
		Message:    errorMsg,
	}
}

// ClearError removes configured error
func (m *MockLLMServer) ClearError() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorResp = nil
}

// SetLatency simulates slow LLM API
func (m *MockLLMServer) SetLatency(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.latency = duration
}

// GetRequestCount returns number of LLM requests made
func (m *MockLLMServer) GetRequestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}

// GetLastRequest returns most recent request (or nil)
func (m *MockLLMServer) GetLastRequest() *ClassificationRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.requests) == 0 {
		return nil
	}
	return m.requests[len(m.requests)-1]
}

// GetRequests returns all requests made to mock server
func (m *MockLLMServer) GetRequests() []*ClassificationRequest {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return copy to avoid race conditions
	reqs := make([]*ClassificationRequest, len(m.requests))
	copy(reqs, m.requests)
	return reqs
}

// Reset clears all configured responses, errors, and requests
func (m *MockLLMServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.responses = make(map[string]*ClassificationResponse)
	m.errorResp = nil
	m.latency = 0
	m.requests = make([]*ClassificationRequest, 0)
}

// URL returns mock server URL
func (m *MockLLMServer) URL() string {
	return m.server.URL
}

// Close stops mock server
func (m *MockLLMServer) Close() {
	m.server.Close()
}

// WaitForRequests waits until mock server has received expected number of requests
// Returns true if condition met within timeout, false otherwise
func (m *MockLLMServer) WaitForRequests(expected int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if m.GetRequestCount() >= expected {
			return true
		}
		<-ticker.C
	}
	return false
}

// SetDefaultResponses configures common test responses
func (m *MockLLMServer) SetDefaultResponses() {
	m.SetResponse("HighMemoryUsage", &ClassificationResponse{
		Severity:     "critical",
		Category:     "resource",
		Confidence:   0.95,
		Reasoning:    "Memory usage exceeds 95% threshold, immediate action required",
		ActionItems:  []string{"Scale up pods", "Check for memory leaks", "Review memory limits"},
		Model:        "mock-llm-v1",
		ProcessingMS: 45,
	})

	m.SetResponse("HighCPUUsage", &ClassificationResponse{
		Severity:     "warning",
		Category:     "resource",
		Confidence:   0.88,
		Reasoning:    "CPU usage is elevated but within acceptable range",
		ActionItems:  []string{"Monitor trends", "Review application efficiency"},
		Model:        "mock-llm-v1",
		ProcessingMS: 42,
	})

	m.SetResponse("DiskSpaceLow", &ClassificationResponse{
		Severity:     "critical",
		Category:     "storage",
		Confidence:   0.92,
		Reasoning:    "Disk space below 10%, risk of service disruption",
		ActionItems:  []string{"Clean up old logs", "Expand disk", "Enable log rotation"},
		Model:        "mock-llm-v1",
		ProcessingMS: 48,
	})

	m.SetResponse("ServiceDown", &ClassificationResponse{
		Severity:     "critical",
		Category:     "availability",
		Confidence:   0.98,
		Reasoning:    "Service is unreachable, impacting users",
		ActionItems:  []string{"Check service health", "Review logs", "Initiate incident"},
		Model:        "mock-llm-v1",
		ProcessingMS: 38,
	})

	m.SetResponse("HighErrorRate", &ClassificationResponse{
		Severity:     "warning",
		Category:     "reliability",
		Confidence:   0.85,
		Reasoning:    "Error rate increased but below critical threshold",
		ActionItems:  []string{"Review recent deployments", "Check error logs", "Monitor trends"},
		Model:        "mock-llm-v1",
		ProcessingMS: 50,
	})
}
