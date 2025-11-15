package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// LoggingMiddleware logs request and response details
func LoggingMiddleware(logger *slog.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap ResponseWriter to capture status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Log request
			logger.Info("HTTP request received",
				"request_id", GetRequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.Header.Get("User-Agent"),
				"content_length", r.ContentLength,
			)

			// Call next handler
			next.ServeHTTP(rw, r)

			// Log response
			duration := time.Since(start)
			logLevel := slog.LevelInfo
			if rw.statusCode >= 400 && rw.statusCode < 500 {
				logLevel = slog.LevelWarn
			}
			if rw.statusCode >= 500 {
				logLevel = slog.LevelError
			}

			logger.Log(r.Context(), logLevel, "HTTP response sent",
				"request_id", GetRequestID(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

