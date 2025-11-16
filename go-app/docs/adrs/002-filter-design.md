# ADR-002: Enhanced Filter System Design

**Status**: Accepted
**Date**: 2025-11-16
**Deciders**: Architecture Team
**Context**: TN-63 History Endpoint

---

## Context

The history endpoint needs to support 18+ filter types with complex combinations, while maintaining high performance and preventing SQL injection.

## Decision

We will implement a **Registry-based Filter System** with:
- **Filter Interface**: Common interface for all filters
- **Filter Registry**: Centralized filter creation and management
- **Query Builder**: Dynamic SQL query construction
- **Type Safety**: Strong typing for filter parameters

## Rationale

### Why Registry Pattern?

1. **Extensibility**: Easy to add new filter types
2. **Maintainability**: Centralized filter management
3. **Testability**: Each filter can be tested independently
4. **Type Safety**: Compile-time validation

### Why Query Builder?

1. **SQL Injection Prevention**: Parameterized queries
2. **Index Optimization**: Hints for query planner
3. **Dynamic Construction**: Build queries based on filters
4. **Performance**: Optimized for PostgreSQL

### Why 18+ Filter Types?

- **Status**: firing/resolved
- **Severity**: critical/warning/info/noise
- **Namespace**: exact match
- **Labels**: exact, regex, exists, not exists
- **Time Range**: from/to
- **Fingerprint**: exact match
- **Alert Name**: exact, pattern, regex
- **Duration**: min/max
- **Generator URL**: exact match
- **Flapping**: boolean
- **Resolved**: boolean
- **Search**: full-text

## Alternatives Considered

### Alternative 1: Single Filter Function
- ❌ Hard to extend
- ❌ Complex conditional logic
- ✅ Simpler initial implementation

### Alternative 2: SQL String Concatenation
- ❌ SQL injection risk
- ❌ Hard to optimize
- ✅ Simple implementation

### Alternative 3: ORM-Based Filtering
- ❌ Less control over SQL
- ❌ Performance overhead
- ✅ Type safety

## Consequences

### Positive
- ✅ 18+ filter types supported
- ✅ Easy to add new filters
- ✅ SQL injection prevented
- ✅ Query optimization hints

### Negative
- ⚠️ More code (18 filter files)
- ⚠️ Learning curve for new developers
- ⚠️ Filter validation complexity

### Mitigations
- Comprehensive documentation
- Code examples for each filter
- Unit tests for all filters
- Integration tests for combinations

## Implementation Details

- **Filter Interface**: `Type()`, `Validate()`, `ApplyToQuery()`, `CacheKey()`
- **Filter Registry**: `Create()`, `CreateFromQueryParams()`
- **Query Builder**: `AddWhere()`, `SetLimit()`, `SetOffset()`, `AddOrderBy()`
- **ReDoS Protection**: Regex timeout (5s)

## Security Considerations

- Parameterized queries (SQL injection prevention)
- Regex timeout (ReDoS prevention)
- Input validation (all parameters)
- Pattern detection (SQL/XSS)

---

**Approved by**: Architecture Team
**Date**: 2025-11-16
