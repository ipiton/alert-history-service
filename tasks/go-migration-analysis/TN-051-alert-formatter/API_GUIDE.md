# Alert Formatter - API Integration Guide

**Version**: 1.0
**Date**: 2025-11-08
**Audience**: Backend Engineers, Platform Engineers, DevOps
**Estimated Reading Time**: 15 minutes

---

## 📑 Table of Contents

1. [Quick Start (5 minutes)](#1-quick-start-5-minutes)
2. [API Overview](#2-api-overview)
3. [Format Guide](#3-format-guide)
4. [Code Examples](#4-code-examples)
5. [Error Handling](#5-error-handling)
6. [Best Practices](#6-best-practices)
7. [Troubleshooting](#7-troubleshooting)
8. [Performance Tuning](#8-performance-tuning)

---

## 1. Quick Start (5 minutes)

### 1.1 Installation

**Already Installed**: Alert Formatter is part of the Publishing System in `go-app/internal/infrastructure/publishing/`

### 1.2 Basic Usage (3 lines)

```go
import "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"

formatter := publishing.NewAlertFormatter()
result, _ := formatter.FormatAlert(ctx, enrichedAlert, core.FormatRootly)
```

### 1.3 Complete Example

```go
package main

import (
    "context"
    "encoding/json"
    "log"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/core"
    "github.com/vitaliisemenov/alert-history/internal/infrastructure/publishing"
)

func main() {
    // 1. Create formatter
    formatter := publishing.NewAlertFormatter()

    // 2. Prepare enriched alert
    alert := &core.EnrichedAlert{
        Alert: core.Alert{
            Fingerprint: "abc123def456",
            AlertName:   "HighCPUUsage",
            Status:      core.AlertStatusFiring,
            Labels: map[string]string{
                "severity": "critical",
                "instance": "prod-web-01",
            },
            Annotations: map[string]string{
                "description": "CPU usage above 90%",
                "runbook_url": "https://wiki.company.com/runbooks/high-cpu",
            },
            StartsAt: time.Now(),
        },
        Classification: &core.ClassificationResult{
            Severity:    core.AlertSeverityCritical,
            Confidence:  0.95,
            Reasoning:   "High CPU usage on production web server",
            ActionItems: []string{"Check processes", "Scale if needed"},
        },
    }

    // 3. Format for Rootly
    ctx := context.Background()
    result, err := formatter.FormatAlert(ctx, alert, core.FormatRootly)
    if err != nil {
        log.Fatalf("formatting failed: %v", err)
    }

    // 4. Print result
    jsonBytes, _ := json.MarshalIndent(result, "", "  ")
    log.Printf("Formatted for Rootly:\n%s", string(jsonBytes))
}
```

**Output**:
```json
{
  "title": "[CRITICAL] HighCPUUsage",
  "description": "**Alert**: HighCPUUsage\n**Status**: firing\n...",
  "severity": "critical",
  "tags": ["severity:critical", "instance:prod-web-01"],
  "custom_fields": {
    "fingerprint": "abc123def456",
    "ai_classification": {...}
  }
}
```

---

## 2. API Overview

### 2.1 AlertFormatter Interface

```go
type AlertFormatter interface {
    // FormatAlert transforms an enriched alert into the specified format
    FormatAlert(ctx context.Context, enrichedAlert *core.EnrichedAlert, format core.PublishingFormat) (map[string]interface{}, error)
}
```

**Parameters**:
- `ctx`: Context for cancellation and tracing
- `enrichedAlert`: Alert with LLM classification (or nil classification)
- `format`: Target format (alertmanager, rootly, pagerduty, slack, webhook)

**Returns**:
- `map[string]interface{}`: Formatted alert (ready for JSON marshaling)
- `error`: Formatting error or nil on success

### 2.2 Supported Formats

| Format | Constant | Target System | Documentation |
|--------|----------|---------------|---------------|
| **Alertmanager** | `core.FormatAlertmanager` | Prometheus Alertmanager | [Webhook v4](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config) |
| **Rootly** | `core.FormatRootly` | Rootly Incident Management | [API Docs](https://rootly.com/api-reference) |
| **PagerDuty** | `core.FormatPagerDuty` | PagerDuty Incident Response | [Events API v2](https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgw-events-api-v2) |
| **Slack** | `core.FormatSlack` | Slack Workspace | [Block Kit](https://api.slack.com/block-kit) |
| **Webhook** | `core.FormatWebhook` | Generic HTTP Endpoint | N/A (simple JSON) |

### 2.3 EnrichedAlert Structure

```go
type EnrichedAlert struct {
    Alert          core.Alert
    Classification *core.ClassificationResult  // Optional, can be nil
    EnrichedAt     time.Time
}

type Alert struct {
    Fingerprint  string            // Required: unique alert ID
    AlertName    string            // Required: alert name
    Status       AlertStatus       // Required: firing or resolved
    Labels       map[string]string // Optional: key-value labels
    Annotations  map[string]string // Optional: key-value annotations
    StartsAt     time.Time         // Required: alert start time
    EndsAt       *time.Time        // Optional: alert end time (resolved only)
    GeneratorURL *string           // Optional: source URL
}

type ClassificationResult struct {
    Severity        AlertSeverity  // Critical, High, Medium, Low, Info
    Confidence      float64        // 0.0 to 1.0
    Reasoning       string         // LLM explanation
    ActionItems     []string       // Recommended actions
    AffectedSystems []string       // Affected systems
    EstimatedImpact string         // Impact description
}
```

---

## 3. Format Guide

### 3.1 Alertmanager Format

**Use Case**: Forward alerts to Prometheus Alertmanager

**Output Structure**:
```json
{
  "receiver": "alerthistory",
  "status": "firing",
  "alerts": [{
    "status": "firing",
    "labels": {
      "alertname": "HighCPUUsage",
      "severity": "critical",
      "instance": "prod-web-01"
    },
    "annotations": {
      "description": "CPU usage above 90%",
      "ai_severity": "critical",
      "ai_confidence": "0.95",
      "ai_reasoning": "High CPU usage on production web server"
    },
    "startsAt": "2025-11-08T10:00:00Z",
    "endsAt": "0001-01-01T00:00:00Z",
    "generatorURL": "",
    "fingerprint": "abc123def456"
  }],
  "groupLabels": {
    "alertname": "HighCPUUsage"
  },
  "commonLabels": {
    "alertname": "HighCPUUsage",
    "severity": "critical"
  },
  "commonAnnotations": {
    "description": "CPU usage above 90%"
  }
}
```

**LLM Integration**: Classification data injected into `annotations` (ai_severity, ai_confidence, ai_reasoning, ai_recommendations)

**Example**:
```go
result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatAlertmanager)
```

---

### 3.2 Rootly Format

**Use Case**: Create incidents in Rootly

**Output Structure**:
```json
{
  "title": "[CRITICAL] HighCPUUsage",
  "description": "**Alert**: HighCPUUsage\n**Status**: firing\n**Severity**: critical (AI confidence: 95%)\n**Reasoning**: High CPU usage on production web server\n\n**Recommended Actions**:\n- Check processes\n- Scale if needed\n\n**Labels**:\n- severity: critical\n- instance: prod-web-01",
  "severity": "critical",
  "tags": ["severity:critical", "instance:prod-web-01"],
  "custom_fields": {
    "fingerprint": "abc123def456",
    "starts_at": "2025-11-08T10:00:00Z",
    "generator_url": "",
    "ai_classification": {
      "severity": "critical",
      "confidence": 0.95,
      "reasoning": "High CPU usage on production web server",
      "action_items": ["Check processes", "Scale if needed"]
    }
  }
}
```

**Severity Mapping**:
- `critical` → Rootly "critical"
- `high` → Rootly "high"
- `medium` → Rootly "medium"
- `low` → Rootly "low"
- `info` → Rootly "low"

**LLM Integration**: AI severity in title, reasoning in description, full classification in custom_fields

**Example**:
```go
result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatRootly)
```

---

### 3.3 PagerDuty Format

**Use Case**: Trigger incidents in PagerDuty

**Output Structure**:
```json
{
  "routing_key": "<service_integration_key>",
  "event_action": "trigger",
  "dedup_key": "abc123def456",
  "payload": {
    "summary": "[CRITICAL] HighCPUUsage",
    "source": "AlertHistory",
    "severity": "critical",
    "timestamp": "2025-11-08T10:00:00Z",
    "custom_details": {
      "alert_name": "HighCPUUsage",
      "status": "firing",
      "labels": {
        "severity": "critical",
        "instance": "prod-web-01"
      },
      "annotations": {
        "description": "CPU usage above 90%"
      },
      "ai_classification": {
        "severity": "critical",
        "confidence": 0.95,
        "reasoning": "High CPU usage on production web server"
      }
    }
  },
  "links": [{
    "href": "",
    "text": "Generator URL"
  }]
}
```

**Severity Mapping**:
- `critical` → PagerDuty "critical"
- `high` → PagerDuty "error"
- `medium` → PagerDuty "warning"
- `low` → PagerDuty "info"
- `info` → PagerDuty "info"

**Event Actions**:
- Alert status "firing" → `event_action: "trigger"`
- Alert status "resolved" → `event_action: "resolve"`

**LLM Integration**: Full classification in `custom_details.ai_classification`

**Example**:
```go
result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatPagerDuty)
```

---

### 3.4 Slack Format

**Use Case**: Send rich notifications to Slack

**Output Structure** (Blocks API):
```json
{
  "attachments": [{
    "color": "#FF0000",
    "blocks": [
      {
        "type": "header",
        "text": {
          "type": "plain_text",
          "text": "🚨 Alert: HighCPUUsage"
        }
      },
      {
        "type": "section",
        "fields": [
          {"type": "mrkdwn", "text": "*Status:*\nfiring"},
          {"type": "mrkdwn", "text": "*Severity:*\ncritical"},
          {"type": "mrkdwn", "text": "*Confidence:*\n95%"}
        ]
      },
      {
        "type": "section",
        "text": {
          "type": "mrkdwn",
          "text": "*AI Analysis:*\nHigh CPU usage on production web server"
        }
      },
      {
        "type": "section",
        "text": {
          "type": "mrkdwn",
          "text": "*Recommended Actions:*\n• Check processes\n• Scale if needed"
        }
      }
    ]
  }]
}
```

**Color Mapping**:
- `critical` → Red (#FF0000)
- `high` → Orange (#FF9900)
- `medium` → Yellow (#FFCC00)
- `low` → Blue (#0099FF)
- `info` → Gray (#808080)

**Emoji Mapping**:
- `critical` → 🚨
- `high` → ⚠️
- `medium` → ℹ️
- `low` → 💡
- `info` → 📊

**LLM Integration**: AI analysis and recommended actions as separate blocks

**Example**:
```go
result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatSlack)
```

---

### 3.5 Webhook Format

**Use Case**: Generic HTTP webhooks

**Output Structure**:
```json
{
  "fingerprint": "abc123def456",
  "alert_name": "HighCPUUsage",
  "status": "firing",
  "labels": {
    "severity": "critical",
    "instance": "prod-web-01"
  },
  "annotations": {
    "description": "CPU usage above 90%"
  },
  "starts_at": "2025-11-08T10:00:00Z",
  "ends_at": null,
  "generator_url": "",
  "classification": {
    "severity": "critical",
    "confidence": 0.95,
    "reasoning": "High CPU usage on production web server",
    "action_items": ["Check processes", "Scale if needed"]
  },
  "enriched_at": "2025-11-08T10:05:00Z"
}
```

**LLM Integration**: Full classification as top-level field

**Example**:
```go
result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatWebhook)
```

---

## 4. Code Examples

### 4.1 Format for All Targets

```go
func formatForAllTargets(ctx context.Context, enrichedAlert *core.EnrichedAlert) {
    formatter := publishing.NewAlertFormatter()

    formats := []core.PublishingFormat{
        core.FormatAlertmanager,
        core.FormatRootly,
        core.FormatPagerDuty,
        core.FormatSlack,
        core.FormatWebhook,
    }

    for _, format := range formats {
        result, err := formatter.FormatAlert(ctx, enrichedAlert, format)
        if err != nil {
            log.Printf("Format %s failed: %v", format, err)
            continue
        }

        log.Printf("Format %s success: %d bytes", format, len(result))
    }
}
```

### 4.2 Handle Context Cancellation

```go
func formatWithTimeout(enrichedAlert *core.EnrichedAlert) error {
    formatter := publishing.NewAlertFormatter()

    // Create context with 5-second timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    result, err := formatter.FormatAlert(ctx, enrichedAlert, core.FormatRootly)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            return fmt.Errorf("formatting timeout")
        }
        return err
    }

    log.Printf("Formatted: %v", result)
    return nil
}
```

### 4.3 Fallback to Webhook on Unknown Format

```go
func formatWithFallback(ctx context.Context, enrichedAlert *core.EnrichedAlert, format core.PublishingFormat) (map[string]interface{}, error) {
    formatter := publishing.NewAlertFormatter()

    result, err := formatter.FormatAlert(ctx, enrichedAlert, format)
    if err != nil {
        // Fallback to webhook format
        log.Printf("Format %s failed, falling back to webhook: %v", format, err)
        return formatter.FormatAlert(ctx, enrichedAlert, core.FormatWebhook)
    }

    return result, nil
}
```

### 4.4 Handle Nil Classification

```go
func formatWithoutClassification(ctx context.Context) {
    formatter := publishing.NewAlertFormatter()

    // Alert without LLM classification
    alert := &core.EnrichedAlert{
        Alert: core.Alert{
            Fingerprint: "abc123",
            AlertName:   "TestAlert",
            Status:      core.AlertStatusFiring,
            StartsAt:    time.Now(),
        },
        Classification: nil,  // No LLM classification
    }

    // Will still format, but without AI data
    result, err := formatter.FormatAlert(ctx, alert, core.FormatRootly)
    if err != nil {
        log.Fatalf("Failed: %v", err)
    }

    log.Printf("Formatted without classification: %v", result)
}
```

### 4.5 Batch Formatting

```go
func batchFormat(ctx context.Context, alerts []*core.EnrichedAlert, format core.PublishingFormat) ([]map[string]interface{}, error) {
    formatter := publishing.NewAlertFormatter()
    results := make([]map[string]interface{}, 0, len(alerts))

    for _, alert := range alerts {
        result, err := formatter.FormatAlert(ctx, alert, format)
        if err != nil {
            log.Printf("Alert %s formatting failed: %v", alert.Alert.Fingerprint, err)
            continue  // Skip failed alerts
        }
        results = append(results, result)
    }

    return results, nil
}
```

---

## 5. Error Handling

### 5.1 Error Types

```go
const (
    ErrNilAlert          = "nil enriched alert"
    ErrInvalidFormat     = "unsupported publishing format"
    ErrFormattingFailed  = "formatting failed"
    ErrContextCancelled  = "context cancelled"
)
```

### 5.2 Error Handling Patterns

**Pattern 1: Log and Continue**
```go
result, err := formatter.FormatAlert(ctx, alert, format)
if err != nil {
    log.Printf("Formatting failed: %v", err)
    return nil  // Continue with other alerts
}
```

**Pattern 2: Fallback to Webhook**
```go
result, err := formatter.FormatAlert(ctx, alert, format)
if err != nil {
    result, err = formatter.FormatAlert(ctx, alert, core.FormatWebhook)
}
```

**Pattern 3: Fail Fast**
```go
result, err := formatter.FormatAlert(ctx, alert, format)
if err != nil {
    return fmt.Errorf("critical: formatting failed: %w", err)
}
```

**Pattern 4: Retry with Exponential Backoff**
```go
func formatWithRetry(ctx context.Context, alert *core.EnrichedAlert, format core.PublishingFormat) (map[string]interface{}, error) {
    formatter := publishing.NewAlertFormatter()
    maxRetries := 3
    baseDelay := 100 * time.Millisecond

    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := formatter.FormatAlert(ctx, alert, format)
        if err == nil {
            return result, nil
        }

        if attempt < maxRetries-1 {
            delay := baseDelay * time.Duration(1<<uint(attempt))
            log.Printf("Retry %d/%d after %v: %v", attempt+1, maxRetries, delay, err)
            time.Sleep(delay)
        }
    }

    return nil, fmt.Errorf("formatting failed after %d retries", maxRetries)
}
```

---

## 6. Best Practices

### 6.1 Always Use Context

✅ **Good**:
```go
ctx := context.Background()
result, err := formatter.FormatAlert(ctx, alert, format)
```

❌ **Bad**:
```go
result, err := formatter.FormatAlert(nil, alert, format)  // Will panic
```

### 6.2 Handle Nil Classification Gracefully

✅ **Good** (formatter handles nil):
```go
// Formatter handles nil classification automatically
result, err := formatter.FormatAlert(ctx, alert, format)
// Will format without AI data if classification is nil
```

❌ **Bad** (panics):
```go
// Don't access classification fields without nil check
severity := alert.Classification.Severity  // PANIC if nil
```

### 6.3 Marshal Carefully

✅ **Good** (check error):
```go
jsonBytes, err := json.Marshal(result)
if err != nil {
    return fmt.Errorf("JSON marshaling failed: %w", err)
}
```

❌ **Bad** (ignore error):
```go
jsonBytes, _ := json.Marshal(result)  // Could be nil
http.Post(url, "application/json", bytes.NewReader(jsonBytes))
```

### 6.4 Use Appropriate Format for Target

✅ **Good**:
```go
// Slack → core.FormatSlack (Blocks API)
result, err := formatter.FormatAlert(ctx, alert, core.FormatSlack)
```

❌ **Bad**:
```go
// Don't use webhook format for Slack (won't render blocks)
result, err := formatter.FormatAlert(ctx, alert, core.FormatWebhook)
```

### 6.5 Don't Modify Result Map

✅ **Good** (copy if needed):
```go
result, _ := formatter.FormatAlert(ctx, alert, format)
resultCopy := make(map[string]interface{})
for k, v := range result {
    resultCopy[k] = v
}
resultCopy["custom_field"] = "value"
```

❌ **Bad** (mutate original):
```go
result, _ := formatter.FormatAlert(ctx, alert, format)
result["custom_field"] = "value"  // May affect internal state
```

---

## 7. Troubleshooting

### 7.1 "nil enriched alert" Error

**Cause**: Passed nil alert to FormatAlert

**Solution**:
```go
if enrichedAlert == nil {
    return fmt.Errorf("cannot format nil alert")
}
result, err := formatter.FormatAlert(ctx, enrichedAlert, format)
```

### 7.2 "unsupported publishing format" Error

**Cause**: Used unknown format constant

**Solution**:
```go
// Use only these 5 formats
validFormats := []core.PublishingFormat{
    core.FormatAlertmanager,
    core.FormatRootly,
    core.FormatPagerDuty,
    core.FormatSlack,
    core.FormatWebhook,
}
```

### 7.3 JSON Marshaling Issues

**Symptom**: `json.Marshal(result)` returns error

**Cause**: Result contains unmarshalable types (channels, functions)

**Solution**: This should never happen with formatter output. If it does, it's a bug.

### 7.4 Slack Blocks Not Rendering

**Symptom**: Slack shows plain text instead of blocks

**Cause 1**: Used wrong format (webhook instead of slack)

**Solution**:
```go
// Use core.FormatSlack
result, _ := formatter.FormatAlert(ctx, alert, core.FormatSlack)
```

**Cause 2**: Sent to Slack without proper structure

**Solution**:
```go
// Slack expects: { "attachments": [...] }
jsonBytes, _ := json.Marshal(result)
http.Post(slackWebhookURL, "application/json", bytes.NewReader(jsonBytes))
```

### 7.5 PagerDuty Incident Not Created

**Symptom**: No incident appears in PagerDuty

**Cause**: Missing or invalid `routing_key`

**Solution**:
```go
result, _ := formatter.FormatAlert(ctx, alert, core.FormatPagerDuty)

// Must set routing_key before sending
result["routing_key"] = "<your_integration_key>"

jsonBytes, _ := json.Marshal(result)
http.Post("https://events.pagerduty.com/v2/enqueue", "application/json", bytes.NewReader(jsonBytes))
```

---

## 8. Performance Tuning

### 8.1 Benchmark Results

**Typical Latencies** (from existing tests):
- **Alertmanager**: ~5ms (baseline)
- **Rootly**: ~7ms (baseline, more string operations)
- **PagerDuty**: ~4ms (baseline)
- **Slack**: ~12ms (baseline, complex blocks)
- **Webhook**: ~2ms (baseline, simplest)

**150% Target**: <500μs (10x improvement, requires Phase 4 optimizations)

### 8.2 Optimization Tips

**Tip 1: Reuse Formatter Instance**
```go
// ✅ Good: Create once, reuse
var formatter = publishing.NewAlertFormatter()

func handleAlerts(alerts []*core.EnrichedAlert) {
    for _, alert := range alerts {
        formatter.FormatAlert(ctx, alert, format)
    }
}

// ❌ Bad: Create every time
func handleAlerts(alerts []*core.EnrichedAlert) {
    for _, alert := range alerts {
        formatter := publishing.NewAlertFormatter()  // Wasteful
        formatter.FormatAlert(ctx, alert, format)
    }
}
```

**Tip 2: Batch Process Alerts**
```go
// Process 100 alerts at once
func batchProcess(alerts []*core.EnrichedAlert) {
    results := make([]map[string]interface{}, 0, len(alerts))
    for _, alert := range alerts {
        result, err := formatter.FormatAlert(ctx, alert, format)
        if err == nil {
            results = append(results, result)
        }
    }
    // Send batch to target system
}
```

**Tip 3: Concurrent Formatting (Thread-safe)**
```go
func concurrentFormat(alerts []*core.EnrichedAlert, format core.PublishingFormat) {
    var wg sync.WaitGroup
    formatter := publishing.NewAlertFormatter()  // Thread-safe

    for _, alert := range alerts {
        wg.Add(1)
        go func(a *core.EnrichedAlert) {
            defer wg.Done()
            formatter.FormatAlert(context.Background(), a, format)
        }(alert)
    }

    wg.Wait()
}
```

**Tip 4: Pool Result Maps (Advanced)**
```go
var resultPool = sync.Pool{
    New: func() interface{} {
        return make(map[string]interface{}, 32)  // Pre-allocate
    },
}

func formatOptimized(ctx context.Context, alert *core.EnrichedAlert, format core.PublishingFormat) map[string]interface{} {
    result := resultPool.Get().(map[string]interface{})
    defer resultPool.Put(result)

    // Use result...
    formatter.FormatAlert(ctx, alert, format)
    return result
}
```

### 8.3 Monitoring Performance

**Use Prometheus Metrics** (Phase 6 enhancement):
```go
// Duration histogram
publishing_formatter_duration_seconds{format="rootly"} 0.005

// Total counter
publishing_formatter_total{format="rootly",status="success"} 1234

// Error counter
publishing_formatter_errors_total{format="rootly",error_type="validation"} 5
```

**Alerting Rules**:
```yaml
# Alert if p99 latency > 1ms
- alert: FormatterSlowP99
  expr: histogram_quantile(0.99, publishing_formatter_duration_seconds) > 0.001
  for: 5m
  annotations:
    summary: "Alert formatter p99 latency > 1ms"
```

---

## Document Metadata

**Version**: 1.0
**Author**: AI Assistant (TN-051 API Integration Guide)
**Date**: 2025-11-08
**Target Audience**: Backend Engineers, Platform Engineers, DevOps
**Estimated Reading Time**: 15 minutes

**Related Documentation**:
- [Requirements](./requirements.md) - Functional/non-functional requirements
- [Design](./design.md) - Technical architecture and patterns
- [Tasks](./tasks.md) - Implementation roadmap
- [Completion Report](./COMPLETION_REPORT.md) - Final achievements

**Support**:
- **Implementation**: `go-app/internal/infrastructure/publishing/formatter.go`
- **Tests**: `go-app/internal/infrastructure/publishing/formatter_test.go`
- **Questions**: Platform Team Slack channel #alert-history

---

**🚀 Ready to integrate? Start with the [Quick Start](#1-quick-start-5-minutes)!**
