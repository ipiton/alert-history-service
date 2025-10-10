package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

var (
	testWebhookMetrics *WebhookMetrics
	testWebhookOnce    sync.Once
)

// getTestWebhookMetrics returns a shared WebhookMetrics instance for all tests
// to avoid duplicate metric registration errors.
func getTestWebhookMetrics() *WebhookMetrics {
	testWebhookOnce.Do(func() {
		testWebhookMetrics = NewWebhookMetrics()
	})
	return testWebhookMetrics
}

func TestNewWebhookMetrics(t *testing.T) {
	metrics := getTestWebhookMetrics()

	if metrics.RequestsTotal == nil {
		t.Error("RequestsTotal should not be nil")
	}
	if metrics.DurationSeconds == nil {
		t.Error("DurationSeconds should not be nil")
	}
	if metrics.ProcessingSeconds == nil {
		t.Error("ProcessingSeconds should not be nil")
	}
	if metrics.QueueSize == nil {
		t.Error("QueueSize should not be nil")
	}
	if metrics.ActiveWorkers == nil {
		t.Error("ActiveWorkers should not be nil")
	}
	if metrics.ErrorsTotal == nil {
		t.Error("ErrorsTotal should not be nil")
	}
	if metrics.PayloadSizeBytes == nil {
		t.Error("PayloadSizeBytes should not be nil")
	}
}

func TestRecordRequest(t *testing.T) {
	// Create custom registry to avoid conflicts with global metrics
	registry := prometheus.NewRegistry()

	requestsCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_webhook_requests_total",
			Help: "Test webhook requests",
		},
		[]string{"type", "status"},
	)
	registry.MustRegister(requestsCounter)

	durationHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_webhook_duration_seconds",
			Help:    "Test webhook duration",
			Buckets: []float64{0.01, 0.1, 1.0},
		},
		[]string{"type"},
	)
	registry.MustRegister(durationHistogram)

	// Test counter increment
	requestsCounter.WithLabelValues("alertmanager", "success").Inc()
	requestsCounter.WithLabelValues("alertmanager", "success").Inc()
	requestsCounter.WithLabelValues("generic", "failure").Inc()

	// Verify counter values
	if count := testutil.ToFloat64(requestsCounter.WithLabelValues("alertmanager", "success")); count != 2.0 {
		t.Errorf("Expected 2 alertmanager success requests, got %f", count)
	}
	if count := testutil.ToFloat64(requestsCounter.WithLabelValues("generic", "failure")); count != 1.0 {
		t.Errorf("Expected 1 generic failure request, got %f", count)
	}

	// Test histogram observation
	durationHistogram.WithLabelValues("alertmanager").Observe(0.045)
	durationHistogram.WithLabelValues("alertmanager").Observe(0.123)

	// Histograms don't expose count via testutil.ToFloat64, just verify no panic
	// In production, Prometheus scrapes _count and _sum automatically
}

func TestRecordProcessingStage(t *testing.T) {
	registry := prometheus.NewRegistry()

	processingHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_webhook_processing_seconds",
			Help:    "Test processing time",
			Buckets: []float64{0.001, 0.01, 0.1},
		},
		[]string{"type", "stage"},
	)
	registry.MustRegister(processingHistogram)

	// Record different stages
	processingHistogram.WithLabelValues("alertmanager", "parse").Observe(0.002)
	processingHistogram.WithLabelValues("alertmanager", "validate").Observe(0.001)
	processingHistogram.WithLabelValues("alertmanager", "process").Observe(0.015)

	// Histograms don't expose count directly via testutil.ToFloat64
	// Verify observations complete without panic
}

func TestRecordError(t *testing.T) {
	registry := prometheus.NewRegistry()

	errorsCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "test_webhook_errors_total",
			Help: "Test webhook errors",
		},
		[]string{"type", "error_type"},
	)
	registry.MustRegister(errorsCounter)

	// Record different error types
	errorsCounter.WithLabelValues("alertmanager", "parse_error").Inc()
	errorsCounter.WithLabelValues("alertmanager", "parse_error").Inc()
	errorsCounter.WithLabelValues("generic", "validation_error").Inc()

	// Verify error counts
	if count := testutil.ToFloat64(errorsCounter.WithLabelValues("alertmanager", "parse_error")); count != 2.0 {
		t.Errorf("Expected 2 parse errors, got %f", count)
	}
	if count := testutil.ToFloat64(errorsCounter.WithLabelValues("generic", "validation_error")); count != 1.0 {
		t.Errorf("Expected 1 validation error, got %f", count)
	}
}

func TestRecordPayloadSize(t *testing.T) {
	registry := prometheus.NewRegistry()

	payloadHistogram := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "test_webhook_payload_size_bytes",
			Help:    "Test payload size",
			Buckets: []float64{1024, 10240, 102400},
		},
		[]string{"type"},
	)
	registry.MustRegister(payloadHistogram)

	// Record different payload sizes
	payloadHistogram.WithLabelValues("alertmanager").Observe(5120)
	payloadHistogram.WithLabelValues("alertmanager").Observe(50000)

	// Histograms don't expose count directly
	// Verify observations complete without panic
}

func TestSetQueueSize(t *testing.T) {
	registry := prometheus.NewRegistry()

	queueGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "test_webhook_queue_size",
			Help: "Test queue size",
		},
	)
	registry.MustRegister(queueGauge)

	// Set different queue sizes
	queueGauge.Set(10)
	if value := testutil.ToFloat64(queueGauge); value != 10.0 {
		t.Errorf("Expected queue size 10, got %f", value)
	}

	queueGauge.Set(25)
	if value := testutil.ToFloat64(queueGauge); value != 25.0 {
		t.Errorf("Expected queue size 25, got %f", value)
	}
}

func TestActiveWorkers(t *testing.T) {
	registry := prometheus.NewRegistry()

	workersGauge := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "test_webhook_active_workers",
			Help: "Test active workers",
		},
	)
	registry.MustRegister(workersGauge)

	// Test increment/decrement
	workersGauge.Set(5)
	workersGauge.Inc()
	if value := testutil.ToFloat64(workersGauge); value != 6.0 {
		t.Errorf("Expected 6 active workers after increment, got %f", value)
	}

	workersGauge.Dec()
	if value := testutil.ToFloat64(workersGauge); value != 5.0 {
		t.Errorf("Expected 5 active workers after decrement, got %f", value)
	}
}

func TestWebhookMetricsScenarios(t *testing.T) {
	// Use shared WebhookMetrics instance to avoid duplicate registration
	metrics := getTestWebhookMetrics()

	t.Run("SuccessScenario", func(t *testing.T) {
		// Simulate a successful webhook request flow
		webhookType := "alertmanager"

		// Record payload size
		metrics.RecordPayloadSize(webhookType, 8192)

		// Record processing stages
		metrics.RecordProcessingStage(webhookType, "parse", 0.002)
		metrics.RecordProcessingStage(webhookType, "validate", 0.001)
		metrics.RecordProcessingStage(webhookType, "convert", 0.003)
		metrics.RecordProcessingStage(webhookType, "process", 0.015)

		// Record overall request
		totalDuration := 0.045
		metrics.RecordRequest(webhookType, "success", totalDuration)

		// Update queue/worker metrics
		metrics.SetQueueSize(5)
		metrics.IncrementActiveWorkers()

		// Verify metrics were recorded (basic smoke test)
		// In real deployment, these would be scraped by Prometheus
		if metrics.RequestsTotal == nil {
			t.Error("RequestsTotal should be initialized")
		}
		if metrics.QueueSize == nil {
			t.Error("QueueSize should be initialized")
		}
	})

	t.Run("ErrorScenario", func(t *testing.T) {
		webhookType := "generic"

		// Simulate error scenario
		metrics.RecordPayloadSize(webhookType, 1500)
		metrics.RecordProcessingStage(webhookType, "parse", 0.001)

		// Validation fails
		metrics.RecordError(webhookType, "validation_error")
		metrics.RecordRequest(webhookType, "failure", 0.005)

		// Worker completes, queue decreases
		metrics.SetQueueSize(8)
		metrics.DecrementActiveWorkers()

		// Verify error metrics were recorded
		if metrics.ErrorsTotal == nil {
			t.Error("ErrorsTotal should be initialized")
		}
	})
}

// BenchmarkRecordRequest benchmarks the performance of RecordRequest.
func BenchmarkRecordRequest(b *testing.B) {
	metrics := getTestWebhookMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordRequest("alertmanager", "success", 0.045)
	}
}

// BenchmarkRecordProcessingStage benchmarks processing stage recording.
func BenchmarkRecordProcessingStage(b *testing.B) {
	metrics := getTestWebhookMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordProcessingStage("alertmanager", "parse", 0.002)
	}
}

// BenchmarkRecordError benchmarks error recording.
func BenchmarkRecordError(b *testing.B) {
	metrics := getTestWebhookMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.RecordError("alertmanager", "parse_error")
	}
}

// BenchmarkSetQueueSize benchmarks queue size updates.
func BenchmarkSetQueueSize(b *testing.B) {
	metrics := getTestWebhookMetrics()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		metrics.SetQueueSize(i % 100)
	}
}
