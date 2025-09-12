# TN-031: Design Domain Models

## Структуры
```go
// Alert represents an alert from Alertmanager
type Alert struct {
    Fingerprint  string            `json:"fingerprint" validate:"required"`
    Status       AlertStatus       `json:"status" validate:"required"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"startsAt"`
    EndsAt       *time.Time        `json:"endsAt,omitempty"`
    GeneratorURL string            `json:"generatorURL,omitempty"`
}

type AlertStatus string
const (
    AlertStatusFiring   AlertStatus = "firing"
    AlertStatusResolved AlertStatus = "resolved"
)

// Classification represents LLM classification result
type Classification struct {
    Fingerprint     string    `json:"fingerprint"`
    Severity        Severity  `json:"severity"`
    Confidence      float64   `json:"confidence" validate:"min=0,max=1"`
    Reasoning       string    `json:"reasoning"`
    Recommendations []string  `json:"recommendations"`
    ProcessingTime  float64   `json:"processing_time"`
    CreatedAt       time.Time `json:"created_at"`
}

type Severity string
const (
    SeverityCritical Severity = "critical"
    SeverityHigh     Severity = "high"
    SeverityMedium   Severity = "medium"
    SeverityLow      Severity = "low"
    SeverityInfo     Severity = "info"
)

// PublishingTarget represents external system
type PublishingTarget struct {
    Name     string                 `json:"name"`
    Type     PublishingType         `json:"type"`
    URL      string                 `json:"url"`
    Config   map[string]interface{} `json:"config"`
    Enabled  bool                   `json:"enabled"`
    LastSeen time.Time              `json:"last_seen"`
}

type PublishingType string
const (
    PublishingTypeRootly     PublishingType = "rootly"
    PublishingTypePagerDuty  PublishingType = "pagerduty"
    PublishingTypeSlack      PublishingType = "slack"
    PublishingTypeWebhook    PublishingType = "webhook"
)
```
