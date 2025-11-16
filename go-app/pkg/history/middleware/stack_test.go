package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestStack_Apply tests middleware stack application
func TestStack_Apply(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	config := DefaultStackConfig(nil)
	config.EnableAuth = false // Disable auth for unit tests
	config.EnableAuthz = false
	config.EnableRateLimit = false

	stack := NewStack(config)
	wrapped := stack.Apply(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stack.Apply() status = %v, want %v", w.Code, http.StatusOK)
	}
}

// TestStack_ApplyFunc tests ApplyFunc convenience method
func TestStack_ApplyFunc(t *testing.T) {
	handlerFunc := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}

	config := DefaultStackConfig(nil)
	config.EnableAuth = false
	config.EnableAuthz = false
	config.EnableRateLimit = false

	stack := NewStack(config)
	wrapped := stack.ApplyFunc(handlerFunc)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Stack.ApplyFunc() status = %v, want %v", w.Code, http.StatusOK)
	}
}

// TestStackConfig_Default tests default configuration
func TestStackConfig_Default(t *testing.T) {
	config := DefaultStackConfig(nil)

	if !config.EnableRecovery {
		t.Error("DefaultStackConfig() EnableRecovery = false, want true")
	}
	if !config.EnableRequestID {
		t.Error("DefaultStackConfig() EnableRequestID = false, want true")
	}
	if !config.EnableLogging {
		t.Error("DefaultStackConfig() EnableLogging = false, want true")
	}
	if !config.EnableMetrics {
		t.Error("DefaultStackConfig() EnableMetrics = false, want true")
	}
}
