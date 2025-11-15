package middleware

import (
	"log/slog"
	"time"

	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// MiddlewareConfig holds configuration for all middleware
type MiddlewareConfig struct {
	// Logger for logging middleware
	Logger *slog.Logger

	// MetricsRegistry for metrics middleware
	MetricsRegistry *metrics.Registry

	// RateLimiter configuration
	RateLimiter *RateLimitConfig

	// AuthConfig for authentication middleware
	AuthConfig *AuthConfig

	// CORSConfig for CORS middleware
	CORSConfig *CORSConfig

	// MaxRequestSize for size limit middleware (bytes)
	MaxRequestSize int64

	// RequestTimeout for timeout middleware
	RequestTimeout time.Duration

	// EnableCompression enables gzip compression
	EnableCompression bool
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	Enabled     bool   // Enable rate limiting
	PerIPLimit  int    // Requests per minute per IP
	GlobalLimit int    // Requests per minute globally
	Logger      *slog.Logger
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled   bool   // Enable authentication
	Type      string // "api_key", "jwt", "hmac"
	APIKey    string // API key for validation
	JWTSecret string // JWT secret for validation
	Logger    *slog.Logger
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	Enabled        bool   // Enable CORS
	AllowedOrigins string // e.g., "*" or "https://example.com"
	AllowedMethods string // e.g., "POST, OPTIONS"
	AllowedHeaders string // e.g., "Content-Type, X-API-Key"
}

