# TN-136: Silence UI Components - Technical Design

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-136
**Status**: 🟡 IN PROGRESS
**Created**: 2025-11-06
**Target Quality**: 150% (Enterprise-Grade)

---

## 📐 Architecture Overview

TN-136 реализует enterprise-grade UI layer для управления silences с использованием **Go-native подхода**. Архитектура следует принципам Server-Side Rendering (SSR), Progressive Enhancement и Zero-Framework philosophy.

```
┌──────────────────────────────────────────────────────────────────┐
│                     CLIENT TIER (Browser)                         │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  HTML5 UI (Server-Side Rendered)                           │  │
│  │  - Tables, Forms, Modals                                   │  │
│  │  - Vanilla JavaScript (no frameworks)                      │  │
│  │  - WebSocket Client (real-time updates)                    │  │
│  │  - Service Worker (offline support)                        │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           ↕ HTTP/WS                               │
└──────────────────────────────────────────────────────────────────┘
                              ↕
┌──────────────────────────────────────────────────────────────────┐
│                  APPLICATION TIER (Go Server)                     │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  SilenceUIHandler (handlers/silence_ui.go)                 │  │
│  │  - RenderDashboard()     - RenderCreateForm()              │  │
│  │  - RenderEditForm()      - RenderDetailView()              │  │
│  │  - RenderTemplates()     - RenderAnalytics()               │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           ↕                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  WebSocketHub (handlers/silence_ws.go)                     │  │
│  │  - Broadcast()           - Subscribe()                     │  │
│  │  - HandleEvents()        - ManageConnections()             │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           ↕                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Template Engine (templates/silences/*.html)               │  │
│  │  - dashboard.html        - create_form.html                │  │
│  │  - detail_view.html      - analytics.html                  │  │
│  │  (Go html/template with embed.FS)                          │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           ↕                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  SilenceHandler API (handlers/silence.go - TN-135)         │  │
│  │  - POST /api/v2/silences       - GET /api/v2/silences      │  │
│  │  - GET /api/v2/silences/{id}   - PUT /api/v2/silences/{id} │  │
│  │  - DELETE /api/v2/silences/{id}                            │  │
│  │  - POST /api/v2/silences/check                             │  │
│  │  - POST /api/v2/silences/bulk/delete                       │  │
│  └────────────────────────────────────────────────────────────┘  │
│                           ↕                                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  SilenceManager (business/silencing/manager.go - TN-134)   │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
                              ↕
┌──────────────────────────────────────────────────────────────────┐
│                     DATA TIER (Persistence)                       │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  PostgreSQL (silences table)                               │  │
│  │  Redis (session cache, WebSocket pub/sub)                  │  │
│  └────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Server-Side Rendering (SSR)**: Primary rendering на сервере для SEO, performance, accessibility
2. **Progressive Enhancement**: Core functionality работает без JavaScript
3. **Zero-Framework**: Vanilla JavaScript + Go templates, no React/Vue/Angular
4. **Mobile-First**: Responsive design, touch-friendly
5. **Accessibility-First**: WCAG 2.1 AA compliance из коробки
6. **Real-Time**: WebSocket для live updates, fall back to polling
7. **Offline-Capable**: Service Worker кеширует критические assets

---

## 🏗️ Component Design

### 1. SilenceUIHandler

**File**: `go-app/cmd/server/handlers/silence_ui.go`

**Responsibility**: Render HTML pages для Silence Management UI

**Structure**:
```go
package handlers

import (
    "embed"
    "html/template"
    "log/slog"
    "net/http"

    "github.com/vitaliisemenov/alert-history/internal/business/silencing"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
)

//go:embed templates/silences/*.html templates/common/*.html
var templatesFS embed.FS

type SilenceUIHandler struct {
    manager       silencing.SilenceManager  // Business logic
    apiHandler    *SilenceHandler           // API handler (reuse logic)
    templates     *template.Template        // Parsed templates
    wsHub         *WebSocketHub             // WebSocket hub
    cache         cache.Cache               // Response caching
    logger        *slog.Logger
}

// Constructor
func NewSilenceUIHandler(
    manager silencing.SilenceManager,
    apiHandler *SilenceHandler,
    wsHub *WebSocketHub,
    cache cache.Cache,
    logger *slog.Logger,
) (*SilenceUIHandler, error) {
    // Parse templates from embed.FS
    tmpl, err := template.New("").
        Funcs(templateFuncs()).
        ParseFS(templatesFS,
            "templates/silences/*.html",
            "templates/common/*.html",
        )
    if err != nil {
        return nil, fmt.Errorf("failed to parse templates: %w", err)
    }

    return &SilenceUIHandler{
        manager:    manager,
        apiHandler: apiHandler,
        templates:  tmpl,
        wsHub:      wsHub,
        cache:      cache,
        logger:     logger,
    }, nil
}
```

**Template Functions** (custom helpers):
```go
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        "formatTime": func(t time.Time) string {
            return t.Format("2006-01-02 15:04")
        },
        "humanDuration": func(d time.Duration) string {
            if d < time.Minute {
                return fmt.Sprintf("%ds", int(d.Seconds()))
            }
            if d < time.Hour {
                return fmt.Sprintf("%dm", int(d.Minutes()))
            }
            return fmt.Sprintf("%dh", int(d.Hours()))
        },
        "statusBadge": func(status string) string {
            colors := map[string]string{
                "active":  "green",
                "pending": "blue",
                "expired": "gray",
            }
            return fmt.Sprintf(`<span class="badge badge-%s">%s</span>`,
                colors[status], status)
        },
        "truncate": func(s string, length int) string {
            if len(s) <= length {
                return s
            }
            return s[:length] + "..."
        },
        "contains": func(slice []string, item string) bool {
            for _, s := range slice {
                if s == item {
                    return true
                }
            }
            return false
        },
    }
}
```

**HTTP Handlers** (6 routes):
```go
// GET /ui/silences - Dashboard
func (h *SilenceUIHandler) RenderDashboard(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Parse query params (filters, pagination)
    filters := parseFilters(r.URL.Query())

    // Fetch silences from manager (via API handler)
    silences, total, err := h.manager.ListSilences(ctx, filters)
    if err != nil {
        h.logger.Error("Failed to list silences", "error", err)
        h.renderError(w, r, "Failed to load silences", 500)
        return
    }

    // Prepare template data
    data := DashboardData{
        Silences:   silences,
        Total:      total,
        Filters:    filters,
        Page:       filters.Page,
        PageSize:   filters.PageSize,
        TotalPages: (total + filters.PageSize - 1) / filters.PageSize,
    }

    // Render template
    if err := h.templates.ExecuteTemplate(w, "dashboard.html", data); err != nil {
        h.logger.Error("Failed to render template", "error", err)
        h.renderError(w, r, "Failed to render page", 500)
        return
    }
}

// GET /ui/silences/create - Create form
func (h *SilenceUIHandler) RenderCreateForm(w http.ResponseWriter, r *http.Request) {
    data := CreateFormData{
        CSRF:       generateCSRFToken(r),
        Matchers:   []Matcher{}, // Empty, user will add
        TimePresets: []TimePreset{
            {Label: "1 hour", Duration: time.Hour},
            {Label: "4 hours", Duration: 4 * time.Hour},
            {Label: "8 hours", Duration: 8 * time.Hour},
            {Label: "24 hours", Duration: 24 * time.Hour},
        },
    }

    if err := h.templates.ExecuteTemplate(w, "create_form.html", data); err != nil {
        h.logger.Error("Failed to render create form", "error", err)
        h.renderError(w, r, "Failed to render form", 500)
        return
    }
}

// GET /ui/silences/{id} - Detail view
func (h *SilenceUIHandler) RenderDetailView(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Extract ID from URL
    id := extractIDFromPath(r.URL.Path, "/ui/silences/")
    if id == "" {
        h.renderError(w, r, "Invalid silence ID", 400)
        return
    }

    // Fetch silence
    silence, err := h.manager.GetSilence(ctx, id)
    if err != nil {
        h.logger.Error("Failed to get silence", "error", err, "id", id)
        h.renderError(w, r, "Silence not found", 404)
        return
    }

    // Count matched alerts (via IsAlertSilenced check)
    matchedCount := h.countMatchedAlerts(ctx, silence)

    data := DetailViewData{
        Silence:      silence,
        MatchedCount: matchedCount,
        CSRF:         generateCSRFToken(r),
    }

    if err := h.templates.ExecuteTemplate(w, "detail_view.html", data); err != nil {
        h.logger.Error("Failed to render detail view", "error", err)
        h.renderError(w, r, "Failed to render page", 500)
        return
    }
}

// GET /ui/silences/{id}/edit - Edit form
func (h *SilenceUIHandler) RenderEditForm(w http.ResponseWriter, r *http.Request) {
    // Similar to RenderDetailView, but uses edit_form.html template
}

// GET /ui/silences/templates - Templates page
func (h *SilenceUIHandler) RenderTemplates(w http.ResponseWriter, r *http.Request) {
    templates := []SilenceTemplate{
        {
            Name:        "Maintenance Window",
            Description: "Silence all alerts during maintenance",
            Matchers:    []Matcher{{Name: "type", Operator: "=", Value: "maintenance"}},
            Duration:    2 * time.Hour,
        },
        {
            Name:        "On-Call Handoff",
            Description: "Silence on-call pages during handoff",
            Matchers:    []Matcher{{Name: "alertname", Operator: "=", Value: "OnCallPageCritical"}},
            Duration:    time.Hour,
        },
        {
            Name:        "Incident Response",
            Description: "Silence critical alerts during incident",
            Matchers: []Matcher{
                {Name: "severity", Operator: "=", Value: "critical"},
                {Name: "incident", Operator: "=~", Value: "INC-.*"},
            },
            Duration:    4 * time.Hour,
        },
    }

    data := TemplatesData{
        Templates: templates,
        CSRF:      generateCSRFToken(r),
    }

    if err := h.templates.ExecuteTemplate(w, "templates.html", data); err != nil {
        h.logger.Error("Failed to render templates page", "error", err)
        h.renderError(w, r, "Failed to render page", 500)
        return
    }
}

// GET /ui/silences/analytics - Analytics dashboard
func (h *SilenceUIHandler) RenderAnalytics(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Fetch analytics data
    stats, err := h.manager.GetStats(ctx)
    if err != nil {
        h.logger.Error("Failed to get silence stats", "error", err)
        h.renderError(w, r, "Failed to load analytics", 500)
        return
    }

    data := AnalyticsData{
        Stats:       stats,
        TimeRange:   "last_7_days",
        RefreshRate: 5 * time.Minute,
    }

    if err := h.templates.ExecuteTemplate(w, "analytics.html", data); err != nil {
        h.logger.Error("Failed to render analytics page", "error", err)
        h.renderError(w, r, "Failed to render page", 500)
        return
    }
}
```

**Helper Methods**:
```go
func (h *SilenceUIHandler) renderError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
    data := ErrorData{
        Message:    message,
        StatusCode: statusCode,
        RequestID:  extractRequestID(r.Context()),
    }

    w.WriteHeader(statusCode)
    if err := h.templates.ExecuteTemplate(w, "error.html", data); err != nil {
        h.logger.Error("Failed to render error page", "error", err)
        http.Error(w, message, statusCode)
    }
}

func (h *SilenceUIHandler) countMatchedAlerts(ctx context.Context, silence *silencing.Silence) int {
    // Use GetActiveSilences or query active alerts
    // For each alert, call IsAlertSilenced
    // Count matches
    // TODO: Optimize with batch check
    return 0 // Placeholder
}
```

---

### 2. WebSocketHub

**File**: `go-app/cmd/server/handlers/silence_ws.go`

**Responsibility**: Manage WebSocket connections and broadcast silence events

**Structure**:
```go
package handlers

import (
    "context"
    "encoding/json"
    "log/slog"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
        // TODO: Implement proper origin check
        return true // Allow all origins for now
    },
}

type WebSocketHub struct {
    clients    map[*websocket.Conn]bool
    broadcast  chan SilenceEvent
    register   chan *websocket.Conn
    unregister chan *websocket.Conn
    mu         sync.RWMutex
    logger     *slog.Logger
}

type SilenceEvent struct {
    Type      string                 `json:"type"`      // silence_created, silence_updated, etc.
    Data      map[string]interface{} `json:"data"`      // Event payload
    Timestamp time.Time              `json:"timestamp"` // Event timestamp
}

func NewWebSocketHub(logger *slog.Logger) *WebSocketHub {
    return &WebSocketHub{
        clients:    make(map[*websocket.Conn]bool),
        broadcast:  make(chan SilenceEvent, 256),
        register:   make(chan *websocket.Conn),
        unregister: make(chan *websocket.Conn),
        logger:     logger,
    }
}

// Start hub (run in goroutine)
func (h *WebSocketHub) Start(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            h.logger.Info("WebSocket hub stopping")
            return

        case client := <-h.register:
            h.mu.Lock()
            h.clients[client] = true
            h.mu.Unlock()
            h.logger.Debug("WebSocket client registered", "total_clients", len(h.clients))

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client]; ok {
                delete(h.clients, client)
                client.Close()
            }
            h.mu.Unlock()
            h.logger.Debug("WebSocket client unregistered", "total_clients", len(h.clients))

        case event := <-h.broadcast:
            h.mu.RLock()
            for client := range h.clients {
                go h.sendToClient(client, event)
            }
            h.mu.RUnlock()
        }
    }
}

func (h *WebSocketHub) sendToClient(client *websocket.Conn, event SilenceEvent) {
    if err := client.WriteJSON(event); err != nil {
        h.logger.Warn("Failed to send WebSocket message", "error", err)
        h.unregister <- client
    }
}

// Broadcast event to all clients
func (h *WebSocketHub) Broadcast(eventType string, data map[string]interface{}) {
    event := SilenceEvent{
        Type:      eventType,
        Data:      data,
        Timestamp: time.Now(),
    }

    select {
    case h.broadcast <- event:
    default:
        h.logger.Warn("WebSocket broadcast channel full, dropping event")
    }
}

// HTTP handler для WebSocket endpoint
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        h.logger.Error("Failed to upgrade WebSocket connection", "error", err)
        return
    }

    h.register <- conn

    // Keep connection alive with ping/pong
    go h.readPump(conn)
}

func (h *WebSocketHub) readPump(conn *websocket.Conn) {
    defer func() {
        h.unregister <- conn
    }()

    conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    conn.SetPongHandler(func(string) error {
        conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                h.logger.Warn("WebSocket read error", "error", err)
            }
            break
        }
    }
}
```

**Integration with SilenceHandler**:
```go
// In handlers/silence.go (TN-135), add WebSocketHub integration
func (h *SilenceHandler) CreateSilence(w http.ResponseWriter, r *http.Request) {
    // ... existing logic ...

    // After successful creation, broadcast event
    if h.wsHub != nil {
        h.wsHub.Broadcast("silence_created", map[string]interface{}{
            "id":        silence.ID,
            "creator":   silence.CreatedBy,
            "status":    silence.Status,
            "starts_at": silence.StartsAt,
            "ends_at":   silence.EndsAt,
        })
    }

    // ... rest of handler ...
}
```

---

### 3. HTML Templates

**Directory**: `go-app/cmd/server/handlers/templates/`

**Structure**:
```
templates/
├── common/
│   ├── base.html          # Base layout (header, footer, nav)
│   ├── error.html         # Error page template
│   └── components/
│       ├── badge.html     # Status badge component
│       ├── table.html     # Reusable table component
│       └── modal.html     # Modal dialog component
│
└── silences/
    ├── dashboard.html     # Main dashboard
    ├── create_form.html   # Create silence form
    ├── edit_form.html     # Edit silence form
    ├── detail_view.html   # Silence detail view
    ├── templates.html     # Templates page
    └── analytics.html     # Analytics dashboard
```

**base.html** (common layout):
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <meta name="description" content="Alert History - Silence Management">
    <title>{{block "title" .}}Silence Management{{end}} - Alert History</title>

    <!-- Preconnect for performance -->
    <link rel="preconnect" href="/api">
    <link rel="dns-prefetch" href="/api">

    <!-- Inline critical CSS (for performance) -->
    <style>
        /* Critical CSS inlined for First Contentful Paint */
        * { box-sizing: border-box; margin: 0; padding: 0; }
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif; }
        .container { max-width: 1200px; margin: 0 auto; padding: 20px; }
        /* ... more critical CSS ... */
    </style>

    <!-- Load full CSS async -->
    <link rel="stylesheet" href="/static/css/main.css" media="print" onload="this.media='all'">

    <!-- PWA manifest -->
    <link rel="manifest" href="/static/manifest.json">
    <meta name="theme-color" content="#1976d2">

    <!-- Favicon -->
    <link rel="icon" type="image/svg+xml" href="/static/favicon.svg">
</head>
<body>
    <header class="header">
        <div class="container">
            <nav class="navbar">
                <a href="/ui/silences" class="logo">🔕 Silence Management</a>
                <ul class="nav-links">
                    <li><a href="/ui/silences">Dashboard</a></li>
                    <li><a href="/ui/silences/create">Create</a></li>
                    <li><a href="/ui/silences/templates">Templates</a></li>
                    <li><a href="/ui/silences/analytics">Analytics</a></li>
                    <li><a href="/metrics">Metrics</a></li>
                </ul>
                <div class="badge" id="active-silences-badge">0</div>
            </nav>
        </div>
    </header>

    <main class="main-content">
        <div class="container">
            {{block "content" .}}{{end}}
        </div>
    </main>

    <footer class="footer">
        <div class="container">
            <p>&copy; 2025 Alert History Service. All rights reserved.</p>
        </div>
    </footer>

    <!-- Load JavaScript async -->
    <script src="/static/js/main.js" defer></script>

    <!-- WebSocket connection -->
    <script>
        // WebSocket for real-time updates
        const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${wsProtocol}//${window.location.host}/ws/silences`;
        let ws;

        function connectWebSocket() {
            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                console.log('WebSocket connected');
            };

            ws.onmessage = (event) => {
                const data = JSON.parse(event.data);
                handleWebSocketEvent(data);
            };

            ws.onerror = (error) => {
                console.error('WebSocket error:', error);
            };

            ws.onclose = () => {
                console.log('WebSocket disconnected, reconnecting...');
                setTimeout(connectWebSocket, 5000); // Reconnect after 5s
            };
        }

        function handleWebSocketEvent(event) {
            switch(event.type) {
                case 'silence_created':
                    showToast(`Silence created by ${event.data.creator}`);
                    refreshDashboard();
                    updateBadge();
                    break;
                case 'silence_updated':
                    showToast(`Silence ${event.data.id} updated`);
                    refreshDashboard();
                    break;
                case 'silence_deleted':
                    showToast(`Silence deleted`);
                    refreshDashboard();
                    updateBadge();
                    break;
                case 'silence_expired':
                    showToast(`Silence expired`);
                    refreshDashboard();
                    updateBadge();
                    break;
            }
        }

        // Connect on page load
        document.addEventListener('DOMContentLoaded', () => {
            connectWebSocket();
            updateBadge(); // Initial badge update
        });

        // Update active silences badge
        async function updateBadge() {
            try {
                const resp = await fetch('/api/v2/silences?status=active&limit=0');
                const data = await resp.json();
                document.getElementById('active-silences-badge').textContent = data.total;
            } catch (error) {
                console.error('Failed to update badge:', error);
            }
        }
    </script>
</body>
</html>
```

**dashboard.html**:
```html
{{define "title"}}Silence Dashboard{{end}}

{{define "content"}}
<div class="dashboard">
    <header class="dashboard-header">
        <h1>Silences</h1>
        <a href="/ui/silences/create" class="btn btn-primary">Create Silence</a>
    </header>

    <!-- Filters -->
    <div class="filters" id="filters-panel">
        <form method="GET" action="/ui/silences" class="filter-form">
            <div class="filter-group">
                <label for="status">Status</label>
                <select name="status" id="status">
                    <option value="all" {{if eq .Filters.Status "all"}}selected{{end}}>All</option>
                    <option value="pending" {{if eq .Filters.Status "pending"}}selected{{end}}>Pending</option>
                    <option value="active" {{if eq .Filters.Status "active"}}selected{{end}}>Active</option>
                    <option value="expired" {{if eq .Filters.Status "expired"}}selected{{end}}>Expired</option>
                </select>
            </div>

            <div class="filter-group">
                <label for="creator">Creator</label>
                <input type="email" name="creator" id="creator"
                       value="{{.Filters.Creator}}"
                       placeholder="ops@example.com">
            </div>

            <div class="filter-group">
                <label for="starts_after">Start After</label>
                <input type="datetime-local" name="starts_after" id="starts_after"
                       value="{{.Filters.StartsAfter}}">
            </div>

            <button type="submit" class="btn btn-secondary">Apply Filters</button>
            <a href="/ui/silences" class="btn btn-link">Clear</a>
        </form>
    </div>

    <!-- Bulk Operations Toolbar (hidden by default) -->
    <div class="bulk-toolbar" id="bulk-toolbar" style="display: none;">
        <span id="bulk-selected-count">0 selected</span>
        <button type="button" class="btn btn-danger" id="bulk-delete-btn">Bulk Delete</button>
        <button type="button" class="btn btn-link" id="bulk-cancel-btn">Cancel</button>
    </div>

    <!-- Silences Table -->
    <div class="table-container">
        {{if gt .Total 0}}
        <table class="table table-hover">
            <thead>
                <tr>
                    <th><input type="checkbox" id="select-all"></th>
                    <th>Status</th>
                    <th>Creator</th>
                    <th>Comment</th>
                    <th>Time Range</th>
                    <th>Actions</th>
                </tr>
            </thead>
            <tbody>
                {{range .Silences}}
                <tr data-silence-id="{{.ID}}">
                    <td><input type="checkbox" class="row-checkbox" value="{{.ID}}"></td>
                    <td>{{statusBadge .Status}}</td>
                    <td>{{.CreatedBy}}</td>
                    <td>{{truncate .Comment 50}}</td>
                    <td>
                        <div class="time-range">
                            <span>{{formatTime .StartsAt}}</span>
                            <span>→</span>
                            <span>{{formatTime .EndsAt}}</span>
                        </div>
                    </td>
                    <td>
                        <div class="action-buttons">
                            <a href="/ui/silences/{{.ID}}" class="btn-icon" title="View Details">
                                👁️
                            </a>
                            <a href="/ui/silences/{{.ID}}/edit" class="btn-icon" title="Edit">
                                ✏️
                            </a>
                            <button type="button" class="btn-icon btn-delete"
                                    data-silence-id="{{.ID}}" title="Delete">
                                🗑️
                            </button>
                        </div>
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <!-- Pagination -->
        <nav class="pagination">
            <div class="pagination-info">
                Showing {{add (mul .Page .PageSize) 1}} to {{min (mul (add .Page 1) .PageSize) .Total}} of {{.Total}} silences
            </div>
            <div class="pagination-controls">
                {{if gt .Page 0}}
                <a href="?page={{sub .Page 1}}&page_size={{.PageSize}}" class="btn btn-secondary">← Previous</a>
                {{end}}

                <span>Page {{add .Page 1}} of {{.TotalPages}}</span>

                {{if lt (add .Page 1) .TotalPages}}
                <a href="?page={{add .Page 1}}&page_size={{.PageSize}}" class="btn btn-secondary">Next →</a>
                {{end}}
            </div>
        </nav>
        {{else}}
        <div class="empty-state">
            <p>No silences found</p>
            <a href="/ui/silences/create" class="btn btn-primary">Create your first silence</a>
        </div>
        {{end}}
    </div>
</div>

<!-- Delete Confirmation Modal -->
<div class="modal" id="delete-modal" style="display: none;">
    <div class="modal-content">
        <h2>Confirm Delete</h2>
        <p>Are you sure you want to delete this silence?</p>
        <p><strong>This action cannot be undone.</strong></p>
        <div class="modal-actions">
            <button type="button" class="btn btn-secondary" id="delete-cancel">Cancel</button>
            <button type="button" class="btn btn-danger" id="delete-confirm">Delete</button>
        </div>
    </div>
</div>

<script>
    // Bulk selection
    document.getElementById('select-all').addEventListener('change', (e) => {
        const checkboxes = document.querySelectorAll('.row-checkbox');
        checkboxes.forEach(cb => cb.checked = e.target.checked);
        updateBulkToolbar();
    });

    document.querySelectorAll('.row-checkbox').forEach(cb => {
        cb.addEventListener('change', updateBulkToolbar);
    });

    function updateBulkToolbar() {
        const selected = document.querySelectorAll('.row-checkbox:checked');
        const toolbar = document.getElementById('bulk-toolbar');
        const count = document.getElementById('bulk-selected-count');

        if (selected.length > 0) {
            toolbar.style.display = 'flex';
            count.textContent = `${selected.length} selected`;
        } else {
            toolbar.style.display = 'none';
        }
    }

    // Delete single silence
    let deleteTarget = null;
    document.querySelectorAll('.btn-delete').forEach(btn => {
        btn.addEventListener('click', (e) => {
            deleteTarget = e.target.dataset.silenceId;
            document.getElementById('delete-modal').style.display = 'flex';
        });
    });

    document.getElementById('delete-cancel').addEventListener('click', () => {
        document.getElementById('delete-modal').style.display = 'none';
        deleteTarget = null;
    });

    document.getElementById('delete-confirm').addEventListener('click', async () => {
        if (!deleteTarget) return;

        try {
            const resp = await fetch(`/api/v2/silences/${deleteTarget}`, {
                method: 'DELETE',
            });

            if (resp.ok) {
                showToast('Silence deleted successfully');
                window.location.reload();
            } else {
                const error = await resp.text();
                showToast(`Failed to delete: ${error}`, 'error');
            }
        } catch (error) {
            showToast(`Error: ${error.message}`, 'error');
        }

        document.getElementById('delete-modal').style.display = 'none';
        deleteTarget = null;
    });

    // Bulk delete
    document.getElementById('bulk-delete-btn').addEventListener('click', async () => {
        const selected = Array.from(document.querySelectorAll('.row-checkbox:checked'))
            .map(cb => cb.value);

        if (selected.length === 0) return;

        if (!confirm(`Are you sure you want to delete ${selected.length} silences?`)) {
            return;
        }

        try {
            const resp = await fetch('/api/v2/silences/bulk/delete', {
                method: 'POST',
                headers: {'Content-Type': 'application/json'},
                body: JSON.stringify({ids: selected}),
            });

            const result = await resp.json();

            if (result.deleted > 0) {
                showToast(`${result.deleted} silences deleted`);
                window.location.reload();
            }

            if (result.errors && result.errors.length > 0) {
                showToast(`${result.errors.length} errors occurred`, 'error');
            }
        } catch (error) {
            showToast(`Error: ${error.message}`, 'error');
        }
    });

    document.getElementById('bulk-cancel-btn').addEventListener('click', () => {
        document.querySelectorAll('.row-checkbox').forEach(cb => cb.checked = false);
        document.getElementById('select-all').checked = false;
        updateBulkToolbar();
    });

    // Auto-refresh every 30s
    setInterval(() => {
        fetch(window.location.href)
            .then(resp => resp.text())
            .then(html => {
                // Parse and update table only (avoid full page reload)
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');
                const newTableBody = doc.querySelector('tbody');
                const currentTableBody = document.querySelector('tbody');
                if (newTableBody && currentTableBody) {
                    currentTableBody.innerHTML = newTableBody.innerHTML;
                }
            })
            .catch(err => console.error('Auto-refresh failed:', err));
    }, 30000);
</script>
{{end}}
```

**(Continued in next message due to length...)**

---

## 📊 Data Models

### Template Data Structures

```go
// Dashboard page data
type DashboardData struct {
    Silences   []silencing.Silence
    Total      int
    Filters    FilterParams
    Page       int
    PageSize   int
    TotalPages int
}

// Create form data
type CreateFormData struct {
    CSRF        string
    Matchers    []Matcher
    TimePresets []TimePreset
    Error       string // Validation errors
}

// Detail view data
type DetailViewData struct {
    Silence      *silencing.Silence
    MatchedCount int
    CSRF         string
}

// Analytics data
type AnalyticsData struct {
    Stats       *silencing.Stats
    TimeRange   string
    RefreshRate time.Duration
}

// Error page data
type ErrorData struct {
    Message    string
    StatusCode int
    RequestID  string
}
```

---

## 🔌 Integration Points

### 1. Main.go Registration

```go
// In go-app/cmd/server/main.go

// Initialize WebSocket hub
wsHub := handlers.NewWebSocketHub(appLogger)
go wsHub.Start(context.Background())

// Initialize Silence UI Handler
silenceUIHandler, err := handlers.NewSilenceUIHandler(
    silenceManager,
    silenceAPIHandler, // TN-135 handler
    wsHub,
    cacheClient,
    appLogger,
)
if err != nil {
    slog.Error("Failed to create Silence UI Handler", "error", err)
    os.Exit(1)
}

// Register UI routes
mux.HandleFunc("/ui/silences", silenceUIHandler.RenderDashboard)
mux.HandleFunc("/ui/silences/create", silenceUIHandler.RenderCreateForm)
mux.HandleFunc("/ui/silences/templates", silenceUIHandler.RenderTemplates)
mux.HandleFunc("/ui/silences/analytics", silenceUIHandler.RenderAnalytics)
// Dynamic routes (ID-based)
mux.HandleFunc("/ui/silences/", silenceUIHandler.HandleDynamicRoutes)

// Register WebSocket route
mux.HandleFunc("/ws/silences", wsHub.HandleWebSocket)

// Static assets
mux.Handle("/static/", http.StripPrefix("/static/",
    http.FileServer(http.FS(staticFS))))

slog.Info("✅ Silence UI initialized successfully",
    "routes", []string{
        "GET /ui/silences - Dashboard",
        "GET /ui/silences/create - Create form",
        "GET /ui/silences/{id} - Detail view",
        "GET /ui/silences/{id}/edit - Edit form",
        "GET /ui/silences/templates - Templates",
        "GET /ui/silences/analytics - Analytics",
        "WS /ws/silences - Real-time updates",
    })
```

### 2. Metrics Integration

```go
// Add UI-specific metrics to pkg/metrics/business.go

// UI page render duration
PageRenderDurationSeconds *prometheus.HistogramVec = prometheus.NewHistogramVec(
    prometheus.HistogramOpts{
        Namespace: "alert_history",
        Subsystem: "ui",
        Name:      "page_render_duration_seconds",
        Help:      "Duration of UI page rendering",
        Buckets:   []float64{.001, .005, .01, .05, .1, .5, 1, 2, 5},
    },
    []string{"page"},
)

// WebSocket connections
WebSocketConnectionsGauge prometheus.Gauge = prometheus.NewGauge(
    prometheus.GaugeOpts{
        Namespace: "alert_history",
        Subsystem: "ui",
        Name:      "websocket_connections",
        Help:      "Current number of WebSocket connections",
    },
)

// WebSocket messages
WebSocketMessagesTotalCounter *prometheus.CounterVec = prometheus.NewCounterVec(
    prometheus.CounterOpts{
        Namespace: "alert_history",
        Subsystem: "ui",
        Name:      "websocket_messages_total",
        Help:      "Total number of WebSocket messages sent",
    },
    []string{"event_type"},
)
```

---

## 🎨 Frontend Assets

### CSS Architecture (BEM Methodology)

**File**: `go-app/cmd/server/handlers/static/css/main.css`

```css
/* CSS Variables (Design Tokens) */
:root {
    /* Colors */
    --color-primary: #1976d2;
    --color-primary-dark: #1565c0;
    --color-secondary: #424242;
    --color-success: #4caf50;
    --color-danger: #f44336;
    --color-warning: #ff9800;
    --color-info: #2196f3;
    --color-bg: #ffffff;
    --color-bg-secondary: #f5f5f5;
    --color-text: #212121;
    --color-text-secondary: #757575;
    --color-border: #e0e0e0;

    /* Spacing */
    --spacing-xs: 4px;
    --spacing-sm: 8px;
    --spacing-md: 16px;
    --spacing-lg: 24px;
    --spacing-xl: 32px;

    /* Typography */
    --font-family-base: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
    --font-size-base: 16px;
    --font-size-sm: 14px;
    --font-size-lg: 18px;
    --font-size-xl: 24px;

    /* Shadows */
    --shadow-sm: 0 1px 3px rgba(0,0,0,0.12);
    --shadow-md: 0 3px 6px rgba(0,0,0,0.16);
    --shadow-lg: 0 10px 20px rgba(0,0,0,0.19);

    /* Transitions */
    --transition-fast: 150ms ease-in-out;
    --transition-base: 300ms ease-in-out;
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
    :root {
        --color-bg: #121212;
        --color-bg-secondary: #1e1e1e;
        --color-text: #e0e0e0;
        --color-text-secondary: #9e9e9e;
        --color-border: #424242;
    }
}

/* Base styles */
body {
    font-family: var(--font-family-base);
    font-size: var(--font-size-base);
    color: var(--color-text);
    background-color: var(--color-bg);
    line-height: 1.5;
}

/* Header */
.header {
    background-color: var(--color-primary);
    color: white;
    box-shadow: var(--shadow-md);
}

.navbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--spacing-md) 0;
}

.logo {
    font-size: var(--font-size-xl);
    font-weight: bold;
    color: white;
    text-decoration: none;
}

.nav-links {
    display: flex;
    gap: var(--spacing-lg);
    list-style: none;
}

.nav-links a {
    color: white;
    text-decoration: none;
    transition: opacity var(--transition-fast);
}

.nav-links a:hover {
    opacity: 0.8;
}

/* Badge */
.badge {
    display: inline-block;
    padding: var(--spacing-xs) var(--spacing-sm);
    border-radius: 12px;
    font-size: var(--font-size-sm);
    font-weight: bold;
}

.badge-green { background-color: var(--color-success); color: white; }
.badge-blue { background-color: var(--color-info); color: white; }
.badge-gray { background-color: var(--color-text-secondary); color: white; }

/* Table */
.table-container {
    background-color: white;
    border-radius: 8px;
    box-shadow: var(--shadow-sm);
    overflow: hidden;
}

.table {
    width: 100%;
    border-collapse: collapse;
}

.table th,
.table td {
    padding: var(--spacing-md);
    text-align: left;
    border-bottom: 1px solid var(--color-border);
}

.table thead th {
    background-color: var(--color-bg-secondary);
    font-weight: 600;
}

.table tbody tr:hover {
    background-color: var(--color-bg-secondary);
}

/* Buttons */
.btn {
    display: inline-block;
    padding: var(--spacing-sm) var(--spacing-md);
    border: none;
    border-radius: 4px;
    font-size: var(--font-size-base);
    font-weight: 500;
    text-align: center;
    text-decoration: none;
    cursor: pointer;
    transition: all var(--transition-fast);
}

.btn-primary {
    background-color: var(--color-primary);
    color: white;
}

.btn-primary:hover {
    background-color: var(--color-primary-dark);
}

.btn-danger {
    background-color: var(--color-danger);
    color: white;
}

.btn-danger:hover {
    opacity: 0.9;
}

/* Responsive Design */
@media (max-width: 768px) {
    .navbar {
        flex-direction: column;
        gap: var(--spacing-md);
    }

    .nav-links {
        flex-direction: column;
        gap: var(--spacing-sm);
    }

    .table {
        font-size: var(--font-size-sm);
    }
}

/* Accessibility */
.sr-only {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border-width: 0;
}

/* Focus visible (keyboard navigation) */
:focus-visible {
    outline: 2px solid var(--color-primary);
    outline-offset: 2px;
}

/* High contrast mode */
@media (prefers-contrast: high) {
    .btn {
        border: 2px solid currentColor;
    }
}
```

### JavaScript (Vanilla)

**File**: `go-app/cmd/server/handlers/static/js/main.js`

```javascript
// Toast notifications
function showToast(message, type = 'success') {
    const toast = document.createElement('div');
    toast.className = `toast toast-${type}`;
    toast.textContent = message;
    toast.setAttribute('role', 'alert');
    toast.setAttribute('aria-live', 'polite');

    document.body.appendChild(toast);

    // Fade in
    setTimeout(() => toast.classList.add('show'), 10);

    // Remove after 5s
    setTimeout(() => {
        toast.classList.remove('show');
        setTimeout(() => document.body.removeChild(toast), 300);
    }, 5000);
}

// Dashboard auto-refresh
function refreshDashboard() {
    if (window.location.pathname === '/ui/silences') {
        window.location.reload();
    }
}

// Form validation
function validateSilenceForm(form) {
    const creator = form.querySelector('[name="creator"]').value;
    const comment = form.querySelector('[name="comment"]').value;
    const startsAt = new Date(form.querySelector('[name="starts_at"]').value);
    const endsAt = new Date(form.querySelector('[name="ends_at"]').value);

    const errors = [];

    if (!creator || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(creator)) {
        errors.push('Creator must be a valid email');
    }

    if (!comment || comment.length < 3) {
        errors.push('Comment must be at least 3 characters');
    }

    if (startsAt >= endsAt) {
        errors.push('End time must be after start time');
    }

    const matchers = form.querySelectorAll('.matcher-row');
    if (matchers.length === 0) {
        errors.push('At least one matcher is required');
    }

    return errors;
}

// Service Worker registration (PWA support)
if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
        navigator.serviceWorker.register('/sw.js')
            .then(reg => console.log('Service Worker registered'))
            .catch(err => console.log('Service Worker registration failed', err));
    });
}

// Accessibility: Skip to content
document.addEventListener('DOMContentLoaded', () => {
    const skipLink = document.createElement('a');
    skipLink.href = '#main-content';
    skipLink.textContent = 'Skip to main content';
    skipLink.className = 'skip-link sr-only';
    skipLink.addEventListener('focus', () => skipLink.classList.remove('sr-only'));
    skipLink.addEventListener('blur', () => skipLink.classList.add('sr-only'));
    document.body.insertBefore(skipLink, document.body.firstChild);
});
```

---

## 🧪 Testing Strategy

### 1. Unit Tests (Go)

**File**: `go-app/cmd/server/handlers/silence_ui_test.go`

```go
package handlers_test

import (
    "bytes"
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/vitaliisemenov/alert-history/cmd/server/handlers"
    // ... other imports
)

func TestSilenceUIHandler_RenderDashboard(t *testing.T) {
    tests := []struct {
        name           string
        queryParams    string
        expectedStatus int
        checkContent   func(*testing.T, string)
    }{
        {
            name:           "empty dashboard",
            queryParams:    "",
            expectedStatus: http.StatusOK,
            checkContent: func(t *testing.T, html string) {
                assert.Contains(t, html, "No silences found")
            },
        },
        {
            name:           "with status filter",
            queryParams:    "?status=active",
            expectedStatus: http.StatusOK,
            checkContent: func(t *testing.T, html string) {
                assert.Contains(t, html, "Status")
                assert.Contains(t, html, "active")
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup handler with mocks
            handler := setupUIHandler(t)

            req := httptest.NewRequest("GET", "/ui/silences"+tt.queryParams, nil)
            w := httptest.NewRecorder()

            handler.RenderDashboard(w, req)

            assert.Equal(t, tt.expectedStatus, w.Code)
            tt.checkContent(t, w.Body.String())
        })
    }
}

func TestWebSocketHub(t *testing.T) {
    t.Run("broadcast to all clients", func(t *testing.T) {
        hub := handlers.NewWebSocketHub(logger)

        // Start hub
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        go hub.Start(ctx)

        // Register clients (mock)
        client1 := &mockWebSocketConn{}
        client2 := &mockWebSocketConn{}
        hub.Register() <- client1
        hub.Register() <- client2

        // Broadcast event
        hub.Broadcast("silence_created", map[string]interface{}{
            "id": "test-id",
        })

        // Wait for messages
        time.Sleep(100 * time.Millisecond)

        assert.Equal(t, 1, client1.messageCount)
        assert.Equal(t, 1, client2.messageCount)
    })
}
```

### 2. Integration Tests

Test full user flows with real database.

### 3. E2E Tests (Playwright)

**File**: `go-app/e2e/silence_ui_test.spec.js`

```javascript
const { test, expect } = require('@playwright/test');

test.describe('Silence UI', () => {
    test('create silence flow', async ({ page }) => {
        // Navigate to create form
        await page.goto('http://localhost:8080/ui/silences/create');

        // Fill form
        await page.fill('[name="creator"]', 'ops@example.com');
        await page.fill('[name="comment"]', 'Test maintenance window');
        await page.fill('[name="starts_at"]', '2025-11-06T12:00');
        await page.fill('[name="ends_at"]', '2025-11-06T14:00');

        // Add matcher
        await page.click('button:has-text("Add Matcher")');
        await page.fill('.matcher-row:last-child [name="name"]', 'alertname');
        await page.fill('.matcher-row:last-child [name="value"]', 'HighCPU');

        // Submit
        await page.click('button[type="submit"]');

        // Verify redirect to dashboard
        await expect(page).toHaveURL(/\/ui\/silences$/);

        // Verify success message
        await expect(page.locator('.toast')).toContainText('Silence created');
    });

    test('bulk delete', async ({ page }) => {
        await page.goto('http://localhost:8080/ui/silences');

        // Select multiple silences
        await page.click('.row-checkbox:nth-child(1)');
        await page.click('.row-checkbox:nth-child(2)');

        // Click bulk delete
        await page.click('#bulk-delete-btn');

        // Confirm
        page.on('dialog', dialog => dialog.accept());

        // Verify success
        await expect(page.locator('.toast')).toContainText('deleted');
    });
});
```

### 4. Accessibility Tests

```javascript
const { test, expect } = require('@playwright/test');
const AxeBuilder = require('@axe-core/playwright').default;

test('accessibility - dashboard', async ({ page }) => {
    await page.goto('http://localhost:8080/ui/silences');

    const results = await new AxeBuilder({ page }).analyze();

    expect(results.violations).toHaveLength(0);
});
```

---

## 📚 Documentation Deliverables

1. **UI Usage Guide** (`SILENCE_UI_GUIDE.md`) - 1,500+ lines
2. **Template Development Guide** (`TEMPLATE_DEVELOPMENT.md`) - 800+ lines
3. **Accessibility Guide** (`ACCESSIBILITY_COMPLIANCE.md`) - 500+ lines
4. **Integration Examples** (`UI_INTEGRATION_EXAMPLES.md`) - 400+ lines
5. **Deployment Guide** (`UI_DEPLOYMENT_GUIDE.md`) - 300+ lines

---

**Document Version**: 1.0
**Created**: 2025-11-06
**Author**: Kilo Code AI
**Status**: APPROVED FOR IMPLEMENTATION
