package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// mockRefreshManager implements publishing.RefreshManager for testing
type mockRefreshManager struct {
	refreshNowFunc  func() error
	getStatusFunc   func() publishing.RefreshStatus
	startFunc       func() error
	stopFunc        func(timeout time.Duration) error
	refreshNowCalls int
}

func (m *mockRefreshManager) RefreshNow() error {
	m.refreshNowCalls++
	if m.refreshNowFunc != nil {
		return m.refreshNowFunc()
	}
	return nil
}

func (m *mockRefreshManager) GetStatus() publishing.RefreshStatus {
	if m.getStatusFunc != nil {
		return m.getStatusFunc()
	}
	return publishing.RefreshStatus{
		State:              "success",
		LastRefresh:        time.Now().Add(-5 * time.Minute),
		NextRefresh:        time.Now().Add(5 * time.Minute),
		RefreshDuration:    2 * time.Second,
		TargetsDiscovered:  10,
		TargetsValid:       10,
		TargetsInvalid:     0,
		ConsecutiveFailures: 0,
		Error:              "",
	}
}

func (m *mockRefreshManager) Start() error {
	if m.startFunc != nil {
		return m.startFunc()
	}
	return nil
}

func (m *mockRefreshManager) Stop(timeout time.Duration) error {
	if m.stopFunc != nil {
		return m.stopFunc(timeout)
	}
	return nil
}

// TestHandleRefreshTargets_Success tests successful refresh trigger
func TestHandleRefreshTargets_Success(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusAccepted, rec.Code, "Expected 202 Accepted")
	assert.Equal(t, 1, mockMgr.refreshNowCalls, "RefreshNow should be called once")

	// Verify response body
	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err, "Response should be valid JSON")

	assert.Equal(t, "Refresh triggered", response["message"])
	assert.NotEmpty(t, response["request_id"], "request_id should be present")
	assert.NotEmpty(t, response["refresh_started_at"], "refresh_started_at should be present")

	// Verify security headers
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Equal(t, "default-src 'none'", rec.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", rec.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", rec.Header().Get("X-Frame-Options"))
	assert.Contains(t, rec.Header().Get("Strict-Transport-Security"), "max-age=31536000")
	assert.NotEmpty(t, rec.Header().Get("X-Request-ID"), "X-Request-ID should be present")
}

// TestHandleRefreshTargets_ResponseFormat tests response JSON structure
func TestHandleRefreshTargets_ResponseFormat(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	// Verify all required fields
	require.Contains(t, response, "message")
	require.Contains(t, response, "request_id")
	require.Contains(t, response, "refresh_started_at")

	// Verify types
	assert.IsType(t, "", response["message"])
	assert.IsType(t, "", response["request_id"])
	assert.IsType(t, "", response["refresh_started_at"])

	// Verify request_id is valid UUID format
	requestID := response["request_id"].(string)
	assert.Regexp(t, `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`, requestID)

	// Verify timestamp is valid RFC3339
	timestamp := response["refresh_started_at"].(string)
	_, err = time.Parse(time.RFC3339, timestamp)
	assert.NoError(t, err, "Timestamp should be valid RFC3339")
}

// TestHandleRefreshTargets_RateLimitExceeded tests 429 response
func TestHandleRefreshTargets_RateLimitExceeded(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{
		refreshNowFunc: func() error {
			return publishing.ErrRateLimitExceeded
		},
	}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusTooManyRequests, rec.Code, "Expected 429 Too Many Requests")

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "rate_limit_exceeded", response["error"])
	assert.Equal(t, "Max 1 refresh per minute", response["message"])
	assert.Equal(t, float64(60), response["retry_after_seconds"])
	assert.NotEmpty(t, response["request_id"])

	// Verify Retry-After header
	assert.Equal(t, "60", rec.Header().Get("Retry-After"))
}

// TestHandleRefreshTargets_RefreshInProgress tests 503 when already running
func TestHandleRefreshTargets_RefreshInProgress(t *testing.T) {
	// Arrange
	startedAt := time.Now().Add(-10 * time.Second)
	mockMgr := &mockRefreshManager{
		refreshNowFunc: func() error {
			return publishing.ErrRefreshInProgress
		},
		getStatusFunc: func() publishing.RefreshStatus {
			return publishing.RefreshStatus{
				State:       "in_progress",
				LastRefresh: startedAt,
			}
		},
	}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "Expected 503 Service Unavailable")

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "refresh_in_progress", response["error"])
	assert.Equal(t, "Target refresh already running", response["message"])
	assert.NotEmpty(t, response["started_at"])
	assert.NotEmpty(t, response["request_id"])

	// Verify started_at timestamp
	startedAtStr := response["started_at"].(string)
	parsedTime, err := time.Parse(time.RFC3339, startedAtStr)
	assert.NoError(t, err)
	assert.WithinDuration(t, startedAt, parsedTime, time.Second)
}

// TestHandleRefreshTargets_NotStarted tests 503 when manager not started
func TestHandleRefreshTargets_NotStarted(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{
		refreshNowFunc: func() error {
			return publishing.ErrNotStarted
		},
	}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code, "Expected 503 Service Unavailable")

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "manager_not_started", response["error"])
	assert.Equal(t, "Refresh manager not started", response["message"])
	assert.NotEmpty(t, response["request_id"])
}

// TestHandleRefreshTargets_UnknownError tests 500 on unexpected errors
func TestHandleRefreshTargets_UnknownError(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{
		refreshNowFunc: func() error {
			return errors.New("unexpected database error")
		},
	}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusInternalServerError, rec.Code, "Expected 500 Internal Server Error")

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "internal_error", response["error"])
	assert.Equal(t, "Internal server error", response["message"])
	assert.NotEmpty(t, response["request_id"])
}

// TestHandleRefreshTargets_NonEmptyBody tests 400 when body is not empty
func TestHandleRefreshTargets_NonEmptyBody(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	body := strings.NewReader(`{"invalid": "data"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Expected 400 Bad Request")
	assert.Equal(t, 0, mockMgr.refreshNowCalls, "RefreshNow should NOT be called")

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "invalid_request", response["error"])
	assert.Equal(t, "Request body must be empty", response["message"])
	assert.NotEmpty(t, response["request_id"])
}

// TestHandleRefreshTargets_OversizedRequest tests size limit enforcement
func TestHandleRefreshTargets_OversizedRequest(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	// Create 2KB payload (exceeds 1KB limit)
	largePayload := strings.Repeat("x", 2048)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", strings.NewReader(largePayload))
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	// Should fail on ContentLength check before hitting MaxBytesReader
	assert.Equal(t, http.StatusBadRequest, rec.Code, "Expected 400 Bad Request")
	assert.Equal(t, 0, mockMgr.refreshNowCalls, "RefreshNow should NOT be called")
}

// TestHandleRefreshTargets_SecurityHeaders tests all security headers present
func TestHandleRefreshTargets_SecurityHeaders(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert - verify all security headers
	headers := rec.Header()

	assert.Equal(t, "application/json; charset=utf-8", headers.Get("Content-Type"))
	assert.Equal(t, "default-src 'none'", headers.Get("Content-Security-Policy"))
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "max-age=31536000; includeSubDomains", headers.Get("Strict-Transport-Security"))
	assert.Equal(t, "no-store, no-cache, must-revalidate, private", headers.Get("Cache-Control"))
	assert.Equal(t, "no-cache", headers.Get("Pragma"))
	assert.NotEmpty(t, headers.Get("X-Request-ID"))
}

// TestHandleRefreshTargets_ConcurrentRequests tests thread safety
func TestHandleRefreshTargets_ConcurrentRequests(t *testing.T) {
	// Arrange
	callCount := 0
	mockMgr := &mockRefreshManager{
		refreshNowFunc: func() error {
			callCount++
			if callCount == 1 {
				return nil // First succeeds
			}
			return publishing.ErrRefreshInProgress // Others fail
		},
	}
	handler := HandleRefreshTargets(mockMgr)

	// Act - spawn 10 concurrent requests
	const numRequests = 10
	responses := make([]*httptest.ResponseRecorder, numRequests)
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(idx int) {
			req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
			rec := httptest.NewRecorder()
			responses[idx] = rec
			handler.ServeHTTP(rec, req)
			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// Assert - at least one should succeed (202), others should fail (503)
	successCount := 0
	failCount := 0

	for _, rec := range responses {
		if rec.Code == http.StatusAccepted {
			successCount++
		} else if rec.Code == http.StatusServiceUnavailable {
			failCount++
		}
	}

	assert.Greater(t, successCount, 0, "At least one request should succeed")
	assert.Equal(t, numRequests, successCount+failCount, "All requests should complete")
}

// TestHandleRefreshTargets_RequestIDUnique tests request IDs are unique
func TestHandleRefreshTargets_RequestIDUnique(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	const numRequests = 100
	requestIDs := make(map[string]bool)

	// Act - make many requests
	for i := 0; i < numRequests; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		var response map[string]interface{}
		err := json.NewDecoder(rec.Body).Decode(&response)
		require.NoError(t, err)

		requestID := response["request_id"].(string)
		require.NotEmpty(t, requestID)

		// Check for duplicates
		assert.False(t, requestIDs[requestID], "Request ID should be unique: %s", requestID)
		requestIDs[requestID] = true
	}

	// Assert - all IDs should be unique
	assert.Equal(t, numRequests, len(requestIDs), "All request IDs should be unique")
}

// TestHandleRefreshTargets_EmptyContentLength tests explicit empty body
func TestHandleRefreshTargets_EmptyContentLength(t *testing.T) {
	// Arrange
	mockMgr := &mockRefreshManager{}
	handler := HandleRefreshTargets(mockMgr)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/publishing/targets/refresh", nil)
	req.ContentLength = 0 // Explicitly set to 0
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusAccepted, rec.Code, "Expected 202 Accepted for empty body")
	assert.Equal(t, 1, mockMgr.refreshNowCalls, "RefreshNow should be called")
}

// TestHandleRefreshStatus tests status endpoint
func TestHandleRefreshStatus(t *testing.T) {
	// Arrange
	lastRefresh := time.Now().Add(-5 * time.Minute)
	nextRefresh := time.Now().Add(5 * time.Minute)

	mockMgr := &mockRefreshManager{
		getStatusFunc: func() publishing.RefreshStatus {
			return publishing.RefreshStatus{
				State:               "success",
				LastRefresh:         lastRefresh,
				NextRefresh:         nextRefresh,
				RefreshDuration:     1856 * time.Millisecond,
				TargetsDiscovered:   15,
				TargetsValid:        14,
				TargetsInvalid:      1,
				ConsecutiveFailures: 0,
				Error:               "",
			}
		},
	}
	handler := HandleRefreshStatus(mockMgr)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/targets/status", nil)
	rec := httptest.NewRecorder()

	// Act
	handler.ServeHTTP(rec, req)

	// Assert
	assert.Equal(t, http.StatusOK, rec.Code)

	var response map[string]interface{}
	err := json.NewDecoder(rec.Body).Decode(&response)
	require.NoError(t, err)

	assert.Equal(t, "success", response["status"])
	assert.Equal(t, float64(15), response["targets_discovered"])
	assert.Equal(t, float64(14), response["targets_valid"])
	assert.Equal(t, float64(1), response["targets_invalid"])
	assert.Equal(t, float64(0), response["consecutive_failures"])
	assert.Equal(t, float64(1856), response["refresh_duration_ms"])
	assert.NotEmpty(t, response["last_refresh"])
	assert.NotEmpty(t, response["next_refresh"])
}
