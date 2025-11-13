package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDMiddleware generates or extracts request ID from headers
// and adds it to both the request context and response headers.
//
// If the incoming request has an X-Request-ID header, it will be used.
// Otherwise, a new UUID will be generated.
//
// The request ID can be retrieved from context using GetRequestID().
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Try to get request ID from header
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			// Generate new UUID if not provided
			requestID = uuid.New().String()
		}

		// Add request ID to context
		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)
		r = r.WithContext(ctx)

		// Add request ID to response headers
		w.Header().Set(RequestIDHeader, requestID)

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// GetRequestID extracts request ID from context
// Returns empty string if request ID is not found
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDContextKey).(string); ok {
		return id
	}
	return ""
}
