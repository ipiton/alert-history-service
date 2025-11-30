# TN-99: Redis/Valkey StatefulSet - Requirements Specification

**Task ID**: TN-99
**Task Name**: Redis/Valkey StatefulSet - Standard Profile Only
**Priority**: P2 (Production Packaging)
**Status**: 📝 **IN PROGRESS**
**Target Quality**: **150% (Grade A+ EXCEPTIONAL)**
**Estimated Effort**: 22 hours
**Phase**: 13 (Production Packaging)

---

## Executive Summary

Implement a **production-ready Redis/Valkey StatefulSet** for Alertmanager++ OSS Core's **Standard Profile**, enabling persistent L2 caching, distributed state management, and high-availability support for 2-10 application replicas. This task delivers enterprise-grade caching infrastructure with **150% quality**, including comprehensive monitoring, security hardening, and operational runbooks.

**Key Deliverables**:
- Redis StatefulSet with 5Gi persistent storage
- Production-tuned configuration (AOF + RDB persistence)
- Comprehensive monitoring (50+ metrics, 10 alerts, Grafana dashboard)
- Security hardening (NetworkPolicy, Secret management)
- Complete operational documentation (2,000+ LOC)

---

## 1. Business Context

### 1.1 Project Overview

**Alertmanager++ OSS Core** is a complete replacement for Prometheus Alertmanager with AI/ML classification capabilities. The project uses a **dual-profile architecture**:

| Profile | Target Use Case | External Dependencies | Redis Role |
|---------|-----------------|----------------------|-----------|
| **Lite** | Dev, testing, <1K alerts/day | ❌ None (zero deps) | N/A (memory-only) |
| **Standard** | Production, >1K alerts/day, HA | ✅ PostgreSQL + Redis | **L2 Cache (REQUIRED)** |

### 1.2 Redis Role in Standard Profile

Redis/Valkey serves **4 critical functions**:

1. **L2 Cache** (PRIMARY)
   - Classification results caching (200MB data, 100K alerts)
   - Two-tier strategy: L1 (memory, <5ms) → L2 (Redis, <10ms) → LLM API (~500ms)
   - Cache hit rate: 93-97% combined L1+L2
   - Cost savings: 95% reduction in LLM API calls

2. **Timer Persistence** (TN-124)
   - Group Wait/Interval timers for HA recovery
   - Max 1,000 concurrent groups × 500B = 500KB
   - Critical for zero-downtime restarts

3. **Inhibition State** (TN-129)
   - Distributed state management across 2-10 replicas
   - Max 10,000 concurrent inhibitions × 1KB = 10MB
   - Enables HA coordination

4. **Session Management** (Future)
   - Rate limiting counters (global)
   - Cross-instance coordination
   - Distributed locks

**Total Memory Requirement**: 210.5MB data + 20% overhead = **252.5MB**
**Provisioned**: 384MB (maxmemory) with 131.5MB headroom (52% buffer) ✅

### 1.3 Business Impact

**Without Redis/Valkey** (Lite Profile):
- ✅ Zero external dependencies
- ✅ Simpler deployment
- ❌ Memory-only cache (lost on restart)
- ❌ No HA support (single replica only)
- ❌ Limited to ~1K alerts/day

**With Redis/Valkey** (Standard Profile):
- ✅ Persistent L2 cache (survives restarts)
- ✅ HA-ready (2-10 replicas)
- ✅ Cost optimization (95% LLM API savings)
- ✅ Production-grade (>1K alerts/day)
- ⚠️ Requires external dependency

**ROI Calculation**:
```
LLM API cost without cache: $0.002/classification × 10K alerts/day = $20/day = $7,300/year
LLM API cost with L2 cache: $20/day × 5% (miss rate) = $1/day = $365/year
Annual savings: $6,935/year per deployment
Payback: Immediate (infrastructure cost << savings)
```

---

## 2. Functional Requirements

### FR-1: Redis StatefulSet Deployment

**Priority**: CRITICAL
**Description**: Deploy Redis/Valkey as Kubernetes StatefulSet with persistent storage

**Acceptance Criteria**:
- StatefulSet manifest created with 1 replica (expandable to 3 for future HA)
- Pod naming: `alerthistory-redis-0`, `alerthistory-redis-1`, `alerthistory-redis-2`
- Persistent Volume Claim per pod: 5Gi (adjustable via values.yaml)
- Container image: `redis:7-alpine` or `valkey/valkey:7-alpine` (configurable)
- Resource limits: CPU 500m, Memory 512Mi
- Resource requests: CPU 250m, Memory 256Mi
- Init container for configuration setup
- Graceful shutdown handling (TERM signal, 30s grace period)

**Implementation Details**:
```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis
spec:
  replicas: 1  # Expandable to 3 for Sentinel mode
  serviceName: {{ include "alerthistory.fullname" . }}-redis-headless
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "alerthistory.name" . }}-redis
  volumeClaimTemplates:
  - metadata:
      name: redis-data
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: {{ .Values.valkey.storage.requestedSize }}
```

**Success Metrics**:
- Pod achieves Running state within 30s
- Volume binds successfully
- Redis responds to PING within 5s

---

### FR-2: Production-Tuned Redis Configuration

**Priority**: CRITICAL
**Description**: Provide enterprise-grade redis.conf with optimized settings for Standard Profile

**Acceptance Criteria**:

**Memory Management**:
- `maxmemory 384mb` (75% of 512Mi limit)
- `maxmemory-policy allkeys-lru` (evict least recently used)
- `maxmemory-samples 5` (LRU sample size)

**Persistence (Hybrid AOF + RDB)**:
- `appendonly yes` (enable AOF)
- `appendfsync everysec` (balance durability/performance)
- `auto-aof-rewrite-percentage 100` (rewrite when 2x size)
- `auto-aof-rewrite-min-size 64mb` (minimum size for rewrite)
- `save 900 1` (RDB snapshot: 1 change in 15 min)
- `save 300 10` (RDB snapshot: 10 changes in 5 min)
- `save 60 10000` (RDB snapshot: 10K changes in 1 min)

**Performance Tuning**:
- `tcp-keepalive 300` (detect dead connections)
- `timeout 0` (no client timeout)
- `tcp-backlog 511` (connection queue size)
- `databases 1` (single database, save memory)

**Connection Limits**:
- `maxclients 10000` (default, sufficient for 500 connections from app)

**Logging & Monitoring**:
- `loglevel notice` (balanced verbosity)
- `logfile ""` (stdout for K8s logging)
- `slowlog-log-slower-than 10000` (log queries >10ms)
- `slowlog-max-len 128` (keep last 128 slow queries)

**Security**:
- `requirepass ${REDIS_PASSWORD}` (from Secret)
- `protected-mode yes` (reject external connections)
- `rename-command CONFIG ""` (disable dangerous commands)

**Configuration Storage**:
- ConfigMap: `alerthistory-redis-config`
- Mount path: `/usr/local/etc/redis/redis.conf`
- Template variables: `{{ .Values.valkey.settings.* }}`

**Success Metrics**:
- Configuration loads without errors
- Memory usage stable at <384MB
- AOF file created within 1s of first write
- RDB snapshots triggered according to schedule

---

### FR-3: Three-Tier Service Configuration

**Priority**: CRITICAL
**Description**: Create multiple Kubernetes Services for different access patterns

**Acceptance Criteria**:

**1. Headless Service** (StatefulSet DNS):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-headless
spec:
  clusterIP: None  # Headless
  selector:
    app.kubernetes.io/name: {{ include "alerthistory.name" . }}-redis
  ports:
  - name: redis
    port: 6379
    targetPort: 6379
```
- Purpose: StatefulSet pod DNS (e.g., `alerthistory-redis-0.alerthistory-redis-headless.default.svc.cluster.local`)
- Used by: Future Sentinel mode, direct pod access

**2. ClusterIP Service** (App Connections):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: {{ include "alerthistory.name" . }}-redis
    statefulset.kubernetes.io/pod-name: {{ include "alerthistory.fullname" . }}-redis-0  # Primary only
  ports:
  - name: redis
    port: 6379
    targetPort: 6379
```
- Purpose: Application connections (load-balanced to primary)
- Used by: Go app Redis client (`cfg.Redis.Addr`)

**3. Metrics Service** (Prometheus Scraping):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-metrics
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9121"
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/name: {{ include "alerthistory.name" . }}-redis
  ports:
  - name: metrics
    port: 9121
    targetPort: 9121
```
- Purpose: Expose redis-exporter metrics
- Used by: Prometheus ServiceMonitor

**Success Metrics**:
- All 3 services created successfully
- DNS resolution works for headless service
- App connects to ClusterIP service
- Prometheus scrapes metrics service

---

### FR-4: Persistent Storage Management

**Priority**: CRITICAL
**Description**: Ensure data durability through persistent volumes

**Acceptance Criteria**:

**Volume Claim Template**:
```yaml
volumeClaimTemplates:
- metadata:
    name: redis-data
  spec:
    accessModes: [ "ReadWriteOnce" ]
    storageClassName: {{ .Values.valkey.storage.className | default "" | quote }}
    resources:
      requests:
        storage: {{ .Values.valkey.storage.requestedSize | default "5Gi" }}
```

**Mount Configuration**:
```yaml
volumeMounts:
- name: redis-data
  mountPath: /data
- name: redis-config
  mountPath: /usr/local/etc/redis
  readOnly: true
```

**Data Paths**:
- AOF file: `/data/appendonly.aof` (default Redis path)
- RDB snapshot: `/data/dump.rdb`
- Configuration: `/usr/local/etc/redis/redis.conf` (readonly)

**Retention Policy**:
- PVC retention: `Retain` (manual cleanup)
- Backup frequency: RDB snapshots (15min/5min/1min triggers)
- AOF fsync: Every 1 second (everysec)

**Storage Sizing**:
```
Data requirement: 252.5MB (base)
Growth factor: 2x (500MB)
Safety margin: 10x (5GB) ✅
Actual size: 5Gi (configurable)
Usage at capacity: 500MB / 5GB = 10% utilization
```

**Success Metrics**:
- PVC binds within 30s
- AOF file persists after pod restart
- RDB snapshots created successfully
- Storage monitoring alerts configured

---

### FR-5: Connection Pool Integration

**Priority**: HIGH
**Description**: Ensure Redis supports application connection requirements

**Acceptance Criteria**:

**Connection Requirements**:
```
App Replicas: 10 (max via HPA - TN-97)
Connections per replica: 50 (PoolSize in go-app/internal/infrastructure/cache/redis.go)
Total connections: 10 × 50 = 500 connections
```

**Redis Configuration**:
```
maxclients: 10,000 (default)
Active connections: 500
Utilization: 500 / 10,000 = 5% ✅
Headroom: 9,500 connections (19x overhead)
```

**Go Client Configuration** (already implemented):
```go
// go-app/internal/infrastructure/cache/redis.go:39-51
client := redis.NewClient(&redis.Options{
    Addr:            config.Addr,           // Service ClusterIP
    Password:        config.Password,        // From Secret
    DB:              config.DB,              // 0 (single database)
    PoolSize:        config.PoolSize,        // 50 per pod
    MinIdleConns:    config.MinIdleConns,    // 1
    DialTimeout:     config.DialTimeout,     // 5s
    ReadTimeout:     config.ReadTimeout,     // 3s
    WriteTimeout:    config.WriteTimeout,    // 3s
    MaxRetries:      config.MaxRetries,      // 3
    MinRetryBackoff: config.MinRetryBackoff, // 8ms
    MaxRetryBackoff: config.MaxRetryBackoff, // 512ms
})
```

**Connection Lifecycle**:
1. App pod starts → initializes Redis client
2. Client creates connection pool (PoolSize: 50)
3. Connections lazily created on first request
4. Idle connections maintained (MinIdleConns: 1)
5. Dead connections detected (tcp-keepalive: 300s)
6. Retry on failure (MaxRetries: 3, exponential backoff)
7. Graceful shutdown closes all connections

**Monitoring**:
- Track active connections: `connected_clients` metric
- Alert on high utilization: `> 8,000 connections (80%)`
- Monitor connection errors: `rejected_connections` metric

**Success Metrics**:
- All 500 connections established successfully
- Connection pool warm-up <5s
- Zero connection rejections
- Graceful connection cleanup on shutdown

---

### FR-6: Profile-Based Conditional Deployment

**Priority**: HIGH
**Description**: Redis StatefulSet deployed ONLY for Standard Profile

**Acceptance Criteria**:

**Helm Template Conditional**:
```yaml
# helm/alert-history/templates/redis-statefulset.yaml
{{- if eq .Values.profile "standard" }}
apiVersion: apps/v1
kind: StatefulSet
# ... StatefulSet definition ...
{{- end }}
```

**Values.yaml Integration**:
```yaml
# Profile determines deployment
profile: "standard"  # "lite" | "standard"

# Valkey configuration (Standard Profile only)
valkey:
  enabled: true  # Managed by profile conditional
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 256Mi
  storage:
    className: ""
    requestedSize: 5Gi
  settings:
    maxmemory: 384mb
    maxmemoryPolicy: allkeys-lru
    appendonly: "yes"
    appendfsync: everysec
```

**App Configuration** (already implemented):
```go
// go-app/cmd/server/main.go:359-409
if cfg.Profile == appconfig.ProfileLite {
    // Lite: Skip Redis (memory-only)
    redisCache = nil
} else if cfg.Profile == appconfig.ProfileStandard && cfg.Redis.Addr != "" {
    // Standard: Initialize Redis
    redisCache, err = cache.NewRedisCache(&cacheConfig, appLogger)
}
```

**Deployment Behavior**:

| Profile | StatefulSet Deployed | PVC Created | App Behavior |
|---------|---------------------|-------------|--------------|
| **Lite** | ❌ No | ❌ No | Memory-only cache |
| **Standard** | ✅ Yes | ✅ Yes | L1 (memory) + L2 (Redis) |

**Upgrade Path**:
- Lite → Standard: Redis StatefulSet auto-created on `helm upgrade`
- Standard → Lite: Redis StatefulSet remains (manual cleanup required)

**Success Metrics**:
- Helm template renders correctly for both profiles
- Redis StatefulSet created only for Standard Profile
- App detects Redis unavailability and falls back to memory-only

---

## 3. Non-Functional Requirements

### NFR-1: Performance

**Priority**: CRITICAL

**Latency Targets**:
- Connection establishment: <10ms p95
- GET operation: <1ms p95
- SET operation: <2ms p95
- AOF fsync overhead: <5% CPU
- Connection pool warm-up: <5s

**Throughput Targets**:
- Operations per second: >10,000 ops/sec
- Concurrent connections: 500 (10 replicas × 50 pool)
- Network bandwidth: <10 Mbps (typical)

**Cache Hit Rate**:
- Combined L1+L2: >93%
- L1 (memory): ~90%
- L2 (Redis): ~3% (fills L1)
- LLM API miss: ~7%

**Success Metrics**:
- All latency targets met at p95
- Throughput sustained under load (k6 test)
- Zero performance degradation over 7 days

---

### NFR-2: Reliability & Availability

**Priority**: CRITICAL

**Recovery Time Objective (RTO)**:
- Pod crash recovery: <30s (AOF replay)
- Volume reattachment: <60s
- Service restoration: <90s total

**Recovery Point Objective (RPO)**:
- Data loss on crash: <1s (AOF everysec)
- Complete data loss: <15min (RDB snapshot)

**Failure Modes & Responses**:

| Failure | Detection Time | Recovery Action | RTO | RPO |
|---------|---------------|-----------------|-----|-----|
| **Pod crash** | Immediate (kubelet) | Restart + AOF replay | <30s | <1s |
| **Node failure** | <30s (K8s) | Reschedule + volume reattach | <60s | <1s |
| **Volume corruption** | Manual | Restore from RDB | <5min | <15min |
| **Complete loss** | Manual | Rebuild from PostgreSQL | <10min | Full rebuild |

**High Availability (Future)**:
- Single primary (current): 99.5% uptime (4h 23m downtime/year)
- Sentinel mode (future): 99.95% uptime (26min downtime/year)

**Success Metrics**:
- RTO targets met in failover tests
- RPO verified through data loss tests
- Uptime monitoring >99.5%

---

### NFR-3: Scalability

**Priority**: HIGH

**Vertical Scaling**:
- Memory: 512Mi → 1Gi → 2Gi (adjustable via values.yaml)
- CPU: 500m → 1000m (adjustable)
- Storage: 5Gi → 10Gi → 20Gi (volume expansion supported)

**Horizontal Scaling** (Future):
- Replicas: 1 → 3 (Sentinel mode)
- Read replicas: Offload read traffic
- Sharding: Not required (single dataset <500MB)

**Connection Scaling**:
- Current: 500 connections (10 app replicas)
- Headroom: 9,500 connections (19x)
- Max theoretical: 10,000 connections (maxclients)

**Data Growth**:
```
Current: 252.5MB (100K alerts cached)
1 year: 252.5MB × 2 = 505MB (assume 2x growth)
5 years: 252.5MB × 5 = 1.26GB
Storage: 5Gi (5,120MB) accommodates 10+ years growth ✅
```

**Success Metrics**:
- Supports 10 app replicas without degradation
- Memory usage <80% of limit
- Storage usage <50% of capacity

---

### NFR-4: Security

**Priority**: HIGH

**Authentication**:
- Redis password: Required (from Secret)
- TLS/SSL: Recommended for production (future)
- Protected mode: Enabled (reject external connections)

**Network Isolation**:
- NetworkPolicy: Allow only app pods (label selector)
- Service type: ClusterIP (internal only)
- Ingress: Disabled (no external access)

**Secret Management**:
```yaml
# Manual Secret (default)
apiVersion: v1
kind: Secret
metadata:
  name: alerthistory-redis-secret
type: Opaque
data:
  password: <base64-encoded-password>

# External Secrets Operator (future - TN-100)
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: alerthistory-redis-secret
spec:
  secretStoreRef:
    name: aws-secrets-manager
  target:
    name: alerthistory-redis-secret
  data:
  - secretKey: password
    remoteRef:
      key: alertmanager-plus-plus/redis-password
```

**Dangerous Command Restriction**:
- `CONFIG`: Renamed to prevent runtime reconfiguration
- `FLUSHDB`: Kept (useful for testing)
- `FLUSHALL`: Kept (useful for testing)
- `SHUTDOWN`: Kept (graceful shutdown)

**Success Metrics**:
- Password authentication working
- NetworkPolicy blocks unauthorized access
- Secret rotation ready (via ESO)

---

### NFR-5: Observability

**Priority**: CRITICAL

**Metrics Collection**:
- **Source**: redis-exporter (sidecar container)
- **Endpoint**: `:9121/metrics`
- **Scrape Interval**: 30s (Prometheus)
- **Metrics Count**: 50+ metrics

**Key Metrics** (minimum 50):

**Connection Metrics**:
- `redis_connected_clients` (Gauge)
- `redis_connected_slaves` (Gauge)
- `redis_rejected_connections_total` (Counter)

**Memory Metrics**:
- `redis_memory_used_bytes` (Gauge)
- `redis_memory_max_bytes` (Gauge)
- `redis_memory_fragmentation_ratio` (Gauge)

**Performance Metrics**:
- `redis_commands_total` (Counter by command)
- `redis_commands_duration_seconds_total` (Counter by command)
- `redis_keyspace_hits_total` (Counter)
- `redis_keyspace_misses_total` (Counter)

**Persistence Metrics**:
- `redis_aof_last_rewrite_duration_sec` (Gauge)
- `redis_rdb_last_save_timestamp_seconds` (Gauge)
- `redis_rdb_changes_since_last_save` (Gauge)

**Replication Metrics** (Future):
- `redis_connected_slaves` (Gauge)
- `redis_replication_lag_seconds` (Gauge)

**Alerting Rules** (10 total):

**Critical Alerts** (5):
1. `RedisDown` - Redis unavailable
2. `RedisOutOfMemory` - Memory >90% of maxmemory
3. `RedisTooManyConnections` - Connections >80% of maxclients
4. `RedisRejectedConnections` - Rejected connections detected
5. `RedisPersistenceFailure` - AOF/RDB write failures

**Warning Alerts** (5):
6. `RedisHighMemoryUsage` - Memory >75% of maxmemory
7. `RedisHighConnectionUsage` - Connections >60% of maxclients
8. `RedisSlowQueries` - Slow queries detected
9. `RedisReplicationLag` - Lag >5s (future HA)
10. `RedisLowHitRate` - Cache hit rate <80%

**Grafana Dashboard**:
- Panel count: 12 panels
- Refresh interval: 30s
- Time range: Last 6 hours
- Metrics: All 50+ metrics visualized

**Success Metrics**:
- All 50+ metrics exposed successfully
- Prometheus scrapes without errors
- All 10 alerts configured and firing correctly
- Grafana dashboard renders successfully

---

### NFR-6: Maintainability

**Priority**: HIGH

**Operational Documentation**:
- REDIS_OPERATIONS_GUIDE.md (800+ LOC)
- TROUBLESHOOTING.md (500+ LOC)
- DISASTER_RECOVERY.md (400+ LOC)

**Upgrade Path**:
- Rolling updates: Zero downtime (single primary)
- Configuration changes: Requires pod restart
- Volume expansion: Online (if StorageClass supports)

**Backup & Restore**:
- Automatic: RDB snapshots (15min/5min/1min)
- Manual: `kubectl exec` + `redis-cli SAVE`
- Restore: Copy RDB to `/data/dump.rdb` + restart

**Monitoring & Alerts**:
- Prometheus alerts configured
- Grafana dashboard provided
- Runbook for common issues

**Success Metrics**:
- Operational documentation complete
- Zero-downtime upgrade tested
- Backup/restore procedures validated

---

## 4. Constraints & Assumptions

### 4.1 Technical Constraints

**Kubernetes**:
- Kubernetes version: 1.21+ (StatefulSet stable)
- Persistent volumes: ReadWriteOnce (RWO) required
- Service type: ClusterIP only (no LoadBalancer)

**Redis/Valkey**:
- Version: Redis 7.x or Valkey 7.x (latest stable)
- Architecture: Single primary (no clustering)
- Persistence: Local disk only (no remote backup)

**Networking**:
- Cluster network required
- No external access (internal only)
- DNS resolution required (headless service)

### 4.2 Assumptions

**Deployment Environment**:
- Standard Profile only (Lite Profile skips Redis)
- Kubernetes cluster with PV provisioner
- Prometheus monitoring installed
- Grafana dashboard support (optional)

**Capacity Planning**:
- Max 10 app replicas (HPA target)
- Max 100K alerts cached (~200MB)
- Max 1K concurrent groups (~500KB)
- Storage growth: 2x per year

**Operational**:
- Ops team trained on Redis operations
- Backup automation via CronJob (future)
- Disaster recovery plan documented

---

## 5. Success Criteria

### 5.1 Baseline Requirements (100%)

1. ✅ Redis StatefulSet deploys successfully
2. ✅ Persistent volumes bound correctly
3. ✅ App pods connect successfully (500 connections)
4. ✅ L2 cache working (>93% hit rate)
5. ✅ AOF persistence enabled (everysec)
6. ✅ RDB snapshots created successfully
7. ✅ Pod restart triggers AOF replay (<30s)
8. ✅ redis-exporter exposes 50+ metrics
9. ✅ Prometheus scrapes metrics successfully
10. ✅ Basic documentation complete

### 5.2 150% Quality Targets

11. ✅ **Performance**: All latency targets met (p95 <2ms)
12. ✅ **Reliability**: Zero data loss on pod restart
13. ✅ **Observability**: 10 alerts + Grafana dashboard
14. ✅ **Security**: NetworkPolicy + Secret management
15. ✅ **Documentation**: 2,000+ LOC comprehensive guides
16. ✅ **Testing**: Load tests + failover tests + persistence tests
17. ✅ **HA-Ready**: Expandable to 3 replicas (Sentinel mode)
18. ✅ **Integration**: Seamless helm upgrade from current state
19. ✅ **Zero Breaking Changes**: Backward compatible
20. ✅ **Operational Excellence**: Complete runbooks + troubleshooting

### 5.3 Acceptance Tests

**Deployment Tests**:
```bash
# 1. Helm template renders correctly
helm template alerthistory ./helm/alert-history --set profile=standard | grep -A 20 "kind: StatefulSet"

# 2. StatefulSet deploys successfully
helm install alerthistory ./helm/alert-history --set profile=standard
kubectl wait --for=condition=ready pod/alerthistory-redis-0 --timeout=60s

# 3. Volume bound successfully
kubectl get pvc alerthistory-redis-data-alerthistory-redis-0 -o jsonpath='{.status.phase}' | grep Bound

# 4. Redis responds to commands
kubectl exec alerthistory-redis-0 -- redis-cli PING
# Expected: PONG
```

**Connection Tests**:
```bash
# 5. App connects successfully
kubectl logs deployment/alerthistory | grep "Redis cache initialized successfully"

# 6. Connection pool working
kubectl exec alerthistory-redis-0 -- redis-cli INFO clients | grep connected_clients

# 7. Cache operations working
kubectl exec alerthistory-redis-0 -- redis-cli SET test-key test-value
kubectl exec alerthistory-redis-0 -- redis-cli GET test-key
# Expected: test-value
```

**Persistence Tests**:
```bash
# 8. AOF file created
kubectl exec alerthistory-redis-0 -- ls -lh /data/appendonly.aof

# 9. RDB snapshot created
kubectl exec alerthistory-redis-0 -- ls -lh /data/dump.rdb

# 10. Restart preserves data
kubectl delete pod alerthistory-redis-0
kubectl wait --for=condition=ready pod/alerthistory-redis-0 --timeout=60s
kubectl exec alerthistory-redis-0 -- redis-cli GET test-key
# Expected: test-value (data preserved)
```

**Monitoring Tests**:
```bash
# 11. redis-exporter running
kubectl exec alerthistory-redis-0 -- wget -qO- localhost:9121/metrics | head -5

# 12. Prometheus scraping
kubectl get servicemonitor alerthistory-redis-metrics

# 13. Metrics visible in Prometheus
# Query: redis_memory_used_bytes{job="redis"}

# 14. Grafana dashboard imported
# Dashboard ID: 11835 (Redis Dashboard)
```

**Security Tests**:
```bash
# 15. NetworkPolicy blocking unauthorized access
kubectl run -it --rm unauthorized-pod --image=redis:7-alpine -- redis-cli -h alerthistory-redis PING
# Expected: Timeout (connection blocked)

# 16. Password authentication working
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD PING
# Expected: PONG

# 17. Dangerous commands disabled
kubectl exec alerthistory-redis-0 -- redis-cli CONFIG GET maxmemory
# Expected: Error (command renamed)
```

---

## 6. Dependencies & Risks

### 6.1 Dependencies

**Completed Dependencies** ✅:
- TN-200: Deployment Profile Configuration (162%, A+)
- TN-201: Storage Backend Selection (152%, A+)
- TN-202: Redis Conditional Init (100%, A)
- TN-203: Main.go Profile Init (100%, A)
- TN-96: Production Helm Chart (100%, A)
- TN-97: HPA Configuration (150%, A+)
- TN-98: PostgreSQL StatefulSet (150%, A+) - Pattern established

**No Blockers** 🎉

### 6.2 Risk Assessment

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| **Connection pool exhaustion** | HIGH | MEDIUM | ✅ Sizing analysis (5% utilization) |
| **Memory overflow (OOM)** | HIGH | LOW | ✅ maxmemory=384MB + LRU eviction |
| **Data loss on crash** | MEDIUM | LOW | ✅ AOF everysec (max 1s loss) |
| **Slow AOF replay** | LOW | MEDIUM | ✅ Expected <30s for 200MB |
| **Storage exhaustion** | MEDIUM | LOW | ✅ Monitoring + alerts |
| **Network latency** | LOW | LOW | ✅ L1 cache absorbs 95% hits |
| **Breaking changes** | HIGH | LOW | ✅ Backward compatible design |
| **Complex operations** | MEDIUM | HIGH | ✅ Comprehensive documentation |

---

## 7. Out of Scope

**Explicitly Excluded**:
1. ❌ **Redis Sentinel HA** - Future enhancement (3 replicas)
2. ❌ **Redis Cluster** - Not required (single dataset <500MB)
3. ❌ **TLS/SSL encryption** - Future enhancement
4. ❌ **Remote backup automation** - Future enhancement (TN-100)
5. ❌ **Lite Profile Redis** - Intentionally excluded (memory-only)
6. ❌ **Cross-region replication** - Not required for OSS
7. ❌ **Redis modules** - Not required (RedisJSON, RedisBloom, etc.)

---

## 8. Deliverables Checklist

### Phase 1: Documentation ✅
- [x] requirements.md (600+ LOC) - **THIS DOCUMENT**
- [ ] design.md (800+ LOC)
- [ ] tasks.md (600+ LOC)

### Phase 2: Implementation
- [ ] redis-statefulset.yaml (400+ LOC)
- [ ] redis-config.yaml (300+ LOC)
- [ ] redis-service.yaml (150 LOC, 3 services)
- [ ] values.yaml integration (100 LOC)

### Phase 3: Monitoring
- [ ] redis-exporter sidecar (100 LOC)
- [ ] ServiceMonitor CRD (50 LOC)
- [ ] PrometheusRule (200 LOC, 10 alerts)
- [ ] Grafana dashboard JSON (500 LOC)

### Phase 4: Security
- [ ] NetworkPolicy (50 LOC)
- [ ] Secret manifest (50 LOC)
- [ ] RBAC minimal permissions (50 LOC)

### Phase 5: Testing
- [ ] Helm template tests (200 LOC)
- [ ] K6 connection pool tests (300 LOC)
- [ ] Failover tests (200 LOC)
- [ ] Persistence tests (150 LOC)

### Phase 6: Documentation
- [ ] REDIS_OPERATIONS_GUIDE.md (800+ LOC)
- [ ] TROUBLESHOOTING.md (500+ LOC)
- [ ] DISASTER_RECOVERY.md (400+ LOC)

### Phase 7: Integration
- [ ] Main tasks.md updates
- [ ] CHANGELOG.md entry
- [ ] COMPLETION_REPORT.md (600+ LOC)

**Total Estimated LOC**: 7,850+ (112% of 7,000 target for 150% quality) ✅

---

## 9. References

### Internal Documentation
- [TN-98: PostgreSQL StatefulSet](../TN-098-postgresql-statefulset/COMPLETION_REPORT.md) - Pattern reference
- [TN-97: HPA Configuration](../TN-097-hpa-configuration/COMPLETION_REPORT.md) - Scaling context
- [TN-202: Redis Conditional Init](../TN-202-redis-conditional/README.md) - App integration
- [TN-200: Profile Configuration](../TN-200-deployment-profiles/README.md) - Profile system

### External References
- [Redis Documentation](https://redis.io/docs/)
- [Valkey Documentation](https://valkey.io/docs/)
- [redis-exporter Metrics](https://github.com/oliver006/redis_exporter)
- [Kubernetes StatefulSet](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
- [Redis Persistence](https://redis.io/docs/management/persistence/)
- [Redis Security](https://redis.io/docs/management/security/)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-30
**Author**: Vitalii Semenov (AI-assisted)
**Status**: ✅ APPROVED FOR IMPLEMENTATION
**Next**: design.md (Architecture & Technical Design)
