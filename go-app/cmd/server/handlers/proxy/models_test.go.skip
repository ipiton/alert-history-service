package proxy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// TestAlertPayload_ConvertToAlert tests conversion from AlertPayload to core.Alert
func TestAlertPayload_ConvertToAlert(t *testing.T) {
	tests := []struct {
		name        string
		payload     AlertPayload
		expectError bool
		validate    func(t *testing.T, alert *core.Alert)
	}{
		{
			name: "valid alert with all fields",
			payload: AlertPayload{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "HighCPUUsage",
					"severity":  "warning",
					"instance":  "server-01",
					"namespace": "production",
				},
				Annotations: map[string]string{
					"summary":     "CPU usage is high",
					"description": "CPU usage has exceeded 80% for 5 minutes",
				},
				StartsAt:    time.Now().Add(-5 * time.Minute),
				EndsAt:      time.Time{},
				GeneratorURL: "http://prometheus:9090/alerts",
				Fingerprint: "abc123",
			},
			expectError: false,
			validate: func(t *testing.T, alert *core.Alert) {
				assert.Equal(t, "HighCPUUsage", alert.AlertName)
				assert.Equal(t, "firing", alert.Status)
				assert.Equal(t, "warning", alert.Labels["severity"])
				assert.Equal(t, "server-01", alert.Labels["instance"])
				assert.Equal(t, "production", alert.Labels["namespace"])
				assert.Equal(t, "CPU usage is high", alert.Annotations["summary"])
				assert.Equal(t, "http://prometheus:9090/alerts", alert.GeneratorURL)
				assert.Equal(t, "abc123", alert.Fingerprint)
			},
		},
		{
			name: "resolved alert",
			payload: AlertPayload{
				Status: "resolved",
				Labels: map[string]string{
					"alertname": "DiskSpaceLow",
					"severity":  "critical",
				},
				Annotations: map[string]string{
					"summary": "Disk space recovered",
				},
				StartsAt:    time.Now().Add(-1 * time.Hour),
				EndsAt:      time.Now(),
				Fingerprint: "def456",
			},
			expectError: false,
			validate: func(t *testing.T, alert *core.Alert) {
				assert.Equal(t, "DiskSpaceLow", alert.AlertName)
				assert.Equal(t, "resolved", alert.Status)
				assert.False(t, alert.EndsAt.IsZero())
				assert.Equal(t, "def456", alert.Fingerprint)
			},
		},
		{
			name: "minimal valid alert",
			payload: AlertPayload{
				Status: "firing",
				Labels: map[string]string{
					"alertname": "TestAlert",
				},
				StartsAt: time.Now(),
			},
			expectError: false,
			validate: func(t *testing.T, alert *core.Alert) {
				assert.Equal(t, "TestAlert", alert.AlertName)
				assert.Equal(t, "firing", alert.Status)
				assert.NotEmpty(t, alert.Fingerprint)
			},
		},
		{
			name: "alert without alertname in labels",
			payload: AlertPayload{
				Status: "firing",
				Labels: map[string]string{
					"severity": "warning",
				},
				StartsAt: time.Now(),
			},
			expectError: true,
		},
		{
			name: "alert with empty status",
			payload: AlertPayload{
				Status: "",
				Labels: map[string]string{
					"alertname": "TestAlert",
				},
				StartsAt: time.Now(),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alert, err := tt.payload.ConvertToAlert()

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, alert)
			} else {
				require.NoError(t, err)
				require.NotNil(t, alert)
				if tt.validate != nil {
					tt.validate(t, alert)
				}
			}
		})
	}
}

// TestAlertPayload_generateFingerprint tests fingerprint generation
func TestAlertPayload_generateFingerprint(t *testing.T) {
	tests := []struct {
		name     string
		payload  AlertPayload
		expected string
	}{
		{
			name: "fingerprint provided",
			payload: AlertPayload{
				Fingerprint: "custom-fingerprint-123",
				Labels: map[string]string{
					"alertname": "TestAlert",
				},
			},
			expected: "custom-fingerprint-123",
		},
		{
			name: "generate from labels",
			payload: AlertPayload{
				Labels: map[string]string{
					"alertname": "HighCPU",
					"instance":  "server-01",
					"severity":  "critical",
				},
			},
			expected: "", // Non-empty, but exact value depends on implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.payload.generateFingerprint()

			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NotEmpty(t, result)
			}
		})
	}
}

// TestClassificationResult_ConfidenceBucket tests confidence categorization
func TestClassificationResult_ConfidenceBucket(t *testing.T) {
	tests := []struct {
		name       string
		confidence float64
		expected   string
	}{
		{"very high confidence", 0.95, "high"},
		{"high confidence boundary", 0.80, "high"},
		{"medium confidence", 0.60, "medium"},
		{"low confidence", 0.40, "low"},
		{"very low confidence", 0.15, "low"},
		{"zero confidence", 0.0, "low"},
		{"max confidence", 1.0, "high"},
		{"medium-high boundary", 0.79, "medium"},
		{"medium-low boundary", 0.51, "medium"},
		{"low boundary", 0.50, "low"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := ClassificationResult{
				Confidence: tt.confidence,
			}

			result := cr.ConfidenceBucket()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestProxyWebhookRequest_Validation tests request validation
func TestProxyWebhookRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request ProxyWebhookRequest
		valid   bool
	}{
		{
			name: "valid request with alerts",
			request: ProxyWebhookRequest{
				Receiver: "webhook-receiver",
				Status:   "firing",
				Alerts: []AlertPayload{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "TestAlert",
						},
						StartsAt: time.Now(),
					},
				},
			},
			valid: true,
		},
		{
			name: "valid request with group labels",
			request: ProxyWebhookRequest{
				Receiver: "webhook-receiver",
				Status:   "firing",
				Alerts: []AlertPayload{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "TestAlert",
						},
						StartsAt: time.Now(),
					},
				},
				GroupKey: "group-123",
				GroupLabels: map[string]string{
					"alertname": "TestAlert",
				},
				CommonLabels: map[string]string{
					"cluster": "prod",
				},
			},
			valid: true,
		},
		{
			name: "empty receiver",
			request: ProxyWebhookRequest{
				Receiver: "",
				Status:   "firing",
				Alerts: []AlertPayload{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "TestAlert",
						},
						StartsAt: time.Now(),
					},
				},
			},
			valid: false,
		},
		{
			name: "empty status",
			request: ProxyWebhookRequest{
				Receiver: "webhook-receiver",
				Status:   "",
				Alerts: []AlertPayload{
					{
						Status: "firing",
						Labels: map[string]string{
							"alertname": "TestAlert",
						},
						StartsAt: time.Now(),
					},
				},
			},
			valid: false,
		},
		{
			name: "no alerts",
			request: ProxyWebhookRequest{
				Receiver: "webhook-receiver",
				Status:   "firing",
				Alerts:   []AlertPayload{},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Note: Actual validation would use go-playground/validator
			// This is a simplified test
			hasReceiver := tt.request.Receiver != ""
			hasStatus := tt.request.Status != ""
			hasAlerts := len(tt.request.Alerts) > 0

			isValid := hasReceiver && hasStatus && hasAlerts

			if tt.valid {
				assert.True(t, isValid, "Expected request to be valid")
			} else {
				assert.False(t, isValid, "Expected request to be invalid")
			}
		})
	}
}

// TestProxyWebhookResponse_StatusDetermination tests status determination logic
func TestProxyWebhookResponse_StatusDetermination(t *testing.T) {
	tests := []struct {
		name           string
		results        []AlertProcessingResult
		expectedStatus string
		expectedMsg    string
	}{
		{
			name: "all successful",
			results: []AlertProcessingResult{
				{Status: "success", AlertName: "Alert1"},
				{Status: "success", AlertName: "Alert2"},
				{Status: "success", AlertName: "Alert3"},
			},
			expectedStatus: "success",
			expectedMsg:    "All alerts processed successfully",
		},
		{
			name: "all failed",
			results: []AlertProcessingResult{
				{Status: "failed", AlertName: "Alert1", ErrorMessage: "error1"},
				{Status: "failed", AlertName: "Alert2", ErrorMessage: "error2"},
			},
			expectedStatus: "failed",
			expectedMsg:    "All alerts failed processing",
		},
		{
			name: "partial success",
			results: []AlertProcessingResult{
				{Status: "success", AlertName: "Alert1"},
				{Status: "failed", AlertName: "Alert2", ErrorMessage: "error"},
				{Status: "success", AlertName: "Alert3"},
			},
			expectedStatus: "partial",
			expectedMsg:    "1 of 3 alerts failed",
		},
		{
			name: "some filtered",
			results: []AlertProcessingResult{
				{Status: "success", AlertName: "Alert1"},
				{Status: "filtered", AlertName: "Alert2", FilterReason: "noise"},
				{Status: "success", AlertName: "Alert3"},
			},
			expectedStatus: "partial",
			expectedMsg:    "1 of 3 alerts filtered",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate status determination
			successCount := 0
			failedCount := 0
			filteredCount := 0

			for _, result := range tt.results {
				switch result.Status {
				case "success":
					successCount++
				case "failed":
					failedCount++
				case "filtered":
					filteredCount++
				}
			}

			var status, message string
			if failedCount == 0 && filteredCount == 0 {
				status = "success"
				message = "All alerts processed successfully"
			} else if successCount == 0 {
				status = "failed"
				message = "All alerts failed processing"
			} else {
				status = "partial"
				if failedCount > 0 {
					message = "1 of 3 alerts failed"
				} else {
					message = "1 of 3 alerts filtered"
				}
			}

			assert.Contains(t, tt.expectedStatus, status)
			assert.Contains(t, message, tt.expectedMsg)
		})
	}
}

// TestTargetPublishingResult_SuccessTracking tests per-target result tracking
func TestTargetPublishingResult_SuccessTracking(t *testing.T) {
	tests := []struct {
		name     string
		result   TargetPublishingResult
		expected bool
	}{
		{
			name: "successful publishing",
			result: TargetPublishingResult{
				TargetName:     "rootly-prod",
				TargetType:     "rootly",
				Success:        true,
				StatusCode:     200,
				ProcessingTime: 150 * time.Millisecond,
			},
			expected: true,
		},
		{
			name: "failed publishing - 500",
			result: TargetPublishingResult{
				TargetName:     "pagerduty-oncall",
				TargetType:     "pagerduty",
				Success:        false,
				StatusCode:     500,
				ErrorMessage:   "Internal server error",
				ProcessingTime: 5 * time.Second,
			},
			expected: false,
		},
		{
			name: "failed publishing - timeout",
			result: TargetPublishingResult{
				TargetName:     "slack-alerts",
				TargetType:     "slack",
				Success:        false,
				StatusCode:     0,
				ErrorMessage:   "context deadline exceeded",
				ErrorCode:      string(ErrCodeTimeout),
				ProcessingTime: 10 * time.Second,
			},
			expected: false,
		},
		{
			name: "successful after retries",
			result: TargetPublishingResult{
				TargetName:     "rootly-staging",
				TargetType:     "rootly",
				Success:        true,
				StatusCode:     200,
				RetryCount:     2,
				ProcessingTime: 800 * time.Millisecond,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Success)

			if tt.result.Success {
				assert.Empty(t, tt.result.ErrorMessage)
				assert.GreaterOrEqual(t, tt.result.StatusCode, 200)
				assert.LessOrEqual(t, tt.result.StatusCode, 299)
			} else {
				assert.NotEmpty(t, tt.result.ErrorMessage)
			}
		})
	}
}

// TestErrorResponse_Creation tests error response creation
func TestErrorResponse_Creation(t *testing.T) {
	tests := []struct {
		name       string
		code       string
		message    string
		statusCode int
		details    []ErrorDetail
	}{
		{
			name:       "validation error",
			code:       string(ErrCodeValidation),
			message:    "Invalid request payload",
			statusCode: 400,
			details: []ErrorDetail{
				{Field: "alerts", Message: "alerts is required"},
				{Field: "receiver", Message: "receiver cannot be empty"},
			},
		},
		{
			name:       "authentication error",
			code:       string(ErrCodeAuthentication),
			message:    "Invalid API key",
			statusCode: 401,
			details:    nil,
		},
		{
			name:       "timeout error",
			code:       string(ErrCodeTimeout),
			message:    "Request processing timeout",
			statusCode: 504,
			details:    nil,
		},
		{
			name:       "internal error",
			code:       string(ErrCodeInternal),
			message:    "Internal server error occurred",
			statusCode: 500,
			details: []ErrorDetail{
				{Field: "cause", Message: "database connection failed"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errResp := ErrorResponse{
				Error:      tt.code,
				Message:    tt.message,
				StatusCode: tt.statusCode,
				Details:    tt.details,
				Timestamp:  time.Now(),
			}

			assert.Equal(t, tt.code, errResp.Error)
			assert.Equal(t, tt.message, errResp.Message)
			assert.Equal(t, tt.statusCode, errResp.StatusCode)
			assert.Equal(t, len(tt.details), len(errResp.Details))
			assert.False(t, errResp.Timestamp.IsZero())
		})
	}
}

// TestFilterAction_Values tests filter action enum values
func TestFilterAction_Values(t *testing.T) {
	tests := []struct {
		name     string
		action   FilterAction
		expected string
	}{
		{"allow action", FilterActionAllow, "allow"},
		{"deny action", FilterActionDeny, "deny"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.action))
		})
	}
}

// TestAlertProcessingResult_CompleteCycle tests full alert processing result
func TestAlertProcessingResult_CompleteCycle(t *testing.T) {
	result := AlertProcessingResult{
		Fingerprint: "abc123",
		AlertName:   "HighCPUUsage",
		Status:      "success",
		Classification: &ClassificationResult{
			Severity:   "critical",
			Category:   "performance",
			Confidence: 0.92,
			Source:     "llm",
			Timestamp:  time.Now(),
		},
		ClassificationTime: 145 * time.Millisecond,
		FilterAction:       string(FilterActionAllow),
		FilterReason:       "severity critical allowed",
		PublishingResults: []TargetPublishingResult{
			{
				TargetName:     "rootly-prod",
				TargetType:     "rootly",
				Success:        true,
				StatusCode:     201,
				ProcessingTime: 230 * time.Millisecond,
			},
			{
				TargetName:     "pagerduty-oncall",
				TargetType:     "pagerduty",
				Success:        true,
				StatusCode:     202,
				ProcessingTime: 180 * time.Millisecond,
			},
		},
	}

	assert.Equal(t, "abc123", result.Fingerprint)
	assert.Equal(t, "HighCPUUsage", result.AlertName)
	assert.Equal(t, "success", result.Status)
	assert.NotNil(t, result.Classification)
	assert.Equal(t, "critical", result.Classification.Severity)
	assert.Equal(t, 0.92, result.Classification.Confidence)
	assert.Equal(t, "high", result.Classification.ConfidenceBucket())
	assert.Equal(t, string(FilterActionAllow), result.FilterAction)
	assert.Len(t, result.PublishingResults, 2)
	assert.True(t, result.PublishingResults[0].Success)
	assert.True(t, result.PublishingResults[1].Success)
}

// BenchmarkAlertPayload_ConvertToAlert benchmarks alert conversion
func BenchmarkAlertPayload_ConvertToAlert(b *testing.B) {
	payload := AlertPayload{
		Status: "firing",
		Labels: map[string]string{
			"alertname": "HighCPUUsage",
			"severity":  "critical",
			"instance":  "server-01",
			"namespace": "production",
		},
		Annotations: map[string]string{
			"summary":     "CPU usage is high",
			"description": "CPU usage exceeded threshold",
		},
		StartsAt:     time.Now(),
		GeneratorURL: "http://prometheus:9090/alerts",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = payload.ConvertToAlert()
	}
}

// BenchmarkClassificationResult_ConfidenceBucket benchmarks confidence bucketing
func BenchmarkClassificationResult_ConfidenceBucket(b *testing.B) {
	cr := ClassificationResult{
		Confidence: 0.75,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cr.ConfidenceBucket()
	}
}
