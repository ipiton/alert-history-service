// Package middleware provides HTTP middleware for the Alert History Service.
package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"regexp"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Context keys
const (
	RequestIDKey contextKey = "request_id"
)

// GetRequestID extracts request ID from context
// Returns "unknown" if request ID is not found
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return "unknown"
}

// SetRequestID adds request ID to context
func SetRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// generateRequestID generates a UUID v4 request ID
func generateRequestID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("req-%d", timeNow().UnixNano())
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40 // Version 4
	b[8] = (b[8] & 0x3f) | 0x80 // Variant 10

	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// isValidUUID validates UUID v4 format
func isValidUUID(s string) bool {
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	return uuidPattern.MatchString(s)
}

// timeNow is a variable to allow mocking in tests
var timeNow = func() interface{ UnixNano() int64 } {
	return timeProvider{}
}

type timeProvider struct{}

func (timeProvider) UnixNano() int64 {
	return 0 // Placeholder, will be replaced with actual time.Now() in production
}
