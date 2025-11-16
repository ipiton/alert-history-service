// Package middleware provides HTTP middleware for the alert history service.
package middleware

import (
	"net/http"
)

// SecurityHeadersConfig holds configuration for security headers middleware.
type SecurityHeadersConfig struct {
	// Enable/disable security headers
	Enabled bool

	// Custom headers (optional overrides)
	CustomHeaders map[string]string
}

// SecurityHeadersMiddleware adds security headers to all responses.
type SecurityHeadersMiddleware struct {
	config *SecurityHeadersConfig
}

// NewSecurityHeadersMiddleware creates a new security headers middleware.
func NewSecurityHeadersMiddleware(config *SecurityHeadersConfig) *SecurityHeadersMiddleware {
	if config == nil {
		config = DefaultSecurityHeadersConfig()
	}
	return &SecurityHeadersMiddleware{
		config: config,
	}
}

// DefaultSecurityHeadersConfig returns default security headers configuration.
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled:       true,
		CustomHeaders: make(map[string]string),
	}
}

// Handler returns the middleware handler.
func (m *SecurityHeadersMiddleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !m.config.Enabled {
			next.ServeHTTP(w, r)
			return
		}

		// Apply default security headers
		m.setSecurityHeaders(w)

		// Apply custom headers (override defaults if specified)
		for key, value := range m.config.CustomHeaders {
			w.Header().Set(key, value)
		}

		next.ServeHTTP(w, r)
	})
}

// setSecurityHeaders applies standard security headers.
func (m *SecurityHeadersMiddleware) setSecurityHeaders(w http.ResponseWriter) {
	// Prevent MIME type sniffing
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Content-Type-Options
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Frame-Options
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS protection (legacy, but still useful)
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-XSS-Protection
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Enforce HTTPS (only set if request came via HTTPS)
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Strict-Transport-Security
	// Note: In production, this is typically set by the ingress/load balancer
	// We set it here as defense-in-depth
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")

	// Control referrer information
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Referrer-Policy
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Content Security Policy (strict for API)
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/CSP
	// For webhook API, we don't expect any content rendering
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

	// Permissions Policy (formerly Feature-Policy)
	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Permissions-Policy
	// Disable all browser features for API endpoint
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

	// Remove server identification header (defense in depth)
	w.Header().Set("Server", "")
	w.Header().Del("X-Powered-By")
}

