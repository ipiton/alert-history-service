# TN-97: Database Connections Analysis - Cluster Scaling

**Date**: 2025-11-29
**Status**: 🔴 **CRITICAL GAP IDENTIFIED**
**Priority**: P1 (Must Address Before Production)

---

## 🚨 Problem Statement

В текущей реализации HPA (2-10 replicas) **НЕ УЧТЕНЫ** модели записи/чтения из PostgreSQL в кластерном варианте, что создает **критический риск connection pool exhaustion**.

---

## 📊 Current State Analysis

### Connection Pool Configuration (Per Pod)

**File**: `go-app/internal/database/postgres/config.go`

```go
func DefaultConfig() *PostgresConfig {
    return &PostgresConfig{
        MaxConns: 20,  // ⚠️ 20 connections per pod
        MinConns: 2,   // 2 minimum connections per pod
        // ...
    }
}
```

**Environment Variables**:
- `DB_MAX_CONNS` (default: 20)
- `DB_MIN_CONNS` (default: 2)

### HPA Configuration (2-10 replicas)

**File**: `helm/alert-history/templates/hpa.yaml`

```yaml
spec:
  minReplicas: 2      # Minimum 2 pods
  maxReplicas: 10     # Maximum 10 pods
```

### PostgreSQL StatefulSet

**File**: `helm/alert-history/templates/postgresql-statefulset.yaml`

```yaml
spec:
  replicas: 1         # ⚠️ Single PostgreSQL instance
```

**PostgreSQL Default Configuration**:
- `max_connections`: **100** (default for Postgres 16)

---

## 🔴 Critical Gap: Connection Math

### Scenario 1: Minimum Replicas (2 pods)

```
Total Connections = pods × MaxConns
                  = 2 × 20
                  = 40 connections
PostgreSQL Limit  = 100 connections
Utilization       = 40%  ✅ OK
```

### Scenario 2: Average Load (5 pods)

```
Total Connections = 5 × 20
                  = 100 connections
PostgreSQL Limit  = 100 connections
Utilization       = 100%  ⚠️ AT LIMIT
```

### Scenario 3: Maximum Replicas (10 pods)

```
Total Connections = 10 × 20
                  = 200 connections
PostgreSQL Limit  = 100 connections
Utilization       = 200%  🔴 EXHAUSTED!
```

**🔴 CRITICAL ISSUE**: At 10 replicas, we need **200 connections** but PostgreSQL only allows **100 connections**.

---

## 🚨 Impact Assessment

### Risk Level: 🔴 **CRITICAL**

| Severity | Probability | Impact |
|----------|-------------|--------|
| **CRITICAL** | **HIGH** (при scale to 10) | **HIGH** (service degradation) |

### Failure Modes

1. **Connection Refused**
   - Symptom: `pq: sorry, too many clients already`
   - When: At 6+ replicas (120+ connections)
   - Impact: New pods cannot connect to database

2. **Degraded Performance**
   - Symptom: Connection wait time increases
   - When: At 4-5 replicas (80-100 connections)
   - Impact: Increased latency, timeouts

3. **Service Unavailability**
   - Symptom: All requests fail
   - When: At 10 replicas (200 connections attempted)
   - Impact: Complete service outage

### Production Scenarios

**Scenario A: Load Spike**
```
1. Traffic increases → HPA scales to 10 replicas
2. Each pod tries to open 20 connections
3. PostgreSQL refuses connections at 101+ (100 max)
4. Pods fail health checks → restart loop
5. Service degradation/outage
```

**Scenario B: Rolling Update**
```
1. Deploy new version (10 new pods + 10 old pods = 20 pods)
2. Total connections needed: 20 × 20 = 400
3. PostgreSQL accepts only 100
4. 75% of pods fail to start
5. Deployment fails
```

**Scenario C: Database Maintenance**
```
1. DBA reduces max_connections for maintenance
2. Existing pods hold 80 connections
3. New pod cannot acquire 20 connections
4. Scaling fails
```

---

## 🔍 Current Implementation Gaps

### 1. Static Connection Pool ❌

**Current**:
```go
MaxConns: 20  // Fixed per pod, ignores cluster size
```

**Problem**: No awareness of:
- Total number of replicas
- Database connection limit
- Other services using same database

### 2. No PostgreSQL Tuning ❌

**Current**:
```yaml
replicas: 1  # Single instance, default config
```

**Problem**:
- `max_connections`: 100 (default, too low)
- No connection pooler (PgBouncer/Pgpool)
- No read replicas for read-heavy workloads

### 3. No Dynamic Scaling ❌

**Current**: Connection pool size is static

**Problem**:
- Cannot adapt to changing cluster size
- Cannot respond to database capacity changes

### 4. No Monitoring/Alerting ❌

**Current**: HPA monitors CPU/Memory, not database connections

**Problem**:
- No alert before connection exhaustion
- No visibility into connection pool utilization

---

## ✅ Recommended Solutions

### Solution 1: Dynamic Connection Pool Sizing 🎯 **RECOMMENDED**

**Approach**: Calculate `MaxConns` dynamically based on replica count

```go
// Formula: MaxConns = (PostgreSQL_max_connections - reserved) / expected_replicas
//
// Example:
// PostgreSQL max_connections = 200
// Reserved (monitoring, admin) = 20
// Expected replicas = 10
// MaxConns per pod = (200 - 20) / 10 = 18
```

**Implementation**:
```yaml
# helm/alert-history/values.yaml
autoscaling:
  maxReplicas: 10

database:
  maxConnections: 200                    # PostgreSQL max_connections
  reservedConnections: 20                # Reserved for admin/monitoring
  maxConnsPerPod:                        # Calculated dynamically
    formula: "(maxConnections - reservedConnections) / maxReplicas"
    # Result: (200 - 20) / 10 = 18 per pod
```

**Pros**:
- ✅ Prevents connection exhaustion
- ✅ Works with any replica count
- ✅ Simple to configure

**Cons**:
- ⚠️ Requires restart to apply new connection count
- ⚠️ May underutilize connections at low replica counts

### Solution 2: Increase PostgreSQL max_connections 🎯 **QUICK FIX**

**Approach**: Increase PostgreSQL `max_connections` to support 10 replicas

```sql
-- PostgreSQL configuration
max_connections = 250  -- Increased from 100

-- Calculation:
-- 10 replicas × 20 conns = 200 connections
-- + 50 reserved = 250 total
```

**Implementation**:
```yaml
# helm/alert-history/templates/postgresql-configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "alert-history.postgresql.fullname" . }}-config
data:
  postgresql.conf: |
    max_connections = 250
    shared_buffers = 256MB         # 25% of max_connections
    effective_cache_size = 1GB
    work_mem = 4MB
```

**Pros**:
- ✅ Quick to implement
- ✅ No code changes required
- ✅ Works immediately

**Cons**:
- ⚠️ Requires more PostgreSQL memory (shared_buffers)
- ⚠️ May hit OS limits (file descriptors)
- ⚠️ Doesn't prevent oversubscription

### Solution 3: Connection Pooler (PgBouncer) 🎯 **BEST PRACTICE**

**Approach**: Deploy PgBouncer as a middleware for connection multiplexing

```
┌─────────────────────────────────────────────┐
│  Alert History Pods (2-10 replicas)        │
│  Each pod: 20 connections to PgBouncer     │
└──────────────────┬──────────────────────────┘
                   │ 200 connections (max)
                   ↓
┌─────────────────────────────────────────────┐
│  PgBouncer (Connection Pooler)              │
│  Pool Mode: Transaction                     │
│  Max Client Conns: 200                      │
│  Max DB Conns: 50                           │
└──────────────────┬──────────────────────────┘
                   │ 50 connections
                   ↓
┌─────────────────────────────────────────────┐
│  PostgreSQL (Single Instance)               │
│  max_connections: 100                       │
└─────────────────────────────────────────────┘
```

**Implementation**:
```yaml
# helm/alert-history/templates/pgbouncer-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pgbouncer
spec:
  replicas: 2  # HA for pgbouncer
  template:
    spec:
      containers:
      - name: pgbouncer
        image: pgbouncer/pgbouncer:1.21
        env:
        - name: DATABASES_HOST
          value: postgresql
        - name: PGBOUNCER_POOL_MODE
          value: transaction
        - name: PGBOUNCER_MAX_CLIENT_CONN
          value: "200"
        - name: PGBOUNCER_DEFAULT_POOL_SIZE
          value: "50"
```

**Pros**:
- ✅ **Best practice** for production
- ✅ Connection multiplexing (200 → 50)
- ✅ Works with existing code (transparent)
- ✅ Transaction pooling (optimal for REST API)
- ✅ Query routing (read/write split ready)

**Cons**:
- ⚠️ Additional component to manage
- ⚠️ Slight latency overhead (~1ms)
- ⚠️ Requires PgBouncer expertise

### Solution 4: Read Replicas (Future Enhancement) 🔮

**Approach**: Deploy PostgreSQL with streaming replication

```
┌────────────────────────────────────────────┐
│  Alert History Pods (2-10 replicas)       │
└──────┬─────────────────────────┬───────────┘
       │ Write (INSERT/UPDATE)   │ Read (SELECT)
       ↓                         ↓
┌──────────────┐         ┌──────────────┐
│  PostgreSQL  │────────>│  PostgreSQL  │
│  Primary     │ Replica │  Replica 1   │
│  (Write)     │ Stream  │  (Read)      │
└──────────────┘         └──────────────┘
                                ↓
                         ┌──────────────┐
                         │  PostgreSQL  │
                         │  Replica 2   │
                         │  (Read)      │
                         └──────────────┘
```

**Pros**:
- ✅ Horizontal read scaling
- ✅ Reduced load on primary
- ✅ Better fault tolerance

**Cons**:
- ⚠️ Complex setup (replication lag, failover)
- ⚠️ Requires code changes (read/write routing)
- ⚠️ Increased infrastructure cost

---

## 🎯 Recommended Implementation Plan

### Phase 1: Immediate (TN-97 blocker) 🔴 **CRITICAL**

**Goal**: Prevent connection exhaustion

1. ✅ **Document the issue** (this file)
2. ⏳ **Implement Solution 2** (Increase PostgreSQL max_connections to 250)
3. ⏳ **Add validation** (startup check: `total_replicas × MaxConns < max_connections`)
4. ⏳ **Update documentation** (TN-97 README, COMPLETION_REPORT)

**Time**: 1-2 hours
**Priority**: P1 (blocks TN-97 certification)

### Phase 2: Short-Term (TN-98 enhancement) ⚠️

**Goal**: Production-ready database configuration

1. ⏳ **Implement Solution 1** (Dynamic connection pool sizing)
2. ⏳ **PostgreSQL ConfigMap** (custom postgresql.conf)
3. ⏳ **Resource tuning** (shared_buffers, work_mem based on connections)
4. ⏳ **Monitoring** (connection pool utilization metrics)
5. ⏳ **Alerting** (alert at 80% connection utilization)

**Time**: 4-6 hours
**Priority**: P2 (TN-98 scope)

### Phase 3: Long-Term (Future) 🔮

**Goal**: Enterprise-grade database scaling

1. ⏳ **Implement Solution 3** (PgBouncer connection pooler)
2. ⏳ **Implement Solution 4** (Read replicas, if needed)
3. ⏳ **Query optimization** (read/write splitting)
4. ⏳ **Database monitoring** (query performance, slow queries)

**Time**: 2-3 days
**Priority**: P3 (future enhancement)

---

## 📏 Sizing Guidelines

### Connection Pool Formula

```
MaxConns_per_pod = (PostgreSQL_max_connections - reserved) / max_replicas

Where:
- PostgreSQL_max_connections: Configured limit (default 100, recommended 200-500)
- reserved: Connections for monitoring, admin, other services (default 20)
- max_replicas: HPA maxReplicas (default 10)
```

### Example Configurations

#### Development (2 replicas)
```yaml
postgresql:
  maxConnections: 100

autoscaling:
  maxReplicas: 2

database:
  maxConnsPerPod: 40  # (100 - 20) / 2 = 40
```

#### Production (10 replicas)
```yaml
postgresql:
  maxConnections: 250

autoscaling:
  maxReplicas: 10

database:
  maxConnsPerPod: 23  # (250 - 20) / 10 = 23
```

#### Enterprise (20 replicas + PgBouncer)
```yaml
postgresql:
  maxConnections: 100

pgbouncer:
  enabled: true
  maxClientConnections: 500
  defaultPoolSize: 80

autoscaling:
  maxReplicas: 20

database:
  maxConnsPerPod: 20  # Connects to PgBouncer, not PostgreSQL directly
```

---

## 🔍 Monitoring & Alerting

### Metrics to Track

1. **Connection Pool Utilization** (per pod)
```promql
# Current connections / Max connections
alert_history_db_connections{state="active"} / alert_history_db_connections{state="max"}
```

2. **Total Cluster Connections**
```promql
# Sum across all pods
sum(alert_history_db_connections{state="active"})
```

3. **PostgreSQL Connection Limit**
```promql
# PostgreSQL max_connections setting
pg_settings_max_connections
```

4. **Connection Wait Time**
```promql
# Time waiting to acquire connection
rate(alert_history_db_connection_wait_seconds_sum[5m])
```

### Recommended Alerts

1. **Connection Pool Near Limit** (Warning)
```yaml
alert: DatabaseConnectionPoolNearLimit
expr: |
  sum(alert_history_db_connections{state="active"}) /
  scalar(pg_settings_max_connections) > 0.8
for: 5m
severity: warning
description: "Database connection pool at {{ $value }}% capacity"
```

2. **Connection Pool Exhausted** (Critical)
```yaml
alert: DatabaseConnectionPoolExhausted
expr: |
  sum(alert_history_db_connections{state="active"}) /
  scalar(pg_settings_max_connections) > 0.95
for: 2m
severity: critical
description: "Database connection pool exhausted ({{ $value }}%)"
```

3. **Connection Acquisition Errors** (Critical)
```yaml
alert: DatabaseConnectionErrors
expr: |
  rate(alert_history_db_connection_errors_total[5m]) > 1
for: 2m
severity: critical
description: "Database connection errors: {{ $value }}/s"
```

---

## 🧪 Testing Plan

### Test 1: Connection Pool Under Load

**Scenario**: Scale to max replicas (10) and measure connections

```bash
# Scale to 10 replicas
kubectl scale deployment alert-history --replicas=10

# Monitor PostgreSQL connections
kubectl exec -it postgresql-0 -- psql -U alerthistory -c \
  "SELECT count(*) as active_connections FROM pg_stat_activity WHERE datname='alerthistory';"

# Expected: 200 connections (10 pods × 20 conns)
# Actual with current config: ~100 (connection limit reached)
```

### Test 2: Connection Exhaustion

**Scenario**: Verify behavior when connections exhausted

```bash
# Set PostgreSQL max_connections to 50
kubectl exec -it postgresql-0 -- psql -U postgres -c \
  "ALTER SYSTEM SET max_connections = 50;"

# Reload configuration
kubectl exec -it postgresql-0 -- psql -U postgres -c \
  "SELECT pg_reload_conf();"

# Scale to 5 replicas (5 × 20 = 100 connections needed, but only 50 available)
kubectl scale deployment alert-history --replicas=5

# Expected: 2-3 pods fail to start (connection refused)
```

### Test 3: Rolling Update

**Scenario**: Verify connections during deployment

```bash
# Before update: 10 old pods (200 connections)
# During update: 10 old + 10 new pods (400 connections needed)

# Start rolling update
kubectl set image deployment/alert-history alert-history=new-image

# Monitor connection errors
kubectl logs -f deployment/alert-history | grep "connection refused"

# Expected with current config: ~50% of pods fail during rollout
```

---

## 📚 References

### PostgreSQL Connection Tuning
- [PostgreSQL Connection Pooling](https://www.postgresql.org/docs/16/runtime-config-connection.html)
- [Connection Pool Sizing](https://wiki.postgresql.org/wiki/Number_Of_Database_Connections)

### PgBouncer
- [PgBouncer Official Docs](https://www.pgbouncer.org/)
- [PgBouncer on Kubernetes](https://github.com/pgbouncer/pgbouncer-k8s)

### Best Practices
- [Kubernetes Database Connections](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/)
- [pgx Connection Pool](https://github.com/jackc/pgx/wiki/Connection-Pool)

---

## 🎯 Action Items

### Immediate (Today)

- [x] Document connection pool issue
- [ ] Update TN-97 COMPLETION_REPORT with critical gap
- [ ] Create TN-98 enhancement scope (PostgreSQL tuning)
- [ ] Add validation in deployment.yaml (connection math check)

### Short-Term (Next Sprint)

- [ ] Implement PostgreSQL max_connections increase (TN-98)
- [ ] Implement dynamic connection pool sizing (TN-98)
- [ ] Add connection pool monitoring (TN-98)
- [ ] Add connection pool alerts (TN-98)

### Long-Term (Q1 2025)

- [ ] Evaluate PgBouncer deployment
- [ ] Evaluate read replicas (if read-heavy workload)
- [ ] Load testing with 10+ replicas
- [ ] Database performance optimization

---

**Status**: 🔴 **CRITICAL GAP IDENTIFIED**
**Next Step**: Update TN-97 certification with connection pool considerations
**Owner**: Vitalii Semenov
**Date**: 2025-11-29
