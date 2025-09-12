# TN-25: План тестирования

## Сценарии
1. Webhook processing: 1000 RPS
2. History API: 500 RPS
3. Mixed load: webhooks + queries

## Метрики
- Requests per second
- Latency (p50, p95, p99)
- Memory usage
- CPU usage
- Error rate

## Инструменты
```bash
# Load test
k6 run load-test.js

# CPU profiling
go tool pprof http://localhost:8080/debug/pprof/profile

# Memory profiling
go tool pprof http://localhost:8080/debug/pprof/heap
```
