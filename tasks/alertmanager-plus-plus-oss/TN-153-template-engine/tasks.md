# TN-153: Template Engine Integration — Task Breakdown

**Date**: 2025-11-22
**Task ID**: TN-153
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Status**: 📋 Planning Complete → Implementation Started
**Estimated Total Effort**: 8-12 hours
**Priority**: P0 (Critical for Sprint 3)

---

## 📊 Task Overview

**Total Tasks**: 40
**Phases**: 5
**Estimated Duration**: 8-12 hours (same-day completion target)

---

## ✅ Phase 0: Pre-Implementation Analysis (COMPLETED)

### Task 0.1: Requirements Analysis ✅
- [x] Изучить существующую UI template engine
- [x] Изучить formatter и publisher integration
- [x] Определить gap: receiver configs не поддерживают templates
- [x] Создать requirements.md (750+ LOC)

**Status**: ✅ COMPLETED
**Duration**: 1h
**Output**: requirements.md (750 LOC)

### Task 0.2: Technical Design ✅
- [x] Спроектировать NotificationTemplateEngine
- [x] Определить TemplateData structure
- [x] Спроектировать 50+ template functions
- [x] Спроектировать template cache (LRU)
- [x] Создать design.md (1,200+ LOC)

**Status**: ✅ COMPLETED
**Duration**: 1.5h
**Output**: design.md (1,200 LOC)

### Task 0.3: Task Planning ✅
- [x] Разбить на детальные задачи
- [x] Определить dependencies
- [x] Оценить effort
- [x] Создать tasks.md

**Status**: ✅ COMPLETED
**Duration**: 30min
**Output**: tasks.md (this file)

---

## 🔧 Phase 1: Template Engine Core (IN PROGRESS)

**Goal**: Implement core template engine with parsing, execution, and caching.

**Estimated Duration**: 2-3 hours

### Task 1.1: Create Package Structure
- [ ] Create `internal/notification/template/` directory
- [ ] Create base files: engine.go, data.go, cache.go, errors.go
- [ ] Define package documentation
- [ ] Setup imports

**Estimated Time**: 15min
**Files**: 4 files
**LOC**: ~50 LOC (structure)

### Task 1.2: Implement TemplateData
- [ ] Define TemplateData struct (all fields)
- [ ] Implement NewTemplateData() constructor
- [ ] Add helper methods (e.g., IsResolved(), Duration())
- [ ] Add validation

**Estimated Time**: 30min
**Files**: data.go, data_test.go
**LOC**: ~150 LOC

### Task 1.3: Implement Error Types
- [ ] Define error constants (ErrTemplateParse, ErrTemplateExecute, etc.)
- [ ] Implement error wrapping
- [ ] Add error helpers (IsParseError, IsTimeoutError)

**Estimated Time**: 15min
**Files**: errors.go
**LOC**: ~50 LOC

### Task 1.4: Implement TemplateCache
- [ ] Define TemplateCache struct
- [ ] Implement LRU cache (github.com/hashicorp/golang-lru)
- [ ] Implement Get/Set/Invalidate methods
- [ ] Implement Stats() method
- [ ] Add generateCacheKey() helper (SHA256)
- [ ] Thread-safety (sync.RWMutex)

**Estimated Time**: 45min
**Files**: cache.go, cache_test.go
**LOC**: ~200 LOC

### Task 1.5: Implement NotificationTemplateEngine Interface
- [ ] Define NotificationTemplateEngine interface
- [ ] Define TemplateEngineOptions struct
- [ ] Implement DefaultNotificationTemplateEngine struct
- [ ] Implement NewNotificationTemplateEngine() constructor

**Estimated Time**: 30min
**Files**: engine.go
**LOC**: ~150 LOC

### Task 1.6: Implement Execute() Method
- [ ] Implement Execute(ctx, tmpl, data) method
- [ ] Add cache lookup
- [ ] Add template parsing (text/template)
- [ ] Add template execution
- [ ] Add error handling with fallback
- [ ] Add context timeout support (5s)
- [ ] Add metrics recording

**Estimated Time**: 1h
**Files**: engine.go
**LOC**: ~200 LOC

### Task 1.7: Implement ExecuteMultiple() Method
- [ ] Implement ExecuteMultiple(ctx, templates, data) method
- [ ] Add parallel execution (goroutines)
- [ ] Add error aggregation
- [ ] Add timeout handling

**Estimated Time**: 30min
**Files**: engine.go
**LOC**: ~100 LOC

### Task 1.8: Implement Cache Management
- [ ] Implement InvalidateCache() method
- [ ] Implement GetCacheStats() method
- [ ] Add cache size monitoring

**Estimated Time**: 15min
**Files**: engine.go
**LOC**: ~50 LOC

---

## 📚 Phase 2: Template Functions Library (PENDING)

**Goal**: Implement 50+ Alertmanager-compatible template functions.

**Estimated Duration**: 2-3 hours

### Task 2.1: Time Functions (20 functions)
- [ ] Implement date(), humanizeTimestamp(), since(), until()
- [ ] Implement humanizeDuration()
- [ ] Implement now(), ago(), toDate()
- [ ] Implement dateInZone(), dateModify()
- [ ] Integrate sprig time functions

**Estimated Time**: 1h
**Files**: functions.go, functions_test.go
**LOC**: ~300 LOC

### Task 2.2: String Functions (15 functions)
- [ ] Implement toUpper(), toLower(), title()
- [ ] Implement truncate(), truncateWords()
- [ ] Implement join(), split()
- [ ] Implement trim(), trimPrefix(), trimSuffix()
- [ ] Integrate sprig string functions

**Estimated Time**: 45min
**Files**: functions.go, functions_test.go
**LOC**: ~250 LOC

### Task 2.3: URL Functions (5 functions)
- [ ] Implement urlEncode(), urlDecode()
- [ ] Implement urlQuery()
- [ ] Implement pathJoin(), pathBase()

**Estimated Time**: 30min
**Files**: functions.go, functions_test.go
**LOC**: ~100 LOC

### Task 2.4: Math Functions (10 functions)
- [ ] Implement add(), sub(), mul(), div(), mod()
- [ ] Implement max(), min()
- [ ] Implement round(), ceil(), floor()
- [ ] Implement humanize(), humanize1024()

**Estimated Time**: 45min
**Files**: functions.go, functions_test.go
**LOC**: ~200 LOC

### Task 2.5: Conditional Functions (5 functions)
- [ ] Implement default(), empty()
- [ ] Implement ternary()
- [ ] Implement has(), coalesce()

**Estimated Time**: 30min
**Files**: functions.go, functions_test.go
**LOC**: ~100 LOC

### Task 2.6: Collection Functions (10 functions)
- [ ] Implement sortAlpha(), reverse()
- [ ] Implement uniq(), without()
- [ ] Implement first(), last(), slice()
- [ ] Integrate sprig collection functions

**Estimated Time**: 45min
**Files**: functions.go, functions_test.go
**LOC**: ~200 LOC

### Task 2.7: Encoding Functions (5 functions)
- [ ] Implement b64enc(), b64dec()
- [ ] Implement toJson(), fromJson()
- [ ] Implement toPrettyJson()

**Estimated Time**: 30min
**Files**: functions.go, functions_test.go
**LOC**: ~100 LOC

---

## 🔗 Phase 3: Receiver Integration (PENDING)

**Goal**: Integrate template engine with receiver configs.

**Estimated Duration**: 2-3 hours

### Task 3.1: Slack Integration
- [ ] Implement ProcessSlackConfig()
- [ ] Render Title, Text, Pretext
- [ ] Render Fields (title, value)
- [ ] Add error handling
- [ ] Add tests

**Estimated Time**: 1h
**Files**: integration.go, integration_test.go
**LOC**: ~200 LOC

### Task 3.2: PagerDuty Integration
- [ ] Implement ProcessPagerDutyConfig()
- [ ] Render Summary
- [ ] Render Details (map[string]string)
- [ ] Add error handling
- [ ] Add tests

**Estimated Time**: 45min
**Files**: integration.go, integration_test.go
**LOC**: ~150 LOC

### Task 3.3: Email Integration (FUTURE - TN-154)
- [ ] Implement ProcessEmailConfig()
- [ ] Render Subject, Body
- [ ] Add error handling
- [ ] Add tests

**Estimated Time**: 45min
**Files**: integration.go, integration_test.go
**LOC**: ~150 LOC

### Task 3.4: Webhook Integration
- [ ] Implement ProcessWebhookConfig()
- [ ] Render custom fields
- [ ] Add error handling
- [ ] Add tests

**Estimated Time**: 30min
**Files**: integration.go, integration_test.go
**LOC**: ~100 LOC

### Task 3.5: Update Alert Formatter
- [ ] Integrate template engine into formatter
- [ ] Update formatSlack() to use templates
- [ ] Update formatPagerDuty() to use templates
- [ ] Add backward compatibility (non-template strings)

**Estimated Time**: 1h
**Files**: go-app/internal/infrastructure/publishing/formatter.go
**LOC**: ~150 LOC (modifications)

---

## 🧪 Phase 4: Testing & Validation (PENDING)

**Goal**: Comprehensive testing with 90%+ coverage.

**Estimated Duration**: 2-3 hours

### Task 4.1: Unit Tests - Engine
- [ ] Test Execute() - valid template
- [ ] Test Execute() - invalid template
- [ ] Test Execute() - missing fields
- [ ] Test Execute() - timeout
- [ ] Test ExecuteMultiple()
- [ ] Test cache hit/miss
- [ ] Test cache invalidation
- [ ] Test concurrent execution
- [ ] Test fallback on error
- [ ] Test metrics recording

**Estimated Time**: 1h
**Files**: engine_test.go
**LOC**: ~400 LOC
**Tests**: 10 tests

### Task 4.2: Unit Tests - Functions
- [ ] Test time functions (5 tests)
- [ ] Test string functions (5 tests)
- [ ] Test URL functions (3 tests)
- [ ] Test math functions (5 tests)
- [ ] Test conditional functions (3 tests)
- [ ] Test collection functions (5 tests)
- [ ] Test encoding functions (3 tests)

**Estimated Time**: 1h
**Files**: functions_test.go
**LOC**: ~500 LOC
**Tests**: 29 tests

### Task 4.3: Unit Tests - Cache
- [ ] Test Get/Set
- [ ] Test Invalidate
- [ ] Test Stats
- [ ] Test LRU eviction
- [ ] Test thread-safety

**Estimated Time**: 30min
**Files**: cache_test.go
**LOC**: ~200 LOC
**Tests**: 5 tests

### Task 4.4: Integration Tests
- [ ] Test ProcessSlackConfig
- [ ] Test ProcessPagerDutyConfig
- [ ] Test ProcessEmailConfig
- [ ] Test end-to-end notification flow
- [ ] Test error handling

**Estimated Time**: 1h
**Files**: integration_test.go
**LOC**: ~300 LOC
**Tests**: 5 tests

### Task 4.5: Benchmarks
- [ ] Benchmark template parse
- [ ] Benchmark execute (cached)
- [ ] Benchmark execute (uncached)
- [ ] Benchmark function calls
- [ ] Verify performance targets

**Estimated Time**: 30min
**Files**: engine_bench_test.go
**LOC**: ~150 LOC
**Tests**: 5 benchmarks

---

## 📊 Phase 5: Observability & Documentation (PENDING)

**Goal**: Add metrics, logging, and comprehensive documentation.

**Estimated Duration**: 1-2 hours

### Task 5.1: Prometheus Metrics
- [ ] Implement TemplateMetrics struct
- [ ] Add executionsTotal counter
- [ ] Add executionDuration histogram
- [ ] Add parseErrors counter
- [ ] Add cache metrics (hits, misses, size)
- [ ] Add functionCalls counter
- [ ] Register metrics

**Estimated Time**: 45min
**Files**: metrics.go
**LOC**: ~200 LOC

### Task 5.2: Structured Logging
- [ ] Add slog logging to Execute()
- [ ] Add slog logging to parse errors
- [ ] Add slog logging to execution errors
- [ ] Add slog logging to cache operations
- [ ] Add debug-level logging for function calls

**Estimated Time**: 30min
**Files**: engine.go, functions.go
**LOC**: ~50 LOC (additions)

### Task 5.3: Package Documentation
- [ ] Create README.md with overview
- [ ] Add quick start guide
- [ ] Add function reference
- [ ] Add examples
- [ ] Add troubleshooting guide

**Estimated Time**: 1h
**Files**: README.md
**LOC**: ~500 LOC

### Task 5.4: User Guide
- [ ] Create USER_GUIDE.md
- [ ] Add template syntax guide
- [ ] Add function examples
- [ ] Add receiver integration examples
- [ ] Add migration guide (Alertmanager)
- [ ] Add best practices

**Estimated Time**: 1h
**Files**: USER_GUIDE.md
**LOC**: ~600 LOC

---

## 📋 Summary

### Phase Breakdown

| Phase | Tasks | Estimated Time | LOC |
|-------|-------|----------------|-----|
| Phase 0: Planning | 3 | 2h | 2,000 |
| Phase 1: Core Engine | 8 | 2-3h | 900 |
| Phase 2: Functions | 7 | 2-3h | 1,250 |
| Phase 3: Integration | 5 | 2-3h | 750 |
| Phase 4: Testing | 5 | 2-3h | 1,550 |
| Phase 5: Observability | 4 | 1-2h | 1,350 |
| **TOTAL** | **32** | **11-16h** | **7,800** |

### Deliverables

**Production Code**:
- Template engine core (900 LOC)
- Template functions (1,250 LOC)
- Receiver integration (750 LOC)
- Metrics & logging (250 LOC)
- **Total**: 3,150 LOC

**Test Code**:
- Unit tests (1,100 LOC)
- Integration tests (300 LOC)
- Benchmarks (150 LOC)
- **Total**: 1,550 LOC

**Documentation**:
- requirements.md (750 LOC)
- design.md (1,200 LOC)
- tasks.md (900 LOC)
- README.md (500 LOC)
- USER_GUIDE.md (600 LOC)
- **Total**: 3,950 LOC

**Grand Total**: 8,650 LOC

### Quality Targets

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test Coverage | ≥ 90% | go test -cover |
| Unit Tests | ≥ 30 | Test count |
| Template Functions | ≥ 50 | Function count |
| Parse Time | < 10ms p95 | Benchmark |
| Execute Time (cached) | < 5ms p95 | Benchmark |
| Execute Time (uncached) | < 20ms p95 | Benchmark |
| Cache Hit Ratio | > 95% | Prometheus |
| Documentation | 100% | Manual review |
| Linter Errors | 0 | golangci-lint |

---

## 🎯 Success Criteria

- ✅ All 50+ template functions implemented
- ✅ Template engine integrated with all receivers
- ✅ 30+ unit tests passing
- ✅ 90%+ test coverage
- ✅ Performance targets met
- ✅ Zero linter errors
- ✅ Comprehensive documentation
- ✅ Backward compatibility maintained

---

## 🚀 Next Steps

1. **Phase 1**: Implement core template engine (2-3h)
2. **Phase 2**: Implement template functions (2-3h)
3. **Phase 3**: Integrate with receivers (2-3h)
4. **Phase 4**: Write tests (2-3h)
5. **Phase 5**: Add observability & docs (1-2h)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-22
**Author**: AI Assistant
**Status**: ✅ APPROVED - Ready for Phase 1 Implementation
