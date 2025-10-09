package services

import (
	"log/slog"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// SimpleFilterEngine is a basic implementation of FilterEngine
// TODO: Implement full filter rules engine
type SimpleFilterEngine struct {
	logger *slog.Logger
}

// NewSimpleFilterEngine creates a new simple filter engine
func NewSimpleFilterEngine(logger *slog.Logger) *SimpleFilterEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &SimpleFilterEngine{
		logger: logger,
	}
}

// ShouldBlock determines if an alert should be blocked
// Currently implements basic rules:
// - Block alerts with severity="noise" (if classified)
// - Block test alerts (alertname contains "test" or "Test")
func (f *SimpleFilterEngine) ShouldBlock(alert *core.Alert, classification *core.ClassificationResult) (bool, string) {
	// Rule 1: Block test alerts
	if isTestAlert(alert) {
		return true, "test alert"
	}

	// Rule 2: Block noise alerts (if we have classification)
	if classification != nil && classification.Severity == core.SeverityNoise {
		return true, "noise alert (LLM classified as noise)"
	}

	// Rule 3: Block alerts with very low confidence (< 0.3)
	if classification != nil && classification.Confidence < 0.3 {
		return true, "low confidence classification"
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
