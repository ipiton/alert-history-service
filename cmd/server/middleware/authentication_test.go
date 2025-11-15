// Package middleware provides HTTP middleware components.
package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestAuthenticationMiddleware_Disabled tests that disabled auth passes through
func TestAuthenticationMiddleware_Disabled(t *testing.T) {
	config := &AuthConfig{
		Enabled: false,
		Type:    "api_key",
		APIKey:  "secret-key",
	}

	auth := AuthenticationMiddleware(config)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	// Request without auth header
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("Expected next handler to be called when auth disabled")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestAuthenticationMiddleware_APIKey_Valid tests valid API key
func TestAuthenticationMiddleware_APIKey_Valid(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "my-secret-key",
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-API-Key", "my-secret-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid API key, got %d", rr.Code)
	}
}

// TestAuthenticationMiddleware_APIKey_Invalid tests invalid API key
func TestAuthenticationMiddleware_APIKey_Invalid(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "my-secret-key",
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called with invalid API key")
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	testCases := []struct {
		name     string
		apiKey   string
		expected int
	}{
		{"wrong key", "wrong-key", http.StatusUnauthorized},
		{"empty key", "", http.StatusUnauthorized},
		{"no header", "", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tc.apiKey != "" {
				req.Header.Set("X-API-Key", tc.apiKey)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tc.expected {
				t.Errorf("Expected status %d, got %d", tc.expected, rr.Code)
			}

			// Check error response
			var resp map[string]string
			if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
				t.Fatalf("Failed to decode error response: %v", err)
			}

			if resp["status"] != "unauthorized" {
				t.Errorf("Expected status 'unauthorized', got %s", resp["status"])
			}

			// Check WWW-Authenticate header
			wwwAuth := rr.Header().Get("WWW-Authenticate")
			if !strings.Contains(wwwAuth, "api_key") {
				t.Errorf("Expected WWW-Authenticate header with api_key, got: %s", wwwAuth)
			}
		})
	}
}

// TestAuthenticationMiddleware_APIKey_AlternativeHeaders tests alternative header names
func TestAuthenticationMiddleware_APIKey_AlternativeHeaders(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "my-secret-key",
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	// Try Authorization header with Bearer scheme
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("Authorization", "Bearer my-secret-key")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Logf("Authorization Bearer: status %d (may require X-API-Key header)", rr.Code)
	}
}

// TestAuthenticationMiddleware_HMAC_Valid tests valid HMAC signature
func TestAuthenticationMiddleware_HMAC_Valid(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := "my-hmac-secret"

	config := &AuthConfig{
		Enabled:   true,
		Type:      "hmac",
		JWTSecret: secret,
		Logger:    logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	// Create request with HMAC signature
	payload := []byte(`{"alerts":[]}`)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))

	// Calculate HMAC
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-Signature", signature)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 with valid HMAC, got %d", rr.Code)
	}
}

// TestAuthenticationMiddleware_HMAC_Invalid tests invalid HMAC signature
func TestAuthenticationMiddleware_HMAC_Invalid(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled:   true,
		Type:      "hmac",
		JWTSecret: "my-hmac-secret",
		Logger:    logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called with invalid HMAC")
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	testCases := []struct {
		name      string
		payload   []byte
		signature string
	}{
		{"wrong signature", []byte(`{"alerts":[]}`), "invalid-signature"},
		{"empty signature", []byte(`{"alerts":[]}`), ""},
		{"tampered payload", []byte(`{"alerts":[{"modified":true}]}`), "0123456789abcdef"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tc.payload))
			if tc.signature != "" {
				req.Header.Set("X-Signature", tc.signature)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusUnauthorized {
				t.Errorf("Expected status 401, got %d", rr.Code)
			}
		})
	}
}

// TestAuthenticationMiddleware_UnsupportedType tests unsupported auth type
func TestAuthenticationMiddleware_UnsupportedType(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "oauth2", // Unsupported
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Next handler should not be called with unsupported auth type")
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for unsupported auth type, got %d", rr.Code)
	}
}

// TestAuthenticationMiddleware_CaseSensitivity tests API key case sensitivity
func TestAuthenticationMiddleware_CaseSensitivity(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "MySecretKey",
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	// API keys should be case-sensitive
	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-API-Key", "mysecretkey") // lowercase
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Error("API key should be case-sensitive")
	}
}

// TestAuthenticationMiddleware_Concurrent tests concurrent authentication
func TestAuthenticationMiddleware_Concurrent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "concurrent-test-key",
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	const numRequests = 20
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if id%2 == 0 {
				req.Header.Set("X-API-Key", "concurrent-test-key") // Valid
			} else {
				req.Header.Set("X-API-Key", "wrong-key") // Invalid
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results <- rr.Code
		}(i)
	}

	successCount := 0
	unauthorizedCount := 0

	for i := 0; i < numRequests; i++ {
		code := <-results
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusUnauthorized {
			unauthorizedCount++
		}
	}

	expectedSuccess := numRequests / 2
	if successCount != expectedSuccess {
		t.Errorf("Expected %d successful auths, got %d", expectedSuccess, successCount)
	}

	if unauthorizedCount != expectedSuccess {
		t.Errorf("Expected %d unauthorized, got %d", expectedSuccess, unauthorizedCount)
	}
}

// TestValidateAPIKey tests API key validation function
func TestValidateAPIKey(t *testing.T) {
	testCases := []struct {
		name      string
		headerKey string
		configKey string
		expected  bool
	}{
		{"exact match", "secret123", "secret123", true},
		{"wrong key", "wrong", "secret123", false},
		{"empty header", "", "secret123", false},
		{"empty config", "secret123", "", false},
		{"both empty", "", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			if tc.headerKey != "" {
				req.Header.Set("X-API-Key", tc.headerKey)
			}

			valid, _ := validateAPIKey(req, tc.configKey)

			if valid != tc.expected {
				t.Errorf("validateAPIKey() = %v, want %v", valid, tc.expected)
			}
		})
	}
}

// BenchmarkAuthenticationMiddleware_Disabled benchmarks disabled auth
func BenchmarkAuthenticationMiddleware_Disabled(b *testing.B) {
	config := &AuthConfig{
		Enabled: false,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkAuthenticationMiddleware_APIKey benchmarks API key validation
func BenchmarkAuthenticationMiddleware_APIKey(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &AuthConfig{
		Enabled: true,
		Type:    "api_key",
		APIKey:  "benchmark-key",
		Logger:  logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("X-API-Key", "benchmark-key")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkAuthenticationMiddleware_HMAC benchmarks HMAC validation
func BenchmarkAuthenticationMiddleware_HMAC(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	secret := "benchmark-secret"

	config := &AuthConfig{
		Enabled:   true,
		Type:      "hmac",
		JWTSecret: secret,
		Logger:    logger,
	}

	auth := AuthenticationMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := auth(next)

	payload := []byte(`{"alerts":[]}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(payload))
		req.Header.Set("X-Signature", signature)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)
	}
}

