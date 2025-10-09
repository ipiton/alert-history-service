# TN-23: Дизайн Webhook

## Структуры данных
```go
type AlertmanagerWebhook struct {
    Version     string  `json:"version"`
    GroupKey    string  `json:"groupKey"`
    Status      string  `json:"status"`
    Receiver    string  `json:"receiver"`
    Alerts      []Alert `json:"alerts"`
}

type Alert struct {
    Status       string            `json:"status"`
    Labels       map[string]string `json:"labels"`
    Annotations  map[string]string `json:"annotations"`
    StartsAt     time.Time         `json:"startsAt"`
    EndsAt       time.Time         `json:"endsAt"`
    Fingerprint  string            `json:"fingerprint"`
}

// Handler
func HandleWebhook(db Database) fiber.Handler {
    return func(c *fiber.Ctx) error {
        var webhook AlertmanagerWebhook
        if err := c.BodyParser(&webhook); err != nil {
            return c.Status(400).JSON(fiber.Map{"error": "invalid payload"})
        }

        for _, alert := range webhook.Alerts {
            if err := db.SaveAlert(c.Context(), &alert); err != nil {
                // Log error but continue
                logger.Error("failed to save alert", "error", err)
            }
        }

        return c.JSON(fiber.Map{"status": "ok"})
    }
}
```
