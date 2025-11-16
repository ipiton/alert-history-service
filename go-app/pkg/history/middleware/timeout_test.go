package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestTimeoutMiddleware_Timeout tests timeout behavior
func TestTimeoutMiddleware_Timeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow handler
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	mw := TimeoutMiddleware(100*time.Millisecond, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	// RequestID will be added by RequestIDMiddleware
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	// Should return 504 Gateway Timeout
	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("TimeoutMiddleware() status = %v, want %v", w.Code, http.StatusGatewayTimeout)
	}
}

// TestTimeoutMiddleware_NoTimeout tests normal request handling
func TestTimeoutMiddleware_NoTimeout(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mw := TimeoutMiddleware(1*time.Second, nil)
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("TimeoutMiddleware() status = %v, want %v", w.Code, http.StatusOK)
	}
}
