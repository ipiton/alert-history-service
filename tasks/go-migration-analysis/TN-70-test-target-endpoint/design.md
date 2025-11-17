# TN-70: POST /publishing/test/{target} - Design Document

## 1. Архитектурное решение

### 1.1 Обзор

Endpoint `POST /api/v2/publishing/targets/{target}/test` реализует функциональность тестирования publishing targets через отправку test alert и возврат детального результата.

**Архитектурный паттерн**: Request-Response Handler с синхронной публикацией

### 1.2 Компоненты системы

```
┌─────────────────────────────────────────────────────────────┐
│                    HTTP Request Handler                      │
│  POST /api/v2/publishing/targets/{target}/test             │
│  - Request validation                                       │
│  - Target lookup                                            │
│  - Test alert creation                                      │
│  - Response formatting                                      │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Target Discovery Manager                        │
│  - GetTarget(name) → PublishingTarget                       │
│  - Validation: existence, enabled status                    │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Test Alert Builder                              │
│  - Create EnrichedAlert with test labels                     │
│  - Support custom alert payload                             │
│  - Generate unique fingerprint                               │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│         Publishing Coordinator (Synchronous)                │
│  - PublishToTargets(ctx, alert, [target])                   │
│  - Context timeout support                                  │
│  - Error handling                                           │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│              Publisher (Rootly/PagerDuty/etc)               │
│  - Format alert                                             │
│  - Send HTTP request                                        │
│  - Return status code                                       │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Поток данных

1. **Request Reception**: HTTP handler получает POST request
2. **Validation**: Валидация path parameter `target` и опционального body
3. **Target Lookup**: Получение target через TargetDiscoveryManager
4. **Test Alert Creation**: Создание EnrichedAlert с test метками
5. **Publishing**: Синхронная публикация через PublishingCoordinator
6. **Response Formatting**: Форматирование результата с метриками
7. **Response**: Возврат HTTP 200 с детальным response

## 2. Формат данных

### 2.1 Request

**Path**: `/api/v2/publishing/targets/{target}/test`
**Method**: POST
**Content-Type**: `application/json`

**Path Parameters**:
- `target` (string, required): Name of the target to test

**Request Body** (optional):
```json
{
  "alert_name": "string (optional, default: 'TestAlert')",
  "test_alert": {
    "fingerprint": "string (optional)",
    "labels": {
      "key": "value"
    },
    "annotations": {
      "key": "value"
    },
    "status": "firing|resolved (optional, default: 'firing')"
  },
  "timeout_seconds": "integer (optional, default: 30, min: 1, max: 300)"
}
```

**Go Struct**:
```go
type TestTargetRequest struct {
    AlertName      string                 `json:"alert_name"`
    TestAlert      *CustomTestAlert       `json:"test_alert,omitempty"`
    TimeoutSeconds int                    `json:"timeout_seconds"`
}

type CustomTestAlert struct {
    Fingerprint string            `json:"fingerprint,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
    Annotations map[string]string `json:"annotations,omitempty"`
    Status      string            `json:"status,omitempty"` // "firing" | "resolved"
}
```

### 2.2 Response

**Status Code**: 200 OK (always, даже при failure)

**Response Body**:
```json
{
  "success": "boolean",
  "message": "string",
  "target_name": "string",
  "status_code": "integer (optional, HTTP status от target API)",
  "response_time_ms": "integer",
  "error": "string (optional, только при success: false)",
  "test_timestamp": "string (ISO 8601)"
}
```

**Go Struct**:
```go
type TestTargetResponse struct {
    Success         bool      `json:"success"`
    Message         string    `json:"message"`
    TargetName      string    `json:"target_name"`
    StatusCode      *int      `json:"status_code,omitempty"`
    ResponseTimeMs  int       `json:"response_time_ms"`
    Error           string    `json:"error,omitempty"`
    TestTimestamp   time.Time `json:"test_timestamp"`
}
```

### 2.3 Error Responses

**404 Not Found** (target не найден):
```json
{
  "error": "Target not found",
  "message": "Target 'invalid-target' does not exist",
  "request_id": "uuid"
}
```

**400 Bad Request** (invalid request body):
```json
{
  "error": "Invalid request body",
  "message": "JSON decode error: ...",
  "request_id": "uuid"
}
```

**429 Too Many Requests** (rate limit exceeded):
```json
{
  "error": "Rate limit exceeded",
  "message": "Maximum 10 requests per minute",
  "request_id": "uuid"
}
```

## 3. API контракты

### 3.1 Handler Interface

```go
// TestTargetHandler handles POST /api/v2/publishing/targets/{target}/test
type TestTargetHandler interface {
    TestTarget(w http.ResponseWriter, r *http.Request)
}
```

### 3.2 Dependencies

```go
type TestTargetHandlerDeps struct {
    DiscoveryManager  TargetDiscoveryManager  // TN-47
    Coordinator       PublishingCoordinator   // TN-56
    Logger           *slog.Logger
    Metrics          *TestTargetMetrics      // New
}
```

### 3.3 Test Alert Builder Interface

```go
// TestAlertBuilder creates test alerts for target testing
type TestAlertBuilder interface {
    BuildTestAlert(req *TestTargetRequest) (*core.EnrichedAlert, error)
}
```

## 4. Сценарии ошибок и edge cases

### 4.1 Target не найден

**Сценарий**: Target с указанным именем не существует
**Обработка**: HTTP 404 Not Found
**Response**:
```json
{
  "error": "Target not found",
  "message": "Target 'invalid-target' does not exist"
}
```

### 4.2 Target disabled

**Сценарий**: Target существует, но disabled
**Обработка**: HTTP 200 OK с `success: false`
**Response**:
```json
{
  "success": false,
  "message": "Target is disabled",
  "target_name": "disabled-target",
  "response_time_ms": 5,
  "test_timestamp": "2025-01-17T10:00:00Z"
}
```

### 4.3 Invalid request body

**Сценарий**: Request body содержит invalid JSON
**Обработка**: HTTP 400 Bad Request
**Response**:
```json
{
  "error": "Invalid request body",
  "message": "JSON decode error: invalid character 'x'"
}
```

### 4.4 Timeout

**Сценарий**: Publishing превышает timeout
**Обработка**: HTTP 200 OK с `success: false`, `error: "timeout"`
**Response**:
```json
{
  "success": false,
  "message": "Test timeout",
  "target_name": "slow-target",
  "response_time_ms": 30000,
  "error": "Test timeout after 30 seconds",
  "test_timestamp": "2025-01-17T10:00:00Z"
}
```

### 4.5 Publishing failure

**Сценарий**: Publishing завершился с ошибкой
**Обработка**: HTTP 200 OK с `success: false`, детальный error
**Response**:
```json
{
  "success": false,
  "message": "Test alert sent",
  "target_name": "rootly-prod",
  "status_code": 401,
  "response_time_ms": 150,
  "error": "authentication failed: invalid API key",
  "test_timestamp": "2025-01-17T10:00:00Z"
}
```

### 4.6 Empty request body

**Сценарий**: Request body пустой или отсутствует
**Обработка**: Используются default значения (alert_name: "TestAlert", timeout: 30s)

### 4.7 Custom alert payload

**Сценарий**: Предоставлен custom test_alert
**Обработка**: Используется custom alert, но добавляются test метки

### 4.8 Context cancellation

**Сценарий**: Request context отменен (client disconnect)
**Обработка**: HTTP 200 OK с `success: false`, `error: "context cancelled"`

## 5. Производительность

### 5.1 Оптимизации

1. **Синхронная публикация**: Используется синхронная публикация для immediate feedback (не через queue)
2. **Context timeout**: Настраиваемый timeout для предотвращения hanging requests
3. **Early validation**: Валидация target existence перед созданием alert
4. **Minimal allocations**: Переиспользование структур где возможно

### 5.2 Целевые метрики

- **P50 latency**: < 100ms
- **P95 latency**: < 250ms (2x лучше target 500ms)
- **P99 latency**: < 500ms
- **Throughput**: > 100 req/s

### 5.3 Bottlenecks

1. **Target API latency**: Зависит от внешнего API (Rootly, PagerDuty, etc.)
2. **Network latency**: Зависит от network conditions
3. **Target discovery**: K8s API lookup (может быть медленным)

## 6. Безопасность

### 6.1 Authentication & Authorization

- **Required**: Operator+ role (через middleware.AuthMiddleware + OperatorMiddleware)
- **RBAC**: Проверка permissions через middleware

### 6.2 Rate Limiting

- **Limit**: 10 requests per minute per IP
- **Implementation**: Token bucket через middleware.RateLimitMiddleware
- **Response**: HTTP 429 Too Many Requests при превышении

### 6.3 Input Validation

- **Path parameter**: Валидация target name (alphanumeric, dashes, underscores)
- **Request body**: Валидация JSON structure, timeout range (1-300s)
- **Custom alert**: Валидация структуры test_alert

### 6.4 Error Sanitization

- **Sensitive data**: Не раскрывать API keys, tokens в error messages
- **Stack traces**: Не возвращать stack traces в production
- **Internal errors**: Общие сообщения для internal errors

## 7. Observability

### 7.1 Prometheus Metrics

```go
// publishing_test_requests_total - Total number of test requests
publishing_test_requests_total{target, status} counter

// publishing_test_duration_seconds - Test duration histogram
publishing_test_duration_seconds{target, status} histogram

// publishing_test_errors_total - Test errors by type
publishing_test_errors_total{target, error_type} counter

// publishing_test_timeouts_total - Test timeouts
publishing_test_timeouts_total{target} counter

// publishing_test_success_rate - Success rate gauge
publishing_test_success_rate{target} gauge
```

### 7.2 Structured Logging

```go
logger.Info("Test target started",
    "request_id", requestID,
    "target", targetName,
    "alert_name", alertName,
    "timeout_seconds", timeoutSeconds,
)

logger.Info("Test target completed",
    "request_id", requestID,
    "target", targetName,
    "success", success,
    "response_time_ms", responseTimeMs,
    "status_code", statusCode,
)
```

### 7.3 Distributed Tracing

- **Span name**: `publishing.test_target`
- **Tags**: `target.name`, `alert.name`, `success`, `duration_ms`
- **Error tags**: При failure добавляются error tags

## 8. Тестирование

### 8.1 Unit Tests

- **Handler tests**: Mock dependencies, test все error paths
- **Test alert builder tests**: Test создание test alerts
- **Validation tests**: Test валидацию входных данных

### 8.2 Integration Tests

- **E2E tests**: Полный flow с mock publishers
- **Target discovery tests**: Test с реальным TargetDiscoveryManager
- **Publishing tests**: Test с реальным PublishingCoordinator

### 8.3 Benchmarks

- **Handler benchmark**: Test latency handler
- **Test alert creation**: Benchmark создания test alerts
- **End-to-end**: Benchmark полного flow

## 9. Интеграция

### 9.1 Router Integration

```go
// В router.go
targetsOperator.HandleFunc("/{name}/test",
    handlers.HandleTestTarget(config.TestTargetHandler)).Methods("POST")
```

### 9.2 Middleware Stack

1. **Recovery**: Panic recovery
2. **RequestID**: Request ID generation
3. **Logging**: Structured logging
4. **Metrics**: Prometheus metrics
5. **Auth**: Authentication (Operator+)
6. **RateLimit**: Rate limiting (10 req/min)
7. **Timeout**: Request timeout (35s, больше чем publishing timeout)

### 9.3 Dependencies Injection

```go
type RouterConfig struct {
    // ... existing fields
    TestTargetHandler *handlers.TestTargetHandler
}
```

## 10. Миграция и совместимость

### 10.1 Backward Compatibility

- **API version**: `/api/v2/` (новый endpoint)
- **Old endpoint**: `/api/v1/publishing/targets/{name}/test` (deprecated, но поддерживается)

### 10.2 Migration Path

1. **Phase 1**: Реализовать новый endpoint в `/api/v2/`
2. **Phase 2**: Добавить deprecation notice для старого endpoint
3. **Phase 3**: Удалить старый endpoint после migration period (30 дней)

## 11. Документация

### 11.1 OpenAPI 3.0 Spec

- Полная спецификация endpoint
- Request/Response schemas
- Error responses
- Examples

### 11.2 API Guide

- Quick start guide
- Примеры использования (curl, Go, Python)
- Troubleshooting guide
- Best practices

### 11.3 Code Documentation

- Godoc comments для всех exported types
- Inline comments для complex logic
- Examples в документации

---

**Дата создания**: 2025-01-17
**Автор**: AI Assistant
**Статус**: Draft → Ready for Implementation
