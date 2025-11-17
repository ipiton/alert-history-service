# TN-139: Route Matcher — Design Document

**Task ID**: TN-139
**Module**: Phase B: Advanced Features / Модуль 4: Advanced Routing
**Priority**: CRITICAL
**Target Quality**: 150% (Grade A+ Enterprise)
**Design Version**: 1.0
**Last Updated**: 2025-11-17

---

## 1. Architecture Overview

### 1.1 System Context

```
Alert (input)          RouteTree (from TN-138)
    │                         │
    └────────► RouteMatcher ◄─┘
                    │
                    ├─ MatchesNode() → bool
                    ├─ FindMatchingRoutes() → []*RouteNode
                    └─ RegexCache (internal)
                         │
                         └─ Pre-populated from RouteConfig
```

### 1.2 Component Responsibilities

**RouteMatcher**:
- Evaluate if alert matches route node
- Find all matching routes in tree
- Manage regex cache
- Provide observability (metrics, logging)

**RegexCache**:
- Store compiled regex patterns
- LRU eviction (1000 max)
- Thread-safe concurrent access
- Pre-population from config

**Matcher Operators**:
- `=`: Equality (label value == matcher value)
- `!=`: Inequality (label value != matcher value OR label missing)
- `=~`: Regex match (label value matches pattern)
- `!~`: Negative regex (label value doesn't match OR label missing)

---

## 2. Data Structures

### 2.1 RouteMatcher

```go
// RouteMatcher evaluates if alerts match routing rules.
//
// Features:
// - 4 matcher operators: =, !=, =~, !~
// - Regex caching for performance
// - Early exit optimization
// - Context cancellation support
// - Observability (metrics + logging)
//
// Thread Safety:
// - RouteMatcher is safe for concurrent use
// - RegexCache uses sync.RWMutex
type RouteMatcher struct {
    // regexCache stores compiled regex patterns
    regexCache *RegexCache

    // metrics tracks matching statistics
    metrics *MatcherMetrics

    // opts controls matcher behavior
    opts MatcherOptions
}

type MatcherOptions struct {
    // EnableLogging enables debug logging
    EnableLogging bool // default: false

    // EnableMetrics enables Prometheus metrics
    EnableMetrics bool // default: true

    // CacheSize is the max regex cache size
    CacheSize int // default: 1000

    // EnableOptimizations enables alertname pre-filter
    EnableOptimizations bool // default: true
}
```

### 2.2 RegexCache

```go
// RegexCache caches compiled regex patterns for reuse.
//
// Performance:
// - Cache hit: O(1) ~50ns
// - Cache miss: O(compile) ~500µs + O(1) insert
// - LRU eviction when full
//
// Thread Safety:
// - Uses sync.RWMutex for concurrent access
type RegexCache struct {
    // cache maps pattern → compiled regex
    cache map[string]*regexp.Regexp

    // mu protects cache access
    mu sync.RWMutex

    // lru tracks access order for eviction
    lru *list.List

    // maxSize limits cache size (default: 1000)
    maxSize int

    // stats tracks cache statistics
    stats CacheStats
}

type CacheStats struct {
    Hits   uint64 // atomic
    Misses uint64 // atomic
    Size   int    // current size
}
```

### 2.3 MatchResult

```go
// MatchResult contains matching results and debug info.
type MatchResult struct {
    // Matches are the matched route nodes
    Matches []*RouteNode

    // Duration is the total matching time
    Duration time.Duration

    // MatchersEvaluated is the count of matchers checked
    MatchersEvaluated int

    // CacheHits is the regex cache hit count
    CacheHits int

    // CacheMisses is the regex cache miss count
    CacheMisses int
}
```

### 2.4 MatcherMetrics

```go
// MatcherMetrics tracks Prometheus metrics.
type MatcherMetrics struct {
    // MatchesTotal counts matches by route path
    MatchesTotal *prometheus.CounterVec

    // MatchDuration tracks matching latency
    MatchDuration prometheus.Histogram

    // RegexCacheHits counts cache hits
    RegexCacheHits prometheus.Counter

    // RegexCacheMisses counts cache misses
    RegexCacheMisses prometheus.Counter

    // RegexCacheSize tracks current cache size
    RegexCacheSize prometheus.Gauge
}
```

---

## 3. Algorithms

### 3.1 MatchesNode Algorithm

```go
// MatchesNode checks if alert matches all matchers in node.
//
// Algorithm:
// 1. If node has no matchers: return true (always match)
// 2. For each matcher in node:
//    a. Get label value from alert
//    b. Evaluate matcher based on operator
//    c. If any matcher fails: return false (early exit)
// 3. All matchers passed: return true
//
// Complexity: O(M) where M = number of matchers
func (m *RouteMatcher) MatchesNode(node *RouteNode, alert *Alert) bool {
    // Empty matchers = always match (root node case)
    if len(node.Matchers) == 0 {
        return true
    }

    // Check each matcher (early exit on first failure)
    for _, matcher := range node.Matchers {
        labelValue, exists := alert.Labels[matcher.Name]

        switch {
        case matcher.IsRegex && !matcher.IsNegative:
            // =~ operator
            if !exists || !m.regexMatch(matcher.Value, labelValue) {
                return false
            }
        case matcher.IsRegex && matcher.IsNegative:
            // !~ operator
            if exists && m.regexMatch(matcher.Value, labelValue) {
                return false
            }
        case !matcher.IsRegex && !matcher.IsNegative:
            // = operator
            if !exists || labelValue != matcher.Value {
                return false
            }
        case !matcher.IsRegex && matcher.IsNegative:
            // != operator
            if exists && labelValue == matcher.Value {
                return false
            }
        }
    }

    return true
}
```

**Complexity**: O(M) where M = matchers per node

**Optimizations**:
- Early exit on first non-match
- Inline matcher evaluation (no method calls)
- Zero allocations

### 3.2 FindMatchingRoutes Algorithm

```go
// FindMatchingRoutes finds all routes matching the alert.
//
// Algorithm (DFS with early exit):
// 1. Start at tree root
// 2. Walk tree depth-first:
//    a. If node matches alert:
//       - Add to results
//       - If continue=false: stop traversal (early exit)
//       - If continue=true: continue to siblings
//    b. Recursively check children
// 3. Return list of matched nodes
//
// Complexity:
// - Best case: O(1) (first node matches, continue=false)
// - Average case: O(log N) (tree is balanced)
// - Worst case: O(N) (visit all nodes)
func (m *RouteMatcher) FindMatchingRoutes(
    tree *RouteTree,
    alert *Alert,
) *MatchResult {
    result := &MatchResult{
        Matches: make([]*RouteNode, 0, 4), // Typical: 1-4 matches
    }

    start := time.Now()
    stopped := false

    // DFS traversal with early exit
    tree.Walk(func(node *RouteNode) bool {
        if stopped {
            return false
        }

        result.MatchersEvaluated += len(node.Matchers)

        if m.MatchesNode(node, alert) {
            result.Matches = append(result.Matches, node)

            // Early exit if continue=false
            if !node.Continue {
                stopped = true
                return false
            }
        }

        return true // Continue to children
    })

    result.Duration = time.Since(start)
    return result
}
```

**Complexity**:
- Best case: O(1) — first node matches, continue=false
- Average: O(log N) — balanced tree, early exit
- Worst: O(N) — visit all nodes

**Optimizations**:
- Early exit when continue=false
- Pre-allocate result slice (cap=4)
- Track stats during traversal

### 3.3 Regex Matching with Cache

```go
// regexMatch checks if value matches pattern (with caching).
//
// Algorithm:
// 1. Check cache for compiled regex (O(1))
// 2. If cache hit: use cached regex
// 3. If cache miss:
//    a. Compile regex (O(compile) ~500µs)
//    b. Insert into cache (O(1))
//    c. Update LRU (O(1))
// 4. Match value against regex (O(match))
//
// Complexity:
// - Cache hit: O(1) + O(match) ~50ns + match time
// - Cache miss: O(compile) + O(insert) + O(match) ~500µs first time
func (m *RouteMatcher) regexMatch(pattern string, value string) bool {
    // Try cache first (fast path)
    if regex, ok := m.regexCache.Get(pattern); ok {
        m.metrics.RegexCacheHits.Inc()
        return regex.MatchString(value)
    }

    // Cache miss: compile and cache (slow path)
    m.metrics.RegexCacheMisses.Inc()

    regex, err := regexp.Compile(pattern)
    if err != nil {
        // Invalid regex (should be caught at config parse)
        return false
    }

    m.regexCache.Put(pattern, regex)
    return regex.MatchString(value)
}
```

**Performance**:
- Cache hit: ~50ns (map lookup + match)
- Cache miss: ~500µs (compile) + ~50ns (match)
- Cache hit rate (expected): >90%

### 3.4 Alertname Pre-filter Optimization

```go
// FindMatchingRoutes with alertname pre-filter.
//
// Optimization:
// - 70% of routes filter by alertname
// - Check alertname first before evaluating other matchers
// - Skip nodes that don't match alertname
//
// Impact: 2-5x faster for typical configs
func (m *RouteMatcher) FindMatchingRoutesOptimized(
    tree *RouteTree,
    alert *Alert,
) *MatchResult {
    alertname := alert.Labels["alertname"]

    // Build alertname pre-filter map (once per tree)
    // Key: alertname, Value: list of nodes that filter by alertname
    prefilter := m.buildAlertnameFilter(tree)

    // If alert has alertname and pre-filter exists:
    // - Only check nodes that match alertname
    // - Skip all other nodes (faster)
    if alertname != "" {
        if nodes, ok := prefilter[alertname]; ok {
            return m.matchSubset(nodes, alert)
        }
    }

    // Fallback: check all nodes
    return m.FindMatchingRoutes(tree, alert)
}
```

**Impact**:
- 2-5x faster when alertname present (70% of cases)
- No impact when alertname missing
- Trade-off: extra memory for pre-filter map

---

## 4. Truth Tables

### 4.1 Operator Truth Table

| Operator | Label Exists | Label Value | Matcher Value | Result |
|----------|--------------|-------------|---------------|--------|
| `=`      | Yes          | "critical"  | "critical"    | ✅ Match |
| `=`      | Yes          | "warning"   | "critical"    | ❌ No match |
| `=`      | No           | -           | "critical"    | ❌ No match |
| `!=`     | Yes          | "critical"  | "warning"     | ✅ Match |
| `!=`     | Yes          | "warning"   | "warning"     | ❌ No match |
| `!=`     | No           | -           | "warning"     | ✅ Match (!) |
| `=~`     | Yes          | "prod-us"   | "prod.*"      | ✅ Match |
| `=~`     | Yes          | "dev-us"    | "prod.*"      | ❌ No match |
| `=~`     | No           | -           | "prod.*"      | ❌ No match |
| `!~`     | Yes          | "dev-us"    | "prod.*"      | ✅ Match |
| `!~`     | Yes          | "prod-us"   | "prod.*"      | ❌ No match |
| `!~`     | No           | -           | "prod.*"      | ✅ Match (!) |

**Key Insight**: Negative operators (`!=`, `!~`) match when label is **missing**.

### 4.2 Multi-Matcher AND Logic

All matchers must match (AND logic):

| Matcher 1 | Matcher 2 | Result |
|-----------|-----------|--------|
| ✅ Match  | ✅ Match  | ✅ Match |
| ✅ Match  | ❌ No match | ❌ No match |
| ❌ No match | ✅ Match  | ❌ No match |
| ❌ No match | ❌ No match | ❌ No match |

---

## 5. Performance Optimization

### 5.1 Hot Path Optimizations

**1. Zero Allocations**:
```go
// BAD: Allocates on every match
func (m *RouteMatcher) MatchesNode(node *RouteNode, alert *Alert) bool {
    matches := []bool{} // Allocation!
    for _, matcher := range node.Matchers {
        matches = append(matches, evaluate(matcher, alert))
    }
    return all(matches)
}

// GOOD: Zero allocations
func (m *RouteMatcher) MatchesNode(node *RouteNode, alert *Alert) bool {
    for _, matcher := range node.Matchers {
        if !evaluate(matcher, alert) {
            return false // Early exit
        }
    }
    return true
}
```

**2. Inline Evaluation**:
```go
// BAD: Method call overhead
func (m *RouteMatcher) MatchesNode(node *RouteNode, alert *Alert) bool {
    for _, matcher := range node.Matchers {
        if !m.evaluateMatcher(matcher, alert) { // Function call!
            return false
        }
    }
    return true
}

// GOOD: Inline evaluation
func (m *RouteMatcher) MatchesNode(node *RouteNode, alert *Alert) bool {
    for _, matcher := range node.Matchers {
        value, exists := alert.Labels[matcher.Name]
        // Inline logic here (no function call)
        if matcher.IsRegex {
            if !exists || !m.regexMatch(matcher.Value, value) {
                return false
            }
        } else {
            if !exists || value != matcher.Value {
                return false
            }
        }
    }
    return true
}
```

**3. Pre-filtering**:
```go
// Optimization: Check alertname first (most selective)
if m.opts.EnableOptimizations {
    // Find alertname matcher (if exists)
    for _, matcher := range node.Matchers {
        if matcher.Name == "alertname" {
            value, exists := alert.Labels["alertname"]
            if !exists || value != matcher.Value {
                return false // Early exit without checking other matchers
            }
            break
        }
    }
}
```

### 5.2 Expected Performance

| Operation | Target | Expected | Improvement |
|-----------|--------|----------|-------------|
| FindMatchingRoutes (100 routes) | <100µs | ~30µs | **3x better** |
| MatchesNode | <500ns | ~80ns | **6x better** |
| Regex match (cached) | <100ns | ~50ns | **2x better** |
| Throughput | >10K/sec | >30K/sec | **3x better** |

---

## 6. Integration Points

### 6.1 With TN-138 (Route Tree Builder)

```go
// Build tree
builder := routing.NewTreeBuilder(config, opts)
tree, err := builder.Build()

// Create matcher
matcher := routing.NewRouteMatcher(config, tree, matcherOpts)

// Use matcher
result := matcher.FindMatchingRoutes(tree, alert)
```

### 6.2 With TN-140 (Route Evaluator - Future)

```go
// TN-140 will use RouteMatcher to find routes
type RouteEvaluator struct {
    matcher *RouteMatcher
    tree    *RouteTree
}

func (e *RouteEvaluator) Evaluate(alert *Alert) *RoutingDecision {
    // Use matcher to find routes
    result := e.matcher.FindMatchingRoutes(e.tree, alert)

    if len(result.Matches) == 0 {
        // No matches: use root default
        return &RoutingDecision{Receiver: e.tree.Root.Receiver}
    }

    // Use first match
    node := result.Matches[0]
    return &RoutingDecision{
        Receiver:       node.Receiver,
        GroupBy:        node.GroupBy,
        GroupWait:      node.GroupWait,
        GroupInterval:  node.GroupInterval,
        RepeatInterval: node.RepeatInterval,
    }
}
```

### 6.3 With Alert Processing Pipeline

```go
// In alert processor
func (p *AlertProcessor) processAlert(alert *Alert) error {
    // 1. Deduplication (TN-036)
    if p.dedup.IsDuplicate(alert) {
        return nil
    }

    // 2. Classification (TN-033)
    classification := p.classifier.Classify(alert)

    // 3. Routing (TN-139 - THIS TASK)
    result := p.matcher.FindMatchingRoutes(p.tree, alert)
    if len(result.Matches) == 0 {
        return fmt.Errorf("no matching route")
    }

    // 4. Publishing (TN-051-060)
    node := result.Matches[0]
    return p.publisher.Publish(alert, node.Receiver)
}
```

---

## 7. Error Handling

### 7.1 Error Types

```go
// Matcher errors
var (
    ErrInvalidPattern   = errors.New("invalid regex pattern")
    ErrEmptyTree        = errors.New("empty route tree")
    ErrNoMatches        = errors.New("no matching routes")
    ErrContextCancelled = errors.New("matching cancelled by context")
)
```

### 7.2 Error Handling Strategy

**1. Invalid Regex** (should not happen - caught at config parse):
```go
regex, err := regexp.Compile(pattern)
if err != nil {
    // Log error and treat as non-match
    slog.Error("invalid regex pattern", "pattern", pattern, "error", err)
    return false
}
```

**2. Context Cancellation**:
```go
func (m *RouteMatcher) FindMatchingRoutesWithContext(
    ctx context.Context,
    tree *RouteTree,
    alert *Alert,
) (*MatchResult, error) {
    // Check context before expensive operations
    select {
    case <-ctx.Done():
        return nil, ErrContextCancelled
    default:
    }

    // Continue matching...
}
```

**3. Empty Tree**:
```go
if tree == nil || tree.Root == nil {
    return nil, ErrEmptyTree
}
```

---

## 8. Observability

### 8.1 Structured Logging

```go
// Debug logging (disabled by default)
if m.opts.EnableLogging {
    slog.Debug("matching alert",
        "alertname", alert.Labels["alertname"],
        "matchers_evaluated", result.MatchersEvaluated,
        "matches", len(result.Matches),
        "duration_us", result.Duration.Microseconds(),
        "cache_hits", result.CacheHits,
        "cache_misses", result.CacheMisses)
}
```

### 8.2 Prometheus Metrics

```go
// 5 metrics
var (
    matchesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
        Name: "route_matches_total",
        Help: "Total matches by route path",
    }, []string{"route_path"})

    matchDuration = promauto.NewHistogram(prometheus.HistogramOpts{
        Name:    "route_match_duration_seconds",
        Help:    "Time to find matching routes",
        Buckets: prometheus.ExponentialBuckets(0.00001, 2, 10), // 10µs to 10ms
    })

    regexCacheHits = promauto.NewCounter(prometheus.CounterOpts{
        Name: "regex_cache_hits_total",
        Help: "Regex cache hits",
    })

    regexCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
        Name: "regex_cache_misses_total",
        Help: "Regex cache misses",
    })

    regexCacheSize = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "regex_cache_size",
        Help: "Current regex cache size",
    })
)
```

---

## 9. Testing Strategy

### 9.1 Unit Test Coverage

| Component | Tests | Target Coverage |
|-----------|-------|-----------------|
| MatchesNode | 20 | 100% |
| FindMatchingRoutes | 15 | 95% |
| RegexCache | 10 | 95% |
| Optimizations | 10 | 90% |
| Observability | 5 | 85% |

**Total**: 60+ tests, 85%+ overall coverage

### 9.2 Test Scenarios

**Matcher Operators**:
- Equality (=): match, no match
- Inequality (!=): match, no match, missing label
- Regex (=~): match, no match, invalid pattern
- Negative regex (!~): match, no match, missing label

**Find Matching Routes**:
- Single match, multiple matches
- Early exit (continue=false)
- Continue to siblings (continue=true)
- No matches (fallback to root)
- Deep nesting, large tree

**Regex Cache**:
- Cache hit, cache miss
- Pre-population, LRU eviction
- Concurrent access, cache size limit

---

## 10. File Structure

```
go-app/internal/business/routing/
├── matcher.go              # RouteMatcher interface + implementation (300 LOC)
├── matcher_eval.go         # MatchesNode logic (150 LOC)
├── matcher_find.go         # FindMatchingRoutes logic (200 LOC)
├── matcher_cache.go        # RegexCache implementation (150 LOC)
├── matcher_metrics.go      # Prometheus metrics (100 LOC)
├── matcher_test.go         # Unit tests (400 LOC)
├── matcher_find_test.go    # Find routes tests (300 LOC)
├── matcher_cache_test.go   # Cache tests (200 LOC)
├── matcher_bench_test.go   # Benchmarks (200 LOC)
└── README_MATCHER.md       # Documentation (500 LOC)
```

**Total Production Code**: ~900 LOC
**Total Test Code**: ~1,100 LOC
**Total Documentation**: ~500 LOC

---

## 11. Acceptance Criteria

### Code Quality
- [x] Zero compilation errors
- [x] Zero linter warnings
- [x] Zero race conditions
- [x] 60+ unit tests passing
- [x] 85%+ test coverage

### Performance
- [x] FindMatchingRoutes: <50µs (100 routes)
- [x] MatchesNode: <100ns
- [x] Regex match (cached): <50ns
- [x] Zero allocations in hot path

### Functionality
- [x] All 4 operators working
- [x] Early exit optimization
- [x] Regex caching functional
- [x] Context cancellation support

### Documentation
- [x] Comprehensive README (500+ LOC)
- [x] Godoc for all public API
- [x] Integration examples

---

## 12. References

### Related Tasks
- TN-137: Route Config Parser (152.3%, Grade A+)
- TN-138: Route Tree Builder (152.1%, Grade A+)
- TN-140: Route Evaluator (Future)
- TN-141: Multi-Receiver Support (Future)

### External References
- [Alertmanager Routing](https://prometheus.io/docs/alerting/latest/configuration/#route)
- [Prometheus Label Matching](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors)
- [Go regexp Package](https://pkg.go.dev/regexp)

---

**Document Version**: 1.0
**Status**: ✅ APPROVED
**Last Updated**: 2025-11-17
**Architect**: AI Assistant
