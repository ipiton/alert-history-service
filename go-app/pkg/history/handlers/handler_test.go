package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vitaliisemenov/alert-history/internal/core"
	"github.com/vitaliisemenov/alert-history/pkg/history/cache"
	"github.com/vitaliisemenov/alert-history/pkg/history/filters"
)

var (
	// Test errors
	errDatabaseConnection = errors.New("database connection failed")
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
	if m.err != nil {
		return nil, m.err
	}
	return []*core.Alert{}, nil
}

func (m *MockRepository) GetAggregatedStats(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &core.AggregatedStats{}, nil
}

func (m *MockRepository) GetTopAlerts(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []*core.TopAlert{}, nil
}

func (m *MockRepository) GetFlappingAlerts(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []*core.FlappingAlert{}, nil
}

// createTestCacheManager creates a fresh cache manager for tests
func createTestCacheManager() *cache.Manager {
	cacheConfig := cache.DefaultConfig()
	cacheConfig.L1Enabled = true
	cacheConfig.L2Enabled = false
	cm, _ := cache.NewManager(cacheConfig, nil)
	return cm
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
	handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

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
	handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

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
	handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

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
	handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

	req := httptest.NewRequest("GET", "/api/v2/history/stats", nil)
	w := httptest.NewRecorder()

	handler.GetStats(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GetStats() status = %v, want %v", w.Code, http.StatusOK)
	}
}

// TestHandler_GetHistory_Comprehensive - Comprehensive table-driven tests
func TestHandler_GetHistory_Comprehensive(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		mockHistory    *core.HistoryResponse
		mockErr        error
		wantStatus     int
		wantTotalInResp bool
	}{
		{
			name: "happy path - page 1",
			url:  "/api/v2/history?page=1&per_page=50",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   100,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "happy path - with status filter",
			url:  "/api/v2/history?status=firing",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   50,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "happy path - with severity filter",
			url:  "/api/v2/history?severity=critical",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   25,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "happy path - with namespace filter",
			url:  "/api/v2/history?namespace=production",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   75,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "happy path - multiple filters",
			url:  "/api/v2/history?status=firing&severity=critical&namespace=production",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   10,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "happy path - custom per_page",
			url:  "/api/v2/history?per_page=100",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   200,
				Page:    1,
				PerPage: 100,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "per_page exceeds max (1000) - should cap",
			url:  "/api/v2/history?per_page=5000",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   0,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "invalid page (0) - should default to 1",
			url:  "/api/v2/history?page=0",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   0,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "invalid page (negative) - should default to 1",
			url:  "/api/v2/history?page=-1",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   0,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "invalid per_page (0) - should default to 50",
			url:  "/api/v2/history?per_page=0",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   0,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "invalid per_page (negative) - should default to 50",
			url:  "/api/v2/history?per_page=-1",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   0,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "empty results",
			url:  "/api/v2/history",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   0,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name:            "database error - should return 500",
			url:             "/api/v2/history?test=db_error", // unique URL to avoid cache hit
			mockHistory:     nil,
			mockErr:         errDatabaseConnection,
			wantStatus:      http.StatusInternalServerError,
			wantTotalInResp: false,
		},
		{
			name: "with time range filter",
			url:  "/api/v2/history?from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   150,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
		{
			name: "with sort_field and sort_order",
			url:  "/api/v2/history?sort_field=created_at&sort_order=asc",
			mockHistory: &core.HistoryResponse{
				Alerts:  []*core.Alert{},
				Total:   50,
				Page:    1,
				PerPage: 50,
			},
			wantStatus:      http.StatusOK,
			wantTotalInResp: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{
				history: tt.mockHistory,
				err:     tt.mockErr,
			}
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			handler.GetHistory(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}

			if tt.wantTotalInResp && w.Code == http.StatusOK {
				var response core.HistoryResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("%s: Failed to decode response: %v", tt.name, err)
				}
			}
		})
	}
}

// TestHandler_GetRecentAlerts_Comprehensive - Comprehensive tests for GetRecentAlerts
func TestHandler_GetRecentAlerts_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "default limit",
			url:        "/api/v2/history/recent",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit - 10",
			url:        "/api/v2/history/recent?limit=10",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit - 50",
			url:        "/api/v2/history/recent?limit=50",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit - 100",
			url:        "/api/v2/history/recent?limit=100",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid limit (0) - should use default",
			url:        "/api/v2/history/recent?limit=0",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid limit (negative) - should use default",
			url:        "/api/v2/history/recent?limit=-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "very large limit - should cap",
			url:        "/api/v2/history/recent?limit=10000",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{}
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			handler.GetRecentAlerts(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandler_GetStats_Comprehensive - Comprehensive tests for GetStats
func TestHandler_GetStats_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "no time range",
			url:        "/api/v2/history/stats",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with time range",
			url:        "/api/v2/history/stats?from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with from only",
			url:        "/api/v2/history/stats?from=2024-01-01T00:00:00Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with to only",
			url:        "/api/v2/history/stats?to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid time format - should ignore",
			url:        "/api/v2/history/stats?from=invalid",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{}
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			handler.GetStats(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandler_GetAlertTimeline_Comprehensive - Comprehensive tests for GetAlertTimeline
// NOTE: These tests are skipped because GetAlertTimeline uses gorilla/mux for path parameters
// which requires full router setup. Coverage for this handler is achieved through integration tests.
func TestHandler_GetAlertTimeline_Comprehensive(t *testing.T) {
	t.Skip("Skipping GetAlertTimeline tests - requires gorilla/mux router setup")
}

// TestHandler_SearchAlerts_Comprehensive - Comprehensive tests for SearchAlerts
func TestHandler_SearchAlerts_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "valid search - simple query",
			body:       `{"query":"test"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid search - with filters",
			body:       `{"query":"test","filters":{"status":"firing"}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid search - with pagination",
			body:       `{"query":"test","pagination":{"page":1,"per_page":20}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid search - complex query",
			body:       `{"query":"critical alerts","filters":{"severity":"critical","namespace":"production"},"pagination":{"page":2,"per_page":50}}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "empty query",
			body:       `{"query":""}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "missing query field",
			body:       `{"filters":{"status":"firing"}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid JSON",
			body:       `{invalid json`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty body",
			body:       ``,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid pagination - page 0",
			body:       `{"query":"test","pagination":{"page":0,"per_page":50}}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "invalid pagination - negative per_page",
			body:       `{"query":"test","pagination":{"page":1,"per_page":-10}}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{}
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			req := httptest.NewRequest("POST", "/api/v2/history/search", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler.SearchAlerts(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandler_GetTopAlerts_Comprehensive - Comprehensive tests for GetTopAlerts
func TestHandler_GetTopAlerts_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "default limit (10)",
			url:        "/api/v2/history/top",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit - 20",
			url:        "/api/v2/history/top?limit=20",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom limit - 100 (max)",
			url:        "/api/v2/history/top?limit=100",
			wantStatus: http.StatusOK,
		},
		{
			name:       "limit exceeds max - should cap",
			url:        "/api/v2/history/top?limit=500",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with time range",
			url:        "/api/v2/history/top?from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with from only",
			url:        "/api/v2/history/top?from=2024-01-01T00:00:00Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with to only",
			url:        "/api/v2/history/top?to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid time format - should ignore",
			url:        "/api/v2/history/top?from=invalid",
			wantStatus: http.StatusOK,
		},
		{
			name:       "all params combined",
			url:        "/api/v2/history/top?limit=50&from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{}
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			handler.GetTopAlerts(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandler_GetFlappingAlerts_Comprehensive - Comprehensive tests for GetFlappingAlerts
func TestHandler_GetFlappingAlerts_Comprehensive(t *testing.T) {
	tests := []struct {
		name       string
		url        string
		wantStatus int
	}{
		{
			name:       "default threshold (5)",
			url:        "/api/v2/history/flapping",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom threshold - 10",
			url:        "/api/v2/history/flapping?threshold=10",
			wantStatus: http.StatusOK,
		},
		{
			name:       "custom threshold - 100 (max)",
			url:        "/api/v2/history/flapping?threshold=100",
			wantStatus: http.StatusOK,
		},
		{
			name:       "threshold exceeds max - should cap",
			url:        "/api/v2/history/flapping?threshold=500",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with time range",
			url:        "/api/v2/history/flapping?from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with from only",
			url:        "/api/v2/history/flapping?from=2024-01-01T00:00:00Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "with to only",
			url:        "/api/v2/history/flapping?to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid time format - should ignore",
			url:        "/api/v2/history/flapping?from=invalid",
			wantStatus: http.StatusOK,
		},
		{
			name:       "all params combined",
			url:        "/api/v2/history/flapping?threshold=15&from=2024-01-01T00:00:00Z&to=2024-12-31T23:59:59Z",
			wantStatus: http.StatusOK,
		},
		{
			name:       "zero threshold - should use default",
			url:        "/api/v2/history/flapping?threshold=0",
			wantStatus: http.StatusOK,
		},
		{
			name:       "negative threshold - should use default",
			url:        "/api/v2/history/flapping?threshold=-5",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &MockRepository{}
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			req := httptest.NewRequest("GET", tt.url, nil)
			w := httptest.NewRecorder()

			handler.GetFlappingAlerts(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}
		})
	}
}

// TestHandler_ErrorPaths - Test error handling for all handlers
func TestHandler_ErrorPaths(t *testing.T) {
	tests := []struct {
		name       string
		handler    func(http.ResponseWriter, *http.Request)
		url        string
		method     string
		body       string
		setupRepo  func() *MockRepository
		wantStatus int
	}{
		{
			name:   "GetRecentAlerts - repository error",
			url:    "/api/v2/history/recent",
			method: "GET",
			setupRepo: func() *MockRepository {
				return &MockRepository{err: errDatabaseConnection}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "GetTopAlerts - repository error",
			url:    "/api/v2/history/top",
			method: "GET",
			setupRepo: func() *MockRepository {
				return &MockRepository{err: errDatabaseConnection}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "GetFlappingAlerts - repository error",
			url:    "/api/v2/history/flapping",
			method: "GET",
			setupRepo: func() *MockRepository {
				return &MockRepository{err: errDatabaseConnection}
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name:   "GetStats - repository error",
			url:    "/api/v2/history/stats",
			method: "GET",
			setupRepo: func() *MockRepository {
				return &MockRepository{err: errDatabaseConnection}
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			filterRegistry := filters.NewRegistry(nil)
			handler := NewHandler(repo, filterRegistry, createTestCacheManager(), nil)

			var req *http.Request
			if tt.body != "" {
				req = httptest.NewRequest(tt.method, tt.url, strings.NewReader(tt.body))
			} else {
				req = httptest.NewRequest(tt.method, tt.url, nil)
			}
			w := httptest.NewRecorder()

			// Route to correct handler
			switch tt.url {
			case "/api/v2/history/recent":
				handler.GetRecentAlerts(w, req)
			case "/api/v2/history/top":
				handler.GetTopAlerts(w, req)
			case "/api/v2/history/flapping":
				handler.GetFlappingAlerts(w, req)
			case "/api/v2/history/stats":
				handler.GetStats(w, req)
			}

			if w.Code != tt.wantStatus {
				t.Errorf("%s: status = %v, want %v", tt.name, w.Code, tt.wantStatus)
			}
		})
	}
}
