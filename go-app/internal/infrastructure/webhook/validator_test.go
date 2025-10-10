package webhook

import (
	"testing"
	"time"
)

func TestNewWebhookValidator(t *testing.T) {
	validator := NewWebhookValidator()
	if validator == nil {
		t.Fatal("NewWebhookValidator should not return nil")
	}
}

func TestValidateAlertmanager_ValidPayload(t *testing.T) {
	validator := NewWebhookValidator()

	webhook := &AlertmanagerWebhook{
		Version:  "4",
		GroupKey: "{}:{alertname=\"TestAlert\"}",
		Status:   "firing",
		Receiver: "default",
		GroupLabels: map[string]string{
			"alertname": "TestAlert",
		},
		CommonLabels: map[string]string{
			"alertname": "TestAlert",
			"severity":  "critical",
		},
		ExternalURL: "http://alertmanager.example.com",
		Alerts: []AlertmanagerAlert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary": "Test alert",
				},
				StartsAt:     time.Now().Add(-5 * time.Minute),
				EndsAt:       time.Time{}, // Ongoing alert
				GeneratorURL: "http://prometheus.example.com/graph",
				Fingerprint:  "abc123",
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if !result.Valid {
		t.Errorf("Expected valid webhook, got errors: %+v", result.Errors)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Expected no errors, got %d errors", len(result.Errors))
	}
}

func TestValidateAlertmanager_NilWebhook(t *testing.T) {
	validator := NewWebhookValidator()

	result := validator.ValidateAlertmanager(nil)
	if result.Valid {
		t.Error("Expected invalid result for nil webhook")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for nil webhook")
	}
	if result.Errors[0].Field != "webhook" {
		t.Errorf("Expected field 'webhook', got '%s'", result.Errors[0].Field)
	}
}

func TestValidateAlertmanager_MissingRequiredFields(t *testing.T) {
	validator := NewWebhookValidator()

	tests := []struct {
		name          string
		webhook       *AlertmanagerWebhook
		expectedField string
	}{
		{
			name: "missing version",
			webhook: &AlertmanagerWebhook{
				GroupKey: "test",
				Status:   "firing",
				Alerts: []AlertmanagerAlert{
					{
						Status: "firing",
						Labels: map[string]string{"alertname": "test"},
					},
				},
			},
			expectedField: "version",
		},
		{
			name: "missing status",
			webhook: &AlertmanagerWebhook{
				Version:  "4",
				GroupKey: "test",
				Alerts: []AlertmanagerAlert{
					{
						Status: "firing",
						Labels: map[string]string{"alertname": "test"},
					},
				},
			},
			expectedField: "status",
		},
		{
			name: "missing groupKey",
			webhook: &AlertmanagerWebhook{
				Version: "4",
				Status:  "firing",
				Alerts: []AlertmanagerAlert{
					{
						Status: "firing",
						Labels: map[string]string{"alertname": "test"},
					},
				},
			},
			expectedField: "groupKey",
		},
		{
			name: "empty alerts",
			webhook: &AlertmanagerWebhook{
				Version:  "4",
				GroupKey: "test",
				Status:   "firing",
				Alerts:   []AlertmanagerAlert{},
			},
			expectedField: "alerts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateAlertmanager(tt.webhook)
			if result.Valid {
				t.Error("Expected invalid result")
			}
			if len(result.Errors) == 0 {
				t.Fatal("Expected validation errors")
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.expectedField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected error for field '%s', got errors: %+v", tt.expectedField, result.Errors)
			}
		})
	}
}

func TestValidateAlertmanager_InvalidStatus(t *testing.T) {
	validator := NewWebhookValidator()

	webhook := &AlertmanagerWebhook{
		Version:  "4",
		GroupKey: "test",
		Status:   "invalid_status",
		Alerts: []AlertmanagerAlert{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "test"},
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if result.Valid {
		t.Error("Expected invalid result for invalid status")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "status" && err.Tag == "webhook_status" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected status validation error, got: %+v", result.Errors)
	}
}

func TestValidateAlertmanager_InvalidURL(t *testing.T) {
	validator := NewWebhookValidator()

	webhook := &AlertmanagerWebhook{
		Version:     "4",
		GroupKey:    "test",
		Status:      "firing",
		ExternalURL: "not-a-valid-url",
		Alerts: []AlertmanagerAlert{
			{
				Status: "firing",
				Labels: map[string]string{"alertname": "test"},
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if result.Valid {
		t.Error("Expected invalid result for invalid URL")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "externalURL" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected externalURL validation error, got: %+v", result.Errors)
	}
}

func TestValidateAlert_MissingAlertname(t *testing.T) {
	validator := NewWebhookValidator()

	webhook := &AlertmanagerWebhook{
		Version:  "4",
		GroupKey: "test",
		Status:   "firing",
		Alerts: []AlertmanagerAlert{
			{
				Status: "firing",
				Labels: map[string]string{}, // Missing alertname
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if result.Valid {
		t.Error("Expected invalid result for missing alertname")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "alerts[0].labels.alertname" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected alertname validation error, got: %+v", result.Errors)
	}
}

func TestValidateAlert_InvalidSeverity(t *testing.T) {
	validator := NewWebhookValidator()

	webhook := &AlertmanagerWebhook{
		Version:  "4",
		GroupKey: "test",
		Status:   "firing",
		Alerts: []AlertmanagerAlert{
			{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "test",
					"severity":  "invalid_severity",
				},
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if result.Valid {
		t.Error("Expected invalid result for invalid severity")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "alerts[0].labels.severity" && err.Tag == "severity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected severity validation error, got: %+v", result.Errors)
	}
}

func TestValidateAlert_ValidSeverities(t *testing.T) {
	validator := NewWebhookValidator()

	validSeverities := []string{"critical", "warning", "info", "debug"}

	for _, severity := range validSeverities {
		t.Run(severity, func(t *testing.T) {
			webhook := &AlertmanagerWebhook{
				Version:  "4",
				GroupKey: "test",
				Status:   "firing",
				Alerts: []AlertmanagerAlert{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "test",
							"severity":  severity,
						},
					},
				},
			}

			result := validator.ValidateAlertmanager(webhook)
			if !result.Valid {
				t.Errorf("Expected valid result for severity '%s', got errors: %+v", severity, result.Errors)
			}
		})
	}
}

func TestValidateAlert_InvalidTimestamps(t *testing.T) {
	validator := NewWebhookValidator()

	startsAt := time.Now()
	endsAt := startsAt.Add(-1 * time.Hour) // EndsAt before StartsAt

	webhook := &AlertmanagerWebhook{
		Version:  "4",
		GroupKey: "test",
		Status:   "firing",
		Alerts: []AlertmanagerAlert{
			{
				Status: "resolved",
				Labels: map[string]string{"alertname": "test"},
				StartsAt: startsAt,
				EndsAt:   endsAt,
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if result.Valid {
		t.Error("Expected invalid result for endsAt before startsAt")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "alerts[0].endsAt" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected endsAt validation error, got: %+v", result.Errors)
	}
}

func TestValidateAlert_InvalidGeneratorURL(t *testing.T) {
	validator := NewWebhookValidator()

	webhook := &AlertmanagerWebhook{
		Version:  "4",
		GroupKey: "test",
		Status:   "firing",
		Alerts: []AlertmanagerAlert{
			{
				Status:       "firing",
				Labels:       map[string]string{"alertname": "test"},
				GeneratorURL: "not-a-valid-url",
			},
		},
	}

	result := validator.ValidateAlertmanager(webhook)
	if result.Valid {
		t.Error("Expected invalid result for invalid generatorURL")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "alerts[0].generatorURL" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected generatorURL validation error, got: %+v", result.Errors)
	}
}

func TestValidateAlertmanager_TruncatedAlerts(t *testing.T) {
	validator := NewWebhookValidator()

	tests := []struct {
		name              string
		truncatedAlerts   int
		expectValid       bool
	}{
		{"positive truncated", 5, true},
		{"zero truncated", 0, true},
		{"negative truncated", -1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			webhook := &AlertmanagerWebhook{
				Version:         "4",
				GroupKey:        "test",
				Status:          "firing",
				TruncatedAlerts: tt.truncatedAlerts,
				Alerts: []AlertmanagerAlert{
					{
						Status: "firing",
						Labels: map[string]string{"alertname": "test"},
					},
				},
			}

			result := validator.ValidateAlertmanager(webhook)
			if result.Valid != tt.expectValid {
				t.Errorf("Expected valid=%v for truncatedAlerts=%d, got valid=%v, errors: %+v",
					tt.expectValid, tt.truncatedAlerts, result.Valid, result.Errors)
			}
		})
	}
}

func TestValidateGeneric_ValidPayload(t *testing.T) {
	validator := NewWebhookValidator()

	data := map[string]interface{}{
		"alertname": "TestAlert",
		"status":    "firing",
		"severity":  "critical",
		"message":   "Test message",
	}

	result := validator.ValidateGeneric(data)
	if !result.Valid {
		t.Errorf("Expected valid result, got errors: %+v", result.Errors)
	}
}

func TestValidateGeneric_NilData(t *testing.T) {
	validator := NewWebhookValidator()

	result := validator.ValidateGeneric(nil)
	if result.Valid {
		t.Error("Expected invalid result for nil data")
	}
	if len(result.Errors) == 0 {
		t.Error("Expected validation errors for nil data")
	}
}

func TestValidateGeneric_MissingRequiredFields(t *testing.T) {
	validator := NewWebhookValidator()

	tests := []struct {
		name          string
		data          map[string]interface{}
		missingField  string
	}{
		{
			name: "missing alertname",
			data: map[string]interface{}{
				"status": "firing",
			},
			missingField: "alertname",
		},
		{
			name: "missing status",
			data: map[string]interface{}{
				"alertname": "test",
			},
			missingField: "status",
		},
		{
			name: "empty alertname",
			data: map[string]interface{}{
				"alertname": "",
				"status":    "firing",
			},
			missingField: "alertname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateGeneric(tt.data)
			if result.Valid {
				t.Error("Expected invalid result")
			}

			found := false
			for _, err := range result.Errors {
				if err.Field == tt.missingField {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected error for field '%s', got: %+v", tt.missingField, result.Errors)
			}
		})
	}
}

func TestValidateGeneric_InvalidStatus(t *testing.T) {
	validator := NewWebhookValidator()

	data := map[string]interface{}{
		"alertname": "test",
		"status":    "invalid_status",
	}

	result := validator.ValidateGeneric(data)
	if result.Valid {
		t.Error("Expected invalid result for invalid status")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "status" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected status validation error, got: %+v", result.Errors)
	}
}

func TestValidateGeneric_InvalidSeverity(t *testing.T) {
	validator := NewWebhookValidator()

	data := map[string]interface{}{
		"alertname": "test",
		"status":    "firing",
		"severity":  "invalid_severity",
	}

	result := validator.ValidateGeneric(data)
	if result.Valid {
		t.Error("Expected invalid result for invalid severity")
	}

	found := false
	for _, err := range result.Errors {
		if err.Field == "severity" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected severity validation error, got: %+v", result.Errors)
	}
}

func TestIsValidSeverity(t *testing.T) {
	tests := []struct {
		severity string
		expected bool
	}{
		{"critical", true},
		{"warning", true},
		{"info", true},
		{"debug", true},
		{"CRITICAL", true}, // Case insensitive
		{"Warning", true},
		{"invalid", false},
		{"error", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			result := isValidSeverity(tt.severity)
			if result != tt.expected {
				t.Errorf("isValidSeverity(%q) = %v, expected %v", tt.severity, result, tt.expected)
			}
		})
	}
}

func TestIsValidWebhookStatus(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{"firing", true},
		{"resolved", true},
		{"FIRING", false},   // Case sensitive
		{"Resolved", false},
		{"pending", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := isValidWebhookStatus(tt.status)
			if result != tt.expected {
				t.Errorf("isValidWebhookStatus(%q) = %v, expected %v", tt.status, result, tt.expected)
			}
		})
	}
}
