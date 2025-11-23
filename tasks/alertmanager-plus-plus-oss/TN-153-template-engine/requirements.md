# TN-153: Template Engine Integration — Requirements

**Date**: 2025-11-22
**Task ID**: TN-153
**Priority**: High (Sprint 3 - Config & Templates)
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Estimated Effort**: 8-12 hours
**Depends On**: TN-152 (Hot Reload) ✅

---

## 📖 Executive Summary

**Goal**: Реализовать unified template engine для обработки Go text/template в notification messages (Slack, PagerDuty, Email), обеспечивая динамическое форматирование alert notifications с поддержкой custom functions, caching, и hot reload.

**Business Value**:
- 🎨 **Flexible Notifications**: Кастомизация сообщений без изменения кода
- 🔄 **Alertmanager Compatibility**: Поддержка Alertmanager template syntax
- ⚡ **Performance**: Template caching для production
- 🔧 **Maintainability**: Централизованная template обработка
- 📊 **Observability**: Prometheus metrics для template rendering

**Current State**:
- ✅ UI Template Engine существует (`internal/ui/template_engine.go`)
- ✅ Alert Formatter существует (`internal/infrastructure/publishing/formatter.go`)
- ❌ **GAP**: Receiver configs (SlackConfig.Title, Text) не поддерживают Go templates
- ❌ **GAP**: Hardcoded notification formatting

**Target State**:
```yaml
receivers:
  - name: slack-oncall
    slack_configs:
      - api_url: "${SLACK_WEBHOOK}"
        channel: "#oncall"
        title: "🔥 {{ .GroupLabels.alertname }} - {{ .Status }}"
        text: |
          *Severity*: {{ .Labels.severity }}
          *Instance*: {{ .Labels.instance }}
          *Summary*: {{ .Annotations.summary }}
          {{ if .Annotations.runbook_url }}
          📖 [Runbook]({{ .Annotations.runbook_url }})
          {{ end }}
```

**Success Criteria**:
- ✅ Template engine обрабатывает все receiver configs (Slack, PagerDuty, Email)
- ✅ 20+ custom template functions (Alertmanager-compatible)
- ✅ Template caching с hot reload support
- ✅ 30+ unit tests, 90%+ coverage
- ✅ Performance: < 5ms per template render (p95)
- ✅ Comprehensive documentation

---

## 1. Functional Requirements (FR)

### FR-1: Template Engine Core

**Priority**: CRITICAL

**Description**: Unified template engine для обработки Go text/template в notification messages.

**Requirements**:
- **FR-1.1**: Support Go `text/template` syntax (NOT html/template - no escaping)
- **FR-1.2**: Template parsing с validation
- **FR-1.3**: Template execution с error handling
- **FR-1.4**: Template caching (LRU cache, 1000 templates max)
- **FR-1.5**: Thread-safe concurrent execution
- **FR-1.6**: Context timeout support (5s max per render)

**Template Data Structure**:
```go
type TemplateData struct {
    // Alert fields
    Status       string            // "firing" | "resolved"
    Labels       map[string]string // Alert labels
    Annotations  map[string]string // Alert annotations
    StartsAt     time.Time         // Alert start time
    EndsAt       time.Time         // Alert end time (if resolved)
    GeneratorURL string            // Prometheus generator URL
    Fingerprint  string            // Alert fingerprint

    // Group fields (for grouped notifications)
    GroupLabels  map[string]string // Grouping labels
    CommonLabels map[string]string // Common labels across group
    CommonAnnotations map[string]string // Common annotations

    // External URL
    ExternalURL  string            // Alert History external URL

    // Receiver context
    Receiver     string            // Receiver name
}
```

**Example Usage**:
```go
engine := NewNotificationTemplateEngine()

tmpl := `🔥 {{ .GroupLabels.alertname }} - {{ .Status }}`
data := &TemplateData{
    Status: "firing",
    GroupLabels: map[string]string{"alertname": "HighCPU"},
}

result, err := engine.Execute(ctx, tmpl, data)
// result: "🔥 HighCPU - firing"
```

---

### FR-2: Custom Template Functions

**Priority**: CRITICAL

**Description**: Alertmanager-compatible template functions для форматирования данных.

**Requirements**:
- **FR-2.1**: Time formatting functions (20+ functions)
- **FR-2.2**: String manipulation functions (15+ functions)
- **FR-2.3**: URL encoding functions (5+ functions)
- **FR-2.4**: Math functions (10+ functions)
- **FR-2.5**: Conditional functions (5+ functions)

**Function Categories**:

#### 1. Time Functions (Alertmanager-compatible)
```go
// Format time
{{ .StartsAt | date "2006-01-02 15:04:05" }}
{{ .StartsAt | humanizeTimestamp }}  // "2 hours ago"
{{ .StartsAt | since }}               // "2h15m"

// Duration
{{ sub .EndsAt .StartsAt | humanizeDuration }}  // "1h 30m"
```

#### 2. String Functions
```go
// Case conversion
{{ .Labels.alertname | toUpper }}
{{ .Labels.alertname | toLower }}
{{ .Labels.alertname | title }}

// Truncation
{{ .Annotations.description | truncate 100 }}
{{ .Annotations.description | truncateWords 20 }}

// Joining
{{ .Labels | sortedPairs | join ", " }}
```

#### 3. URL Functions
```go
// URL encoding
{{ .Labels.instance | urlEncode }}
{{ .ExternalURL | urlQuery "filter" .Labels.alertname }}

// Path joining
{{ .ExternalURL | pathJoin "/alerts" .Fingerprint }}
```

#### 4. Math Functions
```go
// Arithmetic
{{ add .Value 10 }}
{{ sub .Value 5 }}
{{ mul .Value 2 }}
{{ div .Value 3 }}

// Formatting
{{ .Value | humanize }}      // "1.23k"
{{ .Value | humanize1024 }}  // "1.2 KiB"
```

#### 5. Conditional Functions
```go
// Conditionals
{{ if gt .Value 100 }}CRITICAL{{ else }}OK{{ end }}
{{ .Labels.severity | default "warning" }}
{{ .Annotations.runbook_url | empty | not }}
```

**Full Function List** (Alertmanager-compatible):
```
Time Functions (20):
- date, humanizeTimestamp, since, until
- toDate, toDateInZone, duration, durationRound
- now, unixEpoch, dateInZone, dateModify
- htmlDate, htmlDateInZone, dateAgo
- mustDateModify, ago, fromNow, toNow, fromNow

String Functions (15):
- toUpper, toLower, title, untitle
- trim, trimAll, trimPrefix, trimSuffix
- repeat, substr, nospace, trunc
- abbrev, abbrevboth, initials, wrap, wrapWith

URL Functions (5):
- urlEncode, urlDecode, urlQuery, pathJoin, pathBase

Math Functions (10):
- add, sub, mul, div, mod
- max, min, round, ceil, floor

Conditional Functions (5):
- default, empty, coalesce, ternary, has

Collections (10):
- join, split, sortAlpha, reverse
- uniq, without, has, compact, slice, append

Encoding (5):
- b64enc, b64dec, jsonEncode, jsonDecode, toJson
```

---

### FR-3: Receiver Integration

**Priority**: CRITICAL

**Description**: Интеграция template engine с receiver configs (Slack, PagerDuty, Email).

**Requirements**:
- **FR-3.1**: Process SlackConfig.Title, Text, Pretext
- **FR-3.2**: Process PagerDutyConfig.Summary, Details
- **FR-3.3**: Process EmailConfig.Subject, Body (FUTURE - TN-154)
- **FR-3.4**: Process WebhookConfig custom fields
- **FR-3.5**: Backward compatibility (non-template strings work as-is)

**Integration Points**:

#### 1. Slack Integration
```go
// Before (hardcoded)
slackConfig := &SlackConfig{
    Title: "Alert: HighCPU",
    Text:  "CPU usage is high",
}

// After (templated)
slackConfig := &SlackConfig{
    Title: "🔥 {{ .GroupLabels.alertname }} - {{ .Status }}",
    Text: `*Severity*: {{ .Labels.severity }}
*Instance*: {{ .Labels.instance }}
*Started*: {{ .StartsAt | humanizeTimestamp }}`,
}

// Rendering
title, _ := engine.Execute(ctx, slackConfig.Title, templateData)
text, _ := engine.Execute(ctx, slackConfig.Text, templateData)
```

#### 2. PagerDuty Integration
```go
pagerDutyConfig := &PagerDutyConfig{
    Summary: "{{ .Labels.severity | toUpper }}: {{ .GroupLabels.alertname }}",
    Details: map[string]string{
        "instance": "{{ .Labels.instance }}",
        "value":    "{{ .Value | humanize }}",
    },
}
```

#### 3. Email Integration (FUTURE - TN-154)
```go
emailConfig := &EmailConfig{
    Subject: "[{{ .Labels.severity }}] {{ .GroupLabels.alertname }}",
    Body: `
Alert: {{ .GroupLabels.alertname }}
Status: {{ .Status }}
Started: {{ .StartsAt | date "2006-01-02 15:04:05" }}

{{ .Annotations.description }}
`,
}
```

---

### FR-4: Template Caching

**Priority**: HIGH

**Description**: LRU cache для parsed templates с hot reload support.

**Requirements**:
- **FR-4.1**: LRU cache (1000 templates max)
- **FR-4.2**: Cache key: template string hash (SHA256)
- **FR-4.3**: Cache invalidation on config reload (SIGHUP)
- **FR-4.4**: Cache hit/miss metrics
- **FR-4.5**: Thread-safe cache access

**Cache Design**:
```go
type TemplateCache struct {
    cache *lru.Cache // github.com/hashicorp/golang-lru
    mu    sync.RWMutex
}

// Cache operations
func (c *TemplateCache) Get(key string) (*template.Template, bool)
func (c *TemplateCache) Set(key string, tmpl *template.Template)
func (c *TemplateCache) Invalidate()
func (c *TemplateCache) Size() int
```

**Performance Target**:
- Cache hit: < 1ms
- Cache miss (parse + cache): < 5ms
- Cache hit ratio: > 95% in production

---

### FR-5: Error Handling

**Priority**: HIGH

**Description**: Graceful error handling для template parsing и execution.

**Requirements**:
- **FR-5.1**: Parse errors → detailed error messages
- **FR-5.2**: Execution errors → fallback to raw template
- **FR-5.3**: Missing fields → empty string (не panic)
- **FR-5.4**: Timeout errors → context deadline exceeded
- **FR-5.5**: Error logging с structured fields

**Error Types**:
```go
var (
    ErrTemplateParse    = errors.New("template parse failed")
    ErrTemplateExecute  = errors.New("template execute failed")
    ErrTemplateTimeout  = errors.New("template execution timeout")
    ErrInvalidData      = errors.New("invalid template data")
)
```

**Error Handling Example**:
```go
result, err := engine.Execute(ctx, tmpl, data)
if err != nil {
    if errors.Is(err, ErrTemplateParse) {
        // Log parse error, return raw template
        logger.Error("template parse failed", "error", err, "template", tmpl)
        return tmpl, nil
    }
    if errors.Is(err, ErrTemplateTimeout) {
        // Log timeout, return partial result
        logger.Warn("template execution timeout", "error", err)
        return "", err
    }
    // Other errors
    return "", err
}
```

---

## 2. Non-Functional Requirements (NFR)

### NFR-1: Performance

**Priority**: HIGH

**Requirements**:
- **NFR-1.1**: Template parse: < 10ms p95
- **NFR-1.2**: Template execute: < 5ms p95 (cached)
- **NFR-1.3**: Template execute: < 20ms p95 (uncached)
- **NFR-1.4**: Cache hit ratio: > 95%
- **NFR-1.5**: Memory usage: < 50MB for 1000 cached templates

**Benchmarks**:
```go
BenchmarkTemplateParse-8         100000    10000 ns/op
BenchmarkTemplateExecuteCached-8 500000     2000 ns/op
BenchmarkTemplateExecuteUncached-8 50000   20000 ns/op
```

---

### NFR-2: Reliability

**Priority**: HIGH

**Requirements**:
- **NFR-2.1**: Zero panics (recover from template errors)
- **NFR-2.2**: Graceful degradation (fallback to raw template)
- **NFR-2.3**: Thread-safe concurrent execution
- **NFR-2.4**: Context timeout support (5s max)
- **NFR-2.5**: 90%+ test coverage

---

### NFR-3: Observability

**Priority**: HIGH

**Requirements**:
- **NFR-3.1**: 6+ Prometheus metrics
- **NFR-3.2**: Structured logging (slog)
- **NFR-3.3**: Error tracking per template
- **NFR-3.4**: Cache hit/miss tracking

**Prometheus Metrics**:
```promql
# Total template executions
template_executions_total{status="success|error"}

# Execution duration
template_execution_duration_seconds{cached="true|false"}

# Parse errors
template_parse_errors_total{error_type="syntax|timeout"}

# Cache metrics
template_cache_hits_total
template_cache_misses_total
template_cache_size

# Function calls
template_function_calls_total{function="date|humanize|..."}
```

---

### NFR-4: Maintainability

**Priority**: MEDIUM

**Requirements**:
- **NFR-4.1**: Clean architecture (interface-based)
- **NFR-4.2**: Comprehensive documentation
- **NFR-4.3**: Unit tests (30+ tests)
- **NFR-4.4**: Integration tests (10+ tests)
- **NFR-4.5**: Example templates

---

## 3. User Stories

### US-1: SRE Customizes Slack Notifications

**As a** SRE engineer
**I want to** customize Slack notification format using templates
**So that** I can include relevant alert information in a readable format

**Acceptance Criteria**:
- ✅ Can use Go template syntax in slack_configs.title
- ✅ Can use Go template syntax in slack_configs.text
- ✅ Can access alert labels, annotations, status
- ✅ Can use template functions (date, humanize, etc.)
- ✅ Invalid templates show clear error messages

**Example**:
```yaml
slack_configs:
  - title: "🔥 {{ .GroupLabels.alertname }} - {{ .Status | toUpper }}"
    text: |
      *Severity*: {{ .Labels.severity | default "unknown" }}
      *Instance*: {{ .Labels.instance }}
      *Started*: {{ .StartsAt | humanizeTimestamp }}
      {{ if .Annotations.runbook_url }}
      📖 [Runbook]({{ .Annotations.runbook_url }})
      {{ end }}
```

---

### US-2: DevOps Engineer Migrates from Alertmanager

**As a** DevOps engineer
**I want to** use existing Alertmanager templates
**So that** I can migrate without rewriting notification configs

**Acceptance Criteria**:
- ✅ Alertmanager template syntax works as-is
- ✅ All Alertmanager functions supported
- ✅ Migration guide available
- ✅ Backward compatibility maintained

---

### US-3: Platform Engineer Monitors Template Performance

**As a** Platform engineer
**I want to** monitor template execution performance
**So that** I can identify slow or failing templates

**Acceptance Criteria**:
- ✅ Prometheus metrics available
- ✅ Can track parse/execute duration
- ✅ Can track cache hit ratio
- ✅ Can track error rate per template

---

## 4. Technical Constraints

### TC-1: Compatibility

- **TC-1.1**: Must support Alertmanager template syntax
- **TC-1.2**: Must use Go text/template (NOT html/template)
- **TC-1.3**: Must be backward compatible (non-template strings work)

### TC-2: Performance

- **TC-2.1**: Template execution must not block notification sending
- **TC-2.2**: Cache must be memory-efficient (< 50MB)
- **TC-2.3**: Must support 1000+ concurrent template executions

### TC-3: Security

- **TC-3.1**: No arbitrary code execution (templates sandboxed)
- **TC-3.2**: Timeout protection (5s max per execution)
- **TC-3.3**: No access to filesystem or network from templates

---

## 5. Out of Scope

- ❌ HTML templates (use text/template only)
- ❌ Custom template functions from config (security risk)
- ❌ Template inheritance/includes (keep simple)
- ❌ Template versioning (use config versioning)
- ❌ Template storage in database (config file only)

---

## 6. Dependencies

### Internal Dependencies
- ✅ **TN-152**: Hot Reload (config reload triggers cache invalidation)
- ✅ **TN-150**: Config Update API (template validation)
- ✅ **TN-137-141**: Routing Engine (receiver configs)

### External Dependencies
- Go text/template (stdlib)
- github.com/Masterminds/sprig (template functions)
- github.com/hashicorp/golang-lru (LRU cache)

---

## 7. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Template Parse Time | < 10ms p95 | Benchmark |
| Template Execute Time (cached) | < 5ms p95 | Benchmark |
| Template Execute Time (uncached) | < 20ms p95 | Benchmark |
| Cache Hit Ratio | > 95% | Prometheus |
| Test Coverage | > 90% | go test -cover |
| Function Count | 50+ | Code review |
| Documentation | 100% | Manual review |

---

## 8. Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Template execution timeout | High | Medium | Context timeout (5s), fallback to raw |
| Memory leak from cache | High | Low | LRU cache with size limit (1000) |
| Incompatible with Alertmanager | High | Low | Use sprig functions, test compatibility |
| Performance degradation | Medium | Medium | Aggressive caching, benchmarks |
| Security vulnerabilities | High | Low | Sandboxed execution, no filesystem access |

---

## 9. Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 0: Planning | 1h | requirements.md, design.md, tasks.md |
| Phase 1: Core Engine | 2-3h | Template engine, cache, functions |
| Phase 2: Integration | 2-3h | Receiver integration, formatter updates |
| Phase 3: Testing | 2-3h | Unit tests, integration tests, benchmarks |
| Phase 4: Documentation | 1-2h | User guide, examples, API docs |
| **TOTAL** | **8-12h** | **Production-ready template engine** |

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Status**: ✅ APPROVED - Ready for Implementation
