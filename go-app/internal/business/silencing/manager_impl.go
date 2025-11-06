package silencing

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core/silencing"
	infrasilencing "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
)

// DefaultSilenceManager implements SilenceManager interface.
//
// Architecture:
//   - Storage layer: SilenceRepository (PostgreSQL)
//   - Matching layer: SilenceMatcher (regex-based)
//   - Cache layer: In-memory silenceCache (fast lookups)
//   - Worker layer: gcWorker, syncWorker (background tasks)
//
// Lifecycle:
//  1. Create with NewDefaultSilenceManager()
//  2. Call Start(ctx) to initialize
//  3. Use CRUD and filtering methods
//  4. Call Stop(ctx) for graceful shutdown
//
// Thread-safety:
//   - All public methods are safe for concurrent use
//   - Cache protected by RWMutex
//   - State flags protected by atomic.Bool
//   - Workers use WaitGroup for goroutine tracking
//
// Example:
//
//	repo := infrasilencing.NewPostgresSilenceRepository(pool, logger)
//	matcher := silencing.NewDefaultSilenceMatcher(logger)
//	manager := NewDefaultSilenceManager(repo, matcher, logger, nil)
//
//	if err := manager.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer manager.Stop(context.WithTimeout(context.Background(), 30*time.Second))
//
//	// Use manager...
type DefaultSilenceManager struct {
	// Storage & Matching dependencies
	repo    infrasilencing.SilenceRepository
	matcher silencing.SilenceMatcher

	// Cache layer
	cache *silenceCache

	// Background workers
	gcWorker   *gcWorker   // Garbage collection worker
	syncWorker *syncWorker // Cache synchronization worker

	// Observability
	metrics *SilenceMetrics
	logger  *slog.Logger

	// Configuration
	config SilenceManagerConfig

	// Lifecycle management
	started  atomic.Bool        // True if manager has been started
	shutdown atomic.Bool        // True if manager is shutting down
	wg       sync.WaitGroup     // Tracks background goroutines
	ctx      context.Context    // Manager context (cancelled on Stop)
	cancel   context.CancelFunc // Cancels manager context
}

// NewDefaultSilenceManager creates a new silence manager.
//
// This constructor initializes the manager structure but does NOT start background workers.
// Call manager.Start(ctx) after creation to begin operations.
//
// Parameters:
//   - repo: Silence repository (required, must not be nil)
//   - matcher: Silence matcher (required, must not be nil)
//   - logger: Structured logger (optional, defaults to slog.Default())
//   - config: Configuration (optional, defaults to DefaultSilenceManagerConfig())
//
// Returns:
//   - *DefaultSilenceManager: Initialized manager (not started)
//
// Panics:
//   - If repo is nil
//   - If matcher is nil
//
// Example:
//
//	// With default config
//	manager := NewDefaultSilenceManager(repo, matcher, logger, nil)
//
//	// With custom config
//	config := DefaultSilenceManagerConfig()
//	config.GCInterval = 10 * time.Minute
//	config.SyncInterval = 2 * time.Minute
//	manager := NewDefaultSilenceManager(repo, matcher, logger, &config)
func NewDefaultSilenceManager(
	repo infrasilencing.SilenceRepository,
	matcher silencing.SilenceMatcher,
	logger *slog.Logger,
	config *SilenceManagerConfig,
) *DefaultSilenceManager {
	// Validate required dependencies
	if repo == nil {
		panic("SilenceRepository cannot be nil")
	}
	if matcher == nil {
		panic("SilenceMatcher cannot be nil")
	}

	// Apply defaults
	if logger == nil {
		logger = slog.Default()
	}
	if config == nil {
		defaultCfg := DefaultSilenceManagerConfig()
		config = &defaultCfg
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		logger.Warn("Invalid configuration, using defaults", "error", err)
		defaultCfg := DefaultSilenceManagerConfig()
		config = &defaultCfg
	}

	// Create manager context
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize manager
	sm := &DefaultSilenceManager{
		repo:    repo,
		matcher: matcher,
		cache:   newSilenceCache(),
		metrics: NewSilenceMetrics(),
		logger:  logger,
		config:  *config,
		ctx:     ctx,
		cancel:  cancel,
	}

	// Initialize workers (not started yet)
	sm.gcWorker = newGCWorker(
		repo,
		sm.cache,
		config.GCInterval,
		config.GCRetention,
		config.GCBatchSize,
		logger,
		sm.metrics,
	)
	sm.syncWorker = newSyncWorker(
		repo,
		sm.cache,
		config.SyncInterval,
		logger,
		sm.metrics,
	)

	logger.Info("Silence manager created",
		"gc_interval", config.GCInterval,
		"gc_retention", config.GCRetention,
		"sync_interval", config.SyncInterval,
	)

	return sm
}

// Validate validates configuration parameters.
//
// Returns:
//   - error: Validation error, or nil if config is valid
//
// Validation rules:
//   - GCInterval >= 1m
//   - GCRetention >= 1h
//   - GCBatchSize > 0 and <= 10000
//   - SyncInterval >= 10s
//   - ShutdownTimeout >= 5s
func (c *SilenceManagerConfig) Validate() error {
	if c.GCInterval < 1*time.Minute {
		return fmt.Errorf("GCInterval too short: %v (minimum: 1m)", c.GCInterval)
	}
	if c.GCRetention < 1*time.Hour {
		return fmt.Errorf("GCRetention too short: %v (minimum: 1h)", c.GCRetention)
	}
	if c.GCBatchSize <= 0 || c.GCBatchSize > 10000 {
		return fmt.Errorf("GCBatchSize out of range: %d (valid: 1-10000)", c.GCBatchSize)
	}
	if c.SyncInterval < 10*time.Second {
		return fmt.Errorf("SyncInterval too short: %v (minimum: 10s)", c.SyncInterval)
	}
	if c.ShutdownTimeout < 5*time.Second {
		return fmt.Errorf("ShutdownTimeout too short: %v (minimum: 5s)", c.ShutdownTimeout)
	}
	return nil
}

// ==================== CRUD Operations Implementation ====================

// CreateSilence implements SilenceManager.CreateSilence.
func (sm *DefaultSilenceManager) CreateSilence(
	ctx context.Context,
	silence *silencing.Silence,
) (*silencing.Silence, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		// Metrics will be implemented in Phase 7
		sm.logger.Debug("CreateSilence completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return nil, ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return nil, ErrManagerShutdown
	}

	// Step 2: Delegate to repository (validation happens there)
	created, err := sm.repo.CreateSilence(ctx, silence)
	if err != nil {
		sm.logger.Error("Failed to create silence in repository",
			"error", err,
			"created_by", silence.CreatedBy,
		)
		return nil, fmt.Errorf("create silence: %w", err)
	}

	// Step 3: Add to cache if status is active
	if created.Status == silencing.SilenceStatusActive {
		sm.cache.Set(created)
		sm.logger.Debug("Added active silence to cache", "silence_id", created.ID)
	}

	sm.logger.Info("Silence created",
		"silence_id", created.ID,
		"created_by", created.CreatedBy,
		"status", created.Status,
		"starts_at", created.StartsAt.Format(time.RFC3339),
		"ends_at", created.EndsAt.Format(time.RFC3339),
	)

	return created, nil
}

// GetSilence implements SilenceManager.GetSilence.
func (sm *DefaultSilenceManager) GetSilence(
	ctx context.Context,
	id string,
) (*silencing.Silence, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		sm.logger.Debug("GetSilence completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return nil, ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return nil, ErrManagerShutdown
	}

	// Step 2: Try cache first (fast path)
	if silence, found := sm.cache.Get(id); found {
		sm.logger.Debug("Cache hit", "silence_id", id)
		return silence, nil
	}

	sm.logger.Debug("Cache miss, querying repository", "silence_id", id)

	// Step 3: Fallback to repository (slow path)
	silence, err := sm.repo.GetSilenceByID(ctx, id)
	if err != nil {
		// Don't log error for "not found" (expected case)
		if err != infrasilencing.ErrSilenceNotFound {
			sm.logger.Error("Failed to get silence from repository",
				"error", err,
				"silence_id", id,
			)
		}
		return nil, err
	}

	// Step 4: Update cache if active
	if silence.Status == silencing.SilenceStatusActive {
		sm.cache.Set(silence)
		sm.logger.Debug("Added silence to cache after retrieval", "silence_id", id)
	}

	return silence, nil
}

// UpdateSilence implements SilenceManager.UpdateSilence.
func (sm *DefaultSilenceManager) UpdateSilence(
	ctx context.Context,
	silence *silencing.Silence,
) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		sm.logger.Debug("UpdateSilence completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return ErrManagerShutdown
	}

	// Step 2: Update in repository (validation happens there)
	err := sm.repo.UpdateSilence(ctx, silence)
	if err != nil {
		sm.logger.Error("Failed to update silence in repository",
			"error", err,
			"silence_id", silence.ID,
		)
		return fmt.Errorf("update silence: %w", err)
	}

	// Step 3: Invalidate cache entry
	sm.cache.Delete(silence.ID)
	sm.logger.Debug("Invalidated cache entry", "silence_id", silence.ID)

	// Step 4: Re-add to cache if new status is active
	if silence.Status == silencing.SilenceStatusActive {
		sm.cache.Set(silence)
		sm.logger.Debug("Re-added updated silence to cache", "silence_id", silence.ID)
	}

	sm.logger.Info("Silence updated",
		"silence_id", silence.ID,
		"status", silence.Status,
	)

	return nil
}

// DeleteSilence implements SilenceManager.DeleteSilence.
func (sm *DefaultSilenceManager) DeleteSilence(
	ctx context.Context,
	id string,
) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		sm.logger.Debug("DeleteSilence completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return ErrManagerShutdown
	}

	// Step 2: Delete from repository
	err := sm.repo.DeleteSilence(ctx, id)
	if err != nil {
		sm.logger.Error("Failed to delete silence from repository",
			"error", err,
			"silence_id", id,
		)
		return fmt.Errorf("delete silence: %w", err)
	}

	// Step 3: Remove from cache
	sm.cache.Delete(id)
	sm.logger.Debug("Removed silence from cache", "silence_id", id)

	sm.logger.Info("Silence deleted", "silence_id", id)

	return nil
}

// ListSilences implements SilenceManager.ListSilences.
func (sm *DefaultSilenceManager) ListSilences(
	ctx context.Context,
	filter infrasilencing.SilenceFilter,
) ([]*silencing.Silence, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		sm.logger.Debug("ListSilences completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return nil, ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return nil, ErrManagerShutdown
	}

	// Step 2: Fast path - if filter is only "status=active" and no pagination, use cache
	if len(filter.Statuses) == 1 &&
		filter.Statuses[0] == silencing.SilenceStatusActive &&
		filter.Limit == 0 &&
		filter.Offset == 0 &&
		filter.CreatedBy == "" &&
		filter.MatcherName == "" &&
		filter.MatcherValue == "" &&
		filter.StartsAfter == nil &&
		filter.StartsBefore == nil &&
		filter.EndsAfter == nil &&
		filter.EndsBefore == nil {
		silences := sm.cache.GetByStatus(silencing.SilenceStatusActive)
		sm.logger.Debug("List silences (cache hit)", "count", len(silences))
		return silences, nil
	}

	sm.logger.Debug("List silences (cache miss, querying repository)")

	// Step 3: Slow path - complex filters, query repository
	silences, err := sm.repo.ListSilences(ctx, filter)
	if err != nil {
		sm.logger.Error("Failed to list silences from repository",
			"error", err,
			"filter_statuses", filter.Statuses,
		)
		return nil, fmt.Errorf("list silences: %w", err)
	}

	sm.logger.Debug("List silences completed", "count", len(silences))

	return silences, nil
}

// ==================== Alert Filtering Integration ====================

// GetActiveSilences implements SilenceManager.GetActiveSilences.
func (sm *DefaultSilenceManager) GetActiveSilences(
	ctx context.Context,
) ([]*silencing.Silence, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		sm.logger.Debug("GetActiveSilences completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return nil, ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return nil, ErrManagerShutdown
	}

	// Step 2: Fast path - return from cache
	silences := sm.cache.GetByStatus(silencing.SilenceStatusActive)

	// Step 3: Fallback - query repository if cache is empty
	if len(silences) == 0 {
		sm.logger.Debug("Cache empty, querying repository for active silences")

		filter := infrasilencing.SilenceFilter{
			Statuses: []silencing.SilenceStatus{silencing.SilenceStatusActive},
			Limit:    10000, // Max active silences to fetch
		}

		var err error
		silences, err = sm.repo.ListSilences(ctx, filter)
		if err != nil {
			sm.logger.Warn("Failed to fetch active silences from repository",
				"error", err,
			)
			return nil, fmt.Errorf("get active silences: %w", err)
		}

		sm.logger.Debug("Fetched active silences from repository", "count", len(silences))
	} else {
		sm.logger.Debug("Returned active silences from cache", "count", len(silences))
	}

	return silences, nil
}

// IsAlertSilenced implements SilenceManager.IsAlertSilenced.
func (sm *DefaultSilenceManager) IsAlertSilenced(
	ctx context.Context,
	alert *silencing.Alert,
) (bool, []string, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		sm.logger.Debug("IsAlertSilenced completed", "duration_seconds", duration)
	}()

	// Step 1: Check manager state
	if !sm.started.Load() {
		return false, nil, ErrManagerNotStarted
	}
	if sm.shutdown.Load() {
		return false, nil, ErrManagerShutdown
	}

	// Step 2: Validate input
	if alert == nil || alert.Labels == nil {
		sm.logger.Error("Invalid alert: nil alert or nil labels")
		return false, nil, ErrInvalidAlert
	}

	// Step 3: Get active silences
	silences, err := sm.GetActiveSilences(ctx)
	if err != nil {
		// Fail-safe: on error, assume alert is NOT silenced (fail open)
		// This prevents blocking alerts if silence system has issues
		sm.logger.Warn("Failed to get active silences, assuming alert NOT silenced (fail-safe)",
			"error", err,
			"alert_labels", alert.Labels,
		)
		return false, nil, nil
	}

	// Step 4: Check if alert matches any silence
	var matchedIDs []string

	for _, silence := range silences {
		// Check context cancellation (prevent long-running operations)
		select {
		case <-ctx.Done():
			sm.logger.Warn("IsAlertSilenced cancelled", "error", ctx.Err())
			return false, nil, ctx.Err()
		default:
			// Continue processing
		}

		// Use matcher to check if alert matches this silence
		matched, err := sm.matcher.Matches(ctx, *alert, silence)
		if err != nil {
			// Log error but continue checking other silences (graceful degradation)
			sm.logger.Warn("Matcher error, skipping silence",
				"silence_id", silence.ID,
				"error", err,
			)
			continue
		}

		if matched {
			matchedIDs = append(matchedIDs, silence.ID)
			sm.logger.Debug("Alert matched silence",
				"silence_id", silence.ID,
				"alert_labels", alert.Labels,
			)
		}
	}

	// Step 5: Return result
	silenced := len(matchedIDs) > 0

	if silenced {
		sm.logger.Info("Alert is silenced",
			"matched_count", len(matchedIDs),
			"silence_ids", matchedIDs,
			"alert_labels", alert.Labels,
		)
	} else {
		sm.logger.Debug("Alert is NOT silenced",
			"alert_labels", alert.Labels,
			"checked_silences", len(silences),
		)
	}

	return silenced, matchedIDs, nil
}
