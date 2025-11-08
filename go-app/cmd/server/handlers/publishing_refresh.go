// Package handlers provides HTTP handlers for Alert History Service API.
package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
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
		// Generate request ID for tracking
		requestID := uuid.New().String()

		logger := slog.With(
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		logger.Info("Manual refresh requested")

		// Trigger async refresh
		err := refreshMgr.RefreshNow()
		if err != nil {
			if errors.Is(err, publishing.ErrRefreshInProgress) {
				// Refresh already running - return 503
				status := refreshMgr.GetStatus()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "refresh_in_progress",
					"message":    "Target refresh already running",
					"started_at": status.LastRefresh.Format(time.RFC3339),
				})
				logger.Warn("Manual refresh rejected (already in progress)")
				return
			}

			if errors.Is(err, publishing.ErrRateLimitExceeded) {
				// Rate limit exceeded - return 429
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":               "rate_limit_exceeded",
					"message":             "Max 1 refresh per minute",
					"retry_after_seconds": 60,
				})
				logger.Warn("Manual refresh rate limit exceeded")
				return
			}

			if errors.Is(err, publishing.ErrNotStarted) {
				// Manager not started - return 503
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "manager_not_started",
					"message": "Refresh manager not started",
				})
				logger.Error("Manual refresh failed (manager not started)")
				return
			}

			// Unknown error - return 500
			logger.Error("Failed to trigger refresh", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Success - return 202 Accepted
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":             "Refresh triggered",
			"request_id":          requestID,
			"refresh_started_at":  time.Now().Format(time.RFC3339),
		})

		logger.Info("Manual refresh triggered successfully")
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
