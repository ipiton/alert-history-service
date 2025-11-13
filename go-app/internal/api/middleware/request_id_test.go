package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		existingID     string
		expectNewID    bool
		expectInHeader bool
	}{
		{
			name:           "generates new ID when not present",
			existingID:     "",
			expectNewID:    true,
			expectInHeader: true,
		},
		{
			name:           "preserves existing ID",
			existingID:     "existing-request-id-123",
			expectNewID:    false,
			expectInHeader: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if request ID is in context
				requestID := GetRequestID(r.Context())
				if requestID == "" {
					t.Error("Request ID not found in context")
					return
				}

				if tt.existingID != "" && requestID != tt.existingID {
					t.Errorf("Expected request ID %s, got %s", tt.existingID, requestID)
				}

				if tt.expectNewID && requestID == "" {
					t.Error("Expected new request ID to be generated")
				}

				w.WriteHeader(http.StatusOK)
			})

			// Wrap with middleware
			wrappedHandler := RequestIDMiddleware(handler)

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			if tt.existingID != "" {
				req.Header.Set("X-Request-ID", tt.existingID)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute request
			wrappedHandler.ServeHTTP(rr, req)

			// Check response header
			if tt.expectInHeader {
				headerID := rr.Header().Get("X-Request-ID")
				if headerID == "" {
					t.Error("X-Request-ID header not set in response")
				}

				if tt.existingID != "" && headerID != tt.existingID {
					t.Errorf("Expected X-Request-ID header %s, got %s", tt.existingID, headerID)
				}
			}
		})
	}
}

func TestRequestIDMiddleware_Integration(t *testing.T) {
	// Test that request ID flows through multiple handlers
	var capturedID string

	handler1 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("Request ID not found in first handler")
			return
		}
		capturedID = id
	})

	handler2 := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := GetRequestID(r.Context())
		if id == "" {
			t.Error("Request ID not found in second handler")
			return
		}

		if id != capturedID {
			t.Errorf("Request ID changed between handlers: %s != %s", capturedID, id)
		}

		w.WriteHeader(http.StatusOK)
	})

	// Chain handlers
	chain := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler1.ServeHTTP(w, r)
		handler2.ServeHTTP(w, r)
	}))

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	chain.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// Benchmark RequestIDMiddleware
func BenchmarkRequestIDMiddleware(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestIDMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rr, req)
	}
}
