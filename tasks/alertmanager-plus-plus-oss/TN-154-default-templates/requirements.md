# TN-154: Default Templates - Requirements

**Task ID**: TN-154
**Sprint**: Sprint 3 (Week 3) - Config & Templates
**Priority**: High
**Complexity**: Medium
**Estimate**: 6-8 hours
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Dependencies**: TN-153 (Template Engine) ✅
**Date**: 2025-11-22

---

## 📋 Overview

Create production-ready default templates for Slack, PagerDuty, and Email notification receivers. These templates should provide a comprehensive, out-of-the-box experience that follows best practices and is compatible with Alertmanager's template system.

### Goals

1. **Slack Templates**: Rich, visually appealing messages with structured fields
2. **PagerDuty Templates**: Detailed incident descriptions with context
3. **Email Templates**: Professional HTML emails with full alert details
4. **Alertmanager Compatibility**: 100% compatible with Alertmanager template syntax
5. **Production Ready**: Tested, documented, and ready for immediate use

---

## 🎯 Functional Requirements

### FR-1: Slack Default Templates

**Priority**: P0 (Critical)

#### FR-1.1: Slack Message Structure
- **Title**: Clear, concise alert summary with severity indicator
- **Text**: Main message body with alert details
- **Pretext**: Optional context above the message
- **Fields**: Structured key-value pairs for quick scanning
- **Color**: Dynamic color based on alert severity (good/warning/danger)
- **Actions**: Optional interactive buttons (future)

#### FR-1.2: Slack Template Fields
Must support templating for:
- `title`: Alert name and status
- `text`: Alert descriptions and summaries
- `pretext`: Group-level context
- `fields[].title`: Field labels
- `fields[].value`: Field values
- `color`: Severity-based coloring

#### FR-1.3: Slack Template Examples
Provide templates for:
- **Critical Alerts**: Red color, urgent tone, detailed context
- **Warning Alerts**: Yellow color, informative tone, key metrics
- **Info Alerts**: Blue/green color, minimal details
- **Resolved Alerts**: Green color, resolution confirmation

---

### FR-2: PagerDuty Default Templates

**Priority**: P0 (Critical)

#### FR-2.1: PagerDuty Incident Structure
- **Description**: Clear incident summary (max 1024 chars)
- **Details**: Structured key-value pairs for incident context
- **Severity**: Dynamic severity mapping (critical/error/warning/info)
- **Class/Component/Group**: Optional categorization

#### FR-2.2: PagerDuty Template Fields
Must support templating for:
- `description`: Incident summary
- `details`: Map of contextual information
- `severity`: Alert severity
- `class`, `component`, `group`: Categorization

#### FR-2.3: PagerDuty Template Examples
Provide templates for:
- **Critical Incidents**: Detailed description, full context
- **Error Incidents**: Standard description, key metrics
- **Warning Incidents**: Brief description, minimal context

---

### FR-3: Email Default Templates

**Priority**: P0 (Critical)

#### FR-3.1: Email Structure
- **Subject**: Clear, scannable subject line with alert count
- **HTML Body**: Professional, responsive HTML template
- **Text Body**: Plain text fallback for email clients
- **Headers**: Proper MIME headers and encoding

#### FR-3.2: Email Template Fields
Must support templating for:
- `subject`: Email subject line
- `html`: HTML email body
- `text`: Plain text email body (optional)
- `to`: Recipient addresses
- `from`: Sender address

#### FR-3.3: Email Template Features
- **Responsive Design**: Works on desktop and mobile
- **Alert Table**: Structured table of all alerts in group
- **Color Coding**: Severity-based visual indicators
- **Links**: Direct links to alert sources and dashboards
- **Grouping**: Clear separation of firing vs resolved alerts

#### FR-3.4: Email Template Examples
Provide templates for:
- **Multi-Alert Summary**: Table of all alerts in group
- **Single Alert Detail**: Detailed view of one alert
- **Resolved Notification**: Confirmation of alert resolution

---

## 🔧 Non-Functional Requirements

### NFR-1: Template Performance

**Target**: < 10ms template execution per notification

- Templates must be optimized for fast rendering
- No complex loops or nested templates
- Cache-friendly (static structure, dynamic data)

### NFR-2: Template Size

**Limits**:
- Slack templates: < 3000 chars (Slack API limit)
- PagerDuty description: < 1024 chars (API limit)
- Email HTML: < 100KB (reasonable email size)

### NFR-3: Alertmanager Compatibility

**100% Compatible** with Alertmanager template syntax:
- Use `.Alerts`, `.GroupLabels`, `.CommonLabels`, `.CommonAnnotations`
- Support `range`, `if`, `with` control structures
- Use Sprig functions (from TN-153)
- Follow Alertmanager naming conventions

### NFR-4: Customizability

**Easy to Customize**:
- Well-documented template variables
- Clear structure and formatting
- Modular components (can override parts)
- Examples of common customizations

### NFR-5: Internationalization

**English Default**:
- All default templates in English
- Structure supports future i18n
- No hardcoded locale-specific formatting

---

## 📊 Template Data Model

All templates have access to the following data structure (from TN-153):

```go
type TemplateData struct {
    // Alert is the current alert being processed
    Alert *Alert

    // Labels are the alert's labels
    Labels map[string]string

    // Annotations are the alert's annotations
    Annotations map[string]string

    // GroupLabels are the labels used for grouping
    GroupLabels map[string]string

    // CommonLabels are labels common to all alerts in group
    CommonLabels map[string]string

    // CommonAnnotations are annotations common to all alerts
    CommonAnnotations map[string]string

    // ExternalURL is the Alertmanager external URL
    ExternalURL string

    // GeneratorURL is the source of the alert
    GeneratorURL string

    // Status is "firing" or "resolved"
    Status string

    // Receiver is the receiver name
    Receiver string
}
```

---

## 🎨 Template Design Principles

### 1. **Clarity First**
- Alert name and status immediately visible
- Key information highlighted
- No unnecessary details

### 2. **Actionable Information**
- Include links to dashboards/runbooks
- Show relevant metrics and thresholds
- Provide context for decision-making

### 3. **Consistent Structure**
- All templates follow similar layout
- Predictable information hierarchy
- Easy to scan quickly

### 4. **Severity Awareness**
- Visual indicators for severity
- Tone appropriate to urgency
- Critical alerts stand out

### 5. **Group-Aware**
- Show alert count in group
- Distinguish between single and multi-alert
- Summarize common attributes

---

## 📦 Deliverables

### 1. Template Files

Create Go package with default templates:

```
go-app/internal/notification/template/defaults/
├── slack.go          # Slack default templates
├── pagerduty.go      # PagerDuty default templates
├── email.go          # Email default templates
├── defaults.go       # Template registry and loader
└── defaults_test.go  # Template tests
```

### 2. Template Constants

Each file defines template constants:
- `DefaultSlackTitle`
- `DefaultSlackText`
- `DefaultSlackFields`
- `DefaultPagerDutyDescription`
- `DefaultPagerDutyDetails`
- `DefaultEmailSubject`
- `DefaultEmailHTML`

### 3. Template Registry

Centralized registry for loading templates:
```go
type TemplateRegistry struct {
    Slack      SlackTemplates
    PagerDuty  PagerDutyTemplates
    Email      EmailTemplates
}

func GetDefaultTemplates() *TemplateRegistry
```

### 4. Documentation

- **README.md**: Overview and quick start
- **EXAMPLES.md**: Template examples with sample output
- **CUSTOMIZATION.md**: Guide for customizing templates

### 5. Tests

- Unit tests for all templates
- Integration tests with template engine (TN-153)
- Validation tests (size limits, syntax)
- Output examples for visual review

---

## ✅ Acceptance Criteria

### AC-1: Slack Templates
- ✅ Default templates for all Slack fields
- ✅ Severity-based color mapping
- ✅ Structured fields for key information
- ✅ Support for single and multi-alert groups
- ✅ < 3000 chars per message
- ✅ Renders correctly in Slack UI

### AC-2: PagerDuty Templates
- ✅ Default description template
- ✅ Default details template (key-value pairs)
- ✅ Severity mapping
- ✅ < 1024 chars description
- ✅ All required PagerDuty fields populated

### AC-3: Email Templates
- ✅ Professional HTML template
- ✅ Plain text fallback
- ✅ Responsive design (mobile-friendly)
- ✅ Alert table with all details
- ✅ Severity-based color coding
- ✅ < 100KB HTML size

### AC-4: Quality
- ✅ All templates tested with template engine
- ✅ 100% Alertmanager compatibility
- ✅ Comprehensive documentation
- ✅ Examples for all templates
- ✅ Zero linter errors
- ✅ 90%+ test coverage

### AC-5: Integration
- ✅ Works with TN-153 template engine
- ✅ Compatible with existing receiver configs
- ✅ Easy to integrate into notification flow
- ✅ No breaking changes to existing code

---

## 🔗 Dependencies

### Upstream (Must Complete First)
- ✅ **TN-153**: Template Engine Integration (COMPLETED)

### Downstream (Blocked By This Task)
- **TN-155**: Template API (CRUD operations)
- **TN-156**: Template Validator

---

## 📈 Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Template Execution Time | < 10ms | Benchmark tests |
| Slack Message Size | < 3000 chars | Length validation |
| PagerDuty Description | < 1024 chars | Length validation |
| Email HTML Size | < 100KB | Size validation |
| Test Coverage | ≥ 90% | go test -cover |
| Documentation | 100% | All templates documented |
| Alertmanager Compatibility | 100% | Integration tests |

---

## 🎯 Quality Target: 150% (Grade A+ EXCEPTIONAL)

### Baseline (100%)
- ✅ All functional requirements met
- ✅ Templates work correctly
- ✅ Basic documentation

### Enhanced (125%)
- ✅ Comprehensive examples
- ✅ Multiple template variants
- ✅ Customization guide
- ✅ Performance optimized

### Exceptional (150%)
- ✅ Production-ready templates
- ✅ Extensive documentation
- ✅ Visual examples (screenshots)
- ✅ Integration tests
- ✅ Performance benchmarks
- ✅ Best practices guide
- ✅ Migration guide from Alertmanager

---

## 📝 Notes

- Templates should be usable as-is without customization
- Follow Alertmanager conventions for familiarity
- Consider future webhook integrations (Teams, Discord)
- Email templates should work in all major email clients
- Templates should gracefully handle missing data

---

**Status**: ✅ Requirements Complete
**Next**: Design Phase
**Author**: AI Assistant
**Reviewed**: Pending
