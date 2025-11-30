package metrics

import (
	"sync"
	"testing"
	"time"
)

var (
	testBusinessMetrics *BusinessMetrics
	testBusinessOnce    sync.Once
)

func getTestBusinessMetrics() *BusinessMetrics {
	testBusinessOnce.Do(func() {
		testBusinessMetrics = NewBusinessMetrics("alert_history_test")
	})
	return testBusinessMetrics
}

// TestBusinessMetrics_ClassificationCache tests cache hit recording
func TestBusinessMetrics_ClassificationCache(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("getTestBusinessMetrics() returned nil")
	}

	// Test L1 cache hit
	bm.RecordClassificationL1CacheHit()
	// No assertion - just verify it doesn't panic

	// Test L2 cache hit
	bm.RecordClassificationL2CacheHit()
	// No assertion - just verify it doesn't panic
}

// TestBusinessMetrics_ClassificationDuration tests duration recording
func TestBusinessMetrics_ClassificationDuration(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("NewBusinessMetrics() returned nil")
	}

	tests := []struct {
		name   string
		source string
		duration float64
	}{
		{
			name:   "LLM success",
			source: "llm",
			duration: 0.100, // 100ms in seconds
		},
		{
			name:   "cache hit",
			source: "cache",
			duration: 0.001, // 1ms in seconds
		},
		{
			name:   "fallback",
			source: "fallback",
			duration: 0.010, // 10ms in seconds
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bm.RecordClassificationDuration(tt.source, tt.duration)
			// No assertion - just verify it doesn't panic
		})
	}
}

// TestBusinessMetrics_Grouping tests grouping metrics
func TestBusinessMetrics_Grouping(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("NewBusinessMetrics() returned nil")
	}

	// Test IncActiveGroups
	bm.IncActiveGroups()

	// Test DecActiveGroups
	bm.DecActiveGroups()

	// Test RecordGroupSize
	tests := []struct {
		name string
		size int
	}{
		{name: "small group", size: 5},
		{name: "medium group", size: 50},
		{name: "large group", size: 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bm.RecordGroupSize(tt.size)
			// No assertion - just verify it doesn't panic
		})
	}
}

// TestBusinessMetrics_GroupOperations tests group operation metrics
func TestBusinessMetrics_GroupOperations(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("NewBusinessMetrics() returned nil")
	}

	operations := []struct {
		operation string
		status    string
	}{
		{"create", "success"},
		{"update", "success"},
		{"merge", "success"},
		{"flush", "success"},
		{"create", "error"},
		{"update", "error"},
	}

	for _, op := range operations {
		t.Run(op.operation+"_"+op.status, func(t *testing.T) {
			// Record operation
			bm.RecordGroupOperation(op.operation, op.status)

			// Record duration
			duration := 10 * time.Millisecond
			bm.RecordGroupOperationDuration(op.operation, duration)

			// No assertion - just verify they don't panic
		})
	}
}

// TestBusinessMetrics_Timers tests timer metrics
func TestBusinessMetrics_Timers(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("NewBusinessMetrics() returned nil")
	}

	timerTypes := []string{
		"group_wait",
		"group_interval",
		"repeat_interval",
	}

	for _, timerType := range timerTypes {
		t.Run(timerType, func(t *testing.T) {
			// Record started
			bm.RecordTimerStarted(timerType)

			// Record expired
			bm.RecordTimerExpired(timerType)

			// Record cancelled
			bm.RecordTimerCancelled(timerType)

			// No assertion - just verify they don't panic
		})
	}
}

// TestBusinessMetrics_Comprehensive tests all methods together
func TestBusinessMetrics_Comprehensive(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("NewBusinessMetrics() returned nil")
	}

	// Classification metrics
	bm.RecordClassificationL1CacheHit()
	bm.RecordClassificationL2CacheHit()
	bm.RecordClassificationDuration("llm", 0.100) // 100ms in seconds

	// Grouping metrics
	bm.IncActiveGroups()
	bm.RecordGroupSize(25)
	bm.RecordGroupOperation("create", "success")
	bm.RecordGroupOperationDuration("create", 5*time.Millisecond)
	bm.DecActiveGroups()

	// Timer metrics
	bm.RecordTimerStarted("group_wait")
	bm.RecordTimerExpired("group_wait")
	bm.RecordTimerCancelled("group_interval")

	// If we reach here without panic, all methods work
}

// TestBusinessMetrics_TimerExtended tests extended timer methods
func TestBusinessMetrics_TimerExtended(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("getTestBusinessMetrics() returned nil")
	}

	// Timer reset
	bm.RecordTimerReset("group_wait")

	// Timer duration
	bm.RecordTimerDuration("group_wait", 5500*time.Millisecond) // 5.5 seconds

	// Active timers counters
	bm.IncActiveTimers("group_wait")
	bm.DecActiveTimers("group_wait")

	// Timers restored/missed
	bm.RecordTimersRestored(10)
	bm.RecordTimersMissed(2)
}

// TestBusinessMetrics_Storage tests storage fallback/recovery
func TestBusinessMetrics_Storage(t *testing.T) {
	bm := getTestBusinessMetrics()
	if bm == nil {
		t.Fatal("getTestBusinessMetrics() returned nil")
	}

	// Storage fallback
	bm.IncStorageFallback("health_check_failed")
	bm.IncStorageFallback("store_error")

	// Storage recovery
	bm.IncStorageRecovery()

	// Groups restored
	bm.RecordGroupsRestored(25)
}
