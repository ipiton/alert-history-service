package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics contains all business-level metrics for Alert History Service.
//
// Business metrics track high-level business operations:
//   - Alert processing (how many alerts processed, enriched, filtered)
//   - LLM classification (confidence scores, recommendations)
//   - Publishing (success/failure rates, latency)
//
// All metrics follow the taxonomy:
// alert_history_business_<subsystem>_<metric_name>_<unit>
//
// Example:
//
//	bm := NewBusinessMetrics("alert_history")
//	bm.AlertsProcessedTotal.Inc()
//	bm.LLMConfidenceScore.Observe(0.95)
type BusinessMetrics struct {
	namespace string

	// Alerts subsystem - alert processing metrics
	AlertsProcessedTotal *prometheus.CounterVec // Total alerts processed by the system
	AlertsEnrichedTotal  *prometheus.CounterVec // Total alerts enriched with LLM data
	AlertsFilteredTotal  *prometheus.CounterVec // Total alerts filtered (allowed/blocked)

	// Deduplication subsystem - alert deduplication metrics (TN-036)
	DeduplicationCreatedTotal    *prometheus.CounterVec   // New alerts created (not duplicates)
	DeduplicationUpdatedTotal    *prometheus.CounterVec   // Existing alerts updated
	DeduplicationIgnoredTotal    *prometheus.CounterVec   // Duplicate alerts ignored
	DeduplicationDurationSeconds *prometheus.HistogramVec // Deduplication operation duration

	// LLM subsystem - AI classification metrics
	LLMClassificationsTotal *prometheus.CounterVec   // Total LLM classifications performed
	LLMRecommendationsTotal prometheus.Counter      // Total LLM recommendations generated
	LLMConfidenceScore      prometheus.Histogram    // Distribution of LLM confidence scores
	ClassificationL1CacheHitsTotal prometheus.Counter   // L1 (memory) cache hits for classifications
	ClassificationL2CacheHitsTotal prometheus.Counter   // L2 (Redis) cache hits for classifications
	ClassificationDurationSeconds   *prometheus.HistogramVec // Classification operation duration

	// Publishing subsystem - alert delivery metrics
	PublishingSuccessTotal    *prometheus.CounterVec   // Successful alert publishes
	PublishingFailedTotal     *prometheus.CounterVec   // Failed alert publishes
	PublishingDurationSeconds *prometheus.HistogramVec // Publishing operation duration

	// Grouping subsystem - alert group management metrics (TN-123)
	AlertGroupsActiveTotal       prometheus.Gauge           // Number of currently active alert groups
	AlertGroupSize               prometheus.Histogram       // Distribution of alert group sizes
	AlertGroupOperationsTotal    *prometheus.CounterVec    // Total number of group operations (add/remove/cleanup)
	AlertGroupOperationDurationSeconds *prometheus.HistogramVec // Duration of group operations

	// Timers subsystem - group timer metrics (TN-124)
	TimersActiveTotal         *prometheus.GaugeVec      // Number of currently active timers by type
	TimersExpiredTotal        *prometheus.CounterVec    // Total number of expired timers by type
	TimerDurationSeconds      *prometheus.HistogramVec  // Distribution of timer durations by type
	TimerResetsTotal          *prometheus.CounterVec    // Total number of timer resets by type
	TimersRestoredTotal       prometheus.Counter        // Total number of timers restored after restart
	TimersMissedTotal         prometheus.Counter        // Total number of timers missed due to downtime
	TimerOperationDurationSeconds *prometheus.HistogramVec // Duration of timer operations

	// Storage subsystem - group storage metrics (TN-125)
	StorageFallbackTotal      *prometheus.CounterVec    // Total storage fallback events by reason
	StorageRecoveryTotal      prometheus.Counter        // Total storage recovery events
	GroupsRestoredTotal       prometheus.Counter        // Total groups restored from storage on startup
	StorageOperationsTotal    *prometheus.CounterVec    // Total storage operations (store/load/delete)
	StorageDurationSeconds    *prometheus.HistogramVec  // Duration of storage operations
	StorageHealthGauge        *prometheus.GaugeVec      // Storage health status by backend (1=healthy, 0=unhealthy)
}

// NewBusinessMetrics creates a new BusinessMetrics instance with standard configuration.
//
// Parameters:
//   - namespace: The Prometheus namespace (typically "alert_history")
//
// Returns:
//   - *BusinessMetrics: Initialized business metrics manager
func NewBusinessMetrics(namespace string) *BusinessMetrics {
	return &BusinessMetrics{
		namespace: namespace,

		// Alerts subsystem metrics
		AlertsProcessedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_alerts",
				Name:      "processed_total",
				Help:      "Total number of alerts processed by the system",
			},
			[]string{"source"}, // source: alertmanager, webhook, api
		),

		AlertsEnrichedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_alerts",
				Name:      "enriched_total",
				Help:      "Total number of alerts enriched with LLM data",
			},
			[]string{"mode", "status"}, // mode: enriched|transparent_recommendations, status: success|failure
		),

		AlertsFilteredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_alerts",
				Name:      "filtered_total",
				Help:      "Total number of alerts filtered (allowed or blocked)",
			},
			[]string{"result", "reason"}, // result: allowed|blocked, reason: test_alert|noise|low_confidence
		),

		// Deduplication subsystem metrics (TN-036)
		DeduplicationCreatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_deduplication",
				Name:      "created_total",
				Help:      "Total number of new alerts created (not duplicates)",
			},
			[]string{"source"}, // source: webhook, api
		),

		DeduplicationUpdatedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_deduplication",
				Name:      "updated_total",
				Help:      "Total number of existing alerts updated (status changes)",
			},
			[]string{"status_from", "status_to"}, // status transitions: firing->resolved, resolved->firing
		),

		DeduplicationIgnoredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_deduplication",
				Name:      "ignored_total",
				Help:      "Total number of duplicate alerts ignored (already exists with same data)",
			},
			[]string{"reason"}, // reason: duplicate, unchanged
		),

		DeduplicationDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_deduplication",
				Name:      "duration_seconds",
				Help:      "Duration of deduplication operations (fingerprint generation + lookup)",
				Buckets:   []float64{0.000001, 0.000005, 0.00001, 0.00005, 0.0001, 0.0005, 0.001, 0.005, 0.01}, // 1µs to 10ms
			},
			[]string{"action"}, // action: created, updated, ignored
		),

		// LLM subsystem metrics
		LLMClassificationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_llm",
				Name:      "classifications_total",
				Help:      "Total number of LLM classifications performed",
			},
			[]string{"severity", "confidence"}, // severity: critical|warning|info, confidence: high|medium|low
		),

		LLMRecommendationsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_llm",
				Name:      "recommendations_total",
				Help:      "Total number of LLM recommendations generated",
			},
		),

		LLMConfidenceScore: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_llm",
				Name:      "confidence_score",
				Help:      "Distribution of LLM confidence scores (0.0 to 1.0)",
				// Buckets optimized for confidence scores: 0.5 to 0.99
				Buckets: []float64{0.5, 0.6, 0.7, 0.8, 0.85, 0.9, 0.95, 0.99},
			},
		),

		// Classification caching metrics (TN-033)
		ClassificationL1CacheHitsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_classification",
				Name:      "l1_cache_hits_total",
				Help:      "Total number of L1 (memory) cache hits for classifications",
			},
		),

		ClassificationL2CacheHitsTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_classification",
				Name:      "l2_cache_hits_total",
				Help:      "Total number of L2 (Redis) cache hits for classifications",
			},
		),

		ClassificationDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_classification",
				Name:      "duration_seconds",
				Help:      "Duration of classification operations in seconds",
				// Buckets optimized for classification: 1ms to 10s
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0, 10.0},
			},
			[]string{"source"}, // source: llm|fallback|cache
		),

		// Publishing subsystem metrics
		PublishingSuccessTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_publishing",
				Name:      "success_total",
				Help:      "Total number of successful alert publishes",
			},
			[]string{"destination"}, // destination: webhook|slack|pagerduty|etc
		),

		PublishingFailedTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_publishing",
				Name:      "failed_total",
				Help:      "Total number of failed alert publishes",
			},
			[]string{"destination", "error_type"}, // error_type: timeout|connection_refused|4xx|5xx
		),

		PublishingDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_publishing",
				Name:      "duration_seconds",
				Help:      "Duration of publishing operations in seconds",
				// Buckets optimized for webhook/API calls: 10ms to 5s
				Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
			},
			[]string{"destination"}, // destination: webhook|slack|pagerduty|etc
		),

		// Grouping subsystem metrics (TN-123)
		AlertGroupsActiveTotal: promauto.NewGauge(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "alert_groups_active_total",
				Help:      "Number of currently active alert groups",
			},
		),

		AlertGroupSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "alert_group_size",
				Help:      "Distribution of alert group sizes",
				// Buckets optimized for group sizes: 1 to 1000+
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
		),

		AlertGroupOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "alert_group_operations_total",
				Help:      "Total number of group operations",
			},
			[]string{"operation", "result"}, // operation: add|remove|cleanup, result: success|error
		),

		AlertGroupOperationDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "alert_group_operation_duration_seconds",
				Help:      "Duration of group operations",
				// Buckets optimized for group operations: 100µs to 100ms
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
			},
			[]string{"operation"}, // operation: add|remove|cleanup|get|list
		),

		// Timers subsystem metrics (TN-124)
		TimersActiveTotal: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timers_active_total",
				Help:      "Number of currently active timers by type",
			},
			[]string{"type"}, // type: group_wait|group_interval|repeat_interval
		),

		TimersExpiredTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timers_expired_total",
				Help:      "Total number of expired timers by type",
			},
			[]string{"type"}, // type: group_wait|group_interval|repeat_interval
		),

		TimerDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timer_duration_seconds",
				Help:      "Distribution of timer durations by type",
				// Buckets optimized for timers: 1s to 4h
				Buckets: []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600, 14400},
			},
			[]string{"type"}, // type: group_wait|group_interval|repeat_interval
		),

		TimerResetsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timer_resets_total",
				Help:      "Total number of timer resets by type",
			},
			[]string{"type"}, // type: group_wait|group_interval|repeat_interval
		),

		TimersRestoredTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timers_restored_total",
				Help:      "Total number of timers restored after restart (HA recovery)",
			},
		),

		TimersMissedTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timers_missed_total",
				Help:      "Total number of timers missed due to service downtime",
			},
		),

		TimerOperationDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "timer_operation_duration_seconds",
				Help:      "Duration of timer operations",
				// Buckets optimized for timer operations: 100µs to 100ms
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05, 0.1},
			},
			[]string{"operation"}, // operation: start|cancel|reset|restore
		),

		// Storage subsystem metrics (TN-125)
		StorageFallbackTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "storage_fallback_total",
				Help:      "Total number of storage fallback events (Redis → Memory)",
			},
			[]string{"reason"}, // reason: health_check_failed|store_error|delete_error|store_all_error
		),

		StorageRecoveryTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "storage_recovery_total",
				Help:      "Total number of storage recovery events (Memory → Redis)",
			},
		),

		GroupsRestoredTotal: promauto.NewCounter(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "groups_restored_total",
				Help:      "Total number of groups restored from storage on startup (HA recovery)",
			},
		),

		StorageOperationsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "storage_operations_total",
				Help:      "Total number of group storage operations",
			},
			[]string{"operation", "result"}, // operation: store|load|delete|load_all, result: success|error
		),

		StorageDurationSeconds: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "storage_duration_seconds",
				Help:      "Duration of group storage operations",
				// Buckets optimized for storage operations: 100µs to 100ms
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.010, 0.050, 0.100},
			},
			[]string{"operation"}, // operation: store|load|delete|load_all|store_all
		),

		StorageHealthGauge: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "business_grouping",
				Name:      "storage_health",
				Help:      "Storage health status by backend (1=healthy, 0=unhealthy)",
			},
			[]string{"backend"}, // backend: redis|memory
		),
	}
}

// RecordAlertProcessed records an alert being processed.
//
// Parameters:
//   - source: The source of the alert (e.g., "alertmanager", "webhook")
func (m *BusinessMetrics) RecordAlertProcessed(source string) {
	m.AlertsProcessedTotal.WithLabelValues(source).Inc()
}

// RecordAlertEnriched records an alert enrichment operation.
//
// Parameters:
//   - mode: The enrichment mode (e.g., "enriched", "transparent_recommendations")
//   - success: Whether the enrichment was successful
func (m *BusinessMetrics) RecordAlertEnriched(mode string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	m.AlertsEnrichedTotal.WithLabelValues(mode, status).Inc()
}

// RecordAlertFiltered records an alert being filtered.
//
// Parameters:
//   - result: The filter result ("allowed" or "blocked")
//   - reason: The reason for the decision (e.g., "test_alert", "noise")
func (m *BusinessMetrics) RecordAlertFiltered(result, reason string) {
	m.AlertsFilteredTotal.WithLabelValues(result, reason).Inc()
}

// RecordLLMClassification records an LLM classification.
//
// Parameters:
//   - severity: The classified severity (e.g., "critical", "warning")
//   - confidence: The confidence level (e.g., "high", "medium", "low")
func (m *BusinessMetrics) RecordLLMClassification(severity, confidence string) {
	m.LLMClassificationsTotal.WithLabelValues(severity, confidence).Inc()
}

// RecordLLMRecommendation records an LLM recommendation being generated.
func (m *BusinessMetrics) RecordLLMRecommendation() {
	m.LLMRecommendationsTotal.Inc()
}

// RecordLLMConfidenceScore records an LLM confidence score.
//
// Parameters:
//   - score: The confidence score (0.0 to 1.0)
func (m *BusinessMetrics) RecordLLMConfidenceScore(score float64) {
	m.LLMConfidenceScore.Observe(score)
}

// RecordClassificationL1CacheHit records an L1 (memory) cache hit for classification.
func (m *BusinessMetrics) RecordClassificationL1CacheHit() {
	m.ClassificationL1CacheHitsTotal.Inc()
}

// RecordClassificationL2CacheHit records an L2 (Redis) cache hit for classification.
func (m *BusinessMetrics) RecordClassificationL2CacheHit() {
	m.ClassificationL2CacheHitsTotal.Inc()
}

// RecordClassificationDuration records the duration of a classification operation.
//
// Parameters:
//   - source: The classification source ("llm", "fallback", "cache")
//   - duration: The operation duration in seconds
func (m *BusinessMetrics) RecordClassificationDuration(source string, duration float64) {
	m.ClassificationDurationSeconds.WithLabelValues(source).Observe(duration)
}

// RecordPublishingSuccess records a successful alert publish.
//
// Parameters:
//   - destination: The destination (e.g., "webhook", "slack")
//   - duration: The operation duration in seconds
func (m *BusinessMetrics) RecordPublishingSuccess(destination string, duration float64) {
	m.PublishingSuccessTotal.WithLabelValues(destination).Inc()
	m.PublishingDurationSeconds.WithLabelValues(destination).Observe(duration)
}

// RecordPublishingFailure records a failed alert publish.
//
// Parameters:
//   - destination: The destination (e.g., "webhook", "slack")
//   - errorType: The error type (e.g., "timeout", "4xx")
//   - duration: The operation duration in seconds
func (m *BusinessMetrics) RecordPublishingFailure(destination, errorType string, duration float64) {
	m.PublishingFailedTotal.WithLabelValues(destination, errorType).Inc()
	m.PublishingDurationSeconds.WithLabelValues(destination).Observe(duration)
}

// === Grouping Subsystem Methods (TN-123) ===

// IncActiveGroups increments the active groups gauge by 1.
// Called when a new group is created.
func (m *BusinessMetrics) IncActiveGroups() {
	m.AlertGroupsActiveTotal.Inc()
}

// DecActiveGroups decrements the active groups gauge by 1.
// Called when a group is deleted.
func (m *BusinessMetrics) DecActiveGroups() {
	m.AlertGroupsActiveTotal.Dec()
}

// RecordGroupSize records the size of an alert group.
//
// Parameters:
//   - size: Number of alerts in the group
func (m *BusinessMetrics) RecordGroupSize(size int) {
	m.AlertGroupSize.Observe(float64(size))
}

// RecordGroupOperation records a group management operation.
//
// Parameters:
//   - operation: The operation type ("add", "remove", "cleanup")
//   - result: The result ("success" or "error")
func (m *BusinessMetrics) RecordGroupOperation(operation, result string) {
	m.AlertGroupOperationsTotal.WithLabelValues(operation, result).Inc()
}

// RecordGroupOperationDuration records the duration of a group operation.
//
// Parameters:
//   - operation: The operation type ("add", "remove", "cleanup", "get", "list")
//   - duration: The operation duration
func (m *BusinessMetrics) RecordGroupOperationDuration(operation string, duration time.Duration) {
	m.AlertGroupOperationDurationSeconds.WithLabelValues(operation).Observe(duration.Seconds())
}

// === Timer Subsystem Methods (TN-124) ===

// RecordTimerStarted records a timer being started.
//
// Parameters:
//   - timerType: The type of timer (e.g., "group_wait", "group_interval", "repeat_interval")
func (m *BusinessMetrics) RecordTimerStarted(timerType string) {
	// Active count managed via IncActiveTimers/DecActiveTimers
}

// RecordTimerExpired records a timer expiration.
//
// Parameters:
//   - timerType: The type of timer that expired
func (m *BusinessMetrics) RecordTimerExpired(timerType string) {
	m.TimersExpiredTotal.WithLabelValues(timerType).Inc()
}

// RecordTimerCancelled records a timer being cancelled.
//
// Parameters:
//   - timerType: The type of timer that was cancelled
func (m *BusinessMetrics) RecordTimerCancelled(timerType string) {
	// Recorded via DecActiveTimers
}

// RecordTimerReset records a timer being reset.
//
// Parameters:
//   - timerType: The type of timer that was reset
func (m *BusinessMetrics) RecordTimerReset(timerType string) {
	m.TimerResetsTotal.WithLabelValues(timerType).Inc()
}

// RecordTimerDuration records the duration of a timer.
//
// Parameters:
//   - timerType: The type of timer
//   - duration: The timer duration
func (m *BusinessMetrics) RecordTimerDuration(timerType string, duration time.Duration) {
	m.TimerDurationSeconds.WithLabelValues(timerType).Observe(duration.Seconds())
}

// IncActiveTimers increments the count of active timers.
//
// Parameters:
//   - timerType: The type of timer
func (m *BusinessMetrics) IncActiveTimers(timerType string) {
	m.TimersActiveTotal.WithLabelValues(timerType).Inc()
}

// DecActiveTimers decrements the count of active timers.
//
// Parameters:
//   - timerType: The type of timer
func (m *BusinessMetrics) DecActiveTimers(timerType string) {
	m.TimersActiveTotal.WithLabelValues(timerType).Dec()
}

// RecordTimersRestored records the number of timers restored after restart.
//
// Parameters:
//   - count: Number of timers restored
func (m *BusinessMetrics) RecordTimersRestored(count int) {
	m.TimersRestoredTotal.Add(float64(count))
}

// RecordTimersMissed records the number of timers missed due to downtime.
//
// Parameters:
//   - count: Number of timers missed
func (m *BusinessMetrics) RecordTimersMissed(count int) {
	m.TimersMissedTotal.Add(float64(count))
}

// === Storage Subsystem Methods (TN-125) ===

// IncStorageFallback increments the storage fallback counter.
// Called when switching from primary (Redis) to fallback (Memory) storage.
//
// Parameters:
//   - reason: Reason for fallback ("health_check_failed", "store_error", "delete_error", "store_all_error")
func (m *BusinessMetrics) IncStorageFallback(reason string) {
	m.StorageFallbackTotal.WithLabelValues(reason).Inc()
}

// IncStorageRecovery increments the storage recovery counter.
// Called when switching back from fallback (Memory) to primary (Redis) storage.
func (m *BusinessMetrics) IncStorageRecovery() {
	m.StorageRecoveryTotal.Inc()
}

// RecordGroupsRestored records the number of groups restored from storage on startup.
// Called after successful LoadAll() during HA recovery.
//
// Parameters:
//   - count: Number of groups restored
func (m *BusinessMetrics) RecordGroupsRestored(count int) {
	m.GroupsRestoredTotal.Add(float64(count))
}

// RecordStorageOperation records a storage operation.
//
// Parameters:
//   - operation: The operation type ("store", "load", "delete", "load_all", "store_all")
//   - result: The result ("success" or "error")
func (m *BusinessMetrics) RecordStorageOperation(operation, result string) {
	m.StorageOperationsTotal.WithLabelValues(operation, result).Inc()
}

// RecordStorageDuration records the duration of a storage operation.
//
// Parameters:
//   - operation: The operation type ("store", "load", "delete", "load_all", "store_all")
//   - duration: The operation duration
func (m *BusinessMetrics) RecordStorageDuration(operation string, duration time.Duration) {
	m.StorageDurationSeconds.WithLabelValues(operation).Observe(duration.Seconds())
}

// SetStorageHealth sets the storage health status.
//
// Parameters:
//   - backend: The storage backend ("redis" or "memory")
//   - healthy: Whether the backend is healthy (true=1, false=0)
func (m *BusinessMetrics) SetStorageHealth(backend string, healthy bool) {
	value := 0.0
	if healthy {
		value = 1.0
	}
	m.StorageHealthGauge.WithLabelValues(backend).Set(value)
}

// RecordTimerOperationDuration records the duration of a timer operation.
//
// Parameters:
//   - operation: The operation name (e.g., "start", "cancel", "reset", "restore")
//   - duration: The duration of the operation
func (m *BusinessMetrics) RecordTimerOperationDuration(operation string, duration time.Duration) {
	m.TimerOperationDurationSeconds.WithLabelValues(operation).Observe(duration.Seconds())
}

// RecordGroupsCleanedUp records the number of groups cleaned up in a cleanup operation.
//
// Parameters:
//   - count: Number of groups deleted
func (m *BusinessMetrics) RecordGroupsCleanedUp(count int) {
	// Record as multiple increments for the counter
	// Note: Consider adding a separate metric for cleanup batch size if needed
	for i := 0; i < count; i++ {
		m.AlertGroupOperationsTotal.WithLabelValues("cleanup", "success").Inc()
	}
}
