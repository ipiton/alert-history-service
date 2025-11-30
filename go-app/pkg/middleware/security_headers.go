package middleware

import (
	"net/http"
)

// SecurityHeadersConfig contains configuration for security headers middleware.
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy defines the CSP header value
	ContentSecurityPolicy string

	// StrictTransportSecurity defines the HSTS header value (HTTPS only)
	StrictTransportSecurity string

	// ReferrerPolicy defines the Referrer-Policy header value
	ReferrerPolicy string

	// PermissionsPolicy defines the Permissions-Policy header value
	PermissionsPolicy string

	// EnableHSTS enables HTTP Strict Transport Security (only over HTTPS)
	EnableHSTS bool
}

// DefaultSecurityHeadersConfig returns the default security headers configuration.
func DefaultSecurityHeadersConfig() SecurityHeadersConfig {
	return SecurityHeadersConfig{
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy:       "geolocation=(), microphone=(), camera=()",
		EnableHSTS:              true,
	}
}

// SecurityHeaders returns a middleware that sets security-related HTTP headers.
//
// Headers set:
// - X-Content-Type-Options: nosniff (prevents MIME type sniffing)
// - X-Frame-Options: DENY (prevents clickjacking)
// - X-XSS-Protection: 1; mode=block (enables XSS filter)
// - Content-Security-Policy: configurable CSP policy
// - Strict-Transport-Security: configurable HSTS (HTTPS only)
// - Referrer-Policy: configurable referrer policy
// - Permissions-Policy: configurable permissions policy
//
// Security benefits:
// - Prevents MIME type confusion attacks
// - Protects against clickjacking
// - Enables browser XSS protection
// - Restricts resource loading (CSP)
// - Forces HTTPS (HSTS)
// - Controls referrer information leakage
// - Restricts browser features access
func SecurityHeaders(config SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prevent MIME type sniffing
			// Protects against: MIME confusion attacks
			w.Header().Set("X-Content-Type-Options", "nosniff")

			// Prevent clickjacking
			// Protects against: UI redress attacks, clickjacking
			w.Header().Set("X-Frame-Options", "DENY")

			// Enable XSS filter in older browsers
			// Protects against: Cross-Site Scripting (XSS)
			// Note: Modern browsers use CSP instead
			w.Header().Set("X-XSS-Protection", "1; mode=block")

			// Content Security Policy
			// Protects against: XSS, injection attacks, unauthorized resource loading
			if config.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", config.ContentSecurityPolicy)
			}

			// HTTP Strict Transport Security (HSTS)
			// Protects against: Man-in-the-middle attacks, protocol downgrade attacks
			// Only set over HTTPS to avoid browser warnings
			if config.EnableHSTS && r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", config.StrictTransportSecurity)
			}

			// Referrer Policy
			// Protects against: Information leakage via referrer header
			if config.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

		// Permissions Policy (formerly Feature-Policy)
		// Protects against: Unauthorized access to browser features
		if config.PermissionsPolicy != "" {
			w.Header().Set("Permissions-Policy", config.PermissionsPolicy)
		}

		next.ServeHTTP(w, r)

		// Remove potentially sensitive server information (after handler runs)
		w.Header().Del("Server")
		w.Header().Del("X-Powered-By")
		})
	}
}

// SecureHeaders is a convenience wrapper around SecurityHeaders with default configuration.
func SecureHeaders() func(http.Handler) http.Handler {
	return SecurityHeaders(DefaultSecurityHeadersConfig())
}
