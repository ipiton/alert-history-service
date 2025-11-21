// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// GracefulDegradation provides fallback mechanisms for component failures.
// Phase 12: Error Handling enhancement.
type GracefulDegradation struct {
	logger *slog.Logger
}

// NewGracefulDegradation creates a new graceful degradation handler.
func NewGracefulDegradation(logger *slog.Logger) *GracefulDegradation {
	return &GracefulDegradation{
		logger: logger,
	}
}

// FallbackTemplateCache provides fallback when template cache fails.
func (h *SilenceUIHandler) FallbackTemplateCache(
	w http.ResponseWriter,
	r *http.Request,
	templateName string,
	data interface{},
) error {
	// Try cache first
	if h.templateCache != nil {
		cacheKey := h.generateCacheKey(templateName, data)
		cached, etag, found := h.templateCache.Get(cacheKey)
		if found {
			// Cache hit - use cached content
			ifNoneMatch := r.Header.Get("If-None-Match")
			if ifNoneMatch == etag {
				w.WriteHeader(http.StatusNotModified)
				return nil
			}
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "public, max-age=300")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, err := w.Write(cached)
			return err
		}
	}

	// Cache miss or cache unavailable - render directly
	return h.renderDirect(w, templateName, data)
}

// renderDirect renders template without caching.
func (h *SilenceUIHandler) renderDirect(
	w http.ResponseWriter,
	templateName string,
	data interface{},
) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(w, templateName, data)
}

// FallbackMetrics provides fallback when metrics fail.
func (h *SilenceUIHandler) FallbackMetrics(fn func()) {
	if h.metrics != nil {
		defer func() {
			if r := recover(); r != nil {
				h.logger.Warn("Metrics recording failed, continuing without metrics",
					"error", r,
				)
			}
		}()
		fn()
	}
}

// FallbackCSRF provides fallback when CSRF validation fails.
func (h *SilenceUIHandler) FallbackCSRF(r *http.Request) bool {
	if h.csrfManager == nil {
		// No CSRF manager - allow in development mode
		h.logger.Debug("CSRF validation skipped (no manager configured)")
		return true
	}

	// Try validation
	if !h.validateCSRFToken(r) {
		h.logger.Warn("CSRF validation failed",
			"remote_addr", r.RemoteAddr,
			"method", r.Method,
		)
		return false
	}

	return true
}

// FallbackRateLimit provides fallback when rate limiting fails.
func (h *SilenceUIHandler) FallbackRateLimit(r *http.Request) bool {
	if h.rateLimiter == nil {
		// No rate limiter - allow in development mode
		return true
	}

	ip := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	}

	// Try rate limit check
	if !h.rateLimiter.Allow(ip) {
		h.logger.Warn("Rate limit exceeded",
			"ip", ip,
			"remote_addr", r.RemoteAddr,
		)
		return false
	}

	return true
}

// FallbackWebSocket provides fallback when WebSocket fails.
func (h *SilenceUIHandler) FallbackWebSocket(eventType string, data map[string]interface{}) {
	if h.wsHub == nil {
		h.logger.Debug("WebSocket hub not available, skipping broadcast",
			"event_type", eventType,
		)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			h.logger.Warn("WebSocket broadcast failed, continuing without real-time updates",
				"error", r,
				"event_type", eventType,
			)
		}
	}()

	h.wsHub.Broadcast(eventType, data)
}

// FallbackManager provides fallback when manager operations fail.
func (h *SilenceUIHandler) FallbackManager(
	ctx context.Context,
	operation string,
	fn func() error,
) error {
	err := fn()
	if err != nil {
		h.logger.Warn("Manager operation failed, attempting graceful degradation",
			"operation", operation,
			"error", err,
		)

		// Check if error is recoverable
		if isRecoverableError(err) {
			// Retry once with timeout
			retryCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// Retry with timeout context
			done := make(chan error, 1)
			go func() {
				done <- fn()
			}()

			select {
			case retryErr := <-done:
				if retryErr != nil {
					return fmt.Errorf("operation %s failed after retry: %w", operation, retryErr)
				}
				return nil
			case <-retryCtx.Done():
				return fmt.Errorf("operation %s retry timeout: %w", operation, retryCtx.Err())
			}
		}

		return err
	}

	return nil
}

// isRecoverableError checks if an error is recoverable.
func isRecoverableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	recoverablePatterns := []string{
		"timeout",
		"connection",
		"temporary",
		"503",
		"502",
		"504",
	}

	for _, pattern := range recoverablePatterns {
		if containsError(errStr, pattern) {
			return true
		}
	}

	return false
}

// HealthCheck performs health check of all components.
func (h *SilenceUIHandler) HealthCheck(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	// Check manager
	if h.manager != nil {
		_, err := h.manager.GetStats(ctx)
		health["manager"] = err == nil
	} else {
		health["manager"] = false
	}

	// Check template cache
	if h.templateCache != nil {
		stats := h.templateCache.Stats()
		health["template_cache"] = stats["size"] != nil
	} else {
		health["template_cache"] = false
	}

	// Check CSRF manager
	health["csrf_manager"] = h.csrfManager != nil

	// Check metrics
	health["metrics"] = h.metrics != nil

	// Check WebSocket hub
	health["websocket_hub"] = h.wsHub != nil

	// Check rate limiter
	health["rate_limiter"] = h.rateLimiter != nil

	return health
}
