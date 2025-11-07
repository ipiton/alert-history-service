package publishing

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PublishingMetrics holds all Prometheus metrics for publishing system
type PublishingMetrics struct {
	// Alerts metrics
	AlertsPublished prometheus.Counter
	AlertsPublishErrors prometheus.Counter

	// Per-target metrics
	AlertsPublishedByTarget *prometheus.CounterVec
	PublishErrorsByTarget   *prometheus.CounterVec
	PublishDurationByTarget *prometheus.HistogramVec

	// Queue metrics
	QueueSize             prometheus.Gauge
	QueueCapacity         prometheus.Gauge
	QueueSubmissions      prometheus.Counter
	QueueSubmissionErrors prometheus.Counter
	QueueProcessingTime   prometheus.Histogram

	// Circuit breaker metrics
	CircuitBreakerState *prometheus.GaugeVec
	CircuitBreakerTrips *prometheus.CounterVec

	// Target discovery metrics
	DiscoveredTargets       prometheus.Gauge
	EnabledTargets          prometheus.Gauge
	TargetDiscoveryDuration prometheus.Histogram
	TargetRefreshes         prometheus.Counter
	TargetRefreshErrors     prometheus.Counter

	// Formatter metrics
	FormatDuration *prometheus.HistogramVec
	FormatErrors   *prometheus.CounterVec
}

// NewPublishingMetrics creates and registers all publishing metrics
func NewPublishingMetrics() *PublishingMetrics {
	return &PublishingMetrics{
		// Alert metrics
		AlertsPublished: promauto.NewCounter(prometheus.CounterOpts{
			Name: "publishing_alerts_published_total",
			Help: "Total number of alerts successfully published",
		}),
		AlertsPublishErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "publishing_alerts_errors_total",
			Help: "Total number of alert publishing errors",
		}),

		// Per-target metrics
		AlertsPublishedByTarget: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "publishing_alerts_published_by_target_total",
				Help: "Total number of alerts published per target",
			},
			[]string{"target_name", "target_type"},
		),
		PublishErrorsByTarget: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "publishing_errors_by_target_total",
				Help: "Total number of publishing errors per target",
			},
			[]string{"target_name", "target_type", "error_type"},
		),
		PublishDurationByTarget: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "publishing_duration_seconds",
				Help:    "Duration of publishing operation per target",
				Buckets: prometheus.DefBuckets, // 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10
			},
			[]string{"target_name", "target_type"},
		),

		// Queue metrics
		QueueSize: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "publishing_queue_size",
			Help: "Current number of jobs in the publishing queue",
		}),
		QueueCapacity: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "publishing_queue_capacity",
			Help: "Maximum capacity of the publishing queue",
		}),
		QueueSubmissions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "publishing_queue_submissions_total",
			Help: "Total number of jobs submitted to the queue",
		}),
		QueueSubmissionErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "publishing_queue_submission_errors_total",
			Help: "Total number of queue submission errors (queue full, etc)",
		}),
		QueueProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "publishing_queue_processing_seconds",
			Help:    "Time spent processing jobs in the queue",
			Buckets: prometheus.DefBuckets,
		}),

		// Circuit breaker metrics
		CircuitBreakerState: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "publishing_circuit_breaker_state",
				Help: "Circuit breaker state per target (0=closed, 1=open, 2=half-open)",
			},
			[]string{"target_name"},
		),
		CircuitBreakerTrips: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "publishing_circuit_breaker_trips_total",
				Help: "Total number of circuit breaker trips per target",
			},
			[]string{"target_name"},
		),

		// Target discovery metrics
		DiscoveredTargets: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "publishing_discovered_targets",
			Help: "Current number of discovered publishing targets",
		}),
		EnabledTargets: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "publishing_enabled_targets",
			Help: "Current number of enabled publishing targets",
		}),
		TargetDiscoveryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "publishing_target_discovery_duration_seconds",
			Help:    "Duration of target discovery operations",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10}, // K8s API calls can be slower
		}),
		TargetRefreshes: promauto.NewCounter(prometheus.CounterOpts{
			Name: "publishing_target_refreshes_total",
			Help: "Total number of target refresh operations",
		}),
		TargetRefreshErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "publishing_target_refresh_errors_total",
			Help: "Total number of target refresh errors",
		}),

		// Formatter metrics
		FormatDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "publishing_format_duration_seconds",
				Help:    "Duration of alert formatting operations",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.05}, // Formatting should be fast
			},
			[]string{"format"},
		),
		FormatErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "publishing_format_errors_total",
				Help: "Total number of formatting errors",
			},
			[]string{"format"},
		),
	}
}

// RecordPublishSuccess records a successful publish to a target
func (m *PublishingMetrics) RecordPublishSuccess(targetName, targetType string, duration float64) {
	m.AlertsPublished.Inc()
	m.AlertsPublishedByTarget.WithLabelValues(targetName, targetType).Inc()
	m.PublishDurationByTarget.WithLabelValues(targetName, targetType).Observe(duration)
}

// RecordPublishError records a failed publish to a target
func (m *PublishingMetrics) RecordPublishError(targetName, targetType, errorType string) {
	m.AlertsPublishErrors.Inc()
	m.PublishErrorsByTarget.WithLabelValues(targetName, targetType, errorType).Inc()
}

// RecordQueueSubmission records a queue submission
func (m *PublishingMetrics) RecordQueueSubmission(success bool) {
	m.QueueSubmissions.Inc()
	if !success {
		m.QueueSubmissionErrors.Inc()
	}
}

// UpdateQueueMetrics updates queue size and capacity gauges
func (m *PublishingMetrics) UpdateQueueMetrics(size, capacity int) {
	m.QueueSize.Set(float64(size))
	m.QueueCapacity.Set(float64(capacity))
}

// RecordCircuitBreakerState updates circuit breaker state
func (m *PublishingMetrics) RecordCircuitBreakerState(targetName string, state CircuitBreakerState) {
	m.CircuitBreakerState.WithLabelValues(targetName).Set(float64(state))
}

// RecordCircuitBreakerTrip records a circuit breaker trip
func (m *PublishingMetrics) RecordCircuitBreakerTrip(targetName string) {
	m.CircuitBreakerTrips.WithLabelValues(targetName).Inc()
}

// UpdateTargetCounts updates target discovery counts
func (m *PublishingMetrics) UpdateTargetCounts(discovered, enabled int) {
	m.DiscoveredTargets.Set(float64(discovered))
	m.EnabledTargets.Set(float64(enabled))
}

// RecordTargetRefresh records a target refresh operation
func (m *PublishingMetrics) RecordTargetRefresh(success bool, duration float64) {
	m.TargetRefreshes.Inc()
	m.TargetDiscoveryDuration.Observe(duration)
	if !success {
		m.TargetRefreshErrors.Inc()
	}
}

// RecordFormatOperation records an alert formatting operation
func (m *PublishingMetrics) RecordFormatOperation(format string, success bool, duration float64) {
	m.FormatDuration.WithLabelValues(format).Observe(duration)
	if !success {
		m.FormatErrors.WithLabelValues(format).Inc()
	}
}
