package middleware

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeaders(t *testing.T) {
	tests := []struct {
		name           string
		config         SecurityHeadersConfig
		useTLS         bool
		expectedHSTS   bool
		expectedCSP    string
		expectedServer string
	}{
		{
			name:         "Default config over HTTP",
			config:       DefaultSecurityHeadersConfig(),
			useTLS:       false,
			expectedHSTS: false,
			expectedCSP:  "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		},
		{
			name:         "Default config over HTTPS",
			config:       DefaultSecurityHeadersConfig(),
			useTLS:       true,
			expectedHSTS: true,
			expectedCSP:  "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		},
		{
			name: "Custom CSP policy",
			config: SecurityHeadersConfig{
				ContentSecurityPolicy: "default-src 'none'; script-src 'self'",
				EnableHSTS:            false,
			},
			useTLS:       false,
			expectedHSTS: false,
			expectedCSP:  "default-src 'none'; script-src 'self'",
		},
		{
			name: "HSTS disabled",
			config: SecurityHeadersConfig{
				ContentSecurityPolicy: "default-src 'self'",
				EnableHSTS:            false,
			},
			useTLS:       true,
			expectedHSTS: false,
		},
		{
			name: "Empty CSP",
			config: SecurityHeadersConfig{
				ContentSecurityPolicy: "",
				EnableHSTS:            false,
			},
			useTLS:      false,
			expectedCSP: "",
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set a server header to test removal
		w.Header().Set("Server", "TestServer/1.0")
		w.Header().Set("X-Powered-By", "TestFramework")
		w.WriteHeader(http.StatusOK)
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := SecurityHeaders(tt.config)
			wrappedHandler := middleware(handler)

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.useTLS {
				req.TLS = &tls.ConnectionState{}
			}
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			// Check standard security headers (always set)
			headers := []struct {
				name     string
				expected string
			}{
				{"X-Content-Type-Options", "nosniff"},
				{"X-Frame-Options", "DENY"},
				{"X-XSS-Protection", "1; mode=block"},
			}

			for _, h := range headers {
				if got := rec.Header().Get(h.name); got != h.expected {
					t.Errorf("%s header = %q, want %q", h.name, got, h.expected)
				}
			}

			// Check CSP
			if tt.expectedCSP != "" {
				if got := rec.Header().Get("Content-Security-Policy"); got != tt.expectedCSP {
					t.Errorf("Content-Security-Policy = %q, want %q", got, tt.expectedCSP)
				}
			} else {
				if got := rec.Header().Get("Content-Security-Policy"); got != "" {
					t.Errorf("Content-Security-Policy should be empty, got %q", got)
				}
			}

			// Check HSTS
			hstsHeader := rec.Header().Get("Strict-Transport-Security")
			if tt.expectedHSTS {
				if hstsHeader == "" {
					t.Error("Strict-Transport-Security header should be set")
				}
			} else {
				if hstsHeader != "" {
					t.Errorf("Strict-Transport-Security should not be set, got %q", hstsHeader)
				}
			}

			// Check that Server and X-Powered-By headers are removed
			if got := rec.Header().Get("Server"); got != "" {
				t.Errorf("Server header should be removed, got %q", got)
			}
			if got := rec.Header().Get("X-Powered-By"); got != "" {
				t.Errorf("X-Powered-By header should be removed, got %q", got)
			}

			// Check Referrer-Policy
			if tt.config.ReferrerPolicy != "" {
				if got := rec.Header().Get("Referrer-Policy"); got != tt.config.ReferrerPolicy {
					t.Errorf("Referrer-Policy = %q, want %q", got, tt.config.ReferrerPolicy)
				}
			}

			// Check Permissions-Policy
			if tt.config.PermissionsPolicy != "" {
				if got := rec.Header().Get("Permissions-Policy"); got != tt.config.PermissionsPolicy {
					t.Errorf("Permissions-Policy = %q, want %q", got, tt.config.PermissionsPolicy)
				}
			}
		})
	}
}

func TestSecureHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecureHeaders()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Check that default headers are set
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
	if got := rec.Header().Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want DENY", got)
	}
	if got := rec.Header().Get("Content-Security-Policy"); got == "" {
		t.Error("Content-Security-Policy should be set with default config")
	}
	if got := rec.Header().Get("Strict-Transport-Security"); got == "" {
		t.Error("Strict-Transport-Security should be set over HTTPS with default config")
	}
}

func TestSecurityHeaders_PreservesResponse(t *testing.T) {
	expectedBody := "test response"
	expectedStatus := http.StatusCreated

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(expectedStatus)
		w.Write([]byte(expectedBody))
	})

	middleware := SecureHeaders()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodPost, "/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != expectedStatus {
		t.Errorf("Status code = %d, want %d", rec.Code, expectedStatus)
	}

	if got := rec.Body.String(); got != expectedBody {
		t.Errorf("Body = %q, want %q", got, expectedBody)
	}
}

func TestSecurityHeaders_MultipleRequests(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecureHeaders()
	wrappedHandler := middleware(handler)

	// Make multiple requests to ensure headers are set consistently
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		wrappedHandler.ServeHTTP(rec, req)

		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("Request %d: X-Content-Type-Options = %q, want nosniff", i, got)
		}
	}
}

func BenchmarkSecurityHeaders(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecureHeaders()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

func BenchmarkSecurityHeaders_WithTLS(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := SecureHeaders()
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}

