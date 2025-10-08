package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWebhookHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		payload        interface{}
		expectedStatus int
		expectedFields []string
	}{
		{
			name:   "Valid webhook request",
			method: http.MethodPost,
			payload: WebhookRequest{
				AlertName: "test-alert",
				Status:    "firing",
				Labels: map[string]string{
					"severity": "critical",
					"service":  "api",
				},
				Annotations: map[string]string{
					"summary":     "Test alert",
					"description": "This is a test alert",
				},
				StartsAt:     "2023-01-01T00:00:00Z",
				EndsAt:       "",
				GeneratorURL: "http://prometheus:9090/graph",
				Fingerprint:  "abc123",
			},
			expectedStatus: http.StatusOK,
			expectedFields: []string{"status", "message", "alert_id", "timestamp", "processing_time"},
		},
		{
			name:           "Invalid HTTP method",
			method:         http.MethodGet,
			payload:        nil,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "Invalid JSON payload",
			method:         http.MethodPost,
			payload:        "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "Missing required field",
			method: http.MethodPost,
			payload: WebhookRequest{
				Status: "firing",
				Labels: map[string]string{
					"severity": "critical",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Prepare request body
			var body bytes.Buffer
			if tt.payload != nil {
				if str, ok := tt.payload.(string); ok {
					body.WriteString(str)
				} else {
					json.NewEncoder(&body).Encode(tt.payload)
				}
			}

			// Create request
			req := httptest.NewRequest(tt.method, "/webhook", &body)
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			w := httptest.NewRecorder()

			// Call handler
			WebhookHandler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check response fields for successful requests
			if tt.expectedStatus == http.StatusOK {
				var response WebhookResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("Failed to decode response: %v", err)
				}

				// Check that all expected fields are present
				responseMap := map[string]interface{}{
					"status":          response.Status,
					"message":         response.Message,
					"alert_id":        response.AlertID,
					"timestamp":       response.Timestamp,
					"processing_time": response.ProcessingTime,
				}

				for _, field := range tt.expectedFields {
					if value, exists := responseMap[field]; !exists || value == "" {
						t.Errorf("Expected field '%s' to be present and non-empty", field)
					}
				}

				// Check specific values
				if response.Status != "success" {
					t.Errorf("Expected status 'success', got '%s'", response.Status)
				}
			}
		})
	}
}

func TestProcessWebhook(t *testing.T) {
	req := &WebhookRequest{
		AlertName: "test-alert",
		Status:    "firing",
		Labels: map[string]string{
			"severity": "critical",
		},
		Annotations: map[string]string{
			"summary": "Test alert",
		},
		StartsAt: "2023-01-01T00:00:00Z",
	}

	alertID, err := processWebhook(req)
	if err != nil {
		t.Fatalf("processWebhook failed: %v", err)
	}

	if alertID == "" {
		t.Error("Expected non-empty alert ID")
	}

	// Check that alert ID has expected format
	if len(alertID) < 10 {
		t.Errorf("Alert ID seems too short: %s", alertID)
	}
}
