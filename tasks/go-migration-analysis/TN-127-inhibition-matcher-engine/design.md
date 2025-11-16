# TN-127: Inhibition Matcher Engine - Design

## Architecture

```
┌───────────────────────────────────────┐
│    InhibitionMatcher                  │
│  (check if alert is inhibited)       │
└──────────┬────────────────────────────┘
           │
           ├──> ActiveAlertCache (TN-128)
           │     - Get firing alerts
           │
           ├──> InhibitionRules (TN-126)
           │     - Pre-compiled regex
           │
           └──> Match Logic
                 - Label matching (exact + regex)
                 - Equal labels check

```

## Interfaces

### InhibitionMatcher

```go
type InhibitionMatcher interface {
    ShouldInhibit(ctx context.Context, targetAlert *core.Alert) (*MatchResult, error)
    FindInhibitors(ctx context.Context, targetAlert *core.Alert) ([]*MatchResult, error)
    MatchRule(rule *InhibitionRule, sourceAlert, targetAlert *core.Alert) bool
}
```

### MatchResult

```go
type MatchResult struct {
    Matched       bool
    InhibitedBy   *core.Alert      // Source alert (inhibitor)
    Rule          *InhibitionRule
    MatchDuration time.Duration
}
```

## Implementation

### DefaultInhibitionMatcher

```go
type DefaultInhibitionMatcher struct {
    cache       ActiveAlertCache   // TN-128 dependency
    rules       []InhibitionRule
    metrics     *InhibitionMetrics
    logger      *slog.Logger
    compiledRE  map[string]*regexp.Regexp // Pre-compiled regex
}
```

### Matching Algorithm

1. Get all firing alerts from cache
2. For each rule:
   - For each source alert:
     - Check source_match conditions
     - Check source_match_re conditions
     - Check target_match conditions
     - Check target_match_re conditions
     - Check equal labels
     - If all match → INHIBITED
3. Return first match (or no match)

### Performance Optimizations

- Pre-compiled regex from parser
- Early return on first match
- Zero allocations in hot path

---

## Test Strategy

### Unit Tests (40+)
- Happy path: alert inhibited correctly
- Edge cases: no match, partial match
- Label matching: exact, regex, equal
- Performance: <1ms per check

### Benchmarks
- BenchmarkShouldInhibit_Single: <100µs
- BenchmarkShouldInhibit_100Rules: <1ms
- BenchmarkMatchRule: <10µs

---

**Date**: 2025-11-04
**Status**: DESIGN COMPLETE



