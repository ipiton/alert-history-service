// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// EnrichmentHandlers handles enrichment mode endpoints
type EnrichmentHandlers struct {
	manager services.EnrichmentModeManager
	logger  *slog.Logger
}

// NewEnrichmentHandlers creates new enrichment handlers
func NewEnrichmentHandlers(manager services.EnrichmentModeManager, logger *slog.Logger) *EnrichmentHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &EnrichmentHandlers{
		manager: manager,
		logger:  logger,
	}
}

// EnrichmentModeResponse represents the enrichment mode response
type EnrichmentModeResponse struct {
	Mode   string `json:"mode"`
	Source string `json:"source"`
}

// SetEnrichmentModeRequest represents the request to set enrichment mode
type SetEnrichmentModeRequest struct {
	Mode string `json:"mode"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
}

// GetMode handles GET /enrichment/mode
func (h *EnrichmentHandlers) GetMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Info("Get enrichment mode requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	mode, source, err := h.manager.GetModeWithSource(ctx)
	if err != nil {
		h.logger.Error("Failed to get enrichment mode", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get enrichment mode"})
		return
	}

	response := EnrichmentModeResponse{
		Mode:   mode.String(),
		Source: source,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		return
	}

	h.logger.Info("Get enrichment mode completed",
		"mode", mode,
		"source", source,
	)
}

// SetMode handles POST /enrichment/mode
func (h *EnrichmentHandlers) SetMode(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	h.logger.Info("Set enrichment mode requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
	)

	// Parse request body
	var req SetEnrichmentModeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid JSON in request body", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
		return
	}

	// Validate mode
	mode := services.EnrichmentMode(req.Mode)
	if err := h.manager.ValidateMode(mode); err != nil {
		h.logger.Warn("Invalid enrichment mode", "mode", req.Mode, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}

	// Set mode
	if err := h.manager.SetMode(ctx, mode); err != nil {
		h.logger.Error("Failed to set enrichment mode", "mode", mode, "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to set enrichment mode"})
		return
	}

	// Get updated state
	updatedMode, source, err := h.manager.GetModeWithSource(ctx)
	if err != nil {
		h.logger.Error("Failed to get updated mode", "error", err)
		// Still return success since mode was set
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(EnrichmentModeResponse{
			Mode:   mode.String(),
			Source: "unknown",
		})
		return
	}

	response := EnrichmentModeResponse{
		Mode:   updatedMode.String(),
		Source: source,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		return
	}

	h.logger.Info("Set enrichment mode completed",
		"mode", updatedMode,
		"source", source,
	)
}
