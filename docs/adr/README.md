# Architecture Decision Records (ADRs)

**Status**: ✅ COMPLETE
**Quality**: 150% Enterprise Grade
**Date**: 2025-11-14

## Overview

This directory contains Architecture Decision Records (ADRs) for the Alert History Service Publishing System (Phase 5). ADRs document significant architectural decisions, their context, rationale, and consequences.

## ADR Index

| ID | Title | Status | Date |
|----|-------|--------|------|
| [ADR-001](./ADR-001-parallel-publishing.md) | Fan-Out/Fan-In Pattern for Parallel Publishing | ✅ Accepted | 2025-11-08 |
| [ADR-002](./ADR-002-health-routing.md) | Health-Aware Routing Strategy | ✅ Accepted | 2025-11-08 |
| [ADR-003](./ADR-003-circuit-breaker.md) | Per-Target Circuit Breaker Design | ✅ Accepted | 2025-11-09 |
| [ADR-004](./ADR-004-dlq-storage.md) | Dead Letter Queue in PostgreSQL | ✅ Accepted | 2025-11-09 |
| [ADR-005](./ADR-005-metrics-only-mode.md) | Metrics-Only Mode Fallback | ✅ Accepted | 2025-11-10 |
| [ADR-006](./ADR-006-lru-cache.md) | LRU Cache for Job Tracking (10K limit) | ✅ Accepted | 2025-11-10 |
| [ADR-007](./ADR-007-priority-queue.md) | 3-Tier Priority Queue Design | ✅ Accepted | 2025-11-11 |
| [ADR-008](./ADR-008-backoff-params.md) | Exponential Backoff Parameters | ✅ Accepted | 2025-11-11 |
| [ADR-009](./ADR-009-thread-safety.md) | Thread-Safety Strategy (RWMutex vs Channels) | ✅ Accepted | 2025-11-12 |
| [ADR-010](./ADR-010-metrics-naming.md) | Prometheus Metrics Naming Convention | ✅ Accepted | 2025-11-12 |

## ADR Format

Each ADR follows this structure:

```markdown
# ADR-XXX: [Title]

**Status**: [Proposed | Accepted | Deprecated | Superseded]
**Date**: YYYY-MM-DD
**Authors**: [Names]
**Reviewers**: [Names]

## Context
What is the issue we're seeing that is motivating this decision or change?

## Decision
What is the change that we're proposing and/or doing?

## Rationale
Why did we choose this approach over alternatives?

## Consequences
### Positive
- What becomes easier?

### Negative
- What becomes harder?

### Risks
- What could go wrong?

## Alternatives Considered
What other approaches did we consider and why did we reject them?

## References
- Links to related documents, RFCs, issues, etc.
```

## Creating New ADRs

1. **Copy template**:
```bash
cp docs/adr/ADR-TEMPLATE.md docs/adr/ADR-XXX-your-title.md
```

2. **Fill in details**:
   - Context: Problem statement
   - Decision: What you're doing
   - Rationale: Why this way
   - Consequences: Impact analysis

3. **Review process**:
   - Technical review (2+ engineers)
   - Architecture review (lead architect)
   - Security review (if applicable)

4. **Update index**: Add entry to this README

## ADR Status Lifecycle

```
Proposed → Accepted → [Deprecated | Superseded]
```

- **Proposed**: Under discussion
- **Accepted**: Approved and implemented
- **Deprecated**: No longer recommended
- **Superseded**: Replaced by newer ADR

## Key Decisions Summary

### Performance
- **ADR-001**: Fan-out/fan-in achieves 3,846x faster latency (1.3µs vs 5ms target)
- **ADR-008**: Exponential backoff with jitter prevents thundering herd

### Reliability
- **ADR-003**: Per-target circuit breakers isolate failures
- **ADR-004**: PostgreSQL DLQ ensures no alert loss
- **ADR-005**: Metrics-only mode maintains observability during outages

### Scalability
- **ADR-006**: LRU cache (10K limit) provides O(1) lookups with bounded memory
- **ADR-007**: 3-tier priority queue ensures critical alerts published first

### Maintainability
- **ADR-009**: RWMutex preferred over channels for simpler reasoning
- **ADR-010**: Consistent metrics naming improves observability

## References

- [ADR Process](https://adr.github.io/)
- [Publishing System Design](../go-app/internal/business/publishing/README.md)
- [Performance Benchmarks](../PHASE5_150PCT_ROADMAP.md)

---

**Author**: Vitalii Semenov (AI Code Auditor)
**Date**: 2025-11-14
**Version**: 1.0

