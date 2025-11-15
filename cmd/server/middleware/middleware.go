// Package middleware provides HTTP middleware for the Alert History Service.
package middleware

import (
	"net/http"
)

// Middleware is a function that wraps an http.Handler
type Middleware func(http.Handler) http.Handler

// Chain builds a middleware chain from multiple middleware functions
// Middleware are applied in reverse order (last middleware wraps first)
func Chain(middlewares ...Middleware) Middleware {
	return func(final http.Handler) http.Handler {
		// Apply middleware in reverse order
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// BuildWebhookMiddlewareStack builds complete middleware stack for webhook endpoint
func BuildWebhookMiddlewareStack(config *MiddlewareConfig) Middleware {
	middlewares := []Middleware{
		// 1. Recovery (outermost - catch all panics)
		RecoveryMiddleware(config.Logger),

		// 2. RequestID (generate/validate X-Request-ID)
		RequestIDMiddleware(),

		// 3. Logging (log request/response)
		LoggingMiddleware(config.Logger),
	}

	// 4. Metrics (record Prometheus metrics) - optional
	if config.MetricsRegistry != nil {
		middlewares = append(middlewares, MetricsMiddleware(config.MetricsRegistry))
	}

	// 5. RateLimit (enforce rate limits) - optional
	if config.RateLimiter != nil && config.RateLimiter.Enabled {
		middlewares = append(middlewares, RateLimitMiddleware(config.RateLimiter))
	}

	// 6. Authentication (validate API key/JWT) - optional
	if config.AuthConfig != nil && config.AuthConfig.Enabled {
		middlewares = append(middlewares, AuthenticationMiddleware(config.AuthConfig))
	}

	// 7. Compression (gzip/deflate support) - optional
	if config.EnableCompression {
		middlewares = append(middlewares, CompressionMiddleware())
	}

	// 8. CORS (cross-origin headers) - optional
	if config.CORSConfig != nil && config.CORSConfig.Enabled {
		middlewares = append(middlewares, CORSMiddleware(config.CORSConfig))
	}

	// 9. SizeLimit (max request size)
	if config.MaxRequestSize > 0 {
		middlewares = append(middlewares, SizeLimitMiddleware(config.MaxRequestSize))
	}

	// 10. Timeout (context timeout)
	if config.RequestTimeout > 0 {
		middlewares = append(middlewares, TimeoutMiddleware(config.RequestTimeout))
	}

	return Chain(middlewares...)
}
