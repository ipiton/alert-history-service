# Migration Gaps Analysis

**Date**: 2025-01-09
**Purpose**: Identify Python features not yet implemented in Go
**Status**: 6 critical gaps identified

---

## Critical Gaps (🔴 HIGH Priority)

### GAP-1: Intelligent Proxy Endpoints
**Python**: `src/alert_history/api/proxy_endpoints.py` (~400 LOC)
**Go**: ❌ Not implemented
**Impact**: 🔴 **CRITICAL** - Core feature

#### What's Missing:
- `/webhook/proxy` endpoint - intelligent webhook proxying
- LLM-based routing decisions
- Dynamic target selection based on alert classification
- Fallback routing logic
- Performance monitoring for proxy decisions

#### Business Impact:
- **Users affected**: All users relying on intelligent routing
- **Feature loss**: Cannot automatically route alerts based on AI classification
- **Workaround**: Manual routing configuration (reduced intelligence)

#### Migration Plan:
**Assigned**: TN-41 to TN-45 (Webhook Processing)
**Timeline**: 1-2 weeks
**Complexity**: HIGH (complex LLM integration)
**Dependencies**: TN-33 (Classification Service) ✅ Complete

**Implementation Steps**:
1. Create `internal/api/handlers/proxy.go`
2. Integrate with classification service
3. Implement routing engine
4. Add fallback logic
5. Comprehensive tests

---

### GAP-2: Publishing System
**Python**: `src/alert_history/api/publishing_endpoints.py` (~350 LOC)
**Go**: ❌ Not implemented
**Impact**: 🔴 **CRITICAL** - Essential for delivery

#### What's Missing:
- `GET /publishing/targets` - list discovered targets
- `POST /publishing/targets/refresh` - refresh target discovery
- `GET /publishing/mode` - current publishing mode
- `GET /publishing/stats` - publishing statistics
- `POST /publishing/test/{target}` - test target connectivity

#### Business Impact:
- **Users affected**: All users publishing alerts
- **Feature loss**: No visibility into publishing targets
- **Workaround**: Direct database queries (poor UX)

#### Migration Plan:
**Assigned**: TN-46 to TN-60 (Publishing System)
**Timeline**: 2-3 weeks
**Complexity**: HIGH (K8s integration + multi-target)
**Dependencies**: None (can start immediately)

**Implementation Steps**:
1. K8s secrets discovery (TN-46)
2. Target manager (TN-47)
3. Multi-target publishing (TN-51 to TN-55)
4. API endpoints (TN-59)
5. Metrics collection (TN-57)

---

### GAP-3: Target Discovery Service
**Python**: `src/alert_history/services/target_discovery.py` (~250 LOC)
**Go**: ❌ Not implemented
**Impact**: 🔴 **CRITICAL** - Required for dynamic publishing

#### What's Missing:
- Kubernetes secrets discovery
- Label selector matching (`alert-history.io/publish-target=true`)
- Dynamic target refresh
- Target health checking
- RBAC permissions handling

#### Business Impact:
- **Users affected**: All users with dynamic targets
- **Feature loss**: Cannot discover new targets automatically
- **Workaround**: Static configuration (loses dynamic benefit)

#### Migration Plan:
**Assigned**: TN-46, TN-47, TN-48, TN-49
**Timeline**: 1 week
**Complexity**: MEDIUM (K8s API well-documented)
**Dependencies**: None

**Implementation Steps**:
1. Create `internal/infrastructure/discovery/k8s_client.go`
2. Implement label selectors
3. Periodic refresh mechanism
4. Health monitoring
5. Integration tests with k8s mock

---

### GAP-4: Alert Publisher Service
**Python**: `src/alert_history/services/alert_publisher.py` (~300 LOC)
**Go**: ❌ Not implemented
**Impact**: 🔴 **CRITICAL** - Core publishing logic

#### What's Missing:
- Multi-target parallel publishing
- Per-target retry logic
- Publishing queue management
- Target-specific formatting
- Error aggregation and reporting

#### Business Impact:
- **Users affected**: All alert recipients
- **Feature loss**: Alerts not delivered to downstream systems
- **Workaround**: None (blocks alert delivery)

#### Migration Plan:
**Assigned**: TN-56, TN-57, TN-58
**Timeline**: 1-2 weeks
**Complexity**: HIGH (concurrency + error handling)
**Dependencies**: TN-46 to TN-55 (target discovery + formatters)

**Implementation Steps**:
1. Create `internal/core/publishing/publisher.go`
2. Worker pool for parallel publishing
3. Redis-backed queue (TN-56)
4. Retry logic with exponential backoff
5. Metrics and monitoring (TN-57)

---

### GAP-5: Webhook Processor
**Python**: `src/alert_history/services/webhook_processor.py` (~300 LOC)
**Go**: Partially implemented in `cmd/server/handlers/webhook.go`
**Impact**: 🟡 **MEDIUM** - Enhanced processing

#### What's Missing:
- Complex alert transformation logic
- Multi-format support (Alertmanager, Prometheus, custom)
- Alert correlation
- Batch processing optimization
- Advanced validation rules

#### Business Impact:
- **Users affected**: Users with custom webhook formats
- **Feature loss**: Cannot handle complex webhook payloads
- **Workaround**: Basic webhook works, but limited formats

#### Migration Plan:
**Assigned**: TN-41 to TN-45
**Timeline**: 1 week
**Complexity**: MEDIUM (well-defined transformation logic)
**Dependencies**: None

**Implementation Steps**:
1. Expand `webhook.go` with format detection
2. Add transformation pipelines
3. Implement validation rules
4. Add batch processing
5. Comprehensive test suite

---

### GAP-6: Alert Formatter Service
**Python**: `src/alert_history/services/alert_formatter.py` (~200 LOC)
**Go**: ❌ Not implemented
**Impact**: 🟡 **MEDIUM** - Required for publishing

#### What's Missing:
- Rootly incident format
- PagerDuty event format
- Slack block format
- Generic webhook format
- Template-based formatting

#### Business Impact:
- **Users affected**: All users publishing to external systems
- **Feature loss**: Cannot format alerts for specific targets
- **Workaround**: Send raw alerts (poor integration quality)

#### Migration Plan:
**Assigned**: TN-51 to TN-55
**Timeline**: 3-5 days
**Complexity**: LOW (straightforward data transformation)
**Dependencies**: None (well-defined formats)

**Implementation Steps**:
1. Create `internal/infrastructure/formatting/` package
2. Implement Rootly formatter (TN-52)
3. Implement PagerDuty formatter (TN-53)
4. Implement Slack formatter (TN-54)
5. Generic webhook formatter (TN-55)
6. Unit tests for each format

---

## Medium Priority Gaps (🟡)

### GAP-7: Dashboard Endpoints
**Python**: `src/alert_history/api/dashboard_endpoints.py` (~300 LOC)
**Go**: ❌ Not implemented
**Impact**: 🟡 MEDIUM
**Assigned**: TN-76 to TN-85
**Timeline**: 1-2 weeks

**What's Missing**:
- `/dashboard` HTML page
- `/api/dashboard/overview` - stats overview
- `/api/dashboard/charts` - chart data
- `/api/dashboard/alerts/recent` - recent alerts
- Real-time updates (SSE/WebSocket)

**Decision**: Can keep Python version temporarily, migrate after critical features

---

### GAP-8: Enrichment API
**Python**: `src/alert_history/api/enrichment_endpoints.py` (~200 LOC)
**Go**: Partially in `internal/core/enrichment.go`
**Impact**: 🟡 MEDIUM
**Assigned**: Part of TN-34 (already 160% complete!)
**Timeline**: 3-5 days to complete API layer

**What's Missing**:
- `POST /enrichment/mode` - switch modes dynamically
- `GET /enrichment/stats` - enrichment statistics
- Mode validation API

**Decision**: Low priority, enrichment core works, API is nice-to-have

---

### GAP-9: Classification Endpoints
**Python**: `src/alert_history/api/classification_endpoints.py` (~150 LOC)
**Go**: ❌ Not implemented
**Impact**: 🟢 LOW
**Assigned**: TN-71 to TN-73
**Timeline**: 2-3 days

**What's Missing**:
- `GET /classification/stats` - LLM usage statistics
- `POST /classification/classify` - manual classification
- `GET /classification/models` - available models

**Decision**: Nice-to-have, not blocking. Core classification works (TN-33 ✅)

---

## Low Priority (🟢)

### GAP-10: Legacy Adapter
**Python**: `src/alert_history/api/legacy_adapter.py` (~250 LOC)
**Go**: N/A (temporary compatibility layer)
**Impact**: 🟢 LOW
**Decision**: Keep in Python temporarily, remove after full migration

---

### GAP-11: CLI Database Migration
**Python**: `src/alert_history/cli/database_migrate.py` (~100 LOC)
**Go**: ✅ `cmd/migrate/main.go` (Complete)
**Impact**: 🟢 NONE
**Decision**: Python version can be deleted

---

## Non-Gaps (Already Complete) ✅

The following are **NOT gaps** - Go implementation is complete:

- ✅ Configuration management (TN-19)
- ✅ Logging (TN-20)
- ✅ Database adapters (TN-12, TN-13)
- ✅ Migration system (TN-14)
- ✅ Redis cache (TN-16)
- ✅ Distributed locks (TN-17)
- ✅ Prometheus metrics (TN-21)
- ✅ Health endpoints (TN-22, TN-23)
- ✅ Alert domain models (TN-31)
- ✅ Alert storage (TN-32)
- ✅ Classification service (TN-33)
- ✅ Enrichment system (TN-34)
- ✅ Filtering engine (TN-35)
- ✅ Deduplication (TN-36)
- ✅ History repository (TN-37)

---

## Gap Priority Matrix

| Priority | Count | Examples | Timeline |
|----------|-------|----------|----------|
| 🔴 Critical | 4 | Proxy, Publishing, Discovery, Publisher | 1-3 weeks |
| 🟡 Medium | 3 | Dashboard, Enrichment API, Webhook | 1-2 weeks |
| 🟢 Low | 3 | Classification API, Legacy adapter | As needed |
| ✅ Complete | 18 | Core infrastructure, storage, classification | DONE |

---

## Migration Path Recommendations

### Phase 1 (Week 1-2): Critical Infrastructure
**Goal**: Enable basic alert delivery

1. **TN-46 to TN-49**: Target Discovery (K8s integration)
   - Blocks: Publishing system
   - Priority: 🔴 CRITICAL
   - Effort: 5 days

2. **TN-51 to TN-55**: Alert Formatters
   - Blocks: Publishing system
   - Priority: 🔴 CRITICAL
   - Effort: 3-5 days

3. **TN-56 to TN-58**: Publishing Core
   - Blocks: Alert delivery
   - Priority: 🔴 CRITICAL
   - Effort: 5-7 days

### Phase 2 (Week 3-4): Intelligent Processing
**Goal**: Enable AI-powered routing

4. **TN-41 to TN-45**: Webhook Processor + Proxy
   - Blocks: Intelligent routing
   - Priority: 🔴 CRITICAL
   - Effort: 7-10 days

### Phase 3 (Week 5-6): Enhanced Features
**Goal**: Full feature parity

5. **TN-76 to TN-85**: Dashboard (optional)
   - Can keep Python version
   - Priority: 🟡 MEDIUM
   - Effort: 7-10 days

6. **TN-71 to TN-75**: Enrichment + Classification APIs
   - Nice-to-have
   - Priority: 🟢 LOW
   - Effort: 5 days

---

## Impact if Gaps Not Filled

### Scenario 1: Deploy Go without filling gaps

**Consequences**:
- ✅ Health checks work
- ✅ Metrics collection works
- ✅ Basic webhook ingestion works
- ✅ Alert storage works
- ✅ Classification works
- ❌ **NO ALERT DELIVERY** (no publishing system)
- ❌ **NO INTELLIGENT ROUTING** (no proxy)
- ❌ **NO TARGET DISCOVERY** (static config only)

**Verdict**: Not production-ready without Critical gaps filled

---

### Scenario 2: Fill only Critical gaps (TN-41 to TN-60)

**Consequences**:
- ✅ Full alert delivery pipeline
- ✅ Intelligent routing
- ✅ Dynamic target discovery
- ✅ Multi-target publishing
- ⚠️ No dashboard (can use Python)
- ⚠️ Limited API visibility

**Verdict**: Production-ready for core use cases

---

### Scenario 3: Fill all gaps

**Consequences**:
- ✅ 100% feature parity with Python
- ✅ Can sunset Python completely
- ✅ Best user experience

**Verdict**: Ideal state, but not blocking

---

## Recommended Strategy

**Priority 1** (Blocking Python sunset):
- TN-46 to TN-60: Publishing System (3 weeks)
- TN-41 to TN-45: Intelligent Proxy (2 weeks)

**Priority 2** (Enhanced UX):
- TN-76 to TN-85: Dashboard (2 weeks)

**Priority 3** (Nice-to-have):
- TN-71 to TN-73: Classification API (3 days)

**Total time to Python sunset**: ~5 weeks for Priority 1

**Python Cleanup can proceed in parallel** with Gap filling:
- Move unused code to `legacy/`
- Clean up dependencies
- Update documentation
- Prepare for transition

---

**Conclusion**: 4 critical gaps must be filled before Python sunset. Estimated 5 weeks to complete. Python Cleanup can proceed in parallel to prepare infrastructure.

**Next Actions**:
1. Prioritize TN-46 to TN-60 (Publishing)
2. Start Python Cleanup (parallel work)
3. Create detailed implementation plan for gaps
4. Set realistic Python sunset date (5-8 weeks)
