package publishing

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	apiservices "github.com/vitaliisemenov/alert-history/internal/api/services/publishing"
)

// PublishingModeHandler handles GET /publishing/mode requests for both API v1 and v2.
//
// This handler provides information about the current publishing mode (normal vs metrics-only)
// with support for HTTP caching, conditional requests, and comprehensive error handling.
//
// Features:
//   - HTTP caching (Cache-Control, ETag)
//   - Conditional requests (304 Not Modified)
//   - Structured logging with request ID
//   - Comprehensive error handling
//   - Thread-safe and stateless
//
// Dependencies:
//   - ModeService: Business logic for mode detection
//   - Middleware: RequestID, Logging, Metrics (applied at router level)
//
// Example:
//   handler := NewPublishingModeHandler(modeService, logger)
//   router.HandleFunc("/api/v1/publishing/mode", handler.GetPublishingMode).Methods("GET")
//   router.HandleFunc("/api/v2/publishing/mode", handler.GetPublishingMode).Methods("GET")
type PublishingModeHandler struct {
	service apiservices.ModeService
	logger  *slog.Logger
}

// NewPublishingModeHandler creates a new publishing mode handler.
//
// Parameters:
//   - service: ModeService for business logic (required)
//   - logger: Logger for structured logging (defaults to slog.Default if nil)
//
// Returns:
//   - *PublishingModeHandler: New handler instance
//
// Example:
//   handler := NewPublishingModeHandler(modeService, logger)
func NewPublishingModeHandler(service apiservices.ModeService, logger *slog.Logger) *PublishingModeHandler {
	if logger == nil {
		logger = slog.Default()
	}

	return &PublishingModeHandler{
		service: service,
		logger:  logger,
	}
}

// GetPublishingMode handles GET requests for publishing mode information.
//
// This method:
//   - Extracts request ID from context (set by RequestIDMiddleware)
//   - Retrieves mode information from service layer
//   - Handles conditional requests (If-None-Match → 304)
//   - Sets HTTP caching headers (Cache-Control, ETag)
//   - Returns JSON response with mode information
//   - Handles errors gracefully with structured error responses
//
// HTTP Methods: GET
// Paths: /api/v1/publishing/mode, /api/v2/publishing/mode
//
// Response Codes:
//   - 200 OK: Mode information returned successfully
//   - 304 Not Modified: Client has cached version (conditional request)
//   - 500 Internal Server Error: Service error or panic recovery
//
// Response Headers:
//   - Content-Type: application/json; charset=utf-8
//   - Cache-Control: max-age=5, public
//   - ETag: "mode-enabled_targets-transition_count"
//   - X-Request-ID: UUID for tracing
//
// Example Request:
//   GET /api/v1/publishing/mode HTTP/1.1
//   Host: localhost:8080
//
// Example Response (200 OK):
//   HTTP/1.1 200 OK
//   Content-Type: application/json; charset=utf-8
//   Cache-Control: max-age=5, public
//   ETag: "normal-5-12"
//   X-Request-ID: 550e8400-e29b-41d4-a716-446655440000
//
//   {
//     "mode": "normal",
//     "targets_available": true,
//     "enabled_targets": 5,
//     "metrics_only_active": false,
//     "transition_count": 12,
//     "current_mode_duration_seconds": 3600.5,
//     "last_transition_time": "2025-11-17T10:30:00Z",
//     "last_transition_reason": "targets_available"
//   }
func (h *PublishingModeHandler) GetPublishingMode(w http.ResponseWriter, r *http.Request) {
	// Extract request ID from context (set by RequestIDMiddleware)
	requestID := middleware.GetRequestID(r.Context())
	if requestID == "" {
		// Fallback: generate request ID if middleware didn't set it
		requestID = fmt.Sprintf("fallback-%d", time.Now().UnixNano())
		h.logger.Warn("Request ID not found in context, using fallback",
			"fallback_id", requestID)
	}

	// Log request start
	h.logger.Info("Handling GET /publishing/mode",
		"request_id", requestID,
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr)

	// Validate HTTP method (should be GET, but check anyway)
	if r.Method != http.MethodGet {
		h.logger.Warn("Invalid HTTP method for publishing mode endpoint",
			"request_id", requestID,
			"method", r.Method,
			"expected", http.MethodGet)
		h.sendError(w, http.StatusMethodNotAllowed, "Method not allowed", requestID)
		return
	}

	// Get mode info from service layer
	startTime := time.Now()
	modeInfo, err := h.service.GetCurrentModeInfo(r.Context())
	duration := time.Since(startTime)

	// Handle service errors
	if err != nil {
		h.logger.Error("Failed to get mode info",
			"request_id", requestID,
			"error", err,
			"duration_ms", duration.Milliseconds())

		h.sendError(w, http.StatusInternalServerError, "Failed to retrieve mode information", requestID)
		return
	}

	// Set security headers (OWASP Top 10 compliance)
	h.setSecurityHeaders(w, r)

	// Generate ETag for caching
	etag := h.generateETag(modeInfo)

	// Handle conditional request (If-None-Match header)
	if ifNoneMatch := r.Header.Get(middleware.IfNoneMatchHeader); ifNoneMatch != "" {
		if ifNoneMatch == etag {
			// Client has cached version, return 304 Not Modified
			h.logger.Debug("Conditional request: client has cached version",
				"request_id", requestID,
				"etag", etag)

			w.Header().Set(middleware.ETagHeader, etag)
			w.Header().Set(middleware.CacheControlHeader, "max-age=5, public")
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Set caching headers
	h.setCacheHeaders(w, modeInfo, etag)

	// Log success
	h.logger.Info("Successfully retrieved mode info",
		"request_id", requestID,
		"mode", modeInfo.Mode,
		"enabled_targets", modeInfo.EnabledTargets,
		"duration_ms", duration.Milliseconds())

	// Send JSON response
	h.sendJSON(w, http.StatusOK, modeInfo)
}

// setCacheHeaders sets HTTP caching headers for the response.
//
// Headers set:
//   - Cache-Control: max-age=5, public (5 seconds TTL, aligned with ModeManager periodic check)
//   - ETag: Generated based on mode, enabled_targets, and transition_count
//
// Rationale:
//   - Mode changes are rare (typically minutes/hours)
//   - 5s TTL balances freshness with performance
//   - Public cache allows CDN/proxy caching
//   - ETag enables conditional requests (304 Not Modified)
func (h *PublishingModeHandler) setCacheHeaders(w http.ResponseWriter, modeInfo *apiservices.ModeInfo, etag string) {
	// Cache-Control: 5 seconds TTL (aligned with ModeManager periodic check interval)
	w.Header().Set(middleware.CacheControlHeader, "max-age=5, public")

	// ETag: For conditional requests (304 Not Modified)
	w.Header().Set(middleware.ETagHeader, etag)
}

// generateETag generates an ETag for the given mode information.
//
// ETag format: "mode-enabled_targets-transition_count"
// Example: "normal-5-12" or "metrics-only-0-13"
//
// The ETag changes when:
//   - Mode changes (normal ↔ metrics-only)
//   - Enabled targets count changes
//   - Transition count changes (indicates mode transition occurred)
//
// This ensures clients get fresh data when mode actually changes, while
// allowing efficient caching when mode is stable.
func (h *PublishingModeHandler) generateETag(modeInfo *apiservices.ModeInfo) string {
	// ETag format: "mode-enabled_targets-transition_count"
	return fmt.Sprintf(`"%s-%d-%d"`,
		modeInfo.Mode,
		modeInfo.EnabledTargets,
		modeInfo.TransitionCount)
}

// sendJSON sends a JSON response with the given status code and data.
//
// This method:
//   - Sets Content-Type header (application/json; charset=utf-8)
//   - Sets status code
//   - Encodes data as JSON
//   - Handles encoding errors gracefully
//
// Parameters:
//   - w: HTTP response writer
//   - status: HTTP status code
//   - data: Data to encode as JSON
func (h *PublishingModeHandler) sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// Log encoding error (but status already sent, so can't change response)
		h.logger.Error("Failed to encode JSON response",
			"error", err,
			"status", status)
	}
}

// setSecurityHeaders sets OWASP Top 10 compliant security headers.
//
// Headers set:
//   - X-Content-Type-Options: nosniff (prevents MIME type sniffing)
//   - X-Frame-Options: DENY (prevents clickjacking)
//   - X-XSS-Protection: 1; mode=block (enables XSS filter)
//   - Content-Security-Policy: default-src 'none'; frame-ancestors 'none' (strict CSP for API)
//   - Strict-Transport-Security: max-age=31536000; includeSubDomains (HSTS, HTTPS only)
//   - Referrer-Policy: strict-origin-when-cross-origin (controls referrer information)
//   - Permissions-Policy: geolocation=(), microphone=(), camera=() (restricts browser features)
//
// Security benefits:
//   - Prevents MIME type confusion attacks
//   - Protects against clickjacking
//   - Enables browser XSS protection
//   - Restricts resource loading (CSP)
//   - Forces HTTPS (HSTS)
//   - Controls referrer information leakage
//   - Restricts browser features access
func (h *PublishingModeHandler) setSecurityHeaders(w http.ResponseWriter, r *http.Request) {
	// Prevent MIME type sniffing
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Prevent clickjacking
	w.Header().Set("X-Frame-Options", "DENY")

	// Enable XSS filter in older browsers
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Content Security Policy (strict for API endpoint)
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")

	// HTTP Strict Transport Security (HSTS) - only over HTTPS
	if r.TLS != nil {
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	}

	// Referrer Policy
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Permissions Policy (formerly Feature-Policy)
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

	// Remove potentially sensitive server information
	w.Header().Del("Server")
	w.Header().Del("X-Powered-By")
}

// sendError sends a structured error response.
//
// This method creates an ErrorResponse with:
//   - HTTP status text as error code
//   - Human-readable error message
//   - Request ID for tracing
//   - Timestamp for auditing
//
// Parameters:
//   - w: HTTP response writer
//   - status: HTTP status code (400, 429, 500, etc.)
//   - message: Human-readable error message
//   - requestID: Request ID for tracing
//
// Example Response:
//   {
//     "error": "Internal Server Error",
//     "message": "Failed to retrieve mode information",
//     "request_id": "550e8400-e29b-41d4-a716-446655440000",
//     "timestamp": "2025-11-17T12:36:00Z"
//   }
func (h *PublishingModeHandler) sendError(w http.ResponseWriter, status int, message string, requestID string) {
	errorResponse := apiservices.ErrorResponse{
		Error:     http.StatusText(status),
		Message:   message,
		RequestID: requestID,
		Timestamp: time.Now(),
	}

	h.sendJSON(w, status, errorResponse)
}

