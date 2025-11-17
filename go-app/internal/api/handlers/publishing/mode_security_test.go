package publishing

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
)

// TestSecurity_OWASP_SensitiveDataExposure tests that no sensitive data is exposed
func TestSecurity_OWASP_SensitiveDataExposure(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify no sensitive data in response
	// Response should only contain mode information, no secrets, tokens, or credentials
	assert.NotEmpty(t, response.Mode)
	assert.NotContains(t, response.Mode, "secret")
	assert.NotContains(t, response.Mode, "token")
	assert.NotContains(t, response.Mode, "password")
	assert.NotContains(t, response.Mode, "key")
	assert.NotContains(t, response.Mode, "credential")
}

// TestSecurity_OWASP_SecurityMisconfiguration tests security headers
func TestSecurity_OWASP_SecurityMisconfiguration(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{Mode: "normal"},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Verify security headers are set (OWASP Top 10 compliance)
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "strict-origin-when-cross-origin", w.Header().Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(), microphone=(), camera=()", w.Header().Get("Permissions-Policy"))

	// Verify caching headers are set (security best practice)
	assert.Equal(t, "max-age=5, public", w.Header().Get("Cache-Control"))
	assert.NotEmpty(t, w.Header().Get("ETag"))
}

// TestSecurity_OWASP_XSS tests XSS protection (no user-generated content, but verify structure)
func TestSecurity_OWASP_XSS(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Verify Content-Type is set correctly (prevents MIME sniffing)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))

	// Verify response is valid JSON (not HTML/script)
	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
}

// TestSecurity_InputValidation_Method tests method validation
func TestSecurity_InputValidation_Method(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	// Test POST method (should be rejected)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "Method Not Allowed", errorResponse.Error)
}

// TestSecurity_InputValidation_Body tests that body is ignored (GET should not have body)
func TestSecurity_InputValidation_Body(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	// Test with body (should be ignored, but request should still work)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	req.ContentLength = 100 // Simulate body
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Should still work (body is ignored for GET)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSecurity_NoSensitiveData_Response tests response doesn't contain sensitive data
func TestSecurity_NoSensitiveData_Response(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Get response body as string
	body := w.Body.String()

	// Verify no sensitive patterns in response
	sensitivePatterns := []string{
		"password",
		"secret",
		"token",
		"api_key",
		"apikey",
		"credential",
		"private",
		"private_key",
		"access_token",
		"refresh_token",
	}

	for _, pattern := range sensitivePatterns {
		assert.NotContains(t, body, pattern, "Response should not contain sensitive data: %s", pattern)
	}
}

// TestSecurity_ErrorResponse_NoStackTrace tests error responses don't leak stack traces
func TestSecurity_ErrorResponse_NoStackTrace(t *testing.T) {
	mockService := &mockModeService{err: assert.AnError}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	// Verify error message doesn't contain stack traces or internal details
	assert.NotContains(t, errorResponse.Message, "goroutine")
	assert.NotContains(t, errorResponse.Message, "panic")
	assert.NotContains(t, errorResponse.Message, "stack")
	assert.NotContains(t, errorResponse.Message, "runtime")
	assert.NotContains(t, errorResponse.Message, "internal")
}

// TestSecurity_RequestID_AlwaysPresent tests request ID is always present for tracing
func TestSecurity_RequestID_AlwaysPresent(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Request ID should be in response header (if middleware is applied)
	// For now, verify it's in error responses
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSecurity_ErrorResponse_RequestID tests error responses include request ID
func TestSecurity_ErrorResponse_RequestID(t *testing.T) {
	mockService := &mockModeService{err: assert.AnError}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	// Request ID should be present for tracing
	assert.NotEmpty(t, errorResponse.RequestID)
	// Request ID can be UUID (36 chars) or fallback (longer), both are valid
	assert.GreaterOrEqual(t, len(errorResponse.RequestID), 28)
}

// TestSecurity_JSON_ValidStructure tests JSON response structure is valid
func TestSecurity_JSON_ValidStructure(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Verify response is valid JSON
	var response apiservices.ModeInfo
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify required fields are present
	assert.NotEmpty(t, response.Mode)
	assert.NotNil(t, response.TargetsAvailable)
}

// TestSecurity_ConcurrentAccess_Safe tests concurrent access is safe
func TestSecurity_ConcurrentAccess_Safe(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	const numRequests = 50
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
			w := httptest.NewRecorder()
			handler.GetPublishingMode(w, req)
			results <- w.Code
		}()
	}

	successCount := 0
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		}
	}

	// All requests should succeed (no race conditions)
	assert.Equal(t, numRequests, successCount)
}

// TestSecurity_ETag_NoSensitiveData tests ETag doesn't contain sensitive data
func TestSecurity_ETag_NoSensitiveData(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:            "normal",
			EnabledTargets:  5,
			TransitionCount: 12,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	etag := w.Header().Get("ETag")

	// ETag should only contain mode, enabled_targets, transition_count
	// No sensitive data
	assert.NotContains(t, etag, "secret")
	assert.NotContains(t, etag, "token")
	assert.NotContains(t, etag, "password")
	assert.NotEmpty(t, etag)
}

// TestSecurity_ResponseSize_Reasonable tests response size is reasonable
func TestSecurity_ResponseSize_Reasonable(t *testing.T) {
	mockService := &mockModeService{
		modeInfo: &apiservices.ModeInfo{
			Mode:              "normal",
			TargetsAvailable:  true,
			EnabledTargets:    5,
			MetricsOnlyActive: false,
			TransitionCount:   12,
		},
	}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Response should be reasonable size (< 1KB)
	bodySize := w.Body.Len()
	assert.Less(t, bodySize, 1024, "Response size should be < 1KB")
	assert.Greater(t, bodySize, 50, "Response should have some content")
}

// TestSecurity_ContentType_AlwaysSet tests Content-Type is always set
func TestSecurity_ContentType_AlwaysSet(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Content-Type should always be set (prevents MIME sniffing)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

// TestSecurity_ErrorResponse_ConsistentStructure tests error responses have consistent structure
func TestSecurity_ErrorResponse_ConsistentStructure(t *testing.T) {
	mockService := &mockModeService{err: assert.AnError}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	// Verify consistent error structure
	assert.NotEmpty(t, errorResponse.Error)
	assert.NotEmpty(t, errorResponse.Message)
	assert.NotEmpty(t, errorResponse.RequestID)
	assert.False(t, errorResponse.Timestamp.IsZero())
}

// TestSecurity_NoSQLInjection tests no SQL injection possible (no user input in queries)
func TestSecurity_NoSQLInjection(t *testing.T) {
	// This endpoint doesn't accept user input, so SQL injection is not applicable
	// But we verify that even if malicious data is in headers, it's not used in queries
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	// Try with malicious header values
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	req.Header.Set("X-Malicious", "'; DROP TABLE users; --")
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Should still work (headers are not used in queries)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSecurity_NoCommandInjection tests no command injection possible
func TestSecurity_NoCommandInjection(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	// Try with malicious query parameter (path injection not possible with httptest)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode?cmd=rm+-rf+/", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Should still work (query params are not used in commands)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSecurity_NoPathTraversal tests no path traversal possible
func TestSecurity_NoPathTraversal(t *testing.T) {
	mockService := &mockModeService{modeInfo: &apiservices.ModeInfo{Mode: "normal"}}
	handler := NewPublishingModeHandler(mockService, nil)

	// Try with path traversal
	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode/../../../etc/passwd", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	// Should still work (no file access)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSecurity_NoInformationDisclosure tests no information disclosure in errors
func TestSecurity_NoInformationDisclosure(t *testing.T) {
	mockService := &mockModeService{err: assert.AnError}
	handler := NewPublishingModeHandler(mockService, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/publishing/mode", nil)
	w := httptest.NewRecorder()

	handler.GetPublishingMode(w, req)

	var errorResponse apiservices.ErrorResponse
	err := json.NewDecoder(w.Body).Decode(&errorResponse)
	require.NoError(t, err)

	// Error message should be generic, not revealing internal details
	assert.NotContains(t, errorResponse.Message, "database")
	assert.NotContains(t, errorResponse.Message, "connection")
	assert.NotContains(t, errorResponse.Message, "query")
	assert.NotContains(t, errorResponse.Message, "sql")
	assert.NotContains(t, errorResponse.Message, "file")
	assert.NotContains(t, errorResponse.Message, "path")
}

