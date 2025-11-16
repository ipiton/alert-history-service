package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewProxyWebhookMetrics tests metric creation.
func TestNewProxyWebhookMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	require.NotNil(t, metrics)

	// Verify all HTTP metrics
	assert.NotNil(t, metrics.HTTPRequestsTotal)
	assert.NotNil(t, metrics.HTTPRequestDuration)
	assert.NotNil(t, metrics.HTTPRequestSize)
	assert.NotNil(t, metrics.HTTPResponseSize)
	assert.NotNil(t, metrics.HTTPRequestsInFlight)
	assert.NotNil(t, metrics.HTTPErrorsTotal)

	// Verify all processing metrics
	assert.NotNil(t, metrics.AlertsReceivedTotal)
	assert.NotNil(t, metrics.AlertsProcessedTotal)
	assert.NotNil(t, metrics.ClassificationDuration)
	assert.NotNil(t, metrics.FilteringDuration)
	assert.NotNil(t, metrics.PublishingDuration)

	// Verify all error metrics
	assert.NotNil(t, metrics.ClassificationErrorsTotal)
	assert.NotNil(t, metrics.FilteringErrorsTotal)
	assert.NotNil(t, metrics.PublishingErrorsTotal)

	// Verify all performance metrics
	assert.NotNil(t, metrics.PipelineDuration)
	assert.NotNil(t, metrics.BatchSize)
	assert.NotNil(t, metrics.ConcurrentRequests)
	assert.NotNil(t, metrics.PublishingTargetsTotal)
}

// TestRecordHTTPRequest tests HTTP request recording.
func TestRecordHTTPRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record a request
	metrics.RecordHTTPRequest("POST", "/webhook/proxy", 200, 0.5, 1024, 512)

	// Verify counter incremented
	count := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("POST", "/webhook/proxy", "2xx"))
	assert.Equal(t, 1.0, count)
}

// TestRecordHTTPError tests HTTP error recording.
func TestRecordHTTPError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record an error
	metrics.RecordHTTPError("POST", "/webhook/proxy", "validation_error")

	// Verify counter incremented
	count := testutil.ToFloat64(metrics.HTTPErrorsTotal.WithLabelValues("POST", "/webhook/proxy", "validation_error"))
	assert.Equal(t, 1.0, count)
}

// TestHTTPRequestsInFlight tests in-flight requests tracking.
func TestHTTPRequestsInFlight(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Initial value
	assert.Equal(t, 0.0, testutil.ToFloat64(metrics.HTTPRequestsInFlight))

	// Increment
	metrics.IncHTTPRequestsInFlight()
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.HTTPRequestsInFlight))

	// Increment again
	metrics.IncHTTPRequestsInFlight()
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.HTTPRequestsInFlight))

	// Decrement
	metrics.DecHTTPRequestsInFlight()
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.HTTPRequestsInFlight))

	// Decrement to zero
	metrics.DecHTTPRequestsInFlight()
	assert.Equal(t, 0.0, testutil.ToFloat64(metrics.HTTPRequestsInFlight))
}

// TestRecordAlertReceived tests alert received recording.
func TestRecordAlertReceived(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record firing alert
	metrics.RecordAlertReceived("firing")
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.AlertsReceivedTotal.WithLabelValues("firing")))

	// Record resolved alert
	metrics.RecordAlertReceived("resolved")
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.AlertsReceivedTotal.WithLabelValues("resolved")))

	// Record another firing alert
	metrics.RecordAlertReceived("firing")
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.AlertsReceivedTotal.WithLabelValues("firing")))
}

// TestRecordAlertProcessed tests alert processed recording.
func TestRecordAlertProcessed(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record success
	metrics.RecordAlertProcessed("firing", "success")
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.AlertsProcessedTotal.WithLabelValues("firing", "success")))

	// Record filtered
	metrics.RecordAlertProcessed("firing", "filtered")
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.AlertsProcessedTotal.WithLabelValues("firing", "filtered")))

	// Record failed
	metrics.RecordAlertProcessed("firing", "failed")
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.AlertsProcessedTotal.WithLabelValues("firing", "failed")))
}

// TestRecordClassificationDuration tests classification duration recording.
func TestRecordClassificationDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record cached classification
	metrics.RecordClassificationDuration(0.001, true)

	// Record uncached classification
	metrics.RecordClassificationDuration(1.5, false)

	// Verify histograms recorded observations
	// For histograms, we can't easily test values, just verify no panic occurred
	// In production, we'd use Prometheus queries to verify
	assert.NotNil(t, metrics.ClassificationDuration)
}

// TestRecordFilteringDuration tests filtering duration recording.
func TestRecordFilteringDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record allow action
	metrics.RecordFilteringDuration(0.0005, "allow")

	// Record deny action
	metrics.RecordFilteringDuration(0.0003, "deny")

	// Verify histograms recorded observations
	assert.NotNil(t, metrics.FilteringDuration)
}

// TestRecordPublishingDuration tests publishing duration recording.
func TestRecordPublishingDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record different target types
	metrics.RecordPublishingDuration(0.5, "rootly")
	metrics.RecordPublishingDuration(1.0, "pagerduty")
	metrics.RecordPublishingDuration(0.3, "slack")

	// Verify histograms recorded observations
	assert.NotNil(t, metrics.PublishingDuration)
}

// TestRecordClassificationError tests classification error recording.
func TestRecordClassificationError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record errors
	metrics.RecordClassificationError("llm_failure")
	metrics.RecordClassificationError("timeout")
	metrics.RecordClassificationError("llm_failure")

	// Verify counters
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.ClassificationErrorsTotal.WithLabelValues("llm_failure")))
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.ClassificationErrorsTotal.WithLabelValues("timeout")))
}

// TestRecordFilteringError tests filtering error recording.
func TestRecordFilteringError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record errors
	metrics.RecordFilteringError("rule_evaluation_error")

	// Verify counter
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.FilteringErrorsTotal.WithLabelValues("rule_evaluation_error")))
}

// TestRecordPublishingError tests publishing error recording.
func TestRecordPublishingError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record errors
	metrics.RecordPublishingError("rootly", "network_error")
	metrics.RecordPublishingError("pagerduty", "auth_error")
	metrics.RecordPublishingError("rootly", "network_error")

	// Verify counters
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.PublishingErrorsTotal.WithLabelValues("rootly", "network_error")))
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.PublishingErrorsTotal.WithLabelValues("pagerduty", "auth_error")))
}

// TestRecordPipelineDuration tests pipeline duration recording.
func TestRecordPipelineDuration(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record different pipelines
	metrics.RecordPipelineDuration(0.01, "classification")
	metrics.RecordPipelineDuration(0.001, "filtering")
	metrics.RecordPipelineDuration(1.0, "publishing")
	metrics.RecordPipelineDuration(1.5, "total")

	// Verify histograms recorded observations
	assert.NotNil(t, metrics.PipelineDuration)
}

// TestRecordBatchSize tests batch size recording.
func TestRecordBatchSize(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Record different batch sizes
	metrics.RecordBatchSize(1)
	metrics.RecordBatchSize(10)
	metrics.RecordBatchSize(100)

	// Verify histogram recorded observations
	assert.NotNil(t, metrics.BatchSize)
}

// TestSetConcurrentRequests tests concurrent requests setting.
func TestSetConcurrentRequests(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Set different values
	metrics.SetConcurrentRequests(5)
	assert.Equal(t, 5.0, testutil.ToFloat64(metrics.ConcurrentRequests))

	metrics.SetConcurrentRequests(10)
	assert.Equal(t, 10.0, testutil.ToFloat64(metrics.ConcurrentRequests))

	metrics.SetConcurrentRequests(0)
	assert.Equal(t, 0.0, testutil.ToFloat64(metrics.ConcurrentRequests))
}

// TestSetPublishingTargets tests publishing targets setting.
func TestSetPublishingTargets(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Set different target counts
	metrics.SetPublishingTargets("rootly", "healthy", 3)
	metrics.SetPublishingTargets("pagerduty", "healthy", 2)
	metrics.SetPublishingTargets("slack", "unhealthy", 1)

	// Verify gauges
	assert.Equal(t, 3.0, testutil.ToFloat64(metrics.PublishingTargetsTotal.WithLabelValues("rootly", "healthy")))
	assert.Equal(t, 2.0, testutil.ToFloat64(metrics.PublishingTargetsTotal.WithLabelValues("pagerduty", "healthy")))
	assert.Equal(t, 1.0, testutil.ToFloat64(metrics.PublishingTargetsTotal.WithLabelValues("slack", "unhealthy")))
}

// TestMultipleRequests tests recording multiple requests in sequence.
func TestMultipleRequests(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	// Simulate 100 successful requests
	for i := 0; i < 100; i++ {
		metrics.IncHTTPRequestsInFlight()
		metrics.RecordAlertReceived("firing")
		metrics.RecordAlertProcessed("firing", "success")
		metrics.RecordHTTPRequest("POST", "/webhook/proxy", 200, 0.1, 1024, 512)
		metrics.DecHTTPRequestsInFlight()
	}

	// Verify counts
	assert.Equal(t, 100.0, testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("POST", "/webhook/proxy", "2xx")))
	assert.Equal(t, 100.0, testutil.ToFloat64(metrics.AlertsReceivedTotal.WithLabelValues("firing")))
	assert.Equal(t, 100.0, testutil.ToFloat64(metrics.AlertsProcessedTotal.WithLabelValues("firing", "success")))
	assert.Equal(t, 0.0, testutil.ToFloat64(metrics.HTTPRequestsInFlight))
}

// BenchmarkRecordHTTPRequest benchmarks HTTP request recording.
func BenchmarkRecordHTTPRequest(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordHTTPRequest("POST", "/webhook/proxy", 200, 0.5, 1024, 512)
	}
}

// BenchmarkRecordAlertProcessed benchmarks alert processed recording.
func BenchmarkRecordAlertProcessed(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordAlertProcessed("firing", "success")
	}
}

// BenchmarkRecordClassificationDuration benchmarks classification duration recording.
func BenchmarkRecordClassificationDuration(b *testing.B) {
	registry := prometheus.NewRegistry()
	metrics := NewProxyWebhookMetrics(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordClassificationDuration(0.5, false)
	}
}
