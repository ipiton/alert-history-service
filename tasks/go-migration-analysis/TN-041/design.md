# TN-041: Webhook Parser Design

```go
type AlertmanagerWebhook struct {
    Version           string                 `json:"version"`
    GroupKey          string                 `json:"groupKey"`
    TruncatedAlerts   int                   `json:"truncatedAlerts"`
    Status            string                `json:"status"`
    Receiver          string                `json:"receiver"`
    GroupLabels       map[string]string     `json:"groupLabels"`
    CommonLabels      map[string]string     `json:"commonLabels"`
    CommonAnnotations map[string]string     `json:"commonAnnotations"`
    ExternalURL       string                `json:"externalURL"`
    Alerts            []AlertmanagerAlert   `json:"alerts"`
}

type WebhookParser interface {
    Parse(data []byte) (*AlertmanagerWebhook, error)
    Validate(webhook *AlertmanagerWebhook) error
    ConvertToDomain(webhook *AlertmanagerWebhook) ([]*domain.Alert, error)
}
```
