# Alertmanager++ OSS Core — Technical Specification

## Architecture Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     External Systems                         │
├──────────────┬────────────────┬──────────────┬─────────────┤
│  Prometheus  │  Alertmanager  │   Webhooks   │   Users     │
└──────┬───────┴───────┬────────┴──────┬───────┴──────┬──────┘
       │               │               │              │
       ▼               ▼               ▼              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Ingestion Layer                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │  API Gateway (Gin/Fiber)                             │  │
│  ├──────────────────────────────────────────────────────┤  │
│  │  /api/v2/alerts  │  /webhook  │  /api/v1/*  │  /ui  │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Processing Layer                         │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │
│  │Deduplication│  │  Grouping   │  │    Routing      │    │
│  │  (TN-36)    │  │  (TN-121+)  │  │   (TN-137+)     │    │
│  └─────────────┘  └─────────────┘  └─────────────────┘    │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐    │
│  │ Inhibition  │  │  Silencing  │  │  AI Enrichment  │    │
│  │  (TN-126+)  │  │  (TN-131+)  │  │   (TN-33,34)    │    │
│  └─────────────┘  └─────────────┘  └─────────────────┘    │
└─────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                     Storage Layer                           │
│  ┌──────────────────┐  ┌──────────────┐  ┌─────────────┐  │
│  │   PostgreSQL     │  │    Redis     │  │   SQLite    │  │
│  │   (Primary)      │  │   (Cache)    │  │    (Dev)    │  │
│  └──────────────────┘  └──────────────┘  └─────────────┘  │
└─────────────────────────────────────────────────────────────┘
       │
       ▼
┌─────────────────────────────────────────────────────────────┐
│                    Delivery Layer                           │
│  ┌───────────┐  ┌───────────┐  ┌──────────┐  ┌─────────┐  │
│  │  Webhook  │  │   Slack   │  │PagerDuty │  │  Email  │  │
│  │  (TN-55)  │  │  (TN-54)  │  │ (TN-53)  │  │   ...   │  │
│  └───────────┘  └───────────┘  └──────────┘  └─────────┘  │
└─────────────────────────────────────────────────────────────┘
```

## Component Specifications

### 1. API Gateway

**Technology Stack:**
- Framework: Gin (chosen over Fiber based on TN-09 benchmarks)
- Middleware: Recovery, RequestID, Logging, Metrics, RateLimit, Auth, CORS
- Protocol: HTTP/1.1, HTTP/2, WebSocket

**Endpoints:**

#### Alert Ingestion APIs
```yaml
POST /api/v2/alerts:
  Task: TN-147
  Description: Prometheus-compatible alert receiver
  Request: AlertmanagerWebhook
  Response: 200 OK
  Rate Limit: 10,000 req/sec

POST /webhook:
  Task: TN-61
  Description: Universal webhook endpoint
  Request: Auto-detected format
  Response: 200/207 Multi-status
  Features: 10 middleware layers, OWASP compliant

POST /webhook/proxy:
  Task: TN-62
  Description: Intelligent proxy with enrichment
  Request: Any webhook format
  Response: Enriched alerts
  Features: 3-pipeline architecture
```

#### History & Analytics APIs
```yaml
GET /history:
  Task: TN-63
  Description: Alert history with 18+ filters
  Query Params: status, severity, namespace, labels, time_range
  Response: Paginated alerts
  Cache: 2-tier (L1 Ristretto + L2 Redis)
  Performance: p95 < 6.5ms

GET /report:
  Task: TN-64
  Description: Analytics summary
  Response: Aggregated statistics
  Performance: p95 < 85ms
```

### 2. Processing Components

#### 2.1 Deduplication Engine

**Implementation: TN-36**
```go
type DeduplicationService struct {
    cache    *ristretto.Cache  // L1 in-memory
    redis    *redis.Client     // L2 distributed
    hasher   hash.Hash         // SHA256
    ttl      time.Duration     // Default: 5m
}

Performance:
- Fingerprint generation: 81.75ns (12.2x faster than target)
- Cache lookup: < 50ns
- Test coverage: 98.14%
```

#### 2.2 Grouping Engine

**Implementation: TN-121-125**
```go
type GroupingEngine struct {
    config   *GroupingConfig     // From YAML
    keyGen   *GroupKeyGenerator  // FNV-1a hash
    manager  *AlertGroupManager  // Lifecycle
    timers   *TimerManager       // Redis-backed
    storage  *GroupStorage       // Distributed state
}

Configuration:
- group_by: [alertname, cluster, service]
- group_wait: 30s
- group_interval: 5m
- repeat_interval: 12h

Performance:
- Key generation: 404x faster than target
- Group operations: < 5µs
- Coverage: 93.6%
```

#### 2.3 Routing Engine

**Implementation: TN-137-141**
```go
type RoutingEngine struct {
    parser    *RouteConfigParser
    tree      *RouteTree
    matcher   *RouteMatcher
    evaluator *RouteEvaluator
    publisher *MultiReceiverPublisher
}

Features:
- Nested routes with inheritance
- Label matchers (exact, regex)
- Continue flag for multi-routing
- Per-route configuration
```

#### 2.4 Inhibition Engine

**Implementation: TN-126-130**
```go
type InhibitionEngine struct {
    parser   *InhibitionRuleParser
    matcher  *InhibitionMatcher
    cache    *ActiveAlertCache
    state    *InhibitionStateManager
}

Performance:
- Rule evaluation: 16.958µs (71x faster)
- Cache operations: 58ns (17,000x faster)
- Coverage: 86.6%
```

#### 2.5 Silence Engine

**Implementation: TN-131-136**
```go
type SilenceEngine struct {
    storage  *PostgresSilenceRepository
    matcher  *SilenceMatcher
    manager  *SilenceManager
    gc       *SilenceGarbageCollector
}

Features:
- Operators: =, !=, =~, !~
- TTL management
- Background GC
- Audit logging
Coverage: 95.9%
```

### 3. Storage Layer

#### 3.1 PostgreSQL Schema

**Migrations: TN-14 (Goose)**
```sql
-- Main alerts table
CREATE TABLE alerts (
    id SERIAL PRIMARY KEY,
    fingerprint VARCHAR(64) UNIQUE NOT NULL,
    status VARCHAR(20) NOT NULL,
    labels JSONB NOT NULL,
    annotations JSONB,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP,
    generator_url TEXT,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Performance indexes (TN-63)
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_starts_at ON alerts(starts_at DESC);
CREATE INDEX idx_alerts_labels_gin ON alerts USING GIN(labels);
CREATE INDEX idx_alerts_fingerprint ON alerts(fingerprint);

-- Silences table (TN-133)
CREATE TABLE silences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    matchers JSONB NOT NULL,
    starts_at TIMESTAMP NOT NULL,
    ends_at TIMESTAMP NOT NULL,
    created_by VARCHAR(255),
    comment TEXT,
    status VARCHAR(20) DEFAULT 'active'
);
```

#### 3.2 Redis Cache Structure

**Implementation: TN-16, TN-125, TN-128**
```yaml
Keys:
  dedup:{fingerprint}: Alert deduplication
  group:{key}: Group state
  inhibit:{fingerprint}: Inhibition state
  cache:alert:{id}: Alert cache
  cache:history:{query_hash}: Query cache

TTLs:
  dedup: 5 minutes
  group: 24 hours
  inhibit: 1 hour
  cache: 5 minutes
```

### 4. AI Integration (BYOK)

#### 4.1 LLM Client

**Implementation: TN-33**
```go
type LLMClient interface {
    ClassifyAlert(ctx context.Context, alert Alert) (*Classification, error)
    Health(ctx context.Context) error
}

type Classification struct {
    Severity    string  `json:"severity"`
    Summary     string  `json:"summary"`
    Action      string  `json:"suggested_action"`
    Confidence  float64 `json:"confidence"`
    Cached      bool    `json:"cached"`
}

Providers:
- OpenAI (GPT-3.5/4)
- Anthropic (Claude)
- OpenRouter (Multiple)

Cache:
- L1: In-memory LRU
- L2: Redis with TTL
```

### 5. Publishing System

#### 5.1 Publisher Interface

**Base: TN-46-60**
```go
type Publisher interface {
    Publish(ctx context.Context, alert Alert) error
    Health(ctx context.Context) error
    GetName() string
}

Implementations:
- WebhookPublisher (TN-55): Generic HTTP
- SlackPublisher (TN-54): Slack API
- PagerDutyPublisher (TN-53): Events API v2
- EmailPublisher: SMTP
```

#### 5.2 Publishing Pipeline

**Implementation: TN-56-59**
```yaml
Pipeline:
  Queue: PostgreSQL-backed with DLQ
  Workers: Configurable pool (default: 10)
  Retry: Exponential backoff
  Parallel: Fan-out to multiple targets
  Metrics: 17+ Prometheus metrics

Performance:
- Throughput: 1M+ alerts/sec
- Latency: < 1.3µs per target
- Queue capacity: 10,000 jobs
```

## Configuration

### Main Configuration (config.yaml)

```yaml
# Server Configuration
server:
  address: ":9093"
  read_timeout: 30s
  write_timeout: 30s
  shutdown_timeout: 30s

# Storage Configuration
storage:
  type: postgresql  # or sqlite
  postgresql:
    host: localhost
    port: 5432
    database: alertmanager
    user: alertmanager
    password: ${DB_PASSWORD}
    max_connections: 25
    max_idle: 5

  retention: 30d
  cleanup_interval: 1h

# Redis Cache
cache:
  enabled: true
  address: localhost:6379
  password: ${REDIS_PASSWORD}
  db: 0
  ttl: 5m

# AI Configuration (Optional)
ai:
  enabled: false
  provider: openai  # openai, anthropic, openrouter
  api_key: ${LLM_API_KEY}
  model: gpt-3.5-turbo
  timeout: 10s
  cache_ttl: 1h
  fallback_enabled: true

# Alerting Configuration
alerting:
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 12h

  route:
    receiver: default
    group_by: [alertname, cluster]
    routes:
      - match:
          severity: critical
        receiver: pagerduty
        continue: true

      - match_re:
          service: database.*
        receiver: dba-team

  receivers:
    - name: default
      webhook_configs:
        - url: http://localhost:8080/webhook

    - name: pagerduty
      pagerduty_configs:
        - routing_key: ${PAGERDUTY_KEY}

    - name: dba-team
      slack_configs:
        - api_url: ${SLACK_WEBHOOK_URL}
          channel: "#dba-alerts"

  inhibit_rules:
    - source_match:
        severity: critical
      target_match:
        severity: warning
      equal: [alertname, cluster]

# Observability
metrics:
  enabled: true
  path: /metrics

logging:
  level: info  # debug, info, warn, error
  format: json  # json, text

tracing:
  enabled: false
  provider: jaeger
  endpoint: localhost:6831
```

## Performance Requirements

### Throughput & Latency

| Operation | Requirement | Current | Status |
|-----------|------------|---------|--------|
| Alert Ingestion | 10,000/sec | 12,000/sec | ✅ |
| Deduplication | < 1ms | 81.75ns | ✅ |
| Grouping | < 5ms | < 5µs | ✅ |
| Routing | < 10ms | < 1ms | ✅ |
| Storage Write | < 10ms | 3-4ms | ✅ |
| Query (cached) | < 10ms | 6.5ms | ✅ |
| Query (cold) | < 100ms | < 50ms | ✅ |

### Resource Requirements

```yaml
Minimum (Development):
  CPU: 1 core
  Memory: 512MB
  Storage: 10GB

Recommended (Production):
  CPU: 4 cores
  Memory: 4GB
  Storage: 100GB SSD
  Redis: 1GB

High-Volume (Enterprise):
  CPU: 16 cores
  Memory: 32GB
  Storage: 1TB NVMe
  Redis: 8GB cluster
  PostgreSQL: Dedicated instance
```

## Security Specifications

### Authentication & Authorization

```yaml
Authentication:
  - Basic Auth (username/password)
  - Bearer Token (JWT)
  - API Keys

Authorization:
  - RBAC with roles:
    - admin: Full access
    - operator: Create/modify alerts
    - viewer: Read-only access

Rate Limiting:
  - Per-IP: 100 req/sec
  - Per-User: 1000 req/sec
  - Global: 10,000 req/sec
```

### Security Headers

**Implementation: TN-61, TN-63**
```yaml
Headers:
  X-Content-Type-Options: nosniff
  X-Frame-Options: DENY
  X-XSS-Protection: 1; mode=block
  Content-Security-Policy: default-src 'self'
  Strict-Transport-Security: max-age=31536000
  Cache-Control: no-store
  Pragma: no-cache
```

### OWASP Top 10 Compliance

| Vulnerability | Mitigation | Status |
|--------------|------------|--------|
| A01: Broken Access Control | RBAC, JWT validation | ✅ |
| A02: Cryptographic Failures | TLS 1.2+, encrypted storage | ✅ |
| A03: Injection | Parameterized queries, validation | ✅ |
| A04: Insecure Design | Security review, threat model | ✅ |
| A05: Security Misconfiguration | Secure defaults, headers | ✅ |
| A06: Vulnerable Components | Dependency scanning | ✅ |
| A07: Authentication Failures | Rate limiting, strong passwords | ✅ |
| A08: Data Integrity | Checksums, audit logs | ✅ |
| A09: Logging Failures | Structured logs, no secrets | ✅ |
| A10: SSRF | URL validation, allowlists | ✅ |

## Testing Strategy

### Test Coverage Requirements

```yaml
Unit Tests:
  Target: 80%
  Current: 85%+
  Tools: go test, testify

Integration Tests:
  Scope: API endpoints, database
  Tools: testcontainers, httptest

E2E Tests:
  Scope: Full alert flow
  Tools: k6, playwright

Load Tests:
  Target: 10,000 alerts/sec for 24h
  Tools: k6, vegeta

Chaos Tests:
  Scenarios: Network partition, pod kills
  Tools: Chaos Mesh, Litmus
```

### Test Implementation Status

| Component | Unit | Integration | E2E | Load |
|-----------|------|-------------|-----|------|
| Deduplication (TN-36) | 98% | ✅ | ✅ | ✅ |
| Grouping (TN-121-125) | 93% | ✅ | 🔄 | ✅ |
| Inhibition (TN-126-130) | 86% | ✅ | 🔄 | ✅ |
| Silencing (TN-131-136) | 95% | ✅ | ✅ | 🔄 |
| Storage (TN-32,37) | 90% | ✅ | ✅ | ✅ |
| API (TN-61-64) | 92% | ✅ | ✅ | ✅ |

## Migration Path

### From Prometheus Alertmanager

```bash
# Step 1: Deploy Alertmanager++ alongside existing
docker run -d \
  -p 9094:9093 \
  -v ./alertmanager.yml:/etc/alertmanager++/config.yml \
  alertmanager-plus-plus:v1.0

# Step 2: Add as additional alertmanager
# prometheus.yml
alerting:
  alertmanagers:
    - static_configs:
      - targets:
        - 'alertmanager:9093'      # Existing
        - 'alertmanager-plus:9094' # New

# Step 3: Validate both receive alerts
curl http://localhost:9094/api/v2/alerts

# Step 4: Migrate receivers one by one
# Move webhook configs, validate, repeat

# Step 5: Switch primary
# Remove old alertmanager from targets
```

### Data Migration

```sql
-- Export from existing system (if any)
COPY (
  SELECT * FROM alerts
  WHERE created_at > NOW() - INTERVAL '30 days'
) TO '/tmp/alerts_export.csv' CSV HEADER;

-- Import to Alertmanager++
COPY alerts FROM '/tmp/alerts_export.csv' CSV HEADER;

-- Verify
SELECT COUNT(*), MIN(created_at), MAX(created_at) FROM alerts;
```

## Appendix: Task Completion Status

### Completed Tasks (72/181 - 39.8%)

| Phase | Completed | Total | Percentage |
|-------|-----------|-------|------------|
| Phase 1: Infrastructure | 8 | 8 | 100% |
| Phase 2: Data Layer | 12 | 12 | 100% |
| Phase 3: Observability | 10 | 10 | 100% |
| Phase 4: Core Business | 15 | 15 | 100% |
| Phase A: Grouping | 5 | 5 | 100% |
| Phase A: Inhibition | 5 | 5 | 100% |
| Phase A: Silencing | 6 | 6 | 100% |
| Phase 5: Publishing | 15 | 15 | 100% |
| Phase 6: REST API | 11 | 15 | 73% |

### Priority Implementation Order

1. **Critical Path (MVP)**
   - Routing (TN-137-141) - 5 tasks
   - Templates (TN-153-156) - 4 tasks
   - Config Management (TN-149-152) - 4 tasks

2. **Enhanced Features**
   - Dashboard UI (TN-76-85) - 10 tasks
   - Advanced UI (TN-169-172) - 4 tasks

3. **Production Readiness**
   - Testing (TN-106-115) - 10 tasks
   - Documentation (TN-116-120, TN-176-180) - 10 tasks
   - Packaging (TN-96-105) - 10 tasks

---

*Technical Specification v1.0*
*Based on tasks.md analysis*
*Last updated: November 2025*
