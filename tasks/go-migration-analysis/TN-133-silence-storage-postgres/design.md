# TN-133: Silence Storage (PostgreSQL Repository) - Design Document

**Module**: PHASE A - Module 3: Silencing System
**Task ID**: TN-133
**Version**: 1.0
**Last Updated**: 2025-11-05
**Status**: 🔄 IN PROGRESS

---

## 🎯 Design Overview

Silence Storage реализует **enterprise-grade PostgreSQL repository** для хранения silence rules с focus на:

1. **Performance**: Sub-10ms CRUD operations через optimized indexing
2. **Scalability**: Support 100K+ silences с automatic TTL cleanup
3. **Reliability**: ACID guarantees, optimistic locking, transaction support
4. **Observability**: 6 Prometheus metrics для real-time monitoring
5. **Maintainability**: Clean architecture, comprehensive testing, godoc

### Key Design Decisions

| Decision | Rationale | Trade-offs |
|----------|-----------|------------|
| **PostgreSQL (не Redis)** | ACID guarantees, audit trail, complex querying | Slightly slower than in-memory |
| **JSONB для matchers** | Flexible schema, GIN indexing, fast label searches | ~10% storage overhead |
| **Optimistic Locking** | High concurrency, no deadlocks | Requires retry logic |
| **Batch TTL Cleanup** | Minimize database load, efficient deletes | Max 1000 per run |
| **Connection Pooling** | Reuse connections, reduce overhead | Max 25 connections |
| **Structured Logging** | Machine-readable, easy debugging | Slight perf overhead |

---

## 📐 Architecture

### Component Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Layer                         │
│  (TN-134: SilenceManager, TN-135: API Endpoints)                │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ Uses
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                   SilenceRepository Interface                    │
│  (9 methods: Create, Get, List, Update, Delete, Count, etc.)   │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ Implements
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│             PostgresSilenceRepository (TN-133)                   │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Core Operations:                                       │   │
│  │  - CreateSilence()     : INSERT + validation           │   │
│  │  - GetSilenceByID()    : SELECT by UUID                │   │
│  │  - ListSilences()      : SELECT + filters + pagination │   │
│  │  - UpdateSilence()     : UPDATE + optimistic lock      │   │
│  │  - DeleteSilence()     : DELETE by ID                  │   │
│  │  - CountSilences()     : COUNT(*) with filters         │   │
│  │  - ExpireSilences()    : Batch UPDATE/DELETE           │   │
│  │  - GetExpiringSoon()   : SELECT + time window          │   │
│  │  - BulkUpdateStatus()  : Batch UPDATE in transaction   │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Supporting Components:                                 │   │
│  │  - SilenceMetrics      : 6 Prometheus metrics           │   │
│  │  - TTLCleanupWorker    : Background cleanup             │   │
│  │  - FilterBuilder       : Dynamic SQL generation         │   │
│  │  - ResultScanner       : Row scanning + JSONB parsing   │   │
│  └─────────────────────────────────────────────────────────┘   │
└────────────────┬────────────────────────────────────────────────┘
                 │
                 │ Uses
                 ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                           │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Table: silences                                        │   │
│  │  - id (UUID PRIMARY KEY)                                │   │
│  │  - created_by (VARCHAR(255))                            │   │
│  │  - comment (TEXT)                                       │   │
│  │  - starts_at, ends_at (TIMESTAMPTZ)                     │   │
│  │  - matchers (JSONB) ← GIN index                         │   │
│  │  - status (VARCHAR(20))                                 │   │
│  │  - created_at, updated_at (TIMESTAMPTZ)                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │  Indexes (7):                                           │   │
│  │  1. idx_silences_status (partial, exclude expired)      │   │
│  │  2. idx_silences_active (composite: status, ends_at)    │   │
│  │  3. idx_silences_starts_at (btree)                      │   │
│  │  4. idx_silences_ends_at (btree)                        │   │
│  │  5. idx_silences_created_by (btree)                     │   │
│  │  6. idx_silences_matchers (GIN, JSONB)                  │   │
│  │  7. idx_silences_created_at (btree DESC)                │   │
│  └─────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Data Flow

**1. Create Silence Flow**:
```
┌──────────┐    Validate()    ┌──────────────┐    INSERT    ┌────────────┐
│ Client   │ ──────────────▶  │ Repository   │ ──────────▶  │ PostgreSQL │
│ (API)    │                  │              │              │            │
└──────────┘                  └──────────────┘              └────────────┘
                                   │
                                   │ Generate UUID
                                   │ Calculate status
                                   │ Marshal matchers to JSONB
                                   │ Record metrics
                                   │
                                   ▼
                              Return Silence
```

**2. List Silences Flow (with filters)**:
```
┌──────────┐    SilenceFilter    ┌──────────────┐    Build SQL    ┌────────────┐
│ Client   │ ─────────────────▶  │ FilterBuilder│ ──────────────▶  │ PostgreSQL │
│          │                     │              │                  │            │
└──────────┘                     └──────────────┘                  └────────────┘
                                      │                                 │
                                      │ WHERE status = ANY($1)          │
                                      │ AND created_by = $2             │
                                      │ AND matchers @> $3::jsonb       │
                                      │ ORDER BY created_at DESC        │
                                      │ LIMIT 100 OFFSET 0              │
                                      │                                 │
                                      ▼                                 ▼
                                 Execute Query ◀──────────────── Return Rows
                                      │
                                      │ Scan rows
                                      │ Unmarshal JSONB
                                      │ Map to []Silence
                                      │
                                      ▼
                                 Return Results
```

**3. TTL Cleanup Flow**:
```
┌──────────────────┐    Every 1h    ┌──────────────┐    Batch DELETE    ┌────────────┐
│ TTLCleanupWorker │ ─────────────▶  │ Repository   │ ─────────────────▶ │ PostgreSQL │
│ (Background)     │                 │              │                    │            │
└──────────────────┘                 └──────────────┘                    └────────────┘
         │                                 │                                   │
         │ Ticker fires                    │ SELECT expired silences          │
         │                                 │ WHERE ends_at < NOW() - 24h      │
         │                                 │ LIMIT 1000                        │
         │                                 │                                   │
         │                                 ▼                                   ▼
         │                            Execute DELETE                     Return count
         │                                 │
         │                                 │ Record metrics
         │                                 │ Log deleted count
         │                                 │
         └─────────────────────────────────┘
```

---

## 🗂️ Package Structure

```
go-app/internal/infrastructure/silencing/
├── repository.go                          # SilenceRepository interface (150 LOC)
├── postgres_silence_repository.go         # Implementation (600 LOC)
├── postgres_silence_repository_test.go    # Unit tests (800 LOC)
├── postgres_silence_repository_integration_test.go  # Integration tests (400 LOC)
├── postgres_silence_repository_bench_test.go        # Benchmarks (200 LOC)
├── silence_repository_errors.go           # Custom errors (60 LOC)
├── filter_builder.go                      # Dynamic SQL builder (250 LOC)
├── filter_builder_test.go                 # Filter tests (200 LOC)
├── ttl_cleanup_worker.go                  # Background cleanup (150 LOC)
├── ttl_cleanup_worker_test.go             # Worker tests (150 LOC)
├── metrics.go                             # Prometheus metrics (150 LOC)
└── README.md                              # Usage guide (600 LOC)

Total: ~3,700 LOC (production + tests + docs)
```

---

## 🔧 Implementation Details

### 1. SilenceRepository Interface

**File**: `repository.go`

```go
package silencing

import (
    "context"
    "time"

    "github.com/vitaliisemenov/alert-history/internal/core/silencing"
)

// SilenceRepository provides persistence operations for silence rules.
// All methods are safe for concurrent use and support context cancellation.
//
// Thread-safety: All methods are safe for concurrent calls.
// Context handling: All methods respect ctx.Done() for cancellation.
// Error handling: Returns wrapped errors with context.
type SilenceRepository interface {
    // CreateSilence creates a new silence in the database.
    // Generates a new UUID if silence.ID is empty.
    // Validates the silence before insertion.
    // Returns the created silence with ID, CreatedAt populated.
    //
    // Errors:
    //   - ErrSilenceExists if a silence with the same ID already exists
    //   - ErrValidation if silence.Validate() fails
    //   - ErrDatabase for database errors
    CreateSilence(ctx context.Context, silence *silencing.Silence) (*silencing.Silence, error)

    // GetSilenceByID retrieves a single silence by its UUID.
    //
    // Errors:
    //   - ErrSilenceNotFound if no silence with the given ID exists
    //   - ErrInvalidUUID if id is not a valid UUID
    //   - ErrDatabase for database errors
    GetSilenceByID(ctx context.Context, id string) (*silencing.Silence, error)

    // ListSilences retrieves silences matching the provided filter.
    // Returns an empty slice if no silences match.
    // Results are paginated according to filter.Limit and filter.Offset.
    //
    // Errors:
    //   - ErrInvalidFilter if filter parameters are invalid
    //   - ErrDatabase for database errors
    ListSilences(ctx context.Context, filter SilenceFilter) ([]*silencing.Silence, error)

    // UpdateSilence updates an existing silence.
    // Uses optimistic locking: compares silence.UpdatedAt before update.
    // Sets UpdatedAt to NOW() on successful update.
    //
    // Errors:
    //   - ErrSilenceNotFound if the silence does not exist
    //   - ErrSilenceConflict if optimistic locking fails (concurrent update)
    //   - ErrValidation if silence.Validate() fails
    //   - ErrDatabase for database errors
    UpdateSilence(ctx context.Context, silence *silencing.Silence) error

    // DeleteSilence deletes a silence by ID.
    // This is a hard delete (permanent removal).
    //
    // Errors:
    //   - ErrSilenceNotFound if the silence does not exist
    //   - ErrDatabase for database errors
    DeleteSilence(ctx context.Context, id string) error

    // CountSilences returns the total number of silences matching the filter.
    // Useful for pagination (total pages = count / limit).
    //
    // Errors:
    //   - ErrInvalidFilter if filter parameters are invalid
    //   - ErrDatabase for database errors
    CountSilences(ctx context.Context, filter SilenceFilter) (int64, error)

    // ExpireSilences marks silences with EndsAt < before as expired.
    // If deleteExpired is true, also deletes them from the database.
    // Returns the number of silences affected.
    //
    // Batch limit: processes max 1000 silences per call.
    //
    // Errors:
    //   - ErrDatabase for database errors
    ExpireSilences(ctx context.Context, before time.Time, deleteExpired bool) (int64, error)

    // GetExpiringSoon returns silences expiring within the specified window.
    // Example: GetExpiringSoon(ctx, 1*time.Hour) returns silences expiring in next hour.
    //
    // Errors:
    //   - ErrDatabase for database errors
    GetExpiringSoon(ctx context.Context, window time.Duration) ([]*silencing.Silence, error)

    // BulkUpdateStatus updates the status of multiple silences atomically.
    // Uses a transaction to ensure all-or-nothing semantics.
    //
    // Errors:
    //   - ErrTransactionFailed if the transaction fails
    //   - ErrDatabase for database errors
    BulkUpdateStatus(ctx context.Context, ids []string, status silencing.SilenceStatus) error
}

// SilenceFilter defines filtering and pagination options for ListSilences.
type SilenceFilter struct {
    // Statuses filters by one or more status values (pending, active, expired).
    // Empty slice matches all statuses.
    Statuses []silencing.SilenceStatus

    // CreatedBy filters by creator email/username (exact match).
    // Empty string matches all creators.
    CreatedBy string

    // MatcherName searches for silences with this matcher name in JSONB.
    // Uses JSONB containment operator: matchers @> '[{"name":"..."}]'
    // Empty string skips this filter.
    MatcherName string

    // MatcherValue searches for silences with this matcher value in JSONB.
    // Uses JSONB containment operator: matchers @> '[{"value":"..."}]'
    // Empty string skips this filter.
    MatcherValue string

    // Time range filters
    StartsAfter  *time.Time // Filter: starts_at >= value
    StartsBefore *time.Time // Filter: starts_at <= value
    EndsAfter    *time.Time // Filter: ends_at >= value
    EndsBefore   *time.Time // Filter: ends_at <= value

    // Pagination
    Limit  int // Max results (default: 100, max: 1000)
    Offset int // Skip N results (default: 0)

    // Sorting
    OrderBy   string // Field: created_at|starts_at|ends_at|updated_at (default: created_at)
    OrderDesc bool   // Sort descending (default: true - newest first)
}

// Validate validates the filter parameters.
func (f *SilenceFilter) Validate() error {
    if f.Limit < 0 {
        return fmt.Errorf("%w: limit must be >= 0", ErrInvalidFilter)
    }
    if f.Limit > 1000 {
        return fmt.Errorf("%w: limit must be <= 1000", ErrInvalidFilter)
    }
    if f.Offset < 0 {
        return fmt.Errorf("%w: offset must be >= 0", ErrInvalidFilter)
    }

    validOrderBy := map[string]bool{
        "created_at": true,
        "starts_at":  true,
        "ends_at":    true,
        "updated_at": true,
    }
    if f.OrderBy != "" && !validOrderBy[f.OrderBy] {
        return fmt.Errorf("%w: invalid order_by field: %s", ErrInvalidFilter, f.OrderBy)
    }

    return nil
}

// ApplyDefaults sets default values for empty fields.
func (f *SilenceFilter) ApplyDefaults() {
    if f.Limit == 0 {
        f.Limit = 100
    }
    if f.OrderBy == "" {
        f.OrderBy = "created_at"
    }
}
```

---

### 2. PostgresSilenceRepository Implementation

**File**: `postgres_silence_repository.go`

```go
package silencing

import (
    "context"
    "encoding/json"
    "fmt"
    "log/slog"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    "github.com/vitaliisemenov/alert-history/internal/core/silencing"
)

// PostgresSilenceRepository implements SilenceRepository for PostgreSQL.
type PostgresSilenceRepository struct {
    pool    *pgxpool.Pool
    logger  *slog.Logger
    metrics *SilenceMetrics
}

// NewPostgresSilenceRepository creates a new PostgreSQL silence repository.
func NewPostgresSilenceRepository(pool *pgxpool.Pool, logger *slog.Logger) *PostgresSilenceRepository {
    if logger == nil {
        logger = slog.Default()
    }

    return &PostgresSilenceRepository{
        pool:    pool,
        logger:  logger,
        metrics: NewSilenceMetrics(),
    }
}

// CreateSilence implements SilenceRepository.CreateSilence
func (r *PostgresSilenceRepository) CreateSilence(ctx context.Context, silence *silencing.Silence) (*silencing.Silence, error) {
    start := time.Now()
    operation := "create"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Validate silence before insert
    if err := silence.Validate(); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "validation").Inc()
        return nil, fmt.Errorf("validation failed: %w", err)
    }

    // Generate UUID if not set
    if silence.ID == "" {
        silence.ID = uuid.New().String()
    }

    // Validate UUID format
    if _, err := uuid.Parse(silence.ID); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "invalid_uuid").Inc()
        return nil, fmt.Errorf("%w: %s", ErrInvalidUUID, err)
    }

    // Calculate initial status
    silence.Status = silence.CalculateStatus()

    // Marshal matchers to JSONB
    matchersJSON, err := json.Marshal(silence.Matchers)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "marshal").Inc()
        return nil, fmt.Errorf("marshal matchers: %w", err)
    }

    // Insert silence
    query := `
        INSERT INTO silences (id, created_by, comment, starts_at, ends_at, matchers, status, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
        RETURNING created_at
    `

    var createdAt time.Time
    err = r.pool.QueryRow(ctx, query,
        silence.ID,
        silence.CreatedBy,
        silence.Comment,
        silence.StartsAt,
        silence.EndsAt,
        matchersJSON,
        silence.Status,
    ).Scan(&createdAt)

    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "insert").Inc()

        // Check for duplicate key error
        if pgErr, ok := err.(*pgconn.PgError); ok && pgErr.Code == "23505" {
            return nil, fmt.Errorf("%w: silence with ID %s already exists", ErrSilenceExists, silence.ID)
        }

        return nil, fmt.Errorf("insert silence: %w", err)
    }

    silence.CreatedAt = createdAt
    r.metrics.Operations.WithLabelValues(operation, "success").Inc()
    r.metrics.ActiveSilences.WithLabelValues(string(silence.Status)).Inc()

    r.logger.Info("silence created",
        "silence_id", silence.ID,
        "created_by", silence.CreatedBy,
        "starts_at", silence.StartsAt.Format(time.RFC3339),
        "ends_at", silence.EndsAt.Format(time.RFC3339),
        "status", silence.Status,
    )

    return silence, nil
}

// GetSilenceByID implements SilenceRepository.GetSilenceByID
func (r *PostgresSilenceRepository) GetSilenceByID(ctx context.Context, id string) (*silencing.Silence, error) {
    start := time.Now()
    operation := "get_by_id"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Validate UUID format
    if _, err := uuid.Parse(id); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "invalid_uuid").Inc()
        return nil, fmt.Errorf("%w: %s", ErrInvalidUUID, err)
    }

    query := `
        SELECT id, created_by, comment, starts_at, ends_at, matchers, status, created_at, updated_at
        FROM silences
        WHERE id = $1
    `

    var silence silencing.Silence
    var matchersJSON []byte
    var updatedAt *time.Time

    err := r.pool.QueryRow(ctx, query, id).Scan(
        &silence.ID,
        &silence.CreatedBy,
        &silence.Comment,
        &silence.StartsAt,
        &silence.EndsAt,
        &matchersJSON,
        &silence.Status,
        &silence.CreatedAt,
        &updatedAt,
    )

    if err != nil {
        if err == pgx.ErrNoRows {
            r.metrics.Errors.WithLabelValues(operation, "not_found").Inc()
            return nil, fmt.Errorf("%w: silence with ID %s", ErrSilenceNotFound, id)
        }
        r.metrics.Errors.WithLabelValues(operation, "query").Inc()
        return nil, fmt.Errorf("query silence: %w", err)
    }

    // Unmarshal JSONB matchers
    if err := json.Unmarshal(matchersJSON, &silence.Matchers); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "unmarshal").Inc()
        return nil, fmt.Errorf("unmarshal matchers: %w", err)
    }

    silence.UpdatedAt = updatedAt
    r.metrics.Operations.WithLabelValues(operation, "success").Inc()

    return &silence, nil
}

// ListSilences implements SilenceRepository.ListSilences
func (r *PostgresSilenceRepository) ListSilences(ctx context.Context, filter SilenceFilter) ([]*silencing.Silence, error) {
    start := time.Now()
    operation := "list"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Validate and apply defaults
    filter.ApplyDefaults()
    if err := filter.Validate(); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "validation").Inc()
        return nil, err
    }

    // Build query dynamically
    query, args := r.buildListQuery(filter)

    rows, err := r.pool.Query(ctx, query, args...)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "query").Inc()
        return nil, fmt.Errorf("query silences: %w", err)
    }
    defer rows.Close()

    // Scan results
    silences := []*silencing.Silence{}
    for rows.Next() {
        var silence silencing.Silence
        var matchersJSON []byte
        var updatedAt *time.Time

        err := rows.Scan(
            &silence.ID,
            &silence.CreatedBy,
            &silence.Comment,
            &silence.StartsAt,
            &silence.EndsAt,
            &matchersJSON,
            &silence.Status,
            &silence.CreatedAt,
            &updatedAt,
        )
        if err != nil {
            r.metrics.Errors.WithLabelValues(operation, "scan").Inc()
            return nil, fmt.Errorf("scan silence: %w", err)
        }

        // Unmarshal JSONB matchers
        if err := json.Unmarshal(matchersJSON, &silence.Matchers); err != nil {
            r.metrics.Errors.WithLabelValues(operation, "unmarshal").Inc()
            return nil, fmt.Errorf("unmarshal matchers: %w", err)
        }

        silence.UpdatedAt = updatedAt
        silences = append(silences, &silence)
    }

    if err := rows.Err(); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "rows").Inc()
        return nil, fmt.Errorf("iterate rows: %w", err)
    }

    r.metrics.Operations.WithLabelValues(operation, "success").Inc()

    r.logger.Debug("silences listed",
        "count", len(silences),
        "filter_statuses", filter.Statuses,
        "filter_created_by", filter.CreatedBy,
    )

    return silences, nil
}

// buildListQuery builds a dynamic SQL query based on the filter.
func (r *PostgresSilenceRepository) buildListQuery(filter SilenceFilter) (string, []interface{}) {
    query := `
        SELECT id, created_by, comment, starts_at, ends_at, matchers, status, created_at, updated_at
        FROM silences
        WHERE 1=1
    `

    args := []interface{}{}
    argIdx := 1

    // Filter by statuses
    if len(filter.Statuses) > 0 {
        query += fmt.Sprintf(" AND status = ANY($%d)", argIdx)
        statusStrings := make([]string, len(filter.Statuses))
        for i, status := range filter.Statuses {
            statusStrings[i] = string(status)
        }
        args = append(args, statusStrings)
        argIdx++
    }

    // Filter by creator
    if filter.CreatedBy != "" {
        query += fmt.Sprintf(" AND created_by = $%d", argIdx)
        args = append(args, filter.CreatedBy)
        argIdx++
    }

    // JSONB filters
    if filter.MatcherName != "" {
        query += fmt.Sprintf(" AND matchers @> $%d::jsonb", argIdx)
        args = append(args, fmt.Sprintf(`[{"name":"%s"}]`, filter.MatcherName))
        argIdx++
    }

    if filter.MatcherValue != "" {
        query += fmt.Sprintf(" AND matchers @> $%d::jsonb", argIdx)
        args = append(args, fmt.Sprintf(`[{"value":"%s"}]`, filter.MatcherValue))
        argIdx++
    }

    // Time range filters
    if filter.StartsAfter != nil {
        query += fmt.Sprintf(" AND starts_at >= $%d", argIdx)
        args = append(args, *filter.StartsAfter)
        argIdx++
    }

    if filter.StartsBefore != nil {
        query += fmt.Sprintf(" AND starts_at <= $%d", argIdx)
        args = append(args, *filter.StartsBefore)
        argIdx++
    }

    if filter.EndsAfter != nil {
        query += fmt.Sprintf(" AND ends_at >= $%d", argIdx)
        args = append(args, *filter.EndsAfter)
        argIdx++
    }

    if filter.EndsBefore != nil {
        query += fmt.Sprintf(" AND ends_at <= $%d", argIdx)
        args = append(args, *filter.EndsBefore)
        argIdx++
    }

    // Add ORDER BY
    direction := "DESC"
    if !filter.OrderDesc {
        direction = "ASC"
    }
    query += fmt.Sprintf(" ORDER BY %s %s", filter.OrderBy, direction)

    // Add LIMIT/OFFSET
    query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
    args = append(args, filter.Limit, filter.Offset)

    return query, args
}

// UpdateSilence implements SilenceRepository.UpdateSilence
func (r *PostgresSilenceRepository) UpdateSilence(ctx context.Context, silence *silencing.Silence) error {
    start := time.Now()
    operation := "update"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Validate silence
    if err := silence.Validate(); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "validation").Inc()
        return fmt.Errorf("validation failed: %w", err)
    }

    // Calculate current status
    silence.Status = silence.CalculateStatus()

    // Marshal matchers to JSONB
    matchersJSON, err := json.Marshal(silence.Matchers)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "marshal").Inc()
        return fmt.Errorf("marshal matchers: %w", err)
    }

    // Optimistic locking: check updated_at
    query := `
        UPDATE silences
        SET created_by = $1,
            comment = $2,
            starts_at = $3,
            ends_at = $4,
            matchers = $5,
            status = $6,
            updated_at = NOW()
        WHERE id = $7
          AND (updated_at IS NULL OR updated_at = $8)
        RETURNING updated_at
    `

    var updatedAt time.Time
    err = r.pool.QueryRow(ctx, query,
        silence.CreatedBy,
        silence.Comment,
        silence.StartsAt,
        silence.EndsAt,
        matchersJSON,
        silence.Status,
        silence.ID,
        silence.UpdatedAt,
    ).Scan(&updatedAt)

    if err != nil {
        if err == pgx.ErrNoRows {
            // Check if silence exists
            exists, _ := r.silenceExists(ctx, silence.ID)
            if !exists {
                r.metrics.Errors.WithLabelValues(operation, "not_found").Inc()
                return fmt.Errorf("%w: silence with ID %s", ErrSilenceNotFound, silence.ID)
            }
            // Optimistic lock conflict
            r.metrics.Errors.WithLabelValues(operation, "conflict").Inc()
            return fmt.Errorf("%w: silence was modified by another transaction", ErrSilenceConflict)
        }
        r.metrics.Errors.WithLabelValues(operation, "update").Inc()
        return fmt.Errorf("update silence: %w", err)
    }

    silence.UpdatedAt = &updatedAt
    r.metrics.Operations.WithLabelValues(operation, "success").Inc()

    r.logger.Info("silence updated",
        "silence_id", silence.ID,
        "created_by", silence.CreatedBy,
        "status", silence.Status,
    )

    return nil
}

// DeleteSilence implements SilenceRepository.DeleteSilence
func (r *PostgresSilenceRepository) DeleteSilence(ctx context.Context, id string) error {
    start := time.Now()
    operation := "delete"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Validate UUID format
    if _, err := uuid.Parse(id); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "invalid_uuid").Inc()
        return fmt.Errorf("%w: %s", ErrInvalidUUID, err)
    }

    query := `DELETE FROM silences WHERE id = $1`

    result, err := r.pool.Exec(ctx, query, id)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "delete").Inc()
        return fmt.Errorf("delete silence: %w", err)
    }

    if result.RowsAffected() == 0 {
        r.metrics.Errors.WithLabelValues(operation, "not_found").Inc()
        return fmt.Errorf("%w: silence with ID %s", ErrSilenceNotFound, id)
    }

    r.metrics.Operations.WithLabelValues(operation, "success").Inc()
    r.metrics.ActiveSilences.WithLabelValues("deleted").Dec()

    r.logger.Info("silence deleted", "silence_id", id)

    return nil
}

// CountSilences implements SilenceRepository.CountSilences
func (r *PostgresSilenceRepository) CountSilences(ctx context.Context, filter SilenceFilter) (int64, error) {
    start := time.Now()
    operation := "count"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Build count query (similar to ListSilences but with COUNT(*))
    query, args := r.buildCountQuery(filter)

    var count int64
    err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "query").Inc()
        return 0, fmt.Errorf("count silences: %w", err)
    }

    r.metrics.Operations.WithLabelValues(operation, "success").Inc()

    return count, nil
}

// ExpireSilences implements SilenceRepository.ExpireSilences
func (r *PostgresSilenceRepository) ExpireSilences(ctx context.Context, before time.Time, deleteExpired bool) (int64, error) {
    start := time.Now()
    operation := "expire"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.CleanupDuration.Observe(duration)
    }()

    var query string
    if deleteExpired {
        query = `DELETE FROM silences WHERE ends_at < $1 AND status = 'expired' LIMIT 1000`
    } else {
        query = `UPDATE silences SET status = 'expired' WHERE ends_at < $1 AND status != 'expired' LIMIT 1000`
    }

    result, err := r.pool.Exec(ctx, query, before)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "execute").Inc()
        return 0, fmt.Errorf("expire silences: %w", err)
    }

    affected := result.RowsAffected()
    if deleteExpired {
        r.metrics.CleanupDeleted.Add(float64(affected))
    }

    r.logger.Info("silences expired",
        "count", affected,
        "before", before.Format(time.RFC3339),
        "deleted", deleteExpired,
    )

    return affected, nil
}

// GetExpiringSoon implements SilenceRepository.GetExpiringSoon
func (r *PostgresSilenceRepository) GetExpiringSoon(ctx context.Context, window time.Duration) ([]*silencing.Silence, error) {
    start := time.Now()
    operation := "expiring_soon"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    expiresBy := time.Now().Add(window)

    query := `
        SELECT id, created_by, comment, starts_at, ends_at, matchers, status, created_at, updated_at
        FROM silences
        WHERE status = 'active'
          AND ends_at <= $1
        ORDER BY ends_at ASC
    `

    rows, err := r.pool.Query(ctx, query, expiresBy)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "query").Inc()
        return nil, fmt.Errorf("query expiring silences: %w", err)
    }
    defer rows.Close()

    // Scan results (similar to ListSilences)
    silences := []*silencing.Silence{}
    for rows.Next() {
        var silence silencing.Silence
        var matchersJSON []byte
        var updatedAt *time.Time

        err := rows.Scan(
            &silence.ID,
            &silence.CreatedBy,
            &silence.Comment,
            &silence.StartsAt,
            &silence.EndsAt,
            &matchersJSON,
            &silence.Status,
            &silence.CreatedAt,
            &updatedAt,
        )
        if err != nil {
            r.metrics.Errors.WithLabelValues(operation, "scan").Inc()
            return nil, fmt.Errorf("scan silence: %w", err)
        }

        if err := json.Unmarshal(matchersJSON, &silence.Matchers); err != nil {
            r.metrics.Errors.WithLabelValues(operation, "unmarshal").Inc()
            return nil, fmt.Errorf("unmarshal matchers: %w", err)
        }

        silence.UpdatedAt = updatedAt
        silences = append(silences, &silence)
    }

    r.metrics.Operations.WithLabelValues(operation, "success").Inc()

    return silences, nil
}

// BulkUpdateStatus implements SilenceRepository.BulkUpdateStatus
func (r *PostgresSilenceRepository) BulkUpdateStatus(ctx context.Context, ids []string, status silencing.SilenceStatus) error {
    start := time.Now()
    operation := "bulk_update_status"

    defer func() {
        duration := time.Since(start).Seconds()
        r.metrics.OperationDuration.WithLabelValues(operation, "success").Observe(duration)
    }()

    // Begin transaction
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "begin_tx").Inc()
        return fmt.Errorf("begin transaction: %w", err)
    }
    defer tx.Rollback(ctx)

    // Batch update
    query := `UPDATE silences SET status = $1, updated_at = NOW() WHERE id = ANY($2)`

    result, err := tx.Exec(ctx, query, status, ids)
    if err != nil {
        r.metrics.Errors.WithLabelValues(operation, "execute").Inc()
        return fmt.Errorf("bulk update status: %w", err)
    }

    affected := result.RowsAffected()

    // Commit transaction
    if err := tx.Commit(ctx); err != nil {
        r.metrics.Errors.WithLabelValues(operation, "commit_tx").Inc()
        return fmt.Errorf("commit transaction: %w", err)
    }

    r.metrics.Operations.WithLabelValues(operation, "success").Inc()

    r.logger.Info("bulk status update",
        "count", affected,
        "new_status", status,
    )

    return nil
}

// silenceExists checks if a silence with the given ID exists.
func (r *PostgresSilenceRepository) silenceExists(ctx context.Context, id string) (bool, error) {
    query := `SELECT EXISTS(SELECT 1 FROM silences WHERE id = $1)`

    var exists bool
    err := r.pool.QueryRow(ctx, query, id).Scan(&exists)
    if err != nil {
        return false, err
    }

    return exists, nil
}
```

---

### 3. TTL Cleanup Worker

**File**: `ttl_cleanup_worker.go`

```go
package silencing

import (
    "context"
    "log/slog"
    "time"
)

// TTLCleanupWorker periodically deletes expired silences from the database.
type TTLCleanupWorker struct {
    repo      SilenceRepository
    interval  time.Duration // Cleanup frequency (default: 1h)
    retention time.Duration // Keep expired for this long before deletion (default: 24h)
    batchSize int           // Max silences per cleanup run (default: 1000)
    logger    *slog.Logger

    stopCh chan struct{}
    doneCh chan struct{}
}

// NewTTLCleanupWorker creates a new TTL cleanup worker.
func NewTTLCleanupWorker(
    repo SilenceRepository,
    interval time.Duration,
    retention time.Duration,
    batchSize int,
    logger *slog.Logger,
) *TTLCleanupWorker {
    if logger == nil {
        logger = slog.Default()
    }

    return &TTLCleanupWorker{
        repo:      repo,
        interval:  interval,
        retention: retention,
        batchSize: batchSize,
        logger:    logger,
        stopCh:    make(chan struct{}),
        doneCh:    make(chan struct{}),
    }
}

// Start starts the TTL cleanup worker.
// Runs in a goroutine until Stop() is called or ctx is cancelled.
func (w *TTLCleanupWorker) Start(ctx context.Context) {
    ticker := time.NewTicker(w.interval)
    defer ticker.Stop()

    w.logger.Info("TTL cleanup worker started",
        "interval", w.interval,
        "retention", w.retention,
        "batch_size", w.batchSize,
    )

    // Run immediately on start
    w.runCleanup(ctx)

    for {
        select {
        case <-ctx.Done():
            w.logger.Info("TTL cleanup worker stopped (context cancelled)")
            close(w.doneCh)
            return
        case <-w.stopCh:
            w.logger.Info("TTL cleanup worker stopped (stop signal)")
            close(w.doneCh)
            return
        case <-ticker.C:
            w.runCleanup(ctx)
        }
    }
}

// runCleanup executes a single cleanup cycle.
func (w *TTLCleanupWorker) runCleanup(ctx context.Context) {
    start := time.Now()
    before := time.Now().Add(-w.retention)

    w.logger.Debug("starting TTL cleanup",
        "before", before.Format(time.RFC3339),
    )

    deleted, err := w.repo.ExpireSilences(ctx, before, true)
    if err != nil {
        w.logger.Error("TTL cleanup failed",
            "error", err,
            "duration_ms", time.Since(start).Milliseconds(),
        )
        return
    }

    w.logger.Info("TTL cleanup completed",
        "deleted_count", deleted,
        "duration_ms", time.Since(start).Milliseconds(),
    )
}

// Stop stops the TTL cleanup worker and waits for it to finish.
// Blocks until the worker has stopped.
func (w *TTLCleanupWorker) Stop() {
    close(w.stopCh)
    <-w.doneCh
    w.logger.Info("TTL cleanup worker stopped gracefully")
}
```

---

### 4. Metrics Definition

**File**: `metrics.go`

```go
package silencing

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

// SilenceMetrics contains Prometheus metrics for silence repository operations.
type SilenceMetrics struct {
    // Operations count by operation type and status
    Operations *prometheus.CounterVec

    // Operation duration by operation type
    OperationDuration *prometheus.HistogramVec

    // Error count by operation and error type
    Errors *prometheus.CounterVec

    // Active silences gauge by status
    ActiveSilences *prometheus.GaugeVec

    // Cleanup operations stats
    CleanupDeleted *prometheus.Counter
    CleanupDuration *prometheus.Histogram
}

// NewSilenceMetrics creates and registers Prometheus metrics for silence repository.
func NewSilenceMetrics() *SilenceMetrics {
    return &SilenceMetrics{
        Operations: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "infra_silence_repo",
                Name:      "operations_total",
                Help:      "Total silence repository operations by type and status",
            },
            []string{"operation", "status"},
        ),
        OperationDuration: promauto.NewHistogramVec(
            prometheus.HistogramOpts{
                Namespace: "alert_history",
                Subsystem: "infra_silence_repo",
                Name:      "operation_duration_seconds",
                Help:      "Duration of silence repository operations in seconds",
                Buckets:   []float64{.001, .003, .005, .01, .02, .05, .1, .2, .5, 1},
            },
            []string{"operation", "status"},
        ),
        Errors: promauto.NewCounterVec(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "infra_silence_repo",
                Name:      "errors_total",
                Help:      "Total silence repository errors by operation and error type",
            },
            []string{"operation", "error_type"},
        ),
        ActiveSilences: promauto.NewGaugeVec(
            prometheus.GaugeOpts{
                Namespace: "alert_history",
                Subsystem: "business_silence",
                Name:      "active_total",
                Help:      "Number of active silences by status",
            },
            []string{"status"},
        ),
        CleanupDeleted: promauto.NewCounter(
            prometheus.CounterOpts{
                Namespace: "alert_history",
                Subsystem: "infra_silence_repo",
                Name:      "cleanup_deleted_total",
                Help:      "Total silences deleted by TTL cleanup worker",
            },
        ),
        CleanupDuration: promauto.NewHistogram(
            prometheus.HistogramOpts{
                Namespace: "alert_history",
                Subsystem: "infra_silence_repo",
                Name:      "cleanup_duration_seconds",
                Help:      "Duration of TTL cleanup operations in seconds",
                Buckets:   []float64{.1, .25, .5, 1, 2, 5, 10, 30, 60},
            },
        ),
    }
}
```

---

### 5. Error Types

**File**: `silence_repository_errors.go`

```go
package silencing

import "errors"

var (
    // ErrSilenceNotFound is returned when a silence does not exist.
    ErrSilenceNotFound = errors.New("silence not found")

    // ErrSilenceExists is returned when trying to create a duplicate silence.
    ErrSilenceExists = errors.New("silence already exists")

    // ErrSilenceConflict is returned when optimistic locking fails.
    ErrSilenceConflict = errors.New("silence was modified by another transaction")

    // ErrInvalidFilter is returned when filter parameters are invalid.
    ErrInvalidFilter = errors.New("invalid filter parameters")

    // ErrInvalidUUID is returned when a UUID is malformed.
    ErrInvalidUUID = errors.New("invalid UUID format")

    // ErrDatabaseConnection is returned for connection issues.
    ErrDatabaseConnection = errors.New("database connection error")

    // ErrTransactionFailed is returned for transaction errors.
    ErrTransactionFailed = errors.New("database transaction failed")

    // ErrValidation is returned for validation failures.
    ErrValidation = errors.New("validation failed")
)
```

---

## 🧪 Testing Strategy

### Test Structure

```
postgres_silence_repository_test.go          # Unit tests (800 LOC)
├── TestCreateSilence (8 tests)
│   ├── Valid silence
│   ├── Invalid validation
│   ├── Duplicate ID
│   ├── Empty ID (auto-generate)
│   ├── Invalid UUID
│   ├── Matchers marshal error
│   ├── Database error
│   └── Context cancellation
├── TestGetSilenceByID (5 tests)
│   ├── Silence found
│   ├── Silence not found
│   ├── Invalid UUID
│   ├── Database error
│   └── JSONB unmarshal error
├── TestListSilences (12 tests)
│   ├── Empty result
│   ├── Filter by status (single, multiple)
│   ├── Filter by creator
│   ├── Filter by time range
│   ├── Filter by matcher name/value
│   ├── Pagination (limit, offset)
│   ├── Sorting (asc, desc)
│   ├── Combined filters
│   └── Invalid filter
├── TestUpdateSilence (6 tests)
│   ├── Success
│   ├── Not found
│   ├── Optimistic lock conflict
│   ├── Validation error
│   ├── Database error
│   └── Context cancellation
├── TestDeleteSilence (4 tests)
│   ├── Success
│   ├── Not found
│   ├── Invalid UUID
│   └── Database error
├── TestCountSilences (5 tests)
├── TestExpireSilences (6 tests)
├── TestGetExpiringSoon (4 tests)
└── TestBulkUpdateStatus (5 tests)

postgres_silence_repository_integration_test.go  # Integration tests (400 LOC)
├── TestIntegration_FullCRUDCycle
├── TestIntegration_ConcurrentCreates
├── TestIntegration_ConcurrentUpdates
├── TestIntegration_TTLCleanup
└── TestIntegration_BulkOperations

postgres_silence_repository_bench_test.go        # Benchmarks (200 LOC)
├── BenchmarkCreateSilence
├── BenchmarkGetSilenceByID
├── BenchmarkListSilences_100
├── BenchmarkListSilences_1000
├── BenchmarkUpdateSilence
├── BenchmarkDeleteSilence
├── BenchmarkExpireSilences_1000
└── BenchmarkJSONBSearch

ttl_cleanup_worker_test.go                       # Worker tests (150 LOC)
├── TestTTLCleanupWorker_StartStop
├── TestTTLCleanupWorker_CleanupExecution
├── TestTTLCleanupWorker_ContextCancellation
└── TestTTLCleanupWorker_MultipleCleanupCycles
```

---

## 🔒 Security Considerations

1. **SQL Injection Prevention**: Parameterized queries (`$1`, `$2`, ...)
2. **JSONB Injection**: Validate JSON structure before insertion
3. **UUID Validation**: Parse UUID before database queries
4. **Input Sanitization**: Validate all user inputs через `silence.Validate()`
5. **Transaction Isolation**: READ COMMITTED level (default)
6. **Connection Security**: TLS для PostgreSQL connections
7. **Audit Trail**: Record all operations via structured logging

---

## 📊 Performance Optimization

### Index Usage

| Query Pattern | Index Used | Speedup |
|---------------|------------|---------|
| `WHERE status = 'active'` | `idx_silences_status` | ~100x |
| `WHERE status IN ('pending', 'active')` | `idx_silences_active` | ~50x |
| `WHERE matchers @> '{"name":"alertname"}'` | `idx_silences_matchers` | ~20x |
| `WHERE created_by = 'user@example.com'` | `idx_silences_created_by` | ~50x |
| `ORDER BY created_at DESC` | `idx_silences_created_at` | ~30x |

### Connection Pool Configuration

```go
config, err := pgxpool.ParseConfig(dsn)
config.MaxConns = 25                  // Max connections
config.MinConns = 5                   // Keep-alive connections
config.MaxConnLifetime = 1 * time.Hour
config.MaxConnIdleTime = 15 * time.Minute
config.HealthCheckPeriod = 1 * time.Minute
```

---

## 🎯 Definition of Done

- ✅ All 9 repository methods implemented
- ✅ 40+ tests passing (90%+ coverage)
- ✅ 8+ benchmarks meet performance targets
- ✅ 6 Prometheus metrics exported
- ✅ TTL cleanup worker implemented
- ✅ Comprehensive godoc documentation
- ✅ README with usage examples
- ✅ Integration tested with real PostgreSQL
- ✅ Zero linter errors
- ✅ Code reviewed and approved

---

**Designed**: 2025-11-05
**Approved**: 2025-11-05
**Status**: 🔄 IN PROGRESS
**Target Completion**: 2025-11-05 (10-14 hours)
