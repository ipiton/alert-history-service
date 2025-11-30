# Alertmanager++ OSS Architecture Documentation

**Version**: 2.0.0
**Last Updated**: 2025-11-30
**Audience**: Architects, Senior Engineers, Technical Leaders

---

## 📋 Table of Contents

1. [System Overview](#system-overview)
2. [Architecture Diagrams](#architecture-diagrams)
3. [Component Deep Dive](#component-deep-dive)
4. [Data Flow](#data-flow)
5. [Deployment Architectures](#deployment-architectures)
6. [Technology Stack](#technology-stack)
7. [Design Decisions](#design-decisions)
8. [Security Architecture](#security-architecture)
9. [Scalability & High Availability](#scalability--high-availability)

---

## System Overview

### Purpose

Alertmanager++ is an **enterprise-grade alert management and intelligence platform** that extends Prometheus Alertmanager with:
- 📊 **Long-term alert history** (PostgreSQL/SQLite storage)
- 🤖 **AI-powered classification** (LLM integration)
- 📤 **Multi-target publishing** (Slack, PagerDuty, Rootly, Webhooks)
- 🔍 **Advanced analytics** (trends, flapping detection, top alerts)
- 🎯 **Intelligent routing** (grouping, inhibition, silencing)
- 📈 **Comprehensive observability** (Prometheus metrics, structured logging)

### Key Characteristics

- **Cloud-Native**: Kubernetes-first design, 12-factor principles
- **Highly Available**: Multi-replica, stateless application pods
- **Scalable**: Horizontal auto-scaling (HPA), handles >10K alerts/day
- **Observable**: 50+ Prometheus metrics, structured logging
- **Extensible**: Plugin architecture for publishers, formatters
- **API-First**: RESTful API v2, Alertmanager-compatible

---

## Architecture Diagrams

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      Prometheus Ecosystem                       │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│  │Prometheus│  │Prometheus│  │   Other  │  │Alertmgr  │      │
│  │ Server 1 │  │ Server 2 │  │  Sources │  │  (Old)   │      │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘      │
│       │             │             │             │             │
└───────┼─────────────┼─────────────┼─────────────┼─────────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                        │
                        ▼
        ╔═══════════════════════════════════════════════╗
        ║      Alertmanager++ Ingestion Layer          ║
        ║  ┌────────────────────────────────────────┐  ║
        ║  │  /webhook (Alertmanager compatible)    │  ║
        ║  │  /webhook/proxy (Intelligent proxy)    │  ║
        ║  │  /api/v2/alerts (Prometheus format)    │  ║
        ║  └────────────────────────────────────────┘  ║
        ╚═══════════════════════════════════════════════╝
                        │
                        ▼
        ╔═══════════════════════════════════════════════╗
        ║      Alert Processing Pipeline               ║
        ║  ┌─────────────────────────────────────────┐ ║
        ║  │ 1. Deduplication (SHA-256 fingerprint)  │ ║
        ║  │ 2. Grouping (by labels/namespace)       │ ║
        ║  │ 3. Inhibition (suppress duplicates)     │ ║
        ║  │ 4. Classification (LLM-powered AI)      │ ║
        ║  │ 5. Enrichment (metadata injection)      │ ║
        ║  │ 6. Filtering (severity/namespace)       │ ║
        ║  │ 7. Publishing (multi-target fanout)     │ ║
        ║  └─────────────────────────────────────────┘ ║
        ╚═══════════════════════════════════════════════╝
                        │
          ┌─────────────┼─────────────┐
          │             │             │
          ▼             ▼             ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │PostgreSQL│  │  Redis   │  │   LLM    │
    │ (History)│  │ (Cache)  │  │ Service  │
    └──────────┘  └──────────┘  └──────────┘
                        │
          ┌─────────────┼─────────────┐
          │             │             │
          ▼             ▼             ▼
    ┌──────────┐  ┌──────────┐  ┌──────────┐
    │  Slack   │  │PagerDuty │  │  Rootly  │
    └──────────┘  └──────────┘  └──────────┘
```

### Request Flow (Alert Ingestion)

```
┌──────────┐
│Prometheus│
└────┬─────┘
     │ POST /webhook
     │ (Alertmanager format)
     ▼
┌─────────────────────────────────────┐
│  Ingestion Handler                  │
│  - Validate request                 │
│  - Parse alert format               │
│  - Generate fingerprint (SHA-256)   │
└────┬────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  Alert Processor                    │
│  1. Deduplication Service           │
│     - Check fingerprint in cache    │
│     - Skip if duplicate (5min TTL)  │
│  2. Classification Service (opt)    │
│     - L1 cache (memory) check       │
│     - L2 cache (Redis) check        │
│     - LLM API call if miss          │
│  3. Filtering Service               │
│     - Apply severity filters        │
│     - Apply namespace filters       │
│     - Apply label filters           │
│  4. Storage Service                 │
│     - Save to PostgreSQL/SQLite     │
│     - Update Redis cache            │
└────┬────────────────────────────────┘
     │
     ▼
┌─────────────────────────────────────┐
│  Publishing Coordinator             │
│  - Discover targets (K8s Secrets)   │
│  - Parallel fanout (goroutines)     │
│  - Health-aware routing             │
│  - Retry with backoff               │
└────┬────────────────────────────────┘
     │
     ├──────────┬──────────┬──────────┐
     ▼          ▼          ▼          ▼
  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐
  │Slack │  │PagerD│  │Rootly│  │Webhook│
  └──────┘  └──────┘  └──────┘  └──────┘
```

### Component Layering

```
┌────────────────────────────────────────────────────────┐
│                  Presentation Layer                    │
│  ┌──────────────────────────────────────────────────┐ │
│  │ REST API (Gin framework)                         │ │
│  │ - /webhook, /api/v2/*, /health, /metrics         │ │
│  │ - Middleware: Auth, CORS, Rate Limiting          │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
                        │
┌────────────────────────────────────────────────────────┐
│                  Business Logic Layer                  │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Services:                                        │ │
│  │ - Alert Processor                                │ │
│  │ - Classification Service                         │ │
│  │ - Deduplication Service                          │ │
│  │ - Filtering Service                              │ │
│  │ - Grouping Service                               │ │
│  │ - Inhibition Service                             │ │
│  │ - Silencing Service                              │ │
│  │ - Publishing Coordinator                         │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
                        │
┌────────────────────────────────────────────────────────┐
│               Infrastructure Layer                     │
│  ┌──────────────────────────────────────────────────┐ │
│  │ Repositories:                                    │ │
│  │ - PostgreSQL Alert History Repository           │ │
│  │ - Redis Cache Repository                         │ │
│  │ Publishers:                                      │ │
│  │ - Slack, PagerDuty, Rootly, Generic Webhook     │ │
│  │ External Integrations:                           │ │
│  │ - LLM HTTP Client                                │ │
│  │ - K8s Client (target discovery)                  │ │
│  └──────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────┘
                        │
┌────────────────────────────────────────────────────────┐
│                   Data Layer                           │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐   │
│  │ PostgreSQL  │  │   Redis     │  │   SQLite    │   │
│  │ (Standard)  │  │   (Cache)   │  │   (Lite)    │   │
│  └─────────────┘  └─────────────┘  └─────────────┘   │
└────────────────────────────────────────────────────────┘
```

---

## Component Deep Dive

### 1. Ingestion Layer

#### Webhook Handler
**Location**: `go-app/cmd/server/handlers/webhook.go`
**Responsibility**: Universal alert ingestion endpoint

**Key Features**:
- **Format Detection**: Auto-detects Alertmanager vs Generic JSON
- **Validation**: Schema validation, size limits (10MB), required fields
- **Parsing**: Converts external format → internal `core.Alert` model
- **Fingerprinting**: SHA-256 hash of labels for deduplication

**Supported Formats**:
1. Alertmanager v4 Webhook (array of alerts)
2. Prometheus Alerts API v2 (grouped by labels)
3. Generic JSON Webhook (custom fields)

**Performance**: <5ms p95 latency

---

#### Webhook Proxy Handler
**Location**: `go-app/cmd/server/handlers/webhook_proxy.go`
**Responsibility**: Intelligent proxy with classification

**Pipeline**:
1. **Classification**: LLM-powered alert analysis
2. **Filtering**: Apply severity/namespace rules
3. **Publishing**: Forward to downstream targets (Rootly, PagerDuty, Slack)

**Caching**:
- **L1 Cache**: In-memory LRU (1000 alerts)
- **L2 Cache**: Redis (24h TTL)
- **Hit Rate**: 95%+ (measured via metrics)

---

### 2. Business Logic Layer

#### Alert Processor
**Location**: `go-app/internal/business/processing/alert_processor.go`
**Responsibility**: Orchestrates alert processing pipeline

**Pipeline Stages**:
```go
1. Deduplication    → Check fingerprint, skip if seen within 5min
2. Grouping         → Group by labels (alertname, severity, namespace)
3. Inhibition       → Suppress alerts matching inhibition rules
4. Classification   → LLM analysis (optional, based on enrichment mode)
5. Enrichment       → Inject AI metadata (severity, confidence, reasoning)
6. Filtering        → Apply user-defined filters
7. Storage          → Persist to PostgreSQL/SQLite
8. Publishing       → Fanout to configured targets
```

**Error Handling**: Fail-safe design, continues on non-critical errors

---

#### Classification Service
**Location**: `go-app/internal/business/classification/classification_service.go`
**Responsibility**: AI-powered alert analysis

**Architecture**:
```
┌────────────────────────────────────────┐
│  Classification Service                │
│  ┌──────────────────────────────────┐ │
│  │ L1 Cache (Memory LRU)            │ │
│  │ - 1000 alerts                    │ │
│  │ - <5ms lookup                    │ │
│  └────────┬─────────────────────────┘ │
│           │ Miss                       │
│  ┌────────▼─────────────────────────┐ │
│  │ L2 Cache (Redis)                 │ │
│  │ - 24h TTL                        │ │
│  │ - <10ms lookup                   │ │
│  └────────┬─────────────────────────┘ │
│           │ Miss                       │
│  ┌────────▼─────────────────────────┐ │
│  │ LLM HTTP Client                  │ │
│  │ - OpenAI/Anthropic/etc           │ │
│  │ - <500ms latency                 │ │
│  │ - Fallback: Rule-based           │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
```

**Output**:
```json
{
  "severity": "critical",
  "category": "infrastructure",
  "confidence": 0.95,
  "reasoning": "High CPU indicates resource exhaustion",
  "action_items": ["Scale up", "Investigate process"],
  "tags": ["performance", "cpu"]
}
```

---

#### Publishing Coordinator
**Location**: `go-app/internal/business/publishing/publishing_coordinator.go`
**Responsibility**: Multi-target alert publishing

**Architecture**:
```
┌────────────────────────────────────────┐
│  Publishing Coordinator                │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ Target Discovery (K8s Secrets)   │ │
│  │ - Label selector: publishing-    │ │
│  │   target=true                    │ │
│  └────────┬─────────────────────────┘ │
│           │                            │
│  ┌────────▼─────────────────────────┐ │
│  │ Health Check (per target)        │ │
│  │ - HTTP connectivity test         │ │
│  │ - Fail-fast: 50ms TCP handshake  │ │
│  └────────┬─────────────────────────┘ │
│           │                            │
│  ┌────────▼─────────────────────────┐ │
│  │ Parallel Fanout (goroutines)     │ │
│  │ - Fan-out: N goroutines          │ │
│  │ - Fan-in: WaitGroup              │ │
│  │ - Circuit breaker per target     │ │
│  └────────┬─────────────────────────┘ │
│           │                            │
│  ┌────────▼─────────────────────────┐ │
│  │ Retry Logic (per target)         │ │
│  │ - Exponential backoff: 100ms→5s  │ │
│  │ - Max retries: 3                 │ │
│  │ - Smart error classification     │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘
```

**Publishers**:
- **Slack**: Block Kit format, threading (24h cache)
- **PagerDuty**: Events API v2, lifecycle (trigger/ack/resolve)
- **Rootly**: Incidents API v1, lifecycle (create/update/resolve)
- **Generic Webhook**: Custom JSON, configurable headers/auth

**Performance**: 1.3µs per target (parallel), 3,846x faster than serial

---

### 3. Infrastructure Layer

#### PostgreSQL Repository
**Location**: `go-app/internal/infrastructure/repository/postgres_history_repository.go`
**Responsibility**: Alert history persistence

**Schema**:
```sql
CREATE TABLE alert_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  fingerprint VARCHAR(64) UNIQUE NOT NULL,
  status VARCHAR(20) NOT NULL,
  severity VARCHAR(20),
  labels JSONB NOT NULL,
  annotations JSONB,
  starts_at TIMESTAMP WITH TIME ZONE NOT NULL,
  ends_at TIMESTAMP WITH TIME ZONE,
  generator_url TEXT,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  classifications JSONB,
  published_to JSONB
);

-- Indexes for performance
CREATE INDEX idx_alert_history_fingerprint ON alert_history(fingerprint);
CREATE INDEX idx_alert_history_status ON alert_history(status);
CREATE INDEX idx_alert_history_severity ON alert_history(severity);
CREATE INDEX idx_alert_history_created_at ON alert_history(created_at DESC);
CREATE INDEX idx_alert_history_labels_gin ON alert_history USING GIN(labels);
CREATE INDEX idx_alert_history_status_firing ON alert_history(status, severity)
  WHERE status = 'firing';
```

**Operations**:
- **SaveAlert**: Upsert (INSERT ... ON CONFLICT UPDATE)
- **GetHistory**: Paginated queries with filtering
- **GetTopAlerts**: Window functions for top N
- **GetFlappingAlerts**: Flapping detection via state transitions

**Performance**:
- Single insert: 3-4ms
- Query 100 alerts: 15-18ms
- Aggregation (stats): 20-25ms

---

#### Redis Cache Repository
**Location**: `go-app/internal/infrastructure/cache/redis_cache.go`
**Responsibility**: Two-tier caching

**Use Cases**:
1. **Classification Cache**: LLM results (24h TTL)
2. **Deduplication Cache**: Fingerprints (5min TTL)
3. **Message ID Cache**: Slack thread IDs (24h TTL)
4. **Enrichment Mode**: Current mode (Redis SET)

**Operations**:
- **Get/Set**: Standard key-value
- **SAdd/SMembers**: Set operations (for fingerprints)
- **TTL Management**: Automatic expiration
- **Pub/Sub**: Mode change notifications (future)

**Performance**: <1ms for all operations

---

### 4. External Integrations

#### LLM HTTP Client
**Location**: `go-app/internal/infrastructure/llm/http_client.go`
**Responsibility**: LLM API integration

**Supported Providers**:
- OpenAI (GPT-4, GPT-3.5)
- Anthropic (Claude 3)
- Custom (generic HTTP endpoint)

**Request**:
```json
{
  "model": "gpt-4",
  "messages": [{
    "role": "system",
    "content": "You are an alert classification expert..."
  }, {
    "role": "user",
    "content": "Alert: HighCPUUsage, instance: web-01, value: 95%"
  }],
  "temperature": 0.3,
  "max_tokens": 500
}
```

**Resilience**:
- **Timeout**: 10s (configurable)
- **Retry**: 3 attempts with exponential backoff
- **Fallback**: Rule-based classification
- **Circuit Breaker**: After 5 consecutive failures

---

#### Kubernetes Client
**Location**: `go-app/internal/infrastructure/k8s/client.go`
**Responsibility**: Target discovery via K8s API

**Operations**:
- **ListSecrets**: Discover publishing targets
- **GetSecret**: Retrieve target configuration
- **Watch**: React to secret changes (future)

**Label Selector**: `publishing-target=true`

**Secret Format**:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: slack-dev
  labels:
    publishing-target: "true"
type: Opaque
stringData:
  target.json: |
    {
      "name": "slack-dev",
      "type": "slack",
      "url": "https://hooks.slack.com/services/...",
      "headers": {
        "Content-Type": "application/json"
      }
    }
```

---

## Data Flow

### Alert Lifecycle

```
1. Alert Fired (Prometheus)
   │
   ▼
2. Ingestion (POST /webhook)
   - Validate & parse
   - Generate fingerprint
   │
   ▼
3. Deduplication
   - Check if seen (5min window)
   - Skip if duplicate
   │
   ▼
4. Grouping
   - Group by labels
   - Apply group_wait timer
   │
   ▼
5. Inhibition
   - Check inhibition rules
   - Suppress if matched
   │
   ▼
6. Classification (optional)
   - L1 cache hit? → Use cached
   - L2 cache hit? → Use cached
   - LLM API call → Classify & cache
   │
   ▼
7. Enrichment
   - Inject AI metadata
   - Add classification results
   │
   ▼
8. Filtering
   - Apply severity filters
   - Apply namespace filters
   │
   ▼
9. Storage
   - Save to PostgreSQL/SQLite
   - Update Redis cache
   │
   ▼
10. Publishing
    - Discover targets (K8s)
    - Health check targets
    - Parallel fanout
    - Retry on failure
    │
    ▼
11. Alert Delivered
    - Slack notification
    - PagerDuty incident
    - Rootly incident
    - Webhook POST
```

### State Transitions

```
┌─────────────┐
│   pending   │ ← Initial state
└──────┬──────┘
       │
       ▼
┌─────────────┐
│   firing    │ ← Active alert
└──────┬──────┘
       │
       ├──────────────┐
       │              │
       ▼              ▼
┌─────────────┐  ┌─────────────┐
│  inhibited  │  │  silenced   │
└─────────────┘  └─────────────┘
       │              │
       └──────┬───────┘
              │
              ▼
       ┌─────────────┐
       │  resolved   │ ← Final state
       └─────────────┘
```

---

## Deployment Architectures

### Lite Profile (Development/Testing)

```
┌────────────────────────────────────────┐
│  Kubernetes Namespace                  │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ alert-history (1 replica)        │ │
│  │ - CPU: 250m                      │ │
│  │ - Memory: 512Mi                  │ │
│  │ - Storage: SQLite (PVC 10Gi)     │ │
│  │ - Cache: Memory-only             │ │
│  └──────────────────────────────────┘ │
│                                        │
│  ┌──────────────────────────────────┐ │
│  │ PVC (ReadWriteOnce)              │ │
│  │ - Size: 10Gi                     │ │
│  │ - Mount: /data/alerthistory.db   │ │
│  └──────────────────────────────────┘ │
└────────────────────────────────────────┘

Use Case:
- Development environment
- Testing & staging
- Small-scale production (<1K alerts/day)
- Single-node requirement

Advantages:
✅ Zero external dependencies
✅ Fast startup (<10s)
✅ Simple deployment
✅ Low resource usage

Limitations:
❌ No horizontal scaling
❌ No high availability
❌ Limited to ~1K alerts/day
❌ PVC required (no true stateless)
```

### Standard Profile (Production)

```
┌──────────────────────────────────────────────────────────────┐
│  Kubernetes Namespace                                        │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ alert-history Deployment (2-10 replicas, HPA)         │ │
│  │                                                        │ │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐           │ │
│  │  │  Pod 1   │  │  Pod 2   │  │  Pod N   │           │ │
│  │  │ CPU: 500m│  │ CPU: 500m│  │ CPU: 500m│           │ │
│  │  │ Mem: 1Gi │  │ Mem: 1Gi │  │ Mem: 1Gi │           │ │
│  │  └────┬─────┘  └────┬─────┘  └────┬─────┘           │ │
│  │       │             │             │                   │ │
│  └───────┼─────────────┼─────────────┼───────────────────┘ │
│          │             │             │                     │
│          └─────────────┴─────────────┘                     │
│                        │                                    │
│          ┌─────────────┼─────────────┐                     │
│          │             │             │                     │
│          ▼             ▼             ▼                     │
│  ┌────────────┐ ┌────────────┐ ┌────────────┐            │
│  │PostgreSQL  │ │   Valkey   │ │ LLM Service│            │
│  │StatefulSet │ │StatefulSet │ │ (External) │            │
│  │(3 replicas)│ │(3 replicas)│ │            │            │
│  │            │ │            │ │            │            │
│  │ CPU: 500m  │ │ CPU: 100m  │ │            │            │
│  │ Mem: 2Gi   │ │ Mem: 256Mi │ │            │            │
│  │ PVC: 50Gi  │ │ PVC: 10Gi  │ │            │            │
│  └────────────┘ └────────────┘ └────────────┘            │
└──────────────────────────────────────────────────────────────┘

Use Case:
- Production environments
- High-volume (>1K alerts/day)
- High availability requirements
- Multi-tenant scenarios

Advantages:
✅ Horizontal auto-scaling (HPA)
✅ High availability (multi-replica)
✅ Extended history (PostgreSQL)
✅ High performance (Redis cache)
✅ Handles >10K alerts/day

Requirements:
- PostgreSQL (required)
- Redis/Valkey (optional, recommended)
- Storage class for PVCs
- Metrics server (for HPA)
```

### High Availability Setup

```
┌────────────────────────────────────────────────────────────┐
│  Multi-AZ Kubernetes Cluster                               │
│                                                            │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Zone A     │  │   Zone B     │  │   Zone C     │   │
│  │              │  │              │  │              │   │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │   │
│  │ │App Pod 1 │ │  │ │App Pod 2 │ │  │ │App Pod 3 │ │   │
│  │ └──────────┘ │  │ └──────────┘ │  │ └──────────┘ │   │
│  │              │  │              │  │              │   │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │   │
│  │ │PG Pod 1  │ │  │ │PG Pod 2  │ │  │ │PG Pod 3  │ │   │
│  │ │(Primary) │ │  │ │(Standby) │ │  │ │(Standby) │ │   │
│  │ └──────────┘ │  │ └──────────┘ │  │ └──────────┘ │   │
│  │              │  │              │  │              │   │
│  │ ┌──────────┐ │  │ ┌──────────┐ │  │ ┌──────────┐ │   │
│  │ │Redis 1   │ │  │ │Redis 2   │ │  │ │Redis 3   │ │   │
│  │ │(Master)  │ │  │ │(Replica) │ │  │ │(Replica) │ │   │
│  │ └──────────┘ │  │ └──────────┘ │  │ └──────────┘ │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└────────────────────────────────────────────────────────────┘

Features:
- Anti-affinity rules (spread pods across zones)
- PodDisruptionBudget (minAvailable: 1)
- PostgreSQL streaming replication
- Redis Sentinel for automatic failover
- Ingress with multiple replicas
```

---

## Technology Stack

### Application Layer

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Language** | Go | 1.22+ | High performance, concurrency |
| **Web Framework** | Gin | 1.9+ | HTTP routing, middleware |
| **Database** | PostgreSQL | 15+ | Primary data store (Standard) |
| **Database** | SQLite | 3.40+ | Embedded storage (Lite) |
| **Cache** | Redis/Valkey | 7+ | L2 cache, pub/sub |
| **ORM** | pgx | 5+ | PostgreSQL driver |
| **Migrations** | golang-migrate | 4+ | Schema migrations |

### Infrastructure

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Container** | Docker | 24+ | Containerization |
| **Orchestration** | Kubernetes | 1.25+ | Container orchestration |
| **Package Manager** | Helm | 3.12+ | K8s package management |
| **Service Mesh** | (Optional) Istio | 1.19+ | Traffic management, security |

### Observability

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **Metrics** | Prometheus | 2.45+ | Metrics collection |
| **Logging** | slog (stdlib) | Go 1.21+ | Structured logging |
| **Tracing** | OpenTelemetry | 1.20+ | Distributed tracing (optional) |
| **Dashboards** | Grafana | 10+ | Visualization |

### CI/CD

| Component | Technology | Version | Purpose |
|-----------|-----------|---------|---------|
| **CI** | GitHub Actions | - | Build, test, lint |
| **Registry** | GHCR | - | Container registry |
| **Linting** | golangci-lint | 1.55+ | Code quality |
| **Testing** | Go test | - | Unit & integration tests |

---

## Design Decisions

### 1. Why Go?

**Decision**: Use Go as primary language

**Rationale**:
- **Performance**: Native compilation, efficient memory usage
- **Concurrency**: Goroutines for parallel processing (publishing fanout)
- **Simplicity**: Easy to read, maintain, onboard new developers
- **Ecosystem**: Excellent K8s client libraries, HTTP frameworks
- **Deployment**: Single binary, minimal dependencies

**Alternatives Considered**:
- Python: Slower, GIL limits concurrency
- Java/JVM: Larger memory footprint, slower startup
- Rust: Steeper learning curve, smaller ecosystem

---

### 2. Two-Tier Caching

**Decision**: L1 (memory) + L2 (Redis) caching

**Rationale**:
- **L1 Cache**: Ultra-fast (<5ms), no network latency, limited size (1000 alerts)
- **L2 Cache**: Persistent across restarts, shared across pods, larger capacity
- **Hit Rate**: 95%+ combined, reduces LLM API costs by 95%
- **Graceful Degradation**: Falls back to L1 if Redis unavailable

**Performance**:
- L1 hit: <5ms
- L2 hit: <10ms
- LLM miss: <500ms

---

### 3. Deployment Profiles

**Decision**: Support two profiles (Lite vs Standard)

**Rationale**:
- **Lite**: Simplifies development/testing, zero external dependencies
- **Standard**: Production-grade HA, horizontal scaling
- **Flexibility**: Users choose based on requirements
- **Progressive Enhancement**: Start with Lite, upgrade to Standard

**Configuration**:
```yaml
# Lite Profile
profile: lite
storage:
  backend: filesystem
cache:
  enabled: false

# Standard Profile
profile: standard
storage:
  backend: postgres
postgresql:
  enabled: true
valkey:
  enabled: true
```

---

### 4. Fail-Safe Processing

**Decision**: Continue processing on non-critical errors

**Rationale**:
- **Availability**: Alert ingestion should never fail completely
- **Partial Success**: Process what we can, log what we can't
- **Graceful Degradation**: Skip classification if LLM down, skip publishing if target unhealthy

**Examples**:
- Classification fails → Use rule-based fallback
- Publishing fails → Retry with backoff, log error
- Storage fails → Return error, alert Prometheus

---

### 5. API Versioning

**Decision**: Use API v2 (`/api/v2/*`)

**Rationale**:
- **Backward Compatibility**: v1 preserved for legacy clients
- **Future-Proofing**: Easy to add v3 without breaking v2
- **Clear Contract**: Version in URL makes expectations clear

**Path Structure**:
```
/api/v2/alerts          # Alert ingestion
/api/v2/history         # Alert history queries
/api/v2/classification  # Classification management
/api/v2/publishing      # Publishing management
/api/v2/enrichment      # Enrichment mode
```

---

### 6. Horizontal Scaling

**Decision**: Stateless application pods with HPA

**Rationale**:
- **Scalability**: Add/remove pods based on load
- **High Availability**: Multiple replicas for redundancy
- **Cost Efficiency**: Scale down during low traffic

**HPA Configuration**:
```yaml
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
```

---

## Security Architecture

### Authentication & Authorization

```
┌────────────────────────────────────────┐
│  Ingress (TLS Termination)            │
│  - HTTPS only                          │
│  - TLS 1.2+                            │
│  - Cert-Manager integration            │
└────────┬───────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  API Gateway / Service Mesh (Optional) │
│  - mTLS between services               │
│  - JWT validation                      │
│  - Rate limiting                       │
└────────┬───────────────────────────────┘
         │
         ▼
┌────────────────────────────────────────┐
│  Alertmanager++ Application            │
│  - Bearer token validation             │
│  - RBAC enforcement                    │
│  - Audit logging                       │
└────────────────────────────────────────┘
```

### Security Layers

1. **Network Security**
   - **Ingress**: TLS 1.2+, HTTPS only
   - **Network Policies**: Restrict pod-to-pod traffic
   - **Service Mesh** (optional): mTLS between services

2. **Application Security**
   - **Input Validation**: All inputs validated (size, format, content)
   - **SQL Injection**: Parameterized queries (pgx)
   - **XSS Prevention**: Template auto-escaping (html/template)
   - **SSRF Protection**: URL validation, private IP blocking

3. **Secret Management**
   - **K8s Secrets**: Base64 encoded, RBAC controlled
   - **External Secrets Operator** (optional): Vault, AWS Secrets Manager
   - **Environment Variables**: Secrets injected at runtime
   - **Credential Rotation**: Automated via operators

4. **RBAC**
   - **ServiceAccount**: Least privilege principle
   - **Role/ClusterRole**: Read-only secrets access
   - **RoleBinding**: Namespace-scoped permissions

5. **Audit & Compliance**
   - **Structured Logging**: All actions logged
   - **Request IDs**: Tracing across services
   - **Metrics**: Security events tracked (auth failures, rate limits)

---

## Scalability & High Availability

### Horizontal Scalability

**Application Layer**:
- **Stateless Pods**: No local state, scale horizontally
- **HPA**: Auto-scale 2-10 replicas based on CPU/memory
- **Load Balancing**: K8s Service distributes traffic round-robin
- **Connection Pooling**: 20 DB connections per pod

**Database Layer**:
- **PostgreSQL**: Read replicas (future), connection pooling (pgBouncer)
- **Redis**: Master-replica setup, Sentinel for failover

**Publishing Layer**:
- **Parallel Fanout**: N goroutines for N targets
- **Circuit Breaker**: Isolate unhealthy targets
- **Health Checks**: Fail-fast, skip unhealthy targets

---

### High Availability

**Application Pods**:
- **Min Replicas**: 2 (always 2+ pods running)
- **Anti-Affinity**: Spread pods across nodes/zones
- **PodDisruptionBudget**: minAvailable: 1 (rolling updates)
- **Readiness/Liveness Probes**: Health checks

**PostgreSQL**:
- **Streaming Replication**: Primary + 2 standbys
- **Automatic Failover**: Patroni or similar
- **Backup**: Daily pg_dump to S3

**Redis**:
- **Sentinel**: Automatic failover
- **AOF Persistence**: appendonly + appendfsync everysec
- **Replication**: Master + 2 replicas

---

### Performance Optimization

1. **Caching**
   - L1 (memory): 1000 alerts, <5ms
   - L2 (Redis): 24h TTL, <10ms
   - Hit rate: 95%+

2. **Database**
   - **Indexes**: 7 indexes for common queries
   - **Connection Pooling**: 20 connections per pod
   - **Prepared Statements**: Reduce parsing overhead
   - **JSONB**: Efficient storage for labels/annotations

3. **Concurrency**
   - **Goroutines**: Parallel publishing (N targets)
   - **Worker Pool**: Bounded concurrency (10-20 workers)
   - **Non-blocking**: Async operations where possible

4. **Resource Limits**
   - **CPU**: 500m request, 2000m limit
   - **Memory**: 1Gi request, 4Gi limit
   - **Prevents noisy neighbor issues**

---

## Additional Resources

- **Deployment Guide**: [../deployment/DEPLOYMENT_GUIDE.md](../deployment/DEPLOYMENT_GUIDE.md)
- **Operations Runbook**: [../operations/RUNBOOK.md](../operations/RUNBOOK.md)
- **Troubleshooting**: [../operations/TROUBLESHOOTING.md](../operations/TROUBLESHOOTING.md)
- **API Documentation**: [../api/openapi.yaml](../api/openapi.yaml)
- **GitHub Repository**: https://github.com/ipiton/alert-history-service

---

**Questions or Feedback?**

- 💬 [GitHub Discussions](https://github.com/ipiton/alert-history-service/discussions)
- 🐛 [Report Issues](https://github.com/ipiton/alert-history-service/issues)
- 📖 [Read the Docs](https://ipiton.github.io/alert-history-service)
