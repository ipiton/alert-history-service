package services

import (
	"log/slog"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// SimpleFilterEngine is a basic implementation of FilterEngine
type SimpleFilterEngine struct {
	logger  *slog.Logger
	metrics *metrics.FilterMetrics
}

// NewSimpleFilterEngine creates a new simple filter engine
func NewSimpleFilterEngine(logger *slog.Logger) *SimpleFilterEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &SimpleFilterEngine{
		logger:  logger,
		metrics: metrics.NewFilterMetrics(),
	}
}

// NewSimpleFilterEngineWithMetrics creates a new simple filter engine with custom metrics
// If filterMetrics is nil, metrics collection will be disabled
func NewSimpleFilterEngineWithMetrics(logger *slog.Logger, filterMetrics *metrics.FilterMetrics) *SimpleFilterEngine {
	if logger == nil {
		logger = slog.Default()
	}
	// Don't create metrics if nil is passed - allow disabling metrics
	return &SimpleFilterEngine{
		logger:  logger,
		metrics: filterMetrics,
	}
}

// ShouldBlock determines if an alert should be blocked
// Currently implements basic rules:
// - Block alerts with severity="noise" (if classified)
// - Block test alerts (alertname contains "test" or "Test")
func (f *SimpleFilterEngine) ShouldBlock(alert *core.Alert, classification *core.ClassificationResult) (bool, string) {
	start := time.Now()

	blocked, reason := f.shouldBlockInternal(alert, classification)

	// Record metrics (only if metrics are enabled)
	if f.metrics != nil {
		duration := time.Since(start).Seconds()
		result := "allowed"
		if blocked {
			result = "blocked"
			f.metrics.RecordBlockedAlert(reason)
		}
		f.metrics.RecordAlertFiltered(result)
		f.metrics.RecordFilterDuration(duration, result)
	}

	return blocked, reason
}

// shouldBlockInternal contains the actual filtering logic
func (f *SimpleFilterEngine) shouldBlockInternal(alert *core.Alert, classification *core.ClassificationResult) (bool, string) {
	// Rule 1: Block test alerts
	if isTestAlert(alert) {
		return true, "test_alert"
	}

	// Rule 2: Block noise alerts (if we have classification)
	if classification != nil && classification.Severity == core.SeverityNoise {
		return true, "noise"
	}

	// Rule 3: Block alerts with very low confidence (< 0.3)
	if classification != nil && classification.Confidence < 0.3 {
		return true, "low_confidence"
	}

	// Default: allow
	return false, ""
}

// isTestAlert checks if alert is a test alert
func isTestAlert(alert *core.Alert) bool {
	// Check alert name
	if containsTest(alert.AlertName) {
		return true
	}

	// Check labels
	if value, ok := alert.Labels["alertname"]; ok && containsTest(value) {
		return true
	}

	if value, ok := alert.Labels["environment"]; ok && (value == "test" || value == "testing") {
		return true
	}

	return false
}

// containsTest checks if string contains "test" or "Test"
func containsTest(s string) bool {
	return len(s) >= 4 && ((s[0] == 't' || s[0] == 'T') &&
		(s[1] == 'e' || s[1] == 'E') &&
		(s[2] == 's' || s[2] == 'S') &&
		(s[3] == 't' || s[3] == 'T'))
}
