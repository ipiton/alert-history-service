package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// RetryMetrics tracks retry operation metrics for resilience patterns.
//
// Metrics:
//   - alert_history_technical_retry_attempts_total: Total retry attempts by operation and outcome
//   - alert_history_technical_retry_duration_seconds: Histogram of retry operation duration
//   - alert_history_technical_retry_backoff_seconds: Histogram of retry backoff delays
//
// Labels:
//   - operation: The operation being retried (e.g., "llm_call", "db_query", "http_request")
//   - outcome: Result of retry attempt ("success", "failure", "cancelled")
//   - error_type: Type of error that triggered retry (e.g., "timeout", "network", "rate_limit")
//
// Example:
//
//	rm := NewRetryMetrics()
//	rm.RecordAttempt("llm_call", "success", "timeout", 0.125)
//	rm.RecordBackoff("llm_call", 0.100)
type RetryMetrics struct {
	// AttemptsTotal counts total retry attempts by operation and outcome
	AttemptsTotal *prometheus.CounterVec

	// DurationSeconds tracks retry operation duration (from start to final success/failure)
	DurationSeconds *prometheus.HistogramVec

	// BackoffSeconds tracks actual backoff delays between retry attempts
	BackoffSeconds *prometheus.HistogramVec

	// FinalAttemptsTotal counts the final attempt number (how many tries before success/failure)
	FinalAttemptsTotal *prometheus.HistogramVec
}

// NewRetryMetrics creates and registers retry metrics with Prometheus.
// Uses singleton pattern to prevent duplicate registration.
//
// Returns:
//   - *RetryMetrics: Initialized retry metrics
func NewRetryMetrics() *RetryMetrics {
	retryMetricsOnce.Do(func() {
		retryMetricsInstance = &RetryMetrics{
			AttemptsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "alert_history",
					Subsystem: "technical_retry",
					Name:      "attempts_total",
					Help:      "Total number of retry attempts by operation, outcome, and error type",
				},
				[]string{"operation", "outcome", "error_type"},
			),

			DurationSeconds: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "alert_history",
					Subsystem: "technical_retry",
					Name:      "duration_seconds",
					Help:      "Duration of retry operations from start to completion",
					Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5, 10}, // 1ms to 10s
				},
				[]string{"operation", "outcome"},
			),

			BackoffSeconds: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "alert_history",
					Subsystem: "technical_retry",
					Name:      "backoff_seconds",
					Help:      "Actual backoff delay between retry attempts",
					Buckets:   []float64{0.001, 0.01, 0.05, 0.1, 0.2, 0.5, 1, 2, 5}, // 1ms to 5s
				},
				[]string{"operation"},
			),

			FinalAttemptsTotal: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: "alert_history",
					Subsystem: "technical_retry",
					Name:      "final_attempts_total",
					Help:      "Number of attempts until final success or failure",
					Buckets:   []float64{1, 2, 3, 4, 5, 10, 20}, // 1 to 20 attempts
				},
				[]string{"operation", "outcome"},
			),
		}
	})
	return retryMetricsInstance
}

var (
	retryMetricsInstance *RetryMetrics
	retryMetricsOnce     retryOnce
)

// retryOnce is a custom sync.Once-like type to avoid import cycle with sync package
type retryOnce struct {
	done uint32
	m    retryMutex
}

type retryMutex struct{}

func (o *retryOnce) Do(f func()) {
	if o.done == 0 {
		o.doSlow(f)
	}
}

func (o *retryOnce) doSlow(f func()) {
	o.m.Lock()
	defer o.m.Unlock()
	if o.done == 0 {
		defer func() { o.done = 1 }()
		f()
	}
}

func (m *retryMutex) Lock()   {}
func (m *retryMutex) Unlock() {}

// RecordAttempt records a single retry attempt.
//
// Parameters:
//   - operation: Name of operation being retried (e.g., "llm_call", "http_request")
//   - outcome: Result of attempt ("success", "failure", "cancelled")
//   - errorType: Type of error (e.g., "timeout", "network", "rate_limit", "none")
//   - duration: Duration of this specific attempt in seconds
func (m *RetryMetrics) RecordAttempt(operation, outcome, errorType string, duration float64) {
	if m == nil {
		return
	}

	m.AttemptsTotal.WithLabelValues(operation, outcome, errorType).Inc()
	m.DurationSeconds.WithLabelValues(operation, outcome).Observe(duration)
}

// RecordBackoff records the backoff delay before a retry attempt.
//
// Parameters:
//   - operation: Name of operation being retried
//   - delaySeconds: Actual backoff delay in seconds
func (m *RetryMetrics) RecordBackoff(operation string, delaySeconds float64) {
	if m == nil {
		return
	}

	m.BackoffSeconds.WithLabelValues(operation).Observe(delaySeconds)
}

// RecordFinalAttempt records the final number of attempts when operation completes.
//
// Parameters:
//   - operation: Name of operation
//   - outcome: Final outcome ("success" or "failure")
//   - attempts: Total number of attempts made
func (m *RetryMetrics) RecordFinalAttempt(operation, outcome string, attempts int) {
	if m == nil {
		return
	}

	m.FinalAttemptsTotal.WithLabelValues(operation, outcome).Observe(float64(attempts))
}

// Reset resets all retry metrics to zero.
// Primarily used for testing purposes.
func (m *RetryMetrics) Reset() {
	if m == nil {
		return
	}

	m.AttemptsTotal.Reset()
	m.DurationSeconds.Reset()
	m.BackoffSeconds.Reset()
	m.FinalAttemptsTotal.Reset()
}
