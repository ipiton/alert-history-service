package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewMetricsManager(t *testing.T) {
	config := Config{
		Namespace: "test",
		Subsystem: "http",
	}

	manager := NewMetricsManager(config)

	if manager == nil {
		t.Fatal("Expected non-nil MetricsManager")
	}
}

func TestHTTPMetricsMiddleware(t *testing.T) {
	config := Config{
		Namespace: "test",
		Subsystem: "http",
	}

	manager := NewMetricsManager(config)

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	// Wrap with metrics middleware
	wrappedHandler := manager.Middleware(testHandler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)
	rec := httptest.NewRecorder()

	// Execute request
	wrappedHandler.ServeHTTP(rec, req)

	// Verify response
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Body.String() != "test response" {
		t.Errorf("Expected 'test response', got '%s'", rec.Body.String())
	}
}

func TestHTTPMetricsMiddlewareWithDifferentStatusCodes(t *testing.T) {
	config := Config{
		Namespace: "test",
		Subsystem: "http",
	}

	manager := NewMetricsManager(config)

	tests := []struct {
		name       string
		statusCode int
		path       string
		method     string
	}{
		{"success", http.StatusOK, "/api/v1/test", "GET"},
		{"not found", http.StatusNotFound, "/api/v1/missing", "GET"},
		{"server error", http.StatusInternalServerError, "/api/v1/error", "POST"},
		{"created", http.StatusCreated, "/api/v1/create", "POST"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			})

			wrappedHandler := manager.Middleware(testHandler)
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			wrappedHandler.ServeHTTP(rec, req)

			if rec.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, rec.Code)
			}
		})
	}
}

func TestResponseWriterWrapper(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := &responseWriter{
		ResponseWriter: rec,
		statusCode:     http.StatusOK,
		responseSize:   0,
	}

	// Test WriteHeader
	wrapper.WriteHeader(http.StatusCreated)
	if wrapper.statusCode != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, wrapper.statusCode)
	}

	// Test Write
	testData := []byte("test response data")
	n, err := wrapper.Write(testData)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if n != len(testData) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testData), n)
	}
	if wrapper.responseSize != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), wrapper.responseSize)
	}
}

func TestConfig(t *testing.T) {
	config := Config{
		Namespace: "test_namespace",
		Subsystem: "test_subsystem",
	}

	if config.Namespace != "test_namespace" {
		t.Errorf("Expected namespace 'test_namespace', got '%s'", config.Namespace)
	}

	if config.Subsystem != "test_subsystem" {
		t.Errorf("Expected subsystem 'test_subsystem', got '%s'", config.Subsystem)
	}
}

func TestMiddlewareChain(t *testing.T) {
	config := Config{
		Namespace: "test",
		Subsystem: "http",
	}

	manager := NewMetricsManager(config)

	// Create a chain of handlers
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Test", "success")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("final response"))
	})

	// Wrap with metrics middleware
	wrappedHandler := manager.Middleware(finalHandler)

	// Test the chain
	req := httptest.NewRequest("POST", "/api/test", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	// Verify the response passed through correctly
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("X-Test") != "success" {
		t.Errorf("Expected header X-Test=success, got %s", rec.Header().Get("X-Test"))
	}

	if rec.Body.String() != "final response" {
		t.Errorf("Expected 'final response', got '%s'", rec.Body.String())
	}
}

// Benchmark tests
func BenchmarkMetricsMiddleware(b *testing.B) {
	config := Config{
		Namespace: "bench",
		Subsystem: "http",
	}

	manager := NewMetricsManager(config)

	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("benchmark response"))
	})

	wrappedHandler := manager.Middleware(testHandler)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/benchmark", nil)
		rec := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(rec, req)
	}
}
