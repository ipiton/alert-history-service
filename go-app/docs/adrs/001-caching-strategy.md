# ADR-001: 2-Tier Caching Strategy

**Status**: Accepted
**Date**: 2025-11-16
**Deciders**: Architecture Team
**Context**: TN-63 History Endpoint

---

## Context

The GET /history endpoint needs to handle high traffic (1000+ req/s) with low latency (p95 < 10ms). Database queries are expensive, especially with complex filters and large datasets.

## Decision

We will implement a **2-tier caching strategy**:
- **L1 Cache**: In-memory cache (Ristretto) for ultra-fast access
- **L2 Cache**: Distributed cache (Redis) for shared state across instances

## Rationale

### Why 2-Tier?

1. **Performance**: L1 cache provides sub-millisecond access (< 0.1ms)
2. **Scalability**: L2 cache enables horizontal scaling
3. **Resilience**: L1 cache works even if Redis is unavailable
4. **Cost**: L1 cache is free (memory), L2 cache is shared

### Why Ristretto for L1?

- High performance (millions of ops/sec)
- Memory-efficient (bounded size)
- Thread-safe (concurrent access)
- TTL support (automatic expiration)

### Why Redis for L2?

- Distributed (shared across instances)
- Persistence (survives restarts)
- Compression (gzip for large responses)
- Pub/Sub (cache invalidation)

## Alternatives Considered

### Alternative 1: Single-Tier (Redis Only)
- ❌ Higher latency (network round-trip)
- ❌ Single point of failure
- ✅ Simpler architecture

### Alternative 2: Single-Tier (In-Memory Only)
- ❌ Not shared across instances
- ❌ Cache duplication
- ✅ Lowest latency

### Alternative 3: 3-Tier (L1 + L2 + L3 Database)
- ❌ Over-engineered for current needs
- ❌ Higher complexity
- ✅ Maximum performance

## Consequences

### Positive
- ✅ p95 latency < 10ms achieved
- ✅ Cache hit rate > 90%
- ✅ Horizontal scaling supported
- ✅ Resilience to Redis failures

### Negative
- ⚠️ Increased complexity (2 cache layers)
- ⚠️ Memory usage (L1 cache)
- ⚠️ Redis dependency (L2 cache)

### Mitigations
- Cache warming for popular queries
- Cache size limits (prevent OOM)
- Redis connection pooling
- Fallback to database if both caches fail

## Implementation Details

- **L1 Cache**: 10K entries, 5min TTL, LRU eviction
- **L2 Cache**: 1M entries, 1h TTL, gzip compression
- **Cache Key**: SHA256 hash of request parameters
- **Invalidation**: TTL-based + manual invalidation

## Metrics

- Cache hit rate (target: > 90%)
- Cache latency (L1: < 0.1ms, L2: < 1ms)
- Cache size (L1: < 100MB, L2: < 1GB)

---

**Approved by**: Architecture Team
**Date**: 2025-11-16
