# TN-72: POST /classification/classify - Design Document

## 1. Архитектурный обзор

### 1.1 Цель
Реализовать REST API endpoint для ручной классификации алертов с поддержкой кэширования, fallback механизмов и comprehensive observability.

### 1.2 Архитектурные принципы
- **Separation of Concerns**: Handler → Service → Infrastructure
- **Dependency Injection**: все зависимости через конструкторы
- **Graceful Degradation**: fallback на rule-based классификацию
- **Observability First**: метрики, логи, трейсинг
- **Security by Design**: валидация, rate limiting, audit logging

### 1.3 Компонентная диаграмма

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request Layer                       │
│  POST /api/v2/classification/classify                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Middleware Stack:                                   │   │
│  │  - Recovery, RequestID, Logging, Metrics            │   │
│  │  - Auth (optional), RateLimit, Validation          │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                  Handler Layer                              │
│  ClassificationHandlers.ClassifyAlert()                     │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Responsibilities:                                    │   │
│  │  - Request parsing & validation                     │   │
│  │  - Response formatting                               │   │
│  │  - Error handling                                    │   │
│  │  - Metrics recording                                 │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────────────────┬─────────────────────────────────┘
                             │
                             ▼
┌─────────────────────────────────────────────────────────────┐
│                  Service Layer                               │
│  ClassificationService.ClassifyAlert()                      │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  Pipeline:                                           │   │
│  │  1. Check L1 Cache (memory)                          │   │
│  │  2. Check L2 Cache (Redis)                          │   │
│  │  3. Call LLM (if force=true or cache miss)          │   │
│  │  4. Fallback (if LLM unavailable)                  │   │
│  │  5. Save to cache                                    │   │
│  └──────────────────────────────────────────────────────┘   │
└───────────────┬───────────────────────┬─────────────────────┘
                │                       │
                ▼                       ▼
┌──────────────────────────┐  ┌──────────────────────────┐
│   Infrastructure Layer    │  │   Infrastructure Layer    │
│   LLM Client             │  │   Cache (Redis)            │
│   - HTTP Client          │  │   - L2 Cache Storage       │
│   - Circuit Breaker      │  │   - TTL Management        │
│   - Retry Logic          │  └──────────────────────────┘
└──────────────────────────┘
```

## 2. Детальный дизайн компонентов

### 2.1 Handler Layer

#### 2.1.1 ClassificationHandlers.ClassifyAlert()

**Сигнатура:**
```go
func (h *ClassificationHandlers) ClassifyAlert(
    w http.ResponseWriter,
    r *http.Request,
)
```

**Алгоритм:**
1. **Parse Request**: декодирование JSON в `ClassifyRequest`
2. **Validate Input**: валидация структуры и полей алерта
3. **Extract Force Flag**: проверка параметра `force` (default: false)
4. **Create Context**: создание context с таймаутом (5s)
5. **Call Service**: вызов `ClassificationService.ClassifyAlert()` или `GetCachedClassification()`
6. **Format Response**: форматирование результата в `ClassifyResponse`
7. **Record Metrics**: запись метрик (duration, status, cache hit/miss)
8. **Send Response**: отправка JSON ответа

**Обработка ошибок:**
- **400 Bad Request**: ошибки валидации, невалидный JSON
- **429 Too Many Requests**: превышен rate limit
- **500 Internal Server Error**: ошибка классификации
- **503 Service Unavailable**: LLM недоступен, circuit breaker открыт

**Метрики:**
- `classification_api_requests_total{status, method}`
- `classification_api_duration_seconds{method}`
- `classification_api_cache_hits_total`
- `classification_api_cache_misses_total`
- `classification_api_errors_total{error_type}`

#### 2.1.2 Request/Response Models

**ClassifyRequest:**
```go
type ClassifyRequest struct {
    Alert *core.Alert `json:"alert" validate:"required"`
    Force bool        `json:"force,omitempty"` // Default: false
}
```

**ClassifyResponse:**
```go
type ClassifyResponse struct {
    Result         *core.ClassificationResult `json:"result"`
    ProcessingTime string                     `json:"processing_time"` // e.g., "50ms", "1.2s"
    Cached         bool                       `json:"cached"`         // Was result from cache?
    Model          string                     `json:"model,omitempty"` // LLM model used
    Timestamp      time.Time                  `json:"timestamp"`
}
```

### 2.2 Service Layer

#### 2.2.1 ClassificationService Integration

**Использование существующего сервиса:**
- `ClassificationService.ClassifyAlert(ctx, alert)` - основная классификация
- `ClassificationService.GetCachedClassification(ctx, fingerprint)` - получение из кэша
- `ClassificationService.InvalidateCache(ctx, fingerprint)` - инвалидация кэша (при force=true)

**Логика обработки force flag:**
```go
if force {
    // Invalidate cache first
    service.InvalidateCache(ctx, alert.Fingerprint)
    // Force new classification
    result, err := service.ClassifyAlert(ctx, alert)
} else {
    // Try cache first
    if cached, err := service.GetCachedClassification(ctx, alert.Fingerprint); err == nil {
        return cached, nil
    }
    // Cache miss - classify
    result, err := service.ClassifyAlert(ctx, alert)
}
```

### 2.3 Validation Layer

#### 2.3.1 Input Validation

**Структурная валидация (validator/v10):**
- `alert` поле обязательное (required)
- `alert.fingerprint` обязательное, непустое
- `alert.alert_name` обязательное, непустое
- `alert.status` обязательное, одно из: firing, resolved
- `alert.starts_at` обязательное, валидная дата
- `alert.generator_url` опциональное, валидный URL (если указан)

**Бизнес-валидация:**
- Fingerprint должен быть непустым и валидным (SHA-256 формат)
- Labels и annotations должны быть валидными map[string]string
- Timestamp не должен быть в будущем

**Валидация через middleware:**
- Request size limit: максимум 100KB
- Content-Type: application/json
- JSON syntax validation

### 2.4 Caching Strategy

#### 2.4.1 Two-Tier Cache

**L1 Cache (Memory):**
- Тип: `sync.Map` (thread-safe)
- Capacity: ~1000 записей (LRU eviction)
- TTL: 1 час (настраиваемо)
- Key: `alert.Fingerprint`

**L2 Cache (Redis):**
- Тип: Redis SET/GET с JSON serialization
- TTL: 24 часа (настраиваемо)
- Key: `classification:{fingerprint}`
- Graceful degradation: работает без Redis (только L1)

**Cache Flow:**
```
Request → Check L1 → Hit? → Return
                    ↓ Miss
                  Check L2 → Hit? → Save to L1 → Return
                              ↓ Miss
                            Classify → Save to L1+L2 → Return
```

#### 2.4.2 Cache Invalidation

**При force=true:**
1. Invalidate L1 cache (remove from sync.Map)
2. Invalidate L2 cache (delete from Redis)
3. Force new classification
4. Save new result to cache

### 2.5 Error Handling

#### 2.5.1 Error Types

**Validation Errors (400):**
- `ErrInvalidRequest`: невалидный JSON
- `ErrMissingAlert`: отсутствует поле alert
- `ErrInvalidFingerprint`: невалидный fingerprint
- `ErrInvalidStatus`: невалидный status
- `ErrInvalidTimestamp`: невалидная дата

**Service Errors (500/503):**
- `ErrClassificationFailed`: ошибка классификации
- `ErrLLMUnavailable`: LLM недоступен
- `ErrCircuitBreakerOpen`: circuit breaker открыт
- `ErrTimeout`: таймаут классификации

**Rate Limit Errors (429):**
- `ErrRateLimitExceeded`: превышен rate limit

#### 2.5.2 Error Response Format

```go
type ErrorResponse struct {
    Error     string    `json:"error"`
    Message   string    `json:"message"`
    RequestID string    `json:"request_id"`
    Timestamp time.Time `json:"timestamp"`
    Details   map[string]interface{} `json:"details,omitempty"`
}
```

### 2.6 Observability

#### 2.6.1 Prometheus Metrics

**Request Metrics:**
- `classification_api_requests_total{status="200|400|500|503", method="POST"}` - Counter
- `classification_api_duration_seconds{method="POST"}` - Histogram (buckets: 0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2, 5)

**Cache Metrics:**
- `classification_api_cache_hits_total{level="l1|l2"}` - Counter
- `classification_api_cache_misses_total` - Counter

**Error Metrics:**
- `classification_api_errors_total{error_type="validation|service|timeout"}` - Counter

**LLM Metrics (from ClassificationService):**
- `alert_history_business_llm_classifications_total{severity}`
- `alert_history_business_llm_confidence_score`
- `alert_history_business_classification_duration_seconds{source="llm|cache|fallback"}`

#### 2.6.2 Structured Logging

**Log Levels:**
- **DEBUG**: детальная информация о запросах (fingerprint, force flag)
- **INFO**: успешные классификации (severity, confidence, cached)
- **WARN**: fallback использование, cache misses
- **ERROR**: ошибки классификации, недоступность LLM

**Log Fields:**
- `request_id`: уникальный ID запроса
- `fingerprint`: fingerprint алерта
- `severity`: результат классификации
- `confidence`: confidence score
- `cached`: был ли результат из кэша
- `duration_ms`: время обработки
- `error`: ошибка (если есть)

#### 2.6.3 Distributed Tracing

**Опционально (если OpenTelemetry доступен):**
- Span: `classification.classify`
- Tags: `fingerprint`, `force`, `cached`, `severity`, `confidence`
- Events: `cache_hit`, `cache_miss`, `llm_call`, `fallback`

## 3. API Contract

### 3.1 Request

**Endpoint:** `POST /api/v2/classification/classify`

**Headers:**
```
Content-Type: application/json
X-API-Key: <api_key> (optional, if auth enabled)
```

**Body:**
```json
{
  "alert": {
    "fingerprint": "abc123...",
    "alert_name": "DatabaseDown",
    "status": "firing",
    "labels": {
      "severity": "critical",
      "namespace": "production"
    },
    "annotations": {
      "summary": "Database is not responding"
    },
    "starts_at": "2025-11-17T10:00:00Z",
    "generator_url": "https://prometheus.example.com/..."
  },
  "force": false
}
```

### 3.2 Response

**Success (200 OK):**
```json
{
  "result": {
    "severity": "critical",
    "confidence": 0.95,
    "reasoning": "Database outage in production environment requires immediate attention",
    "recommendations": [
      "Check database logs",
      "Verify network connectivity",
      "Escalate to database team"
    ],
    "processing_time": 0.05
  },
  "processing_time": "50ms",
  "cached": false,
  "model": "gpt-4",
  "timestamp": "2025-11-17T10:00:00Z"
}
```

**Error (400 Bad Request):**
```json
{
  "error": "validation_error",
  "message": "Invalid request: alert.fingerprint is required",
  "request_id": "req-12345",
  "timestamp": "2025-11-17T10:00:00Z",
  "details": {
    "field": "alert.fingerprint",
    "reason": "required"
  }
}
```

**Error (503 Service Unavailable):**
```json
{
  "error": "service_unavailable",
  "message": "LLM service is currently unavailable. Using fallback classification.",
  "request_id": "req-12345",
  "timestamp": "2025-11-17T10:00:00Z"
}
```

### 3.3 OpenAPI 3.0 Specification

**См. отдельный файл:** `openapi-classification-classify.yaml`

## 4. Интеграция с существующими компонентами

### 4.1 Router Integration

**Файл:** `go-app/internal/api/router.go`

**Изменения:**
```go
func setupClassificationRoutes(router *mux.Router, config RouterConfig) {
    // ... existing code ...

    // Replace placeholder with actual handler
    if config.ClassificationHandlers != nil {
        classProtected.HandleFunc("/classify",
            config.ClassificationHandlers.ClassifyAlert).Methods("POST")
    } else {
        classProtected.HandleFunc("/classify",
            PlaceholderHandler("ClassifyAlert")).Methods("POST")
    }
}
```

### 4.2 Main.go Integration

**Файл:** `go-app/cmd/server/main.go`

**Инициализация:**
```go
// Create ClassificationService (if not exists)
classificationService := services.NewClassificationService(...)

// Create ClassificationHandlers
classificationHandlers := classification.NewClassificationHandlersWithService(
    classificationService,
    classificationService,
    logger,
)

// Register in router config
routerConfig := api.RouterConfig{
    ClassificationHandlers: classificationHandlers,
    // ... other config ...
}
```

### 4.3 Middleware Stack

**Порядок middleware:**
1. Recovery (panic recovery)
2. RequestID (генерация request ID)
3. Logging (structured logging)
4. Metrics (Prometheus metrics)
5. CORS (если нужно)
6. Compression (gzip)
7. Auth (если включено)
8. RateLimit (rate limiting)
9. Validation (request validation)
10. Timeout (request timeout)

## 5. Тестирование

### 5.1 Unit Tests

**Покрытие:**
- Handler логика (parsing, validation, response formatting)
- Error handling (все типы ошибок)
- Force flag логика
- Cache invalidation

**Файлы:**
- `handlers_test.go`: unit tests для handlers
- `validation_test.go`: тесты валидации
- `error_test.go`: тесты обработки ошибок

### 5.2 Integration Tests

**Покрытие:**
- End-to-end flow (request → service → response)
- Cache integration (L1 + L2)
- LLM integration (success + failure)
- Fallback integration

**Файлы:**
- `integration_test.go`: integration tests
- `cache_integration_test.go`: тесты кэша
- `llm_integration_test.go`: тесты LLM

### 5.3 Performance Tests

**Benchmarks:**
- Handler performance (cache hit/miss scenarios)
- Validation performance
- Serialization performance

**Файлы:**
- `bench_test.go`: benchmarks

### 5.4 Security Tests

**Покрытие:**
- Input validation (injection attacks)
- Rate limiting
- Authentication (если включено)
- Request size limits

**Файлы:**
- `security_test.go`: security tests

## 6. Производительность

### 6.1 Целевые показатели

**Cache Hit (L1):**
- p50: < 1ms
- p95: < 5ms
- p99: < 10ms

**Cache Hit (L2):**
- p50: < 10ms
- p95: < 50ms
- p99: < 100ms

**Cache Miss + LLM:**
- p50: < 500ms
- p95: < 2s
- p99: < 5s

**Fallback:**
- p50: < 5ms
- p95: < 10ms
- p99: < 20ms

### 6.2 Оптимизации

**Handler Level:**
- JSON parsing optimization (streaming для больших тел)
- Response pooling (reuse buffers)
- Early validation (fail fast)

**Service Level:**
- Async cache writes (не блокируют ответ)
- Connection pooling для LLM client
- Batch cache operations (если нужно)

## 7. Безопасность

### 7.1 Input Validation

**Защита от:**
- JSON injection
- Path traversal (в generator_url)
- XSS (в labels/annotations)
- DoS (request size limits)

**Валидация:**
- Structural validation (validator/v10)
- Business rule validation
- Sanitization (если нужно)

### 7.2 Rate Limiting

**Стратегия:**
- Token bucket algorithm
- Per-IP limiting (100 req/min)
- Global limiting (1000 req/min)

**Реализация:**
- Middleware: `RateLimitMiddleware`
- Configurable limits через config

### 7.3 Audit Logging

**Логирование:**
- Все запросы (request ID, fingerprint, force flag)
- Все ошибки (error type, message)
- Security events (rate limit hits, auth failures)

## 8. Мониторинг и алертинг

### 8.1 Prometheus Alerts

**Алерты:**
- `HighErrorRate`: error rate > 1%
- `HighLatency`: p95 latency > 2s
- `LLMUnavailable`: LLM unavailable > 5min
- `CacheHitRateLow`: cache hit rate < 50%

### 8.2 Grafana Dashboard

**Панели:**
- Request rate (req/s)
- Latency (p50, p95, p99)
- Error rate (%)
- Cache hit rate (%)
- LLM usage rate (%)

## 9. Развертывание

### 9.1 Конфигурация

**Environment Variables:**
- `CLASSIFICATION_TIMEOUT`: таймаут классификации (default: 5s)
- `CLASSIFICATION_CACHE_TTL`: TTL кэша (default: 24h)
- `CLASSIFICATION_RATE_LIMIT`: rate limit (default: 100/min)

### 9.2 Миграция

**План:**
1. Deploy handler (backward compatible)
2. Enable endpoint (feature flag)
3. Monitor metrics
4. Gradual rollout (10% → 50% → 100%)

---

**Версия документа:** 1.0
**Дата создания:** 2025-11-17
**Автор:** AI Assistant
**Статус:** Draft → Ready for Review
