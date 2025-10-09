# TN-045: Webhook Metrics Design

```go
type WebhookMetrics struct {
    RequestsTotal     *prometheus.CounterVec
    RequestDuration   *prometheus.HistogramVec
    ProcessingTime    *prometheus.HistogramVec
    QueueSize         prometheus.Gauge
    ActiveWorkers     prometheus.Gauge
    ErrorsTotal       *prometheus.CounterVec
}

func NewWebhookMetrics() *WebhookMetrics {
    return &WebhookMetrics{
        RequestsTotal: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Name: "webhook_requests_total",
                Help: "Total number of webhook requests",
            },
            []string{"type", "status"},
        ),
        RequestDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Name: "webhook_request_duration_seconds",
                Help: "Webhook request duration",
            },
            []string{"type"},
        ),
    }
}
```
