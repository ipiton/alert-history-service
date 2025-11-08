# TN-049: Target Health Monitoring - Technical Design

**Module**: PHASE 5 - Publishing System
**Task ID**: TN-049
**Status**: 🟡 IN PROGRESS
**Target Quality**: 150% (Enterprise-Grade)
**Design Version**: 1.0
**Last Updated**: 2025-11-08

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Data Structures](#3-data-structures)
4. [Health Check Algorithm](#4-health-check-algorithm)
5. [Status Management](#5-status-management)
6. [HTTP Connectivity Test](#6-http-connectivity-test)
7. [Observability](#7-observability)
8. [Thread Safety](#8-thread-safety)
9. [Performance Optimization](#9-performance-optimization)
10. [Error Handling](#10-error-handling)
11. [Configuration](#11-configuration)
12. [API Design](#12-api-design)
13. [Testing Strategy](#13-testing-strategy)
14. [Integration Points](#14-integration-points)
15. [Deployment](#15-deployment)
16. [Monitoring & Alerting](#16-monitoring--alerting)

---

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Alert History Service                            │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │           HealthMonitor (TN-049)                              │  │
│  ├───────────────────────────────────────────────────────────────┤  │
│  │                                                                 │  │
│  │  ┌──────────────────┐      ┌──────────────────────────────┐  │  │
│  │  │ Background       │      │ HTTP API Handlers            │  │  │
│  │  │ Health Worker    │      │                              │  │  │
│  │  │ (Periodic 2m)    │      │ GET  /health                 │  │  │
│  │  │                  │      │ GET  /health/{name}          │  │  │
│  │  │                  │      │ POST /health/{name}/check    │  │  │
│  │  └─────────┬────────┘      └──────────┬───────────────────┘  │  │
│  │            │                           │                      │  │
│  │            └────────────┬──────────────┘                      │  │
│  │                         │                                     │  │
│  │              ┌──────────▼──────────────┐                      │  │
│  │              │  HealthCheckExecutor    │                      │  │
│  │              │  (Parallel Goroutines)  │                      │  │
│  │              └──────────┬──────────────┘                      │  │
│  │                         │                                     │  │
│  │              ┌──────────▼──────────────┐                      │  │
│  │              │  HTTPConnectivityTest   │                      │  │
│  │              │  (TCP + HTTP GET/POST)  │                      │  │
│  │              └──────────┬──────────────┘                      │  │
│  │                         │                                     │  │
│  │              ┌──────────▼──────────────┐                      │  │
│  │              │   HealthStatusManager   │                      │  │
│  │              │   (Status Cache)        │                      │  │
│  │              └──────────┬──────────────┘                      │  │
│  │                         │                                     │  │
│  │              ┌──────────▼──────────────┐                      │  │
│  │              │   MetricsRecorder       │                      │  │
│  │              │   (6 Prometheus)        │                      │  │
│  │              └─────────────────────────┘                      │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                              │                                      │
│                              │ ListTargets()                        │
│                              ▼                                      │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │       TargetDiscoveryManager (TN-047)                       │  │
│  │       - ListTargets()                                        │  │
│  │       - GetTarget(name)                                      │  │
│  └─────────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┴────────────────┐
              │                                │
     ┌────────▼──────────┐         ┌──────────▼────────────┐
     │  Publishing       │         │  Grafana Dashboard   │
     │  Pipeline         │         │  (Health Metrics)    │
     │  (TN-51-60)       │         └──────────────────────┘
     └───────────────────┘
```

### 1.2 Component Responsibilities

| Component | Responsibility | Performance | Owner |
|-----------|----------------|-------------|-------|
| **HealthMonitor** | Orchestrates health checks lifecycle | <1ms Start/Stop | TN-049 |
| **Background Worker** | Periodic health checks (2m interval) | <10s for 20 targets | TN-049 |
| **HTTP API Handlers** | REST endpoints for health query/trigger | <50ms (cached) | TN-049 |
| **HealthCheckExecutor** | Parallel health check execution | <500ms per target | TN-049 |
| **HTTPConnectivityTest** | TCP + HTTP GET/POST | <500ms (with timeout) | TN-049 |
| **HealthStatusManager** | Status cache & state transitions | <10ms lookup | TN-049 |
| **MetricsRecorder** | Prometheus metrics recording | <10µs per metric | TN-049 |
| **TargetDiscoveryManager** | Provides target list | O(1) lookup | TN-047 ✅ |

### 1.3 Data Flow

#### Periodic Health Check Flow

```
1. Background Worker (ticker 2m)
   ↓
2. Get all targets from TargetDiscoveryManager.ListTargets()
   ↓
3. Filter enabled targets (skip disabled)
   ↓
4. HealthCheckExecutor.CheckAll(targets) [Parallel]
   │
   ├─→ Target 1: HTTPConnectivityTest.Check()
   │   ├─→ TCP Handshake (net.DialTimeout)
   │   ├─→ HTTP GET/POST (http.Client.Do)
   │   ├─→ Measure latency (time.Since)
   │   └─→ Return HealthCheckResult
   │
   ├─→ Target 2: HTTPConnectivityTest.Check()
   │   └─→ ...
   │
   └─→ Target N: HTTPConnectivityTest.Check()
       └─→ ...
   ↓
5. HealthStatusManager.Update(results)
   ├─→ Apply failure threshold (3 consecutive failures)
   ├─→ Detect recovery (1 success)
   ├─→ Detect degraded (latency >= 5s)
   └─→ Update status cache
   ↓
6. MetricsRecorder.Record(results)
   ├─→ Increment health_checks_total
   ├─→ Observe duration histogram
   ├─→ Set target_health_status gauge
   └─→ Update consecutive_failures gauge
   ↓
7. Logger.LogStatusChanges(transitions)
   └─→ INFO: "target X transitioned to unhealthy"
   ↓
8. Sleep until next interval (2m)
   ↓
9. Repeat from step 2
```

#### Manual Health Check Flow (API)

```
POST /api/v2/publishing/targets/health/{name}/check

1. HTTP Handler (handlers/publishing_health.go)
   ↓
2. Validate target exists (404 if not found)
   ↓
3. HealthCheckExecutor.CheckSingle(targetName)
   ├─→ Get target from TargetDiscoveryManager
   ├─→ HTTPConnectivityTest.Check(target)
   ├─→ Measure latency
   └─→ Return HealthCheckResult
   ↓
4. HealthStatusManager.Update(result) [Bypass failure threshold]
   └─→ Update status immediately (no 3 failures threshold)
   ↓
5. MetricsRecorder.Record(result)
   ↓
6. Return JSON response:
   {
     "name": "rootly-prod",
     "status": "healthy",
     "latency_ms": 123,
     "last_check": "2025-11-08T10:30:45Z"
   }
```

---

## 2. Component Design

### 2.1 HealthMonitor Interface

```go
// HealthMonitor manages target health checks lifecycle.
//
// This interface provides methods for:
//   - Starting/stopping background health worker (periodic checks)
//   - Getting current health status (for API/publishing)
//   - Triggering immediate health check (manual via API)
//
// Thread Safety:
//   - All methods are safe for concurrent use
//
// Performance:
//   - Start/Stop: <1ms (O(1), non-blocking)
//   - GetHealth: <10ms (O(1) cache lookup)
//   - CheckNow: <500ms (includes HTTP request)
type HealthMonitor interface {
	// Start begins background health check worker.
	//
	// This method:
	//   1. Validates monitor not already started
	//   2. Creates background goroutine for periodic checks
	//   3. Schedules first check (after 10s warmup)
	//   4. Returns immediately (non-blocking)
	//
	// Returns:
	//   - nil on success
	//   - ErrAlreadyStarted if monitor already running
	//
	// Performance: <1ms (spawns goroutine)
	// Thread-Safe: Yes
	Start() error

	// Stop gracefully stops background health check worker.
	//
	// This method:
	//   1. Cancels context (stops new checks)
	//   2. Waits for current check to complete (max timeout)
	//   3. Cleans up goroutine resources
	//   4. Returns when fully stopped or timeout exceeded
	//
	// Parameters:
	//   - timeout: Max time to wait (e.g., 10s)
	//
	// Returns:
	//   - nil if stopped cleanly
	//   - ErrShutdownTimeout if timeout exceeded
	//   - ErrNotStarted if monitor not running
	//
	// Performance: <10s (waits for current check)
	// Thread-Safe: Yes
	Stop(timeout time.Duration) error

	// GetHealth returns current health status for all targets.
	//
	// This method:
	//   1. Retrieves all targets from TargetDiscoveryManager
	//   2. Looks up health status from cache (O(1))
	//   3. Returns array of TargetHealthStatus
	//
	// Returns:
	//   - []TargetHealthStatus: Health status for all targets
	//   - error: If failed to get targets from discovery
	//
	// Performance: <50ms (O(n) for n targets)
	// Thread-Safe: Yes
	GetHealth(ctx context.Context) ([]TargetHealthStatus, error)

	// GetHealthByName returns health status for single target.
	//
	// Parameters:
	//   - targetName: Name of target (e.g., "rootly-prod")
	//
	// Returns:
	//   - *TargetHealthStatus: Health status
	//   - error: ErrTargetNotFound if target doesn't exist
	//
	// Performance: <10ms (O(1) lookup)
	// Thread-Safe: Yes
	GetHealthByName(ctx context.Context, targetName string) (*TargetHealthStatus, error)

	// CheckNow triggers immediate health check for target.
	//
	// This method:
	//   1. Validates target exists
	//   2. Performs immediate HTTP connectivity test
	//   3. Updates health status (bypasses failure threshold)
	//   4. Returns updated health status
	//
	// Parameters:
	//   - targetName: Name of target to check
	//
	// Returns:
	//   - *TargetHealthStatus: Updated health status
	//   - error: ErrTargetNotFound if target doesn't exist
	//
	// Performance: <500ms (includes HTTP request)
	// Thread-Safe: Yes
	CheckNow(ctx context.Context, targetName string) (*TargetHealthStatus, error)

	// GetStats returns health check statistics.
	//
	// Returns aggregate stats:
	//   - Total targets
	//   - Healthy count
	//   - Unhealthy count
	//   - Degraded count
	//   - Unknown count
	//   - Last check time
	//
	// Performance: <20ms (O(n) for n targets)
	// Thread-Safe: Yes
	GetStats(ctx context.Context) (*HealthStats, error)
}
```

### 2.2 DefaultHealthMonitor Implementation

```go
// DefaultHealthMonitor is production implementation of HealthMonitor.
type DefaultHealthMonitor struct {
	// Dependencies
	discoveryMgr publishing.TargetDiscoveryManager // Get targets
	httpClient   *http.Client                      // HTTP connectivity tests
	config       HealthConfig                      // Configuration

	// State
	statusCache *healthStatusCache // Thread-safe cache
	running     atomic.Bool        // Is worker running?
	cancel      context.CancelFunc // Cancel worker
	wg          sync.WaitGroup     // Track worker goroutine

	// Observability
	logger  *slog.Logger
	metrics *HealthMetrics
}

// NewHealthMonitor creates DefaultHealthMonitor.
func NewHealthMonitor(
	discoveryMgr publishing.TargetDiscoveryManager,
	config HealthConfig,
	logger *slog.Logger,
	metricsRegistry *metrics.Registry,
) (*DefaultHealthMonitor, error) {
	// Validation
	if discoveryMgr == nil {
		return nil, ErrNilDiscoveryManager
	}
	if logger == nil {
		logger = slog.Default()
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
		},
	}

	// Create metrics
	healthMetrics, err := NewHealthMetrics(metricsRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create health metrics: %w", err)
	}

	return &DefaultHealthMonitor{
		discoveryMgr: discoveryMgr,
		httpClient:   httpClient,
		config:       config,
		statusCache:  newHealthStatusCache(),
		logger:       logger,
		metrics:      healthMetrics,
	}, nil
}
```

### 2.3 healthStatusCache (Thread-Safe Cache)

```go
// healthStatusCache is thread-safe cache for target health status.
type healthStatusCache struct {
	mu     sync.RWMutex
	data   map[string]*TargetHealthStatus // key: target name
	maxAge time.Duration                   // Max age for stale entries
}

func newHealthStatusCache() *healthStatusCache {
	return &healthStatusCache{
		data:   make(map[string]*TargetHealthStatus),
		maxAge: 10 * time.Minute, // Consider stale after 10m
	}
}

// Get retrieves health status (O(1)).
func (c *healthStatusCache) Get(targetName string) (*TargetHealthStatus, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status, exists := c.data[targetName]
	if !exists {
		return nil, false
	}

	// Check if stale
	if time.Since(status.LastCheck) > c.maxAge {
		return nil, false // Treat stale as not found
	}

	return status, true
}

// Set stores health status (O(1)).
func (c *healthStatusCache) Set(status *TargetHealthStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[status.TargetName] = status
}

// GetAll returns all health statuses (O(n)).
func (c *healthStatusCache) GetAll() []TargetHealthStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	statuses := make([]TargetHealthStatus, 0, len(c.data))
	for _, status := range c.data {
		statuses = append(statuses, *status)
	}

	return statuses
}

// Delete removes health status (O(1)).
func (c *healthStatusCache) Delete(targetName string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.data, targetName)
}

// Clear removes all entries.
func (c *healthStatusCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]*TargetHealthStatus)
}
```

---

## 3. Data Structures

### 3.1 TargetHealthStatus

```go
// TargetHealthStatus represents health status of a publishing target.
type TargetHealthStatus struct {
	// Target Info
	TargetName string        `json:"target_name"`
	TargetType string        `json:"target_type"` // rootly/pagerduty/slack/webhook
	Enabled    bool          `json:"enabled"`

	// Health Status
	Status             HealthStatus  `json:"status"`              // healthy/unhealthy/degraded/unknown
	LatencyMs          *int64        `json:"latency_ms"`          // Response time (null if error)
	ErrorMessage       *string       `json:"error_message"`       // Error details (null if healthy)

	// Timestamps
	LastCheck          time.Time     `json:"last_check"`          // Last health check time
	LastSuccess        *time.Time    `json:"last_success"`        // Last successful check (null if never)
	LastFailure        *time.Time    `json:"last_failure"`        // Last failed check (null if never)

	// Statistics
	ConsecutiveFailures int          `json:"consecutive_failures"` // Current streak
	TotalChecks         int64        `json:"total_checks"`         // Lifetime checks
	TotalSuccesses      int64        `json:"total_successes"`      // Lifetime successes
	TotalFailures       int64        `json:"total_failures"`       // Lifetime failures
	SuccessRate         float64      `json:"success_rate"`         // (successes / total_checks) * 100
}

// HealthStatus represents health state.
type HealthStatus string

const (
	HealthStatusUnknown   HealthStatus = "unknown"   // No checks yet
	HealthStatusHealthy   HealthStatus = "healthy"   // Working normally
	HealthStatusUnhealthy HealthStatus = "unhealthy" // Failed (3+ consecutive)
	HealthStatusDegraded  HealthStatus = "degraded"  // Slow (latency >= 5s)
)

// IsHealthy returns true if status is healthy.
func (s HealthStatus) IsHealthy() bool {
	return s == HealthStatusHealthy
}

// IsUnhealthy returns true if status is unhealthy.
func (s HealthStatus) IsUnhealthy() bool {
	return s == HealthStatusUnhealthy
}

// IsDegraded returns true if status is degraded.
func (s HealthStatus) IsDegraded() bool {
	return s == HealthStatusDegraded
}
```

### 3.2 HealthCheckResult

```go
// HealthCheckResult represents result of single health check.
type HealthCheckResult struct {
	// Target Info
	TargetName string `json:"target_name"`
	TargetURL  string `json:"target_url"`

	// Check Result
	Success      bool          `json:"success"`       // Did check succeed?
	LatencyMs    *int64        `json:"latency_ms"`    // Response time (null if error)
	StatusCode   *int          `json:"status_code"`   // HTTP status code (null if TCP error)
	ErrorMessage *string       `json:"error_message"` // Error details (null if success)
	ErrorType    *ErrorType    `json:"error_type"`    // Error classification

	// Metadata
	CheckedAt    time.Time     `json:"checked_at"`    // When check was performed
	CheckType    CheckType     `json:"check_type"`    // periodic/manual
}

// CheckType represents health check trigger type.
type CheckType string

const (
	CheckTypePeriodic CheckType = "periodic" // Background worker
	CheckTypeManual   CheckType = "manual"   // HTTP API trigger
)

// ErrorType classifies health check errors.
type ErrorType string

const (
	ErrorTypeTimeout      ErrorType = "timeout"       // Connection timeout
	ErrorTypeDNS          ErrorType = "dns"           // DNS resolution failed
	ErrorTypeTLS          ErrorType = "tls"           // TLS handshake failed
	ErrorTypeRefused      ErrorType = "refused"       // Connection refused
	ErrorTypeHTTP         ErrorType = "http_error"    // HTTP status >= 400
	ErrorTypeUnknown      ErrorType = "unknown"       // Other error
)
```

### 3.3 HealthConfig

```go
// HealthConfig configures health monitoring.
type HealthConfig struct {
	// Timing
	CheckInterval       time.Duration // Interval between checks (default: 2m)
	HTTPTimeout         time.Duration // HTTP request timeout (default: 5s)
	WarmupDelay         time.Duration // Delay before first check (default: 10s)

	// Thresholds
	FailureThreshold    int           // Consecutive failures → unhealthy (default: 3)
	DegradedThreshold   time.Duration // Latency threshold for degraded (default: 5s)

	// Parallelism
	MaxConcurrentChecks int           // Max parallel health checks (default: 10)

	// HTTP Client
	MaxIdleConns        int           // HTTP client connection pool (default: 100)
	TLSSkipVerify       bool          // Skip TLS verification (default: false)
	FollowRedirects     bool          // Follow HTTP redirects (default: true)
	MaxRedirects        int           // Max redirect hops (default: 3)
}

// DefaultHealthConfig returns default configuration.
func DefaultHealthConfig() HealthConfig {
	return HealthConfig{
		CheckInterval:       2 * time.Minute,
		HTTPTimeout:         5 * time.Second,
		WarmupDelay:         10 * time.Second,
		FailureThreshold:    3,
		DegradedThreshold:   5 * time.Second,
		MaxConcurrentChecks: 10,
		MaxIdleConns:        100,
		TLSSkipVerify:       false,
		FollowRedirects:     true,
		MaxRedirects:        3,
	}
}
```

### 3.4 HealthStats

```go
// HealthStats represents aggregate health statistics.
type HealthStats struct {
	// Counts
	TotalTargets    int `json:"total_targets"`
	HealthyCount    int `json:"healthy_count"`
	UnhealthyCount  int `json:"unhealthy_count"`
	DegradedCount   int `json:"degraded_count"`
	UnknownCount    int `json:"unknown_count"`

	// Last Check
	LastCheckTime   *time.Time `json:"last_check_time"`

	// Success Rate
	OverallSuccessRate float64 `json:"overall_success_rate"` // Across all targets
}
```

---

## 4. Health Check Algorithm

### 4.1 Periodic Health Check Loop (Background Worker)

```go
// runHealthCheckWorker is background goroutine for periodic checks.
func (m *DefaultHealthMonitor) runHealthCheckWorker(ctx context.Context) {
	defer m.wg.Done()

	m.logger.Info("Health check worker started",
		"check_interval", m.config.CheckInterval,
		"warmup_delay", m.config.WarmupDelay)

	// Wait warmup period before first check
	select {
	case <-time.After(m.config.WarmupDelay):
		// Continue to first check
	case <-ctx.Done():
		m.logger.Info("Health check worker cancelled during warmup")
		return
	}

	// Create ticker for periodic checks
	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	// Perform initial check immediately
	if err := m.checkAllTargets(ctx, CheckTypePeriodic); err != nil {
		m.logger.Error("Initial health check failed", "error", err)
	}

	// Periodic check loop
	for {
		select {
		case <-ticker.C:
			// Perform health check
			if err := m.checkAllTargets(ctx, CheckTypePeriodic); err != nil {
				m.logger.Error("Periodic health check failed", "error", err)
			}

		case <-ctx.Done():
			m.logger.Info("Health check worker stopped")
			return
		}
	}
}
```

### 4.2 Check All Targets (Parallel Execution)

```go
// checkAllTargets checks health of all enabled targets (parallel).
func (m *DefaultHealthMonitor) checkAllTargets(ctx context.Context, checkType CheckType) error {
	// Get all targets from discovery manager
	targets, err := m.discoveryMgr.ListTargets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list targets: %w", err)
	}

	// Filter enabled targets only
	enabledTargets := make([]core.PublishingTarget, 0, len(targets))
	for _, t := range targets {
		if t.Enabled {
			enabledTargets = append(enabledTargets, t)
		}
	}

	if len(enabledTargets) == 0 {
		m.logger.Debug("No enabled targets to check")
		return nil
	}

	m.logger.Debug("Checking target health",
		"total_targets", len(targets),
		"enabled_targets", len(enabledTargets),
		"check_type", checkType)

	// Create goroutine pool for parallel checks
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.config.MaxConcurrentChecks) // Limit concurrency
	results := make(chan HealthCheckResult, len(enabledTargets))

	// Launch health check goroutines
	for _, target := range enabledTargets {
		wg.Add(1)
		go func(t core.PublishingTarget) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Perform health check
			result := m.checkTarget(ctx, t, checkType)
			results <- result
		}(target)
	}

	// Wait for all checks to complete
	wg.Wait()
	close(results)

	// Process results
	for result := range results {
		m.processHealthCheckResult(result)
	}

	return nil
}
```

### 4.3 Check Single Target (HTTP Connectivity Test)

```go
// checkTarget performs health check for single target.
func (m *DefaultHealthMonitor) checkTarget(
	ctx context.Context,
	target core.PublishingTarget,
	checkType CheckType,
) HealthCheckResult {
	startTime := time.Now()

	m.logger.Debug("Checking target health",
		"target_name", target.Name,
		"target_url", target.URL,
		"check_type", checkType)

	// Perform HTTP connectivity test
	success, statusCode, latencyMs, errorMsg, errorType := m.httpConnectivityTest(ctx, target.URL)

	duration := time.Since(startTime)

	// Record metrics
	m.metrics.RecordHealthCheck(target.Name, success, duration)

	if !success {
		m.metrics.RecordHealthCheckError(target.Name, errorType)
	}

	// Build result
	result := HealthCheckResult{
		TargetName:   target.Name,
		TargetURL:    target.URL,
		Success:      success,
		LatencyMs:    latencyMs,
		StatusCode:   statusCode,
		ErrorMessage: errorMsg,
		ErrorType:    &errorType,
		CheckedAt:    time.Now(),
		CheckType:    checkType,
	}

	return result
}
```

---

## 5. Status Management

### 5.1 Process Health Check Result

```go
// processHealthCheckResult updates health status based on check result.
func (m *DefaultHealthMonitor) processHealthCheckResult(result HealthCheckResult) {
	// Get current status from cache (if exists)
	currentStatus, exists := m.statusCache.Get(result.TargetName)

	// Initialize new status if not exists
	if !exists {
		currentStatus = &TargetHealthStatus{
			TargetName: result.TargetName,
			Status:     HealthStatusUnknown,
		}
	}

	// Update statistics
	currentStatus.TotalChecks++
	currentStatus.LastCheck = result.CheckedAt

	if result.Success {
		// Success: Update success counters
		currentStatus.TotalSuccesses++
		currentStatus.LastSuccess = &result.CheckedAt
		currentStatus.LatencyMs = result.LatencyMs
		currentStatus.ErrorMessage = nil

		// Check if degraded (latency >= 5s)
		if result.LatencyMs != nil && time.Duration(*result.LatencyMs)*time.Millisecond >= m.config.DegradedThreshold {
			// Degraded: Slow response
			m.transitionStatus(currentStatus, HealthStatusDegraded, "latency >= 5s")
		} else {
			// Healthy: Fast response
			m.transitionStatus(currentStatus, HealthStatusHealthy, "check succeeded")
		}

		// Reset consecutive failures
		currentStatus.ConsecutiveFailures = 0

	} else {
		// Failure: Update failure counters
		currentStatus.TotalFailures++
		currentStatus.LastFailure = &result.CheckedAt
		currentStatus.LatencyMs = nil
		currentStatus.ErrorMessage = result.ErrorMessage

		// Increment consecutive failures
		currentStatus.ConsecutiveFailures++

		// Check failure threshold
		if currentStatus.ConsecutiveFailures >= m.config.FailureThreshold {
			// Unhealthy: Too many consecutive failures
			m.transitionStatus(currentStatus, HealthStatusUnhealthy,
				fmt.Sprintf("%d consecutive failures", currentStatus.ConsecutiveFailures))
		} else {
			// Still healthy (below threshold)
			m.logger.Debug("Target failure (below threshold)",
				"target_name", result.TargetName,
				"consecutive_failures", currentStatus.ConsecutiveFailures,
				"threshold", m.config.FailureThreshold)
		}
	}

	// Calculate success rate
	if currentStatus.TotalChecks > 0 {
		currentStatus.SuccessRate = (float64(currentStatus.TotalSuccesses) / float64(currentStatus.TotalChecks)) * 100
	}

	// Update cache
	m.statusCache.Set(currentStatus)

	// Update Prometheus gauge
	m.metrics.SetTargetHealthStatus(result.TargetName, currentStatus.Status)
	m.metrics.SetConsecutiveFailures(result.TargetName, currentStatus.ConsecutiveFailures)
	m.metrics.SetSuccessRate(result.TargetName, currentStatus.SuccessRate)
}
```

### 5.2 Status Transitions

```go
// transitionStatus transitions health status with logging.
func (m *DefaultHealthMonitor) transitionStatus(
	status *TargetHealthStatus,
	newStatus HealthStatus,
	reason string,
) {
	oldStatus := status.Status

	// Skip if no change
	if oldStatus == newStatus {
		return
	}

	// Update status
	status.Status = newStatus

	// Log transition
	logLevel := slog.LevelInfo
	if newStatus == HealthStatusUnhealthy {
		logLevel = slog.LevelWarn
	} else if newStatus == HealthStatusHealthy && oldStatus == HealthStatusUnhealthy {
		logLevel = slog.LevelInfo // Recovery
	}

	m.logger.Log(context.Background(), logLevel,
		"Target health status changed",
		"target_name", status.TargetName,
		"old_status", oldStatus,
		"new_status", newStatus,
		"reason", reason,
		"consecutive_failures", status.ConsecutiveFailures,
		"success_rate", fmt.Sprintf("%.1f%%", status.SuccessRate))
}
```

---

## 6. HTTP Connectivity Test

### 6.1 HTTP Connectivity Test Implementation

```go
// httpConnectivityTest performs TCP + HTTP connectivity test.
//
// Returns:
//   - success: true if HTTP request succeeded (200-299)
//   - statusCode: HTTP status code (or nil if TCP error)
//   - latencyMs: Response time in milliseconds (or nil if error)
//   - errorMsg: Error message (or nil if success)
//   - errorType: Classified error type
func (m *DefaultHealthMonitor) httpConnectivityTest(
	ctx context.Context,
	targetURL string,
) (success bool, statusCode *int, latencyMs *int64, errorMsg *string, errorType ErrorType) {
	startTime := time.Now()

	// Parse URL
	parsedURL, err := url.Parse(targetURL)
	if err != nil {
		msg := fmt.Sprintf("invalid URL: %s", err)
		return false, nil, nil, &msg, ErrorTypeUnknown
	}

	// Step 1: TCP Handshake (fail fast)
	host := parsedURL.Host
	if parsedURL.Port() == "" {
		// Add default port
		if parsedURL.Scheme == "https" {
			host = net.JoinHostPort(host, "443")
		} else {
			host = net.JoinHostPort(host, "80")
		}
	}

	conn, err := net.DialTimeout("tcp", host, m.config.HTTPTimeout)
	if err != nil {
		latency := time.Since(startTime).Milliseconds()
		msg := fmt.Sprintf("TCP handshake failed: %s", err)

		// Classify error
		errType := classifyNetworkError(err)

		return false, nil, &latency, &msg, errType
	}
	conn.Close() // Close TCP connection (we'll use HTTP client)

	// Step 2: HTTP Request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		latency := time.Since(startTime).Milliseconds()
		msg := fmt.Sprintf("failed to create HTTP request: %s", err)
		return false, nil, &latency, &msg, ErrorTypeUnknown
	}

	// Set User-Agent
	req.Header.Set("User-Agent", "alert-history-health-checker/1.0")

	// Perform HTTP request
	resp, err := m.httpClient.Do(req)
	if err != nil {
		latency := time.Since(startTime).Milliseconds()
		msg := fmt.Sprintf("HTTP request failed: %s", err)

		// Classify error
		errType := classifyHTTPError(err)

		return false, nil, &latency, &msg, errType
	}
	defer resp.Body.Close()

	// Measure latency
	latency := time.Since(startTime).Milliseconds()

	// Check HTTP status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// Success: 2xx status
		return true, &resp.StatusCode, &latency, nil, ""
	}

	// Failure: Non-2xx status
	msg := fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)
	return false, &resp.StatusCode, &latency, &msg, ErrorTypeHTTP
}
```

### 6.2 Error Classification

```go
// classifyNetworkError classifies network errors.
func classifyNetworkError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	errStr := err.Error()

	// Timeout
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return ErrorTypeTimeout
	}

	// DNS
	if strings.Contains(errStr, "no such host") {
		return ErrorTypeDNS
	}

	// Connection refused
	if strings.Contains(errStr, "connection refused") {
		return ErrorTypeRefused
	}

	// TLS
	if strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate") {
		return ErrorTypeTLS
	}

	return ErrorTypeUnknown
}

// classifyHTTPError classifies HTTP client errors.
func classifyHTTPError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	errStr := err.Error()

	// Timeout
	if errors.Is(err, context.DeadlineExceeded) {
		return ErrorTypeTimeout
	}

	// TLS
	if strings.Contains(errStr, "tls") || strings.Contains(errStr, "certificate") {
		return ErrorTypeTLS
	}

	// DNS
	if strings.Contains(errStr, "no such host") {
		return ErrorTypeDNS
	}

	return ErrorTypeUnknown
}
```

---

## 7. Observability

### 7.1 Prometheus Metrics

```go
// HealthMetrics tracks health check metrics.
type HealthMetrics struct {
	// Counters
	checksTotal       *prometheus.CounterVec   // By target_name, status (success/failure)
	errorsTotal       *prometheus.CounterVec   // By target_name, error_type

	// Histograms
	checkDuration     *prometheus.HistogramVec // By target_name

	// Gauges
	targetHealthStatus *prometheus.GaugeVec    // By target_name (0=unknown, 1=healthy, 2=degraded, 3=unhealthy)
	consecutiveFailures *prometheus.GaugeVec   // By target_name
	successRate       *prometheus.GaugeVec     // By target_name (percentage)
}

// NewHealthMetrics creates HealthMetrics.
func NewHealthMetrics(registry *metrics.Registry) (*HealthMetrics, error) {
	m := &HealthMetrics{}

	// 1. Health checks total (Counter)
	m.checksTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alert_history_publishing_health_checks_total",
			Help: "Total number of health checks performed by target and status",
		},
		[]string{"target_name", "status"}, // status: success/failure
	)

	// 2. Health check duration (Histogram)
	m.checkDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "alert_history_publishing_health_check_duration_seconds",
			Help:    "Health check duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to 16s
		},
		[]string{"target_name"},
	)

	// 3. Target health status (Gauge)
	m.targetHealthStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alert_history_publishing_target_health_status",
			Help: "Target health status (0=unknown, 1=healthy, 2=degraded, 3=unhealthy)",
		},
		[]string{"target_name", "target_type"},
	)

	// 4. Consecutive failures (Gauge)
	m.consecutiveFailures = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alert_history_publishing_target_consecutive_failures",
			Help: "Number of consecutive health check failures for target",
		},
		[]string{"target_name"},
	)

	// 5. Success rate (Gauge)
	m.successRate = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "alert_history_publishing_target_success_rate",
			Help: "Health check success rate percentage for target",
		},
		[]string{"target_name"},
	)

	// 6. Errors total (Counter)
	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "alert_history_publishing_health_check_errors_total",
			Help: "Total health check errors by target and error type",
		},
		[]string{"target_name", "error_type"}, // error_type: timeout/dns/tls/refused/http_error
	)

	// Register all metrics
	if err := registry.Register("health_checks_total", m.checksTotal); err != nil {
		return nil, err
	}
	if err := registry.Register("health_check_duration_seconds", m.checkDuration); err != nil {
		return nil, err
	}
	if err := registry.Register("target_health_status", m.targetHealthStatus); err != nil {
		return nil, err
	}
	if err := registry.Register("consecutive_failures", m.consecutiveFailures); err != nil {
		return nil, err
	}
	if err := registry.Register("success_rate", m.successRate); err != nil {
		return nil, err
	}
	if err := registry.Register("health_check_errors_total", m.errorsTotal); err != nil {
		return nil, err
	}

	return m, nil
}

// RecordHealthCheck records health check metrics.
func (m *HealthMetrics) RecordHealthCheck(targetName string, success bool, duration time.Duration) {
	status := "failure"
	if success {
		status = "success"
	}

	m.checksTotal.WithLabelValues(targetName, status).Inc()
	m.checkDuration.WithLabelValues(targetName).Observe(duration.Seconds())
}

// RecordHealthCheckError records health check error.
func (m *HealthMetrics) RecordHealthCheckError(targetName string, errorType ErrorType) {
	m.errorsTotal.WithLabelValues(targetName, string(errorType)).Inc()
}

// SetTargetHealthStatus sets health status gauge.
func (m *HealthMetrics) SetTargetHealthStatus(targetName string, status HealthStatus) {
	var value float64
	switch status {
	case HealthStatusUnknown:
		value = 0
	case HealthStatusHealthy:
		value = 1
	case HealthStatusDegraded:
		value = 2
	case HealthStatusUnhealthy:
		value = 3
	}

	m.targetHealthStatus.WithLabelValues(targetName, "").Set(value)
}

// SetConsecutiveFailures sets consecutive failures gauge.
func (m *HealthMetrics) SetConsecutiveFailures(targetName string, count int) {
	m.consecutiveFailures.WithLabelValues(targetName).Set(float64(count))
}

// SetSuccessRate sets success rate gauge.
func (m *HealthMetrics) SetSuccessRate(targetName string, rate float64) {
	m.successRate.WithLabelValues(targetName).Set(rate)
}
```

### 7.2 Structured Logging

```go
// Logging patterns throughout implementation

// DEBUG: Each health check
m.logger.Debug("Health check completed",
	"target_name", targetName,
	"target_url", targetURL,
	"success", success,
	"latency_ms", latencyMs,
	"status_code", statusCode)

// INFO: Status transitions (healthy ↔ unhealthy)
m.logger.Info("Target health status changed",
	"target_name", targetName,
	"old_status", oldStatus,
	"new_status", newStatus,
	"reason", reason)

// WARN: Degraded targets
m.logger.Warn("Target degraded (slow response)",
	"target_name", targetName,
	"latency_ms", latencyMs,
	"threshold_ms", m.config.DegradedThreshold.Milliseconds())

// ERROR: Health check failures (with error classification)
m.logger.Error("Health check failed",
	"target_name", targetName,
	"target_url", targetURL,
	"error", errorMsg,
	"error_type", errorType,
	"consecutive_failures", consecutiveFailures)
```

---

## 8. Thread Safety

### 8.1 Concurrency Guarantees

| Component | Mechanism | Guarantee |
|-----------|-----------|-----------|
| **healthStatusCache** | `sync.RWMutex` | Safe concurrent reads/writes |
| **DefaultHealthMonitor.running** | `atomic.Bool` | Atomic state check (no race) |
| **Background Worker** | `sync.WaitGroup` | Proper goroutine tracking |
| **Parallel Health Checks** | Goroutine pool + semaphore | Limited concurrency (10 max) |
| **Metrics Recording** | Prometheus thread-safe | No manual locking needed |

### 8.2 Lock Hierarchy

```
1. HealthMonitor state (atomic.Bool) - No lock needed
2. healthStatusCache.mu (RWMutex)     - Short-lived lock
3. Prometheus metrics                 - Thread-safe (internal locking)
```

**No deadlock risk**: Single lock per component, no nested locking.

---

## 9. Performance Optimization

### 9.1 Optimization Techniques

| Optimization | Benefit | Impact |
|--------------|---------|--------|
| **Parallel health checks** | 10 goroutines pool | 10x faster for 20 targets |
| **TCP handshake first** | Fail fast (no HTTP) | ~50% faster for unreachable |
| **Connection pooling** | Reuse HTTP connections | ~30% faster repeated checks |
| **O(1) cache lookups** | RWMutex + map | <100ns per Get() |
| **Timeout enforcement** | Cancel slow checks | Prevents hanging |
| **Semaphore limiting** | Max 10 concurrent | Avoid resource exhaustion |

### 9.2 Performance Targets vs Actual

| Operation | Target | Expected Actual | Improvement |
|-----------|--------|-----------------|-------------|
| Single health check | <500ms | ~100-300ms | 2-5x better |
| All targets (20) | <10s | ~2-5s (parallel) | 2-5x better |
| GET /health (all) | <50ms | ~10-20ms | 2-5x better |
| GET /health/{name} | <10ms | ~1-2ms | 5-10x better |
| POST /check | <500ms | ~100-300ms | 2-5x better |

---

## 10. Error Handling

### 10.1 Error Types

```go
// Error types
var (
	ErrNilDiscoveryManager = errors.New("discovery manager cannot be nil")
	ErrAlreadyStarted      = errors.New("health monitor already started")
	ErrNotStarted          = errors.New("health monitor not started")
	ErrShutdownTimeout     = errors.New("shutdown timeout exceeded")
	ErrTargetNotFound      = errors.New("target not found")
	ErrInvalidTargetURL    = errors.New("invalid target URL")
)
```

### 10.2 Error Handling Patterns

```go
// Pattern 1: Graceful degradation (continue on errors)
if err := m.checkAllTargets(ctx, CheckTypePeriodic); err != nil {
	m.logger.Error("Periodic health check failed", "error", err)
	// Continue running (don't crash worker)
}

// Pattern 2: Detailed error context
return fmt.Errorf("failed to list targets: %w", err)

// Pattern 3: Error sanitization (no sensitive data in logs)
errorMsg := sanitizeError(err.Error()) // Remove auth headers, tokens
m.logger.Error("Health check failed", "error", errorMsg)
```

---

## 11. Configuration

### 11.1 Environment Variables

```bash
# Timing
TARGET_HEALTH_CHECK_INTERVAL=2m       # Check interval
TARGET_HEALTH_CHECK_TIMEOUT=5s        # HTTP timeout
TARGET_HEALTH_WARMUP_DELAY=10s        # Warmup delay

# Thresholds
TARGET_HEALTH_FAILURE_THRESHOLD=3     # Consecutive failures
TARGET_HEALTH_DEGRADED_THRESHOLD=5s   # Latency threshold

# Parallelism
TARGET_HEALTH_MAX_CONCURRENT_CHECKS=10 # Max parallel checks

# HTTP Client
TARGET_HEALTH_TLS_SKIP_VERIFY=false   # Skip TLS verification
TARGET_HEALTH_FOLLOW_REDIRECTS=true   # Follow redirects
TARGET_HEALTH_MAX_REDIRECTS=3         # Max redirect hops
```

### 11.2 Configuration Loading

```go
// LoadHealthConfigFromEnv loads config from environment variables.
func LoadHealthConfigFromEnv() HealthConfig {
	config := DefaultHealthConfig()

	if v := os.Getenv("TARGET_HEALTH_CHECK_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.CheckInterval = d
		}
	}

	if v := os.Getenv("TARGET_HEALTH_CHECK_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			config.HTTPTimeout = d
		}
	}

	// ... (similar for other config values)

	return config
}
```

---

## 12. API Design

### 12.1 HTTP Endpoints

#### GET /api/v2/publishing/targets/health

**Description**: Get health status for all targets.

**Request**:
```http
GET /api/v2/publishing/targets/health HTTP/1.1
Host: localhost:8080
```

**Response** (200 OK):
```json
[
  {
    "target_name": "rootly-prod",
    "target_type": "rootly",
    "enabled": true,
    "status": "healthy",
    "latency_ms": 123,
    "error_message": null,
    "last_check": "2025-11-08T10:30:45Z",
    "last_success": "2025-11-08T10:30:45Z",
    "last_failure": null,
    "consecutive_failures": 0,
    "total_checks": 1234,
    "total_successes": 1232,
    "total_failures": 2,
    "success_rate": 99.8
  },
  {
    "target_name": "slack-ops",
    "target_type": "slack",
    "enabled": true,
    "status": "unhealthy",
    "latency_ms": null,
    "error_message": "connection timeout after 5s",
    "last_check": "2025-11-08T10:32:00Z",
    "last_success": "2025-11-08T09:15:30Z",
    "last_failure": "2025-11-08T10:32:00Z",
    "consecutive_failures": 5,
    "total_checks": 1120,
    "total_successes": 1080,
    "total_failures": 40,
    "success_rate": 96.4
  }
]
```

---

#### GET /api/v2/publishing/targets/health/{name}

**Description**: Get health status for single target.

**Request**:
```http
GET /api/v2/publishing/targets/health/rootly-prod HTTP/1.1
Host: localhost:8080
```

**Response** (200 OK):
```json
{
  "target_name": "rootly-prod",
  "target_type": "rootly",
  "enabled": true,
  "status": "healthy",
  "latency_ms": 123,
  "error_message": null,
  "last_check": "2025-11-08T10:30:45Z",
  "last_success": "2025-11-08T10:30:45Z",
  "last_failure": null,
  "consecutive_failures": 0,
  "total_checks": 1234,
  "total_successes": 1232,
  "total_failures": 2,
  "success_rate": 99.8
}
```

**Response** (404 Not Found):
```json
{
  "error": "target not found",
  "target_name": "invalid-target"
}
```

---

#### POST /api/v2/publishing/targets/health/{name}/check

**Description**: Trigger immediate health check for target.

**Request**:
```http
POST /api/v2/publishing/targets/health/rootly-prod/check HTTP/1.1
Host: localhost:8080
```

**Response** (200 OK - healthy target):
```json
{
  "target_name": "rootly-prod",
  "target_type": "rootly",
  "enabled": true,
  "status": "healthy",
  "latency_ms": 145,
  "error_message": null,
  "last_check": "2025-11-08T10:45:12Z",
  "last_success": "2025-11-08T10:45:12Z",
  "last_failure": null,
  "consecutive_failures": 0,
  "total_checks": 1235,
  "total_successes": 1233,
  "total_failures": 2,
  "success_rate": 99.8
}
```

**Response** (503 Service Unavailable - unhealthy target):
```json
{
  "target_name": "slack-ops",
  "target_type": "slack",
  "enabled": true,
  "status": "unhealthy",
  "latency_ms": null,
  "error_message": "connection timeout after 5s",
  "last_check": "2025-11-08T10:45:20Z",
  "last_success": "2025-11-08T09:15:30Z",
  "last_failure": "2025-11-08T10:45:20Z",
  "consecutive_failures": 6,
  "total_checks": 1121,
  "total_successes": 1080,
  "total_failures": 41,
  "success_rate": 96.4
}
```

**Response** (404 Not Found):
```json
{
  "error": "target not found",
  "target_name": "invalid-target"
}
```

---

## 13. Testing Strategy

### 13.1 Unit Tests

**Test Coverage Target**: ≥85% (150% quality)

**Test Categories**:

1. **HealthMonitor Lifecycle** (5 tests)
   - `TestStart_Success`
   - `TestStart_AlreadyStarted`
   - `TestStop_Success`
   - `TestStop_Timeout`
   - `TestStop_NotStarted`

2. **Health Check Logic** (8 tests)
   - `TestCheckTarget_Success`
   - `TestCheckTarget_Failure_Timeout`
   - `TestCheckTarget_Failure_DNS`
   - `TestCheckTarget_Failure_TLS`
   - `TestCheckTarget_Failure_Refused`
   - `TestCheckTarget_Degraded`
   - `TestCheckAllTargets_Parallel`
   - `TestCheckAllTargets_MixedResults`

3. **Status Management** (7 tests)
   - `TestProcessHealthCheckResult_FirstSuccess`
   - `TestProcessHealthCheckResult_ConsecutiveFailures`
   - `TestProcessHealthCheckResult_Recovery`
   - `TestProcessHealthCheckResult_Degraded`
   - `TestTransitionStatus_HealthyToUnhealthy`
   - `TestTransitionStatus_UnhealthyToHealthy`
   - `TestSuccessRateCalculation`

4. **Cache Operations** (5 tests)
   - `TestHealthStatusCache_GetSet`
   - `TestHealthStatusCache_GetAll`
   - `TestHealthStatusCache_Delete`
   - `TestHealthStatusCache_Clear`
   - `TestHealthStatusCache_Concurrent`

5. **HTTP API** (5 tests)
   - `TestGetHealth_AllTargets`
   - `TestGetHealthByName_Found`
   - `TestGetHealthByName_NotFound`
   - `TestCheckNow_Success`
   - `TestCheckNow_NotFound`

**Total**: 30 unit tests

### 13.2 Benchmarks

**Benchmark Target**: 6+ benchmarks

1. `BenchmarkCheckTarget` - Health check performance
2. `BenchmarkCheckAllTargets_20` - Parallel checks (20 targets)
3. `BenchmarkHealthStatusCache_Get` - Cache lookup speed
4. `BenchmarkHealthStatusCache_GetAll` - List all (20 targets)
5. `BenchmarkGetHealth` - API endpoint performance
6. `BenchmarkProcessHealthCheckResult` - Status update speed

### 13.3 Integration Tests

**Test Scenarios** (optional, deferred):

1. **Real HTTP Servers**: Test against actual HTTP servers (not mocked)
2. **K8s Integration**: Test with real TargetDiscoveryManager
3. **Long-Running**: Run health checks for 5 minutes (stability)
4. **Stress Test**: 100+ targets concurrently

---

## 14. Integration Points

### 14.1 Integration with TargetDiscoveryManager (TN-047)

```go
// main.go integration

// Create TargetDiscoveryManager (TN-047)
discoveryMgr, err := publishing.NewTargetDiscoveryManager(
	k8sClient,
	cfg.Publishing.Namespace,
	cfg.Publishing.LabelSelector,
	logger,
	metricsRegistry,
)
if err != nil {
	log.Fatal("Failed to create target discovery manager", "error", err)
}

// Create HealthMonitor (TN-049)
healthConfig := publishing.LoadHealthConfigFromEnv()
healthMonitor, err := publishing.NewHealthMonitor(
	discoveryMgr,
	healthConfig,
	logger,
	metricsRegistry,
)
if err != nil {
	log.Fatal("Failed to create health monitor", "error", err)
}

// Start health monitoring
if err := healthMonitor.Start(); err != nil {
	log.Fatal("Failed to start health monitor", "error", err)
}
defer healthMonitor.Stop(10 * time.Second)
```

### 14.2 Integration with RefreshManager (TN-048)

```go
// Re-check targets after refresh
refreshMgr.OnRefreshComplete(func() {
	// Trigger immediate health check
	if err := healthMonitor.CheckAllNow(); err != nil {
		logger.Error("Failed to re-check targets after refresh", "error", err)
	}
})
```

### 14.3 Integration with Publishing Pipeline (TN-51+)

```go
// Alert publishing logic (TN-051 Alert Formatter)

// Check target health before publishing
health, err := healthMonitor.GetHealthByName(ctx, targetName)
if err != nil {
	return fmt.Errorf("failed to get target health: %w", err)
}

if health.Status.IsUnhealthy() {
	// Skip unhealthy target
	logger.Warn("Skipping unhealthy target",
		"target_name", targetName,
		"consecutive_failures", health.ConsecutiveFailures)
	return nil // Don't publish
}

// Publish to healthy/degraded target
return publisher.Publish(ctx, alert, target)
```

---

## 15. Deployment

### 15.1 K8s Deployment (Helm)

**values.yaml** (helm chart):
```yaml
health:
  enabled: true
  config:
    check_interval: "2m"
    http_timeout: "5s"
    failure_threshold: 3
    degraded_threshold: "5s"
    max_concurrent_checks: 10
```

**deployment.yaml**:
```yaml
env:
  - name: TARGET_HEALTH_CHECK_INTERVAL
    value: {{ .Values.health.config.check_interval | quote }}
  - name: TARGET_HEALTH_CHECK_TIMEOUT
    value: {{ .Values.health.config.http_timeout | quote }}
  - name: TARGET_HEALTH_FAILURE_THRESHOLD
    value: {{ .Values.health.config.failure_threshold | quote }}
  - name: TARGET_HEALTH_DEGRADED_THRESHOLD
    value: {{ .Values.health.config.degraded_threshold | quote }}
  - name: TARGET_HEALTH_MAX_CONCURRENT_CHECKS
    value: {{ .Values.health.config.max_concurrent_checks | quote }}
```

### 15.2 Production Checklist

- [ ] Configure check interval (2m default)
- [ ] Configure HTTP timeout (5s default)
- [ ] Configure failure threshold (3 default)
- [ ] Set up Grafana dashboard (health metrics)
- [ ] Create Prometheus alerting rules (unhealthy targets > 0)
- [ ] Test graceful shutdown (10s timeout)
- [ ] Monitor CPU/memory usage (<5% CPU)

---

## 16. Monitoring & Alerting

### 16.1 Grafana Dashboard Panels

**Panel 1: Target Health Status (Gauge)**
```promql
alert_history_publishing_target_health_status{target_name="rootly-prod"}
```

**Panel 2: Unhealthy Targets Count (Single Stat)**
```promql
count(alert_history_publishing_target_health_status == 3)
```

**Panel 3: Success Rate Over Time (Graph)**
```promql
alert_history_publishing_target_success_rate{target_name="rootly-prod"}
```

**Panel 4: Health Check Duration (Heatmap)**
```promql
rate(alert_history_publishing_health_check_duration_seconds_bucket[5m])
```

**Panel 5: Consecutive Failures (Table)**
```promql
alert_history_publishing_target_consecutive_failures
```

**Panel 6: Error Types Distribution (Pie Chart)**
```promql
sum(rate(alert_history_publishing_health_check_errors_total[5m])) by (error_type)
```

### 16.2 Prometheus Alerting Rules

**Alert 1: Unhealthy Target**
```yaml
- alert: PublishingTargetUnhealthy
  expr: alert_history_publishing_target_health_status == 3
  for: 5m
  labels:
    severity: warning
  annotations:
    summary: "Publishing target {{ $labels.target_name }} is unhealthy"
    description: "Target has failed 3+ consecutive health checks"
```

**Alert 2: Multiple Unhealthy Targets**
```yaml
- alert: MultiplePublishingTargetsUnhealthy
  expr: count(alert_history_publishing_target_health_status == 3) >= 2
  for: 5m
  labels:
    severity: critical
  annotations:
    summary: "{{ $value }} publishing targets are unhealthy"
    description: "Multiple targets failing, investigate K8s cluster health"
```

**Alert 3: Degraded Target**
```yaml
- alert: PublishingTargetDegraded
  expr: alert_history_publishing_target_health_status == 2
  for: 15m
  labels:
    severity: info
  annotations:
    summary: "Publishing target {{ $labels.target_name }} is degraded (slow)"
    description: "Target latency >= 5s, consider scaling or optimizing"
```

**Alert 4: Health Check Errors High**
```yaml
- alert: PublishingHealthCheckErrorsHigh
  expr: rate(alert_history_publishing_health_check_errors_total[5m]) > 0.1
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "High health check error rate for {{ $labels.target_name }}"
    description: "Error type: {{ $labels.error_type }}"
```

---

## 17. Files to Create

### 17.1 Production Code

| File | Path | LOC | Purpose |
|------|------|-----|---------|
| **health.go** | `go-app/internal/business/publishing/health.go` | 300 | HealthMonitor interface |
| **health_impl.go** | `go-app/internal/business/publishing/health_impl.go` | 500 | DefaultHealthMonitor implementation |
| **health_worker.go** | `go-app/internal/business/publishing/health_worker.go` | 250 | Background worker |
| **health_checker.go** | `go-app/internal/business/publishing/health_checker.go` | 300 | HTTP connectivity test |
| **health_status.go** | `go-app/internal/business/publishing/health_status.go` | 200 | Status management |
| **health_cache.go** | `go-app/internal/business/publishing/health_cache.go` | 150 | Status cache |
| **health_metrics.go** | `go-app/internal/business/publishing/health_metrics.go` | 250 | Prometheus metrics |
| **health_errors.go** | `go-app/internal/business/publishing/health_errors.go` | 100 | Error types |
| **publishing_health.go** | `go-app/cmd/server/handlers/publishing_health.go` | 350 | HTTP API handlers |

**Total Production Code**: ~2,400 LOC

### 17.2 Test Code

| File | Path | LOC | Purpose |
|------|------|-----|---------|
| **health_test.go** | `go-app/internal/business/publishing/health_test.go` | 600 | Unit tests (30+ tests) |
| **health_bench_test.go** | `go-app/internal/business/publishing/health_bench_test.go` | 300 | Benchmarks (6+ benchmarks) |
| **publishing_health_test.go** | `go-app/cmd/server/handlers/publishing_health_test.go` | 400 | HTTP API tests |

**Total Test Code**: ~1,300 LOC

### 17.3 Documentation

| File | Path | LOC | Purpose |
|------|------|-----|---------|
| **HEALTH_README.md** | `go-app/internal/business/publishing/HEALTH_README.md` | 1,000 | Comprehensive guide |
| **requirements.md** | `tasks/go-migration-analysis/TN-049-target-health-monitoring/requirements.md` | 3,800 | ✅ CREATED |
| **design.md** | `tasks/go-migration-analysis/TN-049-target-health-monitoring/design.md` | 5,000 | This file |
| **tasks.md** | `tasks/go-migration-analysis/TN-049-target-health-monitoring/tasks.md` | 1,000 | Task breakdown |
| **COMPLETION_REPORT.md** | `tasks/go-migration-analysis/TN-049-target-health-monitoring/COMPLETION_REPORT.md` | 800 | Final report |

**Total Documentation**: ~11,600 LOC

---

## 18. Summary

### 18.1 Key Design Decisions

| Decision | Rationale | Impact |
|----------|-----------|--------|
| **Parallel health checks** | 10 goroutine pool | 10x faster for 20 targets |
| **Failure threshold (3)** | Avoid false positives | More stable status |
| **TCP handshake first** | Fail fast (no HTTP) | 50% faster for unreachable |
| **O(1) cache lookups** | RWMutex + map | <100ns per Get() |
| **4 health statuses** | Granular monitoring | Better observability |
| **6 Prometheus metrics** | Comprehensive monitoring | Grafana dashboards ready |

### 18.2 Quality Targets (150%)

| Metric | Target | Plan |
|--------|--------|------|
| **Test Coverage** | ≥85% | 30 unit tests + 6 benchmarks |
| **Performance** | 100%+ | All operations 2-5x faster than targets |
| **Documentation** | 100% | 11,600 LOC comprehensive docs |
| **Code Quality** | A+ | Zero linter errors, race detector clean |

---

**Document Version**: 1.0
**Last Updated**: 2025-11-08
**Word Count**: 7,500+ words
**Quality Level**: Enterprise-Grade (150% Target)
**Status**: ✅ APPROVED FOR IMPLEMENTATION
