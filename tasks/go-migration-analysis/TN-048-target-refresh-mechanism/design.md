# TN-048: Target Refresh Mechanism - Technical Design

**Module**: PHASE 5 - Publishing System
**Task ID**: TN-048
**Status**: 🟡 IN PROGRESS
**Target Quality**: 150% (Enterprise-Grade)
**Design Version**: 1.0
**Last Updated**: 2025-11-08

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [RefreshManager Interface](#3-refreshmanager-interface)
4. [Background Worker](#4-background-worker)
5. [HTTP API Handlers](#5-http-api-handlers)
6. [Error Handling](#6-error-handling)
7. [Retry Logic](#7-retry-logic)
8. [State Management](#8-state-management)
9. [Observability](#9-observability)
10. [Performance Optimization](#10-performance-optimization)
11. [Thread Safety](#11-thread-safety)
12. [Lifecycle Management](#12-lifecycle-management)
13. [Configuration](#13-configuration)
14. [Testing Strategy](#14-testing-strategy)
15. [Integration Points](#15-integration-points)
16. [Deployment](#16-deployment)
17. [Monitoring & Alerting](#17-monitoring--alerting)

---

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Alert History Service                    │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────────────────────────────────────────────────┐   │
│  │           RefreshManager (TN-048)                    │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │                                                       │   │
│  │  ┌─────────────────┐      ┌────────────────────┐   │   │
│  │  │ Background      │      │ HTTP API Handlers  │   │   │
│  │  │ Worker          │      │                    │   │   │
│  │  │ (Periodic)      │      │ POST /refresh      │   │   │
│  │  │                 │      │ GET  /status       │   │   │
│  │  └────────┬────────┘      └─────────┬──────────┘   │   │
│  │           │                          │              │   │
│  │           └──────────┬───────────────┘              │   │
│  │                      │                              │   │
│  │           ┌──────────▼──────────┐                   │   │
│  │           │ RefreshCoordinator  │                   │   │
│  │           │ (Single-Flight)     │                   │   │
│  │           └──────────┬──────────┘                   │   │
│  │                      │                              │   │
│  │           ┌──────────▼──────────┐                   │   │
│  │           │ TargetDiscovery     │ ◄──TN-047        │   │
│  │           │ Manager.Discover()  │                   │   │
│  │           └──────────┬──────────┘                   │   │
│  │                      │                              │   │
│  │           ┌──────────▼──────────┐                   │   │
│  │           │  K8s Client         │ ◄──TN-046        │   │
│  │           │  ListSecrets()      │                   │   │
│  │           └──────────┬──────────┘                   │   │
│  └──────────────────────┼──────────────────────────────┘   │
│                         │                                   │
└─────────────────────────┼───────────────────────────────────┘
                          │
              ┌───────────▼────────────┐
              │  Kubernetes API Server │
              │  (Secrets)             │
              └────────────────────────┘
```

### 1.2 Component Responsibilities

| Component | Responsibility | Owned By |
|-----------|----------------|----------|
| **RefreshManager** | Orchestrates refresh lifecycle (start/stop) | TN-048 |
| **Background Worker** | Periodic refresh (5m interval) | TN-048 |
| **HTTP Handlers** | Manual refresh API (POST/GET) | TN-048 |
| **RefreshCoordinator** | Single-flight pattern, rate limiting | TN-048 |
| **Retry Logic** | Exponential backoff, error recovery | TN-048 |
| **State Manager** | Track refresh status/history | TN-048 |
| **TargetDiscoveryManager** | Actual discovery logic | TN-047 ✅ |
| **K8sClient** | K8s API integration | TN-046 ✅ |

### 1.3 Data Flow

**Periodic Refresh Flow**:
```
1. Background Worker (ticker 5m)
   ↓
2. RefreshCoordinator.TryRefresh()
   ↓
3. Acquire lock (single-flight)
   ↓
4. manager.DiscoverTargets(ctx)
   ↓
5. Update State (success/failure)
   ↓
6. Record Metrics
   ↓
7. Release lock
```

**Manual Refresh Flow**:
```
1. POST /api/v2/publishing/targets/refresh
   ↓
2. Validate rate limit (max 1/min)
   ↓
3. Trigger async refresh (goroutine)
   ↓
4. Return 202 Accepted immediately
   ↓
5. Background: RefreshCoordinator.TryRefresh()
   ↓
6. Client polls: GET /status
```

---

## 2. Component Design

### 2.1 RefreshManager (Core)

**Location**: `go-app/internal/business/publishing/refresh_manager.go`

```go
// RefreshManager orchestrates target refresh lifecycle.
//
// This component:
//   - Starts/stops background worker (periodic refresh)
//   - Handles manual refresh requests (API endpoint)
//   - Enforces single-flight pattern (only 1 refresh at a time)
//   - Tracks refresh state (success/failure, timestamp, duration)
//   - Records Prometheus metrics (5 metrics)
//
// Thread Safety:
//   - All methods safe for concurrent use
//   - Internal state protected by sync.RWMutex
//
// Performance:
//   - Start/Stop: O(1), <1ms
//   - RefreshNow: O(1), <100ms (async trigger)
//   - GetStatus: O(1), <10ms (read-only)
type RefreshManager interface {
	// Start begins background refresh worker.
	//
	// This method:
	//   1. Validates manager not already started
	//   2. Creates background goroutine
	//   3. Schedules first refresh (after 30s warmup)
	//   4. Returns immediately (non-blocking)
	//
	// Returns:
	//   - nil on success
	//   - error if already started
	//
	// Example:
	//   if err := manager.Start(); err != nil {
	//       log.Fatal("Failed to start refresh manager", err)
	//   }
	Start() error

	// Stop gracefully stops background worker.
	//
	// This method:
	//   1. Cancels context (stops new refreshes)
	//   2. Waits for current refresh to complete (max timeout)
	//   3. Cleans up goroutine
	//   4. Returns when fully stopped
	//
	// Parameters:
	//   - timeout: Max time to wait (e.g., 30s)
	//
	// Returns:
	//   - nil if stopped cleanly
	//   - error if timeout exceeded
	//
	// Example:
	//   if err := manager.Stop(30*time.Second); err != nil {
	//       log.Error("Force shutdown", err)
	//   }
	Stop(timeout time.Duration) error

	// RefreshNow triggers immediate refresh (async).
	//
	// This method:
	//   1. Validates rate limit (max 1/min)
	//   2. Checks if refresh already in progress
	//   3. Spawns goroutine for async execution
	//   4. Returns immediately (202 Accepted)
	//
	// Returns:
	//   - nil if triggered successfully
	//   - ErrRefreshInProgress if already running
	//   - ErrRateLimitExceeded if called too frequently
	//
	// Example:
	//   if err := manager.RefreshNow(); err != nil {
	//       if errors.Is(err, ErrRefreshInProgress) {
	//           // Return 503
	//       }
	//   }
	RefreshNow() error

	// GetStatus returns current refresh state.
	//
	// Returns:
	//   - RefreshStatus with current state
	//   - Thread-safe (copy of internal state)
	//
	// Example:
	//   status := manager.GetStatus()
	//   log.Info("Refresh status",
	//       "state", status.State,
	//       "last_refresh", status.LastRefresh,
	//       "targets", status.TargetsDiscovered)
	GetStatus() RefreshStatus
}
```

### 2.2 DefaultRefreshManager (Implementation)

**Location**: `go-app/internal/business/publishing/refresh_manager_impl.go`

```go
// DefaultRefreshManager is production implementation of RefreshManager.
type DefaultRefreshManager struct {
	// Dependencies
	discovery k8s.TargetDiscoveryManager // TN-047
	logger    *slog.Logger
	metrics   *RefreshMetrics

	// Configuration
	interval      time.Duration // Refresh interval (default: 5m)
	maxRetries    int          // Max retry attempts (default: 5)
	baseBackoff   time.Duration // Initial backoff (default: 30s)
	maxBackoff    time.Duration // Max backoff (default: 5m)
	rateLimitPer  time.Duration // Rate limit window (default: 1m)

	// State (protected by mu)
	state         RefreshState  // current/idle/success/failed/in_progress
	lastRefresh   time.Time     // Last successful refresh timestamp
	lastError     error         // Last error (if failed)
	nextRefresh   time.Time     // Next scheduled refresh
	inProgress    bool          // True if refresh running
	targetStats   TargetStats   // Discovered/valid/invalid counts
	mu            sync.RWMutex

	// Lifecycle
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	started       bool

	// Rate limiting (protected by rateMu)
	lastManualRefresh time.Time
	rateMu            sync.Mutex
}

// NewRefreshManager creates new refresh manager.
//
// Parameters:
//   - discovery: TargetDiscoveryManager from TN-047
//   - config: RefreshConfig with intervals, retries
//   - logger: Structured logger (slog)
//   - metrics: Prometheus metrics registry
//
// Returns:
//   - RefreshManager instance
//   - error if invalid config
//
// Example:
//   config := DefaultRefreshConfig()
//   config.Interval = 5 * time.Minute
//   manager, err := NewRefreshManager(discovery, config, logger, metrics)
func NewRefreshManager(
	discovery k8s.TargetDiscoveryManager,
	config RefreshConfig,
	logger *slog.Logger,
	metrics *RefreshMetrics,
) (RefreshManager, error) {
	// Implementation...
}
```

---

## 3. RefreshManager Interface

### 3.1 Interface Definition

**Location**: `go-app/internal/business/publishing/refresh_manager.go`

```go
package publishing

import (
	"context"
	"time"
)

// RefreshManager manages target refresh lifecycle.
type RefreshManager interface {
	// Start begins background refresh worker
	Start() error

	// Stop gracefully stops refresh worker
	Stop(timeout time.Duration) error

	// RefreshNow triggers immediate refresh (async)
	RefreshNow() error

	// GetStatus returns current refresh state
	GetStatus() RefreshStatus
}

// RefreshStatus represents current refresh state.
type RefreshStatus struct {
	// State is current refresh state (idle/in_progress/success/failed)
	State RefreshState

	// LastRefresh is timestamp of last successful refresh
	LastRefresh time.Time

	// NextRefresh is timestamp of next scheduled refresh
	NextRefresh time.Time

	// RefreshDuration is duration of last refresh
	RefreshDuration time.Duration

	// TargetsDiscovered is total secrets discovered
	TargetsDiscovered int

	// TargetsValid is number of valid targets
	TargetsValid int

	// TargetsInvalid is number of invalid targets
	TargetsInvalid int

	// Error is last error message (empty if success)
	Error string

	// ConsecutiveFailures is count of consecutive failures
	ConsecutiveFailures int
}

// RefreshState represents refresh state.
type RefreshState string

const (
	// RefreshStateIdle means no refresh running, no scheduled
	RefreshStateIdle RefreshState = "idle"

	// RefreshStateInProgress means refresh currently running
	RefreshStateInProgress RefreshState = "in_progress"

	// RefreshStateSuccess means last refresh succeeded
	RefreshStateSuccess RefreshState = "success"

	// RefreshStateFailed means last refresh failed
	RefreshStateFailed RefreshState = "failed"
)

// RefreshConfig configures refresh behavior.
type RefreshConfig struct {
	// Interval is periodic refresh interval (default: 5m)
	Interval time.Duration

	// MaxRetries is max retry attempts (default: 5)
	MaxRetries int

	// BaseBackoff is initial backoff duration (default: 30s)
	BaseBackoff time.Duration

	// MaxBackoff is maximum backoff duration (default: 5m)
	MaxBackoff time.Duration

	// RateLimitPer is rate limit window (default: 1m)
	RateLimitPer time.Duration

	// RefreshTimeout is max time for single refresh (default: 30s)
	RefreshTimeout time.Duration
}

// DefaultRefreshConfig returns sensible defaults.
func DefaultRefreshConfig() RefreshConfig {
	return RefreshConfig{
		Interval:       5 * time.Minute,
		MaxRetries:     5,
		BaseBackoff:    30 * time.Second,
		MaxBackoff:     5 * time.Minute,
		RateLimitPer:   1 * time.Minute,
		RefreshTimeout: 30 * time.Second,
	}
}
```

---

## 4. Background Worker

### 4.1 Worker Goroutine

**Implementation**: `go-app/internal/business/publishing/refresh_worker.go`

```go
// runBackgroundWorker is the main goroutine for periodic refresh.
//
// This goroutine:
//   1. Waits for first scheduled refresh (warmup period)
//   2. Executes refresh via coordinator
//   3. Schedules next refresh based on interval
//   4. Repeats until context cancelled
//   5. Cleans up on exit
//
// Lifecycle:
//   - Started by Start()
//   - Stopped by Stop() (via context cancellation)
//   - Tracked by WaitGroup
func (m *DefaultRefreshManager) runBackgroundWorker() {
	defer m.wg.Done()

	m.logger.Info("Background refresh worker started",
		"interval", m.interval,
		"first_refresh_in", 30*time.Second)

	// Warmup period (avoid refresh immediately on startup)
	select {
	case <-time.After(30 * time.Second):
		// Continue to first refresh
	case <-m.ctx.Done():
		m.logger.Info("Worker cancelled during warmup")
		return
	}

	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Periodic refresh triggered
			m.logger.Debug("Periodic refresh triggered")
			m.executeRefresh(false) // isManual=false

		case <-m.ctx.Done():
			// Graceful shutdown
			m.logger.Info("Background worker stopping")
			return
		}
	}
}

// executeRefresh performs actual refresh with retry logic.
//
// This method:
//   1. Checks if refresh already in progress (skip if yes)
//   2. Acquires lock (single-flight pattern)
//   3. Calls coordinator.TryRefresh()
//   4. Updates state based on result
//   5. Records metrics
//   6. Releases lock
//
// Parameters:
//   - isManual: true if triggered via API, false if periodic
func (m *DefaultRefreshManager) executeRefresh(isManual bool) {
	// Acquire lock
	m.mu.Lock()
	if m.inProgress {
		m.logger.Debug("Refresh already in progress, skipping")
		m.mu.Unlock()
		return
	}
	m.inProgress = true
	m.state = RefreshStateInProgress
	m.mu.Unlock()

	// Record start time
	startTime := time.Now()

	// Update metrics
	m.metrics.InProgress.Set(1)
	defer m.metrics.InProgress.Set(0)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(m.ctx, m.config.RefreshTimeout)
	defer cancel()

	// Execute refresh
	err := m.refreshWithRetry(ctx)

	// Calculate duration
	duration := time.Since(startTime)

	// Update state
	m.mu.Lock()
	defer m.mu.Unlock()

	m.inProgress = false
	m.nextRefresh = time.Now().Add(m.interval)

	if err != nil {
		// Refresh failed
		m.state = RefreshStateFailed
		m.lastError = err
		m.consecutiveFailures++

		m.logger.Error("Refresh failed",
			"error", err,
			"duration", duration,
			"consecutive_failures", m.consecutiveFailures)

		// Record metrics
		m.metrics.Total.WithLabelValues("failed").Inc()
		m.metrics.Duration.WithLabelValues("failed").Observe(duration.Seconds())
		m.metrics.ErrorsTotal.WithLabelValues(classifyError(err)).Inc()
	} else {
		// Refresh succeeded
		m.state = RefreshStateSuccess
		m.lastRefresh = time.Now()
		m.lastError = nil
		m.consecutiveFailures = 0

		// Get stats from discovery manager
		stats := m.discovery.GetStats()
		m.targetStats = TargetStats{
			Total:   stats.TotalTargets,
			Valid:   stats.ValidTargets,
			Invalid: stats.InvalidTargets,
		}

		m.logger.Info("Refresh completed successfully",
			"duration", duration,
			"targets_total", stats.TotalTargets,
			"targets_valid", stats.ValidTargets,
			"targets_invalid", stats.InvalidTargets)

		// Record metrics
		m.metrics.Total.WithLabelValues("success").Inc()
		m.metrics.Duration.WithLabelValues("success").Observe(duration.Seconds())
		m.metrics.LastSuccessTimestamp.Set(float64(time.Now().Unix()))
	}
}
```

### 4.2 Warmup Period

**Rationale**: Avoid immediate refresh on service startup (allows K8s to stabilize).

**Duration**: 30 seconds (configurable via `REFRESH_WARMUP_PERIOD`)

**Behavior**:
- First refresh scheduled 30s after Start()
- Subsequent refreshes at regular interval (5m)
- Skipped if Stop() called during warmup

---

## 5. HTTP API Handlers

### 5.1 POST /api/v2/publishing/targets/refresh

**Handler Location**: `go-app/cmd/server/handlers/publishing_refresh.go`

```go
// HandleRefreshTargets triggers manual target refresh (async).
//
// This handler:
//   1. Validates rate limit (max 1/min)
//   2. Triggers async refresh
//   3. Returns 202 Accepted immediately
//   4. Returns 503 if refresh in progress
//
// Request:
//   POST /api/v2/publishing/targets/refresh
//   Content-Type: application/json
//   Body: (empty)
//
// Response (202 Accepted):
//   {
//     "message": "Refresh triggered",
//     "request_id": "abc123",
//     "refresh_started_at": "2025-11-08T10:30:45Z"
//   }
//
// Response (503 Service Unavailable):
//   {
//     "error": "refresh_in_progress",
//     "message": "Target refresh already running",
//     "started_at": "2025-11-08T10:30:40Z"
//   }
//
// Response (429 Too Many Requests):
//   {
//     "error": "rate_limit_exceeded",
//     "message": "Max 1 refresh per minute",
//     "retry_after_seconds": 45
//   }
func HandleRefreshTargets(refreshMgr publishing.RefreshManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Generate request ID
		requestID := generateRequestID()

		logger := slog.With(
			"request_id", requestID,
			"method", r.Method,
			"path", r.URL.Path,
		)

		logger.Info("Manual refresh requested")

		// Trigger async refresh
		err := refreshMgr.RefreshNow()
		if err != nil {
			if errors.Is(err, ErrRefreshInProgress) {
				// Refresh already running
				status := refreshMgr.GetStatus()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":      "refresh_in_progress",
					"message":    "Target refresh already running",
					"started_at": status.LastRefresh,
				})
				return
			}

			if errors.Is(err, ErrRateLimitExceeded) {
				// Rate limit exceeded
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":               "rate_limit_exceeded",
					"message":             "Max 1 refresh per minute",
					"retry_after_seconds": 60,
				})
				return
			}

			// Unknown error
			logger.Error("Failed to trigger refresh", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Success - return 202 Accepted
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":             "Refresh triggered",
			"request_id":          requestID,
			"refresh_started_at":  time.Now().Format(time.RFC3339),
		})

		logger.Info("Manual refresh triggered successfully")
	}
}
```

### 5.2 GET /api/v2/publishing/targets/status

**Handler Location**: `go-app/cmd/server/handlers/publishing_status.go`

```go
// HandleRefreshStatus returns current refresh status.
//
// This handler:
//   1. Gets status from RefreshManager
//   2. Returns JSON response
//   3. <10ms response time (read-only)
//
// Request:
//   GET /api/v2/publishing/targets/status
//
// Response (200 OK):
//   {
//     "status": "success",
//     "last_refresh": "2025-11-08T10:30:45Z",
//     "next_refresh": "2025-11-08T10:35:45Z",
//     "refresh_duration_ms": 1856,
//     "targets_discovered": 15,
//     "targets_valid": 14,
//     "targets_invalid": 1,
//     "consecutive_failures": 0,
//     "error": null
//   }
func HandleRefreshStatus(refreshMgr publishing.RefreshManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := refreshMgr.GetStatus()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                string(status.State),
			"last_refresh":          status.LastRefresh.Format(time.RFC3339),
			"next_refresh":          status.NextRefresh.Format(time.RFC3339),
			"refresh_duration_ms":   status.RefreshDuration.Milliseconds(),
			"targets_discovered":    status.TargetsDiscovered,
			"targets_valid":         status.TargetsValid,
			"targets_invalid":       status.TargetsInvalid,
			"consecutive_failures":  status.ConsecutiveFailures,
			"error":                 status.Error,
		})
	}
}
```

---

## 6. Error Handling

### 6.1 Error Types

**Location**: `go-app/internal/business/publishing/refresh_errors.go`

```go
package publishing

import (
	"errors"
	"fmt"
)

var (
	// ErrRefreshInProgress indicates refresh already running.
	ErrRefreshInProgress = errors.New("refresh already in progress")

	// ErrRateLimitExceeded indicates too many refresh requests.
	ErrRateLimitExceeded = errors.New("rate limit exceeded")

	// ErrAlreadyStarted indicates manager already started.
	ErrAlreadyStarted = errors.New("refresh manager already started")

	// ErrNotStarted indicates manager not started.
	ErrNotStarted = errors.New("refresh manager not started")

	// ErrShutdownTimeout indicates Stop() timeout exceeded.
	ErrShutdownTimeout = errors.New("shutdown timeout exceeded")
)

// RefreshError wraps refresh failures with context.
type RefreshError struct {
	Op        string        // Operation (e.g., "discover_targets")
	Err       error         // Underlying error
	Retries   int           // Number of retries attempted
	Duration  time.Duration // Total duration
	Transient bool          // True if error is transient (retry OK)
}

func (e *RefreshError) Error() string {
	return fmt.Sprintf("%s failed after %d retries (%v): %v",
		e.Op, e.Retries, e.Duration, e.Err)
}

func (e *RefreshError) Unwrap() error {
	return e.Err
}

// classifyError classifies error as transient or permanent.
//
// Transient errors (retry OK):
//   - Network timeout
//   - Connection refused
//   - 503 Service Unavailable
//   - Context deadline exceeded
//
// Permanent errors (no retry):
//   - 401 Unauthorized
//   - 403 Forbidden
//   - Invalid configuration
//   - Parse errors
func classifyError(err error) (errorType string, transient bool) {
	if err == nil {
		return "", false
	}

	// Check for specific error types
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "timeout", true

	case errors.Is(err, context.Canceled):
		return "cancelled", false

	case isNetworkError(err):
		return "network", true

	case isAuthError(err):
		return "auth", false

	case isParseError(err):
		return "parse", false

	default:
		return "unknown", true
	}
}
```

### 6.2 Error Response Format

**API Error Response**:
```json
{
  "error": "error_code",
  "message": "Human-readable error message",
  "details": {
    "operation": "discover_targets",
    "retries": 3,
    "duration_ms": 15000
  }
}
```

---

## 7. Retry Logic

### 7.1 Exponential Backoff

**Implementation**: `go-app/internal/business/publishing/refresh_retry.go`

```go
// refreshWithRetry executes refresh with exponential backoff retry.
//
// This method:
//   1. Attempts refresh (manager.DiscoverTargets)
//   2. On failure, classifies error (transient vs permanent)
//   3. If transient, retries with exponential backoff
//   4. If permanent, fails immediately
//   5. Returns after maxRetries or success
//
// Backoff Schedule:
//   - Attempt 1: 0s (immediate)
//   - Attempt 2: 30s (baseBackoff)
//   - Attempt 3: 1m (2x)
//   - Attempt 4: 2m (2x)
//   - Attempt 5: 4m (2x)
//   - Attempt 6: 5m (maxBackoff, capped)
//
// Returns:
//   - nil on success
//   - RefreshError with retry context
func (m *DefaultRefreshManager) refreshWithRetry(ctx context.Context) error {
	var lastErr error
	backoff := m.baseBackoff

	for attempt := 0; attempt < m.maxRetries; attempt++ {
		// Attempt refresh
		startTime := time.Now()
		err := m.discovery.DiscoverTargets(ctx)
		duration := time.Since(startTime)

		if err == nil {
			// Success!
			m.logger.Info("Refresh succeeded",
				"attempt", attempt+1,
				"duration", duration)
			return nil
		}

		// Failure - classify error
		errorType, transient := classifyError(err)
		lastErr = err

		m.logger.Warn("Refresh attempt failed",
			"attempt", attempt+1,
			"error", err,
			"error_type", errorType,
			"transient", transient,
			"duration", duration)

		// Permanent error - no retry
		if !transient {
			return &RefreshError{
				Op:        "discover_targets",
				Err:       err,
				Retries:   attempt,
				Duration:  duration,
				Transient: false,
			}
		}

		// Transient error - retry with backoff
		if attempt < m.maxRetries-1 {
			// Calculate next backoff (exponential)
			nextBackoff := backoff * 2
			if nextBackoff > m.maxBackoff {
				nextBackoff = m.maxBackoff
			}

			m.logger.Info("Retrying refresh",
				"next_attempt", attempt+2,
				"backoff", nextBackoff)

			// Wait for backoff (respecting context)
			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-ctx.Done():
				// Context cancelled during backoff
				return ctx.Err()
			}

			backoff = nextBackoff
		}
	}

	// Max retries exceeded
	return &RefreshError{
		Op:        "discover_targets",
		Err:       lastErr,
		Retries:   m.maxRetries,
		Duration:  0, // Total duration not tracked
		Transient: true,
	}
}
```

### 7.2 Retry Configuration

**Environment Variables**:
```bash
# Max retry attempts (default: 5)
REFRESH_MAX_RETRIES=5

# Initial backoff duration (default: 30s)
REFRESH_BASE_BACKOFF=30s

# Maximum backoff duration (default: 5m)
REFRESH_MAX_BACKOFF=5m
```

---

## 8. State Management

### 8.1 State Transitions

```
┌──────┐
│ IDLE │  (initial state)
└───┬──┘
    │
    │ Start() or RefreshNow()
    ▼
┌──────────────┐
│ IN_PROGRESS  │  (refresh running)
└───┬──────────┘
    │
    ├─► Success ──► ┌─────────┐
    │               │ SUCCESS │
    │               └─────────┘
    │
    └─► Failure ──► ┌─────────┐
                    │ FAILED  │
                    └─────────┘
```

### 8.2 State Storage

**Thread-Safe State**:
```go
type refreshState struct {
	state               RefreshState
	lastRefresh         time.Time
	nextRefresh         time.Time
	lastError           error
	inProgress          bool
	consecutiveFailures int
	targetStats         TargetStats
	mu                  sync.RWMutex
}

// GetState returns copy of current state (thread-safe).
func (s *refreshState) GetState() RefreshStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return RefreshStatus{
		State:               s.state,
		LastRefresh:         s.lastRefresh,
		NextRefresh:         s.nextRefresh,
		Error:               errorString(s.lastError),
		ConsecutiveFailures: s.consecutiveFailures,
		TargetsDiscovered:   s.targetStats.Total,
		TargetsValid:        s.targetStats.Valid,
		TargetsInvalid:      s.targetStats.Invalid,
	}
}
```

---

## 9. Observability

### 9.1 Prometheus Metrics

**Location**: `go-app/internal/business/publishing/refresh_metrics.go`

```go
package publishing

import (
	"github.com/prometheus/client_golang/prometheus"
)

// RefreshMetrics holds Prometheus metrics for target refresh.
type RefreshMetrics struct {
	// Total tracks total refresh attempts by status.
	// Labels: status (success/failed)
	Total *prometheus.CounterVec

	// Duration tracks refresh duration by status.
	// Labels: status (success/failed)
	Duration *prometheus.HistogramVec

	// ErrorsTotal tracks errors by type.
	// Labels: error_type (k8s_api/timeout/network/auth/parse/unknown)
	ErrorsTotal *prometheus.CounterVec

	// LastSuccessTimestamp tracks last successful refresh.
	LastSuccessTimestamp prometheus.Gauge

	// InProgress indicates if refresh currently running.
	InProgress prometheus.Gauge
}

// NewRefreshMetrics creates refresh metrics.
func NewRefreshMetrics(reg prometheus.Registerer) *RefreshMetrics {
	m := &RefreshMetrics{
		Total: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "publishing",
				Name:      "refresh_total",
				Help:      "Total number of target refresh attempts",
			},
			[]string{"status"},
		),

		Duration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "alert_history",
				Subsystem: "publishing",
				Name:      "refresh_duration_seconds",
				Help:      "Target refresh duration in seconds",
				Buckets:   []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
			},
			[]string{"status"},
		),

		ErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "alert_history",
				Subsystem: "publishing",
				Name:      "refresh_errors_total",
				Help:      "Total number of refresh errors by type",
			},
			[]string{"error_type"},
		),

		LastSuccessTimestamp: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "publishing",
				Name:      "refresh_last_success_timestamp",
				Help:      "Unix timestamp of last successful refresh",
			},
		),

		InProgress: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: "alert_history",
				Subsystem: "publishing",
				Name:      "refresh_in_progress",
				Help:      "1 if refresh currently running, 0 otherwise",
			},
		),
	}

	reg.MustRegister(m.Total, m.Duration, m.ErrorsTotal,
		m.LastSuccessTimestamp, m.InProgress)

	return m
}
```

### 9.2 Structured Logging

**Log Events**:
```go
// Startup
logger.Info("Refresh manager started",
	"interval", interval,
	"max_retries", maxRetries,
	"base_backoff", baseBackoff)

// Periodic refresh triggered
logger.Debug("Periodic refresh triggered")

// Refresh succeeded
logger.Info("Refresh completed successfully",
	"duration", duration,
	"targets_total", total,
	"targets_valid", valid,
	"targets_invalid", invalid)

// Refresh failed
logger.Error("Refresh failed",
	"error", err,
	"error_type", errorType,
	"attempt", attempt,
	"duration", duration,
	"consecutive_failures", consecutiveFailures)

// Manual refresh requested
logger.Info("Manual refresh requested",
	"request_id", requestID,
	"source", "api")

// Graceful shutdown
logger.Info("Refresh manager stopping",
	"in_progress", inProgress,
	"timeout", timeout)
```

---

## 10. Performance Optimization

### 10.1 Performance Targets

| Operation | Baseline | Target (150%) | Strategy |
|-----------|----------|---------------|----------|
| Start() | <1ms | <500µs | O(1), no blocking |
| Stop() | <5s | <3s | Graceful timeout |
| RefreshNow() | <100ms | <50ms | Async trigger |
| GetStatus() | <10ms | <5ms | Read-only, O(1) |
| Full Refresh | <5s | <3s | Parallel processing |

### 10.2 Optimization Techniques

1. **Async API Handlers**: Return 202 immediately, spawn goroutine
2. **Single-Flight Pattern**: Only 1 refresh at a time (skip duplicates)
3. **Rate Limiting**: Prevent excessive API calls
4. **Context Timeout**: 30s max for single refresh
5. **Zero Allocations**: GetStatus() uses sync.Pool

---

## 11. Thread Safety

### 11.1 Concurrency Patterns

**RWMutex Usage**:
```go
// Read operations (can run in parallel)
func (m *DefaultRefreshManager) GetStatus() RefreshStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Read state...
}

// Write operations (exclusive access)
func (m *DefaultRefreshManager) updateState(state RefreshState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = state
	// Update other fields...
}
```

**Single-Flight Pattern**:
```go
func (m *DefaultRefreshManager) executeRefresh() {
	m.mu.Lock()
	if m.inProgress {
		m.logger.Debug("Refresh already in progress, skipping")
		m.mu.Unlock()
		return
	}
	m.inProgress = true
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		m.inProgress = false
		m.mu.Unlock()
	}()

	// Perform refresh...
}
```

### 11.2 Race Condition Prevention

**Testing**:
```bash
# Run tests with race detector
go test -race ./internal/business/publishing/...

# Stress test concurrent refreshes
go test -race -run TestConcurrentRefreshes -count=100
```

---

## 12. Lifecycle Management

### 12.1 Start Sequence

```
1. NewRefreshManager()
   ├─ Validate config
   ├─ Initialize state
   ├─ Create context
   └─ Return manager

2. manager.Start()
   ├─ Check if already started
   ├─ Spawn background goroutine
   ├─ Schedule first refresh (after 30s warmup)
   ├─ Register with WaitGroup
   └─ Return immediately

3. Background Worker Running
   ├─ Wait for warmup period (30s)
   ├─ Execute first refresh
   ├─ Schedule next refresh (5m interval)
   └─ Repeat forever
```

### 12.2 Stop Sequence

```
1. manager.Stop(timeout)
   ├─ Cancel context (stops new refreshes)
   ├─ Wait for WaitGroup (max timeout)
   ├─ If timeout exceeded:
   │  └─ Force shutdown (goroutine may leak)
   └─ Return

2. Background Worker Cleanup
   ├─ Detect context cancelled
   ├─ Complete current refresh (if running)
   ├─ Clean up resources
   ├─ Call wg.Done()
   └─ Exit goroutine
```

---

## 13. Configuration

### 13.1 Environment Variables

```bash
# Refresh interval (default: 5m)
TARGET_REFRESH_INTERVAL=5m

# Max retry attempts (default: 5)
TARGET_REFRESH_MAX_RETRIES=5

# Initial backoff (default: 30s)
TARGET_REFRESH_BASE_BACKOFF=30s

# Max backoff (default: 5m)
TARGET_REFRESH_MAX_BACKOFF=5m

# Rate limit window (default: 1m)
TARGET_REFRESH_RATE_LIMIT=1m

# Refresh timeout (default: 30s)
TARGET_REFRESH_TIMEOUT=30s

# Warmup period (default: 30s)
TARGET_REFRESH_WARMUP=30s
```

### 13.2 Configuration Struct

```go
type RefreshConfig struct {
	Interval       time.Duration
	MaxRetries     int
	BaseBackoff    time.Duration
	MaxBackoff     time.Duration
	RateLimitPer   time.Duration
	RefreshTimeout time.Duration
	WarmupPeriod   time.Duration
}

func LoadRefreshConfig() (RefreshConfig, error) {
	return RefreshConfig{
		Interval:       getEnvDuration("TARGET_REFRESH_INTERVAL", 5*time.Minute),
		MaxRetries:     getEnvInt("TARGET_REFRESH_MAX_RETRIES", 5),
		BaseBackoff:    getEnvDuration("TARGET_REFRESH_BASE_BACKOFF", 30*time.Second),
		MaxBackoff:     getEnvDuration("TARGET_REFRESH_MAX_BACKOFF", 5*time.Minute),
		RateLimitPer:   getEnvDuration("TARGET_REFRESH_RATE_LIMIT", 1*time.Minute),
		RefreshTimeout: getEnvDuration("TARGET_REFRESH_TIMEOUT", 30*time.Second),
		WarmupPeriod:   getEnvDuration("TARGET_REFRESH_WARMUP", 30*time.Second),
	}, nil
}
```

---

## 14. Testing Strategy

### 14.1 Unit Tests (90%+ coverage)

**Test Files**:
1. `refresh_manager_test.go` - Manager lifecycle (Start/Stop)
2. `refresh_worker_test.go` - Background worker logic
3. `refresh_retry_test.go` - Retry logic with backoff
4. `refresh_api_test.go` - HTTP handlers
5. `refresh_errors_test.go` - Error classification
6. `refresh_state_test.go` - State management

**Test Scenarios** (minimum 15):
1. ✅ Start manager successfully
2. ✅ Stop manager gracefully (within timeout)
3. ✅ Stop manager with timeout (force shutdown)
4. ✅ Periodic refresh (happy path)
5. ✅ Manual refresh via API (POST /refresh)
6. ✅ Get refresh status (GET /status)
7. ✅ Concurrent refresh attempts (idempotency)
8. ✅ Rate limiting (max 1/min)
9. ✅ Retry with exponential backoff
10. ✅ Permanent error (no retry)
11. ✅ Transient error (retry succeeds)
12. ✅ Max retries exceeded
13. ✅ Context cancellation during refresh
14. ✅ Context timeout during refresh
15. ✅ Zero goroutine leaks

### 14.2 Integration Tests

**Scenarios**:
1. ✅ End-to-end refresh flow (K8s API → cache update)
2. ✅ Multiple refreshes with K8s API failures
3. ✅ Service restart with state recovery
4. ✅ Concurrent API calls (stress test)

### 14.3 Benchmarks (6+)

```go
func BenchmarkRefreshManagerStart(b *testing.B)
func BenchmarkRefreshManagerStop(b *testing.B)
func BenchmarkRefreshNow(b *testing.B)
func BenchmarkGetStatus(b *testing.B)
func BenchmarkFullRefresh(b *testing.B)
func BenchmarkRetryBackoff(b *testing.B)
```

---

## 15. Integration Points

### 15.1 Main.go Integration

**Location**: `go-app/cmd/server/main.go`

```go
// Initialize Target Discovery Manager (TN-047)
discoveryMgr, err := publishing.NewTargetDiscoveryManager(
	k8sClient,
	cfg.K8s.Namespace,
	"publishing-target=true",
	logger,
	metricsRegistry,
)
if err != nil {
	logger.Error("Failed to create discovery manager", "error", err)
	os.Exit(1)
}

// Initialize Refresh Manager (TN-048)
refreshConfig := publishing.DefaultRefreshConfig()
refreshConfig.Interval = cfg.Publishing.RefreshInterval

refreshMgr, err := publishing.NewRefreshManager(
	discoveryMgr,
	refreshConfig,
	logger,
	metricsRegistry,
)
if err != nil {
	logger.Error("Failed to create refresh manager", "error", err)
	os.Exit(1)
}

// Start refresh manager
if err := refreshMgr.Start(); err != nil {
	logger.Error("Failed to start refresh manager", "error", err)
	os.Exit(1)
}
defer refreshMgr.Stop(30 * time.Second)

// Register HTTP handlers
mux.HandleFunc("POST /api/v2/publishing/targets/refresh",
	handlers.HandleRefreshTargets(refreshMgr))
mux.HandleFunc("GET /api/v2/publishing/targets/status",
	handlers.HandleRefreshStatus(refreshMgr))

logger.Info("Refresh manager started",
	"interval", refreshConfig.Interval,
	"max_retries", refreshConfig.MaxRetries)
```

---

## 16. Deployment

### 16.1 Kubernetes Deployment

**ConfigMap** (`config/refresh-config.yaml`):
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: alert-history-refresh-config
data:
  TARGET_REFRESH_INTERVAL: "5m"
  TARGET_REFRESH_MAX_RETRIES: "5"
  TARGET_REFRESH_BASE_BACKOFF: "30s"
  TARGET_REFRESH_MAX_BACKOFF: "5m"
  TARGET_REFRESH_RATE_LIMIT: "1m"
  TARGET_REFRESH_TIMEOUT: "30s"
```

**Deployment Update** (`helm/alert-history/templates/deployment.yaml`):
```yaml
spec:
  containers:
  - name: alert-history
    envFrom:
    - configMapRef:
        name: alert-history-refresh-config
    livenessProbe:
      httpGet:
        path: /healthz
        port: 8080
      initialDelaySeconds: 60  # Allow warmup period
      periodSeconds: 30
```

---

## 17. Monitoring & Alerting

### 17.1 Grafana Dashboard

**Panels**:
1. **Refresh Rate**: `rate(alert_history_publishing_refresh_total[5m])`
2. **Success Rate**: `rate(alert_history_publishing_refresh_total{status="success"}[5m])`
3. **Error Rate**: `rate(alert_history_publishing_refresh_errors_total[5m])`
4. **Refresh Duration (p95)**: `histogram_quantile(0.95, alert_history_publishing_refresh_duration_seconds)`
5. **Last Success**: `time() - alert_history_publishing_refresh_last_success_timestamp`
6. **In Progress**: `alert_history_publishing_refresh_in_progress`

### 17.2 Prometheus Alerts

```yaml
groups:
- name: target_refresh
  rules:
  # Alert if no successful refresh in 15m
  - alert: TargetRefreshStale
    expr: time() - alert_history_publishing_refresh_last_success_timestamp > 900
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "Target refresh stale for >15m"

  # Alert if 3+ consecutive failures
  - alert: TargetRefreshFailing
    expr: |
      increase(alert_history_publishing_refresh_total{status="failed"}[15m]) >= 3
    labels:
      severity: critical
    annotations:
      summary: "Target refresh failing (3+ failures in 15m)"

  # Alert if refresh duration >30s
  - alert: TargetRefreshSlow
    expr: |
      histogram_quantile(0.95,
        alert_history_publishing_refresh_duration_seconds{status="success"}
      ) > 30
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "Target refresh slow (p95 >30s)"
```

---

## 18. Future Enhancements

### 18.1 Phase 2 Features (Post-MVP)

1. **Circuit Breaker** (TN-49):
   - Open circuit after 5 consecutive failures
   - Half-open state for recovery testing
   - Automatic recovery after cooldown

2. **Selective Refresh**:
   - Refresh specific target by name
   - API: `POST /api/v2/publishing/targets/{name}/refresh`

3. **Webhook Notifications**:
   - Notify on successful refresh
   - Alert on failures (Slack/PagerDuty)

4. **Caching Layer**:
   - Redis-backed cache for HA recovery
   - Fallback to memory on Redis unavailable

5. **Advanced Scheduling**:
   - Cron-based schedules (e.g., "0 */5 * * *")
   - Different intervals per target type

---

**Document Version**: 1.0
**Last Updated**: 2025-11-08
**Author**: AI Assistant
**Status**: ✅ COMPLETE (Design Phase)
