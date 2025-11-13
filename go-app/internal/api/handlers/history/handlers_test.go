package history

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTopAlerts_Success(t *testing.T) {
	handlers := NewHistoryHandlers(slog.Default())

	tests := []struct {
		name       string
		queryParam string
		expectCode int
	}{
		{
			name:       "default parameters",
			queryParam: "",
			expectCode: http.StatusOK,
		},
		{
			name:       "with period parameter",
			queryParam: "?period=7d",
			expectCode: http.StatusOK,
		},
		{
			name:       "with limit parameter",
			queryParam: "?limit=20",
			expectCode: http.StatusOK,
		},
		{
			name:       "with both parameters",
			queryParam: "?period=24h&limit=10",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v2/history/top"+tt.queryParam, nil)
			rr := httptest.NewRecorder()

			handlers.GetTopAlerts(rr, req)

			if rr.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, rr.Code)
			}

			var response TopAlertsResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Alerts == nil {
				t.Error("Expected alerts array, got nil")
			}
		})
	}
}

func TestGetFlappingAlerts_Success(t *testing.T) {
	handlers := NewHistoryHandlers(slog.Default())

	tests := []struct {
		name       string
		queryParam string
		expectCode int
	}{
		{
			name:       "default parameters",
			queryParam: "",
			expectCode: http.StatusOK,
		},
		{
			name:       "with threshold parameter",
			queryParam: "?threshold=10",
			expectCode: http.StatusOK,
		},
		{
			name:       "with all parameters",
			queryParam: "?period=7d&threshold=5&limit=15",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v2/history/flapping"+tt.queryParam, nil)
			rr := httptest.NewRecorder()

			handlers.GetFlappingAlerts(rr, req)

			if rr.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, rr.Code)
			}

			var response FlappingAlertsResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Alerts == nil {
				t.Error("Expected alerts array, got nil")
			}
		})
	}
}

func TestGetRecentAlerts_Success(t *testing.T) {
	handlers := NewHistoryHandlers(slog.Default())

	tests := []struct {
		name       string
		queryParam string
		expectCode int
	}{
		{
			name:       "default parameters",
			queryParam: "",
			expectCode: http.StatusOK,
		},
		{
			name:       "with pagination",
			queryParam: "?limit=100&offset=50",
			expectCode: http.StatusOK,
		},
		{
			name:       "with filters",
			queryParam: "?status=firing&severity=critical",
			expectCode: http.StatusOK,
		},
		{
			name:       "with all parameters",
			queryParam: "?limit=25&offset=0&status=resolved&severity=warning",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v2/history/recent"+tt.queryParam, nil)
			rr := httptest.NewRecorder()

			handlers.GetRecentAlerts(rr, req)

			if rr.Code != tt.expectCode {
				t.Errorf("Expected status %d, got %d", tt.expectCode, rr.Code)
			}

			var response RecentAlertsResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Alerts == nil {
				t.Error("Expected alerts array, got nil")
			}
		})
	}
}

func TestGetRecentAlerts_PaginationLimits(t *testing.T) {
	handlers := NewHistoryHandlers(slog.Default())

	tests := []struct {
		name        string
		limit       string
		expectLimit int
	}{
		{
			name:        "default limit",
			limit:       "",
			expectLimit: 50,
		},
		{
			name:        "custom limit within range",
			limit:       "25",
			expectLimit: 25,
		},
		{
			name:        "limit exceeds max (capped at 1000)",
			limit:       "5000",
			expectLimit: 50, // Falls back to default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			queryParam := ""
			if tt.limit != "" {
				queryParam = "?limit=" + tt.limit
			}

			req := httptest.NewRequest("GET", "/api/v2/history/recent"+queryParam, nil)
			rr := httptest.NewRecorder()

			handlers.GetRecentAlerts(rr, req)

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", rr.Code)
			}

			var response RecentAlertsResponse
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.Limit != tt.expectLimit {
				t.Errorf("Expected limit %d, got %d", tt.expectLimit, response.Limit)
			}
		})
	}
}

// Benchmark GetTopAlerts
func BenchmarkGetTopAlerts(b *testing.B) {
	handlers := NewHistoryHandlers(slog.Default())
	req := httptest.NewRequest("GET", "/api/v2/history/top", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetTopAlerts(rr, req)
	}
}

// Benchmark GetRecentAlerts
func BenchmarkGetRecentAlerts(b *testing.B) {
	handlers := NewHistoryHandlers(slog.Default())
	req := httptest.NewRequest("GET", "/api/v2/history/recent", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handlers.GetRecentAlerts(rr, req)
	}
}
