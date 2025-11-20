# TN-79: Alert List with Filtering — Design Document

**Task ID**: TN-79
**Module**: Phase 9: Dashboard & UI
**Target Quality**: 150% (Grade A+ Enterprise)
**Status**: 🔄 **ANALYSIS IN PROGRESS** (2025-11-20)

---

## 📋 Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Component Design](#2-component-design)
3. [Data Flow](#3-data-flow)
4. [UI/UX Design](#4-uiux-design)
5. [API Integration](#5-api-integration)
6. [Real-time Updates](#6-real-time-updates)
7. [Performance Optimization](#7-performance-optimization)
8. [Security Considerations](#8-security-considerations)
9. [Error Handling](#9-error-handling)
10. [Testing Strategy](#10-testing-strategy)

---

## 1. Architecture Overview

### 1.1 System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Browser (Client)                         │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  Alert List Page (HTML + CSS + JS)                   │  │
│  │  - Template Engine (TN-76) SSR                       │  │
│  │  - Real-time Updates (TN-78 SSE/WebSocket)           │  │
│  │  - Filter UI Components                              │  │
│  │  - Pagination UI                                     │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ HTTP GET /ui/alerts
                            │ SSE/WebSocket /api/v2/events/stream
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Alert History Service (Go)                      │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  AlertListUIHandler (handlers/alert_list_ui.go)    │  │
│  │  - Render alert list page                          │  │
│  │  - Parse filter params                             │  │
│  │  - Fetch data from API                             │  │
│  │  - Render template                                 │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                 │
│                            │ GET /api/v2/history             │
│                            ▼                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  HistoryHandler (TN-63)                             │  │
│  │  - Filter parsing                                   │  │
│  │  - Pagination                                       │  │
│  │  - Sorting                                          │  │
│  │  - Caching                                          │  │
│  └──────────────────────────────────────────────────────┘  │
│                            │                                 │
│                            │ Query Database                 │
│                            ▼                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  PostgresHistoryRepository                          │  │
│  │  - ListAlerts with filters                          │  │
│  │  - Pagination                                       │  │
│  │  - Sorting                                          │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Component Hierarchy

```
AlertListUIHandler
├── Template Engine (TN-76)
│   ├── Base Layout (TN-77)
│   ├── Alert List Page Template
│   └── Partials
│       ├── alert-card.html (reuse)
│       ├── filter-sidebar.html
│       ├── pagination.html
│       └── bulk-actions.html
├── API Client (internal)
│   └── GET /api/v2/history
└── Real-time Client (TN-78)
    └── SSE/WebSocket connection
```

---

## 2. Component Design

### 2.1 AlertListUIHandler

**Location**: `go-app/cmd/server/handlers/alert_list_ui.go`

**Responsibilities**:
- Render alert list page
- Parse filter parameters from URL query
- Fetch alert data from API
- Handle errors and empty states
- Integrate with Template Engine (TN-76)
- Integrate with Real-time Updates (TN-78)

**Interface**:
```go
type AlertListUIHandler struct {
    templateEngine *ui.TemplateEngine  // TN-76
    apiClient      *http.Client         // Internal API client
    wsHub          *WebSocketHub        // TN-78 (optional)
    cache          cache.Cache          // Response caching
    logger         *slog.Logger
}

func (h *AlertListUIHandler) RenderAlertList(w http.ResponseWriter, r *http.Request)
func (h *AlertListUIHandler) parseFilterParams(query url.Values) (*core.AlertFilters, error)
func (h *AlertListUIHandler) fetchAlerts(ctx context.Context, filters *core.AlertFilters) (*core.HistoryResponse, error)
func (h *AlertListUIHandler) renderError(w http.ResponseWriter, r *http.Request, message string, status int)
```

**Design Decisions**:
- ✅ Reuse Template Engine from TN-76 (no duplication)
- ✅ Use internal HTTP client for API calls (avoid circular dependencies)
- ✅ Optional WebSocketHub (graceful degradation if not available)
- ✅ Response caching for performance (reduce API calls)

---

### 2.2 Filter Sidebar Component

**Location**: `go-app/templates/partials/filter-sidebar.html`

**Responsibilities**:
- Display all filter types (15+)
- Handle filter input (dropdowns, text inputs, date pickers)
- Show active filters (chips)
- Filter presets (Last 1h, Last 24h, Critical Only)
- Clear all filters button

**Filter Types**:
1. **Status Filter**: Dropdown (firing, resolved, all)
2. **Severity Filter**: Multi-select (critical, warning, info, noise)
3. **Namespace Filter**: Autocomplete (fetch from API)
4. **Time Range Filter**: Date picker (from/to)
5. **Label Filters**: Dynamic key=value pairs
6. **Alert Name Filter**: Text input (exact match)
7. **Alert Name Pattern**: Text input (LIKE pattern)
8. **Alert Name Regex**: Text input (regex pattern)
9. **Fingerprint Filter**: Text input (exact match)
10. **Search Filter**: Text input (full-text search)
11. **Duration Filter**: Range slider (min/max)
12. **Flapping Filter**: Checkbox (is_flapping)
13. **Resolved Filter**: Checkbox (is_resolved)
14. **Label Exists Filter**: Multi-select (labels that must exist)
15. **Label Not Exists Filter**: Multi-select (labels that must not exist)

**Design Decisions**:
- ✅ Collapsible on mobile (save screen space)
- ✅ Filter state in URL query params (shareable URLs)
- ✅ Progressive enhancement (basic filters first, advanced later)
- ✅ Filter validation (client-side + server-side)

---

### 2.3 Alert List Component

**Location**: `go-app/templates/pages/alert-list.html`

**Responsibilities**:
- Display list of alert cards
- Handle empty/loading/error states
- Support bulk selection
- Display pagination
- Handle sorting

**Alert Card** (reuse from TN-77):
- Alert name (link to details)
- Status badge (firing/resolved)
- Severity badge (critical/warning/info/noise)
- Summary (truncated)
- Labels (collapsible)
- Timestamps (starts_at, ends_at)
- AI Classification badge (if available)
- Quick actions (silence, acknowledge)

**Design Decisions**:
- ✅ Reuse alert-card.html partial (DRY principle)
- ✅ Virtual scrolling for large lists (performance)
- ✅ Lazy load alert details (reduce initial load)
- ✅ Skeleton loaders (better UX)

---

### 2.4 Pagination Component

**Location**: `go-app/templates/partials/pagination.html`

**Responsibilities**:
- Display page numbers
- Previous/Next buttons
- First/Last buttons
- Page size selector
- Total count display

**Design Decisions**:
- ✅ Offset-based pagination (simple, compatible with TN-63)
- ✅ Cursor-based pagination (optional, for large datasets)
- ✅ Page size selector (10, 25, 50, 100)
- ✅ Pagination state in URL (shareable URLs)

---

### 2.5 Real-time Updates Component

**Location**: `go-app/templates/partials/realtime-updates.html` (client-side JS)

**Responsibilities**:
- Connect to SSE/WebSocket (TN-78)
- Handle reconnection
- Update alert list on events
- Highlight new/updated alerts
- Graceful degradation (fallback to polling)

**Event Types** (from TN-78):
- `alert_created` - добавить новый алерт в список
- `alert_resolved` - обновить статус алерта
- `alert_firing` - обновить статус алерта
- `stats_updated` - обновить счетчики

**Design Decisions**:
- ✅ Reuse TN-78 implementation (no duplication)
- ✅ Graceful degradation (fallback to polling if SSE/WebSocket unavailable)
- ✅ Connection status indicator (show connection state)
- ✅ Auto-reconnect with exponential backoff

---

## 3. Data Flow

### 3.1 Page Load Flow

```
1. User navigates to /ui/alerts?status=firing&severity=critical
   │
   ▼
2. AlertListUIHandler.RenderAlertList() called
   │
   ▼
3. Parse filter params from URL query
   │
   ▼
4. Check cache (cache key = filters hash)
   │
   ├─ Cache Hit → Return cached HTML
   │
   └─ Cache Miss → Continue
      │
      ▼
5. Fetch alerts from GET /api/v2/history (TN-63)
   │
   ├─ Success → Continue
   │
   └─ Error → Render error page
      │
      ▼
6. Render template with data
   │
   ▼
7. Return HTML response
   │
   ▼
8. Browser renders page
   │
   ▼
9. Client-side JS connects to SSE/WebSocket (TN-78)
   │
   ▼
10. Real-time updates start
```

### 3.2 Filter Change Flow

```
1. User changes filter (e.g., selects "critical" severity)
   │
   ▼
2. JavaScript updates URL query params
   │
   ▼
3. Browser navigates to new URL (or uses History API)
   │
   ▼
4. AlertListUIHandler.RenderAlertList() called with new filters
   │
   ▼
5. Fetch alerts with new filters
   │
   ▼
6. Update alert list (replace DOM or use virtual scrolling)
   │
   ▼
7. Update pagination (reset to page 1)
```

### 3.3 Real-time Update Flow

```
1. SSE/WebSocket receives event (e.g., alert_created)
   │
   ▼
2. JavaScript event handler processes event
   │
   ▼
3. Check if alert matches current filters
   │
   ├─ Matches → Add/update alert in list
   │
   └─ Doesn't match → Update stats only
      │
      ▼
4. Update alert list DOM (add/update/remove alert card)
   │
   ▼
5. Highlight new/updated alert (fade-in animation)
   │
   ▼
6. Update pagination if needed (if new alert added)
```

---

## 4. UI/UX Design

### 4.1 Layout Structure

```
┌─────────────────────────────────────────────────────────────┐
│  Header (TN-77 base layout)                                 │
│  - Logo, Navigation, User Menu                              │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│  Breadcrumbs: Home → Alerts                                 │
└─────────────────────────────────────────────────────────────┘
┌──────────────┬──────────────────────────────────────────────┐
│              │  Alert List Page                             │
│  Filter      │  ┌────────────────────────────────────────┐ │
│  Sidebar     │  │  Active Filters (chips)                │ │
│  (collapsible│  │  [firing] [critical] [Last 24h] [×]     │ │
│   on mobile) │  └────────────────────────────────────────┘ │
│              │  ┌────────────────────────────────────────┐ │
│  [Status]    │  │  Alert Card 1                          │ │
│  [Severity]  │  │  Alert Card 2                          │ │
│  [Namespace] │  │  Alert Card 3                          │ │
│  [Time Range]│  │  ...                                    │ │
│  [Labels]    │  │  Alert Card 50                          │ │
│  [Advanced]  │  └────────────────────────────────────────┘ │
│              │  ┌────────────────────────────────────────┐ │
│  [Presets]   │  │  Pagination: [<] [1] [2] [3] ... [>] │ │
│  [Clear All] │  │  Showing 1-50 of 1,234                 │ │
│              │  └────────────────────────────────────────┘ │
└──────────────┴──────────────────────────────────────────────┘
```

### 4.2 Responsive Breakpoints

**Mobile (<768px)**:
- Filter sidebar: Collapsed by default (hamburger menu)
- Alert cards: Full width, stacked
- Pagination: Simplified (Previous/Next only)

**Tablet (768px-1024px)**:
- Filter sidebar: Collapsible, 250px width
- Alert cards: 2 columns
- Pagination: Full (page numbers + Previous/Next)

**Desktop (>1024px)**:
- Filter sidebar: Always visible, 300px width
- Alert cards: 3 columns
- Pagination: Full (page numbers + First/Last/Previous/Next)

### 4.3 Color Scheme (reuse TN-77)

- **Primary**: #2563eb (blue)
- **Success**: #10b981 (green)
- **Warning**: #f59e0b (amber)
- **Danger**: #ef4444 (red)
- **Info**: #3b82f6 (blue)
- **Background**: #ffffff (white)
- **Surface**: #f9fafb (gray-50)
- **Border**: #e5e7eb (gray-200)

---

## 5. API Integration

### 5.1 API Endpoint

**Endpoint**: `GET /api/v2/history` (TN-63)

**Query Parameters** (all optional):
```
?page=1
&per_page=50
&status=firing
&severity=critical
&namespace=production
&from=2025-11-20T00:00:00Z
&to=2025-11-20T23:59:59Z
&alert_name=HighCPU
&labels[env]=production
&sort_field=starts_at
&sort_order=desc
```

**Response**:
```json
{
  "alerts": [...],
  "total": 1234,
  "page": 1,
  "per_page": 50,
  "total_pages": 25,
  "has_next": true,
  "has_prev": false
}
```

### 5.2 Error Handling

**400 Bad Request**: Invalid query parameters
- Display error message in filter sidebar
- Highlight invalid filter fields

**401 Unauthorized**: Missing/invalid API key
- Redirect to login page

**403 Forbidden**: Insufficient permissions
- Display error message
- Hide restricted filters

**429 Too Many Requests**: Rate limit exceeded
- Display rate limit message
- Retry after delay

**500 Internal Server Error**: Server error
- Display error message
- Show retry button

---

## 6. Real-time Updates

### 6.1 SSE/WebSocket Integration (TN-78)

**Endpoint**: `GET /api/v2/events/stream` (SSE) or `/ws/dashboard` (WebSocket)

**Event Types**:
- `alert_created` - новый алерт создан
- `alert_resolved` - алерт разрешен
- `alert_firing` - алерт перешел в firing
- `stats_updated` - статистика обновлена

**Event Payload**:
```json
{
  "type": "alert_created",
  "data": {
    "alert": {...},
    "timestamp": "2025-11-20T10:00:00Z"
  }
}
```

### 6.2 Update Strategy

1. **New Alert** (alert_created):
   - Check if matches current filters
   - If matches: Add to top of list (or appropriate position based on sort)
   - Highlight with fade-in animation
   - Update pagination if needed

2. **Updated Alert** (alert_resolved, alert_firing):
   - Find alert in list (by fingerprint)
   - Update alert card
   - Highlight with pulse animation
   - Update stats if needed

3. **Stats Update** (stats_updated):
   - Update stats counters (firing/resolved counts)
   - No DOM manipulation needed

### 6.3 Graceful Degradation

**If SSE/WebSocket unavailable**:
- Fallback to polling (every 30 seconds)
- Show connection status indicator ("Polling mode")
- Allow user to manually refresh

---

## 7. Performance Optimization

### 7.1 Caching Strategy

**Server-side**:
- Cache rendered HTML (5 minutes TTL)
- Cache key = filters hash + page number
- Invalidate on alert updates (via cache tags)

**Client-side**:
- Cache API responses (1 minute TTL)
- Use browser cache for static assets
- Service Worker for offline support (optional)

### 7.2 Rendering Optimization

**Server-side**:
- Template caching (TN-76)
- Lazy load alert details
- Virtual scrolling for large lists (client-side)

**Client-side**:
- Debounce filter inputs (300ms)
- Throttle scroll events (100ms)
- Use requestAnimationFrame for animations

### 7.3 API Optimization

**Reduce API calls**:
- Debounce filter changes (500ms)
- Cache API responses
- Use pagination (don't load all alerts)

**Optimize queries**:
- Use database indexes (TN-63)
- Limit result set (max 1000 per page)
- Use cursor-based pagination for large datasets

---

## 8. Security Considerations

### 8.1 XSS Protection

- ✅ Template auto-escaping (html/template)
- ✅ Input validation (client + server)
- ✅ Output encoding (JSON encoding)

### 8.2 CSRF Protection

- ✅ CSRF tokens in forms
- ✅ SameSite cookies
- ✅ Origin validation

### 8.3 Input Validation

- ✅ Filter parameter validation (TN-63)
- ✅ URL query parameter sanitization
- ✅ SQL injection prevention (parameterized queries)

### 8.4 Rate Limiting

- ✅ Rate limiting middleware (reuse)
- ✅ Per-IP limits (100 req/min)
- ✅ Per-user limits (if authenticated)

---

## 9. Error Handling

### 9.1 Error States

**Empty State**:
- Display "No alerts found" message
- Show filter suggestions
- Provide "Clear filters" button

**Loading State**:
- Display skeleton loaders
- Show loading spinner
- Disable filter inputs

**Error State**:
- Display error message
- Show retry button
- Log error to console (development)

### 9.2 Error Recovery

**Network Errors**:
- Retry with exponential backoff (3 attempts)
- Show error message after retries exhausted
- Allow manual retry

**API Errors**:
- Display error message
- Show error details (development only)
- Allow user to report error

---

## 10. Testing Strategy

### 10.1 Unit Tests

**Handler Tests**:
- Filter parameter parsing
- API client calls
- Template rendering
- Error handling

**Template Tests**:
- Template rendering
- Partial inclusion
- Custom function calls

### 10.2 Integration Tests

**API Integration**:
- Filter combinations
- Pagination
- Sorting
- Error responses

**Real-time Updates**:
- SSE/WebSocket connection
- Event handling
- Reconnection
- Graceful degradation

### 10.3 E2E Tests

**User Flows**:
- Filter alerts
- Paginate results
- Sort alerts
- Real-time updates
- Bulk operations

### 10.4 Performance Tests

**Load Tests**:
- Concurrent users (100+)
- Large result sets (10K+ alerts)
- Filter complexity (15+ filters)

**Lighthouse Tests**:
- Performance score (>90)
- Accessibility score (>90)
- Best practices score (>90)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-20
**Author**: AI Assistant (Enterprise Architecture Team)
**Status**: 🔄 ANALYSIS IN PROGRESS
**Review**: Pending Architecture Board Review
