# TN-128: Active Alert Cache - Design

## Architecture

```
┌──────────────────────────────────────┐
│   TwoTierAlertCache                  │
│   (L1 + L2 fallback)                │
└─────────┬────────────────────────────┘
          │
          ├──> L1: In-Memory LRU
          │     - 1000 alerts max
          │     - <1ms access
          │
          └──> L2: Redis
                - Persistent
                - Distributed
                - <10ms access
```

## Implementation

### TwoTierAlertCache

```go
type TwoTierAlertCache struct {
    l1Cache    *lru.Cache           // In-memory LRU
    redisCache cache.Cache          // Redis (from existing infra)
    logger     *slog.Logger
    stopCh     chan struct{}        // For cleanup worker
}
```

### Algorithms

#### GetFiringAlerts
1. Try L1 cache → return if found
2. Try L2 (Redis) → populate L1 → return
3. Return empty (graceful degradation)

#### AddFiringAlert
1. Add to L1
2. Add to L2 (async, best-effort)

#### Background Cleanup
- Every 1 minute
- Remove alerts with `endsAt < now`

---

**Date**: 2025-11-04
**Status**: DESIGN COMPLETE
