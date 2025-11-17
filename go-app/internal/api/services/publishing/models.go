package publishing

import "time"

// ModeInfo represents current publishing mode information returned by the API.
//
// This struct contains both basic mode information (always present) and enhanced
// metrics (present when ModeManager is available).
//
// Basic fields indicate the current operational mode of the publishing system:
// - "normal" mode: System is publishing alerts to targets (enabled_targets > 0)
// - "metrics-only" mode: System is only collecting metrics (enabled_targets == 0)
//
// Enhanced fields provide additional telemetry about mode transitions and duration,
// useful for monitoring and troubleshooting (available when using TN-060 ModeManager).
type ModeInfo struct {
	// Basic fields (always present)

	// Mode indicates current publishing mode: "normal" or "metrics-only"
	Mode string `json:"mode"`

	// TargetsAvailable indicates whether any publishing targets are available
	TargetsAvailable bool `json:"targets_available"`

	// EnabledTargets is the count of currently enabled publishing targets
	EnabledTargets int `json:"enabled_targets"`

	// MetricsOnlyActive indicates whether system is in metrics-only mode
	MetricsOnlyActive bool `json:"metrics_only_active"`

	// Enhanced fields (present if ModeManager available, TN-060)

	// TransitionCount is the total number of mode transitions since startup.
	// Omitted if ModeManager is not available.
	TransitionCount int64 `json:"transition_count,omitempty"`

	// CurrentModeDurationSeconds is the duration in seconds that system has been
	// in the current mode. Omitted if ModeManager is not available.
	CurrentModeDurationSeconds float64 `json:"current_mode_duration_seconds,omitempty"`

	// LastTransitionTime is the RFC3339 timestamp of the last mode transition.
	// Omitted if ModeManager is not available or no transitions have occurred.
	LastTransitionTime time.Time `json:"last_transition_time,omitempty"`

	// LastTransitionReason describes why the last mode transition occurred.
	// Possible values:
	// - "targets_available": Transition to normal (targets became available)
	// - "no_enabled_targets": Transition to metrics-only (all targets disabled)
	// - "targets_disabled": Transition to metrics-only (targets manually disabled)
	// - "startup": Initial mode at system startup
	// Omitted if ModeManager is not available or no transitions have occurred.
	LastTransitionReason string `json:"last_transition_reason,omitempty"`
}

// ErrorResponse represents an API error response.
//
// This struct provides structured error information including a request ID
// for tracing and a timestamp for auditing.
//
// All error responses from the publishing mode endpoint use this structure
// to ensure consistency and enable effective troubleshooting.
type ErrorResponse struct {
	// Error is the HTTP status text (e.g., "Internal Server Error", "Too Many Requests")
	Error string `json:"error"`

	// Message is a human-readable error message explaining what went wrong
	Message string `json:"message"`

	// RequestID is a unique identifier for this request, useful for tracing
	// and correlating logs across services
	RequestID string `json:"request_id"`

	// Timestamp is the RFC3339 timestamp when the error occurred
	Timestamp time.Time `json:"timestamp"`
}
