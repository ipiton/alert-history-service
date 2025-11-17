# TN-69: GET /publishing/stats - Statistics - Requirements

**Version**: 1.0
**Date**: 2025-11-17
**Status**: Requirements Complete ✅
**Quality Target**: 150%+ (Grade A+, Enterprise-Grade)
**Branch**: `feature/TN-69-publishing-stats-endpoint-150pct`

---

## 📋 Table of Contents

1. [Executive Summary](#executive-summary)
2. [Business Requirements](#business-requirements)
3. [Functional Requirements](#functional-requirements)
4. [Non-Functional Requirements](#non-functional-requirements)
5. [Technical Requirements](#technical-requirements)
6. [Dependencies](#dependencies)
7. [Constraints](#constraints)
8. [Acceptance Criteria](#acceptance-criteria)
9. [Success Metrics](#success-metrics)

---

## 1. Executive Summary

### 1.1 Purpose

Формализовать, задокументировать и улучшить до 150%+ качества существующий API endpoint **GET /publishing/stats**, который предоставляет агрегированную статистику Publishing System для мониторинга и операционного управления.

### 1.2 Context

**Критическое открытие**: Эндпоинт `GET /api/v2/publishing/stats` **уже реализован** как часть задачи **TN-057 (Publishing Metrics & Stats)**, который был завершён на уровне 150%+ quality. Однако:
- ❌ Отсутствует отдельная документация для TN-69
- ❌ Нет API v1 версии для backward compatibility (только v2)
- ❌ Пробелы в security hardening (rate limiting, headers)
- ❌ Пробелы в testing coverage (security tests, load tests)
- ❌ Отсутствует HTTP caching (ETag, Cache-Control)
- ❌ Нет query parameters для фильтрации (filter, group_by)
- ❌ Задача не отмечена как complete в tasks.md

### 1.3 Scope

**In Scope**:
- Документирование существующего API v2 endpoint
- Добавление API v1 endpoint для backward compatibility (`/api/v1/publishing/stats`)
- Request validation и error handling
- HTTP caching (Cache-Control, ETag)
- Rate limiting (100 req/min)
- Security headers (9 headers, OWASP compliant)
- Query parameters (filter, group_by, format)
- Comprehensive testing (90%+ coverage)
- Performance optimization (P95 < 5ms)
- API documentation (OpenAPI 3.0.3)
- 150% Quality Certification (Grade A+)

**Out of Scope**:
- Изменение логики MetricsCollector (уже реализовано в TN-057)
- Historical data aggregation (beyond current snapshot)
- Real-time streaming (SSE/WebSocket)
- Multi-region aggregation

### 1.4 Stakeholders

- **Primary**: DevOps Team, Platform Team, SRE Team
- **Secondary**: Monitoring Team, Frontend Team, Analytics Team
- **End Users**: Operations engineers, Monitoring tools, Dashboards

### 1.5 Business Value

- **Operational Visibility**: Real-time статистика publishing system
- **Incident Response**: Быстрая диагностика проблем и bottlenecks
- **Automation**: Programmatic access для CI/CD, monitoring, alerting
- **Compliance**: Audit trail через metrics и logs
- **Integration**: API для frontend dashboard, external systems, Grafana

---

## 2. Business Requirements

### BR-001: API Endpoint Availability
**Priority**: Critical
**Description**: API endpoint должен быть доступен 24/7 с high availability.

**Rationale**: Операторы и мониторинг системы зависят от этого endpoint для проверки статуса.

**Success Criteria**:
- Endpoint доступен с uptime 99.9%+
- Response time P95 < 5ms
- Graceful handling при проблемах с dependencies

### BR-002: Real-Time Statistics
**Priority**: Critical
**Description**: Endpoint должен предоставлять актуальную статистику без задержки.

**Rationale**: Устаревшая статистика может привести к неправильным operational decisions.

**Success Criteria**:
- Statistics актуальны (latency < 1s от actual change)
- Cache invalidation при significant changes
- Metrics включают timestamp

### BR-003: API Consistency
**Priority**: High
**Description**: Endpoint должен быть консистентен с другими publishing endpoints (TN-66, TN-67, TN-68).

**Rationale**: Консистентность упрощает интеграцию и поддержку.

**Success Criteria**:
- API v1 и v2 версии доступны
- Response format консистентен с другими endpoints
- Error handling единообразен

### BR-004: Performance
**Priority**: High
**Description**: Endpoint должен отвечать быстро даже под нагрузкой.

**Rationale**: Медленные ответы блокируют мониторинг и dashboards.

**Success Criteria**:
- P50 < 2ms
- P95 < 5ms
- P99 < 10ms
- Throughput > 10,000 req/s

### BR-005: Security
**Priority**: High
**Description**: Endpoint должен быть защищён от злоупотреблений и атак.

**Rationale**: Публичный endpoint может быть целью атак.

**Success Criteria**:
- Rate limiting (100 req/min per IP)
- Security headers (9 headers, OWASP compliant)
- Input validation
- No sensitive data exposure

---

## 3. Functional Requirements

### FR-1: GET /api/v2/publishing/stats (Primary Endpoint)
**Priority**: Critical
**Description**: Возвращает агрегированную статистику Publishing System.

**Details**:
- **Method**: GET
- **Path**: `/api/v2/publishing/stats`
- **Query Parameters**:
  - `filter` (optional): Filter by target type (rootly, slack, pagerduty, webhook)
  - `group_by` (optional): Group by field (target, type, status)
  - `format` (optional): Response format (json, prometheus) - default: json
- **Response Format (JSON)**:
```json
{
  "timestamp": "2025-11-17T10:30:00Z",
  "system": {
    "total_targets": 10,
    "healthy_targets": 8,
    "unhealthy_targets": 2,
    "success_rate_percent": 95.5,
    "queue_size": 15,
    "queue_capacity": 1000
  },
  "target_stats": {
    "targets_by_type": {
      "rootly": 5,
      "slack": 3,
      "pagerduty": 2
    },
    "targets_by_status": {
      "healthy": 8,
      "degraded": 1,
      "unhealthy": 2
    }
  },
  "queue_stats": {
    "size": 15,
    "capacity": 1000,
    "utilization_percent": 1.5,
    "workers_active": 5,
    "workers_idle": 5
  },
  "job_stats": {
    "total_submitted": 10000,
    "total_completed": 9500,
    "total_failed": 500,
    "success_rate_percent": 95.0
  }
}
```

**Acceptance Criteria**:
- [ ] Returns 200 OK with valid JSON
- [ ] Includes all required fields
- [ ] Handles empty metrics gracefully
- [ ] Supports query parameters
- [ ] Returns 400 Bad Request for invalid parameters
- [ ] Returns 500 Internal Server Error for collection failures

### FR-2: GET /api/v1/publishing/stats (Backward Compatibility)
**Priority**: Medium
**Description**: Возвращает статистику в формате совместимом с legacy API.

**Details**:
- **Method**: GET
- **Path**: `/api/v1/publishing/stats`
- **Response Format**: Simplified version of v2 response
- **Rationale**: Обеспечивает backward compatibility с существующими интеграциями

**Acceptance Criteria**:
- [ ] Returns 200 OK with valid JSON
- [ ] Response format совместим с legacy API
- [ ] All required fields присутствуют

### FR-3: Query Parameters Support
**Priority**: Medium
**Description**: Поддержка фильтрации и группировки через query parameters.

**Details**:
- `filter=type:rootly` - фильтр по типу target
- `filter=status:healthy` - фильтр по статусу
- `group_by=type` - группировка по типу
- `group_by=status` - группировка по статусу
- `format=prometheus` - экспорт в Prometheus format

**Acceptance Criteria**:
- [ ] Filtering работает корректно
- [ ] Grouping работает корректно
- [ ] Invalid parameters возвращают 400 Bad Request
- [ ] Performance не деградирует при использовании параметров

### FR-4: HTTP Caching
**Priority**: Medium
**Description**: Поддержка HTTP caching для снижения нагрузки.

**Details**:
- Cache-Control header (max-age=5s)
- ETag header (based on metrics hash)
- 304 Not Modified response

**Acceptance Criteria**:
- [ ] Cache-Control header присутствует
- [ ] ETag header присутствует
- [ ] 304 Not Modified возвращается для unchanged data
- [ ] Cache invalidation работает корректно

---

## 4. Non-Functional Requirements

### NFR-1: Performance
**Priority**: Critical
**Description**: Endpoint должен отвечать быстро.

**Targets**:
- P50: < 2ms
- P95: < 5ms
- P99: < 10ms
- Throughput: > 10,000 req/s
- Memory: < 10MB per request

**Measurement**:
- Benchmarks (go test -bench)
- Load testing (k6/wrk)
- Production metrics (Prometheus)

### NFR-2: Reliability
**Priority**: Critical
**Description**: Endpoint должен быть надёжным.

**Targets**:
- Uptime: 99.9%+
- Error rate: < 0.1%
- Graceful degradation при проблемах с collectors

**Measurement**:
- Production monitoring
- Error tracking
- Health checks

### NFR-3: Security
**Priority**: High
**Description**: Endpoint должен быть защищён.

**Requirements**:
- Rate limiting (100 req/min per IP)
- Security headers (9 headers, OWASP compliant)
- Input validation
- No sensitive data exposure
- Audit logging

**Measurement**:
- Security audit
- Penetration testing
- OWASP Top 10 compliance

### NFR-4: Observability
**Priority**: High
**Description**: Endpoint должен быть наблюдаемым.

**Requirements**:
- Structured logging (slog)
- Prometheus metrics (requests, duration, errors)
- Distributed tracing (Request ID)
- Performance metrics

**Measurement**:
- Log analysis
- Metrics dashboards
- Tracing analysis

### NFR-5: Testability
**Priority**: High
**Description**: Endpoint должен быть тестируемым.

**Requirements**:
- Unit tests (90%+ coverage)
- Integration tests
- Security tests
- Performance benchmarks
- Load tests

**Measurement**:
- Test coverage reports
- Benchmark results
- Load test results

### NFR-6: Documentation
**Priority**: Medium
**Description**: Endpoint должен быть документирован.

**Requirements**:
- OpenAPI 3.0.3 specification
- API guide with examples
- Troubleshooting guide
- Integration examples

**Measurement**:
- Documentation completeness
- Example quality
- User feedback

---

## 5. Technical Requirements

### TR-1: Implementation Language
**Language**: Go 1.21+
**Rationale**: Консистентность с остальным проектом

### TR-2: HTTP Framework
**Framework**: net/http (standard library) + gorilla/mux
**Rationale**: Консистентность с остальными endpoints

### TR-3: Metrics Collection
**Interface**: `MetricsCollectorInterface` (from TN-057)
**Rationale**: Использование существующей инфраструктуры

### TR-4: Response Format
**Format**: JSON (default), Prometheus (optional)
**Rationale**: Гибкость для разных use cases

### TR-5: Error Handling
**Approach**: Structured errors with HTTP status codes
**Rationale**: Консистентность с остальными endpoints

### TR-6: Logging
**Library**: log/slog (structured logging)
**Rationale**: Консистентность с остальными endpoints

### TR-7: Testing
**Framework**: testing (standard library) + testify
**Rationale**: Консистентность с остальными endpoints

---

## 6. Dependencies

### Internal Dependencies
- **TN-057**: Publishing Metrics & Stats (MetricsCollector)
- **TN-060**: Metrics-Only Mode Fallback (ModeManager)
- **TN-066**: GET /publishing/targets (TargetDiscoveryManager)
- **TN-067**: POST /publishing/targets/refresh (RefreshManager)
- **TN-068**: GET /publishing/mode (ModeService)

### External Dependencies
- Go 1.21+
- gorilla/mux
- log/slog

### Infrastructure Dependencies
- Prometheus (metrics collection)
- Redis (optional, для distributed caching)

---

## 7. Constraints

### C-1: Backward Compatibility
**Constraint**: Должна быть сохранена совместимость с существующими интеграциями
**Impact**: Требуется поддержка API v1

### C-2: Performance
**Constraint**: Endpoint не должен замедлять систему
**Impact**: Требуется оптимизация и caching

### C-3: Security
**Constraint**: Endpoint публичный, требует защиты
**Impact**: Требуется rate limiting и security headers

### C-4: Resource Usage
**Constraint**: Endpoint не должен потреблять много ресурсов
**Impact**: Требуется оптимизация памяти и CPU

---

## 8. Acceptance Criteria

### AC-1: Functional Completeness
- [ ] GET /api/v2/publishing/stats возвращает корректную статистику
- [ ] GET /api/v1/publishing/stats возвращает корректную статистику
- [ ] Query parameters работают корректно
- [ ] HTTP caching работает корректно
- [ ] Error handling работает корректно

### AC-2: Performance
- [ ] P95 < 5ms
- [ ] P99 < 10ms
- [ ] Throughput > 10,000 req/s
- [ ] Memory < 10MB per request

### AC-3: Security
- [ ] Rate limiting работает
- [ ] Security headers присутствуют
- [ ] Input validation работает
- [ ] OWASP Top 10 compliant

### AC-4: Testing
- [ ] Unit tests: 90%+ coverage
- [ ] Integration tests: все сценарии покрыты
- [ ] Security tests: все уязвимости проверены
- [ ] Performance benchmarks: все targets достигнуты

### AC-5: Documentation
- [ ] OpenAPI 3.0.3 specification
- [ ] API guide с примерами
- [ ] Troubleshooting guide
- [ ] Integration examples

---

## 9. Success Metrics

### 9.1 Performance Metrics
- **P50 latency**: < 2ms ✅
- **P95 latency**: < 5ms ✅
- **P99 latency**: < 10ms ✅
- **Throughput**: > 10,000 req/s ✅
- **Error rate**: < 0.1% ✅

### 9.2 Quality Metrics
- **Test coverage**: > 90% ✅
- **Security score**: OWASP Top 10 compliant ✅
- **Documentation completeness**: 100% ✅
- **Code quality**: Grade A+ ✅

### 9.3 Business Metrics
- **Uptime**: > 99.9% ✅
- **User satisfaction**: > 95% ✅
- **Integration success rate**: > 99% ✅

---

## 10. Risk Assessment

### R-1: Performance Degradation
**Risk**: Высокая нагрузка может замедлить endpoint
**Mitigation**: HTTP caching, оптимизация кода, rate limiting
**Probability**: Medium
**Impact**: High

### R-2: Security Vulnerabilities
**Risk**: Публичный endpoint может быть целью атак
**Mitigation**: Rate limiting, security headers, input validation
**Probability**: Medium
**Impact**: High

### R-3: Backward Compatibility
**Risk**: Изменения могут сломать существующие интеграции
**Mitigation**: Поддержка API v1, версионирование
**Probability**: Low
**Impact**: Medium

---

## 11. Timeline

### Phase 0: Analysis (COMPLETE)
- ✅ Анализ текущего состояния
- ✅ Определение gaps
- ✅ Планирование улучшений

### Phase 1: Documentation (IN PROGRESS)
- [ ] requirements.md
- [ ] design.md
- [ ] tasks.md

### Phase 2: Implementation
- [ ] API v1 endpoint
- [ ] Query parameters
- [ ] HTTP caching
- [ ] Security hardening

### Phase 3: Testing
- [ ] Unit tests
- [ ] Integration tests
- [ ] Security tests
- [ ] Performance benchmarks

### Phase 4: Documentation
- [ ] OpenAPI specification
- [ ] API guide
- [ ] Troubleshooting guide

### Phase 5: Certification
- [ ] Quality certification
- [ ] Production readiness review
- [ ] Merge to main

**Estimated Total Time**: 8-12 hours

---

**Document Status**: ✅ Requirements Complete
**Next Steps**: Create design.md and tasks.md
