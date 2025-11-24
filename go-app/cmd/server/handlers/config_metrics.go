// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	configExportMetricsOnce sync.Once
	configExportMetrics     *ConfigExportMetrics
)

// ConfigExportMetrics tracks metrics for GET /api/v2/config endpoint.
//
// This metrics collector provides comprehensive observability for the
// configuration export endpoint, tracking request patterns, format usage,
// errors, and performance characteristics.
//
// Metrics Categories:
//   - HTTP Request metrics: Total requests, duration, status
//   - Format metrics: JSON vs YAML usage
//   - Security metrics: Sanitized vs unsanitized requests
//   - Error metrics: Serialization errors, validation errors
//   - Performance metrics: Response size, processing time
//
// All metrics are prefixed with "alert_history_api_config_" to ensure
// uniqueness in the Prometheus namespace.
//
// Example Usage:
//   metrics := NewConfigExportMetrics()
//   metrics.RecordRequest("success", "json", true, 5*time.Millisecond, 1024)
//   metrics.RecordError("serialization")
//
// Grafana Integration:
//   See design.md for PromQL query examples and dashboard configuration.
type ConfigExportMetrics struct {
	// HTTP request metrics
	requestsTotal   *prometheus.CounterVec   // Total HTTP requests by format, sanitized, status
	requestDuration *prometheus.HistogramVec // Request duration by format, sanitized

	// Error metrics
	errorsTotal *prometheus.CounterVec // Errors by type (serialization, validation, etc.)

	// Performance metrics
	responseSize prometheus.Histogram // Response size distribution in bytes
}

// NewConfigExportMetrics initializes metrics for GET /api/v2/config endpoint.
//
// Creates and registers 4 Prometheus metrics:
//   1. alert_history_api_config_export_requests_total (CounterVec)
//   2. alert_history_api_config_export_duration_seconds (HistogramVec)
//   3. alert_history_api_config_export_errors_total (CounterVec)
//   4. alert_history_api_config_export_size_bytes (Histogram)
//
// Metrics are automatically registered with the default Prometheus registry.
// Uses sync.Once to ensure metrics are registered only once, preventing duplicate registration errors.
//
// Returns:
//   - *ConfigExportMetrics: Initialized metrics collector (singleton)
//
// Thread-safe: Multiple calls return the same instance
func NewConfigExportMetrics() *ConfigExportMetrics {
	configExportMetricsOnce.Do(func() {
		configExportMetrics = &ConfigExportMetrics{
		// Metric 1: HTTP request counter by format, sanitized, status
		requestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_api_config_export_requests_total",
				Help: "Total HTTP requests to GET /api/v2/config endpoint by format, sanitized, status",
			},
			[]string{"format", "sanitized", "status"}, // format: json/yaml, sanitized: true/false, status: success/error
		),

		// Metric 2: HTTP request duration histogram by format, sanitized
		requestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "alert_history_api_config_export_duration_seconds",
				Help: "Request processing duration for GET /api/v2/config endpoint",
				// Buckets optimized for < 5ms p95 target (150% quality)
				// Covers: 0.1ms, 0.5ms, 1ms, 2ms, 5ms (target), 10ms, 20ms, 50ms, 100ms
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.002, 0.005, 0.01, 0.02, 0.05, 0.1},
			},
			[]string{"format", "sanitized"}, // format: json/yaml, sanitized: true/false
		),

		// Metric 3: Errors counter by type
		errorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_api_config_export_errors_total",
				Help: "Total errors for GET /api/v2/config endpoint by type",
			},
			[]string{"error_type"}, // serialization, validation, service, unknown
		),

		// Metric 4: Response size histogram
		responseSize: promauto.NewHistogram(
			prometheus.HistogramOpts{
				Name: "alert_history_api_config_export_size_bytes",
				Help: "Response size distribution for GET /api/v2/config endpoint in bytes",
				// Buckets: 100B, 500B, 1KB, 5KB, 10KB, 50KB, 100KB, 500KB, 1MB
				Buckets: []float64{100, 500, 1000, 5000, 10000, 50000, 100000, 500000, 1000000},
			},
		),
		}
	})
	return configExportMetrics
}

// RecordRequest records metrics for a completed request.
//
// Parameters:
//   - format: Response format ("json" or "yaml")
//   - sanitized: Whether secrets were sanitized
//   - status: Request status ("success" or "error")
//   - duration: Request processing duration
//   - sizeBytes: Response size in bytes
func (m *ConfigExportMetrics) RecordRequest(
	format string,
	sanitized bool,
	status string,
	duration time.Duration,
	sizeBytes int,
) {
	sanitizedStr := "false"
	if sanitized {
		sanitizedStr = "true"
	}

	m.requestsTotal.WithLabelValues(format, sanitizedStr, status).Inc()
	m.requestDuration.WithLabelValues(format, sanitizedStr).Observe(duration.Seconds())
	m.responseSize.Observe(float64(sizeBytes))
}

// RecordError records an error metric.
//
// Parameters:
//   - errorType: Type of error ("serialization", "validation", "service", "unknown")
func (m *ConfigExportMetrics) RecordError(errorType string) {
	m.errorsTotal.WithLabelValues(errorType).Inc()
}
