package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
)

// TimeoutMiddleware adds a timeout to request context
// If timeout is exceeded, returns 504 Gateway Timeout
func TimeoutMiddleware(timeout time.Duration, logger *slog.Logger) func(http.Handler) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	
	if timeout <= 0 {
		timeout = 30 * time.Second // Default timeout
	}
	
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Create context with timeout
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			
			// Create channel to detect if handler completed
			done := make(chan bool, 1)
			
			// Wrap response writer to detect writes
			rw := &timeoutResponseWriter{
				ResponseWriter: w,
				done:           done,
			}
			
			// Run handler in goroutine
			go func() {
				next.ServeHTTP(rw, r.WithContext(ctx))
				done <- true
			}()
			
			// Wait for either completion or timeout
			select {
			case <-done:
				// Handler completed successfully
				return
			case <-ctx.Done():
				// Timeout exceeded
				if !rw.wroteHeader {
					requestID := middleware.GetRequestID(r.Context())
					logger.Warn("Request timeout exceeded",
						"request_id", requestID,
						"timeout", timeout,
						"method", r.Method,
						"path", r.URL.Path,
					)
					
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusGatewayTimeout)
					
					// Write timeout error response
					timeoutError := map[string]interface{}{
						"error": map[string]interface{}{
							"code":    "REQUEST_TIMEOUT",
							"message": "Request timeout exceeded",
							"request_id": requestID,
							"timeout_seconds": timeout.Seconds(),
						},
					}
					
					if err := json.NewEncoder(w).Encode(timeoutError); err != nil {
						logger.Error("Failed to encode timeout error", "error", err)
					}
				}
			}
		})
	}
}

// timeoutResponseWriter wraps http.ResponseWriter to detect writes
type timeoutResponseWriter struct {
	http.ResponseWriter
	done        chan bool
	wroteHeader bool
}

func (rw *timeoutResponseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *timeoutResponseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

