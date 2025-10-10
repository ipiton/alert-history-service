package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	webhookMetricsInstance *WebhookMetrics
	webhookMetricsOnce     sync.Once
)

// WebhookMetrics contains webhook processing metrics for Alert History Service.
//
// Webhook metrics track webhook request handling, parsing, validation, and processing:
//   - Request rates and success/failure by webhook type
//   - Request and processing duration histograms
//   - Queue size and worker pool status (for async processing)
//   - Error rates by type and error category
//   - Payload size distribution
//
// All metrics follow the unified taxonomy:
// alert_history_technical_webhook_<metric_name>_<unit>
//
// Example:
//
//	wm := NewWebhookMetrics()
//	wm.RecordRequest("alertmanager", "success", 0.045)
//	wm.RecordProcessingStage("alertmanager", "parse", 0.002)
//	wm.RecordError("alertmanager", "validation_error")
type WebhookMetrics struct {
	// RequestsTotal tracks total webhook requests by type and status.
	// Labels: type (alertmanager, generic, prometheus), status (success, failure)
	RequestsTotal *prometheus.CounterVec

	// DurationSeconds tracks end-to-end webhook request duration.
	// Labels: type (alertmanager, generic, prometheus)
	// Buckets: 10ms to 5s (optimized for webhook processing)
	DurationSeconds *prometheus.HistogramVec

	// ProcessingSeconds tracks processing time by stage.
	// Labels: type, stage (parse, validate, convert, process)
	// Buckets: 100µs to 1s (optimized for individual stages)
	ProcessingSeconds *prometheus.HistogramVec

	// QueueSize tracks current webhook processing queue size.
	// For async processing (TN-044), indicates backlog.
	QueueSize prometheus.Gauge

	// ActiveWorkers tracks currently active webhook workers.
	// For async processing (TN-044), indicates worker pool utilization.
	ActiveWorkers prometheus.Gauge

	// ErrorsTotal tracks webhook errors by type and error category.
	// Labels: type, error_type (parse_error, validation_error, processing_error, timeout)
	ErrorsTotal *prometheus.CounterVec

	// PayloadSizeBytes tracks webhook payload size distribution.
	// Labels: type
	// Buckets: 1KB to 1MB (typical alert payload sizes)
	PayloadSizeBytes *prometheus.HistogramVec
}

// NewWebhookMetrics creates or returns the singleton WebhookMetrics instance.
//
// This function uses sync.Once to ensure metrics are registered only once,
// preventing "duplicate metrics collector registration" errors when called
// multiple times (e.g., in tests or when multiple components initialize).
//
// All metrics use the namespace "alert_history" and subsystem "technical_webhook"
// following the unified taxonomy from TN-181.
//
// Returns:
//   - *WebhookMetrics: Initialized webhook metrics (singleton)
func NewWebhookMetrics() *WebhookMetrics {
	webhookMetricsOnce.Do(func() {
		webhookMetricsInstance = &WebhookMetrics{
			RequestsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "requests_total",
					Help:      "Total number of webhook requests by type and status",
				},
				[]string{"type", "status"}, // type: alertmanager|generic|prometheus, status: success|failure
			),

			DurationSeconds: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "duration_seconds",
					Help:      "Webhook request duration in seconds",
					// Buckets optimized for webhook processing: 10ms to 5s
					Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
				},
				[]string{"type"}, // type: alertmanager|generic|prometheus
			),

			ProcessingSeconds: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "processing_seconds",
					Help:      "Webhook processing time by stage in seconds",
					// Buckets optimized for individual stages: 100µs to 1s
					Buckets: []float64{0.0001, 0.0005, 0.001, 0.0025, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
				},
				[]string{"type", "stage"}, // stage: parse|validate|convert|process
			),

			QueueSize: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "queue_size",
					Help:      "Current webhook processing queue size (async mode)",
				},
			),

			ActiveWorkers: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "active_workers",
					Help:      "Currently active webhook workers (async mode)",
				},
			),

			ErrorsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "errors_total",
					Help:      "Total webhook errors by type and error category",
				},
				[]string{"type", "error_type"}, // error_type: parse_error|validation_error|processing_error|timeout
			),

			PayloadSizeBytes: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "alert_history",
					Subsystem: "technical_webhook",
					Name:      "payload_size_bytes",
					Help:      "Webhook payload size distribution in bytes",
					// Buckets optimized for alert payloads: 1KB to 1MB
					Buckets: []float64{1024, 5120, 10240, 51200, 102400, 512000, 1048576},
				},
				[]string{"type"},
			),
		}
	})
	return webhookMetricsInstance
}

// RecordRequest records a webhook request with its outcome.
//
// Parameters:
//   - webhookType: The webhook type (alertmanager, generic, prometheus)
//   - status: Request status (success, failure)
//   - durationSeconds: Total request duration in seconds
func (m *WebhookMetrics) RecordRequest(webhookType, status string, durationSeconds float64) {
	m.RequestsTotal.WithLabelValues(webhookType, status).Inc()
	m.DurationSeconds.WithLabelValues(webhookType).Observe(durationSeconds)
}

// RecordProcessingStage records processing time for a specific stage.
//
// Parameters:
//   - webhookType: The webhook type
//   - stage: Processing stage (parse, validate, convert, process)
//   - durationSeconds: Stage duration in seconds
func (m *WebhookMetrics) RecordProcessingStage(webhookType, stage string, durationSeconds float64) {
	m.ProcessingSeconds.WithLabelValues(webhookType, stage).Observe(durationSeconds)
}

// RecordError records a webhook error.
//
// Parameters:
//   - webhookType: The webhook type
//   - errorType: Error category (parse_error, validation_error, processing_error, timeout)
func (m *WebhookMetrics) RecordError(webhookType, errorType string) {
	m.ErrorsTotal.WithLabelValues(webhookType, errorType).Inc()
}

// RecordPayloadSize records webhook payload size.
//
// Parameters:
//   - webhookType: The webhook type
//   - sizeBytes: Payload size in bytes
func (m *WebhookMetrics) RecordPayloadSize(webhookType string, sizeBytes int) {
	m.PayloadSizeBytes.WithLabelValues(webhookType).Observe(float64(sizeBytes))
}

// SetQueueSize sets the current queue size (for async processing).
//
// Parameters:
//   - size: Current queue size
func (m *WebhookMetrics) SetQueueSize(size int) {
	m.QueueSize.Set(float64(size))
}

// SetActiveWorkers sets the number of active workers (for async processing).
//
// Parameters:
//   - count: Number of active workers
func (m *WebhookMetrics) SetActiveWorkers(count int) {
	m.ActiveWorkers.Set(float64(count))
}

// IncrementActiveWorkers increments the active workers gauge.
func (m *WebhookMetrics) IncrementActiveWorkers() {
	m.ActiveWorkers.Inc()
}

// DecrementActiveWorkers decrements the active workers gauge.
func (m *WebhookMetrics) DecrementActiveWorkers() {
	m.ActiveWorkers.Dec()
}
