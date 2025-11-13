package publishing

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
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

// TestTargetRequest represents test request
type TestTargetRequest struct {
	AlertName string `json:"alert_name"`
}

// TestTargetResponse represents test result
type TestTargetResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
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

// ===== Target Management Handlers =====

// ListTargets handles GET /api/v2/publishing/targets
//
// @Summary List all publishing targets
// @Description Returns a list of all configured publishing targets
// @Tags Targets
// @Produce json
// @Param type query string false "Filter by target type"
// @Param enabled query bool false "Filter by enabled status"
// @Success 200 {array} TargetResponse
// @Router /publishing/targets [get]
func (h *PublishingHandlers) ListTargets(w http.ResponseWriter, r *http.Request) {
	targets := h.discoveryManager.ListTargets()

	response := make([]TargetResponse, 0, len(targets))
	for _, t := range targets {
		response = append(response, TargetResponse{
			Name:    t.Name,
			Type:    t.Type,
			URL:     t.URL,
			Enabled: t.Enabled,
			Format:  string(t.Format),
			Headers: t.Headers,
		})
	}

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
// @Failure 404 {object} apierrors.ErrorResponse
// @Router /publishing/targets/{name}/test [post]
func (h *PublishingHandlers) TestTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	var req TestTargetRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			apiErr := apierrors.ValidationError("Invalid request body").
				WithRequestID(middleware.GetRequestID(r.Context()))
			apierrors.WriteError(w, apiErr)
			return
		}
	}

	target, err := h.discoveryManager.GetTarget(name)
	if err != nil {
		apiErr := apierrors.NotFoundError("Target").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	if !target.Enabled {
		h.sendJSON(w, http.StatusOK, TestTargetResponse{
			Success: false,
			Message: "Target is disabled",
		})
		return
	}

	testAlert := h.createTestAlert(req.AlertName)
	results, err := h.coordinator.PublishToTargets(r.Context(), testAlert, []string{name})
	if err != nil {
		apiErr := apierrors.InternalError("Test failed: " + err.Error()).
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	if len(results) == 0 {
		apiErr := apierrors.InternalError("No results returned").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	result := results[0]
	response := TestTargetResponse{
		Success: result.Success,
		Message: "Test alert sent",
	}

	if result.Error != nil {
		response.Error = result.Error.Error()
	}

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

func (h *PublishingHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *PublishingHandlers) createTestAlert(alertName string) *core.EnrichedAlert {
	if alertName == "" {
		alertName = "TestAlert"
	}

	now := time.Now()
	generatorURL := "http://test/alert"

	return &core.EnrichedAlert{
		Alert: &core.Alert{
			Fingerprint: "test-" + alertName,
			AlertName:   alertName,
			Status:      core.StatusFiring,
			Labels: map[string]string{
				"alertname": alertName,
				"severity":  "info",
				"test":      "true",
			},
			Annotations: map[string]string{
				"summary":     "Test alert for target validation",
				"description": "This is a test alert sent via API",
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
	}
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
