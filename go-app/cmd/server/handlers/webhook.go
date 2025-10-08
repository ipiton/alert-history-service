// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// WebhookRequest represents the incoming webhook payload.
type WebhookRequest struct {
	AlertName   string                 `json:"alertname"`
	Status      string                 `json:"status"`
	Labels      map[string]string      `json:"labels"`
	Annotations map[string]string      `json:"annotations"`
	StartsAt    string                 `json:"startsAt"`
	EndsAt      string                 `json:"endsAt"`
	GeneratorURL string                `json:"generatorURL"`
	Fingerprint string                 `json:"fingerprint"`
	Extra       map[string]interface{} `json:"-"` // Catch-all for additional fields
}

// WebhookResponse represents the webhook response.
type WebhookResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	AlertID   string `json:"alert_id,omitempty"`
	Timestamp string `json:"timestamp"`
	ProcessingTime string `json:"processing_time"`
}

// WebhookHandler handles incoming webhook requests from alerting systems.
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Log the webhook request
	slog.Info("Webhook request received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"content_type", r.Header.Get("Content-Type"),
		"user_agent", r.Header.Get("User-Agent"),
	)

	// Only accept POST requests
	if r.Method != http.MethodPost {
		slog.Warn("Invalid HTTP method for webhook", "method", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		slog.Error("Failed to read request body", "error", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Log raw payload for debugging
	slog.Debug("Webhook payload received", "payload", string(body))

	// Parse JSON payload
	var webhookReq WebhookRequest
	if err := json.Unmarshal(body, &webhookReq); err != nil {
		slog.Error("Failed to parse webhook JSON", "error", err, "payload", string(body))
		http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if webhookReq.AlertName == "" {
		slog.Warn("Missing required field: alertname", "payload", webhookReq)
		http.Error(w, "Missing required field: alertname", http.StatusBadRequest)
		return
	}

	// Process the webhook
	alertID, err := processWebhook(&webhookReq)
	if err != nil {
		slog.Error("Failed to process webhook", "error", err, "alert", webhookReq.AlertName)
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	// Calculate processing time
	processingTime := time.Since(startTime)

	// Create response
	response := WebhookResponse{
		Status:         "success",
		Message:        "Webhook processed successfully",
		AlertID:        alertID,
		Timestamp:      time.Now().UTC().Format(time.RFC3339),
		ProcessingTime: processingTime.String(),
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Send response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		slog.Error("Failed to encode webhook response", "error", err)
		return
	}

	slog.Info("Webhook processed successfully",
		"alert_name", webhookReq.AlertName,
		"alert_id", alertID,
		"status", webhookReq.Status,
		"processing_time", processingTime,
	)
}

// processWebhook handles the business logic for processing webhook data.
func processWebhook(req *WebhookRequest) (string, error) {
	// Generate a simple alert ID for now
	alertID := fmt.Sprintf("alert_%d", time.Now().Unix())

	// Log the alert details
	slog.Info("Processing alert",
		"alert_id", alertID,
		"alert_name", req.AlertName,
		"status", req.Status,
		"starts_at", req.StartsAt,
		"ends_at", req.EndsAt,
		"labels", req.Labels,
		"annotations", req.Annotations,
	)

	// TODO: Implement actual alert processing logic:
	// 1. Store alert in database
	// 2. Apply business rules
	// 3. Send notifications if needed
	// 4. Update metrics

	// For now, just simulate processing
	time.Sleep(10 * time.Millisecond)

	return alertID, nil
}
