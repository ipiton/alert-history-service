# TN-20: Дизайн логирования

## Инициализация
```go
package logger

import (
    "log/slog"
    "os"
)

func NewLogger(level string, format string) *slog.Logger {
    var handler slog.Handler

    opts := &slog.HandlerOptions{
        Level: parseLevel(level),
    }

    if format == "json" {
        handler = slog.NewJSONHandler(os.Stdout, opts)
    } else {
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    return slog.New(handler)
}

// HTTP Middleware
func LoggingMiddleware(logger *slog.Logger) fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()

        logger.Info("request",
            "method", c.Method(),
            "path", c.Path(),
            "status", c.Response().StatusCode(),
            "duration", time.Since(start),
        )

        return err
    }
}
```
