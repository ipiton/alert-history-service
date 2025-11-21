# TN-83: GET /api/dashboard/health (basic) - Requirements

## 1. Обоснование задачи

### 1.1 Бизнес-контекст

Dashboard страница (TN-77) и другие UI компоненты требуют API endpoint для проверки здоровья системы. Endpoint должен предоставлять детальную информацию о состоянии всех критических компонентов:
- **Database** (PostgreSQL) - connection pool, latency, availability
- **Redis Cache** - connection status, memory usage, latency
- **LLM Service** - availability, latency, request rate (если включен)
- **Publishing System** - targets health, mode, availability
- **System Metrics** - CPU, memory, request rate, error rate

### 1.2 Пользовательские сценарии

#### US-1: Dashboard User - Мониторинг здоровья системы
**Как** пользователь dashboard
**Я хочу** видеть статус здоровья всех компонентов системы
**Чтобы** быстро определить проблемы и оценить работоспособность

**Критерии приемки:**
- [ ] GET /api/dashboard/health возвращает детальную информацию о здоровье
- [ ] Время ответа < 500ms (p95) для всех проверок
- [ ] Статус каждого компонента четко определен (healthy/degraded/unhealthy/not_configured)
- [ ] Graceful degradation при отсутствии компонентов

#### US-2: Operations Team - Автоматический мониторинг
**Как** операционная команда
**Я хочу** использовать endpoint для автоматического мониторинга
**Чтобы** интегрировать с системами мониторинга (Prometheus, Grafana, Alertmanager)

**Критерии приемки:**
- [ ] HTTP status code отражает общее состояние (200/503)
- [ ] JSON response структурирован и легко парсится
- [ ] Метрики доступны для экспорта в Prometheus

---

## 2. Функциональные требования

### FR-1: Database Health Check
**Приоритет:** HIGH (P0)
**Описание:** Проверка здоровья PostgreSQL database

**Детали:**
- `status` - статус (healthy/unhealthy)
- `latency_ms` - задержка выполнения простого запроса (SELECT 1)
- `connection_pool` - статистика пула соединений (active/total)
- `type` - тип базы данных (postgresql)

**Источники данных:**
- `PostgresPool.Health(ctx)` - проверка здоровья
- `PostgresPool.Stats()` - статистика пула соединений
- Измерение latency через `time.Since()` при выполнении SELECT 1

**Ожидаемое поведение:**
- Если database недоступна → `status: "unhealthy"`, HTTP 503
- Если database доступна → `status: "healthy"`, HTTP 200
- Timeout: 5 секунд на проверку

### FR-2: Redis Cache Health Check
**Приоритет:** HIGH (P0)
**Описание:** Проверка здоровья Redis cache

**Детали:**
- `status` - статус (healthy/unhealthy/not_configured)
- `latency_ms` - задержка выполнения PING
- `memory_usage` - использование памяти (если доступно)

**Источники данных:**
- `Cache.HealthCheck(ctx)` - проверка здоровья
- `Cache.GetStats(ctx)` - статистика (если доступна)
- Измерение latency через `time.Since()` при выполнении PING

**Ожидаемое поведение:**
- Если Redis не настроен → `status: "not_configured"`, не влияет на общий статус
- Если Redis недоступен → `status: "unhealthy"`, HTTP 503 (если критичен)
- Если Redis доступен → `status: "healthy"`, HTTP 200
- Timeout: 2 секунды на проверку

### FR-3: LLM Service Health Check
**Приоритет:** MEDIUM (P1)
**Описание:** Проверка здоровья LLM classification service

**Детали:**
- `status` - статус (available/unavailable/not_configured)
- `latency_ms` - задержка последнего запроса (если доступно)
- `requests_per_minute` - частота запросов (если доступно)

**Источники данных:**
- `ClassificationService.Health(ctx)` - проверка здоровья
- Метрики из Prometheus (если доступны)

**Ожидаемое поведение:**
- Если LLM не настроен → `status: "not_configured"`, не влияет на общий статус
- Если LLM недоступен → `status: "unavailable"`, HTTP 200 (не критично для базовой функциональности)
- Если LLM доступен → `status: "available"`, HTTP 200
- Timeout: 3 секунды на проверку

### FR-4: Publishing System Health Check
**Приоритет:** MEDIUM (P1)
**Описание:** Проверка здоровья publishing system

**Детали:**
- `status` - статус (healthy/degraded/unhealthy/not_configured)
- `targets_count` - количество publishing targets
- `mode` - режим publishing (intelligent/metrics_only)
- `unhealthy_targets` - количество нездоровых targets (если доступно)

**Источники данных:**
- `TargetDiscoveryManager.GetStats()` - статистика discovery
- `HealthMonitor.GetHealth()` - здоровье targets (если доступно)
- `ModeManager.GetCurrentMode()` - текущий режим

**Ожидаемое поведение:**
- Если publishing не настроен → `status: "not_configured"`, не влияет на общий статус
- Если publishing в режиме metrics-only → `status: "degraded"`, HTTP 200
- Если publishing работает → `status: "healthy"`, HTTP 200
- Timeout: 5 секунд на проверку

### FR-5: System Metrics
**Приоритет:** LOW (P2)
**Описание:** Системные метрики (CPU, memory, request rate, error rate)

**Детали:**
- `cpu_usage` - использование CPU (0.0-1.0)
- `memory_usage` - использование памяти (0.0-1.0)
- `request_rate` - частота запросов (req/s)
- `error_rate` - частота ошибок (0.0-1.0)

**Источники данных:**
- Prometheus metrics (если доступны)
- Runtime metrics (runtime.ReadMemStats, если доступно)

**Ожидаемое поведение:**
- Если метрики недоступны → возвращаем null или defaults
- Не влияет на общий статус здоровья

---

## 3. Нефункциональные требования

### NFR-1: Производительность
**Приоритет:** HIGH
**Описание:** Endpoint должен быть быстрым и не блокировать систему

**Требования:**
- Время ответа: < 500ms (p95) для всех проверок
- Параллельное выполнение проверок (goroutines)
- Timeout на каждую проверку (2-5 секунд)
- Поддержка до 100 concurrent requests

**Метрики:**
- Response time: < 500ms (p95), < 1s (p99)
- Throughput: > 100 req/s
- Timeout rate: < 1%

### NFR-2: Надежность
**Приоритет:** HIGH
**Описание:** Graceful degradation при отсутствии компонентов

**Требования:**
- Работает без Redis (возвращает `not_configured`)
- Работает без LLM service (возвращает `not_configured`)
- Работает без Publishing system (возвращает `not_configured`)
- Частичные ошибки не блокируют весь endpoint
- Database недоступность → HTTP 503 (критично)

**Правила определения общего статуса:**
- Если database unhealthy → общий статус `unhealthy`, HTTP 503
- Если Redis unhealthy (и критичен) → общий статус `degraded`, HTTP 200
- Если LLM unavailable → общий статус `healthy` (не критично)
- Если Publishing degraded → общий статус `degraded`, HTTP 200
- Иначе → общий статус `healthy`, HTTP 200

### NFR-3: Observability
**Приоритет:** MEDIUM
**Описание:** Логирование и метрики

**Требования:**
- Structured logging (slog) для всех проверок
- Prometheus metrics для health checks:
  - `dashboard_health_checks_total` (Counter, by component, status)
  - `dashboard_health_check_duration_seconds` (Histogram, by component)
  - `dashboard_health_status` (Gauge, by component)
- Логирование ошибок с контекстом

### NFR-4: Безопасность
**Приоритет:** MEDIUM
**Описание:** Безопасность endpoint

**Требования:**
- Не раскрывать sensitive информацию (пароли, токены)
- Rate limiting (опционально, через middleware)
- CORS support (если требуется)

---

## 4. Зависимости

### Upstream (Все завершены ✅)
- ✅ **TN-12**: Postgres Pool (150%+, Grade A+)
  - `PostgresPool.Health(ctx)` - проверка здоровья
  - `PostgresPool.Stats()` - статистика пула
- ✅ **TN-16**: Redis Cache Wrapper (150%+, Grade A+)
  - `Cache.HealthCheck(ctx)` - проверка здоровья
  - `Cache.GetStats(ctx)` - статистика (опционально)
- ✅ **TN-33**: Classification Service (150%, Grade A+)
  - `ClassificationService.Health(ctx)` - проверка здоровья
- ✅ **TN-47**: Target Discovery Manager (147%, Grade A+)
  - `TargetDiscoveryManager.GetStats()` - статистика
- ✅ **TN-49**: Target Health Monitoring (140%, Grade A)
  - `HealthMonitor.GetHealth()` - здоровье targets (опционально)
- ✅ **TN-60**: Metrics-Only Mode Fallback (150%+, Grade A+)
  - `ModeManager.GetCurrentMode()` - текущий режим
- ✅ **TN-21**: Prometheus Metrics (100%, Grade A)
  - Метрики для observability

### Downstream (Unblocked)
- 🎯 **TN-77**: Modern Dashboard Page (может использовать этот endpoint)
- 🎯 **TN-81**: GET /api/dashboard/overview (может использовать этот endpoint)
- 🎯 **Future**: Monitoring integrations (Prometheus, Grafana)

---

## 5. Риски и митигация

### Risk 1: Производительность деградация при множественных проверках
**Probability:** MEDIUM
**Impact:** MEDIUM
**Mitigation:**
- Параллельное выполнение проверок (goroutines с WaitGroup)
- Timeout на каждую проверку (2-5 секунд)
- Кэширование результатов (опционально, 10-30 секунд)
- Приоритизация критичных проверок (database → redis → остальные)

### Risk 2: Частичные ошибки блокируют endpoint
**Probability:** LOW
**Impact:** HIGH
**Mitigation:**
- Graceful degradation для каждого компонента
- Частичные ошибки логируются, но не блокируют
- Возврат defaults при ошибках (not_configured/unavailable)
- Database недоступность → HTTP 503, остальные → HTTP 200 с degraded

### Risk 3: Timeout на проверках вызывает медленный ответ
**Probability:** MEDIUM
**Impact:** MEDIUM
**Mitigation:**
- Короткие timeout (2-5 секунд)
- Параллельное выполнение проверок
- Fail-fast для критичных компонентов (database)
- Логирование timeout для диагностики

### Risk 4: Недостаточная информация для диагностики
**Probability:** LOW
**Impact:** LOW
**Mitigation:**
- Детальная информация в response (latency, connection pool, etc.)
- Structured logging с контекстом
- Prometheus metrics для исторического анализа

---

## 6. Критерии приемки

### Must Have (P0)
- [ ] GET /api/dashboard/health возвращает JSON с информацией о здоровье
- [ ] Database health check работает (status, latency_ms, connection_pool)
- [ ] Redis health check работает (status, latency_ms, memory_usage если доступно)
- [ ] HTTP status code отражает общее состояние (200 для healthy/degraded, 503 для unhealthy)
- [ ] Время ответа < 500ms (p95)
- [ ] Graceful degradation при отсутствии компонентов
- [ ] Structured logging для всех проверок

### Should Have (P1)
- [ ] LLM service health check работает (status, latency_ms если доступно)
- [ ] Publishing system health check работает (status, targets_count, mode)
- [ ] Параллельное выполнение проверок (goroutines)
- [ ] Prometheus metrics для health checks

### Nice to Have (P2)
- [ ] System metrics (CPU, memory, request rate, error rate)
- [ ] Кэширование результатов (10-30 секунд)
- [ ] OpenAPI 3.0 specification

---

## 7. Формат ответа

### Успешный ответ (200 OK - Healthy)
```json
{
  "status": "healthy",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 15,
      "connection_pool": "8/20",
      "type": "postgresql"
    },
    "redis": {
      "status": "healthy",
      "latency_ms": 2,
      "memory_usage": "45MB"
    },
    "llm_service": {
      "status": "available",
      "latency_ms": 850,
      "requests_per_minute": 5.2
    },
    "publishing": {
      "status": "healthy",
      "targets_count": 5,
      "mode": "intelligent",
      "unhealthy_targets": 0
    }
  },
  "metrics": {
    "cpu_usage": 0.25,
    "memory_usage": 0.40,
    "request_rate": 12.5,
    "error_rate": 0.02
  }
}
```

### Успешный ответ (200 OK - Degraded)
```json
{
  "status": "degraded",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "healthy",
      "latency_ms": 15,
      "connection_pool": "8/20",
      "type": "postgresql"
    },
    "redis": {
      "status": "unhealthy",
      "latency_ms": null,
      "error": "connection timeout"
    },
    "llm_service": {
      "status": "not_configured"
    },
    "publishing": {
      "status": "degraded",
      "targets_count": 5,
      "mode": "intelligent",
      "unhealthy_targets": 2
    }
  },
  "metrics": {
    "cpu_usage": 0.25,
    "memory_usage": 0.40,
    "request_rate": 12.5,
    "error_rate": 0.02
  }
}
```

### Ошибка (503 Service Unavailable - Unhealthy)
```json
{
  "status": "unhealthy",
  "timestamp": "2025-11-21T10:30:45Z",
  "services": {
    "database": {
      "status": "unhealthy",
      "latency_ms": null,
      "error": "connection refused"
    },
    "redis": {
      "status": "not_configured"
    },
    "llm_service": {
      "status": "not_configured"
    },
    "publishing": {
      "status": "not_configured"
    }
  },
  "metrics": null
}
```

---

## 8. Технические детали

### 8.1 Структура данных

```go
type DashboardHealthResponse struct {
    Status    string                 `json:"status"`    // healthy/degraded/unhealthy
    Timestamp time.Time              `json:"timestamp"`
    Services  map[string]ServiceHealth `json:"services"`
    Metrics   *SystemMetrics         `json:"metrics,omitempty"`
}

type ServiceHealth struct {
    Status    string                 `json:"status"`
    LatencyMS *int64                 `json:"latency_ms,omitempty"`
    Details   map[string]interface{} `json:"details,omitempty"`
    Error     string                 `json:"error,omitempty"`
}

type SystemMetrics struct {
    CPUUsage    float64 `json:"cpu_usage,omitempty"`
    MemoryUsage float64 `json:"memory_usage,omitempty"`
    RequestRate float64 `json:"request_rate,omitempty"`
    ErrorRate   float64 `json:"error_rate,omitempty"`
}
```

### 8.2 HTTP Status Codes
- `200 OK` - Система healthy или degraded (работает, но есть проблемы)
- `503 Service Unavailable` - Система unhealthy (database недоступна)

### 8.3 Timeout Configuration
- Database check: 5 секунд
- Redis check: 2 секунды
- LLM check: 3 секунды
- Publishing check: 5 секунд
- Общий timeout: 10 секунд (max)

---

## 9. Метрики успешности

### Performance Metrics
- Response time: < 500ms (p95), < 1s (p99) ✅
- Throughput: > 100 req/s ✅
- Timeout rate: < 1% ✅

### Quality Metrics
- Test coverage: > 85% ✅
- Zero race conditions ✅
- Zero linter warnings ✅
- 100% backward compatibility ✅

### Production Readiness
- Comprehensive error handling ✅
- Structured logging ✅
- Prometheus metrics ✅
- Documentation complete ✅

---

## 10. Принятые решения

### Decision 1: Параллельное выполнение проверок
**Решение:** Использовать goroutines с sync.WaitGroup для параллельного выполнения всех проверок
**Обоснование:** Уменьшает общее время ответа с ~15s (последовательно) до ~5s (параллельно)

### Decision 2: Graceful degradation
**Решение:** Возвращать `not_configured` для опциональных компонентов вместо ошибки
**Обоснование:** Позволяет endpoint работать даже если некоторые компоненты не настроены

### Decision 3: HTTP Status Code Logic
**Решение:** 200 для healthy/degraded, 503 только для unhealthy (database недоступна)
**Обоснование:** Degraded система все еще работает, только unhealthy требует внимания

### Decision 4: Timeout Configuration
**Решение:** Разные timeout для разных компонентов (2-5 секунд)
**Обоснование:** Критичные компоненты (database) требуют больше времени, опциональные (LLM) - меньше

---

*Requirements Document Version: 1.0*
*Last Updated: 2025-11-21*
*Author: AI Assistant*
*Status: DRAFT → READY FOR DESIGN*
