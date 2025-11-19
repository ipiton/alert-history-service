package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// ============================================================================
// TN-63: Unit Tests for History Endpoint
// ============================================================================

// Test 1-5: Request Parsing Tests for History
func TestParseHistoryRequest_ValidBasic(t *testing.T) {
	handler := &HistoryHandlerV2{}
	req := httptest.NewRequest("GET", "/history?page=1&per_page=50", nil)

	result, err := handler.parseHistoryRequest(req.URL.Query())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.Pagination.Page != 1 {
		t.Errorf("Expected Page=1, got %d", result.Pagination.Page)
	}
	if result.Pagination.PerPage != 50 {
		t.Errorf("Expected PerPage=50, got %d", result.Pagination.PerPage)
	}
}

func TestParseHistoryRequest_WithAllFilters(t *testing.T) {
	handler := &HistoryHandlerV2{}
	req := httptest.NewRequest("GET", "/history?status=firing&severity=critical&namespace=prod&search=cpu&from=2024-01-01T00:00:00Z&to=2024-01-02T00:00:00Z&sort_field=created_at&sort_order=desc", nil)

	result, err := handler.parseHistoryRequest(req.URL.Query())
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if *result.Filters.Status != core.StatusFiring {
		t.Error("Expected Status=firing")
	}
	if *result.Filters.Severity != "critical" {
		t.Error("Expected Severity=critical")
	}
	if *result.Filters.Namespace != "prod" {
		t.Error("Expected Namespace=prod")
	}
	if result.Sorting.Field != "created_at" {
		t.Error("Expected SortField=created_at")
	}
	if result.Sorting.Order != core.SortOrderDesc {
		t.Error("Expected SortOrder=desc")
	}
}

func TestParseHistoryRequest_InvalidPagination(t *testing.T) {
	handler := &HistoryHandlerV2{}
	tests := []struct {
		name  string
		query string
	}{
		{"page too low", "page=0"},
		{"per_page too low", "per_page=0"},
		{"per_page too high", "per_page=1001"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/history?"+tt.query, nil)
			_, err := handler.parseHistoryRequest(req.URL.Query())
			if err == nil {
				t.Error("Expected error, got nil")
			}
		})
	}
}

func TestParseHistoryRequest_InvalidSorting(t *testing.T) {
	handler := &HistoryHandlerV2{}
	req := httptest.NewRequest("GET", "/history?sort_field=invalid", nil)

	_, err := handler.parseHistoryRequest(req.URL.Query())
	if err == nil {
		t.Error("Expected error for invalid sort field, got nil")
	}
}

func TestParseHistoryRequest_InvalidFilters(t *testing.T) {
	handler := &HistoryHandlerV2{}
	req := httptest.NewRequest("GET", "/history?status=invalid", nil)

	_, err := handler.parseHistoryRequest(req.URL.Query())
	if err == nil {
		t.Error("Expected error for invalid status, got nil")
	}
}

// Test 6-10: Handler Logic Tests
func TestHandleHistory_Success(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetHistoryFunc: func(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
			return &core.HistoryResponse{
				Alerts: []*core.Alert{
					{Fingerprint: "fp1", AlertName: "TestAlert", Status: core.StatusFiring},
				},
				Total: 1,
				Page:  1,
			}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history", nil)
	w := httptest.NewRecorder()

	handler.HandleHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp core.HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(resp.Alerts))
	}
}

func TestHandleHistory_InvalidMethod(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)
	req := httptest.NewRequest("POST", "/history", nil)
	w := httptest.NewRecorder()

	handler.HandleHistory(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleHistory_RepositoryError(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetHistoryFunc: func(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
			return nil, errors.New("db error")
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history", nil)
	w := httptest.NewRecorder()

	handler.HandleHistory(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleHistory_EmptyResults(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetHistoryFunc: func(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
			return &core.HistoryResponse{
				Alerts: []*core.Alert{},
				Total:  0,
			}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history", nil)
	w := httptest.NewRecorder()

	handler.HandleHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp core.HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(resp.Alerts) != 0 {
		t.Error("Expected empty alerts list")
	}
}

func TestHandleHistory_WithPagination(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetHistoryFunc: func(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
			if req.Pagination.Page != 2 {
				t.Errorf("Expected Page=2, got %d", req.Pagination.Page)
			}
			return &core.HistoryResponse{Alerts: []*core.Alert{}}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history?page=2", nil)
	w := httptest.NewRecorder()

	handler.HandleHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// Test 11-15: Supporting Handlers Tests
func TestHandleRecentAlerts_Success(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetRecentAlertsFunc: func(ctx context.Context, limit int) ([]*core.Alert, error) {
			return []*core.Alert{{Fingerprint: "fp1"}}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/recent", nil)
	w := httptest.NewRecorder()

	handler.HandleRecentAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleStats_Success(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetAggregatedStatsFunc: func(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
			return &core.AggregatedStats{TotalAlerts: 100}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/stats", nil)
	w := httptest.NewRecorder()

	handler.HandleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleTopAlerts_Success(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetTopAlertsFunc: func(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
			return []*core.TopAlert{{Fingerprint: "fp1"}}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/top", nil)
	w := httptest.NewRecorder()

	handler.HandleTopAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleFlappingAlerts_Success(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetFlappingAlertsFunc: func(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
			return []*core.FlappingAlert{{Fingerprint: "fp1"}}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/flapping", nil)
	w := httptest.NewRecorder()

	handler.HandleFlappingAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleHistory_TimeoutContext(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetHistoryFunc: func(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
			time.Sleep(100 * time.Millisecond) // Simulate slow DB
			return nil, context.DeadlineExceeded
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	// Create request with very short timeout
	req := httptest.NewRequest("GET", "/history", nil)
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Millisecond)
	defer cancel()
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()

	handler.HandleHistory(w, req)

	// Should return 503 or 500 depending on implementation
	// Checking that it doesn't panic and returns error status
	if w.Code != http.StatusServiceUnavailable && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected error status for timeout, got %d", w.Code)
	}
}

func TestHandleRecentAlerts_Error(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetRecentAlertsFunc: func(ctx context.Context, limit int) ([]*core.Alert, error) {
			return nil, errors.New("db error")
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/recent", nil)
	w := httptest.NewRecorder()

	handler.HandleRecentAlerts(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleStats_Error(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetAggregatedStatsFunc: func(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
			return nil, errors.New("db error")
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/stats", nil)
	w := httptest.NewRecorder()

	handler.HandleStats(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleTopAlerts_Error(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetTopAlertsFunc: func(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
			return nil, errors.New("db error")
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/top", nil)
	w := httptest.NewRecorder()

	handler.HandleTopAlerts(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleFlappingAlerts_Error(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetFlappingAlertsFunc: func(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
			return nil, errors.New("db error")
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/flapping", nil)
	w := httptest.NewRecorder()

	handler.HandleFlappingAlerts(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}

func TestHandleRecentAlerts_InvalidMethod(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)
	req := httptest.NewRequest("POST", "/history/recent", nil)
	w := httptest.NewRecorder()

	handler.HandleRecentAlerts(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleRecentAlerts_WithLimit(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetRecentAlertsFunc: func(ctx context.Context, limit int) ([]*core.Alert, error) {
			if limit != 5 {
				t.Errorf("Expected limit=5, got %d", limit)
			}
			return []*core.Alert{}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/recent?limit=5", nil)
	w := httptest.NewRecorder()

	handler.HandleRecentAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleStats_InvalidMethod(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)
	req := httptest.NewRequest("POST", "/history/stats", nil)
	w := httptest.NewRecorder()

	handler.HandleStats(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleStats_WithParams(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetAggregatedStatsFunc: func(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
			if timeRange == nil {
				t.Error("Expected timeRange to be set")
			}
			return &core.AggregatedStats{TotalAlerts: 50}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/stats?from=2024-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()

	handler.HandleStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleTopAlerts_InvalidMethod(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)
	req := httptest.NewRequest("POST", "/history/top", nil)
	w := httptest.NewRecorder()

	handler.HandleTopAlerts(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleTopAlerts_WithParams(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetTopAlertsFunc: func(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
			if limit != 5 {
				t.Errorf("Expected limit=5, got %d", limit)
			}
			if timeRange == nil {
				t.Error("Expected timeRange to be set")
			}
			return []*core.TopAlert{}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/top?limit=5&from=2024-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()

	handler.HandleTopAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHandleFlappingAlerts_InvalidMethod(t *testing.T) {
	handler := NewHistoryHandlerV2(&MockHistoryRepository{}, nil)
	req := httptest.NewRequest("POST", "/history/flapping", nil)
	w := httptest.NewRecorder()

	handler.HandleFlappingAlerts(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

func TestHandleFlappingAlerts_WithParams(t *testing.T) {
	mockRepo := &MockHistoryRepository{
		GetFlappingAlertsFunc: func(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
			if threshold != 5 {
				t.Errorf("Expected threshold=5, got %d", threshold)
			}
			if timeRange == nil {
				t.Error("Expected timeRange to be set")
			}
			return []*core.FlappingAlert{}, nil
		},
	}
	handler := NewHistoryHandlerV2(mockRepo, nil)

	req := httptest.NewRequest("GET", "/history/flapping?threshold=5&from=2024-01-01T00:00:00Z", nil)
	w := httptest.NewRecorder()

	handler.HandleFlappingAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
