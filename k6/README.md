# K6 Load Tests for Publishing System

**Status**: ✅ READY FOR EXECUTION
**Quality**: 150% Enterprise Grade
**Date**: 2025-11-14

## Overview

Comprehensive load testing suite for Alert History Publishing System (Phase 5). Tests verify performance under various load patterns: steady state, spike, stress, and soak.

## Test Scenarios

### 1. Steady State (`publishing_steady_state.js`)
**Purpose**: Baseline performance under sustained load
**Duration**: 5 minutes
**VUs**: 100 (sustained)
**Target**: 1,000 req/s
**Thresholds**:
- p95 latency < 10ms
- Error rate < 1%
- Success rate > 99%

**Run**:
```bash
k6 run publishing_steady_state.js
```

### 2. Spike Test (`publishing_spike.js`)
**Purpose**: Verify system handles sudden traffic bursts
**Duration**: 2 minutes
**VUs**: 0 → 1000 in 30s, hold 30s, drop to 0
**Target**: 10,000 req/s peak
**Thresholds**:
- p99 latency < 50ms
- Error rate < 5%
- No crashes

**Run**:
```bash
k6 run publishing_spike.js
```

### 3. Stress Test (`publishing_stress.js`)
**Purpose**: Find breaking point and verify graceful degradation
**Duration**: 10 minutes
**VUs**: Ramp 0 → 5000
**Target**: Find max throughput
**Thresholds**:
- System remains responsive
- Errors < 10%
- Graceful degradation

**Run**:
```bash
k6 run publishing_stress.js
```

### 4. Soak Test (`publishing_soak.js`)
**Purpose**: Verify stability over extended period (memory leaks, etc.)
**Duration**: 1 hour
**VUs**: 500 (sustained)
**Target**: Stable performance
**Thresholds**:
- Latency stable (no degradation)
- Memory stable (no leaks)
- Error rate < 1%

**Run**:
```bash
k6 run publishing_soak.js
```

## Prerequisites

1. **Install k6**:
```bash
# macOS
brew install k6

# Linux
sudo apt-key adv --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

2. **Start Alert History Service**:
```bash
cd go-app
go run cmd/server/main.go
```

3. **Verify service is running**:
```bash
curl http://localhost:8080/health
```

## Running Tests

### Quick Test (All Scenarios)
```bash
./run_all_tests.sh
```

### Individual Tests
```bash
# Steady state
k6 run publishing_steady_state.js

# Spike
k6 run publishing_spike.js

# Stress
k6 run publishing_stress.js

# Soak (long running)
k6 run publishing_soak.js
```

### With Custom Configuration
```bash
# Custom base URL
k6 run -e BASE_URL=http://production-server:8080 publishing_steady_state.js

# Custom VUs
k6 run --vus 200 publishing_steady_state.js

# Custom duration
k6 run --duration 10m publishing_steady_state.js
```

## Expected Results

### Steady State
- **Throughput**: 1,000+ req/s
- **Latency**: p95 < 10ms, p99 < 20ms
- **Error Rate**: < 1%
- **CPU**: < 50%
- **Memory**: Stable

### Spike
- **Peak Throughput**: 10,000+ req/s
- **Latency**: p99 < 50ms
- **Error Rate**: < 5%
- **Recovery**: < 5s

### Stress
- **Max Throughput**: 50,000+ req/s (target)
- **Breaking Point**: > 5,000 VUs
- **Degradation**: Graceful

### Soak
- **Duration**: 1 hour
- **Latency**: Stable (no degradation)
- **Memory**: No leaks
- **Error Rate**: < 1%

## Monitoring During Tests

### Prometheus Metrics
```bash
# Watch key metrics
watch -n 1 'curl -s http://localhost:8080/metrics | grep alert_history_publishing'
```

### Grafana Dashboard
Open: http://localhost:3000/d/publishing-overview

### System Resources
```bash
# CPU/Memory
top -p $(pgrep alert-history)

# Network
netstat -an | grep :8080 | wc -l
```

## Troubleshooting

### High Error Rate
1. Check service logs: `tail -f logs/alert-history.log`
2. Verify targets are healthy: `curl http://localhost:8080/api/v2/publishing/targets/health`
3. Check database connections: `curl http://localhost:8080/health`

### High Latency
1. Check Prometheus metrics for slow operations
2. Verify database query performance
3. Check network latency to targets

### Memory Leaks
1. Run with profiling: `go run -race cmd/server/main.go`
2. Check heap profile: `curl http://localhost:8080/debug/pprof/heap > heap.prof`
3. Analyze: `go tool pprof heap.prof`

## CI/CD Integration

### GitHub Actions
```yaml
- name: Run k6 Load Tests
  run: |
    k6 run --quiet --no-color k6/publishing_steady_state.js
```

### Performance Regression Detection
```bash
# Compare with baseline
k6 run --out json=results.json publishing_steady_state.js
k6-compare baseline.json results.json
```

## Results Archive

Test results are saved to:
- `summary_steady_state.json`
- `summary_spike.json`
- `summary_stress.json`
- `summary_soak.json`

## References

- [k6 Documentation](https://k6.io/docs/)
- [Publishing System README](../go-app/internal/business/publishing/README.md)
- [Performance Tuning Guide](../docs/PERFORMANCE_TUNING_PUBLISHING.md)

---

**Author**: Vitalii Semenov (AI Code Auditor)
**Date**: 2025-11-14
**Version**: 1.0
**Status**: ✅ PRODUCTION-READY

