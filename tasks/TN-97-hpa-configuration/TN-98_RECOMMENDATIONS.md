# TN-98 Recommendations: PostgreSQL Configuration for HPA Cluster

**Date**: 2025-11-29
**Priority**: 🔴 P1 CRITICAL
**Blocker For**: TN-97 Production Deployment

---

## 🚨 Critical Issue

HPA scales to 2-10 replicas, но PostgreSQL не настроен для кластерного варианта:

| Replicas | Connections Needed | PostgreSQL Limit | Status |
|----------|-------------------|------------------|---------|
| 2 | 40 | 100 | ✅ OK (40%) |
| 5 | 100 | 100 | ⚠️ AT LIMIT (100%) |
| **10** | **200** | **100** | 🔴 **EXHAUSTED (200%)** |

**Connection Formula**: `Total = replicas × MaxConns_per_pod = 10 × 20 = 200`

---

## ✅ TN-98 Must Include

### 1. PostgreSQL Configuration Tuning 🔴 CRITICAL

```yaml
# helm/alert-history/templates/postgresql-configmap.yaml (NEW)
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgresql-config
data:
  postgresql.conf: |
    # Connection Settings
    max_connections = 250              # ⬆️ from 100 (default)

    # Memory Settings (tuned for max_connections = 250)
    shared_buffers = 256MB             # 25% of max_connections
    effective_cache_size = 1GB
    work_mem = 4MB
    maintenance_work_mem = 64MB

    # Performance
    random_page_cost = 1.1            # SSD-optimized
    effective_io_concurrency = 200    # SSD-optimized
```

**Reasoning**:
- 10 replicas × 20 conns/pod = 200 connections
- + 50 reserved (monitoring, admin, other services)
- = 250 `max_connections`

### 2. Dynamic Connection Pool Sizing 🎯 RECOMMENDED

```yaml
# helm/alert-history/values.yaml (MODIFY)
database:
  postgresql:
    maxConnections: 250              # PostgreSQL max_connections
    reservedConnections: 50          # Reserved for system

autoscaling:
  maxReplicas: 10

  # Auto-calculate MaxConns per pod
  connectionPool:
    auto: true
    maxConnsPerPod:                  # Calculated: (250 - 50) / 10 = 20
      formula: "(database.postgresql.maxConnections - database.postgresql.reservedConnections) / autoscaling.maxReplicas"
```

```yaml
# helm/alert-history/templates/deployment.yaml (MODIFY)
env:
  - name: DB_MAX_CONNS
    value: "{{ div (sub .Values.database.postgresql.maxConnections .Values.database.postgresql.reservedConnections) .Values.autoscaling.maxReplicas }}"
    # Result: (250 - 50) / 10 = 20
```

### 3. Connection Pool Monitoring 📊 REQUIRED

```yaml
# helm/alert-history/templates/servicemonitor.yaml (NEW/MODIFY)
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: alert-history-db
spec:
  selector:
    matchLabels:
      app: alert-history
  endpoints:
  - port: metrics
    interval: 30s
    path: /metrics
```

**Metrics to expose** (already in code):
- `alert_history_db_connections{state="active"}` - Current active connections
- `alert_history_db_connections{state="idle"}` - Idle connections
- `alert_history_db_connections{state="max"}` - Max connections configured
- `alert_history_db_connection_wait_seconds` - Time waiting for connection

### 4. Connection Pool Alerts 🚨 REQUIRED

```yaml
# helm/alert-history/templates/prometheusrule.yaml (NEW/MODIFY)
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: alert-history-database
spec:
  groups:
  - name: database_connection_pool
    interval: 30s
    rules:

    # Alert 1: Connection Pool Near Limit (Warning)
    - alert: DatabaseConnectionPoolNearLimit
      expr: |
        (sum(alert_history_db_connections{state="active"}) /
         scalar(pg_settings_max_connections)) > 0.8
      for: 5m
      labels:
        severity: warning
        component: database
      annotations:
        summary: "Database connection pool near limit"
        description: "Connection pool at {{ $value | humanizePercentage }} capacity (threshold: 80%)"
        runbook_url: "https://docs.alert-history.io/runbooks/database-connection-pool"

    # Alert 2: Connection Pool Exhausted (Critical)
    - alert: DatabaseConnectionPoolExhausted
      expr: |
        (sum(alert_history_db_connections{state="active"}) /
         scalar(pg_settings_max_connections)) > 0.95
      for: 2m
      labels:
        severity: critical
        component: database
      annotations:
        summary: "Database connection pool EXHAUSTED"
        description: "Connection pool at {{ $value | humanizePercentage }} capacity (threshold: 95%)"
        runbook_url: "https://docs.alert-history.io/runbooks/database-connection-pool"

    # Alert 3: Connection Acquisition Errors (Critical)
    - alert: DatabaseConnectionErrors
      expr: |
        rate(alert_history_db_connection_errors_total[5m]) > 1
      for: 2m
      labels:
        severity: critical
        component: database
      annotations:
        summary: "Database connection errors detected"
        description: "Connection errors: {{ $value | humanize }}/s"
        runbook_url: "https://docs.alert-history.io/runbooks/database-connection-errors"
```

### 5. Startup Validation 🛡️ REQUIRED

```go
// go-app/cmd/server/main.go (ADD)

func validateDatabaseConnectionPool(cfg *config.Config, replicas int) error {
    maxConns := cfg.Database.MaxConns
    totalConns := replicas * int(maxConns)
    pgMaxConns := 250 // From PostgreSQL config

    if totalConns > int(float64(pgMaxConns) * 0.8) {
        return fmt.Errorf(
            "connection pool oversubscription detected: "+
            "%d replicas × %d conns/pod = %d total connections, "+
            "but PostgreSQL max_connections = %d (80%% threshold = %d). "+
            "Reduce DB_MAX_CONNS or increase PostgreSQL max_connections",
            replicas, maxConns, totalConns, pgMaxConns, int(float64(pgMaxConns)*0.8),
        )
    }

    return nil
}
```

---

## 🎯 Optional Enhancements

### Option A: PgBouncer (Connection Pooler) 🏆 BEST PRACTICE

**Benefits**:
- Connection multiplexing (200 client → 50 database connections)
- Transparent to application (no code changes)
- Transaction pooling (optimal for REST API)

```yaml
# helm/alert-history/charts/pgbouncer/values.yaml (NEW CHART)
image:
  repository: pgbouncer/pgbouncer
  tag: 1.21

replicaCount: 2  # HA for pgbouncer

config:
  databases:
    alerthistory:
      host: postgresql
      port: 5432
      dbname: alerthistory

  pgbouncer:
    pool_mode: transaction         # Optimal for stateless REST API
    max_client_conn: 200           # Matches 10 pods × 20 conns
    default_pool_size: 50          # Actual connections to PostgreSQL
    min_pool_size: 10
    reserve_pool_size: 10
    reserve_pool_timeout: 5
    server_idle_timeout: 600

resources:
  requests:
    cpu: 100m
    memory: 128Mi
  limits:
    cpu: 500m
    memory: 512Mi
```

**Application Change**:
```yaml
# helm/alert-history/values.yaml (MODIFY)
database:
  host: pgbouncer              # ⬆️ from postgresql
  port: 6432                   # ⬆️ from 5432
```

### Option B: Read Replicas (Read-Heavy Workloads) 🔮

**Use Case**: If >70% of queries are SELECT (read-heavy)

```yaml
# helm/alert-history/templates/postgresql-statefulset.yaml (MODIFY)
spec:
  replicas: 3  # 1 primary + 2 replicas
```

**Requires**:
- Application-level read/write splitting
- Replication lag handling
- Failover logic

**NOT RECOMMENDED** for Phase 13 (too complex, TN-97 scope).

---

## 📋 TN-98 Task Scope (Updated)

### Original TN-98 Scope
- [ ] PostgreSQL StatefulSet (basic)
- [ ] PersistentVolume configuration
- [ ] Service & Headless Service
- [ ] Secrets management

### **Enhanced TN-98 Scope** (includes HPA support) 🔴

- [ ] PostgreSQL StatefulSet (basic)
- [ ] **PostgreSQL ConfigMap** (`max_connections = 250`)
- [ ] **Resource tuning** (shared_buffers, work_mem)
- [ ] PersistentVolume configuration
- [ ] Service & Headless Service
- [ ] Secrets management
- [ ] **Dynamic connection pool sizing** (Helm formula)
- [ ] **Connection pool monitoring** (ServiceMonitor)
- [ ] **Connection pool alerts** (PrometheusRule)
- [ ] **Startup validation** (connection math check)
- [ ] **Documentation** (connection pool sizing guide)

**Time Estimate**: +2-3 hours (was 4h, now 6-7h)

---

## 🧪 Testing Checklist

### Test 1: Connection Pool Under Max Load

```bash
# Deploy with HPA
helm install alert-history ./helm/alert-history --set profile=standard

# Scale to max replicas
kubectl scale deployment alert-history --replicas=10 -n production

# Wait 30s for connections to establish
sleep 30

# Check PostgreSQL connections
kubectl exec -it postgresql-0 -n production -- psql -U alerthistory -c \
  "SELECT count(*) as active_connections, max_connections FROM pg_stat_activity, pg_settings WHERE name='max_connections' GROUP BY max_connections;"

# Expected:
#  active_connections | max_connections
# --------------------+-----------------
#                 200 |             250
# (1 row)

# Utilization: 200/250 = 80% ✅
```

### Test 2: Connection Pool Alerts

```bash
# Simulate connection exhaustion (reduce max_connections)
kubectl exec -it postgresql-0 -n production -- psql -U postgres -c \
  "ALTER SYSTEM SET max_connections = 100; SELECT pg_reload_conf();"

# Scale to 10 replicas (needs 200, but only 100 available)
kubectl scale deployment alert-history --replicas=10 -n production

# Check Prometheus alerts (should fire within 2-5 min)
kubectl get prometheusalerts -n monitoring | grep Database

# Expected:
# DatabaseConnectionPoolExhausted   firing   critical   2m
```

### Test 3: Rolling Update Connection Spike

```bash
# Before: 10 pods (200 connections)
# During: 10 old + 10 new = 20 pods (400 connections)

# With max_connections=250, expect failures

# Start rolling update
kubectl set image deployment/alert-history alert-history=new-image:v2 -n production

# Monitor connection errors in logs
kubectl logs -f deployment/alert-history -n production | grep -i "connection refused\|too many clients"

# Expected with 250 max_connections:
# Some pods will retry/fail during rollout (expected behavior)

# Solution: Stagger rollout or increase max_connections to 500
```

---

## 📊 Recommended Values

### Development (2 replicas)
```yaml
postgresql:
  maxConnections: 100
database:
  maxConnsPerPod: 40   # (100 - 20) / 2
```

### Staging (5 replicas)
```yaml
postgresql:
  maxConnections: 150
database:
  maxConnsPerPod: 26   # (150 - 20) / 5
```

### Production (10 replicas)
```yaml
postgresql:
  maxConnections: 250
database:
  maxConnsPerPod: 23   # (250 - 50) / 10
```

### Production + PgBouncer (10 replicas)
```yaml
postgresql:
  maxConnections: 100  # Lower, PgBouncer handles multiplexing
pgbouncer:
  enabled: true
  maxClientConnections: 200
  defaultPoolSize: 80
database:
  host: pgbouncer      # App connects to PgBouncer
  maxConnsPerPod: 20   # Per-pod limit to PgBouncer
```

---

## 🚀 Rollout Strategy

### Phase 1: TN-98 (This Sprint)
1. ✅ PostgreSQL ConfigMap (`max_connections = 250`)
2. ✅ Dynamic connection pool sizing (Helm formula)
3. ✅ Connection pool monitoring & alerts
4. ✅ Startup validation
5. ✅ Documentation

**Time**: 6-7 hours
**Risk**: LOW (configuration-only changes)

### Phase 2: PgBouncer (Next Sprint)
1. Deploy PgBouncer as Helm subchart
2. Update application to connect via PgBouncer
3. Load testing (10+ replicas)
4. Production migration (blue-green)

**Time**: 2-3 days
**Risk**: MEDIUM (new component, requires testing)

### Phase 3: Read Replicas (Future)
1. PostgreSQL streaming replication setup
2. Application-level read/write splitting
3. Replication lag monitoring
4. Failover automation

**Time**: 1-2 weeks
**Risk**: HIGH (complex, requires DB expertise)

---

## 📚 References

- [PostgreSQL max_connections Tuning](https://www.postgresql.org/docs/16/runtime-config-connection.html)
- [pgx Connection Pool Sizing](https://github.com/jackc/pgx/wiki/Connection-Pool)
- [PgBouncer Documentation](https://www.pgbouncer.org/config.html)
- [Kubernetes Database Best Practices](https://kubernetes.io/docs/tasks/run-application/run-replicated-stateful-application/)

---

**Status**: 🔴 CRITICAL for TN-97 Production Deployment
**Owner**: TN-98 Assignee
**Reviewer**: Vitalii Semenov (TN-97)
**Date**: 2025-11-29
