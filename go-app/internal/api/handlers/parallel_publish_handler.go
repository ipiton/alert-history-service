package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// ParallelPublishHandler handles HTTP requests for parallel publishing operations.
//
// Endpoints:
//   - POST /api/v1/publish/parallel - Publish to specific targets
//   - POST /api/v1/publish/parallel/all - Publish to all targets
//   - POST /api/v1/publish/parallel/healthy - Publish to healthy targets only
//   - GET /api/v1/publish/parallel/status - Get parallel publishing status
type ParallelPublishHandler struct {
	publisher publishing.ParallelPublisher
	logger    *slog.Logger
}

// NewParallelPublishHandler creates a new parallel publish handler.
func NewParallelPublishHandler(publisher publishing.ParallelPublisher, logger *slog.Logger) *ParallelPublishHandler {
	return &ParallelPublishHandler{
		publisher: publisher,
		logger:    logger,
	}
}

// PublishToTargetsRequest represents the request body for publishing to specific targets.
type PublishToTargetsRequest struct {
	Alert       *core.EnrichedAlert `json:"alert"`
	TargetNames []string            `json:"target_names"`
}

// PublishRequest represents the request body for publishing to all/healthy targets.
type PublishRequest struct {
	Alert *core.EnrichedAlert `json:"alert"`
}

// PublishResponse represents the response for publishing operations.
type PublishResponse struct {
	Success          bool                                 `json:"success"`
	TotalTargets     int                                  `json:"total_targets"`
	SuccessCount     int                                  `json:"success_count"`
	FailureCount     int                                  `json:"failure_count"`
	SkippedCount     int                                  `json:"skipped_count"`
	IsPartialSuccess bool                                 `json:"is_partial_success"`
	Duration         string                               `json:"duration"` // e.g., "123ms"
	Results          []publishing.TargetPublishResult     `json:"results"`
	Error            string                               `json:"error,omitempty"`
}

// PublishToTargets handles POST /api/v1/publish/parallel
//
// Request Body:
//
//	{
//	  "alert": {...},
//	  "target_names": ["rootly-prod", "pagerduty-oncall", "slack-alerts"]
//	}
//
// Response:
//
//	{
//	  "success": true,
//	  "total_targets": 3,
//	  "success_count": 3,
//	  "failure_count": 0,
//	  "skipped_count": 0,
//	  "is_partial_success": false,
//	  "duration": "123ms",
//	  "results": [...]
//	}
func (h *ParallelPublishHandler) PublishToTargets(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req PublishToTargetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request", "error", err)
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Alert == nil {
		h.writeErrorResponse(w, "Alert is required", http.StatusBadRequest)
		return
	}
	if len(req.TargetNames) == 0 {
		h.writeErrorResponse(w, "Target names are required", http.StatusBadRequest)
		return
	}

	// TODO: Resolve target names to PublishingTarget objects
	// For now, this is a placeholder. In a real implementation, you would:
	// 1. Call discoveryManager.GetTarget(name) for each target name
	// 2. Build targets slice
	// 3. Pass to publisher.PublishToMultiple()
	//
	// Example:
	// targets := make([]*core.PublishingTarget, 0, len(req.TargetNames))
	// for _, name := range req.TargetNames {
	//     target, err := h.discoveryManager.GetTarget(name)
	//     if err != nil {
	//         h.logger.Warn("Target not found", "name", name, "error", err)
	//         continue
	//     }
	//     targets = append(targets, target)
	// }

	h.writeErrorResponse(w, "Not yet implemented - target name resolution pending", http.StatusNotImplemented)
}

// PublishToAll handles POST /api/v1/publish/parallel/all
//
// Request Body:
//
//	{
//	  "alert": {...}
//	}
//
// Response: Same as PublishToTargets
func (h *ParallelPublishHandler) PublishToAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request", "error", err)
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Alert == nil {
		h.writeErrorResponse(w, "Alert is required", http.StatusBadRequest)
		return
	}

	// Publish to all targets
	ctx := r.Context()
	result, err := h.publisher.PublishToAll(ctx, req.Alert)
	if err != nil {
		h.logger.Error("Failed to publish to all targets", "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write success response
	h.writePublishResponse(w, result, nil)
}

// PublishToHealthy handles POST /api/v1/publish/parallel/healthy
//
// Request Body:
//
//	{
//	  "alert": {...}
//	}
//
// Response: Same as PublishToTargets
func (h *ParallelPublishHandler) PublishToHealthy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("Failed to parse request", "error", err)
		h.writeErrorResponse(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.Alert == nil {
		h.writeErrorResponse(w, "Alert is required", http.StatusBadRequest)
		return
	}

	// Publish to healthy targets
	ctx := r.Context()
	result, err := h.publisher.PublishToHealthy(ctx, req.Alert)
	if err != nil {
		h.logger.Error("Failed to publish to healthy targets", "error", err)
		h.writeErrorResponse(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write success response
	h.writePublishResponse(w, result, nil)
}

// GetStatus handles GET /api/v1/publish/parallel/status
//
// Response:
//
//	{
//	  "enabled": true,
//	  "max_concurrent": 50,
//	  "timeout": "30s",
//	  "health_checks_enabled": true,
//	  "health_strategy": "skip_unhealthy"
//	}
func (h *ParallelPublishHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement status endpoint
	// This would return:
	// - Configuration (MaxConcurrent, Timeout, etc.)
	// - Current goroutine count
	// - Publishing statistics (total, success, failure rates)
	// - Health of underlying services

	response := map[string]interface{}{
		"enabled":               true,
		"max_concurrent":        50,
		"timeout":               "30s",
		"health_checks_enabled": true,
		"health_strategy":       "skip_unhealthy",
		"status":                "operational",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// writePublishResponse writes a successful publish response.
func (h *ParallelPublishHandler) writePublishResponse(w http.ResponseWriter, result *publishing.ParallelPublishResult, err error) {
	response := &PublishResponse{
		Success:          result.Success(),
		TotalTargets:     result.TotalTargets,
		SuccessCount:     result.SuccessCount,
		FailureCount:     result.FailureCount,
		SkippedCount:     result.SkippedCount,
		IsPartialSuccess: result.IsPartialSuccess,
		Duration:         formatDuration(result.Duration),
		Results:          result.Results,
	}

	if err != nil {
		response.Error = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	} else if result.AllFailed() {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(response)
}

// writeErrorResponse writes an error response.
func (h *ParallelPublishHandler) writeErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	response := &PublishResponse{
		Success: false,
		Error:   message,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// formatDuration formats a duration to a human-readable string.
func formatDuration(d time.Duration) string {
	if d < time.Microsecond {
		return d.String()
	}
	if d < time.Millisecond {
		return d.Round(time.Microsecond).String()
	}
	if d < time.Second {
		return d.Round(time.Millisecond).String()
	}
	return d.Round(10 * time.Millisecond).String()
}
