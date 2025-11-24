# TN-154: Default Templates - Technical Design

**Task ID**: TN-154
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Date**: 2025-11-22

---

## 🏗️ Architecture Overview

### Component Structure

```
go-app/internal/notification/template/defaults/
├── slack.go          # Slack template constants and helpers
├── pagerduty.go      # PagerDuty template constants and helpers
├── email.go          # Email template constants and helpers
├── defaults.go       # Template registry and loader
└── defaults_test.go  # Comprehensive tests
```

### Integration with TN-153

```
┌─────────────────────────────────────────────────────────┐
│                    Notification Flow                     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│              TN-154: Default Templates                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │
│  │    Slack     │  │  PagerDuty   │  │    Email     │  │
│  │  Templates   │  │  Templates   │  │  Templates   │  │
│  └──────────────┘  └──────────────┘  └──────────────┘  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│           TN-153: Template Engine                        │
│  • Parse templates                                       │
│  • Execute with TemplateData                             │
│  • Cache compiled templates                              │
│  • 50+ Sprig functions                                   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│              Receiver Configurations                     │
│  • SlackConfig                                           │
│  • PagerDutyConfig                                       │
│  • EmailConfig                                           │
└─────────────────────────────────────────────────────────┘
```

---

## 📐 Design Decisions

### D-1: Template Storage Format

**Decision**: Store templates as Go string constants

**Rationale**:
- ✅ No external file dependencies
- ✅ Compile-time validation
- ✅ Easy to version control
- ✅ Fast loading (no I/O)
- ✅ Type-safe access

**Alternative Considered**: External YAML/JSON files
- ❌ Runtime file I/O
- ❌ Deployment complexity
- ❌ No compile-time checks

### D-2: Template Organization

**Decision**: Separate file per receiver type

**Rationale**:
- ✅ Clear separation of concerns
- ✅ Easy to maintain
- ✅ Modular structure
- ✅ Independent testing

### D-3: Template Variants

**Decision**: Provide multiple variants per receiver

**Rationale**:
- ✅ Different use cases (critical, warning, info)
- ✅ Single vs multi-alert scenarios
- ✅ Detailed vs summary views
- ✅ User choice and flexibility

### D-4: Color Mapping

**Decision**: Dynamic severity-based colors

**Rationale**:
- ✅ Visual distinction
- ✅ Alertmanager compatibility
- ✅ Industry standard

**Mapping**:
```
critical → danger (red)
error    → danger (red)
warning  → warning (yellow)
info     → good (green)
resolved → good (green)
```

### D-5: Email HTML Design

**Decision**: Inline CSS, responsive design

**Rationale**:
- ✅ Maximum email client compatibility
- ✅ Works without external resources
- ✅ Mobile-friendly
- ✅ Professional appearance

---

## 📋 Template Specifications

### Slack Templates

#### 1. Default Slack Title
```go
const DefaultSlackTitle = `{{ if eq .Status "resolved" }}✅ RESOLVED{{ else }}🔥 ALERT{{ end }}: {{ .GroupLabels.alertname | title }}`
```

**Features**:
- Status emoji (🔥 firing, ✅ resolved)
- Alert name from group labels
- Title case formatting

#### 2. Default Slack Text
```go
const DefaultSlackText = `{{ if gt (len .Alerts) 1 }}*{{ len .Alerts }} alerts* in this group{{ else }}{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}{{ end }}`
```

**Features**:
- Multi-alert count
- Single alert summary
- Conditional rendering

#### 3. Default Slack Fields
```go
const DefaultSlackFieldsTemplate = `[
  {"title": "Severity", "value": "{{ .CommonLabels.severity | upper }}", "short": true},
  {"title": "Environment", "value": "{{ .CommonLabels.environment | default "unknown" }}", "short": true},
  {"title": "Instance", "value": "{{ .CommonLabels.instance }}", "short": false},
  {"title": "Description", "value": "{{ .CommonAnnotations.description }}", "short": false}
]`
```

**Features**:
- Structured key-value pairs
- Short/long field support
- Default values for missing labels

#### 4. Color Mapping Function
```go
func GetSlackColor(severity string) string {
    switch strings.ToLower(severity) {
    case "critical", "error":
        return "danger"
    case "warning":
        return "warning"
    case "info":
        return "good"
    default:
        return "#439FE0" // Default blue
    }
}
```

---

### PagerDuty Templates

#### 1. Default PagerDuty Description
```go
const DefaultPagerDutyDescription = `{{ if eq .Status "resolved" }}[RESOLVED] {{ end }}{{ .GroupLabels.alertname }}: {{ .CommonAnnotations.summary | truncate 100 }}`
```

**Features**:
- Status prefix for resolved
- Alert name
- Summary (truncated to 100 chars)
- Total < 1024 chars (API limit)

#### 2. Default PagerDuty Details
```go
const DefaultPagerDutyDetailsTemplate = `{
  "alert_count": "{{ len .Alerts }}",
  "severity": "{{ .CommonLabels.severity }}",
  "environment": "{{ .CommonLabels.environment }}",
  "cluster": "{{ .CommonLabels.cluster }}",
  "instance": "{{ .CommonLabels.instance }}",
  "description": "{{ .CommonAnnotations.description }}",
  "runbook_url": "{{ .CommonAnnotations.runbook_url }}",
  "dashboard_url": "{{ .CommonAnnotations.dashboard_url }}",
  "generator_url": "{{ .GeneratorURL }}"
}`
```

**Features**:
- Comprehensive context
- Links to runbooks/dashboards
- Alert count
- All relevant labels

#### 3. Severity Mapping Function
```go
func GetPagerDutySeverity(alertSeverity string) string {
    switch strings.ToLower(alertSeverity) {
    case "critical":
        return "critical"
    case "error":
        return "error"
    case "warning":
        return "warning"
    default:
        return "info"
    }
}
```

---

### Email Templates

#### 1. Default Email Subject
```go
const DefaultEmailSubject = `{{ if eq .Status "resolved" }}[RESOLVED]{{ else }}[ALERT]{{ end }} {{ .GroupLabels.alertname }} ({{ len .Alerts }} alert{{ if gt (len .Alerts) 1 }}s{{ end }})`
```

**Features**:
- Status prefix
- Alert name
- Alert count with pluralization

#### 2. Default Email HTML
```go
const DefaultEmailHTML = `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Alert Notification</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; max-width: 800px; margin: 0 auto; padding: 20px; }
        .header { background: {{ if eq .Status "resolved" }}#28a745{{ else }}#dc3545{{ end }}; color: white; padding: 20px; border-radius: 5px; margin-bottom: 20px; }
        .alert-table { width: 100%; border-collapse: collapse; margin: 20px 0; }
        .alert-table th { background: #f8f9fa; padding: 12px; text-align: left; border-bottom: 2px solid #dee2e6; }
        .alert-table td { padding: 12px; border-bottom: 1px solid #dee2e6; }
        .severity-critical { color: #dc3545; font-weight: bold; }
        .severity-warning { color: #ffc107; font-weight: bold; }
        .severity-info { color: #17a2b8; }
        .footer { margin-top: 30px; padding-top: 20px; border-top: 1px solid #dee2e6; font-size: 12px; color: #6c757d; }
    </style>
</head>
<body>
    <div class="header">
        <h1>{{ if eq .Status "resolved" }}✅ Alerts Resolved{{ else }}🔥 Alert Notification{{ end }}</h1>
        <p><strong>{{ .GroupLabels.alertname }}</strong></p>
        <p>{{ len .Alerts }} alert{{ if gt (len .Alerts) 1 }}s{{ end }} • {{ .Status | upper }}</p>
    </div>

    <h2>Alert Details</h2>
    <table class="alert-table">
        <thead>
            <tr>
                <th>Alert</th>
                <th>Severity</th>
                <th>Instance</th>
                <th>Description</th>
            </tr>
        </thead>
        <tbody>
            {{ range .Alerts }}
            <tr>
                <td><strong>{{ .Labels.alertname }}</strong></td>
                <td class="severity-{{ .Labels.severity }}">{{ .Labels.severity | upper }}</td>
                <td>{{ .Labels.instance }}</td>
                <td>{{ .Annotations.description }}</td>
            </tr>
            {{ end }}
        </tbody>
    </table>

    <h2>Common Labels</h2>
    <ul>
        {{ range $key, $value := .CommonLabels }}
        <li><strong>{{ $key }}:</strong> {{ $value }}</li>
        {{ end }}
    </ul>

    {{ if .CommonAnnotations.runbook_url }}
    <p><a href="{{ .CommonAnnotations.runbook_url }}" style="display: inline-block; padding: 10px 20px; background: #007bff; color: white; text-decoration: none; border-radius: 5px;">View Runbook</a></p>
    {{ end }}

    <div class="footer">
        <p>Generated by Alertmanager++ OSS</p>
        <p>Receiver: {{ .Receiver }}</p>
        {{ if .ExternalURL }}<p>Alertmanager: <a href="{{ .ExternalURL }}">{{ .ExternalURL }}</a></p>{{ end }}
    </div>
</body>
</html>`
```

**Features**:
- Responsive design
- Status-based header color
- Alert table with all details
- Common labels section
- Runbook link button
- Professional footer

#### 3. Default Email Text (Plain Text Fallback)
```go
const DefaultEmailText = `{{ if eq .Status "resolved" }}[RESOLVED]{{ else }}[ALERT]{{ end }} {{ .GroupLabels.alertname }}

{{ len .Alerts }} alert{{ if gt (len .Alerts) 1 }}s{{ end }} - {{ .Status | upper }}

ALERTS:
{{ range .Alerts }}
- {{ .Labels.alertname }} ({{ .Labels.severity | upper }})
  Instance: {{ .Labels.instance }}
  Description: {{ .Annotations.description }}
{{ end }}

COMMON LABELS:
{{ range $key, $value := .CommonLabels }}
- {{ $key }}: {{ $value }}
{{ end }}

{{ if .CommonAnnotations.runbook_url }}
Runbook: {{ .CommonAnnotations.runbook_url }}
{{ end }}

---
Generated by Alertmanager++ OSS
Receiver: {{ .Receiver }}
{{ if .ExternalURL }}Alertmanager: {{ .ExternalURL }}{{ end }}`
```

---

## 🔧 Implementation Details

### Template Registry Structure

```go
package defaults

import (
    "github.com/vitaliisemenov/alert-history/go-app/internal/notification/template"
)

// TemplateRegistry holds all default templates
type TemplateRegistry struct {
    Slack      *SlackTemplates
    PagerDuty  *PagerDutyTemplates
    Email      *EmailTemplates
}

// SlackTemplates contains Slack default templates
type SlackTemplates struct {
    Title       string
    Text        string
    Pretext     string
    Fields      string
    ColorFunc   func(severity string) string
}

// PagerDutyTemplates contains PagerDuty default templates
type PagerDutyTemplates struct {
    Description   string
    Details       string
    SeverityFunc  func(severity string) string
}

// EmailTemplates contains Email default templates
type EmailTemplates struct {
    Subject string
    HTML    string
    Text    string
}

// GetDefaultTemplates returns the default template registry
func GetDefaultTemplates() *TemplateRegistry {
    return &TemplateRegistry{
        Slack: &SlackTemplates{
            Title:     DefaultSlackTitle,
            Text:      DefaultSlackText,
            Pretext:   DefaultSlackPretext,
            Fields:    DefaultSlackFieldsTemplate,
            ColorFunc: GetSlackColor,
        },
        PagerDuty: &PagerDutyTemplates{
            Description:  DefaultPagerDutyDescription,
            Details:      DefaultPagerDutyDetailsTemplate,
            SeverityFunc: GetPagerDutySeverity,
        },
        Email: &EmailTemplates{
            Subject: DefaultEmailSubject,
            HTML:    DefaultEmailHTML,
            Text:    DefaultEmailText,
        },
    }
}
```

### Helper Functions

```go
// ApplySlackDefaults applies default templates to SlackConfig if fields are empty
func ApplySlackDefaults(config *routing.SlackConfig, registry *TemplateRegistry) {
    if config.Title == "" {
        config.Title = registry.Slack.Title
    }
    if config.Text == "" {
        config.Text = registry.Slack.Text
    }
    if config.Color == "" {
        // Color will be determined at runtime based on severity
        config.Color = "{{ .CommonLabels.severity | slackColor }}"
    }
}

// ApplyPagerDutyDefaults applies default templates to PagerDutyConfig
func ApplyPagerDutyDefaults(config *routing.PagerDutyConfig, registry *TemplateRegistry) {
    if config.Description == "" {
        config.Description = registry.PagerDuty.Description
    }
    if config.Details == nil {
        // Parse details template at runtime
        config.Details = make(map[string]string)
    }
}

// ApplyEmailDefaults applies default templates to EmailConfig
func ApplyEmailDefaults(config *routing.EmailConfig, registry *TemplateRegistry) {
    if config.Subject == "" {
        config.Subject = registry.Email.Subject
    }
    if config.HTML == "" {
        config.HTML = registry.Email.HTML
    }
    if config.Text == "" {
        config.Text = registry.Email.Text
    }
}
```

---

## 🧪 Testing Strategy

### Unit Tests

```go
func TestDefaultSlackTitle(t *testing.T) {
    tests := []struct {
        name     string
        data     *template.TemplateData
        expected string
    }{
        {
            name: "firing alert",
            data: &template.TemplateData{
                Status: "firing",
                GroupLabels: map[string]string{"alertname": "HighCPU"},
            },
            expected: "🔥 ALERT: HighCPU",
        },
        {
            name: "resolved alert",
            data: &template.TemplateData{
                Status: "resolved",
                GroupLabels: map[string]string{"alertname": "HighCPU"},
            },
            expected: "✅ RESOLVED: HighCPU",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := template.NewNotificationTemplateEngine(...)
            result, err := engine.Execute(context.Background(), DefaultSlackTitle, tt.data)
            assert.NoError(t, err)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Integration Tests

```go
func TestSlackTemplateIntegration(t *testing.T) {
    // Create template engine
    engine := template.NewNotificationTemplateEngine(...)

    // Create sample alert data
    data := createSampleAlertData()

    // Get default templates
    registry := GetDefaultTemplates()

    // Execute all Slack templates
    title, err := engine.Execute(ctx, registry.Slack.Title, data)
    assert.NoError(t, err)
    assert.NotEmpty(t, title)

    text, err := engine.Execute(ctx, registry.Slack.Text, data)
    assert.NoError(t, err)
    assert.NotEmpty(t, text)

    // Verify size limits
    assert.Less(t, len(title)+len(text), 3000, "Slack message too large")
}
```

### Visual Tests

Generate sample outputs for manual review:

```go
func TestGenerateSampleOutputs(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping sample output generation")
    }

    // Generate samples for documentation
    generateSlackSamples(t)
    generatePagerDutySamples(t)
    generateEmailSamples(t)
}
```

---

## 📊 Performance Considerations

### Template Caching

- All default templates are cached by TN-153 engine
- First execution: ~5-10ms (parse + execute)
- Subsequent executions: < 1ms (cache hit)

### Memory Usage

- Template constants: ~50KB total
- Compiled templates: ~100KB in cache
- Total memory footprint: < 200KB

### Execution Time

Target: < 10ms per template execution

Optimization strategies:
- Simple templates (no complex logic)
- Minimal loops
- Pre-computed values where possible

---

## 🔄 Migration Path

### From Alertmanager

Users migrating from Alertmanager can:

1. **Use defaults as-is**: Templates are Alertmanager-compatible
2. **Copy existing templates**: Syntax is identical
3. **Mix and match**: Use defaults for some fields, custom for others

### Example Migration

**Before (Alertmanager)**:
```yaml
slack_configs:
  - api_url: "${SLACK_WEBHOOK}"
    title: "{{ .GroupLabels.alertname }}"
    text: "{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}"
```

**After (Alertmanager++ with defaults)**:
```yaml
slack_configs:
  - api_url: "${SLACK_WEBHOOK}"
    # title and text use defaults automatically
```

---

## 📝 Documentation Plan

### 1. Package README
- Overview of default templates
- Quick start guide
- Integration examples

### 2. Template Reference
- Complete list of all templates
- Template variables reference
- Function reference

### 3. Customization Guide
- How to override defaults
- Common customization patterns
- Best practices

### 4. Visual Examples
- Screenshots of Slack messages
- PagerDuty incident examples
- Email screenshots (desktop/mobile)

---

## ✅ Implementation Checklist

### Phase 1: Slack Templates
- [ ] Define Slack template constants
- [ ] Implement color mapping function
- [ ] Create SlackTemplates struct
- [ ] Write unit tests
- [ ] Generate sample outputs

### Phase 2: PagerDuty Templates
- [ ] Define PagerDuty template constants
- [ ] Implement severity mapping function
- [ ] Create PagerDutyTemplates struct
- [ ] Write unit tests
- [ ] Generate sample outputs

### Phase 3: Email Templates
- [ ] Define Email template constants (HTML + Text)
- [ ] Test HTML in multiple email clients
- [ ] Create EmailTemplates struct
- [ ] Write unit tests
- [ ] Generate sample outputs

### Phase 4: Template Registry
- [ ] Implement TemplateRegistry
- [ ] Create GetDefaultTemplates()
- [ ] Add helper functions (ApplyDefaults)
- [ ] Write integration tests

### Phase 5: Documentation
- [ ] Write package README
- [ ] Create template reference
- [ ] Write customization guide
- [ ] Add visual examples
- [ ] Document migration path

---

**Status**: ✅ Design Complete
**Next**: Implementation
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
