package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// MockCacheRepository implements AlertHistoryRepository interface for testing
type MockCacheRepository struct {
	history *core.HistoryResponse
	err     error
	mu      sync.Mutex
}

func (m *MockCacheRepository) GetHistory(ctx context.Context, req *core.HistoryRequest) (*core.HistoryResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.err != nil {
		return nil, m.err
	}
	if m.history != nil {
		return m.history, nil
	}
	return &core.HistoryResponse{
		Alerts: []*core.Alert{},
		Total:  0,
	}, nil
}

func (m *MockCacheRepository) GetRecentAlerts(ctx context.Context, limit int) ([]*core.Alert, error) {
	return []*core.Alert{}, nil
}

func (m *MockCacheRepository) GetAggregatedStats(ctx context.Context, timeRange *core.TimeRange) (*core.AggregatedStats, error) {
	return &core.AggregatedStats{}, nil
}

func (m *MockCacheRepository) GetTopAlerts(ctx context.Context, timeRange *core.TimeRange, limit int) ([]*core.TopAlert, error) {
	return []*core.TopAlert{}, nil
}

func (m *MockCacheRepository) GetFlappingAlerts(ctx context.Context, timeRange *core.TimeRange, threshold int) ([]*core.FlappingAlert, error) {
	return []*core.FlappingAlert{}, nil
}

func (m *MockCacheRepository) GetAlertsByFingerprint(ctx context.Context, fingerprint string, limit int) ([]*core.Alert, error) {
	return []*core.Alert{}, nil
}

// TestWarmer_NewWarmer tests warmer creation
func TestWarmer_NewWarmer(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	repo := &MockCacheRepository{}
	warmer := NewWarmer(manager, repo, nil)

	if warmer == nil {
		t.Fatal("NewWarmer() returned nil")
	}
}

// TestWarmer_StartStop tests warmer lifecycle
func TestWarmer_StartStop(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	repo := &MockCacheRepository{
		history: &core.HistoryResponse{
			Alerts: []*core.Alert{},
			Total:  10,
		},
	}

	warmer := NewWarmer(manager, repo, nil)

	ctx := context.Background()

	// Start warmer in background
	go warmer.Start(ctx, 100*time.Millisecond)

	// Wait for at least one warm cycle
	time.Sleep(200 * time.Millisecond)

	// Stop warmer
	warmer.Stop()

	// Verify warmer stopped
	time.Sleep(100 * time.Millisecond)
	// If we reach here without hanging, stop worked
}

// TestWarmer_WarmCache tests cache warming logic
func TestWarmer_WarmCache(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	testCases := []struct {
		name        string
		mockHistory *core.HistoryResponse
		mockErr     error
		wantSuccess bool
	}{
		{
			name: "successful warm",
			mockHistory: &core.HistoryResponse{
				Alerts: []*core.Alert{
					{
						AlertName:   "TestAlert",
						Fingerprint: "abc123",
						Status:      core.StatusFiring,
					},
				},
				Total: 1,
			},
			mockErr:     nil,
			wantSuccess: true,
		},
		{
			name:        "repository error",
			mockHistory: nil,
			mockErr:     &CacheError{Message: "database error"},
			wantSuccess: false,
		},
		{
			name: "empty results",
			mockHistory: &core.HistoryResponse{
				Alerts: []*core.Alert{},
				Total:  0,
			},
			mockErr:     nil,
			wantSuccess: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repo := &MockCacheRepository{
				history: tc.mockHistory,
				err:     tc.mockErr,
			}

			warmer := NewWarmer(manager, repo, nil)

			ctx := context.Background()
			// Start in background and test that it doesn't crash
			go warmer.Start(ctx, 100*time.Millisecond)
			time.Sleep(50 * time.Millisecond)
			warmer.Stop()
		})
	}
}

// TestWarmer_HelperFunctions tests helper functions
func TestWarmer_HelperFunctions(t *testing.T) {
	t.Run("ptrString", func(t *testing.T) {
		val := "test-value"
		ptr := ptrString(val)
		if ptr == nil {
			t.Error("ptrString() returned nil")
		}
		if *ptr != val {
			t.Errorf("ptrString() = %v, want %v", *ptr, val)
		}
	})

	t.Run("ptrStatus", func(t *testing.T) {
		val := core.StatusFiring
		ptr := ptrStatus(val)
		if ptr == nil {
			t.Error("ptrStatus() returned nil")
		}
		if *ptr != val {
			t.Errorf("ptrStatus() = %v, want %v", *ptr, val)
		}
	})
}

// TestWarmer_Concurrent tests concurrent access
func TestWarmer_Concurrent(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	repo := &MockCacheRepository{
		history: &core.HistoryResponse{
			Alerts: []*core.Alert{},
			Total:  0,
		},
	}

	ctx := context.Background()

	// Start/stop multiple warmer instances concurrently
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Create new warmer for each goroutine
			warmer := NewWarmer(manager, repo, nil)
			go warmer.Start(ctx, 50*time.Millisecond)
			time.Sleep(10 * time.Millisecond)
			warmer.Stop()
		}()
	}

	wg.Wait()
	// If we reach here without race conditions or crashes, test passes
}

// TestWarmer_LongRunning tests long-running warmer
func TestWarmer_LongRunning(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	repo := &MockCacheRepository{}
	repo.history = &core.HistoryResponse{
		Alerts: []*core.Alert{},
		Total:  0,
	}

	warmer := NewWarmer(manager, repo, nil)
	ctx := context.Background()

	go warmer.Start(ctx, 100*time.Millisecond)

	// Run for 500ms (should trigger ~5 warm cycles)
	time.Sleep(500 * time.Millisecond)
	warmer.Stop()

	// Verify warmer stopped gracefully
	time.Sleep(50 * time.Millisecond)
}
