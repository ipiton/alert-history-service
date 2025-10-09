# TN-21: Дизайн метрик

## Middleware
```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    httpDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "http_request_duration_seconds",
            Help: "Duration of HTTP requests.",
        },
        []string{"method", "path", "status"},
    )

    httpRequests = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "http_requests_total",
            Help: "Total number of HTTP requests.",
        },
        []string{"method", "path", "status"},
    )
)

func PrometheusMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        start := time.Now()
        err := c.Next()

        status := strconv.Itoa(c.Response().StatusCode())
        httpRequests.WithLabelValues(c.Method(), c.Path(), status).Inc()
        httpDuration.WithLabelValues(c.Method(), c.Path(), status).
            Observe(time.Since(start).Seconds())

        return err
    }
}
```
