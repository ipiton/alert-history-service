// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Alert ConverterDependencies holds optional dependencies for format conversion.
//
// These dependencies enable enhanced features like silence/inhibition status.
// All fields are optional - converter works without them but with reduced functionality.
type ConverterDependencies struct {
	SilenceChecker    SilenceChecker    // Optional: Check if alerts are silenced
	InhibitionChecker InhibitionChecker // Optional: Check if alerts are inhibited
	Logger            *slog.Logger      // Optional: Structured logging
}

// SilenceChecker checks if an alert is silenced.
//
// This interface abstracts the TN-133 Silence Storage to avoid circular dependencies.
type SilenceChecker interface {
	// IsAlertSilenced checks if an alert is currently silenced.
	// Returns list of silence IDs that silence this alert.
	IsAlertSilenced(ctx context.Context, alert *core.Alert) ([]string, error)
}

// InhibitionChecker checks if an alert is inhibited.
//
// This interface abstracts the TN-129 Inhibition State Manager to avoid circular dependencies.
type InhibitionChecker interface {
	// IsAlertInhibited checks if an alert is currently inhibited.
	// Returns list of fingerprints of inhibiting alerts.
	IsAlertInhibited(ctx context.Context, alert *core.Alert) ([]string, error)
}

// ConvertToAlertmanagerFormat converts core.Alert to Alertmanager v2 format.
//
// This is the main conversion function that transforms internal domain models
// to the wire format expected by Prometheus/Grafana/amtool.
//
// Conversion logic:
//   - Labels: Direct copy (map[string]string)
//   - Annotations: Direct copy (map[string]string)
//   - StartsAt: Format as RFC3339
//   - EndsAt: Format as RFC3339 (omit if nil/zero for active alerts)
//   - GeneratorURL: Copy if present
//   - Status: Build from alert status + silence/inhibition checks
//   - Receivers: Extract from alert metadata (placeholder for now)
//   - Fingerprint: Direct copy
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - alerts: List of core.Alert domain models
//   - deps: Optional converter dependencies (silence/inhibition checkers)
//
// Returns:
//   - []AlertmanagerAlert: Alerts in Alertmanager format
//   - error: Conversion error
//
// Example:
//   deps := &ConverterDependencies{
//       SilenceChecker: silenceStorage,
//       InhibitionChecker: inhibitionManager,
//       Logger: logger,
//   }
//   amAlerts, err := ConvertToAlertmanagerFormat(ctx, alerts, deps)
func ConvertToAlertmanagerFormat(
	ctx context.Context,
	alerts []*core.Alert,
	deps *ConverterDependencies,
) ([]AlertmanagerAlert, error) {
	if alerts == nil {
		return []AlertmanagerAlert{}, nil
	}

	result := make([]AlertmanagerAlert, 0, len(alerts))

	for _, alert := range alerts {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("conversion cancelled: %w", ctx.Err())
		default:
		}

		amAlert, err := convertSingleAlert(ctx, alert, deps)
		if err != nil {
			// Log error but continue conversion (best-effort)
			if deps != nil && deps.Logger != nil {
				deps.Logger.Warn("Failed to convert alert",
					"fingerprint", alert.Fingerprint,
					"alertname", alert.AlertName,
					"error", err,
				)
			}
			continue
		}

		result = append(result, amAlert)
	}

	return result, nil
}

// convertSingleAlert converts a single core.Alert to AlertmanagerAlert.
//
// Parameters:
//   - ctx: Context for cancellation
//   - alert: Core alert domain model
//   - deps: Optional dependencies
//
// Returns:
//   - AlertmanagerAlert: Converted alert
//   - error: Conversion error
func convertSingleAlert(
	ctx context.Context,
	alert *core.Alert,
	deps *ConverterDependencies,
) (AlertmanagerAlert, error) {
	// Build alert status
	status, err := buildAlertStatus(ctx, alert, deps)
	if err != nil {
		return AlertmanagerAlert{}, fmt.Errorf("failed to build alert status: %w", err)
	}

	// Format timestamps
	startsAt := alert.StartsAt.Format(time.RFC3339)
	endsAt := ""
	if alert.EndsAt != nil && !alert.EndsAt.IsZero() {
		endsAt = alert.EndsAt.Format(time.RFC3339)
	}

	// Get generator URL
	generatorURL := ""
	if alert.GeneratorURL != nil {
		generatorURL = *alert.GeneratorURL
	}

	// Build receivers list (placeholder - will be enhanced with TN-141 integration)
	receivers := buildReceivers(alert)

	return AlertmanagerAlert{
		Labels:       copyLabels(alert.Labels),
		Annotations:  copyLabels(alert.Annotations),
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		GeneratorURL: generatorURL,
		Status:       status,
		Receivers:    receivers,
		Fingerprint:  alert.Fingerprint,
	}, nil
}

// buildAlertStatus constructs the AlertStatus for an alert.
//
// Status determination logic:
//   1. Check if alert is silenced (via TN-133 if available)
//   2. Check if alert is inhibited (via TN-129 if available)
//   3. Determine state:
//      - "suppressed" if silenced or inhibited
//      - "active" if firing and not suppressed
//      - "resolved" if status is resolved (implied, not used in Alertmanager)
//
// Parameters:
//   - ctx: Context
//   - alert: Core alert
//   - deps: Optional dependencies
//
// Returns:
//   - AlertStatus: Constructed status
//   - error: Error checking silence/inhibition
func buildAlertStatus(
	ctx context.Context,
	alert *core.Alert,
	deps *ConverterDependencies,
) (AlertStatus, error) {
	silencedBy := []string{}
	inhibitedBy := []string{}

	// Check if alert is silenced (via TN-133)
	if deps != nil && deps.SilenceChecker != nil {
		silences, err := deps.SilenceChecker.IsAlertSilenced(ctx, alert)
		if err != nil {
			// Log warning but don't fail (best-effort)
			if deps.Logger != nil {
				deps.Logger.Warn("Failed to check silence status",
					"fingerprint", alert.Fingerprint,
					"error", err,
				)
			}
		} else {
			silencedBy = silences
		}
	}

	// Check if alert is inhibited (via TN-129)
	if deps != nil && deps.InhibitionChecker != nil {
		inhibitors, err := deps.InhibitionChecker.IsAlertInhibited(ctx, alert)
		if err != nil {
			// Log warning but don't fail (best-effort)
			if deps.Logger != nil {
				deps.Logger.Warn("Failed to check inhibition status",
					"fingerprint", alert.Fingerprint,
					"error", err,
				)
			}
		} else {
			inhibitedBy = inhibitors
		}
	}

	// Determine alert state
	state := "active"
	if len(silencedBy) > 0 || len(inhibitedBy) > 0 {
		state = "suppressed"
	}

	// Override state if alert is resolved
	if alert.Status == core.StatusResolved {
		state = "active" // Alertmanager doesn't have "resolved" state, resolved alerts just have endsAt set
	}

	return AlertStatus{
		State:       state,
		SilencedBy:  silencedBy,
		InhibitedBy: inhibitedBy,
	}, nil
}

// buildReceivers extracts receiver names from alert metadata.
//
// Current implementation returns placeholder receivers.
// This will be enhanced when TN-141 (Multi-Receiver Support) routing is integrated.
//
// Parameters:
//   - alert: Core alert
//
// Returns:
//   - []string: List of receiver names
func buildReceivers(alert *core.Alert) []string {
	// Placeholder: Extract from alert labels if available
	// In full implementation, this would query the routing tree
	// to determine which receivers match this alert.

	// Check if alert has receiver label
	if receiver, ok := alert.Labels["receiver"]; ok {
		return []string{receiver}
	}

	// Default receiver
	return []string{"default"}
}

// copyLabels creates a deep copy of label/annotation map.
//
// This ensures the original alert is not modified and prevents
// potential nil pointer issues.
//
// Parameters:
//   - labels: Source labels map
//
// Returns:
//   - map[string]string: Copied labels
func copyLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return map[string]string{}
	}

	result := make(map[string]string, len(labels))
	for k, v := range labels {
		result[k] = v
	}
	return result
}

// BuildAlertmanagerListResponse constructs the complete response for GET /api/v2/alerts.
//
// This wraps the alert list in the standard Alertmanager API v2 response format
// with pagination metadata.
//
// Parameters:
//   - alerts: Converted Alertmanager alerts
//   - total: Total count of alerts matching the filter
//   - page: Current page number
//   - limit: Results per page
//
// Returns:
//   - *AlertmanagerListResponse: Complete response ready for JSON encoding
func BuildAlertmanagerListResponse(
	alerts []AlertmanagerAlert,
	total, page, limit int,
) *AlertmanagerListResponse {
	return &AlertmanagerListResponse{
		Status: "success",
		Data: &AlertmanagerListData{
			Alerts: alerts,
			Total:  total,
			Page:   page,
			Limit:  limit,
		},
	}
}

// BuildErrorResponse constructs an error response.
//
// Parameters:
//   - errorMessage: Error description
//
// Returns:
//   - *AlertmanagerListResponse: Error response
func BuildErrorResponse(errorMessage string) *AlertmanagerListResponse {
	return &AlertmanagerListResponse{
		Status: "error",
		Error:  errorMessage,
	}
}
