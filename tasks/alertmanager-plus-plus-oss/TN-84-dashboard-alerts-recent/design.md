# TN-84: GET /api/dashboard/alerts/recent - Design Document

## 1. Архитектурный обзор

### 1.1 Цель дизайна

Спроектировать оптимизированный API endpoint для получения недавних алертов для dashboard с фокусом на:
- **Производительность** - быстрая загрузка для dashboard (< 100ms)
- **Компактность** - минимальный набор полей
- **Гибкость** - опциональное включение classification
- **Интеграция** - использование существующих компонентов

### 1.2 Архитектурные принципы

1. **Reuse Existing Components** - переиспользование AlertHistoryRepository и ClassificationEnricher
2. **Performance First** - оптимизация для dashboard использования
3. **Graceful Degradation** - работа без classification service
4. **Backward Compatibility** - совместимость с существующим dashboard

### 1.3 Компонентная архитектура

```
┌─────────────────────────────────────────────────────────────┐
│              HTTP Handler Layer                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │ DashboardAlertsHandler                               │   │
│  │ - Parse query params                                 │   │
│  │ - Validate parameters                                │   │
│  │ - Call repository                                    │   │
│  │ - Enrich with classification (optional)             │   │
│  │ - Format response                                    │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│            Repository Layer                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ AlertHistory │  │ Classification│  │ Cache        │      │
│  │ Repository   │  │ Enricher     │  │ (optional)   │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Детальный дизайн компонентов

### 2.1 DashboardAlertsHandler

**Назначение:** HTTP handler для dashboard alerts endpoint

**Интерфейс:**
```go
type DashboardAlertsHandler struct {
    historyRepo         core.AlertHistoryRepository
    classificationEnricher ui.ClassificationEnricher // optional
    cache               cache.Cache                    // optional, for response caching
    logger              *slog.Logger
}

func (h *DashboardAlertsHandler) GetRecentAlerts(w http.ResponseWriter, r *http.Request)
```

**Методы:**
- `GetRecentAlerts` - основной handler метод
- `parseQueryParams` - парсинг query параметров
- `formatResponse` - форматирование response в компактный формат
- `applyFilters` - применение фильтров (status, severity)

### 2.2 Response Format

**Компактный формат:**
```go
type DashboardAlertResponse struct {
    Alerts []DashboardAlert `json:"alerts"`
    Count  int             `json:"count"`
    Limit  int             `json:"limit"`
    Filters *ResponseFilters `json:"filters,omitempty"`
    Timestamp string       `json:"timestamp"`
}

type DashboardAlert struct {
    Fingerprint string            `json:"fingerprint"`
    AlertName   string            `json:"alert_name"`
    Status      string            `json:"status"`
    Severity    string            `json:"severity"`
    Summary     string            `json:"summary,omitempty"`
    StartsAt    time.Time         `json:"starts_at"`
    Labels      map[string]string `json:"labels,omitempty"` // только важные labels

    // Опционально (если include_classification=true)
    Classification *ClassificationSummary `json:"classification,omitempty"`
}

type ClassificationSummary struct {
    Severity   string  `json:"severity"`
    Confidence float64 `json:"confidence"`
    Source     string  `json:"source"`
}
```

---

## 3. Формат данных

### 3.1 Request Parameters

```
GET /api/dashboard/alerts/recent?limit=10&status=firing&severity=critical&include_classification=true
```

**Query Parameters:**
- `limit` (int, optional, default: 10, max: 50) - количество алертов
- `status` (string, optional) - фильтр: "firing" или "resolved"
- `severity` (string, optional) - фильтр: "critical", "warning", "info", "noise"
- `include_classification` (bool, optional, default: false) - включить classification

### 3.2 Response Format

```json
{
  "alerts": [
    {
      "fingerprint": "abc123",
      "alert_name": "HighCPU",
      "status": "firing",
      "severity": "critical",
      "summary": "CPU usage exceeds 90%",
      "starts_at": "2025-11-20T10:00:00Z",
      "labels": {
        "namespace": "production",
        "instance": "app-1"
      },
      "classification": {
        "severity": "critical",
        "confidence": 0.85,
        "source": "llm"
      }
    }
  ],
  "count": 1,
  "limit": 10,
  "filters": {
    "status": "firing",
    "severity": "critical"
  },
  "timestamp": "2025-11-20T10:05:00Z"
}
```

---

## 4. API контракты

### 4.1 GET /api/dashboard/alerts/recent

**Request:**
```
GET /api/dashboard/alerts/recent?limit=10&status=firing&include_classification=true
```

**Response (200 OK):**
```json
{
  "alerts": [...],
  "count": 10,
  "limit": 10,
  "timestamp": "2025-11-20T10:05:00Z"
}
```

**Response (400 Bad Request):**
```json
{
  "error": "Invalid limit parameter",
  "message": "Limit must be between 1 and 50"
}
```

**Response (500 Internal Server Error):**
```json
{
  "error": "Internal server error",
  "message": "Failed to retrieve recent alerts"
}
```

---

## 5. Интеграция с существующими компонентами

### 5.1 AlertHistoryRepository Integration

**Использование:**
- `AlertHistoryRepository.GetRecentAlerts()` - получение недавних алертов
- Применение фильтров через `AlertFilters` перед вызовом

**Оптимизация:**
- Использовать существующий метод (уже оптимизирован)
- Добавить фильтры через `AlertFilters` структуру

### 5.2 ClassificationEnricher Integration

**Использование:**
- `ClassificationEnricher.EnrichAlerts()` - обогащение classification
- Опциональное использование (только если `include_classification=true`)

**Оптимизация:**
- Batch enrichment для списка алертов
- Graceful degradation при отсутствии enricher

### 5.3 Response Caching (Optional)

**Стратегия:**
- Кэширование response на 5-10 секунд
- Cache key: `dashboard:alerts:recent:{limit}:{status}:{severity}`
- Invalidate при новых алертах (опционально)

---

## 6. Сценарии ошибок и Edge Cases

### 6.1 Invalid Parameters

**Сценарий:** Некорректные query параметры

**Обработка:**
- Валидация limit (1-50)
- Валидация status (firing/resolved)
- Валидация severity (critical/warning/info/noise)
- Возврат 400 Bad Request с описанием ошибки

### 6.2 Repository Error

**Сценарий:** Ошибка при получении алертов из repository

**Обработка:**
- Логирование ошибки (ERROR level)
- Возврат 500 Internal Server Error
- Graceful error message (без деталей внутренней ошибки)

### 6.3 Classification Service Unavailable

**Сценарий:** Classification service недоступен, но `include_classification=true`

**Обработка:**
- Graceful degradation: возврат алертов без classification
- Логирование предупреждения (WARN level)
- Продолжение работы без блокировки

### 6.4 Empty Result

**Сценарий:** Нет недавних алертов

**Обработка:**
- Возврат пустого массива `[]`
- `count: 0`
- Нормальный response (200 OK)

---

## 7. Тестирование стратегия

### 7.1 Unit Tests

**Покрытие:**
- ParseQueryParams (90%+ coverage)
- FormatResponse (90%+ coverage)
- ApplyFilters (90%+ coverage)
- Error handling (100% coverage)

**Тесты:**
- Valid parameters
- Invalid parameters
- Edge cases (empty result, max limit)
- Classification integration (with/without)
- Filter combinations

### 7.2 Integration Tests

**Покрытие:**
- End-to-end flow: request → repository → enricher → response
- Performance: response time < 100ms
- Error scenarios: repository error, classification error

### 7.3 Performance Tests

**Метрики:**
- Response time: < 100ms (p95) для 10 алертов
- Response time: < 200ms (p95) для 50 алертов
- Throughput: > 100 req/s

---

## 8. Производительность и оптимизация

### 8.1 SQL Optimization

**Запрос:**
- Использовать существующий `GetRecentAlerts` (уже оптимизирован)
- Добавить WHERE условия для фильтров (status, severity)
- Использовать индексы: `idx_alerts_starts_at_desc`, `idx_alerts_status`

### 8.2 Response Caching

**Стратегия:**
- Cache TTL: 5-10 секунд
- Cache key: включает все параметры запроса
- Invalidation: при новых алертах (опционально)

### 8.3 Classification Optimization

**Стратегия:**
- Batch enrichment (10-20 алертов за раз)
- Request-scoped cache (из ClassificationEnricher)
- Опциональное включение (только если запрошено)

---

## 9. Безопасность

### 9.1 Input Validation

**Меры:**
- Валидация limit (1-50, предотвращение DoS)
- Валидация status и severity (whitelist)
- Sanitization всех строковых параметров

### 9.2 Rate Limiting

**Меры:**
- Rate limiting через middleware (100 req/min per IP)
- Защита от злоупотребления

---

## 10. Accessibility (N/A)

Endpoint является API endpoint, accessibility не применимо.

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Author:** AI Assistant (Enterprise Architecture Team)
**Status:** ✅ APPROVED FOR IMPLEMENTATION
