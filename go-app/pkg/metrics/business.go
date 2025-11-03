package metrics

import (
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
