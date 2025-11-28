# TN-74: GET /enrichment/mode - Technical Design Document

**Version**: 1.0
**Date**: 2025-11-28
**Status**: Draft
**Target Quality**: 150% (Grade A+ EXCELLENT)

---

## 📋 Table of Contents

1. [Executive Summary](#executive-summary)
2. [System Architecture](#system-architecture)
3. [Component Design](#component-design)
4. [Data Flow](#data-flow)
5. [Performance Architecture](#performance-architecture)
6. [Error Handling Strategy](#error-handling-strategy)
7. [Security Design](#security-design)
8. [Observability Design](#observability-design)
9. [Testing Strategy](#testing-strategy)
10. [Deployment Architecture](#deployment-architecture)
11. [Migration & Rollback](#migration--rollback)
12. [Appendix](#appendix)

---

## 📝 Executive Summary

### Design Goals
1. **Ultra-Fast**: < 100ns p50 latency (in-memory cache hit)
2. **Highly Available**: 99.99% uptime, graceful degradation
3. **Scalable**: 100K+ req/s, linear horizontal scaling
4. **Observable**: Comprehensive metrics, logs, tracing
5. **Secure**: Optional auth, rate limiting, CORS
6. **Maintainable**: Clean code, 90%+ test coverage, extensive docs

### Architecture Style
- **Layered Architecture**: Handler → Service → Infrastructure
- **Dependency Injection**: Clean dependencies, testable
- **Cache-First Pattern**: In-memory cache → Redis → Fallback chain
- **Fail-Fast**: Timeouts, circuit breakers, early returns
- **12-Factor App**: Config via env vars, stateless, logs to stdout

### Key Technical Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| **In-memory cache** | 50ns read latency, zero network calls | Memory per pod (~10MB), 30s stale data |
| **RWMutex** | Thread-safe reads, minimal lock contention | Write lock blocks all reads (rare event) |
| **Redis fallback** | Distributed cache, pod restart resilience | 1-2ms latency on cache miss, Redis dependency |
| **Structured logging (slog)** | JSON format, fast, native Go | Go 1.21+ required |
| **Prometheus metrics** | Industry standard, Grafana integration | Pull-based (scrape interval 15s) |

---

## 🏗️ System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         Client Applications                          │
│  (Dashboard, CLI, External Services, Monitoring, Grafana)           │
└───────────────┬─────────────────────────────────────────────────────┘
                │
                │ HTTP GET /enrichment/mode
                │
┌───────────────▼─────────────────────────────────────────────────────┐
│                      Load Balancer (Kubernetes Service)             │
│                    (Round-robin, Health checks)                     │
└───────────────┬─────────────────────────────────────────────────────┘
                │
                │ Distribute to pods (2-10 replicas, HPA)
                │
        ┌───────┴────────┬────────────┬────────────┐
        │                │            │            │
┌───────▼──────┐ ┌───────▼──────┐ ┌──▼──────┐ ┌──▼──────┐
│   Pod 1      │ │   Pod 2      │ │  Pod N  │ │ Pod N+1 │
│  (Primary)   │ │  (Replica)   │ │ (Scale) │ │ (Scale) │
└───────┬──────┘ └───────┬──────┘ └──┬──────┘ └──┬──────┘
        │                │            │            │
        │  Each pod contains:                     │
        │  - HTTP Handler                         │
        │  - EnrichmentModeManager (in-memory)   │
        │  - Redis client                         │
        │                                          │
        └──────────┬───────────┬──────────────────┘
                   │           │
         ┌─────────▼───┐   ┌───▼──────────────┐
         │   Redis     │   │   Prometheus     │
         │   (Mode)    │   │   (Metrics)      │
         └─────────────┘   └──────────────────┘
```

---

### Layered Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        PRESENTATION LAYER                        │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  HTTP Handler (go-app/cmd/server/handlers/enrichment.go)  │ │
│  │  - GetMode(w http.ResponseWriter, r *http.Request)        │ │
│  │  - Request parsing, Response encoding                      │ │
│  │  - Logging, Metrics recording                              │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               │ Calls
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                         SERVICE LAYER                            │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  EnrichmentModeManager (internal/core/services/)          │ │
│  │  - GetModeWithSource(ctx) (EnrichmentMode, string, error) │ │
│  │  - In-memory cache (currentMode, currentSource)           │ │
│  │  - Auto-refresh logic (30s interval)                      │ │
│  │  - Stats tracking (switches, timestamps)                  │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────┬───────────────────────────────────┘
                               │
                               │ Queries
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                      INFRASTRUCTURE LAYER                        │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  Redis Cache (internal/infrastructure/cache/)             │ │
│  │  - Get("enrichment:mode") → {mode, timestamp}             │ │
│  │  - Fallback to ENV, Default on Redis failure             │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
```

---

### Component Interaction Diagram

```
┌──────────┐         ┌─────────────────────────┐         ┌──────────────────┐
│  Client  │────────▶│  EnrichmentHandlers    │────────▶│ EnrichmentMode  │
│          │  HTTP   │  (Presentation Layer)   │ Call    │ Manager         │
│          │  GET    │                         │         │ (Service Layer) │
└──────────┘         │  - GetMode(w, r)        │         │                 │
                     │  - JSON encode          │         │  - GetModeWith │
                     │  - Error handling       │         │    Source()     │
                     │  - Logging              │         │  - In-memory   │
                     │                         │         │    cache       │
                     └─────────────────────────┘         └────────┬────────┘
                                                                  │
                                                                  │ Query
                                                                  │
                     ┌─────────────────────────────────────────▼─┴─────────┐
                     │  Fallback Chain:                                     │
                     │  1. In-memory cache (50ns read)                      │
                     │  2. Redis GET "enrichment:mode" (1-2ms)             │
                     │  3. os.Getenv("ENRICHMENT_MODE") (100ns)           │
                     │  4. Default: "enriched" (0ns)                        │
                     └──────────────────────────────────────────────────────┘
```

---

## 🔧 Component Design

### 1. HTTP Handler (`EnrichmentHandlers`)

#### Structure
```go
// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/vitaliisemenov/alert-history/internal/core/services"
)

// EnrichmentHandlers handles enrichment mode endpoints
type EnrichmentHandlers struct {
	manager services.EnrichmentModeManager
	logger  *slog.Logger
	metrics *metrics.MetricsManager // ← New for 150%
}

// NewEnrichmentHandlers creates new enrichment handlers
func NewEnrichmentHandlers(
	manager services.EnrichmentModeManager,
	logger *slog.Logger,
	metrics *metrics.MetricsManager,
) *EnrichmentHandlers {
	if logger == nil {
		logger = slog.Default()
	}

	return &EnrichmentHandlers{
		manager: manager,
		logger:  logger,
		metrics: metrics,
	}
}
```

#### Request Flow
```go
// GetMode handles GET /enrichment/mode
func (h *EnrichmentHandlers) GetMode(w http.ResponseWriter, r *http.Request) {
	// 1. Record start time (for metrics)
	start := time.Now()

	// 2. Extract request context
	ctx := r.Context()
	requestID := getRequestID(ctx)

	// 3. Log request
	h.logger.Info("Get enrichment mode requested",
		"method", r.Method,
		"path", r.URL.Path,
		"remote_addr", r.RemoteAddr,
		"request_id", requestID,
	)

	// 4. Query service layer
	mode, source, err := h.manager.GetModeWithSource(ctx)
	if err != nil {
		h.recordError(err, start)
		h.logger.Error("Failed to get enrichment mode", "error", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Failed to get enrichment mode"})
		return
	}

	// 5. Build response
	response := EnrichmentModeResponse{
		Mode:   mode.String(),
		Source: source,
	}

	// 6. Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=30")
	w.Header().Set("X-Request-ID", requestID)

	// 7. Write response
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.logger.Error("Failed to encode response", "error", err)
		return
	}

	// 8. Record metrics
	h.recordSuccess(mode, source, start)

	// 9. Log response
	h.logger.Info("Get enrichment mode completed",
		"mode", mode,
		"source", source,
		"duration_ms", time.Since(start).Milliseconds(),
		"status", http.StatusOK,
	)
}
```

#### Metrics Recording (150% Enhancement)
```go
// recordSuccess records successful request metrics
func (h *EnrichmentHandlers) recordSuccess(mode services.EnrichmentMode, source string, start time.Time) {
	if h.metrics == nil {
		return
	}

	duration := time.Since(start)

	// Counter: Total requests
	h.metrics.Enrichment().RecordRequest("GET", "200")

	// Histogram: Request duration
	h.metrics.Enrichment().RecordDuration("GET", duration)

	// Counter: Cache hits by source
	h.metrics.Enrichment().RecordCacheHit(source)

	// Gauge: Last request timestamp
	h.metrics.Enrichment().SetLastRequestTimestamp(time.Now())
}

// recordError records error metrics
func (h *EnrichmentHandlers) recordError(err error, start time.Time) {
	if h.metrics == nil {
		return
	}

	duration := time.Since(start)

	// Counter: Total requests (with error status)
	h.metrics.Enrichment().RecordRequest("GET", "500")

	// Histogram: Request duration (even for errors)
	h.metrics.Enrichment().RecordDuration("GET", duration)

	// Counter: Errors by type
	errorType := classifyError(err)
	h.metrics.Enrichment().RecordError(errorType)
}
```

---

### 2. Service Layer (`EnrichmentModeManager`)

#### Interface
```go
// EnrichmentModeManager manages enrichment mode state
type EnrichmentModeManager interface {
	// GetMode returns current enrichment mode (uses in-memory cache)
	GetMode(ctx context.Context) (EnrichmentMode, error)

	// GetModeWithSource returns mode and source (redis/memory/env/default)
	GetModeWithSource(ctx context.Context) (EnrichmentMode, string, error)

	// SetMode sets new enrichment mode (saves to Redis + memory)
	SetMode(ctx context.Context, mode EnrichmentMode) error

	// ValidateMode validates if mode is supported
	ValidateMode(mode EnrichmentMode) error

	// GetStats returns enrichment statistics
	GetStats(ctx context.Context) (*EnrichmentStats, error)

	// RefreshCache forces cache refresh from Redis
	RefreshCache(ctx context.Context) error
}
```

#### Implementation
```go
type enrichmentModeManager struct {
	cache   cache.Cache
	logger  *slog.Logger
	metrics *metrics.MetricsManager

	// In-memory cache for fast access
	currentMode   EnrichmentMode
	currentSource string
	lastRefresh   time.Time

	// Stats
	totalSwitches  int64
	lastSwitchTime *time.Time
	lastSwitchFrom EnrichmentMode

	mu sync.RWMutex // protects in-memory state
}
```

#### GetModeWithSource() - Hot Path
```go
// GetModeWithSource returns mode and source
func (m *enrichmentModeManager) GetModeWithSource(ctx context.Context) (EnrichmentMode, string, error) {
	// Fast path: RWMutex read lock (50ns)
	m.mu.RLock()
	mode := m.currentMode
	source := m.currentSource
	lastRefresh := m.lastRefresh
	m.mu.RUnlock()

	// Auto-refresh if cache is stale (>30s old)
	if time.Since(lastRefresh) > cacheRefreshInterval {
		// Background refresh (non-blocking)
		go func() {
			if err := m.RefreshCache(context.Background()); err != nil {
				m.logger.Debug("Background cache refresh failed", "error", err)
			}
		}()
	}

	return mode, source, nil
}
```

#### RefreshCache() - Fallback Chain
```go
// RefreshCache forces cache refresh from Redis
func (m *enrichmentModeManager) RefreshCache(ctx context.Context) error {
	var mode EnrichmentMode
	var source string

	// 1. Try Redis (1-2ms on cache hit)
	if m.cache != nil {
		var data map[string]any
		err := m.cache.Get(ctx, redisKeyMode, &data)
		if err == nil && data != nil {
			if modeStr, ok := data["mode"].(string); ok {
				mode = EnrichmentMode(modeStr)
				if mode.IsValid() {
					source = "redis"
					goto found
				}
			}
		}

		// Check if error is not "not found"
		if err != nil && !cache.IsNotFound(err) {
			m.logger.Debug("Redis get failed", "error", err)
		}
	}

	// 2. Try ENV variable (100ns)
	if envMode := os.Getenv("ENRICHMENT_MODE"); envMode != "" {
		mode = EnrichmentMode(envMode)
		if mode.IsValid() {
			source = "env"
			goto found
		}
		m.logger.Warn("Invalid ENRICHMENT_MODE env variable", "value", envMode)
	}

	// 3. Use default (0ns)
	mode = defaultMode
	source = "default"

found:
	// Update in-memory cache (write lock)
	m.mu.Lock()
	oldMode := m.currentMode
	m.currentMode = mode
	m.currentSource = source
	m.lastRefresh = time.Now()
	m.mu.Unlock()

	// Update metrics if mode changed
	if oldMode != mode && m.metrics != nil {
		m.updateMetrics(oldMode, mode)
	}

	m.logger.Debug("Cache refreshed",
		"mode", mode,
		"source", source,
	)

	return nil
}
```

---

### 3. Data Models

#### EnrichmentMode
```go
// EnrichmentMode represents alert processing mode
type EnrichmentMode string

const (
	// EnrichmentModeTransparent - proxy alerts without LLM, WITH filtering
	EnrichmentModeTransparent EnrichmentMode = "transparent"

	// EnrichmentModeEnriched - classify with LLM, WITH filtering (default)
	EnrichmentModeEnriched EnrichmentMode = "enriched"

	// EnrichmentModeTransparentWithRecommendations - proxy without LLM, WITHOUT filtering
	EnrichmentModeTransparentWithRecommendations EnrichmentMode = "transparent_with_recommendations"
)

// Valid modes map
var validModes = map[EnrichmentMode]bool{
	EnrichmentModeTransparent:                    true,
	EnrichmentModeEnriched:                       true,
	EnrichmentModeTransparentWithRecommendations: true,
}

// IsValid checks if mode is valid
func (m EnrichmentMode) IsValid() bool {
	return validModes[m]
}

// String returns string representation
func (m EnrichmentMode) String() string {
	return string(m)
}

// ToMetricValue converts mode to metric gauge value (0, 1, 2)
func (m EnrichmentMode) ToMetricValue() float64 {
	switch m {
	case EnrichmentModeTransparent:
		return 0
	case EnrichmentModeEnriched:
		return 1
	case EnrichmentModeTransparentWithRecommendations:
		return 2
	default:
		return 1 // default to enriched
	}
}
```

#### HTTP Response Models
```go
// EnrichmentModeResponse represents the enrichment mode response
type EnrichmentModeResponse struct {
	Mode   string `json:"mode"`   // "transparent" | "enriched" | "transparent_with_recommendations"
	Source string `json:"source"` // "redis" | "env" | "memory" | "default"
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"` // Human-readable error message
}
```

---

## 🔄 Data Flow

### Sequence Diagram: Successful Request

```
┌──────┐     ┌────────────────┐     ┌────────────────────┐     ┌────────┐
│Client│     │EnrichmentHandler│     │EnrichmentModeManager│     │ Redis  │
└──┬───┘     └───────┬────────┘     └─────────┬──────────┘     └───┬────┘
   │                 │                         │                     │
   │ GET /enrichment/mode                      │                     │
   │────────────────▶│                         │                     │
   │                 │                         │                     │
   │                 │ GetModeWithSource(ctx)  │                     │
   │                 │────────────────────────▶│                     │
   │                 │                         │                     │
   │                 │                         │ RLock (50ns)        │
   │                 │                         │─────┐               │
   │                 │                         │     │ Read from     │
   │                 │                         │     │ in-memory     │
   │                 │                         │     │ cache         │
   │                 │                         │◀────┘               │
   │                 │                         │                     │
   │                 │  mode="enriched"        │                     │
   │                 │  source="memory"        │                     │
   │                 │◀────────────────────────│                     │
   │                 │                         │                     │
   │                 │ JSON encode             │                     │
   │                 │─────┐                   │                     │
   │                 │     │                   │                     │
   │                 │◀────┘                   │                     │
   │                 │                         │                     │
   │  200 OK         │                         │                     │
   │  {"mode": "enriched", "source": "memory"}│                     │
   │◀────────────────│                         │                     │
   │                 │                         │                     │
   │                 │ Record metrics          │                     │
   │                 │─────┐                   │                     │
   │                 │     │                   │                     │
   │                 │◀────┘                   │                     │
   │                 │                         │                     │
```

**Performance**:
- **Total latency**: ~50-100ns (in-memory cache hit)
- **RWMutex RLock**: ~50ns (read lock, no contention)
- **JSON encode**: ~50ns (2 string fields)
- **HTTP write**: ~0ns (buffered)

---

### Sequence Diagram: Cache Miss (Redis Fallback)

```
┌──────┐     ┌────────────────┐     ┌────────────────────┐     ┌────────┐
│Client│     │EnrichmentHandler│     │EnrichmentModeManager│     │ Redis  │
└──┬───┘     └───────┬────────┘     └─────────┬──────────┘     └───┬────┘
   │                 │                         │                     │
   │ GET /enrichment/mode                      │                     │
   │────────────────▶│                         │                     │
   │                 │                         │                     │
   │                 │ GetModeWithSource(ctx)  │                     │
   │                 │────────────────────────▶│                     │
   │                 │                         │                     │
   │                 │                         │ RLock (50ns)        │
   │                 │                         │─────┐               │
   │                 │                         │     │ Cache stale   │
   │                 │                         │     │ (>30s old)    │
   │                 │                         │◀────┘               │
   │                 │                         │                     │
   │                 │                         │ Background refresh  │
   │                 │                         │ (goroutine)         │
   │                 │                         │───────────────────┐ │
   │                 │                         │                   │ │
   │                 │                         │ RefreshCache()    │ │
   │                 │                         │◀──────────────────┘ │
   │                 │                         │                     │
   │                 │                         │ GET "enrichment:mode"
   │                 │                         │────────────────────▶│
   │                 │                         │                     │
   │                 │                         │ {mode: "enriched"}  │
   │                 │                         │◀────────────────────│
   │                 │                         │                     │
   │                 │                         │ Lock (update cache) │
   │                 │                         │─────┐               │
   │                 │                         │     │ currentMode   │
   │                 │                         │     │ currentSource │
   │                 │                         │◀────┘               │
   │                 │                         │                     │
   │                 │  mode="enriched"        │                     │
   │                 │  source="memory"        │                     │
   │                 │◀────────────────────────│                     │
   │                 │                         │                     │
   │  200 OK         │                         │                     │
   │◀────────────────│                         │                     │
```

**Performance**:
- **Client request**: ~50-100ns (returns stale cache immediately)
- **Background refresh**: ~1-2ms (Redis network call, non-blocking)
- **Cache update**: ~50ns (write lock, update 3 fields)

**Key Design**: Client gets immediate response with potentially stale data (max 30s old), while cache refreshes in background.

---

### State Transition Diagram

```
┌────────────┐
│  Initial   │  (Default mode: "enriched", Source: "default")
└─────┬──────┘
      │
      │ RefreshCache() called
      │
      ▼
┌──────────────────────────────────────────────────────────────┐
│                   Try Redis                                  │
│  GET "enrichment:mode" → {mode: "transparent", ts: 1732...} │
└─────┬────────────────────────────────────────────────────────┘
      │
      │ Success: Redis has data
      │
      ▼
┌─────────────────────────────────────────────┐
│  Redis Mode                                 │
│  (Mode: "transparent", Source: "redis")    │
└─────┬───────────────────────────────────────┘
      │
      │ Redis failure OR Redis returns empty
      │
      ▼
┌──────────────────────────────────────────────────────────────┐
│                   Try ENV Variable                           │
│  os.Getenv("ENRICHMENT_MODE") → "enriched"                  │
└─────┬────────────────────────────────────────────────────────┘
      │
      │ Success: ENV var set and valid
      │
      ▼
┌─────────────────────────────────────────────┐
│  ENV Mode                                   │
│  (Mode: "enriched", Source: "env")         │
└─────┬───────────────────────────────────────┘
      │
      │ ENV var missing OR invalid
      │
      ▼
┌──────────────────────────────────────────────────────────────┐
│                   Default Mode                               │
│  (Mode: "enriched", Source: "default")                      │
└──────────────────────────────────────────────────────────────┘
```

**Fallback Priority**:
1. **Redis** (highest priority, persistent across pod restarts)
2. **ENV** (second priority, pod-specific config)
3. **Default** (fallback, hardcoded "enriched")

---

## ⚡ Performance Architecture

### Performance Targets

| Metric | Target | Actual (Goal) | Multiplier |
|--------|--------|---------------|------------|
| **p50 latency** | < 100ns | ~50ns | 2x better |
| **p95 latency** | < 1ms | ~500ns | 2x better |
| **p99 latency** | < 5ms | ~2ms | 2.5x better |
| **Throughput** | > 100K req/s | ~200K req/s | 2x better |
| **Memory per pod** | < 10MB | ~5MB | 2x better |
| **CPU per 100K req/s** | < 0.1 cores | ~0.05 cores | 2x better |

---

### Performance Optimization Techniques

#### 1. In-Memory Cache (Hot Path)
```go
// GetModeWithSource() - Ultra-fast path
func (m *enrichmentModeManager) GetModeWithSource(ctx context.Context) (EnrichmentMode, string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Zero allocations, ~50ns read
	return m.currentMode, m.currentSource, nil
}
```

**Benefits**:
- ✅ **50ns latency** (RWMutex RLock)
- ✅ **Zero allocations** (no heap escape)
- ✅ **Zero network calls** (no Redis)
- ✅ **Thread-safe** (RWMutex)

---

#### 2. Background Cache Refresh (Non-Blocking)
```go
// Auto-refresh if cache is stale
if time.Since(lastRefresh) > cacheRefreshInterval {
	// Background refresh (non-blocking)
	go func() {
		if err := m.RefreshCache(context.Background()); err != nil {
			m.logger.Debug("Background cache refresh failed", "error", err)
		}
	}()
}
```

**Benefits**:
- ✅ **Non-blocking** (client gets immediate response)
- ✅ **Eventual consistency** (cache updates in background)
- ✅ **Max staleness**: 30s (acceptable for mode switch)

---

#### 3. Efficient JSON Encoding
```go
// EnrichmentModeResponse - Small struct (16 bytes)
type EnrichmentModeResponse struct {
	Mode   string `json:"mode"`   // 8 bytes (pointer)
	Source string `json:"source"` // 8 bytes (pointer)
}

// JSON encoder (streaming, no intermediate buffer)
json.NewEncoder(w).Encode(response)
```

**Benefits**:
- ✅ **Small payload** (~40 bytes JSON)
- ✅ **Streaming encoder** (no buffering)
- ✅ **Minimal allocations** (2-3 allocs for strings)

---

#### 4. RWMutex Read Lock (Minimal Contention)
```go
// Read path (99.9% of requests)
m.mu.RLock()
defer m.mu.RUnlock()
// ... read currentMode, currentSource

// Write path (0.1% of requests, mode switch)
m.mu.Lock()
defer m.mu.Unlock()
// ... update currentMode, currentSource
```

**Benefits**:
- ✅ **Multiple concurrent readers** (no blocking)
- ✅ **Fast read lock** (~50ns, no contention)
- ✅ **Rare write lock** (only on mode switch)

---

### Benchmarking Strategy

#### Benchmark Suite
```go
// enrichment_bench_test.go

// BenchmarkGetMode_CacheHit tests cache hit scenario (hot path)
func BenchmarkGetMode_CacheHit(b *testing.B) {
	// Setup: In-memory cache with "enriched" mode
	manager := setupManager()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			mode, source, err := manager.GetModeWithSource(ctx)
			if err != nil {
				b.Fatal(err)
			}
			_ = mode
			_ = source
		}
	})
}

// BenchmarkGetMode_RedisFallback tests Redis fallback scenario
func BenchmarkGetMode_RedisFallback(b *testing.B) {
	// Setup: Expired cache, Redis available
	manager := setupManagerWithExpiredCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		mode, source, err := manager.GetModeWithSource(ctx)
		if err != nil {
			b.Fatal(err)
		}
		_ = mode
		_ = source
	}
}

// BenchmarkGetMode_Concurrent tests concurrent access
func BenchmarkGetMode_Concurrent(b *testing.B) {
	manager := setupManager()

	b.RunParallel(func(pb *testing.PB) {
		ctx := context.Background()
		for pb.Next() {
			go manager.GetModeWithSource(ctx)
		}
	})
}
```

**Target Results**:
```
BenchmarkGetMode_CacheHit-8        20000000        50.2 ns/op        0 B/op        0 allocs/op
BenchmarkGetMode_RedisFallback-8      10000      1200 ns/op      512 B/op        4 allocs/op
BenchmarkGetMode_Concurrent-8      10000000       100 ns/op        0 B/op        0 allocs/op
```

---

## 🚨 Error Handling Strategy

### Error Classification

```go
// ErrorType represents error categories
type ErrorType string

const (
	ErrorTypeRedisTimeout    ErrorType = "redis_timeout"
	ErrorTypeRedisConnection ErrorType = "redis_connection"
	ErrorTypeValidation      ErrorType = "validation"
	ErrorTypeInternal        ErrorType = "internal"
	ErrorTypeTimeout         ErrorType = "timeout"
)

// classifyError classifies error for metrics
func classifyError(err error) ErrorType {
	if err == nil {
		return ""
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return ErrorTypeTimeout
	case errors.Is(err, redis.ErrClosed):
		return ErrorTypeRedisConnection
	case errors.Is(err, redis.ErrPoolTimeout):
		return ErrorTypeRedisTimeout
	case errors.Is(err, ErrInvalidMode):
		return ErrorTypeValidation
	default:
		return ErrorTypeInternal
	}
}
```

---

### Error Response Format

```go
// ErrorResponse represents standardized error response
type ErrorResponse struct {
	Error     string `json:"error"`               // Human-readable message
	RequestID string `json:"request_id,omitempty"` // For correlation (optional)
}

// HTTP status code mapping
func (h *EnrichmentHandlers) handleError(w http.ResponseWriter, err error, requestID string) {
	errorType := classifyError(err)
	statusCode := http.StatusInternalServerError
	message := "Failed to get enrichment mode"

	switch errorType {
	case ErrorTypeTimeout:
		statusCode = http.StatusServiceUnavailable
		message = "Enrichment service timeout"
	case ErrorTypeValidation:
		statusCode = http.StatusBadRequest
		message = err.Error() // Specific validation message
	default:
		statusCode = http.StatusInternalServerError
		message = "Failed to get enrichment mode"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:     message,
		RequestID: requestID,
	})

	// Record error metrics
	h.metrics.Enrichment().RecordError(string(errorType))
}
```

---

### Graceful Degradation Strategy

```
┌─────────────────────────────────────────────────────────────────┐
│                     Graceful Degradation                        │
└─────────────────────────────────────────────────────────────────┘

1. Redis unavailable
   ├─▶ Fallback to ENV variable
   │   └─▶ If ENV set and valid → Use ENV
   │
   └─▶ Fallback to Default mode ("enriched")

2. Redis timeout (>5s)
   ├─▶ Cancel Redis query (context.WithTimeout)
   │
   └─▶ Return stale cache (max 30s old)
       └─▶ If no stale cache → Fallback to ENV/Default

3. Invalid mode in Redis
   ├─▶ Log warning
   │
   └─▶ Fallback to ENV/Default

4. Context cancellation (client disconnect)
   ├─▶ Return immediately (no processing)
   │
   └─▶ Log as INFO (not an error)
```

---

## 🔐 Security Design

### 1. Rate Limiting (Optional)

```go
// RateLimiter middleware (token bucket algorithm)
func RateLimiter(requestsPerMinute int) func(http.Handler) http.Handler {
	// Create token bucket (100 tokens, 1 token/600ms)
	limiter := rate.NewLimiter(rate.Limit(requestsPerMinute/60.0), 10)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow 1 token per request
			if !limiter.Allow() {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Rate limit exceeded. Try again later.",
				})
				return
			}

			// Set rate limit headers
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(requestsPerMinute))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(requestsPerMinute-1))

			next.ServeHTTP(w, r)
		})
	}
}
```

---

### 2. CORS Policy (Configurable)

```go
// CORS middleware
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin || allowedOrigin == "*" {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			// Handle preflight
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
```

---

### 3. Authentication (Optional, JWT)

```go
// JWTMiddleware validates JWT bearer tokens
func JWTMiddleware(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Missing Authorization header",
				})
				return
			}

			// Extract token
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Invalid Authorization format. Use: Bearer <token>",
				})
				return
			}

			token := parts[1]

			// Validate JWT (placeholder)
			claims, err := validateJWT(token, secret)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(ErrorResponse{
					Error: "Invalid token",
				})
				return
			}

			// Inject claims into context
			ctx := context.WithValue(r.Context(), "user", claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
```

---

## 📊 Observability Design

### Prometheus Metrics

```go
// Enrichment metrics interface
type EnrichmentMetrics interface {
	// RecordRequest records total requests
	RecordRequest(method string, status string)

	// RecordDuration records request duration
	RecordDuration(method string, duration time.Duration)

	// RecordCacheHit records cache hits by source
	RecordCacheHit(source string)

	// RecordError records errors by type
	RecordError(errorType string)

	// SetLastRequestTimestamp sets last request timestamp
	SetLastRequestTimestamp(timestamp time.Time)

	// SetConcurrentRequests sets concurrent requests gauge
	SetConcurrentRequests(count int64)
}

// Prometheus implementation
type enrichmentMetrics struct {
	requestsTotal           *prometheus.CounterVec
	requestDuration         *prometheus.HistogramVec
	cacheHitsTotal          *prometheus.CounterVec
	errorsTotal             *prometheus.CounterVec
	lastRequestTimestamp    prometheus.Gauge
	concurrentRequests      prometheus.Gauge
}

// Register metrics
func (m *enrichmentMetrics) Register() {
	m.requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_mode_requests_total",
			Help: "Total number of enrichment mode requests",
		},
		[]string{"method", "status"},
	)

	m.requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "enrichment_mode_request_duration_seconds",
			Help: "Request duration in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10), // 100µs to 51ms
		},
		[]string{"method"},
	)

	m.cacheHitsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_mode_cache_hits_total",
			Help: "Total number of cache hits by source",
		},
		[]string{"source"},
	)

	m.errorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "enrichment_mode_errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type"},
	)

	m.lastRequestTimestamp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "enrichment_mode_last_request_timestamp_seconds",
			Help: "Timestamp of last request (Unix seconds)",
		},
	)

	m.concurrentRequests = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "enrichment_mode_concurrent_requests",
			Help: "Number of concurrent requests",
		},
	)

	prometheus.MustRegister(
		m.requestsTotal,
		m.requestDuration,
		m.cacheHitsTotal,
		m.errorsTotal,
		m.lastRequestTimestamp,
		m.concurrentRequests,
	)
}
```

---

### PromQL Queries (Grafana Dashboard)

```promql
# Request rate (req/s)
rate(enrichment_mode_requests_total{method="GET",status="200"}[5m])

# Error rate (%)
100 * (
  rate(enrichment_mode_requests_total{status=~"5.."}[5m])
  /
  rate(enrichment_mode_requests_total[5m])
)

# p50 latency
histogram_quantile(0.50, rate(enrichment_mode_request_duration_seconds_bucket[5m]))

# p95 latency
histogram_quantile(0.95, rate(enrichment_mode_request_duration_seconds_bucket[5m]))

# p99 latency
histogram_quantile(0.99, rate(enrichment_mode_request_duration_seconds_bucket[5m]))

# Cache hit rate by source (%)
100 * (
  enrichment_mode_cache_hits_total{source="redis"}
  /
  sum(enrichment_mode_cache_hits_total)
)

# Concurrent requests (current)
enrichment_mode_concurrent_requests
```

---

## 🧪 Testing Strategy

### Test Pyramid

```
┌───────────────────────────────────────────────────────────────┐
│                        E2E Tests                              │
│                     (k6 load tests)                           │
│                        2 tests                                │
└───────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌───────────────────────────────────────────────────────────────┐
│                    Integration Tests                          │
│              (Real Redis, Real HTTP server)                   │
│                        5 tests                                │
└───────────────────────────────────────────────────────────────┘
                              ▲
                              │
┌───────────────────────────────────────────────────────────────┐
│                       Unit Tests                              │
│           (Mocked dependencies, table-driven)                 │
│                        20+ tests                              │
└───────────────────────────────────────────────────────────────┘
```

---

### 1. Unit Tests (enrichment_test.go)

```go
// TestEnrichmentHandlers_GetMode tests GET /enrichment/mode endpoint
func TestEnrichmentHandlers_GetMode(t *testing.T) {
	tests := []struct {
		name                  string
		mockGetModeWithSource func(ctx context.Context) (services.EnrichmentMode, string, error)
		expectedStatus        int
		expectedMode          string
		expectedSource        string
		expectError           bool
	}{
		{
			name: "returns enriched mode from redis",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeEnriched, "redis", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "enriched",
			expectedSource: "redis",
			expectError:    false,
		},
		{
			name: "returns transparent mode from env",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparent, "env", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "transparent",
			expectedSource: "env",
			expectError:    false,
		},
		{
			name: "returns transparent_with_recommendations mode from memory",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return services.EnrichmentModeTransparentWithRecommendations, "memory", nil
			},
			expectedStatus: http.StatusOK,
			expectedMode:   "transparent_with_recommendations",
			expectedSource: "memory",
			expectError:    false,
		},
		{
			name: "handles service error gracefully",
			mockGetModeWithSource: func(ctx context.Context) (services.EnrichmentMode, string, error) {
				return "", "", errors.New("service failure")
			},
			expectedStatus: http.StatusInternalServerError,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock manager
			mockManager := &mockEnrichmentManager{
				getModeWithSource: tt.mockGetModeWithSource,
			}

			// Create handlers
			handlers := NewEnrichmentHandlers(mockManager, slog.Default(), nil)

			// Create request
			req, err := http.NewRequest("GET", "/enrichment/mode", nil)
			require.NoError(t, err)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			handlers.GetMode(rr, req)

			// Assert status code
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Assert response body
			if tt.expectError {
				var errorResp ErrorResponse
				err := json.Unmarshal(rr.Body.Bytes(), &errorResp)
				require.NoError(t, err)
				assert.NotEmpty(t, errorResp.Error)
			} else {
				var response EnrichmentModeResponse
				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMode, response.Mode)
				assert.Equal(t, tt.expectedSource, response.Source)
			}
		})
	}
}
```

---

### 2. Integration Tests (enrichment_integration_test.go)

```go
// TestEnrichmentEndpoint_Integration tests full integration with real Redis
func TestEnrichmentEndpoint_Integration(t *testing.T) {
	// Skip if no Redis available
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup real Redis
	redisClient := setupRedis(t)
	defer redisClient.Close()

	// Setup real cache
	cache := cache.NewRedisCache(redisClient)

	// Setup real manager
	manager := services.NewEnrichmentModeManager(cache, slog.Default(), nil)

	// Setup real handler
	handler := handlers.NewEnrichmentHandlers(manager, slog.Default(), nil)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.GetMode))
	defer server.Close()

	t.Run("returns mode from Redis", func(t *testing.T) {
		// Set mode in Redis
		ctx := context.Background()
		err := manager.SetMode(ctx, services.EnrichmentModeTransparent)
		require.NoError(t, err)

		// Make HTTP request
		resp, err := http.Get(server.URL + "/enrichment/mode")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response handlers.EnrichmentModeResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		assert.Equal(t, "transparent", response.Mode)
		assert.Equal(t, "redis", response.Source)
	})

	t.Run("handles Redis unavailable", func(t *testing.T) {
		// Stop Redis
		redisClient.Close()

		// Make HTTP request (should fallback to ENV/default)
		resp, err := http.Get(server.URL + "/enrichment/mode")
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert response (should still work with fallback)
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response handlers.EnrichmentModeResponse
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)

		// Should return ENV or default mode
		assert.Contains(t, []string{"enriched", "transparent"}, response.Mode)
		assert.Contains(t, []string{"env", "default"}, response.Source)
	})
}
```

---

### 3. Load Tests (k6/enrichment_mode_get.js)

```javascript
import http from 'k6/http';
import { check, sleep } from 'k6';

// Test configuration
export let options = {
	stages: [
		{ duration: '10s', target: 100 },   // Ramp-up to 100 users
		{ duration: '30s', target: 1000 },  // Ramp-up to 1000 users
		{ duration: '60s', target: 1000 },  // Stay at 1000 users
		{ duration: '10s', target: 0 },     // Ramp-down
	],
	thresholds: {
		'http_req_duration': ['p(50)<100', 'p(95)<1000', 'p(99)<5000'],
		'http_req_failed': ['rate<0.01'], // Error rate < 1%
	},
};

// Test scenario
export default function() {
	// GET /enrichment/mode
	let response = http.get('http://localhost:8080/enrichment/mode');

	// Assertions
	check(response, {
		'status is 200': (r) => r.status === 200,
		'response has mode': (r) => JSON.parse(r.body).mode !== undefined,
		'response has source': (r) => JSON.parse(r.body).source !== undefined,
		'response time < 100ms': (r) => r.timings.duration < 100,
	});

	// Simulate realistic traffic (1 request per second per user)
	sleep(1);
}
```

**Expected Results**:
```
✓ http_req_duration...........: avg=50ms   p50=20ms  p95=100ms  p99=500ms  max=2s
✓ http_req_failed.............: 0.01%
✓ http_reqs...................: 100000 (1666/s)
```

---

## 🚀 Deployment Architecture

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-history
spec:
  replicas: 2
  selector:
    matchLabels:
      app: alert-history
  template:
    metadata:
      labels:
        app: alert-history
    spec:
      containers:
      - name: alert-history
        image: alert-history:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: ENRICHMENT_MODE
          value: "enriched"
        resources:
          requests:
            memory: "64Mi"
            cpu: "50m"
          limits:
            memory: "128Mi"
            cpu: "200m"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: alert-history-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: alert-history
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 80
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

---

## 🔄 Migration & Rollback

### Zero-Downtime Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1        # Create 1 extra pod during update
      maxUnavailable: 0  # Never have 0 healthy pods
```

**Deployment Steps**:
1. Create new pod with updated code
2. Wait for readiness probe to pass
3. Add new pod to service
4. Remove old pod from service
5. Terminate old pod
6. Repeat for remaining pods

---

### Rollback Strategy

```bash
# Rollback to previous version (<60s)
kubectl rollout undo deployment/alert-history

# Check rollout status
kubectl rollout status deployment/alert-history

# Verify GET /enrichment/mode still works
curl http://alert-history/enrichment/mode
```

---

## 📚 Appendix

### A. Performance Benchmarks (Target Results)

```
goos: darwin
goarch: arm64
pkg: github.com/vitaliisemenov/alert-history/cmd/server/handlers

BenchmarkGetMode_CacheHit-8                20000000        50.2 ns/op        0 B/op        0 allocs/op
BenchmarkGetMode_RedisFallback-8              10000      1200 ns/op      512 B/op        4 allocs/op
BenchmarkGetMode_Concurrent-8              10000000       100 ns/op        0 B/op        0 allocs/op
BenchmarkGetModeWithSource-8               20000000        45.8 ns/op        0 B/op        0 allocs/op
BenchmarkJSONEncode-8                       5000000       350 ns/op      128 B/op        2 allocs/op
BenchmarkRWMutexRLock-8                   100000000        10.5 ns/op        0 B/op        0 allocs/op
BenchmarkRWMutexLock-8                     50000000        25.3 ns/op        0 B/op        0 allocs/op

PASS
ok   github.com/vitaliisemenov/alert-history/cmd/server/handlers 15.234s
```

---

### B. API Examples

#### curl
```bash
# GET current mode
curl -X GET http://localhost:8080/enrichment/mode

# GET with request ID
curl -X GET \
  -H "X-Request-ID: 550e8400-e29b-41d4-a716-446655440000" \
  http://localhost:8080/enrichment/mode

# GET with JWT authentication (if enabled)
curl -X GET \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  http://localhost:8080/enrichment/mode
```

#### Go Client
```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type EnrichmentModeResponse struct {
	Mode   string `json:"mode"`
	Source string `json:"source"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/enrichment/mode", nil)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("unexpected status: %d", resp.StatusCode))
	}

	var response EnrichmentModeResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		panic(err)
	}

	fmt.Printf("Mode: %s, Source: %s\n", response.Mode, response.Source)
}
```

---

### C. Monitoring Checklist

**Metrics to Watch**:
- ✅ Request rate (enrichment_mode_requests_total)
- ✅ Error rate (enrichment_mode_requests_total{status=~"5.."})
- ✅ p50/p95/p99 latency (enrichment_mode_request_duration_seconds)
- ✅ Cache hit rate by source (enrichment_mode_cache_hits_total)
- ✅ Concurrent requests (enrichment_mode_concurrent_requests)
- ✅ Redis errors (enrichment_mode_errors_total{type="redis_timeout"})

**Alerts**:
- 🚨 Error rate > 1% for 5 minutes
- 🚨 p99 latency > 10ms for 5 minutes
- 🚨 Redis unavailable for 1 minute
- 🚨 No requests for 5 minutes (service down?)

---

### D. Troubleshooting Guide

| Symptom | Cause | Solution |
|---------|-------|----------|
| 500 errors | Service failure | Check logs, verify Redis connection |
| Slow responses (>10ms) | Cache miss, Redis slow | Check Redis latency, scale pods |
| No requests | Service down | Check pod status, readiness probe |
| Redis timeout | Network partition | Check Redis connectivity, increase timeout |
| Memory leak | Bug in cache | Monitor memory, restart pods |

---

## 📝 Conclusion

This design document provides a **comprehensive technical blueprint** for implementing the GET /enrichment/mode endpoint at **150% quality (Grade A+ EXCELLENT)**. The design prioritizes:

1. ✅ **Ultra-fast performance** (< 100ns p50 latency)
2. ✅ **High availability** (99.99% uptime, graceful degradation)
3. ✅ **Horizontal scalability** (100K+ req/s, 2-10 replicas)
4. ✅ **Comprehensive observability** (metrics, logs, tracing)
5. ✅ **Production-ready security** (rate limiting, CORS, optional auth)
6. ✅ **Extensive testing** (unit, integration, benchmarks, load tests)

**Next Steps**:
1. ⏳ Create tasks.md (implementation roadmap)
2. ⏳ Begin Phase 2 (Performance Enhancement - Benchmarks)
3. ⏳ Implement advanced features (cache headers, rate limiting)

---

**Document Version**: 1.0
**Author**: AI Assistant
**Review Status**: Draft
**Approval Status**: Pending Review
**Last Updated**: 2025-11-28
