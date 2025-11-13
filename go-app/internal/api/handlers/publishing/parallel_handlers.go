package publishing

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// ParallelPublishHandlers provides HTTP handlers for parallel publishing (TN-058)
type ParallelPublishHandlers struct {
	publisher        publishing.ParallelPublisher
	discoveryManager publishing.TargetDiscoveryManager
	logger           *slog.Logger
}

// NewParallelPublishHandlers creates new parallel publish handlers
func NewParallelPublishHandlers(
	publisher publishing.ParallelPublisher,
	discoveryManager publishing.TargetDiscoveryManager,
	logger *slog.Logger,
) *ParallelPublishHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &ParallelPublishHandlers{
		publisher:        publisher,
		discoveryManager: discoveryManager,
		logger:           logger,
	}
}

// PublishToTargetsRequest represents the request body for publishing to specific targets
type PublishToTargetsRequest struct {
	Alert       *core.EnrichedAlert `json:"alert" validate:"required"`
	TargetNames []string            `json:"target_names" validate:"required,min=1"`
}

// PublishRequest represents the request body for publishing to all/healthy targets
type PublishRequest struct {
	Alert *core.EnrichedAlert `json:"alert" validate:"required"`
}

// PublishResponse represents the response for publishing operations
type PublishResponse struct {
	Success          bool                             `json:"success"`
	TotalTargets     int                              `json:"total_targets"`
	SuccessCount     int                              `json:"success_count"`
	FailureCount     int                              `json:"failure_count"`
	SkippedCount     int                              `json:"skipped_count"`
	IsPartialSuccess bool                             `json:"is_partial_success"`
	Duration         string                           `json:"duration"`
	Results          []publishing.TargetPublishResult `json:"results"`
	Error            string                           `json:"error,omitempty"`
}

// PublishStatusResponse represents parallel publishing status
type PublishStatusResponse struct {
	Enabled       bool   `json:"enabled"`
	TotalTargets  int    `json:"total_targets"`
	HealthyCount  int    `json:"healthy_count"`
	UnhealthyCount int   `json:"unhealthy_count"`
	Status        string `json:"status"`
}

// PublishToSpecificTargets handles POST /api/v2/publishing/parallel
//
// @Summary Publish alert to specific targets in parallel
// @Description Publishes an enriched alert to a list of specified targets concurrently
// @Tags Parallel Publishing
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body PublishToTargetsRequest true "Publish request"
// @Success 200 {object} PublishResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/parallel [post]
func (h *ParallelPublishHandlers) PublishToSpecificTargets(w http.ResponseWriter, r *http.Request) {
	var req PublishToTargetsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := apierrors.ValidationError("Invalid request body").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Validate using validator
	if err := middleware.ValidateStruct(req); err != nil {
		apiErr := apierrors.ValidationError("Validation failed").
			WithDetails(middleware.FormatValidationErrors(err)).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Resolve target names to targets
	targets := make([]*core.PublishingTarget, 0, len(req.TargetNames))
	for _, name := range req.TargetNames {
		target, err := h.discoveryManager.GetTarget(name)
		if err != nil {
			apiErr := apierrors.NotFoundError("Target: " + name).
				WithRequestID(middleware.GetRequestID(r.Context()))
			apierrors.WriteError(w, apiErr)
			return
		}
		targets = append(targets, target)
	}

	start := time.Now()
	result, err := h.publisher.PublishToMultiple(r.Context(), req.Alert, targets)
	duration := time.Since(start)

	if err != nil {
		h.logger.Error("Parallel publish to targets failed", "error", err, "duration", duration)
		apiErr := apierrors.InternalError("Publish failed: " + err.Error()).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	response := h.buildPublishResponseFromResult(result, duration)
	h.sendJSON(w, http.StatusOK, response)
}

// PublishToAllTargets handles POST /api/v2/publishing/parallel/all
//
// @Summary Publish alert to all targets in parallel
// @Description Publishes an enriched alert to all configured targets concurrently
// @Tags Parallel Publishing
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body PublishRequest true "Publish request"
// @Success 200 {object} PublishResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/parallel/all [post]
func (h *ParallelPublishHandlers) PublishToAllTargets(w http.ResponseWriter, r *http.Request) {
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := apierrors.ValidationError("Invalid request body").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Validate using validator
	if err := middleware.ValidateStruct(req); err != nil {
		apiErr := apierrors.ValidationError("Validation failed").
			WithDetails(middleware.FormatValidationErrors(err)).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	start := time.Now()
	result, err := h.publisher.PublishToAll(r.Context(), req.Alert)
	duration := time.Since(start)

	if err != nil {
		h.logger.Error("Parallel publish to all targets failed", "error", err, "duration", duration)
		apiErr := apierrors.InternalError("Publish failed: " + err.Error()).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	response := h.buildPublishResponseFromResult(result, duration)
	h.sendJSON(w, http.StatusOK, response)
}

// PublishToHealthyTargets handles POST /api/v2/publishing/parallel/healthy
//
// @Summary Publish alert to healthy targets in parallel
// @Description Publishes an enriched alert to all healthy targets concurrently
// @Tags Parallel Publishing
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body PublishRequest true "Publish request"
// @Success 200 {object} PublishResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/parallel/healthy [post]
func (h *ParallelPublishHandlers) PublishToHealthyTargets(w http.ResponseWriter, r *http.Request) {
	var req PublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apiErr := apierrors.ValidationError("Invalid request body").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Validate using validator
	if err := middleware.ValidateStruct(req); err != nil {
		apiErr := apierrors.ValidationError("Validation failed").
			WithDetails(middleware.FormatValidationErrors(err)).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	start := time.Now()
	result, err := h.publisher.PublishToHealthy(r.Context(), req.Alert)
	duration := time.Since(start)

	if err != nil {
		h.logger.Error("Parallel publish to healthy targets failed", "error", err, "duration", duration)
		apiErr := apierrors.InternalError("Publish failed: " + err.Error()).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	response := h.buildPublishResponseFromResult(result, duration)
	h.sendJSON(w, http.StatusOK, response)
}

// GetParallelPublishingStatus handles GET /api/v2/publishing/parallel/status
//
// @Summary Get parallel publishing status
// @Description Returns the current status of parallel publishing system
// @Tags Parallel Publishing
// @Produce json
// @Success 200 {object} PublishStatusResponse
// @Router /publishing/parallel/status [get]
func (h *ParallelPublishHandlers) GetParallelPublishingStatus(w http.ResponseWriter, r *http.Request) {
	// TODO: Add GetStats method to ParallelPublisher interface
	// For now, return basic status
	response := PublishStatusResponse{
		Enabled:        true,
		TotalTargets:   0,
		HealthyCount:   0,
		UnhealthyCount: 0,
		Status:         "unknown",
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ===== Helper Methods =====

func (h *ParallelPublishHandlers) buildPublishResponseFromResult(result *publishing.ParallelPublishResult, duration time.Duration) PublishResponse {
	isPartialSuccess := result.SuccessCount > 0 && result.FailureCount > 0
	overallSuccess := result.FailureCount == 0 && result.TotalTargets > 0

	return PublishResponse{
		Success:          overallSuccess,
		TotalTargets:     result.TotalTargets,
		SuccessCount:     result.SuccessCount,
		FailureCount:     result.FailureCount,
		SkippedCount:     result.SkippedCount,
		IsPartialSuccess: isPartialSuccess,
		Duration:         duration.String(),
		Results:          result.Results,
	}
}

func (h *ParallelPublishHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}
