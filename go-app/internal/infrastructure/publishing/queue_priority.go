package publishing

import (
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// determinePriority classifies jobs into priority tiers based on alert severity and status
//
// Priority Rules:
//   - HIGH: Critical firing alerts (severity=critical && status=firing)
//   - HIGH: LLM classification severity=critical
//   - LOW: Resolved alerts (status=resolved)
//   - LOW: Info severity alerts (severity=info)
//   - MEDIUM: All other alerts (default)
//
// Parameters:
//   - enrichedAlert: Alert with optional LLM classification
//
// Returns:
//   - Priority: PriorityHigh (0), PriorityMedium (1), or PriorityLow (2)
//
// Example:
//
//	priority := determinePriority(enrichedAlert)
//	if priority == PriorityHigh {
//	    // Process immediately
//	}
func determinePriority(enrichedAlert *core.EnrichedAlert) Priority {
	if enrichedAlert == nil || enrichedAlert.Alert == nil {
		return PriorityMedium // Safe default
	}

	alert := enrichedAlert.Alert

	// HIGH priority: Critical firing alerts
	if alert.Severity == "critical" && alert.Status == "firing" {
		return PriorityHigh
	}

	// HIGH priority: LLM confidence = critical
	if enrichedAlert.Classification != nil {
		if enrichedAlert.Classification.Severity == "critical" {
			return PriorityHigh
		}
	}

	// LOW priority: Resolved alerts
	if alert.Status == "resolved" {
		return PriorityLow
	}

	// LOW priority: Info severity
	if alert.Severity == "info" {
		return PriorityLow
	}

	// DEFAULT: MEDIUM priority for everything else
	return PriorityMedium
}
