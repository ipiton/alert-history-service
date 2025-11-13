package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string // Allowed origins (["*"] for all)
	AllowedMethods   []string // Allowed HTTP methods
	AllowedHeaders   []string // Allowed request headers
	ExposedHeaders   []string // Headers exposed to browser
	AllowCredentials bool     // Allow credentials (cookies, auth)
	MaxAge           int      // Preflight cache duration (seconds)
}

// DefaultCORSConfig returns default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodDelete,
			http.MethodOptions,
			http.MethodPatch,
		},
		AllowedHeaders: []string{
			"Accept",
			"Accept-Language",
			"Content-Type",
			"Content-Language",
			"Origin",
			RequestIDHeader,
			AuthorizationHeader,
		},
		ExposedHeaders: []string{
			RequestIDHeader,
			RateLimitLimitHeader,
			RateLimitRemainingHeader,
			RateLimitResetHeader,
			APIVersionHeader,
		},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORSMiddleware handles Cross-Origin Resource Sharing (CORS)
//
// Configuration:
//   - AllowedOrigins: List of allowed origins (use ["*"] for all)
//   - AllowedMethods: HTTP methods allowed for CORS
//   - AllowedHeaders: Request headers allowed for CORS
//   - AllowCredentials: Allow credentials (cookies, auth headers)
//   - MaxAge: Preflight cache duration in seconds
//
// For production, restrict AllowedOrigins to specific domains.
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			// Set other CORS headers
			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				w.Header().Set("Access-Control-Max-Age", string(rune(config.MaxAge)))
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if origin is in allowed origins list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
		// Support wildcard subdomains (e.g., "*.example.com")
		if strings.HasPrefix(allowed, "*.") {
			domain := allowed[2:]
			if strings.HasSuffix(origin, domain) {
				return true
			}
		}
	}
	return false
}
