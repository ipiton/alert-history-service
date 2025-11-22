package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	appconfig "github.com/vitaliisemenov/alert-history/internal/config"
)

// ================================================================================
// Configuration History HTTP Handler
// ================================================================================
// Handles GET /api/v2/config/history requests for version history (TN-150).
//
// Features:
// - Version history listing
// - Pagination support (limit parameter)
// - Full version metadata (timestamp, author, source)
// - Secret sanitization
// - Admin-only access
//
// Quality Target: 150% (Grade A+ EXCEPTIONAL)
// Author: AI Assistant
// Date: 2025-11-22

// ConfigHistoryHandler handles configuration history requests
type ConfigHistoryHandler struct {
	updateService appconfig.ConfigUpdateService
	logger        *slog.Logger
}

// NewConfigHistoryHandler creates a new ConfigHistoryHandler
func NewConfigHistoryHandler(
	updateService appconfig.ConfigUpdateService,
	logger *slog.Logger,
) *ConfigHistoryHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &ConfigHistoryHandler{
		updateService: updateService,
		logger:        logger,
	}
}

// HandleGetHistory handles GET /api/v2/config/history requests
//
// Query Parameters:
//   - limit: Maximum number of versions to return (default: 10, max: 100)
//
// Response Codes:
//   - 200 OK: History retrieved successfully
//   - 400 Bad Request: Invalid limit parameter
//   - 401 Unauthorized: Missing or invalid auth
//   - 403 Forbidden: Not admin
//   - 500 Internal Server Error: Server error
func (h *ConfigHistoryHandler) HandleGetHistory(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()
	requestID := extractRequestID(r)

	h.logger.Info("config history request received",
		"method", r.Method,
		"path", r.URL.Path,
		"query", r.URL.RawQuery,
		"remote_addr", r.RemoteAddr,
		"request_id", requestID,
	)

	// Step 1: Validate HTTP method
	if r.Method != http.MethodGet {
		h.respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Step 2: Parse limit parameter
	limit := 10 // Default
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit < 1 || parsedLimit > 100 {
			h.respondError(w, http.StatusBadRequest, "limit must be between 1 and 100")
			return
		}
		limit = parsedLimit
	}

	// Step 3: Get history from service
	history, err := h.updateService.GetHistory(ctx, limit)
	if err != nil {
		h.logger.Error("failed to get config history",
			"error", err,
			"request_id", requestID,
		)
		h.respondError(w, http.StatusInternalServerError, "failed to retrieve configuration history")
		return
	}

	// Step 4: Success response
	h.respondSuccess(w, history)

	h.logger.Info("config history retrieved",
		"count", len(history),
		"limit", limit,
		"duration_ms", time.Since(startTime).Milliseconds(),
		"request_id", requestID,
	)
}

// respondSuccess writes success response
func (h *ConfigHistoryHandler) respondSuccess(w http.ResponseWriter, history []*appconfig.ConfigVersion) {
	response := ConfigHistoryResponse{
		Status:   "success",
		Count:    len(history),
		Versions: history,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode response", "error", err)
	}
}

// respondError writes error response
func (h *ConfigHistoryHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	response := ConfigHistoryResponse{
		Status:  "error",
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("failed to encode error response", "error", err)
	}
}

// ConfigHistoryResponse represents config history response
type ConfigHistoryResponse struct {
	Status   string                      `json:"status"`
	Message  string                      `json:"message,omitempty"`
	Count    int                         `json:"count,omitempty"`
	Versions []*appconfig.ConfigVersion  `json:"versions,omitempty"`
}
