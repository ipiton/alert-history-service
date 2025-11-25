# TN-155: Template API (CRUD) - Implementation Tasks

**Task ID**: TN-155
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Estimate**: 16-20 hours
**Status**: 🔄 IN PROGRESS
**Date**: 2025-11-25

---

## 📋 Task Breakdown (10 Phases)

### ✅ Phase 0: Analysis & Documentation (COMPLETE - 3h)

- [x] **0.1** Comprehensive analysis document
  - Dependencies analysis
  - Architecture overview
  - Risk assessment
  - Success metrics definition

- [x] **0.2** Requirements document
  - Functional requirements (FR-1 to FR-5)
  - Non-functional requirements (NFR-1 to NFR-5)
  - Acceptance criteria
  - Out of scope definition

- [x] **0.3** Technical design document
  - Database schema design
  - Component architecture
  - API specifications
  - Integration patterns

- [x] **0.4** Implementation tasks checklist (THIS FILE)

**Deliverables**: 4 comprehensive markdown documents
**Status**: ✅ **COMPLETE**

---

### Phase 1: Database Foundation (3h)

#### 1.1 Database Migrations

- [ ] **1.1.1** Create migration file `20251125_create_templates_tables.sql`
  - `templates` table with all fields and constraints
  - `template_versions` table with history tracking
  - Indexes for performance (8 indexes total)
  - Full-text search index
  - Triggers for `updated_at`

- [ ] **1.1.2** Create rollback migration `20251125_create_templates_tables_down.sql`
  - DROP tables in correct order
  - DROP triggers
  - DROP indexes

- [ ] **1.1.3** Test migrations
  - Run `goose up`
  - Verify schema
  - Run `goose down`
  - Verify cleanup

**Files**:
- `go-app/migrations/20251125_create_templates_tables.sql`

**Acceptance**: Schema matches design.md exactly

---

#### 1.2 Go Domain Models

- [ ] **1.2.1** Create `Template` model
  - All fields from schema
  - JSON tags
  - Validation tags
  - Helper methods

- [ ] **1.2.2** Create `TemplateVersion` model
  - Version history fields
  - JSON serialization

- [ ] **1.2.3** Create `TemplateType` enum
  - Constants for each type
  - Validation method

- [ ] **1.2.4** Create filter/pagination structs
  - `ListFilters`
  - `VersionFilters`
  - `DeleteOptions`

**Files**:
- `go-app/internal/core/domain/template.go`

**Acceptance**: All models compile, validation tags correct

---

### Phase 2: Repository Layer (4h)

#### 2.1 Repository Interface

- [ ] **2.1.1** Define `TemplateRepository` interface
  - CRUD methods (8 methods)
  - Version control methods (3 methods)
  - Utility methods (2 methods)

**Files**:
- `go-app/internal/infrastructure/template/repository.go`

---

#### 2.2 PostgreSQL Implementation

- [ ] **2.2.1** Implement `PostgresTemplateRepository`
  - Constructor with pgxpool
  - Error handling wrapper

- [ ] **2.2.2** Implement CRUD operations
  - `Create()` - INSERT with tx
  - `GetByName()` - SELECT with cache key
  - `GetByID()` - SELECT by UUID
  - `List()` - SELECT with filters, pagination, sorting
  - `Update()` - UPDATE + INSERT version
  - `Delete()` - Soft/hard delete

- [ ] **2.2.3** Implement version operations
  - `CreateVersion()` - INSERT into template_versions
  - `ListVersions()` - SELECT with pagination
  - `GetVersion()` - SELECT specific version

- [ ] **2.2.4** Implement utility methods
  - `Exists()` - Check name uniqueness
  - `CountByType()` - Aggregate stats

**Files**:
- `go-app/internal/infrastructure/template/postgres_repository.go`

**Acceptance**: All methods work with real PostgreSQL

---

#### 2.3 Repository Tests

- [ ] **2.3.1** Unit tests for each method (20+ tests)
  - Happy path tests
  - Error cases
  - Edge cases
  - Transaction rollback tests

- [ ] **2.3.2** Integration tests with PostgreSQL
  - Use testcontainers
  - Test concurrent access
  - Test constraints

- [ ] **2.3.3** Benchmarks
  - CREATE benchmark
  - GET benchmark
  - LIST benchmark

**Files**:
- `go-app/internal/infrastructure/template/postgres_repository_test.go`
- `go-app/internal/infrastructure/template/repository_bench_test.go`

**Acceptance**: 80%+ coverage, all tests pass

---

### Phase 3: Cache Layer (2h)

#### 3.1 Cache Interface

- [ ] **3.1.1** Define `TemplateCache` interface
  - Get, Set, Invalidate methods
  - Stats method

**Files**:
- `go-app/internal/infrastructure/template/cache.go`

---

#### 3.2 Two-Tier Cache Implementation

- [ ] **3.2.1** Implement `TwoTierTemplateCache`
  - L1: LRU in-memory cache (1000 entries)
  - L2: Redis cache (5min TTL)
  - Fallback chain: L1 → L2 → miss

- [ ] **3.2.2** Implement cache operations
  - `Get()` - Try L1, then L2, update L1 on L2 hit
  - `Set()` - Set both L1 and L2
  - `Invalidate()` - Clear both caches
  - `GetStats()` - Hit ratio, entry count

**Files**:
- `go-app/internal/infrastructure/template/two_tier_cache.go`

**Acceptance**: Cache hit ratio > 90% in tests

---

#### 3.3 Cache Tests

- [ ] **3.3.1** Unit tests (10+ tests)
  - L1 hit scenario
  - L2 hit scenario (L1 miss)
  - Cache miss scenario
  - Invalidation tests

- [ ] **3.3.2** Performance tests
  - Benchmark Get (cached vs uncached)
  - Concurrent access test

**Files**:
- `go-app/internal/infrastructure/template/cache_test.go`

**Acceptance**: All tests pass, < 10ms L1 hit

---

### Phase 4: Business Logic Layer (4h)

#### 4.1 Validator Implementation

- [ ] **4.1.1** Define `TemplateValidator` interface
  - ValidateSyntax method
  - ValidateWithData method
  - ValidateBusinessRules method

- [ ] **4.1.2** Implement `DefaultTemplateValidator`
  - Integration with TN-153 engine
  - Mock data generation
  - Error parsing
  - Suggestion generation (fuzzy matching)

**Files**:
- `go-app/internal/business/template/validator.go`

**Acceptance**: Catches all syntax errors from TN-153

---

#### 4.2 Manager Implementation

- [ ] **4.2.1** Define `TemplateManager` interface
  - CRUD methods (5 methods)
  - Version control methods (3 methods)
  - Advanced methods (3 methods)

- [ ] **4.2.2** Implement `DefaultTemplateManager`
  - Constructor with dependencies
  - Transaction management
  - Error handling

- [ ] **4.2.3** Implement CRUD operations
  - `CreateTemplate()` - Validate, check uniqueness, insert
  - `GetTemplate()` - Try cache, fallback to DB
  - `ListTemplates()` - Apply filters, cache results
  - `UpdateTemplate()` - Validate, increment version, update
  - `DeleteTemplate()` - Soft/hard delete with checks

- [ ] **4.2.4** Implement version control
  - `ListVersions()` - Query versions table
  - `GetVersion()` - Get historical content
  - `RollbackToVersion()` - Create new version from old content

- [ ] **4.2.5** Implement advanced features (150%)
  - `BatchCreate()` - Atomic batch insert
  - `GetDiff()` - Text diff between versions
  - `GetStats()` - Aggregate statistics

**Files**:
- `go-app/internal/business/template/manager.go`

**Acceptance**: All business rules enforced

---

#### 4.3 Business Logic Tests

- [ ] **4.3.1** Unit tests with mocks (30+ tests)
  - Mock repository
  - Mock validator
  - Mock cache
  - Test all scenarios

- [ ] **4.3.2** Integration tests
  - Real dependencies
  - End-to-end flows

**Files**:
- `go-app/internal/business/template/manager_test.go`
- `go-app/internal/business/template/validator_test.go`

**Acceptance**: 85%+ coverage

---

### Phase 5: HTTP Handler Layer (3h)

#### 5.1 Request/Response DTOs

- [ ] **5.1.1** Define request structs
  - `CreateTemplateRequest`
  - `UpdateTemplateRequest`
  - `ValidateTemplateRequest`
  - `RollbackRequest`

- [ ] **5.1.2** Define response structs
  - `TemplateResponse`
  - `ListTemplatesResponse`
  - `VersionListResponse`
  - `ValidationResultResponse`
  - `TemplateStatsResponse`

**Files**:
- `go-app/cmd/server/handlers/template_models.go`

---

#### 5.2 Handler Implementation

- [ ] **5.2.1** Create `TemplateHandler` struct
  - Dependencies injection
  - Middleware integration

- [ ] **5.2.2** Implement CRUD endpoints (5 handlers)
  - `CreateTemplate()` - POST /api/v2/templates
  - `ListTemplates()` - GET /api/v2/templates
  - `GetTemplate()` - GET /api/v2/templates/{name}
  - `UpdateTemplate()` - PUT /api/v2/templates/{name}
  - `DeleteTemplate()` - DELETE /api/v2/templates/{name}

- [ ] **5.2.3** Implement validation endpoint
  - `ValidateTemplate()` - POST /api/v2/templates/validate

- [ ] **5.2.4** Implement version endpoints (2 handlers)
  - `ListTemplateVersions()` - GET /api/v2/templates/{name}/versions
  - `GetTemplateVersion()` - GET /api/v2/templates/{name}/versions/{version}
  - `RollbackTemplate()` - POST /api/v2/templates/{name}/rollback

- [ ] **5.2.5** Implement advanced endpoints (150%)
  - `BatchCreate()` - POST /api/v2/templates/batch
  - `GetTemplateDiff()` - GET /api/v2/templates/{name}/diff
  - `GetTemplateStats()` - GET /api/v2/templates/stats
  - `TestTemplate()` - POST /api/v2/templates/{name}/test

**Files**:
- `go-app/cmd/server/handlers/template.go`

**Acceptance**: All endpoints return correct status codes

---

#### 5.3 Handler Tests

- [ ] **5.3.1** Unit tests with mocks (25+ tests)
  - Test each endpoint
  - Happy path
  - Error cases
  - Validation errors

- [ ] **5.3.2** HTTP integration tests
  - httptest.Server
  - Test full request/response cycle

**Files**:
- `go-app/cmd/server/handlers/template_test.go`

**Acceptance**: 80%+ coverage

---

### Phase 6: Metrics & Logging (1h)

#### 6.1 Prometheus Metrics

- [ ] **6.1.1** Define metrics in `pkg/metrics/business.go`
  - 10+ metrics (counters, histograms, gauges)

- [ ] **6.1.2** Add metrics recording in handler
  - Record on every request
  - Record duration
  - Record errors

- [ ] **6.1.3** Add metrics recording in manager
  - Cache hit/miss
  - Validation errors
  - Version operations

**Files**:
- `pkg/metrics/business.go` (update existing)

**Acceptance**: All metrics appear in /metrics endpoint

---

#### 6.2 Structured Logging

- [ ] **6.2.1** Add logging in handler
  - Log all requests (INFO level)
  - Log errors (ERROR level)
  - Include request ID, user ID

- [ ] **6.2.2** Add logging in manager
  - Log business operations
  - Log validation failures
  - Log performance warnings

**Acceptance**: Logs parseable as JSON

---

### Phase 7: Integration & Main.go (1h)

#### 7.1 Dependency Initialization

- [ ] **7.1.1** Initialize repository in main.go
  - Create PostgresTemplateRepository
  - Pass pgxpool.Pool

- [ ] **7.1.2** Initialize cache
  - Create TwoTierTemplateCache
  - Pass Redis cache

- [ ] **7.1.3** Initialize validator
  - Create DefaultTemplateValidator
  - Pass TN-153 engine

- [ ] **7.1.4** Initialize manager
  - Create DefaultTemplateManager
  - Wire all dependencies

- [ ] **7.1.5** Initialize handler
  - Create TemplateHandler
  - Wire manager, metrics, logger

**Files**:
- `go-app/cmd/server/main.go` (update)

---

#### 7.2 Route Registration

- [ ] **7.2.1** Register all routes
  - 7 baseline endpoints
  - 4 advanced endpoints
  - Apply middleware (auth, metrics, logging)

**Example**:
```go
// Template API routes (admin-only for mutations)
router.Handle("POST /api/v2/templates",
    authMiddleware(adminOnly(templateHandler.CreateTemplate)))
router.Handle("GET /api/v2/templates",
    authMiddleware(templateHandler.ListTemplates))
router.Handle("GET /api/v2/templates/{name}",
    authMiddleware(templateHandler.GetTemplate))
// ... etc
```

**Acceptance**: All routes accessible

---

#### 7.3 Data Seeding

- [ ] **7.3.1** Create seeding script
  - Import TN-154 default templates
  - Insert into database
  - Handle duplicates gracefully

**Files**:
- `go-app/cmd/seed/seed_templates.go`

**Acceptance**: Default templates loaded on first run

---

### Phase 8: Testing & Validation (3h)

#### 8.1 Unit Tests

- [ ] **8.1.1** Run all unit tests
  - `go test ./...`
  - Verify 80%+ coverage
  - Fix failing tests

**Acceptance**: All tests pass, coverage target met

---

#### 8.2 Integration Tests

- [ ] **8.2.1** Write E2E integration tests
  - Create → Get → Update → Get → Delete flow
  - Version control flow
  - Rollback flow
  - Cache invalidation flow

- [ ] **8.2.2** Run integration tests
  - Use testcontainers for PostgreSQL + Redis
  - Verify all flows work

**Files**:
- `go-app/cmd/server/handlers/template_integration_test.go`

**Acceptance**: All integration tests pass

---

#### 8.3 Performance Benchmarks

- [ ] **8.3.1** Write benchmarks
  - GET (cached)
  - GET (uncached)
  - POST
  - PUT
  - DELETE

- [ ] **8.3.2** Run benchmarks
  - Verify targets met (< 10ms GET cached, etc.)
  - Profile with pprof if needed

**Files**:
- `go-app/cmd/server/handlers/template_bench_test.go`

**Acceptance**: All performance targets met

---

#### 8.4 Load Testing

- [ ] **8.4.1** Create k6 scenarios
  - Steady state (1000 req/s for 5min)
  - Spike test (2000 req/s burst)
  - Stress test (find breaking point)

- [ ] **8.4.2** Run k6 tests
  - Record p95, p99 latencies
  - Verify throughput > 1000 req/s

**Files**:
- `k6/templates_load_test.js`

**Acceptance**: Throughput target met, no errors

---

### Phase 9: Documentation (2h)

#### 9.1 OpenAPI Specification

- [ ] **9.1.1** Complete OpenAPI 3.0.3 spec
  - All endpoints documented
  - Request/response schemas
  - Error responses
  - Examples

**Files**:
- `docs/openapi/template_api.yaml`

**Acceptance**: Spec validates with Swagger validator

---

#### 9.2 User Documentation

- [ ] **9.2.1** Create README.md
  - Quick start guide
  - Usage examples (curl, Go, Python)
  - Configuration
  - Troubleshooting

- [ ] **9.2.2** Create MIGRATION_GUIDE.md
  - How to seed default templates
  - How to migrate custom templates
  - Version compatibility notes

**Files**:
- `go-app/internal/business/template/README.md`
- `tasks/alertmanager-plus-plus-oss/TN-155-template-api-crud/MIGRATION_GUIDE.md`

**Acceptance**: Clear, actionable documentation

---

#### 9.3 Troubleshooting Guide

- [ ] **9.3.1** Document common issues
  - Template syntax errors
  - Permission denied
  - Cache inconsistency
  - Database connection errors

- [ ] **9.3.2** Add runbook sections
  - How to debug slow queries
  - How to clear cache
  - How to rollback template

**Files**:
- `tasks/alertmanager-plus-plus-oss/TN-155-template-api-crud/TROUBLESHOOTING.md`

**Acceptance**: Covers all common scenarios

---

### Phase 10: 150% Quality Certification (1h)

#### 10.1 Quality Audit

- [ ] **10.1.1** Verify all acceptance criteria met
  - Implementation: 7 baseline + 3 advanced endpoints ✅
  - Testing: 80%+ coverage ✅
  - Performance: All targets met ✅
  - Security: RBAC enforced ✅
  - Observability: 10+ metrics ✅
  - Documentation: Complete ✅

- [ ] **10.1.2** Run quality checklist
  - Zero linter errors
  - Zero race conditions (go test -race)
  - Zero known bugs
  - Clean git history

---

#### 10.2 Certification Report

- [ ] **10.2.1** Create COMPLETION_REPORT.md
  - Executive summary
  - Deliverables breakdown
  - Quality metrics
  - Performance results
  - Test coverage report
  - Lessons learned

- [ ] **10.2.2** Calculate quality score
  - Implementation: 40 pts
  - Testing: 30 pts
  - Performance: 20 pts
  - Documentation: 15 pts
  - Code quality: 10 pts
  - Advanced features: +10 pts
  - **Target**: 150/100 pts (Grade A+)

**Files**:
- `tasks/alertmanager-plus-plus-oss/TN-155-template-api-crud/COMPLETION_REPORT.md`

**Acceptance**: Quality score ≥ 150%

---

#### 10.3 Git & Merge

- [ ] **10.3.1** Create feature branch
  - Name: `feature/TN-155-template-api-150pct`
  - Commit convention: `feat(TN-155): <message>`

- [ ] **10.3.2** Commit all changes
  - Migrations
  - Implementation files
  - Tests
  - Documentation

- [ ] **10.3.3** Update CHANGELOG.md
  - Add comprehensive TN-155 entry
  - List all features
  - Breaking changes (none expected)

- [ ] **10.3.4** Update TASKS.md
  - Mark TN-155 as complete
  - Add quality metrics

- [ ] **10.3.5** Merge to main
  - Create PR (if applicable)
  - Merge with --no-ff
  - Push to origin

**Acceptance**: Successfully merged, zero conflicts

---

## 📊 Progress Tracking

### Overall Progress: 30% (3/10 phases complete)

- [x] Phase 0: Analysis & Documentation ✅ **COMPLETE**
- [ ] Phase 1: Database Foundation (0/13 subtasks)
- [ ] Phase 2: Repository Layer (0/17 subtasks)
- [ ] Phase 3: Cache Layer (0/7 subtasks)
- [ ] Phase 4: Business Logic Layer (0/17 subtasks)
- [ ] Phase 5: HTTP Handler Layer (0/13 subtasks)
- [ ] Phase 6: Metrics & Logging (0/5 subtasks)
- [ ] Phase 7: Integration & Main.go (0/9 subtasks)
- [ ] Phase 8: Testing & Validation (0/10 subtasks)
- [ ] Phase 9: Documentation (0/7 subtasks)
- [ ] Phase 10: 150% Certification (0/8 subtasks)

**Total Subtasks**: 106 subtasks (3 complete, 103 remaining)

---

## 🎯 Quality Gates

Each phase must pass these gates before proceeding:

### Code Quality
- [ ] Zero linter errors (`golangci-lint run`)
- [ ] Zero race conditions (`go test -race`)
- [ ] Code compiles successfully
- [ ] All imports used

### Testing
- [ ] All tests pass (`go test ./...`)
- [ ] Coverage meets target (80%+ for phase)
- [ ] Benchmarks validate performance

### Documentation
- [ ] Godoc comments for public APIs
- [ ] README updated
- [ ] Examples provided

### Integration
- [ ] No breaking changes to existing APIs
- [ ] Dependencies properly injected
- [ ] Graceful error handling

---

## 📈 Success Metrics (150% Target)

### Implementation (40 points)
- [x] 7 baseline endpoints (20 pts)
- [ ] 3+ advanced endpoints (10 pts)
- [ ] Clean architecture (10 pts)

### Testing (30 points)
- [ ] 80%+ coverage (15 pts)
- [ ] 30+ unit tests (10 pts)
- [ ] 5+ integration tests (5 pts)

### Performance (20 points)
- [ ] < 10ms GET cached (10 pts)
- [ ] > 90% cache hit ratio (5 pts)
- [ ] > 1000 req/s throughput (5 pts)

### Documentation (15 points)
- [ ] OpenAPI spec (5 pts)
- [ ] README (5 pts)
- [ ] Troubleshooting guide (5 pts)

### Code Quality (10 points)
- [ ] Zero linter errors (5 pts)
- [ ] Zero race conditions (5 pts)

### Advanced Features Bonus (+10 points)
- [ ] Batch operations (+3 pts)
- [ ] Template diff (+3 pts)
- [ ] Template analytics (+2 pts)
- [ ] Template testing (+2 pts)

**Total Target**: 150/100 points

---

## 🚀 Next Steps

1. ✅ Complete analysis & documentation (Phase 0)
2. ⏳ Create Git feature branch
3. ⏳ Implement database migrations (Phase 1)
4. ⏳ Build repository layer (Phase 2)
5. ⏳ Add caching (Phase 3)
6. ⏳ Implement business logic (Phase 4)
7. ⏳ Create HTTP handlers (Phase 5)
8. ⏳ Add observability (Phase 6)
9. ⏳ Integrate in main.go (Phase 7)
10. ⏳ Complete testing (Phase 8)
11. ⏳ Write documentation (Phase 9)
12. ⏳ Certification & merge (Phase 10)

---

**Status**: ✅ Phase 0 COMPLETE, Ready for Phase 1
**Timeline**: 16-20 hours remaining
**Quality Target**: 150% (Grade A+)
**Author**: AI Assistant
**Date**: 2025-11-25
