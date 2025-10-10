package webhook

import (
	"encoding/json"
	"fmt"
)

// WebhookType represents the detected webhook format type.
type WebhookType string

const (
	// WebhookTypeAlertmanager represents Prometheus Alertmanager webhook format
	WebhookTypeAlertmanager WebhookType = "alertmanager"

	// WebhookTypeGeneric represents a generic webhook format
	WebhookTypeGeneric WebhookType = "generic"

	// WebhookTypePrometheus represents Prometheus direct alert format (future use)
	WebhookTypePrometheus WebhookType = "prometheus"
)

// WebhookDetector defines the interface for detecting webhook types.
type WebhookDetector interface {
	// Detect analyzes the payload and determines the webhook type.
	Detect(payload []byte) (WebhookType, error)
}

// webhookDetector implements WebhookDetector.
type webhookDetector struct{}

// NewWebhookDetector creates a new webhook type detector.
func NewWebhookDetector() WebhookDetector {
	return &webhookDetector{}
}

// Detect analyzes the webhook payload and determines its type.
//
// Detection logic:
//  1. Try to parse as JSON
//  2. Check for Alertmanager-specific fields (version, groupKey, receiver)
//  3. If Alertmanager fields present → WebhookTypeAlertmanager
//  4. Otherwise → WebhookTypeGeneric
//
// Parameters:
//   - payload: Raw webhook payload bytes
//
// Returns:
//   - WebhookType: Detected webhook type
//   - error: Detection error (invalid JSON, empty payload)
func (d *webhookDetector) Detect(payload []byte) (WebhookType, error) {
	if len(payload) == 0 {
		return "", fmt.Errorf("webhook payload is empty")
	}

	// Parse as generic JSON to inspect structure
	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return "", fmt.Errorf("invalid JSON payload: %w", err)
	}

	// Check for Alertmanager-specific fields
	if d.isAlertmanagerWebhook(data) {
		return WebhookTypeAlertmanager, nil
	}

	// Default to generic format
	return WebhookTypeGeneric, nil
}

// isAlertmanagerWebhook checks if the payload has Alertmanager-specific fields.
//
// Alertmanager webhooks have these characteristic fields:
//   - "version" (string) - Alertmanager API version (typically "4")
//   - "groupKey" (string) - Alert grouping key
//   - "receiver" (string) - Receiver name
//   - "alerts" (array) - Array of alerts
//
// We require at least 2 of these fields to confidently detect Alertmanager format.
func (d *webhookDetector) isAlertmanagerWebhook(data map[string]interface{}) bool {
	matchCount := 0

	// Check for "version" field (Alertmanager-specific)
	if version, ok := data["version"]; ok {
		if versionStr, ok := version.(string); ok && versionStr != "" {
			matchCount++
		}
	}

	// Check for "groupKey" field (Alertmanager-specific)
	if groupKey, ok := data["groupKey"]; ok {
		if groupKeyStr, ok := groupKey.(string); ok && groupKeyStr != "" {
			matchCount++
		}
	}

	// Check for "receiver" field (Alertmanager-specific)
	if receiver, ok := data["receiver"]; ok {
		if receiverStr, ok := receiver.(string); ok && receiverStr != "" {
			matchCount++
		}
	}

	// Check for "alerts" array structure
	if alerts, ok := data["alerts"]; ok {
		if alertsArray, ok := alerts.([]interface{}); ok && len(alertsArray) > 0 {
			// Verify alerts have Alertmanager structure (status, labels, annotations)
			if d.hasAlertmanagerAlertStructure(alertsArray) {
				matchCount++
			}
		}
	}

	// Require at least 2 Alertmanager-specific fields to confidently detect
	return matchCount >= 2
}

// hasAlertmanagerAlertStructure checks if alerts have Alertmanager-specific structure.
func (d *webhookDetector) hasAlertmanagerAlertStructure(alerts []interface{}) bool {
	if len(alerts) == 0 {
		return false
	}

	// Check first alert for Alertmanager fields
	firstAlert, ok := alerts[0].(map[string]interface{})
	if !ok {
		return false
	}

	// Alertmanager alerts have: status, labels, annotations, startsAt
	hasStatus := false
	hasLabels := false

	if status, ok := firstAlert["status"]; ok {
		if statusStr, ok := status.(string); ok && (statusStr == "firing" || statusStr == "resolved") {
			hasStatus = true
		}
	}

	if labels, ok := firstAlert["labels"]; ok {
		if labelsMap, ok := labels.(map[string]interface{}); ok && len(labelsMap) > 0 {
			hasLabels = true
		}
	}

	// Annotations are optional in Alertmanager format, so we don't check them

	// Require status + labels (annotations optional)
	return hasStatus && hasLabels
}

// DetectWithHints analyzes payload with additional hints from HTTP headers.
//
// This method can use Content-Type, User-Agent, or custom headers to assist detection.
// For now, it falls back to standard detection, but can be extended.
//
// Parameters:
//   - payload: Raw webhook payload bytes
//   - contentType: HTTP Content-Type header value
//   - userAgent: HTTP User-Agent header value
//
// Returns:
//   - WebhookType: Detected webhook type
//   - error: Detection error
func (d *webhookDetector) DetectWithHints(payload []byte, contentType string, userAgent string) (WebhookType, error) {
	// Future: Use Content-Type or User-Agent hints
	// For example, User-Agent might contain "Alertmanager"

	// For now, use standard detection
	return d.Detect(payload)
}
