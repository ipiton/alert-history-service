package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRecoveryMiddleware_PanicRecovery tests panic recovery
func TestRecoveryMiddleware_PanicRecovery(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	mw := RecoveryMiddleware(nil)
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	// RequestID will be added by RequestIDMiddleware
	w := httptest.NewRecorder()

	// Should not panic
	wrapped.ServeHTTP(w, req)

	// Should return 500 error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("RecoveryMiddleware() status = %v, want %v", w.Code, http.StatusInternalServerError)
	}

	// Should return JSON error response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}
}

// TestRecoveryMiddleware_NoPanic tests normal request handling
func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mw := RecoveryMiddleware(nil)
	wrapped := mw(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("RecoveryMiddleware() status = %v, want %v", w.Code, http.StatusOK)
	}
}
