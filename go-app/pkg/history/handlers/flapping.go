package handlers

import (
	"net/http"
	"strconv"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// GetFlappingAlerts handles GET /api/v2/history/flapping - Flapping detection
func (h *Handler) GetFlappingAlerts(w http.ResponseWriter, r *http.Request) {
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
	
	// Parse threshold (minimum transition count to be considered flapping)
	threshold := 5 // default
	if thresholdStr := queryParams.Get("threshold"); thresholdStr != "" {
		if t, err := strconv.Atoi(thresholdStr); err == nil && t > 0 {
			threshold = t
			if threshold > 100 {
				threshold = 100
			}
		}
	}
	
	// Query repository
	flappingAlerts, err := h.repository.GetFlappingAlerts(r.Context(), timeRange, threshold)
	if err != nil {
		h.logger.Error("Failed to get flapping alerts",
			"request_id", requestID,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to retrieve flapping alerts").WithRequestID(requestID))
		return
	}
	
	// Build response
	response := map[string]interface{}{
		"alerts":    flappingAlerts,
		"count":     len(flappingAlerts),
		"threshold": threshold,
	}
	if timeRange != nil {
		response["time_range"] = timeRange
	}
	
	duration := time.Since(start)
	h.logger.Info("Flapping alerts request completed",
		"request_id", requestID,
		"count", len(flappingAlerts),
		"threshold", threshold,
		"duration_ms", duration.Milliseconds())
	
	h.sendJSON(w, http.StatusOK, response)
}

