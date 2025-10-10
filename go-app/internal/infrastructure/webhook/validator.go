package webhook

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
)

// ValidationError represents a single validation error for a webhook field.
type ValidationError struct {
	Field   string      `json:"field"`             // Field name that failed validation
	Message string      `json:"message"`           // Human-readable error message
	Value   interface{} `json:"value,omitempty"`   // The value that failed validation (optional)
	Tag     string      `json:"tag,omitempty"`     // Validation tag that failed (e.g., "required", "url")
}

// ValidationResult contains the result of webhook validation.
type ValidationResult struct {
	Valid  bool               `json:"valid"`             // Whether validation passed
	Errors []*ValidationError `json:"errors,omitempty"`  // List of validation errors (if any)
}

// WebhookValidator defines the interface for webhook validation.
type WebhookValidator interface {
	// ValidateAlertmanager validates an Alertmanager webhook payload.
	ValidateAlertmanager(webhook *AlertmanagerWebhook) *ValidationResult

	// ValidateGeneric validates a generic webhook payload.
	ValidateGeneric(data map[string]interface{}) *ValidationResult
}

// webhookValidator implements WebhookValidator using go-playground/validator.
type webhookValidator struct {
	validate *validator.Validate
}

// NewWebhookValidator creates a new webhook validator with custom validation rules.
func NewWebhookValidator() WebhookValidator {
	v := validator.New()

	// Register custom validation functions
	_ = v.RegisterValidation("alertname", validateAlertname)
	_ = v.RegisterValidation("severity", validateSeverity)
	_ = v.RegisterValidation("confidence", validateConfidence)
	_ = v.RegisterValidation("webhook_status", validateWebhookStatus)

	return &webhookValidator{
		validate: v,
	}
}

// ValidateAlertmanager validates an Alertmanager webhook payload.
func (v *webhookValidator) ValidateAlertmanager(webhook *AlertmanagerWebhook) *ValidationResult {
	if webhook == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{
				{
					Field:   "webhook",
					Message: "webhook payload is nil",
				},
			},
		}
	}

	result := &ValidationResult{
		Valid:  true,
		Errors: []*ValidationError{},
	}

	// Validate required fields
	if webhook.Version == "" {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "version",
			Message: "version is required",
			Tag:     "required",
		})
	}

	if webhook.Status == "" {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "status",
			Message: "status is required",
			Tag:     "required",
		})
	} else if !isValidWebhookStatus(webhook.Status) {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "status",
			Message: fmt.Sprintf("invalid status '%s', must be 'firing' or 'resolved'", webhook.Status),
			Value:   webhook.Status,
			Tag:     "webhook_status",
		})
	}

	if webhook.GroupKey == "" {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "groupKey",
			Message: "groupKey is required",
			Tag:     "required",
		})
	}

	// Validate ExternalURL format
	if webhook.ExternalURL != "" {
		if _, err := url.ParseRequestURI(webhook.ExternalURL); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "externalURL",
				Message: fmt.Sprintf("invalid URL format: %v", err),
				Value:   webhook.ExternalURL,
				Tag:     "url",
			})
		}
	}

	// Validate alerts array
	if len(webhook.Alerts) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "alerts",
			Message: "alerts array cannot be empty",
			Tag:     "required",
		})
	} else {
		// Validate each alert
		for i, alert := range webhook.Alerts {
			alertErrors := v.validateAlert(&alert, i)
			result.Errors = append(result.Errors, alertErrors...)
			if len(alertErrors) > 0 {
				result.Valid = false
			}
		}
	}

	// Check for truncated alerts
	if webhook.TruncatedAlerts < 0 {
		result.Valid = false
		result.Errors = append(result.Errors, &ValidationError{
			Field:   "truncatedAlerts",
			Message: "truncatedAlerts cannot be negative",
			Value:   webhook.TruncatedAlerts,
		})
	}

	return result
}

// validateAlert validates a single alert within an Alertmanager webhook.
func (v *webhookValidator) validateAlert(alert *AlertmanagerAlert, index int) []*ValidationError {
	errors := []*ValidationError{}
	prefix := fmt.Sprintf("alerts[%d]", index)

	// Validate required fields
	if alert.Status == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.status", prefix),
			Message: "status is required",
			Tag:     "required",
		})
	} else if !isValidWebhookStatus(alert.Status) {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.status", prefix),
			Message: fmt.Sprintf("invalid status '%s', must be 'firing' or 'resolved'", alert.Status),
			Value:   alert.Status,
			Tag:     "webhook_status",
		})
	}

	// Validate alertname label
	if alertname, ok := alert.Labels["alertname"]; !ok || alertname == "" {
		errors = append(errors, &ValidationError{
			Field:   fmt.Sprintf("%s.labels.alertname", prefix),
			Message: "alertname label is required",
			Tag:     "required",
		})
	}

	// Validate severity (if present)
	if severity, ok := alert.Labels["severity"]; ok {
		if !isValidSeverity(severity) {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.labels.severity", prefix),
				Message: fmt.Sprintf("invalid severity '%s', must be one of: critical, warning, info, debug", severity),
				Value:   severity,
				Tag:     "severity",
			})
		}
	}

	// Validate timestamps
	if !alert.StartsAt.IsZero() && !alert.EndsAt.IsZero() {
		if alert.EndsAt.Before(alert.StartsAt) {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.endsAt", prefix),
				Message: "endsAt cannot be before startsAt",
				Value:   alert.EndsAt,
			})
		}
	}

	// Validate GeneratorURL format (if present)
	if alert.GeneratorURL != "" {
		if _, err := url.ParseRequestURI(alert.GeneratorURL); err != nil {
			errors = append(errors, &ValidationError{
				Field:   fmt.Sprintf("%s.generatorURL", prefix),
				Message: fmt.Sprintf("invalid URL format: %v", err),
				Value:   alert.GeneratorURL,
				Tag:     "url",
			})
		}
	}

	return errors
}

// ValidateGeneric validates a generic webhook payload.
func (v *webhookValidator) ValidateGeneric(data map[string]interface{}) *ValidationResult {
	if data == nil {
		return &ValidationResult{
			Valid: false,
			Errors: []*ValidationError{
				{
					Field:   "data",
					Message: "webhook data is nil",
				},
			},
		}
	}

	result := &ValidationResult{
		Valid:  true,
		Errors: []*ValidationError{},
	}

	// Check for required fields in generic webhook
	requiredFields := []string{"alertname", "status"}
	for _, field := range requiredFields {
		if value, ok := data[field]; !ok || value == nil || value == "" {
			result.Valid = false
			result.Errors = append(result.Errors, &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("%s is required", field),
				Tag:     "required",
			})
		}
	}

	// Validate status if present
	if status, ok := data["status"].(string); ok {
		if !isValidWebhookStatus(status) {
			result.Valid = false
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "status",
				Message: fmt.Sprintf("invalid status '%s', must be 'firing' or 'resolved'", status),
				Value:   status,
				Tag:     "webhook_status",
			})
		}
	}

	// Validate severity if present
	if severity, ok := data["severity"].(string); ok {
		if !isValidSeverity(severity) {
			result.Valid = false
			result.Errors = append(result.Errors, &ValidationError{
				Field:   "severity",
				Message: fmt.Sprintf("invalid severity '%s'", severity),
				Value:   severity,
				Tag:     "severity",
			})
		}
	}

	return result
}

// Custom validation functions

func validateAlertname(fl validator.FieldLevel) bool {
	alertname := fl.Field().String()
	return alertname != "" && len(alertname) <= 256
}

func validateSeverity(fl validator.FieldLevel) bool {
	severity := fl.Field().String()
	return isValidSeverity(severity)
}

func validateConfidence(fl validator.FieldLevel) bool {
	confidence := fl.Field().Float()
	return confidence >= 0.0 && confidence <= 1.0
}

func validateWebhookStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	return isValidWebhookStatus(status)
}

// Helper functions

func isValidSeverity(severity string) bool {
	validSeverities := map[string]bool{
		"critical": true,
		"warning":  true,
		"info":     true,
		"debug":    true,
	}
	return validSeverities[strings.ToLower(severity)]
}

func isValidWebhookStatus(status string) bool {
	return status == "firing" || status == "resolved"
}

// AlertmanagerWebhook represents the structure of an Alertmanager webhook payload.
// This is a minimal definition for validation purposes.
// Full implementation will be in TN-041 (Alertmanager Parser).
type AlertmanagerWebhook struct {
	Version           string                 `json:"version" validate:"required"`
	GroupKey          string                 `json:"groupKey" validate:"required"`
	TruncatedAlerts   int                    `json:"truncatedAlerts"`
	Status            string                 `json:"status" validate:"required,webhook_status"`
	Receiver          string                 `json:"receiver"`
	GroupLabels       map[string]string      `json:"groupLabels"`
	CommonLabels      map[string]string      `json:"commonLabels"`
	CommonAnnotations map[string]string      `json:"commonAnnotations"`
	ExternalURL       string                 `json:"externalURL" validate:"omitempty,url"`
	Alerts            []AlertmanagerAlert    `json:"alerts" validate:"required,min=1"`
}

// AlertmanagerAlert represents a single alert in an Alertmanager webhook.
type AlertmanagerAlert struct {
	Status       string            `json:"status" validate:"required,webhook_status"`
	Labels       map[string]string `json:"labels" validate:"required"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	EndsAt       time.Time         `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL" validate:"omitempty,url"`
	Fingerprint  string            `json:"fingerprint"`
}
