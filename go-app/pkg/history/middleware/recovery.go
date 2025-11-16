package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
	
	apierrors "github.com/vitaliisemenov/alert-history/internal/api/errors"
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
)

// RecoveryMiddleware recovers from panics and returns a proper error response
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					// Get request ID from context
					requestID := middleware.GetRequestID(r.Context())
					
					// Log panic with stack trace
					logger.Error("Panic recovered",
						"request_id", requestID,
						"error", err,
						"stack", string(debug.Stack()),
						"method", r.Method,
						"path", r.URL.Path,
					)
					
					// Return 500 error response
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					
					errorResp := apierrors.InternalError("An internal error occurred").
						WithRequestID(requestID)
					
					if err := json.NewEncoder(w).Encode(errorResp); err != nil {
						logger.Error("Failed to encode error response", "error", err)
					}
				}
			}()
			
			next.ServeHTTP(w, r)
		})
	}
}

