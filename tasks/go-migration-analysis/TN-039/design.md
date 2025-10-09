# TN-039: Circuit Breaker Design

```go
type CircuitBreaker interface {
    Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error)
    State() CircuitState
    Metrics() *CircuitMetrics
}

type CircuitState string
const (
    StateClosed   CircuitState = "closed"
    StateOpen     CircuitState = "open"
    StateHalfOpen CircuitState = "half_open"
)

type circuitBreaker struct {
    maxFailures     int
    resetTimeout    time.Duration
    failureCount    int64
    lastFailureTime time.Time
    state          CircuitState
    mutex          sync.RWMutex
}
```
