# ADR-003: Pagination Strategy

**Status**: Accepted  
**Date**: 2025-11-16  
**Deciders**: Architecture Team  
**Context**: TN-63 History Endpoint

---

## Context

The history endpoint needs to support pagination for large result sets while maintaining performance and preventing expensive queries.

## Decision

We will implement **offset-based pagination** with:
- **Page-based**: Page number + items per page
- **Limits**: Max page 10,000, max per_page 1,000
- **Metadata**: Total count, total pages, has_next, has_prev
- **Performance**: Indexed queries, LIMIT/OFFSET

## Rationale

### Why Offset-Based?

1. **Simplicity**: Easy to understand and implement
2. **Compatibility**: Standard REST API pattern
3. **Metadata**: Total count available
4. **User-Friendly**: Page numbers are intuitive

### Why Not Cursor-Based?

- ❌ More complex implementation
- ❌ No total count
- ✅ Better performance for deep pagination
- ✅ No offset performance issues

**Decision**: Offset-based for simplicity, cursor-based can be added later if needed.

### Why Limits?

- **Max Page 10,000**: Prevents expensive deep pagination queries
- **Max Per-Page 1,000**: Prevents memory issues
- **Performance**: Indexed queries handle offset efficiently

## Alternatives Considered

### Alternative 1: Cursor-Based Pagination
- ✅ Better performance for deep pagination
- ❌ More complex implementation
- ❌ No total count
- ❌ Less user-friendly

### Alternative 2: Keyset Pagination
- ✅ Best performance
- ❌ Most complex
- ❌ Requires sorted unique key

### Alternative 3: No Pagination
- ❌ Memory issues with large results
- ❌ Poor performance
- ✅ Simplest implementation

## Consequences

### Positive
- ✅ Simple to use and understand
- ✅ Total count available
- ✅ Standard REST pattern
- ✅ Good performance for first 1000 pages

### Negative
- ⚠️ Performance degrades for deep pagination (> 1000 pages)
- ⚠️ Offset can be expensive for large datasets
- ⚠️ Total count query can be slow

### Mitigations
- Index optimization for sorting
- Limit max page to 10,000
- Cache total count
- Consider cursor-based for deep pagination (future)

## Implementation Details

- **Default**: Page 1, 50 items per page
- **Max Page**: 10,000 (configurable)
- **Max Per-Page**: 1,000 (configurable)
- **SQL**: `LIMIT $1 OFFSET $2`
- **Total Count**: Separate COUNT query (can be cached)

## Performance Considerations

- Indexed columns for sorting (starts_at, created_at)
- LIMIT prevents full table scans
- OFFSET performance acceptable for first 1000 pages
- Consider cursor-based for pages > 1000 (future enhancement)

---

**Approved by**: Architecture Team  
**Date**: 2025-11-16

