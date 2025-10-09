# TN-043: Validation Design

```go
type WebhookValidator interface {
    ValidateAlertmanager(webhook *AlertmanagerWebhook) error
    ValidateGeneric(data map[string]interface{}) error
}

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
    Value   interface{} `json:"value,omitempty"`
}

type ValidationResult struct {
    Valid  bool              `json:"valid"`
    Errors []*ValidationError `json:"errors,omitempty"`
}
```
