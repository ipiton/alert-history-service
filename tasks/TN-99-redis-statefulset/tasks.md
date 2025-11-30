# TN-99: Redis/Valkey StatefulSet - Implementation Tasks

**Task ID**: TN-99
**Document Type**: Implementation Checklist
**Status**: 📝 **TASKS DEFINED**
**Target Quality**: **150% (Grade A+ EXCEPTIONAL)**
**Estimated Effort**: 22 hours
**Phase**: 13 (Production Packaging)

---

## Table of Contents

1. [Phase 0: Analysis (COMPLETE)](#phase-0-analysis-complete)
2. [Phase 1: Documentation (IN PROGRESS)](#phase-1-documentation-in-progress)
3. [Phase 2: Core Kubernetes Resources](#phase-2-core-kubernetes-resources)
4. [Phase 3: Monitoring & Observability](#phase-3-monitoring--observability)
5. [Phase 4: Security Hardening](#phase-4-security-hardening)
6. [Phase 5: Testing & Validation](#phase-5-testing--validation)
7. [Phase 6: Operational Documentation](#phase-6-operational-documentation)
8. [Phase 7: Integration & Finalization](#phase-7-integration--finalization)
9. [Task Summary](#task-summary)
10. [Git Workflow](#git-workflow)
11. [Quality Gates](#quality-gates)

---

## Phase 0: Analysis (COMPLETE)

**Status**: ✅ COMPLETE
**Duration**: 2 hours
**Deliverables**: COMPREHENSIVE_ANALYSIS.md (800+ LOC)

- [x] Project context analysis (Phase 13, dependencies TN-098, TN-200-203)
- [x] Current Redis integration audit (go-app/internal/infrastructure/cache/)
- [x] Dual-profile architecture analysis (Lite vs Standard)
- [x] Resource requirements calculation (CPU, memory, storage)
- [x] Performance target definition (cache hit rate, latency)
- [x] Risk assessment (6 risks with mitigations)
- [x] Success criteria definition (7 metrics)
- [x] Phase roadmap (0-7, 22 hours total)

**Completion**: 2025-11-30 ✅

---

## Phase 1: Documentation (IN PROGRESS)

**Status**: ⏳ 75% COMPLETE
**Duration**: 4 hours
**Deliverables**: requirements.md (963 LOC), design.md (1,970 LOC), tasks.md (600+ LOC)

### 1.1 Requirements Specification

- [x] Executive summary (project overview, key deliverables)
- [x] System requirements (15 functional, 10 non-functional)
- [x] Performance requirements (cache hit rate, latency, throughput)
- [x] Security requirements (authentication, RBAC, NetworkPolicy)
- [x] Operational requirements (backup, recovery, monitoring)
- [x] Dependencies (TN-098, TN-200-203)
- [x] Assumptions & constraints (Standard Profile only, single-node)
- [x] Success criteria (7 metrics with acceptance criteria)
- [x] Quality metrics (150% achievement targets)
- [x] Risk register (6 risks with mitigations)

**File**: `tasks/TN-99-redis-statefulset/requirements.md`
**LOC**: 963 (160% of 600 target) ✅
**Completion**: 2025-11-30 ✅

### 1.2 Technical Design

- [x] Architecture overview (diagram, component responsibilities, data flow)
- [x] StatefulSet design (manifest structure, pod lifecycle, resource requirements)
- [x] Configuration management (redis.conf, values.yaml)
- [x] Service architecture (headless, ClusterIP, metrics services)
- [x] Persistent storage design (PVC, volume layout, persistence mechanisms)
- [x] Monitoring & observability (redis-exporter, ServiceMonitor, alerts, dashboard)
- [x] Security architecture (NetworkPolicy, Secret management, RBAC)
- [x] Deployment workflow (initial deploy, upgrade, rollback)
- [x] Integration points (application, Helm chart, monitoring)
- [x] Testing strategy (Helm template, load tests, failover tests)
- [x] Operational procedures (backup, restore, maintenance)
- [x] Design decisions (Redis vs Valkey, single vs Sentinel HA, AOF settings)

**File**: `tasks/TN-99-redis-statefulset/design.md`
**LOC**: 1,970 (246% of 800 target) ✅
**Completion**: 2025-11-30 ✅

### 1.3 Implementation Checklist

- [x] Phase 0-7 task breakdown
- [x] Detailed subtasks with acceptance criteria
- [x] Git workflow strategy (commit messages, PR description)
- [x] Quality gates (documentation, implementation, testing)
- [ ] Completion report template (COMPLETION_REPORT.md structure)
- [ ] Success metrics tracking (table format for easy updates)

**File**: `tasks/TN-99-redis-statefulset/tasks.md` (THIS FILE)
**Target LOC**: 600+
**Expected Completion**: 2025-11-30

---

## Phase 2: Core Kubernetes Resources

**Status**: ⏳ PENDING
**Duration**: 6 hours
**Deliverables**: 4 Kubernetes manifests (~1,000 LOC)

### 2.1 Redis StatefulSet

**File**: `helm/alert-history/templates/redis-statefulset.yaml`
**Target LOC**: 400

- [ ] StatefulSet metadata (name, namespace, labels)
- [ ] Replica configuration (replicas: 1, expandable to 3)
- [ ] Update strategy (RollingUpdate, partition: 0)
- [ ] Pod template (labels, annotations, checksum)
- [ ] Service account (reuse existing)
- [ ] Security context (pod-level: fsGroup 999, runAsUser 999)
- [ ] Init container (config-init):
  - [ ] Copy redis.conf from ConfigMap
  - [ ] Inject password from Secret
  - [ ] Set permissions (chmod 644)
- [ ] Redis container:
  - [ ] Image configuration (redis:7-alpine or valkey/valkey:7-alpine)
  - [ ] Command (redis-server /data/redis.conf)
  - [ ] Port configuration (6379)
  - [ ] Environment variables (REDIS_PASSWORD from Secret)
  - [ ] Resource limits (500m CPU, 512Mi memory)
  - [ ] Liveness probe (redis-cli ping, 30s initial delay)
  - [ ] Readiness probe (redis-cli ping, 5s initial delay)
  - [ ] Startup probe (redis-cli ping, 30 attempts × 5s)
  - [ ] Volume mounts (/data, /usr/local/etc/redis)
  - [ ] Security context (container-level)
- [ ] redis-exporter sidecar:
  - [ ] Image configuration (oliver006/redis_exporter:v1.55.0)
  - [ ] Environment variables (REDIS_ADDR, REDIS_PASSWORD)
  - [ ] Port configuration (9121)
  - [ ] Resource limits (100m CPU, 128Mi memory)
  - [ ] Liveness/readiness probes (HTTP /metrics)
  - [ ] Security context (readOnlyRootFilesystem: true)
- [ ] Volumes (ConfigMap, Secret)
- [ ] Node affinity (optional, values.yaml controlled)
- [ ] Tolerations (optional, values.yaml controlled)
- [ ] Pod anti-affinity (preferredDuringScheduling)
- [ ] Volume claim template (5Gi PVC per replica)
- [ ] Profile conditional ({{- if eq .Values.profile "standard" }})

**Acceptance Criteria**:
- [ ] Helm template renders without errors (`helm template`)
- [ ] Profile check works (standard ✅, lite ❌)
- [ ] All probes configured correctly
- [ ] Resource limits within budget
- [ ] Security contexts applied
- [ ] PVC provisioning works
- [ ] No linter errors (`helm lint`)

**Git Commit**: `feat(TN-99): Phase 2a - Redis StatefulSet manifest (400+ LOC)`

### 2.2 Redis Configuration ConfigMap

**File**: `helm/alert-history/templates/redis-config.yaml`
**Target LOC**: 300

- [ ] ConfigMap metadata (name, namespace, labels)
- [ ] redis.conf data:
  - [ ] Network configuration (bind, port, protected-mode, tcp-keepalive)
  - [ ] General configuration (daemonize, pidfile, loglevel, databases)
  - [ ] RDB snapshotting (save rules: 900/1, 300/10, 60/10000)
  - [ ] RDB settings (rdbcompression, rdbchecksum, dbfilename, dir)
  - [ ] Replication settings (commented for future HA)
  - [ ] Security (requirepass placeholder, rename dangerous commands)
  - [ ] Memory management (maxmemory 384mb, allkeys-lru policy)
  - [ ] Lazy freeing (lazyfree-lazy-eviction, replica-lazy-flush)
  - [ ] AOF persistence (appendonly yes, appendfsync everysec)
  - [ ] AOF settings (auto-aof-rewrite-percentage 100, aof-use-rdb-preamble)
  - [ ] Lua scripting (lua-time-limit 5000)
  - [ ] Slow log (slowlog-log-slower-than 10000, slowlog-max-len 128)
  - [ ] Latency monitor (latency-monitor-threshold 100)
  - [ ] Advanced config (hash/list/set/zset ziplist settings)
- [ ] values.yaml template injection ({{.Values.valkey.settings.*}})
- [ ] Profile conditional ({{- if eq .Values.profile "standard" }})

**Acceptance Criteria**:
- [ ] ConfigMap renders correctly
- [ ] All critical settings present (maxmemory, appendonly, etc.)
- [ ] Template variables work (loglevel, maxmemory, etc.)
- [ ] No syntax errors in redis.conf
- [ ] File parseable by Redis (`redis-server --test-memory 1`)

**Git Commit**: `feat(TN-99): Phase 2b - Redis ConfigMap with comprehensive redis.conf (300+ LOC)`

### 2.3 Redis Services (3 services)

**File**: `helm/alert-history/templates/redis-service.yaml`
**Target LOC**: 150

- [ ] Headless Service (StatefulSet DNS):
  - [ ] Metadata (name: alerthistory-redis-headless)
  - [ ] Type: ClusterIP, clusterIP: None
  - [ ] Selector (app.kubernetes.io/component: redis)
  - [ ] Port 6379
  - [ ] publishNotReadyAddresses: true
- [ ] ClusterIP Service (Application connections):
  - [ ] Metadata (name: alerthistory-redis)
  - [ ] Type: ClusterIP
  - [ ] Selector (pod-name: alerthistory-redis-0, primary only)
  - [ ] Port 6379
  - [ ] SessionAffinity: None
- [ ] Metrics Service (Prometheus scraping):
  - [ ] Metadata (name: alerthistory-redis-metrics)
  - [ ] Type: ClusterIP
  - [ ] Selector (app.kubernetes.io/component: redis)
  - [ ] Port 9121
  - [ ] Annotations (prometheus.io/scrape: true, port: 9121)
- [ ] Profile conditional for all 3 services

**Acceptance Criteria**:
- [ ] All 3 services render correctly
- [ ] DNS records created (headless service)
- [ ] Application can connect via alerthistory-redis:6379
- [ ] Prometheus can scrape metrics via alerthistory-redis-metrics:9121
- [ ] Service selectors match StatefulSet labels

**Git Commit**: `feat(TN-99): Phase 2c - Redis Services (headless, ClusterIP, metrics) 150 LOC`

### 2.4 values.yaml Integration

**File**: `helm/alert-history/values.yaml`
**Target LOC**: ~100 (additions/modifications)

- [ ] Add `valkey:` section:
  - [ ] enabled: true
  - [ ] replicas: 1
  - [ ] image (repository, tag, pullPolicy)
  - [ ] resources (limits, requests)
  - [ ] storage (className, requestedSize: 5Gi)
  - [ ] settings (maxmemory, maxmemoryPolicy, appendonly, appendfsync, loglevel, slowlogThreshold)
  - [ ] exporter (enabled, image, tag, resources)
  - [ ] password (existingSecret, secretKey)
  - [ ] networkPolicy (enabled: false by default)
- [ ] Update `profile:` section documentation
- [ ] Ensure backward compatibility (Lite profile unaffected)

**Acceptance Criteria**:
- [ ] values.yaml schema valid (`helm lint`)
- [ ] Defaults sensible (5Gi storage, 384mb maxmemory, etc.)
- [ ] Profile conditional works (standard enables Redis, lite disables)
- [ ] All settings templated correctly in manifests

**Git Commit**: `feat(TN-99): Phase 2d - values.yaml integration for Redis/Valkey (100+ LOC)`

---

## Phase 3: Monitoring & Observability

**Status**: ⏳ PENDING
**Duration**: 4 hours
**Deliverables**: 3 manifests (~850 LOC)

### 3.1 redis-exporter Sidecar

**Status**: Already in StatefulSet manifest (Phase 2.1)
**LOC**: ~100 (part of StatefulSet)

- [x] Covered in Phase 2.1 StatefulSet

### 3.2 ServiceMonitor CRD

**File**: `helm/alert-history/templates/redis-servicemonitor.yaml`
**Target LOC**: 50

- [ ] ServiceMonitor metadata (name, namespace, labels)
- [ ] Selector (matchLabels: app.kubernetes.io/component: redis-metrics)
- [ ] Endpoints configuration:
  - [ ] Port: metrics
  - [ ] Interval: 30s
  - [ ] ScrapeTimeout: 10s
  - [ ] Path: /metrics
  - [ ] Scheme: http
- [ ] Relabelings:
  - [ ] sourceLabels: [__meta_kubernetes_pod_name] → targetLabel: pod
  - [ ] sourceLabels: [__meta_kubernetes_pod_node_name] → targetLabel: node
- [ ] Profile conditional
- [ ] Monitoring conditional ({{- if .Values.monitoring.prometheusEnabled }})

**Acceptance Criteria**:
- [ ] ServiceMonitor renders correctly
- [ ] Prometheus Operator discovers ServiceMonitor
- [ ] Metrics appear in Prometheus targets
- [ ] Scraping successful (up{job="alerthistory-redis"} == 1)

**Git Commit**: `feat(TN-99): Phase 3a - ServiceMonitor CRD for Prometheus auto-discovery (50 LOC)`

### 3.3 PrometheusRule with 10 Alerts

**File**: `helm/alert-history/templates/redis-prometheusrule.yaml`
**Target LOC**: 200

- [ ] PrometheusRule metadata
- [ ] Alert group: redis.critical (5 alerts):
  - [ ] RedisDown (redis_up == 0 for 1m)
  - [ ] RedisOutOfMemory (memory >90% for 5m)
  - [ ] RedisTooManyConnections (>8000 for 5m)
  - [ ] RedisRejectedConnections (increase >0 in 5m)
  - [ ] RedisPersistenceFailure (rdb_last_bgsave_status == 0 for 10m)
- [ ] Alert group: redis.warning (5 alerts):
  - [ ] RedisHighMemoryUsage (memory >75% for 10m)
  - [ ] RedisHighConnectionUsage (>6000 for 10m)
  - [ ] RedisSlowQueries (slowlog_length >10 for 5m)
  - [ ] RedisReplicationLag (future HA, slave offset lag >1000)
  - [ ] RedisLowHitRate (hit rate <80% for 15m)
- [ ] All alerts with:
  - [ ] severity label
  - [ ] component: redis label
  - [ ] summary annotation
  - [ ] description annotation (with {{$value}})
  - [ ] runbook_url annotation (docs.alertmanager-plus-plus.io)
- [ ] Profile conditional
- [ ] Monitoring conditional

**Acceptance Criteria**:
- [ ] PrometheusRule renders correctly
- [ ] All 10 alerts present in Prometheus
- [ ] Alert expressions valid (no syntax errors)
- [ ] Runbook URLs valid (placeholder OK for now)
- [ ] Test alerts fire when thresholds crossed

**Git Commit**: `feat(TN-99): Phase 3b - PrometheusRule with 10 alerting rules (200 LOC)`

### 3.4 Grafana Dashboard JSON

**File**: `helm/alert-history/templates/redis-dashboard-configmap.yaml`
**Target LOC**: 500

**Option 1: Import existing dashboard**:
- [ ] ConfigMap metadata
- [ ] Dashboard JSON from https://grafana.com/grafana/dashboards/11835
- [ ] Dashboard ID: 11835 (Redis Dashboard for Prometheus Redis Exporter)
- [ ] Datasource variable: Prometheus
- [ ] 12 panels (uptime, clients, memory, commands/s, hit rate, network I/O, keyspace, evicted/expired, persistence, slow queries, fragmentation, connection age)

**Option 2: Custom dashboard** (if Option 1 not suitable):
- [ ] Create custom Grafana dashboard JSON
- [ ] Minimum 12 panels (matching imported dashboard)
- [ ] Use redis-exporter metrics
- [ ] Include alerting states

**Acceptance Criteria**:
- [ ] ConfigMap renders correctly
- [ ] Dashboard importable to Grafana (`curl -X POST ...`)
- [ ] All panels display data (no "No data" errors)
- [ ] Dashboard UID unique
- [ ] Variables work (datasource, instance)

**Git Commit**: `feat(TN-99): Phase 3c - Grafana dashboard ConfigMap (500+ LOC JSON)`

---

## Phase 4: Security Hardening

**Status**: ⏳ PENDING
**Duration**: 3 hours
**Deliverables**: 2 manifests (~150 LOC)

### 4.1 NetworkPolicy

**File**: `helm/alert-history/templates/redis-networkpolicy.yaml`
**Target LOC**: 80

- [ ] NetworkPolicy metadata
- [ ] Pod selector (app.kubernetes.io/component: redis)
- [ ] Policy types (Ingress, Egress)
- [ ] Ingress rules:
  - [ ] Allow from app pods (selector: app.kubernetes.io/component: application)
  - [ ] Port 6379
  - [ ] Allow from Prometheus (namespace: monitoring, port 9121)
- [ ] Egress rules:
  - [ ] Allow DNS resolution (namespace: kube-system, port 53 UDP)
  - [ ] Allow API server (port 6443, for future Sentinel)
- [ ] Profile conditional
- [ ] NetworkPolicy conditional ({{- if .Values.valkey.networkPolicy.enabled }})

**Acceptance Criteria**:
- [ ] NetworkPolicy renders correctly
- [ ] App pods can connect to Redis
- [ ] Prometheus can scrape metrics
- [ ] External connections blocked (test from outside namespace)
- [ ] DNS resolution works
- [ ] No false-positive connection failures

**Git Commit**: `feat(TN-99): Phase 4a - NetworkPolicy for pod isolation (80 LOC)`

### 4.2 Secret Management

**File**: `helm/alert-history/templates/redis-secret.yaml`
**Target LOC**: 40

**Manual Secret** (default):
- [ ] Secret metadata
- [ ] Type: Opaque
- [ ] data.password: Base64 encoded (from values.yaml or randAlphaNum 32)
- [ ] Profile conditional
- [ ] Conditional: Only create if .Values.valkey.password.existingSecret is empty

**External Secrets Operator Integration** (future, TN-100):
- [ ] Add commented section for ExternalSecret
- [ ] Documentation on how to enable (reference TN-100)

**Acceptance Criteria**:
- [ ] Secret renders correctly
- [ ] Password retrieved in init container
- [ ] Redis requirepass set correctly (verify with redis-cli AUTH)
- [ ] Secret not exposed in logs
- [ ] existingSecret conditional works (skip creation if provided)

**Git Commit**: `feat(TN-99): Phase 4b - Secret management with password (40 LOC)`

### 4.3 RBAC (No Additional Required)

**Status**: Reuse existing ServiceAccount
**File**: `helm/alert-history/templates/serviceaccount.yaml` (already exists)

- [x] No additional RBAC required for Redis pod
- [x] StatefulSet doesn't need K8s API access
- [x] ServiceAccount already created for app pods

**Note**: No additional work needed. Documented in design.md section 7.3.

---

## Phase 5: Testing & Validation

**Status**: ⏳ PENDING
**Duration**: 4 hours
**Deliverables**: 4 test scripts/manifests (~400 LOC)

### 5.1 Helm Template Rendering Tests

**File**: `scripts/test-redis-helm-templates.sh`
**Target LOC**: 80

- [ ] Test 1: Template renders for Standard Profile
  - [ ] `helm template --set profile=standard | grep "kind: StatefulSet"`
  - [ ] Expected: 1 StatefulSet found
- [ ] Test 2: No Redis for Lite Profile
  - [ ] `helm template --set profile=lite | grep -c "kind: StatefulSet"`
  - [ ] Expected: 0 (PostgreSQL only)
- [ ] Test 3: ConfigMap rendered correctly
  - [ ] `helm template --set profile=standard | grep -A 5 "maxmemory"`
  - [ ] Expected: maxmemory 384mb
- [ ] Test 4: Services created
  - [ ] `helm template --set profile=standard | grep -c "kind: Service"`
  - [ ] Expected: 6 (3 app + 3 Redis)
- [ ] Test 5: ServiceMonitor conditional
  - [ ] With monitoring: `--set monitoring.prometheusEnabled=true`
  - [ ] Without monitoring: `--set monitoring.prometheusEnabled=false`
- [ ] All tests executable with `bash test-redis-helm-templates.sh`

**Acceptance Criteria**:
- [ ] All 5 tests pass
- [ ] Script exits 0 on success
- [ ] Color-coded output (green pass, red fail)
- [ ] No false positives

**Git Commit**: `test(TN-99): Phase 5a - Helm template rendering tests (80 LOC)`

### 5.2 Connection Pool Load Tests (k6)

**File**: `k6/redis-connection-pool.js`
**Target LOC**: 120

- [ ] k6 script configuration:
  - [ ] Stages: Ramp up to 500 connections, hold 5min, ramp down
  - [ ] VUs: 500 virtual users
  - [ ] Duration: 7 minutes total
- [ ] Redis client initialization (k6/x/redis)
- [ ] Test SET operation (check successful)
- [ ] Test GET operation (check value matches)
- [ ] Metrics collection (request duration, success rate)
- [ ] Thresholds:
  - [ ] http_req_duration (p95 < 50ms)
  - [ ] http_req_failed (rate < 0.01)
- [ ] Comments explaining test scenarios

**Acceptance Criteria**:
- [ ] k6 script runs without errors
- [ ] 500 concurrent connections handled
- [ ] No connection rejections
- [ ] p95 latency < 50ms
- [ ] Success rate >99%
- [ ] Redis memory usage stable

**Git Commit**: `test(TN-99): Phase 5b - k6 connection pool load test (120 LOC)`

### 5.3 Failover Simulation Test

**File**: `scripts/test-redis-failover.sh`
**Target LOC**: 100

- [ ] Step 1: Write test data (redis-cli SET test-key test-value)
- [ ] Step 2: Verify data written (redis-cli GET test-key)
- [ ] Step 3: Delete pod (kubectl delete pod alerthistory-redis-0)
- [ ] Step 4: Wait for pod recreation (kubectl wait --for=condition=ready)
- [ ] Step 5: Verify data persisted (redis-cli GET test-key)
- [ ] Step 6: Verify AOF replay (kubectl logs | grep "Loading DB")
- [ ] All steps with error handling and timeouts
- [ ] Color-coded output

**Acceptance Criteria**:
- [ ] Test passes (data persisted after pod deletion)
- [ ] Recovery time <60s
- [ ] No data loss (AOF replay successful)
- [ ] Script exits 0 on success

**Git Commit**: `test(TN-99): Phase 5c - Failover simulation test with AOF validation (100 LOC)`

### 5.4 Persistence Validation Test

**File**: `scripts/test-redis-persistence.sh`
**Target LOC**: 100

- [ ] Step 1: Check AOF enabled (redis-cli CONFIG GET appendonly)
- [ ] Step 2: Write test data (1000 keys)
- [ ] Step 3: Verify AOF file created (ls -lh /data/appendonly.aof)
- [ ] Step 4: Force RDB snapshot (redis-cli SAVE)
- [ ] Step 5: Verify RDB file created (ls -lh /data/dump.rdb)
- [ ] Step 6: Verify both files exist (ls -lh /data/)
- [ ] All steps with error handling
- [ ] Color-coded output

**Acceptance Criteria**:
- [ ] Both AOF and RDB files created
- [ ] AOF file size >0
- [ ] RDB snapshot created within 5s
- [ ] All 1000 keys persisted
- [ ] Script exits 0 on success

**Git Commit**: `test(TN-99): Phase 5d - Persistence validation (AOF + RDB) test (100 LOC)`

---

## Phase 6: Operational Documentation

**Status**: ⏳ PENDING
**Duration**: 3 hours
**Deliverables**: 3 operational guides (~1,700 LOC)

### 6.1 Redis Operations Guide

**File**: `tasks/TN-99-redis-statefulset/REDIS_OPERATIONS_GUIDE.md`
**Target LOC**: 800

- [ ] Table of Contents
- [ ] Section 1: Overview
  - [ ] Purpose
  - [ ] Architecture summary
  - [ ] Key components
- [ ] Section 2: Deployment
  - [ ] Initial deployment (Helm install commands)
  - [ ] Configuration options (values.yaml)
  - [ ] Verification steps (kubectl commands)
- [ ] Section 3: Day-2 Operations
  - [ ] Scaling (future: 1 → 3 replicas)
  - [ ] Configuration updates (maxmemory, appendonly, etc.)
  - [ ] Upgrading Redis version
  - [ ] Rolling updates
- [ ] Section 4: Backup & Restore
  - [ ] Manual backup procedures (RDB, AOF)
  - [ ] Automated backup setup (CronJob)
  - [ ] Restore procedures (step-by-step)
  - [ ] RTO/RPO expectations
- [ ] Section 5: Monitoring & Alerting
  - [ ] Key metrics to watch
  - [ ] Grafana dashboard usage
  - [ ] Alert interpretation (10 alerts)
  - [ ] PromQL query examples
- [ ] Section 6: Performance Tuning
  - [ ] Memory optimization
  - [ ] Connection pool tuning
  - [ ] AOF rewrite optimization
  - [ ] Slow query analysis
- [ ] Section 7: Security Operations
  - [ ] Password rotation
  - [ ] NetworkPolicy updates
  - [ ] RBAC auditing
  - [ ] Security scanning
- [ ] Section 8: Common Tasks
  - [ ] Flushing cache (redis-cli FLUSHDB)
  - [ ] Restarting Redis gracefully
  - [ ] Checking Redis status
  - [ ] Analyzing memory usage

**Acceptance Criteria**:
- [ ] All sections complete (1-8)
- [ ] All commands tested and verified
- [ ] No placeholder TODOs
- [ ] Readable formatting (headings, code blocks)
- [ ] 800+ LOC

**Git Commit**: `docs(TN-99): Phase 6a - Redis Operations Guide (800+ LOC)`

### 6.2 Troubleshooting Guide

**File**: `tasks/TN-99-redis-statefulset/TROUBLESHOOTING.md`
**Target LOC**: 500

- [ ] Table of Contents
- [ ] Section 1: Common Issues
  - [ ] Redis pod not starting
    - [ ] Symptoms
    - [ ] Root causes (PVC provisioning, image pull, config errors)
    - [ ] Diagnosis steps
    - [ ] Resolution steps
  - [ ] Connection refused errors
    - [ ] Symptoms
    - [ ] Root causes (NetworkPolicy, Service selector, password mismatch)
    - [ ] Diagnosis steps
    - [ ] Resolution steps
  - [ ] Out of memory errors
    - [ ] Symptoms
    - [ ] Root causes (maxmemory exceeded, memory leak, large keys)
    - [ ] Diagnosis steps
    - [ ] Resolution steps
  - [ ] High latency / slow queries
    - [ ] Symptoms
    - [ ] Root causes (network, large keys, slow commands)
    - [ ] Diagnosis steps (SLOWLOG, redis-cli --latency)
    - [ ] Resolution steps
  - [ ] Persistence failures (RDB/AOF)
    - [ ] Symptoms
    - [ ] Root causes (disk full, permissions, corruption)
    - [ ] Diagnosis steps
    - [ ] Resolution steps
- [ ] Section 2: Debugging Commands
  - [ ] kubectl logs alerthistory-redis-0 -c redis
  - [ ] kubectl exec alerthistory-redis-0 -- redis-cli INFO
  - [ ] kubectl describe pod alerthistory-redis-0
  - [ ] kubectl get events --field-selector involvedObject.name=alerthistory-redis-0
- [ ] Section 3: Performance Debugging
  - [ ] redis-cli --latency
  - [ ] redis-cli --bigkeys
  - [ ] redis-cli --memkeys
  - [ ] redis-cli SLOWLOG GET
- [ ] Section 4: Escalation Paths
  - [ ] When to escalate
  - [ ] Required information for escalation
  - [ ] Contact points (placeholder: support@alertmanager-plus-plus.io)

**Acceptance Criteria**:
- [ ] All common issues covered (5+ issues)
- [ ] Each issue with symptoms, root causes, diagnosis, resolution
- [ ] All commands tested
- [ ] Escalation paths defined
- [ ] 500+ LOC

**Git Commit**: `docs(TN-99): Phase 6b - Troubleshooting Guide (500+ LOC)`

### 6.3 Disaster Recovery Guide

**File**: `tasks/TN-99-redis-statefulset/DISASTER_RECOVERY.md`
**Target LOC**: 400

- [ ] Table of Contents
- [ ] Section 1: Disaster Scenarios
  - [ ] Complete data loss (PVC deleted)
  - [ ] Cluster failure (node unavailable)
  - [ ] Corruption (AOF/RDB corrupted)
  - [ ] Accidental FLUSHALL (all keys deleted)
- [ ] Section 2: Recovery Procedures
  - [ ] Data loss recovery:
    - [ ] Step 1: Restore from backup (RDB/AOF)
    - [ ] Step 2: Rebuild cache from PostgreSQL classification history
    - [ ] Step 3: Verify data integrity
    - [ ] RTO: <10 minutes
  - [ ] Cluster failure recovery:
    - [ ] Step 1: Wait for node recovery or reschedule pod
    - [ ] Step 2: Verify PVC reattached
    - [ ] Step 3: Verify AOF replay successful
    - [ ] RTO: <5 minutes
  - [ ] Corruption recovery:
    - [ ] Step 1: Disable AOF (appendonly no)
    - [ ] Step 2: Restore from last good RDB snapshot
    - [ ] Step 3: Re-enable AOF
    - [ ] RTO: <5 minutes
  - [ ] Accidental FLUSHALL recovery:
    - [ ] Step 1: Stop AOF fsync (CONFIG SET appendfsync no)
    - [ ] Step 2: Restore from RDB snapshot (before FLUSHALL)
    - [ ] Step 3: Re-enable AOF
    - [ ] RPO: Max 15 minutes (last RDB save)
- [ ] Section 3: RTO/RPO Matrix
  - [ ] Table with all scenarios, RTO, RPO, steps
- [ ] Section 4: Testing DR Procedures
  - [ ] Quarterly DR drills (schedule)
  - [ ] Test scenarios (pod deletion, data loss simulation)
  - [ ] Success criteria
- [ ] Section 5: Backup Validation
  - [ ] How to test backups
  - [ ] Restore to non-production environment
  - [ ] Verification steps

**Acceptance Criteria**:
- [ ] All disaster scenarios covered (4+ scenarios)
- [ ] Recovery procedures detailed (step-by-step)
- [ ] RTO/RPO documented
- [ ] DR testing procedures defined
- [ ] 400+ LOC

**Git Commit**: `docs(TN-99): Phase 6c - Disaster Recovery Guide (400+ LOC)`

---

## Phase 7: Integration & Finalization

**Status**: ⏳ PENDING
**Duration**: 2 hours
**Deliverables**: Main project files updated, COMPLETION_REPORT.md

### 7.1 Main tasks.md Updates

**File**: `tasks/alertmanager-plus-plus-oss/TASKS.md`
**Changes**: ~20 lines

- [ ] Mark TN-99 as COMPLETE
- [ ] Add completion date (2025-11-30 or actual)
- [ ] Add quality achievement (150%+ target)
- [ ] Add LOC delivered (~4,000+)
- [ ] Update Phase 13 progress (60% → 80%, 4/5 tasks)
- [ ] Update P2 summary (2/25 → 3/25)
- [ ] Update Total Progress (75/114 → 76/114)

**Acceptance Criteria**:
- [ ] TN-99 status updated
- [ ] Phase 13 percentages correct
- [ ] Total progress accurate
- [ ] No formatting errors

**Git Commit**: `docs(TN-99): Phase 7a - Update main tasks.md (TN-99 complete)`

### 7.2 CHANGELOG.md Entry

**File**: `CHANGELOG.md`
**Changes**: ~100 lines (comprehensive entry)

- [ ] Add new section: `## [Unreleased]` or `## [2025-11-30]`
- [ ] Subsection: `### Added - TN-99: Redis/Valkey StatefulSet (Standard Profile)`
- [ ] List all deliverables:
  - [ ] StatefulSet with persistent storage (5Gi PVC)
  - [ ] ConfigMap with production-grade redis.conf
  - [ ] 3 Services (headless, ClusterIP, metrics)
  - [ ] redis-exporter sidecar (50+ metrics)
  - [ ] ServiceMonitor CRD (Prometheus auto-discovery)
  - [ ] PrometheusRule with 10 alerting rules
  - [ ] Grafana dashboard (12 panels)
  - [ ] NetworkPolicy (pod isolation)
  - [ ] Secret management (password)
  - [ ] Comprehensive documentation (3,700+ LOC: requirements, design, tasks, ops guides)
  - [ ] Testing suite (4 test scripts: Helm, k6, failover, persistence)
- [ ] Performance improvements:
  - [ ] L2 cache with <10ms average latency
  - [ ] 95%+ cache hit rate (L1 + L2)
  - [ ] 384MB cache capacity (100K alerts × 2KB)
  - [ ] AOF persistence with <1s RPO
- [ ] Quality achievement: 150% (Grade A+ EXCEPTIONAL)
- [ ] Files changed: 20+ files, 4,000+ LOC
- [ ] Link to TN-99 documentation: `tasks/TN-99-redis-statefulset/`

**Acceptance Criteria**:
- [ ] Entry comprehensive (all deliverables listed)
- [ ] CHANGELOG format valid (KeepAChangelog style)
- [ ] Links work
- [ ] No duplicate entries

**Git Commit**: `docs(TN-99): Phase 7b - CHANGELOG entry (comprehensive TN-99 summary)`

### 7.3 Completion Report

**File**: `tasks/TN-99-redis-statefulset/COMPLETION_REPORT.md`
**Target LOC**: 600

- [ ] Executive Summary
  - [ ] Task overview
  - [ ] Status (COMPLETE, 150%+ quality)
  - [ ] Timeline (start/end dates, duration)
  - [ ] Team (contributors)
- [ ] Deliverables Summary
  - [ ] Table with all files, LOC, status
  - [ ] Total LOC: ~4,000+
  - [ ] Files created: 20+
- [ ] Quality Metrics
  - [ ] Documentation: 3,700+ LOC (185% of 2,000 target)
  - [ ] Kubernetes manifests: 1,000+ LOC (125% of 800 target)
  - [ ] Test suite: 400+ LOC (133% of 300 target)
  - [ ] Overall quality: 150%+ (Grade A+ EXCEPTIONAL)
- [ ] Performance Results
  - [ ] Cache hit rate: 95%+ (target 90%+)
  - [ ] Average latency: <10ms (target <20ms)
  - [ ] Throughput: 1,000+ req/s (target 500+)
  - [ ] Storage efficiency: 10.7% utilization (4.5GB headroom)
- [ ] Testing Results
  - [ ] Helm template tests: 5/5 passing ✅
  - [ ] k6 load test: 500 connections, <50ms p95 ✅
  - [ ] Failover test: <60s recovery, zero data loss ✅
  - [ ] Persistence test: AOF + RDB both working ✅
- [ ] Integration Status
  - [ ] Application integration: ✅ COMPLETE (already implemented in TN-201)
  - [ ] Helm chart: ✅ COMPLETE (values.yaml updated)
  - [ ] Monitoring: ✅ COMPLETE (ServiceMonitor, PrometheusRule, Grafana)
  - [ ] Security: ✅ COMPLETE (NetworkPolicy, Secret, RBAC)
- [ ] Production Readiness Checklist (30/30)
  - [ ] All items checked ✅
- [ ] Lessons Learned
  - [ ] What went well
  - [ ] What could be improved
  - [ ] Recommendations for future tasks
- [ ] Next Steps
  - [ ] TN-100: External Secrets Operator integration (optional)
  - [ ] Future: Sentinel HA mode (3 replicas)
- [ ] Certification
  - [ ] Quality grade: A+ (EXCEPTIONAL)
  - [ ] Approved for production deployment: ✅
  - [ ] Date: 2025-11-30
  - [ ] Signed: Vitalii Semenov

**Acceptance Criteria**:
- [ ] All sections complete
- [ ] Metrics accurate (verified with `wc -l`)
- [ ] Production readiness checklist 100%
- [ ] Certification included
- [ ] 600+ LOC

**Git Commit**: `docs(TN-99): Phase 7c - Completion Report (600+ LOC, Grade A+)`

---

## Task Summary

### Total LOC Breakdown

| Phase | Component | Target LOC | Status |
|-------|-----------|------------|--------|
| **Phase 0** | COMPREHENSIVE_ANALYSIS.md | 800 | ✅ COMPLETE (800) |
| **Phase 1** | requirements.md | 600 | ✅ COMPLETE (963, 160%) |
| **Phase 1** | design.md | 800 | ✅ COMPLETE (1,970, 246%) |
| **Phase 1** | tasks.md | 600 | ⏳ IN PROGRESS (est. 700+) |
| **Phase 2** | redis-statefulset.yaml | 400 | ⏳ PENDING |
| **Phase 2** | redis-config.yaml | 300 | ⏳ PENDING |
| **Phase 2** | redis-service.yaml | 150 | ⏳ PENDING |
| **Phase 2** | values.yaml integration | 100 | ⏳ PENDING |
| **Phase 3** | redis-servicemonitor.yaml | 50 | ⏳ PENDING |
| **Phase 3** | redis-prometheusrule.yaml | 200 | ⏳ PENDING |
| **Phase 3** | redis-dashboard-configmap.yaml | 500 | ⏳ PENDING |
| **Phase 4** | redis-networkpolicy.yaml | 80 | ⏳ PENDING |
| **Phase 4** | redis-secret.yaml | 40 | ⏳ PENDING |
| **Phase 5** | test-redis-helm-templates.sh | 80 | ⏳ PENDING |
| **Phase 5** | k6-redis-connection-pool.js | 120 | ⏳ PENDING |
| **Phase 5** | test-redis-failover.sh | 100 | ⏳ PENDING |
| **Phase 5** | test-redis-persistence.sh | 100 | ⏳ PENDING |
| **Phase 6** | REDIS_OPERATIONS_GUIDE.md | 800 | ⏳ PENDING |
| **Phase 6** | TROUBLESHOOTING.md | 500 | ⏳ PENDING |
| **Phase 6** | DISASTER_RECOVERY.md | 400 | ⏳ PENDING |
| **Phase 7** | COMPLETION_REPORT.md | 600 | ⏳ PENDING |
| **TOTAL** | **All Components** | **7,320** | **Est. 9,000+ (123%)** |

### Progress Tracking

| Phase | Status | LOC Complete | LOC Remaining | % Complete |
|-------|--------|--------------|---------------|------------|
| Phase 0 | ✅ COMPLETE | 800 | 0 | 100% |
| Phase 1 | ⏳ 75% | 2,933 | 700 | 75% |
| Phase 2 | ⏳ 0% | 0 | 950 | 0% |
| Phase 3 | ⏳ 0% | 0 | 750 | 0% |
| Phase 4 | ⏳ 0% | 0 | 120 | 0% |
| Phase 5 | ⏳ 0% | 0 | 400 | 0% |
| Phase 6 | ⏳ 0% | 0 | 1,700 | 0% |
| Phase 7 | ⏳ 0% | 0 | 600 | 0% |
| **TOTAL** | **⏳ 40%** | **3,733** | **5,220** | **40%** |

---

## Git Workflow

### Branch Strategy

**Feature Branch**: `feature/TN-99-redis-statefulset-150pct`

```bash
# Create feature branch (already done)
git checkout -b feature/TN-99-redis-statefulset-150pct

# Commit pattern: <type>(TN-99): <subject>
# Types: feat, docs, test, chore
```

### Commit Messages

**Phase 0**:
- ✅ `docs(TN-99): Phase 0 - Comprehensive Analysis (800 LOC)`

**Phase 1**:
- ✅ `docs(TN-99): Phase 1a - requirements.md (963 LOC, 160% achievement)`
- ✅ `docs(TN-99): Phase 1b - design.md (1,970 LOC, 246% achievement)`
- ⏳ `docs(TN-99): Phase 1c - tasks.md (700+ LOC implementation checklist)`

**Phase 2**:
- ⏳ `feat(TN-99): Phase 2a - Redis StatefulSet manifest (400+ LOC)`
- ⏳ `feat(TN-99): Phase 2b - Redis ConfigMap with comprehensive redis.conf (300+ LOC)`
- ⏳ `feat(TN-99): Phase 2c - Redis Services (headless, ClusterIP, metrics) 150 LOC`
- ⏳ `feat(TN-99): Phase 2d - values.yaml integration for Redis/Valkey (100+ LOC)`

**Phase 3**:
- ⏳ `feat(TN-99): Phase 3a - ServiceMonitor CRD for Prometheus (50 LOC)`
- ⏳ `feat(TN-99): Phase 3b - PrometheusRule with 10 alerting rules (200 LOC)`
- ⏳ `feat(TN-99): Phase 3c - Grafana dashboard ConfigMap (500+ LOC JSON)`

**Phase 4**:
- ⏳ `feat(TN-99): Phase 4a - NetworkPolicy for pod isolation (80 LOC)`
- ⏳ `feat(TN-99): Phase 4b - Secret management with password (40 LOC)`

**Phase 5**:
- ⏳ `test(TN-99): Phase 5a - Helm template rendering tests (80 LOC)`
- ⏳ `test(TN-99): Phase 5b - k6 connection pool load test (120 LOC)`
- ⏳ `test(TN-99): Phase 5c - Failover simulation test (100 LOC)`
- ⏳ `test(TN-99): Phase 5d - Persistence validation test (100 LOC)`

**Phase 6**:
- ⏳ `docs(TN-99): Phase 6a - Redis Operations Guide (800+ LOC)`
- ⏳ `docs(TN-99): Phase 6b - Troubleshooting Guide (500+ LOC)`
- ⏳ `docs(TN-99): Phase 6c - Disaster Recovery Guide (400+ LOC)`

**Phase 7**:
- ⏳ `docs(TN-99): Phase 7a - Update main tasks.md (TN-99 complete)`
- ⏳ `docs(TN-99): Phase 7b - CHANGELOG entry (comprehensive TN-99 summary)`
- ⏳ `docs(TN-99): Phase 7c - Completion Report (600+ LOC, Grade A+)`

### Final Commit Strategy

**Squash or Multi-commit Merge?**
- **Recommendation**: Multi-commit merge (preserve history)
- **Rationale**: Each phase is substantial (200-1,000 LOC), atomic commits useful for review

**Merge to Main**:
```bash
# Before merge: Ensure all commits clean
git log --oneline feature/TN-99-redis-statefulset-150pct

# Merge to main
git checkout main
git merge --no-ff feature/TN-99-redis-statefulset-150pct

# Push to origin
git push origin main
```

### Pull Request Description Template

```markdown
# TN-99: Redis/Valkey StatefulSet - Standard Profile Only

## Summary
Implement production-ready Redis/Valkey StatefulSet for Alertmanager++ OSS Core Standard Profile, enabling persistent L2 caching, distributed state management, and HA-ready infrastructure.

## Quality Achievement
- **Target**: 150% (Grade A+ EXCEPTIONAL)
- **Actual**: 160%+ (estimated)
- **Duration**: 22 hours (on target)

## Deliverables (4,000+ LOC)
- ✅ 4 Kubernetes manifests (StatefulSet, ConfigMap, Services, PVC)
- ✅ 3 Monitoring manifests (ServiceMonitor, PrometheusRule, Grafana dashboard)
- ✅ 2 Security manifests (NetworkPolicy, Secret)
- ✅ 4 Test scripts (Helm, k6, failover, persistence)
- ✅ 4 Documentation files (requirements, design, tasks, completion report)
- ✅ 3 Operational guides (operations, troubleshooting, disaster recovery)

## Key Features
- Persistent L2 cache with 5Gi PVC (AOF + RDB)
- redis-exporter sidecar (50+ metrics)
- 10 Prometheus alerting rules (5 critical, 5 warning)
- NetworkPolicy pod isolation
- Password authentication via Secret
- Comprehensive operational documentation

## Testing
- ✅ Helm template rendering (5/5 tests passing)
- ✅ k6 load test (500 connections, <50ms p95)
- ✅ Failover test (<60s recovery, zero data loss)
- ✅ Persistence test (AOF + RDB validated)

## Production Readiness: 100% ✅
- Zero breaking changes
- Zero linter errors
- Zero technical debt
- Backward compatible (Lite profile unaffected)

## Dependencies
- TN-098: PostgreSQL StatefulSet ✅ (completed)
- TN-200: Deployment Profile Configuration ✅ (completed)
- TN-201: Storage Backend Selection ✅ (completed)

## Next Steps
- TN-100: External Secrets Operator integration (optional)
- Future: Sentinel HA mode (3 replicas)

## Documentation
- `tasks/TN-99-redis-statefulset/` (3,700+ LOC)
- Operational guides: operations, troubleshooting, disaster recovery

## Reviewers
@vitalii-semenov (self-review, Grade A+)
```

---

## Quality Gates

### Documentation Quality Gate (Phase 1)

**Criteria**:
- [x] requirements.md ≥600 LOC ✅ (963, 160%)
- [x] design.md ≥800 LOC ✅ (1,970, 246%)
- [ ] tasks.md ≥600 LOC ⏳ (est. 700+)
- [ ] All sections complete (no TODOs)
- [ ] All diagrams present (architecture diagram in design.md)
- [ ] No spelling errors (visual check)
- [ ] No broken links (internal references only at this stage)

**Gate Status**: ⏳ 66% (2/3 files complete)

### Implementation Quality Gate (Phase 2-4)

**Criteria**:
- [ ] All Kubernetes manifests created (9 files)
- [ ] `helm template` renders without errors
- [ ] `helm lint` passes (zero warnings)
- [ ] Profile conditional works (standard ✅, lite ❌)
- [ ] values.yaml defaults sensible
- [ ] No hardcoded passwords (use Secret)
- [ ] SecurityContext applied (runAsNonRoot: true)
- [ ] Resource limits set (CPU, memory)
- [ ] Probes configured (liveness, readiness, startup)
- [ ] PVC provisioning tested (manual kubectl apply)

**Gate Status**: ⏳ 0% (not started)

### Testing Quality Gate (Phase 5)

**Criteria**:
- [ ] All 4 test scripts created
- [ ] Helm template tests passing (5/5)
- [ ] k6 load test passing (<50ms p95, 500 connections)
- [ ] Failover test passing (<60s recovery, zero data loss)
- [ ] Persistence test passing (AOF + RDB both created)
- [ ] No test failures (100% pass rate)
- [ ] Test scripts executable (`bash test-*.sh`)
- [ ] Color-coded output (green pass, red fail)

**Gate Status**: ⏳ 0% (not started)

### Final Quality Gate (Phase 7)

**Criteria**:
- [ ] All phases complete (0-7)
- [ ] Total LOC ≥7,000 (current target: 9,000+)
- [ ] 150%+ quality achieved (calculated across all deliverables)
- [ ] COMPLETION_REPORT.md created (600+ LOC)
- [ ] Main tasks.md updated (TN-99 marked complete)
- [ ] CHANGELOG.md updated (comprehensive entry)
- [ ] All tests passing (Helm, k6, failover, persistence)
- [ ] Zero linter errors (`helm lint`, `yamllint`)
- [ ] Zero breaking changes (Lite profile unaffected)
- [ ] Production readiness checklist 100% (30/30)
- [ ] Grade A+ certification approved

**Gate Status**: ⏳ 0% (not started)

---

## Success Criteria Tracking

| Criterion | Target | Actual | Status | Achievement |
|-----------|--------|--------|--------|-------------|
| **Total LOC** | 7,000+ | TBD | ⏳ | TBD |
| **Documentation LOC** | 2,000+ | 3,733 | ✅ | 187% |
| **Kubernetes Manifests** | 9 files | 0 | ⏳ | 0% |
| **Test Scripts** | 4 files | 0 | ⏳ | 0% |
| **Quality Grade** | A+ (150%+) | TBD | ⏳ | TBD |
| **Helm Lint** | 0 errors | TBD | ⏳ | TBD |
| **Test Pass Rate** | 100% | TBD | ⏳ | TBD |
| **Production Ready** | 30/30 checklist | TBD | ⏳ | TBD |
| **Breaking Changes** | 0 | TBD | ⏳ | TBD |
| **Duration** | 22h | TBD | ⏳ | TBD |

---

**Document Version**: 1.0
**Last Updated**: 2025-11-30
**Author**: Vitalii Semenov (AI-assisted)
**Status**: ✅ TASKS DEFINED - READY TO IMPLEMENT
**Next**: Phase 2 - Core Kubernetes Resources Implementation
