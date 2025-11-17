package publishing

import (
	"context"
	"fmt"
	"log/slog"

	infrapublishing "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

// ModeService provides business logic for publishing mode operations.
//
// This service orchestrates mode detection, caching, and metrics aggregation
// by delegating to underlying infrastructure components (ModeManager, DiscoveryManager).
//
// It supports both enhanced mode detection (when ModeManager is available) and
// fallback mode detection (basic logic) for backward compatibility.
type ModeService interface {
	// GetCurrentModeInfo returns current publishing mode information.
	//
	// This method returns comprehensive mode information including basic fields
	// (mode, targets_available, enabled_targets, metrics_only_active) and enhanced
	// fields (transition_count, current_mode_duration, last_transition_time, etc.)
	// when ModeManager is available.
	//
	// The method supports two code paths:
	// 1. Enhanced path (with ModeManager): Returns full metrics from TN-060
	// 2. Fallback path (without ModeManager): Returns basic mode detection
	//
	// Parameters:
	//   - ctx: Context for cancellation and tracing
	//
	// Returns:
	//   - *ModeInfo: Current mode information
	//   - error: Error if mode detection failed
	//
	// Example:
	//   modeInfo, err := service.GetCurrentModeInfo(ctx)
	//   if err != nil {
	//       // Handle error
	//   }
	//   fmt.Printf("Current mode: %s\n", modeInfo.Mode)
	GetCurrentModeInfo(ctx context.Context) (*ModeInfo, error)
}

// DefaultModeService implements ModeService interface.
//
// This is the production implementation that delegates to ModeManager (TN-060)
// and TargetDiscoveryManager (TN-047) for mode detection and target enumeration.
//
// The service is stateless and thread-safe, suitable for concurrent use.
type DefaultModeService struct {
	// modeManager provides enhanced mode detection and metrics (TN-060).
	// May be nil for systems that haven't integrated TN-060 yet (fallback path).
	modeManager infrapublishing.ModeManager

	// discoveryManager provides target enumeration and health information (TN-047).
	// Required for both enhanced and fallback paths.
	discoveryManager infrapublishing.TargetDiscoveryManager

	// logger for structured logging
	logger *slog.Logger
}

// NewModeService creates a new mode service with the given dependencies.
//
// The modeManager parameter may be nil to support fallback mode detection.
// The discoveryManager parameter is required and must not be nil.
//
// Parameters:
//   - modeManager: ModeManager for enhanced mode detection (may be nil)
//   - discoveryManager: TargetDiscoveryManager for target enumeration (required)
//   - logger: Logger for structured logging (defaults to slog.Default if nil)
//
// Returns:
//   - ModeService: New mode service instance
//
// Example:
//   service := NewModeService(modeManager, discoveryManager, logger)
func NewModeService(
	modeManager infrapublishing.ModeManager,
	discoveryManager infrapublishing.TargetDiscoveryManager,
	logger *slog.Logger,
) ModeService {
	if logger == nil {
		logger = slog.Default()
	}

	return &DefaultModeService{
		modeManager:      modeManager,
		discoveryManager: discoveryManager,
		logger:           logger,
	}
}

// GetCurrentModeInfo returns current mode information using either enhanced
// (ModeManager) or fallback (basic) mode detection.
//
// This method implements the dual-path strategy for graceful degradation:
// 1. If ModeManager available: Use enhanced path with full metrics
// 2. If ModeManager nil: Use fallback path with basic detection
//
// The method is thread-safe and can be called concurrently.
func (s *DefaultModeService) GetCurrentModeInfo(ctx context.Context) (*ModeInfo, error) {
	// Validate context
	if ctx == nil {
		return nil, fmt.Errorf("context cannot be nil")
	}

	// Validate required dependencies
	if s.discoveryManager == nil {
		return nil, fmt.Errorf("discoveryManager is required but was nil")
	}

	// Use ModeManager if available (TN-060 integration)
	if s.modeManager != nil {
		s.logger.Debug("Using enhanced mode detection (ModeManager available)")
		return s.getModeInfoFromManager(ctx)
	}

	// Fallback to basic mode detection (backward compatibility)
	s.logger.Debug("Using fallback mode detection (ModeManager not available)")
	return s.getModeInfoFallback(ctx)
}

// getModeInfoFromManager gets mode info from ModeManager (enhanced path).
//
// This method uses TN-060 ModeManager to get comprehensive mode information
// including transition metrics, duration, and history.
//
// Performance characteristics (from TN-060 benchmarks):
// - GetCurrentMode(): ~34 ns/op (0 allocs)
// - GetModeMetrics(): ~173 ns/op (1 alloc)
//
// The method is thread-safe due to ModeManager's internal locking.
func (s *DefaultModeService) getModeInfoFromManager(ctx context.Context) (*ModeInfo, error) {
	// Get current mode and metrics from ModeManager (TN-060)
	currentMode := s.modeManager.GetCurrentMode()
	modeMetrics := s.modeManager.GetModeMetrics()

	// Count enabled targets from DiscoveryManager (TN-047)
	targets := s.discoveryManager.ListTargets()
	enabledCount := 0
	for _, t := range targets {
		if t.Enabled {
			enabledCount++
		}
	}
	targetsAvailable := enabledCount > 0

	// Log mode information
	s.logger.Debug("Retrieved mode info from ModeManager",
		"mode", currentMode.String(),
		"enabled_targets", enabledCount,
		"transition_count", modeMetrics.TransitionCount,
		"mode_duration_seconds", modeMetrics.CurrentModeDuration.Seconds())

	// Build response with enhanced metrics
	modeInfo := &ModeInfo{
		// Basic fields
		Mode:              currentMode.String(),
		TargetsAvailable:  targetsAvailable,
		EnabledTargets:    enabledCount,
		MetricsOnlyActive: currentMode == infrapublishing.ModeMetricsOnly,

		// Enhanced fields (TN-060)
		TransitionCount:            modeMetrics.TransitionCount,
		CurrentModeDurationSeconds: modeMetrics.CurrentModeDuration.Seconds(),
		LastTransitionTime:         modeMetrics.LastTransitionTime,
		LastTransitionReason:       modeMetrics.LastTransitionReason,
	}

	return modeInfo, nil
}

// getModeInfoFallback gets mode info using basic detection (fallback path).
//
// This method provides backward compatibility for systems that haven't
// integrated TN-060 ModeManager yet. It performs simple mode detection
// based on enabled target count:
// - enabled_targets > 0 → "normal" mode
// - enabled_targets == 0 → "metrics-only" mode
//
// Enhanced fields (transition_count, etc.) are omitted in this path.
//
// The method is thread-safe if DiscoveryManager.ListTargets() is thread-safe.
func (s *DefaultModeService) getModeInfoFallback(ctx context.Context) (*ModeInfo, error) {
	// Count enabled targets from DiscoveryManager (TN-047)
	targets := s.discoveryManager.ListTargets()
	enabledCount := 0
	for _, t := range targets {
		if t.Enabled {
			enabledCount++
		}
	}
	targetsAvailable := enabledCount > 0

	// Determine mode based on target availability
	mode := "normal"
	metricsOnly := false
	if !targetsAvailable {
		mode = "metrics-only"
		metricsOnly = true
	}

	// Log mode information
	s.logger.Debug("Retrieved mode info using fallback detection",
		"mode", mode,
		"enabled_targets", enabledCount)

	// Build response (basic fields only, enhanced fields omitted)
	modeInfo := &ModeInfo{
		Mode:              mode,
		TargetsAvailable:  targetsAvailable,
		EnabledTargets:    enabledCount,
		MetricsOnlyActive: metricsOnly,
		// Enhanced fields omitted (TransitionCount, etc.)
	}

	return modeInfo, nil
}
