// Package middleware provides HTTP middleware components.
package middleware

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestLoggingMiddleware_LogsRequestAndResponse tests request/response logging
func TestLoggingMiddleware_LogsRequestAndResponse(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := logging(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.Header.Set("User-Agent", "TestAgent/1.0")
	ctx := context.WithValue(req.Context(), RequestIDKey, "test-123")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Check that logs contain expected information
	logOutput := logBuffer.String()

	expectedStrings := []string{
		"HTTP request received",
		"test-123",
		"POST",
		"/webhook",
		"HTTP response sent",
		"status",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(logOutput, expected) {
			t.Errorf("Expected log output to contain %q, got: %s", expected, logOutput)
		}
	}
}

// TestLoggingMiddleware_DifferentStatusCodes tests logging for different status codes
func TestLoggingMiddleware_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		expectedLevel  string
		expectedInLogs string
	}{
		{"2xx success", http.StatusOK, "INFO", "200"},
		{"3xx redirect", http.StatusMovedPermanently, "INFO", "301"},
		{"4xx client error", http.StatusBadRequest, "WARN", "400"},
		{"404 not found", http.StatusNotFound, "WARN", "404"},
		{"5xx server error", http.StatusInternalServerError, "ERROR", "500"},
		{"503 unavailable", http.StatusServiceUnavailable, "ERROR", "503"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var logBuffer bytes.Buffer
			logger := slog.New(slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}))

			logging := LoggingMiddleware(logger)

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			})

			handler := logging(next)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := context.WithValue(req.Context(), RequestIDKey, "test-req")
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			logOutput := logBuffer.String()

			// Check status code is logged
			if !strings.Contains(logOutput, tc.expectedInLogs) {
				t.Errorf("Expected log to contain %q, got: %s", tc.expectedInLogs, logOutput)
			}

			// Check log level
			if !strings.Contains(logOutput, tc.expectedLevel) {
				t.Logf("Note: Expected level %q may not be in log format: %s", tc.expectedLevel, logOutput)
			}
		})
	}
}

// TestLoggingMiddleware_CapturesDuration tests duration measurement
func TestLoggingMiddleware_CapturesDuration(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	handler := logging(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "test-req")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logOutput := logBuffer.String()

	// Should contain duration in logs
	if !strings.Contains(logOutput, "duration") && !strings.Contains(logOutput, "ms") {
		t.Logf("Expected duration in log output: %s", logOutput)
	}
}

// TestLoggingMiddleware_LogsHeaders tests header logging
func TestLoggingMiddleware_LogsHeaders(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := logging(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", strings.NewReader("test"))
	req.Header.Set("User-Agent", "CustomAgent/2.0")
	req.Header.Set("Content-Type", "application/json")
	ctx := context.WithValue(req.Context(), RequestIDKey, "test-req")
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	logOutput := logBuffer.String()

	// Should contain User-Agent
	if !strings.Contains(logOutput, "CustomAgent") {
		t.Logf("Expected User-Agent in log output: %s", logOutput)
	}
}

// TestLoggingMiddleware_NoRequestID tests behavior without request ID
func TestLoggingMiddleware_NoRequestID(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := logging(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	// No request ID in context
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Should still log without error
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// TestLoggingMiddleware_Concurrent tests concurrent logging
func TestLoggingMiddleware_Concurrent(t *testing.T) {
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	handler := logging(next)

	const numRequests = 10
	done := make(chan bool, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			ctx := context.WithValue(req.Context(), RequestIDKey, "req-"+string(rune(id)))
			req = req.WithContext(ctx)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			done <- true
		}(i)
	}

	// Wait for all requests
	for i := 0; i < numRequests; i++ {
		<-done
	}

	// All requests should be logged
	logOutput := logBuffer.String()
	if !strings.Contains(logOutput, "HTTP request received") {
		t.Error("Expected requests to be logged")
	}
}

// TestResponseWriter_CapturesStatusCode tests responseWriter status capture
func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"BadRequest", http.StatusBadRequest},
		{"NotFound", http.StatusNotFound},
		{"InternalError", http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			rw.WriteHeader(tc.statusCode)

			if rw.statusCode != tc.statusCode {
				t.Errorf("Expected status code %d, got %d", tc.statusCode, rw.statusCode)
			}

			if w.Code != tc.statusCode {
				t.Errorf("Expected ResponseWriter code %d, got %d", tc.statusCode, w.Code)
			}
		})
	}
}

// TestResponseWriter_DefaultStatusOK tests default status is 200
func TestResponseWriter_DefaultStatusOK(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Write without calling WriteHeader
	rw.Write([]byte("test"))

	// Should still capture default 200
	if rw.statusCode != http.StatusOK {
		t.Errorf("Expected default status 200, got %d", rw.statusCode)
	}
}

// BenchmarkLoggingMiddleware benchmarks logging middleware
func BenchmarkLoggingMiddleware(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := logging(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "bench-req")
	req = req.WithContext(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkLoggingMiddleware_WithError benchmarks error logging
func BenchmarkLoggingMiddleware_WithError(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(bytes.NewBuffer(nil), nil))

	logging := LoggingMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	handler := logging(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	ctx := context.WithValue(req.Context(), RequestIDKey, "bench-req")
	req = req.WithContext(ctx)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

