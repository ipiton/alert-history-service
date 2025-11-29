# TN-201: Storage Backend Selection Logic - Implementation Tasks

**Date**: 2025-11-29
**Target Quality**: 150% (Grade A+ EXCEPTIONAL)
**Duration Estimate**: 8-12 hours
**Phase**: 13 (Production Packaging)

---

## 📊 Task Overview

### Summary
Implement intelligent storage backend selection logic based on deployment profile with 150% quality target.

### Deliverables
- [x] Requirements documentation (3,067 LOC) ✅
- [x] Technical design (2,552 LOC) ✅
- [ ] Implementation code (~800 LOC)
- [ ] Comprehensive tests (~700 LOC)
- [ ] User documentation (~400 LOC)
- [ ] Completion report (~500 LOC)

### Quality Targets (150%)
| Metric | Baseline | Target (150%) | Status |
|--------|----------|---------------|--------|
| Code LOC | 500 | 800+ | ⏳ |
| Test LOC | 400 | 700+ | ⏳ |
| Test Coverage | 70% | 85%+ | ⏳ |
| Documentation | 2K | 3K+ | ✅ (5,619 LOC) |
| Performance | Baseline | 2x faster | ⏳ |

---

## 🎯 Phase 0: Foundation (COMPLETE ✅)

### Task 0.1: Create Project Structure ✅
**Duration**: 5 minutes
**Status**: ✅ COMPLETE

**Deliverables**:
- [x] Directory: `tasks/TN-201-storage-backend-selection/`
- [x] File: `requirements.md` (3,067 LOC)
- [x] File: `design.md` (2,552 LOC)
- [x] File: `tasks.md` (this file)

---

## 🎯 Phase 1: Storage Interface & Factory (2-3 hours)

### Task 1.1: Create Storage Package Structure
**Duration**: 15 minutes
**Priority**: P0
**Status**: ⏳ TODO

**Steps**:
1. Create directory: `go-app/internal/storage/`
2. Create subdirectories:
   - `go-app/internal/storage/sqlite/`
   - `go-app/internal/storage/memory/`
   - `go-app/internal/storage/postgres/` (exists, reorganize if needed)
3. Create files:
   - `storage/factory.go` (factory pattern)
   - `storage/metrics.go` (Prometheus metrics)
   - `storage/errors.go` (custom errors)

**Acceptance Criteria**:
- [x] Directory structure matches design document
- [x] All files created with package declarations
- [x] Zero compilation errors

**Commands**:
```bash
mkdir -p go-app/internal/storage/{sqlite,memory,postgres}
touch go-app/internal/storage/{factory.go,metrics.go,errors.go}
```

---

### Task 1.2: Implement Storage Factory
**Duration**: 45 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/factory.go`

**Implementation**:
```go
package storage

import (
    "context"
    "fmt"
    "log/slog"

    "github.com/jackc/pgxpool/v5"
    "github.com/vitaliisemenov/alert-history/go-app/internal/config"
    "github.com/vitaliisemenov/alert-history/go-app/internal/core"
)

// NewStorage creates appropriate storage backend based on config profile
func NewStorage(
    ctx context.Context,
    cfg *config.Config,
    pgPool *pgxpool.Pool,
    logger *slog.Logger,
) (core.AlertStorage, error) {
    // Implementation from design.md
}

func initLiteStorage(...) (core.AlertStorage, error) {
    // SQLite initialization
}

func initStandardStorage(...) (core.AlertStorage, error) {
    // Postgres initialization (delegate to existing)
}

func NewFallbackStorage(logger *slog.Logger) core.AlertStorage {
    // Memory storage for degradation
}
```

**Acceptance Criteria**:
- [x] Profile detection via `cfg.IsLiteProfile()` / `cfg.IsStandardProfile()`
- [x] Separate init functions for clarity
- [x] Descriptive error wrapping
- [x] Structured logging (profile, backend, duration)
- [x] Zero compilation errors

**Testing**:
- [ ] Unit test: Lite profile → SQLite
- [ ] Unit test: Standard profile → Postgres
- [ ] Unit test: Invalid profile → error
- [ ] Unit test: Nil pool (Standard) → error
- [ ] Unit test: Empty path (Lite) → error

---

### Task 1.3: Implement Storage Metrics
**Duration**: 30 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/metrics.go`

**Implementation**:
```go
package storage

import "github.com/prometheus/client_golang/prometheus"

var (
    StorageBackendType = prometheus.NewGaugeVec(...)
    StorageOperationsTotal = prometheus.NewCounterVec(...)
    StorageOperationDuration = prometheus.NewHistogramVec(...)
    StorageErrorsTotal = prometheus.NewCounterVec(...)
    SQLiteFileSizeBytes = prometheus.NewGauge(...)
    StorageHealthStatus = prometheus.NewGaugeVec(...)
    StorageConnections = prometheus.NewGaugeVec(...)
)

func init() {
    prometheus.MustRegister(...)
}
```

**Metrics** (7 total):
1. `alert_history_storage_backend_type` (Gauge)
2. `alert_history_storage_operations_total` (Counter)
3. `alert_history_storage_operation_duration_seconds` (Histogram)
4. `alert_history_storage_errors_total` (Counter)
5. `alert_history_storage_file_size_bytes` (Gauge, SQLite only)
6. `alert_history_storage_health_status` (Gauge)
7. `alert_history_storage_connections` (Gauge, Postgres only)

**Acceptance Criteria**:
- [x] All 7 metrics defined
- [x] Registered in `init()` function
- [x] Labels match design document
- [x] Histogram buckets appropriate (0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0)

---

### Task 1.4: Define Custom Errors
**Duration**: 15 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/errors.go`

**Implementation**:
```go
package storage

import "fmt"

// ErrInvalidProfile indicates invalid deployment profile
type ErrInvalidProfile struct {
    Profile string
}

func (e ErrInvalidProfile) Error() string {
    return fmt.Sprintf("invalid deployment profile: %s", e.Profile)
}

// ErrStorageInitFailed indicates storage initialization failure
type ErrStorageInitFailed struct {
    Backend string
    Cause   error
}

func (e ErrStorageInitFailed) Error() string {
    return fmt.Sprintf("storage initialization failed (%s): %v", e.Backend, e.Cause)
}

// ... (add more error types as needed)
```

**Acceptance Criteria**:
- [x] Custom error types implement `error` interface
- [x] Error messages descriptive and actionable
- [x] Wrap underlying errors (maintain stack trace)

---

## 🎯 Phase 2: SQLite Adapter Implementation (3-4 hours)

### Task 2.1: Implement SQLite Storage Struct
**Duration**: 30 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/sqlite_storage.go`

**Implementation**:
```go
package sqlite

import (
    "context"
    "database/sql"
    "log/slog"
    "sync"

    _ "modernc.org/sqlite"

    "github.com/vitaliisemenov/alert-history/go-app/internal/core"
)

type SQLiteStorage struct {
    db     *sql.DB
    logger *slog.Logger
    path   string
    mu     sync.RWMutex
}

func NewSQLiteStorage(
    ctx context.Context,
    path string,
    logger *slog.Logger,
) (*SQLiteStorage, error) {
    // Implementation from design.md
}

func (s *SQLiteStorage) initSchema(ctx context.Context) error {
    // Create tables and indexes
}
```

**Steps**:
1. Add dependency: `go get modernc.org/sqlite@latest`
2. Implement struct with connection pool config
3. Enable WAL mode (`?_journal_mode=WAL`)
4. Enable foreign keys (`PRAGMA foreign_keys = ON`)
5. Create schema (alerts table + 6 indexes)

**Acceptance Criteria**:
- [x] WAL mode enabled for concurrency
- [x] Foreign keys enabled
- [x] Schema compatible with Postgres (same columns)
- [x] Secure file permissions (0600)
- [x] Directory creation (parent directory 0700)
- [x] Zero compilation errors

**Testing**:
- [ ] Unit test: NewSQLiteStorage success
- [ ] Unit test: Invalid path → error
- [ ] Unit test: Schema creation
- [ ] Unit test: File permissions check

---

### Task 2.2: Implement CRUD Operations
**Duration**: 90 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/sqlite_storage.go`

**Methods to Implement**:
1. `CreateAlert(ctx, alert)` - UPSERT logic with `ON CONFLICT`
2. `GetAlert(ctx, fingerprint)` - SELECT by primary key
3. `UpdateAlert(ctx, alert)` - Reuse CreateAlert (UPSERT)
4. `DeleteAlert(ctx, fingerprint)` - DELETE with existence check

**Implementation Details**:
- Serialize labels/annotations to JSON TEXT
- Convert timestamps to Unix milliseconds (int64)
- Use parameterized queries (prevent SQL injection)
- Return `core.ErrAlertNotFound` when appropriate
- Record metrics for each operation

**Acceptance Criteria**:
- [x] All methods implement `core.AlertStorage` interface
- [x] JSON serialization/deserialization working
- [x] Timestamps converted correctly (UnixMilli)
- [x] UPSERT idempotency (duplicate fingerprints)
- [x] Errors wrapped with context
- [x] Metrics recorded (operation, duration, status)

**Testing**:
- [ ] Unit test: CreateAlert (new alert)
- [ ] Unit test: CreateAlert (duplicate fingerprint → update)
- [ ] Unit test: GetAlert (exists)
- [ ] Unit test: GetAlert (not found → ErrAlertNotFound)
- [ ] Unit test: UpdateAlert
- [ ] Unit test: DeleteAlert (exists)
- [ ] Unit test: DeleteAlert (not found → ErrAlertNotFound)

---

### Task 2.3: Implement Query Operations
**Duration**: 60 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/sqlite_storage.go`

**Methods to Implement**:
1. `ListAlerts(ctx, filter)` - SELECT with WHERE + pagination + sorting
2. `CountAlerts(ctx, filter)` - SELECT COUNT(*) with WHERE

**Filtering**:
- Status IN (firing, resolved)
- Severity IN (critical, warning, info)
- Namespace IN (...)
- Fingerprints IN (...)
- Labels JSONB (basic key=value matching)
- Time ranges (starts_at, ends_at)

**Pagination & Sorting**:
- LIMIT / OFFSET
- ORDER BY (created_at, starts_at, updated_at) ASC/DESC

**Acceptance Criteria**:
- [x] All filters implemented
- [x] Pagination working (limit, offset)
- [x] Sorting working (all fields, ASC/DESC)
- [x] Efficient SQL (indexed columns)
- [x] Empty result set → empty array (not nil)
- [x] Metrics recorded

**Testing**:
- [ ] Unit test: ListAlerts (no filters)
- [ ] Unit test: ListAlerts (status filter)
- [ ] Unit test: ListAlerts (severity filter)
- [ ] Unit test: ListAlerts (namespace filter)
- [ ] Unit test: ListAlerts (pagination)
- [ ] Unit test: ListAlerts (sorting ASC)
- [ ] Unit test: ListAlerts (sorting DESC)
- [ ] Unit test: CountAlerts (no filters)
- [ ] Unit test: CountAlerts (with filters)

---

### Task 2.4: Implement Lifecycle Methods
**Duration**: 20 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/sqlite_storage.go`

**Methods to Implement**:
1. `Close()` - Gracefully close database connection
2. `Health(ctx)` - Connection liveness check (Ping)
3. `GetFileSize()` - Return SQLite file size in bytes

**Acceptance Criteria**:
- [x] Close() idempotent (can call multiple times)
- [x] Health() returns error if connection closed
- [x] GetFileSize() handles missing file (return 0)
- [x] Metrics recorded (health status)

**Testing**:
- [ ] Unit test: Close() success
- [ ] Unit test: Close() idempotent
- [ ] Unit test: Health() success
- [ ] Unit test: Health() after close → error
- [ ] Unit test: GetFileSize() existing file
- [ ] Unit test: GetFileSize() missing file → 0

---

### Task 2.5: Implement Helper Functions
**Duration**: 15 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/helpers.go`

**Functions**:
```go
// placeholders generates SQL placeholders ("?", "?, ?", etc.)
func placeholders(count int) string

// buildWhereClause constructs WHERE clause from filter
func buildWhereClause(filter core.AlertFilter) (string, []interface{})

// validateFilePath prevents directory traversal attacks
func validateFilePath(path string) error
```

**Acceptance Criteria**:
- [x] placeholders() handles count=0
- [x] buildWhereClause() combines all filters
- [x] validateFilePath() rejects ".." and forbidden paths

**Testing**:
- [ ] Unit test: placeholders(0) → ""
- [ ] Unit test: placeholders(3) → "?, ?, ?"
- [ ] Unit test: buildWhereClause (multiple filters)
- [ ] Unit test: validateFilePath valid paths
- [ ] Unit test: validateFilePath invalid paths → error

---

## 🎯 Phase 3: In-Memory Fallback Storage (1 hour)

### Task 3.1: Implement Memory Storage
**Duration**: 45 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/memory/memory_storage.go`

**Implementation**:
```go
package memory

import (
    "context"
    "log/slog"
    "sync"

    "github.com/vitaliisemenov/alert-history/go-app/internal/core"
)

type MemoryStorage struct {
    mu       sync.RWMutex
    alerts   map[string]*core.Alert
    logger   *slog.Logger
    capacity int
}

func NewMemoryStorage(logger *slog.Logger) *MemoryStorage {
    // Implementation from design.md
}
```

**Methods**:
- CreateAlert, GetAlert, UpdateAlert, DeleteAlert
- ListAlerts (basic filtering only)
- CountAlerts
- Close, Health

**Features**:
- Thread-safe (RWMutex)
- Capacity limit (10K alerts)
- Deep copies (prevent mutation)
- Warning logs (non-persistent reminder)

**Acceptance Criteria**:
- [x] All methods implement `core.AlertStorage` interface
- [x] Thread-safe concurrent access
- [x] Capacity limit enforced (FIFO eviction)
- [x] Warning logs on every operation
- [x] Health() always returns success

**Testing**:
- [ ] Unit test: CreateAlert
- [ ] Unit test: GetAlert (exists)
- [ ] Unit test: GetAlert (not found)
- [ ] Unit test: UpdateAlert
- [ ] Unit test: DeleteAlert
- [ ] Unit test: ListAlerts (basic)
- [ ] Unit test: CountAlerts
- [ ] Unit test: Capacity limit (eviction)
- [ ] Unit test: Concurrent access (10 goroutines)

---

## 🎯 Phase 4: Main.go Integration (1 hour)

### Task 4.1: Update Main.go Initialization
**Duration**: 45 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/cmd/server/main.go` (lines ~230-280)

**Changes Required**:
1. Import storage factory: `import "github.com/vitaliisemenov/alert-history/go-app/internal/storage"`
2. Replace hardcoded Postgres init with profile-based selection
3. Add graceful fallback to memory storage on failures
4. Emit metrics for storage backend type
5. Add startup logging (profile, backend, status)

**Implementation**:
```go
// Initialize storage based on deployment profile (TN-201)
var alertStorage core.AlertStorage
var pool *postgres.PostgresPool
var storageInitErr error

if cfg.UsesPostgresStorage() {
    // Standard profile: PostgreSQL
    // ... (existing Postgres init code)
    alertStorage, storageInitErr = storage.NewStorage(ctx, cfg, pool.Pool(), logger)
} else if cfg.UsesEmbeddedStorage() {
    // Lite profile: SQLite
    logger.Info("Lite profile detected - initializing SQLite storage")
    alertStorage, storageInitErr = storage.NewStorage(ctx, cfg, nil, logger)
} else {
    storageInitErr = fmt.Errorf("unknown storage backend: %s", cfg.Storage.Backend)
}

// Graceful degradation
if storageInitErr != nil {
    logger.Error("Storage init failed, falling back to memory",
        "error", storageInitErr)
    alertStorage = storage.NewFallbackStorage(logger)
    // Emit metric
    metricsRegistry.Business().StorageBackendType.WithLabelValues("memory").Set(0)
} else {
    // Success metric
    backend := "sqlite"
    if cfg.UsesPostgresStorage() {
        backend = "postgres"
    }
    metricsRegistry.Business().StorageBackendType.WithLabelValues(backend).Set(1)
}
```

**Acceptance Criteria**:
- [x] Conditional initialization based on profile
- [x] Graceful fallback to memory storage
- [x] Metrics emitted (backend type, init errors)
- [x] Startup logging comprehensive
- [x] Zero breaking changes to handlers
- [x] Zero compilation errors

**Testing**:
- [ ] Manual test: Lite profile (config.yaml with profile=lite)
- [ ] Manual test: Standard profile (default config.yaml)
- [ ] Manual test: Invalid config → memory fallback

---

### Task 4.2: Verify Handler Compatibility
**Duration**: 15 minutes
**Priority**: P0
**Status**: ⏳ TODO

**Steps**:
1. Review all handlers using `alertStorage`
2. Verify no assumptions about backend type
3. Test with both SQLite and Postgres backends
4. Confirm error handling unchanged

**Handlers to Check**:
- `handlers/webhook.go` (POST /webhook)
- `handlers/history.go` (GET /history)
- `handlers/analytics.go` (GET /report)

**Acceptance Criteria**:
- [x] All handlers work with both backends
- [x] Error responses unchanged
- [x] Performance acceptable (< 10ms p95)

---

## 🎯 Phase 5: Testing (2-3 hours)

### Task 5.1: Factory Unit Tests
**Duration**: 30 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/factory_test.go`

**Test Cases** (10 tests):
1. Lite profile + valid path → SQLite
2. Standard profile + valid pool → Postgres
3. Invalid profile → error
4. Lite profile + empty path → error
5. Standard profile + nil pool → error
6. Fallback storage creation
7. Metrics recorded (backend type)
8. Structured logging (profile, backend)
9. Error wrapping (descriptive messages)
10. Concurrent factory calls (thread-safe)

**Acceptance Criteria**:
- [x] All 10 tests passing
- [x] Coverage: 90%+ of factory.go
- [x] Zero test flakes
- [x] Benchmarks: Factory creation < 1ms

---

### Task 5.2: SQLite Unit Tests
**Duration**: 90 minutes
**Priority**: P0
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/sqlite_storage_test.go`

**Test Cases** (35+ tests):
1. NewSQLiteStorage (valid path)
2. NewSQLiteStorage (invalid path → error)
3. CreateAlert (new)
4. CreateAlert (duplicate → update)
5. GetAlert (exists)
6. GetAlert (not found → ErrAlertNotFound)
7. UpdateAlert
8. DeleteAlert (exists)
9. DeleteAlert (not found → error)
10. ListAlerts (no filters)
11. ListAlerts (status=firing)
12. ListAlerts (severity=critical)
13. ListAlerts (namespace=prod)
14. ListAlerts (pagination: limit 10, offset 0)
15. ListAlerts (pagination: limit 10, offset 10)
16. ListAlerts (sort by created_at ASC)
17. ListAlerts (sort by created_at DESC)
18. CountAlerts (no filters)
19. CountAlerts (status=firing)
20. Close()
21. Close() idempotent
22. Health() success
23. Health() after close → error
24. GetFileSize() existing file
25. GetFileSize() missing file → 0
26. Concurrent CreateAlert (10 goroutines)
27. Concurrent GetAlert (10 goroutines)
28. UPSERT idempotency (duplicate fingerprints)
29. JSON serialization (labels)
30. JSON serialization (annotations)
31. Timestamp conversion (UnixMilli)
32. Schema creation (tables + indexes)
33. WAL mode enabled
34. Foreign keys enabled
35. Metrics recorded (all operations)

**Acceptance Criteria**:
- [x] All 35+ tests passing
- [x] Coverage: 85%+ of sqlite_storage.go
- [x] Zero test flakes
- [x] Zero race conditions (run with `-race`)

---

### Task 5.3: Memory Storage Unit Tests
**Duration**: 30 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/memory/memory_storage_test.go`

**Test Cases** (15 tests):
1. NewMemoryStorage
2. CreateAlert
3. GetAlert (exists)
4. GetAlert (not found)
5. UpdateAlert
6. DeleteAlert (exists)
7. DeleteAlert (not found)
8. ListAlerts (basic)
9. CountAlerts
10. Capacity limit (10K alerts)
11. Eviction (FIFO)
12. Concurrent access (10 goroutines)
13. Deep copy (mutation prevention)
14. Health() always success
15. Close() (no-op)

**Acceptance Criteria**:
- [x] All 15 tests passing
- [x] Coverage: 90%+ of memory_storage.go
- [x] Zero race conditions

---

### Task 5.4: Integration Tests
**Duration**: 45 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/integration_test.go`

**Test Cases** (10 tests):
1. Real file I/O (create, write, read, delete)
2. Persistence across restarts (close, reopen, data intact)
3. Concurrent reads/writes (WAL mode validation)
4. Crash recovery (WAL replay)
5. Disk full scenario (graceful error)
6. Large dataset (1000 alerts)
7. Bulk insert (100 alerts in transaction)
8. Schema migration (future-proofing)
9. File permissions (0600 validation)
10. Directory creation (parent directory 0700)

**Acceptance Criteria**:
- [x] All 10 tests passing
- [x] Real SQLite file created/deleted
- [x] Zero data loss on restart
- [x] Concurrent safety validated

---

### Task 5.5: Benchmarks
**Duration**: 30 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/sqlite/bench_test.go`

**Benchmarks** (12 total):
1. BenchmarkSQLiteCreate
2. BenchmarkSQLiteGet
3. BenchmarkSQLiteUpdate
4. BenchmarkSQLiteDelete
5. BenchmarkSQLiteList10
6. BenchmarkSQLiteList100
7. BenchmarkSQLiteCount
8. BenchmarkSQLiteConcurrent (10 goroutines)
9. BenchmarkMemoryCreate
10. BenchmarkMemoryGet
11. BenchmarkFactoryLite
12. BenchmarkFactoryStandard

**Targets**:
- SQLite Create: < 3ms (p95)
- SQLite Get: < 1ms (p95)
- SQLite List(100): < 20ms (p95)
- Memory Create: < 1µs
- Factory: < 1ms

**Acceptance Criteria**:
- [x] All benchmarks passing
- [x] Performance meets/exceeds targets
- [x] Zero allocations in hot paths (Create, Get)

---

## 🎯 Phase 6: Documentation (1 hour)

### Task 6.1: Create User README
**Duration**: 30 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `go-app/internal/storage/README.md`

**Sections**:
1. Overview (Lite vs Standard profiles)
2. Quick Start (config examples)
3. SQLite Configuration (file path, PVC setup)
4. PostgreSQL Configuration (connection string)
5. Performance Comparison (SQLite vs Postgres)
6. Troubleshooting (common issues + solutions)
7. Migration Guide (Standard → Lite, Lite → Standard)
8. API Reference (AlertStorage interface)

**Acceptance Criteria**:
- [x] README: 400+ LOC
- [x] Config examples (both profiles)
- [x] Troubleshooting guide (5+ issues)
- [x] Migration guide (step-by-step)

---

### Task 6.2: Create Completion Report
**Duration**: 30 minutes
**Priority**: P1
**Status**: ⏳ TODO
**File**: `tasks/TN-201-storage-backend-selection/COMPLETION_REPORT.md`

**Sections**:
1. Executive Summary
2. Deliverables Summary (LOC breakdown)
3. Quality Metrics (150% achievement proof)
4. Performance Results (benchmarks)
5. Test Coverage Report (85%+ achieved)
6. Integration Status (main.go changes)
7. Deployment Checklist
8. Known Limitations
9. Future Enhancements
10. Certification (Grade A+)

**Acceptance Criteria**:
- [x] Report: 500+ LOC
- [x] Mathematical proof of 150% quality
- [x] Benchmark results (all targets met)
- [x] Coverage report (screenshot or table)

---

## 🎯 Phase 7: Final Validation (30 minutes)

### Task 7.1: Lint & Format
**Duration**: 10 minutes
**Priority**: P0
**Status**: ⏳ TODO

**Commands**:
```bash
# Format all Go files
gofmt -w go-app/internal/storage/

# Run linter
golangci-lint run go-app/internal/storage/...

# Run go vet
go vet ./go-app/internal/storage/...
```

**Acceptance Criteria**:
- [x] Zero linter warnings
- [x] Zero vet warnings
- [x] Consistent code formatting

---

### Task 7.2: Security Audit
**Duration**: 10 minutes
**Priority**: P1
**Status**: ⏳ TODO

**Commands**:
```bash
# Run gosec (security scanner)
gosec -exclude=G104 ./go-app/internal/storage/...

# Check for hardcoded secrets
grep -r "password\|secret\|token" go-app/internal/storage/
```

**Acceptance Criteria**:
- [x] Zero high-severity gosec findings
- [x] No hardcoded secrets
- [x] File permissions validated (0600)

---

### Task 7.3: Manual Testing
**Duration**: 10 minutes
**Priority**: P0
**Status**: ⏳ TODO

**Scenarios**:
1. **Lite Profile**: Edit config.yaml (profile=lite), start server, verify SQLite file created
2. **Standard Profile**: Use default config.yaml, start server, verify Postgres connection
3. **Fallback**: Invalid config, start server, verify memory storage warning

**Acceptance Criteria**:
- [x] All 3 scenarios working
- [x] Startup logs clear and descriptive
- [x] Metrics exposed on /metrics endpoint

---

## 📊 Progress Tracking

### Code Statistics
| Component | Files | LOC | Status |
|-----------|-------|-----|--------|
| Factory | 1 | 150 | ⏳ TODO |
| SQLite Adapter | 3 | 500 | ⏳ TODO |
| Memory Adapter | 1 | 150 | ⏳ TODO |
| Metrics | 1 | 80 | ⏳ TODO |
| Errors | 1 | 50 | ⏳ TODO |
| Helpers | 1 | 70 | ⏳ TODO |
| **Total Production** | **8** | **1,000** | **0%** |

### Test Statistics
| Component | Files | Tests | LOC | Status |
|-----------|-------|-------|-----|--------|
| Factory Tests | 1 | 10 | 150 | ⏳ TODO |
| SQLite Tests | 1 | 35 | 450 | ⏳ TODO |
| Memory Tests | 1 | 15 | 150 | ⏳ TODO |
| Integration Tests | 1 | 10 | 200 | ⏳ TODO |
| Benchmarks | 1 | 12 | 150 | ⏳ TODO |
| **Total Tests** | **5** | **82** | **1,100** | **0%** |

### Documentation Statistics
| Document | LOC | Status |
|----------|-----|--------|
| requirements.md | 3,067 | ✅ COMPLETE |
| design.md | 2,552 | ✅ COMPLETE |
| tasks.md | 1,400+ | ✅ COMPLETE |
| README.md | 400+ | ⏳ TODO |
| COMPLETION_REPORT.md | 500+ | ⏳ TODO |
| **Total Docs** | **7,919+** | **60%** |

### Quality Checklist
- [x] Requirements complete (3K LOC)
- [x] Design complete (2.5K LOC)
- [x] Tasks complete (1.4K LOC)
- [ ] Implementation (800+ LOC, 85%+ coverage)
- [ ] Tests (700+ LOC, 82+ tests)
- [ ] Benchmarks (12 benchmarks, all targets met)
- [ ] Documentation (400+ LOC README)
- [ ] Completion report (500+ LOC)
- [ ] Zero linter warnings
- [ ] Zero security issues

### Time Estimate
| Phase | Tasks | Estimated Time | Actual Time | Status |
|-------|-------|----------------|-------------|--------|
| Phase 0 | 1 | 5 min | 5 min | ✅ COMPLETE |
| Phase 1 | 4 | 2-3 hours | - | ⏳ TODO |
| Phase 2 | 5 | 3-4 hours | - | ⏳ TODO |
| Phase 3 | 1 | 1 hour | - | ⏳ TODO |
| Phase 4 | 2 | 1 hour | - | ⏳ TODO |
| Phase 5 | 5 | 2-3 hours | - | ⏳ TODO |
| Phase 6 | 2 | 1 hour | - | ⏳ TODO |
| Phase 7 | 3 | 30 min | - | ⏳ TODO |
| **Total** | **23** | **10.5-13 hours** | **5 min** | **2%** |

---

## 🎯 Success Criteria Summary

### Implementation (150%)
- [ ] 800+ LOC production code (vs 500 baseline = 160%)
- [ ] 700+ LOC test code (vs 400 baseline = 175%)
- [ ] 85%+ test coverage (vs 70% baseline = 121%)
- [ ] 7 Prometheus metrics (vs 4 baseline = 175%)
- [ ] 82+ tests (vs 50 baseline = 164%)

### Performance (150%)
- [ ] SQLite Create: < 3ms (vs 5ms baseline = 167%)
- [ ] SQLite Get: < 1ms (vs 2ms baseline = 200%)
- [ ] SQLite List(100): < 20ms (vs 30ms baseline = 150%)
- [ ] Memory Create: < 1µs (1000x faster than SQLite)

### Documentation (150%)
- [x] 5,619+ LOC documentation (vs 3,500 baseline = 161%)
- [ ] Comprehensive README (400+ LOC)
- [ ] Completion report (500+ LOC)
- [ ] Migration guide included

### Quality (150%)
- [ ] Zero linter warnings
- [ ] Zero security issues (gosec)
- [ ] Zero breaking changes
- [ ] Zero race conditions
- [ ] Grade A+ certification

---

## 🚀 Commit Strategy

### Feature Branch
```bash
git checkout -b feature/TN-201-storage-backend-150pct
```

### Commit Messages
1. `feat(TN-201): Phase 1 - Storage factory and metrics`
2. `feat(TN-201): Phase 2 - SQLite adapter implementation`
3. `feat(TN-201): Phase 3 - Memory fallback storage`
4. `feat(TN-201): Phase 4 - Main.go integration`
5. `test(TN-201): Phase 5 - Comprehensive test suite`
6. `docs(TN-201): Phase 6 - User documentation`
7. `chore(TN-201): Phase 7 - Final validation`
8. `feat(TN-201): Complete 150% quality implementation`

### Merge to Main
```bash
git checkout main
git merge --no-ff feature/TN-201-storage-backend-150pct
git push origin main
```

---

## 📝 Notes

### Dependencies
- **Upstream**: TN-200 (Profile Configuration) ✅
- **Downstream**: TN-202 (Redis Conditional Init), TN-203 (Main.go Profile Init)

### External Libraries
- `modernc.org/sqlite` (Pure Go, no CGO)
- `database/sql` (Standard library)
- `prometheus/client_golang` (Metrics)

### Known Limitations (Document in README)
1. SQLite: Limited to ~1K alerts/day (Lite profile use case)
2. SQLite: No horizontal scaling (single-node only)
3. SQLite: JSONB support limited (TEXT-based JSON storage)
4. Memory storage: Data lost on restart (not production-ready)

### Future Enhancements (Out of Scope)
1. BadgerDB support (alternative embedded storage)
2. Automatic backup/restore (SQLite → Postgres migration tool)
3. Compression (gzip labels/annotations in SQLite)
4. Replication (SQLite streaming replication via Litestream)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-29
**Status**: ✅ READY FOR IMPLEMENTATION
