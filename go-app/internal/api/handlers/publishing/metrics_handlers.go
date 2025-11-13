package publishing

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/business/publishing"
)

// MetricsCollectorInterface abstracts metrics collector for testing
type MetricsCollectorInterface interface {
	CollectAll(ctx context.Context) *publishing.MetricsSnapshot
}

// MetricsHandlers provides HTTP handlers for publishing metrics & stats (TN-057)
type MetricsHandlers struct {
	collector     MetricsCollectorInterface
	trendDetector *publishing.TrendDetector
	logger        *slog.Logger
}

// NewMetricsHandlers creates new metrics handlers
func NewMetricsHandlers(
	collector *publishing.PublishingMetricsCollector,
	trendDetector *publishing.TrendDetector,
	logger *slog.Logger,
) *MetricsHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &MetricsHandlers{
		collector:     collector,
		trendDetector: trendDetector,
		logger:        logger,
	}
}

// MetricsResponse represents raw metrics snapshot response
type MetricsResponse struct {
	Timestamp           time.Time          `json:"timestamp"`
	CollectionDuration  string             `json:"collection_duration_ms"`
	MetricsCount        int                `json:"metrics_count"`
	Metrics             map[string]float64 `json:"metrics"`
	AvailableCollectors []string           `json:"available_collectors"`
	Errors              map[string]string  `json:"errors,omitempty"`
}

// SystemStats represents system-level statistics
type SystemStats struct {
	TotalTargets   int     `json:"total_targets"`
	HealthyTargets int     `json:"healthy_targets"`
	QueueSize      int     `json:"queue_size"`
	QueueCapacity  int     `json:"queue_capacity"`
	Utilization    float64 `json:"utilization_percent"`
}

// AggregatedStatsResponse represents aggregated statistics
type AggregatedStatsResponse struct {
	Timestamp   time.Time          `json:"timestamp"`
	System      SystemStats        `json:"system"`
	TargetStats map[string]float64 `json:"target_stats"`
}

// TargetStatsResponse represents per-target statistics
type TargetStatsResponse struct {
	TargetName string             `json:"target_name"`
	Timestamp  time.Time          `json:"timestamp"`
	Metrics    map[string]float64 `json:"metrics"`
}

// HealthResponse represents system health summary
type HealthResponse struct {
	Status             string    `json:"status"`
	Timestamp          time.Time `json:"timestamp"`
	TotalTargets       int       `json:"total_targets"`
	HealthyTargets     int       `json:"healthy_targets"`
	UnhealthyTargets   int       `json:"unhealthy_targets"`
	QueueUtilization   float64   `json:"queue_utilization_percent"`
	AvgResponseTime    float64   `json:"avg_response_time_ms"`
	ErrorRate          float64   `json:"error_rate_percent"`
	HealthScore        float64   `json:"health_score"`
	Issues             []string  `json:"issues,omitempty"`
}

// TrendResponse represents trend analysis
type TrendResponse struct {
	Timestamp time.Time                  `json:"timestamp"`
	Trends    map[string]TrendAnalysis   `json:"trends"`
}

// TrendAnalysis represents analysis for a single metric
type TrendAnalysis struct {
	MetricName string  `json:"metric_name"`
	Current    float64 `json:"current"`
	Trend      string  `json:"trend"` // "increasing", "decreasing", "stable"
	ChangeRate float64 `json:"change_rate_percent"`
	Severity   string  `json:"severity"` // "normal", "warning", "critical"
}

// GetPublishingMetrics handles GET /api/v2/publishing/metrics
//
// @Summary Get raw metrics snapshot
// @Description Returns raw metrics snapshot from all collectors
// @Tags Metrics
// @Produce json
// @Success 200 {object} MetricsResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/metrics [get]
func (h *MetricsHandlers) GetPublishingMetrics(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	snapshot := h.collector.CollectAll(ctx)
	if snapshot == nil {
		apiErr := apierrors.InternalError("Failed to collect metrics").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	errors := make(map[string]string)
	for name, err := range snapshot.Errors {
		errors[name] = err.Error()
	}

	response := MetricsResponse{
		Timestamp:           snapshot.Timestamp,
		CollectionDuration:  snapshot.CollectionDuration.String(),
		MetricsCount:        len(snapshot.Metrics),
		Metrics:             snapshot.Metrics,
		AvailableCollectors: snapshot.AvailableCollectors,
		Errors:              errors,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetPublishingStats handles GET /api/v2/publishing/stats
//
// @Summary Get aggregated statistics
// @Description Returns aggregated statistics from all subsystems
// @Tags Metrics
// @Produce json
// @Success 200 {object} AggregatedStatsResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/stats [get]
func (h *MetricsHandlers) GetPublishingStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	snapshot := h.collector.CollectAll(ctx)
	if snapshot == nil {
		apiErr := apierrors.InternalError("Failed to collect stats").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Extract system stats
	systemStats := SystemStats{
		TotalTargets:   int(snapshot.Metrics["discovery.total_targets"]),
		HealthyTargets: int(snapshot.Metrics["health.healthy_targets"]),
		QueueSize:      int(snapshot.Metrics["queue.total_size"]),
		QueueCapacity:  int(snapshot.Metrics["queue.capacity"]),
	}

	if systemStats.QueueCapacity > 0 {
		systemStats.Utilization = float64(systemStats.QueueSize) / float64(systemStats.QueueCapacity) * 100
	}

	// Extract target-specific stats
	targetStats := make(map[string]float64)
	for key, value := range snapshot.Metrics {
		if strings.HasPrefix(key, "target.") {
			targetStats[key] = value
		}
	}

	response := AggregatedStatsResponse{
		Timestamp:   snapshot.Timestamp,
		System:      systemStats,
		TargetStats: targetStats,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetTargetPublishingStats handles GET /api/v2/publishing/stats/{target}
//
// @Summary Get per-target statistics
// @Description Returns statistics for a specific target
// @Tags Metrics
// @Produce json
// @Param target path string true "Target name"
// @Success 200 {object} TargetStatsResponse
// @Failure 404 {object} apierrors.ErrorResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/stats/{target} [get]
func (h *MetricsHandlers) GetTargetPublishingStats(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	targetName := vars["target"]

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	snapshot := h.collector.CollectAll(ctx)
	if snapshot == nil {
		apiErr := apierrors.InternalError("Failed to collect stats").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	// Filter metrics for this target
	targetMetrics := make(map[string]float64)
	prefix := "target." + targetName + "."
	for key, value := range snapshot.Metrics {
		if strings.HasPrefix(key, prefix) {
			shortKey := strings.TrimPrefix(key, prefix)
			targetMetrics[shortKey] = value
		}
	}

	if len(targetMetrics) == 0 {
		apiErr := apierrors.NotFoundError("Target metrics").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	response := TargetStatsResponse{
		TargetName: targetName,
		Timestamp:  snapshot.Timestamp,
		Metrics:    targetMetrics,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetPublishingHealth handles GET /api/v2/publishing/health
//
// @Summary Get system health summary
// @Description Returns overall health status of the publishing system
// @Tags Metrics
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/health [get]
func (h *MetricsHandlers) GetPublishingHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	snapshot := h.collector.CollectAll(ctx)
	if snapshot == nil {
		apiErr := apierrors.InternalError("Failed to collect health data").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	totalTargets := int(snapshot.Metrics["discovery.total_targets"])
	healthyTargets := int(snapshot.Metrics["health.healthy_targets"])
	unhealthyTargets := totalTargets - healthyTargets

	queueSize := snapshot.Metrics["queue.total_size"]
	queueCapacity := snapshot.Metrics["queue.capacity"]
	queueUtilization := 0.0
	if queueCapacity > 0 {
		queueUtilization = (queueSize / queueCapacity) * 100
	}

	avgResponseTime := snapshot.Metrics["health.avg_response_time_ms"]
	errorRate := snapshot.Metrics["queue.error_rate_percent"]

	// Calculate health score (0-100)
	healthScore := 100.0
	issues := []string{}

	if healthyTargets < totalTargets {
		healthScore -= float64(unhealthyTargets) * 10.0
		issues = append(issues, "Some targets are unhealthy")
	}

	if queueUtilization > 80 {
		healthScore -= 20.0
		issues = append(issues, "Queue utilization is high")
	}

	if errorRate > 5.0 {
		healthScore -= 15.0
		issues = append(issues, "Error rate is elevated")
	}

	if avgResponseTime > 1000 {
		healthScore -= 10.0
		issues = append(issues, "Response times are slow")
	}

	if healthScore < 0 {
		healthScore = 0
	}

	status := "healthy"
	if healthScore < 50 {
		status = "unhealthy"
	} else if healthScore < 80 {
		status = "degraded"
	}

	response := HealthResponse{
		Status:           status,
		Timestamp:        snapshot.Timestamp,
		TotalTargets:     totalTargets,
		HealthyTargets:   healthyTargets,
		UnhealthyTargets: unhealthyTargets,
		QueueUtilization: queueUtilization,
		AvgResponseTime:  avgResponseTime,
		ErrorRate:        errorRate,
		HealthScore:      healthScore,
		Issues:           issues,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// GetPublishingTrends handles GET /api/v2/publishing/trends
//
// @Summary Get trend analysis
// @Description Returns trend analysis for key metrics
// @Tags Metrics
// @Produce json
// @Success 200 {object} TrendResponse
// @Failure 500 {object} apierrors.ErrorResponse
// @Router /publishing/trends [get]
func (h *MetricsHandlers) GetPublishingTrends(w http.ResponseWriter, r *http.Request) {
	if h.trendDetector == nil {
		apiErr := apierrors.InternalError("Trend detection not available").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	snapshot := h.collector.CollectAll(ctx)
	if snapshot == nil {
		apiErr := apierrors.InternalError("Failed to collect trends").
			WithRequestID(middleware.GetRequestID(r.Context()))
		apierrors.WriteError(w, apiErr)
		return
	}

	trends := make(map[string]TrendAnalysis)

	// Analyze key metrics
	keyMetrics := []string{
		"queue.total_size",
		"queue.error_rate_percent",
		"health.avg_response_time_ms",
		"discovery.total_targets",
	}

	// Get overall trend analysis
	trendAnalysis := h.trendDetector.Analyze()

	// Map to our response format
	for _, metricName := range keyMetrics {
		if current, ok := snapshot.Metrics[metricName]; ok {
			// Use trend analysis data
			trends[metricName] = TrendAnalysis{
				MetricName: metricName,
				Current:    current,
				Trend:      trendAnalysis.SuccessRateTrend, // Simplified for now
				ChangeRate: 0.0,                            // TODO: Calculate per-metric change rate
				Severity:   "normal",                       // TODO: Determine severity from analysis
			}
		}
	}

	response := TrendResponse{
		Timestamp: snapshot.Timestamp,
		Trends:    trends,
	}

	h.sendJSON(w, http.StatusOK, response)
}

// ===== Helper Methods =====

func (h *MetricsHandlers) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(middleware.APIVersionHeader, "2.0.0")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response", "error", err)
	}
}
