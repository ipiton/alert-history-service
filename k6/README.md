# TN-061: Webhook Endpoint - k6 Load Tests

**Purpose**: Validate webhook endpoint performance under various load conditions
**Quality Target**: 150% Enterprise Grade (Grade A++)

---

## 📊 Test Scenarios

### 1. Steady State Test (`webhook-steady-state.js`)
**Purpose**: Validate sustained production load
**Duration**: 10 minutes
**Load**: 10,000 req/s (constant)

**Targets (150% Quality)**:
- ✅ p95 latency < 5ms
- ✅ p99 latency < 10ms
- ✅ Error rate < 0.01%
- ✅ Throughput > 10,000 req/s

**Usage**:
```bash
k6 run --vus 100 --duration 10m k6/webhook-steady-state.js
```

**With environment variables**:
```bash
BASE_URL=http://localhost:8080 \
API_KEY=your-api-key \
k6 run k6/webhook-steady-state.js
```

---

### 2. Spike Test (`webhook-spike-test.js`)
**Purpose**: Test elasticity and recovery from traffic spikes
**Duration**: 7 minutes
**Load**: 1K → 20K → 1K req/s

**Pattern**:
- 0-2m: Baseline (1K req/s)
- 2-2.5m: Ramp up (1K → 20K)
- 2.5-3.5m: Peak (20K req/s)
- 3.5-4m: Ramp down (20K → 1K)
- 4-6m: Recovery (1K req/s)

**Targets**:
- ✅ System remains stable during 20x spike
- ✅ Quick recovery after spike
- ✅ Error rate < 0.1% during spike
- ✅ No lingering effects after recovery

**Usage**:
```bash
k6 run k6/webhook-spike-test.js
```

---

### 3. Stress Test (`webhook-stress-test.js`)
**Purpose**: Find system breaking point and maximum capacity
**Duration**: 17 minutes
**Load**: 1K → 50K req/s (gradual increase)

**Stages**:
1. 1K req/s (baseline)
2. 5K req/s
3. 10K req/s (target)
4. 15K req/s (150%)
5. 20K req/s (200%)
6. 30K req/s (300%)
7. 40K req/s (400%)
8. 50K req/s (500%)
9. 1K req/s (recovery)

**Goals**:
- ✅ Find maximum sustainable throughput
- ✅ Observe graceful degradation (429 not 500)
- ✅ Identify bottlenecks
- ✅ Verify recovery after stress

**Usage**:
```bash
k6 run k6/webhook-stress-test.js
```

---

### 4. Soak Test (`webhook-soak-test.js`)
**Purpose**: Detect memory leaks and resource exhaustion
**Duration**: 4 hours
**Load**: 2,000 req/s (sustained)
**Total Requests**: ~28.8 million

**Monitoring**:
- ✅ Memory leaks
- ✅ Resource exhaustion
- ✅ Performance degradation over time
- ✅ Connection pool issues
- ✅ Goroutine leaks

**Targets**:
- ✅ Stable latency (< 20% degradation)
- ✅ Success rate > 99.99%
- ✅ No memory growth
- ✅ Error rate < 0.01%

**Usage**:
```bash
k6 run k6/webhook-soak-test.js
```

**Note**: This test runs for 4 hours. Monitor Prometheus metrics during execution.

---

## 🛠️ Setup

### Prerequisites
```bash
# Install k6
# macOS
brew install k6

# Linux
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6

# Docker
docker pull grafana/k6:latest
```

### Configuration

**Environment Variables**:
- `BASE_URL`: Webhook endpoint URL (default: `http://localhost:8080`)
- `API_KEY`: API key for authentication (optional)

**Example**:
```bash
export BASE_URL=https://alerts.example.com
export API_KEY=your-secret-api-key
k6 run k6/webhook-steady-state.js
```

---

## 📈 Running Tests

### Quick Start
```bash
# Run all tests sequentially
./k6/run-all-tests.sh

# Or run individually
k6 run k6/webhook-steady-state.js
k6 run k6/webhook-spike-test.js
k6 run k6/webhook-stress-test.js
k6 run k6/webhook-soak-test.js  # Warning: 4 hours!
```

### With Results Output
```bash
# JSON output
k6 run --out json=results.json k6/webhook-steady-state.js

# CSV output
k6 run --out csv=results.csv k6/webhook-steady-state.js

# InfluxDB (for Grafana)
k6 run --out influxdb=http://localhost:8086/k6 k6/webhook-steady-state.js
```

### Cloud Execution (k6 Cloud)
```bash
k6 cloud k6/webhook-steady-state.js
```

---

## 📊 Interpreting Results

### Success Criteria

**Steady State Test**:
```
✓ http_req_duration.......: p(95)<5ms, p(99)<10ms
✓ error_rate..............: <0.01%
✓ http_reqs...............: >9,500/s (allowing 5% margin)
```

**Spike Test**:
```
✓ System stable during 20x spike
✓ Quick recovery (< 30s)
✓ Error rate < 0.1% during spike
✓ No lingering effects in recovery phase
```

**Stress Test**:
```
✓ Graceful degradation (429 before 500)
✓ Maximum capacity identified
✓ System recovers after stress
✓ Bottlenecks identified
```

**Soak Test**:
```
✓ Stable latency throughout (< 20% degradation)
✓ No memory growth
✓ Success rate > 99.99%
✓ No resource leaks
```

### Key Metrics

**Response Time**:
- `http_req_duration`: Total request duration
- `http_req_waiting`: Time to first byte
- `http_req_connecting`: Connection establishment time

**Throughput**:
- `http_reqs`: Requests per second
- `data_received`: Bytes per second

**Errors**:
- `http_req_failed`: Failed requests count
- `error_rate`: Custom error rate metric

**Custom Metrics**:
- `webhook_duration`: Webhook processing time
- `success_rate`: Successful requests rate
- `degradation_score`: Performance degradation over time (soak test)

---

## 🔍 Troubleshooting

### Common Issues

**1. Connection Refused**
```
error: dial tcp: connection refused
```
**Solution**: Ensure service is running at `BASE_URL`

**2. Too Many Open Files**
```
error: too many open files
```
**Solution**: Increase file descriptor limit:
```bash
ulimit -n 10000
```

**3. Rate Limiting**
```
✗ 429 Too Many Requests
```
**Solution**: This is expected behavior. Check rate limiting configuration.

**4. Timeouts**
```
✗ request timeout (30s)
```
**Solution**: Check service performance, database connections, or increase timeout.

---

## 📚 Best Practices

### Before Running Tests

1. **Ensure baseline health**:
   ```bash
   curl http://localhost:8080/healthz
   ```

2. **Check Prometheus metrics available**:
   ```bash
   curl http://localhost:8080/metrics
   ```

3. **Review configuration**:
   - Rate limiting: Should be disabled or set high
   - Authentication: Configure API key if enabled
   - Timeouts: Set appropriately

### During Tests

1. **Monitor Prometheus metrics**:
   - CPU usage
   - Memory usage
   - Goroutine count
   - Request duration
   - Error rates

2. **Watch logs**:
   ```bash
   kubectl logs -f deployment/alert-history --tail=100
   ```

3. **Check database connections**:
   - Connection pool usage
   - Query performance
   - Lock contention

### After Tests

1. **Review k6 summary output**
2. **Check for errors in logs**
3. **Verify resource cleanup**:
   - Memory returned to baseline
   - Goroutines back to normal
   - No connection leaks

4. **Compare results against targets**
5. **Document any issues or anomalies**

---

## 🎯 Performance Targets (150% Quality)

### Baseline Requirements (100%)
- Latency: p99 < 10ms
- Throughput: > 5,000 req/s
- Error rate: < 0.1%
- Uptime: 99.9%

### 150% Quality Targets
- ✅ Latency: p95 < 5ms, p99 < 10ms
- ✅ Throughput: > 10,000 req/s
- ✅ Error rate: < 0.01%
- ✅ Uptime: 99.95%
- ✅ Sustained load: 4 hours without degradation
- ✅ Spike handling: 20x increase without failure
- ✅ Graceful degradation: Rate limiting before errors

---

## 📝 Test Results Template

```markdown
## Load Test Results - YYYY-MM-DD

### Test Environment
- Service: alert-history v1.0.0
- Endpoint: http://localhost:8080/webhook
- Infrastructure: AWS EC2 t3.xlarge (4 vCPU, 16GB RAM)
- Database: PostgreSQL 15 (RDS db.t3.medium)

### Steady State Test
- ✅ Duration: 10 minutes
- ✅ Target load: 10,000 req/s
- ✅ Actual throughput: 10,234 req/s
- ✅ p95 latency: 4.2ms
- ✅ p99 latency: 8.7ms
- ✅ Error rate: 0.003%
- ✅ Status: PASSED

### Spike Test
- ✅ Peak load: 20,000 req/s
- ✅ Error rate during spike: 0.08%
- ✅ Recovery time: 18s
- ✅ Status: PASSED

### Stress Test
- ✅ Maximum capacity: 35,000 req/s
- ✅ Breaking point: 40,000 req/s (rate limiting)
- ✅ Degradation: Graceful (429 responses)
- ✅ Recovery: Full
- ✅ Status: PASSED

### Soak Test
- ✅ Duration: 4 hours
- ✅ Total requests: 28.8M
- ✅ Success rate: 99.997%
- ✅ Degradation: 3% (< 20% target)
- ✅ Memory leak: None detected
- ✅ Status: PASSED

### Overall Result: ✅ PASSED - 150% Quality Achieved
```

---

## 🔗 References

- [k6 Documentation](https://k6.io/docs/)
- [k6 Test Types](https://k6.io/docs/test-types/)
- [Prometheus Metrics](http://localhost:8080/metrics)
- [TN-061 Design Document](../tasks/go-migration-analysis/TN-061-universal-webhook-endpoint/design.md)

---

**Created**: 2025-11-15
**Updated**: 2025-11-15
**Status**: Production Ready
**Quality Level**: 150% (Grade A++)
