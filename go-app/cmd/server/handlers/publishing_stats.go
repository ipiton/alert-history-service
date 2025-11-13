package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// MetricsCollectorInterface abstracts metrics collector for testing.
type MetricsCollectorInterface interface {
	CollectAll(ctx context.Context) *publishing.MetricsSnapshot
}

// PublishingStatsHandler handles HTTP requests for publishing metrics & stats.
//
// This handler provides 5 REST endpoints:
//   1. GET  /api/v2/publishing/metrics       - Raw metrics snapshot
//   2. GET  /api/v2/publishing/stats         - Aggregated statistics
//   3. GET  /api/v2/publishing/stats/{target} - Per-target statistics
//   4. GET  /api/v2/publishing/health        - System health summary
//   5. GET  /api/v2/publishing/trends        - Trend analysis
//
// Performance Target: <10ms total response time
//
// Thread-Safe: Yes (PublishingMetricsCollector is thread-safe)
type PublishingStatsHandler struct {
	collector      MetricsCollectorInterface
	trendDetector  *publishing.TrendDetector
	logger         *slog.Logger
}

// NewPublishingStatsHandler creates a new handler.
func NewPublishingStatsHandler(
	collector *publishing.PublishingMetricsCollector,
	trendDetector *publishing.TrendDetector,
	logger *slog.Logger,
) *PublishingStatsHandler {
	return &PublishingStatsHandler{
		collector:     collector,
		trendDetector: trendDetector,
		logger:        logger,
	}
}

// NewPublishingStatsHandlerWithCollector creates a handler with custom collector interface.
// Used for testing with mock collectors.
func NewPublishingStatsHandlerWithCollector(
	collector MetricsCollectorInterface,
	logger *slog.Logger,
) *PublishingStatsHandler {
	return &PublishingStatsHandler{
		collector: collector,
		logger:    logger,
	}
}

// MockCollectorForHandler is a mock implementation of MetricsCollectorInterface for testing.
type MockCollectorForHandler struct {
	CollectAllFunc        func(ctx context.Context) *publishing.MetricsSnapshot
	GetCollectorCountFunc func() int
	GetCollectorNamesFunc func() []string
}

func (m *MockCollectorForHandler) CollectAll(ctx context.Context) *publishing.MetricsSnapshot {
	if m.CollectAllFunc != nil {
		return m.CollectAllFunc(ctx)
	}
	return &publishing.MetricsSnapshot{
		Timestamp:      time.Now(),
		Metrics:        make(map[string]float64),
		CollectorCount: 0,
	}
}

// ============================================================================
// Response Models
// ============================================================================

// MetricsResponse represents raw metrics snapshot response.
type MetricsResponse struct {
	Timestamp           time.Time          `json:"timestamp"`
	CollectionDuration  string             `json:"collection_duration_ms"`
	MetricsCount        int                `json:"metrics_count"`
	Metrics             map[string]float64 `json:"metrics"`
	AvailableCollectors []string           `json:"available_collectors"`
	Errors              map[string]string  `json:"errors,omitempty"`
}

// StatsResponse represents aggregated statistics.
type StatsResponse struct {
	Timestamp   time.Time          `json:"timestamp"`
	System      SystemStats        `json:"system"`
	TargetStats map[string]float64 `json:"target_stats"`
	QueueStats  map[string]float64 `json:"queue_stats"`
}

// SystemStats represents system-wide statistics.
type SystemStats struct {
	TotalTargets     int     `json:"total_targets"`
	HealthyTargets   int     `json:"healthy_targets"`
	UnhealthyTargets int     `json:"unhealthy_targets"`
	SuccessRate      float64 `json:"success_rate_percent"`
	QueueSize        int     `json:"queue_size"`
	QueueCapacity    int     `json:"queue_capacity"`
}

// PublishingHealthResponse represents publishing system health check summary.
type PublishingHealthResponse struct {
	Status    string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
	Message   string            `json:"message,omitempty"`
}

// ============================================================================
// Endpoint 1: GET /api/v2/publishing/metrics (Raw Metrics)
// ============================================================================

// GetMetrics handles GET /api/v2/publishing/metrics
//
// This endpoint returns raw metrics snapshot from all collectors.
//
// Response Example:
//
//	{
//	  "timestamp": "2025-11-12T10:30:00Z",
//	  "collection_duration_ms": "0.085",
//	  "metrics_count": 42,
//	  "metrics": {
//	    "health_status{target=\"rootly-prod\",type=\"rootly\"}": 1.0,
//	    "queue_size_total": 15.0,
//	    "targets_total": 10.0
//	  },
//	  "available_collectors": ["health", "refresh", "discovery", "queue"]
//	}
func (h *PublishingStatsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	startTime := time.Now()

	// Collect metrics from all collectors
	snapshot := h.collector.CollectAll(ctx)

	// Convert errors map to string map for JSON
	errorsMap := make(map[string]string)
	for name, err := range snapshot.Errors {
		errorsMap[name] = err.Error()
	}

	response := MetricsResponse{
		Timestamp:           snapshot.Timestamp,
		CollectionDuration:  formatDuration(snapshot.CollectionDuration),
		MetricsCount:        len(snapshot.Metrics),
		Metrics:             snapshot.Metrics,
		AvailableCollectors: snapshot.AvailableCollectors,
		Errors:              errorsMap,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode metrics response", "error", err)
	}

	totalDuration := time.Since(startTime)
	h.logger.Debug("Metrics endpoint called",
		"metrics_count", len(snapshot.Metrics),
		"collection_duration", snapshot.CollectionDuration,
		"total_duration", totalDuration,
	)
}

// ============================================================================
// Endpoint 2: GET /api/v2/publishing/stats (Aggregated Stats)
// ============================================================================

// GetStats handles GET /api/v2/publishing/stats
//
// This endpoint returns aggregated statistics computed from raw metrics.
//
// Response Example:
//
//	{
//	  "timestamp": "2025-11-12T10:30:00Z",
//	  "system": {
//	    "total_targets": 10,
//	    "healthy_targets": 8,
//	    "unhealthy_targets": 2,
//	    "success_rate_percent": 95.5,
//	    "queue_size": 15,
//	    "queue_capacity": 1000
//	  },
//	  "target_stats": {...},
//	  "queue_stats": {...}
//	}
func (h *PublishingStatsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Collect metrics
	snapshot := h.collector.CollectAll(ctx)

	// Separate metrics by category
	targetStats := make(map[string]float64)
	queueStats := make(map[string]float64)

	for key, value := range snapshot.Metrics {
		if strings.Contains(key, "target") || strings.Contains(key, "health") || strings.Contains(key, "discovery") {
			targetStats[key] = value
		} else if strings.Contains(key, "queue") || strings.Contains(key, "worker") || strings.Contains(key, "job") {
			queueStats[key] = value
		}
	}

	// Calculate system-wide statistics
	systemStats := SystemStats{
		TotalTargets:     int(getMetricValue(snapshot.Metrics, "targets_total")),
		HealthyTargets:   countHealthyTargets(snapshot.Metrics),
		UnhealthyTargets: countUnhealthyTargets(snapshot.Metrics),
		SuccessRate:      calculateSuccessRate(snapshot.Metrics),
		QueueSize:        int(getMetricValue(snapshot.Metrics, "queue_size_total")),
		QueueCapacity:    int(getMetricValue(snapshot.Metrics, "queue_capacity")),
	}

	response := StatsResponse{
		Timestamp:   time.Now(),
		System:      systemStats,
		TargetStats: targetStats,
		QueueStats:  queueStats,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode stats response", "error", err)
	}

	h.logger.Debug("Stats endpoint called",
		"total_targets", systemStats.TotalTargets,
		"healthy_targets", systemStats.HealthyTargets,
	)
}

// ============================================================================
// Endpoint 3: GET /api/v2/publishing/health (Health Check)
// ============================================================================

// GetHealth handles GET /api/v2/publishing/health
//
// This endpoint returns system health summary for monitoring.
//
// Response Example:
//
//	{
//	  "status": "healthy",
//	  "timestamp": "2025-11-12T10:30:00Z",
//	  "checks": {
//	    "health": "ok",
//	    "refresh": "ok",
//	    "discovery": "ok",
//	    "queue": "ok"
//	  },
//	  "message": "All systems operational"
//	}
//
// HTTP Status Codes:
//   - 200: Healthy (all checks passed)
//   - 503: Degraded or Unhealthy (some checks failed)
func (h *PublishingStatsHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Collect metrics
	snapshot := h.collector.CollectAll(ctx)

	// Initialize health checks
	status := "healthy"
	checks := make(map[string]string)

	// Check if collectors are available
	for _, collectorName := range snapshot.AvailableCollectors {
		checks[collectorName] = "ok"
	}

	// Check for collection errors
	if len(snapshot.Errors) > 0 {
		status = "degraded"
		for name, err := range snapshot.Errors {
			checks[name] = "error: " + err.Error()
		}
	}

	// Check unhealthy targets threshold
	unhealthyCount := countUnhealthyTargets(snapshot.Metrics)
	totalTargets := int(getMetricValue(snapshot.Metrics, "targets_total"))

	if unhealthyCount > 0 {
		if totalTargets > 0 && float64(unhealthyCount)/float64(totalTargets) > 0.5 {
			status = "unhealthy"
			checks["unhealthy_targets"] = "critical"
		} else {
			if status == "healthy" {
				status = "degraded"
			}
			checks["unhealthy_targets"] = "warning"
		}
	}

	// Generate health message
	message := generateHealthMessage(status, unhealthyCount, totalTargets)

	response := PublishingHealthResponse{
		Status:    status,
		Timestamp: time.Now(),
		Checks:    checks,
		Message:   message,
	}

	// Set HTTP status code based on health
	statusCode := http.StatusOK
	if status == "degraded" || status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode health response", "error", err)
	}

	h.logger.Debug("Health endpoint called",
		"status", status,
		"unhealthy_targets", unhealthyCount,
	)
}

// ============================================================================
// Endpoint 4: GET /api/v2/publishing/stats/{target} (Per-Target Stats)
// ============================================================================

// TargetStatsResponse represents per-target statistics.
type TargetStatsResponse struct {
	TargetName string             `json:"target_name"`
	Timestamp  time.Time          `json:"timestamp"`
	Health     TargetHealthInfo   `json:"health"`
	Jobs       TargetJobInfo      `json:"jobs"`
	Metrics    map[string]float64 `json:"metrics"`
}

// TargetHealthInfo represents target health information.
type TargetHealthInfo struct {
	Status              string  `json:"status"` // "healthy", "degraded", "unhealthy"
	SuccessRate         float64 `json:"success_rate_percent"`
	ConsecutiveFailures int     `json:"consecutive_failures"`
	LastCheck           string  `json:"last_check,omitempty"`
}

// TargetJobInfo represents target job processing information.
type TargetJobInfo struct {
	TotalProcessed int     `json:"total_processed"`
	Succeeded      int     `json:"succeeded"`
	Failed         int     `json:"failed"`
	SuccessRate    float64 `json:"success_rate_percent"`
}

// GetTargetStats handles GET /api/v2/publishing/stats/{target}
//
// This endpoint returns statistics for a specific target.
//
// URL Parameters:
//   - target: Target name (e.g., "rootly-prod")
//
// Response Example:
//
//	{
//	  "target_name": "rootly-prod",
//	  "timestamp": "2025-11-12T10:30:00Z",
//	  "health": {
//	    "status": "healthy",
//	    "success_rate_percent": 99.5,
//	    "consecutive_failures": 0
//	  },
//	  "jobs": {
//	    "total_processed": 1000,
//	    "succeeded": 995,
//	    "failed": 5,
//	    "success_rate_percent": 99.5
//	  },
//	  "metrics": {...}
//	}
//
// HTTP Status Codes:
//   - 200: Target found
//   - 404: Target not found
func (h *PublishingStatsHandler) GetTargetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract target name from URL path
	// Expected format: /api/v2/publishing/stats/{target}
	pathParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v2/publishing/stats/"), "/")
	if len(pathParts) == 0 || pathParts[0] == "" {
		http.Error(w, "Target name required", http.StatusBadRequest)
		return
	}
	targetName := pathParts[0]

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Collect all metrics
	snapshot := h.collector.CollectAll(ctx)

	// Filter metrics for this target
	targetMetrics := make(map[string]float64)
	hasMetrics := false

	for key, value := range snapshot.Metrics {
		if strings.Contains(key, targetName) {
			targetMetrics[key] = value
			hasMetrics = true
		}
	}

	// Check if target exists
	if !hasMetrics {
		http.Error(w, "Target not found: "+targetName, http.StatusNotFound)
		return
	}

	// Extract health information
	healthStatus := extractTargetHealthStatus(snapshot.Metrics, targetName)
	healthInfo := TargetHealthInfo{
		Status:              healthStatus,
		SuccessRate:         extractTargetSuccessRate(snapshot.Metrics, targetName),
		ConsecutiveFailures: extractConsecutiveFailures(snapshot.Metrics, targetName),
	}

	// Extract job information
	jobInfo := TargetJobInfo{
		TotalProcessed: extractJobsProcessed(snapshot.Metrics, targetName),
		Succeeded:      extractJobsSucceeded(snapshot.Metrics, targetName),
		Failed:         extractJobsFailed(snapshot.Metrics, targetName),
		SuccessRate:    calculateTargetJobSuccessRate(snapshot.Metrics, targetName),
	}

	response := TargetStatsResponse{
		TargetName: targetName,
		Timestamp:  time.Now(),
		Health:     healthInfo,
		Jobs:       jobInfo,
		Metrics:    targetMetrics,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode target stats response", "error", err)
	}

	h.logger.Debug("Target stats endpoint called",
		"target", targetName,
		"metrics_count", len(targetMetrics),
	)
}

// ============================================================================
// Endpoint 5: GET /api/v2/publishing/trends (Trend Analysis)
// ============================================================================

// TrendsResponse represents trend analysis response.
type TrendsResponse struct {
	Timestamp time.Time                  `json:"timestamp"`
	Trends    publishing.TrendAnalysis   `json:"trends"`
	Summary   string                     `json:"summary"`
}

// GetTrends handles GET /api/v2/publishing/trends
//
// This endpoint returns historical trend analysis including:
//   - Success rate trends (increasing/stable/decreasing)
//   - Latency trends (improving/stable/degrading)
//   - Error spike detection (>3σ anomaly)
//   - Queue growth rate
//
// Response Example:
//
//	{
//	  "timestamp": "2025-11-13T10:30:00Z",
//	  "trends": {
//	    "success_rate_trend": "stable",
//	    "success_rate_change": 0.5,
//	    "latency_trend": "improving",
//	    "latency_change": -15.3,
//	    "error_spike_detected": false,
//	    "queue_growth_rate": 2.5,
//	    "queue_growth_trend": "growing"
//	  },
//	  "summary": "System stable. Latency improving. Queue growing slowly."
//	}
//
// HTTP Status Codes:
//   - 200: Trends available
//   - 503: Not enough historical data (< 10 snapshots)
func (h *PublishingStatsHandler) GetTrends(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Analyze trends
	trends := h.trendDetector.Analyze()

	// Generate human-readable summary
	summary := generateTrendsSummary(trends)

	response := TrendsResponse{
		Timestamp: time.Now(),
		Trends:    trends,
		Summary:   summary,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode trends response", "error", err)
	}

	h.logger.Debug("Trends endpoint called",
		"success_rate_trend", trends.SuccessRateTrend,
		"latency_trend", trends.LatencyTrend,
		"error_spike", trends.ErrorSpikeDetected,
	)
}

// ============================================================================
// Helper Functions
// ============================================================================

// formatDuration formats duration in milliseconds with 3 decimal places.
func formatDuration(d time.Duration) string {
	ms := float64(d.Microseconds()) / 1000.0
	return formatFloat(ms, 3)
}

// formatFloat formats float with specified precision.
func formatFloat(f float64, precision int) string {
	format := "%." + string(rune('0'+precision)) + "f"
	return strings.TrimRight(strings.TrimRight(formatFloatHelper(f, format), "0"), ".")
}
