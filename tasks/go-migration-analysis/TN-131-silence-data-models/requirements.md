# TN-131: Silence Data Models - Requirements

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-131
**Status**: 🟡 IN PROGRESS
**Priority**: HIGH
**Estimated Effort**: 8-12 hours
**Dependencies**: None (first task in Module 3)

---

## 📋 Overview

Реализовать data models для системы silencing (заглушек) алертов, совместимые с Alertmanager API v2. Silencing позволяет временно подавлять уведомления по алертам, соответствующим определенным label matchers.

### Business Value
- **Maintenance Windows**: Возможность заглушить алерты во время планового обслуживания
- **Noise Reduction**: Временное подавление известных проблем
- **Alertmanager Compatibility**: 100% совместимость с существующими клиентами
- **Audit Trail**: Полная история создания/изменения/удаления silences

---

## 🎯 Goals

### Primary Goals
1. ✅ Определить `Silence` data model с полной совместимостью Alertmanager API
2. ✅ Реализовать `Matcher` структуру для label matching (=, !=, =~, !~)
3. ✅ Создать PostgreSQL migration для `silences` таблицы
4. ✅ Реализовать validation logic для silences и matchers
5. ✅ Добавить custom error types для silence operations

### Secondary Goals
- Поддержка комментариев и metadata для audit trail
- UUID-based идентификаторы для silences
- Индексы для быстрого поиска по labels и status
- TTL management для автоматического удаления expired silences

---

## 📐 Functional Requirements

### FR-1: Silence Data Model

**Структура**:
```go
type Silence struct {
    ID          string          // UUID v4
    CreatedBy   string          // Email или username создателя
    Comment     string          // Обязательный комментарий (мин 3 символа)
    StartsAt    time.Time       // Начало действия silence
    EndsAt      time.Time       // Окончание действия silence
    Matchers    []Matcher       // Label matchers для matching алертов
    Status      SilenceStatus   // active, pending, expired
    CreatedAt   time.Time       // Timestamp создания
    UpdatedAt   *time.Time      // Timestamp последнего обновления
}

type SilenceStatus string
const (
    SilenceStatusPending SilenceStatus = "pending" // StartsAt > now
    SilenceStatusActive  SilenceStatus = "active"  // StartsAt <= now < EndsAt
    SilenceStatusExpired SilenceStatus = "expired" // EndsAt <= now
)
```

**Validation Rules**:
- `ID`: Must be valid UUID v4
- `CreatedBy`: Required, non-empty string (max 255 chars)
- `Comment`: Required, min 3 characters, max 1024 characters
- `StartsAt`: Required, must be valid timestamp
- `EndsAt`: Required, must be > StartsAt
- `Matchers`: Required, min 1 matcher, max 100 matchers
- `Status`: Auto-calculated based on StartsAt/EndsAt and current time

### FR-2: Matcher Data Model

**Структура**:
```go
type Matcher struct {
    Name    string       // Label name (required)
    Value   string       // Label value (required)
    Type    MatcherType  // Matching type
    IsRegex bool         // True for regex matchers
}

type MatcherType string
const (
    MatcherTypeEqual    MatcherType = "="   // Exact match
    MatcherTypeNotEqual MatcherType = "!="  // Not equal
    MatcherTypeRegex    MatcherType = "=~"  // Regex match
    MatcherTypeNotRegex MatcherType = "!~"  // Negated regex match
)
```

**Validation Rules**:
- `Name`: Required, valid Prometheus label name ([a-zA-Z_][a-zA-Z0-9_]*)
- `Value`: Required, non-empty string (max 1024 chars)
- `Type`: Must be one of =, !=, =~, !~
- `IsRegex`: Auto-set based on Type (true for =~ and !~)
- For regex matchers: Value must be valid regex pattern

### FR-3: PostgreSQL Schema

**Table: `silences`**:
```sql
CREATE TABLE silences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_by VARCHAR(255) NOT NULL,
    comment TEXT NOT NULL CHECK (length(comment) >= 3),
    starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
    ends_at TIMESTAMP WITH TIME ZONE NOT NULL CHECK (ends_at > starts_at),
    matchers JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT valid_status CHECK (status IN ('pending', 'active', 'expired'))
);

-- Indexes for fast queries
CREATE INDEX idx_silences_status ON silences(status);
CREATE INDEX idx_silences_starts_at ON silences(starts_at);
CREATE INDEX idx_silences_ends_at ON silences(ends_at);
CREATE INDEX idx_silences_created_by ON silences(created_by);
CREATE INDEX idx_silences_matchers ON silences USING GIN (matchers);
CREATE INDEX idx_silences_created_at ON silences(created_at DESC);
```

**JSON Storage for Matchers**:
```json
[
  {"name": "alertname", "value": "HighCPU", "type": "=", "isRegex": false},
  {"name": "severity", "value": "(critical|warning)", "type": "=~", "isRegex": true}
]
```

---

## 🔧 Technical Requirements

### TR-1: Alertmanager API Compatibility

**100% совместимость** с Alertmanager API v2:
- GET /api/v2/silences - List silences
- POST /api/v2/silences - Create silence
- GET /api/v2/silence/{id} - Get silence by ID
- DELETE /api/v2/silence/{id} - Delete silence

**Response Format** должен соответствовать Alertmanager:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "status": {
    "state": "active"
  },
  "createdBy": "ops@example.com",
  "comment": "Maintenance window for DB upgrade",
  "startsAt": "2025-11-04T10:00:00Z",
  "endsAt": "2025-11-04T12:00:00Z",
  "matchers": [
    {"name": "job", "value": "database", "isRegex": false, "isEqual": true}
  ],
  "createdAt": "2025-11-04T09:30:00Z",
  "updatedAt": "2025-11-04T09:30:00Z"
}
```

### TR-2: Performance Targets

| Operation | Target | Notes |
|-----------|--------|-------|
| Silence validation | <1ms | In-memory validation |
| Matcher validation | <500µs | Regex compilation cached |
| DB insert | <10ms | Single silence insert |
| DB query (by ID) | <5ms | Indexed lookup |
| DB query (list active) | <50ms | Up to 1000 silences |

### TR-3: Error Handling

Custom error types:
```go
var (
    ErrSilenceInvalidID       = errors.New("invalid silence ID")
    ErrSilenceInvalidComment  = errors.New("comment must be at least 3 characters")
    ErrSilenceInvalidTimeRange = errors.New("endsAt must be after startsAt")
    ErrSilenceNoMatchers      = errors.New("at least one matcher is required")
    ErrMatcherInvalidName     = errors.New("invalid label name")
    ErrMatcherInvalidRegex    = errors.New("invalid regex pattern")
    ErrMatcherInvalidType     = errors.New("invalid matcher type")
)
```

### TR-4: Testing Requirements

- **Unit Tests**: 30+ tests, 85%+ coverage
- **Test Coverage**:
  - Silence validation (valid/invalid cases)
  - Matcher validation (all 4 types)
  - JSON marshaling/unmarshaling
  - Status calculation (pending/active/expired)
  - PostgreSQL migration (up/down)
  - Error handling
- **Benchmarks**: <1ms validation

---

## 🔒 Security Requirements

### SEC-1: Input Validation
- Sanitize all user inputs (createdBy, comment)
- Validate regex patterns to prevent ReDoS attacks
- Limit matcher count to prevent DoS (max 100 matchers)
- Limit comment size (max 1024 chars)

### SEC-2: Audit Trail
- Record `created_by` for all silences
- Track `created_at` and `updated_at` timestamps
- Support filtering by creator for audit purposes

---

## 📊 Success Criteria

### Must Have
- ✅ `Silence` and `Matcher` structs defined
- ✅ PostgreSQL migration created and tested
- ✅ Validation logic implemented with error handling
- ✅ 30+ unit tests with 85%+ coverage
- ✅ JSON marshaling compatible with Alertmanager API
- ✅ Performance targets met (<1ms validation)

### Should Have
- Comprehensive godoc documentation
- Validation benchmarks
- Migration rollback tested
- Example YAML/JSON samples

### Could Have
- Support for bulk operations
- Advanced audit features (change history)
- Metrics for silence operations

---

## 🔗 Dependencies

### Internal Dependencies
- `internal/core` - Core domain types (if needed)
- `internal/infrastructure/migrations` - Migration framework

### External Dependencies
- `github.com/google/uuid` - UUID generation
- `github.com/lib/pq` - PostgreSQL driver
- PostgreSQL 12+ - Database

---

## 📚 References

- [Alertmanager API v2](https://github.com/prometheus/alertmanager/blob/main/api/v2/openapi.yaml)
- [Alertmanager Silencing](https://prometheus.io/docs/alerting/latest/alertmanager/#silences)
- [Prometheus Label Matchers](https://prometheus.io/docs/prometheus/latest/querying/basics/#time-series-selectors)

---

**Created**: 2025-11-04
**Author**: Alertmanager++ Team
**Last Updated**: 2025-11-04



