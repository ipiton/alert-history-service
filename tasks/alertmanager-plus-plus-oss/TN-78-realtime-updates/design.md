# TN-78: Real-time Updates (SSE/WebSocket) — Design Document

**Task ID**: TN-78
**Module**: Phase 9: Dashboard & UI
**Priority**: HIGH (P1)
**Target Quality**: 150% (Grade A+ Enterprise)
**Design Version**: 1.0
**Last Updated**: 2025-11-20

---

## 1. Architecture Overview

### 1.1 System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                        Browser (Dashboard)                        │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │  RealtimeClient (JavaScript)                                │  │
│  │    ├─ SSE Connection (preferred)                            │  │
│  │    ├─ WebSocket Connection (fallback)                       │  │
│  │    └─ Polling Fallback (if both unavailable)                │  │
│  │                                                              │  │
│  │  Event Handlers:                                             │  │
│  │    ├─ alert_* → Update Alerts Section                       │  │
│  │    ├─ stats_* → Update Stats Cards                          │  │
│  │    ├─ silence_* → Update Silences Section                  │  │
│  │    └─ health_* → Update Health Panel                        │  │
│  └─────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                          │              │
                    SSE (GET)      WebSocket (WS)
                          │              │
┌─────────────────────────────────────────────────────────────────┐
│              Real-time Event System (TN-78)                      │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                    EventBus (Central Hub)                    │  │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐        │  │
│  │  │ SSE Handler │  │WS Hub (ext)  │  │  Event       │        │  │
│  │  │             │  │              │  │  Publishers │        │  │
│  │  └──────────────┘  └──────────────┘  └──────────────┘        │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                          │                                        │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │  Event Sources:                                               │  │
│  │    - AlertProcessor → alert_created, alert_resolved          │  │
│  │    - SilenceManager → silence_* (reuse TN-136)              │  │
│  │    - StatsCollector → stats_updated (periodic)               │  │
│  │    - HealthMonitor → health_changed                          │  │
│  └─────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Diagram

```
┌──────────────────────────────────────────────────────────────────┐
│                      EventBus (Core)                             │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │  type EventBus interface {                                  │ │
│  │      Subscribe(subscriber EventSubscriber) error           │ │
│  │      Unsubscribe(subscriber EventSubscriber) error         │ │
│  │      Publish(event Event) error                             │ │
│  │      GetActiveSubscribers() int                             │ │
│  │  }                                                          │ │
│  │                                                             │ │
│  │  Implementation:                                            │ │
│  │    - subscribers: map[EventSubscriber]bool (sync.RWMutex) │ │
│  │    - eventChannel: chan Event (buffered, 1000)             │ │
│  │    - broadcastWorker: goroutine                            │ │
│  └────────────────────────────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────┘
         │                    │                    │
         │                    │                    │
    ┌────▼────┐         ┌─────▼─────┐      ┌──────▼──────┐
    │   SSE   │         │ WebSocket │      │   Event      │
    │ Handler │         │   Hub     │      │  Publishers  │
    └─────────┘         └───────────┘      └──────────────┘
```

---

## 2. Data Flow

### 2.1 Event Publishing Flow

```
1. Event Source (AlertProcessor/SilenceManager/etc.)
   │
   ├─ Detect change (alert created, silence updated, etc.)
   │
   ├─ Create Event struct
   │  └─ Type, ID, Data, Timestamp, Source
   │
   ├─ Call EventBus.Publish(event)
   │
   ├─ EventBus adds event to broadcast channel
   │
   └─ Broadcast worker picks up event
      │
      ├─ Iterate over all subscribers
      │
      ├─ For each subscriber:
      │  ├─ SSE: Write "data: {...}\n\n" to HTTP response
      │  └─ WebSocket: WriteJSON(event) to WebSocket connection
      │
      └─ Metrics recorded (events_total, latency_seconds)
```

### 2.2 Client Connection Flow

```
1. Browser loads dashboard.html
   │
   ├─ RealtimeClient.connect() called
   │
   ├─ Feature detection:
   │  ├─ If SSE supported → connectSSE()
   │  ├─ Else if WebSocket supported → connectWebSocket()
   │  └─ Else → fallbackPolling()
   │
   ├─ SSE Connection:
   │  ├─ GET /api/v2/events/stream
   │  ├─ Server sets headers (Content-Type: text/event-stream)
   │  ├─ Server sends keep-alive ping every 30s
   │  └─ Server sends events as they occur
   │
   ├─ WebSocket Connection:
   │  ├─ WS /ws/dashboard
   │  ├─ Server upgrades HTTP to WebSocket
   │  ├─ Server sends ping every 54s
   │  └─ Server sends events as they occur
   │
   └─ Event handling:
      ├─ Parse event JSON
      ├─ Determine event type
      ├─ Update corresponding dashboard section
      └─ Show toast notification (if critical)
```

---

## 3. Component Design

### 3.1 EventBus Interface

```go
// Package realtime provides real-time event broadcasting system.
package realtime

import (
    "context"
    "time"
)

// Event represents a real-time event.
type Event struct {
    Type      string                 `json:"type"`       // alert_created, stats_updated, etc.
    ID        string                 `json:"id"`         // Unique event ID (UUID)
    Data      map[string]interface{} `json:"data"`       // Event payload
    Timestamp time.Time              `json:"timestamp"` // Event timestamp
    Source    string                 `json:"source"`     // Event source
    Sequence  int64                  `json:"sequence"`   // Sequence number for ordering
}

// EventSubscriber represents a subscriber to events.
type EventSubscriber interface {
    // ID returns unique subscriber ID
    ID() string

    // Send sends an event to the subscriber
    Send(event Event) error

    // Close closes the subscriber connection
    Close() error

    // Context returns subscriber context (for cancellation)
    Context() context.Context
}

// EventBus manages event subscriptions and broadcasting.
type EventBus interface {
    // Subscribe adds a subscriber to the event bus
    Subscribe(subscriber EventSubscriber) error

    // Unsubscribe removes a subscriber from the event bus
    Unsubscribe(subscriber EventSubscriber) error

    // Publish broadcasts an event to all subscribers
    Publish(event Event) error

    // GetActiveSubscribers returns the number of active subscribers
    GetActiveSubscribers() int

    // Start starts the event bus (run in goroutine)
    Start(ctx context.Context) error

    // Stop stops the event bus gracefully
    Stop(ctx context.Context) error
}

// DefaultEventBus is the default implementation of EventBus.
type DefaultEventBus struct {
    subscribers map[EventSubscriber]bool
    mu          sync.RWMutex
    eventChan   chan Event
    sequence    int64
    logger      *slog.Logger
    metrics     *RealtimeMetrics
}

// NewEventBus creates a new EventBus.
func NewEventBus(logger *slog.Logger, metrics *RealtimeMetrics) *DefaultEventBus {
    return &DefaultEventBus{
        subscribers: make(map[EventSubscriber]bool),
        eventChan:   make(chan Event, 1000), // Buffered channel
        sequence:    0,
        logger:      logger,
        metrics:     metrics,
    }
}
```

---

### 3.2 SSE Handler

```go
// Package handlers provides HTTP handlers for real-time events.
package handlers

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/realtime"
)

// SSEHandler handles Server-Sent Events connections.
type SSEHandler struct {
    eventBus *realtime.DefaultEventBus
    logger   *slog.Logger
    metrics  *RealtimeMetrics
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(eventBus *realtime.DefaultEventBus, logger *slog.Logger, metrics *RealtimeMetrics) *SSEHandler {
    return &SSEHandler{
        eventBus: eventBus,
        logger:   logger.With("component", "sse_handler"),
        metrics:  metrics,
    }
}

// ServeHTTP handles GET /api/v2/events/stream
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

    // CORS headers (if needed)
    if origin := r.Header.Get("Origin"); origin != "" {
        w.Header().Set("Access-Control-Allow-Origin", origin)
        w.Header().Set("Access-Control-Allow-Credentials", "true")
    }

    // Flush headers immediately
    if flusher, ok := w.(http.Flusher); ok {
        flusher.Flush()
    }

    // Create SSE subscriber
    subscriber := NewSSESubscriber(w, r.Context(), h.logger)
    if err := h.eventBus.Subscribe(subscriber); err != nil {
        h.logger.Error("Failed to subscribe SSE client", "error", err)
        http.Error(w, "Failed to establish connection", http.StatusInternalServerError)
        return
    }
    defer h.eventBus.Unsubscribe(subscriber)

    h.logger.Info("SSE client connected",
        "remote_addr", r.RemoteAddr,
        "subscriber_id", subscriber.ID(),
    )

    // Start keep-alive ticker
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    // Event loop
    for {
        select {
        case <-r.Context().Done():
            h.logger.Debug("SSE client disconnected", "subscriber_id", subscriber.ID())
            return

        case <-ticker.C:
            // Send keep-alive ping
            if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
                h.logger.Warn("Failed to send SSE ping", "error", err)
                return
            }
            if flusher, ok := w.(http.Flusher); ok {
                flusher.Flush()
            }

        case event := <-subscriber.EventChan():
            // Send event in SSE format
            if err := h.sendSSEEvent(w, event); err != nil {
                h.logger.Warn("Failed to send SSE event", "error", err)
                return
            }
            if flusher, ok := w.(http.Flusher); ok {
                flusher.Flush()
            }
        }
    }
}

// sendSSEEvent sends an event in SSE format.
func (h *SSEHandler) sendSSEEvent(w http.ResponseWriter, event realtime.Event) error {
    // Format: "data: {...}\n\n"
    data, err := json.Marshal(event)
    if err != nil {
        return fmt.Errorf("failed to marshal event: %w", err)
    }

    _, err = fmt.Fprintf(w, "data: %s\n\n", data)
    return err
}
```

---

### 3.3 SSE Subscriber

```go
// SSESubscriber implements EventSubscriber for SSE connections.
type SSESubscriber struct {
    id        string
    writer    http.ResponseWriter
    ctx       context.Context
    eventChan chan realtime.Event
    logger    *slog.Logger
}

// NewSSESubscriber creates a new SSE subscriber.
func NewSSESubscriber(w http.ResponseWriter, ctx context.Context, logger *slog.Logger) *SSESubscriber {
    return &SSESubscriber{
        id:        uuid.New().String(),
        writer:    w,
        ctx:       ctx,
        eventChan: make(chan realtime.Event, 10), // Buffered channel
        logger:    logger,
    }
}

// ID returns the subscriber ID.
func (s *SSESubscriber) ID() string {
    return s.id
}

// Send sends an event to the subscriber.
func (s *SSESubscriber) Send(event realtime.Event) error {
    select {
    case s.eventChan <- event:
        return nil
    case <-s.ctx.Done():
        return s.ctx.Err()
    default:
        // Channel full, drop event
        s.logger.Warn("SSE subscriber channel full, dropping event",
            "subscriber_id", s.id,
            "event_type", event.Type,
        )
        return fmt.Errorf("subscriber channel full")
    }
}

// EventChan returns the event channel.
func (s *SSESubscriber) EventChan() <-chan realtime.Event {
    return s.eventChan
}

// Close closes the subscriber.
func (s *SSESubscriber) Close() error {
    close(s.eventChan)
    return nil
}

// Context returns the subscriber context.
func (s *SSESubscriber) Context() context.Context {
    return s.ctx
}
```

---

### 3.4 WebSocket Hub Enhancement

```go
// Enhance existing WebSocketHub to support dashboard events.

// DashboardWebSocketHub extends WebSocketHub for dashboard events.
type DashboardWebSocketHub struct {
    *WebSocketHub // Embed existing hub
    eventBus      *realtime.DefaultEventBus
    logger        *slog.Logger
}

// NewDashboardWebSocketHub creates a new dashboard WebSocket hub.
func NewDashboardWebSocketHub(baseHub *WebSocketHub, eventBus *realtime.DefaultEventBus, logger *slog.Logger) *DashboardWebSocketHub {
    hub := &DashboardWebSocketHub{
        WebSocketHub: baseHub,
        eventBus:     eventBus,
        logger:       logger.With("component", "dashboard_ws_hub"),
    }

    // Subscribe hub to EventBus
    go hub.subscribeToEventBus()

    return hub
}

// subscribeToEventBus subscribes hub to EventBus events.
func (h *DashboardWebSocketHub) subscribeToEventBus() {
    // Create WebSocket subscriber that broadcasts to all WS clients
    subscriber := NewWebSocketBroadcastSubscriber(h.WebSocketHub, h.logger)
    h.eventBus.Subscribe(subscriber)
}

// HandleDashboardWebSocket handles WebSocket upgrade for dashboard.
// GET /ws/dashboard
func (h *DashboardWebSocketHub) HandleDashboardWebSocket(w http.ResponseWriter, r *http.Request) {
    // Use existing HandleWebSocket from WebSocketHub
    h.HandleWebSocket(w, r)
}
```

---

### 3.5 Event Publishers

```go
// EventPublisher publishes events to EventBus.
type EventPublisher struct {
    eventBus *realtime.DefaultEventBus
    logger   *slog.Logger
    metrics  *RealtimeMetrics
}

// NewEventPublisher creates a new event publisher.
func NewEventPublisher(eventBus *realtime.DefaultEventBus, logger *slog.Logger, metrics *RealtimeMetrics) *EventPublisher {
    return &EventPublisher{
        eventBus: eventBus,
        logger:   logger.With("component", "event_publisher"),
        metrics:  metrics,
    }
}

// PublishAlertEvent publishes an alert event.
func (p *EventPublisher) PublishAlertEvent(eventType string, alert *core.Alert) error {
    event := realtime.Event{
        Type:      eventType, // alert_created, alert_resolved, etc.
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Source:    "alert_processor",
        Data: map[string]interface{}{
            "fingerprint": alert.Fingerprint,
            "alertname":  alert.AlertName,
            "status":     alert.Status,
            "severity":   alert.Severity,
            "labels":     alert.Labels,
            "starts_at":  alert.StartsAt.Format(time.RFC3339),
        },
    }

    if alert.EndsAt != nil {
        event.Data["ends_at"] = alert.EndsAt.Format(time.RFC3339)
    }

    return p.eventBus.Publish(event)
}

// PublishStatsEvent publishes a stats update event.
func (p *EventPublisher) PublishStatsEvent(stats *DashboardStats) error {
    event := realtime.Event{
        Type:      "stats_updated",
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Source:    "stats_collector",
        Data: map[string]interface{}{
            "firing_alerts":    stats.FiringAlerts,
            "resolved_alerts":  stats.ResolvedAlerts,
            "active_silences":  stats.ActiveSilences,
            "inhibited_alerts": stats.InhibitedAlerts,
        },
    }

    return p.eventBus.Publish(event)
}

// PublishHealthEvent publishes a health change event.
func (p *EventPublisher) PublishHealthEvent(component string, status string, latency float64, message string) error {
    event := realtime.Event{
        Type:      "health_changed",
        ID:        uuid.New().String(),
        Timestamp: time.Now(),
        Source:    "health_monitor",
        Data: map[string]interface{}{
            "component": component,
            "status":    status,
            "latency_ms": latency,
        },
    }

    if message != "" {
        event.Data["message"] = message
    }

    return p.eventBus.Publish(event)
}
```

---

### 3.6 JavaScript Client

```javascript
// TN-78: Real-time Updates Client (150% Quality Target)

class RealtimeClient {
    constructor(options = {}) {
        this.options = {
            sseEndpoint: options.sseEndpoint || '/api/v2/events/stream',
            wsEndpoint: options.wsEndpoint || '/ws/dashboard',
            pollingInterval: options.pollingInterval || 30000, // 30s fallback
            reconnectDelay: options.reconnectDelay || 1000,
            maxReconnectDelay: options.maxReconnectDelay || 30000,
            ...options
        };

        this.eventBus = new EventTarget();
        this.connection = null;
        this.connectionType = null; // 'sse', 'websocket', 'polling'
        this.reconnectAttempts = 0;
        this.isConnected = false;
    }

    // Connect to real-time stream
    connect() {
        if (this.supportsSSE()) {
            this.connectSSE();
        } else if (this.supportsWebSocket()) {
            this.connectWebSocket();
        } else {
            this.fallbackPolling();
        }
    }

    // Check SSE support
    supportsSSE() {
        return typeof EventSource !== 'undefined';
    }

    // Check WebSocket support
    supportsWebSocket() {
        return typeof WebSocket !== 'undefined';
    }

    // Connect via SSE
    connectSSE() {
        try {
            this.connectionType = 'sse';
            const eventSource = new EventSource(this.options.sseEndpoint);

            eventSource.onopen = () => {
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.onConnect('sse');
            };

            eventSource.onmessage = (e) => {
                try {
                    const event = JSON.parse(e.data);
                    this.handleEvent(event);
                } catch (err) {
                    console.error('[RealtimeClient] Failed to parse SSE event:', err);
                }
            };

            eventSource.onerror = (err) => {
                console.error('[RealtimeClient] SSE error:', err);
                this.isConnected = false;
                eventSource.close();
                this.scheduleReconnect();
            };

            this.connection = eventSource;
        } catch (err) {
            console.error('[RealtimeClient] Failed to connect SSE:', err);
            this.fallbackToWebSocket();
        }
    }

    // Connect via WebSocket
    connectWebSocket() {
        try {
            this.connectionType = 'websocket';
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            const wsUrl = `${protocol}//${window.location.host}${this.options.wsEndpoint}`;
            const ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                this.isConnected = true;
                this.reconnectAttempts = 0;
                this.onConnect('websocket');
            };

            ws.onmessage = (e) => {
                try {
                    const event = JSON.parse(e.data);
                    this.handleEvent(event);
                } catch (err) {
                    console.error('[RealtimeClient] Failed to parse WebSocket event:', err);
                }
            };

            ws.onerror = (err) => {
                console.error('[RealtimeClient] WebSocket error:', err);
                this.isConnected = false;
            };

            ws.onclose = () => {
                this.isConnected = false;
                this.scheduleReconnect();
            };

            this.connection = ws;
        } catch (err) {
            console.error('[RealtimeClient] Failed to connect WebSocket:', err);
            this.fallbackPolling();
        }
    }

    // Fallback to polling
    fallbackPolling() {
        this.connectionType = 'polling';
        console.warn('[RealtimeClient] Real-time not available, using polling fallback');

        // Poll dashboard data every 30s
        this.pollInterval = setInterval(() => {
            this.fetchDashboardData();
        }, this.options.pollingInterval);
    }

    // Handle incoming event
    handleEvent(event) {
        // Emit custom event
        this.eventBus.dispatchEvent(new CustomEvent(event.type, { detail: event }));

        // Update dashboard based on event type
        switch (event.type) {
            case 'alert_created':
            case 'alert_resolved':
            case 'alert_firing':
                this.updateAlertsSection(event);
                break;
            case 'stats_updated':
                this.updateStatsSection(event);
                break;
            case 'silence_created':
            case 'silence_updated':
            case 'silence_deleted':
            case 'silence_expired':
                this.updateSilencesSection(event);
                break;
            case 'health_changed':
                this.updateHealthSection(event);
                break;
        }

        // Show toast for critical events
        if (this.isCriticalEvent(event)) {
            this.showToast(event);
        }
    }

    // Update alerts section
    updateAlertsSection(event) {
        const alertsSection = document.querySelector('.alerts-section');
        if (!alertsSection) return;

        // Add visual indicator
        alertsSection.classList.add('updated');
        setTimeout(() => alertsSection.classList.remove('updated'), 2000);

        // Reload alerts (or update DOM directly)
        this.reloadAlertsSection();
    }

    // Update stats section
    updateStatsSection(event) {
        const stats = event.data;

        // Update stat cards
        this.updateStatCard('firing', stats.firing_alerts);
        this.updateStatCard('resolved', stats.resolved_alerts);
        this.updateStatCard('silences', stats.active_silences);
        this.updateStatCard('inhibited', stats.inhibited_alerts);
    }

    // Update stat card
    updateStatCard(type, value) {
        const card = document.querySelector(`.stat-card[data-type="${type}"]`);
        if (card) {
            const valueEl = card.querySelector('.stat-value');
            if (valueEl) {
                // Animate value change
                const oldValue = parseInt(valueEl.textContent) || 0;
                this.animateValue(valueEl, oldValue, value, 500);
            }
        }
    }

    // Animate value change
    animateValue(element, start, end, duration) {
        const startTime = performance.now();
        const change = end - start;

        const animate = (currentTime) => {
            const elapsed = currentTime - startTime;
            const progress = Math.min(elapsed / duration, 1);
            const current = Math.floor(start + change * progress);

            element.textContent = current;

            if (progress < 1) {
                requestAnimationFrame(animate);
            }
        };

        requestAnimationFrame(animate);
    }

    // Schedule reconnect with exponential backoff
    scheduleReconnect() {
        if (this.reconnectAttempts >= 10) {
            console.error('[RealtimeClient] Max reconnect attempts reached, falling back to polling');
            this.fallbackPolling();
            return;
        }

        this.reconnectAttempts++;
        const delay = Math.min(
            this.options.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1),
            this.options.maxReconnectDelay
        );

        console.log(`[RealtimeClient] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts})`);

        setTimeout(() => {
            this.connect();
        }, delay);
    }

    // Event listener registration
    on(eventType, callback) {
        this.eventBus.addEventListener(eventType, (e) => {
            callback(e.detail);
        });
    }

    // Disconnect
    disconnect() {
        if (this.connection) {
            if (this.connectionType === 'sse') {
                this.connection.close();
            } else if (this.connectionType === 'websocket') {
                this.connection.close();
            } else if (this.connectionType === 'polling') {
                clearInterval(this.pollInterval);
            }
        }
        this.isConnected = false;
    }

    // Check if event is critical
    isCriticalEvent(event) {
        const criticalTypes = ['alert_created', 'alert_firing', 'health_changed'];
        return criticalTypes.includes(event.type);
    }

    // Show toast notification
    showToast(event) {
        // Reuse toast function from dashboard.html
        if (typeof showToast === 'function') {
            const message = this.formatEventMessage(event);
            showToast(message, this.getEventSeverity(event));
        }
    }

    // Format event message
    formatEventMessage(event) {
        switch (event.type) {
            case 'alert_created':
                return `New alert: ${event.data.alertname}`;
            case 'alert_firing':
                return `Alert firing: ${event.data.alertname}`;
            case 'health_changed':
                return `Health changed: ${event.data.component} is ${event.data.status}`;
            default:
                return `Event: ${event.type}`;
        }
    }

    // Get event severity
    getEventSeverity(event) {
        if (event.type.startsWith('alert_')) {
            return event.data.severity === 'critical' ? 'error' : 'warning';
        }
        if (event.type === 'health_changed') {
            return event.data.status === 'unhealthy' ? 'error' : 'info';
        }
        return 'info';
    }

    // Fetch dashboard data (polling fallback)
    async fetchDashboardData() {
        try {
            const response = await fetch('/dashboard');
            if (response.ok) {
                // Parse HTML and update sections
                const html = await response.text();
                this.updateDashboardFromHTML(html);
            }
        } catch (err) {
            console.error('[RealtimeClient] Failed to fetch dashboard:', err);
        }
    }

    // Update dashboard from HTML (polling fallback)
    updateDashboardFromHTML(html) {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');

        // Update stats section
        const newStats = doc.querySelector('.stats-section');
        if (newStats) {
            const oldStats = document.querySelector('.stats-section');
            if (oldStats) {
                oldStats.innerHTML = newStats.innerHTML;
            }
        }

        // Update alerts section
        const newAlerts = doc.querySelector('.alerts-section');
        if (newAlerts) {
            const oldAlerts = document.querySelector('.alerts-section');
            if (oldAlerts) {
                oldAlerts.innerHTML = newAlerts.innerHTML;
            }
        }
    }
}

// Initialize RealtimeClient when dashboard loads
document.addEventListener('DOMContentLoaded', function() {
    window.realtimeClient = new RealtimeClient({
        sseEndpoint: '/api/v2/events/stream',
        wsEndpoint: '/ws/dashboard',
    });

    window.realtimeClient.connect();

    // Listen for specific events
    window.realtimeClient.on('alert_created', (event) => {
        console.log('Alert created:', event);
    });

    window.realtimeClient.on('stats_updated', (event) => {
        console.log('Stats updated:', event);
    });
});
```

---

## 4. Integration Points

### 4.1 AlertProcessor Integration

```go
// In AlertProcessor, after processing alert:
if eventPublisher != nil {
    if alert.Status == "firing" {
        eventPublisher.PublishAlertEvent("alert_firing", alert)
    } else if alert.Status == "resolved" {
        eventPublisher.PublishAlertEvent("alert_resolved", alert)
    }
}
```

### 4.2 SilenceManager Integration

```go
// In SilenceManager, reuse existing WebSocketHub.Broadcast():
// Already implemented in TN-136, just ensure EventBus integration
```

### 4.3 StatsCollector Integration

```go
// Periodic stats update (every 10s):
ticker := time.NewTicker(10 * time.Second)
for {
    select {
    case <-ticker.C:
        stats := collectDashboardStats()
        eventPublisher.PublishStatsEvent(stats)
    }
}
```

### 4.4 HealthMonitor Integration

```go
// In HealthMonitor, when status changes:
if oldStatus != newStatus {
    eventPublisher.PublishHealthEvent(component, newStatus, latency, message)
}
```

---

## 5. Performance Considerations

### 5.1 Connection Management
- **Connection Pooling**: Reuse connections где возможно
- **Connection Limits**: Max 10 connections per IP (rate limiting)
- **Idle Timeout**: Close idle connections after 5 minutes
- **Memory Management**: Proper cleanup при закрытии соединений

### 5.2 Event Broadcasting
- **Buffered Channels**: 1000 event buffer для предотвращения блокировок
- **Concurrent Broadcasting**: Goroutine per subscriber для параллельной отправки
- **Event Batching**: Batch multiple events в один message (опционально)
- **Event Filtering**: Filter events по типу перед отправкой (опционально)

### 5.3 Client-Side Optimization
- **Debouncing**: Debounce rapid updates (max 1 update per 100ms)
- **Throttling**: Throttle DOM updates (max 10 updates per second)
- **Virtual Scrolling**: Для больших списков (future enhancement)

---

## 6. Security Considerations

### 6.1 Origin Validation
```go
// WebSocket origin check
upgrader.CheckOrigin = func(r *http.Request) bool {
    origin := r.Header.Get("Origin")
    allowedOrigins := []string{"https://dashboard.example.com"}
    for _, allowed := range allowedOrigins {
        if origin == allowed {
            return true
        }
    }
    return false
}
```

### 6.2 Rate Limiting
```go
// Rate limiter: 10 connections per IP
rateLimiter := limiter.NewRateLimiter(10, time.Minute)
if !rateLimiter.Allow(r.RemoteAddr) {
    http.Error(w, "Too many connections", http.StatusTooManyRequests)
    return
}
```

### 6.3 Authentication (Optional)
```go
// JWT token validation (optional)
token := r.Header.Get("Authorization")
if token != "" {
    claims, err := validateJWT(token)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }
    // Store user info in context
}
```

---

## 7. Error Handling

### 7.1 Connection Errors
- **Graceful Degradation**: Fallback на polling при ошибках
- **Error Logging**: Structured logging всех ошибок
- **Error Metrics**: Prometheus metrics для ошибок
- **Retry Logic**: Exponential backoff для переподключения

### 7.2 Event Errors
- **Event Validation**: Validate event structure перед отправкой
- **Error Events**: Send error events клиентам при критических ошибках
- **Dead Letter Queue**: Store failed events для анализа (опционально)

---

## 8. Testing Strategy

### 8.1 Unit Tests
- EventBus: Subscribe, Unsubscribe, Publish
- SSE Handler: Connection, event sending, keep-alive
- WebSocket Hub: Connection, broadcasting
- Event Publisher: Event creation, publishing

### 8.2 Integration Tests
- Full SSE connection flow
- Full WebSocket connection flow
- Event broadcasting to multiple clients
- Graceful shutdown

### 8.3 E2E Tests
- Browser automation: Connect SSE/WebSocket
- Verify dashboard updates
- Verify toast notifications
- Verify reconnection

### 8.4 Performance Tests
- Load testing: 100+ concurrent connections
- Latency testing: Event delivery time
- Throughput testing: Events per second
- Memory profiling: Connection overhead

---

## 9. Metrics & Observability

### 9.1 Prometheus Metrics

```go
// RealtimeMetrics tracks real-time system metrics.
type RealtimeMetrics struct {
    ConnectionsActive    prometheus.Gauge
    EventsTotal          *prometheus.CounterVec
    EventLatencySeconds  prometheus.Histogram
    ErrorsTotal          *prometheus.CounterVec
    ReconnectTotal       prometheus.Counter
    BroadcastDuration    prometheus.Histogram
}

// Metrics:
// - realtime_connections_active (Gauge)
// - realtime_events_total (Counter by type, source)
// - realtime_event_latency_seconds (Histogram)
// - realtime_errors_total (Counter by error_type)
// - realtime_reconnect_total (Counter)
// - realtime_broadcast_duration_seconds (Histogram)
```

### 9.2 Structured Logging
- Connection events: connect, disconnect, error
- Event publishing: event type, subscriber count
- Performance: latency, throughput

### 9.3 Health Checks
- `/health/realtime` endpoint
- Check EventBus status
- Check active connections
- Check error rate

---

## 10. Deployment Considerations

### 10.1 Horizontal Scaling
- **Shared Event Bus**: Redis pub/sub для multi-instance deployment
- **Sticky Sessions**: Not required (stateless connections)
- **Load Balancing**: Round-robin или least-connections

### 10.2 Monitoring
- **Connection Count**: Alert on high connection count (>500)
- **Error Rate**: Alert on high error rate (>1%)
- **Latency**: Alert on high latency (>200ms)

### 10.3 Rollback Plan
- **Feature Flag**: Disable real-time updates via config
- **Graceful Degradation**: Automatic fallback на polling
- **Monitoring**: Watch error rates during rollout

---

**Document Version**: 1.0
**Last Updated**: 2025-11-20
**Status**: 📝 DRAFT (Design Definition)
