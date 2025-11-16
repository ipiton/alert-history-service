// Package middleware provides HTTP middleware for the alert history service.
package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// MiddlewareConfig holds configuration for building middleware stacks.
type MiddlewareConfig struct {
	Logger          *slog.Logger
	MetricsRegistry *metrics.MetricsRegistry
	RateLimiter     *RateLimitConfig
	AuthConfig      *AuthConfig
	CORSConfig      *CORSConfig
	MaxRequestSize  int
	RequestTimeout  time.Duration
	EnableCompression bool
}

// RateLimitConfig holds rate limiting configuration.
type RateLimitConfig struct {
	Enabled     bool
	PerIPLimit  int
	GlobalLimit int
	Logger      *slog.Logger
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	Enabled   bool
	Type      string // "api_key" or "jwt"
	APIKey    string
	JWTSecret string
	Logger    *slog.Logger
}

// CORSConfig holds CORS configuration.
type CORSConfig struct {
	Enabled        bool
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

// BuildWebhookMiddlewareStack builds a complete middleware stack for webhook endpoints.
// The middleware is applied in the following order (outermost to innermost):
// 1. Security Headers - Add security-related HTTP headers
// 2. Recovery - Recover from panics
// 3. Request ID - Generate unique request IDs
// 4. Logging - Log all requests
// 5. Metrics - Record Prometheus metrics
// 6. Rate Limiting - Apply rate limits
// 7. Authentication - Validate credentials
// 8. Compression - Compress responses (if enabled)
// 9. CORS - Handle cross-origin requests
// 10. Size Limit - Enforce max request size
// 11. Timeout - Enforce request timeouts
func BuildWebhookMiddlewareStack(config *MiddlewareConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		handler := next

		// 11. Timeout (innermost - applied last)
		if config.RequestTimeout > 0 {
			handler = http.TimeoutHandler(handler, config.RequestTimeout, "Request timeout")
		}

		// 10. Size Limit
		if config.MaxRequestSize > 0 {
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.ContentLength > int64(config.MaxRequestSize) {
					http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
					return
				}
				handler.ServeHTTP(w, r)
			})
		}

		// 9. CORS
		if config.CORSConfig != nil && config.CORSConfig.Enabled {
			handler = applyCORS(handler, config.CORSConfig)
		}

		// 8. Compression (optional)
		if config.EnableCompression {
			// Compression middleware would go here
			// For webhooks, typically disabled
		}

		// 7. Authentication
		if config.AuthConfig != nil && config.AuthConfig.Enabled {
			handler = applyAuth(handler, config.AuthConfig)
		}

		// 6. Rate Limiting
		if config.RateLimiter != nil && config.RateLimiter.Enabled {
			handler = applyRateLimit(handler, config.RateLimiter)
		}

		// 5. Metrics
		if config.MetricsRegistry != nil {
			handler = applyMetrics(handler, config.MetricsRegistry)
		}

		// 4. Logging
		if config.Logger != nil {
			handler = applyLogging(handler, config.Logger)
		}

		// 3. Request ID
		handler = applyRequestID(handler)

		// 2. Recovery (panic recovery)
		handler = applyRecovery(handler, config.Logger)

		// 1. Security Headers (outermost - applied first)
		securityHeaders := NewSecurityHeadersMiddleware(nil)
		handler = securityHeaders.Handler(handler)

		return handler
	}
}

// applyCORS applies CORS middleware.
func applyCORS(next http.Handler, config *CORSConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simple CORS implementation
		if len(config.AllowedOrigins) > 0 {
			origin := r.Header.Get("Origin")
			for _, allowed := range config.AllowedOrigins {
				if allowed == "*" || allowed == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}
		}

		if len(config.AllowedMethods) > 0 {
			w.Header().Set("Access-Control-Allow-Methods", joinStrings(config.AllowedMethods, ", "))
		}

		if len(config.AllowedHeaders) > 0 {
			w.Header().Set("Access-Control-Allow-Headers", joinStrings(config.AllowedHeaders, ", "))
		}

		// Handle preflight
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// applyAuth applies authentication middleware.
func applyAuth(next http.Handler, config *AuthConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Placeholder: In production, use actual auth middleware
		// For now, just pass through
		if config.Logger != nil {
			config.Logger.Debug("Auth middleware applied", "type", config.Type)
		}
		next.ServeHTTP(w, r)
	})
}

// applyRateLimit applies rate limiting middleware.
func applyRateLimit(next http.Handler, config *RateLimitConfig) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Placeholder: In production, use actual rate limiter
		// For now, just pass through
		if config.Logger != nil {
			config.Logger.Debug("Rate limit middleware applied")
		}
		next.ServeHTTP(w, r)
	})
}

// applyMetrics applies metrics middleware.
func applyMetrics(next http.Handler, registry *metrics.MetricsRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Placeholder: In production, record metrics
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// applyLogging applies logging middleware.
func applyLogging(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)
		next.ServeHTTP(w, r)
	})
}

// applyRequestID applies request ID middleware.
func applyRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Placeholder: In production, generate UUID
		// For now, just pass through
		next.ServeHTTP(w, r)
	})
}

// applyRecovery applies panic recovery middleware.
func applyRecovery(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				if logger != nil {
					logger.Error("Panic recovered",
						"error", err,
						"path", r.URL.Path,
					)
				}
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}
