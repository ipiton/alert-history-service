# TN-129: Inhibition State Manager - Technical Design

**Version**: 1.0
**Date**: 2025-11-05
**Status**: READY FOR 150% IMPLEMENTATION
**Dependencies**: TN-126 (Parser) ✅, TN-127 (Matcher) ✅, TN-128 (Cache) ✅

---

## 1. Overview

**Цель**: Реализовать enterprise-grade систему управления состоянием inhibition relationships с полной observability, high availability, и comprehensive testing для достижения 150% качества.

**Контекст**: TN-129 является 4-й из 5 задач Module 2 (Inhibition Rules Engine). Зависимости TN-126/127/128 завершены на 150%+ качества с Grade A+.

**Scope расширения (50% → 150%)**:
- ✅ Существующая реализация: InhibitionState model + DefaultStateManager (301 LOC)
- 🎯 **Добавить**: 6 Prometheus metrics + metrics recording
- 🎯 **Добавить**: 30+ comprehensive tests (unit + integration + concurrent + benchmarks)
- 🎯 **Добавить**: Background cleanup worker для expired states
- 🎯 **Добавить**: Integration с InhibitionMatcher
- 🎯 **Добавить**: Comprehensive README + PromQL examples
- 🎯 **Добавить**: Performance benchmarks
- 🎯 **Улучшить**: Error handling и validation

---

## 2. Architecture

### 2.1 Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                   InhibitionMatcher                         │
│         (TN-127, проверяет inhibition rules)                │
└─────────────────────┬───────────────────────────────────────┘
                      │ calls RecordInhibition()
                      │ when match found
                      ↓
┌─────────────────────────────────────────────────────────────┐
│              InhibitionStateManager Interface               │
│  - RecordInhibition(state)                                  │
│  - RemoveInhibition(fingerprint)                            │
│  - GetActiveInhibitions()                                   │
│  - GetInhibitedAlerts()                                     │
│  - IsInhibited(fingerprint)                                 │
│  - GetInhibitionState(fingerprint)                          │
└─────────────────────┬───────────────────────────────────────┘
                      │ implements
                      ↓
┌─────────────────────────────────────────────────────────────┐
│              DefaultStateManager Implementation             │
│                                                             │
│  ┌────────────────┐      ┌────────────────┐               │
│  │  sync.Map      │◄────►│  Redis Store   │               │
│  │  (L1 cache)    │      │  (persistence) │               │
│  └────────────────┘      └────────────────┘               │
│         ↓                        ↓                          │
│  ┌────────────────────────────────────────┐                │
│  │      Cleanup Worker (goroutine)        │                │
│  │  - Remove expired inhibitions          │                │
│  │  - Cleanup interval: 1 minute          │                │
│  └────────────────────────────────────────┘                │
│                     ↓                                       │
│  ┌────────────────────────────────────────┐                │
│  │      StateMetrics (6 metrics)          │                │
│  │  - state_records_total                 │                │
│  │  - state_removals_total                │                │
│  │  - state_active_gauge                  │                │
│  │  - state_expired_total                 │                │
│  │  - state_operations_duration_seconds   │                │
│  │  - state_redis_errors_total            │                │
│  └────────────────────────────────────────┘                │
└─────────────────────────────────────────────────────────────┘
```

### 2.2 Data Model

```go
type InhibitionState struct {
    TargetFingerprint  string     // Inhibited alert fingerprint
    SourceFingerprint  string     // Inhibiting alert fingerprint
    RuleName           string     // Inhibition rule name
    InhibitedAt        time.Time  // When inhibition started
    ExpiresAt          *time.Time // Optional expiration time
}
```

**Storage Strategy**:
- **L1 (Memory)**: `sync.Map` для ultra-fast access (<100ns)
- **L2 (Redis)**: Persistence для HA recovery, TTL 24h
- **Fallback**: Graceful degradation если Redis недоступен

---

## 3. Key Features (150% Quality)

### 3.1 Core Functionality ✅

| Feature | Implementation | Status |
|---------|----------------|--------|
| Record inhibition | `RecordInhibition(ctx, state)` | ✅ Exists |
| Remove inhibition | `RemoveInhibition(ctx, fingerprint)` | ✅ Exists |
| Get active states | `GetActiveInhibitions(ctx)` | ✅ Exists |
| Check if inhibited | `IsInhibited(ctx, fingerprint)` | ✅ Exists |
| Get single state | `GetInhibitionState(ctx, fingerprint)` | ✅ Exists |
| Get all inhibited | `GetInhibitedAlerts(ctx)` | ✅ Exists |

### 3.2 Enhanced Features (NEW for 150%)

| Feature | Description | Priority |
|---------|-------------|----------|
| **StateMetrics** | 6 Prometheus metrics для observability | 🔴 CRITICAL |
| **Cleanup Worker** | Background goroutine для удаления expired states | 🔴 CRITICAL |
| **Integration Tests** | Integration с Matcher + Redis | 🔴 CRITICAL |
| **Concurrent Tests** | Race condition testing | 🟡 HIGH |
| **Benchmarks** | Performance measurement | 🟡 HIGH |
| **Comprehensive README** | Usage guide + examples + PromQL | 🟡 HIGH |
| **Error Wrapping** | Context-aware errors | 🟢 MEDIUM |
| **Validation** | Enhanced input validation | 🟢 MEDIUM |

---

## 4. Prometheus Metrics (6 metrics)

### 4.1 Metrics Definition

```go
// In pkg/metrics/business.go (NEW section)

// Inhibition State subsystem metrics
type InhibitionStateMetrics struct {
    // Records total
    StateRecordsTotal *prometheus.CounterVec // counter by rule_name

    // Removals total
    StateRemovalsTotal *prometheus.CounterVec // counter by reason (expired|manual|source_resolved)

    // Active inhibitions gauge
    StateActiveGauge prometheus.Gauge

    // Expired inhibitions cleaned up
    StateExpiredTotal prometheus.Counter

    // Operation duration
    StateOperationDurationSeconds *prometheus.HistogramVec // histogram by operation (record|remove|get|check)

    // Redis errors
    StateRedisErrorsTotal *prometheus.CounterVec // counter by operation (persist|load|delete)
}
```

### 4.2 Naming Convention

```
alert_history_business_inhibition_state_records_total{rule_name="node-down"}
alert_history_business_inhibition_state_removals_total{reason="expired"}
alert_history_business_inhibition_state_active
alert_history_business_inhibition_state_expired_total
alert_history_business_inhibition_state_operation_duration_seconds{operation="record"}
alert_history_business_inhibition_state_redis_errors_total{operation="persist"}
```

---

## 5. Testing Strategy (30+ tests)

### 5.1 Test Distribution

| Category | Count | Coverage Target | Description |
|----------|-------|-----------------|-------------|
| **Unit Tests** | 15 tests | 90%+ | Individual method testing |
| **Integration Tests** | 6 tests | - | Redis + Matcher integration |
| **Concurrent Tests** | 4 tests | - | Race conditions, goroutine safety |
| **Error Handling** | 5 tests | - | Edge cases, error paths |
| **Benchmarks** | 6 benchmarks | - | Performance measurement |
| **TOTAL** | **36 tests** | **85%+** | Exceeds 30+ requirement |

### 5.2 Unit Tests (15 tests)

```go
// state_manager_test.go

// Basic operations
- TestRecordInhibition_Success
- TestRecordInhibition_NilState
- TestRecordInhibition_EmptyTargetFingerprint
- TestRecordInhibition_EmptySourceFingerprint

// Removal
- TestRemoveInhibition_Success
- TestRemoveInhibition_EmptyFingerprint
- TestRemoveInhibition_NonExistent

// Queries
- TestGetActiveInhibitions_MultipleStates
- TestGetActiveInhibitions_FiltersExpired
- TestGetInhibitedAlerts_ReturnsFingerprints
- TestIsInhibited_True
- TestIsInhibited_False
- TestIsInhibited_Expired
- TestGetInhibitionState_Found
- TestGetInhibitionState_NotFound
```

### 5.3 Integration Tests (6 tests)

```go
// state_manager_integration_test.go

- TestStateManager_RedisIntegration_RecordAndLoad
- TestStateManager_RedisIntegration_PersistAndRecover
- TestStateManager_RedisIntegration_GracefulDegradation
- TestStateManager_WithMatcher_Integration
- TestStateManager_CleanupWorker_RemovesExpired
- TestStateManager_WithCache_Integration
```

### 5.4 Concurrent Tests (4 tests)

```go
// state_manager_concurrent_test.go

- TestStateManager_Concurrent_RecordRemove
- TestStateManager_Concurrent_MultipleReaders
- TestStateManager_Concurrent_ExpirationRace
- TestStateManager_Concurrent_CleanupWorker
```

### 5.5 Benchmarks (6 benchmarks)

```go
// state_manager_bench_test.go

- BenchmarkRecordInhibition_MemoryOnly
- BenchmarkRecordInhibition_WithRedis
- BenchmarkIsInhibited_MemoryHit
- BenchmarkGetActiveInhibitions_100States
- BenchmarkGetInhibitionState_MemoryHit
- BenchmarkRemoveInhibition
```

**Performance Targets**:
- `RecordInhibition`: <10µs (memory), <1ms (with Redis)
- `IsInhibited`: <100ns (memory hit)
- `GetActiveInhibitions`: <50µs (100 states)
- `RemoveInhibition`: <5µs (memory), <500µs (with Redis)

---

## 6. Cleanup Worker Design

### 6.1 Purpose

Автоматически удаляет expired inhibition states для предотвращения memory leaks и поддержания актуальности данных.

### 6.2 Implementation

```go
type DefaultStateManager struct {
    // ... existing fields ...

    // Cleanup worker control
    cleanupInterval time.Duration
    cleanupStop     chan struct{}
    cleanupDone     sync.WaitGroup
}

// StartCleanupWorker starts the background cleanup worker
func (sm *DefaultStateManager) StartCleanupWorker(ctx context.Context) {
    sm.cleanupDone.Add(1)
    go sm.cleanupWorker(ctx)
}

// cleanupWorker periodically removes expired inhibitions
func (sm *DefaultStateManager) cleanupWorker(ctx context.Context) {
    defer sm.cleanupDone.Done()

    ticker := time.NewTicker(sm.cleanupInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-sm.cleanupStop:
            return
        case <-ticker.C:
            sm.cleanupExpiredStates(ctx)
        }
    }
}

// StopCleanupWorker gracefully stops the cleanup worker
func (sm *DefaultStateManager) StopCleanupWorker() {
    close(sm.cleanupStop)
    sm.cleanupDone.Wait()
}
```

**Configuration**:
- Cleanup interval: `1 minute` (configurable)
- Graceful shutdown: `ctx.Done()` + `cleanupStop` channel
- Metrics: Record `StateExpiredTotal` для каждого удаленного state

---

## 7. Integration with Matcher

### 7.1 Matcher Calls State Manager

```go
// In matcher_impl.go (TN-127)

func (m *DefaultInhibitionMatcher) ShouldInhibit(ctx context.Context, target *Alert) (bool, string, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        m.metrics.InhibitionDurationSeconds.WithLabelValues("check").Observe(duration)
    }()

    // ... existing matching logic ...

    if matchedRule != nil {
        // 🆕 Record inhibition state
        state := &inhibition.InhibitionState{
            TargetFingerprint: target.Fingerprint,
            SourceFingerprint: source.Fingerprint,
            RuleName:          matchedRule.Name,
            InhibitedAt:       time.Now(),
            ExpiresAt:         nil, // Until source resolves
        }

        if err := m.stateManager.RecordInhibition(ctx, state); err != nil {
            m.logger.Warn("Failed to record inhibition state", "error", err)
            // Non-critical: inhibition still happens
        }

        return true, matchedRule.Name, nil
    }

    return false, "", nil
}
```

### 7.2 Alert Resolution Handler

```go
// When source alert resolves, remove inhibition

func HandleAlertResolved(ctx context.Context, alert *Alert, stateManager InhibitionStateManager) {
    // Remove all inhibitions caused by this source
    states, _ := stateManager.GetActiveInhibitions(ctx)

    for _, state := range states {
        if state.SourceFingerprint == alert.Fingerprint {
            _ = stateManager.RemoveInhibition(ctx, state.TargetFingerprint)
        }
    }
}
```

---

## 8. Error Handling

### 8.1 Error Types

```go
// state_errors.go (NEW file)

var (
    ErrNilState = errors.New("inhibition state cannot be nil")
    ErrEmptyTargetFingerprint = errors.New("target fingerprint cannot be empty")
    ErrEmptySourceFingerprint = errors.New("source fingerprint cannot be empty")
    ErrStateNotFound = errors.New("inhibition state not found")
)

// StateError wraps errors with context
type StateError struct {
    Op  string // Operation: "record", "remove", "get"
    Err error
}

func (e *StateError) Error() string {
    return fmt.Sprintf("state manager %s: %v", e.Op, e.Err)
}
```

### 8.2 Graceful Degradation

| Scenario | Behavior | Metrics |
|----------|----------|---------|
| Redis unavailable | Continue with memory-only mode | `StateRedisErrorsTotal++` |
| Invalid input | Return validation error | `StateOperationDurationSeconds` recorded |
| Context cancelled | Stop operation immediately | No error logged (expected) |
| Expired state | Auto-cleanup, return nil | `StateExpiredTotal++` |

---

## 9. Performance Requirements (150%)

| Operation | Target | Stretch Goal (150%) | Measurement |
|-----------|--------|---------------------|-------------|
| RecordInhibition | <10µs | <5µs | Benchmark |
| IsInhibited | <100ns | <50ns | Benchmark |
| RemoveInhibition | <5µs | <2µs | Benchmark |
| GetActiveInhibitions (100) | <50µs | <30µs | Benchmark |
| Memory overhead | <100 bytes/state | <80 bytes/state | Profiling |
| Test coverage | 85% | 90%+ | go test -cover |

---

## 10. Documentation (Comprehensive)

### 10.1 README Structure

```markdown
# Inhibition State Manager

## Overview
## Architecture
## Usage Examples
  - Basic usage
  - With Redis
  - Integration with Matcher
  - Cleanup worker
## Metrics & Monitoring
  - All 6 metrics explained
  - PromQL query examples
  - Grafana dashboard queries
## Testing
  - How to run tests
  - Coverage report
  - Benchmarks
## Performance
  - Benchmark results
  - Memory profiling
## Troubleshooting
  - Common issues
  - Debug logging
## API Reference
  - All methods documented
```

### 10.2 PromQL Examples

```promql
# Active inhibitions gauge
alert_history_business_inhibition_state_active

# Inhibition rate (per minute)
rate(alert_history_business_inhibition_state_records_total[1m])

# Removal rate by reason
rate(alert_history_business_inhibition_state_removals_total[5m]) by (reason)

# P95 operation latency
histogram_quantile(0.95,
  rate(alert_history_business_inhibition_state_operation_duration_seconds_bucket[5m])
) by (operation)

# Redis error rate
rate(alert_history_business_inhibition_state_redis_errors_total[5m]) by (operation)

# Expired state cleanup rate
rate(alert_history_business_inhibition_state_expired_total[1m])
```

---

## 11. Technical Risks & Mitigation

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Memory leak** (expired states) | HIGH | MEDIUM | Cleanup worker + tests |
| **Race conditions** (sync.Map) | MEDIUM | LOW | Concurrent tests + race detector |
| **Redis connection loss** | MEDIUM | MEDIUM | Graceful degradation + metrics |
| **Context cancellation** | LOW | HIGH | Proper context handling |
| **Performance regression** | LOW | LOW | Benchmarks + CI integration |

---

## 12. Dependencies

### Upstream (Completed ✅)
- ✅ **TN-126**: InhibitionRule parser (155% quality, Grade A+)
- ✅ **TN-127**: InhibitionMatcher engine (16.958µs, 95% coverage)
- ✅ **TN-128**: Active Alert Cache (58ns, 86.6% coverage)

### Downstream (Blocked by TN-129)
- 🔒 **TN-130**: Inhibition API Endpoints (deferred, optional)

---

## 13. Definition of Done (150% Quality)

### Mandatory (100%)
- [x] InhibitionState model exists ✅
- [ ] DefaultStateManager implements all 6 methods ✅ (exists, needs metrics)
- [ ] 30+ tests passing (unit + integration + concurrent)
- [ ] 85%+ test coverage
- [ ] 6 Prometheus metrics integrated
- [ ] Redis persistence working
- [ ] Cleanup worker implemented

### Enhanced (150%)
- [ ] 36 tests (exceeds 30+ by 20%)
- [ ] 90%+ test coverage (exceeds 85% by +5%)
- [ ] 6 benchmarks with performance targets met
- [ ] Comprehensive README (500+ lines)
- [ ] Integration with Matcher complete
- [ ] Error handling with custom types
- [ ] PromQL examples + Grafana queries
- [ ] Zero technical debt
- [ ] Production-ready quality (Grade A+)

---

## 14. Timeline & Effort

| Phase | Tasks | Effort | Dependencies |
|-------|-------|--------|--------------|
| **Phase 1**: Metrics | Add 6 Prometheus metrics | 30 min | pkg/metrics |
| **Phase 2**: Tests | Write 36 tests | 2 hours | - |
| **Phase 3**: Cleanup Worker | Implement background cleanup | 45 min | - |
| **Phase 4**: Integration | Wire to Matcher | 30 min | TN-127 |
| **Phase 5**: Documentation | README + examples | 45 min | - |
| **Phase 6**: Validation | Coverage + benchmarks | 30 min | - |
| **TOTAL** | - | **5 hours** | - |

**Original estimate**: 1.5 hours
**150% implementation**: 5 hours (3.3x для achieving excellence)

---

## 15. Success Criteria

### Quantitative
- ✅ 36 tests passing (100%)
- ✅ 90%+ test coverage
- ✅ RecordInhibition <5µs
- ✅ IsInhibited <50ns
- ✅ 6 Prometheus metrics operational
- ✅ Zero lint errors

### Qualitative
- ✅ Production-ready code quality
- ✅ Comprehensive documentation
- ✅ Graceful error handling
- ✅ Integration validated
- ✅ Grade A+ achievement

---

**Document Version**: 1.0
**Author**: Kilo Code
**Date**: 2025-11-05
**Status**: APPROVED FOR IMPLEMENTATION 🚀
