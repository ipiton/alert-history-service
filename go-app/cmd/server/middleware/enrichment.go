// Package middleware provides HTTP middleware components.
package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// contextKey is a private type for context keys
type contextKey string

const (
	// EnrichmentModeKey is the context key for enrichment mode
	EnrichmentModeKey contextKey = "enrichment_mode"
	// EnrichmentSourceKey is the context key for enrichment mode source
	EnrichmentSourceKey contextKey = "enrichment_mode_source"
)

// EnrichmentModeMiddleware adds enrichment mode to request context
type EnrichmentModeMiddleware struct {
	manager services.EnrichmentModeManager
	logger  *slog.Logger
}

// NewEnrichmentModeMiddleware creates a new enrichment mode middleware
func NewEnrichmentModeMiddleware(manager services.EnrichmentModeManager, logger *slog.Logger) *EnrichmentModeMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return &EnrichmentModeMiddleware{
		manager: manager,
		logger:  logger,
	}
}

// Middleware wraps an HTTP handler with enrichment mode context
func (m *EnrichmentModeMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get current enrichment mode
		mode, source, err := m.manager.GetModeWithSource(r.Context())
		if err != nil {
			m.logger.Warn("Failed to get enrichment mode, using default",
				"error", err,
				"path", r.URL.Path,
			)
			mode = services.EnrichmentModeEnriched
			source = "default"
		}

		// Add to context
		ctx := context.WithValue(r.Context(), EnrichmentModeKey, mode)
		ctx = context.WithValue(ctx, EnrichmentSourceKey, source)

		// Add to response headers for debugging
		w.Header().Set("X-Enrichment-Mode", mode.String())
		w.Header().Set("X-Enrichment-Source", source)

		// Call next handler
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetEnrichmentModeFromContext extracts enrichment mode from context
func GetEnrichmentModeFromContext(ctx context.Context) (services.EnrichmentMode, bool) {
	mode, ok := ctx.Value(EnrichmentModeKey).(services.EnrichmentMode)
	return mode, ok
}

// GetEnrichmentSourceFromContext extracts enrichment mode source from context
func GetEnrichmentSourceFromContext(ctx context.Context) (string, bool) {
	source, ok := ctx.Value(EnrichmentSourceKey).(string)
	return source, ok
}

// MustGetEnrichmentModeFromContext extracts enrichment mode or returns default
func MustGetEnrichmentModeFromContext(ctx context.Context) services.EnrichmentMode {
	if mode, ok := GetEnrichmentModeFromContext(ctx); ok {
		return mode
	}
	return services.EnrichmentModeEnriched
}
