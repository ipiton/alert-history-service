# TN-132: Silence Matcher Engine - Design Document

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-132
**Version**: 1.0
**Last Updated**: 2025-11-05
**Dependencies**: TN-131 ✅ COMPLETE

---

## 🎯 Design Overview

Silence Matcher Engine реализует высокопроизводительную систему проверки соответствия алертов активным silences. Ключевые особенности дизайна:

1. **Ultra-High Performance**: <1ms для проверки против 100 silences через regex caching
2. **Full Operator Support**: Все 4 оператора (=, !=, =~, !~) с полной семантикой Alertmanager
3. **Thread-Safe Caching**: Concurrent-safe regex cache с RWMutex
4. **Context-Aware**: Graceful cancellation через context.Context
5. **Zero Technical Debt**: Clean architecture, comprehensive tests, no TODOs

### Key Design Decisions

1. **In-Memory Regex Cache**: LRU cache (max 1000 patterns) для избежания repeated compilation
2. **Early Exit Strategy**: Stop matching на first non-matching matcher (AND logic optimization)
3. **Negative Matching Semantics**: Missing labels = "not equal" для != и !~ operators (Alertmanager compatibility)
4. **No Goroutines by Default**: Single-threaded matching (fast enough), concurrent option для >100 silences
5. **Fail-Fast Validation**: Input validation перед любой бизнес-логикой

---

## 📐 Architecture

### Component Structure

```
┌─────────────────────────────────────────────────────────────────┐
│                      Silence Matcher Engine                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌────────────────┐      ┌──────────────────┐                  │
│  │  SilenceMatcher│◄─────│ DefaultMatcher   │                  │
│  │   (Interface)  │      │  (Implementation)│                  │
│  └────────────────┘      └──────────────────┘                  │
│         ▲                         │                             │
│         │                         │ uses                        │
│         │                         ▼                             │
│  ┌────────────────┐      ┌──────────────────┐                  │
│  │  Alert Labels  │      │   RegexCache     │                  │
│  │  (map[string]  │      │  (LRU, 1000 max) │                  │
│  │   string)      │      │  Thread-Safe     │                  │
│  └────────────────┘      └──────────────────┘                  │
│         │                         │                             │
│         │                         │                             │
│         ▼                         ▼                             │
│  ┌──────────────────────────────────────────┐                  │
│  │          Matching Logic                   │                  │
│  │  ┌─────────────────────────────────────┐ │                  │
│  │  │ Operator = (Equal)                  │ │                  │
│  │  │ Operator != (NotEqual)              │ │                  │
│  │  │ Operator =~ (Regex)                 │ │                  │
│  │  │ Operator !~ (NotRegex)              │ │                  │
│  │  └─────────────────────────────────────┘ │                  │
│  └──────────────────────────────────────────┘                  │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
┌─────────────┐
│   Alert     │
│  (Labels)   │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────┐
│  Matches(ctx, alert, silence)       │
│  ┌───────────────────────────────┐  │
│  │ 1. Validate inputs            │  │
│  │ 2. For each matcher in silence│  │
│  │    ├─ Get label value         │  │
│  │    ├─ Match based on type:    │  │
│  │    │  ├─ = : exact match      │  │
│  │    │  ├─ !=: not equal        │  │
│  │    │  ├─ =~: regex match      │  │
│  │    │  └─ !~: not regex match  │  │
│  │    └─ Return false on mismatch│  │
│  │ 3. All matchers passed?       │  │
│  │    └─ Return true             │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
       │
       ▼
┌─────────────┐
│   Result    │
│ (bool, err) │
└─────────────┘
```

---

## 🗂️ Component Design

### 1. SilenceMatcher Interface

```go
package silencing

import (
    "context"
)

// SilenceMatcher проверяет соответствие alerts silences.
//
// Thread-safety: Implementations MUST be thread-safe.
// Context: All methods MUST respect context cancellation.
type SilenceMatcher interface {
    // Matches проверяет, соответствует ли alert данному silence.
    // Возвращает true если ВСЕ matchers в silence совпали (AND logic).
    //
    // Errors:
    //   - ErrInvalidAlert: если alert.Labels == nil
    //   - ErrInvalidSilence: если silence == nil или len(Matchers) == 0
    //   - ErrRegexCompilationFailed: если regex pattern invalid
    //   - ErrContextCancelled: если ctx.Done()
    Matches(ctx context.Context, alert Alert, silence *Silence) (bool, error)

    // MatchesAny проверяет, соответствует ли alert ЛЮБОМУ из silences.
    // Возвращает список matched silence IDs.
    //
    // Performance: O(N*M) где N = len(silences), M = avg matchers per silence
    // Optimization: Early exit на первом совпадении НЕ применяется (нужны ВСЕ matches)
    MatchesAny(ctx context.Context, alert Alert, silences []*Silence) ([]string, error)
}

// Alert представляет alert для matching (simplified).
type Alert struct {
    Labels      map[string]string  // Required: alert labels
    Annotations map[string]string  // Optional: annotations (not used for matching)
}
```

### 2. DefaultSilenceMatcher Implementation

```go
// DefaultSilenceMatcher реализует SilenceMatcher с regex caching.
type DefaultSilenceMatcher struct {
    regexCache *RegexCache  // Shared regex cache for performance
}

// NewSilenceMatcher создает новый DefaultSilenceMatcher с default settings.
func NewSilenceMatcher() *DefaultSilenceMatcher {
    return &DefaultSilenceMatcher{
        regexCache: NewRegexCache(1000), // Max 1000 cached patterns
    }
}

// Matches implements SilenceMatcher interface.
func (m *DefaultSilenceMatcher) Matches(ctx context.Context, alert Alert, silence *Silence) (bool, error) {
    // 1. Validate inputs
    if alert.Labels == nil {
        return false, ErrInvalidAlert
    }
    if silence == nil || len(silence.Matchers) == 0 {
        return false, ErrInvalidSilence
    }

    // 2. Check all matchers (AND logic)
    for _, matcher := range silence.Matchers {
        // Check context cancellation
        select {
        case <-ctx.Done():
            return false, ErrContextCancelled
        default:
        }

        // Match based on operator type
        matched, err := m.matchSingle(alert.Labels, &matcher)
        if err != nil {
            return false, err
        }
        if !matched {
            return false, nil  // Early exit on first mismatch
        }
    }

    return true, nil  // All matchers passed
}

// matchSingle проверяет single matcher против alert labels.
func (m *DefaultSilenceMatcher) matchSingle(labels map[string]string, matcher *Matcher) (bool, error) {
    labelValue, labelExists := labels[matcher.Name]

    switch matcher.Type {
    case MatcherTypeEqual:
        // = operator: label must exist AND equal value
        return labelExists && labelValue == matcher.Value, nil

    case MatcherTypeNotEqual:
        // != operator: label missing OR not equal value
        return !labelExists || labelValue != matcher.Value, nil

    case MatcherTypeRegex:
        // =~ operator: label must exist AND match regex
        if !labelExists {
            return false, nil
        }
        re, err := m.regexCache.Get(matcher.Value)
        if err != nil {
            return false, fmt.Errorf("%w: %v", ErrRegexCompilationFailed, err)
        }
        return re.MatchString(labelValue), nil

    case MatcherTypeNotRegex:
        // !~ operator: label missing OR not match regex
        if !labelExists {
            return true, nil  // Missing = not matched
        }
        re, err := m.regexCache.Get(matcher.Value)
        if err != nil {
            return false, fmt.Errorf("%w: %v", ErrRegexCompilationFailed, err)
        }
        return !re.MatchString(labelValue), nil

    default:
        return false, ErrMatcherInvalidType
    }
}
```

### 3. RegexCache Design

```go
package silencing

import (
    "regexp"
    "sync"
)

// RegexCache кэширует compiled regex patterns для performance.
// Thread-safe: использует RWMutex для concurrent access.
// Eviction: Simple clear при достижении maxSize (LRU можно добавить позже).
type RegexCache struct {
    mu      sync.RWMutex
    cache   map[string]*regexp.Regexp
    maxSize int
}

// NewRegexCache создает новый RegexCache с заданным max size.
func NewRegexCache(maxSize int) *RegexCache {
    return &RegexCache{
        cache:   make(map[string]*regexp.Regexp, maxSize),
        maxSize: maxSize,
    }
}

// Get возвращает compiled regex из cache или компилирует новый.
//
// Performance:
//   - Cache hit: ~10ns (RLock + map lookup)
//   - Cache miss: ~5µs (compile + Lock + insert)
//
// Thread-safety: RWMutex ensures safe concurrent access.
func (rc *RegexCache) Get(pattern string) (*regexp.Regexp, error) {
    // Fast path: Try read lock first
    rc.mu.RLock()
    if re, ok := rc.cache[pattern]; ok {
        rc.mu.RUnlock()
        return re, nil
    }
    rc.mu.RUnlock()

    // Slow path: Compile and cache
    rc.mu.Lock()
    defer rc.mu.Unlock()

    // Double-check after acquiring write lock
    if re, ok := rc.cache[pattern]; ok {
        return re, nil
    }

    // Compile regex
    re, err := regexp.Compile(pattern)
    if err != nil {
        return nil, err
    }

    // Eviction strategy: Simple clear when full
    if len(rc.cache) >= rc.maxSize {
        rc.cache = make(map[string]*regexp.Regexp, rc.maxSize)
    }

    rc.cache[pattern] = re
    return re, nil
}

// Size возвращает текущий размер cache (for testing/metrics).
func (rc *RegexCache) Size() int {
    rc.mu.RLock()
    defer rc.mu.RUnlock()
    return len(rc.cache)
}

// Clear очищает весь cache (for testing).
func (rc *RegexCache) Clear() {
    rc.mu.Lock()
    defer rc.mu.Unlock()
    rc.cache = make(map[string]*regexp.Regexp, rc.maxSize)
}
```

---

## 🚀 Performance Design

### Performance Targets

| Operation | Target | Strategy |
|-----------|--------|----------|
| Single matcher (=) | <10µs | O(1) map lookup |
| Single matcher (!=) | <10µs | O(1) map lookup + negation |
| Single matcher (=~) cached | <10µs | RLock + map lookup + MatchString |
| Single matcher (=~) uncached | <100µs | Compile + cache + match |
| Full silence (10 matchers) | <500µs | Early exit на first mismatch |
| MatchesAny (100 silences) | <1ms | Linear scan с early exits |

### Optimization Strategies

**1. Regex Compilation Cache**
```
WITHOUT Cache:
  - 100 alerts × 10 silences × 3 regex matchers = 3,000 compilations
  - Total time: 3,000 × 5µs = 15ms

WITH Cache (80% hit rate):
  - Cache hits: 3,000 × 0.8 × 10ns = 24µs
  - Cache misses: 3,000 × 0.2 × 5µs = 3ms
  - Total time: 3.024ms (5x improvement!)
```

**2. Early Exit Strategy**
```go
// BAD: Check all matchers even if first one fails
for _, matcher := range silence.Matchers {
    results = append(results, checkMatcher(matcher))
}
return allTrue(results)  // Wasted computation!

// GOOD: Exit immediately on first mismatch
for _, matcher := range silence.Matchers {
    if !checkMatcher(matcher) {
        return false  // ⚡ Early exit
    }
}
return true
```

**3. Negative Matching Optimization**
```go
// != and !~ operators benefit from missing labels (no regex compilation needed)
case MatcherTypeNotEqual:
    return !labelExists || labelValue != matcher.Value, nil
    // ↑ If label missing, return true immediately (no comparison)

case MatcherTypeNotRegex:
    if !labelExists {
        return true, nil  // ⚡ Fast path for missing labels
    }
    // Only compile regex if label exists
```

### Memory Design

**Memory Footprint Estimate**:
```
DefaultSilenceMatcher:
  ├─ regexCache pointer: 8 bytes
  └─ RegexCache struct:
      ├─ mu (RWMutex): 24 bytes
      ├─ cache map: 8 bytes (pointer)
      ├─ maxSize: 8 bytes
      └─ map data (1000 entries):
          └─ ~500 bytes per entry × 1000 = 500 KB

Total: ~500 KB (acceptable for performance benefit)
```

---

## 🧪 Testing Strategy

### Test Categories

**1. Unit Tests - Operator Correctness (24 tests)**
```go
TestMatcherEqual_Matched
TestMatcherEqual_NotMatched
TestMatcherEqual_MissingLabel
TestMatcherEqual_EmptyValue
TestMatcherEqual_CaseSensitive
TestMatcherEqual_Unicode

TestMatcherNotEqual_ValueDifferent
TestMatcherNotEqual_ValueSame
TestMatcherNotEqual_MissingLabel  // Critical: must match!
TestMatcherNotEqual_EmptyValue

TestMatcherRegex_SimplePattern
TestMatcherRegex_ComplexPattern
TestMatcherRegex_CharacterClass
TestMatcherRegex_Quantifiers
TestMatcherRegex_Groups
TestMatcherRegex_Anchors
TestMatcherRegex_MissingLabel
TestMatcherRegex_InvalidPattern

TestMatcherNotRegex_NotMatched
TestMatcherNotRegex_Matched
TestMatcherNotRegex_MissingLabel  // Critical: must match!
TestMatcherNotRegex_InvalidPattern
```

**2. Integration Tests - Multi-Matcher Logic (12 tests)**
```go
TestMultiMatcher_AllMatch
TestMultiMatcher_OneFailsAllFail
TestMultiMatcher_MixedTypes
TestMultiMatcher_EmptyList
TestMultiMatcher_TenMatchers
TestMultiMatcher_OrderIndependent
TestMultiMatcher_ShortCircuit
TestMultiMatcher_Performance

TestMatchesAny_NoSilences
TestMatchesAny_NoMatches
TestMatchesAny_SingleMatch
TestMatchesAny_MultipleMatches
```

**3. Performance Tests - Benchmarks (10 tests)**
```go
BenchmarkMatcherEqual                 // Target: <10µs
BenchmarkMatcherNotEqual              // Target: <10µs
BenchmarkMatcherRegex_CacheHit        // Target: <10µs
BenchmarkMatcherRegex_CacheMiss       // Target: <100µs
BenchmarkMatcherNotRegex              // Target: <10µs
BenchmarkMultiMatcher_10Matchers      // Target: <500µs
BenchmarkMatchesAny_10Silences        // Target: <100µs
BenchmarkMatchesAny_100Silences       // Target: <1ms
BenchmarkMatchesAny_1000Silences      // Target: <10ms
BenchmarkRegexCache_Concurrent        // Validate thread-safety
```

**4. Error Handling Tests (8 tests)**
```go
TestMatches_NilAlert
TestMatches_NilAlertLabels
TestMatches_NilSilence
TestMatches_EmptyMatchers
TestMatches_InvalidRegex
TestMatches_ContextCancelled
TestMatchesAny_ContextCancelled
TestRegexCache_CompilationError
```

**5. Edge Cases (8 tests)**
```go
TestMatcher_VeryLongValue          // 1024 chars
TestMatcher_SpecialCharacters      // \n, \t, etc.
TestMatcher_UnicodeLabels          // 日本語, эмодзи 🎉
TestRegexCache_MaxSize             // Eviction behavior
TestRegexCache_ConcurrentAccess    // Race condition test
TestMultiMatcher_100Matchers       // Max matchers
TestMatchesAny_1000Silences        // Large silence list
TestMatcher_AllOperatorsInOneSilence
```

### Test Coverage Goals

- **Target**: ≥90% (higher than TN-131's 98.2% as stretch goal)
- **Critical Paths**: 100% coverage
  - All 4 operator types
  - Error handling
  - Regex cache
  - Context cancellation

---

## 🔒 Security Design

### Threat Model

| Threat | Severity | Mitigation |
|--------|----------|------------|
| Regex DoS | MEDIUM | Pattern length limit (1024 chars, enforced by TN-131) |
| Memory Exhaustion | LOW | Cache size limit (1000 patterns, ~500 KB max) |
| Context Leak | LOW | Context cancellation checks in loops |
| Invalid Input | LOW | Fail-fast validation at entry points |

### Security Controls

**1. Input Validation**
```go
// Validate BEFORE any processing
if alert.Labels == nil {
    return false, ErrInvalidAlert
}
if silence == nil || len(silence.Matchers) == 0 {
    return false, ErrInvalidSilence
}
```

**2. Regex Pattern Safety**
```go
// TN-131 validation already ensures:
//   - Max pattern length: 1024 chars
//   - Valid regex syntax
//   - No backtracking bombs (enforced by Go's RE2 engine)

// Additional safety in cache:
if len(rc.cache) >= rc.maxSize {
    rc.cache = make(map[string]*regexp.Regexp, rc.maxSize)
    // ↑ Prevent unbounded growth
}
```

**3. Context Cancellation**
```go
// Check cancellation in tight loops
for _, silence := range silences {
    select {
    case <-ctx.Done():
        return matchedIDs, ErrContextCancelled
    default:
    }
    // ... matching logic ...
}
```

---

## 📊 Observability Design (150% Target)

### Prometheus Metrics

```go
package silencing

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // silenceMatchesTotalCounter counts total match operations
    silenceMatchesTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "alert_history_business_silence_matches_total",
            Help: "Total number of silence matching operations",
        },
        []string{"result"},  // result: "matched", "not_matched", "error"
    )

    // silenceMatchDuration tracks match operation latency
    silenceMatchDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "alert_history_business_silence_match_duration_seconds",
            Help: "Duration of silence matching operations",
            Buckets: []float64{.00001, .00005, .0001, .0005, .001, .005, .01},
        },
        []string{"operation"},  // operation: "single", "any"
    )

    // regexCacheHitsTotal counts regex cache hits/misses
    regexCacheHitsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "alert_history_technical_silence_regex_cache_total",
            Help: "Total regex cache hits and misses",
        },
        []string{"result"},  // result: "hit", "miss"
    )

    // regexCacheSizeGauge tracks current cache size
    regexCacheSizeGauge = promauto.NewGauge(
        prometheus.GaugeOpts{
            Name: "alert_history_technical_silence_regex_cache_size",
            Help: "Current number of entries in regex cache",
        },
    )
)
```

### Structured Logging

```go
import "log/slog"

func (m *DefaultSilenceMatcher) Matches(ctx context.Context, alert Alert, silence *Silence) (bool, error) {
    logger := slog.Default().With(
        slog.String("silence_id", silence.ID),
        slog.Int("matcher_count", len(silence.Matchers)),
    )

    logger.Debug("starting silence match",
        slog.Any("alert_labels", alert.Labels))

    // ... matching logic ...

    if matched {
        logger.Info("silence matched",
            slog.Duration("duration", elapsed))
    }

    return matched, nil
}
```

---

## 🔗 Integration Points

### Upstream Dependencies
- ✅ TN-131: `Silence`, `Matcher`, `MatcherType` models (COMPLETE)

### Downstream Consumers
- TN-133: Silence Storage (uses `SilenceMatcher` для проверки active silences)
- TN-134: Silence Manager (lifecycle management)
- Alert Processor: Integration для real-time matching

### External Interfaces
```go
// Integration example: Alert Processor
import "internal/core/silencing"

type AlertProcessor struct {
    silenceMatcher silencing.SilenceMatcher
    // ...
}

func (p *AlertProcessor) Process(ctx context.Context, alert Alert) error {
    // Get active silences from storage
    silences, err := p.silenceStorage.GetActive(ctx)
    if err != nil {
        return err
    }

    // Check if alert matches any silence
    matchedIDs, err := p.silenceMatcher.MatchesAny(ctx, alert, silences)
    if err != nil {
        return err
    }

    if len(matchedIDs) > 0 {
        // Alert is silenced - suppress notification
        log.Info("alert silenced", "silenceIDs", matchedIDs)
        return nil
    }

    // Proceed with alert processing...
}
```

---

## 🎯 Definition of Done

- ✅ `matcher.go` with `SilenceMatcher` interface
- ✅ `matcher_impl.go` with `DefaultSilenceMatcher` implementation
- ✅ `matcher_cache.go` with `RegexCache` implementation
- ✅ `matcher_test.go` with 52+ unit tests
- ✅ `matcher_bench_test.go` with 10+ benchmarks
- ✅ Test coverage ≥90%
- ✅ All benchmarks meet performance targets
- ✅ Zero linter errors (`golangci-lint`)
- ✅ Godoc documentation complete
- ✅ README.md updated with usage examples
- ✅ Code committed to git (feature branch)

### Quality Gates (150% Target)

**Baseline (100%)**:
- ✅ All 4 operators working correctly
- ✅ 90% test coverage
- ✅ <1ms performance

**150% Target**:
- ✅ 95%+ test coverage
- ✅ <500µs performance (2x better)
- ✅ Prometheus metrics integrated
- ✅ Structured logging with slog
- ✅ Comprehensive godoc with examples
- ✅ Benchmarks document cache efficiency

---

**Designed**: 2025-11-05
**Approved**: 2025-11-05
**Target Implementation**: 6-8 hours
**Quality Target**: 150% (Grade A+, matching TN-131's excellence)
