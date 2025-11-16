# TN-65: Дизайн GET /metrics Endpoint

**Дата:** 2025-11-16
**Версия:** 1.0
**Статус:** DRAFT
**Целевой показатель качества:** 150%

## 🎯 Архитектурное решение

### Общая концепция

Endpoint `/metrics` должен быть высокопроизводительным, надёжным и полностью интегрированным с существующей системой метрик. Реализация должна использовать стандартный `promhttp.Handler()` с дополнительными улучшениями для enterprise-среды.

### Архитектурная диаграмма

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request                              │
│                  GET /metrics                                │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              MetricsEndpointHandler                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 1. Request Validation                                  │  │
│  │    - Method check (GET only)                           │  │
│  │    - Path validation                                   │  │
│  │    - Rate limiting (optional)                          │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 2. Metrics Collection                                  │  │
│  │    - Gather from MetricsRegistry                      │  │
│  │    - Gather from HTTPMetrics                          │  │
│  │    - Gather from Go runtime (optional)                │  │
│  │    - Error handling (graceful degradation)             │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 3. Format & Serialization                             │  │
│  │    - Prometheus text format v0.0.4                    │  │
│  │    - Content-Type header                             │  │
│  │    - Charset: utf-8                                  │  │
│  └──────────────────────────────────────────────────────┘  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │ 4. Response                                           │  │
│  │    - HTTP 200 OK                                      │  │
│    │    - Metrics body                                   │  │
│    │    - Performance metrics (self-observability)        │  │
│  └──────────────────────────────────────────────────────┘  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Prometheus Client                               │
│         (promhttp.Handler)                                   │
│  - Default Prometheus registry                              │
│  - Custom registries (MetricsRegistry)                      │
│  - Go runtime metrics (optional)                            │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Metrics Sources                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ MetricsRegistry│  │ HTTPMetrics  │  │ Go Runtime   │      │
│  │ - Business    │  │ - Requests   │  │ - GC         │      │
│  │ - Technical   │  │ - Duration   │  │ - Memory     │      │
│  │ - Infra      │  │ - Size       │  │ - Goroutines  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## 📐 Детальный дизайн

### 1. MetricsEndpointHandler

Основной handler для endpoint `/metrics` с расширенной функциональностью.

#### Структура

```go
// pkg/metrics/endpoint.go

package metrics

import (
    "context"
    "net/http"
    "sync"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsEndpointHandler handles GET /metrics requests.
// Provides enterprise-grade features: performance optimization, error handling,
// self-observability, and security.
type MetricsEndpointHandler struct {
    // Core handler
    handler http.Handler

    // Configuration
    config EndpointConfig

    // Self-observability metrics
    requestsTotal    prometheus.Counter
    requestDuration  prometheus.Histogram
    requestErrors    prometheus.Counter
    requestSize      prometheus.Histogram
    activeRequests   prometheus.Gauge

    // Error handling
    errorHandler ErrorHandler

    // Performance optimization
    gatherer prometheus.Gatherer
    registry *prometheus.Registry

    // Thread safety
    mu sync.RWMutex
}

// EndpointConfig holds configuration for the metrics endpoint.
type EndpointConfig struct {
    // Path for the metrics endpoint (default: "/metrics")
    Path string

    // Enable Go runtime metrics
    EnableGoRuntime bool

    // Enable process metrics
    EnableProcess bool

    // Timeout for gathering metrics
    GatherTimeout time.Duration

    // Maximum response size (0 = unlimited)
    MaxResponseSize int64

    // Enable self-observability metrics
    EnableSelfMetrics bool

    // Custom gatherer (optional)
    CustomGatherer prometheus.Gatherer
}

// DefaultEndpointConfig returns default configuration.
func DefaultEndpointConfig() EndpointConfig {
    return EndpointConfig{
        Path:              "/metrics",
        EnableGoRuntime:   false, // Disabled by default for performance
        EnableProcess:     false, // Disabled by default for security
        GatherTimeout:     5 * time.Second,
        MaxResponseSize:   10 * 1024 * 1024, // 10MB
        EnableSelfMetrics: true,
    }
}
```

#### Методы

```go
// NewMetricsEndpointHandler creates a new metrics endpoint handler.
func NewMetricsEndpointHandler(config EndpointConfig, registry *MetricsRegistry) (*MetricsEndpointHandler, error) {
    // Create Prometheus registry
    promRegistry := prometheus.NewRegistry()

    // Register default metrics
    if config.EnableGoRuntime {
        promRegistry.MustRegister(prometheus.NewGoCollector())
    }
    if config.EnableProcess {
        promRegistry.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
    }

    // Register MetricsRegistry metrics
    if registry != nil {
        // Register all metrics from MetricsRegistry
        // Business metrics
        if business := registry.Business(); business != nil {
            promRegistry.MustRegister(business.AlertsProcessedTotal)
            // ... register all business metrics
        }
        // Technical metrics
        if technical := registry.Technical(); technical != nil {
            // ... register all technical metrics
        }
        // Infra metrics
        if infra := registry.Infra(); infra != nil {
            // ... register all infra metrics
        }
    }

    // Create handler
    handler := &MetricsEndpointHandler{
        config:   config,
        gatherer: promRegistry,
        registry: promRegistry,
        handler:  promhttp.HandlerFor(promRegistry, promhttp.HandlerOpts{}),
    }

    // Initialize self-observability metrics
    if config.EnableSelfMetrics {
        handler.initSelfMetrics()
    }

    return handler, nil
}

// ServeHTTP implements http.Handler interface.
func (h *MetricsEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Validate request method
    if r.Method != http.MethodGet {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    // Validate path
    if r.URL.Path != h.config.Path {
        http.NotFound(w, r)
        return
    }

    start := time.Now()
    h.mu.RLock()
    active := h.activeRequests
    h.mu.RUnlock()

    if active != nil {
        active.Inc()
        defer active.Dec()
    }

    // Set headers
    w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
    w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

    // Gather metrics with timeout
    ctx, cancel := context.WithTimeout(r.Context(), h.config.GatherTimeout)
    defer cancel()

    // Gather metrics
    metricFamilies, err := h.gatherMetrics(ctx)
    if err != nil {
        h.handleError(w, r, err)
        return
    }

    // Write response
    if err := h.writeResponse(w, metricFamilies); err != nil {
        h.handleError(w, r, err)
        return
    }

    // Record metrics
    duration := time.Since(start)
    h.recordMetrics(r, duration, http.StatusOK, 0)
}

// gatherMetrics gathers all metrics from registered collectors.
func (h *MetricsEndpointHandler) gatherMetrics(ctx context.Context) ([]*dto.MetricFamily, error) {
    // Use context for timeout
    done := make(chan struct{})
    var families []*dto.MetricFamily
    var gatherErr error

    go func() {
        defer close(done)
        families, gatherErr = h.gatherer.Gather()
    }()

    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-done:
        return families, gatherErr
    }
}

// writeResponse writes metrics in Prometheus text format.
func (h *MetricsEndpointHandler) writeResponse(w http.ResponseWriter, families []*dto.MetricFamily) error {
    // Use promhttp to format metrics
    // This ensures compatibility with Prometheus format
    encoder := expfmt.NewEncoder(w, expfmt.FmtText)

    for _, family := range families {
        if err := encoder.Encode(family); err != nil {
            return fmt.Errorf("failed to encode metric family: %w", err)
        }
    }

    return nil
}

// handleError handles errors gracefully.
func (h *MetricsEndpointHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
    // Log error
    h.errorHandler.LogError(r.Context(), err)

    // Record error metric
    if h.requestErrors != nil {
        h.requestErrors.Inc()
    }

    // Try to return partial metrics if possible
    // Otherwise return 500
    http.Error(w, "Internal server error", http.StatusInternalServerError)
}

// recordMetrics records self-observability metrics.
func (h *MetricsEndpointHandler) recordMetrics(r *http.Request, duration time.Duration, status int, size int64) {
    if h.requestsTotal != nil {
        h.requestsTotal.Inc()
    }
    if h.requestDuration != nil {
        h.requestDuration.Observe(duration.Seconds())
    }
    if h.requestSize != nil && size > 0 {
        h.requestSize.Observe(float64(size))
    }
}

// initSelfMetrics initializes self-observability metrics.
func (h *MetricsEndpointHandler) initSelfMetrics() {
    namespace := "alert_history"
    subsystem := "metrics_endpoint"

    h.requestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: namespace,
        Subsystem: subsystem,
        Name:      "requests_total",
        Help:      "Total number of requests to /metrics endpoint",
    })

    h.requestDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
        Namespace: namespace,
        Subsystem: subsystem,
        Name:      "request_duration_seconds",
        Help:      "Duration of /metrics endpoint requests",
        Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
    })

    h.requestErrors = prometheus.NewCounter(prometheus.CounterOpts{
        Namespace: namespace,
        Subsystem: subsystem,
        Name:      "errors_total",
        Help:      "Total number of errors in /metrics endpoint",
    })

    h.requestSize = prometheus.NewHistogram(prometheus.HistogramOpts{
        Namespace: namespace,
        Subsystem: subsystem,
        Name:      "response_size_bytes",
        Help:      "Size of /metrics endpoint responses",
        Buckets:   prometheus.ExponentialBuckets(1024, 2, 10), // 1KB to 1MB
    })

    h.activeRequests = prometheus.NewGauge(prometheus.GaugeOpts{
        Namespace: namespace,
        Subsystem: subsystem,
        Name:      "active_requests",
        Help:      "Number of active requests to /metrics endpoint",
    })

    // Register self-metrics
    h.registry.MustRegister(
        h.requestsTotal,
        h.requestDuration,
        h.requestErrors,
        h.requestSize,
        h.activeRequests,
    )
}
```

### 2. Интеграция с существующей системой

#### Интеграция с MetricsRegistry

```go
// pkg/metrics/endpoint.go (continued)

// RegisterMetricsRegistry registers all metrics from MetricsRegistry.
func (h *MetricsEndpointHandler) RegisterMetricsRegistry(registry *MetricsRegistry) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    // Register Business metrics
    if business := registry.Business(); business != nil {
        if err := h.registry.Register(business.AlertsProcessedTotal); err != nil {
            return fmt.Errorf("failed to register business metrics: %w", err)
        }
        // ... register all business metrics
    }

    // Register Technical metrics
    if technical := registry.Technical(); technical != nil {
        // ... register all technical metrics
    }

    // Register Infra metrics
    if infra := registry.Infra(); infra != nil {
        // ... register all infra metrics
    }

    return nil
}
```

#### Интеграция с HTTPMetrics

```go
// pkg/metrics/endpoint.go (continued)

// RegisterHTTPMetrics registers HTTP metrics from MetricsManager.
func (h *MetricsEndpointHandler) RegisterHTTPMetrics(httpMetrics *HTTPMetrics) error {
    h.mu.Lock()
    defer h.mu.Unlock()

    // HTTPMetrics uses promauto, so metrics are already registered
    // We just need to ensure they're in our registry
    // This is handled by using the default registry or custom gatherer

    return nil
}
```

### 3. Обработка ошибок

#### ErrorHandler Interface

```go
// pkg/metrics/endpoint.go (continued)

// ErrorHandler handles errors in metrics endpoint.
type ErrorHandler interface {
    LogError(ctx context.Context, err error)
    ShouldReturnPartialMetrics(err error) bool
}

// DefaultErrorHandler is the default error handler.
type DefaultErrorHandler struct {
    logger Logger
}

// LogError logs the error.
func (h *DefaultErrorHandler) LogError(ctx context.Context, err error) {
    if h.logger != nil {
        h.logger.Error("metrics endpoint error", "error", err)
    }
}

// ShouldReturnPartialMetrics determines if partial metrics should be returned.
func (h *DefaultErrorHandler) ShouldReturnPartialMetrics(err error) bool {
    // Return partial metrics for context timeout, but not for other errors
    return errors.Is(err, context.DeadlineExceeded)
}
```

### 4. Оптимизация производительности

#### Кэширование метрик (опционально)

```go
// pkg/metrics/endpoint.go (continued)

// CachedMetricsEndpointHandler extends MetricsEndpointHandler with caching.
type CachedMetricsEndpointHandler struct {
    *MetricsEndpointHandler

    cache      *sync.Map // cache of serialized metrics
    cacheTTL   time.Duration
    lastUpdate time.Time
    mu         sync.RWMutex
}

// NewCachedMetricsEndpointHandler creates a cached handler.
func NewCachedMetricsEndpointHandler(config EndpointConfig, registry *MetricsRegistry, cacheTTL time.Duration) (*CachedMetricsEndpointHandler, error) {
    base, err := NewMetricsEndpointHandler(config, registry)
    if err != nil {
        return nil, err
    }

    return &CachedMetricsEndpointHandler{
        MetricsEndpointHandler: base,
        cacheTTL:               cacheTTL,
    }, nil
}

// ServeHTTP implements http.Handler with caching.
func (h *CachedMetricsEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Check cache
    h.mu.RLock()
    if time.Since(h.lastUpdate) < h.cacheTTL {
        if cached, ok := h.cache.Load("metrics"); ok {
            w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
            w.Write(cached.([]byte))
            h.mu.RUnlock()
            return
        }
    }
    h.mu.RUnlock()

    // Gather and cache
    h.mu.Lock()
    defer h.mu.Unlock()

    // Double-check after acquiring lock
    if time.Since(h.lastUpdate) < h.cacheTTL {
        if cached, ok := h.cache.Load("metrics"); ok {
            w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
            w.Write(cached.([]byte))
            return
        }
    }

    // Gather metrics
    families, err := h.gatherMetrics(r.Context())
    if err != nil {
        h.handleError(w, r, err)
        return
    }

    // Serialize and cache
    var buf bytes.Buffer
    encoder := expfmt.NewEncoder(&buf, expfmt.FmtText)
    for _, family := range families {
        encoder.Encode(family)
    }

    cached := buf.Bytes()
    h.cache.Store("metrics", cached)
    h.lastUpdate = time.Now()

    // Write response
    w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
    w.Write(cached)
}
```

### 5. Безопасность

#### Rate Limiting

```go
// pkg/metrics/endpoint.go (continued)

// RateLimitedMetricsEndpointHandler adds rate limiting.
type RateLimitedMetricsEndpointHandler struct {
    *MetricsEndpointHandler
    limiter *rate.Limiter
}

// NewRateLimitedMetricsEndpointHandler creates a rate-limited handler.
func NewRateLimitedMetricsEndpointHandler(base *MetricsEndpointHandler, rps float64, burst int) *RateLimitedMetricsEndpointHandler {
    return &RateLimitedMetricsEndpointHandler{
        MetricsEndpointHandler: base,
        limiter:                rate.NewLimiter(rate.Limit(rps), burst),
    }
}

// ServeHTTP implements http.Handler with rate limiting.
func (h *RateLimitedMetricsEndpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    if !h.limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
    }

    h.MetricsEndpointHandler.ServeHTTP(w, r)
}
```

### 6. Интеграция в main.go

```go
// cmd/server/main.go (modification)

// Add metrics endpoint handler
if cfg.Metrics.Enabled {
    // Create MetricsRegistry
    metricsRegistry := metrics.DefaultRegistry()

    // Create endpoint handler
    endpointConfig := metrics.DefaultEndpointConfig()
    endpointConfig.Path = cfg.Metrics.Path
    endpointConfig.EnableGoRuntime = cfg.Metrics.EnableGoRuntime
    endpointConfig.EnableProcess = cfg.Metrics.EnableProcess

    metricsHandler, err := metrics.NewMetricsEndpointHandler(endpointConfig, metricsRegistry)
    if err != nil {
        slog.Error("Failed to create metrics endpoint handler", "error", err)
        return err
    }

    // Register HTTP metrics
    if metricsManager != nil {
        if err := metricsHandler.RegisterHTTPMetrics(metricsManager.Metrics()); err != nil {
            slog.Error("Failed to register HTTP metrics", "error", err)
            return err
        }
    }

    // Register route
    mux.Handle(cfg.Metrics.Path, metricsHandler)
    slog.Info("Prometheus metrics endpoint enabled", "path", cfg.Metrics.Path)
}
```

## 🔍 Сценарии использования

### Сценарий 1: Базовое использование

```go
// Create handler with default config
config := metrics.DefaultEndpointConfig()
handler, err := metrics.NewMetricsEndpointHandler(config, metricsRegistry)
if err != nil {
    log.Fatal(err)
}

// Register route
http.Handle("/metrics", handler)
```

### Сценарий 2: С кэшированием

```go
// Create cached handler
config := metrics.DefaultEndpointConfig()
handler, err := metrics.NewCachedMetricsEndpointHandler(config, metricsRegistry, 5*time.Second)
if err != nil {
    log.Fatal(err)
}

http.Handle("/metrics", handler)
```

### Сценарий 3: С rate limiting

```go
// Create rate-limited handler
baseHandler, _ := metrics.NewMetricsEndpointHandler(config, metricsRegistry)
handler := metrics.NewRateLimitedMetricsEndpointHandler(baseHandler, 10.0, 20)

http.Handle("/metrics", handler)
```

## 🚦 Edge Cases

### Edge Case 1: Timeout при сборе метрик

**Проблема:** Сбор метрик может занять слишком много времени.

**Решение:** Использовать context с timeout, возвращать частичные метрики или ошибку.

### Edge Case 2: Большой объём метрик

**Проблема:** Ответ может быть очень большим (>10MB).

**Решение:** Ограничение размера ответа, streaming response, или фильтрация метрик.

### Edge Case 3: Concurrent requests

**Проблема:** Множественные одновременные запросы могут создать нагрузку.

**Решение:** Кэширование, rate limiting, оптимизация сбора метрик.

### Edge Case 4: Ошибки регистрации метрик

**Проблема:** Дублирование регистрации метрик может вызвать панику.

**Решение:** Проверка перед регистрацией, graceful error handling.

## 📝 API Контракты

### HTTP API

```
GET /metrics

Request:
  Method: GET
  Path: /metrics
  Headers: (optional) Accept: text/plain

Response:
  Status: 200 OK
  Headers:
    Content-Type: text/plain; version=0.0.4; charset=utf-8
    Cache-Control: no-cache, no-store, must-revalidate
  Body: Prometheus text format metrics

Error Responses:
  404 Not Found: Metrics disabled or invalid path
  405 Method Not Allowed: Non-GET method
  429 Too Many Requests: Rate limit exceeded
  500 Internal Server Error: Error gathering metrics
```

### Go API

```go
// NewMetricsEndpointHandler creates a new handler
func NewMetricsEndpointHandler(config EndpointConfig, registry *MetricsRegistry) (*MetricsEndpointHandler, error)

// RegisterMetricsRegistry registers metrics from registry
func (h *MetricsEndpointHandler) RegisterMetricsRegistry(registry *MetricsRegistry) error

// RegisterHTTPMetrics registers HTTP metrics
func (h *MetricsEndpointHandler) RegisterHTTPMetrics(httpMetrics *HTTPMetrics) error
```

## ✅ Acceptance Criteria

### Phase 1: Core Implementation
- [ ] MetricsEndpointHandler реализован
- [ ] Интеграция с MetricsRegistry работает
- [ ] Интеграция с HTTPMetrics работает
- [ ] Базовое тестирование проходит

### Phase 2: Error Handling
- [ ] ErrorHandler реализован
- [ ] Graceful degradation работает
- [ ] Логирование ошибок работает

### Phase 3: Performance
- [ ] Производительность соответствует требованиям
- [ ] Кэширование работает (опционально)
- [ ] Benchmarks показывают хорошие результаты

### Phase 4: Security
- [ ] Rate limiting работает
- [ ] Security headers установлены
- [ ] Валидация запросов работает

### Phase 5: Observability
- [ ] Self-observability metrics работают
- [ ] Логирование структурировано
- [ ] Метрики экспортируются корректно

---

**Next Steps:**
1. Review дизайна с командой
2. Создать tasks.md с детальным планом
3. Начать реализацию
