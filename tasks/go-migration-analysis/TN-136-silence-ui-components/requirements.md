# TN-136: Silence UI Components - Requirements

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-136
**Status**: 🟡 IN PROGRESS
**Priority**: HIGH
**Estimated Effort**: 14-18 hours
**Target Quality**: 150% (Enterprise-Grade)
**Dependencies**: TN-131 ✅, TN-132 ✅, TN-133 ✅, TN-134 ✅, TN-135 ✅

---

## 📋 Overview

Реализовать комплексный UI layer для управления silences, включающий dashboard widget, формы создания/редактирования, bulk operations interface и real-time monitoring. Решение должно быть **Go-native** с использованием `html/template`, обеспечивая полную миграцию с Python-based UI и предоставляя enterprise-grade пользовательский опыт.

### Business Value

- **Self-Service Operations**: Операторы могут создавать/управлять silences без знания API
- **Bulk Operations**: Массовые операции для эффективного управления большим количеством silences
- **Real-Time Visibility**: Live updates через WebSocket для мгновенной обратной связи
- **User Experience**: Intuitive UI с modern design patterns (responsive, accessible)
- **Migration Path**: Полная замена Python dashboard, продвижение Go-first архитектуры
- **Cost Reduction**: Единая кодовая база (Go), упрощение deployment pipeline

---

## 🎯 Goals

### Primary Goals (100% Must-Have)

1. ✅ **Silence Dashboard Widget**
   - Display active/pending/expired silences в табличном виде
   - Filtering по status, creator, time range
   - Sorting по created_at, starts_at, ends_at
   - Pagination для больших datasets

2. ✅ **Create/Edit Silence Forms**
   - Форма создания silence (matchers, time range, comment)
   - Форма редактирования существующего silence
   - Client-side validation (JavaScript)
   - Server-side validation (Go)

3. ✅ **Bulk Operations Interface**
   - Multi-select для silences
   - Bulk delete с confirmation dialog
   - Progress indicator для long-running operations

4. ✅ **Silence Detail View**
   - Full silence details (matchers, creator, timestamps)
   - Matched alerts count (real-time)
   - Quick actions (edit, delete, extend)

5. ✅ **Integration с TN-135 API**
   - Использует все 7 REST endpoints
   - Error handling с user-friendly messages
   - Loading states для async operations

### Secondary Goals (150% Quality)

6. ✅ **Real-Time Updates (WebSocket)**
   - Live updates при создании/удалении silences
   - Active silences count badge
   - Notification toasts для events

7. ✅ **Advanced Filtering UI**
   - Multi-filter panel (status AND creator AND matcher)
   - Date range picker для time filters
   - Saved filter presets

8. ✅ **Silence Templates**
   - Pre-defined silence templates (maintenance, oncall, incident)
   - Template editor для custom templates
   - Quick-create from template

9. ✅ **Analytics Dashboard**
   - Silences statistics (по status, по creator)
   - Timeline chart (silences over time)
   - Top silenced alerts

10. ✅ **Mobile-Responsive Design**
    - Adaptive layout для mobile/tablet/desktop
    - Touch-friendly controls
    - Progressive Web App (PWA) manifest

11. ✅ **Accessibility (WCAG 2.1 AA)**
    - Keyboard navigation
    - Screen reader support (ARIA labels)
    - High contrast mode

12. ✅ **Go-Native Implementation**
    - `html/template` для SSR (Server-Side Rendering)
    - Zero external frontend frameworks (Vue/React)
    - Minimal JavaScript (vanilla JS)
    - Embedded assets (embed.FS)

---

## 📐 Functional Requirements

### FR-1: Silence Dashboard Widget

**Route**: `GET /ui/silences`

**Display Elements**:
- Header: "Silences Management" + Create Silence button
- Filter panel: Status dropdown, Creator input, Date range picker
- Table columns: Status badge, Creator, Comment, Time range, Actions
- Pagination: Page size selector (10/25/50/100), Prev/Next buttons
- Empty state: "No silences found" message

**Table Row Actions**:
- View Details (eye icon) → Navigate to `/ui/silences/{id}`
- Edit (pencil icon) → Open edit modal
- Delete (trash icon) → Confirmation dialog → DELETE API call

**Default Behavior**:
- Load first 25 silences (sorted by created_at desc)
- Status filter: "all" (show pending + active + expired)
- Auto-refresh every 30 seconds (configurable)

**Performance Target**:
- Initial load: <500ms (SSR)
- Filter update: <200ms (client-side)
- Auto-refresh: <100ms (fetch + DOM update)

---

### FR-2: Create Silence Form

**Route**: `GET /ui/silences/create` (form page)
**Submit**: `POST /api/v2/silences` (API endpoint)

**Form Fields**:
1. **Creator** (email input)
   - Validation: required, email format, 1-255 chars
   - Auto-fill from current user (if authenticated)

2. **Comment** (textarea)
   - Validation: required, 3-1024 chars
   - Character counter

3. **Time Range** (datetime-local inputs)
   - Start Time: required, must be >= now
   - End Time: required, must be > Start Time
   - Quick presets: 1h, 4h, 8h, 24h, 7d

4. **Matchers** (dynamic list)
   - Name (input): required, 1-255 chars
   - Operator (select): `=`, `!=`, `=~`, `!~`
   - Value (input): required, 1-1024 chars
   - Add/Remove matcher buttons
   - Minimum 1 matcher, maximum 100

**Submit Behavior**:
- Client-side validation → Show inline errors
- Server-side validation → Show error toast
- Success → Redirect to `/ui/silences` + Success toast
- Failure → Stay on form + Show error message

**Example Form**:
```
┌─────────────────────────────────────────┐
│ Create Silence                          │
├─────────────────────────────────────────┤
│ Creator: ops@example.com                │
│ Comment: Maintenance window for DB      │
│                                         │
│ Start: 2025-11-06 12:00                 │
│ End:   2025-11-06 14:00  [1h][4h][8h]   │
│                                         │
│ Matchers:                               │
│ ┌─────────────────────────────────────┐ │
│ │ alertname  [=]  HighCPU      [×]    │ │
│ │ job        [=]  api-server   [×]    │ │
│ │ [+ Add Matcher]                     │ │
│ └─────────────────────────────────────┘ │
│                                         │
│ [Cancel] [Create Silence]               │
└─────────────────────────────────────────┘
```

---

### FR-3: Edit Silence Form

**Route**: `GET /ui/silences/{id}/edit`
**Submit**: `PUT /api/v2/silences/{id}`

**Editable Fields**:
- Comment (textarea)
- End Time (datetime-local)
- Matchers (dynamic list)

**Read-Only Fields** (displayed but not editable):
- ID
- Creator
- Start Time
- Created At

**Submit Behavior**:
- Same as FR-2 Create Form
- Success → Redirect to `/ui/silences/{id}` + Success toast

---

### FR-4: Bulk Operations Interface

**Activation**:
- Checkbox in table header → Select all on current page
- Checkbox per row → Individual selection
- Toolbar appears when ≥1 silence selected

**Bulk Toolbar**:
```
┌─────────────────────────────────────────┐
│ [5 selected] [Bulk Delete] [Cancel]     │
└─────────────────────────────────────────┘
```

**Bulk Delete Flow**:
1. User clicks "Bulk Delete"
2. Confirmation modal:
   ```
   Are you sure you want to delete 5 silences?
   This action cannot be undone.
   [Cancel] [Delete]
   ```
3. On confirm → POST `/api/v2/silences/bulk/delete` with IDs
4. Progress indicator (spinner + "Deleting 3/5...")
5. Success → Reload table + Success toast ("5 silences deleted")
6. Partial failure → Show warning toast ("3 deleted, 2 failed") + Error details

**Performance Target**:
- Bulk delete 100 silences: <2s (p95)
- UI responsiveness during operation: No freeze

---

### FR-5: Silence Detail View

**Route**: `GET /ui/silences/{id}`

**Display Sections**:

1. **Header**
   - ID badge
   - Status badge (colored: green=active, blue=pending, gray=expired)
   - Action buttons: Edit, Delete, Extend

2. **Basic Info**
   - Creator
   - Comment
   - Created At / Updated At
   - Time range (Start → End)
   - Duration (human-readable: "2 hours")

3. **Matchers**
   - Table: Name, Operator, Value, IsRegex
   - Badge indicators для regex matchers

4. **Matched Alerts** (real-time)
   - Count: "Currently silencing 12 alerts"
   - Link to filtered alerts view
   - Auto-refresh every 10s

5. **Actions History** (future enhancement)
   - Log of edit/extend operations
   - Who changed what and when

**Quick Actions**:
- **Edit**: Navigate to `/ui/silences/{id}/edit`
- **Delete**: Confirmation → DELETE `/api/v2/silences/{id}` → Redirect to `/ui/silences`
- **Extend**: Quick modal → Update End Time → PUT `/api/v2/silences/{id}`

---

### FR-6: Real-Time Updates (WebSocket)

**WebSocket Endpoint**: `WS /ws/silences`

**Events Published**:
```json
{
  "type": "silence_created",
  "data": {
    "id": "uuid",
    "creator": "ops@example.com",
    "status": "pending"
  },
  "timestamp": "2025-11-06T12:00:00Z"
}
```

**Event Types**:
- `silence_created`
- `silence_updated`
- `silence_deleted`
- `silence_expired` (triggered by GC worker)

**Client Behavior**:
- Subscribe to `/ws/silences` on page load
- On event → Update UI without full reload
- On disconnect → Show reconnecting indicator
- Auto-reconnect with exponential backoff

**UI Updates**:
- New silence → Prepend to table (with fade-in animation)
- Updated silence → Update row in-place (highlight change)
- Deleted silence → Remove row (with fade-out animation)
- Toast notification for each event

---

### FR-7: Advanced Filtering UI

**Filter Panel** (collapsible):
```
┌─────────────────────────────────────────┐
│ Filters [▼]                              │
├─────────────────────────────────────────┤
│ Status:   [All ▼] [Pending][Active]     │
│ Creator:  [Enter email...]               │
│ Matcher:  [alertname=HighCPU]           │
│ Start:    [2025-11-01] to [2025-11-07]  │
│                                         │
│ [Clear Filters] [Save Preset]           │
└─────────────────────────────────────────┘
```

**Filter Presets**:
- "My Silences" (filter by current user email)
- "Active Last 24h" (status=active + starts within 24h)
- "Expiring Soon" (ends within next 1h)
- Custom presets (saved in localStorage)

**URL State Persistence**:
- Filters encoded in query params
- Shareable URLs: `/ui/silences?status=active&creator=ops@example.com`
- Browser back/forward support

---

### FR-8: Silence Templates (150% Feature)

**Route**: `GET /ui/silences/templates`

**Built-In Templates**:
1. **Maintenance Window**
   - Matchers: `{alertname=.*, type="maintenance"}`
   - Duration: 2 hours

2. **On-Call Handoff**
   - Matchers: `{alertname="OnCallPageCritical"}`
   - Duration: 1 hour

3. **Incident Response**
   - Matchers: `{severity="critical", incident=~"INC-.*"}`
   - Duration: 4 hours

**Template Usage**:
- Click template → Pre-fill create form
- User adjusts values → Submit

**Template Editor** (future):
- CRUD operations для custom templates
- Share templates across team

---

### FR-9: Analytics Dashboard (150% Feature)

**Route**: `GET /ui/silences/analytics`

**Widgets**:

1. **Statistics Cards**
   - Total Silences (all time)
   - Active Silences (right now)
   - Expired Last 24h
   - Average Duration

2. **Timeline Chart** (Chart.js or similar)
   - X-axis: Time (last 7 days)
   - Y-axis: Count
   - Stacked bars: Pending / Active / Expired

3. **Top Creators** (table)
   - Creator email
   - Silences created (count)
   - Last created (timestamp)

4. **Top Silenced Alerts** (table)
   - Alert name
   - Times silenced (count)
   - Total duration

**Data Source**:
- Fetch from `/api/v2/silences/analytics` (new endpoint)
- Cache results for 5 minutes
- Auto-refresh every 5 minutes

---

## 🔒 Non-Functional Requirements

### NFR-1: Performance

- **Initial Page Load**: <1s (p95), <2s (p99)
- **SSR Rendering**: <500ms for 100 silences
- **Client-Side Filtering**: <100ms for 1000 silences
- **WebSocket Latency**: <200ms from event to UI update
- **Bundle Size**: <100 KB JavaScript (gzipped)
- **Memory Usage**: <50 MB for browser tab

### NFR-2: Scalability

- Support **10,000+ silences** in UI (pagination + virtualization)
- Handle **100+ concurrent WebSocket connections**
- Server-side pagination to avoid large payloads
- Infinite scroll (alternative to traditional pagination)

### NFR-3: Reliability

- **Error Boundaries**: Graceful error handling (не ломает всю страницу)
- **Retry Logic**: Auto-retry failed API calls (3 attempts)
- **Offline Support**: Show cached data when offline (Service Worker)
- **Validation**: Client + Server-side для всех inputs
- **CSRF Protection**: CSRF tokens для mutating operations

### NFR-4: Security

- **XSS Prevention**: Escape all user inputs в HTML templates
- **CSRF Tokens**: Защита от Cross-Site Request Forgery
- **Content Security Policy**: Restrict inline scripts
- **Input Sanitization**: Validate/sanitize все form inputs
- **Authentication**: JWT token validation (if enabled)
- **Authorization**: Users can only delete their own silences (future)

### NFR-5: Usability

- **Mobile-Responsive**: Breakpoints для mobile/tablet/desktop
- **Touch-Friendly**: Buttons ≥44px для touch targets
- **Loading States**: Spinners для async operations
- **Error Messages**: User-friendly, actionable errors
- **Keyboard Navigation**: Tab order, Enter to submit, Esc to cancel
- **Screen Reader Support**: Semantic HTML, ARIA labels

### NFR-6: Accessibility (WCAG 2.1 AA)

- **Semantic HTML**: `<button>`, `<nav>`, `<main>`, `<form>`
- **ARIA Labels**: `aria-label`, `aria-labelledby` для icons
- **Color Contrast**: ≥4.5:1 для text, ≥3:1 для large text
- **Focus Indicators**: Visible focus rings
- **Alt Text**: Для всех images
- **Screen Reader Announcements**: Live regions для dynamic content

### NFR-7: Browser Compatibility

- **Modern Browsers**:
  - Chrome 90+
  - Firefox 88+
  - Safari 14+
  - Edge 90+
- **No IE11 Support** (EOL)
- **Progressive Enhancement**: Core functionality works без JavaScript

### NFR-8: Observability

- **Client-Side Metrics**: Page load, API latency, errors
- **Server-Side Metrics**: SSR duration, template cache hits
- **Logging**: All API calls logged (request ID for tracing)
- **Error Tracking**: Client-side errors sent to server

---

## 📊 Acceptance Criteria

### AC-1: Core UI Components (100% Must-Have)

- [x] Silence Dashboard Widget (table, filters, pagination)
- [x] Create Silence Form (matchers, validation)
- [x] Edit Silence Form (update comment/end time)
- [x] Bulk Operations Interface (multi-select, bulk delete)
- [x] Silence Detail View (full info, quick actions)

### AC-2: Advanced Features (150% Quality)

- [x] Real-Time Updates (WebSocket, live badge)
- [x] Advanced Filtering UI (multi-filter, presets)
- [x] Silence Templates (3 built-in templates)
- [x] Analytics Dashboard (charts, statistics)
- [x] Mobile-Responsive Design (adaptive layout)
- [x] Accessibility (WCAG 2.1 AA compliant)

### AC-3: Technical Excellence

- [x] Go-Native Implementation (`html/template`)
- [x] Zero external frameworks (no Vue/React)
- [x] Embedded assets (`embed.FS`)
- [x] Server-Side Rendering (SSR)
- [x] Minimal JavaScript (<100 KB gzipped)
- [x] Progressive Web App (PWA manifest)

### AC-4: Integration

- [x] Uses all 7 TN-135 API endpoints
- [x] Integrated into main.go (routes registered)
- [x] Backward compatibility с Python dashboard
- [x] Prometheus metrics для UI operations
- [x] Health checks passing

### AC-5: Testing

- [x] 40+ unit tests (Go template rendering)
- [x] 20+ integration tests (full user flows)
- [x] 10+ E2E tests (Playwright/Cypress)
- [x] Accessibility tests (axe-core)
- [x] Performance tests (Lighthouse score >90)

### AC-6: Documentation

- [x] UI Usage Guide (screenshots, flows)
- [x] Template Development Guide
- [x] Accessibility Guide (WCAG compliance)
- [x] API Integration Examples
- [x] Deployment Guide (assets, CDN)

---

## 🔗 Dependencies

### Upstream Dependencies (Required)

- ✅ **TN-131**: Silence Data Models
- ✅ **TN-132**: Silence Matcher Engine
- ✅ **TN-133**: Silence Storage
- ✅ **TN-134**: Silence Manager Service
- ✅ **TN-135**: Silence API Endpoints (all 7 endpoints)

### Infrastructure Dependencies

- ✅ **TN-16**: Redis Cache (для session storage)
- ✅ **TN-21**: Prometheus Metrics (UI metrics)
- ✅ **TN-20**: Structured Logging (UI operation logs)

### Downstream Consumers

- ⏳ **TN-137**: Advanced Routing (может использовать UI для route configuration)
- ⏳ **Module 12**: Advanced UI/Dashboard (TN-169 to TN-172)

---

## 🚀 Success Metrics

### Quantitative Metrics

- **Performance**:
  - Initial page load: <1s (p95)
  - API response time: <200ms (p95)
  - WebSocket latency: <200ms
  - Lighthouse Performance score: >90

- **Reliability**:
  - Error rate: <0.1% для UI operations
  - Availability: 99.9%+ (health checks)

- **Usability**:
  - Task completion rate: >95% (user testing)
  - Time to create silence: <60s (median)

- **Accessibility**:
  - WCAG 2.1 AA compliance: 100%
  - Keyboard navigation: 100% of features accessible

### Qualitative Metrics

- **Developer Experience**: Clear code structure, easy to maintain
- **User Experience**: Intuitive UI, minimal clicks to complete tasks
- **Design Quality**: Modern look & feel, consistent with enterprise standards
- **Documentation Quality**: Complete usage guide with examples

---

## 📝 Out of Scope

Following features are **explicitly out of scope** for TN-136:

1. **Authentication & Authorization**: User login/logout (deferred to TN-137+)
2. **Advanced Analytics**: ML-powered insights (Module 11)
3. **Mobile Native Apps**: iOS/Android apps (Module 12)
4. **Silence History**: Full audit log UI (future)
5. **Team Management**: User roles, permissions (future)
6. **Notification Integration**: Email/Slack on silence events (future)
7. **Advanced Template Editor**: Visual drag-drop (future)
8. **Multi-Language Support**: i18n (future)

---

## 🎯 Quality Target: 150%

To achieve **150% quality** (Grade A+), TN-136 must deliver:

1. **100% Core Features** (5 UI components)
2. **+50% Advanced Features**:
   - Real-Time Updates (WebSocket)
   - Advanced Filtering UI
   - Silence Templates (3 built-in)
   - Analytics Dashboard
   - Mobile-Responsive Design
   - Accessibility (WCAG 2.1 AA)
   - Go-Native Implementation

3. **Exceptional Quality**:
   - Lighthouse score >90 (Performance, Accessibility, Best Practices)
   - 90%+ test coverage (unit + integration + E2E)
   - Zero accessibility violations (axe-core)
   - <100 KB JavaScript bundle (gzipped)
   - <1s initial page load (p95)

4. **Comprehensive Documentation**:
   - 1,500+ lines UI Usage Guide
   - 800+ lines Template Development Guide
   - 500+ lines Accessibility Guide
   - Screenshots + video demos

**Expected LOC**:
- Go code: ~2,500 lines (handlers, templates, WebSocket)
- HTML templates: ~1,500 lines (Go templates)
- JavaScript: ~1,000 lines (vanilla JS, no frameworks)
- CSS: ~800 lines (modern CSS, flexbox, grid)
- Test code: ~3,000 lines (unit + integration + E2E)
- Documentation: ~3,000 lines (guides, examples)
- **Total**: ~12,000 lines

**Timeline**: 14-18 hours (target: 16h actual)

---

## 📚 References

- [TN-135 Completion Report](/tasks/go-migration-analysis/TN-135-silence-api-endpoints/COMPLETION_REPORT.md)
- [TN-130 Inhibition API](/tasks/go-migration-analysis/TN-130-inhibition-api-endpoints/) (similar UI patterns)
- [Go html/template Package](https://pkg.go.dev/html/template)
- [WCAG 2.1 AA Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [Progressive Web App Checklist](https://web.dev/pwa-checklist/)

---

**Document Version**: 1.0
**Created**: 2025-11-06
**Author**: Kilo Code AI
**Status**: APPROVED FOR IMPLEMENTATION
