package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestInputValidator_ValidateQueryParams tests query parameter validation
func TestInputValidator_ValidateQueryParams(t *testing.T) {
	validator := NewInputValidator(nil, DefaultValidatorConfig())
	
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid query params",
			url:     "/api/v2/history?page=1&per_page=50",
			wantErr: false,
		},
		{
			name:    "invalid page (too large)",
			url:     "/api/v2/history?page=20000",
			wantErr: true,
		},
		{
			name:    "invalid per_page (too large)",
			url:     "/api/v2/history?per_page=2000",
			wantErr: true,
		},
		{
			name:    "invalid sort_field",
			url:     "/api/v2/history?sort_field=invalid_field",
			wantErr: true,
		},
		{
			name:    "SQL injection attempt",
			url:     "/api/v2/history?status='; DROP TABLE alerts; --",
			wantErr: true,
		},
		{
			name:    "XSS attempt",
			url:     "/api/v2/history?status=<script>alert('xss')</script>",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.url, nil)
			err := validator.ValidateQueryParams(req)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateQueryParams() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestInputValidator_ValidateFingerprint tests fingerprint validation
func TestInputValidator_ValidateFingerprint(t *testing.T) {
	validator := NewInputValidator(nil, DefaultValidatorConfig())
	
	tests := []struct {
		name        string
		fingerprint string
		wantErr     bool
	}{
		{
			name:        "valid fingerprint",
			fingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef1234567890",
			wantErr:     false,
		},
		{
			name:        "invalid length",
			fingerprint: "short",
			wantErr:     true,
		},
		{
			name:        "invalid characters",
			fingerprint: "a1b2c3d4e5f6789012345678901234567890abcdef1234567890abcdef123456789g",
			wantErr:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFingerprint(tt.fingerprint)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFingerprint() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestRequestSizeLimiter tests request size limiting
func TestRequestSizeLimiter(t *testing.T) {
	limiter := NewRequestSizeLimiter(1024, nil) // 1KB limit
	
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	
	mw := limiter.Middleware()
	wrapped := mw(handler)
	
	req := httptest.NewRequest("POST", "/test", nil)
	req.ContentLength = 2048 // Exceeds limit
	
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("RequestSizeLimiter() status = %v, want %v", w.Code, http.StatusBadRequest)
	}
}

// TestAuditLogger tests audit logging
func TestAuditLogger(t *testing.T) {
	logger := NewAuditLogger(nil)
	
	req := httptest.NewRequest("GET", "/api/v2/history", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	
	// Test authentication failure logging
	logger.LogAuthenticationFailure(req.Context(), req, "Invalid API key")
	
	// Test authorization failure logging
	logger.LogAuthorizationFailure(req.Context(), req, "user123", "Insufficient permissions")
	
	// Test rate limit violation logging
	logger.LogRateLimitViolation(req.Context(), req, 100, 0)
	
	// Test input validation failure logging
	logger.LogInputValidationFailure(req.Context(), req, "Invalid query parameter")
	
	// Test suspicious activity logging
	logger.LogSuspiciousActivity(req.Context(), req, "Multiple failed login attempts", map[string]interface{}{
		"attempts": 5,
		"time_window": "5 minutes",
	})
}

