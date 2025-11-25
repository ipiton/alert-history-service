# TN-155: Template API (CRUD) - Requirements

**Task ID**: TN-155
**Sprint**: Sprint 3 (Week 3) - Config & Templates
**Priority**: P1 (High)
**Complexity**: Medium-High
**Estimate**: 16-20 hours
**Quality Target**: 150% (Grade A+ EXCEPTIONAL)
**Dependencies**: TN-153 ✅, TN-154 ✅
**Date**: 2025-11-25

---

## 📋 Overview

### Mission Statement

Создать **enterprise-grade REST API для управления notification templates** с полной поддержкой CRUD операций, версионирования, валидации через TN-153 Template Engine, и интеграцией с TN-154 Default Templates.

### Business Value

1. **Self-Service**: Пользователи могут создавать custom templates без изменения кода
2. **Version Control**: Полная история изменений с возможностью rollback
3. **Quality Assurance**: Автоматическая валидация перед deployment
4. **Operational Excellence**: Управление templates через REST API, не через файлы

### Goals

1. ✅ **CRUD Operations**: Create, Read, Update, Delete templates
2. ✅ **Validation**: Syntax + semantic validation через TN-153
3. ✅ **Versioning**: Full history с rollback capabilities
4. ✅ **Storage**: PostgreSQL persistence с indexes
5. ✅ **Caching**: Two-tier cache (L1 memory + L2 Redis)
6. ✅ **Security**: Admin-only access + audit logging
7. ✅ **Performance**: < 10ms p95 GET latency
8. ✅ **Observability**: 10+ Prometheus metrics
9. ✅ **150% Quality**: Advanced features (batch ops, diff, analytics)

---

## 🎯 Functional Requirements

### FR-1: Template CRUD Operations (Priority: P0 - Critical)

#### FR-1.1: Create Template

**Endpoint**: `POST /api/v2/templates`

**Request Body**:
```json
{
  "name": "slack_critical_alert",
  "type": "slack",
  "content": "{{ .Status | toUpper }}: {{ .GroupLabels.alertname }}",
  "description": "Critical alert notification for Slack",
  "metadata": {
    "author": "platform-team",
    "tags": ["slack", "critical", "production"],
    "version": "1.0.0"
  }
}
```

**Validation Rules**:
- ✅ `name`: Required, 3-64 chars, alphanumeric + underscore, unique, lowercase
- ✅ `type`: Required, enum (slack, pagerduty, email, webhook, generic)
- ✅ `content`: Required, non-empty, max 64KB, valid Go template syntax
- ✅ `description`: Optional, max 500 chars
- ✅ `metadata`: Optional, JSON object, max 10 fields, max 2KB total

**Response** (201 Created):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "slack_critical_alert",
  "type": "slack",
  "content": "{{ .Status | toUpper }}: {{ .GroupLabels.alertname }}",
  "description": "Critical alert notification for Slack",
  "metadata": {
    "author": "platform-team",
    "tags": ["slack", "critical", "production"],
    "version": "1.0.0"
  },
  "version": 1,
  "created_at": "2025-11-25T10:00:00Z",
  "updated_at": "2025-11-25T10:00:00Z",
  "created_by": "admin@example.com"
}
```

**Error Responses**:
- `400 Bad Request`: Invalid input (missing required fields, validation errors)
- `409 Conflict`: Template with same name already exists
- `422 Unprocessable Entity`: Template syntax invalid (TN-153 validation failed)
- `500 Internal Server Error`: Database error

**Acceptance Criteria**:
- [x] Request validation with detailed error messages
- [x] Duplicate name detection
- [x] Template syntax validation via TN-153
- [x] Database persistence with transaction
- [x] Cache invalidation after creation
- [x] Audit logging (who created, when)
- [x] Prometheus metric: `template_api_creates_total{type, status}`

---

#### FR-1.2: List Templates

**Endpoint**: `GET /api/v2/templates`

**Query Parameters**:
```
?type=slack                 # Filter by type
&tag=critical              # Filter by metadata tag
&search=alert              # Search in name/description
&limit=50                  # Page size (default 50, max 200)
&offset=0                  # Pagination offset
&sort=name                 # Sort field (name, created_at, updated_at)
&order=asc                 # Sort order (asc, desc)
&include_deleted=false     # Include soft-deleted templates
```

**Response** (200 OK):
```json
{
  "templates": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "slack_critical_alert",
      "type": "slack",
      "description": "Critical alert notification",
      "version": 3,
      "created_at": "2025-11-25T10:00:00Z",
      "updated_at": "2025-11-25T11:30:00Z",
      "created_by": "admin@example.com"
    }
  ],
  "pagination": {
    "total": 142,
    "limit": 50,
    "offset": 0,
    "has_more": true
  }
}
```

**Performance Targets**:
- Baseline (100%): < 200ms p95
- Target (150%): < 50ms p95 (with caching)

**Acceptance Criteria**:
- [x] Filtering by type, tag, search query
- [x] Pagination support (limit/offset)
- [x] Sorting by multiple fields
- [x] Cache headers (ETag, Cache-Control)
- [x] Performance: < 50ms p95 with cache

---

#### FR-1.3: Get Single Template

**Endpoint**: `GET /api/v2/templates/{name}`

**Response** (200 OK):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "name": "slack_critical_alert",
  "type": "slack",
  "content": "{{ .Status | toUpper }}: {{ .GroupLabels.alertname }}",
  "description": "Critical alert notification for Slack",
  "metadata": {...},
  "version": 3,
  "created_at": "2025-11-25T10:00:00Z",
  "updated_at": "2025-11-25T11:30:00Z",
  "created_by": "admin@example.com",
  "updated_by": "admin@example.com"
}
```

**HTTP Headers**:
```
ETag: "abc123def456"
Cache-Control: max-age=300, must-revalidate
X-Template-Version: 3
```

**Error Responses**:
- `304 Not Modified`: If-None-Match matches ETag (client cache valid)
- `404 Not Found`: Template not found

**Performance Targets**:
- Baseline (100%): < 100ms p95
- Target (150%): < 10ms p95 (cached)

**Acceptance Criteria**:
- [x] Efficient cache lookup (L1 → L2 → DB)
- [x] ETag generation (hash of content)
- [x] Conditional request support (If-None-Match)
- [x] Performance: < 10ms p95 cached

---

#### FR-1.4: Update Template

**Endpoint**: `PUT /api/v2/templates/{name}`

**Request Body** (partial update supported):
```json
{
  "content": "{{ .Status | toUpper }}: {{ .GroupLabels.alertname }} [UPDATED]",
  "description": "Updated description",
  "metadata": {
    "version": "1.1.0",
    "changelog": "Fixed severity mapping"
  }
}
```

**Behavior**:
- ✅ Creates new version (increments version number)
- ✅ Validates new content via TN-153
- ✅ Preserves old version in `template_versions` table
- ✅ Invalidates all caches
- ✅ Logs audit trail

**Response** (200 OK): Full template with new version

**Error Responses**:
- `404 Not Found`: Template not found
- `409 Conflict`: Concurrent modification (ETag mismatch)
- `422 Unprocessable Entity`: Validation failed

**Acceptance Criteria**:
- [x] Version increment automatic
- [x] Old version preserved in history
- [x] Optimistic locking (If-Match header)
- [x] Content validation before save
- [x] Cache invalidation (L1 + L2)

---

#### FR-1.5: Delete Template

**Endpoint**: `DELETE /api/v2/templates/{name}`

**Query Parameters**:
```
?hard_delete=false         # Soft delete by default
&force=false               # Force delete even if in use
```

**Behavior**:
- ✅ **Soft delete** (default): Set `deleted_at` timestamp
- ✅ **Hard delete** (`?hard_delete=true`): Physically remove from DB
- ✅ **Protection**: Prevent deletion if template is referenced in active configs
- ✅ **Force delete** (`?force=true`): Override protection (admin only)

**Response** (204 No Content)

**Error Responses**:
- `404 Not Found`: Template not found
- `409 Conflict`: Template in use, cannot delete

**Acceptance Criteria**:
- [x] Soft delete preserves data
- [x] Reference check before deletion
- [x] Force delete option for admins
- [x] Audit logging

---

### FR-2: Template Validation (Priority: P0 - Critical)

#### FR-2.1: Validate Template Syntax

**Endpoint**: `POST /api/v2/templates/validate`

**Request Body**:
```json
{
  "content": "{{ .Status | toUpper }}: {{ .GroupLabels.alertname }}",
  "type": "slack",
  "test_data": {
    "Status": "firing",
    "GroupLabels": {
      "alertname": "HighCPU"
    }
  }
}
```

**Response** (200 OK) - Valid:
```json
{
  "valid": true,
  "syntax_errors": [],
  "warnings": [],
  "functions_used": ["toUpper"],
  "variables_used": ["Status", "GroupLabels.alertname"],
  "rendered_output": "FIRING: HighCPU"
}
```

**Response** (422 Unprocessable Entity) - Invalid:
```json
{
  "valid": false,
  "syntax_errors": [
    {
      "line": 1,
      "column": 15,
      "message": "unknown function: toUpperCase",
      "suggestion": "Did you mean 'toUpper'?"
    }
  ],
  "warnings": [
    "Variable 'GroupLabels.severity' referenced but not in test data"
  ]
}
```

**Validation Steps**:
1. Parse template with TN-153 `NotificationTemplateEngine`
2. Execute with test data (or mock data)
3. Collect errors, warnings, metadata
4. Return detailed feedback

**Acceptance Criteria**:
- [x] Integration with TN-153 engine
- [x] Line/column error reporting
- [x] Function suggestions (fuzzy matching)
- [x] Variable usage analysis
- [x] Performance: < 20ms p95

---

### FR-3: Template Version Control (Priority: P1 - High)

#### FR-3.1: List Template Versions

**Endpoint**: `GET /api/v2/templates/{name}/versions`

**Query Parameters**:
```
?limit=20                  # Default 20, max 100
&offset=0
```

**Response** (200 OK):
```json
{
  "versions": [
    {
      "version": 3,
      "created_at": "2025-11-25T11:30:00Z",
      "created_by": "admin@example.com",
      "change_summary": "Updated severity mapping",
      "content_hash": "abc123"
    },
    {
      "version": 2,
      "created_at": "2025-11-25T10:45:00Z",
      "created_by": "admin@example.com",
      "change_summary": "Fixed typo in alert name"
    }
  ],
  "total": 3
}
```

**Acceptance Criteria**:
- [x] List all versions with metadata
- [x] Pagination support
- [x] Performance: < 50ms p95

---

#### FR-3.2: Get Specific Version

**Endpoint**: `GET /api/v2/templates/{name}/versions/{version}`

**Response** (200 OK): Full template at that version (including content)

**Acceptance Criteria**:
- [x] Retrieve historical content
- [x] All metadata preserved
- [x] Performance: < 100ms p95

---

#### FR-3.3: Rollback Template

**Endpoint**: `POST /api/v2/templates/{name}/rollback`

**Request Body**:
```json
{
  "version": 2,
  "reason": "Version 3 caused notification failures in production"
}
```

**Behavior**:
- ✅ Creates **new version** with content from version 2
- ✅ Does NOT delete version 3 (preserves history)
- ✅ Logs rollback in audit trail
- ✅ Invalidates caches

**Response** (200 OK): New template (e.g., version 4 with content from version 2)

**Acceptance Criteria**:
- [x] Preserve all history
- [x] Create new version (not revert)
- [x] Audit logging with reason

---

### FR-4: Advanced Features (150% Quality)

#### FR-4.1: Batch Operations

**Endpoint**: `POST /api/v2/templates/batch`

**Request Body**:
```json
{
  "operation": "create",
  "templates": [
    {...},
    {...}
  ]
}
```

**Operations**: `create`, `update`, `delete`

**Acceptance Criteria**:
- [x] Atomic batch (all or nothing)
- [x] Validation all templates before commit
- [x] Performance: < 100ms per template

---

#### FR-4.2: Template Diff

**Endpoint**: `GET /api/v2/templates/{name}/diff?from=v1&to=v2`

**Response**: Unified diff format

**Acceptance Criteria**:
- [x] Line-by-line comparison
- [x] Visual markers (+/-)

---

#### FR-4.3: Template Analytics

**Endpoint**: `GET /api/v2/templates/stats`

**Response**:
```json
{
  "total_templates": 142,
  "by_type": {
    "slack": 50,
    "pagerduty": 40,
    "email": 30,
    "webhook": 20,
    "generic": 2
  },
  "most_used": [
    {"name": "slack_critical", "usage_count": 1500}
  ],
  "validation_error_rate": 0.05
}
```

---

### FR-5: Template Testing

**Endpoint**: `POST /api/v2/templates/{name}/test`

**Request Body**:
```json
{
  "alert": {
    "labels": {...},
    "annotations": {...}
  }
}
```

**Response**: Rendered output + performance metrics

---

## 📏 Non-Functional Requirements

### NFR-1: Performance (Priority: P0)

| Operation | Baseline (100%) | Target (150%) |
|-----------|-----------------|---------------|
| GET (cached) | < 50ms p95 | < 10ms p95 ✅ |
| GET (uncached) | < 200ms p95 | < 100ms p95 ✅ |
| POST | < 100ms p95 | < 50ms p95 ✅ |
| PUT | < 150ms p95 | < 75ms p95 ✅ |
| DELETE | < 100ms p95 | < 50ms p95 ✅ |
| Validate | < 50ms p95 | < 20ms p95 ✅ |
| Cache Hit Ratio | > 80% | > 90% ✅ |
| Throughput | > 500 req/s | > 1000 req/s ✅ |

**Measurement**: Benchmarks + k6 load tests

---

### NFR-2: Scalability (Priority: P1)

**Horizontal Scaling**:
- ✅ Stateless handlers (can run multiple replicas)
- ✅ Distributed caching (Redis)
- ✅ Connection pooling (PostgreSQL)

**Capacity Targets**:
- Baseline (100%): 10,000 templates, 100 concurrent users
- Target (150%): 100,000 templates, 500 concurrent users

---

### NFR-3: Security (Priority: P0)

**Authentication & Authorization**:
- ✅ All endpoints require authentication
- ✅ Admin role required for mutations (POST/PUT/DELETE)
- ✅ Read-only users can GET/validate

**Input Validation**:
- ✅ SQL injection prevention (parameterized queries)
- ✅ XSS prevention (input sanitization)
- ✅ Template injection prevention (TN-153 sandboxing)
- ✅ Content size limits (64KB max)

**Audit Trail**:
- ✅ Log all mutations (who, what, when)
- ✅ Store in separate audit table
- ✅ Immutable audit logs

**Rate Limiting**:
- ✅ 100 requests/minute per user (read)
- ✅ 20 requests/minute per user (write)

---

### NFR-4: Observability (Priority: P0)

**Prometheus Metrics** (10+ metrics):
```
template_api_requests_total{method, endpoint, status}
template_api_duration_seconds{method, endpoint}
template_api_validation_errors_total{type, error_code}
template_api_cache_hit_ratio
template_api_cache_size_bytes
template_versions_total{template_name}
template_storage_size_bytes
template_creates_total{type}
template_updates_total{type}
template_deletes_total{type, soft_hard}
```

**Structured Logging** (slog):
- ✅ Request ID tracking
- ✅ User ID tracking
- ✅ DEBUG/INFO/WARN/ERROR levels
- ✅ JSON format for parsing

---

### NFR-5: Reliability (Priority: P0)

**Data Durability**:
- ✅ PostgreSQL persistence
- ✅ Transaction support (ACID)
- ✅ Backups (automated)

**Error Handling**:
- ✅ Graceful degradation (cache failures)
- ✅ Retry logic (database)
- ✅ Circuit breaker (optional)

**Availability**:
- ✅ 99.9% uptime target
- ✅ Health check endpoint

---

## ✅ Acceptance Criteria (Master Checklist)

### Implementation
- [ ] All 7 baseline endpoints working
- [ ] 3+ advanced features (batch, diff, analytics)
- [ ] Database migrations applied
- [ ] Cache layer functional

### Testing
- [ ] 80%+ code coverage
- [ ] 30+ unit tests passing
- [ ] 5+ integration tests passing
- [ ] Benchmarks validate performance targets

### Performance
- [ ] < 10ms p95 GET (cached)
- [ ] > 90% cache hit ratio
- [ ] > 1000 req/s throughput

### Security
- [ ] RBAC enforced (admin-only mutations)
- [ ] Input validation comprehensive
- [ ] Audit logging complete

### Observability
- [ ] 10+ Prometheus metrics
- [ ] Structured logging (slog)
- [ ] Health check endpoint

### Documentation
- [ ] OpenAPI 3.0 spec complete
- [ ] README with examples
- [ ] Migration guide

---

## 🚫 Out of Scope

- ❌ Template approval workflow (Phase 2)
- ❌ Visual template editor (future UI task)
- ❌ Template marketplace (public sharing)
- ❌ Template A/B testing (future analytics)
- ❌ Multi-tenancy (single org for MVP)

---

## 📦 Dependencies

### Required (Blocking)
✅ **TN-153**: Template Engine (validation)
✅ **TN-154**: Default Templates (seeding)
✅ **TN-012**: PostgreSQL Pool
✅ **TN-016**: Redis Cache
✅ **TN-020**: Structured Logging
✅ **TN-021**: Prometheus Metrics

### Optional (Non-blocking)
⚠️ **TN-150**: Config API (hot reload)
⚠️ **TN-137-141**: Routing Engine (consumers)

---

## 🎯 Success Metrics

### Quantitative
- [ ] 150/100 quality score
- [ ] Grade A+ certification
- [ ] Zero P0 bugs in production
- [ ] < 1% error rate

### Qualitative
- [ ] Positive user feedback
- [ ] Easy to use API
- [ ] Clear error messages
- [ ] Comprehensive documentation

---

**Status**: ✅ Requirements COMPLETE
**Next**: Technical Design Document
**Author**: AI Assistant
**Date**: 2025-11-25
