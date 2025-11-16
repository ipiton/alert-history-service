package publishing

import (
	"context"
	"log/slog"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// Stub implementations for TN-062 integration until full TN-047/TN-058 are enabled

// StubTargetDiscoveryManager is a stub implementation of TargetDiscoveryManager
// Used temporarily until K8s integration (TN-047) is fully enabled
type StubTargetDiscoveryManager struct {
	logger *slog.Logger
}

// NewStubTargetDiscoveryManager creates a stub target discovery manager
func NewStubTargetDiscoveryManager(logger *slog.Logger) *StubTargetDiscoveryManager {
	return &StubTargetDiscoveryManager{
		logger: logger,
	}
}

func (s *StubTargetDiscoveryManager) DiscoverTargets(ctx context.Context) error {
	// No-op: returns no targets
	return nil
}

func (s *StubTargetDiscoveryManager) GetTarget(name string) (*core.PublishingTarget, error) {
	// No targets available in stub
	return nil, nil
}

func (s *StubTargetDiscoveryManager) ListTargets() []*core.PublishingTarget {
	// Return empty list (no targets configured yet)
	return []*core.PublishingTarget{}
}

func (s *StubTargetDiscoveryManager) GetTargetsByType(targetType string) []*core.PublishingTarget {
	// Return empty list
	return []*core.PublishingTarget{}
}

func (s *StubTargetDiscoveryManager) GetTargetCount() int {
	// No targets in stub
	return 0
}

// StubParallelPublisher is a stub implementation of ParallelPublisher
// Used temporarily until TN-058 is fully integrated
type StubParallelPublisher struct {
	logger *slog.Logger
}

// NewStubParallelPublisher creates a stub parallel publisher
func NewStubParallelPublisher(logger *slog.Logger) *StubParallelPublisher {
	return &StubParallelPublisher{
		logger: logger,
	}
}

func (s *StubParallelPublisher) PublishToMultiple(
	ctx context.Context,
	alert *core.EnrichedAlert,
	targets []*core.PublishingTarget,
) (*ParallelPublishResult, error) {
	// Stub implementation: simulate successful publish to 0 targets
	s.logger.Debug("Stub parallel publisher: no targets to publish",
		"fingerprint", alert.Alert.Fingerprint,
		"targets_count", len(targets))

	return &ParallelPublishResult{
		SuccessCount: 0,
		FailureCount: 0,
		TotalTargets: len(targets),
		Results:      []TargetPublishResult{},
		Duration:     1 * time.Millisecond, // Minimal duration
	}, nil
}

func (s *StubParallelPublisher) PublishToAll(
	ctx context.Context,
	alert *core.EnrichedAlert,
) (*ParallelPublishResult, error) {
	// Stub: no targets available
	return &ParallelPublishResult{
		SuccessCount: 0,
		FailureCount: 0,
		TotalTargets: 0,
		Results:      []TargetPublishResult{},
		Duration:     1 * time.Millisecond,
	}, nil
}

func (s *StubParallelPublisher) PublishToHealthy(
	ctx context.Context,
	alert *core.EnrichedAlert,
) (*ParallelPublishResult, error) {
	// Stub: no targets available
	return &ParallelPublishResult{
		SuccessCount: 0,
		FailureCount: 0,
		TotalTargets: 0,
		Results:      []TargetPublishResult{},
		Duration:     1 * time.Millisecond,
	}, nil
}

