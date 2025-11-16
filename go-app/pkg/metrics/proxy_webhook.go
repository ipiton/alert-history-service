// Package metrics provides Prometheus metrics for the alert history service.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ProxyWebhookMetrics holds all Prometheus metrics for the proxy webhook endpoint.
type ProxyWebhookMetrics struct {
	// HTTP Metrics (6 metrics)
	HTTPRequestsTotal      *prometheus.CounterVec
	HTTPRequestDuration    *prometheus.HistogramVec
	HTTPRequestSize        *prometheus.HistogramVec
	HTTPResponseSize       *prometheus.HistogramVec
	HTTPRequestsInFlight   prometheus.Gauge
	HTTPErrorsTotal        *prometheus.CounterVec

	// Processing Metrics (5 metrics)
	AlertsReceivedTotal      *prometheus.CounterVec
	AlertsProcessedTotal     *prometheus.CounterVec
	ClassificationDuration   *prometheus.HistogramVec
	FilteringDuration        *prometheus.HistogramVec
	PublishingDuration       *prometheus.HistogramVec

	// Error Metrics (3 metrics)
	ClassificationErrorsTotal *prometheus.CounterVec
	FilteringErrorsTotal      *prometheus.CounterVec
	PublishingErrorsTotal     *prometheus.CounterVec

	// Performance Metrics (4 metrics)
	PipelineDuration       *prometheus.HistogramVec
	BatchSize              prometheus.Histogram
	ConcurrentRequests     prometheus.Gauge
	PublishingTargetsTotal *prometheus.GaugeVec
}

// NewProxyWebhookMetrics creates and registers all proxy webhook metrics.
func NewProxyWebhookMetrics(registry prometheus.Registerer) *ProxyWebhookMetrics {
	if registry == nil {
		registry = prometheus.DefaultRegisterer
	}

	factory := promauto.With(registry)

	return &ProxyWebhookMetrics{
		// HTTP Metrics
		HTTPRequestsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_http_requests_total",
				Help: "Total number of HTTP requests received by the proxy webhook endpoint",
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPRequestDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPRequestSize: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_http_request_size_bytes",
				Help:    "HTTP request body size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000, 1000000, 10000000},
			},
			[]string{"method", "path"},
		),

		HTTPResponseSize: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_http_response_size_bytes",
				Help:    "HTTP response body size in bytes",
				Buckets: []float64{100, 1000, 10000, 100000},
			},
			[]string{"method", "path", "status_code"},
		),

		HTTPRequestsInFlight: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "alert_history_proxy_http_requests_in_flight",
				Help: "Current number of HTTP requests being processed",
			},
		),

		HTTPErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"method", "path", "error_type"},
		),

		// Processing Metrics
		AlertsReceivedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_alerts_received_total",
				Help: "Total number of alerts received",
			},
			[]string{"status"}, // firing, resolved
		),

		AlertsProcessedTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_alerts_processed_total",
				Help: "Total number of alerts processed",
			},
			[]string{"status", "result"}, // result: success, filtered, failed
		),

		ClassificationDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_classification_duration_seconds",
				Help:    "LLM classification pipeline duration in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1.0, 5.0},
			},
			[]string{"cached"}, // true, false
		),

		FilteringDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_filtering_duration_seconds",
				Help:    "Filtering pipeline duration in seconds",
				Buckets: []float64{0.0001, 0.001, 0.01},
			},
			[]string{"action"}, // allow, deny
		),

		PublishingDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_publishing_duration_seconds",
				Help:    "Publishing pipeline duration in seconds",
				Buckets: []float64{0.1, 0.5, 1.0, 5.0, 10.0},
			},
			[]string{"target_type"}, // rootly, pagerduty, slack
		),

		// Error Metrics
		ClassificationErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_classification_errors_total",
				Help: "Total number of classification pipeline errors",
			},
			[]string{"error_type"}, // llm_failure, timeout, etc.
		),

		FilteringErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_filtering_errors_total",
				Help: "Total number of filtering pipeline errors",
			},
			[]string{"error_type"},
		),

		PublishingErrorsTotal: factory.NewCounterVec(
			prometheus.CounterOpts{
				Name: "alert_history_proxy_publishing_errors_total",
				Help: "Total number of publishing pipeline errors",
			},
			[]string{"target_type", "error_type"},
		),

		// Performance Metrics
		PipelineDuration: factory.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_pipeline_duration_seconds",
				Help:    "End-to-end pipeline duration in seconds",
				Buckets: []float64{0.001, 0.01, 0.1, 1.0, 10.0},
			},
			[]string{"pipeline"}, // classification, filtering, publishing, total
		),

		BatchSize: factory.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "alert_history_proxy_batch_size",
				Help:    "Number of alerts per batch",
				Buckets: []float64{1, 5, 10, 50, 100, 500},
			},
		),

		ConcurrentRequests: factory.NewGauge(
			prometheus.GaugeOpts{
				Name: "alert_history_proxy_concurrent_requests",
				Help: "Current number of concurrent requests",
			},
		),

		PublishingTargetsTotal: factory.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "alert_history_proxy_publishing_targets_total",
				Help: "Number of publishing targets by type and health status",
			},
			[]string{"target_type", "health_status"}, // healthy, unhealthy, unknown
		),
	}
}

// RecordHTTPRequest records an HTTP request.
func (m *ProxyWebhookMetrics) RecordHTTPRequest(method, path string, statusCode int, duration float64, requestSize, responseSize int64) {
	statusCodeStr := statusCodeToString(statusCode)

	m.HTTPRequestsTotal.WithLabelValues(method, path, statusCodeStr).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, path, statusCodeStr).Observe(duration)
	m.HTTPRequestSize.WithLabelValues(method, path).Observe(float64(requestSize))
	m.HTTPResponseSize.WithLabelValues(method, path, statusCodeStr).Observe(float64(responseSize))
}

// RecordHTTPError records an HTTP error.
func (m *ProxyWebhookMetrics) RecordHTTPError(method, path, errorType string) {
	m.HTTPErrorsTotal.WithLabelValues(method, path, errorType).Inc()
}

// IncHTTPRequestsInFlight increments the in-flight requests gauge.
func (m *ProxyWebhookMetrics) IncHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Inc()
}

// DecHTTPRequestsInFlight decrements the in-flight requests gauge.
func (m *ProxyWebhookMetrics) DecHTTPRequestsInFlight() {
	m.HTTPRequestsInFlight.Dec()
}

// RecordAlertReceived records a received alert.
func (m *ProxyWebhookMetrics) RecordAlertReceived(status string) {
	m.AlertsReceivedTotal.WithLabelValues(status).Inc()
}

// RecordAlertProcessed records a processed alert.
func (m *ProxyWebhookMetrics) RecordAlertProcessed(status, result string) {
	m.AlertsProcessedTotal.WithLabelValues(status, result).Inc()
}

// RecordClassificationDuration records classification duration.
func (m *ProxyWebhookMetrics) RecordClassificationDuration(duration float64, cached bool) {
	cachedStr := "false"
	if cached {
		cachedStr = "true"
	}
	m.ClassificationDuration.WithLabelValues(cachedStr).Observe(duration)
}

// RecordFilteringDuration records filtering duration.
func (m *ProxyWebhookMetrics) RecordFilteringDuration(duration float64, action string) {
	m.FilteringDuration.WithLabelValues(action).Observe(duration)
}

// RecordPublishingDuration records publishing duration.
func (m *ProxyWebhookMetrics) RecordPublishingDuration(duration float64, targetType string) {
	m.PublishingDuration.WithLabelValues(targetType).Observe(duration)
}

// RecordClassificationError records a classification error.
func (m *ProxyWebhookMetrics) RecordClassificationError(errorType string) {
	m.ClassificationErrorsTotal.WithLabelValues(errorType).Inc()
}

// RecordFilteringError records a filtering error.
func (m *ProxyWebhookMetrics) RecordFilteringError(errorType string) {
	m.FilteringErrorsTotal.WithLabelValues(errorType).Inc()
}

// RecordPublishingError records a publishing error.
func (m *ProxyWebhookMetrics) RecordPublishingError(targetType, errorType string) {
	m.PublishingErrorsTotal.WithLabelValues(targetType, errorType).Inc()
}

// RecordPipelineDuration records pipeline duration.
func (m *ProxyWebhookMetrics) RecordPipelineDuration(duration float64, pipeline string) {
	m.PipelineDuration.WithLabelValues(pipeline).Observe(duration)
}

// RecordBatchSize records batch size.
func (m *ProxyWebhookMetrics) RecordBatchSize(size int) {
	m.BatchSize.Observe(float64(size))
}

// SetConcurrentRequests sets the current concurrent requests.
func (m *ProxyWebhookMetrics) SetConcurrentRequests(count int) {
	m.ConcurrentRequests.Set(float64(count))
}

// SetPublishingTargets sets the number of publishing targets.
func (m *ProxyWebhookMetrics) SetPublishingTargets(targetType, healthStatus string, count int) {
	m.PublishingTargetsTotal.WithLabelValues(targetType, healthStatus).Set(float64(count))
}

// statusCodeToString converts an HTTP status code to a string.
// This is a helper function to ensure consistent label values.
func statusCodeToString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500:
		return "5xx"
	default:
		return "unknown"
	}
}

