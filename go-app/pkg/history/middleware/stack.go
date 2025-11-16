package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	securitymw "github.com/vitaliisemenov/alert-history/pkg/middleware"
)

// StackConfig contains configuration for the middleware stack
type StackConfig struct {
	// Recovery configuration
	EnableRecovery bool

	// RequestID configuration
	EnableRequestID bool

	// Logging configuration
	EnableLogging bool
	Logger        *slog.Logger

	// Metrics configuration
	EnableMetrics bool

	// Rate limiting configuration
	EnableRateLimit bool
	RateLimitPerIP  int // requests per second per IP
	RateLimitGlobal int // global requests per second
	RateLimitBurst  int // burst allowance

	// Authentication configuration
	EnableAuth bool
	AuthConfig middleware.AuthConfig

	// Authorization configuration
	EnableAuthz bool
	RequiredRole string // Required role for access (viewer, operator, admin)

	// CORS configuration
	EnableCORS bool
	CORSConfig middleware.CORSConfig

	// Compression configuration
	EnableCompression bool

	// Timeout configuration
	EnableTimeout bool
	Timeout       time.Duration // Request timeout (default: 30s)

	// Security headers configuration
	EnableSecurityHeaders bool
	SecurityHeadersConfig securitymw.SecurityHeadersConfig
}

// DefaultStackConfig returns default middleware stack configuration
func DefaultStackConfig(logger *slog.Logger) StackConfig {
	return StackConfig{
		EnableRecovery:        true,
		EnableRequestID:       true,
		EnableLogging:         true,
		Logger:                logger,
		EnableMetrics:         true,
		EnableRateLimit:       true,
		RateLimitPerIP:        100, // 100 req/s per IP
		RateLimitGlobal:       10000, // 10K req/s global
		RateLimitBurst:        200, // 200 requests burst
		EnableAuth:            true,
		AuthConfig: middleware.AuthConfig{
			EnableAPIKey: true,
			EnableJWT:    false,
			APIKeys:      make(map[string]*middleware.User),
		},
		EnableAuthz:           true,
		RequiredRole:           middleware.RoleViewer, // Minimum role: viewer
		EnableCORS:            true,
		CORSConfig:            middleware.DefaultCORSConfig(),
		EnableCompression:      true,
		EnableTimeout:         true,
		Timeout:               30 * time.Second,
		EnableSecurityHeaders: true,
		SecurityHeadersConfig: securitymw.DefaultSecurityHeadersConfig(),
	}
}

// Stack represents the complete middleware stack for history endpoint
type Stack struct {
	config StackConfig
}

// NewStack creates a new middleware stack
func NewStack(config StackConfig) *Stack {
	return &Stack{
		config: config,
	}
}

// Apply applies all middleware to a handler in the correct order
//
// Middleware order (critical for functionality):
//  1. Recovery - Must be outermost to catch all panics
//  2. RequestID - Generate request ID early for tracing
//  3. Logging - Log all requests
//  4. Metrics - Collect metrics
//  5. Timeout - Set request timeout
//  6. Security Headers - Set security headers
//  7. CORS - Handle CORS preflight
//  8. Compression - Compress responses
//  9. RateLimit - Rate limiting (before auth to prevent auth abuse)
// 10. Authentication - Verify API key
// 11. Authorization - Check RBAC permissions
func (s *Stack) Apply(handler http.Handler) http.Handler {
	// 1. Recovery (outermost - catches all panics)
	if s.config.EnableRecovery {
		handler = RecoveryMiddleware(s.config.Logger)(handler)
	}

	// 2. RequestID (early - needed for tracing)
	if s.config.EnableRequestID {
		handler = middleware.RequestIDMiddleware(handler)
	}

	// 3. Logging (log all requests)
	if s.config.EnableLogging && s.config.Logger != nil {
		handler = middleware.LoggingMiddleware(s.config.Logger)(handler)
	}

	// 4. Metrics (collect metrics)
	if s.config.EnableMetrics {
		handler = middleware.MetricsMiddleware(handler)
	}

	// 5. Timeout (set request timeout)
	if s.config.EnableTimeout {
		handler = TimeoutMiddleware(s.config.Timeout, s.config.Logger)(handler)
	}

	// 6. Security Headers (set security headers)
	if s.config.EnableSecurityHeaders {
		handler = securitymw.SecurityHeaders(s.config.SecurityHeadersConfig)(handler)
	}

	// 7. CORS (handle CORS preflight)
	if s.config.EnableCORS {
		handler = middleware.CORSMiddleware(s.config.CORSConfig)(handler)
	}

	// 8. Compression (compress responses)
	if s.config.EnableCompression {
		handler = middleware.CompressionMiddleware(handler)
	}

	// 9. RateLimit (rate limiting - before auth to prevent auth abuse)
	if s.config.EnableRateLimit {
		// Note: Using per-IP rate limit for now
		// Global rate limit would require shared state (Redis)
		handler = middleware.RateLimitMiddleware(s.config.RateLimitPerIP, s.config.RateLimitBurst)(handler)
	}

	// 10. Authentication (verify API key)
	if s.config.EnableAuth {
		handler = middleware.AuthMiddleware(s.config.AuthConfig)(handler)
	}

	// 11. Authorization (check RBAC permissions)
	if s.config.EnableAuthz {
		handler = middleware.RBACMiddleware(s.config.RequiredRole)(handler)
	}

	return handler
}

// ApplyFunc applies middleware stack to an http.HandlerFunc
func (s *Stack) ApplyFunc(fn http.HandlerFunc) http.Handler {
	return s.Apply(fn)
}
