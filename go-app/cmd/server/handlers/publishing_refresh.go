// Package handlers provides HTTP handlers for Alert History Service API.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// Prometheus metrics for refresh endpoint (TN-67)
var (
	refreshAPIRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "publishing_refresh_api_requests_total",
			Help: "Total number of refresh API requests by status",
		},
		[]string{"status"}, // status: success, rate_limited, in_progress, not_started, error
	)

	refreshAPIDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "publishing_refresh_api_duration_seconds",
			Help:    "Refresh API endpoint latency distribution",
			Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		},
		[]string{"status"},
	)

	refreshAPIRateLimitHits = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "publishing_refresh_api_rate_limit_hits_total",
			Help: "Total number of rate limit hits on refresh endpoint",
		},
	)
)

// HandleRefreshTargets triggers manual target refresh (async).
//
// This handler:
//   1. Validates request (no body required)
//   2. Generates request ID (UUID) for tracking
//   3. Triggers async refresh via RefreshManager.RefreshNow()
//   4. Returns 202 Accepted immediately (async behavior)
//   5. Returns 503 if refresh already in progress
//   6. Returns 429 if rate limit exceeded
//
// Request:
//
//	POST /api/v2/publishing/targets/refresh
//	Content-Type: application/json
//	Body: (empty)
//
// Response (202 Accepted):
//
//	{
//	  "message": "Refresh triggered",
//	  "request_id": "550e8400-e29b-41d4-a716-446655440000",
//	  "refresh_started_at": "2025-11-08T10:30:45Z"
//	}
//
// Response (503 Service Unavailable):
//
//	{
//	  "error": "refresh_in_progress",
//	  "message": "Target refresh already running",
//	  "started_at": "2025-11-08T10:30:40Z"
//	}
//
// Response (429 Too Many Requests):
//
//	{
//	  "error": "rate_limit_exceeded",
//	  "message": "Max 1 refresh per minute",
//	  "retry_after_seconds": 45
//	}
//
// Performance: <100ms (async trigger, immediate return)
//
// Example:
//
//	mux.HandleFunc("POST /api/v2/publishing/targets/refresh",
//	    handlers.HandleRefreshTargets(refreshMgr))
func HandleRefreshTargets(refreshMgr publishing.RefreshManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()

		// Generate request ID for tracking
		requestID := uuid.New().String()

		logger := slog.With(
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)

		// Add security headers (TN-67: Security hardening)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("X-Request-ID", requestID)

		// Validate request body is empty (TN-67: Input validation)
		if r.ContentLength > 0 {
			duration := time.Since(startTime).Seconds()
			refreshAPIDuration.WithLabelValues("bad_request").Observe(duration)
			refreshAPIRequestsTotal.WithLabelValues("bad_request").Inc()

			logger.Warn("Refresh request rejected - non-empty body",
				"content_length", r.ContentLength,
				"duration_ms", time.Since(startTime).Milliseconds())

			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":      "invalid_request",
				"message":    "Request body must be empty",
				"request_id": requestID,
			})
			return
		}

		// Enforce request size limit (TN-67: Security - prevent payload attacks)
		r.Body = http.MaxBytesReader(w, r.Body, 1024) // 1KB max

		logger.Info("Manual refresh requested")

		// Trigger async refresh
		err := refreshMgr.RefreshNow()
		if err != nil {
			duration := time.Since(startTime).Seconds()

			if errors.Is(err, publishing.ErrRefreshInProgress) {
				// Refresh already running - return 503
				refreshAPIDuration.WithLabelValues("in_progress").Observe(duration)
				refreshAPIRequestsTotal.WithLabelValues("in_progress").Inc()

				status := refreshMgr.GetStatus()
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "refresh_in_progress",
					"message":    "Target refresh already running",
					"started_at": status.LastRefresh.Format(time.RFC3339),
					"request_id": requestID,
				})

				logger.Warn("Manual refresh rejected (already in progress)",
					"duration_ms", time.Since(startTime).Milliseconds())
				return
			}

			if errors.Is(err, publishing.ErrRateLimitExceeded) {
				// Rate limit exceeded - return 429 (TN-67: Rate limiting)
				refreshAPIDuration.WithLabelValues("rate_limited").Observe(duration)
				refreshAPIRequestsTotal.WithLabelValues("rate_limited").Inc()
				refreshAPIRateLimitHits.Inc()

				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":               "rate_limit_exceeded",
					"message":             "Max 1 refresh per minute",
					"retry_after_seconds": 60,
					"request_id":          requestID,
				})

				logger.Warn("Manual refresh rate limit exceeded",
					"duration_ms", time.Since(startTime).Milliseconds())
				return
			}

			if errors.Is(err, publishing.ErrNotStarted) {
				// Manager not started - return 503
				refreshAPIDuration.WithLabelValues("not_started").Observe(duration)
				refreshAPIRequestsTotal.WithLabelValues("not_started").Inc()

				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "manager_not_started",
					"message":    "Refresh manager not started",
					"request_id": requestID,
				})

				logger.Error("Manual refresh failed (manager not started)",
					"duration_ms", time.Since(startTime).Milliseconds())
				return
			}

			// Unknown error - return 500
			refreshAPIDuration.WithLabelValues("error").Observe(duration)
			refreshAPIRequestsTotal.WithLabelValues("error").Inc()

			logger.Error("Failed to trigger refresh",
				"error", err,
				"duration_ms", time.Since(startTime).Milliseconds())

			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error":      "internal_error",
				"message":    "Internal server error",
				"request_id": requestID,
			})
			return
		}

		// Success - return 202 Accepted (TN-67: Async pattern)
		duration := time.Since(startTime).Seconds()
		refreshAPIDuration.WithLabelValues("success").Observe(duration)
		refreshAPIRequestsTotal.WithLabelValues("success").Inc()

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":            "Refresh triggered",
			"request_id":         requestID,
			"refresh_started_at": time.Now().UTC().Format(time.RFC3339),
		})

		logger.Info("Manual refresh triggered successfully",
			"duration_ms", time.Since(startTime).Milliseconds())
	}
}

// HandleRefreshStatus returns current refresh status.
//
// This handler:
//   1. Gets status from RefreshManager.GetStatus()
//   2. Formats JSON response
//   3. Returns 200 OK with status details
//
// Request:
//
//	GET /api/v2/publishing/targets/status
//
// Response (200 OK):
//
//	{
//	  "status": "success",
//	  "last_refresh": "2025-11-08T10:30:45Z",
//	  "next_refresh": "2025-11-08T10:35:45Z",
//	  "refresh_duration_ms": 1856,
//	  "targets_discovered": 15,
//	  "targets_valid": 14,
//	  "targets_invalid": 1,
//	  "consecutive_failures": 0,
//	  "error": null
//	}
//
// Performance: <10ms (read-only, O(1))
//
// Example:
//
//	mux.HandleFunc("GET /api/v2/publishing/targets/status",
//	    handlers.HandleRefreshStatus(refreshMgr))
func HandleRefreshStatus(refreshMgr publishing.RefreshManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.With(
			"method", r.Method,
			"path", r.URL.Path,
		)

		logger.Debug("Refresh status requested")

		// Get status
		status := refreshMgr.GetStatus()

		// Format response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                string(status.State),
			"last_refresh":          formatTimeRFC3339(status.LastRefresh),
			"next_refresh":          formatTimeRFC3339(status.NextRefresh),
			"refresh_duration_ms":   status.RefreshDuration.Milliseconds(),
			"targets_discovered":    status.TargetsDiscovered,
			"targets_valid":         status.TargetsValid,
			"targets_invalid":       status.TargetsInvalid,
			"consecutive_failures":  status.ConsecutiveFailures,
			"error":                 status.Error,
		})

		logger.Debug("Refresh status returned",
			"state", status.State,
			"targets_valid", status.TargetsValid)
	}
}

// formatTimeRFC3339 formats time in RFC3339 format (empty string for zero time).
func formatTimeRFC3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
