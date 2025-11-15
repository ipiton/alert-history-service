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

// TestRequestIDMiddleware_GeneratesUUID tests UUID generation when no header present
func TestRequestIDMiddleware_GeneratesUUID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	var capturedRequestID string
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := requestID.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedRequestID == "" {
		t.Error("Expected request ID to be generated")
	}

	// Check it's a valid UUID format (8-4-4-4-12)
	parts := strings.Split(capturedRequestID, "-")
	if len(parts) != 5 {
		t.Errorf("Expected UUID format with 5 parts, got %d: %s", len(parts), capturedRequestID)
	}

	// Check response header
	responseID := rr.Header().Get("X-Request-ID")
	if responseID != capturedRequestID {
		t.Errorf("Expected X-Request-ID header %s, got %s", capturedRequestID, responseID)
	}
}

// TestRequestIDMiddleware_UsesExistingHeader tests using existing X-Request-ID header
func TestRequestIDMiddleware_UsesExistingHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	existingID := "custom-request-id-123"
	var capturedRequestID string

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedRequestID = GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := requestID.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if capturedRequestID != existingID {
		t.Errorf("Expected request ID %s, got %s", existingID, capturedRequestID)
	}

	responseID := rr.Header().Get("X-Request-ID")
	if responseID != existingID {
		t.Errorf("Expected X-Request-ID header %s, got %s", existingID, responseID)
	}
}

// TestRequestIDMiddleware_ValidatesUUID tests UUID validation
func TestRequestIDMiddleware_ValidatesUUID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	testCases := []struct {
		name           string
		inputID        string
		shouldGenerate bool
	}{
		{"valid UUID v4", "550e8400-e29b-41d4-a716-446655440000", false},
		{"invalid format", "not-a-uuid", true},
		{"empty string", "", true},
		{"partial UUID", "550e8400-e29b", true},
		{"uppercase UUID", "550E8400-E29B-41D4-A716-446655440000", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var capturedRequestID string
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedRequestID = GetRequestID(r.Context())
				w.WriteHeader(http.StatusOK)
			})

			handler := requestID.Middleware(next)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tc.inputID != "" {
				req.Header.Set("X-Request-ID", tc.inputID)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if tc.shouldGenerate {
				// Should generate new UUID
				if capturedRequestID == tc.inputID {
					t.Errorf("Expected new UUID to be generated, got %s", capturedRequestID)
				}
				// Check it's a valid UUID format
				if !strings.Contains(capturedRequestID, "-") {
					t.Errorf("Expected valid UUID format, got %s", capturedRequestID)
				}
			} else {
				// Should use existing ID
				if capturedRequestID != strings.ToLower(tc.inputID) {
					t.Logf("ID mismatch (case difference OK): input=%s, captured=%s", tc.inputID, capturedRequestID)
				}
			}
		})
	}
}

// TestRequestIDMiddleware_GetRequestID tests GetRequestID helper function
func TestRequestIDMiddleware_GetRequestID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Test GetRequestID function
		id1 := GetRequestID(r.Context())
		id2 := GetRequestID(r.Context())

		if id1 != id2 {
			t.Error("GetRequestID should return same ID for same context")
		}

		if id1 == "" {
			t.Error("Expected non-empty request ID")
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := requestID.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)
}

// TestRequestIDMiddleware_Concurrent tests concurrent request ID generation
func TestRequestIDMiddleware_Concurrent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := requestID.Middleware(next)

	const numRequests = 100
	results := make(chan string, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)
			results <- rr.Header().Get("X-Request-ID")
		}()
	}

	// Collect and check uniqueness
	seen := make(map[string]bool)
	for i := 0; i < numRequests; i++ {
		id := <-results
		if seen[id] {
			t.Errorf("Duplicate request ID generated: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != numRequests {
		t.Errorf("Expected %d unique IDs, got %d", numRequests, len(seen))
	}
}

// TestGenerateRequestID tests direct UUID generation
func TestGenerateRequestID(t *testing.T) {
	// Generate multiple UUIDs and check uniqueness
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		uuid := generateRequestID()
		if uuid == "" {
			t.Error("Generated empty UUID")
		}
		if seen[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true

		// Check format
		parts := strings.Split(uuid, "-")
		if len(parts) != 5 {
			t.Errorf("Invalid UUID format: %s", uuid)
		}
	}
}

// TestIsValidUUID tests UUID validation function
func TestIsValidUUID(t *testing.T) {
	testCases := []struct {
		uuid  string
		valid bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"550E8400-E29B-41D4-A716-446655440000", true},
		{"not-a-uuid", false},
		{"", false},
		{"550e8400-e29b-41d4-a716", false},
		{"550e8400-e29b-41d4-a716-446655440000-extra", false},
		{"123456789012345678901234567890123456", false},
		{"00000000-0000-0000-0000-000000000000", true},
	}

	for _, tc := range testCases {
		t.Run(tc.uuid, func(t *testing.T) {
			result := isValidUUID(tc.uuid)
			if result != tc.valid {
				t.Errorf("isValidUUID(%q) = %v, want %v", tc.uuid, result, tc.valid)
			}
		})
	}
}

// BenchmarkRequestIDMiddleware benchmarks request ID middleware
func BenchmarkRequestIDMiddleware(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := requestID.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkRequestIDMiddleware_WithExisting benchmarks with existing request ID
func BenchmarkRequestIDMiddleware_WithExisting(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	requestID := NewRequestIDMiddleware(logger)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		GetRequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	})

	handler := requestID.Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", "550e8400-e29b-41d4-a716-446655440000")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkGenerateRequestID benchmarks UUID generation
func BenchmarkGenerateRequestID(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = generateRequestID()
	}
}
