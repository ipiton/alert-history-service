# TN-71: GET /classification/stats - Design Document

## 1. Архитектурное решение

### 1.1 Обзор архитектуры

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request Handler                      │
│              (GetClassificationStats)                        │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Stats Aggregator Service                        │
│  ┌──────────────────┐  ┌──────────────────┐               │
│  │ Classification   │  │ Prometheus       │               │
│  │ Service Stats    │  │ Metrics Client   │               │
│  └──────────────────┘  └──────────────────┘               │
│           │                      │                          │
│           └──────────┬────────────┘                          │
│                     ▼                                        │
│            Stats Aggregation                                 │
│            (merge + enrich)                                  │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Response Builder                                 │
│  - Format by_severity stats                                  │
│  - Calculate percentages                                     │
│  - Add metadata                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              JSON Response                                    │
│              (StatsResponse)                                 │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Компоненты системы

#### Component-1: HTTP Handler Layer
**Файл:** `go-app/internal/api/handlers/classification/handlers.go`
**Ответственность:**
- HTTP request/response handling
- Input validation (query parameters)
- Error handling и status codes
- Response serialization

**Изменения:**
- Реализовать `GetClassificationStats()` метод
- Заменить TODO заглушку на реальную реализацию
- Добавить query parameter parsing (если требуется)

#### Component-2: Stats Aggregator
**Файл:** `go-app/internal/api/handlers/classification/stats_aggregator.go` (новый)
**Ответственность:**
- Агрегация данных из ClassificationService
- Запросы к Prometheus (опционально)
- Объединение данных из разных источников
- Кэширование результатов

**Интерфейс:**
```go
type StatsAggregator interface {
    AggregateStats(ctx context.Context) (*EnhancedStats, error)
}

type EnhancedStats struct {
    // Базовые метрики из ClassificationService
    BaseStats core.ClassificationStats

    // Расширенные метрики из Prometheus
    PrometheusStats *PrometheusStats

    // Агрегированные данные
    BySeverity map[string]SeverityStats
    CacheStats CacheStats
    LLMStats LLMStats
    FallbackStats FallbackStats
}
```

#### Component-3: Prometheus Client (опционально)
**Файл:** `go-app/internal/api/handlers/classification/prometheus_client.go` (новый)
**Ответственность:**
- Запросы к Prometheus API
- Парсинг Prometheus responses
- Graceful degradation при ошибках

**Особенности:**
- Использует HTTP client для запросов к Prometheus
- Timeout: 100ms (не блокирует основной ответ)
- Fallback на ClassificationService при ошибках

#### Component-4: Response Models
**Файл:** `go-app/internal/api/handlers/classification/handlers.go`
**Ответственность:**
- Определение структур ответа
- JSON serialization

**Расширение StatsResponse:**
```go
type StatsResponse struct {
    // Базовые метрики
    TotalClassified int64              `json:"total_classified"`
    TotalRequests   int64              `json:"total_requests"`
    ClassificationRate float64         `json:"classification_rate"`
    AvgConfidence   float64            `json:"avg_confidence"`
    AvgProcessing   float64            `json:"avg_processing_ms"`

    // Статистика по severity
    BySeverity      map[string]SeverityStats `json:"by_severity"`

    // Cache статистика
    CacheStats      CacheStats         `json:"cache_stats"`

    // LLM статистика
    LLMStats        LLMStats           `json:"llm_stats"`

    // Fallback статистика
    FallbackStats   FallbackStats      `json:"fallback_stats"`

    // Error статистика
    ErrorStats      ErrorStats         `json:"error_stats"`

    // Метаданные
    LastClassified  *time.Time         `json:"last_classified,omitempty"`
    Timestamp       time.Time          `json:"timestamp"`
}

type SeverityStats struct {
    Count         int64   `json:"count"`
    AvgConfidence float64 `json:"avg_confidence"`
    Percentage    float64 `json:"percentage,omitempty"`
}

type CacheStats struct {
    HitRate    float64 `json:"hit_rate"`
    L1Hits     int64   `json:"l1_cache_hits"`
    L2Hits     int64   `json:"l2_cache_hits"`
    Misses     int64   `json:"cache_misses"`
}

type LLMStats struct {
    Requests      int64   `json:"requests"`
    SuccessRate   float64 `json:"success_rate"`
    Failures      int64   `json:"failures"`
    AvgLatencyMs  float64 `json:"avg_latency_ms"`
    UsageRate     float64 `json:"usage_rate"`
}

type FallbackStats struct {
    Used          int64   `json:"used"`
    Rate          float64 `json:"rate"`
    AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

type ErrorStats struct {
    Total       int64      `json:"total"`
    Rate        float64    `json:"rate"`
    LastError   string     `json:"last_error,omitempty"`
    LastErrorTime *time.Time `json:"last_error_time,omitempty"`
}
```

### 1.3 Поток данных

#### Flow-1: Базовый запрос (без Prometheus)
```
1. HTTP Request → Handler.GetClassificationStats()
2. Handler → ClassificationService.GetStats()
3. Handler → Aggregate базовых метрик
4. Handler → Build StatsResponse
5. Handler → JSON Serialization
6. HTTP Response (200 OK)
```

**Время выполнения:** ~5-10ms

#### Flow-2: Расширенный запрос (с Prometheus)
```
1. HTTP Request → Handler.GetClassificationStats()
2. Handler → ClassificationService.GetStats() (async)
3. Handler → PrometheusClient.Query() (async, timeout 100ms)
4. Handler → Wait for both (with timeout)
5. Handler → Merge results
6. Handler → Aggregate by_severity из Prometheus
7. Handler → Build StatsResponse
8. Handler → JSON Serialization
9. HTTP Response (200 OK)
```

**Время выполнения:** ~50-150ms (зависит от Prometheus)

#### Flow-3: Graceful degradation
```
1. HTTP Request → Handler.GetClassificationStats()
2. Handler → ClassificationService.GetStats() ✅
3. Handler → PrometheusClient.Query() ❌ (timeout/error)
4. Handler → Log warning
5. Handler → Use only ClassificationService data
6. Handler → Build StatsResponse (partial data)
7. HTTP Response (200 OK, с предупреждением в логах)
```

## 2. Формат данных

### 2.1 Request Format

**Endpoint:** `GET /api/v2/classification/stats`

**Query Parameters:**
- `include_prometheus` (optional, bool, default: true) - включить Prometheus метрики
- `time_range` (optional, string, default: "all") - временной диапазон (all, last_24h, last_7d)

**Headers:**
- `Accept: application/json` (required)
- `X-Request-ID` (optional, для tracing)

### 2.2 Response Format

**Success Response (200 OK):**
```json
{
  "total_classified": 1180,
  "total_requests": 1250,
  "classification_rate": 0.944,
  "avg_confidence": 0.83,
  "avg_processing_ms": 45.2,
  "by_severity": {
    "critical": {
      "count": 85,
      "avg_confidence": 0.91,
      "percentage": 7.2
    },
    "warning": {
      "count": 650,
      "avg_confidence": 0.84,
      "percentage": 55.1
    },
    "info": {
      "count": 380,
      "avg_confidence": 0.78,
      "percentage": 32.2
    },
    "noise": {
      "count": 65,
      "avg_confidence": 0.88,
      "percentage": 5.5
    }
  },
  "cache_stats": {
    "hit_rate": 0.65,
    "l1_cache_hits": 450,
    "l2_cache_hits": 317,
    "cache_misses": 483
  },
  "llm_stats": {
    "requests": 483,
    "success_rate": 0.98,
    "failures": 10,
    "avg_latency_ms": 850.5,
    "usage_rate": 0.386
  },
  "fallback_stats": {
    "used": 10,
    "rate": 0.008,
    "avg_latency_ms": 2.3
  },
  "error_stats": {
    "total": 10,
    "rate": 0.008,
    "last_error": "LLM timeout after 5s",
    "last_error_time": "2025-01-17T10:30:00Z"
  },
  "last_classified": "2025-01-17T10:35:00Z",
  "timestamp": "2025-01-17T10:35:15Z"
}
```

**Error Response (500 Internal Server Error):**
```json
{
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Failed to aggregate statistics",
    "request_id": "abc-123-def"
  }
}
```

### 2.3 API Контракты

#### Contract-1: ClassificationService Interface
```go
type ClassificationService interface {
    GetStats() ClassificationStats
    // ... другие методы
}
```

**Использование:**
- Primary source для базовых метрик
- Всегда доступен (in-memory stats)
- Thread-safe

#### Contract-2: Prometheus Query Interface
```go
type PrometheusClient interface {
    Query(ctx context.Context, query string) (*PrometheusResult, error)
    QueryRange(ctx context.Context, query string, start, end time.Time, step time.Duration) (*PrometheusResult, error)
}
```

**Использование:**
- Optional enhancement для расширенных метрик
- Timeout: 100ms
- Graceful degradation при ошибках

## 3. Сценарии ошибок

### Error-1: ClassificationService недоступен
**Сценарий:** ClassificationService.GetStats() возвращает ошибку
**Обработка:**
- Log error с request ID
- Return 500 Internal Server Error
- Include error details в response

**Response:**
```json
{
  "error": {
    "code": "CLASSIFICATION_SERVICE_ERROR",
    "message": "Failed to retrieve classification statistics",
    "request_id": "abc-123-def"
  }
}
```

### Error-2: Prometheus недоступен
**Сценарий:** Prometheus query timeout или error
**Обработка:**
- Log warning (не error)
- Continue с ClassificationService данными
- Return 200 OK с partial data
- Set `prometheus_available: false` в response metadata

**Response:**
```json
{
  "total_classified": 1180,
  // ... базовые метрики
  "metadata": {
    "prometheus_available": false,
    "warnings": ["Prometheus metrics unavailable, using service stats only"]
  }
}
```

### Error-3: Invalid query parameters
**Сценарий:** Невалидные query parameters
**Обработка:**
- Return 400 Bad Request
- Include validation errors в response

**Response:**
```json
{
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid query parameter: time_range",
    "details": {
      "field": "time_range",
      "value": "invalid",
      "allowed_values": ["all", "last_24h", "last_7d", "last_30d"]
    },
    "request_id": "abc-123-def"
  }
}
```

### Error-4: Concurrent access race condition
**Сценарий:** Race condition при чтении stats
**Обработка:**
- ClassificationService.GetStats() уже thread-safe (RWMutex)
- Prometheus queries не требуют synchronization (read-only)
- Response building - atomic operation

**Prevention:**
- Использовать sync.RWMutex для кэширования (если требуется)
- Избегать shared mutable state

## 4. Edge Cases

### Edge-1: Zero requests
**Сценарий:** Нет ни одного запроса на классификацию
**Обработка:**
- Return 200 OK с нулевыми значениями
- Все метрики = 0 или null
- by_severity = empty map или zeros

### Edge-2: Все запросы из кэша
**Сценарий:** 100% cache hit rate, нет LLM запросов
**Обработка:**
- LLM stats = zeros или null
- Cache stats = 100% hit rate
- Fallback stats = zeros

### Edge-3: Все запросы через fallback
**Сценарий:** LLM недоступен, все через fallback
**Обработка:**
- LLM stats = failures = total requests
- Fallback stats = 100% usage
- Error stats = high rate

### Edge-4: Очень большое количество запросов
**Сценарий:** Миллионы запросов, overflow int64
**Обработка:**
- Использовать int64 (до 9,223,372,036,854,775,807)
- Проверка на overflow при агрегации
- Log warning при приближении к limit

## 5. Производительность

### 5.1 Оптимизации

#### Optimization-1: Кэширование ответа
**Стратегия:** In-memory cache с TTL 5-10 секунд
**Реализация:**
```go
type cachedStats struct {
    stats      *StatsResponse
    expiresAt  time.Time
    mu         sync.RWMutex
}

func (h *ClassificationHandlers) getCachedStats() (*StatsResponse, bool) {
    h.cache.mu.RLock()
    defer h.cache.mu.RUnlock()

    if time.Now().Before(h.cache.expiresAt) {
        return h.cache.stats, true
    }
    return nil, false
}
```

**Benefits:**
- Снижение нагрузки на ClassificationService
- Снижение нагрузки на Prometheus
- Улучшение latency для повторяющихся запросов

#### Optimization-2: Ленивая агрегация
**Стратегия:** Агрегировать только запрошенные метрики
**Реализация:**
- Query parameters определяют, какие метрики включать
- Пропускать Prometheus queries, если не требуется

#### Optimization-3: Parallel queries
**Стратегия:** Параллельные запросы к ClassificationService и Prometheus
**Реализация:**
```go
var baseStats core.ClassificationStats
var promStats *PrometheusStats
var baseErr, promErr error

var wg sync.WaitGroup
wg.Add(2)

go func() {
    defer wg.Done()
    baseStats = h.classifier.GetStats()
}()

go func() {
    defer wg.Done()
    ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
    defer cancel()
    promStats, promErr = h.promClient.Query(ctx, query)
}()

wg.Wait()
```

**Benefits:**
- Снижение total latency
- Независимость источников данных

#### Optimization-4: Zero allocations
**Стратегия:** Переиспользование структур, избежание лишних аллокаций
**Реализация:**
- Pre-allocate maps с известным размером
- Использовать sync.Pool для временных структур
- Избегать string concatenation

### 5.2 Benchmark Targets

**Baseline (current TODO stub):**
- Latency: ~1-2µs
- Allocations: 0-1 allocs/op

**Target (150% quality):**
- Latency: < 30µs (без Prometheus), < 150µs (с Prometheus)
- Allocations: < 5 allocs/op
- Throughput: > 10,000 req/s

## 6. Безопасность

### 6.1 Input Validation
- Query parameters validation
- Type checking (bool, string enums)
- Range checking (time_range values)

### 6.2 Output Sanitization
- Не возвращать sensitive data (credentials, tokens)
- Sanitize error messages (не раскрывать внутреннюю структуру)
- Rate limiting через middleware

### 6.3 Security Headers
- Применять через middleware stack:
  - X-Content-Type-Options: nosniff
  - X-Frame-Options: DENY
  - Content-Security-Policy
  - Strict-Transport-Security (если HTTPS)

## 7. Тестирование

### 7.1 Unit Tests
- Handler tests (mock ClassificationService)
- StatsAggregator tests (mock Prometheus)
- Response builder tests
- Error handling tests

### 7.2 Integration Tests
- End-to-end handler tests
- Prometheus integration tests (mock Prometheus server)
- Concurrent access tests

### 7.3 Benchmarks
- Handler latency benchmark
- Throughput benchmark
- Memory allocation benchmark

### 7.4 Test Coverage Target
- **Target:** > 85%
- **Stretch goal:** > 90% (150% quality)

## 8. Observability

### 8.1 Prometheus Metrics
```go
// Endpoint metrics
classification_stats_requests_total{status="200|500"}
classification_stats_duration_seconds{source="service|prometheus|both"}
classification_stats_prometheus_errors_total
```

### 8.2 Structured Logging
```go
h.logger.Info("Classification stats retrieved",
    "request_id", requestID,
    "total_classified", stats.TotalClassified,
    "cache_hit_rate", stats.CacheStats.HitRate,
    "duration_ms", duration.Milliseconds(),
    "prometheus_available", promAvailable)
```

### 8.3 Request Tracing
- Request ID в logs
- Request ID в response headers
- Correlation с ClassificationService logs

## 9. Документация

### 9.1 Code Documentation
- Godoc comments для всех публичных функций
- Примеры использования
- Edge cases documentation

### 9.2 API Documentation
- OpenAPI 3.0 specification
- Request/response examples
- Error scenarios

### 9.3 Integration Guide
- Как использовать endpoint
- Примеры запросов (curl, Go, Python)
- Troubleshooting guide

## 10. Миграция и совместимость

### 10.1 Backward Compatibility
- Существующий TODO stub возвращает пустые данные
- Новая реализация расширяет response, не ломает структуру
- Старые клиенты продолжат работать (дополнительные поля игнорируются)

### 10.2 Migration Path
1. Реализовать базовую функциональность (ClassificationService только)
2. Добавить Prometheus integration (опционально)
3. Добавить кэширование (оптимизация)
4. Добавить расширенные метрики (enhancement)

## 11. Альтернативные решения

### Alternative-1: Только ClassificationService
**Pros:**
- Простая реализация
- Нет зависимости от Prometheus
- Быстрый ответ

**Cons:**
- Ограниченные метрики
- Нет by_severity breakdown
- Нет исторических данных

**Решение:** Использовать как fallback, но добавить Prometheus для расширенных метрик

### Alternative-2: Только Prometheus
**Pros:**
- Богатые метрики
- Исторические данные
- Гибкая агрегация

**Cons:**
- Зависимость от Prometheus
- Медленнее
- Сложнее реализация

**Решение:** Использовать как optional enhancement, не как primary source

### Alternative-3: Database queries
**Pros:**
- Точные данные
- Исторические данные
- Гибкие запросы

**Cons:**
- Медленнее
- Нагрузка на БД
- Сложнее реализация

**Решение:** Не использовать для real-time stats, только для historical analysis (отдельный endpoint)

## 12. Решение

**Выбранное решение:** Hybrid approach
- **Primary source:** ClassificationService.GetStats() (быстро, надежно)
- **Enhancement:** Prometheus queries (расширенные метрики, опционально)
- **Caching:** In-memory cache с TTL 5-10s (оптимизация)
- **Graceful degradation:** Fallback на ClassificationService при ошибках Prometheus

**Обоснование:**
- Баланс между производительностью и функциональностью
- Надежность (работает даже без Prometheus)
- Расширяемость (можно добавить больше источников)
- Соответствует enterprise требованиям
