package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ErrorCode represents standard API error codes
type ErrorCode string

const (
	// 4xx Client errors
	CodeValidationError    ErrorCode = "VALIDATION_ERROR"
	CodeAuthenticationError ErrorCode = "AUTHENTICATION_ERROR"
	CodeAuthorizationError  ErrorCode = "AUTHORIZATION_ERROR"
	CodeNotFound            ErrorCode = "NOT_FOUND"
	CodeConflict            ErrorCode = "CONFLICT"
	CodeRateLimitExceeded   ErrorCode = "RATE_LIMIT_EXCEEDED"

	// 5xx Server errors
	CodeInternalError          ErrorCode = "INTERNAL_ERROR"
	CodeServiceUnavailable     ErrorCode = "SERVICE_UNAVAILABLE"
	CodeTargetUnavailable      ErrorCode = "TARGET_UNAVAILABLE"
	CodePublishingQueueFull    ErrorCode = "PUBLISHING_QUEUE_FULL"
	CodeClassificationTimeout  ErrorCode = "CLASSIFICATION_TIMEOUT"
	CodeLLMError               ErrorCode = "LLM_ERROR"
	CodeDiscoveryError         ErrorCode = "DISCOVERY_ERROR"
	CodeHealthCheckFailed      ErrorCode = "HEALTH_CHECK_FAILED"
	CodeDLQReplayError         ErrorCode = "DLQ_REPLAY_ERROR"
)

// APIError represents a structured API error
type APIError struct {
	Code           ErrorCode              `json:"code"`
	Message        string                 `json:"message"`
	Details        interface{}            `json:"details,omitempty"`
	RequestID      string                 `json:"request_id,omitempty"`
	Timestamp      string                 `json:"timestamp"`
	DocumentationURL string               `json:"documentation_url,omitempty"`
}

// ErrorResponse wraps APIError for JSON responses
type ErrorResponse struct {
	Error APIError `json:"error"`
}

// NewAPIError creates a new API error
func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// WithDetails adds details to the error
func (e *APIError) WithDetails(details interface{}) *APIError {
	e.Details = details
	return e
}

// WithRequestID adds request ID to the error
func (e *APIError) WithRequestID(requestID string) *APIError {
	e.RequestID = requestID
	return e
}

// WithDocumentationURL adds documentation URL to the error
func (e *APIError) WithDocumentationURL(url string) *APIError {
	e.DocumentationURL = url
	return e
}

// StatusCode returns HTTP status code for error code
func (e *APIError) StatusCode() int {
	switch e.Code {
	case CodeValidationError:
		return http.StatusBadRequest
	case CodeAuthenticationError:
		return http.StatusUnauthorized
	case CodeAuthorizationError:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeConflict:
		return http.StatusConflict
	case CodeRateLimitExceeded:
		return http.StatusTooManyRequests
	case CodeInternalError:
		return http.StatusInternalServerError
	case CodeLLMError:
		return http.StatusBadGateway
	case CodeServiceUnavailable, CodeTargetUnavailable, CodePublishingQueueFull, CodeDiscoveryError, CodeHealthCheckFailed:
		return http.StatusServiceUnavailable
	case CodeClassificationTimeout:
		return http.StatusGatewayTimeout
	case CodeDLQReplayError:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Error implements error interface
func (e *APIError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// WriteError writes APIError as JSON response
func WriteError(w http.ResponseWriter, err *APIError) {
	response := ErrorResponse{Error: *err}
	statusCode := err.StatusCode()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// Helper functions for common errors

// ValidationError creates a validation error
func ValidationError(message string) *APIError {
	return NewAPIError(CodeValidationError, message)
}

// AuthenticationError creates an authentication error
func AuthenticationError(message string) *APIError {
	return NewAPIError(CodeAuthenticationError, message)
}

// AuthorizationError creates an authorization error
func AuthorizationError(message string) *APIError {
	return NewAPIError(CodeAuthorizationError, message)
}

// NotFoundError creates a not found error
func NotFoundError(resource string) *APIError {
	return NewAPIError(CodeNotFound, fmt.Sprintf("%s not found", resource))
}

// ConflictError creates a conflict error
func ConflictError(message string) *APIError {
	return NewAPIError(CodeConflict, message)
}

// RateLimitError creates a rate limit error
func RateLimitError() *APIError {
	return NewAPIError(CodeRateLimitExceeded, "Rate limit exceeded. Please retry later.")
}

// InternalError creates an internal server error
func InternalError(message string) *APIError {
	return NewAPIError(CodeInternalError, message)
}

// ServiceUnavailableError creates a service unavailable error
func ServiceUnavailableError(service string) *APIError {
	return NewAPIError(CodeServiceUnavailable, fmt.Sprintf("%s is currently unavailable", service))
}

// TargetUnavailableError creates a target unavailable error
func TargetUnavailableError(targetName string) *APIError {
	return NewAPIError(CodeTargetUnavailable, fmt.Sprintf("Publishing target '%s' is unavailable", targetName))
}

// PublishingQueueFullError creates a queue full error
func PublishingQueueFullError() *APIError {
	return NewAPIError(CodePublishingQueueFull, "Publishing queue is full. Please retry later.")
}

// ClassificationTimeoutError creates a classification timeout error
func ClassificationTimeoutError() *APIError {
	return NewAPIError(CodeClassificationTimeout, "LLM classification request timed out")
}

// LLMError creates an LLM service error
func LLMError(message string) *APIError {
	return NewAPIError(CodeLLMError, fmt.Sprintf("LLM service error: %s", message))
}
