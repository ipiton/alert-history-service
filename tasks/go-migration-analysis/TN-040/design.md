# TN-040: Retry Logic Design

```go
type RetryPolicy struct {
    MaxRetries  int
    BaseDelay   time.Duration
    MaxDelay    time.Duration
    Multiplier  float64
    Jitter      bool
}

func WithRetry(ctx context.Context, policy *RetryPolicy, fn func() error) error {
    var lastErr error
    for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
        if err := fn(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        if attempt < policy.MaxRetries {
            delay := calculateDelay(policy, attempt)
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(delay):
            }
        }
    }
    return lastErr
}
```
