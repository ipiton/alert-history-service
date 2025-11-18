# TN-146: Prometheus Alert Parser — Technical Design

> **Version**: 1.0
> **Status**: 🚧 IN PROGRESS
> **Target Quality**: **150% (Grade A+)**

---

## 📋 Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Component Design](#component-design)
3. [Data Structures](#data-structures)
4. [Algorithms](#algorithms)
5. [Integration Points](#integration-points)
6. [Error Handling](#error-handling)
7. [Performance Optimization](#performance-optimization)
8. [Testing Strategy](#testing-strategy)

---

## 🏗️ Architecture Overview

### System Context

```
┌─────────────────────────────────────────────────────────────────┐
│                     Alert Ingestion Pipeline                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ┌───────────────┐        ┌──────────────────┐                  │
│  │  Prometheus   │───────▶│ POST /api/v2/    │                  │
│  │  (native)     │        │      alerts      │                  │
│  └───────────────┘        └──────────────────┘                  │
│                                    │                              │
│  ┌───────────────┐                 │                              │
│  │  Alertmanager │───────▶        │                              │
│  │  (webhook)    │                 │                              │
│  └───────────────┘                 ▼                              │
│                          ┌────────────────────┐                  │
│                          │ UniversalWebhook   │◀─── TN-42        │
│                          │     Handler        │                  │
│                          └────────────────────┘                  │
│                                    │                              │
│                    ┌───────────────┼───────────────┐             │
│                    │               │               │             │
│                    ▼               ▼               ▼             │
│         ┌─────────────────┐  ┌─────────┐  ┌──────────────┐     │
│         │ WebhookDetector │  │ Parser  │  │  Validator   │     │
│         │   (TN-146 ✨)   │  │(TN-146) │  │   (TN-43)    │     │
│         └─────────────────┘  └─────────┘  └──────────────┘     │
│                    │               │               │             │
│                    │               │               │             │
│                    ▼               ▼               ▼             │
│              ┌──────────────────────────────────────┐           │
│              │     Alertmanager → Prometheus       │           │
│              │       Format Normalizer             │           │
│              └──────────────────────────────────────┘           │
│                              │                                   │
│                              ▼                                   │
│                  ┌────────────────────────┐                     │
│                  │   core.Alert           │◀─── TN-31          │
│                  │   (Domain Model)       │                     │
│                  └────────────────────────┘                     │
│                              │                                   │
│              ┌───────────────┼───────────────┐                  │
│              ▼               ▼               ▼                  │
│      ┌───────────┐  ┌──────────────┐  ┌──────────┐            │
│      │ Grouping  │  │  Inhibition  │  │ Silencing│            │
│      │  Engine   │  │    Engine    │  │  Engine  │            │
│      │  (TN-121) │  │   (TN-126)   │  │ (TN-131) │            │
│      └───────────┘  └──────────────┘  └──────────┘            │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Design Principles

1. **Adapter Pattern**: Prometheus Parser adapts Prometheus format → UniversalWebhookHandler interface
2. **Strategy Pattern**: Multiple parsers (Alertmanager, Prometheus) selected by detector
3. **Single Responsibility**: Each component has one job (detect, parse, validate, convert)
4. **Open/Closed**: Easy to add new formats без изменения existing code
5. **Dependency Inversion**: Depend on interfaces (WebhookParser), not implementations

---

## 🧩 Component Design

### 1. Format Detector Enhancement

**File**: `go-app/internal/infrastructure/webhook/detector.go` (enhance existing)

**Current State**:
```go
// Already exists in TN-42
const (
    WebhookTypeAlertmanager WebhookType = "alertmanager"
    WebhookTypeGeneric      WebhookType = "generic"
    WebhookTypePrometheus   WebhookType = "prometheus" // ✨ NEW
)
```

**Enhancement**:
```go
// Add Prometheus sub-types
const (
    PrometheusFormatV1 = "prometheus_v1"  // Array format
    PrometheusFormatV2 = "prometheus_v2"  // Grouped format
)

// PrometheusFormatDetector для fine-grained detection
type PrometheusFormatDetector interface {
    DetectPrometheusFormat(payload []byte) (string, error)
}

type prometheusFormatDetector struct{}

func (d *prometheusFormatDetector) DetectPrometheusFormat(payload []byte) (string, error) {
    var data interface{}
    if err := json.Unmarshal(payload, &data); err != nil {
        return "", fmt.Errorf("invalid JSON: %w", err)
    }

    switch v := data.(type) {
    case []interface{}:
        // Array → likely Prometheus v1
        if len(v) > 0 {
            if hasPrometheusV1Fields(v[0]) {
                return PrometheusFormatV1, nil
            }
        }
    case map[string]interface{}:
        // Object → could be v2 or Alertmanager
        if hasField(v, "alerts") && hasField(v, "groupLabels") {
            return PrometheusFormatV2, nil
        }
        if hasField(v, "version") && hasField(v, "groupKey") {
            return "alertmanager", nil // Delegate to existing parser
        }
    }

    return "", fmt.Errorf("unknown Prometheus format")
}

func hasPrometheusV1Fields(alert interface{}) bool {
    m, ok := alert.(map[string]interface{})
    if !ok {
        return false
    }
    // Prometheus-specific indicators:
    // - "state" field (vs "status" in Alertmanager)
    // - "activeAt" field (vs "startsAt" in Alertmanager)
    // - "generatorURL" is required (optional in Alertmanager)
    return hasField(m, "state") &&
           hasField(m, "activeAt") &&
           hasField(m, "labels")
}
```

**Rationale**: Need fine-grained detection для выбора правильного parser logic.

---

### 2. Prometheus Data Models

**File**: `go-app/internal/infrastructure/webhook/prometheus_models.go` (NEW)

```go
package webhook

import "time"

// PrometheusAlert represents a single alert in Prometheus format.
//
// This structure is compatible with both:
//  - Prometheus API v1 (legacy /api/v1/alerts)
//  - Prometheus API v2 (modern /api/v2/alerts)
//
// Key differences from Alertmanager format:
//  - "state" instead of "status" ("firing" | "pending" | "inactive")
//  - "activeAt" instead of "startsAt" (when alert became active)
//  - "generatorURL" is required (always present in Prometheus)
//  - "value" field contains the alert's metric value (optional)
type PrometheusAlert struct {
    // Labels contains all alert labels (including alertname).
    // REQUIRED: Must contain at least "alertname" label.
    //
    // Example:
    //   {
    //     "alertname": "HighCPU",
    //     "instance": "server-1:9100",
    //     "job": "node-exporter",
    //     "severity": "warning"
    //   }
    Labels map[string]string `json:"labels" validate:"required,dive,keys,required,endkeys,required"`

    // Annotations contains alert annotations (descriptions, runbooks, etc).
    // Optional: Can be empty or nil.
    //
    // Example:
    //   {
    //     "summary": "CPU usage is above 80%",
    //     "description": "Instance server-1 has CPU > 80% for 5 minutes",
    //     "runbook_url": "https://wiki.example.com/runbooks/high-cpu"
    //   }
    Annotations map[string]string `json:"annotations"`

    // State represents the current alert state.
    // REQUIRED: Must be one of "firing", "pending", "inactive".
    //
    // States:
    //  - "firing": Alert condition is true and alert is active
    //  - "pending": Alert condition is true but waiting for "for" duration
    //  - "inactive": Alert condition is false (equivalent to "resolved")
    State string `json:"state" validate:"required,oneof=firing pending inactive"`

    // ActiveAt is the timestamp when the alert first became active.
    // REQUIRED: Must be valid RFC3339 timestamp.
    //
    // Note: This is different from Alertmanager's "startsAt" which can be
    // updated on alert re-firing. "activeAt" remains constant.
    ActiveAt time.Time `json:"activeAt" validate:"required"`

    // Value contains the alert's metric value at evaluation time.
    // Optional: May be empty for non-threshold alerts.
    //
    // Example: "0.85" for CPU usage 85%
    Value string `json:"value,omitempty"`

    // GeneratorURL is the URL to the Prometheus expression browser for this alert.
    // REQUIRED in Prometheus format (always present).
    //
    // Example: "http://prometheus:9090/graph?g0.expr=up%7Bjob%3D%22api%22%7D+%3D%3D+0"
    GeneratorURL string `json:"generatorURL" validate:"required,url"`

    // Fingerprint is a unique identifier for this alert based on labels.
    // Optional: Will be generated if not provided (SHA256 of sorted labels).
    //
    // Format: Hex-encoded SHA256 hash (64 characters)
    Fingerprint string `json:"fingerprint,omitempty" validate:"omitempty,hexadecimal,len=64"`
}

// PrometheusAlertGroup represents a group of alerts in Prometheus v2 format.
//
// Prometheus v2 API returns alerts grouped by common labels to reduce payload size.
// This structure is only used when parsing /api/v2/alerts responses.
type PrometheusAlertGroup struct {
    // Labels contains the common labels shared by all alerts in this group.
    // These labels are NOT duplicated in individual alert Labels.
    //
    // Example:
    //   {
    //     "job": "api-server",
    //     "severity": "warning"
    //   }
    Labels map[string]string `json:"labels"`

    // Alerts contains all alerts in this group.
    // The group Labels are implicitly added to each alert during parsing.
    Alerts []PrometheusAlert `json:"alerts" validate:"required,min=1,dive"`
}

// PrometheusWebhook represents the top-level Prometheus webhook payload.
//
// This structure supports both formats:
//  - v1: Direct array of alerts in "Alerts" field
//  - v2: Grouped alerts in "Groups" field
//
// Exactly one of Alerts or Groups must be non-empty.
type PrometheusWebhook struct {
    // Alerts contains alerts in v1 format (direct array).
    // Used when parsing /api/v1/alerts responses.
    //
    // If non-empty, Groups must be empty.
    Alerts []PrometheusAlert `json:"alerts,omitempty" validate:"required_without=Groups,dive"`

    // Groups contains alerts in v2 format (grouped by labels).
    // Used when parsing /api/v2/alerts responses.
    //
    // If non-empty, Alerts must be empty.
    Groups []PrometheusAlertGroup `json:"groups,omitempty" validate:"required_without=Alerts,dive"`
}

// AlertCount returns the total number of alerts in the webhook.
func (w *PrometheusWebhook) AlertCount() int {
    if len(w.Alerts) > 0 {
        return len(w.Alerts)
    }

    count := 0
    for _, group := range w.Groups {
        count += len(group.Alerts)
    }
    return count
}

// FlattenAlerts returns all alerts as a flat array.
// For v2 format, merges group labels into each alert.
func (w *PrometheusWebhook) FlattenAlerts() []PrometheusAlert {
    if len(w.Alerts) > 0 {
        return w.Alerts // v1: already flat
    }

    // v2: flatten groups
    var flattened []PrometheusAlert
    for _, group := range w.Groups {
        for _, alert := range group.Alerts {
            // Merge group labels into alert labels
            merged := make(map[string]string)
            for k, v := range group.Labels {
                merged[k] = v
            }
            for k, v := range alert.Labels {
                merged[k] = v // Alert labels override group labels
            }
            alert.Labels = merged
            flattened = append(flattened, alert)
        }
    }
    return flattened
}
```

**Key Design Decisions**:

1. **Separate Structure**: PrometheusAlert != AlertmanagerAlert (разные поля)
2. **Validation Tags**: Struct tags для automatic validation
3. **Flatten Method**: Упрощает конвертацию v2 → v1 → core.Alert
4. **Fingerprint Optional**: Generate if missing (как в TN-41)

---

### 3. Prometheus Parser

**File**: `go-app/internal/infrastructure/webhook/prometheus_parser.go` (NEW)

```go
package webhook

import (
    "crypto/sha256"
    "encoding/json"
    "fmt"
    "sort"
    "strings"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/core"
)

// prometheusParser implements WebhookParser for Prometheus alerts.
//
// This parser handles both Prometheus v1 and v2 formats:
//  - v1: Direct array of alerts
//  - v2: Grouped alerts with shared labels
//
// The parser normalizes both formats to core.Alert domain model.
type prometheusParser struct {
    validator       WebhookValidator
    formatDetector  PrometheusFormatDetector
}

// NewPrometheusParser creates a new Prometheus alert parser.
//
// Returns:
//   - WebhookParser: Initialized parser with validator and format detector
func NewPrometheusParser() WebhookParser {
    return &prometheusParser{
        validator:      NewWebhookValidator(),
        formatDetector: &prometheusFormatDetector{},
    }
}

// Parse parses raw JSON bytes into PrometheusWebhook structure.
//
// This method handles both Prometheus v1 and v2 formats automatically.
// The format is detected based on payload structure.
//
// Parameters:
//   - data: Raw JSON bytes from webhook request body
//
// Returns:
//   - *AlertmanagerWebhook: Parsed webhook (converted for interface compatibility)
//   - error: JSON parsing error or validation error
func (p *prometheusParser) Parse(data []byte) (*AlertmanagerWebhook, error) {
    if len(data) == 0 {
        return nil, fmt.Errorf("prometheus webhook payload is empty")
    }

    // Detect Prometheus format (v1 or v2)
    format, err := p.formatDetector.DetectPrometheusFormat(data)
    if err != nil {
        return nil, fmt.Errorf("failed to detect Prometheus format: %w", err)
    }

    // Parse based on detected format
    var webhook PrometheusWebhook
    if err := json.Unmarshal(data, &webhook); err != nil {
        return nil, fmt.Errorf("failed to parse Prometheus webhook JSON: %w", err)
    }

    // Validate structure
    if webhook.AlertCount() == 0 {
        return nil, fmt.Errorf("prometheus webhook contains no alerts")
    }

    // Convert to AlertmanagerWebhook for interface compatibility
    // This allows reusing existing validation and downstream processing
    amWebhook := p.convertToAlertmanagerFormat(&webhook, format)

    return amWebhook, nil
}

// Validate validates the parsed webhook using WebhookValidator.
//
// Parameters:
//   - webhook: Parsed AlertmanagerWebhook to validate
//
// Returns:
//   - *ValidationResult: Validation result with errors (if any)
func (p *prometheusParser) Validate(webhook *AlertmanagerWebhook) *ValidationResult {
    // Delegate to existing validator (TN-43)
    return p.validator.ValidateAlertmanager(webhook)
}

// ConvertToDomain converts PrometheusWebhook alerts to core.Alert domain models.
//
// This method performs:
//  1. Status mapping (state → Status)
//  2. Timestamp conversion (activeAt → StartsAt)
//  3. Fingerprint generation (if missing)
//  4. Field validation
//
// Parameters:
//   - webhook: Parsed AlertmanagerWebhook (converted from Prometheus format)
//
// Returns:
//   - []*core.Alert: Converted domain models
//   - error: Conversion error (missing required fields, invalid data)
func (p *prometheusParser) ConvertToDomain(webhook *AlertmanagerWebhook) ([]*core.Alert, error) {
    if webhook == nil {
        return nil, fmt.Errorf("webhook is nil")
    }

    alerts := make([]*core.Alert, 0, len(webhook.Alerts))

    for i, amAlert := range webhook.Alerts {
        alert, err := p.convertSingleAlert(&amAlert, i)
        if err != nil {
            return nil, fmt.Errorf("failed to convert alert[%d]: %w", i, err)
        }
        alerts = append(alerts, alert)
    }

    return alerts, nil
}

// convertToAlertmanagerFormat converts PrometheusWebhook → AlertmanagerWebhook.
//
// This conversion allows reusing existing validation and processing logic.
// The conversion is lossless - all fields are preserved.
func (p *prometheusParser) convertToAlertmanagerFormat(
    prometheus *PrometheusWebhook,
    format string,
) *AlertmanagerWebhook {
    // Flatten alerts (handles both v1 and v2)
    flatAlerts := prometheus.FlattenAlerts()

    // Convert each Prometheus alert → Alertmanager alert
    amAlerts := make([]AlertmanagerAlert, 0, len(flatAlerts))
    for _, promAlert := range flatAlerts {
        amAlert := AlertmanagerAlert{
            Status:       mapPrometheusState(promAlert.State),
            Labels:       promAlert.Labels,
            Annotations:  promAlert.Annotations,
            StartsAt:     promAlert.ActiveAt, // activeAt → startsAt
            EndsAt:       time.Time{},        // Not provided in Prometheus
            GeneratorURL: promAlert.GeneratorURL,
            Fingerprint:  promAlert.Fingerprint,
        }

        // Store original value in annotations
        if promAlert.Value != "" {
            if amAlert.Annotations == nil {
                amAlert.Annotations = make(map[string]string)
            }
            amAlert.Annotations["__prometheus_value__"] = promAlert.Value
        }

        amAlerts = append(amAlerts, amAlert)
    }

    // Create Alertmanager-compatible webhook
    return &AlertmanagerWebhook{
        Version:  "prom_" + format, // e.g. "prom_v1", "prom_v2"
        GroupKey: "prometheus",     // Fake group key
        Receiver: "prometheus",     // Fake receiver
        Status:   "firing",         // Assume firing for Prometheus
        Alerts:   amAlerts,
        GroupLabels:    make(map[string]string),
        CommonLabels:   make(map[string]string),
        CommonAnnotations: make(map[string]string),
        ExternalURL:    "",
    }
}

// convertSingleAlert converts a single AlertmanagerAlert to core.Alert.
func (p *prometheusParser) convertSingleAlert(amAlert *AlertmanagerAlert, index int) (*core.Alert, error) {
    // Extract alertname from labels (required)
    alertName, ok := amAlert.Labels["alertname"]
    if !ok || alertName == "" {
        return nil, fmt.Errorf("alert[%d]: missing required label 'alertname'", index)
    }

    // Map status to core.AlertStatus
    status, err := mapAlertStatus(amAlert.Status)
    if err != nil {
        return nil, fmt.Errorf("alert[%d]: %w", index, err)
    }

    // Generate or use existing fingerprint
    fingerprint := amAlert.Fingerprint
    if fingerprint == "" {
        fingerprint = generateFingerprint(alertName, amAlert.Labels)
    }

    // Validate timestamps
    if amAlert.StartsAt.IsZero() {
        return nil, fmt.Errorf("alert[%d]: startsAt is required", index)
    }

    // Convert EndsAt to pointer (only if not zero)
    var endsAt *time.Time
    if !amAlert.EndsAt.IsZero() {
        endsAt = &amAlert.EndsAt
    }

    // Convert GeneratorURL to pointer
    var generatorURL *string
    if amAlert.GeneratorURL != "" {
        generatorURL = &amAlert.GeneratorURL
    }

    // Set timestamp to current time
    now := time.Now()

    return &core.Alert{
        Fingerprint:  fingerprint,
        AlertName:    alertName,
        Status:       status,
        Labels:       amAlert.Labels,
        Annotations:  amAlert.Annotations,
        StartsAt:     amAlert.StartsAt,
        EndsAt:       endsAt,
        GeneratorURL: generatorURL,
        Timestamp:    &now,
    }, nil
}

// mapPrometheusState maps Prometheus state to Alertmanager status.
//
// Mapping:
//  - "firing"   → "firing"
//  - "pending"  → "firing" (treat pending as firing)
//  - "inactive" → "resolved"
func mapPrometheusState(state string) string {
    switch state {
    case "firing", "pending":
        return "firing"
    case "inactive":
        return "resolved"
    default:
        return "firing" // Default to firing
    }
}

// generateFingerprint generates a deterministic fingerprint for an alert.
//
// The fingerprint is generated from:
//   - alertname
//   - sorted labels (key=value pairs)
//
// Algorithm: SHA256(alertname|label1=value1|label2=value2|...)
//
// This ensures the same alert generates the same fingerprint consistently.
func generateFingerprint(alertName string, labels map[string]string) string {
    // Sort label keys for deterministic ordering
    keys := make([]string, 0, len(labels))
    for k := range labels {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // Build canonical string
    parts := []string{alertName}
    for _, k := range keys {
        parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
    }
    canonical := strings.Join(parts, "|")

    // SHA256 hash
    hash := sha256.Sum256([]byte(canonical))
    return fmt.Sprintf("%x", hash)
}
```

**Key Design Decisions**:

1. **Adapter Pattern**: Convert Prometheus → Alertmanager format для reuse existing code
2. **Lossless Conversion**: Store Prometheus-specific fields (value) in annotations
3. **Unified Fingerprint**: Same algorithm as TN-41 для consistency
4. **State Mapping**: "pending" → "firing" (most conservative approach)

---

### 4. Integration with UniversalWebhookHandler

**File**: `go-app/internal/infrastructure/webhook/handler.go` (MODIFY existing)

```go
// BEFORE (existing):
func NewUniversalWebhookHandler(processor AlertProcessor, logger *slog.Logger) *UniversalWebhookHandler {
    return &UniversalWebhookHandler{
        detector:  NewWebhookDetector(),
        parser:    NewAlertmanagerParser(), // ❌ Hard-coded
        validator: NewWebhookValidator(),
        processor: processor,
        metrics:   metrics.NewWebhookMetrics(),
        logger:    logger,
    }
}

// AFTER (enhanced):
type UniversalWebhookHandler struct {
    detector  WebhookDetector
    parsers   map[WebhookType]WebhookParser // ✨ Map of parsers
    validator WebhookValidator
    processor AlertProcessor
    metrics   *metrics.WebhookMetrics
    logger    *slog.Logger
}

func NewUniversalWebhookHandler(processor AlertProcessor, logger *slog.Logger) *UniversalWebhookHandler {
    if logger == nil {
        logger = slog.Default()
    }

    return &UniversalWebhookHandler{
        detector: NewWebhookDetector(),
        parsers: map[WebhookType]WebhookParser{
            WebhookTypeAlertmanager: NewAlertmanagerParser(), // Existing
            WebhookTypePrometheus:   NewPrometheusParser(),   // ✨ NEW
        },
        validator: NewWebhookValidator(),
        processor: processor,
        metrics:   metrics.NewWebhookMetrics(),
        logger:    logger,
    }
}

func (h *UniversalWebhookHandler) HandleWebhook(ctx context.Context, req *HandleWebhookRequest) (*HandleWebhookResponse, error) {
    startTime := time.Now()

    // ... (existing detection code) ...

    // Step 2: Select parser based on webhook type
    parser, ok := h.parsers[webhookType]
    if !ok {
        h.logger.Warn("Unknown webhook type, falling back to Alertmanager parser",
            "detected_type", webhookType)
        parser = h.parsers[WebhookTypeAlertmanager] // Fallback
    }

    // Step 3: Parse webhook
    parseStart := time.Now()
    webhook, err := parser.Parse(req.Payload)
    parseDuration := time.Since(parseStart).Seconds()
    h.metrics.RecordProcessingStage(string(webhookType), "parse", parseDuration)

    // ... (rest remains same) ...
}
```

**Rationale**: Strategy pattern для dynamic parser selection без hard-coding.

---

## 📊 Data Structures

### Core Domain Model (existing, no changes)

```go
// From TN-31 (go-app/internal/core/interfaces.go)
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    AlertName    string            `json:"alert_name" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required,oneof=firing resolved"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"starts_at" validate:"required"`
    EndsAt       *time.Time        `json:"ends_at,omitempty"`
    GeneratorURL *string           `json:"generator_url,omitempty" validate:"omitempty,url"`
    Timestamp    *time.Time        `json:"timestamp,omitempty"`
}

type AlertStatus string

const (
    StatusFiring   AlertStatus = "firing"
    StatusResolved AlertStatus = "resolved"
)
```

**No Changes Required**: Existing model perfectly supports Prometheus alerts after conversion.

---

### Intermediate Models

#### Prometheus → Alertmanager Conversion

```
PrometheusAlert          →  AlertmanagerAlert
├─ labels                →  labels
├─ annotations           →  annotations
├─ state (string)        →  status (string)
│  ├─ "firing"          →  "firing"
│  ├─ "pending"         →  "firing"
│  └─ "inactive"        →  "resolved"
├─ activeAt (time)       →  startsAt (time)
├─ value (string)        →  annotations["__prometheus_value__"]
├─ generatorURL (string) →  generatorURL (string)
└─ fingerprint (string)  →  fingerprint (string)
```

#### Alertmanager → core.Alert Conversion (existing)

```
AlertmanagerAlert  →  core.Alert
├─ labels          →  labels
├─ annotations     →  annotations
├─ status          →  Status
├─ startsAt        →  StartsAt
├─ endsAt          →  EndsAt
├─ generatorURL    →  GeneratorURL
└─ fingerprint     →  Fingerprint (generate if missing)
```

---

## 🔬 Algorithms

### Algorithm 1: Format Detection

```python
# Pseudo-code for clarity
def detect_format(payload: bytes) -> Format:
    """
    Detect webhook format from payload structure.

    Returns: "prometheus_v1" | "prometheus_v2" | "alertmanager" | "unknown"
    """
    # Step 1: Parse JSON
    try:
        data = json.loads(payload)
    except JSONDecodeError:
        return "unknown"

    # Step 2: Check structure type
    if isinstance(data, list):
        # Array structure → likely Prometheus v1
        if len(data) > 0 and has_prometheus_v1_fields(data[0]):
            return "prometheus_v1"

    elif isinstance(data, dict):
        # Object structure → v2 or Alertmanager

        # Check for Alertmanager webhook indicators
        if all(key in data for key in ["version", "groupKey", "receiver"]):
            return "alertmanager"

        # Check for Prometheus v2 indicators
        if "alerts" in data and "groupLabels" in data:
            return "prometheus_v2"

        # Check for Prometheus v1 wrapped in object
        if "alerts" in data and isinstance(data["alerts"], list):
            if len(data["alerts"]) > 0:
                if has_prometheus_v1_fields(data["alerts"][0]):
                    return "prometheus_v1"

    return "unknown"

def has_prometheus_v1_fields(alert: dict) -> bool:
    """
    Check if alert has Prometheus v1 signature fields.

    Prometheus indicators:
    - "state" field (vs "status" in Alertmanager)
    - "activeAt" field (vs "startsAt" in Alertmanager)
    - "generatorURL" is present (required in Prometheus)
    """
    return (
        "state" in alert and
        "activeAt" in alert and
        "labels" in alert and
        "generatorURL" in alert
    )
```

**Time Complexity**: O(1) - только top-level field checks
**Space Complexity**: O(1) - no allocations

---

### Algorithm 2: Fingerprint Generation

```go
// Same algorithm as TN-41 for consistency
func generateFingerprint(alertName string, labels map[string]string) string {
    // Time: O(n log n) where n = number of labels
    // Space: O(n)

    // Step 1: Sort label keys (O(n log n))
    keys := make([]string, 0, len(labels))
    for k := range labels {
        keys = append(keys, k)
    }
    sort.Strings(keys)

    // Step 2: Build canonical string (O(n))
    parts := []string{alertName}
    for _, k := range keys {
        parts = append(parts, fmt.Sprintf("%s=%s", k, labels[k]))
    }
    canonical := strings.Join(parts, "|")

    // Step 3: SHA256 hash (O(len(canonical)))
    hash := sha256.Sum256([]byte(canonical))
    return fmt.Sprintf("%x", hash)
}
```

**Example**:
```
Input: alertName="HighCPU", labels={"instance":"server-1", "job":"api", "severity":"warning"}
Canonical: "HighCPU|instance=server-1|job=api|severity=warning"
Hash: "a3f8b2c1..." (64 hex characters)
```

**Determinism**: ✅ Same labels → same fingerprint (critical для deduplication)

---

### Algorithm 3: Prometheus → Alertmanager Conversion

```python
def convert_prometheus_to_alertmanager(prom_webhook: PrometheusWebhook) -> AlertmanagerWebhook:
    """
    Convert Prometheus format to Alertmanager format for interface compatibility.

    Handles both v1 (flat) and v2 (grouped) formats.
    """
    # Step 1: Flatten alerts (handles v1 and v2)
    flat_alerts = prom_webhook.flatten_alerts()  # O(n) where n = total alerts

    # Step 2: Convert each alert (O(n))
    am_alerts = []
    for prom_alert in flat_alerts:
        am_alert = AlertmanagerAlert(
            status=map_state(prom_alert.state),        # "firing" | "resolved"
            labels=prom_alert.labels,
            annotations=prom_alert.annotations,
            startsAt=prom_alert.activeAt,
            endsAt=None,  # Not available in Prometheus
            generatorURL=prom_alert.generatorURL,
            fingerprint=prom_alert.fingerprint or generate_fingerprint(...)
        )

        # Store Prometheus-specific value in annotations
        if prom_alert.value:
            am_alert.annotations["__prometheus_value__"] = prom_alert.value

        am_alerts.append(am_alert)

    # Step 3: Create Alertmanager webhook structure
    return AlertmanagerWebhook(
        version="prom_v1",  # or "prom_v2"
        groupKey="prometheus",
        receiver="prometheus",
        status="firing",
        alerts=am_alerts,
        groupLabels={},
        commonLabels={},
        commonAnnotations={},
        externalURL=""
    )

def map_state(state: str) -> str:
    """Map Prometheus state to Alertmanager status."""
    if state in ["firing", "pending"]:
        return "firing"
    elif state == "inactive":
        return "resolved"
    else:
        return "firing"  # Conservative default
```

**Time Complexity**: O(n) where n = number of alerts
**Space Complexity**: O(n) - full copy of alerts

**Rationale**: Lossless conversion allows reusing all existing downstream logic.

---

## 🔗 Integration Points

### 1. UniversalWebhookHandler (TN-42)

**Integration Type**: Strategy Pattern

**Before**:
```go
handler := NewUniversalWebhookHandler(processor, logger)
// Only Alertmanager supported
```

**After**:
```go
handler := NewUniversalWebhookHandler(processor, logger)
// Now supports: Alertmanager ✅ + Prometheus ✅
```

**Changes Required**:
- ✅ Add `parsers map[WebhookType]WebhookParser` field
- ✅ Populate map with both parsers
- ✅ Dynamic parser selection in HandleWebhook()

**Risk**: LOW (backward compatible, existing tests still pass)

---

### 2. WebhookDetector (TN-42)

**Integration Type**: Enhancement

**Before**:
```go
const (
    WebhookTypeAlertmanager WebhookType = "alertmanager"
    WebhookTypeGeneric      WebhookType = "generic"
    WebhookTypePrometheus   WebhookType = "prometheus" // Defined but unused
)
```

**After**:
```go
// Enhance Detect() method to return WebhookTypePrometheus
func (d *webhookDetector) Detect(payload []byte) (WebhookType, error) {
    // ... existing Alertmanager detection ...

    // ✨ NEW: Prometheus detection
    if hasPrometheusFields(data) {
        return WebhookTypePrometheus, nil
    }

    // ... existing Generic fallback ...
}
```

**Risk**: LOW (additive change, no breaking changes)

---

### 3. AlertProcessor (TN-36, TN-35)

**Integration Type**: No changes required

**Flow**:
```
Prometheus Parser
     │
     ├─▶ Convert to core.Alert
     │
     ├─▶ Deduplication Service (TN-36) ✅ Works as-is
     │
     ├─▶ Filtering Engine (TN-35) ✅ Works as-is
     │
     └─▶ Storage (TN-32) ✅ Works as-is
```

**Rationale**: All downstream components работают с `core.Alert` — универсальная модель.

---

### 4. POST /api/v2/alerts Endpoint (TN-147)

**Integration Type**: New HTTP handler (next task)

**Preview**:
```go
// In TN-147 (next task)
func handlePrometheusAlerts(w http.ResponseWriter, r *http.Request) {
    // Read body
    payload, _ := io.ReadAll(r.Body)

    // Use Prometheus parser
    parser := webhook.NewPrometheusParser()
    webhook, err := parser.Parse(payload)

    // Convert to domain
    alerts, err := parser.ConvertToDomain(webhook)

    // Process alerts
    for _, alert := range alerts {
        processor.ProcessAlert(r.Context(), alert)
    }

    // Respond with Prometheus-compatible format
    respondPrometheus(w, alerts)
}
```

---

## 🚨 Error Handling

### Error Hierarchy

```
ParserError (base)
├─ DetectionError
│  ├─ InvalidJSONError
│  ├─ UnknownFormatError
│  └─ EmptyPayloadError
├─ ParseError
│  ├─ JSONUnmarshalError
│  ├─ MissingFieldError
│  └─ InvalidStructureError
├─ ValidationError
│  ├─ MissingAlertNameError
│  ├─ InvalidStateError
│  ├─ InvalidTimestampError
│  └─ InvalidGeneratorURLError
└─ ConversionError
   ├─ FingerprintGenerationError
   └─ StatusMappingError
```

### Error Messages

```go
var (
    // Detection errors
    ErrEmptyPayload      = errors.New("prometheus webhook payload is empty")
    ErrInvalidJSON       = errors.New("invalid JSON in prometheus webhook")
    ErrUnknownFormat     = errors.New("unknown prometheus format")

    // Parse errors
    ErrNoAlerts          = errors.New("prometheus webhook contains no alerts")
    ErrInvalidStructure  = errors.New("invalid prometheus webhook structure")

    // Validation errors
    ErrMissingAlertName  = errors.New("missing required label 'alertname'")
    ErrInvalidState      = errors.New("invalid state, must be firing|pending|inactive")
    ErrMissingActiveAt   = errors.New("activeAt is required")
    ErrInvalidTimestamp  = errors.New("invalid RFC3339 timestamp")
    ErrInvalidGeneratorURL = errors.New("invalid generatorURL format")

    // Conversion errors
    ErrConversionFailed  = errors.New("failed to convert prometheus alert to core.Alert")
)
```

### Error Context

```go
type PrometheusParserError struct {
    Type    string      `json:"type"`
    Message string      `json:"message"`
    Field   string      `json:"field,omitempty"`
    Value   interface{} `json:"value,omitempty"`
    Index   int         `json:"index,omitempty"` // Alert index in array
    Err     error       `json:"-"`
}

func (e *PrometheusParserError) Error() string {
    if e.Index >= 0 {
        return fmt.Sprintf("%s at alert[%d] field '%s': %s", e.Type, e.Index, e.Field, e.Message)
    }
    return fmt.Sprintf("%s: %s", e.Type, e.Message)
}
```

**Example**:
```
parse_error at alert[2] field 'state': invalid state 'unknown', must be firing|pending|inactive
```

---

## ⚡ Performance Optimization

### Target Metrics

| Operation | Target | Strategy |
|-----------|--------|----------|
| Parse single alert | < 10µs | Zero-copy parsing, struct reuse |
| Parse 100 alerts | < 1ms | Batch processing, pre-allocation |
| Fingerprint generation | < 1µs | String builder, sort.Strings |
| Format detection | < 5µs | Field existence checks only |
| Memory per alert | < 1KB | Pointer reuse, avoid copies |

### Optimization Strategies

#### 1. Zero-Copy Parsing
```go
// BAD: Creates unnecessary copies
func parseAlerts(data []byte) []PrometheusAlert {
    var webhook PrometheusWebhook
    json.Unmarshal(data, &webhook)

    alerts := make([]PrometheusAlert, len(webhook.Alerts))
    copy(alerts, webhook.Alerts) // ❌ Unnecessary copy
    return alerts
}

// GOOD: Zero-copy, direct return
func parseAlerts(data []byte) []PrometheusAlert {
    var webhook PrometheusWebhook
    json.Unmarshal(data, &webhook)
    return webhook.Alerts // ✅ Direct return
}
```

#### 2. Pre-Allocation
```go
// Pre-allocate slices with known capacity
alerts := make([]*core.Alert, 0, len(promWebhook.Alerts)) // ✅ Pre-allocated

// Pre-allocate maps
labels := make(map[string]string, len(groupLabels)+len(alertLabels)) // ✅
```

#### 3. String Builder for Fingerprints
```go
// BAD: String concatenation creates many temporaries
canonical := alertName
for _, k := range keys {
    canonical += "|" + k + "=" + labels[k] // ❌ Many allocations
}

// GOOD: strings.Builder (1 allocation)
var sb strings.Builder
sb.WriteString(alertName)
for _, k := range keys {
    sb.WriteString("|")
    sb.WriteString(k)
    sb.WriteString("=")
    sb.WriteString(labels[k])
}
canonical := sb.String() // ✅ Single allocation
```

#### 4. Avoid Reflection
```go
// Use json.Unmarshal with struct tags (fast)
var webhook PrometheusWebhook
json.Unmarshal(data, &webhook) // ✅ Struct-based, no reflection at runtime

// Avoid json.RawMessage unnecessary buffering
```

### Benchmarks

```go
func BenchmarkParseSingleAlert(b *testing.B) {
    payload := []byte(`{"labels":{"alertname":"test"},"state":"firing",...}`)
    parser := NewPrometheusParser()

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _, _ = parser.Parse(payload)
    }
}
// Target: < 10µs/op, < 5 allocs/op

func BenchmarkParse100Alerts(b *testing.B) {
    payload := generateLargePayload(100)
    parser := NewPrometheusParser()

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _, _ = parser.Parse(payload)
    }
}
// Target: < 1ms/op, < 500 allocs/op

func BenchmarkGenerateFingerprint(b *testing.B) {
    labels := map[string]string{
        "alertname": "HighCPU",
        "instance": "server-1",
        "job": "api",
        "severity": "warning",
    }

    b.ResetTimer()
    b.ReportAllocs()
    for i := 0; i < b.N; i++ {
        _ = generateFingerprint("HighCPU", labels)
    }
}
// Target: < 1µs/op, 0 allocs/op (if using string builder pool)
```

---

## 🧪 Testing Strategy

### Test Pyramid

```
           /\
          /  \
         / E2E\           5 tests (10%)
        /──────\
       / Integ \         10 tests (20%)
      /─────────\
     /   Unit    \       35 tests (70%)
    /────────────\
```

### Test Categories

#### 1. Unit Tests (35 tests, 70%)

**Format Detection** (10 tests):
```go
TestDetectPrometheusV1Format()           // ✅ Detect v1 array
TestDetectPrometheusV2Format()           // ✅ Detect v2 grouped
TestDetectAlertmanagerFormat()           // ✅ Regression: still detects AM
TestDetectUnknownFormat()                // ✅ Unknown format handling
TestDetectEmptyPayload()                 // ❌ Empty payload error
TestDetectInvalidJSON()                  // ❌ Invalid JSON error
TestDetectPrometheusV1WithGroupLabels()  // ✅ Edge: v1 with extra fields
TestDetectPrometheusV2FlatAlerts()       // ✅ Edge: v2 without groups
TestDetectLargePayload()                 // ✅ 1000+ alerts performance
TestDetectConcurrentDetection()          // ✅ Thread-safety
```

**Parsing** (10 tests):
```go
TestParsePrometheusV1SingleAlert()       // ✅ Valid v1 single
TestParsePrometheusV1MultipleAlerts()    // ✅ Valid v1 array
TestParsePrometheusV2Grouped()           // ✅ Valid v2 groups
TestParseMissingRequiredField()          // ❌ Missing alertname
TestParseInvalidTimestamp()              // ❌ Invalid activeAt
TestParseInvalidState()                  // ❌ Invalid state value
TestParseMissingGeneratorURL()           // ❌ Missing generatorURL
TestParseLargePayload()                  // ✅ 1000+ alerts
TestParseWithFingerprintProvided()       // ✅ Use existing fingerprint
TestParseWithoutFingerprintGenerate()    // ✅ Generate fingerprint
```

**Validation** (5 tests):
```go
TestValidateRequiredFields()             // ❌ All required fields
TestValidateLabelNames()                 // ❌ Invalid label names
TestValidateTimestampFormat()            // ❌ RFC3339 validation
TestValidateGeneratorURLFormat()         // ❌ URL format validation
TestValidateStateValues()                // ❌ State enum validation
```

**Conversion** (10 tests):
```go
TestConvertPrometheusAlertToCoreAlert()  // ✅ Full conversion
TestConvertStateMapping()                // ✅ firing/pending/inactive
TestConvertMergeGroupLabels()            // ✅ v2 group labels
TestConvertPreserveAnnotations()         // ✅ All annotations preserved
TestConvertStorePrometheusValue()        // ✅ value → annotations
TestConvertGenerateFingerprint()         // ✅ Fingerprint generation
TestConvertNilHandling()                 // ✅ Nil endsAt, nil annotations
TestConvertMultipleAlerts()              // ✅ Batch conversion
TestConvertEmptyLabels()                 // ❌ Empty labels error
TestConvertInvalidStatus()               // ❌ Invalid status error
```

#### 2. Integration Tests (10 tests, 20%)

**End-to-End Pipeline** (5 tests):
```go
TestE2EPrometheusV1ToCoreAlert()         // ✅ Full pipeline v1
TestE2EPrometheusV2ToCoreAlert()         // ✅ Full pipeline v2
TestE2EAlertmanagerStillWorks()          // ✅ Regression: AM works
TestE2EMixedFormatsSequential()          // ✅ AM then Prom
TestE2EErrorPropagation()                // ❌ Errors bubble up correctly
```

**Handler Integration** (5 tests):
```go
TestHandlerSelectsPrometheusParser()     // ✅ Strategy pattern works
TestHandlerFallsBackToAlertmanager()     // ✅ Unknown → AM fallback
TestHandlerRecordsMetrics()              // ✅ Prometheus metrics recorded
TestHandlerLogsCorrectly()               // ✅ Structured logging
TestHandlerConcurrentRequests()          // ✅ Thread-safety
```

#### 3. Benchmarks (8+ benchmarks)

```go
BenchmarkDetectPrometheusFormat          // < 5µs target
BenchmarkParseSingleAlert                // < 10µs target
BenchmarkParse100Alerts                  // < 1ms target
BenchmarkConvertToDomain                 // < 5µs per alert
BenchmarkGenerateFingerprint             // < 1µs target
BenchmarkFlattenGroups                   // < 100µs for 10 groups
BenchmarkHandlerE2E                      // < 100µs full pipeline
BenchmarkConcurrentParsing               // Scalability test
```

### Test Data

**Valid Prometheus v1**:
```json
[
  {
    "labels": {"alertname": "HighCPU", "instance": "server-1", "job": "api", "severity": "warning"},
    "annotations": {"summary": "CPU high", "description": "CPU > 80%"},
    "state": "firing",
    "activeAt": "2025-11-18T10:00:00Z",
    "value": "0.85",
    "generatorURL": "http://prometheus:9090/graph?...",
    "fingerprint": "abc123..."
  }
]
```

**Valid Prometheus v2**:
```json
{
  "groups": [
    {
      "labels": {"job": "api", "severity": "warning"},
      "alerts": [
        {
          "labels": {"alertname": "HighCPU", "instance": "server-1"},
          "annotations": {"summary": "CPU high"},
          "state": "firing",
          "activeAt": "2025-11-18T10:00:00Z",
          "value": "0.85",
          "generatorURL": "http://prometheus:9090/graph?..."
        }
      ]
    }
  ]
}
```

---

## 📝 Documentation Plan

### 1. Code Documentation (500+ lines godoc)

```go
// Package webhook provides webhook parsers for multiple alert formats.
//
// Supported Formats:
//  - Prometheus v1 (legacy /api/v1/alerts)
//  - Prometheus v2 (modern /api/v2/alerts)
//  - Alertmanager webhook (existing support)
//
// The package uses Strategy Pattern to select appropriate parser based on
// detected webhook format. All parsers normalize alerts to core.Alert domain model.
//
// Example Usage:
//
//  // Create universal handler with multiple parsers
//  handler := webhook.NewUniversalWebhookHandler(processor, logger)
//
//  // Handle any webhook format (auto-detected)
//  resp, err := handler.HandleWebhook(ctx, &HandleWebhookRequest{
//      Payload: rawJSON,
//      Headers: httpHeaders,
//  })
//
// Performance:
//  - Parse single alert: < 10µs
//  - Parse 100 alerts: < 1ms
//  - Zero allocations in hot path
//
// Thread Safety:
//  All parsers are safe for concurrent use.
package webhook
```

### 2. README.md (400+ lines)

```markdown
# Prometheus Alert Parser

Enterprise-grade parser for Prometheus native alerts with 150% quality.

## Features

- ✅ Prometheus v1 format support
- ✅ Prometheus v2 format support
- ✅ Backward compatible with Alertmanager
- ✅ Automatic format detection
- ✅ Deterministic fingerprinting
- ✅ < 10µs parsing latency
- ✅ 85%+ test coverage

## Quick Start

```go
import "github.com/vitaliisemenov/alert-history/internal/infrastructure/webhook"

// Create parser
parser := webhook.NewPrometheusParser()

// Parse Prometheus alert
webhook, err := parser.Parse(jsonPayload)

// Convert to domain model
alerts, err := parser.ConvertToDomain(webhook)
```

## Format Support

| Format | Endpoint | Status |
|--------|----------|--------|
| Prometheus v1 | `/api/v1/alerts` | ✅ Supported |
| Prometheus v2 | `/api/v2/alerts` | ✅ Supported |
| Alertmanager | `/api/v2/receiver` | ✅ Supported |

## Field Mapping

...

## Performance

...

## Testing

```

### 3. API Documentation (300+ lines)

### 4. Integration Guide (200+ lines)

---

## 🎯 Success Criteria

### Must Have (150% Quality)
- ✅ All FR + NFR implemented
- ✅ 85%+ test coverage
- ✅ < 10µs parse performance
- ✅ 1,400+ lines documentation
- ✅ Grade A+ (95/100 points)
- ✅ Zero technical debt
- ✅ Backward compatible

### Metrics
| Metric | Target | Stretch |
|--------|--------|---------|
| Test Coverage | 85%+ | 95%+ |
| Unit Tests | 35+ | 50+ |
| Benchmarks | 8+ | 12+ |
| Documentation | 1,400+ LOC | 2,000+ LOC |
| Parse Latency | < 10µs | < 5µs |
| Memory/Alert | < 1KB | < 500B |

---

**Prepared by**: Technical Design Team
**Date**: 2025-11-18
**Version**: 1.0
**Status**: Ready for Implementation
