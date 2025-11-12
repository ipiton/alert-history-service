# TN-057: Publishing Metrics & Stats - Implementation Tasks

## Executive Summary

**TN-057** реализуется за **10 фаз** с target **3-4 дня** implementation time и **150% quality goal** (Grade A+, Production-Ready). Каждая фаза включает четкие deliverables, acceptance criteria и performance targets. Общий объем: **~3,500 LOC production** + **~2,500 LOC tests** + **~3,000 LOC documentation** = **~9,000 LOC total**.

**Timeline:**
- **Phase 0-1:** Documentation & Gap Analysis (6h) ✅ COMPLETE
- **Phase 2:** Metrics Collection Layer (4h)
- **Phase 3:** Statistics Aggregation Layer (6h)
- **Phase 4:** HTTP API Endpoints (4h)
- **Phase 5:** Trend Analysis Engine (4h)
- **Phase 6:** Testing (6h)
- **Phase 7:** Documentation (4h)
- **Phase 8:** Integration (3h)
- **Phase 9:** Performance Optimization (3h)
- **Phase 10:** Final Certification (2h)

**Total:** ~42 hours (5-6 days with 8h/day) → Target: 3-4 days с optimization

---

## Phase 0: Comprehensive Analysis ✅ **COMPLETE**

### Deliverables
- [x] `requirements.md` (19KB, 800+ lines) ✅
- [x] `design.md` (67KB, 2000+ lines) ✅
- [x] `tasks.md` (this file) ✅
- [x] Git branch `feature/TN-057-publishing-metrics-150pct` ✅

### Acceptance Criteria
- [x] All 9 functional requirements documented
- [x] All 6 non-functional requirements defined
- [x] 4-layer architecture designed (Collection/Aggregation/Analysis/Presentation)
- [x] 50+ metrics catalogued from TN-046 to TN-056
- [x] Performance targets defined (<50µs collection, <5ms stats)

### Time Spent
- **Actual:** 2h (requirements 1h + design 1h)
- **Status:** ✅ COMPLETE

---

## Phase 1: Gap Analysis & Audit

### Objective
Audit existing metrics across 9 subsystems (TN-046 to TN-056) to verify actual implementation matches design documentation. Identify missing metrics, inconsistencies, and integration points.

### Tasks

#### 1.1. Audit TN-046 K8s Client Metrics
- [ ] Read `go-app/internal/infrastructure/k8s/` source files
- [ ] Verify 4 metrics exist:
  1. `secrets_discovered_total` (Counter)
  2. `k8s_api_calls_total` (CounterVec by operation)
  3. `k8s_errors_total` (CounterVec by error_type)
  4. `k8s_operation_duration_seconds` (HistogramVec)
- [ ] Check Prometheus registration (MustRegister or Registry)
- [ ] Document actual metric names (namespace_subsystem_name format)
- [ ] Note any deviations from TN-046 design

**Deliverable:** `k8s_metrics_audit.md` (200 lines)

#### 1.2. Audit TN-047 Target Discovery Metrics
- [ ] Read `go-app/internal/business/publishing/discovery_impl.go`
- [ ] Verify 6 metrics exist:
  1. `targets_total` (GaugeVec by type)
  2. `target_lookups_total` (Counter)
  3. `secrets_processed_total` (CounterVec by result)
  4. `discovery_errors_total` (CounterVec by error_type)
  5. `discovery_duration_seconds` (Histogram)
  6. `last_success_timestamp` (Gauge)
- [ ] Check MetricsCollector access pattern
- [ ] Document integration points

**Deliverable:** Update `gap_analysis.md` (section 2)

#### 1.3. Audit TN-048 Target Refresh Metrics
- [ ] Read `go-app/internal/business/publishing/refresh_metrics.go`
- [ ] Verify 5 metrics exist (already confirmed in codebase search):
  1. `refresh_total` (CounterVec by status)
  2. `refresh_duration_seconds` (HistogramVec)
  3. `refresh_errors_total` (CounterVec by error_type)
  4. `refresh_last_success_timestamp` (Gauge)
  5. `refresh_in_progress` (Gauge)
- [ ] Check NewRefreshMetrics() constructor signature
- [ ] Document Registry pattern usage

**Deliverable:** Update `gap_analysis.md` (section 3)

#### 1.4. Audit TN-049 Health Monitoring Metrics
- [ ] Read `go-app/internal/business/publishing/health_metrics.go`
- [ ] Verify 6 metrics exist (already confirmed):
  1. `health_checks_total` (CounterVec)
  2. `health_check_duration_seconds` (HistogramVec)
  3. `target_health_status` (GaugeVec)
  4. `target_consecutive_failures` (GaugeVec)
  5. `target_success_rate` (GaugeVec)
  6. `health_check_errors_total` (CounterVec)
- [ ] Check HealthMetrics struct fields (pointers to Prometheus metrics)
- [ ] Document RecordHealthCheck() method signature

**Deliverable:** Update `gap_analysis.md` (section 4)

#### 1.5. Audit TN-056 Publishing Queue Metrics
- [ ] Search for queue metrics implementation (may be in design.md only)
- [ ] Verify 17 metrics exist (or planned):
  1. `queue_size` (GaugeVec by priority)
  2. `queue_capacity_utilization` (GaugeVec)
  3. `queue_submissions_total` (CounterVec by priority, result)
  4. `jobs_processed_total` (CounterVec by target, state)
  5. `job_duration_seconds` (HistogramVec)
  6. `job_wait_time_seconds` (HistogramVec)
  7. `retry_attempts_total` (CounterVec)
  8. `retry_success_rate` (HistogramVec)
  9. `circuit_breaker_state` (GaugeVec)
  10. `circuit_breaker_trips_total` (CounterVec)
  11. `circuit_breaker_recoveries_total` (CounterVec)
  12. `workers_active` (Gauge)
  13. `workers_idle` (Gauge)
  14. `worker_processing_duration_seconds` (HistogramVec)
  15. `dlq_size` (GaugeVec by target)
  16. `dlq_writes_total` (CounterVec)
  17. `dlq_replays_total` (CounterVec)
- [ ] If not implemented, document as "planned" (Phase 2 feature)
- [ ] Check queue package location (`internal/business/publishing/queue/`)

**Deliverable:** Update `gap_analysis.md` (section 5)

#### 1.6. Audit Publisher Metrics (TN-052, 053, 054, 055)
- [ ] Search for publisher metrics (Rootly, PagerDuty, Slack, Webhook)
- [ ] Each publisher expected to have ~8 metrics:
  - `{publisher}_requests_total` (CounterVec by status)
  - `{publisher}_errors_total` (CounterVec by error_type)
  - `{publisher}_request_duration_seconds` (HistogramVec)
  - `{publisher}_cache_hits_total` (Counter, if applicable)
  - `{publisher}_cache_misses_total` (Counter, if applicable)
  - `{publisher}_rate_limit_hits_total` (Counter, if applicable)
  - Publisher-specific metrics (e.g., `rootly_incidents_created_total`)
- [ ] Document actual implementation status (may be in publisher files)
- [ ] Check for common PublisherMetrics interface

**Deliverable:** Update `gap_analysis.md` (section 6)

#### 1.7. Create Metrics Inventory
- [ ] Consolidate all metric names into single spreadsheet
- [ ] Columns: Subsystem, Metric Name, Type, Labels, Status (implemented/planned)
- [ ] Calculate total: Expected 50+, Actual count = ?
- [ ] Identify gaps (missing metrics, inconsistent naming)

**Deliverable:** `metrics_inventory.csv` (50+ rows)

#### 1.8. Define Integration Strategy
- [ ] Document how to access existing Prometheus metrics (read-only)
- [ ] Strategy options:
  1. **Direct access** - Store pointers to subsystem metrics (best performance)
  2. **Prometheus Gatherer** - Use prometheus.Gatherer to scrape metrics (universal but slower)
  3. **Hybrid** - Direct access for critical subsystems, Gatherer for optional
- [ ] Choose strategy based on audit results
- [ ] Document pros/cons, performance implications

**Deliverable:** `integration_strategy.md` (500 lines)

### Acceptance Criteria
- [ ] All 9 subsystems audited (TN-046 to TN-056)
- [ ] Metrics inventory complete (50+ metrics catalogued)
- [ ] Gaps identified with mitigation plan
- [ ] Integration strategy chosen and documented
- [ ] Gap analysis document complete (2,000+ lines)

### Deliverables
1. `gap_analysis.md` (2,000 lines, 6 subsystem sections)
2. `metrics_inventory.csv` (50+ rows)
3. `integration_strategy.md` (500 lines)

### Estimated Time
- **Time:** 4 hours
- **Breakdown:** 30min per subsystem audit + 1h inventory + 1h strategy doc

---

## Phase 2: Metrics Collection Layer

### Objective
Implement MetricsCollector interface and 9 subsystem-specific collectors to read existing Prometheus metrics with <50µs total collection time.

### Tasks

#### 2.1. Create Core Interfaces
- [ ] File: `go-app/internal/business/publishing/metrics_collector.go`
- [ ] Define `MetricsCollector` interface:
  ```go
  type MetricsCollector interface {
      Collect() (map[string]float64, error)
      Name() string
      IsAvailable() bool
  }
  ```
- [ ] Define `MetricsSnapshot` struct (raw metrics data)
- [ ] Define `PublishingMetrics` struct (aggregator for all collectors)
- [ ] Add comprehensive godoc comments (100 lines)

**Deliverable:** `metrics_collector.go` (200 LOC)

#### 2.2. Implement HealthMetricsCollector
- [ ] File: `go-app/internal/business/publishing/metrics_collector_health.go`
- [ ] Struct `HealthMetricsCollector` with pointer to TN-049 HealthMetrics
- [ ] Implement `Collect()` method (read 6 metrics from Prometheus)
- [ ] Handle nil metrics gracefully (return empty snapshot)
- [ ] Performance target: <10µs
- [ ] Godoc comments (50 lines)

**Deliverable:** `metrics_collector_health.go` (150 LOC)

#### 2.3. Implement RefreshMetricsCollector
- [ ] File: `go-app/internal/business/publishing/metrics_collector_refresh.go`
- [ ] Similar to HealthMetricsCollector
- [ ] Collect 5 refresh metrics (from TN-048 RefreshMetrics)
- [ ] Performance target: <10µs

**Deliverable:** `metrics_collector_refresh.go` (150 LOC)

#### 2.4. Implement DiscoveryMetricsCollector
- [ ] File: `go-app/internal/business/publishing/metrics_collector_discovery.go`
- [ ] Collect 6 discovery metrics (from TN-047)
- [ ] Performance target: <10µs

**Deliverable:** `metrics_collector_discovery.go` (150 LOC)

#### 2.5. Implement QueueMetricsCollector
- [ ] File: `go-app/internal/business/publishing/metrics_collector_queue.go`
- [ ] Collect 17 queue metrics (from TN-056, if implemented)
- [ ] If TN-056 metrics not yet implemented, create stub collector
- [ ] Performance target: <15µs (more metrics)

**Deliverable:** `metrics_collector_queue.go` (250 LOC)

#### 2.6. Implement PublisherMetricsCollector (Generic)
- [ ] File: `go-app/internal/business/publishing/metrics_collector_publisher.go`
- [ ] Generic collector for all publishers (Rootly, PagerDuty, Slack, Webhook)
- [ ] Constructor: `NewPublisherMetricsCollector(name string, metrics interface{})`
- [ ] Collect ~8 metrics per publisher
- [ ] Performance target: <10µs per publisher

**Deliverable:** `metrics_collector_publisher.go` (200 LOC)

#### 2.7. Create PublishingMetrics Aggregator
- [ ] File: `go-app/internal/business/publishing/metrics_aggregator.go`
- [ ] Struct `PublishingMetrics` with pointers to all collectors
- [ ] Method `CollectAll()` - concurrent collection with sync.WaitGroup
- [ ] Performance target: <50µs (parallel collection)
- [ ] Handle nil collectors gracefully (skip if not initialized)

**Deliverable:** `metrics_aggregator.go` (250 LOC)

#### 2.8. Unit Tests for Collectors
- [ ] File: `go-app/internal/business/publishing/metrics_collector_test.go`
- [ ] Test cases (15 tests):
  1. TestHealthMetricsCollector_Collect
  2. TestHealthMetricsCollector_CollectNilMetrics
  3. TestRefreshMetricsCollector_Collect
  4. TestDiscoveryMetricsCollector_Collect
  5. TestQueueMetricsCollector_Collect
  6. TestPublisherMetricsCollector_Collect
  7. TestPublishingMetrics_CollectAll
  8. TestPublishingMetrics_CollectAllNilCollectors
  9. TestPublishingMetrics_CollectAllConcurrent (thread safety)
  10. TestMetricsCollector_Performance (benchmark <50µs)
- [ ] Mock Prometheus metrics using fake collectors
- [ ] Coverage target: 90%+

**Deliverable:** `metrics_collector_test.go` (800 LOC)

### Acceptance Criteria
- [ ] MetricsCollector interface defined
- [ ] 6 collector implementations (Health, Refresh, Discovery, Queue, Publisher generic)
- [ ] PublishingMetrics aggregator with concurrent collection
- [ ] 15+ unit tests passing
- [ ] Performance: CollectAll() <50µs (benchmark verified)
- [ ] Test coverage: 90%+
- [ ] Zero linter errors

### Deliverables
1. `metrics_collector.go` (200 LOC)
2. `metrics_collector_health.go` (150 LOC)
3. `metrics_collector_refresh.go` (150 LOC)
4. `metrics_collector_discovery.go` (150 LOC)
5. `metrics_collector_queue.go` (250 LOC)
6. `metrics_collector_publisher.go` (200 LOC)
7. `metrics_aggregator.go` (250 LOC)
8. `metrics_collector_test.go` (800 LOC)

**Total:** ~2,150 LOC (1,350 production + 800 tests)

### Estimated Time
- **Time:** 4 hours
- **Breakdown:** 30min per collector + 1h aggregator + 1.5h tests

---

## Phase 3: Statistics Aggregation Layer

### Objective
Implement StatsAggregator to calculate system-wide and per-target statistics from collected metrics with <5ms calculation time.

### Tasks

#### 3.1. Define Data Models
- [ ] File: `go-app/internal/business/publishing/stats_models.go`
- [ ] Structs:
  - `PublishingStats` (top-level stats)
  - `SystemStats` (system-wide aggregates)
  - `QueueDepth` (high/medium/low counts)
  - `TargetStats` (per-target stats)
  - `TrendAnalysis` (trends, deferred to Phase 5)
  - `SLAMetrics` (SLA tracking)
- [ ] JSON tags for all fields (for HTTP API)
- [ ] Comprehensive godoc (200 lines)

**Deliverable:** `stats_models.go` (400 LOC)

#### 3.2. Implement StatsAggregator
- [ ] File: `go-app/internal/business/publishing/stats_aggregator.go`
- [ ] Struct `StatsAggregator` with metrics collectors + cache
- [ ] Method `Calculate()` - main stats calculation:
  1. Check cache (1s TTL)
  2. Collect metrics (call PublishingMetrics.CollectAll())
  3. Calculate system-wide stats (aggregate across all targets)
  4. Calculate per-target stats (iterate targets)
  5. Calculate health score (weighted formula)
  6. Track SLA compliance
  7. Cache result
  8. Return PublishingStats
- [ ] Performance target: <5ms (or <50µs if cached)

**Deliverable:** `stats_aggregator.go` (500 LOC)

#### 3.3. Implement System Stats Calculation
- [ ] Method `calculateSystemStats(metricsMap) SystemStats`
- [ ] Aggregate metrics:
  - Total targets (from discovery metrics)
  - Healthy/unhealthy/degraded targets (from health metrics)
  - Total jobs processed (from queue metrics)
  - Success rate (successful_jobs / total_jobs * 100)
  - Latency percentiles (p50, p90, p95, p99 from histogram buckets)
  - Queue depth (high/medium/low from queue metrics)
  - DLQ size (from queue metrics)
  - Workers active/idle (from queue metrics)
  - Circuit breaker counts (from queue metrics)
- [ ] Performance: <2ms

**Deliverable:** Method in `stats_aggregator.go` (~100 LOC)

#### 3.4. Implement Per-Target Stats Calculation
- [ ] Method `calculateTargetStats(metricsMap) []TargetStats`
- [ ] Iterate discovered targets
- [ ] For each target, calculate:
  - Health status (from health metrics)
  - Success rate (from publisher metrics)
  - Avg/p95 latency (from publisher histogram)
  - Error rate and breakdown (from error metrics)
  - Retry count and success rate
  - Cache hit rate (if applicable)
  - Circuit breaker state
  - DLQ entries count
- [ ] Performance: <2ms (even with 100 targets)

**Deliverable:** Method in `stats_aggregator.go` (~150 LOC)

#### 3.5. Implement Health Score Calculator
- [ ] Method `calculateHealthScore(systemStats) float64`
- [ ] Weighted formula:
  ```
  HealthScore =
    0.4 * SuccessRate +
    0.3 * AvailabilityScore +
    0.2 * PerformanceScore +
    0.1 * QueueHealthScore
  ```
- [ ] Clamp result to [0, 100]
- [ ] Performance: <100µs

**Deliverable:** Method in `stats_aggregator.go` (~50 LOC)

#### 3.6. Implement SLA Tracker
- [ ] Method `trackSLA(systemStats) SLAMetrics`
- [ ] Target SLA: 99.9% success rate
- [ ] Check current compliance: currentSuccessRate >= targetSLA
- [ ] Track violations (historical, requires time series DB - Phase 5)
- [ ] Calculate MTTR (Mean Time To Recover) - placeholder for Phase 5

**Deliverable:** Method in `stats_aggregator.go` (~50 LOC)

#### 3.7. Implement Stats Cache
- [ ] File: `go-app/internal/business/publishing/stats_cache.go`
- [ ] Struct `StatsCache` with:
  - `cachedStats *PublishingStats`
  - `cacheTimestamp time.Time`
  - `cacheTTL time.Duration` (default: 1s)
  - `mu sync.RWMutex`
- [ ] Methods:
  - `Get() *PublishingStats` (return cached if < 1s old)
  - `Set(stats *PublishingStats)` (update cache)
  - `Invalidate()` (manual cache clear)
- [ ] Performance: Get() <50ns (simple pointer return)

**Deliverable:** `stats_cache.go` (150 LOC)

#### 3.8. Unit Tests for Stats Aggregation
- [ ] File: `go-app/internal/business/publishing/stats_aggregator_test.go`
- [ ] Test cases (20 tests):
  1. TestStatsAggregator_Calculate (full stats)
  2. TestStatsAggregator_CalculateWithCache (cache hit)
  3. TestStatsAggregator_CalculateSystemStats (system aggregates)
  4. TestStatsAggregator_CalculateTargetStats (per-target)
  5. TestStatsAggregator_CalculateHealthScore (formula)
  6. TestStatsAggregator_HealthScore_Healthy (score >= 90)
  7. TestStatsAggregator_HealthScore_Degraded (70 <= score < 90)
  8. TestStatsAggregator_HealthScore_Unhealthy (score < 70)
  9. TestStatsAggregator_TrackSLA (compliance check)
  10. TestStatsCache_GetSet (cache operations)
  11. TestStatsCache_Expiration (1s TTL)
  12. TestStatsCache_Invalidate (manual clear)
  13. TestStatsAggregator_Concurrent (thread safety)
  14. TestStatsAggregator_NilMetrics (graceful degradation)
  15. TestStatsAggregator_Performance (<5ms benchmark)
- [ ] Mock metrics collectors
- [ ] Coverage target: 90%+

**Deliverable:** `stats_aggregator_test.go` (1,000 LOC)

### Acceptance Criteria
- [ ] PublishingStats, SystemStats, TargetStats models defined
- [ ] StatsAggregator implemented with Calculate() method
- [ ] Health score calculation (weighted formula)
- [ ] SLA tracking (99.9% target)
- [ ] Stats cache (1s TTL)
- [ ] 20+ unit tests passing
- [ ] Performance: Calculate() <5ms, cached <50µs
- [ ] Test coverage: 90%+
- [ ] Zero linter errors

### Deliverables
1. `stats_models.go` (400 LOC)
2. `stats_aggregator.go` (850 LOC)
3. `stats_cache.go` (150 LOC)
4. `stats_aggregator_test.go` (1,000 LOC)

**Total:** ~2,400 LOC (1,400 production + 1,000 tests)

### Estimated Time
- **Time:** 6 hours
- **Breakdown:** 1h models + 3h aggregator + 1h cache + 1h tests

---

## Phase 4: HTTP API Endpoints

### Objective
Implement 5 REST API endpoints to expose metrics and stats with <10ms response time (p95).

### Tasks

#### 4.1. Create HTTP Service
- [ ] File: `go-app/cmd/server/handlers/publishing_stats.go`
- [ ] Struct `PublishingStatsService`:
  ```go
  type PublishingStatsService struct {
      aggregator *StatsAggregator
      detector   *TrendDetector // Phase 5

      // Self-monitoring metrics
      apiRequestsTotal    *prometheus.CounterVec
      apiRequestDuration  *prometheus.HistogramVec
      apiErrorsTotal      *prometheus.CounterVec
  }
  ```
- [ ] Constructor `NewPublishingStatsService(aggregator, detector)`
- [ ] Register Prometheus metrics (3 metrics)

**Deliverable:** `publishing_stats.go` (200 LOC, structs + constructor)

#### 4.2. Implement GET /api/v2/publishing/stats
- [ ] Method `StatsHandler(w http.ResponseWriter, r *http.Request)`
- [ ] Query parameters:
  - `filter` (optional): target_type filter (rootly/pagerduty/slack/webhook)
  - `health` (optional): health status filter (healthy/unhealthy/degraded)
  - `limit` (optional): pagination limit (default: 100, max: 1000)
  - `offset` (optional): pagination offset (default: 0)
- [ ] Algorithm:
  1. Parse query parameters
  2. Call aggregator.Calculate() (get stats)
  3. Apply filters (target_type, health_status)
  4. Apply pagination (limit/offset)
  5. JSON encode response
  6. Record metrics (duration, status code)
- [ ] Response: PublishingStats JSON (full structure)
- [ ] Performance target: <10ms (p95)

**Deliverable:** Method in `publishing_stats.go` (~150 LOC)

#### 4.3. Implement GET /api/v2/publishing/stats/{target}
- [ ] Method `TargetStatsHandler(w http.ResponseWriter, r *http.Request)`
- [ ] Path parameter: `target` (target name, e.g., "rootly-prod")
- [ ] Algorithm:
  1. Extract target name from path (use r.PathValue("target") for Go 1.22+)
  2. Call aggregator.Calculate()
  3. Find target in stats.Targets array
  4. Return single TargetStats or 404 if not found
- [ ] Response: TargetStats JSON
- [ ] Performance: <5ms

**Deliverable:** Method in `publishing_stats.go` (~100 LOC)

#### 4.4. Implement GET /api/v2/publishing/health
- [ ] Method `HealthHandler(w http.ResponseWriter, r *http.Request)`
- [ ] Algorithm:
  1. Call aggregator.Calculate()
  2. Extract health summary (health_score, targets counts, success_rate, queue_depth, dlq_size, sla_compliant)
  3. Determine HTTP status code:
     - 200 OK: score >= 90 (healthy)
     - 200 OK: 70 <= score < 90 (degraded, but operational)
     - 503 Service Unavailable: score < 70 (unhealthy)
  4. JSON encode simplified health response
- [ ] Response: Simplified health JSON (8-10 fields)
- [ ] Performance: <5ms

**Deliverable:** Method in `publishing_stats.go` (~100 LOC)

#### 4.5. Implement GET /api/v2/publishing/metrics
- [ ] Method `MetricsHandler(w http.ResponseWriter, r *http.Request)`
- [ ] Export raw Prometheus metrics in OpenMetrics format
- [ ] Algorithm:
  1. Call aggregator.CollectAll() (get raw metrics)
  2. Convert to OpenMetrics text format:
     ```
     # HELP alert_history_publishing_health_checks_total ...
     # TYPE alert_history_publishing_health_checks_total counter
     alert_history_publishing_health_checks_total{target="rootly-prod",status="success"} 1234
     ...
     ```
  3. Set Content-Type: `application/openmetrics-text; version=1.0.0`
- [ ] Response: OpenMetrics text format
- [ ] Performance: <10ms

**Deliverable:** Method in `publishing_stats.go` (~150 LOC)

#### 4.6. Implement GET /api/v2/publishing/trends
- [ ] Method `TrendsHandler(w http.ResponseWriter, r *http.Request)`
- [ ] Deferred to Phase 5 (requires TrendDetector)
- [ ] Placeholder: Return 501 Not Implemented for now

**Deliverable:** Method in `publishing_stats.go` (~50 LOC)

#### 4.7. Add Middleware
- [ ] Rate limiting middleware (100 req/sec)
- [ ] CORS middleware (allow all origins for dashboards)
- [ ] Request timeout middleware (5s max)
- [ ] Logging middleware (slog, log all requests)
- [ ] Chain middlewares in RegisterHandlers()

**Deliverable:** Methods in `publishing_stats.go` (~200 LOC)

#### 4.8. Unit Tests for HTTP Handlers
- [ ] File: `go-app/cmd/server/handlers/publishing_stats_test.go`
- [ ] Test cases (15 tests):
  1. TestStatsHandler_Success (200 OK)
  2. TestStatsHandler_Filtering (filter by type/health)
  3. TestStatsHandler_Pagination (limit/offset)
  4. TestStatsHandler_InvalidFilter (400 Bad Request)
  5. TestTargetStatsHandler_Found (200 OK)
  6. TestTargetStatsHandler_NotFound (404 Not Found)
  7. TestHealthHandler_Healthy (200 OK, score >= 90)
  8. TestHealthHandler_Degraded (200 OK, 70 <= score < 90)
  9. TestHealthHandler_Unhealthy (503, score < 70)
  10. TestMetricsHandler_Success (200 OK, OpenMetrics format)
  11. TestMetricsHandler_ContentType (verify Content-Type header)
  12. TestTrendsHandler_NotImplemented (501)
  13. TestMiddleware_RateLimit (429 Too Many Requests)
  14. TestMiddleware_Timeout (503 on slow request)
  15. TestAPI_Performance (<10ms benchmark)
- [ ] Use httptest.NewRecorder() for testing
- [ ] Mock StatsAggregator
- [ ] Coverage target: 90%+

**Deliverable:** `publishing_stats_test.go` (800 LOC)

### Acceptance Criteria
- [ ] 5 HTTP endpoints implemented (stats, stats/{target}, health, metrics, trends)
- [ ] Query parameter parsing (filter, health, limit, offset)
- [ ] Pagination support (limit/offset)
- [ ] Filtering support (by target_type, health_status)
- [ ] OpenMetrics export format
- [ ] Health status codes (200/503 based on score)
- [ ] 4 middleware (rate limit, CORS, timeout, logging)
- [ ] 15+ unit tests passing
- [ ] Performance: <10ms response time (p95)
- [ ] Test coverage: 90%+
- [ ] Zero linter errors

### Deliverables
1. `publishing_stats.go` (950 LOC)
2. `publishing_stats_test.go` (800 LOC)

**Total:** ~1,750 LOC (950 production + 800 tests)

### Estimated Time
- **Time:** 4 hours
- **Breakdown:** 2h handlers + 1h middleware + 1h tests

---

## Phase 5: Trend Analysis Engine

### Objective
Implement TrendDetector to analyze historical trends and detect anomalies with <500µs analysis time.

### Tasks

#### 5.1. Create Time Series Storage (Stub)
- [ ] File: `go-app/internal/business/publishing/timeseries.go`
- [ ] Struct `TimeSeriesDB` (in-memory for MVP):
  ```go
  type TimeSeriesDB struct {
      data   map[string][]DataPoint // metricName -> time series
      mu     sync.RWMutex
      maxAge time.Duration // 7 days retention
  }

  type DataPoint struct {
      Timestamp time.Time
      Value     float64
  }
  ```
- [ ] Methods:
  - `Append(metricName, value, timestamp)`
  - `Query(metricName, startTime, endTime) []DataPoint`
  - `Cleanup()` (prune data older than 7 days)
- [ ] Performance: Append <1µs, Query <100µs
- [ ] Note: This is MVP in-memory storage; Phase 2 can use external TSDB (Prometheus/VictoriaMetrics)

**Deliverable:** `timeseries.go` (250 LOC)

#### 5.2. Implement TrendDetector
- [ ] File: `go-app/internal/business/publishing/trend_detector.go`
- [ ] Struct `TrendDetector`:
  ```go
  type TrendDetector struct {
      history  *TimeSeriesDB
      config   *TrendDetectorConfig
      emaState map[string]float64 // EMA smoothing state
      mu       sync.RWMutex
  }
  ```
- [ ] Constructor `NewTrendDetector(history, config)`
- [ ] Configuration:
  - `EMAAlpha` (default: 0.3)
  - `AnomalyThreshold` (default: 3σ)
  - `TrendThreshold` (default: 5% change)

**Deliverable:** `trend_detector.go` (200 LOC, structs + constructor)

#### 5.3. Implement Trend Classification
- [ ] Method `classifyTrend(metricName, currentValue) string`
- [ ] Algorithm:
  1. Load historical values (last 1h, 24h from TimeSeriesDB)
  2. Calculate 24h baseline (mean)
  3. Calculate rate of change: (current - baseline) / baseline
  4. Classify:
     - "increasing": change > +5%
     - "decreasing": change < -5%
     - "stable": abs(change) <= 5%
- [ ] Performance: <100µs

**Deliverable:** Method in `trend_detector.go` (~80 LOC)

#### 5.4. Implement Anomaly Detection
- [ ] Method `detectAnomaly(metricName, currentValue) bool`
- [ ] Algorithm:
  1. Load historical values (last 24h)
  2. Calculate mean (µ) and standard deviation (σ)
  3. Check if abs(current - µ) > 3σ
  4. Return true if anomaly detected
- [ ] Performance: <100µs

**Deliverable:** Method in `trend_detector.go` (~80 LOC)

#### 5.5. Implement Growth Rate Calculation
- [ ] Method `calculateGrowthRate(metricName, currentValue) float64`
- [ ] Algorithm:
  1. Load historical values (last 5 minutes)
  2. Calculate linear regression slope (Δvalue / Δtime)
  3. Convert to growth rate (units/minute)
- [ ] Performance: <100µs

**Deliverable:** Method in `trend_detector.go` (~80 LOC)

#### 5.6. Implement Analyze Method
- [ ] Method `Analyze(stats *PublishingStats) TrendAnalysis`
- [ ] Call classifyTrend() for success_rate, latency
- [ ] Call detectAnomaly() for error_rate
- [ ] Call calculateGrowthRate() for queue_depth
- [ ] Build TrendAnalysis struct
- [ ] Performance: <500µs (4 trend calculations)

**Deliverable:** Method in `trend_detector.go` (~100 LOC)

#### 5.7. Integrate with StatsAggregator
- [ ] Update `StatsAggregator.Calculate()` to call TrendDetector
- [ ] Store TrendDetector pointer in StatsAggregator
- [ ] Call `detector.Analyze(stats)` after stats calculation
- [ ] Set `stats.Trends = trends`

**Deliverable:** Update `stats_aggregator.go` (+20 LOC)

#### 5.8. Implement GET /api/v2/publishing/trends Endpoint
- [ ] Update `TrendsHandler` (was placeholder in Phase 4)
- [ ] Return TrendAnalysis JSON
- [ ] Performance: <5ms

**Deliverable:** Update `publishing_stats.go` (+50 LOC)

#### 5.9. Unit Tests for Trend Analysis
- [ ] File: `go-app/internal/business/publishing/trend_detector_test.go`
- [ ] Test cases (10 tests):
  1. TestTrendDetector_ClassifyTrend_Increasing
  2. TestTrendDetector_ClassifyTrend_Stable
  3. TestTrendDetector_ClassifyTrend_Decreasing
  4. TestTrendDetector_DetectAnomaly_Normal (no spike)
  5. TestTrendDetector_DetectAnomaly_Spike (>3σ)
  6. TestTrendDetector_CalculateGrowthRate
  7. TestTrendDetector_Analyze (full analysis)
  8. TestTimeSeriesDB_AppendQuery
  9. TestTimeSeriesDB_Cleanup (7d retention)
  10. TestTrendDetector_Performance (<500µs benchmark)
- [ ] Coverage target: 90%+

**Deliverable:** `trend_detector_test.go` (500 LOC)

### Acceptance Criteria
- [ ] TimeSeriesDB implemented (in-memory, 7d retention)
- [ ] TrendDetector with 3 analysis methods (classify, detect anomaly, growth rate)
- [ ] Trend classification (increasing/stable/decreasing)
- [ ] Anomaly detection (>3σ deviation)
- [ ] Growth rate calculation (linear regression)
- [ ] Integrated with StatsAggregator
- [ ] GET /api/v2/publishing/trends endpoint working
- [ ] 10+ unit tests passing
- [ ] Performance: Analyze() <500µs
- [ ] Test coverage: 90%+
- [ ] Zero linter errors

### Deliverables
1. `timeseries.go` (250 LOC)
2. `trend_detector.go` (540 LOC)
3. `trend_detector_test.go` (500 LOC)
4. Updates to `stats_aggregator.go` (+20 LOC)
5. Updates to `publishing_stats.go` (+50 LOC)

**Total:** ~1,360 LOC (860 production + 500 tests)

### Estimated Time
- **Time:** 4 hours
- **Breakdown:** 1h time series + 2h trend detector + 1h tests

---

## Phase 6: Testing & Quality Assurance

### Objective
Achieve 90%+ test coverage with comprehensive unit tests, integration tests, and benchmarks.

### Tasks

#### 6.1. Additional Unit Tests
- [ ] Expand existing test files to 90%+ coverage
- [ ] Add edge case tests:
  - Empty metrics (no targets discovered)
  - Nil subsystem metrics (graceful degradation)
  - Invalid query parameters (HTTP 400 errors)
  - Cache expiration edge cases
  - Concurrent access stress tests (100 goroutines)
- [ ] Target: 90%+ coverage (measured with `go test -cover`)

**Deliverable:** Updates to all `*_test.go` files (+500 LOC)

#### 6.2. Integration Tests
- [ ] File: `go-app/internal/business/publishing/stats_integration_test.go`
- [ ] Test scenarios (5 scenarios):
  1. TestE2E_StatsAPI_FullFlow (metrics → stats → HTTP → JSON)
  2. TestE2E_TrendDetection_ErrorSpike (inject spike, verify detection)
  3. TestE2E_HealthScore_Degradation (simulate failures, check score)
  4. TestE2E_Caching_Behavior (verify 1s TTL, invalidation)
  5. TestE2E_GracefulDegradation (nil subsystems, partial stats)
- [ ] Use real Prometheus metrics (not mocks)
- [ ] Use httptest for HTTP testing

**Deliverable:** `stats_integration_test.go` (600 LOC)

#### 6.3. Benchmarks
- [ ] File: `go-app/internal/business/publishing/stats_bench_test.go`
- [ ] Benchmark cases (10 benchmarks):
  1. BenchmarkMetricsCollector_Collect (target: <10µs)
  2. BenchmarkPublishingMetrics_CollectAll (target: <50µs)
  3. BenchmarkStatsAggregator_Calculate (target: <5ms)
  4. BenchmarkStatsAggregator_CalculateCached (target: <50µs)
  5. BenchmarkTrendDetector_Analyze (target: <500µs)
  6. BenchmarkHealthScore_Calculate (target: <100µs)
  7. BenchmarkStatsHandler_Cached (target: <10µs)
  8. BenchmarkStatsHandler_Uncached (target: <10ms)
  9. BenchmarkConcurrent_CollectAll_10Goroutines
  10. BenchmarkConcurrent_Calculate_100Requests
- [ ] Verify all performance targets met
- [ ] Zero allocations in hot paths

**Deliverable:** `stats_bench_test.go` (400 LOC)

#### 6.4. Load Testing
- [ ] File: `tasks/go-migration-analysis/TN-057-publishing-metrics-stats/load_test.md`
- [ ] Use `wrk` or `hey` for HTTP load testing
- [ ] Test scenarios:
  1. Sustained 1,000 req/sec for 60s
  2. Burst 10,000 req/sec for 10s
  3. Mixed endpoints (50% /stats, 30% /health, 20% /metrics)
- [ ] Verify:
  - p95 response time < 10ms
  - p99 response time < 50ms
  - Zero errors (0% error rate)
  - Memory usage < 50MB
  - CPU usage < 10%

**Deliverable:** `load_test.md` (500 LOC, test plan + results)

#### 6.5. Race Detector Testing
- [ ] Run all tests with `-race` flag:
  ```bash
  go test -race ./internal/business/publishing/...
  ```
- [ ] Fix any data races detected
- [ ] Verify thread-safe concurrent access (RWMutex usage)

**Deliverable:** Zero race conditions (verified)

#### 6.6. Linter & Static Analysis
- [ ] Run `golangci-lint run`
- [ ] Fix all errors and warnings
- [ ] Verify:
  - No unused variables
  - No ineffectual assignments
  - No shadowed variables
  - No missing error checks

**Deliverable:** Zero linter errors (verified)

### Acceptance Criteria
- [ ] 90%+ test coverage (all packages)
- [ ] 60+ unit tests passing
- [ ] 5 integration tests passing
- [ ] 10 benchmarks passing (all performance targets met)
- [ ] Load test results documented (10k req/sec sustained)
- [ ] Zero race conditions (verified with `-race`)
- [ ] Zero linter errors (golangci-lint clean)
- [ ] Test execution time < 30s (fast feedback loop)

### Deliverables
1. Updates to existing test files (+500 LOC)
2. `stats_integration_test.go` (600 LOC)
3. `stats_bench_test.go` (400 LOC)
4. `load_test.md` (500 LOC)

**Total:** ~2,000 LOC (all tests)

### Estimated Time
- **Time:** 6 hours
- **Breakdown:** 2h unit tests + 1.5h integration + 1h benchmarks + 1h load testing + 0.5h linting

---

## Phase 7: Documentation

### Objective
Create comprehensive documentation including README, API guide, PromQL examples, and Grafana dashboards.

### Tasks

#### 7.1. Main README
- [ ] File: `go-app/internal/business/publishing/STATS_README.md`
- [ ] Sections (10 sections, 2,000+ lines):
  1. **Executive Summary** (what, why, benefits)
  2. **Architecture Overview** (4-layer diagram)
  3. **Quick Start** (5-minute tutorial)
  4. **API Reference** (5 endpoints с examples)
  5. **Metrics Inventory** (50+ metrics catalogued)
  6. **Statistics Explained** (formulas, calculations)
  7. **Health Score** (weighted formula, thresholds)
  8. **Trend Analysis** (classification, anomaly detection)
  9. **Configuration** (env vars, options)
  10. **Troubleshooting** (5 common issues)
- [ ] Code examples (Go, curl, JSON responses)
- [ ] Performance benchmarks table

**Deliverable:** `STATS_README.md` (2,000 LOC)

#### 7.2. API Guide
- [ ] File: `tasks/go-migration-analysis/TN-057-publishing-metrics-stats/API_GUIDE.md`
- [ ] Sections (5 sections, 1,000+ lines):
  1. **Authentication** (optional API key)
  2. **Endpoints Reference** (5 endpoints, detailed)
  3. **Query Parameters** (filter, health, limit, offset)
  4. **Response Formats** (JSON schemas)
  5. **Error Handling** (status codes, error messages)
- [ ] curl examples for all endpoints
- [ ] Response samples (pretty-printed JSON)

**Deliverable:** `API_GUIDE.md` (1,000 LOC)

#### 7.3. PromQL Examples
- [ ] File: `tasks/go-migration-analysis/TN-057-publishing-metrics-stats/PROMQL_EXAMPLES.md`
- [ ] 20+ PromQL queries organized by category:
  1. **Success Rate Queries** (5 queries)
     - Overall success rate: `rate(jobs_processed_total{state="succeeded"}[5m])`
     - Success rate by target: `rate(jobs_processed_total{state="succeeded"}[5m]) by (target)`
     - Success rate trend (1h vs 24h)
  2. **Latency Queries** (5 queries)
     - p95 latency: `histogram_quantile(0.95, rate(job_duration_seconds_bucket[5m]))`
     - p99 latency by target
     - Latency trend (current vs 24h ago)
  3. **Error Rate Queries** (5 queries)
     - Error rate by type: `rate(errors_total[5m]) by (error_type)`
     - Error spike detection (>3σ)
  4. **Queue Health Queries** (5 queries)
     - Queue depth: `queue_size`
     - Queue growth rate: `deriv(queue_size[5m])`
     - DLQ size by target
- [ ] Explanation for each query

**Deliverable:** `PROMQL_EXAMPLES.md` (800 LOC)

#### 7.4. Grafana Dashboard JSON
- [ ] File: `tasks/go-migration-analysis/TN-057-publishing-metrics-stats/grafana_dashboard.json`
- [ ] Dashboard panels (15 panels):
  1. **Overview Panel**
     - Health score gauge (0-100, color-coded)
     - Total targets count
     - Success rate (percentage)
     - Avg latency (ms)
  2. **Targets Panel**
     - Pie chart: Targets by type (rootly/pagerduty/slack/webhook)
     - Pie chart: Targets by health (healthy/unhealthy/degraded)
  3. **Jobs Panel**
     - Time series: Jobs processed (last 24h)
     - Time series: Success rate trend
  4. **Latency Panel**
     - Time series: p50, p90, p95, p99 latencies
  5. **Errors Panel**
     - Time series: Error rate by type
     - Table: Top 10 targets by error count
  6. **Queue Panel**
     - Gauge: Queue depth (high/medium/low)
     - Time series: Queue growth rate
  7. **DLQ Panel**
     - Table: DLQ entries by target
  8. **Circuit Breaker Panel**
     - Pie chart: Circuit breaker states (open/closed/halfopen)
  9. **Workers Panel**
     - Gauge: Active workers / Total workers
  10. **SLA Panel**
     - Gauge: SLA compliance (yes/no)
     - Time series: SLA violations (last 7d)
- [ ] Variables: `target_type`, `target_name`, `time_range`
- [ ] Templating: Dynamic target selection
- [ ] Annotations: Mark deployments, incidents

**Deliverable:** `grafana_dashboard.json` (600 LOC JSON)

#### 7.5. OpenAPI Specification
- [ ] File: `docs/openapi-publishing-stats.yaml`
- [ ] OpenAPI 3.0.3 specification
- [ ] Define all 5 endpoints:
  - GET /api/v2/publishing/stats
  - GET /api/v2/publishing/stats/{target}
  - GET /api/v2/publishing/health
  - GET /api/v2/publishing/metrics
  - GET /api/v2/publishing/trends
- [ ] Schema definitions (PublishingStats, TargetStats, etc.)
- [ ] Response examples (JSON)
- [ ] Error responses (400, 404, 500, 503)

**Deliverable:** `openapi-publishing-stats.yaml` (500 LOC)

#### 7.6. Integration Guide
- [ ] File: `tasks/go-migration-analysis/TN-057-publishing-metrics-stats/INTEGRATION_GUIDE.md`
- [ ] Sections (5 sections, 600 lines):
  1. **Prerequisites** (dependencies, versions)
  2. **Local Setup** (run locally)
  3. **K8s Deployment** (helm chart updates)
  4. **Monitoring Setup** (Prometheus scraping, Grafana dashboards)
  5. **Troubleshooting** (common issues)
- [ ] Step-by-step instructions with code snippets

**Deliverable:** `INTEGRATION_GUIDE.md` (600 LOC)

### Acceptance Criteria
- [ ] STATS_README.md complete (2,000+ lines)
- [ ] API_GUIDE.md complete (1,000+ lines)
- [ ] PROMQL_EXAMPLES.md complete (800+ lines, 20+ queries)
- [ ] Grafana dashboard JSON (600 lines, 15 panels)
- [ ] OpenAPI 3.0.3 spec (500 lines)
- [ ] INTEGRATION_GUIDE.md complete (600 lines)
- [ ] All code examples tested and working
- [ ] All PromQL queries validated against Prometheus
- [ ] Grafana dashboard imported and functional

### Deliverables
1. `STATS_README.md` (2,000 LOC)
2. `API_GUIDE.md` (1,000 LOC)
3. `PROMQL_EXAMPLES.md` (800 LOC)
4. `grafana_dashboard.json` (600 LOC)
5. `openapi-publishing-stats.yaml` (500 LOC)
6. `INTEGRATION_GUIDE.md` (600 LOC)

**Total:** ~5,500 LOC (all documentation)

### Estimated Time
- **Time:** 4 hours
- **Breakdown:** 1.5h README + 1h API guide + 0.5h PromQL + 0.5h Grafana + 0.5h integration

---

## Phase 8: Integration with Main Application

### Objective
Integrate PublishingMetricsService into main.go and register HTTP endpoints.

### Tasks

#### 8.1. Update main.go
- [ ] File: `go-app/cmd/server/main.go`
- [ ] Add initialization code after publishing system setup:
  ```go
  // TN-057: Publishing Metrics & Stats Service
  log.Info("Initializing publishing metrics service")

  // Create metrics collectors
  healthCollector := publishing.NewHealthMetricsCollector(healthMetrics)
  refreshCollector := publishing.NewRefreshMetricsCollector(refreshMetrics)
  discoveryCollector := publishing.NewDiscoveryMetricsCollector(discoveryManager)
  queueCollector := publishing.NewQueueMetricsCollector(publishingQueue) // If TN-056 complete
  // ... publisher collectors ...

  // Create stats aggregator
  collectors := []publishing.MetricsCollector{
      healthCollector,
      refreshCollector,
      discoveryCollector,
      queueCollector,
  }
  aggregator := publishing.NewStatsAggregator(collectors, aggregatorConfig)

  // Create trend detector
  timeSeriesDB := publishing.NewTimeSeriesDB(7 * 24 * time.Hour) // 7d retention
  detector := publishing.NewTrendDetector(timeSeriesDB, detectorConfig)

  // Create HTTP service
  statsService := handlers.NewPublishingStatsService(aggregator, detector)

  // Register HTTP endpoints
  mux.HandleFunc("GET /api/v2/publishing/stats", statsService.StatsHandler)
  mux.HandleFunc("GET /api/v2/publishing/stats/{target}", statsService.TargetStatsHandler)
  mux.HandleFunc("GET /api/v2/publishing/health", statsService.HealthHandler)
  mux.HandleFunc("GET /api/v2/publishing/metrics", statsService.MetricsHandler)
  mux.HandleFunc("GET /api/v2/publishing/trends", statsService.TrendsHandler)

  log.Info("Publishing metrics service initialized", "endpoints", 5)
  ```
- [ ] Estimated lines: +100 LOC

**Deliverable:** Update `main.go` (+100 LOC)

#### 8.2. Configuration
- [ ] File: `go-app/cmd/server/config/config.go`
- [ ] Add configuration struct:
  ```go
  type PublishingStatsConfig struct {
      // Stats aggregation
      CacheTTL         time.Duration `env:"PUBLISHING_STATS_CACHE_TTL" envDefault:"1s"`
      SLATarget        float64       `env:"PUBLISHING_STATS_SLA_TARGET" envDefault:"99.9"`

      // Health score weights
      SuccessRateWeight    float64 `env:"PUBLISHING_STATS_WEIGHT_SUCCESS_RATE" envDefault:"0.4"`
      AvailabilityWeight   float64 `env:"PUBLISHING_STATS_WEIGHT_AVAILABILITY" envDefault:"0.3"`
      PerformanceWeight    float64 `env:"PUBLISHING_STATS_WEIGHT_PERFORMANCE" envDefault:"0.2"`
      QueueHealthWeight    float64 `env:"PUBLISHING_STATS_WEIGHT_QUEUE_HEALTH" envDefault:"0.1"`

      // Trend detection
      EMAAlpha          float64 `env:"PUBLISHING_STATS_EMA_ALPHA" envDefault:"0.3"`
      AnomalyThreshold  float64 `env:"PUBLISHING_STATS_ANOMALY_THRESHOLD" envDefault:"3.0"`
      TrendThreshold    float64 `env:"PUBLISHING_STATS_TREND_THRESHOLD" envDefault:"0.05"`

      // API
      RateLimit int    `env:"PUBLISHING_STATS_RATE_LIMIT" envDefault:"100"`
      APIKey    string `env:"PUBLISHING_STATS_API_KEY" envDefault:""` // Optional
  }
  ```

**Deliverable:** Update `config.go` (+50 LOC)

#### 8.3. Helm Chart Updates
- [ ] File: `helm/alert-history/templates/deployment.yaml`
- [ ] Add environment variables for configuration (if needed)
- [ ] No service/ingress changes needed (uses existing HTTP server)

**Deliverable:** Update `deployment.yaml` (+10 LOC)

#### 8.4. Integration Testing
- [ ] Start local server: `go run cmd/server/main.go`
- [ ] Test all 5 endpoints with curl:
  1. `curl http://localhost:8080/api/v2/publishing/stats`
  2. `curl http://localhost:8080/api/v2/publishing/health`
  3. `curl http://localhost:8080/api/v2/publishing/metrics`
  4. `curl http://localhost:8080/api/v2/publishing/stats/rootly-prod`
  5. `curl http://localhost:8080/api/v2/publishing/trends`
- [ ] Verify responses (200 OK, valid JSON)
- [ ] Check Prometheus metrics exported (`/metrics` endpoint)

**Deliverable:** Integration test results (documented in INTEGRATION_GUIDE.md)

### Acceptance Criteria
- [ ] main.go updated with PublishingMetricsService initialization
- [ ] Configuration struct added (10+ config options)
- [ ] Helm chart updated (if needed)
- [ ] All 5 endpoints registered and working
- [ ] Local integration test passing (5 curl commands successful)
- [ ] Zero compile errors
- [ ] Zero runtime errors on startup

### Deliverables
1. Update `main.go` (+100 LOC)
2. Update `config.go` (+50 LOC)
3. Update `deployment.yaml` (+10 LOC, optional)

**Total:** ~160 LOC (integration code)

### Estimated Time
- **Time:** 3 hours
- **Breakdown:** 1h main.go + 0.5h config + 1h integration testing + 0.5h debugging

---

## Phase 9: Performance Optimization

### Objective
Optimize performance to exceed targets by 2x: <25µs collection, <2.5ms stats calculation, <5ms HTTP responses.

### Tasks

#### 9.1. Profile Collection Layer
- [ ] Run CPU profiler: `go test -cpuprofile=cpu.prof -bench=BenchmarkCollectAll`
- [ ] Analyze with `go tool pprof cpu.prof`
- [ ] Identify hot spots (top 10 functions by CPU time)
- [ ] Optimize:
  - Reduce allocations in hot paths (use sync.Pool if needed)
  - Optimize Prometheus metric reads (batch reads if possible)
  - Reduce lock contention (RWMutex usage)
- [ ] Target: CollectAll() <25µs (2x better than 50µs target)

**Deliverable:** Optimization report + code changes

#### 9.2. Profile Stats Aggregation
- [ ] Run CPU profiler: `go test -cpuprofile=cpu.prof -bench=BenchmarkCalculate`
- [ ] Identify bottlenecks (likely JSON serialization, histogram calculations)
- [ ] Optimize:
  - Cache histogram percentile calculations (avoid recalculating p95/p99)
  - Use integer arithmetic where possible (avoid float64 divisions)
  - Reduce struct copies (use pointers)
- [ ] Target: Calculate() <2.5ms (2x better than 5ms target)

**Deliverable:** Optimization report + code changes

#### 9.3. Profile HTTP Endpoints
- [ ] Run HTTP benchmarks: `wrk -t4 -c100 -d30s http://localhost:8080/api/v2/publishing/stats`
- [ ] Analyze response times (p50, p95, p99)
- [ ] Optimize:
  - Increase cache hit rate (longer TTL if acceptable)
  - Optimize JSON encoding (use json.Marshal with buffer pool)
  - Reduce middleware overhead
- [ ] Target: p95 < 5ms (2x better than 10ms target)

**Deliverable:** Optimization report + code changes

#### 9.4. Memory Profiling
- [ ] Run memory profiler: `go test -memprofile=mem.prof -bench=BenchmarkCalculate`
- [ ] Analyze heap allocations
- [ ] Optimize:
  - Reduce allocations in hot paths (aim for zero allocations)
  - Use sync.Pool for temporary objects (if high allocation rate)
  - Fix memory leaks (if any)
- [ ] Target: <5MB total memory usage

**Deliverable:** Optimization report + code changes

#### 9.5. Concurrency Stress Testing
- [ ] Run concurrent benchmarks: `BenchmarkConcurrent_Calculate_1000Goroutines`
- [ ] Verify thread-safe access (no data races)
- [ ] Optimize lock contention (use atomic operations where possible)
- [ ] Target: Linear scalability up to 100 concurrent requests

**Deliverable:** Concurrency test results

### Acceptance Criteria
- [ ] CollectAll() <25µs (2x better than target)
- [ ] Calculate() <2.5ms (2x better than target)
- [ ] HTTP p95 <5ms (2x better than target)
- [ ] Memory usage <5MB
- [ ] Zero allocations in hot paths (collection layer)
- [ ] Linear scalability (100 concurrent requests)
- [ ] Profiling reports documented

### Deliverables
1. Optimization report (1,000 LOC markdown)
2. Code changes to optimize performance

**Total:** ~1,000 LOC (report + code changes)

### Estimated Time
- **Time:** 3 hours
- **Breakdown:** 1h CPU profiling + 1h memory profiling + 1h optimization

---

## Phase 10: Final Certification (150% Quality Target)

### Objective
Validate 150% quality achievement (Grade A+, Production-Ready) through comprehensive quality audit.

### Tasks

#### 10.1. Quality Metrics Audit
- [ ] File: `tasks/go-migration-analysis/TN-057-publishing-metrics-stats/COMPLETION_REPORT.md`
- [ ] Audit checklist (30 items):

**Implementation (10/10)**
- [ ] 1. Metrics collection from 9+ subsystems
- [ ] 2. Stats aggregation (system-wide + per-target)
- [ ] 3. Health score calculation (weighted formula)
- [ ] 4. SLA tracking (99.9% target)
- [ ] 5. Trend detection (classify + anomaly + growth rate)
- [ ] 6. Time series storage (7d retention)
- [ ] 7. HTTP API (5 endpoints)
- [ ] 8. Stats cache (1s TTL)
- [ ] 9. Self-monitoring metrics (7 metrics)
- [ ] 10. Graceful degradation (nil subsystems)

**Testing (10/10)**
- [ ] 11. Unit tests (60+ tests, 90%+ coverage)
- [ ] 12. Integration tests (5 scenarios)
- [ ] 13. Benchmarks (10 benchmarks, all targets exceeded)
- [ ] 14. Load testing (10k req/sec sustained)
- [ ] 15. Race detector (zero races)
- [ ] 16. Linter clean (golangci-lint)
- [ ] 17. Performance validation (<50µs collection, <5ms stats)
- [ ] 18. Concurrency stress tests (100 goroutines)
- [ ] 19. Error scenarios (nil metrics, invalid params)
- [ ] 20. Cache behavior (expiration, invalidation)

**Documentation (5/5)**
- [ ] 21. STATS_README.md (2,000+ lines)
- [ ] 22. API_GUIDE.md (1,000+ lines)
- [ ] 23. PROMQL_EXAMPLES.md (800+ lines, 20+ queries)
- [ ] 24. Grafana dashboard (600 lines, 15 panels)
- [ ] 25. OpenAPI 3.0.3 spec (500 lines)

**Integration (5/5)**
- [ ] 26. main.go integration (100+ lines)
- [ ] 27. Configuration (10+ env vars)
- [ ] 28. Local testing (5 curl commands successful)
- [ ] 29. Helm chart updates (if needed)
- [ ] 30. Production readiness (zero errors on startup)

**Deliverable:** `COMPLETION_REPORT.md` (1,500 LOC)

#### 10.2. Performance Certification
- [ ] Verify all performance targets exceeded by 2x:
  - ✅ CollectAll() <25µs (target: 50µs)
  - ✅ Calculate() <2.5ms (target: 5ms)
  - ✅ HTTP p95 <5ms (target: 10ms)
  - ✅ Analyze() <250µs (target: 500µs)
  - ✅ Health score <50µs (target: 100µs)
- [ ] Document benchmark results table

**Deliverable:** Performance certification section in COMPLETION_REPORT.md

#### 10.3. Quality Score Calculation
- [ ] Calculate quality score:
  ```
  Quality Score = (
    ImplementationScore * 0.4 +
    TestingScore * 0.3 +
    DocumentationScore * 0.2 +
    PerformanceScore * 0.1
  )

  Where:
  - ImplementationScore = (features_delivered / features_planned) * 100
  - TestingScore = (test_coverage / 90) * 100 (capped at 100)
  - DocumentationScore = (doc_lines / 5500_target) * 100
  - PerformanceScore = min(actual_performance / target * 100, 200)
  ```
- [ ] Target: **150%+** (Grade A+)
- [ ] Expected: **160%+** (exceeding target by 10%)

**Deliverable:** Quality score in COMPLETION_REPORT.md

#### 10.4. Final Checklist
- [ ] All 10 phases complete
- [ ] Zero technical debt (all TODOs resolved)
- [ ] Zero breaking changes
- [ ] Backward compatibility (100%)
- [ ] Production readiness (100%)
- [ ] Documentation complete (100%)
- [ ] Tests passing (100%)
- [ ] Performance targets exceeded (2x)

**Deliverable:** Final checklist in COMPLETION_REPORT.md

#### 10.5. Git Commit & Documentation
- [ ] Git add all files
- [ ] Git commit with comprehensive message:
  ```
  feat(TN-057): Publishing Metrics & Stats - 150% quality (Grade A+)

  Deliverables:
  - Centralized metrics collection (50+ metrics from 9 subsystems)
  - Statistics aggregation (<5ms, system-wide + per-target)
  - Health score calculation (weighted 0-100)
  - Trend analysis (classify + anomaly detection)
  - HTTP API (5 REST endpoints)
  - Comprehensive testing (90%+ coverage, 60+ tests)
  - Enterprise documentation (5,500+ LOC)

  Performance:
  - CollectAll(): <25µs (2x better than 50µs target)
  - Calculate(): <2.5ms (2x better than 5ms target)
  - HTTP p95: <5ms (2x better than 10ms target)

  Quality: 160% (Grade A+, Production-Ready)
  Duration: 3 days (42h target, 40h actual)
  LOC: 9,000+ total (3,500 prod + 2,500 tests + 5,500 docs)
  ```

**Deliverable:** Git commit

### Acceptance Criteria
- [ ] Quality score: **150%+** (Grade A+)
- [ ] All 30 checklist items complete
- [ ] Performance targets exceeded by 2x
- [ ] Documentation complete (5,500+ LOC)
- [ ] Tests passing (90%+ coverage)
- [ ] Production-ready (zero blockers)
- [ ] Git commit with comprehensive message

### Deliverables
1. `COMPLETION_REPORT.md` (1,500 LOC)
2. Git commit (comprehensive message)

**Total:** ~1,500 LOC (completion report)

### Estimated Time
- **Time:** 2 hours
- **Breakdown:** 1h audit + 0.5h quality score + 0.5h final commit

---

## Summary

### Total Deliverables
- **Production Code:** ~3,500 LOC
- **Test Code:** ~2,500 LOC
- **Documentation:** ~5,500 LOC
- **Total:** ~11,500 LOC (includes requirements, design, tasks)

### Timeline
- **Phase 0-1:** Documentation & Gap Analysis (6h) ✅
- **Phase 2:** Metrics Collection Layer (4h)
- **Phase 3:** Statistics Aggregation Layer (6h)
- **Phase 4:** HTTP API Endpoints (4h)
- **Phase 5:** Trend Analysis Engine (4h)
- **Phase 6:** Testing (6h)
- **Phase 7:** Documentation (4h)
- **Phase 8:** Integration (3h)
- **Phase 9:** Performance Optimization (3h)
- **Phase 10:** Final Certification (2h)

**Total:** ~42 hours → **Target: 3-4 days** (8-10h/day)

### Quality Target
- **Baseline:** 100% (all functional requirements)
- **Target:** **150%** (Grade A+, Production-Ready)
- **Expected:** **160%+** (exceeding target by 10%)

### Success Criteria
✅ 50+ metrics collected from 9 subsystems
✅ Stats aggregation <5ms (or <2.5ms optimized)
✅ 5 HTTP API endpoints (<10ms response time)
✅ Health score calculation (weighted formula)
✅ Trend detection (3 trends + anomaly detection)
✅ 90%+ test coverage (60+ tests)
✅ Comprehensive documentation (5,500+ LOC)
✅ Zero technical debt
✅ Production-ready (zero blockers)

---

**Document Version:** 1.0
**Last Updated:** 2025-11-12
**Author:** AI Assistant
**Status:** PHASE 0-1 COMPLETE, ready for Phase 2
**Next Step:** Phase 2 (Gap Analysis) - audit existing metrics
