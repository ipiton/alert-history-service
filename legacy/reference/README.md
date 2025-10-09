# Reference Python Implementations

> **📦 Reference Only - Not for Production Use**

## Purpose

This directory contains Python implementations of complex algorithms and business logic that serve as **reference material** during Go development. While not actively maintained, these files are preserved for their architectural and algorithmic value.

## Status: Read-Only Archive

**These files are**:
- ✅ Preserved for documentation purposes
- ✅ Referenced by Go developers
- ✅ Kept for algorithm clarity
- ✅ Maintained in read-only state

**These files are NOT**:
- ❌ Production code
- ❌ Actively maintained
- ❌ Receiving bug fixes
- ❌ Being updated

## Use Cases

### 1. Algorithm Reference
When implementing complex logic in Go, developers can reference these files to understand the original Python implementation.

### 2. Business Logic Clarification
When business logic is unclear, these files provide the "source of truth" for how features were originally designed.

### 3. Testing Comparison
During migration, test results can be compared between Python and Go versions to ensure behavior parity.

### 4. Documentation
Technical documentation can reference these implementations for detailed explanations.

## Files in This Directory

### Complex Services

#### `alert_classifier.py` → `go-app/internal/infrastructure/llm/`
**Purpose**: LLM-based alert classification
**Complexity**: HIGH (LLM integration, retry logic, caching)
**Go Status**: ✅ Basic implementation complete, advanced features pending

**Why Preserved**:
- Complex LLM prompt engineering
- Advanced retry logic with exponential backoff
- Sophisticated caching strategy
- Error handling patterns

**Reference Use**: See Python implementation for:
- Prompt templates
- Classification confidence thresholds
- Fallback logic
- Response parsing

---

#### `filter_engine.py` → `go-app/internal/core/filtering.go`
**Purpose**: Alert filtering based on rules
**Complexity**: HIGH (complex rule evaluation)
**Go Status**: ✅ 95% complete, LLM-based filtering pending

**Why Preserved**:
- Complex boolean logic evaluation
- Label matching algorithms
- Regex pattern optimization
- Filter precedence rules

**Reference Use**: See Python implementation for:
- Filter evaluation order
- Edge cases handling
- Performance optimizations
- Test cases

---

#### `webhook_processor.py` → `go-app/cmd/server/handlers/webhook.go`
**Purpose**: Complex webhook payload processing
**Complexity**: MEDIUM-HIGH (multi-format support)
**Go Status**: 🔄 Basic webhook complete, complex processing pending

**Why Preserved**:
- Multi-format detection (Alertmanager, Prometheus, custom)
- Payload transformation pipelines
- Validation rules
- Error recovery

**Reference Use**: See Python implementation for:
- Format detection heuristics
- Transformation logic
- Validation rules
- Error messages

---

#### `alert_formatter.py` → TBD (not yet in Go)
**Purpose**: Format alerts for different publishing targets
**Complexity**: MEDIUM (target-specific formatting)
**Go Status**: ❌ Not implemented yet (TN-51 to TN-55)

**Why Preserved**:
- Rootly incident format
- PagerDuty event format
- Slack block format
- Generic webhook templates

**Reference Use**: Critical for implementing TN-51 to TN-55

---

### Core Implementations

#### `database/sqlite_adapter.py` → `go-app/internal/database/sqlite.go`
**Purpose**: SQLite database adapter
**Complexity**: MEDIUM (SQL query generation)
**Go Status**: ✅ Complete

**Why Preserved**:
- Query optimization techniques
- Transaction handling patterns
- Error handling
- Migration logic

---

#### `database/postgresql_adapter.py` → `go-app/internal/database/postgres.go`
**Purpose**: PostgreSQL database adapter
**Complexity**: MEDIUM (connection pooling, transactions)
**Go Status**: ✅ Complete

**Why Preserved**:
- Connection pool configuration
- Query optimization
- Transaction patterns
- Bulk operations

---

#### `core/interfaces.py` → `go-app/internal/core/interfaces.go`
**Purpose**: Domain models and interfaces
**Complexity**: LOW-MEDIUM (data structures)
**Go Status**: ✅ Complete

**Why Preserved**:
- Original model definitions
- Validation logic
- Serialization patterns

---

## How to Use Reference Code

### For Go Developers

**When implementing a new feature**:

1. **Read the Python reference**:
   ```bash
   cat legacy/reference/alert_classifier.py
   ```

2. **Understand the algorithm**:
   - Identify core logic
   - Note edge cases
   - Review comments and docstrings

3. **Implement in Go**:
   - Maintain same behavior
   - Adapt to Go idioms
   - Improve where possible

4. **Cross-reference tests**:
   - Compare test cases
   - Ensure same coverage
   - Verify edge cases

---

### For Product/Business Team

**When clarifying requirements**:

1. Reference Python implementation shows **how** feature works
2. Check comments for business logic reasoning
3. Review test cases for expected behavior

---

### For Documentation Team

**When writing technical docs**:

1. Link to reference implementations
2. Extract algorithm explanations
3. Include code snippets with attribution

---

## Maintenance Policy

### What We DO

- ✅ Preserve files in current state
- ✅ Keep readable formatting
- ✅ Maintain directory structure
- ✅ Document references in Go code

### What We DON'T DO

- ❌ Fix bugs (use Go version)
- ❌ Add features (Go only)
- ❌ Update dependencies
- ❌ Run security patches
- ❌ Respond to issues

### Exception: Critical Clarifications

If Python code contains **critical business logic ambiguity**:
- ✅ Add clarifying comments
- ✅ Link to relevant documentation
- ✅ Note known limitations

**Process**: Open PR with `legacy-clarification` label

---

## Linking to Reference Code

### In Go Code

```go
// Algorithm based on Python reference implementation:
// See: legacy/reference/alert_classifier.py:classify_alert()
//
// Key differences from Python version:
// - Uses goroutines for concurrent classification
// - Caching implemented with go-redis instead of in-memory
func (c *Classifier) ClassifyAlert(ctx context.Context, alert Alert) (*Classification, error) {
    // Implementation...
}
```

### In Documentation

```markdown
## Alert Classification Algorithm

The classification algorithm is based on the original Python implementation
([reference](../legacy/reference/alert_classifier.py)).

Key concepts:
- LLM-based severity detection
- Confidence scoring
- Fallback to rule-based classification
```

---

## Comparison: Python vs Go

### When Python Reference is Better

Use Python reference when:
- 📖 Algorithm is complex and well-documented
- 🧪 Test coverage is comprehensive
- 📝 Business logic is clearly expressed
- 🔍 Comments explain "why" not just "what"

### When Go Implementation is Better

Use Go implementation when:
- ⚡ Performance matters
- 🔒 Type safety is critical
- 🎯 Go idioms are more appropriate
- 🚀 Feature has been enhanced

---

## Review Schedule

**Quarterly Review** (every 3 months):

1. Assess if reference is still needed
2. Check for broken external links
3. Verify Go implementation hasn't diverged significantly
4. Update comments if needed

**Next Review**: April 2025

---

## Moving Files Out of Reference

### To Active Code

**Never**. These files should not return to active codebase.

### To Deprecated

If Go implementation is complete and reference is no longer needed:

```bash
# Move to deprecated (3-month deletion countdown)
git mv legacy/reference/alert_classifier.py legacy/deprecated/
```

### To Permanent Archive

If historical value but rarely referenced:

```bash
# Move to docs archive
git mv legacy/reference/alert_classifier.py legacy/docs/archive/
```

---

## Questions?

**Can I run this code?**
- Technically yes, but not recommended
- No guarantees it still works
- Dependencies may be outdated

**Can I modify this code?**
- No, read-only
- Clarifying comments only (via PR)

**What if I find a bug?**
- Note it in comments
- Fix in Go version
- Don't patch Python

**How long will this be kept?**
- Indefinitely (low storage cost)
- Reviewed quarterly
- May be archived if unused

---

**Last Updated**: 2025-01-09
**Next Review**: 2025-04-01
**Status**: Active reference material
