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
