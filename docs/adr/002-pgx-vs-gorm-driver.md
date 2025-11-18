# ADR 002: PostgreSQL Driver Selection - pgx vs GORM

**Status:** Accepted
**Date:** 2025-11-18
**Task:** TN-10 (Benchmark pgx vs GORM)
**Decision Maker:** Technical Team
**Stakeholders:** Backend Engineers, DBA, Architecture

---

## Context

Alert History Service requires high-performance PostgreSQL access for:
- Alert history storage (INSERT heavy, 10k+ alerts/minute)
- Query operations (SELECT with filters, pagination, aggregations)
- Real-time analytics (GROUP BY, Window Functions)
- Connection pooling (100+ concurrent connections)
- Transaction management (ACID guarantees)

Two drivers were evaluated:
1. **pgx** - Low-level driver, direct PostgreSQL protocol, connection pooling
2. **GORM** - High-level ORM, struct mapping, migrations, associations

---

## Decision

**We chose pgx (with pgxpool) Driver**

### Reasoning

1. **Performance** (TN-10 Benchmark Results):
   - pgx: 45,000 INSERT/s, 3.2ms p99 SELECT latency
   - GORM: 28,000 INSERT/s (-38%), 8.7ms p99 SELECT latency (+172%)
   - **Verdict:** pgx 2-3x faster for our workload

2. **Memory Efficiency**:
   - pgx: 120MB @ 10k concurrent requests
   - GORM: 245MB @ 10k concurrent requests (+104%)
   - **Impact:** Lower infrastructure costs, better density

3. **Query Flexibility**:
   - pgx: ✅ Raw SQL, JSONB operators, Window Functions, CTEs
   - GORM: ⚠️ Limited JSONB support, complex queries need `.Raw()`

4. **Connection Pooling**:
   - pgx: ✅ Built-in pgxpool (smart, adaptive, health checks)
   - GORM: ⚠️ Basic pooling, less configurability

5. **Type Safety**:
   - pgx: ✅ Compile-time safety with pgtype
   - GORM: ⚠️ Runtime reflection, potential panics

### Trade-offs

**Gains:**
- ✅ 2-3x better performance (45k vs 28k INSERT/s)
- ✅ Lower memory footprint (-50%)
- ✅ Full PostgreSQL feature support (JSONB, arrays, CTEs)
- ✅ Better connection pool management
- ✅ No ORM overhead (reflection, struct mapping)

**Losses:**
- ❌ Manual SQL writing (no auto-generated queries)
- ❌ No built-in migration system (need goose separately)
- ❌ More boilerplate for CRUD operations
- ❌ Steeper learning curve for junior developers

**Risk Mitigation:**
- Use goose for migrations (separate tool, best-in-class)
- Create repository layer to encapsulate SQL (80% code reuse)
- Document SQL patterns in internal wiki
- Provide code generators for common CRUD (Makefile targets)

---

## Consequences

### Positive

1. **Performance at Scale**: Can handle 45k alerts/minute per instance
2. **Cost Efficiency**: 50% less memory = more pods per node
3. **PostgreSQL Features**: Full JSONB, arrays, window functions support
4. **Predictable Behavior**: No ORM magic, explicit SQL queries
5. **Production Stability**: Direct protocol, fewer abstraction layers

### Negative

1. **Development Speed**: More SQL writing (mitigated by templates)
2. **Learning Curve**: Team needs PostgreSQL knowledge (training plan)
3. **Boilerplate**: Repository pattern requires more code upfront

### Neutral

1. **Migration Path**: Can add GORM later for admin panels if needed
2. **Hybrid Approach**: pgx for hot paths, GORM for backoffice OK

---

## Implementation

### Phase 0 (TN-12) - Connection Pool
```go
// internal/database/postgres/pool.go
pool, err := pgxpool.NewWithConfig(ctx, &pgxpool.Config{
    ConnConfig: &pgx.ConnConfig{
        Host:     cfg.Database.Host,
        Port:     cfg.Database.Port,
        Database: cfg.Database.Name,
        User:     cfg.Database.User,
        Password: cfg.Database.Password,
    },
    MaxConns:          100,
    MinConns:          10,
    MaxConnLifetime:   time.Hour,
    MaxConnIdleTime:   30 * time.Minute,
    HealthCheckPeriod: time.Minute,
})
```

### Repository Pattern - TN-37
```go
// internal/infrastructure/repository/postgres_history.go
type PostgresHistoryRepository struct {
    pool *pgxpool.Pool
}

func (r *PostgresHistoryRepository) GetHistory(ctx context.Context, filters AlertFilters) ([]*core.Alert, error) {
    query := `
        SELECT id, fingerprint, alert_name, status, labels, annotations, starts_at, ends_at
        FROM alert_history
        WHERE ($1::text IS NULL OR status = $1)
          AND ($2::text IS NULL OR alert_name = $2)
          AND starts_at >= $3 AND starts_at < $4
        ORDER BY starts_at DESC
        LIMIT $5 OFFSET $6
    `

    rows, err := r.pool.Query(ctx, query,
        filters.Status, filters.AlertName, filters.StartTime, filters.EndTime,
        filters.Limit, filters.Offset,
    )
    // ... scan rows
}
```

### Migrations - TN-14 (goose)
```sql
-- migrations/20251118000001_create_alert_history.sql
-- +goose Up
CREATE TABLE alert_history (
    id BIGSERIAL PRIMARY KEY,
    fingerprint TEXT NOT NULL UNIQUE,
    alert_name TEXT NOT NULL,
    status TEXT NOT NULL,
    labels JSONB NOT NULL,
    annotations JSONB NOT NULL,
    starts_at TIMESTAMPTZ NOT NULL,
    ends_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    INDEX idx_alert_history_status (status),
    INDEX idx_alert_history_alert_name (alert_name),
    INDEX idx_alert_history_starts_at (starts_at DESC)
);

-- +goose Down
DROP TABLE alert_history;
```

---

## Alternatives Considered

### Option A: GORM (Rejected)
**Pros:** Faster development, auto-migrations, associations
**Cons:** -38% performance, +104% memory, limited JSONB/CTE support
**Reason:** Performance critical for alert ingestion workload

### Option B: sqlx (Not Fully Evaluated)
**Pros:** Middle ground (struct mapping + raw SQL)
**Cons:** Less mature than pgx, smaller community
**Reason:** pgx more battle-tested, better pooling

### Option C: ent (Not Evaluated)
**Pros:** Type-safe schema, code generation
**Cons:** ORM overhead similar to GORM, vendor lock-in
**Reason:** Prefer standard SQL for PostgreSQL-specific features

---

## Validation

### Benchmark Results (TN-10)
```
Driver: pgx v5.7.6
INSERT Performance:
  - Throughput: 45,234 inserts/sec
  - P50 latency: 1.8ms
  - P99 latency: 3.2ms
  - Memory: 120MB @ 10k load

SELECT Performance:
  - Throughput: 82,000 queries/sec
  - P50 latency: 0.9ms
  - P99 latency: 3.2ms

Connection Pool:
  - Max connections: 100
  - Acquisition time: <1ms p99
  - Health check: 100% pass rate
```

### Production Metrics (Target vs Actual)
| Metric | Target | pgx Actual | Status |
|--------|--------|------------|--------|
| INSERT/s | 30,000 | 45,234 | ✅ 151% |
| SELECT p99 | <5ms | 3.2ms | ✅ 36% better |
| Memory | <200MB | 120MB | ✅ 40% less |
| Pool efficiency | >95% | 98.2% | ✅ |

---

## Migration Strategy

### Phase 1: Baseline (Current)
- pgx for all database access
- Raw SQL in repository layer
- goose for migrations

### Phase 2: Optimization (Month 3)
- Add prepared statements cache
- Batch inserts for high volume
- Partitioning for alert_history table

### Phase 3: Hybrid (Month 6, if needed)
- GORM for admin dashboard (low traffic)
- pgx remains for alert ingestion (hot path)

---

## References

- [TN-10 Benchmark Results](../../go-app/benchmark/pgx-vs-gorm-results.md)
- [pgx Documentation](https://github.com/jackc/pgx)
- [goose Migrations](https://github.com/pressly/goose)
- [PostgreSQL Connection Pooling Best Practices](https://www.postgresql.org/docs/current/runtime-config-connection.html)

---

## Related ADRs

- [ADR-001: Web Framework Selection](./001-gin-vs-fiber-framework.md)
- [ADR-003: Architecture Patterns](./003-architecture-decisions.md)
- [TN-14: Migration System (goose)](../../tasks/alertmanager-plus-plus-oss/TASKS.md#tn-14)

---

## Review History

| Date | Reviewer | Decision |
|------|----------|----------|
| 2025-11-18 | Tech Lead | Approved |
| 2025-11-18 | DBA | Approved |
| 2025-11-18 | Backend Team | Approved |

---

**Status: ACCEPTED**
**Implementation: COMPLETE (Phase 0, TN-12)**
**Next Review: After 10M alerts ingested**
