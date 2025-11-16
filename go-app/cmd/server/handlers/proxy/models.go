// Package proxy provides handlers for intelligent proxy webhook endpoint.
package proxy

import (
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ProxyWebhookRequest represents the incoming proxy webhook payload.
// Compatible with Alertmanager v0.25+ webhook format.
type ProxyWebhookRequest struct {
	// Required fields
	Alerts   []AlertPayload `json:"alerts" validate:"required,min=1,max=100"`
	Receiver string         `json:"receiver" validate:"required"`

	// Optional fields (Alertmanager metadata)
	Status            string            `json:"status,omitempty"`
	Version           string            `json:"version,omitempty"`
	GroupKey          string            `json:"groupKey,omitempty"`
	GroupLabels       map[string]string `json:"groupLabels,omitempty"`
	CommonLabels      map[string]string `json:"commonLabels,omitempty"`
	CommonAnnotations map[string]string `json:"commonAnnotations,omitempty"`
	ExternalURL       string            `json:"externalURL,omitempty"`
	TruncatedAlerts   int               `json:"truncatedAlerts,omitempty"`
}

// AlertPayload represents a single alert in the webhook.
type AlertPayload struct {
	Status       string            `json:"status" validate:"required,oneof=firing resolved"`
	Labels       map[string]string `json:"labels" validate:"required,min=1"`
	Annotations  map[string]string `json:"annotations,omitempty"`
	StartsAt     time.Time         `json:"startsAt" validate:"required"`
	EndsAt       time.Time         `json:"endsAt,omitempty"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
	Fingerprint  string            `json:"fingerprint,omitempty"` // Auto-generated if empty
}

// ProxyWebhookResponse represents the response for proxy webhook requests.
type ProxyWebhookResponse struct {
	// Overall status
	Status         string        `json:"status"`          // success, partial, failed
	Message        string        `json:"message"`
	Timestamp      time.Time     `json:"timestamp"`
	ProcessingTime time.Duration `json:"processing_time_ms"`

	// Alerts processing summary
	AlertsSummary AlertsProcessingSummary `json:"alerts_summary"`

	// Per-alert results
	AlertResults []AlertProcessingResult `json:"alert_results"`

	// Publishing summary
	PublishingSummary PublishingSummary `json:"publishing_summary"`
}

// AlertsProcessingSummary summarizes alert processing.
type AlertsProcessingSummary struct {
	TotalReceived   int `json:"total_received"`
	TotalProcessed  int `json:"total_processed"`
	TotalClassified int `json:"total_classified"`
	TotalFiltered   int `json:"total_filtered"`
	TotalPublished  int `json:"total_published"`
	TotalFailed     int `json:"total_failed"`
}

// AlertProcessingResult contains per-alert processing details.
type AlertProcessingResult struct {
	// Alert identification
	Fingerprint string `json:"fingerprint"`
	AlertName   string `json:"alert_name"`
	Status      string `json:"status"` // success, filtered, failed

	// Classification details
	Classification     *ClassificationResult `json:"classification,omitempty"`
	ClassificationTime time.Duration         `json:"classification_time_ms,omitempty"`

	// Filtering details
	FilterAction string `json:"filter_action,omitempty"` // allow, deny
	FilterReason string `json:"filter_reason,omitempty"`

	// Publishing details
	PublishingResults []TargetPublishingResult `json:"publishing_results,omitempty"`

	// Error details (if failed)
	ErrorMessage string `json:"error_message,omitempty"`
}

// ClassificationResult contains classification details.
type ClassificationResult struct {
	Severity        string    `json:"severity"`        // critical, warning, info, unknown
	Category        string    `json:"category"`        // infrastructure, application, network, etc.
	Confidence      float64   `json:"confidence"`      // 0.0-1.0
	Source          string    `json:"source"`          // llm, fallback, default
	Recommendations []string  `json:"recommendations,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// TargetPublishingResult contains per-target publishing details.
type TargetPublishingResult struct {
	TargetName     string        `json:"target_name"`
	TargetType     string        `json:"target_type"` // rootly, pagerduty, slack, generic
	Success        bool          `json:"success"`
	StatusCode     int           `json:"status_code,omitempty"`
	ErrorMessage   string        `json:"error_message,omitempty"`
	ErrorCode      string        `json:"error_code,omitempty"` // TIMEOUT, RATE_LIMIT, etc.
	RetryCount     int           `json:"retry_count,omitempty"`
	ProcessingTime time.Duration `json:"processing_time_ms"`
}

// PublishingSummary summarizes publishing results.
type PublishingSummary struct {
	TotalTargets      int           `json:"total_targets"`
	SuccessfulTargets int           `json:"successful_targets"`
	FailedTargets     int           `json:"failed_targets"`
	TotalPublishTime  time.Duration `json:"total_publish_time_ms"`
}

// ErrorResponse is the standard error response.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error details.
type ErrorDetail struct {
	Code      string             `json:"code"`       // Error code
	Message   string             `json:"message"`    // Human-readable message
	Details   []FieldErrorDetail `json:"details,omitempty"` // Field-level errors
	Timestamp time.Time          `json:"timestamp"`
	RequestID string             `json:"request_id"`
}

// FieldErrorDetail contains field-level error details.
type FieldErrorDetail struct {
	Field string `json:"field"` // Field path (e.g., "alerts[0].status")
	Error string `json:"error"` // Error message
}

// Error codes
const (
	ErrCodeValidation         = "VALIDATION_ERROR"
	ErrCodeAuthentication     = "AUTHENTICATION_ERROR"
	ErrCodeAuthorization      = "AUTHORIZATION_ERROR"
	ErrCodeRateLimit          = "RATE_LIMIT_ERROR"
	ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
	ErrCodeInternal           = "INTERNAL_ERROR"
	ErrCodeTimeout            = "TIMEOUT_ERROR"
	ErrCodePayloadTooLarge    = "PAYLOAD_TOO_LARGE"
	ErrCodeUnsupportedMedia   = "UNSUPPORTED_MEDIA_TYPE"
)

// FilterAction represents a filter decision.
type FilterAction string

const (
	// FilterActionAllow indicates the alert should be published
	FilterActionAllow FilterAction = "allow"
	// FilterActionDeny indicates the alert should be filtered out
	FilterActionDeny FilterAction = "deny"
)

// ConvertToAlert converts AlertPayload to core.Alert.
func (ap *AlertPayload) ConvertToAlert() (*core.Alert, error) {
	// Parse status
	status := core.StatusFiring
	if ap.Status == "resolved" {
		status = core.StatusResolved
	}

	// Generate fingerprint if not provided
	fingerprint := ap.Fingerprint
	if fingerprint == "" {
		// Use TN-036 deduplication service to generate fingerprint
		fingerprint = generateFingerprint(ap.Labels)
	}

	// Parse endsAt (if present)
	var endsAt *time.Time
	if !ap.EndsAt.IsZero() {
		endsAt = &ap.EndsAt
	}

	// Parse generatorURL (if present)
	var generatorURL *string
	if ap.GeneratorURL != "" {
		generatorURL = &ap.GeneratorURL
	}

	alert := &core.Alert{
		Fingerprint:  fingerprint,
		AlertName:    ap.Labels["alertname"],
		Status:       status,
		Labels:       ap.Labels,
		Annotations:  ap.Annotations,
		StartsAt:     ap.StartsAt,
		EndsAt:       endsAt,
		GeneratorURL: generatorURL,
		Timestamp:    &ap.StartsAt,
	}

	return alert, nil
}

// generateFingerprint generates a fingerprint from labels.
// TODO: Integrate with TN-036 deduplication service for proper fingerprinting.
func generateFingerprint(labels map[string]string) string {
	// Simplified implementation - should use TN-036's FNV-64a hash
	// For now, use a basic concatenation approach
	fingerprint := ""
	for k, v := range labels {
		fingerprint += k + "=" + v + ":"
	}
	if len(fingerprint) > 64 {
		fingerprint = fingerprint[:64]
	}
	return fingerprint
}

// ConfidenceBucket returns a human-readable confidence bucket.
func (cr *ClassificationResult) ConfidenceBucket() string {
	if cr.Confidence >= 0.8 {
		return "high"
	} else if cr.Confidence >= 0.5 {
		return "medium"
	}
	return "low"
}
