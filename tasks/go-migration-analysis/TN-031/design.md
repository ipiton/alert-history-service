# TN-031: Design Domain Models

## Актуальная реализация (2025-10-08)

**Расположение**: `internal/core/interfaces.go`

## Структуры

### Alert
```go
// Alert represents alert data model
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    AlertName    string            `json:"alert_name" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required,oneof=firing resolved"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"starts_at" validate:"required"`
    EndsAt       *time.Time        `json:"ends_at,omitempty"`
    GeneratorURL *string           `json:"generator_url,omitempty"`
    Timestamp    *time.Time        `json:"timestamp,omitempty"`
}

// Namespace returns alert namespace from labels
func (a *Alert) Namespace() *string {
    if ns, ok := a.Labels["namespace"]; ok {
        return &ns
    }
    return nil
}

// Severity returns alert severity from labels
func (a *Alert) Severity() *string {
    if sev, ok := a.Labels["severity"]; ok {
        return &sev
    }
    return nil
}
```

### AlertStatus
```go
type AlertStatus string

const (
    StatusFiring   AlertStatus = "firing"
    StatusResolved AlertStatus = "resolved"
)
```

### AlertSeverity
```go
// AlertSeverity represents alert severity levels
type AlertSeverity string

const (
    SeverityCritical AlertSeverity = "critical"
    SeverityWarning  AlertSeverity = "warning"
    SeverityInfo     AlertSeverity = "info"
    SeverityNoise    AlertSeverity = "noise"
)
```

**Примечание**: Изменено с оригинального design (critical, high, medium, low, info) на более практичные уровни (critical, warning, info, noise), соответствующие реальному использованию в production.

### ClassificationResult
```go
// ClassificationResult represents LLM classification result
type ClassificationResult struct {
    Severity        AlertSeverity  `json:"severity" validate:"required,oneof=critical warning info noise"`
    Confidence      float64        `json:"confidence" validate:"gte=0,lte=1"`
    Reasoning       string         `json:"reasoning" validate:"required"`
    Recommendations []string       `json:"recommendations"`
    ProcessingTime  float64        `json:"processing_time" validate:"gte=0"`
    Metadata        map[string]any `json:"metadata,omitempty"`
}
```

### PublishingTarget
```go
// PublishingTarget represents publishing target configuration
type PublishingTarget struct {
    Name         string            `json:"name" validate:"required"`
    Type         string            `json:"type" validate:"required"`
    URL          string            `json:"url" validate:"required,url"`
    Enabled      bool              `json:"enabled"`
    FilterConfig map[string]any    `json:"filter_config"`
    Headers      map[string]string `json:"headers"`
    Format       PublishingFormat  `json:"format" validate:"required"`
}
```

### PublishingFormat
```go
type PublishingFormat string

const (
    FormatAlertmanager PublishingFormat = "alertmanager"
    FormatRootly       PublishingFormat = "rootly"
    FormatPagerDuty    PublishingFormat = "pagerduty"
    FormatSlack        PublishingFormat = "slack"
    FormatWebhook      PublishingFormat = "webhook"
)
```

### EnrichedAlert
```go
// EnrichedAlert represents alert enriched with classification data
type EnrichedAlert struct {
    Alert               *Alert                `json:"alert"`
    Classification      *ClassificationResult `json:"classification,omitempty"`
    EnrichmentMetadata  map[string]any        `json:"enrichment_metadata,omitempty"`
    ProcessingTimestamp *time.Time            `json:"processing_timestamp,omitempty"`
}
```

## Validation Rules

**Требуется добавить зависимость:**
```bash
go get github.com/go-playground/validator/v10
```

**Validation использование:**
```go
import "github.com/go-playground/validator/v10"

validate := validator.New()

alert := &Alert{
    Fingerprint: "abc123",
    AlertName:   "HighCPUUsage",
    Status:      StatusFiring,
    StartsAt:    time.Now(),
}

if err := validate.Struct(alert); err != nil {
    // Handle validation errors
}
```

## Отличия от исходного design

| Аспект | Исходный Design | Текущая Реализация | Причина изменения |
|--------|-----------------|---------------------|-------------------|
| Severity levels | 5 уровней (critical, high, medium, low, info) | 4 уровня (critical, warning, info, noise) | Соответствие Alertmanager conventions |
| Alert.AlertName | Отсутствует | Присутствует | Требуется для идентификации типа алерта |
| Alert.Timestamp | Отсутствует | Присутствует | Требуется для tracking обработки |
| Alert.GeneratorURL | `string` | `*string` | Опциональное поле, может быть nil |
| JSON naming | camelCase | snake_case | Go conventions для JSON tags |

## Структура файлов

**Текущая реализация:**
```
internal/core/
  └── interfaces.go  (все модели и интерфейсы)
```

**НЕ использовать:**
```
internal/core/domain/    # Не создавать!
```

**Обоснование**: Все модели и интерфейсы в одном файле `interfaces.go` проще импортировать и поддерживать. Разделение на `domain/` не добавляет ценности для текущего размера проекта.
