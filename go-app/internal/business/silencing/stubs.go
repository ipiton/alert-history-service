package silencing

import (
	"context"
	"log/slog"
	"time"

	infrasilencing "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
)

// Temporary stubs for Phase 1 compilation
// These will be replaced in Phases 5 and 7

// syncWorker stub (will be implemented in Phase 5)
type syncWorker struct{}

func newSyncWorker(
	repo infrasilencing.SilenceRepository,
	cache *silenceCache,
	interval time.Duration,
	logger *slog.Logger,
	metrics *SilenceMetrics,
) *syncWorker {
	return &syncWorker{}
}

func (w *syncWorker) Start(ctx context.Context) {}
func (w *syncWorker) Stop()                     {}

// SilenceMetrics stub (will be implemented in Phase 7)
type SilenceMetrics struct{}

func NewSilenceMetrics() *SilenceMetrics {
	return &SilenceMetrics{}
}
