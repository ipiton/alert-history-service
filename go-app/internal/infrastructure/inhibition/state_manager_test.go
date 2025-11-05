package inhibition

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// InhibitionState Tests
// ============================================================================

func TestInhibitionState_Serialization(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(1 * time.Hour)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "NodeDown_inhibits_InstanceDown",
		InhibitedAt:       now,
		ExpiresAt:         &expiresAt,
	}

	assert.Equal(t, "target123", state.TargetFingerprint)
	assert.Equal(t, "source456", state.SourceFingerprint)
	assert.Equal(t, "NodeDown_inhibits_InstanceDown", state.RuleName)
	assert.Equal(t, now, state.InhibitedAt)
	assert.NotNil(t, state.ExpiresAt)
}

// ============================================================================
// DefaultStateManager Tests
// ============================================================================

func TestNewDefaultStateManager(t *testing.T) {
	sm := NewDefaultStateManager(nil, nil)
	require.NotNil(t, sm)
	assert.NotNil(t, sm.logger)
	assert.Equal(t, "inhibition:state:", sm.redisPrefix)
	assert.Equal(t, 24*time.Hour, sm.redisTTL)
}

func TestDefaultStateManager_RecordInhibition_Success(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}

	err := sm.RecordInhibition(ctx, state)
	assert.NoError(t, err)

	// Verify it was stored
	stored, err := sm.GetInhibitionState(ctx, "target123")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "target123", stored.TargetFingerprint)
	assert.Equal(t, "source456", stored.SourceFingerprint)
	assert.Equal(t, "test_rule", stored.RuleName)
}

func TestDefaultStateManager_RecordInhibition_NilState(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	err := sm.RecordInhibition(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "state cannot be nil")
}

func TestDefaultStateManager_RecordInhibition_EmptyTargetFingerprint(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}

	err := sm.RecordInhibition(ctx, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target fingerprint cannot be empty")
}

func TestDefaultStateManager_RecordInhibition_EmptySourceFingerprint(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}

	err := sm.RecordInhibition(ctx, state)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "source fingerprint cannot be empty")
}

func TestDefaultStateManager_RemoveInhibition_Success(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add a state
	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}
	err := sm.RecordInhibition(ctx, state)
	require.NoError(t, err)

	// Remove it
	err = sm.RemoveInhibition(ctx, "target123")
	assert.NoError(t, err)

	// Verify it's gone
	stored, err := sm.GetInhibitionState(ctx, "target123")
	require.NoError(t, err)
	assert.Nil(t, stored)
}

func TestDefaultStateManager_RemoveInhibition_EmptyFingerprint(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	err := sm.RemoveInhibition(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "target fingerprint cannot be empty")
}

func TestDefaultStateManager_RemoveInhibition_NotExist(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Removing non-existent state should not error
	err := sm.RemoveInhibition(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestDefaultStateManager_GetActiveInhibitions_Empty(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	states, err := sm.GetActiveInhibitions(ctx)
	require.NoError(t, err)
	assert.Empty(t, states)
}

func TestDefaultStateManager_GetActiveInhibitions_Multiple(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add multiple states
	for i := 0; i < 3; i++ {
		state := &InhibitionState{
			TargetFingerprint: string(rune('A' + i)),
			SourceFingerprint: "source",
			RuleName:          "test_rule",
			InhibitedAt:       time.Now(),
		}
		err := sm.RecordInhibition(ctx, state)
		require.NoError(t, err)
	}

	states, err := sm.GetActiveInhibitions(ctx)
	require.NoError(t, err)
	assert.Len(t, states, 3)
}

func TestDefaultStateManager_GetActiveInhibitions_FiltersExpired(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add an expired state
	past := time.Now().Add(-1 * time.Hour)
	expiredState := &InhibitionState{
		TargetFingerprint: "expired",
		SourceFingerprint: "source",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now().Add(-2 * time.Hour),
		ExpiresAt:         &past,
	}
	err := sm.RecordInhibition(ctx, expiredState)
	require.NoError(t, err)

	// Add an active state
	activeState := &InhibitionState{
		TargetFingerprint: "active",
		SourceFingerprint: "source",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}
	err = sm.RecordInhibition(ctx, activeState)
	require.NoError(t, err)

	// Get active inhibitions - should only return active one
	states, err := sm.GetActiveInhibitions(ctx)
	require.NoError(t, err)
	assert.Len(t, states, 1)
	assert.Equal(t, "active", states[0].TargetFingerprint)
}

func TestDefaultStateManager_GetInhibitedAlerts_Empty(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	fingerprints, err := sm.GetInhibitedAlerts(ctx)
	require.NoError(t, err)
	assert.Empty(t, fingerprints)
}

func TestDefaultStateManager_GetInhibitedAlerts_Multiple(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add multiple states
	for i := 0; i < 3; i++ {
		state := &InhibitionState{
			TargetFingerprint: string(rune('A' + i)),
			SourceFingerprint: "source",
			RuleName:          "test_rule",
			InhibitedAt:       time.Now(),
		}
		err := sm.RecordInhibition(ctx, state)
		require.NoError(t, err)
	}

	fingerprints, err := sm.GetInhibitedAlerts(ctx)
	require.NoError(t, err)
	assert.Len(t, fingerprints, 3)
}

func TestDefaultStateManager_IsInhibited_True(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}
	err := sm.RecordInhibition(ctx, state)
	require.NoError(t, err)

	inhibited, err := sm.IsInhibited(ctx, "target123")
	require.NoError(t, err)
	assert.True(t, inhibited)
}

func TestDefaultStateManager_IsInhibited_False(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	inhibited, err := sm.IsInhibited(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, inhibited)
}

func TestDefaultStateManager_IsInhibited_EmptyFingerprint(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	inhibited, err := sm.IsInhibited(ctx, "")
	assert.Error(t, err)
	assert.False(t, inhibited)
	assert.Contains(t, err.Error(), "target fingerprint cannot be empty")
}

func TestDefaultStateManager_IsInhibited_Expired(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add an expired state
	past := time.Now().Add(-1 * time.Hour)
	state := &InhibitionState{
		TargetFingerprint: "expired",
		SourceFingerprint: "source",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now().Add(-2 * time.Hour),
		ExpiresAt:         &past,
	}
	err := sm.RecordInhibition(ctx, state)
	require.NoError(t, err)

	// Should return false for expired state
	inhibited, err := sm.IsInhibited(ctx, "expired")
	require.NoError(t, err)
	assert.False(t, inhibited)

	// Expired state should be cleaned up
	stored, err := sm.GetInhibitionState(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, stored)
}

func TestDefaultStateManager_GetInhibitionState_Found(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}
	err := sm.RecordInhibition(ctx, state)
	require.NoError(t, err)

	stored, err := sm.GetInhibitionState(ctx, "target123")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "target123", stored.TargetFingerprint)
	assert.Equal(t, "source456", stored.SourceFingerprint)
	assert.Equal(t, "test_rule", stored.RuleName)
}

func TestDefaultStateManager_GetInhibitionState_NotFound(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	stored, err := sm.GetInhibitionState(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, stored)
}

func TestDefaultStateManager_GetInhibitionState_EmptyFingerprint(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	stored, err := sm.GetInhibitionState(ctx, "")
	assert.Error(t, err)
	assert.Nil(t, stored)
	assert.Contains(t, err.Error(), "target fingerprint cannot be empty")
}

func TestDefaultStateManager_GetInhibitionState_Expired(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add an expired state
	past := time.Now().Add(-1 * time.Hour)
	state := &InhibitionState{
		TargetFingerprint: "expired",
		SourceFingerprint: "source",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now().Add(-2 * time.Hour),
		ExpiresAt:         &past,
	}
	err := sm.RecordInhibition(ctx, state)
	require.NoError(t, err)

	// Should return nil for expired state
	stored, err := sm.GetInhibitionState(ctx, "expired")
	require.NoError(t, err)
	assert.Nil(t, stored)
}

func TestDefaultStateManager_Concurrency(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			state := &InhibitionState{
				TargetFingerprint: string(rune('A' + id)),
				SourceFingerprint: "source",
				RuleName:          "test_rule",
				InhibitedAt:       time.Now(),
			}
			err := sm.RecordInhibition(ctx, state)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify all were stored
	states, err := sm.GetActiveInhibitions(ctx)
	require.NoError(t, err)
	assert.Len(t, states, 10)
}

func TestDefaultStateManager_UpdateInhibition(t *testing.T) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add initial state
	state1 := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "rule1",
		InhibitedAt:       time.Now(),
	}
	err := sm.RecordInhibition(ctx, state1)
	require.NoError(t, err)

	// Update with new state (same target, different source)
	state2 := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source789",
		RuleName:          "rule2",
		InhibitedAt:       time.Now(),
	}
	err = sm.RecordInhibition(ctx, state2)
	require.NoError(t, err)

	// Verify update
	stored, err := sm.GetInhibitionState(ctx, "target123")
	require.NoError(t, err)
	require.NotNil(t, stored)
	assert.Equal(t, "source789", stored.SourceFingerprint)
	assert.Equal(t, "rule2", stored.RuleName)
}

// ============================================================================
// Benchmarks
// ============================================================================

func BenchmarkDefaultStateManager_RecordInhibition(b *testing.B) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sm.RecordInhibition(ctx, state)
	}
}

func BenchmarkDefaultStateManager_IsInhibited(b *testing.B) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}
	_ = sm.RecordInhibition(ctx, state)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sm.IsInhibited(ctx, "target123")
	}
}

func BenchmarkDefaultStateManager_GetInhibitionState(b *testing.B) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	state := &InhibitionState{
		TargetFingerprint: "target123",
		SourceFingerprint: "source456",
		RuleName:          "test_rule",
		InhibitedAt:       time.Now(),
	}
	_ = sm.RecordInhibition(ctx, state)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sm.GetInhibitionState(ctx, "target123")
	}
}

func BenchmarkDefaultStateManager_GetActiveInhibitions(b *testing.B) {
	ctx := context.Background()
	sm := NewDefaultStateManager(nil, nil)

	// Add 100 states
	for i := 0; i < 100; i++ {
		state := &InhibitionState{
			TargetFingerprint: string(rune('A' + i%26)) + string(rune('A' + i/26)),
			SourceFingerprint: "source",
			RuleName:          "test_rule",
			InhibitedAt:       time.Now(),
		}
		_ = sm.RecordInhibition(ctx, state)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sm.GetActiveInhibitions(ctx)
	}
}
