# TN-181: Дизайн системы унификации метрик Prometheus

**Дата:** 2025-10-09
**Версия:** 1.0
**Статус:** DRAFT

## 🎯 Архитектурное решение

### Общая концепция

Создать централизованную систему управления метриками с единой точкой регистрации, консистентным именованием и автоматической валидацией.

```
┌─────────────────────────────────────────────────────────────┐
│                    Alert History Service                      │
├─────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌───────────────────────────────────────────────────┐      │
│  │         Metrics Registry (Singleton)               │      │
│  │  - Centralized metric registration                 │      │
│  │  - Naming validation                               │      │
│  │  - Category management                             │      │
│  └───────────────┬───────────────────────────────────┘      │
│                  │                                            │
│  ┌───────────────┴───────────────────────────────────┐      │
│  │          Metric Categories                         │      │
│  ├───────────────────┬────────────────┬──────────────┤      │
│  │   Business        │   Technical     │     Infra    │      │
│  ├───────────────────┼────────────────┼──────────────┤      │
│  │ alerts            │ http            │ db           │      │
│  │ llm               │ llm_cb          │ cache        │      │
│  │ publishing        │ filter          │ repository   │      │
│  └───────────────────┴────────────────┴──────────────┘      │
│                                                               │
│  ┌───────────────────────────────────────────────────┐      │
│  │        Prometheus Client (promauto)               │      │
│  └───────────────────────────────────────────────────┘      │
│                                                               │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
              ┌──────────────┐
              │  Prometheus  │
              │   Server     │
              └──────┬───────┘
                     │
                     ▼
              ┌──────────────┐
              │   Grafana    │
              └──────────────┘
```

## 📐 Детальный дизайн

### 1. Metrics Registry

Централизованный реестр для управления всеми метриками приложения.

#### Структура

```go
// pkg/metrics/registry.go

package metrics

import (
    "fmt"
    "sync"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricCategory определяет категорию метрики
type MetricCategory string

const (
    CategoryBusiness   MetricCategory = "business"
    CategoryTechnical  MetricCategory = "technical"
    CategoryInfra      MetricCategory = "infra"
)

// MetricsRegistry централизованный реестр метрик
type MetricsRegistry struct {
    namespace string
    mu        sync.RWMutex
    metrics   map[string]prometheus.Collector

    // Category managers
    business   *BusinessMetrics
    technical  *TechnicalMetrics
    infra      *InfraMetrics
}

// NewMetricsRegistry создает новый реестр метрик
func NewMetricsRegistry(namespace string) *MetricsRegistry {
    if namespace == "" {
        namespace = "alert_history"
    }

    registry := &MetricsRegistry{
        namespace: namespace,
        metrics:   make(map[string]prometheus.Collector),
    }

    // Initialize category managers
    registry.business = NewBusinessMetrics(namespace)
    registry.technical = NewTechnicalMetrics(namespace)
    registry.infra = NewInfraMetrics(namespace)

    return registry
}

// Business возвращает business metrics manager
func (r *MetricsRegistry) Business() *BusinessMetrics {
    return r.business
}

// Technical возвращает technical metrics manager
func (r *MetricsRegistry) Technical() *TechnicalMetrics {
    return r.technical
}

// Infra возвращает infra metrics manager
func (r *MetricsRegistry) Infra() *InfraMetrics {
    return r.infra
}

// ValidateMetricName проверяет корректность имени метрики
func (r *MetricsRegistry) ValidateMetricName(name string) error {
    // Validation rules based on Prometheus best practices
    // 1. Must start with namespace
    // 2. Must contain category
    // 3. Must match pattern: namespace_category_subsystem_name_unit
    // TODO: implement validation logic
    return nil
}
```

### 2. Business Metrics

Бизнес-метрики для отслеживания обработки алертов, обогащения и публикации.

```go
// pkg/metrics/business.go

package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// BusinessMetrics содержит бизнес-метрики
type BusinessMetrics struct {
    namespace string

    // Alerts subsystem
    AlertsProcessedTotal   prometheus.Counter
    AlertsEnrichedTotal    *prometheus.CounterVec
    AlertsFilteredTotal    *prometheus.CounterVec

    // LLM subsystem
    LLMClassificationsTotal      *prometheus.CounterVec
    LLMRecommendationsTotal      prometheus.Counter
    LLMConfidenceScore           prometheus.Histogram

    // Publishing subsystem
    PublishingSuccessTotal       *prometheus.CounterVec
    PublishingFailedTotal        *prometheus.CounterVec
    PublishingDurationSeconds    *prometheus.HistogramVec
}

// NewBusinessMetrics создает новый набор бизнес-метрик
func NewBusinessMetrics(namespace string) *BusinessMetrics {
    return &BusinessMetrics{
        namespace: namespace,

        // Alerts subsystem
        AlertsProcessedTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "business_alerts",
            Name:      "processed_total",
            Help:      "Total number of alerts processed by the system",
        }),

        AlertsEnrichedTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "business_alerts",
                Name:      "enriched_total",
                Help:      "Total number of alerts enriched with LLM data",
            },
            []string{"mode", "status"},
        ),

        AlertsFilteredTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "business_alerts",
                Name:      "filtered_total",
                Help:      "Total number of alerts filtered (allowed/blocked)",
            },
            []string{"result", "reason"},
        ),

        // LLM subsystem
        LLMClassificationsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "business_llm",
                Name:      "classifications_total",
                Help:      "Total number of LLM classifications performed",
            },
            []string{"severity", "confidence"},
        ),

        LLMRecommendationsTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "business_llm",
            Name:      "recommendations_total",
            Help:      "Total number of LLM recommendations generated",
        }),

        LLMConfidenceScore: promauto.NewHistogram(prometheus.HistogramOpts{
            Namespace: namespace,
            Subsystem: "business_llm",
            Name:      "confidence_score",
            Help:      "Distribution of LLM confidence scores",
            Buckets:   []float64{0.5, 0.6, 0.7, 0.8, 0.85, 0.9, 0.95, 0.99},
        }),

        // Publishing subsystem
        PublishingSuccessTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "business_publishing",
                Name:      "success_total",
                Help:      "Total number of successful alert publishes",
            },
            []string{"destination"},
        ),

        PublishingFailedTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "business_publishing",
                Name:      "failed_total",
                Help:      "Total number of failed alert publishes",
            },
            []string{"destination", "error_type"},
        ),

        PublishingDurationSeconds: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: "business_publishing",
                Name:      "duration_seconds",
                Help:      "Duration of publishing operations in seconds",
                Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
            },
            []string{"destination"},
        ),
    }
}
```

### 3. Technical Metrics

Технические метрики для HTTP, LLM Circuit Breaker, фильтров и обогащения.

```go
// pkg/metrics/technical.go

package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// TechnicalMetrics содержит технические метрики
type TechnicalMetrics struct {
    namespace string

    // HTTP subsystem (existing, keeping for reference)
    HTTP *HTTPMetrics

    // LLM Circuit Breaker subsystem
    LLMCB *LLMCircuitBreakerMetrics

    // Filter subsystem (existing, keeping for reference)
    Filter *FilterMetrics

    // Enrichment subsystem (existing, keeping for reference)
    Enrichment *EnrichmentMetrics
}

// NewTechnicalMetrics создает новый набор технических метрик
func NewTechnicalMetrics(namespace string) *TechnicalMetrics {
    return &TechnicalMetrics{
        namespace:  namespace,
        HTTP:       NewHTTPMetrics(), // existing
        LLMCB:      NewLLMCircuitBreakerMetrics(namespace),
        Filter:     NewFilterMetrics(), // existing
        Enrichment: NewEnrichmentMetrics(), // existing
    }
}

// LLMCircuitBreakerMetrics метрики для Circuit Breaker
type LLMCircuitBreakerMetrics struct {
    State                prometheus.Gauge
    FailuresTotal        prometheus.Counter
    SuccessesTotal       prometheus.Counter
    StateChangesTotal    *prometheus.CounterVec
    RequestsBlockedTotal prometheus.Counter
    HalfOpenRequestsTotal prometheus.Counter
    SlowCallsTotal       prometheus.Counter
    CallDurationSeconds  *prometheus.HistogramVec
}

// NewLLMCircuitBreakerMetrics создает метрики CB с новым именованием
func NewLLMCircuitBreakerMetrics(namespace string) *LLMCircuitBreakerMetrics {
    return &LLMCircuitBreakerMetrics{
        State: promauto.NewGauge(prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: "technical_llm_cb",
            Name:      "state",
            Help:      "Current state of LLM circuit breaker (0=closed, 1=open, 2=half_open)",
        }),

        FailuresTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "technical_llm_cb",
            Name:      "failures_total",
            Help:      "Total number of failed LLM calls",
        }),

        SuccessesTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "technical_llm_cb",
            Name:      "successes_total",
            Help:      "Total number of successful LLM calls",
        }),

        StateChangesTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "technical_llm_cb",
                Name:      "state_changes_total",
                Help:      "Total number of circuit breaker state changes",
            },
            []string{"from", "to"},
        ),

        RequestsBlockedTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "technical_llm_cb",
            Name:      "requests_blocked_total",
            Help:      "Total number of requests blocked by circuit breaker",
        }),

        HalfOpenRequestsTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "technical_llm_cb",
            Name:      "half_open_requests_total",
            Help:      "Total number of test requests in half-open state",
        }),

        SlowCallsTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "technical_llm_cb",
            Name:      "slow_calls_total",
            Help:      "Total number of slow LLM calls (exceeding threshold)",
        }),

        CallDurationSeconds: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: "technical_llm_cb",
                Name:      "call_duration_seconds",
                Help:      "Duration of LLM calls in seconds",
                Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 2.0, 3.0, 5.0, 10.0, 30.0},
            },
            []string{"result"},
        ),
    }
}
```

### 4. Infrastructure Metrics

Инфраструктурные метрики для БД, кэша и репозиториев.

```go
// pkg/metrics/infra.go

package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// InfraMetrics содержит инфраструктурные метрики
type InfraMetrics struct {
    namespace string

    // Database subsystem
    DB *DatabaseMetrics

    // Cache subsystem
    Cache *CacheMetrics

    // Repository subsystem
    Repository *RepositoryMetrics
}

// NewInfraMetrics создает новый набор инфраструктурных метрик
func NewInfraMetrics(namespace string) *InfraMetrics {
    return &InfraMetrics{
        namespace:  namespace,
        DB:         NewDatabaseMetrics(namespace),
        Cache:      NewCacheMetrics(namespace),
        Repository: NewRepositoryMetrics(namespace),
    }
}

// DatabaseMetrics метрики для database pool
type DatabaseMetrics struct {
    ConnectionsActive             prometheus.Gauge
    ConnectionsIdle               prometheus.Gauge
    ConnectionsTotal              prometheus.Counter
    ConnectionWaitDurationSeconds prometheus.Histogram
    QueryDurationSeconds          *prometheus.HistogramVec
    QueriesTotal                  *prometheus.CounterVec
    ErrorsTotal                   *prometheus.CounterVec
}

// NewDatabaseMetrics создает метрики БД
func NewDatabaseMetrics(namespace string) *DatabaseMetrics {
    return &DatabaseMetrics{
        ConnectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: "infra_db",
            Name:      "connections_active",
            Help:      "Number of active database connections",
        }),

        ConnectionsIdle: promauto.NewGauge(prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: "infra_db",
            Name:      "connections_idle",
            Help:      "Number of idle database connections",
        }),

        ConnectionsTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "infra_db",
            Name:      "connections_total",
            Help:      "Total number of database connections created",
        }),

        ConnectionWaitDurationSeconds: promauto.NewHistogram(prometheus.HistogramOpts{
            Namespace: namespace,
            Subsystem: "infra_db",
            Name:      "connection_wait_duration_seconds",
            Help:      "Time spent waiting for a database connection",
            Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
        }),

        QueryDurationSeconds: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: "infra_db",
                Name:      "query_duration_seconds",
                Help:      "Duration of database queries in seconds",
                Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
            },
            []string{"operation"},
        ),

        QueriesTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "infra_db",
                Name:      "queries_total",
                Help:      "Total number of database queries executed",
            },
            []string{"operation", "status"},
        ),

        ErrorsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "infra_db",
                Name:      "errors_total",
                Help:      "Total number of database errors",
            },
            []string{"error_type"},
        ),
    }
}

// CacheMetrics метрики для кэша (Redis)
type CacheMetrics struct {
    HitsTotal      *prometheus.CounterVec
    MissesTotal    *prometheus.CounterVec
    ErrorsTotal    *prometheus.CounterVec
    EvictionsTotal prometheus.Counter
    SizeBytes      prometheus.Gauge
}

// NewCacheMetrics создает метрики кэша
func NewCacheMetrics(namespace string) *CacheMetrics {
    return &CacheMetrics{
        HitsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "infra_cache",
                Name:      "hits_total",
                Help:      "Total number of cache hits",
            },
            []string{"cache_type"},
        ),

        MissesTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "infra_cache",
                Name:      "misses_total",
                Help:      "Total number of cache misses",
            },
            []string{"cache_type"},
        ),

        ErrorsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "infra_cache",
                Name:      "errors_total",
                Help:      "Total number of cache errors",
            },
            []string{"cache_type", "error_type"},
        ),

        EvictionsTotal: promauto.NewCounter(prometheus.CounterOpts{
            Namespace: namespace,
            Subsystem: "infra_cache",
            Name:      "evictions_total",
            Help:      "Total number of cache evictions",
        }),

        SizeBytes: promauto.NewGauge(prometheus.GaugeOpts{
            Namespace: namespace,
            Subsystem: "infra_cache",
            Name:      "size_bytes",
            Help:      "Current size of cache in bytes",
        }),
    }
}

// RepositoryMetrics метрики для репозиториев
type RepositoryMetrics struct {
    QueryDurationSeconds *prometheus.HistogramVec
    QueryErrorsTotal     *prometheus.CounterVec
    QueryResultsTotal    *prometheus.HistogramVec
}

// NewRepositoryMetrics создает метрики репозиториев
func NewRepositoryMetrics(namespace string) *RepositoryMetrics {
    return &RepositoryMetrics{
        QueryDurationSeconds: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: "infra_repository",
                Name:      "query_duration_seconds",
                Help:      "Duration of repository queries",
                Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
            },
            []string{"operation", "status"},
        ),

        QueryErrorsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: namespace,
                Subsystem: "infra_repository",
                Name:      "query_errors_total",
                Help:      "Total number of repository query errors",
            },
            []string{"operation", "error_type"},
        ),

        QueryResultsTotal: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: namespace,
                Subsystem: "infra_repository",
                Name:      "query_results_total",
                Help:      "Number of results returned by repository queries",
                Buckets:   []float64{0, 1, 5, 10, 25, 50, 100, 250, 500, 1000},
            },
            []string{"operation"},
        ),
    }
}
```

### 5. Migration Strategy

#### Подход: Dual Emission + Recording Rules

**Фаза 1: Dual Emission (30 дней)**
- Эмитить и старые, и новые метрики одновременно
- Позволяет постепенную миграцию дашбордов
- Zero downtime для monitoring

**Фаза 2: Recording Rules (30 дней)**
- Создать Prometheus recording rules для mapping старых имен на новые
- Обновить все дашборды на новые метрики
- Продолжать поддержку legacy через rules

**Фаза 3: Deprecation (cleanup)**
- Удалить dual emission кода
- Оставить recording rules еще на 30 дней
- Финальный cleanup legacy метрик

#### Recording Rules Example

```yaml
# prometheus_rules.yml

groups:
  - name: alert_history_legacy_metrics
    interval: 10s
    rules:
      # Repository metrics backwards compatibility
      - record: alert_history_query_duration_seconds
        expr: alert_history_infra_repository_query_duration_seconds

      - record: alert_history_query_errors_total
        expr: alert_history_infra_repository_query_errors_total

      - record: alert_history_query_results_total
        expr: alert_history_infra_repository_query_results_total

      # Circuit Breaker metrics backwards compatibility
      - record: alert_history_llm_circuit_breaker_state
        expr: alert_history_technical_llm_cb_state

      - record: alert_history_llm_circuit_breaker_failures_total
        expr: alert_history_technical_llm_cb_failures_total

      # Add more legacy mappings as needed...
```

### 6. Database Pool Integration

Интеграция существующих `internal/database/postgres/metrics.go` с Prometheus.

```go
// internal/database/postgres/prometheus.go (NEW FILE)

package postgres

import (
    "time"
    "github.com/vitaliisemenov/alert-history/pkg/metrics"
)

// PrometheusExporter экспортирует Pool metrics в Prometheus
type PrometheusExporter struct {
    pool    *Pool
    metrics *metrics.DatabaseMetrics
}

// NewPrometheusExporter создает новый exporter
func NewPrometheusExporter(pool *Pool, dbMetrics *metrics.DatabaseMetrics) *PrometheusExporter {
    return &PrometheusExporter{
        pool:    pool,
        metrics: dbMetrics,
    }
}

// Start запускает периодический export метрик
func (e *PrometheusExporter) Start(interval time.Duration) {
    ticker := time.NewTicker(interval)
    go func() {
        for range ticker.C {
            e.exportMetrics()
        }
    }()
}

// exportMetrics экспортирует текущие метрики в Prometheus
func (e *PrometheusExporter) exportMetrics() {
    stats := e.pool.metrics.Snapshot()

    // Export connections
    e.metrics.ConnectionsActive.Set(float64(stats.ActiveConnections))
    e.metrics.ConnectionsIdle.Set(float64(stats.IdleConnections))
    e.metrics.ConnectionsTotal.Add(float64(stats.ConnectionsCreated))

    // Export query stats
    if stats.TotalQueries > 0 {
        avgDuration := stats.QueryExecutionTime / time.Duration(stats.TotalQueries)
        e.metrics.QueryDurationSeconds.WithLabelValues("all").Observe(avgDuration.Seconds())
    }

    // Export errors
    if stats.ConnectionErrors > 0 {
        e.metrics.ErrorsTotal.WithLabelValues("connection").Add(float64(stats.ConnectionErrors))
    }
    if stats.QueryErrors > 0 {
        e.metrics.ErrorsTotal.WithLabelValues("query").Add(float64(stats.QueryErrors))
    }
}
```

## 📊 Формат данных

### Metric Naming Convention

```
<namespace>_<category>_<subsystem>_<metric_name>_<unit>

Where:
- namespace:  alert_history (fixed)
- category:   business|technical|infra
- subsystem:  alerts|llm|http|db|cache|repository|etc
- metric_name: descriptive_name (snake_case)
- unit:       total|seconds|bytes (optional, for clarity)
```

### Examples

✅ **Good:**
- `alert_history_business_alerts_processed_total`
- `alert_history_technical_http_request_duration_seconds`
- `alert_history_infra_db_connections_active`
- `alert_history_infra_cache_hits_total`

❌ **Bad:**
- `alerts_processed` (no namespace)
- `alert_history_processed` (no category/subsystem)
- `my_custom_metric` (doesn't follow convention)

## 🔍 Сценарии использования

### Сценарий 1: Добавление новой метрики

**Разработчик хочет добавить метрику для tracking failed webhooks**

```go
// 1. Определить категорию: Business (alerts processing)
// 2. Определить subsystem: alerts или publishing
// 3. Создать метрику в соответствующем файле

// В pkg/metrics/business.go:
PublishingFailedTotal: promauto.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: namespace,
        Subsystem: "business_publishing",
        Name:      "failed_total",
        Help:      "Total number of failed alert publishes",
    },
    []string{"destination", "error_type"},
)

// 4. Использовать в коде:
metricsRegistry.Business().PublishingFailedTotal.
    WithLabelValues("webhook", "timeout").Inc()
```

### Сценарий 2: Migration существующей метрики

**SRE хочет мигрировать дашборд с `alert_history_query_duration_seconds` на новую метрику**

1. Проверить mapping table в requirements.md
2. Найти новое имя: `alert_history_infra_repository_query_duration_seconds`
3. Обновить PromQL запросы:

```promql
# Old:
histogram_quantile(0.95, rate(alert_history_query_duration_seconds_bucket[5m]))

# New:
histogram_quantile(0.95, rate(alert_history_infra_repository_query_duration_seconds_bucket[5m]))

# Or use recording rule (during transition):
histogram_quantile(0.95, rate(alert_history_query_duration_seconds[5m]))  # still works!
```

### Сценарий 3: Monitoring Database Pool

**SRE хочет создать alert на высокое количество ожиданий соединений**

```promql
# После реализации Database Pool metrics:
alert: HighDatabaseConnectionWaitTime
expr: |
  histogram_quantile(0.95,
    rate(alert_history_infra_db_connection_wait_duration_seconds_bucket[5m])
  ) > 0.1
for: 5m
labels:
  severity: warning
annotations:
  summary: "High database connection wait time (p95 > 100ms)"
```

## 🚦 Edge Cases

### Edge Case 1: Duplicate Metric Registration

**Проблема:** При hot reload или в тестах может возникнуть повторная регистрация метрик.

**Решение:** Использовать singleton pattern для MetricsRegistry:

```go
var (
    defaultRegistry     *MetricsRegistry
    defaultRegistryOnce sync.Once
)

func DefaultRegistry() *MetricsRegistry {
    defaultRegistryOnce.Do(func() {
        defaultRegistry = NewMetricsRegistry("alert_history")
    })
    return defaultRegistry
}
```

### Edge Case 2: High Cardinality Labels

**Проблема:** Label `path` в HTTP метриках может иметь высокую cardinality (UUID в path).

**Решение:** Path normalization middleware:

```go
func normalizePath(path string) string {
    // Replace UUIDs with :id placeholder
    // /api/alerts/123e4567-e89b-12d3-a456-426614174000 -> /api/alerts/:id
    return replaceUUIDs(path)
}
```

### Edge Case 3: Metrics в Multi-tenant Environment

**Проблема:** В будущем может потребоваться разделение метрик по tenants.

**Решение:** Добавить optional tenant label (пока не нужен):

```go
// Future-proofing:
type MetricsConfig struct {
    MultiTenant bool
    TenantLabel string // e.g., "tenant_id"
}
```

## 📝 API Контракты

### MetricsRegistry Interface

```go
type Registry interface {
    Business() *BusinessMetrics
    Technical() *TechnicalMetrics
    Infra() *InfraMetrics
    ValidateMetricName(name string) error
}
```

### MetricsCollector Interface

```go
type MetricsCollector interface {
    Collect(ch chan<- prometheus.Metric)
    Describe(ch chan<- *prometheus.Desc)
}
```

## ✅ Acceptance Criteria

### Критерии для Phase 1 (Аудит)

- [ ] Полный инвентарь всех существующих метрик (CSV/JSON)
- [ ] Документация использования метрик в Grafana
- [ ] Список всех recording rules
- [ ] Аудит отчет с рекомендациями

### Критерии для Phase 2 (Design)

- [ ] Финальная taxonomy метрик утверждена
- [ ] Mapping table создана
- [ ] Guidelines для разработчиков написаны
- [ ] SRE review пройден

### Критерии для Phase 3 (Implementation)

- [ ] MetricsRegistry реализован
- [ ] Все категории метрик мигрированы
- [ ] Database Pool metrics экспортируются
- [ ] 100% unit test coverage

### Критерии для Phase 4 (Migration)

- [ ] Recording rules deployed
- [ ] Grafana dashboards обновлены
- [ ] Legacy support работает
- [ ] Changelog опубликован

---

**Next Steps:**
1. Review дизайна с SRE командой
2. Получить approval на breaking changes
3. Создать POC для MetricsRegistry
4. Начать implementation Phase 3
