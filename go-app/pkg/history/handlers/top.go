package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// GetTopAlerts handles GET /api/v2/history/top - Top firing alerts
func (h *Handler) GetTopAlerts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := middleware.GetRequestID(r.Context())

	queryParams := r.URL.Query()

	// Parse time range
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

	// Parse limit
	limit := 10 // default
	if limitStr := queryParams.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 100 {
				limit = 100
			}
		}
	}

	// Query repository
	topAlerts, err := h.repository.GetTopAlerts(r.Context(), timeRange, limit)
	if err != nil {
		h.logger.Error("Failed to get top alerts",
			"request_id", requestID,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to retrieve top alerts").WithRequestID(requestID))
		return
	}

	// Build response
	response := map[string]interface{}{
		"alerts": topAlerts,
		"count":  len(topAlerts),
		"limit":  limit,
	}
	if timeRange != nil {
		response["time_range"] = timeRange
	}

	duration := time.Since(start)
	h.logger.Info("Top alerts request completed",
		"request_id", requestID,
		"count", len(topAlerts),
		"duration_ms", duration.Milliseconds())

	h.sendJSON(w, http.StatusOK, response)
}
