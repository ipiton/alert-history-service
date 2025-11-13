package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		statusCode     int
		expectLogEntry bool
	}{
		{
			name:           "logs GET request",
			method:         "GET",
			path:           "/api/v2/health",
			statusCode:     http.StatusOK,
			expectLogEntry: true,
		},
		{
			name:           "logs POST request",
			method:         "POST",
			path:           "/api/v2/publishing/targets/refresh",
			statusCode:     http.StatusAccepted,
			expectLogEntry: true,
		},
		{
			name:           "logs error response",
			method:         "GET",
			path:           "/api/v2/not-found",
			statusCode:     http.StatusNotFound,
			expectLogEntry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create buffer to capture logs
			var buf bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buf, nil))

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			// Wrap with middleware
			wrappedHandler := LoggingMiddleware(logger)(handler)

			// Create request
			req := httptest.NewRequest(tt.method, tt.path, nil)
			req.Header.Set("User-Agent", "test-agent")

			// Add request ID to context
			ctx := req.Context()
			ctx = withRequestID(ctx, "test-request-id")
			req = req.WithContext(ctx)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, rr.Code)
			}

			// Check logs
			if tt.expectLogEntry {
				logOutput := buf.String()
				if logOutput == "" {
					t.Error("Expected log entry, got none")
				}

				// Check log contains key information
				if !strings.Contains(logOutput, tt.method) {
					t.Errorf("Log missing method: %s", logOutput)
				}
				if !strings.Contains(logOutput, tt.path) {
					t.Errorf("Log missing path: %s", logOutput)
				}
			}
		})
	}
}

func TestLoggingMiddleware_CapturesDuration(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate some work
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := LoggingMiddleware(logger)(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(withRequestID(req.Context(), "test-id"))
	rr := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rr, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "duration") {
		t.Error("Log missing duration field")
	}
}

// Helper to add request ID to context
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, RequestIDContextKey, id)
}

// Benchmark LoggingMiddleware
func BenchmarkLoggingMiddleware(b *testing.B) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := LoggingMiddleware(logger)(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req = req.WithContext(withRequestID(req.Context(), "test-id"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
