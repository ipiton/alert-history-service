// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PrometheusAlertsMetrics tracks metrics for /api/v2/alerts endpoint.
//
// This metrics collector provides comprehensive observability for the
// Prometheus alerts endpoint, tracking request patterns, processing results,
// errors, and performance characteristics.
//
// Metrics Categories:
//   - HTTP Request metrics: Total requests, duration, status
//   - Alert Processing metrics: Alerts received/processed by format
//   - Error metrics: Validation failures, processing errors
//   - Performance metrics: Concurrent requests, payload size
//
// All metrics are prefixed with "alert_history_prometheus_alerts_" to
// ensure uniqueness in the Prometheus namespace.
//
// Example Usage:
//   metrics := NewPrometheusAlertsMetrics()
//   metrics.RecordRequest("success", 10, 5*time.Millisecond)
//   metrics.RecordAlerts("v1", 10, 10, 0)
//
// Grafana Integration:
//   See design.md for PromQL query examples and dashboard configuration.
type PrometheusAlertsMetrics struct {
	// HTTP request metrics
	requestsTotal   *prometheus.CounterVec   // Total HTTP requests by status
	requestDuration *prometheus.HistogramVec // Request duration by status

	// Alert processing metrics
	alertsReceived  *prometheus.CounterVec // Alerts received by format (v1/v2)
	alertsProcessed *prometheus.CounterVec // Alerts processed by status (success/failed)

	// Error metrics
	validationErrors *prometheus.CounterVec // Validation failures by reason
	processingErrors *prometheus.CounterVec // Processing errors by type

	// Performance metrics
	concurrentReqs prometheus.Gauge     // Current concurrent requests
	payloadSize    prometheus.Histogram // Request payload size distribution
}

// NewPrometheusAlertsMetrics initializes metrics for /api/v2/alerts endpoint.
//
// Creates and registers 8 Prometheus metrics:
//   1. alert_history_prometheus_alerts_requests_total (CounterVec)
//   2. alert_history_prometheus_alerts_duration_seconds (HistogramVec)
//   3. alert_history_prometheus_alerts_received_total (CounterVec)
//   4. alert_history_prometheus_alerts_processed_total (CounterVec)
//   5. alert_history_prometheus_alerts_validation_errors_total (CounterVec)
//   6. alert_history_prometheus_alerts_processing_errors_total (CounterVec)
//   7. alert_history_prometheus_alerts_concurrent_requests (Gauge)
//   8. alert_history_prometheus_alerts_payload_bytes (Histogram)
//
// Metrics are automatically registered with the default Prometheus registry.
// Uses promauto for automatic registration (panic on duplicate registration).
//
// Returns:
//   - *PrometheusAlertsMetrics: Initialized metrics collector
//
// Panics:
//   - If metrics are already registered (duplicate call)
func NewPrometheusAlertsMetrics() *PrometheusAlertsMetrics {
	return &PrometheusAlertsMetrics{
		// Metric 1: HTTP request counter by status
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_prometheus_alerts_requests_total",
				Help: "Total HTTP requests to /api/v2/alerts endpoint by status",
			},
			[]string{"status"}, // success, partial, error, validation_failed
		),

		// Metric 2: HTTP request duration histogram by status
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "alert_history_prometheus_alerts_duration_seconds",
				Help: "Request processing duration for /api/v2/alerts endpoint",
				// Buckets optimized for < 5ms p95 target (150% quality)
				// Covers: 1ms, 2ms, 5ms (target), 10ms, 20ms, 50ms, 100ms, 200ms, 500ms, 1s
				Buckets: []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1, 0.2, 0.5, 1.0},
			},
			[]string{"status"}, // success, partial, error, validation_failed
		),

		// Metric 3: Alerts received counter by format
		alertsReceived: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_prometheus_alerts_received_total",
				Help: "Total alerts received by Prometheus format (v1 array or v2 grouped)",
			},
			[]string{"format"}, // v1, v2
		),

		// Metric 4: Alerts processed counter by status
		alertsProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_prometheus_alerts_processed_total",
				Help: "Total alerts processed by status (success or failed)",
			},
			[]string{"status"}, // success, failed
		),

		// Metric 5: Validation errors counter by reason
		validationErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_prometheus_alerts_validation_errors_total",
				Help: "Total validation errors by reason (parse_error, validation_error, etc.)",
			},
			[]string{"reason"}, // method_not_allowed, read_body_error, parse_error, validation_error, etc.
		),

		// Metric 6: Processing errors counter by type
		processingErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_prometheus_alerts_processing_errors_total",
				Help: "Total processing errors by type (storage_error, processor_error, etc.)",
			},
			[]string{"type"}, // storage_error, processor_error, validation_error, timeout_error, unknown_error
		),

		// Metric 7: Concurrent requests gauge
		concurrentReqs: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "alert_history_prometheus_alerts_concurrent_requests",
				Help: "Current number of concurrent requests being processed by /api/v2/alerts",
			},
		),

		// Metric 8: Payload size histogram
		payloadSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name: "alert_history_prometheus_alerts_payload_bytes",
				Help: "Size of request payload in bytes (JSON body)",
				// Exponential buckets from 100B to 1.6MB (10 MB max, covers 99%+ requests)
				// 100, 200, 400, 800, 1.6K, 3.2K, 6.4K, 12.8K, 25.6K, 51.2K, 102K, 204K, 409K, 819K, 1.6MB
				Buckets: prometheus.ExponentialBuckets(100, 2, 15),
			},
		),
	}
}

// RecordRequest records HTTP request metrics.
//
// Increments request counter and observes duration in histogram.
// Should be called once per request, typically at the end of request processing.
//
// Parameters:
//   - status: Request status (success, partial, error, validation_failed)
//   - alertCount: Number of alerts in request (for logging/context)
//   - duration: Total request processing duration
//
// Example:
//   metrics.RecordRequest("success", 10, 5*time.Millisecond)
//   metrics.RecordRequest("partial", 5, 8*time.Millisecond)
//   metrics.RecordRequest("validation_failed", 0, 1*time.Millisecond)
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) RecordRequest(status string, alertCount int, duration time.Duration) {
	if m == nil {
		return
	}

	m.requestsTotal.WithLabelValues(status).Inc()
	m.requestDuration.WithLabelValues(status).Observe(duration.Seconds())
}

// RecordAlerts records alert processing metrics.
//
// Updates counters for alerts received and processed by format.
// Should be called once per request after processing completes.
//
// Parameters:
//   - format: Alert format (v1 or v2)
//   - received: Number of alerts received (updates alertsReceived if > 0)
//   - processed: Number of alerts successfully processed (updates alertsProcessed)
//   - failed: Number of alerts that failed processing (updates alertsProcessed)
//
// Example:
//   // Initial recording (after parsing)
//   metrics.RecordAlerts("v1", 10, 0, 0)  // 10 alerts received
//
//   // Final recording (after processing)
//   metrics.RecordAlerts("v1", 0, 8, 2)   // 8 succeeded, 2 failed
//
// Note: To avoid double-counting, call with received > 0 once, then call again
// with processed/failed counts.
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) RecordAlerts(format string, received, processed, failed int) {
	if m == nil {
		return
	}

	// Record received alerts (if provided)
	if received > 0 {
		m.alertsReceived.WithLabelValues(format).Add(float64(received))
	}

	// Record processed alerts
	if processed > 0 {
		m.alertsProcessed.WithLabelValues("success").Add(float64(processed))
	}

	// Record failed alerts
	if failed > 0 {
		m.alertsProcessed.WithLabelValues("failed").Add(float64(failed))
	}
}

// RecordValidationError records a validation error.
//
// Increments validation error counter with specific reason.
// Used to track common validation failures for debugging and alerting.
//
// Parameters:
//   - reason: Validation failure reason (method_not_allowed, parse_error,
//     validation_error, conversion_error, too_many_alerts, etc.)
//
// Example:
//   metrics.RecordValidationError("method_not_allowed")
//   metrics.RecordValidationError("parse_error")
//   metrics.RecordValidationError("validation_error")
//
// Common Reasons:
//   - method_not_allowed: Non-POST request
//   - read_body_error: Failed to read request body
//   - parse_error: JSON parsing failed
//   - validation_error: Webhook validation failed
//   - conversion_error: Domain conversion failed
//   - too_many_alerts: Alert count exceeds limit
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) RecordValidationError(reason string) {
	if m == nil {
		return
	}
	m.validationErrors.WithLabelValues(reason).Inc()
}

// RecordProcessingError records a processing error.
//
// Increments processing error counter with classified error type.
// Used to track different types of processing failures.
//
// Parameters:
//   - errorType: Error classification (storage_error, processor_error,
//     validation_error, timeout_error, unknown_error)
//
// Example:
//   metrics.RecordProcessingError("storage_error")
//   metrics.RecordProcessingError("processor_error")
//   metrics.RecordProcessingError("timeout_error")
//
// Error Types:
//   - storage_error: Database connection/query errors
//   - processor_error: AlertProcessor internal errors
//   - validation_error: Unexpected validation errors during processing
//   - timeout_error: Context timeout/cancellation errors
//   - unknown_error: Unclassified errors
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) RecordProcessingError(errorType string) {
	if m == nil {
		return
	}
	m.processingErrors.WithLabelValues(errorType).Inc()
}

// RecordPayloadSize records request payload size.
//
// Observes payload size in histogram for distribution analysis.
// Used to monitor typical payload sizes and detect anomalies.
//
// Parameters:
//   - bytes: Payload size in bytes (request body length)
//
// Example:
//   metrics.RecordPayloadSize(1024)      // 1 KB payload
//   metrics.RecordPayloadSize(10485760)  // 10 MB payload
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) RecordPayloadSize(bytes int) {
	if m == nil {
		return
	}
	m.payloadSize.Observe(float64(bytes))
}

// IncrementConcurrent increments concurrent requests counter.
//
// Should be called at the start of request processing.
// Always pair with DecrementConcurrent() in a defer statement.
//
// Example:
//   metrics.IncrementConcurrent()
//   defer metrics.DecrementConcurrent()
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) IncrementConcurrent() {
	if m == nil {
		return
	}
	m.concurrentReqs.Inc()
}

// DecrementConcurrent decrements concurrent requests counter.
//
// Should be called at the end of request processing (typically via defer).
// Always pair with IncrementConcurrent().
//
// Example:
//   metrics.IncrementConcurrent()
//   defer metrics.DecrementConcurrent()
//
// Safe to call with nil receiver (no-op).
func (m *PrometheusAlertsMetrics) DecrementConcurrent() {
	if m == nil {
		return
	}
	m.concurrentReqs.Dec()
}

// --- PromQL Query Examples ---
//
// These queries can be used in Grafana dashboards or alerting rules.
//
// Request Rate (requests/sec):
//   rate(alert_history_prometheus_alerts_requests_total[5m])
//
// Success Rate (%):
//   sum(rate(alert_history_prometheus_alerts_requests_total{status="success"}[5m]))
//   / sum(rate(alert_history_prometheus_alerts_requests_total[5m])) * 100
//
// p95 Latency (seconds):
//   histogram_quantile(0.95,
//     rate(alert_history_prometheus_alerts_duration_seconds_bucket[5m])
//   )
//
// Alert Processing Rate by Format (alerts/sec):
//   sum by (format) (rate(alert_history_prometheus_alerts_received_total[5m]))
//
// Error Rate by Reason:
//   sum by (reason) (rate(alert_history_prometheus_alerts_validation_errors_total[5m]))
//
// Processing Error Rate by Type:
//   sum by (type) (rate(alert_history_prometheus_alerts_processing_errors_total[5m]))
//
// Current Concurrent Requests:
//   alert_history_prometheus_alerts_concurrent_requests
//
// Average Payload Size (bytes):
//   rate(alert_history_prometheus_alerts_payload_bytes_sum[5m])
//   / rate(alert_history_prometheus_alerts_payload_bytes_count[5m])
//
// --- Alerting Rules Examples ---
//
// High Error Rate (> 5% for 5 minutes):
//   alert: PrometheusAlertsHighErrorRate
//   expr: |
//     sum(rate(alert_history_prometheus_alerts_requests_total{status=~"error|validation_failed"}[5m]))
//     / sum(rate(alert_history_prometheus_alerts_requests_total[5m])) > 0.05
//   for: 5m
//   annotations:
//     summary: "High error rate on /api/v2/alerts endpoint"
//
// High Latency (p95 > 10ms for 5 minutes):
//   alert: PrometheusAlertsHighLatency
//   expr: |
//     histogram_quantile(0.95,
//       rate(alert_history_prometheus_alerts_duration_seconds_bucket[5m])
//     ) > 0.01
//   for: 5m
//   annotations:
//     summary: "p95 latency > 10ms on /api/v2/alerts"
//
// Many Concurrent Requests (> 50 for 2 minutes):
//   alert: PrometheusAlertsManyC oncurrentRequests
//   expr: alert_history_prometheus_alerts_concurrent_requests > 50
//   for: 2m
//   annotations:
//     summary: "Many concurrent requests on /api/v2/alerts (possible overload)"
