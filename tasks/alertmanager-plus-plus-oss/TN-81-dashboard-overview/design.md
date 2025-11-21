# TN-81: GET /api/dashboard/overview - Design Document

## 1. Архитектурный обзор

### 1.1 Цель дизайна

Спроектировать API endpoint для получения консолидированной обзорной статистики системы с фокусом на:
- **Производительность** - быстрая загрузка для dashboard (< 200ms)
- **Надежность** - graceful degradation при отсутствии компонентов
- **Консолидация** - агрегация данных из множества источников
- **Кэширование** - оптимизация повторных запросов

### 1.2 Архитектурные принципы

1. **Parallel Collection** - параллельный сбор статистики из разных источников
2. **Graceful Degradation** - работа при отсутствии компонентов
3. **Timeout Protection** - защита от медленных компонентов
4. **Response Caching** - кэширование для производительности

### 1.3 Компонентная архитектура

```
┌─────────────────────────────────────────────────────────────┐
│              HTTP Handler Layer                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ DashboardOverviewHandler                              │   │
│  │ - Collect statistics in parallel                     │   │
│  │ - Aggregate data                                     │   │
│  │ - Format response                                    │   │
│  │ - Handle errors gracefully                           │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│            Statistics Collection Layer                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Alert        │  │ Classification│  │ Publishing  │      │
│  │ Statistics   │  │ Statistics   │  │ Statistics   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────┐  ┌──────────────┐                        │
│  │ System       │  │ Cache        │                        │
│  │ Health       │  │ (optional)   │                        │
│  └──────────────┘  └──────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Детальный дизайн компонентов

### 2.1 DashboardOverviewHandler

**Назначение:** HTTP handler для dashboard overview endpoint

**Интерфейс:**
```go
type DashboardOverviewHandler struct {
    historyRepo          core.AlertHistoryRepository
    classificationService services.ClassificationService // optional
    publishingStats      *PublishingStatsProvider       // optional
    cache                cache.Cache                     // optional
    logger               *slog.Logger
}

func (h *DashboardOverviewHandler) GetOverview(w http.ResponseWriter, r *http.Request)
```

**Методы:**
- `GetOverview` - основной handler метод
- `collectAlertStats` - сбор статистики алертов
- `collectClassificationStats` - сбор статистики classification
- `collectPublishingStats` - сбор статистики publishing
- `collectSystemHealth` - сбор системного здоровья
- `aggregateStats` - агрегация всех статистик

### 2.2 Response Format

```go
type DashboardOverviewResponse struct {
    // Alert statistics
    TotalAlerts    int `json:"total_alerts"`
    ActiveAlerts   int `json:"active_alerts"`
    ResolvedAlerts int `json:"resolved_alerts"`
    AlertsLast24h  int `json:"alerts_last_24h"`

    // Classification statistics
    ClassificationEnabled      bool    `json:"classification_enabled"`
    ClassifiedAlerts           int64   `json:"classified_alerts"`
    ClassificationCacheHitRate float64 `json:"classification_cache_hit_rate"`
    LLMServiceAvailable       bool    `json:"llm_service_available"`

    // Publishing statistics
    PublishingTargets  int    `json:"publishing_targets"`
    PublishingMode      string `json:"publishing_mode"`
    SuccessfulPublishes int64  `json:"successful_publishes"`
    FailedPublishes     int64  `json:"failed_publishes"`

    // System health
    SystemHealthy    bool `json:"system_healthy"`
    RedisConnected   bool `json:"redis_connected"`
    LLMServiceAvailable bool `json:"llm_service_available"`

    // Metadata
    LastUpdated string `json:"last_updated"`
}
```

---

## 3. Формат данных

### 3.1 Request

```
GET /api/dashboard/overview
```

**Query Parameters:** None (все данные собираются автоматически)

### 3.2 Response Format

```json
{
  "total_alerts": 1250,
  "active_alerts": 45,
  "resolved_alerts": 1205,
  "alerts_last_24h": 89,
  "classification_enabled": true,
  "classified_alerts": 850,
  "classification_cache_hit_rate": 0.85,
  "llm_service_available": true,
  "publishing_targets": 3,
  "publishing_mode": "intelligent",
  "successful_publishes": 12500,
  "failed_publishes": 25,
  "system_healthy": true,
  "redis_connected": true,
  "last_updated": "2025-11-20T23:50:00Z"
}
```

---

## 4. API контракты

### 4.1 GET /api/dashboard/overview

**Request:**
```
GET /api/dashboard/overview
```

**Response (200 OK):**
```json
{
  "total_alerts": 1250,
  "active_alerts": 45,
  ...
}
```

**Response (500 Internal Server Error):**
```json
{
  "error": "Internal server error",
  "message": "Failed to collect overview statistics"
}
```

---

## 5. Интеграция с существующими компонентами

### 5.1 AlertHistoryRepository Integration

**Использование:**
- `AlertHistoryRepository.GetHistory()` - получение статистики алертов
- Фильтры для подсчета active/resolved/last 24h

**Оптимизация:**
- Использовать существующие методы
- Кэширование результатов

### 5.2 ClassificationService Integration

**Использование:**
- `ClassificationService.GetStats()` - получение статистики classification
- `ClassificationService.Health()` - проверка доступности LLM

**Оптимизация:**
- Graceful degradation при отсутствии service
- Timeout на health check (5 секунд)

### 5.3 Publishing Statistics Integration

**Использование:**
- `TargetDiscoveryManager.ListTargets()` - список targets
- `PublishingStatsHandler` или метрики - статистика publishing

**Оптимизация:**
- Graceful degradation при отсутствии publishing system
- Использование метрик вместо прямых вызовов (если доступно)

### 5.4 System Health Integration

**Использование:**
- `Cache.HealthCheck()` - проверка Redis
- `ClassificationService.Health()` - проверка LLM

**Оптимизация:**
- Параллельные health checks
- Timeout на каждый check (5 секунд)

---

## 6. Сценарии ошибок и Edge Cases

### 6.1 Component Unavailable

**Сценарий:** Classification service недоступен

**Обработка:**
- Возврат defaults: `classification_enabled: false`, `classified_alerts: 0`
- Логирование предупреждения (WARN level)
- Продолжение работы без блокировки

### 6.2 Timeout

**Сценарий:** Компонент не отвечает в течение timeout

**Обработка:**
- Возврат defaults для этого компонента
- Логирование предупреждения (WARN level)
- Продолжение работы с остальными компонентами

### 6.3 Partial Failure

**Сценарий:** Один из компонентов вернул ошибку

**Обработка:**
- Graceful degradation: возврат defaults для этого компонента
- Логирование ошибки (ERROR level)
- Продолжение работы с остальными компонентами

---

## 7. Тестирование стратегия

### 7.1 Unit Tests

**Покрытие:**
- CollectAlertStats (90%+ coverage)
- CollectClassificationStats (90%+ coverage)
- CollectPublishingStats (90%+ coverage)
- CollectSystemHealth (90%+ coverage)
- AggregateStats (90%+ coverage)
- Error handling (100% coverage)

**Тесты:**
- All components available
- Component unavailable (graceful degradation)
- Timeout scenarios
- Partial failures
- Empty results

### 7.2 Integration Tests

**Покрытие:**
- End-to-end flow: request → collection → aggregation → response
- Performance: response time < 200ms
- Error scenarios: component failures

### 7.3 Performance Tests

**Метрики:**
- Response time: < 200ms (p95)
- Throughput: > 50 req/s
- Cache hit rate: > 80%

---

## 8. Производительность и оптимизация

### 8.1 Parallel Collection

**Стратегия:**
- Использовать goroutines для параллельного сбора статистики
- WaitGroup для синхронизации
- Timeout на каждый компонент (5 секунд)

### 8.2 Response Caching

**Стратегия:**
- Cache TTL: 10-30 секунд
- Cache key: `dashboard:overview`
- Invalidation: при изменении данных (опционально)

### 8.3 Timeout Protection

**Стратегия:**
- Context with timeout (10 секунд общий)
- Timeout на каждый компонент (5 секунд)
- Graceful degradation при timeout

---

## 9. Безопасность

### 9.1 Input Validation

**Меры:**
- Нет query параметров (все данные собираются автоматически)
- Валидация response перед отправкой

### 9.2 Rate Limiting

**Меры:**
- Rate limiting через middleware (50 req/min per IP)
- Защита от злоупотребления

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Author:** AI Assistant (Enterprise Architecture Team)
**Status:** ✅ APPROVED FOR IMPLEMENTATION
