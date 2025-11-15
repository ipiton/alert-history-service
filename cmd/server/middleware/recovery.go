package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// RecoveryMiddleware recovers from panics and returns 500 error
func RecoveryMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Log panic with stack trace
					logger.Error("Panic recovered",
						"error", err,
						"stack", string(debug.Stack()),
						"request_id", GetRequestID(r.Context()),
						"path", r.URL.Path,
						"method", r.Method,
						"remote_addr", r.RemoteAddr,
					)

					// Write 500 error response
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Request-ID", GetRequestID(r.Context()))
					w.WriteHeader(http.StatusInternalServerError)
					
					json.NewEncoder(w).Encode(map[string]string{
						"status":     "error",
						"message":    "Internal server error",
						"request_id": GetRequestID(r.Context()),
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

