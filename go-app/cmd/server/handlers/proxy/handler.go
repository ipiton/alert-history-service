// Package proxy provides HTTP handler for intelligent proxy webhook endpoint.
package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// ProxyWebhookService defines the interface for proxy webhook business logic.
type ProxyWebhookService interface {
	// ProcessWebhook processes a proxy webhook request end-to-end
	ProcessWebhook(ctx context.Context, req *ProxyWebhookRequest) (*ProxyWebhookResponse, error)

	// Health checks service health
	Health(ctx context.Context) error
}

// ProxyWebhookHTTPHandler handles HTTP requests for proxy webhook endpoint.
type ProxyWebhookHTTPHandler struct {
	service   ProxyWebhookService
	config    *ProxyWebhookConfig
	logger    *slog.Logger
	metrics   *metrics.MetricsRegistry
	validator *validator.Validate
}

// NewProxyWebhookHTTPHandler creates a new HTTP handler.
func NewProxyWebhookHTTPHandler(
	service ProxyWebhookService,
	config *ProxyWebhookConfig,
	logger *slog.Logger,
	metricsRegistry *metrics.MetricsRegistry,
) *ProxyWebhookHTTPHandler {
	if logger == nil {
		logger = slog.Default()
	}
	if config == nil {
		config = DefaultProxyWebhookConfig()
	}

	return &ProxyWebhookHTTPHandler{
		service:   service,
		config:    config,
		logger:    logger,
		metrics:   metricsRegistry,
		validator: validator.New(),
	}
}

// ServeHTTP implements http.Handler interface.
func (h *ProxyWebhookHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handleProxyWebhook(w, r)
}

// handleProxyWebhook is the main handler function.
func (h *ProxyWebhookHTTPHandler) handleProxyWebhook(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Extract request ID from context (set by middleware)
	requestID := r.Context().Value("request_id")
	if requestID == nil {
		requestID = "unknown"
	}

	h.logger.Info("Proxy webhook request received",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"request_id", requestID,
		"content_type", r.Header.Get("Content-Type"),
	)

	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.logger.Warn("Invalid HTTP method", "method", r.Method, "request_id", requestID)
		h.writeError(w, http.StatusMethodNotAllowed, ErrCodeValidation,
			"Method not allowed", nil, requestID.(string))
		return
	}

	// Check Content-Type
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" && contentType != "" {
		h.logger.Warn("Invalid Content-Type", "content_type", contentType, "request_id", requestID)
		h.writeError(w, http.StatusUnsupportedMediaType, ErrCodeUnsupportedMedia,
			"Content-Type must be application/json", nil, requestID.(string))
		return
	}

	// Parse and validate request
	req, err := h.parseRequest(r)
	if err != nil {
		h.logger.Error("Failed to parse request", "error", err, "request_id", requestID)
		h.writeError(w, http.StatusBadRequest, ErrCodeValidation,
			fmt.Sprintf("Invalid request: %v", err), nil, requestID.(string))
		return
	}

	// Add timeout to context
	ctx, cancel := context.WithTimeout(r.Context(), h.config.RequestTimeout)
	defer cancel()

	// Process webhook through service
	response, err := h.service.ProcessWebhook(ctx, req)
	if err != nil {
		h.logger.Error("Failed to process webhook", "error", err, "request_id", requestID)

		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			h.writeError(w, http.StatusGatewayTimeout, ErrCodeTimeout,
				"Request timeout", nil, requestID.(string))
			return
		}

		// Generic internal error
		h.writeError(w, http.StatusInternalServerError, ErrCodeInternal,
			"Internal server error", nil, requestID.(string))
		return
	}

	// Calculate processing time
	processingTime := time.Since(startTime)
	response.ProcessingTime = processingTime

	// Determine HTTP status code based on response status
	statusCode := h.determineStatusCode(response)

	// Write response
	h.writeResponse(w, statusCode, response)

	// Record metrics
	if h.config.EnableMetrics && h.metrics != nil {
		// TODO: Record proxy-specific metrics
		// h.metrics.ProxyRequestsTotal.WithLabelValues(response.Status, req.Receiver).Inc()
		// h.metrics.ProxyRequestDuration.Observe(processingTime.Seconds())
	}

	h.logger.Info("Proxy webhook processed",
		"request_id", requestID,
		"status", response.Status,
		"alerts_received", response.AlertsSummary.TotalReceived,
		"alerts_published", response.AlertsSummary.TotalPublished,
		"alerts_filtered", response.AlertsSummary.TotalFiltered,
		"processing_time", processingTime,
	)
}

// parseRequest parses and validates the request.
func (h *ProxyWebhookHTTPHandler) parseRequest(r *http.Request) (*ProxyWebhookRequest, error) {
	// Check content length
	if r.ContentLength > int64(h.config.MaxRequestSize) {
		return nil, fmt.Errorf("request too large: %d bytes (max %d)", r.ContentLength, h.config.MaxRequestSize)
	}

	// Read request body with size limit
	body, err := io.ReadAll(io.LimitReader(r.Body, int64(h.config.MaxRequestSize)))
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %w", err)
	}
	defer r.Body.Close()

	// Parse JSON
	var req ProxyWebhookRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate request
	if h.config.EnableValidation {
		if err := h.validator.Struct(&req); err != nil {
			return nil, fmt.Errorf("validation failed: %w", err)
		}

		// Additional validation
		if len(req.Alerts) == 0 {
			return nil, fmt.Errorf("alerts array cannot be empty")
		}
		if len(req.Alerts) > h.config.MaxAlertsPerReq {
			return nil, fmt.Errorf("alerts array cannot exceed %d items (got %d)",
				h.config.MaxAlertsPerReq, len(req.Alerts))
		}

		// Validate each alert
		for i, alert := range req.Alerts {
			if err := h.validator.Struct(&alert); err != nil {
				return nil, fmt.Errorf("alert[%d] validation failed: %w", i, err)
			}

			// Check for alertname in labels
			if _, ok := alert.Labels["alertname"]; !ok {
				return nil, fmt.Errorf("alert[%d]: labels must contain 'alertname'", i)
			}
		}
	}

	return &req, nil
}

// determineStatusCode determines the HTTP status code based on response status.
func (h *ProxyWebhookHTTPHandler) determineStatusCode(response *ProxyWebhookResponse) int {
	switch response.Status {
	case "success":
		return http.StatusOK // 200
	case "partial":
		return http.StatusMultiStatus // 207
	case "failed":
		return http.StatusInternalServerError // 500
	default:
		return http.StatusOK // Default to 200
	}
}

// writeResponse writes the response.
func (h *ProxyWebhookHTTPHandler) writeResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		// Can't write error response here since headers already sent
	}
}

// writeError writes an error response.
func (h *ProxyWebhookHTTPHandler) writeError(
	w http.ResponseWriter,
	statusCode int,
	errorCode string,
	message string,
	details []FieldErrorDetail,
	requestID string,
) {
	errorResp := ErrorResponse{
		Error: ErrorDetail{
			Code:      errorCode,
			Message:   message,
			Details:   details,
			Timestamp: time.Now(),
			RequestID: requestID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		h.logger.Error("Failed to encode error response", "error", err)
	}
}
