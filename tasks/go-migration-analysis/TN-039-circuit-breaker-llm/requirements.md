# TN-039: Circuit Breaker для LLM Calls - Requirements

**Дата создания**: 2025-10-09
**Статус**: 📋 TODO - Не начата
**Приоритет**: HIGH
**Зависимости**: TN-33 (Alert Classification Service ✅), TN-29 (LLM Client ✅)
**Блокирует**: -
**Связанные задачи**: TN-40 (Retry Logic), TN-34 (Enrichment Mode System ✅)

---

## 1. Обоснование (Зачем?)

### Проблема

Текущая реализация LLM client (`go-app/internal/infrastructure/llm/client.go`) имеет критические проблемы устойчивости:

1. **Отсутствие Circuit Breaker Pattern**
   - При недоступности LLM сервиса каждый входящий alert вызывает 3+ retry попыток
   - Если LLM отвечает медленно или падает, система продолжает делать попытки на КАЖДЫЙ alert
   - Cascade failure: падение LLM → перегрузка → задержки в обработке → накопление alerts

2. **Проблемы Production**
   - Если LLM proxy недоступен 5 минут, при 100 alerts/sec = 90,000 ненужных HTTP calls
   - Timeout 30 секунд × 3 retries = 90 секунд блокировки на один alert
   - Alert processing time: норма ~100ms → при LLM down ~90 секунд
   - Memory leaks из-за накопления goroutines ожидающих timeout

3. **Отсутствие Graceful Degradation**
   - Система не может быстро переключиться на fallback режим (transparent mode)
   - Нет механизма автоматического восстановления когда LLM вернется онлайн
   - Нет метрик для мониторинга состояния circuit breaker

### Бизнес-ценность

- **Availability**: Система остается доступной даже при падении LLM
- **Performance**: Reduced latency при проблемах с LLM (90s → 100ms при fallback)
- **Cost Optimization**: Экономия на ненужных LLM calls (~90% при downtimes)
- **SLA Compliance**: Соблюдение SLA для alert processing (<200ms p95)
- **Observability**: Мониторинг health состояния интеграций

### Пользовательские сценарии

**Сценарий 1: LLM Service Downtime**
```
GIVEN: LLM proxy полностью недоступен (network issue)
WHEN: Поступает 100 alerts за минуту
THEN:
  - Первые 3-5 alerts открывают circuit breaker (3-5 failures)
  - Circuit breaker переходит в OPEN state
  - Следующие 95 alerts НЕ вызывают LLM (fail-fast)
  - Система переключается на transparent mode (fallback)
  - Alert processing time остается <200ms
  - Каждые 30 секунд circuit breaker переходит в HALF_OPEN для проверки
```

**Сценарий 2: LLM Service Slow Response**
```
GIVEN: LLM proxy отвечает медленно (5+ seconds)
WHEN: Поступает поток alerts
THEN:
  - Circuit breaker отслеживает slow responses как failures
  - После threshold (например, 5 slow responses) → OPEN state
  - Система fallback на transparent mode
  - Alert queue не накапливается
```

**Сценарий 3: LLM Service Recovery**
```
GIVEN: Circuit breaker в OPEN state (LLM был down)
WHEN: Прошло resetTimeout (30 секунд)
THEN:
  - Circuit breaker переходит в HALF_OPEN state
  - Разрешается ОДНА test request к LLM
  - Если успех → переход в CLOSED state, восстановление enriched mode
  - Если failure → обратно в OPEN state на следующий период
```

**Сценарий 4: Monitoring and Alerting**
```
GIVEN: Circuit breaker работает
WHEN: Происходят изменения состояния
THEN:
  - Prometheus метрики обновляются (llm_circuit_breaker_state)
  - Логи содержат structured events с контекстом
  - Grafana dashboard показывает текущее состояние
  - Alert manager может уведомить ops team о проблемах
```

---

## 2. Функциональные требования

### FR-1: Circuit Breaker Implementation

**FR-1.1**: Реализовать circuit breaker pattern с тремя состояниями:
- `CLOSED` - нормальная работа, все requests проходят
- `OPEN` - circuit открыт, requests fail-fast без вызова LLM
- `HALF_OPEN` - проверочное состояние, разрешена одна test request

**FR-1.2**: Конфигурируемые параметры:
```go
type CircuitBreakerConfig struct {
    MaxFailures      int           // Threshold для открытия (default: 5)
    ResetTimeout     time.Duration // Время до HALF_OPEN (default: 30s)
    HalfOpenRequests int           // Сколько test requests в HALF_OPEN (default: 1)
    FailureThreshold float64       // % failures для открытия (default: 0.5 = 50%)
    TimeWindow       time.Duration // Окно для подсчета failures (default: 60s)
}
```

**FR-1.3**: Определение failure:
- HTTP status code >= 500
- Network errors (connection refused, timeout, DNS)
- Context timeout/cancellation
- Response time > SlowCallThreshold (например, 3 секунды)

**FR-1.4**: Определение success:
- HTTP status code 2xx
- Valid JSON response
- Response time < SlowCallThreshold

### FR-2: Integration with LLM Client

**FR-2.1**: Circuit breaker должен оборачивать `classifyAlertOnce()` метод

**FR-2.2**: При OPEN state:
- Возвращать специальную ошибку `ErrCircuitBreakerOpen`
- НЕ выполнять retry logic
- Логировать на DEBUG level (не засорять логи)

**FR-2.3**: При HALF_OPEN state:
- Разрешать ограниченное количество test requests
- Первый успех → переход в CLOSED
- Первый failure → переход в OPEN

**FR-2.4**: Сохранение существующего retry logic:
- Retry logic работает для transient errors (429, network glitches)
- Circuit breaker оборачивает retry logic, не заменяет его

### FR-3: Fallback Strategy

**FR-3.1**: При `ErrCircuitBreakerOpen`:
- AlertProcessor должен автоматически fallback на transparent mode
- Логировать warning один раз при переходе
- Не блокировать alert processing

**FR-3.2**: При восстановлении (CLOSED state):
- Автоматически вернуться к enriched mode
- Логировать info event о восстановлении

### FR-4: Metrics and Observability

**FR-4.1**: Prometheus метрики:
```
llm_circuit_breaker_state{state="closed|open|half_open"} gauge
llm_circuit_breaker_failures_total counter
llm_circuit_breaker_successes_total counter
llm_circuit_breaker_state_changes_total{from="X",to="Y"} counter
llm_circuit_breaker_requests_blocked_total counter
llm_circuit_breaker_half_open_requests_total counter
```

**FR-4.2**: Structured Logging:
- State transitions: INFO level
- Failures: WARN level (с deduplication)
- Блокированные requests: DEBUG level
- Recovery: INFO level

**FR-4.3**: Health Check:
- `/health` endpoint должен включать circuit breaker state
- Status code 200 even при OPEN state (это нормальное поведение)
- Включить в response body:
  ```json
  {
    "llm": {
      "circuit_breaker_state": "open",
      "failure_count": 12,
      "last_failure": "2025-10-09T10:30:00Z",
      "next_retry_at": "2025-10-09T10:30:30Z"
    }
  }
  ```

---

## 3. Нефункциональные требования

### NFR-1: Performance

- **NFR-1.1**: Overhead circuit breaker < 1ms per request
- **NFR-1.2**: Fail-fast при OPEN state < 10ms (no network calls)
- **NFR-1.3**: Thread-safe для concurrent requests (use sync.RWMutex)

### NFR-2: Reliability

- **NFR-2.1**: Circuit breaker должен быть stateless (in-memory state OK)
- **NFR-2.2**: State не теряется при restart (acceptable - это feature flag)
- **NFR-2.3**: No goroutine leaks, no memory leaks

### NFR-3: Testability

- **NFR-3.1**: Unit tests с coverage >90%
- **NFR-3.2**: Integration tests для всех state transitions
- **NFR-3.3**: Table-driven tests для failure scenarios
- **NFR-3.4**: Mocks для LLM client и time (для тестирования timeouts)

### NFR-4: Maintainability

- **NFR-4.1**: Код должен следовать паттернам существующих circuit breakers в проекте
- **NFR-4.2**: Переиспользовать код из `go-app/internal/database/postgres/retry.go`
- **NFR-4.3**: Документация в GoDoc для всех exported types
- **NFR-4.4**: Примеры использования в tests

### NFR-5: Configuration

- **NFR-5.1**: Конфигурация через environment variables
- **NFR-5.2**: Reasonable defaults (работает out-of-the-box)
- **NFR-5.3**: Runtime reconfiguration через API (optional, nice-to-have)

### NFR-6: Backward Compatibility

- **NFR-6.1**: Существующий LLMClient interface НЕ меняется
- **NFR-6.2**: Zero breaking changes для consumers (AlertProcessor)
- **NFR-6.3**: Feature flag для включения/выключения circuit breaker

---

## 4. Ограничения и constraints

### Технические ограничения

1. **Существующая архитектура**
   - LLM client уже реализован в `internal/infrastructure/llm/client.go`
   - AlertProcessor использует LLMClient interface
   - Нельзя менять core interfaces

2. **Dependency на TN-33, TN-34**
   - Alert Classification Service (TN-33) уже использует LLM client
   - Enrichment Mode System (TN-34) управляет режимами
   - Нужна интеграция с enrichment manager

3. **Связь с TN-40**
   - TN-40 (Retry Logic с exponential backoff) - уже реализован частично
   - Circuit breaker должен работать ВМЕСТЕ с retry logic, не заменять его
   - Retry для transient errors, circuit breaker для prolonged failures

### Внешние зависимости

1. **LLM Proxy Service**
   - `https://llm-proxy.b2broker.tech`
   - Может быть недоступен (network, maintenance, overload)
   - SLA неизвестен - поэтому нужен circuit breaker

2. **Prometheus/Grafana**
   - Метрики уже интегрированы в проект
   - Используется `pkg/metrics/metrics.go`

3. **Redis (опционально)**
   - Если потребуется distributed circuit breaker (multi-instance)
   - Сейчас можно обойтись in-memory state (single instance OK)

### Временные constraints

- Задача должна быть завершена ДО начала TN-40 (Retry Logic улучшения)
- Не блокирует Alertmanager++ roadmap (Phase A)
- Приоритет: HIGH но не критический блокер

---

## 5. Критерии приемки

### Definition of Done

- [ ] Circuit breaker реализован с тремя состояниями (CLOSED, OPEN, HALF_OPEN)
- [ ] Интегрирован в HTTPLLMClient без breaking changes
- [ ] Конфигурация через environment variables работает
- [ ] Fallback на transparent mode при OPEN state
- [ ] Prometheus метрики экспортируются (6+ метрик)
- [ ] Structured logging для всех state transitions
- [ ] Health check включает circuit breaker state
- [ ] Unit tests с coverage >90%
- [ ] Integration tests для всех сценариев
- [ ] Документация обновлена (GoDoc + README)
- [ ] CI зеленый (linter, tests, coverage)
- [ ] Reviewed и merged в main

### Success Metrics

**Производительность:**
- Alert processing latency при LLM down: <200ms (было ~90s)
- Fail-fast time: <10ms (вместо 30s timeout)
- Circuit breaker overhead: <1ms

**Reliability:**
- Zero goroutine leaks (проверено leak detector)
- Zero memory leaks (проверено pprof)
- Корректная работа при concurrent load (load test 1000 req/s)

**Observability:**
- Все state transitions видны в логах
- Prometheus метрики обновляются real-time
- Grafana dashboard показывает состояние

---

## 6. Out of Scope (Что НЕ включено)

1. **Distributed Circuit Breaker** - пока только single-instance (Redis для state sync - future)
2. **Adaptive Thresholds** - пока статическая конфигурация (ML для dynamic thresholds - future)
3. **Circuit Breaker для других сервисов** - только для LLM (database уже есть)
4. **Rate Limiting** - это отдельная задача (TN-40 может включать)
5. **Custom Fallback Strategies** - пока только transparent mode (future: cache, pre-trained model)

---

## 7. Связанные задачи и зависимости

### Завершенные (Зависимости)
- ✅ **TN-29**: POC LLM Proxy Client - `internal/infrastructure/llm/client.go`
- ✅ **TN-33**: Alert Classification Service - `internal/core/services/`
- ✅ **TN-34**: Enrichment Mode System - fallback механизм

### Связанные (Coordination)
- 📋 **TN-40**: Retry Logic с exponential backoff
  - Circuit breaker оборачивает retry logic
  - Нужна координация: retry для transient, CB для prolonged
- 📋 **TN-45**: Webhook Metrics and Monitoring
  - Общие метрики и dashboard

### Будущие (Могут использовать)
- **TN-122+**: Alertmanager++ components могут использовать pattern
- **Python Sunset**: При миграции оставшихся Python services

---

## 8. Риски и митigation

| Риск | Вероятность | Влияние | Mitigation |
|------|-------------|---------|------------|
| Circuit breaker слишком агрессивный (false positives) | Medium | High | Настроить higher threshold (10 failures vs 5), увеличить time window |
| Не успеваем fallback на transparent mode | Low | High | Integration tests, stress tests |
| Конфликт с TN-40 retry logic | Medium | Medium | Review existing retry code, документировать interaction |
| Memory leaks в долгоживущих goroutines | Low | High | Thorough review, leak detector в CI |
| Метрики не обновляются корректно | Low | Medium | Unit tests для metrics, verify в Grafana |

---

## References

1. **Circuit Breaker Pattern**: Martin Fowler - https://martinfowler.com/bliki/CircuitBreaker.html
2. **Existing Implementation**: `go-app/internal/database/postgres/retry.go`
3. **Go Libraries**:
   - `github.com/sony/gobreaker` (reference, мы пишем свой)
   - `github.com/afex/hystrix-go` (reference)

---

**Автор**: AI Agent (Cursor)
**Дата последнего обновления**: 2025-10-09
**Версия**: 1.0
