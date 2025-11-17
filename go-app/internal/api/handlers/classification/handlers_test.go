package classification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/api/middleware"
	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// MockClassifier implements core.AlertClassifier for testing
type MockClassifier struct {
	classifyFunc func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error)
}

func (m *MockClassifier) Classify(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
	if m.classifyFunc != nil {
		return m.classifyFunc(ctx, alert)
	}
	return &core.ClassificationResult{
		Severity:       core.SeverityWarning,
		Confidence:     0.95,
		Reasoning:      "Test classification",
		ProcessingTime: 0.05,
	}, nil
}

// MockClassificationService implements both core.AlertClassifier and services.ClassificationService for testing
type MockClassificationService struct {
	classifyFunc              func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error)
	getCachedFunc             func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error)
	invalidateCacheFunc       func(ctx context.Context, fingerprint string) error
	invalidateCacheCalled     bool
	invalidateCacheFingerprint string
	stats                     services.ClassificationStats
}

func (m *MockClassificationService) Classify(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
	if m.classifyFunc != nil {
		return m.classifyFunc(ctx, alert)
	}
	return &core.ClassificationResult{
		Severity:       core.SeverityWarning,
		Confidence:     0.95,
		Reasoning:      "Test classification",
		ProcessingTime: 0.05,
	}, nil
}

func (m *MockClassificationService) ClassifyAlert(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
	return m.Classify(ctx, alert)
}

func (m *MockClassificationService) GetStats() services.ClassificationStats {
	return m.stats
}

func (m *MockClassificationService) GetCachedClassification(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
	if m.getCachedFunc != nil {
		return m.getCachedFunc(ctx, fingerprint)
	}
	return nil, nil
}

func (m *MockClassificationService) ClassifyBatch(ctx context.Context, alerts []*core.Alert) ([]*core.ClassificationResult, error) {
	return nil, nil
}

func (m *MockClassificationService) InvalidateCache(ctx context.Context, fingerprint string) error {
	m.invalidateCacheCalled = true
	m.invalidateCacheFingerprint = fingerprint
	if m.invalidateCacheFunc != nil {
		return m.invalidateCacheFunc(ctx, fingerprint)
	}
	return nil
}

func (m *MockClassificationService) WarmCache(ctx context.Context, alerts []*core.Alert) error {
	return nil
}

func (m *MockClassificationService) Health(ctx context.Context) error {
	return nil
}

func TestClassifyAlert_Success(t *testing.T) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
			Labels:      map[string]string{"severity": "warning"},
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Result == nil {
		t.Error("Expected result, got nil")
		return
	}

	if response.Result.Severity != core.SeverityWarning {
		t.Errorf("Expected severity warning, got %s", response.Result.Severity)
	}

	if response.Result.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", response.Result.Confidence)
	}

	if response.Cached {
		t.Error("Expected cached=false (no service), got cached=true")
	}

	if response.Timestamp.IsZero() {
		t.Error("Expected timestamp, got zero time")
	}
}

func TestClassifyAlert_InvalidJSON(t *testing.T) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestClassifyAlert_MissingAlert(t *testing.T) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: nil,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}
}

func TestClassifyAlert_ClassifierError(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	// Timeout errors return 504 (Gateway Timeout) via ClassificationTimeoutError
	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status 504, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestClassifyAlert_ForceFlag tests force flag functionality
func TestClassifyAlert_ForceFlag(t *testing.T) {
	mockService := &MockClassificationService{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityCritical,
				Confidence:     0.98,
				Reasoning:      "Forced classification",
				ProcessingTime: 0.1,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	// Verify cache invalidation was called
	if !mockService.invalidateCacheCalled {
		t.Error("Expected InvalidateCache to be called when force=true")
	}

	if mockService.invalidateCacheFingerprint != "test-123" {
		t.Errorf("Expected InvalidateCache with fingerprint 'test-123', got '%s'", mockService.invalidateCacheFingerprint)
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Cached {
		t.Error("Expected cached=false when force=true, got cached=true")
	}

	if response.Result.Severity != core.SeverityCritical {
		t.Errorf("Expected severity critical, got %s", response.Result.Severity)
	}
}

// TestClassifyAlert_CacheHit tests cache hit scenario
func TestClassifyAlert_CacheHit(t *testing.T) {
	cachedResult := &core.ClassificationResult{
		Severity:       core.SeverityWarning,
		Confidence:     0.85,
		Reasoning:      "Cached classification",
		ProcessingTime: 0.01,
	}

	mockService := &MockClassificationService{
		getCachedFunc: func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
			if fingerprint == "test-123" {
				return cachedResult, nil
			}
			return nil, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !response.Cached {
		t.Error("Expected cached=true when cache hit, got cached=false")
	}

	if response.Result.Severity != cachedResult.Severity {
		t.Errorf("Expected severity %s, got %s", cachedResult.Severity, response.Result.Severity)
	}
}

// TestClassifyAlert_CacheMiss tests cache miss scenario
func TestClassifyAlert_CacheMiss(t *testing.T) {
	mockService := &MockClassificationService{
		getCachedFunc: func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
			return nil, nil // Cache miss
		},
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityInfo,
				Confidence:     0.75,
				Reasoning:      "New classification",
				ProcessingTime: 0.2,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Cached {
		t.Error("Expected cached=false when cache miss, got cached=true")
	}

	if response.Result.Severity != core.SeverityInfo {
		t.Errorf("Expected severity info, got %s", response.Result.Severity)
	}
}

// TestClassifyAlert_ValidationErrors tests various validation errors
func TestClassifyAlert_ValidationErrors(t *testing.T) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	tests := []struct {
		name    string
		alert   *core.Alert
		wantErr bool
	}{
		{
			name: "missing fingerprint",
			alert: &core.Alert{
				AlertName: "TestAlert",
				Status:    core.StatusFiring,
				StartsAt:  time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing alert_name",
			alert: &core.Alert{
				Fingerprint: "test-123",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      "invalid",
				StartsAt:    time.Now(),
			},
			wantErr: true,
		},
		{
			name: "missing starts_at",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
			},
			wantErr: true,
		},
		{
			name: "invalid generator_url",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
				GeneratorURL: stringPtr("not-a-url"),
			},
			wantErr: true,
		},
		{
			name: "valid alert",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqBody := ClassifyRequest{
				Alert: tt.alert,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

			rr := httptest.NewRecorder()
			handlers.ClassifyAlert(rr, req)

			if tt.wantErr {
				if rr.Code != http.StatusBadRequest {
					t.Errorf("Expected status 400, got %d: %s", rr.Code, rr.Body.String())
				}
			} else {
				if rr.Code != http.StatusOK {
					t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
				}
			}
		})
	}
}

// TestClassifyAlert_TimeoutError tests timeout error handling
func TestClassifyAlert_TimeoutError(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return nil, context.DeadlineExceeded
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	// Timeout errors return 504 (Gateway Timeout) via ClassificationTimeoutError
	if rr.Code != http.StatusGatewayTimeout {
		t.Errorf("Expected status 504, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestClassifyAlert_ServiceUnavailable tests service unavailable error handling
func TestClassifyAlert_ServiceUnavailable(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return nil, errors.New("circuit breaker is open - service unavailable")
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestClassifyAlert_ModelExtraction tests model extraction from metadata
func TestClassifyAlert_ModelExtraction(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.95,
				Reasoning:      "Test",
				ProcessingTime: 0.05,
				Metadata: map[string]interface{}{
					"model": "gpt-4",
				},
			}, nil
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", response.Model)
	}
}

// TestValidateAlert tests validateAlert helper function
func TestValidateAlert(t *testing.T) {
	handlers := NewClassificationHandlers(&MockClassifier{}, slog.Default())

	tests := []struct {
		name    string
		alert   *core.Alert
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid alert",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing fingerprint",
			alert: &core.Alert{
				AlertName: "TestAlert",
				Status:    core.StatusFiring,
				StartsAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "fingerprint is required",
		},
		{
			name: "missing alert_name",
			alert: &core.Alert{
				Fingerprint: "test-123",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "alert_name is required",
		},
		{
			name: "invalid status",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      "invalid",
				StartsAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "status must be",
		},
		{
			name: "missing starts_at",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
			},
			wantErr: true,
			errMsg:  "starts_at is required",
		},
		{
			name: "invalid generator_url",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
				GeneratorURL: stringPtr("not-a-url"),
			},
			wantErr: true,
			errMsg:  "generator_url must be",
		},
		{
			name: "valid generator_url",
			alert: &core.Alert{
				Fingerprint: "test-123",
				AlertName:   "TestAlert",
				Status:      core.StatusFiring,
				StartsAt:    time.Now(),
				GeneratorURL: stringPtr("https://prometheus.example.com"),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.validateAlert(tt.alert)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected validation error, got nil")
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Expected error message containing '%s', got '%s'", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

// TestIsTimeoutError tests isTimeoutError helper function
func TestIsTimeoutError(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name: "timeout error",
			err:   errors.New("context deadline exceeded"),
			want:  true,
		},
		{
			name: "deadline exceeded",
			err:   context.DeadlineExceeded,
			want:  true,
		},
		{
			name: "timeout in message",
			err:   errors.New("request timeout"),
			want:  true,
		},
		{
			name: "other error",
			err:   errors.New("some other error"),
			want:  false,
		},
		{
			name: "nil error",
			err:   nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTimeoutError(tt.err)
			if got != tt.want {
				t.Errorf("isTimeoutError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestIsServiceUnavailable tests isServiceUnavailable helper function
func TestIsServiceUnavailable(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		want    bool
	}{
		{
			name: "circuit breaker",
			err:   errors.New("circuit breaker is open"),
			want:  true,
		},
		{
			name: "service unavailable",
			err:   errors.New("service unavailable"),
			want:  true,
		},
		{
			name: "unavailable in message",
			err:   errors.New("LLM service unavailable"),
			want:  true,
		},
		{
			name: "other error",
			err:   errors.New("some other error"),
			want:  false,
		},
		{
			name: "nil error",
			err:   nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isServiceUnavailable(tt.err)
			if got != tt.want {
				t.Errorf("isServiceUnavailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestFormatDuration tests formatDuration helper function
func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "microseconds",
			duration: 500 * time.Microsecond,
			want:     "500µs",
		},
		{
			name:     "milliseconds",
			duration: 50 * time.Millisecond,
			want:     "50.00ms",
		},
		{
			name:     "seconds",
			duration: 2 * time.Second,
			want:     "2.00s",
		},
		{
			name:     "fractional seconds",
			duration: 1500 * time.Millisecond,
			want:     "1.50s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.duration)
			if got != tt.want {
				t.Errorf("formatDuration() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestClassifyAlert_CacheInvalidationError tests cache invalidation error handling
func TestClassifyAlert_CacheInvalidationError(t *testing.T) {
	mockService := &MockClassificationService{
		invalidateCacheFunc: func(ctx context.Context, fingerprint string) error {
			return errors.New("cache invalidation failed")
		},
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.95,
				Reasoning:      "Test",
				ProcessingTime: 0.05,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: true,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	// Should still succeed even if cache invalidation fails (graceful degradation)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (graceful degradation), got %d: %s", rr.Code, rr.Body.String())
		return
	}

	// Verify cache invalidation was attempted
	if !mockService.invalidateCacheCalled {
		t.Error("Expected InvalidateCache to be called when force=true")
	}
}

// TestClassifyAlert_GetCachedError tests GetCachedClassification error handling
func TestClassifyAlert_GetCachedError(t *testing.T) {
	mockService := &MockClassificationService{
		getCachedFunc: func(ctx context.Context, fingerprint string) (*core.ClassificationResult, error) {
			return nil, errors.New("cache error")
		},
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityInfo,
				Confidence:     0.75,
				Reasoning:      "New classification after cache error",
				ProcessingTime: 0.2,
			}, nil
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: false,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	// Should fallback to classification when cache error occurs
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Cached {
		t.Error("Expected cached=false when cache error occurs, got cached=true")
	}
}

// TestClassifyAlert_MetadataWithoutModel tests metadata without model field
func TestClassifyAlert_MetadataWithoutModel(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.95,
				Reasoning:      "Test",
				ProcessingTime: 0.05,
				Metadata: map[string]interface{}{
					"other_field": "value",
				},
			}, nil
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Model != "" {
		t.Errorf("Expected empty model when metadata doesn't contain model, got '%s'", response.Model)
	}
}

// TestClassifyAlert_MetadataWithNonStringModel tests metadata with non-string model
func TestClassifyAlert_MetadataWithNonStringModel(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.95,
				Reasoning:      "Test",
				ProcessingTime: 0.05,
				Metadata: map[string]interface{}{
					"model": 123, // Non-string model
				},
			}, nil
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Model != "" {
		t.Errorf("Expected empty model when metadata contains non-string model, got '%s'", response.Model)
	}
}

// TestClassifyAlert_WithoutService tests behavior when ClassificationService is nil
func TestClassifyAlert_WithoutService(t *testing.T) {
	mockClassifier := &MockClassifier{
		classifyFunc: func(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
			return &core.ClassificationResult{
				Severity:       core.SeverityWarning,
				Confidence:     0.95,
				Reasoning:      "Test",
				ProcessingTime: 0.05,
			}, nil
		},
	}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
			StartsAt:    time.Now(),
		},
		Force: true, // Force flag should be ignored when service is nil
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
		return
	}

	var response ClassifyResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.Cached {
		t.Error("Expected cached=false when service is nil, got cached=true")
	}
}

// TestClassifyAlert_ResolvedStatus tests resolved status validation
func TestClassifyAlert_ResolvedStatus(t *testing.T) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusResolved,
			StartsAt:    time.Now(),
		},
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = req.WithContext(context.WithValue(req.Context(), middleware.RequestIDContextKey, "test-request-id"))

	rr := httptest.NewRecorder()
	handlers.ClassifyAlert(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 for resolved status, got %d: %s", rr.Code, rr.Body.String())
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func TestGetClassificationStats_Success(t *testing.T) {
	// Create mock with stats
	mockService := &MockClassificationService{
		stats: services.ClassificationStats{
			TotalRequests:   100,
			CacheHitRate:    0.65,
			LLMSuccessRate:  0.98,
			FallbackRate:    0.02,
			AvgResponseTime: 50 * time.Millisecond,
		},
	}
	handlers := NewClassificationHandlersWithService(mockService, mockService, slog.Default())

	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "test-request-id"))
	rr := httptest.NewRecorder()

	handlers.GetClassificationStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var response StatsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response structure
	if response.BySeverity == nil {
		t.Error("Expected by_severity map, got nil")
	}

	if response.TotalRequests != 100 {
		t.Errorf("Expected TotalRequests 100, got %d", response.TotalRequests)
	}

	if response.CacheStats.HitRate != 0.65 {
		t.Errorf("Expected CacheStats.HitRate 0.65, got %f", response.CacheStats.HitRate)
	}

	if response.LLMStats.SuccessRate != 0.98 {
		t.Errorf("Expected LLMStats.SuccessRate 0.98, got %f", response.LLMStats.SuccessRate)
	}

	if response.FallbackStats.Rate != 0.02 {
		t.Errorf("Expected FallbackStats.Rate 0.02, got %f", response.FallbackStats.Rate)
	}

	// Validate severity stats are initialized
	expectedSeverities := []string{"critical", "warning", "info", "noise"}
	for _, severity := range expectedSeverities {
		if _, exists := response.BySeverity[severity]; !exists {
			t.Errorf("Expected severity %s in BySeverity map", severity)
		}
	}
}

func TestGetClassificationStats_WithoutService(t *testing.T) {
	// Test graceful degradation when ClassificationService is not available
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	req := httptest.NewRequest("GET", "/api/v2/classification/stats", nil)
	req = req.WithContext(context.WithValue(req.Context(), "request_id", "test-request-id"))
	rr := httptest.NewRecorder()

	handlers.GetClassificationStats(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 (graceful degradation), got %d: %s", rr.Code, rr.Body.String())
	}

	var response StatsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should return empty stats, not error
	if response.TotalRequests != 0 {
		t.Errorf("Expected TotalRequests 0 (empty stats), got %d", response.TotalRequests)
	}

	if response.BySeverity == nil {
		t.Error("Expected by_severity map (even if empty), got nil")
	}
}

func TestListClassificationModels_Success(t *testing.T) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	req := httptest.NewRequest("GET", "/api/v2/classification/models", nil)
	rr := httptest.NewRecorder()

	handlers.ListClassificationModels(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}

	var response ModelsResponse
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(response.Models) == 0 {
		t.Error("Expected models list, got empty")
	}

	if response.Active == "" {
		t.Error("Expected active model, got empty")
	}
}

// Benchmark ClassifyAlert
func BenchmarkClassifyAlert(b *testing.B) {
	mockClassifier := &MockClassifier{}
	handlers := NewClassificationHandlers(mockClassifier, slog.Default())

	reqBody := ClassifyRequest{
		Alert: &core.Alert{
			Fingerprint: "test-123",
			AlertName:   "TestAlert",
			Status:      core.StatusFiring,
		},
	}

	body, _ := json.Marshal(reqBody)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/api/v2/classification/classify", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handlers.ClassifyAlert(rr, req)
	}
}
