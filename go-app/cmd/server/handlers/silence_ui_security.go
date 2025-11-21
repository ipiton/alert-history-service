// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"net/http"
	"strings"
	"time"
)

// SecurityConfig defines security configuration for UI endpoints.
// Phase 13: Security Hardening enhancement.
type SecurityConfig struct {
	AllowedOrigins []string
	RateLimitEnabled bool
	RateLimitPerIP  int
	RateLimitWindow time.Duration
}

// DefaultSecurityConfig returns default security configuration.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		AllowedOrigins:   []string{"*"}, // Allow all origins by default (development)
		RateLimitEnabled: false,          // Disabled by default
		RateLimitPerIP:   100,            // 100 requests per window
		RateLimitWindow:  1 * time.Minute,
	}
}

// validateOrigin checks if the request origin is allowed.
func (h *SilenceUIHandler) validateOrigin(r *http.Request) bool {
	if h.securityConfig == nil {
		// No security config, allow all (development mode)
		return true
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		// No origin header, allow (same-origin request)
		return true
	}

	// Check if origin is in allowed list
	for _, allowed := range h.securityConfig.AllowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
		// Support wildcard patterns like "https://*.example.com"
		if strings.HasPrefix(allowed, "*.") {
			domain := strings.TrimPrefix(allowed, "*.")
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}

	h.logger.Warn("Origin not allowed",
		"origin", origin,
		"allowed_origins", h.securityConfig.AllowedOrigins,
	)

	return false
}

// sanitizeInput sanitizes user input to prevent XSS and injection attacks.
func sanitizeInput(input string) string {
	// Remove null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// Trim whitespace
	input = strings.TrimSpace(input)

	// Escape HTML special characters (basic XSS prevention)
	// Note: html/template already does this, but we add extra layer
	input = strings.ReplaceAll(input, "<", "&lt;")
	input = strings.ReplaceAll(input, ">", "&gt;")
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")

	return input
}

// validateEmail validates an email address format.
func validateEmail(email string) bool {
	if email == "" {
		return false
	}

	// Basic email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	local, domain := parts[0], parts[1]
	if len(local) == 0 || len(local) > 64 {
		return false
	}
	if len(domain) == 0 || len(domain) > 255 {
		return false
	}

	// Check for valid characters
	if !strings.Contains(domain, ".") {
		return false
	}

	return true
}

// validateUUID validates a UUID v4 format.
func validateUUID(uuid string) bool {
	if len(uuid) != 36 {
		return false
	}

	// Basic UUID v4 format check: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
	parts := strings.Split(uuid, "-")
	if len(parts) != 5 {
		return false
	}

	// Check lengths
	if len(parts[0]) != 8 || len(parts[1]) != 4 || len(parts[2]) != 4 ||
		len(parts[3]) != 4 || len(parts[4]) != 12 {
		return false
	}

	// Check version (4) in third part
	if parts[2][0] != '4' {
		return false
	}

	return true
}

// sanitizePath prevents path traversal attacks.
func sanitizePath(path string) string {
	// Remove any path traversal attempts
	path = strings.ReplaceAll(path, "../", "")
	path = strings.ReplaceAll(path, "..\\", "")
	path = strings.ReplaceAll(path, "//", "/")
	path = strings.TrimPrefix(path, "/")

	return path
}

// SetSecurityConfig sets the security configuration.
func (h *SilenceUIHandler) SetSecurityConfig(config *SecurityConfig) {
	h.securityConfig = config
}

// SecurityMiddleware provides security features for UI endpoints.
func (h *SilenceUIHandler) SecurityMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Origin check (Phase 13 enhancement)
		if !h.validateOrigin(r) {
			h.logger.Warn("Origin validation failed",
				"origin", r.Header.Get("Origin"),
				"remote_addr", r.RemoteAddr,
			)
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}

		// Set security headers
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Continue to next handler
		next(w, r)
	}
}
