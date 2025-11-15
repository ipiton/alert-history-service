// Package middleware provides HTTP middleware components.
package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestRecoveryMiddleware_NoPanic tests that middleware passes through when no panic occurs
func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	})

	handler := recovery.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !nextCalled {
		t.Error("Expected next handler to be called")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	if body := rr.Body.String(); body != "success" {
		t.Errorf("Expected body 'success', got %q", body)
	}
}

// TestRecoveryMiddleware_PanicRecovery tests panic recovery
func TestRecoveryMiddleware_PanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	handler := recovery.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	// Should not panic, but recover
	defer func() {
		if r := recover(); r != nil {
			t.Error("Expected panic to be recovered by middleware")
		}
	}()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d after panic, got %d", http.StatusInternalServerError, rr.Code)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "error") {
		t.Errorf("Expected error response, got: %s", body)
	}
}

// TestRecoveryMiddleware_PanicWithDifferentTypes tests panic with different types
func TestRecoveryMiddleware_PanicWithDifferentTypes(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	testCases := []struct {
		name      string
		panicVal  interface{}
		expectErr bool
	}{
		{"string panic", "error message", true},
		{"error panic", http.ErrServerClosed, true},
		{"int panic", 42, true},
		{"nil panic", nil, true},
		{"struct panic", struct{ msg string }{"error"}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				panic(tc.panicVal)
			})

			handler := recovery.Middleware(next)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tc.expectErr && rr.Code != http.StatusInternalServerError {
				t.Errorf("Expected status 500, got %d", rr.Code)
			}
		})
	}
}

// TestRecoveryMiddleware_HeadersAlreadyWritten tests panic after headers written
func TestRecoveryMiddleware_HeadersAlreadyWritten(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("partial response"))
		panic("panic after headers")
	})

	handler := recovery.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Headers already sent, so still 200
	if rr.Code != http.StatusOK {
		t.Logf("Status code after panic with headers sent: %d", rr.Code)
	}
}

// TestRecoveryMiddleware_Concurrent tests concurrent requests with panics
func TestRecoveryMiddleware_Concurrent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("panic") == "true" {
			panic("concurrent panic")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := recovery.Middleware(next)

	const numRequests = 10
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(shouldPanic bool) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if shouldPanic {
				q := req.URL.Query()
				q.Add("panic", "true")
				req.URL.RawQuery = q.Encode()
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results <- rr.Code
		}(i%2 == 0) // Half requests panic
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		code := <-results
		if code != http.StatusOK && code != http.StatusInternalServerError {
			t.Errorf("Unexpected status code: %d", code)
		}
	}
}

// BenchmarkRecoveryMiddleware benchmarks recovery middleware performance
func BenchmarkRecoveryMiddleware(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := recovery.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkRecoveryMiddleware_WithPanic benchmarks panic recovery overhead
func BenchmarkRecoveryMiddleware_WithPanic(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	recovery := NewRecoveryMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("benchmark panic")
	})

	handler := recovery.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
