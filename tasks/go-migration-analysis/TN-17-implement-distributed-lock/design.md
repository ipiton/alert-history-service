# TN-17: Дизайн Distributed Lock

## Реализация
```go
type DistributedLock struct {
    redis *redis.Client
    key   string
    value string
    ttl   time.Duration
}

func (l *DistributedLock) Acquire(ctx context.Context) (bool, error) {
    return l.redis.SetNX(ctx, l.key, l.value, l.ttl).Result()
}

func (l *DistributedLock) Release(ctx context.Context) error {
    script := `
    if redis.call("get", KEYS[1]) == ARGV[1] then
        return redis.call("del", KEYS[1])
    else
        return 0
    end
    `
    return l.redis.Eval(ctx, script, []string{l.key}, l.value).Err()
}
```
