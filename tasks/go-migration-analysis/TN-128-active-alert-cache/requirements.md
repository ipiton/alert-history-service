# TN-128: Active Alert Cache - Requirements

## Overview

**Task**: Implement two-tier caching (L1 memory + L2 Redis) for active firing alerts.
**Priority**: HIGH
**Dependencies**: None
**Blocks**: TN-127 (cache integration), TN-129

---

## Functional Requirements

### FR-1: Two-Tier Caching
- **L1 Cache**: In-memory LRU cache (fast, limited size)
- **L2 Cache**: Redis cache (persistent, distributed)
- **Fallback**: L1 → L2 → empty (graceful degradation)

### FR-2: Operations
- `GetFiringAlerts(ctx)` - get all firing alerts
- `AddFiringAlert(ctx, alert)` - add alert to cache
- `RemoveAlert(ctx, fingerprint)` - remove alert from cache

### FR-3: Background Cleanup
- Periodic cleanup of expired alerts (every 1 minute)
- TTL: 5 minutes (configurable)

---

## Non-Functional Requirements

### NFR-1: Performance
- L1 hit: <1ms
- L2 hit: <10ms
- Graceful fallback on Redis failure

### NFR-2: Test Coverage
- 80%+ coverage
- 35+ tests (unit + integration)

---

## Acceptance Criteria

- [x] ActiveAlertCache interface defined (already in matcher.go)
- [ ] L1 cache (in-memory LRU)
- [ ] L2 cache (Redis)
- [ ] Background cleanup worker
- [ ] 35+ tests, 80%+ coverage
- [ ] Performance <10ms (p99)

---

**Date**: 2025-11-04
**Status**: READY

