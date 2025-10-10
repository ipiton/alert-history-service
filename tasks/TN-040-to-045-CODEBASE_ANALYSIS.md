# TN-040 to TN-045: Deep Codebase Analysis

**Дата**: 2025-10-10
**Phase**: 2 - Deep Codebase Analysis Complete

---

## 2.1 Анализ существующих retry реализаций

### 🔍 Обнаруженные реализации

#### 1. `internal/database/postgres/retry.go` - **Production-ready** ⭐⭐⭐⭐⭐

**Компоненты**:
```go
type RetryConfig struct {
    MaxRetries    int           // 3
    InitialDelay  time.Duration // 100ms
    MaxDelay      time.Duration // 5s
    BackoffFactor float64       // 2.0
    JitterFactor  float64       // 0.1
}

type RetryExecutor struct {
    config RetryConfig
    logger *slog.Logger
}
```

**Методы**:
- `Execute(ctx, operation func() error) error`
- `ExecuteWithResult(ctx, operation func() (interface{}, error)) (interface{}, error)`

**Возможности**:
- ✅ Exponential backoff
- ✅ Jitter (configurable factor)
- ✅ Context cancellation support
- ✅ RetryableError checking через `IsRetryable(err)` from `errors.go`
- ✅ Structured logging
- ✅ `shouldRetry()` - intelligent error checking

**Качество**: 🏆 **EXCELLENT**
- Полностью реализованы все требования TN-040
- Использует best practices
- Production-tested

**Проблема**: 🔴 Находится в `postgres/` package - нарушает separation of concerns

---

#### 2. `internal/infrastructure/llm/client.go` - **Inline retry** ⭐⭐⭐

**Метод**:
```go
func (c *HTTPLLMClient) classifyAlertWithRetry(ctx, alert) (*ClassificationResult, error)
```

**Возможности**:
- ✅ Exponential backoff (`retryDelay * RetryBackoff`)
- ✅ Context support
- ❌ Нет jitter
- ✅ `IsRetryableError()` - sophisticated error classification

**Качество**: ⭐⭐⭐ **GOOD**
- Работает, но дублирует логику из `postgres/retry.go`
- Менее конфигурируемо

**Проблема**: 🟡 Дублирование кода

---

#### 3. `internal/infrastructure/migrations/errors.go` - **Basic retry** ⭐⭐

**Метод**:
```go
func (eh *ErrorHandler) ExecuteWithRetry(ctx, operation) error
```

**Возможности**:
- ✅ Базовый retry loop
- ✅ Configurable maxRetries, retryDelay
- ❌ Нет exponential backoff
- ❌ Нет jitter
- ✅ Собственная `isRetryable()` логика

**Качество**: ⭐⭐ **BASIC**
- Простейшая реализация
- Отсутствуют advanced features

**Проблема**: 🟡 Дублирование кода + incomplete implementation

---

#### 4. `internal/infrastructure/lock/distributed.go` - **Specialized retry** ⭐⭐⭐

**Метод**:
```go
func (l *DistributedLock) AcquireWithRetry(ctx, maxRetries) (bool, error)
```

**Возможности**:
- ✅ Retry loop для distributed lock
- ✅ `retryInterval()` - exponential-like backoff
- ❌ Специфично для locks

**Качество**: ⭐⭐⭐ **SPECIALIZED**
- Хорошо для своего use case
- Не универсально

**Проблема**: 🟡 Нельзя переиспользовать

---

### 📊 Сравнительная таблица

| Feature | postgres/retry | llm/client | migrations | lock |
|---------|---------------|------------|------------|------|
| Exponential backoff | ✅ | ✅ | ❌ | ⚠️ |
| Jitter | ✅ | ❌ | ❌ | ❌ |
| Context support | ✅ | ✅ | ✅ | ✅ |
| Error classification | ✅ | ✅ | ⚠️ | ⚠️ |
| Configurable | ✅ | ⚠️ | ⚠️ | ❌ |
| Logging | ✅ | ✅ | ❌ | ❌ |
| Generic/Reusable | ❌ | ❌ | ❌ | ❌ |

---

### 🎯 Вывод для TN-040

**Лучшая база**: `postgres/retry.go` - наиболее полная реализация

**Plan рефакторинга**:
1. ✅ Взять `postgres/retry.go` как baseline
2. 🔄 Переместить в `internal/core/resilience/retry.go` (domain layer)
3. ✅ Добавить `RetryableErrorChecker` interface:
   ```go
   type RetryableErrorChecker interface {
       IsRetryable(err error) bool
   }
   ```
4. ✅ Интегрировать метрики
5. ✅ Обновить все 4 места использования:
   - `postgres/` → use core/resilience
   - `llm/client.go` → use core/resilience
   - `migrations/` → use core/resilience
   - `lock/` → keep specialized (or optionally refactor)

**Deleted LOC**: ~200 строк дублирующегося кода
**Added LOC**: ~250 строк универсального retry module

---

## 2.2 Анализ webhook handling

### 🔍 Текущее состояние

**Файл**: `cmd/server/handlers/webhook.go`

**Структуры**:
```go
type WebhookRequest struct {
    AlertName    string
    Status       string
    Labels       map[string]string
    Annotations  map[string]string
    StartsAt     string
    EndsAt       string
    GeneratorURL string
    Fingerprint  string
    Extra        map[string]interface{} // unused
}

type WebhookHandlers struct {
    processor AlertProcessor
    logger    *slog.Logger
}
```

**Методы**:
- `HandleWebhook(w, r)` - HTTP handler
- `webhookRequestToAlert(req)` - converter

---

### ✅ Что уже есть

1. **Базовая обработка**:
   - POST request handling
   - JSON unmarshaling
   - Body reading

2. **Базовая валидация**:
   - Method check (POST only)
   - AlertName required field check
   - JSON parsing

3. **Timestamp parsing**:
   - RFC3339 format
   - Fallback to `time.Now()` on error

4. **Status parsing**:
   - "resolved" → core.StatusResolved
   - Other → core.StatusFiring

5. **Fingerprint generation**:
   - Uses provided fingerprint
   - Fallback: `{alertName}_{timestamp}`

6. **Integration**:
   - Uses existing `AlertProcessor`
   - Structured logging

---

### ❌ Что отсутствует (для TN-41, TN-42, TN-43)

1. **Alertmanager format support** (TN-41):
   - ❌ `GroupKey`, `TruncatedAlerts`, `Receiver`
   - ❌ `CommonLabels`, `CommonAnnotations`, `GroupLabels`
   - ❌ `ExternalURL`, `Version`
   - ❌ `Alerts` array (multiple alerts in one webhook)

2. **Auto-detection** (TN-42):
   - ❌ Нет механизма определения формата
   - ❌ Только один parser (simple format)
   - ❌ Нет routing к разным parsers

3. **Comprehensive validation** (TN-43):
   - ❌ Только одно поле проверяется (alertname)
   - ❌ Нет schema validation
   - ❌ Нет format validation (timestamp formats, label names, etc.)
   - ❌ Нет business rules (severity values, confidence range)
   - ❌ Нет detailed error messages

4. **Edge cases handling**:
   - ⚠️ Invalid timestamps → fallback to now (good)
   - ❌ Empty arrays не обрабатываются
   - ❌ Нет обработки malformed JSON (просто возвращается generic error)
   - ❌ Нет rate limiting
   - ❌ Нет request size limits

5. **Metrics**:
   - ❌ Нет webhook-специфичных метрик
   - ❌ Нет timing metrics
   - ❌ Нет error rate tracking

---

### 🎯 Выводы для TN-41, TN-42, TN-43

**TN-41 (Parser)**:
- Нужно создать `AlertmanagerWebhook` struct с ПОЛНЫМИ полями
- Нужно создать parser interface с multiple implementations
- Можно переиспользовать `webhookRequestToAlert()` для simple format

**TN-42 (Universal Handler)**:
- Существующий handler хорош как baseline
- Добавить auto-detection перед parsing
- Создать map[WebhookType]Parser для routing
- Backward compatibility: старый endpoint остается

**TN-43 (Validation)**:
- Вынести validation в отдельный validator
- Использовать `go-playground/validator` для struct tags
- Custom validators для business rules
- Detailed ValidationError с Field, Message, Value

---

## 2.3 Анализ metrics infrastructure

### 🔍 Текущая архитектура (из TN-181)

**Файл**: `pkg/metrics/registry.go`

**Taxonomy**:
```
alert_history_<category>_<subsystem>_<metric_name>_<unit>
```

**Categories**:
- **Business**: `alert_history_business_*`
- **Technical**: `alert_history_technical_*`
- **Infrastructure**: `alert_history_infra_*`

**Существующие subsystems**:
```go
// Business
business_alerts_*
business_llm_*
business_publishing_*

// Technical
technical_http_*
technical_llm_cb_*     // Circuit Breaker
technical_enrichment_*
technical_filter_*

// Infrastructure
infra_db_*
infra_repository_*
infra_cache_*
```

---

### ✅ Что уже есть

1. **MetricsRegistry** (singleton):
   ```go
   type MetricsRegistry struct {
       business  *BusinessMetrics
       technical *TechnicalMetrics
       infra     *InfraMetrics
   }
   ```

2. **Lazy initialization**:
   - `sync.Once` для каждой category
   - `DefaultRegistry()` - глобальный singleton

3. **Helper methods**:
   - `Business().RecordAlertProcessed()`
   - `Technical().RecordHTTPRequest()`
   - `Infra().RecordDBQuery()`

4. **Prometheus integration**:
   - `promauto` для auto-registration
   - Все metrics exported через `/metrics` endpoint

---

### ❌ Webhook metrics отсутствуют

**Нужно для TN-045**:
```go
// В TechnicalMetrics struct добавить:
type TechnicalMetrics struct {
    // ... existing ...

    // Webhook subsystem
    WebhookRequestsTotal     *prometheus.CounterVec   // labels: type, status
    WebhookDurationSeconds   *prometheus.HistogramVec // labels: type
    WebhookProcessingSeconds *prometheus.HistogramVec // labels: type, stage
    WebhookQueueSize         prometheus.Gauge
    WebhookActiveWorkers     prometheus.Gauge
    WebhookErrorsTotal       *prometheus.CounterVec   // labels: type, error_type
}
```

**Naming convention** (FIXED от TN-045 design):
```
alert_history_technical_webhook_requests_total{type="alertmanager", status="success"}
alert_history_technical_webhook_duration_seconds{type="alertmanager"}
alert_history_technical_webhook_processing_seconds{type="alertmanager", stage="parse"}
alert_history_technical_webhook_queue_size
alert_history_technical_webhook_active_workers
alert_history_technical_webhook_errors_total{type="alertmanager", error_type="parse_error"}
```

---

### 🎯 Выводы для TN-045

**План реализации**:
1. ✅ Расширить `pkg/metrics/technical.go` (НЕ создавать отдельный файл)
2. ✅ Добавить webhook metrics в `TechnicalMetrics` struct
3. ✅ Lazy initialization через `technicalOnce`
4. ✅ Helper methods: `RecordWebhookRequest()`, `RecordWebhookError()`, etc.
5. ✅ Использовать unified taxonomy (alert_history_technical_webhook_*)
6. ✅ Интеграция в handlers через `metrics.DefaultRegistry().Technical()`

**Не делать**:
- ❌ Создавать `internal/core/metrics/webhook.go` (wrong location)
- ❌ Создавать отдельный WebhookMetrics registry
- ❌ Использовать старые names типа `webhook_requests_total`

---

## 2.4 Дублирующийся код - сводка

### 🔴 HIGH Duplication

**Retry Logic**:
- **Locations**: 4 файла
- **Duplicated LOC**: ~200 строк
- **Impact**: HIGH - используется везде
- **Refactor priority**: 🔥 **CRITICAL**

**Timestamp Parsing**:
- **Locations**: 2 файла (webhook.go, parser будущий)
- **Duplicated LOC**: ~10-15 строк
- **Impact**: LOW
- **Refactor priority**: ⚠️ MEDIUM

---

### 🟡 MEDIUM Duplication

**Status Parsing**:
- **Locations**: 2 файла
- **Duplicated LOC**: ~5 строк
- **Impact**: LOW
- **Refactor priority**: ✅ LOW (можно в helper function)

---

### ✅ LOW Duplication

**Validation**:
- **Locations**: Currently 1 (будет 2 после TN-43)
- **Duplicated LOC**: 0 (будет ~20 после TN-43 если не сделать validator)
- **Impact**: MEDIUM
- **Refactor priority**: ✅ PREVENT (сразу делать validator)

---

## Phase 2 Complete - Ключевые выводы

### 📊 Обнаруженные проблемы

1. **Retry logic duplication** - 4 реализации
2. **Webhook handling** - отсутствуют критические компоненты
3. **Metrics** - отсутствует webhook subsystem
4. **Validation** - отсутствует comprehensive validator

### 🎯 Приоритеты для реализации

**Критичные** (Phase 4-5):
1. TN-040 - Retry module (foundation для всех)
2. TN-045 - Metrics (observability с первого дня)

**Важные** (Phase 6-7):
3. TN-043 - Validator (нужен для TN-041)
4. TN-041 - Parser (основа для TN-042)

**Желательные** (Phase 8-9):
5. TN-042 - Universal Handler
6. TN-044 - Async Processing

### ✅ Готовность к Phase 3

- [x] Retry implementations analyzed
- [x] Webhook handling analyzed
- [x] Metrics infrastructure analyzed
- [x] Duplication identified
- [x] Refactoring plan created

**Next**: Phase 3 - Architecture Validation
