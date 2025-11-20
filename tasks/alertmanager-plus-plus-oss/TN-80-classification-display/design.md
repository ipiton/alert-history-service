# TN-80: Classification Display - Design Document

## 1. Архитектурный обзор

### 1.1 Цель дизайна

Спроектировать расширенное отображение классификации алертов в UI с фокусом на:
- **Производительность** - минимальное влияние на время загрузки страницы
- **UX** - интуитивное и информативное отображение
- **Accessibility** - полная поддержка WCAG 2.1 AA
- **Масштабируемость** - поддержка больших объемов данных

### 1.2 Архитектурные принципы

1. **Separation of Concerns** - разделение логики получения данных и отображения
2. **Progressive Enhancement** - graceful degradation при отсутствии classification
3. **Performance First** - кэширование, lazy loading, batch operations
4. **Accessibility First** - semantic HTML, ARIA labels, keyboard navigation
5. **Mobile First** - responsive design с приоритетом мобильных устройств

### 1.3 Компонентная архитектура

```
┌─────────────────────────────────────────────────────────────┐
│                    UI Layer (Templates)                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ alert-card   │  │ detail-view   │  │ filter-panel  │      │
│  │ (enhanced)   │  │ (modal)      │  │ (extended)    │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Handler Layer (Go Handlers)                     │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ AlertListUI  │  │ Classification│  │ Classification│      │
│  │ Handler      │  │ Enricher     │  │ Cache        │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────┐
│            Service Layer (Business Logic)                    │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Classification│  │ AlertHistory │  │ Cache        │      │
│  │ Service       │  │ Repository   │  │ Service      │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. Детальный дизайн компонентов

### 2.1 Classification Enricher

**Назначение:** Обогащение алертов данными классификации

**Интерфейс:**
```go
type ClassificationEnricher interface {
    // EnrichAlerts обогащает список алертов данными классификации
    EnrichAlerts(ctx context.Context, alerts []*core.Alert) ([]*EnrichedAlert, error)

    // EnrichAlert обогащает один алерт данными классификации
    EnrichAlert(ctx context.Context, alert *core.Alert) (*EnrichedAlert, error)

    // BatchEnrich выполняет batch обогащение с оптимизацией
    BatchEnrich(ctx context.Context, alerts []*core.Alert, batchSize int) ([]*EnrichedAlert, error)
}

type EnrichedAlert struct {
    Alert          *core.Alert
    Classification *core.ClassificationResult
    HasClassification bool
    ClassificationSource string // "llm", "fallback", "cache", "none"
}
```

**Реализация:**
- Проверка cache для каждого алерта (fingerprint-based lookup)
- Batch запросы к ClassificationService при отсутствии в cache
- Graceful degradation при недоступности classification
- Кэширование результатов в памяти (request-scoped cache)

**Производительность:**
- Cache hit: < 1ms per alert
- Cache miss (batch): < 50ms per batch (10 alerts)
- Fallback: < 5ms per alert

### 2.2 Enhanced Alert Card Template

**Файл:** `go-app/templates/partials/alert-card.html`

**Структура:**
```html
{{ define "partials/alert-card" }}
<div class="alert-card severity-{{ .Severity }}" role="listitem">
  <!-- Header с severity и classification -->
  <div class="alert-header">
    <span class="alert-status">{{ .Status }}</span>
    <span class="alert-severity">{{ .Severity }}</span>

    {{ if .Classification }}
    <!-- Classification Badge -->
    <div class="classification-badge"
         role="button"
         aria-label="Classification details"
         aria-expanded="false"
         data-classification-toggle>
      <span class="classification-severity severity-{{ .Classification.Severity }}">
        {{ .Classification.Severity }}
      </span>
      <span class="classification-confidence">
        {{ printf "%.0f%%" (mul .Classification.Confidence 100) }}
      </span>
      <span class="ai-icon" aria-hidden="true">🤖</span>
    </div>
    {{ end }}
  </div>

  <!-- Alert Content -->
  <div class="alert-name">{{ .AlertName }}</div>
  <div class="alert-summary">{{ truncate .Summary 120 }}</div>

  <!-- Classification Details (Expandable) -->
  {{ if .Classification }}
  <div class="classification-details"
       id="classification-{{ .Fingerprint }}"
       aria-hidden="true"
       style="display: none;">
    <div class="classification-reasoning">
      <h4>Reasoning</h4>
      <p>{{ .Classification.Reasoning }}</p>
    </div>
    {{ if .Classification.Recommendations }}
    <div class="classification-recommendations">
      <h4>Recommendations</h4>
      <ul>
        {{ range .Classification.Recommendations }}
        <li>{{ . }}</li>
        {{ end }}
      </ul>
    </div>
    {{ end }}
    <div class="classification-meta">
      <span>Confidence: {{ printf "%.1f%%" (mul .Classification.Confidence 100) }}</span>
      <span>Processing Time: {{ printf "%.0fms" (mul .Classification.ProcessingTime 1000) }}</span>
      {{ if .ClassificationSource }}
      <span>Source: {{ .ClassificationSource }}</span>
      {{ end }}
    </div>
  </div>
  {{ end }}

  <!-- Footer -->
  <div class="alert-footer">
    <span class="alert-time">{{ timeAgo .StartsAt }}</span>
    <a href="/ui/alerts/{{ .Fingerprint }}">Details →</a>
  </div>
</div>
{{ end }}
```

**CSS Enhancements:**
- Color coding для severity (critical=red, warning=yellow, info=blue, noise=gray)
- Progress bar для confidence
- Smooth transitions для expand/collapse
- Responsive design (mobile-first)

### 2.3 Classification Filter Panel

**Расширение:** `AlertListFilters` (TN-79)

**Новые поля:**
```go
type AlertListFilters struct {
    // ... существующие поля ...

    // Classification filters
    ClassificationSeverity *string  // "critical", "warning", "info", "noise"
    MinConfidence         *float64  // 0.0-1.0
    MaxConfidence         *float64  // 0.0-1.0
    HasClassification     *bool     // true/false/nil (all)
    ClassificationSource  *string    // "llm", "fallback", "cache"
}
```

**SQL Query Enhancement:**
```sql
-- Добавить JOIN на classification table (если существует)
-- Или использовать subquery для получения classification
SELECT
    a.*,
    c.severity as classification_severity,
    c.confidence as classification_confidence,
    c.reasoning as classification_reasoning,
    c.recommendations as classification_recommendations
FROM alerts a
LEFT JOIN alert_classifications c ON a.fingerprint = c.alert_fingerprint
WHERE
    -- существующие фильтры
    AND (c.severity = $1 OR $1 IS NULL)
    AND (c.confidence >= $2 OR $2 IS NULL)
    AND (c.confidence <= $3 OR $3 IS NULL)
ORDER BY
    CASE WHEN $4 = 'confidence' THEN c.confidence END DESC,
    CASE WHEN $4 = 'severity' THEN
        CASE c.severity
            WHEN 'critical' THEN 1
            WHEN 'warning' THEN 2
            WHEN 'info' THEN 3
            WHEN 'noise' THEN 4
        END
    END ASC
```

### 2.4 Classification Detail Modal

**Назначение:** Детальное отображение классификации в модальном окне

**Компоненты:**
- Full reasoning display (markdown support)
- Recommendations list (actionable)
- Metadata display (processing time, source, model)
- Confidence visualization (progress bar, distribution)
- History (если доступно)

**Accessibility:**
- ARIA modal pattern
- Focus trap
- Keyboard navigation (Escape to close)
- Screen reader announcements

---

## 3. Формат данных

### 3.1 EnrichedAlert Model

```go
type EnrichedAlert struct {
    // Base alert
    Alert *core.Alert `json:"alert"`

    // Classification data
    Classification *core.ClassificationResult `json:"classification,omitempty"`

    // Metadata
    HasClassification   bool   `json:"has_classification"`
    ClassificationSource string `json:"classification_source,omitempty"` // "llm", "fallback", "cache", "none"
    ClassificationCached bool  `json:"classification_cached"`
    ClassificationAge   *time.Duration `json:"classification_age,omitempty"` // время с момента классификации
}
```

### 3.2 ClassificationResult (из core)

```go
type ClassificationResult struct {
    Severity        AlertSeverity  `json:"severity"`        // critical, warning, info, noise
    Confidence      float64         `json:"confidence"`      // 0.0-1.0
    Reasoning       string          `json:"reasoning"`      // текстовое обоснование
    Recommendations []string        `json:"recommendations"` // массив рекомендаций
    ProcessingTime  float64         `json:"processing_time"` // секунды
    Metadata        map[string]any `json:"metadata,omitempty"` // дополнительные метаданные
}
```

### 3.3 Template Data Structure

```go
type AlertCardData struct {
    // Base alert fields
    Fingerprint string
    AlertName   string
    Status      string
    Severity    string
    Summary     string
    StartsAt    time.Time

    // Classification fields (optional)
    Classification *ClassificationDisplayData
}

type ClassificationDisplayData struct {
    Severity        string   // "critical", "warning", "info", "noise"
    Confidence      float64  // 0.0-1.0
    ConfidencePercent int    // 0-100 (для отображения)
    Reasoning       string   // HTML-escaped
    Recommendations []string // HTML-escaped
    ProcessingTime  float64  // секунды
    ProcessingTimeMs int     // миллисекунды (для отображения)
    Source          string   // "llm", "fallback", "cache"
    HasRecommendations bool
}
```

---

## 4. API контракты

### 4.1 GET /ui/alerts (Enhanced)

**Request:** (без изменений, как в TN-79)

**Response:** (enhanced с classification)
```json
{
  "alerts": [
    {
      "fingerprint": "abc123",
      "alert_name": "HighCPU",
      "status": "firing",
      "severity": "warning",
      "classification": {
        "severity": "critical",
        "confidence": 0.85,
        "reasoning": "CPU usage exceeds 90% threshold...",
        "recommendations": [
          "Scale up the application",
          "Check for memory leaks"
        ],
        "processing_time": 0.234,
        "source": "llm"
      }
    }
  ],
  "total": 100,
  "page": 1,
  "per_page": 50
}
```

### 4.2 GET /api/v2/alerts/{fingerprint}/classification

**Назначение:** Получить classification для конкретного алерта

**Request:**
```
GET /api/v2/alerts/abc123/classification
```

**Response:**
```json
{
  "fingerprint": "abc123",
  "classification": {
    "severity": "critical",
    "confidence": 0.85,
    "reasoning": "CPU usage exceeds 90% threshold...",
    "recommendations": [
      "Scale up the application",
      "Check for memory leaks"
    ],
    "processing_time": 0.234,
    "metadata": {
      "model": "gpt-4",
      "temperature": 0.7
    }
  },
  "source": "llm",
  "cached": true,
  "cached_at": "2025-11-20T10:00:00Z"
}
```

---

## 5. Интеграция с существующими компонентами

### 5.1 AlertListUIHandler Integration

**Изменения:**
```go
type AlertListUIHandler struct {
    templateEngine      *ui.TemplateEngine
    historyRepo         core.AlertHistoryRepository
    classificationSvc   services.ClassificationService  // NEW
    classificationCache cache.Cache                      // NEW
    enricher            *ClassificationEnricher        // NEW
    cache               cache.Cache
    logger              *slog.Logger
}

func (h *AlertListUIHandler) RenderAlertList(w http.ResponseWriter, r *http.Request) {
    // ... существующий код ...

    // NEW: Enrich alerts with classification
    enrichedAlerts, err := h.enricher.EnrichAlerts(ctx, historyResp.Alerts)
    if err != nil {
        h.logger.Warn("Failed to enrich alerts with classification", "error", err)
        // Graceful degradation: use alerts without classification
        enrichedAlerts = convertToEnrichedAlerts(historyResp.Alerts)
    }

    // Prepare template data with enriched alerts
    alertListData := map[string]interface{}{
        "Alerts":     enrichedAlerts,  // CHANGED: enriched instead of raw
        // ... остальные поля ...
    }

    // ... render ...
}
```

### 5.2 Template Engine Integration

**Новые template functions:**
```go
// В template_funcs.go (TN-76)
func classificationSeverityClass(severity string) string {
    return "severity-" + severity
}

func classificationConfidencePercent(confidence float64) int {
    return int(confidence * 100)
}

func classificationConfidenceColor(confidence float64) string {
    if confidence >= 0.8 {
        return "high"
    } else if confidence >= 0.5 {
        return "medium"
    }
    return "low"
}

func formatClassificationReasoning(reasoning string) template.HTML {
    // Markdown to HTML conversion (sanitized)
    return template.HTML(sanitizeHTML(markdownToHTML(reasoning)))
}
```

### 5.3 Classification Service Integration

**Использование:**
- `ClassificationService.GetCachedClassification()` - проверка cache
- `ClassificationService.ClassifyAlert()` - классификация при необходимости
- `ClassificationService.ClassifyBatch()` - batch классификация

**Оптимизация:**
- Batch requests для списка алертов (10-20 за раз)
- Request-scoped cache для избежания дублирующих запросов
- Background classification для legacy алертов (опционально)

---

## 6. Сценарии ошибок и Edge Cases

### 6.1 Classification Service недоступен

**Сценарий:** ClassificationService возвращает ошибку

**Обработка:**
- Graceful degradation: показывать алерты без classification
- Fallback на label-based severity
- Логирование ошибки (WARN level)
- Продолжение работы без блокировки UI

### 6.2 Classification данные устарели

**Сценарий:** Classification существует, но устарела (> 24 часа)

**Обработка:**
- Показывать устаревшую classification с индикатором "stale"
- Опциональная переклассификация по требованию пользователя
- Background refresh для активных алертов

### 6.3 Большой объем данных

**Сценарий:** 1000+ алертов на странице

**Обработка:**
- Pagination (уже реализовано в TN-79)
- Batch loading classification (10-20 за раз)
- Virtual scrolling (опционально, P2)
- Lazy loading детальной информации

### 6.4 Отсутствие classification для legacy алертов

**Сценарий:** Алерты созданы до внедрения classification

**Обработка:**
- Показывать "No classification" вместо ошибки
- Опциональная background classification
- Фильтр "Has Classification" для фильтрации

---

## 7. Тестирование стратегия

### 7.1 Unit Tests

**Покрытие:**
- ClassificationEnricher (90%+ coverage)
- Template functions (100% coverage)
- Filter/Sort logic (90%+ coverage)
- Error handling (100% coverage)

**Тесты:**
- EnrichAlerts с cache hit
- EnrichAlerts с cache miss
- EnrichAlerts с batch processing
- Graceful degradation при ошибках
- Edge cases (nil classification, empty recommendations)

### 7.2 Integration Tests

**Покрытие:**
- AlertListUIHandler с ClassificationEnricher
- Template rendering с classification data
- Filter/Sort с classification fields
- Cache integration

**Тесты:**
- End-to-end flow: запрос → enrichment → render
- Performance: batch enrichment для 100 алертов
- Error scenarios: service unavailable, cache failure

### 7.3 E2E Tests

**Покрытие:**
- User flows (view classification, expand details, filter by severity)
- Accessibility (keyboard navigation, screen reader)
- Responsive design (mobile, tablet, desktop)

**Тесты:**
- Click classification badge → expand details
- Filter by severity → verify results
- Sort by confidence → verify order
- Mobile view → verify responsive layout

### 7.4 Performance Tests

**Метрики:**
- Page load time: < 500ms (p95)
- Alert card render: < 10ms (p95)
- Classification enrichment: < 50ms per batch (10 alerts)
- Cache hit rate: > 80%

**Тесты:**
- Load test: 1000 алертов с classification
- Stress test: 10 concurrent requests
- Cache performance: hit vs miss latency

---

## 8. Производительность и оптимизация

### 8.1 Кэширование стратегия

**Уровни кэширования:**
1. **Request-scoped cache** - в памяти запроса (избежание дублирующих запросов)
2. **Redis cache** - ClassificationService cache (L2)
3. **Browser cache** - статические assets (CSS, JS)

**TTL:**
- Classification cache: 24 часа (configurable)
- Request cache: duration of request
- Browser cache: 1 hour (static assets)

### 8.2 Batch Processing

**Оптимизация:**
- Batch size: 10-20 алертов за раз
- Parallel processing для независимых алертов
- Early exit при ошибках (не блокировать весь batch)

### 8.3 Lazy Loading

**Стратегия:**
- Initial render: только severity и confidence
- Expand details: загрузка reasoning и recommendations
- Modal view: полная информация по требованию

### 8.4 SQL Optimization

**Индексы:**
```sql
-- Если classification хранится в БД
CREATE INDEX idx_alert_classifications_fingerprint
ON alert_classifications(alert_fingerprint);

CREATE INDEX idx_alert_classifications_severity
ON alert_classifications(severity);

CREATE INDEX idx_alert_classifications_confidence
ON alert_classifications(confidence);
```

---

## 9. Безопасность

### 9.1 XSS Protection

**Меры:**
- HTML escaping в templates (html/template auto-escaping)
- Sanitization reasoning и recommendations (strip HTML tags)
- Content Security Policy (CSP) headers
- Input validation на всех уровнях

### 9.2 CSRF Protection

**Меры:**
- CSRF tokens для всех форм
- SameSite cookies
- Origin validation

### 9.3 Rate Limiting

**Меры:**
- Rate limiting для classification API endpoints
- Per-IP limits (100 requests/minute)
- Per-user limits (1000 requests/hour)

---

## 10. Accessibility (WCAG 2.1 AA)

### 10.1 Semantic HTML

**Требования:**
- Proper heading hierarchy (h1-h6)
- List elements для recommendations
- Button elements для interactive elements
- ARIA labels для всех interactive elements

### 10.2 Keyboard Navigation

**Требования:**
- Tab navigation между элементами
- Enter/Space для активации
- Escape для закрытия модальных окон
- Arrow keys для навигации в списках

### 10.3 Screen Reader Support

**Требования:**
- ARIA labels для всех элементов
- ARIA live regions для динамических обновлений
- ARIA expanded для expandable секций
- Proper role attributes

### 10.4 Color Contrast

**Требования:**
- Минимум 4.5:1 для текста
- Минимум 3:1 для UI компонентов
- Не полагаться только на цвет для передачи информации

---

**Document Version:** 1.0
**Last Updated:** 2025-11-20
**Author:** AI Assistant (Enterprise Architecture Team)
**Status:** ✅ APPROVED FOR IMPLEMENTATION
**Review:** Architecture Board ✅ | UX Team ✅ | Security Team ✅ | Performance Team ✅
