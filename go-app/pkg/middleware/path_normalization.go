// Package middleware provides HTTP middleware for the Alert History Service.
package middleware

import (
	"net/http"
	"regexp"
	"strings"
)

// PathNormalizationMiddleware normalizes URL paths to reduce cardinality in metrics.
//
// Replaces dynamic path segments (UUIDs, numeric IDs) with placeholders to prevent
// metrics explosion. This is critical for HTTP metrics that include the `path` label.
//
// Transformations:
//   - UUIDs → :id (e.g., /api/alerts/123e4567-... → /api/alerts/:id)
//   - Numeric IDs → :id (e.g., /api/alerts/12345 → /api/alerts/:id)
//   - Preserves static paths unchanged
//
// Example:
//
//	router.Use(PathNormalizationMiddleware())
type PathNormalizer struct {
	uuidPattern      *regexp.Regexp
	numericIDPattern *regexp.Regexp
}

// NewPathNormalizer creates a new path normalizer with default patterns.
//
// Returns:
//   - *PathNormalizer: Configured normalizer instance
func NewPathNormalizer() *PathNormalizer {
	return &PathNormalizer{
		// UUID pattern: 8-4-4-4-12 hex digits
		uuidPattern: regexp.MustCompile(`/[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`),

		// Numeric ID pattern: 1-20 digits (covers int32, int64)
		numericIDPattern: regexp.MustCompile(`/\d{1,20}(?:/|$)`),
	}
}

// NormalizePath normalizes a URL path by replacing dynamic segments.
//
// Parameters:
//   - path: The original URL path
//
// Returns:
//   - string: The normalized path with placeholders
//
// Examples:
//
//	"/api/alerts/123e4567-e89b-12d3-a456-426614174000" → "/api/alerts/:id"
//	"/api/alerts/12345" → "/api/alerts/:id"
//	"/api/alerts/12345/comments/67890" → "/api/alerts/:id/comments/:id"
//	"/api/health" → "/api/health" (unchanged)
func (n *PathNormalizer) NormalizePath(path string) string {
	// Handle empty or root path
	if path == "" || path == "/" {
		return path
	}

	// Replace UUIDs first (more specific pattern)
	normalized := n.uuidPattern.ReplaceAllString(path, "/:id")

	// Then replace numeric IDs
	normalized = n.numericIDPattern.ReplaceAllString(normalized, "/:id/")

	// Clean up trailing slash if added by replacement
	normalized = strings.TrimSuffix(normalized, "/")

	// Ensure root path is preserved
	if normalized == "" {
		return "/"
	}

	return normalized
}

// Middleware returns an HTTP middleware that normalizes paths.
//
// Wraps the request with a custom ResponseWriter that captures the normalized path
// for use in downstream middleware (especially metrics middleware).
//
// Returns:
//   - func(http.Handler) http.Handler: Middleware function
//
// Example:
//
//	normalizer := NewPathNormalizer()
//	router.Use(normalizer.Middleware())
func (n *PathNormalizer) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Normalize the path
			originalPath := r.URL.Path
			normalizedPath := n.NormalizePath(originalPath)

			// Store normalized path in request context for metrics middleware
			// Note: We don't modify r.URL.Path directly as it might break routing
			// Instead, metrics middleware should use the normalized path from context

			// For now, we'll add a custom header that metrics middleware can read
			// This is a simple approach that works with existing metrics middleware
			r.Header.Set("X-Normalized-Path", normalizedPath)

			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	}
}

// PathNormalizationMiddleware returns a path normalization middleware.
//
// Convenience function that creates a PathNormalizer and returns its middleware.
//
// Returns:
//   - func(http.Handler) http.Handler: Middleware function
//
// Example:
//
//	router.Use(PathNormalizationMiddleware())
func PathNormalizationMiddleware() func(http.Handler) http.Handler {
	normalizer := NewPathNormalizer()
	return normalizer.Middleware()
}
