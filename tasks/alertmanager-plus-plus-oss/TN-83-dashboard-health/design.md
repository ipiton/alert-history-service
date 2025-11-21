# TN-83: GET /api/dashboard/health (basic) - Design Document

## 1. Архитектурный обзор

### 1.1 Цель дизайна

Спроектировать API endpoint для проверки здоровья всех критических компонентов системы с фокусом на:
- **Производительность** - быстрая проверка всех компонентов (< 500ms)
- **Надежность** - graceful degradation при отсутствии компонентов
- **Параллелизм** - параллельное выполнение проверок для минимизации времени ответа
- **Детализация** - подробная информация о состоянии каждого компонента

### 1.2 Архитектурные принципы

1. **Parallel Health Checks** - параллельное выполнение всех проверок через goroutines
2. **Graceful Degradation** - работа при отсутствии опциональных компонентов
3. **Timeout Protection** - индивидуальные timeout для каждого компонента
4. **Status Aggregation** - агрегация статусов компонентов в общий статус системы
5. **Fail-Safe Design** - частичные ошибки не блокируют весь endpoint

### 1.3 Компонентная архитектура

```
┌─────────────────────────────────────────────────────────────┐
│              HTTP Handler Layer                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ DashboardHealthHandler                                │   │
│  │ - Orchestrate parallel health checks                 │   │
│  │ - Aggregate component statuses                      │   │
│  │ - Determine overall system status                   │   │
│  │ - Format response                                    │   │
│  │ - Handle errors gracefully                           │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│         Parallel Health Check Execution Layer                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Database     │  │ Redis        │  │ LLM Service  │      │
│  │ Health       │  │ Health       │  │ Health       │      │
│  │ Check        │  │ Check        │  │ Check        │      │
│  │ (5s timeout) │  │ (2s timeout) │  │ (3s timeout) │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │ Publishing   │  │ System       │                        │
│  │ Health       │  │ Metrics      │                        │
│  │ Check        │  │ Collection   │                        │
│  │ (5s timeout) │  │ (optional)   │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│            Infrastructure Layer                               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ PostgreSQL   │  │ Redis        │  │ Classification│     │
│  │ Pool         │  │ Cache        │  │ Service      │      │
│  │              │  │              │  │              │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │ Publishing   │  │ Prometheus   │                        │
│  │ System       │  │ Metrics      │                        │
│  │              │  │              │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Детальный дизайн компонентов

### 2.1 DashboardHealthHandler

**Назначение:** HTTP handler для dashboard health endpoint

**Структура:**
```go
type DashboardHealthHandler struct {
    // Required dependencies
    dbPool      *postgres.PostgresPool
    cache       cache.Cache // optional

    // Optional dependencies
    classificationService services.ClassificationService // optional
    targetDiscovery       publishing.TargetDiscoveryManager // optional
    healthMonitor         publishing.HealthMonitor // optional
    modeManager           publishing.ModeManager // optional

    // Observability
    logger  *slog.Logger
    metrics *metrics.MetricsRegistry

    // Configuration
    config *HealthCheckConfig
}

type HealthCheckConfig struct {
    DatabaseTimeout    time.Duration // 5s
    RedisTimeout       time.Duration // 2s
    LLMTimeout         time.Duration // 3s
    PublishingTimeout  time.Duration // 5s
    OverallTimeout     time.Duration // 10s
    EnableSystemMetrics bool // false by default
}
```

**Методы:**
```go
// Main handler method
func (h *DashboardHealthHandler) GetHealth(w http.ResponseWriter, r *http.Request)

// Parallel health check execution
func (h *DashboardHealthHandler) checkDatabaseHealth(ctx context.Context) ServiceHealth
func (h *DashboardHealthHandler) checkRedisHealth(ctx context.Context) ServiceHealth
func (h *DashboardHealthHandler) checkLLMHealth(ctx context.Context) ServiceHealth
func (h *DashboardHealthHandler) checkPublishingHealth(ctx context.Context) ServiceHealth
func (h *DashboardHealthHandler) collectSystemMetrics(ctx context.Context) *SystemMetrics

// Status aggregation
func (h *DashboardHealthHandler) aggregateStatus(services map[string]ServiceHealth) (string, int)
func (h *DashboardHealthHandler) determineHTTPStatus(overallStatus string, dbStatus string) int
```

### 2.2 ServiceHealth Model

**Структура:**
```go
type ServiceHealth struct {
    Status    string                 `json:"status"`    // healthy/unhealthy/degraded/not_configured/available/unavailable
    LatencyMS *int64                 `json:"latency_ms,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Error     string                 `json:"error,omitempty"`
}
```

**Status Values:**
- `healthy` - компонент работает нормально
- `unhealthy` - компонент недоступен или работает некорректно
- `degraded` - компонент работает, но с проблемами
- `not_configured` - компонент не настроен (не критично)
- `available` - компонент доступен (для LLM)
- `unavailable` - компонент недоступен (для LLM)

### 2.3 DashboardHealthResponse Model

**Структура:**
```go
type DashboardHealthResponse struct {
    Status    string                 `json:"status"`    // healthy/degraded/unhealthy
    Timestamp time.Time              `json:"timestamp"`
    Services  map[string]ServiceHealth `json:"services"`
    Metrics   *SystemMetrics         `json:"metrics,omitempty"`
}

type SystemMetrics struct {
    CPUUsage    float64 `json:"cpu_usage,omitempty"`
    MemoryUsage float64 `json:"memory_usage,omitempty"`
    RequestRate float64 `json:"request_rate,omitempty"`
    ErrorRate   float64 `json:"error_rate,omitempty"`
}
```

---

## 3. Алгоритм выполнения health checks

### 3.1 Параллельное выполнение

```go
func (h *DashboardHealthHandler) GetHealth(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), h.config.OverallTimeout)
    defer cancel()

    // Create response structure
    response := DashboardHealthResponse{
        Timestamp: time.Now(),
        Services:  make(map[string]ServiceHealth),
    }

    // Channel for collecting results
    results := make(chan healthCheckResult, 5)

    // Launch parallel health checks
    var wg sync.WaitGroup

    // Database check (critical)
    wg.Add(1)
    go func() {
        defer wg.Done()
        dbCtx, cancel := context.WithTimeout(ctx, h.config.DatabaseTimeout)
        defer cancel()
        health := h.checkDatabaseHealth(dbCtx)
        results <- healthCheckResult{component: "database", health: health}
    }()

    // Redis check
    wg.Add(1)
    go func() {
        defer wg.Done()
        redisCtx, cancel := context.WithTimeout(ctx, h.config.RedisTimeout)
        defer cancel()
        health := h.checkRedisHealth(redisCtx)
        results <- healthCheckResult{component: "redis", health: health}
    }()

    // LLM check (optional)
    if h.classificationService != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            llmCtx, cancel := context.WithTimeout(ctx, h.config.LLMTimeout)
            defer cancel()
            health := h.checkLLMHealth(llmCtx)
            results <- healthCheckResult{component: "llm_service", health: health}
        }()
    }

    // Publishing check (optional)
    if h.targetDiscovery != nil {
        wg.Add(1)
        go func() {
            defer wg.Done()
            pubCtx, cancel := context.WithTimeout(ctx, h.config.PublishingTimeout)
            defer cancel()
            health := h.checkPublishingHealth(pubCtx)
            results <- healthCheckResult{component: "publishing", health: health}
        }()
    }

    // Wait for all checks to complete
    go func() {
        wg.Wait()
        close(results)
    }()

    // Collect results
    for result := range results {
        response.Services[result.component] = result.health
    }

    // Aggregate overall status
    response.Status, statusCode := h.aggregateStatus(response.Services)

    // Collect system metrics (optional, non-blocking)
    if h.config.EnableSystemMetrics {
        response.Metrics = h.collectSystemMetrics(ctx)
    }

    // Record metrics
    h.recordMetrics(response)

    // Send response
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)
    json.NewEncoder(w).Encode(response)
}
```

### 3.2 Database Health Check

```go
func (h *DashboardHealthHandler) checkDatabaseHealth(ctx context.Context) ServiceHealth {
    start := time.Now()

    health := ServiceHealth{
        Status:  "unhealthy",
        Details: make(map[string]interface{}),
    }

    if h.dbPool == nil {
        health.Status = "not_configured"
        return health
    }

    // Check connection
    err := h.dbPool.Health(ctx)
    if err != nil {
        health.Error = err.Error()
        latency := time.Since(start).Milliseconds()
        health.LatencyMS = &latency
        return health
    }

    // Get pool statistics
    stats := h.dbPool.Stats()
    health.Details["connection_pool"] = fmt.Sprintf("%d/%d",
        stats.ActiveConnections, stats.TotalConnections)
    health.Details["type"] = "postgresql"

    latency := time.Since(start).Milliseconds()
    health.LatencyMS = &latency
    health.Status = "healthy"

    return health
}
```

### 3.3 Redis Health Check

```go
func (h *DashboardHealthHandler) checkRedisHealth(ctx context.Context) ServiceHealth {
    start := time.Now()

    health := ServiceHealth{
        Status:  "not_configured",
        Details: make(map[string]interface{}),
    }

    if h.cache == nil {
        return health
    }

    // Check connection
    err := h.cache.HealthCheck(ctx)
    if err != nil {
        health.Status = "unhealthy"
        health.Error = err.Error()
        latency := time.Since(start).Milliseconds()
        health.LatencyMS = &latency
        return health
    }

    // Get stats if available
    stats, err := h.cache.GetStats(ctx)
    if err == nil {
        if memUsage, ok := stats["memory_usage"].(string); ok {
            health.Details["memory_usage"] = memUsage
        }
    }

    latency := time.Since(start).Milliseconds()
    health.LatencyMS = &latency
    health.Status = "healthy"

    return health
}
```

### 3.4 LLM Service Health Check

```go
func (h *DashboardHealthHandler) checkLLMHealth(ctx context.Context) ServiceHealth {
    start := time.Now()

    health := ServiceHealth{
        Status:  "not_configured",
        Details: make(map[string]interface{}),
    }

    if h.classificationService == nil {
        return health
    }

    // Check health
    err := h.classificationService.Health(ctx)
    if err != nil {
        health.Status = "unavailable"
        health.Error = err.Error()
        latency := time.Since(start).Milliseconds()
        health.LatencyMS = &latency
        return health
    }

    // Get stats if available (optional)
    // stats := h.classificationService.GetStats()
    // health.Details["requests_per_minute"] = stats.RequestsPerMinute

    latency := time.Since(start).Milliseconds()
    health.LatencyMS = &latency
    health.Status = "available"

    return health
}
```

### 3.5 Publishing System Health Check

```go
func (h *DashboardHealthHandler) checkPublishingHealth(ctx context.Context) ServiceHealth {
    start := time.Now()

    health := ServiceHealth{
        Status:  "not_configured",
        Details: make(map[string]interface{}),
    }

    if h.targetDiscovery == nil {
        return health
    }

    // Get stats
    stats := h.targetDiscovery.GetStats()
    health.Details["targets_count"] = stats.TotalTargets

    // Get mode
    if h.modeManager != nil {
        mode := h.modeManager.GetCurrentMode()
        health.Details["mode"] = mode.String()

        if mode == publishing.ModeMetricsOnly {
            health.Status = "degraded"
            latency := time.Since(start).Milliseconds()
            health.LatencyMS = &latency
            return health
        }
    }

    // Check unhealthy targets
    if h.healthMonitor != nil {
        allHealth := h.healthMonitor.GetHealth(ctx)
        unhealthyCount := 0
        for _, h := range allHealth {
            if h.Status == publishing.HealthStatusUnhealthy {
                unhealthyCount++
            }
        }
        health.Details["unhealthy_targets"] = unhealthyCount

        if unhealthyCount > 0 {
            health.Status = "degraded"
        } else {
            health.Status = "healthy"
        }
    } else {
        health.Status = "healthy"
    }

    latency := time.Since(start).Milliseconds()
    health.LatencyMS = &latency

    return health
}
```

### 3.6 Status Aggregation Logic

```go
func (h *DashboardHealthHandler) aggregateStatus(services map[string]ServiceHealth) (string, int) {
    // Check database first (critical)
    dbHealth, dbExists := services["database"]
    if !dbExists || dbHealth.Status == "not_configured" {
        // Database is required
        return "unhealthy", http.StatusServiceUnavailable
    }
    if dbHealth.Status == "unhealthy" {
        return "unhealthy", http.StatusServiceUnavailable
    }

    // Check other services
    hasDegraded := false
    hasUnhealthy := false

    for component, health := range services {
        if component == "database" {
            continue // Already checked
        }

        switch health.Status {
        case "unhealthy":
            // Redis unhealthy might be critical depending on configuration
            if component == "redis" {
                hasDegraded = true // Degrade, but don't fail
            } else {
                hasUnhealthy = true
            }
        case "degraded", "unavailable":
            hasDegraded = true
        }
    }

    // Determine overall status
    if hasUnhealthy {
        return "unhealthy", http.StatusServiceUnavailable
    }
    if hasDegraded {
        return "degraded", http.StatusOK
    }

    return "healthy", http.StatusOK
}
```

---

## 4. Интеграция с существующими компонентами

### 4.1 PostgreSQL Pool Integration

**Использование:**
- `PostgresPool.Health(ctx)` - проверка здоровья
- `PostgresPool.Stats()` - статистика пула соединений

**Особенности:**
- Timeout: 5 секунд
- Критичный компонент - недоступность → HTTP 503

### 4.2 Redis Cache Integration

**Использование:**
- `Cache.HealthCheck(ctx)` - проверка здоровья
- `Cache.GetStats(ctx)` - статистика (опционально)

**Особенности:**
- Timeout: 2 секунды
- Опциональный компонент - недоступность → degraded, но не unhealthy

### 4.3 Classification Service Integration

**Использование:**
- `ClassificationService.Health(ctx)` - проверка здоровья
- `ClassificationService.GetStats()` - статистика (опционально)

**Особенности:**
- Timeout: 3 секунды
- Опциональный компонент - недоступность не влияет на общий статус

### 4.4 Publishing System Integration

**Использование:**
- `TargetDiscoveryManager.GetStats()` - статистика discovery
- `HealthMonitor.GetHealth(ctx)` - здоровье targets (опционально)
- `ModeManager.GetCurrentMode()` - текущий режим

**Особенности:**
- Timeout: 5 секунд
- Опциональный компонент - degraded mode → degraded status

---

## 5. Обработка ошибок

### 5.1 Error Handling Strategy

1. **Timeout Errors** - логируются, возвращается timeout status
2. **Connection Errors** - логируются, возвращается unhealthy/degraded
3. **Partial Errors** - не блокируют другие проверки
4. **Missing Components** - возвращается `not_configured` (не критично)

### 5.2 Error Logging

```go
h.logger.Error("Database health check failed",
    "error", err,
    "component", "database",
    "latency_ms", latency,
)
```

### 5.3 Error Response Format

```json
{
  "status": "unhealthy",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "unhealthy",
      "latency_ms": 5000,
      "error": "connection timeout after 5s"
    }
  }
}
```

---

## 6. Prometheus Metrics

### 6.1 Metrics Definition

```go
// Counter: Total health checks performed
dashboard_health_checks_total{component="database|redis|llm_service|publishing", status="healthy|unhealthy|degraded|not_configured"}

// Histogram: Health check duration
dashboard_health_check_duration_seconds{component="database|redis|llm_service|publishing"}

// Gauge: Current health status (1=healthy, 0.5=degraded, 0=unhealthy)
dashboard_health_status{component="database|redis|llm_service|publishing"}
dashboard_health_overall_status{status="healthy|degraded|unhealthy"}
```

### 6.2 Metrics Recording

```go
func (h *DashboardHealthHandler) recordMetrics(response DashboardHealthResponse) {
    // Record component statuses
    for component, health := range response.Services {
        statusValue := 0.0
        switch health.Status {
        case "healthy", "available":
            statusValue = 1.0
        case "degraded":
            statusValue = 0.5
        case "unhealthy", "unavailable":
            statusValue = 0.0
        default:
            continue // Skip not_configured
        }

        h.metrics.Gauge("dashboard_health_status", statusValue,
            map[string]string{"component": component})

        h.metrics.Counter("dashboard_health_checks_total", 1,
            map[string]string{"component": component, "status": health.Status})

        if health.LatencyMS != nil {
            latencySeconds := float64(*health.LatencyMS) / 1000.0
            h.metrics.Histogram("dashboard_health_check_duration_seconds", latencySeconds,
                map[string]string{"component": component})
        }
    }

    // Record overall status
    overallValue := 0.0
    switch response.Status {
    case "healthy":
        overallValue = 1.0
    case "degraded":
        overallValue = 0.5
    case "unhealthy":
        overallValue = 0.0
    }

    h.metrics.Gauge("dashboard_health_overall_status", overallValue,
        map[string]string{"status": response.Status})
}
```

---

## 7. Производительность и оптимизация

### 7.1 Performance Targets

- **Response Time:** < 500ms (p95), < 1s (p99)
- **Throughput:** > 100 req/s
- **Timeout Rate:** < 1%

### 7.2 Optimization Strategies

1. **Parallel Execution** - все проверки выполняются параллельно
2. **Individual Timeouts** - короткие timeout для каждого компонента
3. **Fail-Fast** - database недоступна → сразу возвращаем unhealthy
4. **Non-Blocking Metrics** - системные метрики собираются опционально

### 7.3 Caching (Future Enhancement)

Опциональное кэширование результатов на 10-30 секунд для снижения нагрузки:
- Кэш в памяти с TTL
- Invalidation при изменении статуса
- Не рекомендуется для production (health checks должны быть real-time)

---

## 8. Тестирование

### 8.1 Unit Tests

- Тестирование каждого health check метода
- Тестирование status aggregation logic
- Тестирование error handling
- Тестирование timeout scenarios

### 8.2 Integration Tests

- Тестирование с реальными компонентами
- Тестирование graceful degradation
- Тестирование параллельного выполнения

### 8.3 Benchmarks

- Benchmark response time
- Benchmark concurrent requests
- Benchmark timeout handling

---

## 9. Безопасность

### 9.1 Security Considerations

- Не раскрывать sensitive информацию (пароли, токены, connection strings)
- Rate limiting через middleware (опционально)
- CORS support (если требуется)

### 9.2 Information Disclosure

**Safe to expose:**
- Component status (healthy/unhealthy)
- Latency metrics
- Connection pool statistics (active/total)
- Memory usage (если не sensitive)

**Not safe to expose:**
- Connection strings
- Authentication tokens
- Detailed error messages с sensitive data

---

## 10. API Контракты

### 10.1 GET /api/dashboard/health

**Request:**
```
GET /api/dashboard/health
```

**Response (200 OK - Healthy):**
```json
{
  "status": "healthy",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 15,
      "details": {
        "connection_pool": "8/20",
        "type": "postgresql"
      }
    },
    "redis": {
      "status": "healthy",
      "latency_ms": 2,
      "details": {
        "memory_usage": "45MB"
      }
    }
  }
}
```

**Response (200 OK - Degraded):**
```json
{
  "status": "degraded",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 15
    },
    "redis": {
      "status": "unhealthy",
      "error": "connection timeout"
    }
  }
}
```

**Response (503 Service Unavailable - Unhealthy):**
```json
{
  "status": "unhealthy",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "unhealthy",
      "error": "connection refused"
    }
  }
}
```

---

*Design Document Version: 1.0*
*Last Updated: 2025-11-21*
*Author: AI Assistant*
*Status: DRAFT → READY FOR IMPLEMENTATION*
