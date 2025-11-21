// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"log/slog"
	"net/http"
	"time"
)

// StructuredLogging provides enhanced structured logging for UI operations.
// Phase 14: Observability enhancement.
type StructuredLogging struct {
	logger *slog.Logger
}

// NewStructuredLogging creates a new structured logging instance.
func NewStructuredLogging(logger *slog.Logger) *StructuredLogging {
	return &StructuredLogging{
		logger: logger,
	}
}

// LogPageRender logs a page render operation.
func (sl *StructuredLogging) LogPageRender(
	page string,
	duration time.Duration,
	statusCode int,
	userAgent string,
	remoteAddr string,
) {
	sl.logger.Info("Page rendered",
		"page", page,
		"duration_ms", duration.Milliseconds(),
		"status_code", statusCode,
		"user_agent", userAgent,
		"remote_addr", remoteAddr,
	)
}

// LogUserAction logs a user action.
func (sl *StructuredLogging) LogUserAction(
	action string,
	status string,
	userID string,
	silenceID string,
) {
	sl.logger.Info("User action",
		"action", action,
		"status", status,
		"user_id", userID,
		"silence_id", silenceID,
	)
}

// LogError logs an error with context.
func (sl *StructuredLogging) LogError(
	err error,
	operation string,
	context map[string]interface{},
) {
	attrs := []interface{}{
		"error", err.Error(),
		"operation", operation,
	}
	for k, v := range context {
		attrs = append(attrs, k, v)
	}
	sl.logger.Error("Operation failed", attrs...)
}

// LogCacheOperation logs a cache operation.
func (sl *StructuredLogging) LogCacheOperation(
	operation string,
	key string,
	hit bool,
	duration time.Duration,
) {
	level := slog.LevelDebug
	if !hit {
		level = slog.LevelInfo
	}
	sl.logger.Log(nil, level, "Cache operation",
		"operation", operation,
		"key", key,
		"hit", hit,
		"duration_ns", duration.Nanoseconds(),
	)
}

// LogSecurityEvent logs a security-related event.
func (sl *StructuredLogging) LogSecurityEvent(
	eventType string,
	remoteAddr string,
	userAgent string,
	details map[string]interface{},
) {
	attrs := []interface{}{
		"event_type", eventType,
		"remote_addr", remoteAddr,
		"user_agent", userAgent,
	}
	for k, v := range details {
		attrs = append(attrs, k, v)
	}
	sl.logger.Warn("Security event", attrs...)
}

// LogPerformanceMetric logs a performance metric.
func (sl *StructuredLogging) LogPerformanceMetric(
	metric string,
	value float64,
	unit string,
	labels map[string]string,
) {
	attrs := []interface{}{
		"metric", metric,
		"value", value,
		"unit", unit,
	}
	for k, v := range labels {
		attrs = append(attrs, k, v)
	}
	sl.logger.Info("Performance metric", attrs...)
}

// Enhanced logging methods for SilenceUIHandler

// logPageRenderWithContext logs page render with full context.
func (h *SilenceUIHandler) logPageRenderWithContext(
	r *http.Request,
	page string,
	duration time.Duration,
	statusCode int,
) {
	if h.logger != nil {
		h.logger.Info("Page rendered",
			"page", page,
			"duration_ms", duration.Milliseconds(),
			"status_code", statusCode,
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"referer", r.Referer(),
		)
	}
}

// logUserActionWithContext logs user action with full context.
func (h *SilenceUIHandler) logUserActionWithContext(
	r *http.Request,
	action string,
	status string,
	silenceID string,
) {
	if h.logger != nil {
		h.logger.Info("User action",
			"action", action,
			"status", status,
			"silence_id", silenceID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	}
}

// logErrorWithContext logs error with full context.
func (h *SilenceUIHandler) logErrorWithContext(
	r *http.Request,
	err error,
	operation string,
) {
	if h.logger != nil {
		h.logger.Error("Operation failed",
			"error", err.Error(),
			"operation", operation,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	}
}

// logCacheHit logs a cache hit.
func (h *SilenceUIHandler) logCacheHit(key string, duration time.Duration) {
	if h.logger != nil {
		h.logger.Debug("Cache hit",
			"key", key,
			"duration_ns", duration.Nanoseconds(),
		)
	}
}

// logCacheMiss logs a cache miss.
func (h *SilenceUIHandler) logCacheMiss(key string) {
	if h.logger != nil {
		h.logger.Debug("Cache miss",
			"key", key,
		)
	}
}

// logSecurityEvent logs a security event.
func (h *SilenceUIHandler) logSecurityEvent(
	r *http.Request,
	eventType string,
	details map[string]interface{},
) {
	if h.logger != nil {
		attrs := []interface{}{
			"event_type", eventType,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
			"method", r.Method,
			"path", r.URL.Path,
		}
		for k, v := range details {
			attrs = append(attrs, k, v)
		}
		h.logger.Warn("Security event", attrs...)
	}
}
