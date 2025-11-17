package publishing

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
	"github.com/vitaliisemenov/alert-history/internal/core"
	infrapub "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// PublishingHandlers provides HTTP handlers for publishing management (API v2)
type PublishingHandlers struct {
	discoveryManager publishing.TargetDiscoveryManager
	refreshManager   publishing.RefreshManager
	queue            *infrapub.PublishingQueue
	coordinator      *infrapub.PublishingCoordinator
	logger           *slog.Logger
}

// NewPublishingHandlers creates new publishing HTTP handlers
func NewPublishingHandlers(
	discoveryManager publishing.TargetDiscoveryManager,
	refreshManager publishing.RefreshManager,
	queue *infrapub.PublishingQueue,
	coordinator *infrapub.PublishingCoordinator,
	logger *slog.Logger,
) *PublishingHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &PublishingHandlers{
		discoveryManager: discoveryManager,
		refreshManager:   refreshManager,
		queue:            queue,
		coordinator:      coordinator,
		logger:           logger,
	}
}

// TargetResponse represents a publishing target in API responses
type TargetResponse struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	URL     string            `json:"url"`
	Enabled bool              `json:"enabled"`
	Format  string            `json:"format"`
	Headers map[string]string `json:"headers,omitempty"`
}

// StatsResponse represents publishing statistics
type StatsResponse struct {
	TotalTargets     int            `json:"total_targets"`
	EnabledTargets   int            `json:"enabled_targets"`
	TargetsByType    map[string]int `json:"targets_by_type"`
	QueueSize        int            `json:"queue_size"`
	QueueCapacity    int            `json:"queue_capacity"`
	QueueUtilization float64        `json:"queue_utilization_percent"`
}

// QueueStatusResponse represents queue status
type QueueStatusResponse struct {
	Size         int     `json:"size"`
	Capacity     int     `json:"capacity"`
	Utilization  float64 `json:"utilization_percent"`
	WorkersCount int     `json:"workers_count"`
}

// PublishingModeResponse represents current publishing mode
type PublishingModeResponse struct {
	Mode              string `json:"mode"`
	TargetsAvailable  bool   `json:"targets_available"`
	EnabledTargets    int    `json:"enabled_targets"`
	MetricsOnlyActive bool   `json:"metrics_only_active"`
}

// TestTargetRequest represents test target request configuration.
//
// This struct is used to configure test alert parameters when testing
// publishing target connectivity via POST /api/v2/publishing/targets/{name}/test.
//
// Example:
//
//	req := TestTargetRequest{
//	    AlertName:      "CustomTestAlert",
//	    TimeoutSeconds: 60,
//	    TestAlert: &CustomTestAlert{
//	        Labels: map[string]string{"severity": "warning"},
//	        Status: "firing",
//	    },
//	}
type TestTargetRequest struct {
	// AlertName is an optional custom alert name.
	// If not provided, defaults to "TestAlert".
	AlertName string `json:"alert_name"`

	// TestAlert is an optional custom test alert payload.
	// If not provided, a default test alert will be created.
	TestAlert *CustomTestAlert `json:"test_alert,omitempty"`

	// TimeoutSeconds is the timeout for the test in seconds.
	// Must be between 1 and 300 (default: 30).
	TimeoutSeconds int `json:"timeout_seconds"`
}

// CustomTestAlert represents a custom test alert payload for target testing.
//
// This allows testing with custom labels, annotations, and status to simulate
// different alert scenarios. A "test: true" label is automatically added.
//
// Example:
//
//	customAlert := &CustomTestAlert{
//	    Fingerprint: "custom-fp-123",
//	    Labels: map[string]string{
//	        "alertname": "CustomAlert",
//	        "severity":  "critical",
//	    },
//	    Annotations: map[string]string{
//	        "summary": "Custom test alert",
//	    },
//	    Status: "firing",
//	}
type CustomTestAlert struct {
	// Fingerprint is an optional custom fingerprint.
	// If not provided, a fingerprint will be auto-generated.
	Fingerprint string `json:"fingerprint,omitempty"`

	// Labels are alert labels. A "test: true" label is automatically added.
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are alert annotations.
	Annotations map[string]string `json:"annotations,omitempty"`

	// Status is the alert status: "firing" or "resolved" (default: "firing").
	Status string `json:"status,omitempty"` // "firing" | "resolved"
}

// TestTargetResponse represents the result of a target connectivity test.
//
// This response is always returned with HTTP 200 OK. The success field
// indicates whether the test succeeded. Error details are provided in
// the error field if the test failed.
//
// Example:
//
//	{
//	    "success": true,
//	    "message": "Test alert sent",
//	    "target_name": "rootly-prod",
//	    "status_code": 200,
//	    "response_time_ms": 150,
//	    "test_timestamp": "2025-11-17T19:00:00Z"
//	}
type TestTargetResponse struct {
	// Success indicates whether the test succeeded.
	Success bool `json:"success"`

	// Message is a human-readable message describing the result.
	Message string `json:"message"`

	// TargetName is the name of the target that was tested.
	TargetName string `json:"target_name"`

	// StatusCode is the HTTP status code from the target API (if available).
	StatusCode *int `json:"status_code,omitempty"`

	// ResponseTimeMs is the total response time in milliseconds.
	ResponseTimeMs int `json:"response_time_ms"`

	// Error contains error details if the test failed.
	Error string `json:"error,omitempty"`

	// TestTimestamp is when the test was executed.
	TestTimestamp time.Time `json:"test_timestamp"`
}

// SubmitAlertRequest represents alert submission request
type SubmitAlertRequest struct {
	Alert      *core.Alert `json:"alert" validate:"required"`
	TargetName string      `json:"target_name,omitempty"`
}

// SubmitAlertResponse represents alert submission result
type SubmitAlertResponse struct {
	Success bool     `json:"success"`
	Message string   `json:"message"`
	JobIDs  []string `json:"job_ids,omitempty"`
}

// DetailedQueueStatsResponse represents detailed queue statistics
type DetailedQueueStatsResponse struct {
	TotalSize      int     `json:"total_size"`
	HighPriority   int     `json:"high_priority"`
	MedPriority    int     `json:"med_priority"`
	LowPriority    int     `json:"low_priority"`
	Capacity       int     `json:"capacity"`
	WorkerCount    int     `json:"worker_count"`
	ActiveJobs     int     `json:"active_jobs"`
	TotalSubmitted int64   `json:"total_submitted"`
	TotalCompleted int64   `json:"total_completed"`
	TotalFailed    int64   `json:"total_failed"`
	SuccessRate    float64 `json:"success_rate_percent"`
	DLQSize        int     `json:"dlq_size"`
}

// JobStatusResponse represents job status
type JobStatusResponse struct {
	ID             string     `json:"id"`
	Fingerprint    string     `json:"fingerprint"`
	TargetName     string     `json:"target_name"`
	TargetType     string     `json:"target_type"`
	Priority       string     `json:"priority"`
	State          string     `json:"state"`
	RetryCount     int        `json:"retry_count"`
	MaxRetries     int        `json:"max_retries"`
	LastError      string     `json:"last_error,omitempty"`
	ErrorType      string     `json:"error_type,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ProcessingTime string     `json:"processing_time,omitempty"`
}

// JobListResponse represents list of jobs
type JobListResponse struct {
	Jobs       []JobStatusResponse `json:"jobs"`
	TotalCount int                 `json:"total_count"`
}

// DLQEntryResponse represents a DLQ entry in API
type DLQEntryResponse struct {
	ID           string     `json:"id"`
	JobID        string     `json:"job_id"`
	Fingerprint  string     `json:"fingerprint"`
	TargetName   string     `json:"target_name"`
	TargetType   string     `json:"target_type"`
	ErrorMessage string     `json:"error_message"`
	ErrorType    string     `json:"error_type"`
	RetryCount   int        `json:"retry_count"`
	Priority     string     `json:"priority"`
	FailedAt     time.Time  `json:"failed_at"`
	Replayed     bool       `json:"replayed"`
	ReplayedAt   *time.Time `json:"replayed_at,omitempty"`
}

// DLQListResponse represents list of DLQ entries
type DLQListResponse struct {
	Entries    []DLQEntryResponse `json:"entries"`
	TotalCount int                `json:"total_count"`
	Stats      *DLQStatsResponse  `json:"stats,omitempty"`
}

// DLQStatsResponse represents DLQ statistics
type DLQStatsResponse struct {
	TotalEntries       int            `json:"total_entries"`
	EntriesByErrorType map[string]int `json:"entries_by_error_type"`
	EntriesByTarget    map[string]int `json:"entries_by_target"`
	EntriesByPriority  map[string]int `json:"entries_by_priority"`
	ReplayedCount      int            `json:"replayed_count"`
}

// ReplayDLQResponse represents DLQ replay result
type ReplayDLQResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// PurgeDLQRequest represents DLQ purge request
type PurgeDLQRequest struct {
	OlderThanHours int `json:"older_than_hours"`
}

// PurgeDLQResponse represents DLQ purge result
type PurgeDLQResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	DeletedCount int64  `json:"deleted_count"`
}

// ListTargetsParams represents query parameters for list targets endpoint
type ListTargetsParams struct {
	Type      *string
	Enabled   *bool
	Limit     int
	Offset    int
	SortBy    string
	SortOrder string
}

// TargetListResponse represents paginated list of targets
type TargetListResponse struct {
	Data       []TargetResponse   `json:"data"`
	Pagination PaginationMetadata `json:"pagination"`
	Metadata   ResponseMetadata   `json:"metadata"`
}

// PaginationMetadata represents pagination information
type PaginationMetadata struct {
	Total   int  `json:"total"`
	Count   int  `json:"count"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// ResponseMetadata represents response metadata
type ResponseMetadata struct {
	RequestID       string `json:"request_id"`
	Timestamp       string `json:"timestamp"`
	ProcessingTimeMs int64  `json:"processing_time_ms"`
}

// ===== Target Management Handlers =====

// ListTargets handles GET /api/v2/publishing/targets
//
// @Summary List all publishing targets
// @Description Returns a paginated list of all configured publishing targets with filtering and sorting support
// @Tags Targets
// @Produce json
// @Param type query string false "Filter by target type (rootly, pagerduty, slack, webhook)"
// @Param enabled query bool false "Filter by enabled status"
// @Param limit query int false "Maximum results per page (1-1000, default: 100)"
// @Param offset query int false "Offset for pagination (>=0, default: 0)"
// @Param sort_by query string false "Sort field (name, type, enabled, default: name)"
// @Param sort_order query string false "Sort direction (asc, desc, default: asc)"
// @Success 200 {object} TargetListResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/targets [get]
func (h *PublishingHandlers) ListTargets(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := middleware.GetRequestID(r.Context())

	// Parse and validate query parameters
	params, err := h.parseListTargetsParams(r)
	if err != nil {
		apiErr := apierrors.ValidationError("Invalid query parameters").
			WithDetails(err.Error()).
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Get all targets from discovery manager
	allTargets := h.discoveryManager.ListTargets()
	if allTargets == nil {
		allTargets = []*core.PublishingTarget{}
	}

	// Apply filters
	filteredTargets := h.filterTargets(allTargets, params)

	// Sort targets
	h.sortTargets(filteredTargets, params)

	// Calculate pagination
	total := len(filteredTargets)
	paginatedTargets := h.paginateTargets(filteredTargets, params)

	// Build response
	response := TargetListResponse{
		Data: h.convertToTargetResponses(paginatedTargets),
		Pagination: PaginationMetadata{
			Total:   total,
			Count:   len(paginatedTargets),
			Limit:   params.Limit,
			Offset:  params.Offset,
			HasMore: params.Offset+len(paginatedTargets) < total,
		},
		Metadata: ResponseMetadata{
			RequestID:       requestID,
			Timestamp:       time.Now().UTC().Format(time.RFC3339),
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
		},
	}

	// Log request
	h.logger.Info("List targets request",
		"request_id", requestID,
		"type_filter", params.Type,
		"enabled_filter", params.Enabled,
		"limit", params.Limit,
		"offset", params.Offset,
		"sort_by", params.SortBy,
		"sort_order", params.SortOrder,
		"total_targets", total,
		"returned_count", len(paginatedTargets),
		"processing_time_ms", time.Since(startTime).Milliseconds(),
	)

	h.sendJSON(w, http.StatusOK, response)
}

// GetTarget handles GET /api/v2/publishing/targets/{name}
//
// @Summary Get target by name
// @Description Returns details of a specific publishing target
// @Tags Targets
// @Produce json
// @Param name path string true "Target name"
// @Success 200 {object} TargetResponse
// @Failure 404 {object} apierrors.ErrorResponse
// @Router /publishing/targets/{name} [get]
func (h *PublishingHandlers) GetTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	target, err := h.discoveryManager.GetTarget(name)
	if err != nil {
		apiErr := apierrors.NotFoundError("Target").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	response := TargetResponse{
		Name:    target.Name,
		Type:    target.Type,
		URL:     target.URL,
		Enabled: target.Enabled,
		Format:  string(target.Format),
		Headers: target.Headers,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// RefreshTargets handles POST /api/v2/publishing/targets/refresh
//
// @Summary Refresh target discovery
// @Description Manually triggers target discovery refresh
// @Tags Targets
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/targets/refresh [post]
func (h *PublishingHandlers) RefreshTargets(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Manual target refresh requested via API")

	err := h.refreshManager.RefreshNow()
	if err != nil {
		h.logger.Error("Manual refresh failed", "error", err)
		apiErr := apierrors.InternalError("Failed to refresh targets: " + err.Error()).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	targets := h.discoveryManager.ListTargets()
	targetCount := len(targets)
	enabledCount := 0
	for _, t := range targets {
		if t.Enabled {
			enabledCount++
		}
	}

	response := map[string]interface{}{
		"success":         true,
		"message":         "Targets refreshed successfully",
		"total_targets":   targetCount,
		"enabled_targets": enabledCount,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// TestTarget handles POST /api/v2/publishing/targets/{name}/test
//
// @Summary Test target connectivity
// @Description Sends a test alert to validate target configuration
// @Tags Targets
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param name path string true "Target name"
// @Param request body TestTargetRequest false "Test request"
// @Success 200 {object} TestTargetResponse
// @Failure 400 {object} apierrors.ErrorResponse
// @Failure 404 {object} apierrors.ErrorResponse
// @Router /publishing/targets/{name}/test [post]
func (h *PublishingHandlers) TestTarget(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	requestID := middleware.GetRequestID(r.Context())
	vars := mux.Vars(r)
	targetName := vars["name"]

	// Validate target name
	if targetName == "" {
		apiErr := apierrors.ValidationError("Target name is required").
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Decode request body (optional)
	var req TestTargetRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apiErr := apierrors.ValidationError("Invalid request body").
				WithRequestID(requestID).
				WithDetails(err.Error())
			apierrors.WriteError(w, apiErr)
			return
		}
	}

	// Set defaults
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 30 // Default timeout
	}
	if req.TimeoutSeconds < 1 || req.TimeoutSeconds > 300 {
		apiErr := apierrors.ValidationError("Timeout must be between 1 and 300 seconds").
			WithRequestID(requestID)
		apierrors.WriteError(w, apiErr)
		return
	}

	// Get target
	target, err := h.discoveryManager.GetTarget(targetName)
	if err != nil {
		apiErr := apierrors.NotFoundError("Target").
			WithRequestID(requestID).
			WithDetails(fmt.Sprintf("Target '%s' does not exist", targetName))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Check if target is enabled
	if !target.Enabled {
		responseTimeMs := int(time.Since(startTime).Milliseconds())
		h.sendJSON(w, http.StatusOK, TestTargetResponse{
			Success:        false,
			Message:        "Target is disabled",
			TargetName:     targetName,
			ResponseTimeMs: responseTimeMs,
			TestTimestamp:  startTime,
		})
		return
	}

	// Create test alert
	testAlert, err := h.buildTestAlert(&req, targetName)
	if err != nil {
		apiErr := apierrors.ValidationError("Failed to create test alert").
			WithRequestID(requestID).
			WithDetails(err.Error())
		apierrors.WriteError(w, apiErr)
		return
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(req.TimeoutSeconds)*time.Second)
	defer cancel()

	// Publish test alert
	publishStartTime := time.Now()
	results, err := h.coordinator.PublishToTargets(ctx, testAlert, []string{targetName})
	publishDuration := time.Since(publishStartTime)
	responseTimeMs := int(time.Since(startTime).Milliseconds())

	// Handle publishing errors
	if err != nil {
		// Check if timeout
		if ctx.Err() == context.DeadlineExceeded {
			h.sendJSON(w, http.StatusOK, TestTargetResponse{
				Success:        false,
				Message:        "Test timeout",
				TargetName:     targetName,
				ResponseTimeMs: responseTimeMs,
				Error:          fmt.Sprintf("Test timeout after %d seconds", req.TimeoutSeconds),
				TestTimestamp:  startTime,
			})
			return
		}

		// Other errors
		h.sendJSON(w, http.StatusOK, TestTargetResponse{
			Success:        false,
			Message:        "Test failed",
			TargetName:     targetName,
			ResponseTimeMs: responseTimeMs,
			Error:          err.Error(),
			TestTimestamp:  startTime,
		})
		return
	}

	// Handle empty results
	if len(results) == 0 {
		h.sendJSON(w, http.StatusOK, TestTargetResponse{
			Success:        false,
			Message:        "No results returned",
			TargetName:     targetName,
			ResponseTimeMs: responseTimeMs,
			Error:          "Publishing coordinator returned no results",
			TestTimestamp:  startTime,
		})
		return
	}

	// Extract result
	result := results[0]
	response := TestTargetResponse{
		Success:        result.Success,
		Message:        "Test alert sent",
		TargetName:     targetName,
		ResponseTimeMs: responseTimeMs,
		TestTimestamp:  startTime,
	}

	// Add error if failed
	if result.Error != nil {
		response.Error = result.Error.Error()
		// Try to extract status code from error (if available)
		// This is a best-effort approach, as PublishingResult doesn't expose StatusCode
		// TODO: Enhance PublishingResult to include StatusCode
	}

	// Log result
	h.logger.Info("Test target completed",
		"request_id", requestID,
		"target", targetName,
		"success", result.Success,
		"response_time_ms", responseTimeMs,
		"publish_duration_ms", publishDuration.Milliseconds(),
	)

	h.sendJSON(w, http.StatusOK, response)
}

// ===== Queue Management Handlers =====

// GetQueueStatus handles GET /api/v2/publishing/queue/status
func (h *PublishingHandlers) GetQueueStatus(w http.ResponseWriter, r *http.Request) {
	queueSize := h.queue.GetQueueSize()
	queueCapacity := h.queue.GetQueueCapacity()
	utilization := 0.0
	if queueCapacity > 0 {
		utilization = float64(queueSize) / float64(queueCapacity) * 100
	}

	response := QueueStatusResponse{
		Size:         queueSize,
		Capacity:     queueCapacity,
		Utilization:  utilization,
		WorkersCount: 10,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetDetailedQueueStats handles GET /api/v2/publishing/queue/stats
func (h *PublishingHandlers) GetDetailedQueueStats(w http.ResponseWriter, r *http.Request) {
	stats := h.queue.GetStats()

	successRate := 0.0
	if stats.TotalSubmitted > 0 {
		successRate = float64(stats.TotalCompleted) / float64(stats.TotalSubmitted) * 100
	}

	// DLQ stats - skip for now as dlqRepository is private
	dlqSize := 0
	// TODO: Add public method to queue for DLQ stats

	response := DetailedQueueStatsResponse{
		TotalSize:      stats.TotalSize,
		HighPriority:   stats.HighPriority,
		MedPriority:    stats.MedPriority,
		LowPriority:    stats.LowPriority,
		Capacity:       stats.Capacity,
		WorkerCount:    stats.WorkerCount,
		ActiveJobs:     stats.ActiveJobs,
		TotalSubmitted: stats.TotalSubmitted,
		TotalCompleted: stats.TotalCompleted,
		TotalFailed:    stats.TotalFailed,
		SuccessRate:    successRate,
		DLQSize:        dlqSize,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// SubmitAlert handles POST /api/v2/publishing/queue/submit
func (h *PublishingHandlers) SubmitAlert(w http.ResponseWriter, r *http.Request) {
	var req SubmitAlertRequest
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

	enrichedAlert := &core.EnrichedAlert{
		Alert: req.Alert,
		Classification: &core.ClassificationResult{
			Severity:   core.SeverityInfo,
			Confidence: 0.5,
			Reasoning:  "Submitted via API",
		},
	}

	var jobIDs []string
	if req.TargetName != "" {
		target, err := h.discoveryManager.GetTarget(req.TargetName)
		if err != nil {
			apiErr := apierrors.NotFoundError("Target").
				WithRequestID(middleware.GetRequestID(r.Context()))
			apierrors.WriteError(w, apiErr)
			return
		}

		if err := h.queue.Submit(enrichedAlert, target); err != nil {
			apiErr := apierrors.InternalError("Failed to submit job").
				WithRequestID(middleware.GetRequestID(r.Context()))
			apierrors.WriteError(w, apiErr)
			return
		}
		jobIDs = []string{enrichedAlert.Alert.Fingerprint + ":" + target.Name}
	} else {
		targets := h.discoveryManager.ListTargets()
		for _, target := range targets {
			if !target.Enabled {
				continue
			}
			if err := h.queue.Submit(enrichedAlert, target); err != nil {
				h.logger.Warn("Failed to submit to target", "target", target.Name, "error", err)
				continue
			}
			jobIDs = append(jobIDs, enrichedAlert.Alert.Fingerprint+":"+target.Name)
		}
	}

	if len(jobIDs) == 0 {
		apiErr := apierrors.InternalError("No jobs created").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	h.sendJSON(w, http.StatusAccepted, SubmitAlertResponse{
		Success: true,
		Message: fmt.Sprintf("Alert submitted to %d target(s)", len(jobIDs)),
		JobIDs:  jobIDs,
	})
}

// ListJobs handles GET /api/v2/publishing/queue/jobs
func (h *PublishingHandlers) ListJobs(w http.ResponseWriter, r *http.Request) {
	// TODO: Add public method to queue for listing jobs
	// For now, return empty list
	_ = r.URL.Query() // Suppress unused warning
	responses := make([]JobStatusResponse, 0)

	h.sendJSON(w, http.StatusOK, JobListResponse{
		Jobs:       responses,
		TotalCount: len(responses),
	})
}

// GetJob handles GET /api/v2/publishing/queue/jobs/{id}
func (h *PublishingHandlers) GetJob(w http.ResponseWriter, r *http.Request) {
	// TODO: Add public method to queue for getting job by ID
	_ = mux.Vars(r) // Suppress unused warning
	apiErr := apierrors.InternalError("Job tracking not yet implemented in API v2").
		WithRequestID(middleware.GetRequestID(r.Context()))
	apierrors.WriteError(w, apiErr)
}

// ===== DLQ Management Handlers =====

// ListDLQEntries handles GET /api/v2/publishing/dlq
func (h *PublishingHandlers) ListDLQEntries(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	filters := infrapub.DLQFilters{
		TargetName: query.Get("target"),
		ErrorType:  query.Get("error_type"),
		Priority:   query.Get("priority"),
		Limit:      100,
	}

	if replayedStr := query.Get("replayed"); replayedStr != "" {
		replayed := replayedStr == "true"
		filters.Replayed = &replayed
	}

	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			filters.Limit = parsedLimit
		}
	}

	// TODO: Add public method to queue for DLQ operations
	// For now, return empty list
	responses := make([]DLQEntryResponse, 0)

	h.sendJSON(w, http.StatusOK, DLQListResponse{
		Entries:    responses,
		TotalCount: len(responses),
	})
}

// ReplayDLQEntry handles POST /api/v2/publishing/dlq/{id}/replay
func (h *PublishingHandlers) ReplayDLQEntry(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		apiErr := apierrors.ValidationError("Invalid DLQ entry ID").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// TODO: Add public method to queue for DLQ replay
	_ = id // Suppress unused warning
	apiErr := apierrors.InternalError("DLQ replay not yet implemented in API v2").
		WithRequestID(middleware.GetRequestID(r.Context()))
	apierrors.WriteError(w, apiErr)
}

// PurgeDLQ handles DELETE /api/v2/publishing/dlq/purge
func (h *PublishingHandlers) PurgeDLQ(w http.ResponseWriter, r *http.Request) {
	var req PurgeDLQRequest
	req.OlderThanHours = 168 // Default: 7 days

	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.Debug("Failed to decode purge request, using defaults", "error", err)
		}
	}

	if req.OlderThanHours <= 0 {
		req.OlderThanHours = 168
	}

	// TODO: Add public method to queue for DLQ purge
	_ = req // Suppress unused warning
	apiErr := apierrors.InternalError("DLQ purge not yet implemented in API v2").
		WithRequestID(middleware.GetRequestID(r.Context()))
	apierrors.WriteError(w, apiErr)
}

// ===== Statistics Handlers =====

// GetStats handles GET /api/v2/publishing/stats
func (h *PublishingHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	targets := h.discoveryManager.ListTargets()
	targetsByType := make(map[string]int)
	enabledCount := 0

	for _, t := range targets {
		targetsByType[t.Type]++
		if t.Enabled {
			enabledCount++
		}
	}

	queueSize := h.queue.GetQueueSize()
	queueCapacity := h.queue.GetQueueCapacity()
	utilization := 0.0
	if queueCapacity > 0 {
		utilization = float64(queueSize) / float64(queueCapacity) * 100
	}

	response := StatsResponse{
		TotalTargets:     len(targets),
		EnabledTargets:   enabledCount,
		TargetsByType:    targetsByType,
		QueueSize:        queueSize,
		QueueCapacity:    queueCapacity,
		QueueUtilization: utilization,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetPublishingMode handles GET /api/v2/publishing/mode
func (h *PublishingHandlers) GetPublishingMode(w http.ResponseWriter, r *http.Request) {
	targets := h.discoveryManager.ListTargets()
	enabledCount := 0
	for _, t := range targets {
		if t.Enabled {
			enabledCount++
		}
	}
	targetsAvailable := enabledCount > 0

	mode := "normal"
	metricsOnly := false
	if !targetsAvailable {
		mode = "metrics-only"
		metricsOnly = true
	}

	response := PublishingModeResponse{
		Mode:              mode,
		TargetsAvailable:  targetsAvailable,
		EnabledTargets:    enabledCount,
		MetricsOnlyActive: metricsOnly,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ===== Helper Methods =====

// parseListTargetsParams parses and validates query parameters for list targets endpoint
func (h *PublishingHandlers) parseListTargetsParams(r *http.Request) (*ListTargetsParams, error) {
	params := &ListTargetsParams{
		Limit:     100,
		SortBy:    "name",
		SortOrder: "asc",
	}

	query := r.URL.Query()

	// Parse type filter
	if typeStr := query.Get("type"); typeStr != "" {
		typeStr = strings.ToLower(typeStr)
		validTypes := map[string]bool{
			"rootly":    true,
			"pagerduty": true,
			"slack":     true,
			"webhook":   true,
		}
		if !validTypes[typeStr] {
			return nil, fmt.Errorf("invalid type: must be one of rootly, pagerduty, slack, webhook")
		}
		params.Type = &typeStr
	}

	// Parse enabled filter
	if enabledStr := query.Get("enabled"); enabledStr != "" {
		enabled, err := strconv.ParseBool(enabledStr)
		if err != nil {
			return nil, fmt.Errorf("invalid enabled: must be true or false")
		}
		params.Enabled = &enabled
	}

	// Parse limit
	if limitStr := query.Get("limit"); limitStr != "" {
		limit, err := strconv.ParseInt(limitStr, 10, 32)
		if err != nil || limit < 1 || limit > 1000 {
			return nil, fmt.Errorf("invalid limit: must be between 1 and 1000")
		}
		params.Limit = int(limit)
	}

	// Parse offset
	if offsetStr := query.Get("offset"); offsetStr != "" {
		offset, err := strconv.ParseInt(offsetStr, 10, 32)
		if err != nil || offset < 0 || offset > 1000000 {
			return nil, fmt.Errorf("invalid offset: must be between 0 and 1000000")
		}
		params.Offset = int(offset)
	}

	// Parse sort_by
	if sortBy := query.Get("sort_by"); sortBy != "" {
		sortBy = strings.ToLower(sortBy)
		validSortFields := map[string]bool{
			"name":    true,
			"type":    true,
			"enabled": true,
		}
		if !validSortFields[sortBy] {
			return nil, fmt.Errorf("invalid sort_by: must be one of name, type, enabled")
		}
		params.SortBy = sortBy
	}

	// Parse sort_order
	if sortOrder := query.Get("sort_order"); sortOrder != "" {
		sortOrder = strings.ToLower(sortOrder)
		if sortOrder != "asc" && sortOrder != "desc" {
			return nil, fmt.Errorf("invalid sort_order: must be asc or desc")
		}
		params.SortOrder = sortOrder
	}

	return params, nil
}

// filterTargets applies filters to targets list
func (h *PublishingHandlers) filterTargets(
	targets []*core.PublishingTarget,
	params *ListTargetsParams,
) []*core.PublishingTarget {
	filtered := make([]*core.PublishingTarget, 0, len(targets))

	for _, target := range targets {
		// Filter by type
		if params.Type != nil && !strings.EqualFold(target.Type, *params.Type) {
			continue
		}

		// Filter by enabled
		if params.Enabled != nil && target.Enabled != *params.Enabled {
			continue
		}

		filtered = append(filtered, target)
	}

	return filtered
}

// sortTargets sorts targets by specified field and order
func (h *PublishingHandlers) sortTargets(
	targets []*core.PublishingTarget,
	params *ListTargetsParams,
) {
	sort.Slice(targets, func(i, j int) bool {
		var less bool

		switch params.SortBy {
		case "name":
			less = targets[i].Name < targets[j].Name
		case "type":
			less = targets[i].Type < targets[j].Type
		case "enabled":
			less = !targets[i].Enabled && targets[j].Enabled
		default:
			less = targets[i].Name < targets[j].Name
		}

		if params.SortOrder == "desc" {
			return !less
		}
		return less
	})
}

// paginateTargets applies pagination to targets list
func (h *PublishingHandlers) paginateTargets(
	targets []*core.PublishingTarget,
	params *ListTargetsParams,
) []*core.PublishingTarget {
	start := params.Offset
	if start > len(targets) {
		return []*core.PublishingTarget{}
	}

	end := start + params.Limit
	if end > len(targets) {
		end = len(targets)
	}

	return targets[start:end]
}

// convertToTargetResponses converts PublishingTarget slice to TargetResponse slice
func (h *PublishingHandlers) convertToTargetResponses(
	targets []*core.PublishingTarget,
) []TargetResponse {
	responses := make([]TargetResponse, 0, len(targets))
	for _, t := range targets {
		responses = append(responses, TargetResponse{
			Name:    t.Name,
			Type:    t.Type,
			URL:     t.URL,
			Enabled: t.Enabled,
			Format:  string(t.Format),
			Headers: t.Headers,
		})
	}
	return responses
}

func (h *PublishingHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

// buildTestAlert creates a test alert from the request configuration.
//
// This method supports both default and custom test alerts:
//   - If TestAlert is provided, it uses the custom payload
//   - Otherwise, creates a default test alert with the specified AlertName
//
// The method automatically adds a "test: true" label to identify test alerts
// and creates a ClassificationResult with test data.
//
// Parameters:
//   - req: Test request configuration (may be nil for defaults)
//   - targetName: Target name for fingerprint generation
//
// Returns:
//   - EnrichedAlert: Fully configured test alert ready for publishing
//   - error: Validation error if custom alert is invalid
//
// Example:
//
//	alert, err := handler.buildTestAlert(&TestTargetRequest{
//	    AlertName: "MyTestAlert",
//	}, "rootly-prod")
//	if err != nil {
//	    return err
//	}
//	// alert is ready to publish
func (h *PublishingHandlers) buildTestAlert(req *TestTargetRequest, targetName string) (*core.EnrichedAlert, error) {
	now := time.Now()
	generatorURL := fmt.Sprintf("http://test/alert/%s", targetName)

	// If custom test alert provided, use it
	if req.TestAlert != nil {
		alert := &core.EnrichedAlert{
			Alert: &core.Alert{
				Fingerprint: req.TestAlert.Fingerprint,
				AlertName:   req.TestAlert.Labels["alertname"],
				Status:      core.StatusFiring,
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				StartsAt:    now,
			},
		}

		// Copy labels from custom alert
		if req.TestAlert.Labels != nil {
			for k, v := range req.TestAlert.Labels {
				alert.Alert.Labels[k] = v
			}
		}

		// Copy annotations from custom alert
		if req.TestAlert.Annotations != nil {
			for k, v := range req.TestAlert.Annotations {
				alert.Alert.Annotations[k] = v
			}
		}

		// Ensure test labels are present
		alert.Alert.Labels["test"] = "true"
		if alert.Alert.Labels["severity"] == "" {
			alert.Alert.Labels["severity"] = "info"
		}
		if alert.Alert.AlertName == "" {
			alert.Alert.AlertName = "TestAlert"
		}

		// Set status
		if req.TestAlert.Status == "resolved" {
			alert.Alert.Status = core.StatusResolved
		} else {
			alert.Alert.Status = core.StatusFiring
		}

		// Generate fingerprint if not provided
		if alert.Alert.Fingerprint == "" {
			alert.Alert.Fingerprint = fmt.Sprintf("test-%s-%d", targetName, now.Unix())
		}

		// Set generator URL
		alert.Alert.GeneratorURL = &generatorURL

		// Add classification
		alert.Classification = &core.ClassificationResult{
			Severity:   core.SeverityInfo,
			Confidence: 1.0,
			Reasoning:  "Test alert",
			Recommendations: []string{
				"This is a test - no action required",
			},
		}

		return alert, nil
	}

	// Default test alert
	alertName := req.AlertName
	if alertName == "" {
		alertName = "TestAlert"
	}

	return &core.EnrichedAlert{
		Alert: &core.Alert{
			Fingerprint: fmt.Sprintf("test-%s-%d", targetName, now.Unix()),
			AlertName:   alertName,
			Status:      core.StatusFiring,
			Labels: map[string]string{
				"alertname": alertName,
				"severity":  "info",
				"test":      "true",
			},
			Annotations: map[string]string{
				"summary":     "Test alert for target validation",
				"description": fmt.Sprintf("This is a test alert sent via API for target %s", targetName),
			},
			StartsAt:     now,
			GeneratorURL: &generatorURL,
		},
		Classification: &core.ClassificationResult{
			Severity:   core.SeverityInfo,
			Confidence: 1.0,
			Reasoning:  "Test alert",
			Recommendations: []string{
				"This is a test - no action required",
			},
		},
	}, nil
}

// createTestAlert creates a simple test alert (backward compatibility)
func (h *PublishingHandlers) createTestAlert(alertName string) *core.EnrichedAlert {
	req := &TestTargetRequest{
		AlertName: alertName,
	}
	alert, _ := h.buildTestAlert(req, "default")
	return alert
}

func jobSnapshotToResponse(job *infrapub.JobSnapshot) JobStatusResponse {
	response := JobStatusResponse{
		ID:          job.ID,
		Fingerprint: job.Fingerprint,
		TargetName:  job.TargetName,
		Priority:    job.Priority,
		State:       job.State,
		RetryCount:  job.RetryCount,
		MaxRetries:  0,
		CreatedAt:   time.Unix(job.SubmittedAt, 0),
	}

	if job.ErrorType != "" {
		response.ErrorType = job.ErrorType
	}

	if job.CompletedAt != nil {
		completedTime := time.Unix(*job.CompletedAt, 0)
		response.CompletedAt = &completedTime
		processingTime := completedTime.Sub(time.Unix(job.SubmittedAt, 0))
		response.ProcessingTime = processingTime.String()
	}

	return response
}
