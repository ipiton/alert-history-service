package middleware

import (
	"net/http"
)

// RequestIDMiddleware generates or extracts request ID from X-Request-ID header
// and adds it to the request context
func RequestIDMiddleware() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to extract from X-Request-ID header
			requestID := r.Header.Get("X-Request-ID")

			// Generate if missing
			if requestID == "" {
				requestID = generateRequestID()
			}

			// Validate format (UUID v4)
			if !isValidUUID(requestID) {
				requestID = generateRequestID()
			}

			// Add to context
			ctx := SetRequestID(r.Context(), requestID)
			r = r.WithContext(ctx)

			// Add to response headers
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r)
		})
	}
}

