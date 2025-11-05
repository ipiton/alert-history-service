# TN-132: Silence Matcher Engine - Requirements

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-132
**Status**: 🟡 IN PROGRESS
**Priority**: HIGH
**Estimated Effort**: 10-14 hours
**Dependencies**: TN-131 (Silence Data Models) ✅ COMPLETE

---

## 📋 Overview

Реализовать Silence Matcher Engine для проверки соответствия алертов активным silences. Engine должен поддерживать все 4 типа label matchers (=, !=, =~, !~) с полной совместимостью Alertmanager API v2 и ultra-high performance (<1ms matching time).

### Business Value
- **Alert Suppression**: Автоматическое подавление алертов во время maintenance windows
- **Noise Reduction**: Снижение шума от известных проблем до 70-80%
- **Alertmanager Compatibility**: 100% совместимость с существующими silences
- **Performance**: Sub-millisecond matching для real-time alert processing

---

## 🎯 Goals

### Primary Goals
1. ✅ Реализовать `SilenceMatcher` interface для проверки alerts против silences
2. ✅ Поддержка всех 4 операторов: `=` (equal), `!=` (not equal), `=~` (regex), `!~` (not regex)
3. ✅ Regex pattern compilation и caching для performance
4. ✅ Multi-matcher support (AND logic - все matchers должны совпасть)
5. ✅ Performance optimization: <1ms для проверки single alert против 100 silences

### Secondary Goals
- Comprehensive error handling для invalid regex patterns
- Prometheus metrics для matching operations
- Structured logging для debugging
- Benchmark tests для performance validation
- Memory-efficient matcher implementation

---

## 📐 Functional Requirements

### FR-1: SilenceMatcher Interface

**Interface Definition**:
```go
package silencing

import (
    "context"
)

// SilenceMatcher interface для проверки соответствия alerts silences.
type SilenceMatcher interface {
    // Matches проверяет, соответствует ли alert данному silence.
    // Возвращает true если ВСЕ matchers в silence совпали с alert labels (AND logic).
    //
    // Алгоритм:
    //   1. Iterate через все matchers в silence
    //   2. Для каждого matcher проверить соответствие alert labels
    //   3. Если хотя бы один matcher не совпал → return false
    //   4. Если все matchers совпали → return true
    Matches(ctx context.Context, alert Alert, silence *Silence) (bool, error)

    // MatchesAny проверяет, соответствует ли alert ЛЮБОМУ из silences.
    // Возвращает список matched silence IDs и error.
    // Оптимизация: early return при первом совпадении (опционально).
    MatchesAny(ctx context.Context, alert Alert, silences []*Silence) ([]string, error)
}

// Alert represents alert для matching (subset of full Alert model).
type Alert struct {
    Labels      map[string]string  // Alert labels
    Annotations map[string]string  // Alert annotations (optional)
}
```

### FR-2: Matching Operators

**Operator 1: Equal (=)**
```go
// Matcher: {Name: "job", Value: "api-server", Type: "="}
// Alert label: job="api-server" → MATCH ✅
// Alert label: job="web-server" → NO MATCH ❌
// Alert label: job="" → NO MATCH ❌
// Alert missing "job" label → NO MATCH ❌
```

**Operator 2: Not Equal (!=)**
```go
// Matcher: {Name: "env", Value: "prod", Type: "!="}
// Alert label: env="dev" → MATCH ✅
// Alert label: env="prod" → NO MATCH ❌
// Alert missing "env" label → MATCH ✅ (считается как not equal)
```

**Operator 3: Regex (=~)**
```go
// Matcher: {Name: "severity", Value: "(critical|warning)", Type: "=~"}
// Alert label: severity="critical" → MATCH ✅
// Alert label: severity="warning" → MATCH ✅
// Alert label: severity="info" → NO MATCH ❌
// Alert missing "severity" label → NO MATCH ❌
```

**Operator 4: Not Regex (!~)**
```go
// Matcher: {Name: "instance", Value: ".*-dev-.*", Type: "!~"}
// Alert label: instance="server-prod-01" → MATCH ✅
// Alert label: instance="server-dev-01" → NO MATCH ❌
// Alert missing "instance" label → MATCH ✅ (считается как not match)
```

### FR-3: Multi-Matcher AND Logic

**Example Silence**:
```yaml
matchers:
  - name: alertname
    value: HighCPU
    type: "="
  - name: job
    value: api-server
    type: "="
  - name: severity
    value: "(critical|warning)"
    type: "=~"
```

**Matching Logic**:
- Alert MUST match ALL 3 matchers
- If ANY matcher fails → silence does NOT apply
- Empty matcher list → silence never matches (validation prevents this in TN-131)

### FR-4: Performance Requirements

| Operation | Target | Notes |
|-----------|--------|-------|
| Single matcher check | <10µs | For = and != operators |
| Regex matcher check | <100µs | First match (uncached) |
| Regex matcher check | <10µs | Subsequent (cached) |
| Full silence match (10 matchers) | <500µs | All matchers checked |
| MatchesAny (100 silences) | <1ms | Average case |
| MatchesAny (1000 silences) | <10ms | Worst case |

**Optimization Strategies**:
1. **Regex Compilation Cache**: Pre-compile и cache regex patterns
2. **Early Exit**: Return false on first non-matching matcher
3. **Label Lookup Optimization**: Use map lookup O(1) for exact matches
4. **Concurrent Matching**: Optional parallel checking для large silence lists (if >100 silences)

---

## 🔧 Technical Requirements

### TR-1: Implementation Structure

**File Structure**:
```
go-app/internal/core/silencing/
├── models.go              # (Existing) Silence, Matcher models
├── matcher.go             # NEW: SilenceMatcher interface
├── matcher_impl.go        # NEW: DefaultSilenceMatcher implementation
├── matcher_cache.go       # NEW: Regex cache for performance
├── matcher_test.go        # NEW: Unit tests
└── matcher_bench_test.go  # NEW: Benchmark tests
```

### TR-2: Regex Compilation Cache

**Cache Design**:
```go
package silencing

import (
    "regexp"
    "sync"
)

// RegexCache caches compiled regex patterns для performance.
type RegexCache struct {
    mu     sync.RWMutex
    cache  map[string]*regexp.Regexp  // pattern → compiled regex
    maxSize int                        // Max cache entries (default: 1000)
}

// Get возвращает compiled regex из cache или компилирует новый.
func (rc *RegexCache) Get(pattern string) (*regexp.Regexp, error) {
    // 1. Try read lock first (fast path)
    rc.mu.RLock()
    if re, ok := rc.cache[pattern]; ok {
        rc.mu.RUnlock()
        return re, nil
    }
    rc.mu.RUnlock()

    // 2. Compile and cache (slow path)
    rc.mu.Lock()
    defer rc.mu.Unlock()

    // Double-check after acquiring write lock
    if re, ok := rc.cache[pattern]; ok {
        return re, nil
    }

    // Compile new regex
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, err
    }

    // Cache with size limit (LRU eviction optional)
    if len(rc.cache) >= rc.maxSize {
        // Simple eviction: clear cache (can optimize with LRU later)
        rc.cache = make(map[string]*regexp.Regexp)
    }
    rc.cache[pattern] = re

    return re, nil
}
```

### TR-3: Error Handling

**Error Types**:
```go
var (
    // ErrInvalidAlert indicates alert has no labels (invalid input)
    ErrInvalidAlert = errors.New("invalid alert: labels cannot be nil")

    // ErrInvalidSilence indicates silence is nil or has no matchers
    ErrInvalidSilence = errors.New("invalid silence: cannot be nil or have zero matchers")

    // ErrRegexCompilationFailed indicates regex pattern compilation failed
    ErrRegexCompilationFailed = errors.New("regex pattern compilation failed")

    // ErrContextCancelled indicates context was cancelled during matching
    ErrContextCancelled = errors.New("matching cancelled: context done")
)
```

**Error Wrapping**:
```go
// Wrap regex compilation errors with context
if _, err := regexp.Compile(pattern); err != nil {
    return fmt.Errorf("%w: pattern=%q: %v", ErrRegexCompilationFailed, pattern, err)
}
```

### TR-4: Context Support

**Cancellation Handling**:
```go
func (m *DefaultSilenceMatcher) MatchesAny(ctx context.Context, alert Alert, silences []*Silence) ([]string, error) {
    var matchedIDs []string

    for _, silence := range silences {
        // Check context cancellation on each iteration
        select {
        case <-ctx.Done():
            return matchedIDs, ErrContextCancelled
        default:
        }

        matched, err := m.Matches(ctx, alert, silence)
        if err != nil {
            return matchedIDs, err
        }
        if matched {
            matchedIDs = append(matchedIDs, silence.ID)
        }
    }

    return matchedIDs, nil
}
```

### TR-5: Testing Requirements

**Unit Tests (40+ tests)**:
1. **Exact Match Tests (=)**: 8 tests
   - Match found
   - Match not found
   - Missing label
   - Empty value
   - Case sensitivity
   - Multiple matchers
   - Special characters
   - Unicode labels

2. **Not Equal Tests (!=)**: 6 tests
   - Value different → match
   - Value same → no match
   - Missing label → match (important!)
   - Empty value handling
   - Multiple not-equal matchers
   - Edge cases

3. **Regex Tests (=~)**: 10 tests
   - Simple pattern match
   - Complex pattern match
   - Character classes [a-z]
   - Quantifiers (*, +, ?)
   - Groups and alternation (a|b)
   - Anchors (^, $)
   - Missing label → no match
   - Invalid regex → error
   - Cache hit performance
   - Cache miss performance

4. **Not Regex Tests (!~)**: 6 tests
   - Pattern not matched → match
   - Pattern matched → no match
   - Missing label → match
   - Invalid regex → error
   - Edge cases
   - Cache behavior

5. **Multi-Matcher Tests (AND logic)**: 8 tests
   - All matchers match → success
   - One matcher fails → no match
   - Mixed types (=, !=, =~, !~)
   - Empty matcher list → no match
   - 10 matchers all match
   - Order independence
   - Short-circuit on first failure
   - Performance with many matchers

6. **MatchesAny Tests**: 6 tests
   - No silences → empty result
   - No matches → empty result
   - Single match
   - Multiple matches
   - 100 silences performance
   - Context cancellation

7. **Error Handling Tests**: 4 tests
   - Nil alert
   - Nil silence
   - Invalid regex in matcher
   - Context cancellation

8. **Edge Cases**: 4 tests
   - Alert with no labels
   - Silence with 100 matchers (max)
   - Unicode label names/values
   - Very long label values (1024 chars)

**Total**: 52 comprehensive tests

**Benchmarks (10+ benchmarks)**:
```go
BenchmarkMatcherEqual              // = operator
BenchmarkMatcherNotEqual           // != operator
BenchmarkMatcherRegex_CacheHit     // =~ (cached)
BenchmarkMatcherRegex_CacheMiss    // =~ (uncached)
BenchmarkMatcherNotRegex           // !~
BenchmarkMultiMatcher_10Matchers   // 10 matchers AND logic
BenchmarkMatchesAny_10Silences     // 10 silences
BenchmarkMatchesAny_100Silences    // 100 silences
BenchmarkMatchesAny_1000Silences   // 1000 silences (stress test)
BenchmarkRegexCache_Concurrent     // Cache under load
```

**Test Coverage Target**: ≥90% (higher than TN-131's 98.2%)

---

## 🔒 Security Requirements

### SEC-1: Regex DoS Prevention

**Protection Mechanisms**:
1. **Pattern Length Limit**: Already enforced by TN-131 (max 1024 chars)
2. **Compilation Timeout**: Use `regexp.Compile` (no timeout needed, fast enough)
3. **Cache Size Limit**: Max 1000 cached patterns to prevent memory exhaustion
4. **No User-Controlled Regex**: Regex patterns come from validated silences only

### SEC-2: Input Validation

**Validation Checks**:
```go
func (m *DefaultSilenceMatcher) Matches(ctx context.Context, alert Alert, silence *Silence) (bool, error) {
    // Validate inputs before processing
    if alert.Labels == nil {
        return false, ErrInvalidAlert
    }
    if silence == nil || len(silence.Matchers) == 0 {
        return false, ErrInvalidSilence
    }

    // Proceed with matching...
}
```

### SEC-3: Resource Limits

**Memory Limits**:
- Regex cache: Max 1000 entries × ~500 bytes = ~500 KB
- Total matcher memory: <1 MB (negligible)

**CPU Limits**:
- Context cancellation support for long-running matches
- Early exit on first non-matching matcher

---

## 📊 Success Criteria

### Must Have (100% Required)
- ✅ `SilenceMatcher` interface defined
- ✅ `DefaultSilenceMatcher` implementation with all 4 operators
- ✅ `RegexCache` for performance optimization
- ✅ 52+ unit tests with ≥90% coverage
- ✅ 10+ benchmarks proving <1ms performance
- ✅ Error handling with custom error types
- ✅ Context cancellation support
- ✅ Godoc documentation for all public APIs

### Should Have (150% Target)
- Concurrent matching для >100 silences (optional optimization)
- Prometheus metrics integration (`silencing_matches_total`, `silencing_match_duration_seconds`)
- Structured logging with `slog` for debugging
- Negative label matching optimization (!=, !~)
- Cache eviction strategy (LRU or simple clear)

### Could Have (Nice to Have)
- Matcher explain function для debugging (why alert matched/didn't match)
- Matcher statistics (cache hit rate, avg match time)
- Configuration options (cache size, concurrent threshold)
- Advanced metrics (per-operator latency, cache efficiency)

---

## 🔗 Dependencies

### Internal Dependencies
- ✅ TN-131: Silence Data Models (`Silence`, `Matcher`, `MatcherType`)
- `internal/core/interfaces.go`: Alert model (if available)

### External Dependencies
- `regexp` (standard library) - Regex pattern matching
- `context` (standard library) - Cancellation support
- `sync` (standard library) - Mutex для cache
- `errors` (standard library) - Error handling

---

## 📚 References

- [Alertmanager Silencing](https://prometheus.io/docs/alerting/latest/alertmanager/#silences)
- [Prometheus Label Matchers](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors)
- [Go regexp Package](https://pkg.go.dev/regexp)
- [Alertmanager Source](https://github.com/prometheus/alertmanager/tree/main/silence)

---

## 🎯 Quality Target: 150%

**Baseline (100%)**:
- All functional requirements met
- 52 tests passing
- 90% coverage
- <1ms performance
- Zero linter errors

**150% Target (Exceptional)**:
- 95%+ test coverage (+5% над baseline)
- Performance 2x better than targets (<500µs для 100 silences)
- Comprehensive benchmarks (10+)
- Prometheus metrics integration
- Structured logging
- Comprehensive godoc with examples
- Zero technical debt
- Cache efficiency >80% (hit rate)

---

**Created**: 2025-11-05
**Author**: Alertmanager++ Team
**Target Completion**: 2025-11-05 EOD
**Estimated Duration**: 10-14 hours → **Target: 6-8 hours** (matching TN-131's 2x efficiency)
