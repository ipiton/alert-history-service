package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthHandler(t *testing.T) {
	// Create a request to pass to our handler
	req, err := http.NewRequest("GET", "/healthz", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Create a ResponseRecorder to record the response
	rr := httptest.NewRecorder()

	// Call the handler
	handler := http.HandlerFunc(HealthHandler)
	handler.ServeHTTP(rr, req)

	// Check the status code is what we expect
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the content type
	expectedContentType := "application/json"
	if contentType := rr.Header().Get("Content-Type"); contentType != expectedContentType {
		t.Errorf("handler returned wrong content type: got %v want %v", contentType, expectedContentType)
	}

	// Parse the response body
	var response HealthResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("failed to unmarshal response: %v", err)
	}

	// Check response fields
	if response.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", response.Status)
	}

	if response.Service != "alert-history" {
		t.Errorf("expected service 'alert-history', got '%s'", response.Service)
	}

	if response.Version != "1.0.0" {
		t.Errorf("expected version '1.0.0', got '%s'", response.Version)
	}

	// Check timestamp is recent (within last minute)
	if timestamp, err := time.Parse(time.RFC3339, response.Timestamp); err != nil {
		t.Errorf("invalid timestamp format: %v", err)
	} else if time.Since(timestamp) > time.Minute {
		t.Errorf("timestamp is too old: %v", timestamp)
	}
}
