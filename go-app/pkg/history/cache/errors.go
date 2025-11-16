package cache

import "fmt"

// CacheError represents a cache operation error
type CacheError struct {
	Message string
	Type    string
	Cause   error
}

func (e *CacheError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *CacheError) Unwrap() error {
	return e.Cause
}

// Error types
const (
	ErrTypeNotFound        = "NOT_FOUND"
	ErrTypeConnectionError = "CONNECTION_ERROR"
	ErrTypeSerialization  = "SERIALIZATION_ERROR"
	ErrTypeInvalidConfig   = "INVALID_CONFIG"
	ErrTypeTimeout         = "TIMEOUT"
)

// Predefined errors
var (
	ErrNotFound = &CacheError{
		Message: "cache key not found",
		Type:    ErrTypeNotFound,
	}
	ErrConnectionFailed = &CacheError{
		Message: "cache connection failed",
		Type:    ErrTypeConnectionError,
	}
)

// ErrInvalidConfig creates an invalid configuration error
func ErrInvalidConfig(msg string) error {
	return &CacheError{
		Message: msg,
		Type:    ErrTypeInvalidConfig,
	}
}

// ErrSerialization creates a serialization error
func ErrSerialization(msg string, cause error) error {
	return &CacheError{
		Message: msg,
		Type:    ErrTypeSerialization,
		Cause:   cause,
	}
}

// ErrTimeout creates a timeout error
func ErrTimeout(msg string) error {
	return &CacheError{
		Message: msg,
		Type:    ErrTypeTimeout,
	}
}

