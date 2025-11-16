package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/history/cache"
	"github.com/vitaliisemenov/alert-history/pkg/history/filters"
)

// MockRepository is a mock implementation of AlertHistoryRepository
type MockRepository struct {
	history *core.HistoryResponse
	err     error
}

func (m *MockRepository) GetHistory(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.history != nil {
		return m.history, nil
	}
	return &core.HistoryResponse{
		Alerts: []*core.Alert{},
		Total:  0,
		Page:  req.Pagination.Page,
		PerPage: req.Pagination.PerPage,
	}, nil
}

func (m *MockRepository) GetAlertsByFingerprint(ctx context.Context, fingerprint string, limit int) ([]*core.Alert, error) {
	return []*core.Alert{}, nil
}

func (m *MockRepository) GetRecentAlerts(ctx context.Context, limit int) ([]*core.Alert, error) {
	return []*core.Alert{}, nil
}

func (m *MockRepository) GetAggregatedStats(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
	return &core.AggregatedStats{}, nil
}

func (m *MockRepository) GetTopAlerts(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
	return []*core.TopAlert{}, nil
}

func (m *MockRepository) GetFlappingAlerts(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
	return []*core.FlappingAlert{}, nil
}

// testCacheManager is a shared cache manager for tests (to avoid Prometheus registration issues)
var testCacheManager *cache.Manager

func init() {
	cacheConfig := cache.DefaultConfig()
	cacheConfig.L1Enabled = true
	cacheConfig.L2Enabled = false
	testCacheManager, _ = cache.NewManager(cacheConfig, nil)
}

// TestHandler_GetHistory tests GetHistory handler
func TestHandler_GetHistory(t *testing.T) {
	repo := &MockRepository{
		history: &core.HistoryResponse{
			Alerts: []*core.Alert{},
			Total:  10,
			Page:  1,
			PerPage: 50,
		},
	}

	filterRegistry := filters.NewRegistry(nil)
	handler := NewHandler(repo, filterRegistry, testCacheManager, nil)

	req := httptest.NewRequest("GET", "/api/v2/history?page=1&per_page=50", nil)
	w := httptest.NewRecorder()

	handler.GetHistory(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetHistory() status = %v, want %v", w.Code, http.StatusOK)
	}

	var response core.HistoryResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if response.Total != 10 {
		t.Errorf("GetHistory() Total = %v, want 10", response.Total)
	}
}

// TestHandler_GetHistory_InvalidParams tests invalid query parameters
func TestHandler_GetHistory_InvalidParams(t *testing.T) {
	repo := &MockRepository{}
	filterRegistry := filters.NewRegistry(nil)
	handler := NewHandler(repo, filterRegistry, testCacheManager, nil)

	req := httptest.NewRequest("GET", "/api/v2/history?page=0", nil)
	w := httptest.NewRecorder()

	handler.GetHistory(w, req)

	// Should handle invalid params gracefully (defaults to page=1)
	if w.Code != http.StatusOK {
		t.Errorf("GetHistory() status = %v, want %v", w.Code, http.StatusOK)
	}
}

// TestHandler_GetRecentAlerts tests GetRecentAlerts handler
func TestHandler_GetRecentAlerts(t *testing.T) {
	repo := &MockRepository{}
	filterRegistry := filters.NewRegistry(nil)
	handler := NewHandler(repo, filterRegistry, testCacheManager, nil)

	req := httptest.NewRequest("GET", "/api/v2/history/recent?limit=10", nil)
	w := httptest.NewRecorder()

	handler.GetRecentAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetRecentAlerts() status = %v, want %v", w.Code, http.StatusOK)
	}
}

// TestHandler_GetStats tests GetStats handler
func TestHandler_GetStats(t *testing.T) {
	repo := &MockRepository{}
	filterRegistry := filters.NewRegistry(nil)
	handler := NewHandler(repo, filterRegistry, testCacheManager, nil)

	req := httptest.NewRequest("GET", "/api/v2/history/stats", nil)
	w := httptest.NewRecorder()

	handler.GetStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetStats() status = %v, want %v", w.Code, http.StatusOK)
	}
}
