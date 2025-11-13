package classification

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ClassificationHandlers provides HTTP handlers for classification operations
type ClassificationHandlers struct {
	classifier core.AlertClassifier
	logger     *slog.Logger
}

// NewClassificationHandlers creates new classification handlers
func NewClassificationHandlers(classifier core.AlertClassifier, logger *slog.Logger) *ClassificationHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &ClassificationHandlers{
		classifier: classifier,
		logger:     logger,
	}
}

// ClassifyRequest represents classification request
type ClassifyRequest struct {
	Alert *core.Alert `json:"alert" validate:"required"`
}

// ClassifyResponse represents classification response
type ClassifyResponse struct {
	Result         *core.ClassificationResult `json:"result"`
	ProcessingTime string                     `json:"processing_time"`
}

// StatsResponse represents classification statistics
type StatsResponse struct {
	TotalClassified int64              `json:"total_classified"`
	BySeverity      map[string]int64   `json:"by_severity"`
	AvgConfidence   float64            `json:"avg_confidence"`
	AvgProcessing   float64            `json:"avg_processing_ms"`
	LastClassified  *time.Time         `json:"last_classified,omitempty"`
}

// ModelsResponse represents available classification models
type ModelsResponse struct {
	Models []ModelInfo `json:"models"`
	Active string      `json:"active"`
}

// ModelInfo represents classification model information
type ModelInfo struct {
	Name        string  `json:"name"`
	Version     string  `json:"version"`
	Accuracy    float64 `json:"accuracy"`
	Description string  `json:"description"`
}

// ClassifyAlert handles POST /api/v2/classification/classify
//
// @Summary Classify an alert
// @Description Classifies an alert and returns severity, confidence, and recommendations
// @Tags Classification
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ClassifyRequest true "Classification request"
// @Success 200 {object} ClassifyResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /classification/classify [post]
func (h *ClassificationHandlers) ClassifyAlert(w http.ResponseWriter, r *http.Request) {
	var req ClassifyRequest
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

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	start := time.Now()
	result, err := h.classifier.Classify(ctx, req.Alert)
	duration := time.Since(start)

	if err != nil {
		h.logger.Error("Classification failed", "error", err, "duration", duration)
		apiErr := apierrors.InternalError("Classification failed: " + err.Error()).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	response := ClassifyResponse{
		Result:         result,
		ProcessingTime: duration.String(),
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetClassificationStats handles GET /api/v2/classification/stats
//
// @Summary Get classification statistics
// @Description Returns aggregated statistics about classification operations
// @Tags Classification
// @Produce json
// @Success 200 {object} StatsResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /classification/stats [get]
func (h *ClassificationHandlers) GetClassificationStats(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual stats collection
	// For now, return mock data
	response := StatsResponse{
		TotalClassified: 0,
		BySeverity: map[string]int64{
			"critical": 0,
			"warning":  0,
			"info":     0,
			"noise":    0,
		},
		AvgConfidence: 0.0,
		AvgProcessing: 0.0,
		LastClassified: nil,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ListClassificationModels handles GET /api/v2/classification/models
//
// @Summary List available classification models
// @Description Returns information about available classification models
// @Tags Classification
// @Produce json
// @Success 200 {object} ModelsResponse
// @Router /classification/models [get]
func (h *ClassificationHandlers) ListClassificationModels(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement actual model discovery
	// For now, return mock data
	response := ModelsResponse{
		Active: "llm-classifier-v1",
		Models: []ModelInfo{
			{
				Name:        "llm-classifier-v1",
				Version:     "1.0.0",
				Accuracy:    0.95,
				Description: "LLM-based alert classifier with GPT-4",
			},
			{
				Name:        "rule-based-classifier",
				Version:     "1.0.0",
				Accuracy:    0.85,
				Description: "Rule-based classifier for known patterns",
			},
		},
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ===== Helper Methods =====

func (h *ClassificationHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}
