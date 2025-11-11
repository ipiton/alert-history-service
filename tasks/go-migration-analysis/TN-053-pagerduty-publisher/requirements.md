# TN-053: PagerDuty Integration - Comprehensive Requirements (150% Quality)

**Version**: 1.0
**Date**: 2025-11-11
**Status**: 📋 **REQUIREMENTS PHASE**
**Quality Target**: **150%+ (Enterprise Grade A+)**
**Estimated Effort**: 10 days (80 hours)

---

## 📑 Table of Contents

1. [Executive Summary](#1-executive-summary)
2. [Business Value](#2-business-value)
3. [Functional Requirements](#3-functional-requirements)
4. [Non-Functional Requirements](#4-non-functional-requirements)
5. [PagerDuty Events API v2 Integration](#5-pagerduty-events-api-v2-integration)
6. [Dependencies](#6-dependencies)
7. [Risk Assessment](#7-risk-assessment)
8. [Acceptance Criteria](#8-acceptance-criteria)
9. [Success Metrics](#9-success-metrics)

---

## 1. Executive Summary

### 1.1 Overview

TN-053 transforms the existing **PagerDutyPublisher** from a minimal HTTP wrapper (21 LOC, Grade D+) into a **comprehensive, enterprise-grade PagerDuty Events API v2 integration** (7,500+ LOC, Grade A+) с полным incident lifecycle management, achieving **150%+ quality** через:

- ✅ Full PagerDuty Events API v2 integration
- ✅ Event lifecycle management (trigger, acknowledge, resolve)
- ✅ Change events support for incident context
- ✅ Intelligent retry logic + rate limiting
- ✅ Comprehensive error handling
- ✅ 90%+ test coverage
- ✅ Production-grade observability (8 metrics)
- ✅ Enterprise documentation (4,500+ LOC)

### 1.2 Current State vs Target

| Aspect | Baseline (30%) | Target (150%) | Gap |
|--------|----------------|---------------|-----|
| **API Integration** | Generic HTTP POST | Full Events API v2 | +100% |
| **Event Management** | Fire-and-forget | Trigger, ACK, Resolve | +100% |
| **Code Quality** | 21 LOC | 1,200 LOC | +5,614% |
| **Test Coverage** | ~5% | 90%+ | +85% |
| **Documentation** | 0 LOC | 4,500+ LOC | +∞ |
| **Metrics** | 0 | 8 Prometheus | +8 |
| **Grade** | D+ | A+ | +120% |

### 1.3 Strategic Alignment

**Publishing System Goals**:
- Enable multi-platform alert distribution (Rootly, PagerDuty, Slack, Webhooks)
- Provide incident management automation
- Ensure reliable, observable, enterprise-grade integrations

**TN-053 Contribution**:
- ✅ Complete PagerDuty integration (2 of 4 publishers)
- ✅ Reference implementation alongside TN-052 (Rootly)
- ✅ On-call workflow automation (reduce manual paging)
- ✅ AI-powered incident enrichment (via TN-051 formatter)

---

## 2. Business Value

### 2.1 Problem Statement

**Current Limitations**:
1. **No Real Event Management**: Baseline uses generic HTTP POST, не интегрируется с PagerDuty Events API
2. **One-Way Communication**: Fire-and-forget approach, нет tracking incident key
3. **No Lifecycle Support**: Невозможно acknowledge или resolve events
4. **Limited Context**: Только summary, нет custom_details
5. **Poor Observability**: Generic HTTP metrics, нет PagerDuty-specific insights
6. **No Change Events**: Невозможно отправить change events для incident context

**Impact**:
- ❌ Manual incident management required
- ❌ Duplicate alerts (no dedup_key tracking)
- ❌ Lost AI classification context
- ❌ Poor operational visibility
- ❌ Unreliable under load (no rate limiting)
- ❌ No deployment notifications

### 2.2 Solution Benefits

**Operational Benefits**:
- ✅ **Automated Event Lifecycle**: Alerts automatically trigger, acknowledge, resolve PagerDuty incidents
- ✅ **Full Context Preservation**: custom_details capture fingerprint, AI classification, labels
- ✅ **Deduplication**: dedup_key ensures same alert updates existing incident
- ✅ **Intelligent Retry**: Exponential backoff + rate limit detection
- ✅ **Enhanced Observability**: 8 PagerDuty-specific Prometheus metrics
- ✅ **Change Events**: Deployment notifications linked to incidents

**Team Benefits**:
- 📈 **Reduced MTTR**: Faster incident response через automated PagerDuty integration
- 🎯 **Better Context**: AI classification в PagerDuty incident details
- 🔄 **Automatic Updates**: Alert status changes propagate to PagerDuty
- 📊 **Operational Insights**: Metrics на event rate, API latency, acknowledgments

**Business Benefits**:
- 💰 **Cost Reduction**: Automated incident routing (vs manual paging)
- ⚡ **Faster Resolution**: AI-powered recommendations в incidents
- 🎖️ **SLA Compliance**: Reliable event tracking + metrics
- 🚀 **Scalability**: Rate limiting + retry logic handle production load

### 2.3 Success Indicators

**Quantitative**:
- 95%+ events sent successfully
- <300ms event send latency (p99)
- 100% deduplication tracking (fingerprint → dedup_key)
- Zero rate limit errors after retry logic
- 8 Prometheus metrics operational

**Qualitative**:
- Platform team approval (production-ready)
- SRE team approval (operational excellence)
- Grade A+ quality certification
- Zero breaking changes (backward compatibility)

---

## 3. Functional Requirements

### FR-1: PagerDuty Events API v2 Integration (Critical, 150%)

**Description**: Full integration с PagerDuty Events API v2 для event management

**Acceptance Criteria**:
- ✅ HTTP client с Events API base URL (`https://events.pagerduty.com/v2`)
- ✅ Authentication via Integration Key (routing_key)
- ✅ Content-Type: `application/json` для всех requests
- ✅ Request/response JSON serialization/deserialization
- ✅ HTTPS-only communication (TLS 1.2+)
- ✅ User-Agent: `AlertHistory/1.0 (+github.com/ipiton/alert-history)` for tracking

**Test Coverage**: 90%+

---

### FR-2: Event Triggering (POST /events) (Critical, 150%)

**Description**: Trigger PagerDuty events при получении firing alerts

**API Endpoint**: `POST https://events.pagerduty.com/v2/events`

**Request Model**:
```go
type TriggerEventRequest struct {
    RoutingKey  string                 `json:"routing_key"`  // Integration key
    EventAction string                 `json:"event_action"` // "trigger"
    DedupKey    string                 `json:"dedup_key"`    // Alert fingerprint
    Payload     TriggerEventPayload    `json:"payload"`
    Links       []EventLink            `json:"links,omitempty"`
    Images      []EventImage           `json:"images,omitempty"`
}

type TriggerEventPayload struct {
    Summary        string                 `json:"summary"`         // Required
    Source         string                 `json:"source"`          // Required
    Severity       string                 `json:"severity"`        // critical/warning/error/info
    Timestamp      string                 `json:"timestamp"`       // ISO 8601
    Component      string                 `json:"component,omitempty"`
    Group          string                 `json:"group,omitempty"`
    Class          string                 `json:"class,omitempty"`
    CustomDetails  map[string]interface{} `json:"custom_details,omitempty"`
}
```

**Response Model**:
```go
type EventResponse struct {
    Status   string `json:"status"`    // "success"
    Message  string `json:"message"`   // "Event processed"
    DedupKey string `json:"dedup_key"` // Echo back for tracking
}
```

**Behavior**:
1. Alert status == `firing` → Trigger event
2. Build TriggerEventRequest:
   - `routing_key`: From target configuration
   - `event_action`: "trigger"
   - `dedup_key`: Alert fingerprint (for deduplication)
   - `payload.summary`: "[SEVERITY] AlertName - AI: Classification"
   - `payload.severity`: Map classification → PagerDuty levels
   - `payload.timestamp`: alert.StartsAt
   - `payload.custom_details`: AI classification, labels, annotations
3. POST to `/v2/events`
4. Parse response, extract dedup_key
5. Cache mapping: fingerprint → dedup_key (for future updates)
6. Record metrics: `pagerduty_events_triggered_total`

**Test Coverage**: 95%+

---

### FR-3: Event Acknowledgment (POST /events) (High, 150%)

**Description**: Acknowledge PagerDuty events для silenced/acknowledged alerts

**Request Model**:
```go
type AcknowledgeEventRequest struct {
    RoutingKey  string `json:"routing_key"`  // Integration key
    EventAction string `json:"event_action"` // "acknowledge"
    DedupKey    string `json:"dedup_key"`    // Alert fingerprint
}
```

**Behavior**:
1. Alert status == `acknowledged` OR `silenced` → Acknowledge event
2. Lookup dedup_key from cache (fingerprint → dedup_key)
3. Build AcknowledgeEventRequest:
   - `routing_key`: From target configuration
   - `event_action`: "acknowledge"
   - `dedup_key`: From cache
4. POST to `/v2/events`
5. Record metrics: `pagerduty_events_acknowledged_total`

**Test Coverage**: 90%+

---

### FR-4: Event Resolution (POST /events) (Critical, 150%)

**Description**: Resolve PagerDuty events при получении resolved alerts

**Request Model**:
```go
type ResolveEventRequest struct {
    RoutingKey  string `json:"routing_key"`  // Integration key
    EventAction string `json:"event_action"` // "resolve"
    DedupKey    string `json:"dedup_key"`    // Alert fingerprint
}
```

**Behavior**:
1. Alert status == `resolved` → Resolve event
2. Lookup dedup_key from cache (fingerprint → dedup_key)
3. Build ResolveEventRequest:
   - `routing_key`: From target configuration
   - `event_action`: "resolve"
   - `dedup_key`: From cache
4. POST to `/v2/events`
5. Record metrics: `pagerduty_events_resolved_total`
6. Remove from cache (event lifecycle complete)

**Test Coverage**: 90%+

---

### FR-5: Change Events Support (Medium, 150%)

**Description**: Send change events для deployment/configuration notifications

**API Endpoint**: `POST https://events.pagerduty.com/v2/change/enqueue`

**Request Model**:
```go
type ChangeEventRequest struct {
    RoutingKey string           `json:"routing_key"`
    Payload    ChangeEventPayload `json:"payload"`
    Links      []EventLink      `json:"links,omitempty"`
}

type ChangeEventPayload struct {
    Summary    string                 `json:"summary"`    // Required
    Source     string                 `json:"source"`     // Required
    Timestamp  string                 `json:"timestamp"`  // ISO 8601
    CustomDetails map[string]interface{} `json:"custom_details,omitempty"`
}
```

**Behavior**:
1. Alerts с label `change_event=true` → Send change event
2. Build ChangeEventRequest
3. POST to `/v2/change/enqueue`
4. Record metrics: `pagerduty_change_events_total`

**Test Coverage**: 80%+

---

### FR-6: Deduplication Key Tracking (Critical, 150%)

**Description**: In-memory cache для tracking fingerprint → dedup_key mapping

**Cache Model**:
```go
type EventKeyCache interface {
    Set(fingerprint string, dedupKey string)
    Get(fingerprint string) (dedupKey string, found bool)
    Delete(fingerprint string)
    Cleanup() // Remove expired entries
}

type eventKeyCacheImpl struct {
    data    sync.Map
    ttl     time.Duration // 24h default
}
```

**Behavior**:
1. После trigger: `cache.Set(fingerprint, dedup_key)`
2. Перед acknowledge/resolve: `dedup_key, found := cache.Get(fingerprint)`
3. После resolve: `cache.Delete(fingerprint)`
4. Background cleanup: Every 1h, remove entries older than 24h

**Test Coverage**: 95%+

---

### FR-7: Links and Images Support (Medium, 150%)

**Description**: Attach links и images к events для rich context

**Models**:
```go
type EventLink struct {
    Href string `json:"href"` // URL
    Text string `json:"text"` // Link text
}

type EventImage struct {
    Src  string `json:"src"`  // Image URL
    Href string `json:"href"` // Link URL (optional)
    Alt  string `json:"alt"`  // Alt text
}
```

**Behavior**:
1. Extract Grafana dashboard link from annotations → Add as link
2. Extract Runbook URL from annotations → Add as link
3. Add Grafana snapshot as image (if available)

**Test Coverage**: 80%+

---

## 4. Non-Functional Requirements

### NFR-1: Performance (Critical)

**Requirements**:
- Event send latency (p50): <100ms
- Event send latency (p95): <200ms
- Event send latency (p99): <300ms
- Throughput: 100+ events/sec
- Rate limiting: 120 req/min (PagerDuty limit)

**Validation**:
- Load testing с 1000 concurrent alerts
- Benchmarks для each operation
- Prometheus metrics recording p50/p95/p99

---

### NFR-2: Reliability (Critical)

**Requirements**:
- Event delivery success rate: 99%+
- Retry logic: 3 attempts с exponential backoff
- Graceful degradation: Fallback to HTTP publisher on API errors
- Zero data loss: Events queued during outages

**Validation**:
- Chaos testing (API failures, timeouts, rate limits)
- Retry logic tests (429, 5xx responses)
- Failover tests (API unreachable)

---

### NFR-3: Observability (Critical)

**Prometheus Metrics** (8 total):
1. `pagerduty_events_triggered_total` (Counter by routing_key, severity)
2. `pagerduty_events_acknowledged_total` (Counter by routing_key)
3. `pagerduty_events_resolved_total` (Counter by routing_key)
4. `pagerduty_change_events_total` (Counter by routing_key)
5. `pagerduty_api_requests_total` (Counter by endpoint, status)
6. `pagerduty_api_errors_total` (Counter by error_type)
7. `pagerduty_api_duration_seconds` (Histogram by endpoint)
8. `pagerduty_rate_limit_hits_total` (Counter by routing_key)

**Structured Logging**:
- DEBUG: API request/response details
- INFO: Event triggered/acknowledged/resolved
- WARN: Retry attempts, rate limit hits
- ERROR: API errors, validation failures

---

### NFR-4: Security (Critical)

**Requirements**:
- Integration keys stored в K8s Secrets
- TLS 1.2+ для all API calls
- No secrets logged
- Integration key rotation support

---

### NFR-5: Testability (Critical)

**Requirements**:
- Unit test coverage: 90%+
- Integration tests: 10+ scenarios
- Benchmarks: 8+ operations
- Mock PagerDuty API client для testing

---

## 5. PagerDuty Events API v2 Integration

### 5.1 API Overview

**Base URL**: `https://events.pagerduty.com`

**Authentication**: Integration Key (routing_key) в request body

**Rate Limits**:
- Events API: 120 requests/minute per integration
- Change Events API: 120 requests/minute per integration

**HTTP Status Codes**:
- 202 Accepted: Event received and queued
- 400 Bad Request: Invalid payload
- 401 Unauthorized: Invalid routing_key
- 429 Too Many Requests: Rate limit exceeded
- 500-503: Server errors (retry)

---

### 5.2 Event Actions

| Action | Behavior | idempotent? |
|--------|----------|-------------|
| `trigger` | Create new incident or update existing | Yes (via dedup_key) |
| `acknowledge` | Acknowledge incident | Yes |
| `resolve` | Resolve incident | Yes |

---

### 5.3 Severity Mapping

| Classification Severity | PagerDuty Severity | Urgency |
|------------------------|-------------------|---------|
| `critical` | `critical` | `high` |
| `warning` | `warning` | `high` |
| `error` | `error` | `low` |
| `info` | `info` | `low` |
| `noise` | `info` | `low` |

---

## 6. Dependencies

### 6.1 Completed Dependencies

| Task | Status | Quality | Notes |
|------|--------|---------|-------|
| TN-046 | ✅ COMPLETE | 150% (A+) | K8s Client для secrets |
| TN-047 | ✅ COMPLETE | 147% (A+) | Target Discovery |
| TN-048 | ✅ COMPLETE | 160% (A+) | Target Refresh |
| TN-049 | ✅ COMPLETE | 140% (A) | Health Monitoring |
| TN-050 | ✅ COMPLETE | 155% (A+) | RBAC для secrets |
| TN-051 | ✅ COMPLETE | 155% (A+) | Alert Formatter (PagerDuty format implemented) |
| TN-052 | ✅ COMPLETE | 177% (A+) | Rootly Publisher (reference architecture) |

### 6.2 Integration Points

1. **Alert Formatter (TN-051)**:
   - formatPagerDuty() уже реализован
   - Provides formatted payload для Events API

2. **Target Discovery (TN-047)**:
   - Discovers PagerDuty targets из K8s Secrets
   - Label selector: `publishing-target=true`, `type=pagerduty`

3. **Publisher Factory**:
   - Creates PagerDutyPublisher instances
   - Manages shared cache and metrics

---

## 7. Risk Assessment

### 7.1 Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Rate limit exceeded | Medium | High | Implement rate limiter (120 req/min), exponential backoff |
| API downtime | Low | High | Retry logic, fallback to HTTP publisher, queue events |
| Dedup key collision | Low | Medium | Use SHA-256 fingerprint, validate uniqueness |
| Integration key exposure | Low | Critical | Store в K8s Secrets, no logging, rotation support |

### 7.2 Operational Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Alert storm → API saturation | Medium | High | Rate limiting, batching, priority queue |
| Duplicate events | Low | Medium | Dedup key tracking, cache validation |
| Lost events during deploy | Low | Medium | Graceful shutdown, event queue persistence |

---

## 8. Acceptance Criteria

### 8.1 Functional Acceptance

- [ ] **FR-1**: Events API v2 client implemented and tested
- [ ] **FR-2**: Event triggering working (trigger events for firing alerts)
- [ ] **FR-3**: Event acknowledgment working (acknowledge events)
- [ ] **FR-4**: Event resolution working (resolve events)
- [ ] **FR-5**: Change events support implemented
- [ ] **FR-6**: Dedup key cache operational (24h TTL)
- [ ] **FR-7**: Links and images attachment working

### 8.2 Non-Functional Acceptance

- [ ] **NFR-1**: Performance targets met (p99 < 300ms)
- [ ] **NFR-2**: Reliability: 99%+ success rate, retry logic working
- [ ] **NFR-3**: Observability: 8 Prometheus metrics operational
- [ ] **NFR-4**: Security: Keys в K8s Secrets, TLS 1.2+
- [ ] **NFR-5**: Testability: 90%+ coverage, 10+ integration tests

### 8.3 Quality Gates

- [ ] **Test Coverage**: 90%+ (unit + integration)
- [ ] **Linter**: Zero errors (golangci-lint)
- [ ] **Benchmarks**: All operations meet performance targets
- [ ] **Documentation**: 4,500+ LOC (README, API docs, integration guide)
- [ ] **Code Review**: Platform + SRE team approval
- [ ] **Grade**: A+ (Excellent, Production-Ready)

---

## 9. Success Metrics

### 9.1 Quantitative Metrics

| Metric | Baseline | Target (150%) | Measurement |
|--------|----------|---------------|-------------|
| **Code Quality** | 21 LOC | 1,200 LOC | LOC count |
| **Test Coverage** | ~5% | 90%+ | go test -cover |
| **Performance (p99)** | ~1s | <300ms | Benchmarks |
| **Event Success Rate** | ~70% | 99%+ | Prometheus metrics |
| **Documentation** | 0 LOC | 4,500+ LOC | LOC count |
| **Prometheus Metrics** | 0 | 8 | Metric count |

### 9.2 Qualitative Metrics

- ✅ Platform Team Approval (production-ready certification)
- ✅ SRE Team Approval (operational excellence)
- ✅ Architecture Team Approval (design quality)
- ✅ Security Team Approval (no vulnerabilities)
- ✅ Zero Breaking Changes (backward compatibility)

### 9.3 Deliverables Checklist

- [ ] **Implementation**: 1,200 LOC (client, publisher, models, errors, cache, metrics)
- [ ] **Tests**: 800+ LOC (30+ unit tests, 10+ integration, 8+ benchmarks)
- [ ] **Documentation**: 4,500+ LOC (README, API guide, integration guide, runbook)
- [ ] **K8s Examples**: PagerDuty secret manifest (examples/k8s/pagerduty-secret.yaml)
- [ ] **Metrics Dashboard**: Grafana dashboard для PagerDuty metrics
- [ ] **CHANGELOG**: Comprehensive entry documenting all changes

---

## 10. Timeline and Effort Estimation

| Phase | Tasks | Effort | Dependencies |
|-------|-------|--------|--------------|
| **Phase 1-3** | Docs (requirements, design, tasks) | 4h | None |
| **Phase 4** | Events API v2 client implementation | 12h | TN-051 |
| **Phase 5** | Unit tests (30+) + benchmarks (8+) | 10h | Phase 4 |
| **Phase 6** | Integration tests (10+) | 6h | Phase 5 |
| **Phase 7** | Dedup key cache + cleanup worker | 8h | Phase 4 |
| **Phase 8** | Metrics + observability (8 metrics) | 6h | Phase 4 |
| **Phase 9** | Documentation (README, API guide) | 8h | Phase 8 |
| **Phase 10** | Integration in PublisherFactory | 4h | Phase 8 |
| **Phase 11** | K8s examples + deployment guide | 4h | Phase 10 |
| **Phase 12** | Final validation + certification | 4h | Phase 11 |
| **Total** | 12 phases | **80 hours** | - |

**Estimated Duration**: 10 days (8h/day)

---

## 11. References

### 11.1 PagerDuty Documentation

- [Events API v2 Overview](https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgw-events-api-v2-overview)
- [Send an Event](https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgx-send-an-event-events-api-v2)
- [Change Events](https://developer.pagerduty.com/docs/ZG9jOjExMDI5NTgy-send-change-events-to-the-pagerduty-events-api)

### 11.2 Related Tasks

- TN-051: Alert Formatter (PagerDuty format implemented)
- TN-052: Rootly Publisher (reference architecture)
- TN-054: Slack Publisher (next task)
- TN-055: Generic Webhook Publisher (next task)

---

**Document Status**: ✅ APPROVED FOR IMPLEMENTATION
**Next Step**: Create design.md (Phase 2)
**Estimated Completion**: 2025-11-21 (10 days from start)
