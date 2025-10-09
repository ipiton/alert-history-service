# TN-042: Universal Handler Design

```go
type WebhookHandler struct {
    parsers            map[WebhookType]WebhookParser
    deduplicationSvc   DeduplicationService
    classificationSvc  AlertClassificationService
    publishingSvc      PublishingService
    enrichmentManager  EnrichmentModeManager
    logger            *slog.Logger
    metrics           *prometheus.CounterVec
}

func (h *WebhookHandler) HandleWebhook(c *fiber.Ctx) error {
    body := c.Body()

    // Auto-detect webhook type
    webhookType, err := h.detectWebhookType(body)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "unknown webhook format"})
    }

    // Parse webhook
    parser := h.parsers[webhookType]
    webhook, err := parser.Parse(body)
    if err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "invalid webhook payload"})
    }

    // Process alerts
    alerts, err := parser.ConvertToDomain(webhook)
    if err != nil {
        return c.Status(500).JSON(fiber.Map{"error": "conversion failed"})
    }

    return h.processAlerts(c.Context(), alerts)
}
```
