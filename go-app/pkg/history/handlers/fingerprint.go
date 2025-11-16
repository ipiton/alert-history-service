package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
)

// GetAlertTimeline handles GET /api/v2/history/{fingerprint} - Single alert timeline
func (h *Handler) GetAlertTimeline(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	requestID := middleware.GetRequestID(r.Context())

	// Extract fingerprint from URL
	vars := mux.Vars(r)
	fingerprint := vars["fingerprint"]

	if fingerprint == "" {
		apierrors.WriteError(w, apierrors.ValidationError("fingerprint parameter is required").WithRequestID(requestID))
		return
	}

	// Validate fingerprint format (64 hex characters)
	if len(fingerprint) != 64 {
		apierrors.WriteError(w, apierrors.ValidationError("invalid fingerprint format: must be 64 hex characters").WithRequestID(requestID))
		return
	}

	// Parse limit query parameter
	limit := 100 // default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	// Query repository
	alerts, err := h.repository.GetAlertsByFingerprint(r.Context(), fingerprint, limit)
	if err != nil {
		h.logger.Error("Failed to get alert timeline",
			"request_id", requestID,
			"fingerprint", fingerprint,
			"error", err)
		apierrors.WriteError(w, apierrors.InternalError("Failed to retrieve alert timeline").WithRequestID(requestID))
		return
	}

	// Build response
	response := map[string]interface{}{
		"fingerprint": fingerprint,
		"alerts":      alerts,
		"count":       len(alerts),
		"limit":       limit,
	}

	duration := time.Since(start)
	h.logger.Info("Alert timeline request completed",
		"request_id", requestID,
		"fingerprint", fingerprint,
		"count", len(alerts),
		"duration_ms", duration.Milliseconds())

	h.sendJSON(w, http.StatusOK, response)
}
