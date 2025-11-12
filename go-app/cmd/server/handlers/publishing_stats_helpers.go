package handlers

import (
	"fmt"
	"strings"
)

// ============================================================================
// Metric Extraction Helpers
// ============================================================================

// getMetricValue safely gets metric value by key (returns 0.0 if not found).
func getMetricValue(metrics map[string]float64, key string) float64 {
	if val, ok := metrics[key]; ok {
		return val
	}
	return 0.0
}

// countHealthyTargets counts targets with health_status=1.0 (healthy).
func countHealthyTargets(metrics map[string]float64) int {
	count := 0
	for key, value := range metrics {
		if strings.Contains(key, "health_status") && value == 1.0 {
			count++
		}
	}
	return count
}

// countUnhealthyTargets counts targets with health_status=3.0 (unhealthy).
func countUnhealthyTargets(metrics map[string]float64) int {
	count := 0
	for key, value := range metrics {
		if strings.Contains(key, "health_status") && value == 3.0 {
			count++
		}
	}
	return count
}

// calculateSuccessRate computes overall success rate from job metrics.
func calculateSuccessRate(metrics map[string]float64) float64 {
	completed := getMetricValue(metrics, "jobs_completed_total")
	submitted := getMetricValue(metrics, "jobs_submitted_total")

	if submitted == 0 {
		return 100.0 // No jobs = 100% success (neutral)
	}

	return (completed / submitted) * 100.0
}

// ============================================================================
// Formatting Helpers
// ============================================================================

// formatFloatHelper is helper for formatFloat.
func formatFloatHelper(f float64, format string) string {
	return fmt.Sprintf(format, f)
}

// generateHealthMessage creates human-readable health message.
func generateHealthMessage(status string, unhealthyCount int, totalTargets int) string {
	switch status {
	case "healthy":
		return "All systems operational"
	case "degraded":
		if unhealthyCount > 0 {
			return fmt.Sprintf("%d of %d targets unhealthy", unhealthyCount, totalTargets)
		}
		return "Some collectors reporting errors"
	case "unhealthy":
		return fmt.Sprintf("Critical: %d of %d targets unhealthy (>50%%)", unhealthyCount, totalTargets)
	default:
		return "Unknown status"
	}
}
