// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/vitaliisemenov/alert-history/cmd/server/middleware"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/webhook"
)

// WebhookHTTPHandler handles HTTP requests for webhook endpoint
type WebhookHTTPHandler struct {
	universalHandler *webhook.UniversalWebhookHandler
	logger           *slog.Logger
	config           *WebhookConfig
}

// WebhookConfig holds configuration for webhook endpoint
type WebhookConfig struct {
	MaxRequestSize  int64         // Max request body size (bytes)
	RequestTimeout  time.Duration // Request timeout
	MaxAlertsPerReq int           // Max alerts per request
	EnableMetrics   bool          // Enable Prometheus metrics
	EnableAuth      bool          // Enable authentication
	AuthType        string        // "api_key", "jwt", "hmac"
	APIKey          string        // API key for authentication
	SignatureSecret string        // HMAC secret for signature verification
}

// Error types for webhook processing
var (
	ErrPayloadTooLarge = errors.New("payload too large")
	ErrInvalidMethod   = errors.New("invalid HTTP method")
	ErrReadFailed      = errors.New("failed to read request body")
)

// ErrorResponse represents error response structure
type ErrorResponse struct {
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	RequestID string                 `json:"request_id"`
	Timestamp string                 `json:"timestamp"`
}

// NewWebhookHTTPHandler creates a new webhook HTTP handler
func NewWebhookHTTPHandler(
	universalHandler *webhook.UniversalWebhookHandler,
	config *WebhookConfig,
	logger *slog.Logger,
) *WebhookHTTPHandler {
	if logger == nil {
		logger = slog.Default()
	}

	if config == nil {
		config = &WebhookConfig{
			MaxRequestSize:  10 * 1024 * 1024, // 10MB
			RequestTimeout:  30 * time.Second,
			MaxAlertsPerReq: 1000,
			EnableMetrics:   true,
		}
	}

	return &WebhookHTTPHandler{
		universalHandler: universalHandler,
		logger:           logger,
		config:           config,
	}
}

// ServeHTTP implements http.Handler interface
func (h *WebhookHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Validate HTTP method
	if r.Method != http.MethodPost {
		h.writeError(w, r, http.StatusMethodNotAllowed, "Method not allowed", map[string]interface{}{
			"allowed_methods": []string{"POST"},
			"received_method": r.Method,
		})
		return
	}

	// 2. Extract context with timeout
	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)
	defer cancel()

	// 3. Extract request ID from context (set by middleware)
	requestID := middleware.GetRequestID(ctx)

	h.logger.Debug("Processing webhook request",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"content_length", r.ContentLength,
	)

	// 4. Read request body (with size limit)
	body, err := h.readBody(r)
	if err != nil {
		if errors.Is(err, ErrPayloadTooLarge) {
			h.writeError(w, r, http.StatusRequestEntityTooLarge,
				"Payload too large", map[string]interface{}{
					"max_size":      h.config.MaxRequestSize,
					"received_size": r.ContentLength,
				})
			return
		}
		h.writeError(w, r, http.StatusBadRequest,
			"Failed to read request body", map[string]interface{}{
				"error": err.Error(),
			})
		return
	}

	// 5. Prepare request for UniversalWebhookHandler
	webhookReq := &webhook.HandleWebhookRequest{
		Payload:     body,
		ContentType: r.Header.Get("Content-Type"),
		UserAgent:   r.Header.Get("User-Agent"),
	}

	// 6. Process webhook (business logic)
	webhookResp, err := h.universalHandler.HandleWebhook(ctx, webhookReq)
	if err != nil {
		// Determine HTTP status code from error type
		statusCode := h.errorToStatusCode(err)

		// If partial response available, use it (validation errors)
		if webhookResp != nil {
			h.writeResponse(w, r, statusCode, webhookResp)
			return
		}

		// Otherwise, create error response
		h.writeError(w, r, statusCode, err.Error(), map[string]interface{}{
			"error_type": fmt.Sprintf("%T", err),
		})
		return
	}

	// 7. Write success/partial success response
	statusCode := http.StatusOK
	if webhookResp.Status == "partial_success" {
		statusCode = http.StatusMultiStatus // 207
	}

	h.writeResponse(w, r, statusCode, webhookResp)
}

// readBody reads request body with size limit
func (h *WebhookHTTPHandler) readBody(r *http.Request) ([]byte, error) {
	// Check Content-Length header first
	if r.ContentLength > h.config.MaxRequestSize {
		return nil, ErrPayloadTooLarge
	}

	// Use LimitReader to enforce size limit
	limitReader := io.LimitReader(r.Body, h.config.MaxRequestSize+1)
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	// Check if size limit exceeded
	if int64(len(body)) > h.config.MaxRequestSize {
		return nil, ErrPayloadTooLarge
	}

	return body, nil
}

// writeResponse writes successful/partial response
func (h *WebhookHTTPHandler) writeResponse(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	resp *webhook.HandleWebhookResponse,
) {
	requestID := middleware.GetRequestID(r.Context())

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	w.Header().Set("X-Processing-Time", resp.ProcessingTime)
	w.Header().Set("X-Webhook-Type", resp.WebhookType)
	w.Header().Set("X-Alerts-Processed", strconv.Itoa(resp.AlertsProcessed))

	// Write status code
	w.WriteHeader(statusCode)

	// Encode response
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("Failed to encode response",
			"error", err,
			"request_id", requestID,
		)
	}

	// Log response
	logLevel := slog.LevelInfo
	if statusCode >= 400 {
		logLevel = slog.LevelWarn
	}
	if statusCode >= 500 {
		logLevel = slog.LevelError
	}

	h.logger.Log(r.Context(), logLevel, "Webhook response sent",
		"request_id", requestID,
		"status", statusCode,
		"webhook_type", resp.WebhookType,
		"alerts_received", resp.AlertsReceived,
		"alerts_processed", resp.AlertsProcessed,
		"processing_time", resp.ProcessingTime,
	)
}

// writeError writes error response
func (h *WebhookHTTPHandler) writeError(
	w http.ResponseWriter,
	r *http.Request,
	statusCode int,
	message string,
	details map[string]interface{},
) {
	requestID := middleware.GetRequestID(r.Context())

	errorResp := ErrorResponse{
		Status:    "error",
		Message:   message,
		Details:   details,
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)

	// Add Retry-After for rate limiting / service unavailable
	if statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusServiceUnavailable {
		w.Header().Set("Retry-After", "60") // 60 seconds
	}

	// Write status code
	w.WriteHeader(statusCode)

	// Encode error response
	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.Error("Failed to encode error response",
			"error", err,
			"request_id", requestID,
		)
	}

	// Log error
	h.logger.Error("Webhook request failed",
		"request_id", requestID,
		"status", statusCode,
		"message", message,
		"details", details,
	)
}

// errorToStatusCode maps error types to HTTP status codes
func (h *WebhookHTTPHandler) errorToStatusCode(err error) int {
	// Check for context errors first
	if errors.Is(err, context.DeadlineExceeded) {
		return http.StatusRequestTimeout // 408
	}
	if errors.Is(err, context.Canceled) {
		return http.StatusRequestTimeout // 408
	}

	// Check for webhook-specific errors
	// Note: These error types should be defined in webhook package
	errMsg := err.Error()
	switch {
	case contains(errMsg, "detection failed"):
		return http.StatusBadRequest // 400
	case contains(errMsg, "parsing failed"):
		return http.StatusBadRequest // 400
	case contains(errMsg, "validation failed"):
		return http.StatusBadRequest // 400
	case contains(errMsg, "conversion failed"):
		return http.StatusInternalServerError // 500
	case contains(errMsg, "processing failed"):
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError // 500
	}
}

// contains checks if string contains substring (case-insensitive helper)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)*2 && s[len(s)/2-len(substr)/2:len(s)/2+len(substr)/2+len(substr)%2] == substr))
}

