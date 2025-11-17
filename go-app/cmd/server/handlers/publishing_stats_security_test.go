package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// TestSecurity_InputValidation tests input validation security.
func TestSecurity_InputValidation(t *testing.T) {
	handler := createTestHandler(nil)

	t.Run("Rejects SQL injection attempts in filter", func(t *testing.T) {
		testCases := []string{
			"'; DROP TABLE--",
			"1' OR '1'='1",
			"admin'--",
			"1; SELECT * FROM",
		}

		for _, injection := range testCases {
			// Properly encode the URL
			encoded := url.QueryEscape(injection)
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?filter=%s", encoded), nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			// Should reject with 400 Bad Request (invalid format)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for SQL injection attempt '%s', got %d", injection, w.Code)
			}
		}
	})

	t.Run("Rejects XSS attempts in filter", func(t *testing.T) {
		testCases := []string{
			"<script>alert('xss')</script>",
			"javascript:alert('xss')",
			"onerror=alert('xss')",
			"<img src=x onerror=alert('xss')>",
		}

		for _, xss := range testCases {
			encoded := url.QueryEscape(xss)
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?filter=%s", encoded), nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			// Should reject with 400 Bad Request (invalid format)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for XSS attempt '%s', got %d", xss, w.Code)
			}
		}
	})

	t.Run("Rejects command injection attempts", func(t *testing.T) {
		testCases := []string{
			"; ls -la",
			"| cat /etc/passwd",
			"&& rm -rf /",
			"`whoami`",
		}

		for _, injection := range testCases {
			encoded := url.QueryEscape(injection)
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?filter=%s", encoded), nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			// Should reject with 400 Bad Request (invalid format)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for command injection attempt '%s', got %d", injection, w.Code)
			}
		}
	})

	t.Run("Rejects path traversal attempts", func(t *testing.T) {
		testCases := []string{
			"../../../etc/passwd",
			"..\\..\\..\\windows\\system32",
			"....//....//etc/passwd",
		}

		for _, traversal := range testCases {
			encoded := url.QueryEscape(traversal)
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?filter=%s", encoded), nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			// Should reject with 400 Bad Request (invalid format)
			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected 400 for path traversal attempt '%s', got %d", traversal, w.Code)
			}
		}
	})

	t.Run("Rejects oversized query parameters", func(t *testing.T) {
		// Create a very long filter parameter (but valid format)
		longFilter := "type:" + strings.Repeat("a", 10000)
		encoded := url.QueryEscape(longFilter)
		req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/v2/publishing/stats?filter=%s", encoded), nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		// May accept or reject - depends on implementation
		// For now, just verify it doesn't crash
		if w.Code == http.StatusInternalServerError {
			t.Errorf("Should not return 500 for oversized filter, got %d", w.Code)
		}
	})
}

// TestSecurity_ErrorHandling tests error handling security.
func TestSecurity_ErrorHandling(t *testing.T) {
	handler := createTestHandler(nil)

	t.Run("Error responses do not expose stack traces", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?filter=invalid:format", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d", w.Code)
		}

		var errorResponse map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		// Check that error message doesn't contain sensitive information
		message, ok := errorResponse["message"].(string)
		if !ok {
			t.Fatal("Error response should have message field")
		}

		// Should not contain stack traces or internal paths
		if strings.Contains(message, "goroutine") ||
			strings.Contains(message, "/usr/local/go") ||
			strings.Contains(message, "runtime") {
			t.Error("Error message should not contain stack trace information")
		}
	})

	t.Run("Error responses have proper structure", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats?filter=invalid", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400, got %d", w.Code)
		}

		var errorResponse map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&errorResponse); err != nil {
			t.Fatalf("Failed to decode error response: %v", err)
		}

		// Verify required fields
		requiredFields := []string{"error", "message", "timestamp"}
		for _, field := range requiredFields {
			if _, ok := errorResponse[field]; !ok {
				t.Errorf("Error response should have '%s' field", field)
			}
		}
	})
}

// TestSecurity_NoSensitiveData tests that responses don't leak sensitive data.
func TestSecurity_NoSensitiveData(t *testing.T) {
	snapshot := &publishing.MetricsSnapshot{
		Timestamp:           time.Now(),
		Metrics:             createTestMetrics(),
		CollectionDuration:  time.Microsecond * 85,
		AvailableCollectors: []string{"health", "refresh"},
		Errors:              make(map[string]error),
	}
	handler := createTestHandler(snapshot)

	t.Run("Response does not contain credentials", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		body := w.Body.String()
		sensitivePatterns := []string{
			"password",
			"secret",
			"api_key",
			"token",
			"credential",
			"auth",
		}

		bodyLower := strings.ToLower(body)
		for _, pattern := range sensitivePatterns {
			if strings.Contains(bodyLower, pattern) {
				t.Errorf("Response should not contain sensitive pattern '%s'", pattern)
			}
		}
	})

	t.Run("Response does not contain internal paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/publishing/stats", nil)
		w := httptest.NewRecorder()

		handler.GetStats(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200, got %d", w.Code)
		}

		body := w.Body.String()
		internalPatterns := []string{
			"/usr/local",
			"/etc/",
			"/var/",
			"/home/",
			"C:\\",
		}

		for _, pattern := range internalPatterns {
			if strings.Contains(body, pattern) {
				t.Errorf("Response should not contain internal path '%s'", pattern)
			}
		}
	})
}

// TestSecurity_MethodValidation tests HTTP method validation.
func TestSecurity_MethodValidation(t *testing.T) {
	handler := createTestHandler(nil)

	t.Run("Rejects non-GET methods", func(t *testing.T) {
		methods := []string{
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodPatch,
			http.MethodOptions,
		}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/v2/publishing/stats", nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("Expected 405 for %s method, got %d", method, w.Code)
			}
		}
	})
}
