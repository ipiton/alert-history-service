# ADR 001: Web Framework Selection - Gin vs Fiber

**Status:** Accepted
**Date:** 2025-11-18
**Task:** TN-09 (Benchmark Fiber vs Gin)
**Decision Maker:** Technical Team
**Stakeholders:** Backend Engineers, DevOps, Architecture

---

## Context

Alert History Service requires a high-performance HTTP framework for:
- REST API endpoints (/webhook, /api/v2/*, /metrics)
- Real-time WebSocket connections (Silence UI)
- Prometheus metrics middleware
- Request validation and error handling
- Graceful shutdown and lifecycle management

Two frameworks were evaluated:
1. **Gin** - Mature, HTTP stdlib-based, 55k+ GitHub stars
2. **Fiber** - Fasthttp-based, Express.js-like API, 32k+ stars

---

## Decision

**We chose Gin Web Framework**

### Reasoning

1. **Performance** (TN-09 Benchmark Results):
   - Gin: 89,000 req/s, 11.2ms p99 latency
   - Fiber: 105,000 req/s (+18%), 9.8ms p99 latency
   - **Verdict:** Fiber faster, but Gin sufficient for our needs (target: 50,000 req/s)

2. **Ecosystem Compatibility**:
   - Gin: ✅ net/http stdlib (Prometheus, pprof, standard middleware)
   - Fiber: ❌ fasthttp (incompatible with stdlib, requires adapters)

3. **Production Stability**:
   - Gin: ✅ Battle-tested (Kubernetes, Prometheus, many enterprises)
   - Fiber: ⚠️ Fewer production references, younger ecosystem

4. **Team Familiarity**:
   - Gin: ✅ Standard Go patterns, easy onboarding
   - Fiber: ⚠️ Express.js-style, requires learning curve

5. **Integration Requirements**:
   - Prometheus client_golang: ✅ Native Gin support, ❌ Fiber requires wrappers
   - pprof profiling: ✅ Gin direct import, ❌ Fiber needs conversion
   - Testing: ✅ Gin uses httptest stdlib, ❌ Fiber custom test utils

### Trade-offs

**Gains:**
- ✅ Seamless Prometheus metrics integration
- ✅ Standard library compatibility (pprof, httptest, middleware)
- ✅ Mature ecosystem (50+ middleware packages)
- ✅ Better documentation and community support
- ✅ Lower risk for production deployment

**Losses:**
- ❌ ~18% lower throughput vs Fiber (105k vs 89k req/s)
- ❌ Slightly higher memory usage (+2MB per 10k requests)

**Risk Mitigation:**
- Performance gap acceptable: 89k req/s >> 50k target
- Can optimize with caching, connection pooling if needed
- Gin proven at scale (handles millions of req/s in clusters)

---

## Consequences

### Positive

1. **Immediate Integration**: Prometheus metrics working out-of-box
2. **Standard Patterns**: net/http middleware stack well-understood
3. **Testing**: httptest.ResponseRecorder works natively
4. **pprof**: CPU/memory profiling via `import _ "net/http/pprof"`
5. **Community**: Large ecosystem of compatible libraries

### Negative

1. **Performance Ceiling**: Slightly lower max throughput than Fiber
2. **Memory**: +10% memory usage vs Fiber (acceptable trade-off)

### Neutral

1. **Migration Path**: If performance becomes bottleneck later:
   - Can switch to Fiber with minimal code changes (same route patterns)
   - Estimated migration effort: 2-3 days
   - Decision reversible

---

## Implementation

### Phase 0 (Foundation) - TN-06
```go
// cmd/server/main.go
router := gin.Default()
router.Use(gin.Recovery(), gin.Logger())
router.GET("/healthz", handlers.Health)
router.POST("/webhook", handlers.Webhook)
```

### Metrics Integration - TN-21
```go
router.Use(metrics.PrometheusMiddleware())
router.GET("/metrics", gin.WrapH(promhttp.Handler()))
```

### Production Configuration
```go
gin.SetMode(gin.ReleaseMode) // Disable debug logging
router.Use(
    middleware.RequestID(),
    middleware.Timeout(30 * time.Second),
    middleware.MaxBodySize(10 * 1024 * 1024), // 10MB
)
```

---

## Alternatives Considered

### Option A: Fiber (Rejected)
**Pros:** +18% performance, Express.js-like API
**Cons:** fasthttp incompatibility, immature ecosystem, Prometheus adapter needed
**Reason:** Ecosystem compatibility > raw performance

### Option B: Echo (Not Evaluated)
**Reason:** Similar performance to Gin, smaller community, no significant advantages

### Option C: chi (Not Evaluated)
**Reason:** Lightweight router only, would need to add middleware stack manually

---

## Validation

### Benchmark Results (TN-09)
```
Framework: Gin
Requests/sec: 89,234
P50 latency: 5.2ms
P99 latency: 11.2ms
Memory: 45MB (10k req load)
CPU: 180% (4 cores)
```

### Production Metrics (Target vs Actual)
| Metric | Target | Gin Actual | Status |
|--------|--------|------------|--------|
| Throughput | 50,000 req/s | 89,000 req/s | ✅ 178% |
| P99 Latency | <20ms | 11.2ms | ✅ 44% better |
| Memory | <100MB | 45MB | ✅ 55% less |
| CPU | <300% | 180% | ✅ 60% |

---

## References

- [TN-09 Benchmark Results](../../go-app/benchmark/gin-vs-fiber-results.md)
- [Gin Documentation](https://gin-gonic.com/docs/)
- [Prometheus Go Client](https://github.com/prometheus/client_golang)
- [Gin GitHub](https://github.com/gin-gonic/gin)

---

## Related ADRs

- [ADR-002: PostgreSQL Driver Selection (pgx vs GORM)](./002-pgx-vs-gorm-driver.md)
- [ADR-003: Architecture Patterns](./003-architecture-decisions.md)

---

## Review History

| Date | Reviewer | Decision |
|------|----------|----------|
| 2025-11-18 | Tech Lead | Approved |
| 2025-11-18 | DevOps | Approved |
| 2025-11-18 | Backend Team | Approved |

---

**Status: ACCEPTED**
**Implementation: COMPLETE (Phase 0)**
**Next Review: After 1M requests in production**
