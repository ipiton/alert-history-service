package publishing

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPublishingMetrics(t *testing.T) {
	metrics := NewPublishingMetrics()

	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.AlertsPublished)
	assert.NotNil(t, metrics.AlertsPublishErrors)
	assert.NotNil(t, metrics.QueueSize)
	assert.NotNil(t, metrics.CircuitBreakerState)
	assert.NotNil(t, metrics.DiscoveredTargets)
}

func TestRecordPublishSuccess(t *testing.T) {
	metrics := NewPublishingMetrics()

	// Should not panic
	metrics.RecordPublishSuccess("test-target", "webhook", 0.5)
}

func TestRecordPublishError(t *testing.T) {
	metrics := NewPublishingMetrics()

	// Should not panic
	metrics.RecordPublishError("test-target", "webhook", "timeout")
}

func TestRecordQueueSubmission(t *testing.T) {
	metrics := NewPublishingMetrics()

	// Record success
	metrics.RecordQueueSubmission(true)

	// Record failure
	metrics.RecordQueueSubmission(false)
}

func TestUpdateQueueMetrics(t *testing.T) {
	metrics := NewPublishingMetrics()

	metrics.UpdateQueueMetrics(10, 100)
	// Metrics should be updated (no panic)
}

func TestRecordCircuitBreakerState(t *testing.T) {
	metrics := NewPublishingMetrics()

	metrics.RecordCircuitBreakerState("test-target", StateClosed)
	metrics.RecordCircuitBreakerState("test-target", StateOpen)
	metrics.RecordCircuitBreakerState("test-target", StateHalfOpen)
}

func TestRecordCircuitBreakerTrip(t *testing.T) {
	metrics := NewPublishingMetrics()

	metrics.RecordCircuitBreakerTrip("test-target")
}

func TestUpdateTargetCounts(t *testing.T) {
	metrics := NewPublishingMetrics()

	metrics.UpdateTargetCounts(5, 3)
}

func TestRecordTargetRefresh(t *testing.T) {
	metrics := NewPublishingMetrics()

	// Success
	metrics.RecordTargetRefresh(true, 1.5)

	// Failure
	metrics.RecordTargetRefresh(false, 2.0)
}

func TestRecordFormatOperation(t *testing.T) {
	metrics := NewPublishingMetrics()

	// Success
	metrics.RecordFormatOperation("webhook", true, 0.001)

	// Failure
	metrics.RecordFormatOperation("slack", false, 0.002)
}
