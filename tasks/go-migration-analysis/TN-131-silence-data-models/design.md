# TN-131: Silence Data Models - Design Document

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-131
**Version**: 1.0
**Last Updated**: 2025-11-04

---

## 🎯 Design Overview

Silencing system позволяет временно подавлять алерты, соответствующие определенным label matchers. Дизайн полностью совместим с Alertmanager API v2 для обеспечения drop-in replacement.

### Key Design Decisions

1. **PostgreSQL Storage**: Используем PostgreSQL (не Redis) для persistence, так как silences требуют ACID guarantees и audit trail
2. **JSONB for Matchers**: Храним matchers в JSONB для гибкости и поддержки GIN индексов
3. **Status Auto-Calculation**: Status (pending/active/expired) вычисляется динамически на основе StartsAt/EndsAt
4. **UUID Identifiers**: Используем UUID v4 для глобальной уникальности
5. **No Caching**: Silences не кешируются (низкая частота изменений, требуется consistency)

---

## 📐 Architecture

### Component Structure

```
go-app/internal/core/silencing/
├── models.go          # Silence, Matcher data models
├── errors.go          # Custom error types
├── validator.go       # Validation logic
└── models_test.go     # Unit tests

go-app/internal/infrastructure/migrations/
└── 020_create_silences_table.sql  # PostgreSQL migration
```

### Data Flow

```
┌─────────────────┐
│   API Request   │
│  (JSON/YAML)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Unmarshal     │
│ → Silence model │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Validate()    │
│  - Time range   │
│  - Matchers     │
│  - Comment      │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  PostgreSQL     │
│  INSERT/UPDATE  │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Response      │
│   (with UUID)   │
└─────────────────┘
```

---

## 🗂️ Data Models

### Silence Model

```go
package silencing

import (
	"time"
	"github.com/google/uuid"
)

// Silence represents a silence rule that suppresses alerts matching specific criteria.
// It is fully compatible with Alertmanager API v2 silences.
type Silence struct {
	// ID is the unique identifier for this silence (UUID v4).
	ID string `json:"id" db:"id"`

	// CreatedBy is the email or username of the user who created this silence.
	CreatedBy string `json:"createdBy" db:"created_by"`

	// Comment is a required description explaining why this silence was created.
	// Minimum 3 characters, maximum 1024 characters.
	Comment string `json:"comment" db:"comment"`

	// StartsAt is when the silence becomes active.
	StartsAt time.Time `json:"startsAt" db:"starts_at"`

	// EndsAt is when the silence expires.
	// Must be after StartsAt.
	EndsAt time.Time `json:"endsAt" db:"ends_at"`

	// Matchers defines the label matching criteria for alerts to be silenced.
	// At least one matcher is required, maximum 100 matchers allowed.
	Matchers []Matcher `json:"matchers" db:"matchers"`

	// Status represents the current state of the silence.
	// Auto-calculated based on StartsAt, EndsAt, and current time.
	Status SilenceStatus `json:"status" db:"status"`

	// CreatedAt is the timestamp when this silence was created.
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// UpdatedAt is the timestamp of the last update to this silence.
	// Nil if never updated.
	UpdatedAt *time.Time `json:"updatedAt,omitempty" db:"updated_at"`
}

// SilenceStatus represents the state of a silence.
type SilenceStatus string

const (
	// SilenceStatusPending indicates the silence has not yet become active (StartsAt > now).
	SilenceStatusPending SilenceStatus = "pending"

	// SilenceStatusActive indicates the silence is currently active (StartsAt <= now < EndsAt).
	SilenceStatusActive SilenceStatus = "active"

	// SilenceStatusExpired indicates the silence has expired (EndsAt <= now).
	SilenceStatusExpired SilenceStatus = "expired"
)
```

### Matcher Model

```go
// Matcher defines a label matching criterion for silences.
// Supports four types of matching: =, !=, =~, !~
type Matcher struct {
	// Name is the label name to match against.
	// Must be a valid Prometheus label name: [a-zA-Z_][a-zA-Z0-9_]*
	Name string `json:"name"`

	// Value is the value to match (or regex pattern for regex matchers).
	// Maximum 1024 characters.
	Value string `json:"value"`

	// Type is the matching operator.
	// One of: =, !=, =~, !~
	Type MatcherType `json:"type"`

	// IsRegex indicates whether this is a regex matcher (=~ or !~).
	// Auto-set based on Type.
	IsRegex bool `json:"isRegex"`
}

// MatcherType represents the type of label matching.
type MatcherType string

const (
	// MatcherTypeEqual matches if label value equals the specified value.
	MatcherTypeEqual MatcherType = "="

	// MatcherTypeNotEqual matches if label value does not equal the specified value.
	MatcherTypeNotEqual MatcherType = "!="

	// MatcherTypeRegex matches if label value matches the regex pattern.
	MatcherTypeRegex MatcherType = "=~"

	// MatcherTypeNotRegex matches if label value does not match the regex pattern.
	MatcherTypeNotRegex MatcherType = "!~"
)

// IsValid checks if the MatcherType is one of the valid types.
func (mt MatcherType) IsValid() bool {
	switch mt {
	case MatcherTypeEqual, MatcherTypeNotEqual, MatcherTypeRegex, MatcherTypeNotRegex:
		return true
	default:
		return false
	}
}
```

---

## 🔧 Validation Logic

### Silence Validation

```go
// Validate validates the Silence and returns an error if invalid.
func (s *Silence) Validate() error {
	// Validate ID (if set)
	if s.ID != "" {
		if _, err := uuid.Parse(s.ID); err != nil {
			return fmt.Errorf("%w: %s", ErrSilenceInvalidID, err)
		}
	}

	// Validate CreatedBy
	if s.CreatedBy == "" {
		return ErrSilenceInvalidCreatedBy
	}
	if len(s.CreatedBy) > 255 {
		return ErrSilenceInvalidCreatedBy
	}

	// Validate Comment
	if len(s.Comment) < 3 {
		return ErrSilenceInvalidComment
	}
	if len(s.Comment) > 1024 {
		return ErrSilenceInvalidComment
	}

	// Validate time range
	if s.EndsAt.Before(s.StartsAt) || s.EndsAt.Equal(s.StartsAt) {
		return ErrSilenceInvalidTimeRange
	}

	// Validate matchers
	if len(s.Matchers) == 0 {
		return ErrSilenceNoMatchers
	}
	if len(s.Matchers) > 100 {
		return ErrSilenceTooManyMatchers
	}

	for i, matcher := range s.Matchers {
		if err := matcher.Validate(); err != nil {
			return fmt.Errorf("matcher %d: %w", i, err)
		}
	}

	return nil
}

// CalculateStatus calculates the current status based on StartsAt and EndsAt.
func (s *Silence) CalculateStatus() SilenceStatus {
	now := time.Now()
	if now.Before(s.StartsAt) {
		return SilenceStatusPending
	}
	if now.Before(s.EndsAt) {
		return SilenceStatusActive
	}
	return SilenceStatusExpired
}
```

### Matcher Validation

```go
// Validate validates the Matcher and returns an error if invalid.
func (m *Matcher) Validate() error {
	// Validate Name (Prometheus label name format)
	if !isValidLabelName(m.Name) {
		return ErrMatcherInvalidName
	}

	// Validate Value
	if m.Value == "" {
		return ErrMatcherEmptyValue
	}
	if len(m.Value) > 1024 {
		return ErrMatcherValueTooLong
	}

	// Validate Type
	if !m.Type.IsValid() {
		return ErrMatcherInvalidType
	}

	// Set IsRegex based on Type
	m.IsRegex = (m.Type == MatcherTypeRegex || m.Type == MatcherTypeNotRegex)

	// Validate regex pattern if regex matcher
	if m.IsRegex {
		if _, err := regexp.Compile(m.Value); err != nil {
			return fmt.Errorf("%w: %s", ErrMatcherInvalidRegex, err)
		}
	}

	return nil
}

// isValidLabelName checks if a label name follows Prometheus naming conventions.
// Valid: [a-zA-Z_][a-zA-Z0-9_]*
func isValidLabelName(name string) bool {
	if name == "" {
		return false
	}

	// First character must be [a-zA-Z_]
	first := rune(name[0])
	if !((first >= 'a' && first <= 'z') || (first >= 'A' && first <= 'Z') || first == '_') {
		return false
	}

	// Subsequent characters must be [a-zA-Z0-9_]
	for _, r := range name[1:] {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}
```

---

## 🗄️ Database Schema

### Migration: `020_create_silences_table.sql`

```sql
-- +goose Up
-- Create silences table
CREATE TABLE IF NOT EXISTS silences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by VARCHAR(255) NOT NULL,
    comment TEXT NOT NULL CHECK (length(comment) >= 3 AND length(comment) <= 1024),
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL,
    matchers JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,

    -- Constraints
    CONSTRAINT silences_valid_time_range CHECK (ends_at > starts_at),
    CONSTRAINT silences_valid_status CHECK (status IN ('pending', 'active', 'expired'))
);

-- Indexes for fast queries
CREATE INDEX idx_silences_status ON silences(status) WHERE status != 'expired';
CREATE INDEX idx_silences_starts_at ON silences(starts_at);
CREATE INDEX idx_silences_ends_at ON silences(ends_at);
CREATE INDEX idx_silences_created_by ON silences(created_by);
CREATE INDEX idx_silences_matchers ON silences USING GIN (matchers);
CREATE INDEX idx_silences_created_at ON silences(created_at DESC);

-- Composite index for active silences (most common query)
CREATE INDEX idx_silences_active ON silences(status, ends_at) WHERE status IN ('pending', 'active');

-- +goose Down
DROP TABLE IF EXISTS silences;
```

### Index Strategy

| Index | Purpose | Query Pattern |
|-------|---------|---------------|
| `idx_silences_status` | Filter by status | WHERE status = 'active' |
| `idx_silences_active` | Active silences | WHERE status IN ('pending', 'active') |
| `idx_silences_matchers` | Search by labels | WHERE matchers @> '{"name":"job"}' |
| `idx_silences_created_at` | List recent | ORDER BY created_at DESC |
| `idx_silences_created_by` | Audit queries | WHERE created_by = 'user@example.com' |

---

## 🚀 Performance Considerations

### Validation Performance
- **Target**: <1ms for full silence validation
- **Optimization**:
  - Pre-compile common regex patterns
  - Early return on first error
  - Minimal allocations

### Database Performance
- **Target**: <10ms insert, <5ms lookup by ID
- **Optimization**:
  - UUID primary key for fast lookups
  - GIN index on JSONB matchers
  - Partial index on status (exclude expired)
  - Composite index for common queries

### Memory Usage
- **Estimate**: ~500 bytes per Silence struct
- **Max Active Silences**: 10,000 (5MB total)

---

## 🔒 Security Considerations

### Input Validation
1. **Regex DoS Prevention**: Limit regex complexity (max 1024 chars)
2. **Comment Sanitization**: Escape special characters for JSON/HTML output
3. **Matcher Count Limit**: Max 100 matchers to prevent DoS
4. **CreatedBy Validation**: Max 255 chars, sanitize email format

### Audit Trail
1. **Immutable History**: Track all creates/updates in `updated_at`
2. **Creator Attribution**: Always record `created_by`
3. **Timestamp Integrity**: Use server-side timestamps (NOW())

---

## 📊 Metrics (Future)

Prometheus metrics to track (TN-134):
```
silence_operations_total{operation="create|update|delete",status="success|error"}
silence_validation_duration_seconds{type="silence|matcher"}
silences_active_total{status="pending|active|expired"}
```

---

## 🧪 Testing Strategy

### Unit Tests (30+ tests)

**Silence Validation**:
- ✅ Valid silence with all fields
- ✅ Invalid ID (not UUID)
- ✅ Invalid CreatedBy (empty, too long)
- ✅ Invalid Comment (too short, too long)
- ✅ Invalid time range (EndsAt <= StartsAt)
- ✅ No matchers
- ✅ Too many matchers (>100)
- ✅ Status calculation (pending/active/expired)

**Matcher Validation**:
- ✅ Valid matcher (each type: =, !=, =~, !~)
- ✅ Invalid label name (starts with digit, contains special chars)
- ✅ Empty value
- ✅ Value too long (>1024)
- ✅ Invalid regex pattern
- ✅ IsRegex flag auto-set

**JSON Marshaling**:
- ✅ Marshal/Unmarshal round-trip
- ✅ Alertmanager API compatibility
- ✅ JSONB storage format

**Migration**:
- ✅ Migration up (table created)
- ✅ Migration down (table dropped)
- ✅ Constraints enforced

### Benchmarks
- `BenchmarkSilenceValidate`: Target <1ms
- `BenchmarkMatcherValidate`: Target <100µs
- `BenchmarkCalculateStatus`: Target <10µs

---

## 📚 API Compatibility

### Alertmanager v2 Silence Format

**Request (POST /api/v2/silences)**:
```json
{
  "matchers": [
    {"name": "alertname", "value": "HighCPU", "isRegex": false, "isEqual": true},
    {"name": "job", "value": "api-server", "isRegex": false, "isEqual": true}
  ],
  "startsAt": "2025-11-04T10:00:00Z",
  "endsAt": "2025-11-04T12:00:00Z",
  "createdBy": "ops@example.com",
  "comment": "Planned maintenance window for CPU upgrade"
}
```

**Response**:
```json
{
  "silenceID": "550e8400-e29b-41d4-a716-446655440000"
}
```

### Mapping: Alertmanager ↔ Our Model

| Alertmanager | Our Model | Notes |
|--------------|-----------|-------|
| `id` | `ID` | UUID v4 |
| `createdBy` | `CreatedBy` | Same |
| `comment` | `Comment` | Same |
| `startsAt` | `StartsAt` | RFC3339 format |
| `endsAt` | `EndsAt` | RFC3339 format |
| `matchers[].name` | `Matchers[].Name` | Same |
| `matchers[].value` | `Matchers[].Value` | Same |
| `matchers[].isRegex` | `Matchers[].IsRegex` | Same |
| `matchers[].isEqual` | Derived from `Type` | =: true, !=: false |
| `status.state` | `Status` | Same values |

---

## 🔗 Dependencies

### Internal
- None (first task in Module 3)

### External
- `github.com/google/uuid` v1.3+
- `github.com/lib/pq` v1.10+ (PostgreSQL driver)

---

## 🎯 Definition of Done

- ✅ `models.go` created with Silence and Matcher structs
- ✅ `errors.go` created with 8+ custom error types
- ✅ `validator.go` created with validation logic
- ✅ `020_create_silences_table.sql` migration created
- ✅ `models_test.go` with 30+ unit tests
- ✅ Test coverage ≥85%
- ✅ All tests passing
- ✅ Benchmarks meet performance targets
- ✅ Godoc documentation complete
- ✅ Code committed to git

---

**Designed**: 2025-11-04
**Approved**: 2025-11-04
**Implemented**: TBD

