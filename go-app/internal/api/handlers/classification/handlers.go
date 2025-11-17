package classification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// ClassificationHandlers provides HTTP handlers for classification operations
type ClassificationHandlers struct {
	classifier          core.AlertClassifier
	classificationService services.ClassificationService // Optional: for GetStats()
	logger              *slog.Logger
	statsAggregator     *StatsAggregator
	statsCache          *StatsCache // Optional: for performance optimization
}

// NewClassificationHandlers creates new classification handlers
func NewClassificationHandlers(classifier core.AlertClassifier, logger *slog.Logger) *ClassificationHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	handlers := &ClassificationHandlers{
		classifier: classifier,
		logger:     logger,
		statsCache: NewStatsCache(5 * time.Second), // Default 5s cache TTL
	}

	// Try to get ClassificationService from classifier (type assertion)
	if svc, ok := classifier.(services.ClassificationService); ok {
		handlers.classificationService = svc
		handlers.statsAggregator = NewStatsAggregator(svc, logger)
	}

	return handlers
}

// NewClassificationHandlersWithService creates new classification handlers with explicit ClassificationService
// This is the preferred method when ClassificationService is available
func NewClassificationHandlersWithService(
	classifier core.AlertClassifier,
	classificationService services.ClassificationService,
	logger *slog.Logger,
) *ClassificationHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &ClassificationHandlers{
		classifier:          classifier,
		classificationService: classificationService,
		logger:              logger,
		statsAggregator:     NewStatsAggregator(classificationService, logger),
		statsCache:          NewStatsCache(5 * time.Second), // Default 5s cache TTL
	}
}

// ClassifyRequest represents classification request
type ClassifyRequest struct {
	Alert *core.Alert `json:"alert" validate:"required"`
	Force bool        `json:"force,omitempty"` // Force new classification (bypass cache)
}

// ClassifyResponse represents classification response
type ClassifyResponse struct {
	Result         *core.ClassificationResult `json:"result"`
	ProcessingTime string                     `json:"processing_time"` // e.g., "50ms", "1.2s"
	Cached         bool                       `json:"cached"`         // Was result from cache?
	Model          string                     `json:"model,omitempty"` // LLM model used (if available)
	Timestamp      time.Time                  `json:"timestamp"`
}

// StatsResponse represents classification statistics
type StatsResponse struct {
	// Базовые метрики
	TotalClassified   int64                    `json:"total_classified"`
	TotalRequests     int64                    `json:"total_requests"`
	ClassificationRate float64                 `json:"classification_rate"`
	AvgConfidence     float64                 `json:"avg_confidence"`
	AvgProcessing     float64                 `json:"avg_processing_ms"`

	// Статистика по severity
	BySeverity map[string]SeverityStats `json:"by_severity"`

	// Cache статистика
	CacheStats CacheStats `json:"cache_stats"`

	// LLM статистика
	LLMStats LLMStats `json:"llm_stats"`

	// Fallback статистика
	FallbackStats FallbackStats `json:"fallback_stats"`

	// Error статистика
	ErrorStats ErrorStats `json:"error_stats"`

	// Метаданные
	LastClassified *time.Time `json:"last_classified,omitempty"`
	Timestamp      time.Time  `json:"timestamp"`
}

// SeverityStats represents statistics for a specific severity level
type SeverityStats struct {
	Count         int64   `json:"count"`
	AvgConfidence float64 `json:"avg_confidence"`
	Percentage    float64 `json:"percentage,omitempty"`
}

// CacheStats represents cache statistics
type CacheStats struct {
	HitRate float64 `json:"hit_rate"`
	L1Hits  int64   `json:"l1_cache_hits"`
	L2Hits  int64   `json:"l2_cache_hits"`
	Misses  int64   `json:"cache_misses"`
}

// LLMStats represents LLM usage statistics
type LLMStats struct {
	Requests     int64   `json:"requests"`
	SuccessRate  float64 `json:"success_rate"`
	Failures     int64   `json:"failures"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	UsageRate    float64 `json:"usage_rate"`
}

// FallbackStats represents fallback classification statistics
type FallbackStats struct {
	Used         int64   `json:"used"`
	Rate         float64 `json:"rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
}

// ErrorStats represents error statistics
type ErrorStats struct {
	Total        int64      `json:"total"`
	Rate         float64    `json:"rate"`
	LastError    string     `json:"last_error,omitempty"`
	LastErrorTime *time.Time `json:"last_error_time,omitempty"`
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
// @Description Classifies an alert and returns severity, confidence, and recommendations.
// Supports force flag to bypass cache and force new classification.
// @Tags Classification
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request body ClassifyRequest true "Classification request"
// @Success 200 {object} ClassifyResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 429 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Failure 503 {object} apierrors.ErrorResponse
// @Router /classification/classify [post]
func (h *ClassificationHandlers) ClassifyAlert(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	startTime := time.Now()

	// Parse request
	var req ClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Warn("Invalid request body",
			"request_id", requestID,
			"error", err)
		apiErr := apierrors.ValidationError("Invalid request body: " + err.Error()).
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Validate alert is present
	if req.Alert == nil {
		h.logger.Warn("Alert is required",
			"request_id", requestID)
		apiErr := apierrors.ValidationError("Alert is required").
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Validate alert fields
	if err := h.validateAlert(req.Alert); err != nil {
		h.logger.Warn("Alert validation failed",
			"request_id", requestID,
			"fingerprint", req.Alert.Fingerprint,
			"error", err)
		apiErr := apierrors.ValidationError("Invalid alert: " + err.Error()).
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Handle force flag: invalidate cache if force=true
	cached := false
	if req.Force && h.classificationService != nil {
		if err := h.classificationService.InvalidateCache(ctx, req.Alert.Fingerprint); err != nil {
			h.logger.Warn("Failed to invalidate cache",
				"request_id", requestID,
				"fingerprint", req.Alert.Fingerprint,
				"error", err)
			// Continue anyway - not critical
		} else {
			h.logger.Debug("Cache invalidated due to force flag",
				"request_id", requestID,
				"fingerprint", req.Alert.Fingerprint)
		}
	}

	// Check cache first (if force=false and ClassificationService available)
	var result *core.ClassificationResult
	var err error
	if !req.Force && h.classificationService != nil {
		if cachedResult, cacheErr := h.classificationService.GetCachedClassification(ctx, req.Alert.Fingerprint); cacheErr == nil && cachedResult != nil {
			result = cachedResult
			cached = true
			h.logger.Debug("Classification retrieved from cache",
				"request_id", requestID,
				"fingerprint", req.Alert.Fingerprint,
				"severity", result.Severity,
				"confidence", result.Confidence)
		}
	}

	// If not cached, classify
	if result == nil {
		result, err = h.classifier.Classify(ctx, req.Alert)
		if err != nil {
			duration := time.Since(startTime)
			h.logger.Error("Classification failed",
				"request_id", requestID,
				"fingerprint", req.Alert.Fingerprint,
				"error", err,
				"duration_ms", duration.Milliseconds())

			// Determine error type and status code
			var apiErr *apierrors.APIError
			if isTimeoutError(err) {
				apiErr = apierrors.ClassificationTimeoutError().
					WithRequestID(requestID)
			} else if isServiceUnavailable(err) {
				apiErr = apierrors.ServiceUnavailableError("LLM service").
					WithRequestID(requestID)
			} else {
				apiErr = apierrors.InternalError("Classification failed: " + err.Error()).
					WithRequestID(requestID)
			}
			apierrors.WriteError(w, apiErr)
			return
		}

		h.logger.Info("Alert classified successfully",
			"request_id", requestID,
			"fingerprint", req.Alert.Fingerprint,
			"severity", result.Severity,
			"confidence", result.Confidence,
			"cached", cached,
			"force", req.Force)
	}

	duration := time.Since(startTime)

	// Format response
	response := ClassifyResponse{
		Result:         result,
		ProcessingTime: formatDuration(duration),
		Cached:         cached,
		Timestamp:      time.Now(),
	}

	// Try to extract model from metadata (if available)
	if result.Metadata != nil {
		if model, ok := result.Metadata["model"].(string); ok {
			response.Model = model
		}
	}

	h.logger.Debug("Classification response prepared",
		"request_id", requestID,
		"fingerprint", req.Alert.Fingerprint,
		"severity", result.Severity,
		"cached", cached,
		"duration_ms", duration.Milliseconds())

	h.sendJSON(w, http.StatusOK, response)
}

// validateAlert validates alert structure and fields
func (h *ClassificationHandlers) validateAlert(alert *core.Alert) error {
	if alert.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}
	if alert.AlertName == "" {
		return fmt.Errorf("alert_name is required")
	}
	if alert.Status != core.StatusFiring && alert.Status != core.StatusResolved {
		return fmt.Errorf("status must be 'firing' or 'resolved'")
	}
	if alert.StartsAt.IsZero() {
		return fmt.Errorf("starts_at is required")
	}
	// Validate generator_url if present
	if alert.GeneratorURL != nil && *alert.GeneratorURL != "" {
		// Basic URL validation (more comprehensive validation can be added)
		if !strings.HasPrefix(*alert.GeneratorURL, "http://") && !strings.HasPrefix(*alert.GeneratorURL, "https://") {
			return fmt.Errorf("generator_url must be a valid HTTP/HTTPS URL")
		}
	}
	return nil
}

// isTimeoutError checks if error is a timeout error
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "deadline exceeded") ||
		strings.Contains(err.Error(), "context deadline exceeded")
}

// isServiceUnavailable checks if error indicates service unavailable
func isServiceUnavailable(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "circuit breaker") ||
		strings.Contains(errStr, "service unavailable") ||
		strings.Contains(errStr, "unavailable")
}

// formatDuration formats duration as human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dµs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Nanoseconds())/1e6)
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}

// GetClassificationStats handles GET /api/v2/classification/stats
//
// @Summary Get classification statistics
// @Description Returns aggregated statistics about classification operations including cache hit rate, LLM usage, fallback statistics, and error rates
// @Tags Classification
// @Produce json
// @Success 200 {object} StatsResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /classification/stats [get]
func (h *ClassificationHandlers) GetClassificationStats(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := middleware.GetRequestID(r.Context())

	// Check if ClassificationService is available
	if h.classificationService == nil {
		h.logger.Warn("ClassificationService not available, returning empty stats",
			"request_id", requestID)

		// Return empty stats instead of error (graceful degradation)
		response := &StatsResponse{
			TotalClassified:   0,
			TotalRequests:     0,
			ClassificationRate: 0.0,
			AvgConfidence:     0.0,
			AvgProcessing:     0.0,
			BySeverity:        make(map[string]SeverityStats),
			CacheStats:        CacheStats{},
			LLMStats:          LLMStats{},
			FallbackStats:     FallbackStats{},
			ErrorStats:        ErrorStats{},
			Timestamp:         time.Now(),
		}

		// Initialize severity stats with zeros
		severities := []string{"critical", "warning", "info", "noise"}
		for _, severity := range severities {
			response.BySeverity[severity] = SeverityStats{
				Count:         0,
				AvgConfidence: 0.0,
				Percentage:    0.0,
			}
		}

		h.sendJSON(w, http.StatusOK, response)
		return
	}

	// Check cache first (performance optimization)
	if h.statsCache != nil {
		if cached, hit := h.statsCache.Get(); hit {
			duration := time.Since(startTime)
			h.logger.Info("Classification stats retrieved from cache",
				"request_id", requestID,
				"total_classified", cached.TotalClassified,
				"cache_hit_rate", cached.CacheStats.HitRate,
				"duration_ms", duration.Milliseconds())
			h.sendJSON(w, http.StatusOK, cached)
			return
		}
	}

	// Aggregate statistics
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	response, err := h.statsAggregator.AggregateStats(ctx)
	if err != nil {
		h.logger.Error("Failed to aggregate classification statistics",
			"request_id", requestID,
			"error", err,
			"duration_ms", time.Since(startTime).Milliseconds())

		apiErr := apierrors.InternalError("Failed to retrieve classification statistics: " + err.Error()).
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Store in cache for future requests
	if h.statsCache != nil {
		h.statsCache.Set(response)
	}

	duration := time.Since(startTime)
	h.logger.Info("Classification stats retrieved successfully",
		"request_id", requestID,
		"total_classified", response.TotalClassified,
		"cache_hit_rate", response.CacheStats.HitRate,
		"duration_ms", duration.Milliseconds())

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
