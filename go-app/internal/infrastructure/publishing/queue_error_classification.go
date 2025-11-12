package publishing

import (
	"errors"
	"net"
	"net/http"
	"strings"
	"syscall"
)

// classifyError determines whether an error should be retried (transient) or not (permanent).
//
// Error Classification Rules:
//
// TRANSIENT (retry):
//   - HTTP 429 (Rate Limit) - retry after backoff
//   - HTTP 408 (Request Timeout) - retry immediately
//   - HTTP 502, 503, 504 (Bad Gateway, Service Unavailable, Gateway Timeout) - retry after backoff
//   - Network errors (connection refused, timeout, DNS failure)
//   - Temporary errors (net.Error with Temporary() = true)
//   - syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT
//
// PERMANENT (do NOT retry):
//   - HTTP 400 (Bad Request) - invalid payload
//   - HTTP 401 (Unauthorized) - invalid credentials
//   - HTTP 403 (Forbidden) - insufficient permissions
//   - HTTP 404 (Not Found) - invalid URL
//   - HTTP 405 (Method Not Allowed) - wrong HTTP method
//   - HTTP 422 (Unprocessable Entity) - invalid data format
//   - HTTP 5xx (except 502/503/504) - permanent server errors
//
// UNKNOWN (retry with caution):
//   - All other errors - default to transient with conservative retry
//
// Parameters:
//   - err: The error to classify
//
// Returns:
//   - ErrorType: ErrorTypeTransient, ErrorTypePermanent, or ErrorTypeUnknown
//
// Example:
//
//	errType := classifyError(err)
//	if errType == ErrorTypePermanent {
//	    // Send to DLQ
//	}
func classifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// HTTP response errors
	var httpErr interface{ StatusCode() int }
	if errors.As(err, &httpErr) {
		return classifyHTTPError(httpErr.StatusCode())
	}

	// String-based HTTP error parsing (fallback)
	errMsg := err.Error()
	if strings.Contains(errMsg, "status code:") || strings.Contains(errMsg, "HTTP ") {
		return classifyHTTPErrorString(errMsg)
	}

	// Network errors (transient)
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return ErrorTypeTransient // Network timeout
		}
		if netErr.Temporary() {
			return ErrorTypeTransient // Temporary network error
		}
	}

	// DNS errors (transient)
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return ErrorTypeTransient
	}

	// Connection refused (transient)
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return ErrorTypeTransient
	}

	// Syscall errors
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		switch syscallErr {
		case syscall.ECONNREFUSED, syscall.ECONNRESET, syscall.ETIMEDOUT:
			return ErrorTypeTransient
		}
	}

	// Default: UNKNOWN (retry with caution)
	return ErrorTypeUnknown
}

// classifyHTTPError classifies errors based on HTTP status code
func classifyHTTPError(statusCode int) ErrorType {
	switch statusCode {
	// TRANSIENT - retry
	case http.StatusRequestTimeout: // 408
		return ErrorTypeTransient
	case http.StatusTooManyRequests: // 429
		return ErrorTypeTransient
	case http.StatusBadGateway: // 502
		return ErrorTypeTransient
	case http.StatusServiceUnavailable: // 503
		return ErrorTypeTransient
	case http.StatusGatewayTimeout: // 504
		return ErrorTypeTransient

	// PERMANENT - do NOT retry
	case http.StatusBadRequest: // 400
		return ErrorTypePermanent
	case http.StatusUnauthorized: // 401
		return ErrorTypePermanent
	case http.StatusForbidden: // 403
		return ErrorTypePermanent
	case http.StatusNotFound: // 404
		return ErrorTypePermanent
	case http.StatusMethodNotAllowed: // 405
		return ErrorTypePermanent
	case http.StatusConflict: // 409
		return ErrorTypePermanent
	case http.StatusGone: // 410
		return ErrorTypePermanent
	case http.StatusUnprocessableEntity: // 422
		return ErrorTypePermanent

	// 5xx errors (except 502/503/504) - permanent
	default:
		if statusCode >= 500 && statusCode < 600 {
			// Unknown 5xx - permanent
			return ErrorTypePermanent
		}
		return ErrorTypeUnknown
	}
}

// classifyHTTPErrorString parses error message for HTTP status codes
func classifyHTTPErrorString(errMsg string) ErrorType {
	// Extract status code from error message
	// Common formats:
	// - "HTTP 404 Not Found"
	// - "status code: 503"
	// - "received 429 Too Many Requests"

	transientCodes := []string{"408", "429", "502", "503", "504"}
	permanentCodes := []string{"400", "401", "403", "404", "405", "409", "410", "422"}

	for _, code := range transientCodes {
		if strings.Contains(errMsg, code) {
			return ErrorTypeTransient
		}
	}

	for _, code := range permanentCodes {
		if strings.Contains(errMsg, code) {
			return ErrorTypePermanent
		}
	}

	// Unknown HTTP error
	return ErrorTypeUnknown
}
