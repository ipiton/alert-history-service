package cache

import (
	"context"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// TestManager_InvalidatePattern tests InvalidatePattern method
func TestManager_InvalidatePattern(t *testing.T) {
	config := DefaultConfig()
	// Disable L2 (Redis) since we're testing without Redis
	config.L2Enabled = false
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()

	// InvalidatePattern should not error even when L2 is disabled
	err = manager.InvalidatePattern(ctx, "test:*")
	if err != nil {
		t.Errorf("InvalidatePattern() unexpected error: %v", err)
	}
}

// TestManager_UpdateMetrics tests UpdateMetrics method
func TestManager_UpdateMetrics(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Set some test data
	ctx := context.Background()
	testData := &core.HistoryResponse{
		Alerts: []*core.Alert{},
		Total:  10,
	}

	for i := 0; i < 3; i++ {
		key := "metric_test_" + string(rune('a'+i))
		if err := manager.Set(ctx, key, testData); err != nil {
			t.Errorf("Failed to set key %s: %v", key, err)
		}
	}

	// Call UpdateMetrics - should not panic
	manager.UpdateMetrics()

	// Verify stats still work after UpdateMetrics
	stats := manager.Stats()
	if stats == nil {
		t.Error("Stats() returned nil after UpdateMetrics()")
	}
}

// TestManager_Lifecycle tests complete manager lifecycle
func TestManager_Lifecycle(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ctx := context.Background()
	testData := &core.HistoryResponse{
		Alerts: []*core.Alert{},
		Total:  5,
	}

	// Phase 1: Set operations
	for i := 0; i < 5; i++ {
		key := "lifecycle_" + string(rune('a'+i))
		if err := manager.Set(ctx, key, testData); err != nil {
			t.Errorf("Set failed: %v", err)
		}
	}

	// Phase 2: Get operations
	for i := 0; i < 5; i++ {
		key := "lifecycle_" + string(rune('a'+i))
		_, found := manager.Get(ctx, key)
		if !found {
			t.Errorf("Get failed for key %s", key)
		}
	}

	// Phase 3: Update metrics
	manager.UpdateMetrics()

	// Phase 4: Stats
	stats := manager.Stats()
	if stats == nil {
		t.Error("Stats() returned nil")
	}

	// Phase 5: Invalidate specific keys
	err = manager.Invalidate(ctx, "lifecycle_a")
	if err != nil {
		t.Errorf("Invalidate() failed: %v", err)
	}

	// Phase 6: Invalidate pattern (no-op for L1 only)
	err = manager.InvalidatePattern(ctx, "lifecycle_*")
	if err != nil {
		t.Errorf("InvalidatePattern() failed: %v", err)
	}

	// Phase 7: Close
	if err := manager.Close(); err != nil {
		t.Errorf("Close() failed: %v", err)
	}
}

// TestManager_ConcurrentUpdateMetrics tests concurrent UpdateMetrics calls
func TestManager_ConcurrentUpdateMetrics(t *testing.T) {
	config := DefaultConfig()
	manager, err := NewManager(config, nil)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Call UpdateMetrics concurrently
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			manager.UpdateMetrics()
			done <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// If we reach here without panic, test passes
}

// TestConfig_Validate tests Config.Validate() method
func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "default config is valid",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: &Config{
				L1Enabled:   true,
				L1MaxEntries: 1000,
				L1MaxSizeMB: 50,
				L1TTL:       5 * time.Minute,
				L2Enabled:   false,
			},
			wantErr: false,
		},
		{
			name: "zero L1MaxEntries - should error",
			config: &Config{
				L1Enabled:   true,
				L1MaxEntries: 0,
				L1MaxSizeMB: 100,
				L1TTL:       5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero L1MaxSizeMB - should error",
			config: &Config{
				L1Enabled:   true,
				L1MaxEntries: 10000,
				L1MaxSizeMB: 0,
				L1TTL:       5 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "zero L1TTL - should error",
			config: &Config{
				L1Enabled:   true,
				L1MaxEntries: 10000,
				L1MaxSizeMB: 100,
				L1TTL:       0,
			},
			wantErr: true,
		},
		{
			name: "L2 enabled without Redis addr",
			config: &Config{
				L1Enabled:   true,
				L2Enabled:   true,
				RedisAddr:   "",
			},
			wantErr: true,
		},
		{
			name: "L2 enabled with Redis addr - complete config",
			config: &Config{
				L1Enabled:   true,
				L1MaxEntries: 10000,
				L1MaxSizeMB: 100,
				L1TTL:       5 * time.Minute,
				L2Enabled:   true,
				RedisAddr:   "localhost:6379",
				RedisDB:     0,
				RedisPoolSize: 10,
				L2TTL:       10 * time.Minute,
			},
			wantErr: false,
		},
		{
			name: "both L1 and L2 disabled",
			config: &Config{
				L1Enabled: false,
				L2Enabled: false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("Validate() expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Validate() unexpected error: %v", err)
			}
		})
	}
}
