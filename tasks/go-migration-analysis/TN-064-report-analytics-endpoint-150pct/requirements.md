# TN-064: GET /report - Requirements Specification

**Date**: 2025-11-16
**Status**: 📝 APPROVED
**Priority**: HIGH
**Target Quality**: 150% Enterprise Grade

---

## 1. ОБОСНОВАНИЕ ЗАДАЧИ

### Бизнес-цель
Пользователи Alert History Service нуждаются в быстром и удобном способе получения комплексной аналитики по алертам. В настоящее время для получения полной картины требуется делать 3-4 отдельных API-запроса:
- GET /history/stats - для общей статистики
- GET /history/top - для самых частых алертов
- GET /history/flapping - для флапающих алертов
- GET /history/recent - для последних событий

Это создает дополнительную нагрузку на сеть, увеличивает latency и усложняет интеграцию.

### Техническая цель
Реализовать единый эндпоинт **GET /api/v2/report**, который агрегирует данные из всех существующих аналитических методов в один комплексный отчет с оптимальной производительностью и удобством использования.

### Целевая аудитория
- **SRE Teams**: Мониторинг состояния алертинга
- **Platform Teams**: Анализ трендов и паттернов
- **Dashboard UIs**: Визуализация аналитики в реальном времени
- **Automated Reports**: Генерация периодических отчетов

---

## 2. ПОЛЬЗОВАТЕЛЬСКИЕ СЦЕНАРИИ

### Сценарий 1: Ежедневный отчет для SRE
**Актор**: SRE Engineer
**Цель**: Получить сводку за последние 24 часа
**Шаги**:
1. Отправляет GET-запрос: `/api/v2/report?from=2025-11-15T00:00:00Z&to=2025-11-16T00:00:00Z`
2. Получает JSON с summary, top 10 алертов, флапающими алертами
3. Анализирует данные и выявляет проблемные зоны
4. Принимает решение о необходимости оптимизации

**Ожидаемый результат**: Полный отчет получен за <100ms

### Сценарий 2: Фильтрация по namespace
**Актор**: Platform Team
**Цель**: Анализ алертов только production namespace
**Шаги**:
1. Отправляет запрос: `/api/v2/report?namespace=production&top=20`
2. Получает отчет только по production алертам
3. Сравнивает с предыдущими периодами

**Ожидаемый результат**: Отфильтрованный отчет с 20 топ алертами

### Сценарий 3: Dashboard с автообновлением
**Актор**: Grafana Dashboard
**Цель**: Отображение real-time аналитики
**Шаги**:
1. Каждые 30 секунд делает запрос: `/api/v2/report?from=now-1h`
2. Кэш возвращает результат за <10ms
3. Dashboard обновляется без задержек

**Ожидаемый результат**: 85%+ cache hit rate, <10ms latency

### Сценарий 4: Расследование incident
**Актор**: Incident Commander
**Цель**: Быстро понять ситуацию с алертами за последний час
**Шаги**:
1. Запрашивает: `/api/v2/report?from=now-1h&severity=critical`
2. Получает только critical алерты
3. Анализирует топ проблемных сервисов

**Ожидаемый результат**: Отчет готов за <50ms, только critical severity

---

## 3. ФУНКЦИОНАЛЬНЫЕ ТРЕБОВАНИЯ

### FR-1: Endpoint Path
**Требование**: Эндпоинт доступен по пути `/api/v2/report`
**Приоритет**: MUST HAVE
**Критерий приёмки**:
- ✅ GET /api/v2/report возвращает 200 OK
- ✅ Альтернативный путь /report работает как алиас (backward compatibility)

### FR-2: Query Parameters
**Требование**: Поддержка следующих параметров

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `from` | ISO8601 timestamp | NO | now-24h | Start time |
| `to` | ISO8601 timestamp | NO | now | End time |
| `namespace` | string | NO | all | Filter by namespace |
| `severity` | enum | NO | all | Filter by severity (critical\|warning\|info\|noise) |
| `top` | int | NO | 10 | Number of top alerts (1-100) |
| `min_flap` | int | NO | 3 | Minimum flapping transitions (1-100) |
| `include_recent` | bool | NO | false | Include recent alerts section |

**Критерий приёмки**:
- ✅ Все параметры валидируются
- ✅ Некорректные значения возвращают 400 Bad Request
- ✅ Значения по умолчанию применяются корректно

### FR-3: Response Format
**Требование**: JSON response с следующей структурой

```json
{
  "metadata": {
    "generated_at": "2025-11-16T10:30:00Z",
    "request_id": "req-abc123",
    "processing_time_ms": 45,
    "cache_hit": false,
    "partial_failure": false
  },
  "summary": {
    // AggregatedStats from GetAggregatedStats()
  },
  "top_alerts": [
    // TopAlert[] from GetTopAlerts()
  ],
  "flapping_alerts": [
    // FlappingAlert[] from GetFlappingAlerts()
  ],
  "recent_alerts": [
    // Optional: Alert[] from GetRecentAlerts()
  ]
}
```

**Критерий приёмки**:
- ✅ Response является valid JSON
- ✅ Все поля присутствуют согласно схеме
- ✅ Timestamps в RFC3339 format

### FR-4: Data Aggregation
**Требование**: Агрегация данных из 3-4 источников

**Источники данных**:
1. `GetAggregatedStats(ctx, timeRange)` → summary
2. `GetTopAlerts(ctx, timeRange, limit)` → top_alerts
3. `GetFlappingAlerts(ctx, timeRange, threshold)` → flapping_alerts
4. `GetRecentAlerts(ctx, limit)` → recent_alerts (optional)

**Критерий приёмки**:
- ✅ Все данные корректно агрегированы
- ✅ Filters применяются ко всем источникам
- ✅ Параллельное выполнение запросов (performance optimization)

### FR-5: Error Handling
**Требование**: Graceful error handling с partial failure tolerance

**Error Scenarios**:
- 400 Bad Request - невалидные параметры
- 401 Unauthorized - отсутствует/невалиден JWT token
- 403 Forbidden - недостаточно прав
- 429 Too Many Requests - превышен rate limit
- 500 Internal Server Error - database errors
- 504 Gateway Timeout - query timeout (>10s)

**Partial Failure Behavior**:
```json
{
  "metadata": {
    "partial_failure": true,
    "errors": ["Failed to retrieve flapping alerts: database timeout"]
  },
  "summary": { /* valid data */ },
  "top_alerts": [ /* valid data */ ],
  "flapping_alerts": []  // empty due to error
}
```

**Критерий приёмки**:
- ✅ Все error types обрабатываются корректно
- ✅ Partial failures возвращают 200 OK с metadata.partial_failure=true
- ✅ Error messages informative, не раскрывают sensitive данные

### FR-6: Time Range Validation
**Требование**: Валидация временных диапазонов

**Validation Rules**:
- `from` <= `to` (если оба указаны)
- Time range <= 90 days (prevent large queries)
- Timestamps должны быть valid RFC3339
- Future timestamps are allowed (для scheduled queries)

**Критерий приёмки**:
- ✅ Невалидные time ranges возвращают 400
- ✅ Error message указывает конкретную проблему
- ✅ Large time ranges (>90 days) rejected

### FR-7: Filtering Consistency
**Требование**: Filters применяются ко всем частям отчета

**Поведение**:
- Если указан `namespace=production`, все данные (summary, top, flapping) только для production
- Если указан `severity=critical`, фильтр применяется к summary и top_alerts
- Фильтры комбинируются логически (AND)

**Критерий приёмки**:
- ✅ Фильтры применяются консистентно
- ✅ Результаты соответствуют заданным фильтрам
- ✅ Нет данных вне фильтра

---

## 4. НЕ-ФУНКЦИОНАЛЬНЫЕ ТРЕБОВАНИЯ

### NFR-1: Performance
**Требование**: High performance для различных сценариев

| Scenario | Target Latency | Measurement |
|----------|---------------|-------------|
| Cache hit (L1) | <5ms | P95 |
| Cache hit (L2) | <10ms | P95 |
| Cache miss (fresh query) | <100ms | P95 |
| Large time range (7 days) | <200ms | P95 |
| Peak load (500 req/s) | <150ms | P95 |

**Критерий приёмки**:
- ✅ P50 latency <50ms (without cache)
- ✅ P95 latency <100ms (without cache)
- ✅ P99 latency <200ms (without cache)
- ✅ Throughput >500 req/s (single instance)

### NFR-2: Scalability
**Требование**: Горизонтальное масштабирование

**Capabilities**:
- Stateless design (no session storage)
- Distributed L2 cache (Redis)
- Database connection pooling
- No single point of failure

**Критерий приёмки**:
- ✅ Можно запустить 3+ instances без конфликтов
- ✅ L2 cache shared между instances
- ✅ Load balancing работает корректно

### NFR-3: Availability
**Требование**: High availability (99.9% uptime)

**Features**:
- Partial failure tolerance (graceful degradation)
- Database failover support
- Redis cluster support
- Health check endpoint

**Критерий приёмки**:
- ✅ Uptime >99.9% (measured over 30 days)
- ✅ Partial failures не приводят к 5xx errors
- ✅ Database failures handled gracefully

### NFR-4: Security
**Требование**: OWASP Top 10 compliance (100%)

**Security Controls**:
- JWT token validation (authentication)
- RBAC support (authorization)
- Input validation (injection prevention)
- Rate limiting (100 req/min per IP)
- Security headers (7 headers)
- Request size limits (max 1KB)
- Query timeout (10s max)

**Критерий приёмки**:
- ✅ All OWASP Top 10 vulnerabilities addressed
- ✅ Security audit passed (gosec, nancy)
- ✅ Penetration testing completed

### NFR-5: Observability
**Требование**: Comprehensive monitoring and logging

**Metrics**:
- 21 Prometheus metrics (request, processing, error, DB, resource, security)
- Grafana dashboard (7 panels)
- 10 alerting rules

**Logging**:
- Structured logging (JSON format)
- Request/response logging
- Error stack traces
- Audit trail (who, what, when)

**Критерий приёмки**:
- ✅ All metrics exposed on /metrics
- ✅ Grafana dashboard deployed
- ✅ Alerting rules configured
- ✅ Logs searchable in Loki/ELK

### NFR-6: Maintainability
**Требование**: High code quality and documentation

**Code Quality**:
- Test coverage >90%
- Cyclomatic complexity <10 per function
- Go Vet: 0 warnings
- golangci-lint: 0 errors
- Documentation coverage 100%

**Documentation**:
- OpenAPI 3.0 specification
- 3 Architecture Decision Records (ADRs)
- 3 Runbooks (troubleshooting guides)
- API integration guide (examples in 4 languages)

**Критерий приёмки**:
- ✅ All code quality metrics met
- ✅ All documentation complete
- ✅ Peer review approved

### NFR-7: Caching
**Требование**: 2-tier caching для optimal performance

**L1 Cache (Ristretto)**:
- In-memory cache
- TTL: 1 minute
- Max size: 1000 entries
- Hit rate: ~85%

**L2 Cache (Redis)**:
- Distributed cache
- TTL: 5 minutes
- Max size: 10000 entries
- Hit rate: ~93% (combined with L1)

**Cache Key Format**:
```
report:v1:{from}:{to}:{namespace}:{severity}:{topLimit}:{minFlap}
```

**Критерий приёмки**:
- ✅ L1 cache operational
- ✅ L2 cache operational
- ✅ Cache hit rate >85%
- ✅ Cache invalidation works correctly

---

## 5. ОГРАНИЧЕНИЯ И ЗАВИСИМОСТИ

### External Dependencies
| Dependency | Version | Required | Notes |
|-----------|---------|----------|-------|
| PostgreSQL | 14+ | YES | Database for alert storage |
| Redis | 7+ | YES | L2 cache |
| Prometheus | 2.40+ | YES | Metrics collection |
| Grafana | 9.0+ | YES | Dashboard visualization |

### Internal Dependencies
| Component | Status | Blocker |
|-----------|--------|---------|
| TN-038 Analytics Service | ✅ COMPLETE | NO |
| PostgresHistoryRepository | ✅ READY | NO |
| HistoryHandlerV2 | ✅ READY | NO |
| Core Types (TopAlert, FlappingAlert, AggregatedStats) | ✅ READY | NO |

### Resource Constraints
- Database connection pool: min 10 connections (для параллельных запросов)
- Memory: <50MB cache overhead
- CPU: <5% for cache operations
- Network: <1MB typical response size

### Operational Constraints
- Deployment: Kubernetes cluster (production)
- CI/CD: GitHub Actions (automated tests)
- Monitoring: Prometheus + Grafana stack
- Logging: JSON to stdout (collected by Loki)

---

## 6. КРИТЕРИИ ПРИЁМКИ

### Must Have (100% - Base Quality)
- ✅ GET /api/v2/report endpoint functional
- ✅ All query parameters работают
- ✅ Response format соответствует specification
- ✅ Error handling корректен
- ✅ Basic tests passed (unit + integration)
- ✅ Documentation создана (OpenAPI spec)

### Should Have (125% - Enhanced Quality)
- ✅ L1 cache (Ristretto) implemented
- ✅ Advanced filtering (namespace, severity)
- ✅ Prometheus metrics (10+ metrics)
- ✅ Security headers configured
- ✅ Performance benchmarks completed
- ✅ Grafana dashboard created

### Nice to Have (150% - Exceptional Quality)
- ✅ L2 cache (Redis) implemented
- ✅ Parallel query execution (3x faster)
- ✅ Partial failure tolerance
- ✅ Comprehensive metrics (21 metrics)
- ✅ Load testing (4 k6 scenarios)
- ✅ Complete documentation (OpenAPI + ADRs + Runbooks)
- ✅ OWASP Top 10 compliance (100%)
- ✅ Security audit passed
- ✅ 150% quality certification

---

## 7. ИСКЛЮЧЕНИЯ (OUT OF SCOPE)

### Not Included in TN-064
- ❌ Historical trend analysis (future task)
- ❌ PDF/CSV export (separate feature)
- ❌ Custom report templates (future enhancement)
- ❌ Email delivery (separate service)
- ❌ Slack notifications (use TN-059 Publishing API)
- ❌ Webhook callbacks (use TN-061)
- ❌ GraphQL API (REST only)

---

## 8. ПРИОРИТИЗАЦИЯ

### P0 (Critical - Week 1)
- Core implementation (GET /report handler)
- Basic filtering (time range)
- Response serialization
- Unit tests

### P1 (High - Week 1)
- Advanced filtering (namespace, severity)
- L1 cache implementation
- Integration tests
- Security validation

### P2 (Medium - Week 2)
- L2 cache (Redis)
- Parallel query execution
- Prometheus metrics
- Grafana dashboard

### P3 (Nice to Have - Week 2)
- Load testing (k6)
- Complete documentation (ADRs, Runbooks)
- Security audit
- 150% quality certification

---

## 9. RISKS & MITIGATION (повтор из PHASE0)

| Risk | Severity | Mitigation |
|------|----------|------------|
| DB Connection Pool Exhaustion | 🔴 HIGH | Validate pool size >= 10 |
| Cache Memory Pressure | 🟡 MEDIUM | Configure Ristretto max size |
| Timeout on Large Queries | 🟡 MEDIUM | Implement 10s timeout |
| Partial Data Misinterpretation | 🟡 MEDIUM | Add metadata.partial_failure field |

---

## 10. SUCCESS METRICS

### Implementation Success
- ✅ All acceptance criteria met (100%)
- ✅ Code review approved
- ✅ All tests passed (unit + integration + load)
- ✅ Documentation complete (OpenAPI + ADRs + Runbooks)

### Production Success (after 30 days)
- ✅ P95 latency <100ms
- ✅ Cache hit rate >85%
- ✅ Error rate <0.1%
- ✅ Uptime >99.9%
- ✅ Zero security incidents
- ✅ Positive user feedback

---

## APPENDIX A: API Contract

### Request Example
```bash
GET /api/v2/report?from=2025-11-15T00:00:00Z&to=2025-11-16T00:00:00Z&namespace=production&severity=critical&top=20&min_flap=5
```

### Response Example (200 OK)
```json
{
  "metadata": {
    "generated_at": "2025-11-16T10:30:45Z",
    "request_id": "req-abc123def456",
    "processing_time_ms": 45,
    "cache_hit": false,
    "partial_failure": false
  },
  "summary": {
    "time_range": {
      "from": "2025-11-15T00:00:00Z",
      "to": "2025-11-16T00:00:00Z"
    },
    "total_alerts": 1250,
    "firing_alerts": 45,
    "resolved_alerts": 1205,
    "unique_fingerprints": 150,
    "avg_resolution_time": "PT15M30S",
    "alerts_by_status": {
      "firing": 45,
      "resolved": 1205
    },
    "alerts_by_severity": {
      "critical": 12,
      "warning": 85,
      "info": 1153
    },
    "alerts_by_namespace": {
      "production": 850,
      "staging": 250,
      "development": 150
    }
  },
  "top_alerts": [
    {
      "fingerprint": "abc123def456",
      "alert_name": "CPUThrottlingHigh",
      "namespace": "production",
      "fire_count": 156,
      "last_fired_at": "2025-11-16T10:20:00Z",
      "avg_duration": 900.5
    }
  ],
  "flapping_alerts": [
    {
      "fingerprint": "def456ghi789",
      "alert_name": "DiskSpaceWarning",
      "namespace": "staging",
      "transition_count": 12,
      "flapping_score": 8.5,
      "last_transition_at": "2025-11-16T10:15:00Z"
    }
  ]
}
```

### Error Response Example (400 Bad Request)
```json
{
  "error": {
    "code": "INVALID_PARAMETER",
    "message": "Invalid time range: 'to' must be greater than or equal to 'from'",
    "request_id": "req-abc123",
    "timestamp": "2025-11-16T10:30:00Z",
    "details": {
      "field": "to",
      "value": "2025-11-14T00:00:00Z",
      "constraint": "to >= from"
    }
  }
}
```

---

**Status**: ✅ REQUIREMENTS APPROVED
**Sign-off**: Technical Lead, Product Owner
**Next Step**: Create design.md

---

**END OF REQUIREMENTS SPECIFICATION**
