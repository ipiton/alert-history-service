# TN-77: Modern Dashboard Page — Requirements Document

**Task ID**: TN-77
**Module**: Phase 9: Dashboard & UI
**Priority**: HIGH (P1 - Must Have for Production UI)
**Depends On**: TN-76 (Dashboard Template Engine - 165.9% ✅)
**Target Quality**: **150% (Grade A+ Enterprise)**
**Estimated Effort**: 14-18 hours
**Started**: 2025-11-19
**Status**: 🟡 IN PROGRESS

---

## Executive Summary

**Mission**: Разработать **modern, production-ready dashboard page** для Alertmanager++ OSS с использованием **CSS Grid/Flexbox**, обеспечивающую профессиональный UX, мобильную адаптивность, и высокую производительность.

**Strategic Value**:
- 🎨 **Professional UI/UX** - Современный дизайн уровня SaaS продуктов
- ⚡ **Performance-First** - <1s First Contentful Paint, <50ms render
- 📱 **Mobile-First** - Responsive design (320px→2560px)
- ♿ **Accessibility** - WCAG 2.1 AA compliance
- 🔒 **Security** - CSP headers, XSS protection
- 📊 **Observability** - Comprehensive Prometheus metrics

**User Journey**:
```
User → GET /dashboard →
  Template Engine →
    Fetch Data (Alerts, Silences, Stats) →
      Render Modern Layout (CSS Grid) →
        Browser (SSR, no JS required for core)
```

**Success Criteria (150% Target)**:
- ✅ Modern CSS Grid/Flexbox layout (mobile-first)
- ✅ 6+ dashboard sections (stats, alerts, silences, charts, health)
- ✅ Responsive design (3 breakpoints: mobile, tablet, desktop)
- ✅ Performance: <50ms SSR, <1s First Contentful Paint
- ✅ Accessibility: WCAG 2.1 AA (keyboard nav, ARIA labels, semantic HTML)
- ✅ 8+ Prometheus metrics
- ✅ 85%+ test coverage (handler + integration tests)
- ✅ Comprehensive documentation (README, API guide, style guide)

---

## 1. Functional Requirements (FR)

### FR-1: Modern Dashboard Layout
**Priority**: CRITICAL
**Complexity**: MEDIUM

**Description**: Создать responsive dashboard layout с CSS Grid для основной структуры и Flexbox для компонентов.

**Requirements**:
- **FR-1.1**: CSS Grid для главной страницы
  - 12-column grid система
  - Auto-fit/auto-fill для адаптивности
  - Grid areas для семантической разметки
  - Gap spacing система (8px baseline)

- **FR-1.2**: Flexbox для компонентов
  - Stats cards (row/column switch на mobile)
  - Alert list items (flex alignment)
  - Navigation bar (space-between)
  - Footer (justify-content)

- **FR-1.3**: Responsive breakpoints
  - Mobile: 320px - 767px (1 column, stack all)
  - Tablet: 768px - 1023px (2 columns, side nav collapsible)
  - Desktop: 1024px+ (3-4 columns, full layout)

**Acceptance Criteria**:
```
✅ Grid система работает на всех breakpoints
✅ Flexbox выравнивание корректное
✅ Нет горизонтального скролла на mobile
✅ Touch targets ≥44px (mobile UX)
✅ Visual hierarchy ясен на всех размерах
```

---

### FR-2: Dashboard Sections
**Priority**: CRITICAL
**Complexity**: HIGH

**Description**: Реализовать 6+ dashboard sections с real data integration.

**Sections**:

#### 2.1 **Stats Overview** (Row 1)
```html
<div class="stats-grid">
  <StatCard icon="🔥" value="42" label="Firing Alerts" severity="critical" />
  <StatCard icon="✅" value="128" label="Resolved Today" trend="+12%" />
  <StatCard icon="🔕" value="5" label="Active Silences" />
  <StatCard icon="🎯" value="8" label="Inhibited Alerts" />
</div>
```

**Data Source**:
- GET /api/dashboard/overview
- Real-time stats from PostgreSQL
- Cached for 10s (Redis)

**Metrics**:
- Firing alerts (current)
- Resolved alerts (24h window)
- Active silences (from TN-133)
- Inhibited alerts (from TN-129)

---

#### 2.2 **Recent Alerts** (Row 2, Left)
```html
<div class="alert-feed">
  <AlertCard
    status="firing"
    severity="critical"
    alertname="HighMemoryUsage"
    summary="Pod production-api-7d9f8 at 95% memory"
    labels={{namespace: production, pod: api-7d9f8}}
    confidence="0.92"
    startsAt="2025-11-19T10:30:00Z"
  />
  ...
</div>
```

**Data Source**:
- GET /api/dashboard/alerts/recent?limit=10
- PostgreSQL: `SELECT * FROM alerts ORDER BY starts_at DESC LIMIT 10`
- Include LLM classification data (severity, confidence)

**Features**:
- Click to expand (show full labels)
- Severity color coding (critical=red, warning=orange, info=blue)
- Confidence score badge (if LLM classified)
- Time ago (relative time: "5 minutes ago")
- Quick actions (Silence, View Details)

---

#### 2.3 **Active Silences** (Row 2, Right)
```html
<div class="silence-feed">
  <SilenceCard
    creator="ops-team"
    comment="Maintenance window: DB migration"
    matchers={{alertname: "HighMemoryUsage", namespace: "production"}}
    endsAt="2025-11-19T14:00:00Z"
    status="active"
  />
  ...
</div>
```

**Data Source**:
- Silence Manager API (TN-134)
- Filter: status=active
- Sort: expires_at ASC (soonest first)

**Features**:
- Countdown timer (visual: "Expires in 2h 30m")
- Matcher tags (color coded)
- Quick actions (Extend, Edit, Delete)
- Status indicator (active, pending, expired)

---

#### 2.4 **Alert Timeline** (Row 3, Full Width)
```html
<div class="timeline-chart">
  <canvas id="alert-timeline" aria-label="Alert timeline chart"></canvas>
</div>
```

**Data Source**:
- GET /api/dashboard/charts?range=24h
- Aggregated alert counts per hour
- Breakdown by severity

**Implementation**:
- Server-side SVG generation (no JS required!)
- Or Chart.js (progressive enhancement)
- Responsive sizing (100% width)
- Accessibility: ARIA labels, keyboard navigation

**Chart Types**:
- Stacked bar chart (severity breakdown)
- Line chart (total alerts over time)
- Color scheme: critical=red, warning=orange, info=blue

---

#### 2.5 **System Health** (Row 4, Left)
```html
<div class="health-panel">
  <HealthMetric name="PostgreSQL" status="healthy" latency="2.3ms" />
  <HealthMetric name="Redis" status="healthy" latency="0.8ms" />
  <HealthMetric name="LLM Service" status="degraded" latency="450ms" />
  <HealthMetric name="Publishing Queue" status="healthy" depth="12" />
</div>
```

**Data Source**:
- GET /api/dashboard/health
- Health checks from all subsystems
- Latency metrics from Prometheus

**Status Levels**:
- 🟢 Healthy: <100ms latency, 0 errors
- 🟡 Degraded: 100-500ms latency, <1% errors
- 🔴 Unhealthy: >500ms latency or >1% errors

---

#### 2.6 **Quick Actions** (Row 4, Right)
```html
<div class="quick-actions">
  <ActionButton href="/ui/silences/create" icon="🔕" label="Create Silence" />
  <ActionButton href="/ui/alerts" icon="🔍" label="Search Alerts" />
  <ActionButton href="/ui/settings" icon="⚙️" label="Settings" />
  <ActionButton href="/prometheus/graph" icon="📊" label="Prometheus" />
</div>
```

**Features**:
- Icon + label buttons
- Keyboard shortcuts (data-shortcut="shift+s")
- Tooltips on hover
- ARIA labels for screen readers

---

### FR-3: Responsive Design
**Priority**: CRITICAL
**Complexity**: MEDIUM

**Requirements**:
- **FR-3.1**: Mobile-first approach (320px base)
- **FR-3.2**: 3 breakpoints (mobile, tablet, desktop)
- **FR-3.3**: Touch-friendly (≥44px targets)
- **FR-3.4**: No horizontal scroll
- **FR-3.5**: Collapsible sidebar on mobile

**Mobile Layout** (320px - 767px):
```
┌─────────────────┐
│    Header       │ (fixed, 56px height)
├─────────────────┤
│  Stats (stack)  │ (1 column, 4 rows)
├─────────────────┤
│  Recent Alerts  │ (full width, scrollable)
├─────────────────┤
│ Active Silences │ (full width, scrollable)
├─────────────────┤
│     Footer      │
└─────────────────┘
```

**Tablet Layout** (768px - 1023px):
```
┌───────┬─────────────────┐
│ Side  │    Header       │
│ Nav   ├─────────────────┤
│ (col  │  Stats (2x2)    │
│ lap   ├─────────────────┤
│ si    │ Alerts │Silences│
│ ble)  ├─────────────────┤
│       │  Timeline       │
│       ├─────────────────┤
│       │    Footer       │
└───────┴─────────────────┘
```

**Desktop Layout** (1024px+):
```
┌─────┬───────────────────────────┐
│Side │         Header            │
│Nav  ├───────────────────────────┤
│(240 │    Stats (4 cols)         │
│px)  ├─────────────┬─────────────┤
│     │   Alerts    │  Silences   │
│     │   (60%)     │   (40%)     │
│     ├─────────────┴─────────────┤
│     │      Timeline (full)      │
│     ├─────────────┬─────────────┤
│     │  Health     │Quick Actions│
│     ├─────────────┴─────────────┤
│     │         Footer            │
└─────┴───────────────────────────┘
```

**CSS Media Queries**:
```css
/* Mobile First (default) */
.stats-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1rem;
}

/* Tablet */
@media (min-width: 768px) {
  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* Desktop */
@media (min-width: 1024px) {
  .stats-grid {
    grid-template-columns: repeat(4, 1fr);
  }
  .main-content {
    display: grid;
    grid-template-columns: repeat(12, 1fr);
    gap: 1.5rem;
  }
  .alert-feed { grid-column: 1 / 8; }
  .silence-feed { grid-column: 8 / 13; }
}
```

---

### FR-4: Performance Optimization
**Priority**: HIGH
**Complexity**: MEDIUM

**Requirements**:
- **FR-4.1**: Server-Side Rendering (SSR) for First Contentful Paint
- **FR-4.2**: Progressive Enhancement (works without JS)
- **FR-4.3**: CSS minification + compression
- **FR-4.4**: Image optimization (WebP, lazy loading)
- **FR-4.5**: Font optimization (system fonts, preload)

**Performance Targets (150%)**:
- **SSR Render Time**: <50ms (Go template execution)
- **First Contentful Paint (FCP)**: <1.0s
- **Largest Contentful Paint (LCP)**: <2.5s
- **Cumulative Layout Shift (CLS)**: <0.1
- **Time to Interactive (TTI)**: <3.5s

**Optimization Techniques**:
1. **Critical CSS Inline** (inlined in <head>):
   ```html
   <style>
   /* Critical CSS only (above-the-fold) */
   .stats-grid { display: grid; gap: 1rem; }
   .stat-card { padding: 1rem; background: white; }
   </style>
   ```

2. **Deferred CSS** (non-critical):
   ```html
   <link rel="preload" href="/static/css/dashboard.css" as="style" onload="this.onload=null;this.rel='stylesheet'">
   ```

3. **System Fonts** (no web fonts = 0 network requests):
   ```css
   font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
   ```

4. **CSS Grid/Flexbox** (no framework overhead):
   - Чистый CSS, без Bootstrap/Tailwind
   - Custom CSS variables для theme
   - ~5KB минимальный CSS bundle

---

### FR-5: Accessibility (WCAG 2.1 AA)
**Priority**: HIGH
**Complexity**: MEDIUM

**Requirements**:
- **FR-5.1**: Semantic HTML5 elements
- **FR-5.2**: ARIA labels for dynamic content
- **FR-5.3**: Keyboard navigation (Tab, Enter, Esc)
- **FR-5.4**: Focus indicators (2px outline)
- **FR-5.5**: Color contrast ≥4.5:1 (text), ≥3:1 (UI)

**Accessibility Features**:

#### 5.1 **Semantic HTML**:
```html
<main role="main" aria-label="Dashboard">
  <section aria-labelledby="stats-heading">
    <h2 id="stats-heading">Alert Statistics</h2>
    ...
  </section>
  <section aria-labelledby="alerts-heading">
    <h2 id="alerts-heading">Recent Alerts</h2>
    ...
  </section>
</main>
```

#### 5.2 **ARIA Live Regions**:
```html
<div aria-live="polite" aria-atomic="true" class="sr-only">
  {{ if .NewAlertsCount }}
  {{ .NewAlertsCount }} new alerts received
  {{ end }}
</div>
```

#### 5.3 **Keyboard Navigation**:
- Tab order logical (top→bottom, left→right)
- Skip links: "Skip to main content"
- Escape closes modals/dropdowns
- Arrow keys navigate lists

#### 5.4 **Focus Management**:
```css
:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

/* Remove outline for mouse users (but keep for keyboard) */
:focus:not(:focus-visible) {
  outline: none;
}
```

#### 5.5 **Screen Reader Support**:
```html
<span class="sr-only">42 firing alerts, severity critical</span>
<span aria-hidden="true">🔥 42</span>
```

---

## 2. Non-Functional Requirements (NFR)

### NFR-1: Performance
**Priority**: HIGH

| Metric | Target | Excellent (150%) |
|--------|--------|------------------|
| SSR Render Time | <100ms | <50ms |
| First Contentful Paint | <1.5s | <1.0s |
| Largest Contentful Paint | <3.0s | <2.5s |
| Time to Interactive | <5.0s | <3.5s |
| Cumulative Layout Shift | <0.25 | <0.1 |

**Measurement**:
- Lighthouse CI в GitHub Actions
- Real User Monitoring (RUM) metrics
- Prometheus histogram: `dashboard_render_duration_seconds`

---

### NFR-2: Scalability
**Priority**: MEDIUM

**Requirements**:
- Handle 100+ concurrent users
- Render dashboard with 1000+ alerts (<200ms)
- Graceful degradation при high load
- Pagination для больших списков (50 items per page)

**Caching Strategy**:
- Redis cache для dashboard data (TTL 10s)
- HTTP ETag headers для browser cache
- Stale-while-revalidate для non-critical data

---

### NFR-3: Browser Compatibility
**Priority**: MEDIUM

**Supported Browsers**:
- ✅ Chrome 90+ (CSS Grid, Flexbox, CSS Variables)
- ✅ Firefox 88+ (CSS Grid, Flexbox, CSS Variables)
- ✅ Safari 14+ (CSS Grid, Flexbox, CSS Variables)
- ✅ Edge 90+ (Chromium-based)

**Not Supported**:
- ❌ IE11 (end of life 2022)
- ❌ Chrome <80 (no CSS Variables)

**Feature Detection**:
```css
@supports (display: grid) {
  .stats-grid { display: grid; }
}

@supports not (display: grid) {
  .stats-grid { display: flex; flex-wrap: wrap; }
}
```

---

### NFR-4: Security
**Priority**: HIGH

**Requirements**:
- **NFR-4.1**: Content Security Policy (CSP)
  ```http
  Content-Security-Policy: default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'
  ```

- **NFR-4.2**: XSS Protection
  - html/template auto-escaping ✅
  - No inline event handlers (onclick, onerror)
  - Sanitize all user input

- **NFR-4.3**: CSRF Protection
  - CSRF tokens для POST requests
  - SameSite=Strict cookies

- **NFR-4.4**: Rate Limiting
  - 60 requests/minute per IP
  - Prometheus metric: `dashboard_requests_total{status="429"}`

---

### NFR-5: Observability
**Priority**: HIGH

**Prometheus Metrics** (8+ metrics):
```go
// Request metrics
dashboard_requests_total{page, status}        // Counter
dashboard_request_duration_seconds{page}      // Histogram

// Render metrics
dashboard_render_duration_seconds{template}   // Histogram
dashboard_render_errors_total{template}       // Counter

// Data fetch metrics
dashboard_data_fetch_duration_seconds{source} // Histogram (alerts, silences, stats)
dashboard_data_fetch_errors_total{source}     // Counter

// Cache metrics
dashboard_cache_hits_total{cache_type}        // Counter (redis, memory)
dashboard_cache_misses_total{cache_type}      // Counter

// User engagement metrics
dashboard_page_views_total{page}              // Counter
dashboard_user_interactions_total{action}     // Counter (filter, sort, click)
```

**Structured Logging** (slog):
```go
logger.Info("Dashboard rendered",
  "duration_ms", duration.Milliseconds(),
  "alerts_count", len(alerts),
  "silences_count", len(silences),
  "cache_hit", cacheHit,
  "user_id", userID,
)
```

---

## 3. Data Models

### 3.1 DashboardData
```go
// DashboardData is the main data structure for dashboard page.
type DashboardData struct {
  // Meta
  PageTitle   string    `json:"page_title"`
  CurrentTime time.Time `json:"current_time"`

  // Stats Overview
  Stats DashboardStats `json:"stats"`

  // Recent Data
  RecentAlerts         []AlertSummary   `json:"recent_alerts"`
  ActiveSilences       []SilenceSummary `json:"active_silences"`

  // Charts
  AlertTimeline        TimelineData     `json:"alert_timeline"`

  // System Health
  Health               HealthStatus     `json:"health"`

  // User Context
  User                 *User            `json:"user,omitempty"`

  // UI State
  Breadcrumbs          []Breadcrumb     `json:"breadcrumbs"`
  Flash                *FlashMessage    `json:"flash,omitempty"`
}

// DashboardStats contains aggregate statistics.
type DashboardStats struct {
  FiringAlerts    int     `json:"firing_alerts"`
  ResolvedToday   int     `json:"resolved_today"`
  ActiveSilences  int     `json:"active_silences"`
  InhibitedAlerts int     `json:"inhibited_alerts"`

  // Trends (24h comparison)
  FiringTrend     float64 `json:"firing_trend"`     // +12.5% = 0.125
  ResolvedTrend   float64 `json:"resolved_trend"`

  // LLM Stats
  ClassifiedToday int     `json:"classified_today"`
  AvgConfidence   float64 `json:"avg_confidence"`   // 0.0-1.0
}

// AlertSummary is a compact alert representation for dashboard.
type AlertSummary struct {
  Fingerprint  string            `json:"fingerprint"`
  AlertName    string            `json:"alertname"`
  Status       string            `json:"status"`       // firing, resolved
  Severity     string            `json:"severity"`     // critical, warning, info
  Summary      string            `json:"summary"`
  Description  string            `json:"description"`
  Labels       map[string]string `json:"labels"`
  StartsAt     time.Time         `json:"starts_at"`
  EndsAt       *time.Time        `json:"ends_at,omitempty"`

  // LLM Classification
  AIClassification *AIClassification `json:"ai_classification,omitempty"`
}

// AIClassification contains LLM-generated metadata.
type AIClassification struct {
  Severity    string  `json:"severity"`     // critical, warning, info, noise
  Confidence  float64 `json:"confidence"`   // 0.0-1.0
  Reasoning   string  `json:"reasoning"`
  ActionItems []string `json:"action_items,omitempty"`
}

// SilenceSummary is a compact silence representation.
type SilenceSummary struct {
  ID        string            `json:"id"`
  Creator   string            `json:"creator"`
  Comment   string            `json:"comment"`
  Matchers  []Matcher         `json:"matchers"`
  StartsAt  time.Time         `json:"starts_at"`
  EndsAt    time.Time         `json:"ends_at"`
  Status    string            `json:"status"`       // active, pending, expired
  ExpiresIn string            `json:"expires_in"`   // "2h 30m"
}

// TimelineData for alert timeline chart.
type TimelineData struct {
  Labels []string          `json:"labels"`       // ["00:00", "01:00", ...]
  Series []TimelineSeries  `json:"series"`
}

type TimelineSeries struct {
  Name   string  `json:"name"`    // "Critical", "Warning", "Info"
  Color  string  `json:"color"`   // "#f44336"
  Values []int   `json:"values"`  // [5, 12, 8, ...]
}

// HealthStatus contains system health metrics.
type HealthStatus struct {
  Overall    string        `json:"overall"`    // healthy, degraded, unhealthy
  Components []HealthCheck `json:"components"`
}

type HealthCheck struct {
  Name     string  `json:"name"`       // "PostgreSQL", "Redis", "LLM"
  Status   string  `json:"status"`     // healthy, degraded, unhealthy
  Latency  float64 `json:"latency_ms"` // milliseconds
  Message  string  `json:"message,omitempty"`
}
```

---

## 4. API Integration

### 4.1 GET /api/dashboard/overview
**Purpose**: Fetch dashboard overview data (stats + recent alerts + silences)

**Request**:
```http
GET /api/dashboard/overview HTTP/1.1
Host: alert-history.local
Accept: application/json
```

**Response** (200 OK):
```json
{
  "stats": {
    "firing_alerts": 42,
    "resolved_today": 128,
    "active_silences": 5,
    "inhibited_alerts": 8,
    "firing_trend": 0.125,
    "classified_today": 95,
    "avg_confidence": 0.87
  },
  "recent_alerts": [
    {
      "fingerprint": "abc123",
      "alertname": "HighMemoryUsage",
      "status": "firing",
      "severity": "critical",
      "summary": "Pod production-api-7d9f8 at 95% memory",
      "labels": {
        "namespace": "production",
        "pod": "api-7d9f8"
      },
      "starts_at": "2025-11-19T10:30:00Z",
      "ai_classification": {
        "severity": "critical",
        "confidence": 0.92,
        "reasoning": "Memory usage exceeds 90% threshold, pod restart imminent",
        "action_items": ["Scale horizontally", "Investigate memory leak"]
      }
    }
  ],
  "active_silences": [
    {
      "id": "silence-123",
      "creator": "ops-team",
      "comment": "Maintenance window: DB migration",
      "matchers": [
        {"name": "alertname", "operator": "=", "value": "HighMemoryUsage"}
      ],
      "starts_at": "2025-11-19T12:00:00Z",
      "ends_at": "2025-11-19T14:00:00Z",
      "status": "active",
      "expires_in": "2h 30m"
    }
  ]
}
```

**Performance**: <50ms (cached 10s in Redis)

---

### 4.2 GET /api/dashboard/health
**Purpose**: Fetch system health status

**Request**:
```http
GET /api/dashboard/health HTTP/1.1
```

**Response** (200 OK):
```json
{
  "overall": "healthy",
  "components": [
    {
      "name": "PostgreSQL",
      "status": "healthy",
      "latency_ms": 2.3
    },
    {
      "name": "Redis",
      "status": "healthy",
      "latency_ms": 0.8
    },
    {
      "name": "LLM Service",
      "status": "degraded",
      "latency_ms": 450,
      "message": "High latency detected"
    },
    {
      "name": "Publishing Queue",
      "status": "healthy",
      "latency_ms": 5.1,
      "message": "12 jobs in queue"
    }
  ]
}
```

**Performance**: <20ms (real-time check)

---

### 4.3 GET /api/dashboard/charts
**Purpose**: Fetch alert timeline chart data

**Request**:
```http
GET /api/dashboard/charts?range=24h HTTP/1.1
```

**Query Parameters**:
- `range`: Time range (1h, 6h, 24h, 7d, 30d)

**Response** (200 OK):
```json
{
  "labels": ["00:00", "01:00", "02:00", ..., "23:00"],
  "series": [
    {
      "name": "Critical",
      "color": "#f44336",
      "values": [5, 12, 8, 3, 2, ...]
    },
    {
      "name": "Warning",
      "color": "#ff9800",
      "values": [15, 22, 18, 13, 12, ...]
    },
    {
      "name": "Info",
      "color": "#2196f3",
      "values": [45, 52, 48, 43, 42, ...]
    }
  ]
}
```

**Performance**: <100ms (PostgreSQL aggregation query)

---

## 5. CSS Architecture

### 5.1 CSS Variables (Design Tokens)
```css
:root {
  /* Colors - Primary */
  --color-primary: #1976d2;
  --color-primary-dark: #1565c0;
  --color-primary-light: #42a5f5;

  /* Colors - Severity */
  --color-critical: #f44336;
  --color-warning: #ff9800;
  --color-info: #2196f3;
  --color-success: #4caf50;

  /* Colors - Status */
  --color-firing: var(--color-critical);
  --color-resolved: var(--color-success);
  --color-pending: var(--color-warning);

  /* Colors - Neutral */
  --color-bg: #ffffff;
  --color-bg-secondary: #f5f5f5;
  --color-text: #212121;
  --color-text-secondary: #757575;
  --color-border: #e0e0e0;

  /* Spacing (8px baseline) */
  --spacing-xs: 4px;
  --spacing-sm: 8px;
  --spacing-md: 16px;
  --spacing-lg: 24px;
  --spacing-xl: 32px;
  --spacing-xxl: 48px;

  /* Typography */
  --font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
  --font-size-xs: 12px;
  --font-size-sm: 14px;
  --font-size-base: 16px;
  --font-size-lg: 18px;
  --font-size-xl: 24px;
  --font-size-xxl: 32px;

  /* Layout */
  --header-height: 64px;
  --sidebar-width: 240px;
  --container-max-width: 1400px;

  /* Shadows */
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 2px 4px rgba(0, 0, 0, 0.1);
  --shadow-lg: 0 4px 8px rgba(0, 0, 0, 0.15);

  /* Border Radius */
  --radius-sm: 4px;
  --radius-md: 8px;
  --radius-lg: 12px;

  /* Transitions */
  --transition-fast: 150ms ease;
  --transition-base: 250ms ease;
  --transition-slow: 350ms ease;
}

/* Dark Mode Support (future) */
@media (prefers-color-scheme: dark) {
  :root {
    --color-bg: #121212;
    --color-bg-secondary: #1e1e1e;
    --color-text: #ffffff;
    --color-text-secondary: #b0b0b0;
    --color-border: #333333;
  }
}
```

---

### 5.2 CSS Grid System
```css
/* Main Dashboard Grid */
.dashboard-grid {
  display: grid;
  grid-template-columns: repeat(12, 1fr);
  grid-template-rows: auto;
  gap: var(--spacing-lg);
  max-width: var(--container-max-width);
  margin: 0 auto;
  padding: var(--spacing-lg);
}

/* Stats Cards (Row 1) */
.stats-section {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: var(--spacing-md);
}

/* Alerts Feed (Row 2, Left 60%) */
.alerts-section {
  grid-column: 1 / 8;
}

/* Silences Feed (Row 2, Right 40%) */
.silences-section {
  grid-column: 8 / -1;
}

/* Timeline Chart (Row 3, Full Width) */
.timeline-section {
  grid-column: 1 / -1;
}

/* Health + Quick Actions (Row 4) */
.health-section {
  grid-column: 1 / 7;
}

.actions-section {
  grid-column: 7 / -1;
}

/* Responsive: Tablet */
@media (max-width: 1023px) {
  .dashboard-grid {
    grid-template-columns: repeat(6, 1fr);
  }

  .alerts-section,
  .silences-section,
  .health-section,
  .actions-section {
    grid-column: 1 / -1;
  }
}

/* Responsive: Mobile */
@media (max-width: 767px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
    gap: var(--spacing-md);
    padding: var(--spacing-sm);
  }

  .stats-section {
    grid-template-columns: 1fr;
  }
}
```

---

## 6. Testing Requirements

### 6.1 Unit Tests (Handler)
```go
// dashboard_handler_test.go
func TestDashboardHandler_RenderDashboard(t *testing.T) {
  tests := []struct {
    name           string
    mockStats      DashboardStats
    mockAlerts     []AlertSummary
    expectedStatus int
  }{
    {
      name: "successful render with data",
      mockStats: DashboardStats{
        FiringAlerts: 42,
        ResolvedToday: 128,
      },
      mockAlerts: []AlertSummary{
        {Fingerprint: "abc", AlertName: "Test"},
      },
      expectedStatus: http.StatusOK,
    },
    {
      name: "empty dashboard",
      mockStats: DashboardStats{},
      mockAlerts: []AlertSummary{},
      expectedStatus: http.StatusOK,
    },
    // ... 20+ test cases
  }
}
```

**Coverage Target**: 85%+

---

### 6.2 Integration Tests
```go
func TestDashboard_Integration(t *testing.T) {
  // Setup: Real PostgreSQL testcontainer
  // Setup: Real Redis testcontainer

  // Test: End-to-end dashboard render
  resp := httptest.NewRecorder()
  req := httptest.NewRequest("GET", "/dashboard", nil)

  handler.ServeHTTP(resp, req)

  assert.Equal(t, http.StatusOK, resp.Code)
  assert.Contains(t, resp.Body.String(), "Dashboard")
  assert.Contains(t, resp.Body.String(), "Firing Alerts")
}
```

---

### 6.3 E2E Tests (Playwright)
```javascript
// dashboard.spec.ts
test('dashboard loads and displays stats', async ({ page }) => {
  await page.goto('http://localhost:8080/dashboard');

  // Check title
  await expect(page).toHaveTitle(/Dashboard/);

  // Check stats cards
  await expect(page.locator('.stat-card')).toHaveCount(4);

  // Check responsive layout
  await page.setViewportSize({ width: 375, height: 667 }); // iPhone
  await expect(page.locator('.stats-grid')).toHaveCSS('grid-template-columns', '1fr');

  // Check accessibility
  const violations = await injectAxe(page);
  expect(violations).toHaveLength(0);
});
```

---

### 6.4 Performance Tests (Lighthouse CI)
```yaml
# .lighthouserc.json
{
  "ci": {
    "collect": {
      "url": ["http://localhost:8080/dashboard"],
      "numberOfRuns": 3
    },
    "assert": {
      "assertions": {
        "categories:performance": ["error", {"minScore": 0.9}],
        "categories:accessibility": ["error", {"minScore": 0.95}],
        "first-contentful-paint": ["error", {"maxNumericValue": 1000}],
        "largest-contentful-paint": ["error", {"maxNumericValue": 2500}],
        "cumulative-layout-shift": ["error", {"maxNumericValue": 0.1}]
      }
    }
  }
}
```

---

## 7. Risks & Mitigations

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| **Slow SSR render** (>100ms) | MEDIUM | HIGH | Cache dashboard data (Redis 10s TTL), Optimize SQL queries, Pagination |
| **Poor mobile UX** | MEDIUM | HIGH | Mobile-first design, Touch targets ≥44px, Test on real devices |
| **Accessibility issues** | LOW | MEDIUM | Lighthouse CI, axe-core validation, Manual screen reader testing |
| **CSS Grid browser support** | LOW | LOW | @supports fallback to Flexbox, Autoprefixer |
| **Large CSS bundle** | LOW | MEDIUM | CSS minification, Purge unused CSS, Gzip compression |

---

## 8. Dependencies

### 8.1 Upstream Dependencies (COMPLETED)
- ✅ **TN-76**: Dashboard Template Engine (165.9%, Grade A+)
- ✅ **TN-32**: AlertStorage PostgreSQL (100%)
- ✅ **TN-134**: Silence Manager (150%, Grade A+)
- ✅ **TN-21**: Prometheus Metrics (100%)

### 8.2 Data Sources
- PostgreSQL: alerts table (firing/resolved counts)
- Redis: cached dashboard data
- Silence Manager: active silences
- Health checks: PostgreSQL, Redis, LLM latency

### 8.3 External Resources
- **CSS Frameworks**: NONE (pure CSS Grid/Flexbox)
- **JavaScript**: NONE (progressive enhancement only)
- **Fonts**: System fonts (no web fonts)
- **Icons**: Unicode emoji (no icon fonts)

---

## 9. Success Metrics (150% Achievement)

| Category | Baseline (100%) | Target (150%) | Measurement |
|----------|-----------------|---------------|-------------|
| **Implementation** | 6 sections | 6 sections + charts | Code review |
| **Performance** | <100ms SSR | <50ms SSR | Prometheus histogram |
| **Accessibility** | WCAG 2.1 A | WCAG 2.1 AA | Lighthouse CI |
| **Test Coverage** | 70% | 85%+ | go test -cover |
| **Documentation** | 3,000 LOC | 5,000+ LOC | Line count |
| **Responsive Design** | 2 breakpoints | 3 breakpoints + print | Browser DevTools |
| **Metrics** | 4 metrics | 8+ metrics | Prometheus |

---

## 10. Timeline

### Phase 0: Analysis (0.5h) ✅
- Review TN-76 implementation
- Analyze existing templates
- Define requirements

### Phase 1: Documentation (2h)
- requirements.md (this file)
- design.md
- tasks.md

### Phase 2: Git Branch Setup (0.5h)
- Create feature branch
- Commit docs

### Phase 3: Core Layout (3h)
- dashboard.html template
- CSS Grid system
- Responsive breakpoints

### Phase 4: Dashboard Sections (4h)
- Stats cards
- Alert feed
- Silence feed
- Timeline chart
- Health panel
- Quick actions

### Phase 5: API Integration (2h)
- dashboard_handler.go
- GET /api/dashboard/overview
- GET /api/dashboard/health
- GET /api/dashboard/charts

### Phase 6: Testing (3h)
- Unit tests (handler)
- Integration tests
- E2E tests (Playwright)

### Phase 7: Performance Optimization (2h)
- CSS minification
- Critical CSS inline
- Caching strategy

### Phase 8: Accessibility (1.5h)
- ARIA labels
- Keyboard navigation
- Lighthouse audit

### Phase 9: Documentation (1.5h)
- README.md
- STYLE_GUIDE.md
- COMPLETION_REPORT.md

### Phase 10: Final Validation (1h)
- Manual testing
- Performance benchmarks
- Certification

**Total**: 14-18 hours

---

## 11. Acceptance Criteria (150% Quality Gate)

### Core Requirements (100%)
- ✅ 6+ dashboard sections implemented
- ✅ Responsive design (3 breakpoints)
- ✅ CSS Grid/Flexbox layout
- ✅ All data integrated (alerts, silences, stats, health)
- ✅ Zero compilation errors
- ✅ Zero accessibility violations (WCAG 2.1 A)

### Enhanced Quality (150%)
- ✅ Performance: <50ms SSR, <1s FCP
- ✅ Accessibility: WCAG 2.1 AA
- ✅ 8+ Prometheus metrics
- ✅ 85%+ test coverage
- ✅ 5,000+ LOC documentation
- ✅ Lighthouse score 90+ (performance, accessibility)
- ✅ Mobile-first design (touch-friendly)
- ✅ Progressive enhancement (works without JS)
- ✅ Print-friendly CSS
- ✅ Dark mode support (CSS variables)

---

## 12. References

### Design Inspiration
- Grafana Dashboard (https://grafana.com/)
- Prometheus UI (https://prometheus.io/)
- PagerDuty Incidents (https://www.pagerduty.com/)
- Datadog Dashboards (https://www.datadoghq.com/)

### Technical Resources
- MDN CSS Grid: https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Grid_Layout
- MDN Flexbox: https://developer.mozilla.org/en-US/docs/Web/CSS/CSS_Flexible_Box_Layout
- WCAG 2.1 Guidelines: https://www.w3.org/WAI/WCAG21/quickref/
- Core Web Vitals: https://web.dev/vitals/

---

**Document Version**: 1.0
**Last Updated**: 2025-11-19
**Author**: AI Assistant (Enterprise Requirements Engineer)
**Status**: ✅ APPROVED FOR IMPLEMENTATION
