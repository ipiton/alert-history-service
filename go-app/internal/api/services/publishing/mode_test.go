package publishing

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/core"
	infrapublishing "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// mockModeManager is a mock implementation of ModeManager for testing
type mockModeManager struct {
	currentMode infrapublishing.Mode
	modeMetrics infrapublishing.ModeMetrics
}

func (m *mockModeManager) GetCurrentMode() infrapublishing.Mode {
	return m.currentMode
}

func (m *mockModeManager) IsMetricsOnly() bool {
	return m.currentMode == infrapublishing.ModeMetricsOnly
}

func (m *mockModeManager) CheckModeTransition() (infrapublishing.Mode, bool, error) {
	return m.currentMode, false, nil
}

func (m *mockModeManager) OnTargetsChanged() error {
	return nil
}

func (m *mockModeManager) Subscribe(callback infrapublishing.ModeChangeCallback) infrapublishing.UnsubscribeFunc {
	return func() {}
}

func (m *mockModeManager) GetModeMetrics() infrapublishing.ModeMetrics {
	return m.modeMetrics
}

func (m *mockModeManager) Start(ctx context.Context) error {
	return nil
}

func (m *mockModeManager) Stop() error {
	return nil
}

// mockDiscoveryManager is a mock implementation of TargetDiscoveryManager for testing
type mockDiscoveryManager struct {
	targets []*core.PublishingTarget
	err     error
}

func (m *mockDiscoveryManager) DiscoverTargets(ctx context.Context) error {
	return m.err
}

func (m *mockDiscoveryManager) ListTargets() []*core.PublishingTarget {
	if m.err != nil {
		return nil
	}
	return m.targets
}

func (m *mockDiscoveryManager) GetTarget(name string) (*core.PublishingTarget, error) {
	return nil, errors.New("not implemented")
}

func (m *mockDiscoveryManager) GetTargetsByType(targetType string) []*core.PublishingTarget {
	result := make([]*core.PublishingTarget, 0)
	for _, t := range m.targets {
		if t.Type == targetType {
			result = append(result, t)
		}
	}
	return result
}

func (m *mockDiscoveryManager) GetTargetCount() int {
	return len(m.targets)
}

func TestDefaultModeService_GetCurrentModeInfo_WithModeManager(t *testing.T) {
	// Setup
	modeManager := &mockModeManager{
		currentMode: infrapublishing.ModeNormal,
		modeMetrics: infrapublishing.ModeMetrics{
			CurrentMode:          infrapublishing.ModeNormal,
			CurrentModeDuration:  time.Hour,
			TransitionCount:      12,
			LastTransitionTime:   time.Now(),
			LastTransitionReason: "targets_available",
		},
	}

	discoveryManager := &mockDiscoveryManager{
		targets: []*core.PublishingTarget{
			{Name: "target1", Enabled: true},
			{Name: "target2", Enabled: true},
			{Name: "target3", Enabled: false},
		},
	}

	service := NewModeService(modeManager, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, modeInfo)
	assert.Equal(t, "normal", modeInfo.Mode)
	assert.True(t, modeInfo.TargetsAvailable)
	assert.Equal(t, 2, modeInfo.EnabledTargets)
	assert.False(t, modeInfo.MetricsOnlyActive)
	assert.Equal(t, int64(12), modeInfo.TransitionCount)
	assert.InDelta(t, 3600.0, modeInfo.CurrentModeDurationSeconds, 1.0)
	assert.NotZero(t, modeInfo.LastTransitionTime)
	assert.Equal(t, "targets_available", modeInfo.LastTransitionReason)
}

func TestDefaultModeService_GetCurrentModeInfo_MetricsOnlyMode(t *testing.T) {
	// Setup
	modeManager := &mockModeManager{
		currentMode: infrapublishing.ModeMetricsOnly,
		modeMetrics: infrapublishing.ModeMetrics{
			CurrentMode:          infrapublishing.ModeMetricsOnly,
			CurrentModeDuration:  2 * time.Minute,
			TransitionCount:      13,
			LastTransitionTime:   time.Now(),
			LastTransitionReason: "no_enabled_targets",
		},
	}

	discoveryManager := &mockDiscoveryManager{
		targets: []*core.PublishingTarget{
			{Name: "target1", Enabled: false},
			{Name: "target2", Enabled: false},
		},
	}

	service := NewModeService(modeManager, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, modeInfo)
	assert.Equal(t, "metrics-only", modeInfo.Mode)
	assert.False(t, modeInfo.TargetsAvailable)
	assert.Equal(t, 0, modeInfo.EnabledTargets)
	assert.True(t, modeInfo.MetricsOnlyActive)
	assert.Equal(t, int64(13), modeInfo.TransitionCount)
}

func TestDefaultModeService_GetCurrentModeInfo_Fallback(t *testing.T) {
	// Setup: No ModeManager (nil)
	discoveryManager := &mockDiscoveryManager{
		targets: []*core.PublishingTarget{
			{Name: "target1", Enabled: true},
			{Name: "target2", Enabled: true},
		},
	}

	service := NewModeService(nil, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, modeInfo)
	assert.Equal(t, "normal", modeInfo.Mode)
	assert.True(t, modeInfo.TargetsAvailable)
	assert.Equal(t, 2, modeInfo.EnabledTargets)
	assert.False(t, modeInfo.MetricsOnlyActive)
	// Enhanced fields should be omitted (zero values)
	assert.Zero(t, modeInfo.TransitionCount)
	assert.Zero(t, modeInfo.CurrentModeDurationSeconds)
	assert.True(t, modeInfo.LastTransitionTime.IsZero())
	assert.Empty(t, modeInfo.LastTransitionReason)
}

func TestDefaultModeService_GetCurrentModeInfo_FallbackMetricsOnly(t *testing.T) {
	// Setup: No ModeManager, no enabled targets
	discoveryManager := &mockDiscoveryManager{
		targets: []*core.PublishingTarget{
			{Name: "target1", Enabled: false},
			{Name: "target2", Enabled: false},
		},
	}

	service := NewModeService(nil, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, modeInfo)
	assert.Equal(t, "metrics-only", modeInfo.Mode)
	assert.False(t, modeInfo.TargetsAvailable)
	assert.Equal(t, 0, modeInfo.EnabledTargets)
	assert.True(t, modeInfo.MetricsOnlyActive)
}

func TestDefaultModeService_GetCurrentModeInfo_ZeroTargets(t *testing.T) {
	// Setup
	modeManager := &mockModeManager{
		currentMode: infrapublishing.ModeMetricsOnly,
		modeMetrics: infrapublishing.ModeMetrics{
			CurrentMode: infrapublishing.ModeMetricsOnly,
		},
	}

	discoveryManager := &mockDiscoveryManager{
		targets: []*core.PublishingTarget{}, // Empty list
	}

	service := NewModeService(modeManager, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, modeInfo)
	assert.Equal(t, "metrics-only", modeInfo.Mode)
	assert.False(t, modeInfo.TargetsAvailable)
	assert.Equal(t, 0, modeInfo.EnabledTargets)
}

func TestDefaultModeService_GetCurrentModeInfo_ManyTargets(t *testing.T) {
	// Setup: Many targets
	modeManager := &mockModeManager{
		currentMode: infrapublishing.ModeNormal,
		modeMetrics: infrapublishing.ModeMetrics{
			CurrentMode: infrapublishing.ModeNormal,
		},
	}

	targets := make([]*core.PublishingTarget, 1000)
	for i := 0; i < 1000; i++ {
		targets[i] = &core.PublishingTarget{
			Name:    "target" + string(rune(i)),
			Enabled: i%2 == 0, // Half enabled
		}
	}

	discoveryManager := &mockDiscoveryManager{targets: targets}
	service := NewModeService(modeManager, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, modeInfo)
	assert.Equal(t, "normal", modeInfo.Mode)
	assert.True(t, modeInfo.TargetsAvailable)
	assert.Equal(t, 500, modeInfo.EnabledTargets) // Half of 1000
}

func TestDefaultModeService_GetCurrentModeInfo_NilContext(t *testing.T) {
	// Setup
	modeManager := &mockModeManager{currentMode: infrapublishing.ModeNormal}
	discoveryManager := &mockDiscoveryManager{targets: []*core.PublishingTarget{}}
	service := NewModeService(modeManager, discoveryManager, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(nil)

	// Assert
	require.Error(t, err)
	assert.Nil(t, modeInfo)
	assert.Contains(t, err.Error(), "context cannot be nil")
}

func TestDefaultModeService_GetCurrentModeInfo_NilDiscoveryManager(t *testing.T) {
	// Setup
	modeManager := &mockModeManager{currentMode: infrapublishing.ModeNormal}
	service := NewModeService(modeManager, nil, nil)

	// Execute
	modeInfo, err := service.GetCurrentModeInfo(context.Background())

	// Assert
	require.Error(t, err)
	assert.Nil(t, modeInfo)
	assert.Contains(t, err.Error(), "discoveryManager is required")
}

func TestNewModeService(t *testing.T) {
	modeManager := &mockModeManager{}
	discoveryManager := &mockDiscoveryManager{}

	// Test with logger
	service := NewModeService(modeManager, discoveryManager, nil)
	assert.NotNil(t, service)

	// Test with nil logger (should use default)
	service2 := NewModeService(modeManager, discoveryManager, nil)
	assert.NotNil(t, service2)
}
