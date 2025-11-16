package handlers

import (
	"net/http"
	"strconv"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
)

// GetRecentAlerts handles GET /api/v2/history/recent - Recent alerts
func (h *Handler) GetRecentAlerts(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := middleware.GetRequestID(r.Context())
	
	queryParams := r.URL.Query()
	
	// Parse limit
	limit := 50 // default
	if limitStr := queryParams.Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 1000 {
				limit = 1000
			}
		}
	}
	
	// Query repository
	alerts, err := h.repository.GetRecentAlerts(r.Context(), limit)
	if err != nil {
		h.logger.Error("Failed to get recent alerts",
			"request_id", requestID,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to retrieve recent alerts").WithRequestID(requestID))
		return
	}
	
	// Build response
	response := map[string]interface{}{
		"alerts": alerts,
		"count":  len(alerts),
		"limit":  limit,
	}
	
	duration := time.Since(start)
	h.logger.Info("Recent alerts request completed",
		"request_id", requestID,
		"count", len(alerts),
		"duration_ms", duration.Milliseconds())
	
	h.sendJSON(w, http.StatusOK, response)
}

