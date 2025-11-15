# TN-127: Inhibition Matcher Engine - Requirements

## Overview

**Task**: Implement inhibition matcher engine for evaluating whether alerts should be inhibited based on rules.

**Priority**: HIGH
**Dependencies**: TN-126 (Parser)
**Blocks**: TN-129, TN-130

---

## Functional Requirements

### FR-1: Alert Matching
- Match source and target alerts against inhibition rules
- Support exact label matching (source_match, target_match)
- Support regex label matching (source_match_re, target_match_re)
- Support equal label checking (labels must match between source and target)

### FR-2: Performance
- **Target**: <1ms per inhibition check (p99)
- Pre-compiled regex patterns from parser
- Efficient label matching algorithms

### FR-3: API
```go
ShouldInhibit(ctx, targetAlert) (*MatchResult, error)
FindInhibitors(ctx, targetAlert) ([]*MatchResult, error)
MatchRule(rule, sourceAlert, targetAlert) bool
```

---

## Non-Functional Requirements

### NFR-1: Performance
- <1ms inhibition check (p99)
- <100µs per rule evaluation
- Zero allocations in hot path

### NFR-2: Test Coverage
- 85%+ test coverage
- 40+ unit tests
- Edge cases covered

### NFR-3: Observability
- Metrics: match_duration, checks_total

---

## Acceptance Criteria

- [x] InhibitionMatcher interface defined
- [ ] DefaultInhibitionMatcher implemented
- [ ] Label matching (exact + regex)
- [ ] Equal labels check
- [ ] Performance <1ms (p99)
- [ ] 85%+ test coverage
- [ ] 40+ unit tests passing
- [ ] Benchmarks meet targets

---

**Date**: 2025-11-04
**Status**: READY FOR IMPLEMENTATION

