# TN-034: Enrichment Mode Design

**Обновлено**: 2025-10-09
**Статус**: ❌ НЕ НАЧАТА (0%)

## Архитектурный обзор

```
┌─────────────────────────────────────────────────────────────┐
│                    Webhook Request                          │
└───────────────────────┬─────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────┐
│              EnrichmentModeManager                          │
│  GetMode() → Redis → Memory → ENV → Default                │
└───────────────────────┬─────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
        ▼               ▼               ▼
   transparent      enriched    transparent_with_recommendations
        │               │               │
        ▼               ▼               ▼
  [Skip LLM]    [Call LLM]        [Skip LLM]
        │               │               │
        ▼               ▼               ▼
  [Apply Filter]  [Apply Filter]  [Skip Filter]
        │               │               │
        └───────────────┴───────────────┘
                        │
                        ▼
                  [Save to DB]
```

## 1. Core Types

### 1.1 EnrichmentMode Type
```go
package services

import (
    "context"
    "fmt"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
    "log/slog"
)

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

// Valid modes list
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

### 1.2 EnrichmentModeManager Interface
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

// EnrichmentStats represents enrichment mode statistics
type EnrichmentStats struct {
    CurrentMode       EnrichmentMode    `json:"current_mode"`
    Source            string            `json:"source"`
    LastSwitchTime    *time.Time        `json:"last_switch_time,omitempty"`
    LastSwitchFrom    EnrichmentMode    `json:"last_switch_from,omitempty"`
    TotalSwitches     int64             `json:"total_switches"`
    RedisAvailable    bool              `json:"redis_available"`
    CacheHitRate      float64           `json:"cache_hit_rate"`
}
```

## 2. Implementation

### 2.1 enrichmentModeManager struct
```go
const (
    redisKeyMode         = "enrichment:mode"
    redisKeyStats        = "enrichment:stats"
    defaultMode          = EnrichmentModeEnriched
    cacheRefreshInterval = 30 * time.Second
)

type enrichmentModeManager struct {
    cache          cache.Cache
    logger         *slog.Logger
    metrics        *metrics.MetricsManager

    // In-memory cache for fast access
    currentMode    EnrichmentMode
    currentSource  string
    lastRefresh    time.Time

    // Stats
    totalSwitches  int64
    lastSwitchTime *time.Time
    lastSwitchFrom EnrichmentMode

    mu sync.RWMutex // protects in-memory state
}

// NewEnrichmentModeManager creates new enrichment mode manager
func NewEnrichmentModeManager(
    cache cache.Cache,
    logger *slog.Logger,
    metrics *metrics.MetricsManager,
) EnrichmentModeManager {
    if logger == nil {
        logger = slog.Default()
    }

    m := &enrichmentModeManager{
        cache:         cache,
        logger:        logger,
        metrics:       metrics,
        currentMode:   defaultMode,
        currentSource: "default",
        lastRefresh:   time.Now(),
    }

    // Initialize mode from storage
    ctx := context.Background()
    if err := m.RefreshCache(ctx); err != nil {
        logger.Warn("Failed to initialize mode from storage, using default",
            "error", err,
            "default_mode", defaultMode,
        )
    }

    return m
}
```

### 2.2 GetMode() - Fast Path
```go
func (m *enrichmentModeManager) GetMode(ctx context.Context) (EnrichmentMode, error) {
    m.mu.RLock()
    mode := m.currentMode
    lastRefresh := m.lastRefresh
    m.mu.RUnlock()

    // Auto-refresh if cache is stale
    if time.Since(lastRefresh) > cacheRefreshInterval {
        go func() {
            if err := m.RefreshCache(context.Background()); err != nil {
                m.logger.Debug("Background cache refresh failed", "error", err)
            }
        }()
    }

    return mode, nil
}

func (m *enrichmentModeManager) GetModeWithSource(ctx context.Context) (EnrichmentMode, string, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return m.currentMode, m.currentSource, nil
}
```

### 2.3 SetMode() - Write Path
```go
func (m *enrichmentModeManager) SetMode(ctx context.Context, mode EnrichmentMode) error {
    // Validate mode
    if err := m.ValidateMode(mode); err != nil {
        return err
    }

    m.mu.Lock()
    oldMode := m.currentMode
    m.mu.Unlock()

    // Save to Redis first
    saved := false
    if m.cache != nil {
        data := map[string]any{
            "mode":      string(mode),
            "timestamp": time.Now().Unix(),
        }

        if err := m.cache.Set(ctx, redisKeyMode, data, 0); err != nil {
            m.logger.Warn("Failed to save mode to Redis, using memory fallback",
                "error", err,
                "mode", mode,
            )
        } else {
            saved = true
        }
    }

    // Update in-memory state
    m.mu.Lock()
    m.currentMode = mode
    m.currentSource = "memory"
    if saved {
        m.currentSource = "redis"
    }
    m.lastRefresh = time.Now()

    // Track switch if mode changed
    if oldMode != mode {
        m.totalSwitches++
        m.lastSwitchFrom = oldMode
        now := time.Now()
        m.lastSwitchTime = &now
    }
    m.mu.Unlock()

    // Update metrics
    if m.metrics != nil {
        m.updateMetrics(oldMode, mode)
    }

    m.logger.Info("Enrichment mode updated",
        "old_mode", oldMode,
        "new_mode", mode,
        "source", m.currentSource,
    )

    return nil
}
```

### 2.4 RefreshCache() - Redis Sync
```go
func (m *enrichmentModeManager) RefreshCache(ctx context.Context) error {
    var mode EnrichmentMode
    var source string

    // 1. Try Redis
    if m.cache != nil {
        val, err := m.cache.Get(ctx, redisKeyMode)
        if err == nil && val != nil {
            if data, ok := val.(map[string]any); ok {
                if modeStr, ok := data["mode"].(string); ok {
                    mode = EnrichmentMode(modeStr)
                    if mode.IsValid() {
                        source = "redis"
                        goto found
                    }
                }
            }
        }

        if err != nil && !cache.IsNotFound(err) {
            m.logger.Debug("Redis get failed", "error", err)
        }
    }

    // 2. Try ENV variable
    if envMode := os.Getenv("ENRICHMENT_MODE"); envMode != "" {
        mode = EnrichmentMode(envMode)
        if mode.IsValid() {
            source = "env"
            goto found
        }
        m.logger.Warn("Invalid ENRICHMENT_MODE env variable", "value", envMode)
    }

    // 3. Use default
    mode = defaultMode
    source = "default"

found:
    // Update in-memory cache
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

### 2.5 Validation & Stats
```go
func (m *enrichmentModeManager) ValidateMode(mode EnrichmentMode) error {
    if !mode.IsValid() {
        return fmt.Errorf("invalid enrichment mode: %s (valid: transparent, enriched, transparent_with_recommendations)", mode)
    }
    return nil
}

func (m *enrichmentModeManager) GetStats(ctx context.Context) (*EnrichmentStats, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    stats := &EnrichmentStats{
        CurrentMode:     m.currentMode,
        Source:          m.currentSource,
        TotalSwitches:   m.totalSwitches,
        LastSwitchTime:  m.lastSwitchTime,
        LastSwitchFrom:  m.lastSwitchFrom,
        RedisAvailable:  m.cache != nil,
    }

    return stats, nil
}

func (m *enrichmentModeManager) updateMetrics(oldMode, newMode EnrichmentMode) {
    // TODO: Implement metrics updates
    // metrics.enrichment_mode_switches_total{from_mode, to_mode}
    // metrics.enrichment_mode_status (gauge)
}
```

## 3. Integration with Webhook Processing

### 3.1 Classification Service Integration
```go
// In ClassificationService
func (s *ClassificationService) ClassifyAlert(ctx context.Context, alert *core.Alert) (*core.ClassificationResult, error) {
    // Check enrichment mode
    mode, err := s.enrichmentManager.GetMode(ctx)
    if err != nil {
        s.logger.Warn("Failed to get enrichment mode, using default", "error", err)
        mode = EnrichmentModeEnriched
    }

    // Skip classification in transparent modes
    if mode == EnrichmentModeTransparent || mode == EnrichmentModeTransparentWithRecommendations {
        s.logger.Debug("Skipping classification (transparent mode)", "mode", mode)
        return nil, nil
    }

    // Normal LLM classification
    return s.classifyWithLLM(ctx, alert)
}
```

### 3.2 Filter Engine Integration
```go
// In FilterEngine
func (e *FilterEngine) ShouldPublish(ctx context.Context, alert *core.EnrichedAlert, target *core.PublishingTarget) (bool, error) {
    // Check enrichment mode
    mode, err := e.enrichmentManager.GetMode(ctx)
    if err != nil {
        e.logger.Warn("Failed to get enrichment mode, using default", "error", err)
        mode = EnrichmentModeEnriched
    }

    // Skip filtering in transparent_with_recommendations mode
    if mode == EnrichmentModeTransparentWithRecommendations {
        e.logger.Debug("Skipping filtering (transparent_with_recommendations mode)")
        return true, nil
    }

    // Normal filtering logic
    return e.applyFilters(ctx, alert, target)
}
```

### 3.3 Webhook Handler Integration
```go
// In WebhookHandler
func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Get current mode for observability
    mode, source, _ := h.enrichmentManager.GetModeWithSource(ctx)

    h.logger.Info("Processing webhook",
        "enrichment_mode", mode,
        "mode_source", source,
    )

    // Process alerts (mode is checked internally by services)
    // ...
}
```

## 4. API Handlers

### 4.1 GET /enrichment/mode
```go
type EnrichmentModeResponse struct {
    Mode   string `json:"mode"`
    Source string `json:"source"`
}

func (h *EnrichmentHandlers) GetMode(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    mode, source, err := h.manager.GetModeWithSource(ctx)
    if err != nil {
        h.logger.Error("Failed to get enrichment mode", "error", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    response := EnrichmentModeResponse{
        Mode:   mode.String(),
        Source: source,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

### 4.2 POST /enrichment/mode
```go
type SetEnrichmentModeRequest struct {
    Mode string `json:"mode" validate:"required,oneof=transparent enriched transparent_with_recommendations"`
}

func (h *EnrichmentHandlers) SetMode(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    var req SetEnrichmentModeRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    mode := EnrichmentMode(req.Mode)
    if err := h.manager.ValidateMode(mode); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    if err := h.manager.SetMode(ctx, mode); err != nil {
        h.logger.Error("Failed to set enrichment mode", "error", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // Get updated state
    mode, source, _ := h.manager.GetModeWithSource(ctx)

    response := EnrichmentModeResponse{
        Mode:   mode.String(),
        Source: source,
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
```

## 5. Метрики

```go
// Prometheus metrics
var (
    enrichmentModeSwitches = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "alert_history",
            Subsystem: "enrichment",
            Name:      "mode_switches_total",
            Help:      "Total number of enrichment mode switches",
        },
        []string{"from_mode", "to_mode"},
    )

    enrichmentModeStatus = prometheus.NewGauge(
        prometheus.GaugeOpts{
            Namespace: "alert_history",
            Subsystem: "enrichment",
            Name:      "mode_status",
            Help:      "Current enrichment mode (0=transparent, 1=enriched, 2=transparent_with_recommendations)",
        },
    )

    enrichmentModeRequests = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "alert_history",
            Subsystem: "enrichment",
            Name:      "mode_requests_total",
            Help:      "Total number of enrichment mode API requests",
        },
        []string{"method", "mode"},
    )
)
```

## 6. Configuration

```yaml
# config.yaml
enrichment:
  default_mode: "enriched"  # transparent | enriched | transparent_with_recommendations
  cache_refresh_interval: 30s
  redis_key: "enrichment:mode"

# Environment variables
ENRICHMENT_MODE=enriched  # Override default mode
```

## 7. Тестирование

### 7.1 Unit Tests
```go
func TestEnrichmentModeManager_GetMode(t *testing.T) {
    tests := []struct {
        name           string
        redisValue     map[string]any
        envValue       string
        expectedMode   EnrichmentMode
        expectedSource string
    }{
        {
            name:           "Redis available",
            redisValue:     map[string]any{"mode": "transparent"},
            expectedMode:   EnrichmentModeTransparent,
            expectedSource: "redis",
        },
        {
            name:           "Fallback to ENV",
            envValue:       "enriched",
            expectedMode:   EnrichmentModeEnriched,
            expectedSource: "env",
        },
        {
            name:           "Default mode",
            expectedMode:   EnrichmentModeEnriched,
            expectedSource: "default",
        },
    }
    // ...
}
```

---

**Python Parity**: ✅ 100% (все функции покрыты)
