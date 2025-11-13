# TN-057: Publishing Metrics & Stats - Requirements

## Executive Summary

Задача **TN-057** создает централизованную систему метрик и статистики для всего Publishing System (TN-046 до TN-060). Это критический компонент observability, который агрегирует метрики из 11+ источников (Discovery, Refresh, Health, Queue, Rootly, PagerDuty, Slack, Webhook publishers) и предоставляет unified API для мониторинга, алёртинга и dashboard visualization.

**Ключевые цели:**
- ✅ Централизованный aggregator для 50+ existing metrics
- ✅ Real-time statistics с minimal overhead (<50µs collection)
- ✅ HTTP API endpoints для Grafana/Prometheus/custom tooling
- ✅ Per-target analytics (success rate, latency, error trends)
- ✅ System-wide health score и SLA tracking
- ✅ Historical trends и anomaly detection ready
- ✅ 150% quality target (Grade A+, Production-Ready)

**Бизнес-обоснование:**
1. **Operational Excellence** - unified view of publishing pipeline health
2. **Proactive Monitoring** - catch issues before users notice (SLA 99.9%+)
3. **Capacity Planning** - queue depth, throughput, bottleneck identification
4. **Cost Optimization** - track retries, DLQ overhead, resource utilization
5. **Developer Experience** - single API for all publishing stats
6. **Compliance** - audit trail, error classification, SLI/SLO tracking

---

## 1. Functional Requirements (FR)

### FR-1: Centralized Metrics Collection
**Priority:** CRITICAL
**Description:** Aggregate metrics from all publishing subsystems into single source of truth.

**Details:**
- Collect from 11+ sources:
  1. **TN-046** K8s Client (4 metrics: secrets_discovered, api_calls, errors, duration)
  2. **TN-047** Target Discovery (6 metrics: targets by type, lookups, secrets, errors, duration, last_success)
  3. **TN-048** Target Refresh (5 metrics: total, duration, errors, last_success, in_progress)
  4. **TN-049** Health Monitoring (6 metrics: checks, duration, status, failures, success_rate, errors)
  5. **TN-056** Publishing Queue (17 metrics: queue_size, submissions, jobs_processed, duration, retries, circuit_breaker, workers, DLQ)
  6. **TN-052** Rootly (8 metrics: incidents_created/updated/resolved, api_requests, errors, cache_hits, rate_limit, duration)
  7. **TN-053** PagerDuty (8 metrics: events_published, errors, api_request_duration, cache_hits/misses, cache_size, rate_limit_hits)
  8. **TN-054** Slack (8 metrics: messages_posted, thread_replies, errors, api_request_duration, cache_hits/misses, cache_size, rate_limit)
  9. **TN-055** Generic Webhook (8 metrics: requests, duration, errors, payload_size, auth_failures, validation, timeouts, retries)

- **Total: 50+ metrics** across 9 subsystems
- Non-invasive collection (read-only access to existing Prometheus collectors)
- Real-time aggregation with <50µs overhead per metric read

**Acceptance Criteria:**
- [ ] PublishingMetrics struct with pointers to all subsystem metrics
- [ ] Zero metric duplication (no new Prometheus registrations)
- [ ] Lazy initialization (graceful handling if subsystem not initialized)
- [ ] Thread-safe concurrent reads
- [ ] Performance: <50µs to collect all 50+ metrics

---

### FR-2: Statistics Aggregation Service
**Priority:** CRITICAL
**Description:** Calculate aggregate statistics across all publishers and targets.

**Details:**
- **System-Wide Stats:**
  - Total targets discovered (by type: rootly, pagerduty, slack, webhook)
  - Total jobs processed (last 1m, 5m, 15m, 1h, 24h)
  - Overall success rate (percentage, 0-100)
  - Average processing duration (p50, p90, p95, p99)
  - Active queue depth (high/medium/low priority)
  - DLQ size (by target, by error_type)
  - Circuit breaker status (open/closed/halfopen counts)
  - Worker utilization (active/idle ratio)

- **Per-Target Stats:**
  - Health status (healthy/unhealthy/degraded/unknown)
  - Success rate (last 1h, 24h, 7d)
  - Average latency (last 1h)
  - Error rate (by error_type: timeout/auth/network/http)
  - Retry count (total, success rate)
  - Last successful publish timestamp
  - Consecutive failures count
  - Cache hit rate (if applicable)

- **Trend Analysis:**
  - Success rate trend (increasing/stable/decreasing)
  - Latency trend (improving/stable/degrading)
  - Error spike detection (sudden increase > 3x baseline)
  - Queue growth rate (alerts if > 100 jobs/min)

**Acceptance Criteria:**
- [ ] PublishingStats struct with system-wide and per-target stats
- [ ] CalculateStats() method with <5ms execution time
- [ ] Historical data support (last 1m, 5m, 15m, 1h, 24h windows)
- [ ] Trend detection with configurable thresholds
- [ ] JSON serializable for HTTP API

---

### FR-3: HTTP API Endpoints
**Priority:** HIGH
**Description:** Expose metrics and stats through REST API for dashboards and monitoring.

**Details:**
- **Endpoints:**
  1. `GET /api/v2/publishing/metrics` - Raw Prometheus metrics export (OpenMetrics format)
  2. `GET /api/v2/publishing/stats` - Aggregated statistics (JSON)
  3. `GET /api/v2/publishing/stats/{target}` - Per-target statistics (JSON)
  4. `GET /api/v2/publishing/health` - System health summary (JSON)
  5. `GET /api/v2/publishing/trends` - Historical trends (JSON)

- **Response Format (JSON):**
```json
{
  "timestamp": "2025-11-12T10:30:00Z",
  "system": {
    "total_targets": 23,
    "healthy_targets": 20,
    "unhealthy_targets": 2,
    "degraded_targets": 1,
    "total_jobs_processed": 15420,
    "success_rate": 98.5,
    "average_duration_ms": 245,
    "queue_depth": {"high": 5, "medium": 12, "low": 3},
    "dlq_size": 7,
    "workers_active": 8,
    "workers_idle": 2
  },
  "targets": [
    {
      "name": "rootly-prod",
      "type": "rootly",
      "health": "healthy",
      "success_rate": 99.2,
      "avg_latency_ms": 180,
      "error_rate": 0.8,
      "retry_count": 12,
      "last_success": "2025-11-12T10:29:45Z",
      "consecutive_failures": 0
    }
  ],
  "trends": {
    "success_rate_trend": "stable",
    "latency_trend": "improving",
    "error_spike_detected": false,
    "queue_growth_rate": 2.5
  }
}
```

**Acceptance Criteria:**
- [ ] 5 HTTP endpoints with comprehensive responses
- [ ] OpenAPI 3.0 specification
- [ ] Response time <5ms (cached stats)
- [ ] Pagination support for /stats (limit/offset)
- [ ] Filtering support (by target_type, health_status)
- [ ] CORS headers for dashboard access
- [ ] Content-Type: application/json + application/openmetrics-text

---

### FR-4: Health Score Calculation
**Priority:** HIGH
**Description:** Calculate overall publishing system health score (0-100).

**Details:**
- **Health Score Formula:**
```
Health Score = (
  0.4 * SuccessRate +
  0.3 * AvailabilityScore +
  0.2 * PerformanceScore +
  0.1 * QueueHealthScore
)

Where:
- SuccessRate = (successful_jobs / total_jobs) * 100
- AvailabilityScore = (healthy_targets / total_targets) * 100
- PerformanceScore = 100 * (1 - p95_latency / max_acceptable_latency)
- QueueHealthScore = 100 * (1 - queue_depth / queue_capacity)
```

- **Health Status Mapping:**
  - 90-100: HEALTHY (🟢)
  - 70-89: DEGRADED (🟡)
  - 0-69: UNHEALTHY (🔴)

- **SLA Tracking:**
  - Target SLA: 99.9% success rate
  - Alert if health score < 90 for 5+ minutes
  - Track SLA violations (count, duration, impact)

**Acceptance Criteria:**
- [ ] CalculateHealthScore() returns 0-100 score
- [ ] Weighted formula с configurable weights
- [ ] SLA violation tracking
- [ ] Historical health score (last 24h, 7d, 30d)
- [ ] Health status enum (HEALTHY/DEGRADED/UNHEALTHY)

---

### FR-5: Per-Target Analytics
**Priority:** MEDIUM
**Description:** Detailed analytics per publishing target.

**Details:**
- **Metrics per target:**
  - Total publishes (lifetime, last 24h)
  - Success rate (percentage, 0-100)
  - Average latency (ms, p50/p90/p95/p99)
  - Error breakdown (by error_type with counts)
  - Retry statistics (attempts, success rate)
  - Cache performance (hit rate if applicable)
  - Circuit breaker trips (count, last trip timestamp)
  - DLQ entries (count, oldest entry age)

- **Comparison features:**
  - Compare targets by type (e.g., all rootly targets)
  - Best/worst performers identification
  - Outlier detection (3σ from mean)

**Acceptance Criteria:**
- [ ] TargetAnalytics struct per target
- [ ] Comparison API endpoint
- [ ] Outlier detection algorithm
- [ ] Performance ranking

---

### FR-6: Historical Trends & Time Series
**Priority:** MEDIUM
**Description:** Track metrics over time for trend analysis.

**Details:**
- **Time Windows:**
  - Last 1 minute (real-time)
  - Last 5 minutes
  - Last 15 minutes
  - Last 1 hour
  - Last 24 hours
  - Last 7 days

- **Tracked Metrics:**
  - Success rate over time
  - Latency percentiles (p50, p90, p95, p99)
  - Error rate by type
  - Queue depth
  - Throughput (jobs/min)

- **Trend Detection:**
  - Increasing/stable/decreasing classification
  - Rate of change calculation
  - Anomaly detection (>3σ deviation)
  - Seasonality detection (daily/weekly patterns)

**Acceptance Criteria:**
- [ ] Time series data structure (timestamp, value pairs)
- [ ] Sliding window calculations
- [ ] Trend classification algorithm
- [ ] Anomaly detection with configurable thresholds

---

## 2. Non-Functional Requirements (NFR)

### NFR-1: Performance
- **Metrics Collection:** <50µs to read all 50+ metrics
- **Stats Calculation:** <5ms for full system stats
- **HTTP Endpoints:** <10ms response time (p95)
- **Memory Overhead:** <5MB for stats aggregation
- **CPU Usage:** <1% during normal operation

### NFR-2: Reliability
- **Uptime:** 99.99% (metrics collection should never block publishers)
- **Graceful Degradation:** Continue if individual subsystems unavailable
- **No Single Point of Failure:** Each subsystem metrics independent
- **Recovery:** Auto-recover from transient failures

### NFR-3: Scalability
- **Targets:** Support 100+ publishing targets
- **Metrics:** Handle 100+ Prometheus metrics
- **Throughput:** 10,000+ stats requests/sec
- **Historical Data:** Store 7 days of trends (with downsampling)

### NFR-4: Observability
- **Logging:** Structured logging (slog) at DEBUG/INFO/WARN/ERROR levels
- **Metrics:** Self-monitoring metrics (collection duration, API response time, error rate)
- **Tracing:** OpenTelemetry-compatible spans
- **Health Checks:** /health endpoint with detailed component status

### NFR-5: Maintainability
- **Code Quality:** 90%+ test coverage, zero linter errors
- **Documentation:** Comprehensive README, API guide, PromQL examples
- **Modularity:** Clear separation between collection/aggregation/API layers
- **Extensibility:** Easy to add new metrics sources

### NFR-6: Security
- **Authentication:** Optional API key для production endpoints
- **Rate Limiting:** 100 req/sec per client (configurable)
- **Input Validation:** Sanitize query parameters (target names, filters)
- **No Sensitive Data:** Metrics should not contain secrets/tokens

---

## 3. Dependencies

### Upstream Dependencies (Required)
- ✅ **TN-046** K8s Client (metrics: secrets_discovered, api_calls, errors)
- ✅ **TN-047** Target Discovery (metrics: targets, lookups, duration)
- ✅ **TN-048** Target Refresh (metrics: refresh_total, duration, errors)
- ✅ **TN-049** Health Monitoring (metrics: checks, status, failures)
- ✅ **TN-056** Publishing Queue (metrics: queue_size, jobs_processed, retries, DLQ)
- ✅ **TN-052** Rootly Publisher (metrics: incidents, errors, cache)
- ✅ **TN-053** PagerDuty Publisher (metrics: events, errors, cache)
- ✅ **TN-054** Slack Publisher (metrics: messages, errors, cache)
- ✅ **TN-055** Generic Webhook (metrics: requests, errors, timeouts)

**Status:** ALL COMPLETE (TN-046 to TN-056 completed with 147%-177% quality)

### Downstream Dependencies (Blocked Tasks)
- **TN-058** Parallel Publishing (will use TN-057 stats for optimization)
- **TN-059** Publishing API Endpoints (will expose TN-057 data)
- **TN-060** Metrics-Only Mode (fallback mode using TN-057 health checks)

### External Dependencies
- Prometheus Go Client (`github.com/prometheus/client_golang`)
- Go standard library (`encoding/json`, `net/http`, `sync`, `time`)
- Project metrics package (`internal/infrastructure/metrics`)

---

## 4. Constraints & Limitations

### Technical Constraints
1. **No New Metrics Registration** - Must reuse existing Prometheus metrics (read-only access)
2. **Backward Compatibility** - Cannot break existing publisher interfaces
3. **Zero Performance Impact** - Publishers should not block waiting for stats collection
4. **Memory Budget** - <10MB total for all stats data structures
5. **Go Version** - Must support Go 1.22+ (project standard)

### Business Constraints
1. **Timeline** - Target 2-3 days implementation (150% quality target)
2. **Zero Downtime** - Deploy without restarting publishers
3. **Production-Ready** - Must be deployable immediately after merge

### Known Limitations
1. **Historical Data** - Limited to 7 days (longer-term storage via Prometheus/VictoriaMetrics)
2. **Real-Time Lag** - Up to 1s delay for stats updates (acceptable for monitoring)
3. **No Multi-Cluster** - Single-cluster stats only (Phase 2 feature)

---

## 5. Risks & Mitigations

### Risk 1: Performance Overhead
**Impact:** HIGH
**Probability:** MEDIUM
**Mitigation:**
- Lazy initialization of stats collection
- Cache calculated stats with 1s TTL
- Concurrent metric collection with sync.WaitGroup
- Benchmark all hot paths (<50µs target)

### Risk 2: Incomplete Metrics Coverage
**Impact:** MEDIUM
**Probability:** LOW
**Mitigation:**
- Comprehensive audit of existing metrics (Phase 2)
- Graceful handling of missing metrics (nil checks)
- Fallback to "unknown" status if subsystem unavailable

### Risk 3: API Endpoint Abuse
**Impact:** MEDIUM
**Probability:** MEDIUM
**Mitigation:**
- Rate limiting (100 req/sec per client)
- Optional authentication for production
- Response caching (1s TTL)
- Request timeout (5s max)

### Risk 4: Memory Leaks
**Impact:** HIGH
**Probability:** LOW
**Mitigation:**
- Bounded data structures (max 100 targets, 7d history)
- Periodic cleanup worker (prune old data every 1h)
- Memory profiling in tests
- Heap dump on >50MB usage

### Risk 5: Integration Complexity
**Impact:** MEDIUM
**Probability:** MEDIUM
**Mitigation:**
- Clear interface contracts (PublishingMetricsCollector)
- Comprehensive integration tests
- Phased rollout (start with read-only access)
- Rollback plan (feature flag to disable)

---

## 6. Acceptance Criteria (150% Quality Target)

### Core Features (100%)
- [ ] Centralized metrics collection from 9+ subsystems
- [ ] Statistics aggregation with <5ms calculation time
- [ ] 5 HTTP API endpoints with JSON responses
- [ ] Health score calculation (0-100)
- [ ] Per-target analytics
- [ ] OpenAPI 3.0 specification

### Extended Features (150% Target)
- [ ] Historical trends (7d retention)
- [ ] Anomaly detection (>3σ deviation)
- [ ] Comparison API (best/worst performers)
- [ ] SLA tracking (99.9% target)
- [ ] Self-monitoring metrics (collection_duration, api_response_time)
- [ ] Grafana dashboard JSON templates
- [ ] PromQL query examples (10+ queries)
- [ ] Load testing (10k req/sec sustained)

### Quality Metrics
- [ ] **Test Coverage:** 90%+ (unit + integration)
- [ ] **Performance:** All targets exceeded by 2-5x
- [ ] **Documentation:** 3,000+ LOC (README, API guide, examples)
- [ ] **Zero Technical Debt:** All TODOs resolved, no deprecated code
- [ ] **Production-Ready:** Deployed to staging with zero issues

---

## 7. Success Metrics

### Quantitative KPIs
1. **Response Time:** p95 < 5ms for /stats endpoint
2. **Throughput:** 10,000+ req/sec sustained load
3. **Availability:** 99.99% uptime (metrics collection)
4. **Coverage:** 50+ metrics from 9+ subsystems
5. **Adoption:** Used by 3+ dashboards (Grafana, custom, CLI)

### Qualitative Goals
1. **Developer Experience:** Single API for all publishing stats (no manual Prometheus queries)
2. **Operational Excellence:** Proactive issue detection (alert before failure)
3. **Cost Efficiency:** Reduced MTTR (Mean Time To Resolve) by 50%
4. **Compliance:** Audit-ready metrics trail

---

## 8. Out of Scope (Future Enhancements)

### Phase 2 Features (Post-MVP)
- Multi-cluster metrics aggregation
- Long-term storage (> 7 days via external TSDB)
- Machine learning-based anomaly detection
- Auto-scaling recommendations based on metrics
- Cost attribution per target
- GraphQL API (alternative to REST)
- WebSocket streaming for real-time updates
- Custom alerting rules engine

### Not Planned
- Metrics storage (use Prometheus/VictoriaMetrics)
- Dashboard UI (use Grafana)
- Log aggregation (separate concern)
- Distributed tracing (use OpenTelemetry directly)

---

## 9. References

### Related Documentation
- TN-046 K8s Client README
- TN-047 Target Discovery Design
- TN-048 Target Refresh Design
- TN-049 Health Monitoring README
- TN-056 Publishing Queue Design (17 metrics specification)
- Prometheus Best Practices: https://prometheus.io/docs/practices/naming/
- OpenMetrics Specification: https://openmetrics.io/

### Code References
- `go-app/internal/business/publishing/health_metrics.go` (6 metrics)
- `go-app/internal/business/publishing/refresh_metrics.go` (5 metrics)
- `go-app/internal/business/publishing/discovery_impl.go` (6 metrics)
- `tasks/go-migration-analysis/TN-056-publishing-queue-retry/design.md` (17 metrics)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-12
**Author:** AI Assistant
**Status:** DRAFT (awaiting Phase 1 completion)
**Next Review:** After Phase 2 (Gap Analysis)
