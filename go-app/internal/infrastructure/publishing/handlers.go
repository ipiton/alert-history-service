package publishing

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// PublishingHandlers provides HTTP handlers for publishing management
type PublishingHandlers struct {
	discoveryManager TargetDiscoveryManager
	refreshManager   *RefreshManager
	queue            *PublishingQueue
	coordinator      *PublishingCoordinator
	logger           *slog.Logger
}

// NewPublishingHandlers creates new publishing HTTP handlers
func NewPublishingHandlers(
	discoveryManager TargetDiscoveryManager,
	refreshManager *RefreshManager,
	queue *PublishingQueue,
	coordinator *PublishingCoordinator,
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

// RegisterRoutes registers all publishing routes
func (h *PublishingHandlers) RegisterRoutes(router *mux.Router) {
	// Publishing management endpoints
	api := router.PathPrefix("/api/v1/publishing").Subrouter()

	// 1. List all targets
	api.HandleFunc("/targets", h.ListTargets).Methods("GET")

	// 2. Get specific target
	api.HandleFunc("/targets/{name}", h.GetTarget).Methods("GET")

	// 3. Refresh targets (manual)
	api.HandleFunc("/targets/refresh", h.RefreshTargets).Methods("POST")

	// 4. Test target
	api.HandleFunc("/targets/{name}/test", h.TestTarget).Methods("POST")

	// 5. Get statistics
	api.HandleFunc("/stats", h.GetStats).Methods("GET")

	// 6. Get queue status
	api.HandleFunc("/queue", h.GetQueueStatus).Methods("GET")

	// 7. Get publishing mode
	api.HandleFunc("/mode", h.GetPublishingMode).Methods("GET")
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
	TotalTargets   int                       `json:"total_targets"`
	EnabledTargets int                       `json:"enabled_targets"`
	TargetsByType  map[string]int            `json:"targets_by_type"`
	QueueSize      int                       `json:"queue_size"`
	QueueCapacity  int                       `json:"queue_capacity"`
	QueueUtilization float64                 `json:"queue_utilization_percent"`
}

// QueueStatusResponse represents queue status
type QueueStatusResponse struct {
	Size             int     `json:"size"`
	Capacity         int     `json:"capacity"`
	Utilization      float64 `json:"utilization_percent"`
	WorkersCount     int     `json:"workers_count"`
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

// ErrorResponse represents error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// 1. ListTargets - GET /api/v1/publishing/targets
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

// 2. GetTarget - GET /api/v1/publishing/targets/{name}
func (h *PublishingHandlers) GetTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	target, err := h.discoveryManager.GetTarget(name)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "Target not found", err.Error())
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

// 3. RefreshTargets - POST /api/v1/publishing/targets/refresh
func (h *PublishingHandlers) RefreshTargets(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("Manual target refresh requested via API")

	err := h.refreshManager.RefreshNow(r.Context())
	if err != nil {
		h.logger.Error("Manual refresh failed", "error", err)
		h.sendError(w, http.StatusInternalServerError, "Refresh failed", err.Error())
		return
	}

	targetCount := h.discoveryManager.GetTargetCount()

	// Count enabled targets
	targets := h.discoveryManager.ListTargets()
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

// 4. TestTarget - POST /api/v1/publishing/targets/{name}/test
func (h *PublishingHandlers) TestTarget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]

	// Decode request body (optional)
	var req TestTargetRequest
	if r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.sendError(w, http.StatusBadRequest, "Invalid request body", err.Error())
			return
		}
	}

	// Get target
	target, err := h.discoveryManager.GetTarget(name)
	if err != nil {
		h.sendError(w, http.StatusNotFound, "Target not found", err.Error())
		return
	}

	if !target.Enabled {
		h.sendJSON(w, http.StatusOK, TestTargetResponse{
			Success: false,
			Message: "Target is disabled",
		})
		return
	}

	// Create test alert
	testAlert := h.createTestAlert(req.AlertName)

	// Try to publish
	results, err := h.coordinator.PublishToTargets(r.Context(), testAlert, []string{name})
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Test failed", err.Error())
		return
	}

	if len(results) == 0 {
		h.sendError(w, http.StatusInternalServerError, "No results", "Failed to publish test alert")
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

// 5. GetStats - GET /api/v1/publishing/stats
func (h *PublishingHandlers) GetStats(w http.ResponseWriter, r *http.Request) {
	targets := h.discoveryManager.ListTargets()

	// Count by type
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

// 6. GetQueueStatus - GET /api/v1/publishing/queue
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
		WorkersCount: 10, // From config, hardcoded for now
	}

	h.sendJSON(w, http.StatusOK, response)
}

// 7. GetPublishingMode - GET /api/v1/publishing/mode
func (h *PublishingHandlers) GetPublishingMode(w http.ResponseWriter, r *http.Request) {
	// Count enabled targets
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

// Helper methods

func (h *PublishingHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}

func (h *PublishingHandlers) sendError(w http.ResponseWriter, status int, message string, details string) {
	h.logger.Warn("API error", "status", status, "message", message, "details", details)
	h.sendJSON(w, status, ErrorResponse{
		Error:   message,
		Message: details,
	})
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
