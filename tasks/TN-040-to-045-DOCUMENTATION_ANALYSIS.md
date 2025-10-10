# TN-040 to TN-045: Documentation Analysis & Validation

**Дата анализа**: 2025-10-10
**Исполнитель**: AI Assistant
**Статус**: Phase 1 - Documentation Audit Complete

---

## Матрица зависимостей

| Задача | Название | Зависит от | Может выполняться параллельно с | Приоритет |
|--------|----------|------------|----------------------------------|-----------|
| TN-040 | Retry Logic | - | TN-045, TN-043 | **HIGH** (foundation) |
| TN-041 | Alertmanager Parser | TN-043 | - | MEDIUM |
| TN-042 | Universal Handler | TN-041, TN-043 | - | MEDIUM |
| TN-043 | Webhook Validation | - | TN-040, TN-045 | **HIGH** (foundation) |
| TN-044 | Async Processing | TN-040, TN-042 | - | LOW (nice-to-have) |
| TN-045 | Webhook Metrics | - | TN-040, TN-043 | **HIGH** (observability) |

**Оптимальный порядок выполнения:**
1. **Parallel batch 1**: TN-040 (Retry) + TN-045 (Metrics) + TN-043 (Validation)
2. **Sequential**: TN-041 (Parser) - requires TN-043
3. **Sequential**: TN-042 (Universal Handler) - requires TN-041, TN-043
4. **Sequential**: TN-044 (Async) - requires TN-040, TN-042

**Critical path**: TN-043 → TN-041 → TN-042 → TN-044 (4 задачи sequential)

---

## Валидация документации

### ✅ TN-040: Retry Logic с Exponential Backoff

**Соответствие requirements.md ↔ design.md**: ✅ **PASS**

**Requirements:**
- Exponential backoff ✅
- Jitter для thundering herd ✅
- Configurable retry policies ✅
- Context cancellation support ✅

**Design:**
- `RetryPolicy` struct с MaxRetries, BaseDelay, MaxDelay, Multiplier, Jitter
- `WithRetry()` function для оборачивания операций
- Context cancellation через `select` statement

**Проблемы**: Нет

**Рекомендации**:
- ✅ Design полностью соответствует requirements
- Добавить в design:
  - RetryableErrorChecker interface для определения retryable errors
  - Metrics интеграцию (retry_attempts_total, retry_success_total, etc.)
  - Примеры использования с HTTP clients, database operations

---

### ⚠️ TN-041: Alertmanager Webhook Parser

**Соответствие requirements.md ↔ design.md**: ⚠️ **PARTIAL**

**Requirements:**
- Полная поддержка Alertmanager format ✅
- Валидация входных данных ✅
- Error handling для malformed data ✅
- Support для различных версий ⚠️ (не упомянуто в design)

**Design:**
- `AlertmanagerWebhook` struct с основными полями
- `WebhookParser` interface: Parse, Validate, ConvertToDomain
- НО: отсутствует `AlertmanagerAlert` struct definition в design.md

**Проблемы**:
- ❌ В design.md нет полного определения `AlertmanagerAlert` struct
- ❌ Не указано как обрабатывать разные версии Alertmanager (v0.24, v0.25, v0.26)
- ❌ Нет примеров edge cases (truncatedAlerts > 0, empty alerts array)

**Рекомендации**:
1. Добавить в design.md полное определение `AlertmanagerAlert`:
   ```go
   type AlertmanagerAlert struct {
       Status       string            `json:"status"`
       Labels       map[string]string `json:"labels"`
       Annotations  map[string]string `json:"annotations"`
       StartsAt     time.Time         `json:"startsAt"`
       EndsAt       time.Time         `json:"endsAt"`
       GeneratorURL string            `json:"generatorURL"`
       Fingerprint  string            `json:"fingerprint"`
   }
   ```
2. Добавить версионирование: поле `Version` в WebhookParser
3. Документировать edge cases и их обработку

---

### 🔴 TN-042: Universal Webhook Handler

**Соответствие requirements.md ↔ design.md**: 🔴 **FAIL** - Design устарел!

**Requirements:**
- Auto-detection формата payload ✅
- Support Alertmanager, generic webhooks ✅
- Routing к parsers ✅
- Error handling и logging ✅

**Design:**
- `WebhookHandler` struct с map parsers
- `HandleWebhook()` method с auto-detection
- **ПРОБЛЕМА**: Design использует `fiber.Ctx` (Fiber framework)

**Критическая проблема**:
```go
func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error  // ❌ УСТАРЕЛО!
```

**Факт**: Проект использует **net/http**, НЕ Fiber!
- Текущий код: `func HandleWebhook(w http.ResponseWriter, r *http.Request)`
- Design показывает Fiber API (c.Body(), c.Status(), c.JSON())

**Рекомендации**:
1. ⚠️ **CRITICAL**: Обновить design.md на использование net/http:
   ```go
   func (h *WebhookHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) error
   ```
2. Заменить все Fiber API calls на net/http equivalents:
   - `c.Body()` → `io.ReadAll(r.Body)`
   - `c.Status(400).JSON(...)` → `w.WriteHeader(400); json.NewEncoder(w).Encode(...)`
   - `c.Context()` → `r.Context()`
3. Убрать `fiber` import из всех design документов

---

### ✅ TN-043: Webhook Validation & Error Handling

**Соответствие requirements.md ↔ design.md**: ✅ **PASS**

**Requirements:**
- Schema validation ✅
- Required fields checking ✅
- Format validation ✅
- Detailed error messages ✅

**Design:**
- `WebhookValidator` interface: ValidateAlertmanager, ValidateGeneric
- `ValidationError` struct с Field, Message, Value
- `ValidationResult` struct с Valid bool + Errors array

**Проблемы**: Нет

**Рекомендации**:
- ✅ Design соответствует requirements
- Добавить в design:
  - Интеграцию с `go-playground/validator` (struct tags)
  - Custom validators (severity values, confidence range, etc.)
  - Примеры validation rules
  - Internationalization (русские/английские messages)

---

### ✅ TN-044: Async Webhook Processing

**Соответствие requirements.md ↔ design.md**: ✅ **PASS**

**Requirements:**
- Worker pool ✅
- Queue для задач ✅
- Retry для failed jobs ✅
- Monitoring ✅

**Design:**
- `WebhookProcessor` interface: SubmitJob, Start, Stop, Stats
- `WebhookJob` struct с ID, Type, Payload, CreatedAt, Attempts
- `webhookProcessor` struct с workers, jobQueue, workerPool, quit, wg

**Проблемы**: Нет критических

**Замечания**:
- ⚠️ Design не показывает интеграцию с TN-040 retry logic
- ⚠️ Нет упоминания Dead Letter Queue для permanently failed jobs

**Рекомендации**:
1. Добавить в design.md:
   ```go
   type webhookProcessor struct {
       retry       *resilience.RetryPolicy  // TN-040 integration
       dlq         chan *WebhookJob         // Dead letter queue
   }
   ```
2. Документировать graceful shutdown механизм
3. Добавить метрики: queue_size, active_workers, processing_time

---

### 🔴 TN-045: Webhook Metrics & Monitoring

**Соответствие requirements.md ↔ design.md**: 🔴 **FAIL** - Naming convention устарела!

**Requirements:**
- Request rate metrics ✅
- Processing time histograms ✅
- Error rate tracking ✅
- Queue size monitoring ✅

**Design:**
- `WebhookMetrics` struct с prometheus metrics
- Метрики: RequestsTotal, RequestDuration, ProcessingTime, QueueSize, ActiveWorkers, ErrorsTotal

**Критическая проблема**:
```go
Name: "webhook_requests_total"  // ❌ НЕ СООТВЕТСТВУЕТ taxonomy!
```

**Факт**: Проект использует **unified taxonomy** из TN-181:
- **Правильный формат**: `alert_history_<category>_<subsystem>_<metric_name>_<unit>`
- **Для webhook метрик**: `alert_history_technical_webhook_requests_total`

**Текущие метрики в проекте** (из TN-181):
- Business: `alert_history_business_alerts_processed_total`
- Technical: `alert_history_technical_http_requests_total`
- Infra: `alert_history_infra_db_connections_active`

**Рекомендации**:
1. ⚠️ **CRITICAL**: Обновить design.md на использование unified taxonomy:
   ```go
   Name: "alert_history_technical_webhook_requests_total"
   Subsystem: "technical_webhook"  // не "webhook"!
   ```
2. Все метрики должны использовать namespace `alert_history`
3. Интегрировать в существующий `MetricsRegistry` (pkg/metrics/registry.go)
4. Добавить в `TechnicalMetrics` struct, НЕ создавать отдельный registry

---

## Архитектурные замечания

### 🏗️ Hexagonal Architecture Compliance

**Где должны находиться компоненты:**

| Компонент | Правильное место | Указано в tasks.md | Соответствие |
|-----------|------------------|-------------------|--------------|
| Retry Logic | `internal/core/resilience/` | ✅ `internal/core/resilience/retry.go` | ✅ |
| Webhook Parser | `internal/infrastructure/webhook/` | ✅ `internal/infrastructure/webhook/parser.go` | ✅ |
| Webhook Validator | `internal/infrastructure/webhook/` | ✅ `internal/infrastructure/webhook/validator.go` | ✅ |
| Universal Handler | `cmd/server/handlers/` | ❌ `internal/api/handlers/webhook.go` | ⚠️ |
| Async Processor | `internal/core/processing/` | ✅ `internal/core/processing/webhook_processor.go` | ✅ |
| Webhook Metrics | `pkg/metrics/` | ❌ `internal/core/metrics/webhook.go` | 🔴 |

**Проблемы**:
1. ⚠️ TN-042 tasks.md указывает `internal/api/handlers/` но `internal/api/` пустая директория!
   - **Факт**: Handlers находятся в `cmd/server/handlers/`
   - **Fix**: Обновить tasks.md на `cmd/server/handlers/webhook_v2.go`

2. 🔴 TN-045 tasks.md указывает `internal/core/metrics/webhook.go`
   - **Факт**: Metrics находятся в `pkg/metrics/` (shared package)
   - **Факт**: Уже есть `pkg/metrics/technical.go` для technical метрик
   - **Fix**: Обновить tasks.md на `pkg/metrics/technical.go` (расширить существующий файл)

---

## SOLID Principles Compliance

### Single Responsibility Principle: ✅ PASS
- Каждый компонент делает одну вещь:
  - Retry → только retry logic
  - Parser → только parsing
  - Validator → только validation
  - Handler → только HTTP handling + orchestration

### Open/Closed Principle: ✅ PASS
- Extension через interfaces:
  - `WebhookParser` interface → можно добавить `PrometheusParser`, `GenericParser`
  - `WebhookValidator` interface → extensible validation rules
  - `WebhookProcessor` interface → можно заменить на Redis queue, RabbitMQ, etc.

### Liskov Substitution Principle: ✅ PASS
- Разные парсеры (Alertmanager, Generic) могут заменять друг друга через `WebhookParser` interface

### Interface Segregation Principle: ✅ PASS
- Interfaces минимальные и focused:
  - `WebhookParser` - только Parse, Validate, ConvertToDomain
  - `WebhookValidator` - только validation methods
  - `WebhookProcessor` - только job submission и lifecycle

### Dependency Inversion Principle: ✅ PASS
- Зависимость от abstractions:
  - Handler зависит от `WebhookParser` interface, не конкретной реализации
  - Processor зависит от `resilience.RetryPolicy`, не конкретного retry mechanism

---

## DRY (Don't Repeat Yourself) Analysis

### 🔄 Обнаруженное дублирование кода

#### 1. Retry Logic - **HIGH DUPLICATION** 🔴
**Места с retry:**
- `go-app/internal/database/postgres/retry.go` (RetryExecutor + RetryConfig)
- `go-app/internal/infrastructure/llm/client.go` (classifyAlertWithRetry method)
- Возможно: `internal/infrastructure/lock/distributed.go`

**Дублированная логика:**
- Exponential backoff calculation
- Jitter application
- Context cancellation handling
- Retry attempt counting

**Рефакторинг план (TN-040)**:
1. Создать `internal/core/resilience/retry.go` с универсальной реализацией
2. Рефакторить `postgres/retry.go` чтобы использовать `core/resilience`
3. Рефакторить `llm/client.go` чтобы использовать `core/resilience`
4. Удалить дублированный код

#### 2. Webhook Parsing - **MEDIUM DUPLICATION** ⚠️
**Места с parsing:**
- `cmd/server/handlers/webhook.go` (WebhookRequest struct - simplified)
- Будет добавлено: `internal/infrastructure/webhook/parser.go` (AlertmanagerWebhook - full)

**Потенциальное дублирование:**
- Timestamp parsing logic (RFC3339, альтернативные форматы)
- Status parsing ("firing" vs "resolved")
- Fingerprint generation

**Рефакторинг план (TN-041)**:
1. Вынести общую логику в helper functions
2. Переиспользовать в обоих парсерах (simple + full)

#### 3. Validation - **LOW DUPLICATION** ✅
**Места с validation:**
- `cmd/server/handlers/webhook.go` (базовая проверка: alertname != "")
- Будет добавлено: `internal/infrastructure/webhook/validator.go` (полная валидация)

**Рефакторинг план (TN-043)**:
1. Консолидировать всю валидацию в validator
2. Удалить inline validation из handlers

---

## 12-Factor App Compliance

### I. Codebase: ✅ PASS
- Единый Git репозиторий
- Feature branch workflow

### II. Dependencies: ✅ PASS
- `go.mod` декларирует все зависимости
- Reproducible builds

### III. Config: ✅ PASS
- Configuration через `internal/config/config.go`
- Environment variables support
- Вопрос: нужно ли добавить config для async mode (TN-044)?

### IV. Backing services: ✅ PASS
- PostgreSQL, Redis - attached resources
- LLM service - external service with circuit breaker

### V. Build, release, run: ✅ PASS
- Dockerfile для build
- Separate stages: build, release, run

### VI. Processes: ✅ PASS (will improve with TN-044)
- Stateless design (enrichment mode в Redis, не в памяти)
- TN-044 добавит worker pool - нужно убедиться что workers stateless

### VII. Port binding: ✅ PASS
- HTTP server export services через port binding
- Configurable port через config

### VIII. Concurrency: ⚠️ IMPROVE (TN-044)
- Текущий: один процесс обрабатывает все запросы
- TN-044: добавит horizontal scaling через worker pool

### IX. Disposability: ✅ PASS
- Graceful shutdown уже реализован (TN-022)
- TN-044 должен добавить graceful shutdown для workers

### X. Dev/prod parity: ✅ PASS
- Docker Compose для dev
- Одинаковые dependencies

### XI. Logs: ✅ PASS
- Structured logging с slog
- Logs в stdout

### XII. Admin processes: ⚠️ PARTIAL
- Migrations через `cmd/migrate/main.go`
- Нужно добавить: health checks, metrics scraping

---

## Список изменений для документации

### 🔴 CRITICAL - Требует немедленного исправления:

1. **TN-042/design.md**: Заменить Fiber на net/http
   - Файл: `tasks/TN-042/design.md`
   - Изменение: Переписать все примеры кода с `fiber.Ctx` на `http.ResponseWriter, *http.Request`
   - Причина: Проект не использует Fiber

2. **TN-045/design.md**: Обновить naming convention метрик
   - Файл: `tasks/TN-045/design.md`
   - Изменение: Все metrics names должны начинаться с `alert_history_technical_webhook_`
   - Причина: Unified taxonomy из TN-181

3. **TN-042/tasks.md**: Исправить путь к handlers
   - Файл: `tasks/TN-042/tasks.md`
   - Изменение: `internal/api/handlers/webhook.go` → `cmd/server/handlers/webhook_v2.go`
   - Причина: `internal/api/` пустая, handlers в `cmd/server/handlers/`

4. **TN-045/tasks.md**: Исправить путь к metrics
   - Файл: `tasks/TN-045/tasks.md`
   - Изменение: `internal/core/metrics/webhook.go` → `pkg/metrics/technical.go` (extend existing)
   - Причина: Metrics в shared package `pkg/metrics/`

### ⚠️ MEDIUM - Рекомендуется исправить:

5. **TN-041/design.md**: Добавить определение AlertmanagerAlert struct
6. **TN-044/design.md**: Добавить интеграцию с retry module (TN-040)
7. **TN-043/design.md**: Добавить примеры validation rules
8. **TN-040/design.md**: Добавить RetryableErrorChecker interface

### ✅ LOW - Улучшения качества:

9. Все tasks.md: Добавить секцию "Definition of Done" с конкретными критериями
10. Все design.md: Добавить примеры использования и интеграции

---

## Вывод: Phase 1 Complete

### Статус валидации:
- ✅ **2 задачи PASS**: TN-040, TN-043, TN-044
- ⚠️ **1 задача PARTIAL**: TN-041 (отсутствуют детали)
- 🔴 **2 задачи FAIL**: TN-042 (устаревший design), TN-045 (неправильные метрики)

### Критические действия перед началом реализации:
1. ✅ Создать feature branch `feature/TN-040-to-045-webhook-pipeline` - **DONE**
2. ✅ Прочитать всю документацию - **DONE**
3. ✅ Создать матрицу зависимостей - **DONE**
4. ✅ Идентифицировать проблемы в документации - **DONE**
5. 🔄 Обновить проблемные design.md файлы - **NEXT STEP**

### Рекомендация:
**Обновить документацию перед началом Phase 2** (Deep Codebase Analysis), чтобы design соответствовал реальности проекта.

---

**Next Phase**: Phase 2 - Deep Codebase Analysis
