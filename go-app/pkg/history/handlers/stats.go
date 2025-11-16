package handlers

import (
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// GetStats handles GET /api/v2/history/stats - Aggregated statistics
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := middleware.GetRequestID(r.Context())

	queryParams := r.URL.Query()

	// Parse time range (required for stats)
	var timeRange *core.TimeRange
	if fromStr := queryParams.Get("from"); fromStr != "" {
		if from, err := time.Parse(time.RFC3339, fromStr); err == nil {
			timeRange = &core.TimeRange{
				From: &from,
			}
		}
	}
	if toStr := queryParams.Get("to"); toStr != "" {
		if to, err := time.Parse(time.RFC3339, toStr); err == nil {
			if timeRange == nil {
				timeRange = &core.TimeRange{}
			}
			timeRange.To = &to
		}
	}

	// Time range is optional - if not provided, use default (last 24 hours)
	if timeRange == nil {
		now := time.Now()
		from := now.Add(-24 * time.Hour)
		timeRange = &core.TimeRange{
			From: &from,
			To:   &now,
		}
	}

	// Query repository
	stats, err := h.repository.GetAggregatedStats(r.Context(), timeRange)
	if err != nil {
		h.logger.Error("Failed to get aggregated stats",
			"request_id", requestID,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to retrieve aggregated statistics").WithRequestID(requestID))
		return
	}

	duration := time.Since(start)
	h.logger.Info("Stats request completed",
		"request_id", requestID,
		"total_alerts", stats.TotalAlerts,
		"duration_ms", duration.Milliseconds())

	h.sendJSON(w, http.StatusOK, stats)
}
