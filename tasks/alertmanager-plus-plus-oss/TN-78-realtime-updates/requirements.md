# TN-78: Real-time Updates (SSE/WebSocket) — Requirements

**Task ID**: TN-78
**Module**: Phase 9: Dashboard & UI
**Priority**: HIGH (P1 - Must Have for Real-time UX)
**Depends On**: TN-76 (Dashboard Template Engine), TN-77 (Modern Dashboard Page)
**Target Quality**: 150% (Grade A+ Enterprise)
**Estimated Effort**: 12-16 hours
**Started**: 2025-11-20

---

## 📋 Executive Summary

Реализация системы real-time обновлений для Modern Dashboard Page (TN-77) с поддержкой двух протоколов: **Server-Sent Events (SSE)** для односторонней передачи данных и **WebSocket** для двусторонней коммуникации. Система обеспечивает автоматическое обновление dashboard без перезагрузки страницы, улучшая UX и снижая нагрузку на сервер.

**Цель**: Обеспечить real-time обновления dashboard с минимальной задержкой (<100ms), высокой надежностью (99.9% uptime), и поддержкой graceful degradation.

---

## 🎯 Business Value

### Проблема
- Текущий dashboard требует ручной перезагрузки страницы для обновления данных
- Auto-refresh каждые 30 секунд (TN-77) недостаточно для критических алертов
- Нет возможности получать мгновенные уведомления о новых алертах/силенсах

### Решение
- Real-time обновления через SSE/WebSocket
- Автоматическое обновление UI при изменении данных
- Мгновенные уведомления о критических событиях
- Снижение нагрузки на сервер (меньше HTTP запросов)

### Ожидаемый эффект
- ⚡ **Скорость**: Мгновенные обновления (<100ms latency)
- 📊 **Нагрузка**: Снижение HTTP запросов на 80%+
- 👥 **UX**: Улучшенный пользовательский опыт
- 🔔 **Алертинг**: Мгновенные уведомления о критических событиях

---

## 📊 Functional Requirements

### FR-1: Server-Sent Events (SSE) Support
**Priority**: HIGH (P1)
**Description**: Реализовать SSE endpoint для односторонней передачи событий от сервера к клиенту.

**Requirements**:
- ✅ Endpoint: `GET /api/v2/events/stream` (SSE)
- ✅ Content-Type: `text/event-stream`
- ✅ Поддержка CORS для cross-origin запросов
- ✅ Keep-alive ping каждые 30 секунд
- ✅ Автоматическое переподключение на клиенте (exponential backoff)
- ✅ Graceful shutdown при закрытии соединения

**Acceptance Criteria**:
- [ ] SSE endpoint возвращает `text/event-stream` с правильными headers
- [ ] Клиент получает события в формате SSE (`data: {...}\n\n`)
- [ ] Keep-alive ping отправляется каждые 30 секунд
- [ ] Переподключение работает при разрыве соединения
- [ ] Graceful shutdown закрывает все соединения

---

### FR-2: WebSocket Support (Enhanced)
**Priority**: HIGH (P1)
**Description**: Расширить существующий WebSocketHub (TN-136) для поддержки dashboard events.

**Requirements**:
- ✅ Переиспользовать существующий `WebSocketHub` из `silence_ws.go`
- ✅ Добавить новые event types для dashboard (alert_*, stats_*, health_*)
- ✅ Поддержка ping/pong keep-alive
- ✅ Graceful degradation при ошибках соединения
- ✅ Rate limiting для предотвращения DoS

**Acceptance Criteria**:
- [ ] WebSocket endpoint `/ws/dashboard` работает
- [ ] Поддерживаются новые event types (alert_created, alert_resolved, stats_updated, health_changed)
- [ ] Ping/pong keep-alive работает корректно
- [ ] Rate limiting предотвращает злоупотребления
- [ ] Graceful degradation при ошибках

---

### FR-3: Event Types & Payloads
**Priority**: HIGH (P1)
**Description**: Определить и реализовать все типы событий для dashboard.

**Event Types**:
1. **Alert Events**:
   - `alert_created` - новый алерт создан
   - `alert_resolved` - алерт разрешен
   - `alert_firing` - алерт перешел в firing
   - `alert_inhibited` - алерт подавлен inhibition rule

2. **Stats Events**:
   - `stats_updated` - статистика обновлена (firing/resolved counts)

3. **Silence Events** (reuse from TN-136):
   - `silence_created` - silence создан
   - `silence_updated` - silence обновлен
   - `silence_deleted` - silence удален
   - `silence_expired` - silence истек

4. **Health Events**:
   - `health_changed` - статус здоровья компонента изменился

5. **System Events**:
   - `system_notification` - системные уведомления

**Acceptance Criteria**:
- [ ] Все event types определены и документированы
- [ ] Payload структуры валидны и типизированы
- [ ] События отправляются при соответствующих изменениях
- [ ] Клиент корректно обрабатывает все типы событий

---

### FR-4: Dashboard Integration (TN-77)
**Priority**: HIGH (P1)
**Description**: Интегрировать real-time updates в Modern Dashboard Page (TN-77).

**Requirements**:
- ✅ JavaScript клиент для SSE/WebSocket
- ✅ Автоматическое обновление секций dashboard при получении событий
- ✅ Toast notifications для важных событий
- ✅ Visual indicators для обновленных элементов
- ✅ Graceful fallback на polling при недоступности SSE/WebSocket

**Acceptance Criteria**:
- [ ] Dashboard автоматически обновляется при получении событий
- [ ] Toast notifications показываются для критических событий
- [ ] Visual indicators (badges, highlights) работают корректно
- [ ] Fallback на polling работает при недоступности real-time

---

### FR-5: Event Broadcasting System
**Priority**: HIGH (P1)
**Description**: Система широковещательной рассылки событий всем подключенным клиентам.

**Requirements**:
- ✅ Централизованный EventBus для всех событий
- ✅ Поддержка множественных подписчиков (SSE + WebSocket)
- ✅ Thread-safe broadcasting
- ✅ Event filtering по типам (опционально)
- ✅ Metrics для отслеживания broadcast performance

**Acceptance Criteria**:
- [ ] EventBus корректно рассылает события всем подписчикам
- [ ] Thread-safe операции (нет race conditions)
- [ ] Event filtering работает корректно
- [ ] Metrics записываются для каждого broadcast

---

## 🔧 Non-Functional Requirements

### NFR-1: Performance
**Priority**: HIGH (P1)
**Targets**:
- ✅ Latency: <100ms от события до доставки клиенту
- ✅ Throughput: >1,000 events/second
- ✅ Connection overhead: <1MB memory per connection
- ✅ CPU usage: <5% при 100 активных соединениях

**Measurement**:
- Prometheus metrics: `realtime_event_latency_seconds`, `realtime_events_per_second`
- Benchmarks: Load testing с 100+ concurrent connections

---

### NFR-2: Reliability
**Priority**: HIGH (P1)
**Targets**:
- ✅ Uptime: 99.9% (downtime <8.76 hours/year)
- ✅ Auto-reconnect: Exponential backoff (1s → 30s max)
- ✅ Graceful degradation: Fallback на polling при недоступности
- ✅ Error recovery: Автоматическое восстановление при ошибках

**Measurement**:
- Prometheus metrics: `realtime_connection_uptime`, `realtime_reconnect_total`
- Monitoring: Alerting на длительные разрывы соединений

---

### NFR-3: Scalability
**Priority**: MEDIUM (P2)
**Targets**:
- ✅ Поддержка 1,000+ concurrent connections
- ✅ Horizontal scaling: Multiple instances с shared event bus
- ✅ Memory efficiency: <2MB per 100 connections
- ✅ CPU efficiency: Linear scaling

**Measurement**:
- Load testing: 1,000+ concurrent connections
- Memory profiling: Heap analysis
- CPU profiling: pprof analysis

---

### NFR-4: Security
**Priority**: HIGH (P1)
**Targets**:
- ✅ Origin validation для WebSocket (configurable)
- ✅ Rate limiting: 10 connections per IP
- ✅ Authentication: Optional (JWT/Bearer token)
- ✅ CORS: Configurable allowed origins для SSE
- ✅ Input validation: Sanitize all event payloads

**Measurement**:
- Security audit: OWASP Top 10 compliance
- Penetration testing: WebSocket/SSE security testing

---

### NFR-5: Observability
**Priority**: HIGH (P1)
**Targets**:
- ✅ Prometheus metrics: connections, events, latency, errors
- ✅ Structured logging: Все события логируются
- ✅ Tracing: OpenTelemetry support (optional)
- ✅ Health checks: `/health/realtime` endpoint

**Metrics**:
- `realtime_connections_active` (Gauge)
- `realtime_events_total` (Counter by type)
- `realtime_event_latency_seconds` (Histogram)
- `realtime_errors_total` (Counter by error_type)
- `realtime_reconnect_total` (Counter)

---

## 🏗️ Technical Architecture

### Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Browser (Dashboard)                       │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  SSE Client  │  WebSocket Client  │  Fallback Polling  │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                          │              │
                    SSE (GET)      WebSocket (WS)
                          │              │
┌─────────────────────────────────────────────────────────────┐
│              Real-time Event System (TN-78)                  │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              EventBus (Central Hub)                    │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │  │
│  │  │ SSE Handler │  │WS Hub (ext)  │  │  Event       │ │  │
│  │  │             │  │              │  │  Publishers  │ │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘ │  │
│  └───────────────────────────────────────────────────────┘  │
│                          │                                   │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Event Sources:                                        │  │
│  │    - AlertProcessor (alert_*)                         │  │
│  │    - SilenceManager (silence_*)                       │  │
│  │    - StatsCollector (stats_updated)                   │  │
│  │    - HealthMonitor (health_changed)                    │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### Component Design

#### 1. EventBus (Central Hub)
**Responsibility**: Централизованная система управления событиями и подписчиками.

**Interface**:
```go
type EventBus interface {
    // Subscribe adds a subscriber (SSE or WebSocket)
    Subscribe(subscriber EventSubscriber) error

    // Unsubscribe removes a subscriber
    Unsubscribe(subscriber EventSubscriber) error

    // Publish broadcasts an event to all subscribers
    Publish(event Event) error

    // GetActiveSubscribers returns count of active subscribers
    GetActiveSubscribers() int
}
```

**Implementation**:
- Thread-safe map of subscribers (sync.RWMutex)
- Buffered channel for events (capacity 1000)
- Background goroutine for broadcasting
- Metrics recording for each operation

---

#### 2. SSE Handler
**Responsibility**: Обработка Server-Sent Events соединений.

**Endpoint**: `GET /api/v2/events/stream`

**Features**:
- HTTP/1.1 streaming response
- Keep-alive ping каждые 30 секунд
- Graceful shutdown при закрытии соединения
- CORS support для cross-origin

**Implementation**:
```go
type SSEHandler struct {
    eventBus EventBus
    logger   *slog.Logger
    metrics  *RealtimeMetrics
}

func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")

    // Create SSE subscriber
    subscriber := NewSSESubscriber(w, r.Context())
    h.eventBus.Subscribe(subscriber)
    defer h.eventBus.Unsubscribe(subscriber)

    // Send keep-alive pings
    // Forward events from EventBus
}
```

---

#### 3. WebSocket Hub Enhancement
**Responsibility**: Расширение существующего WebSocketHub для dashboard events.

**Changes**:
- Добавить новые event types (alert_*, stats_*, health_*)
- Интеграция с EventBus
- Rate limiting для новых соединений
- Enhanced metrics

**Implementation**:
- Расширить `WebSocketHub` из `silence_ws.go`
- Добавить методы для dashboard events
- Интегрировать с EventBus

---

#### 4. Event Publishers
**Responsibility**: Публикация событий из различных источников.

**Sources**:
1. **AlertProcessor**: `alert_created`, `alert_resolved`, `alert_firing`, `alert_inhibited`
2. **SilenceManager**: `silence_created`, `silence_updated`, `silence_deleted`, `silence_expired` (reuse)
3. **StatsCollector**: `stats_updated` (периодически, каждые 10s)
4. **HealthMonitor**: `health_changed` (при изменении статуса)

**Implementation**:
```go
type EventPublisher interface {
    PublishAlertEvent(eventType string, alert *core.Alert) error
    PublishStatsEvent(stats *DashboardStats) error
    PublishHealthEvent(component string, status string) error
}
```

---

#### 5. JavaScript Client
**Responsibility**: Клиентская библиотека для подключения к SSE/WebSocket.

**Features**:
- Auto-detection: SSE preferred, WebSocket fallback
- Auto-reconnect: Exponential backoff
- Event handling: Callbacks для каждого event type
- UI updates: Автоматическое обновление dashboard секций

**Implementation**:
```javascript
class RealtimeClient {
    constructor(options) {
        this.options = options;
        this.eventBus = new EventTarget();
        this.connection = null;
    }

    connect() {
        // Try SSE first
        if (this.supportsSSE()) {
            this.connectSSE();
        } else if (this.supportsWebSocket()) {
            this.connectWebSocket();
        } else {
            this.fallbackPolling();
        }
    }

    on(eventType, callback) {
        this.eventBus.addEventListener(eventType, callback);
    }

    updateDashboard(event) {
        // Update specific dashboard section based on event type
    }
}
```

---

## 📦 Data Models

### Event Structure
```go
type Event struct {
    Type      string                 `json:"type"`       // Event type (alert_created, etc.)
    ID        string                 `json:"id"`         // Unique event ID
    Data      map[string]interface{} `json:"data"`       // Event payload
    Timestamp time.Time              `json:"timestamp"` // Event timestamp
    Source    string                 `json:"source"`     // Event source (alert_processor, etc.)
}
```

### Alert Event Payload
```go
type AlertEventData struct {
    Fingerprint string            `json:"fingerprint"`
    AlertName   string            `json:"alertname"`
    Status      string            `json:"status"`      // firing, resolved
    Severity    string            `json:"severity"`    // critical, warning, info
    Labels      map[string]string `json:"labels"`
    StartsAt    time.Time         `json:"starts_at"`
    EndsAt      *time.Time        `json:"ends_at,omitempty"`
}
```

### Stats Event Payload
```go
type StatsEventData struct {
    FiringAlerts    int `json:"firing_alerts"`
    ResolvedAlerts  int `json:"resolved_today"`
    ActiveSilences  int `json:"active_silences"`
    InhibitedAlerts int `json:"inhibited_alerts"`
}
```

### Health Event Payload
```go
type HealthEventData struct {
    Component string `json:"component"` // PostgreSQL, Redis, LLM, Queue
    Status    string `json:"status"`     // healthy, degraded, unhealthy
    Latency   float64 `json:"latency_ms"`
    Message   string `json:"message,omitempty"`
}
```

---

## 🔗 Dependencies

### Required (Must Have)
- ✅ **TN-76**: Dashboard Template Engine (165.9%, Grade A+ EXCEPTIONAL)
  * Используется для рендеринга dashboard
  * Template functions доступны

- ✅ **TN-77**: Modern Dashboard Page (150%, Grade A+ EXCEPTIONAL)
  * Dashboard структура готова
  * JavaScript hooks для обновлений готовы
  * Auto-refresh foundation существует

- ✅ **TN-136**: Silence UI Components (150%, Grade A+)
  * WebSocketHub уже реализован
  * Можно переиспользовать для dashboard

### Optional (Nice to Have)
- ⚠️ **TN-134**: Silence Manager Service (150%+, Grade A+)
  * Источник событий для silence_* events

- ⚠️ **AlertProcessor**: Alert Processing Pipeline
  * Источник событий для alert_* events

---

## ⚠️ Risks & Mitigations

### Risk 1: Connection Scalability
**Risk**: Большое количество одновременных соединений может перегрузить сервер.

**Mitigation**:
- Rate limiting: 10 connections per IP
- Connection pooling: Reuse connections где возможно
- Horizontal scaling: Multiple instances с shared event bus (Redis pub/sub)
- Monitoring: Alerting на высокое количество соединений

**Probability**: MEDIUM
**Impact**: HIGH
**Severity**: HIGH

---

### Risk 2: Event Ordering
**Risk**: События могут приходить в неправильном порядке при высокой нагрузке.

**Mitigation**:
- Event sequencing: Добавить sequence number к каждому событию
- Client-side ordering: Сортировка по timestamp на клиенте
- Idempotency: Обработка дубликатов безопасна

**Probability**: LOW
**Impact**: MEDIUM
**Severity**: MEDIUM

---

### Risk 3: Browser Compatibility
**Risk**: Некоторые браузеры могут не поддерживать SSE или WebSocket.

**Mitigation**:
- Feature detection: Проверка поддержки перед подключением
- Graceful fallback: Polling при недоступности real-time
- Polyfills: Использование polyfills для старых браузеров (опционально)

**Probability**: LOW
**Impact**: MEDIUM
**Severity**: MEDIUM

---

### Risk 4: Memory Leaks
**Risk**: Неправильное управление соединениями может привести к утечкам памяти.

**Mitigation**:
- Proper cleanup: defer unsubscribe при закрытии соединения
- Connection timeouts: Автоматическое закрытие неактивных соединений
- Memory profiling: Регулярный анализ памяти
- Monitoring: Alerting на рост использования памяти

**Probability**: MEDIUM
**Impact**: HIGH
**Severity**: HIGH

---

## ✅ Acceptance Criteria

### Core Functionality
- [ ] SSE endpoint `/api/v2/events/stream` работает и возвращает события
- [ ] WebSocket endpoint `/ws/dashboard` работает и отправляет события
- [ ] Все event types (alert_*, stats_*, silence_*, health_*) поддерживаются
- [ ] Dashboard автоматически обновляется при получении событий
- [ ] Auto-reconnect работает при разрыве соединения

### Performance
- [ ] Latency <100ms от события до доставки клиенту
- [ ] Throughput >1,000 events/second
- [ ] Поддержка 100+ concurrent connections без деградации

### Reliability
- [ ] Graceful degradation на polling при недоступности real-time
- [ ] Auto-reconnect с exponential backoff
- [ ] Error recovery работает корректно

### Security
- [ ] Origin validation для WebSocket
- [ ] Rate limiting работает
- [ ] CORS настроен корректно

### Observability
- [ ] Prometheus metrics записываются
- [ ] Structured logging работает
- [ ] Health check endpoint доступен

---

## 📚 References

1. **TN-76**: Dashboard Template Engine (165.9%, Grade A+)
2. **TN-77**: Modern Dashboard Page (150%, Grade A+)
3. **TN-136**: Silence UI Components (150%, Grade A+)
4. **MDN**: Server-Sent Events API - https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events
5. **MDN**: WebSocket API - https://developer.mozilla.org/en-US/docs/Web/API/WebSockets_API
6. **RFC 6455**: WebSocket Protocol
7. **HTML5**: Server-Sent Events Specification

---

**Document Version**: 1.0
**Last Updated**: 2025-11-20
**Status**: 📝 DRAFT (Requirements Definition)
