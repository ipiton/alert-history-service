# TN-098: PostgreSQL StatefulSet - Standard Profile Only - Requirements

**Date**: 2025-11-30
**Target Quality**: 150% (Grade A+ EXCEPTIONAL)
**Duration Estimate**: 12-16 hours
**Phase**: 13 (Production Packaging)
**Dependencies**: TN-200 ✅, TN-201 ✅, TN-97 ✅ (HPA configuration)
**Profile**: Standard only (disabled for Lite profile)

---

## 📊 Executive Summary

Enhance and production-harden the **PostgreSQL StatefulSet** для Alertmanager++ Standard Profile, providing enterprise-grade distributed database with HA support, automated backups, monitoring, security hardening, and disaster recovery capabilities.

### Mission Statement
**Deliver a production-ready, battle-tested PostgreSQL StatefulSet** that supports Alert History service at scale (10K+ alerts/day), with 99.95% uptime, automatic failover, comprehensive monitoring, and zero-downtime updates.

### Key Objectives
1. ✅ **Production-Ready StatefulSet** (probes, lifecycle hooks, security context)
2. ✅ **Enterprise Configuration** (optimized for alert ingestion workloads)
3. ✅ **High Availability** (multi-replica support, automated failover)
4. ✅ **Automated Backups** (daily full + continuous WAL archiving)
5. ✅ **Comprehensive Monitoring** (Prometheus metrics, Grafana dashboards)
6. ✅ **Security Hardening** (TLS, RBAC, Pod Security Standards)
7. ✅ **Disaster Recovery** (backup/restore procedures, PITR)
8. ✅ **150% Quality Achievement** (exceed baseline requirements by 50%)

---

## 🎯 Business Requirements

### BR-1: Standard Profile Exclusive Deployment
**Priority**: P0 (Critical)
**Description**: PostgreSQL StatefulSet must ONLY deploy with Standard Profile. Lite profile uses embedded SQLite (TN-201).

**Acceptance Criteria**:
- ✅ StatefulSet rendered ONLY when `profile=standard` AND `postgresql.enabled=true`
- ✅ Zero PostgreSQL resources deployed in Lite profile
- ✅ Clear error message if misconfigured
- ✅ Values.yaml defaults aligned with Standard Profile requirements
- ✅ Helm template conditional logic validated via unit tests

**Business Value**: Reduces infrastructure costs by 70-90% for dev/test environments using Lite profile.

**Validation**:
```bash
# Standard Profile: Should render StatefulSet
helm template . --set profile=standard --show-only templates/postgresql-statefulset.yaml

# Lite Profile: Should NOT render StatefulSet
helm template . --set profile=lite --show-only templates/postgresql-statefulset.yaml
# Expected: empty output or "Source: (empty)"
```

---

### BR-2: HPA Cluster Mode Support
**Priority**: P0 (Critical)
**Description**: Support 2-10 application replicas with connection pooling (TN-97 dependency).

**Acceptance Criteria**:
- ✅ `max_connections=250` (supports 10 replicas × 20 conns/pod + 50 reserved)
- ✅ Connection pool validation in NOTES.txt
- ✅ Automatic connection exhaustion detection
- ✅ Prometheus alerts for connection pool saturation
- ✅ Documentation for scaling beyond 10 replicas

**Formula**:
```
max_connections = (max_replicas × conns_per_pod) + reserved + admin_buffer
                = (10 × 20) + 50 + 0 = 250
```

**Monitoring**:
- Alert when `pg_stat_database.numbackends` > 200 (80% utilization)
- Alert when connection wait time > 100ms
- Dashboard panel showing connection pool usage over time

---

### BR-3: High Availability Support
**Priority**: P1 (High)
**Description**: Support multi-replica PostgreSQL with streaming replication (future: TN-99 extension).

**Acceptance Criteria**:
- ✅ Single-node StatefulSet (baseline)
- ✅ Architecture supports future multi-replica via Patroni/CloudNativePG
- ✅ Pod Anti-Affinity (spread across nodes)
- ✅ PodDisruptionBudget (minAvailable=1)
- ✅ Graceful shutdown lifecycle hooks
- ✅ ReadWriteOnce PVC per pod

**Future Enhancement** (TN-99):
- Primary-replica setup with Patroni
- Automatic failover (<30s)
- Synchronous replication
- Read replicas for query offloading

---

### BR-4: Data Durability & Backup
**Priority**: P0 (Critical)
**Description**: Guarantee zero data loss through automated backups and WAL archiving.

**Acceptance Criteria**:
- ✅ Daily full backups to S3/GCS/Azure Blob
- ✅ Continuous WAL archiving (PITR capability)
- ✅ Backup retention policy (30 days default, configurable)
- ✅ Automated backup verification (restore test weekly)
- ✅ Disaster recovery runbook (RTO <4h, RPO <5min)
- ✅ Encrypted backups at rest

**Backup Strategy**:
```yaml
backup:
  enabled: true
  schedule: "0 2 * * *"  # Daily at 2 AM UTC
  retention: 30d
  storage:
    type: s3  # or gcs, azureblob
    bucket: alertmanager-backups
    path: postgresql/$(CLUSTER_NAME)
  verification:
    enabled: true
    schedule: "0 3 * * 0"  # Weekly at 3 AM Sunday
```

**PITR Example**:
```bash
# Restore to specific timestamp
pgbackrest restore \
  --stanza=alerthistory \
  --delta \
  --type=time \
  --target="2025-11-30 14:30:00"
```

---

### BR-5: Observability & Monitoring
**Priority**: P0 (Critical)
**Description**: Comprehensive monitoring with Prometheus metrics and Grafana dashboards.

**Acceptance Criteria**:
- ✅ PostgreSQL Exporter sidecar (prometheus-postgres-exporter)
- ✅ 50+ PostgreSQL metrics exposed
- ✅ Grafana dashboard with 20+ panels
- ✅ 15+ alerting rules (connections, replication lag, disk usage, slow queries)
- ✅ Query performance insights (pg_stat_statements)
- ✅ Logs aggregation (structured JSON logs)

**Key Metrics**:
```yaml
metrics:
  - pg_stat_database_numbackends  # Active connections
  - pg_stat_database_xact_commit  # Transaction rate
  - pg_stat_database_tup_inserted  # Insert rate (alerts/sec)
  - pg_stat_database_blks_hit_ratio  # Cache hit rate
  - pg_stat_replication_lag_seconds  # Replication lag (HA)
  - pg_database_size_bytes  # Database size growth
  - pg_stat_bgwriter_buffers_checkpoint  # Checkpoint frequency
  - pg_locks_count  # Lock contention
```

**Alerting Rules**:
1. High connection usage (>80%)
2. Replication lag >10s
3. Disk usage >85%
4. Cache hit rate <90%
5. Slow queries >5s
6. Too many idle connections
7. Checkpoint frequency >10/min
8. Transaction wraparound risk
9. Dead tuple ratio >20%
10. WAL segment growth >10GB/hour

---

### BR-6: Security Hardening
**Priority**: P0 (Critical)
**Description**: Enterprise-grade security aligned with CIS PostgreSQL Benchmark and Pod Security Standards.

**Acceptance Criteria**:
- ✅ TLS encryption (client-server communication)
- ✅ Pod Security Standards (restricted)
- ✅ Non-root containers (runAsUser: 999)
- ✅ Read-only root filesystem (where possible)
- ✅ Secret management via Kubernetes Secrets (future: External Secrets Operator)
- ✅ Network policies (restrict ingress/egress)
- ✅ RBAC least privilege
- ✅ Audit logging (pg_audit extension)

**Security Context**:
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 999
  runAsGroup: 999
  fsGroup: 999
  allowPrivilegeEscalation: false
  capabilities:
    drop: [ALL]
  readOnlyRootFilesystem: false  # PostgreSQL needs writable /tmp
  seccompProfile:
    type: RuntimeDefault
```

**TLS Configuration**:
```yaml
postgresql:
  tls:
    enabled: true
    certificatesSecret: postgresql-tls
    certFilename: tls.crt
    certKeyFilename: tls.key
    caFilename: ca.crt
```

---

### BR-7: Performance Optimization
**Priority**: P1 (High)
**Description**: Optimize PostgreSQL configuration for alert ingestion workloads (high write, moderate read).

**Acceptance Criteria**:
- ✅ Write-optimized configuration (alerting is write-heavy)
- ✅ Autovacuum tuning (prevent bloat from high INSERT/UPDATE rate)
- ✅ WAL tuning (balance durability vs performance)
- ✅ Connection pooling (pgBouncer optional)
- ✅ Query plan optimization (proper indexes, statistics)
- ✅ Benchmark results: >10K inserts/sec, p95 latency <10ms

**Workload Profile**:
- **Write-heavy**: 80% INSERTs, 15% UPDATEs, 5% SELECTs
- **Peak load**: 10K alerts/sec (600K alerts/min)
- **Data retention**: 90 days (configurable)
- **Database size**: ~500GB (10M alerts × 50KB each)

**Tuning Parameters**:
```yaml
config:
  # WAL Tuning (TN-98: write performance)
  wal_level: replica  # Minimal overhead
  synchronous_commit: off  # Accept slight durability risk for 3x performance
  wal_compression: on  # Reduce I/O
  wal_buffers: 16MB
  checkpoint_timeout: 15min
  checkpoint_completion_target: 0.9

  # Autovacuum Tuning (TN-98: prevent bloat)
  autovacuum_naptime: 1min  # More aggressive than default 1min
  autovacuum_vacuum_threshold: 50
  autovacuum_analyze_threshold: 50
  autovacuum_max_workers: 3
  autovacuum_vacuum_cost_delay: 20ms

  # Memory Tuning
  shared_buffers: 2GB  # 25% of 8GB total memory
  effective_cache_size: 6GB  # 75% of total memory
  work_mem: 16MB  # Per sort operation
  maintenance_work_mem: 512MB  # For VACUUM, CREATE INDEX
```

---

### BR-8: Disaster Recovery
**Priority**: P1 (High)
**Description**: Comprehensive DR capabilities with documented runbooks.

**Acceptance Criteria**:
- ✅ Point-in-Time Recovery (PITR)
- ✅ Backup verification (automated restore tests)
- ✅ DR runbook (step-by-step recovery procedures)
- ✅ RTO target: <4 hours (full restore)
- ✅ RPO target: <5 minutes (WAL archiving frequency)
- ✅ Cross-region backup replication (optional)

**DR Scenarios**:
1. **Data Corruption**: Restore from latest backup
2. **Accidental DELETE**: PITR to timestamp before deletion
3. **Disk Failure**: Restore to new PVC from backup
4. **Cluster Disaster**: Restore to new Kubernetes cluster
5. **Region Outage**: Restore from cross-region backup

**Runbook Template**:
```markdown
## Scenario: Accidental Data Deletion

### Detection
- Alert: Unexpected drop in alert count
- Investigation: Check pg_stat_database for large DELETE operations

### Recovery Steps
1. Identify deletion timestamp: `SELECT max(deleted_at) FROM alerts;`
2. Stop application pods: `kubectl scale deployment alert-history --replicas=0`
3. Initiate PITR restore: `./scripts/restore-pitr.sh --target "2025-11-30 14:30:00"`
4. Verify data integrity: Run data validation queries
5. Resume application: `kubectl scale deployment alert-history --replicas=3`

### Post-Incident
- Root cause analysis
- Update access controls
- Review backup retention policy
```

---

## 🔧 Functional Requirements

### FR-1: StatefulSet Configuration
**Description**: Production-ready StatefulSet with proper lifecycle management.

**Components**:
1. **StatefulSet Spec**:
   - `serviceName`: Headless service for stable network identity
   - `replicas`: 1 (baseline), scalable to 3 (future HA)
   - `podManagementPolicy`: OrderedReady (ensures ordered startup)
   - `updateStrategy`: RollingUpdate with partition support

2. **Pod Spec**:
   - Container: `postgres:16` (latest stable LTS)
   - Resource limits: 2Gi RAM, 1000m CPU (supports max_connections=250)
   - Security context: Non-root user (999), dropped capabilities
   - Affinity rules: Pod anti-affinity (spread across nodes)
   - Tolerations: Optional (for dedicated database nodes)

3. **Probes**:
   - **Startup Probe**: `pg_isready` (30 attempts, 10s interval)
   - **Liveness Probe**: `pg_isready` (failureThreshold=6)
   - **Readiness Probe**: `pg_isready` + `SELECT 1` query (deep check)

4. **Lifecycle Hooks**:
   - **PreStop**: Graceful shutdown via `pg_ctl stop -m fast` (60s timeout)
   - **PostStart**: Optional init scripts (create extensions, users)

5. **Volume Mounts**:
   - `/var/lib/postgresql/data`: PVC for PGDATA (10Gi default)
   - `/etc/postgresql`: ConfigMap for postgresql.conf
   - `/etc/postgresql/tls`: Secret for TLS certificates (optional)
   - `/backup`: Optional PVC for local backups

---

### FR-2: ConfigMap Management
**Description**: Externalized PostgreSQL configuration via ConfigMap.

**Structure**:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgresql-config
data:
  postgresql.conf: |
    # Full configuration (110+ lines)
    max_connections = 250
    shared_buffers = 2GB
    # ... (see postgresql-configmap.yaml)

  pg_hba.conf: |
    # Client authentication
    local   all             all                                     trust
    host    all             all             127.0.0.1/32            trust
    host    all             all             10.0.0.0/8              scram-sha-256
    hostssl all             all             0.0.0.0/0               scram-sha-256

  init.sql: |
    -- Post-initialization SQL
    CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
    CREATE EXTENSION IF NOT EXISTS pg_audit;
    ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements,pg_audit';
```

**Features**:
- ✅ Checksum annotation (trigger pod restart on config change)
- ✅ Validation via `postgres --check` (dry-run)
- ✅ Hot reload support (SIGHUP for reloadable params)
- ✅ Configuration templates per environment (dev/staging/prod)

---

### FR-3: Secret Management
**Description**: Secure credential handling via Kubernetes Secrets.

**Secrets Required**:
1. **PostgreSQL Password** (`postgresql-secret`):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: postgresql-secret
   type: Opaque
   data:
     password: <base64-encoded>
   ```

2. **TLS Certificates** (`postgresql-tls`):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: postgresql-tls
   type: kubernetes.io/tls
   data:
     tls.crt: <base64-encoded>
     tls.key: <base64-encoded>
     ca.crt: <base64-encoded>
   ```

3. **Backup Credentials** (`backup-credentials`):
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: backup-credentials
   type: Opaque
   data:
     AWS_ACCESS_KEY_ID: <base64>
     AWS_SECRET_ACCESS_KEY: <base64>
     # or GOOGLE_APPLICATION_CREDENTIALS for GCS
   ```

**Future Enhancement** (TN-100):
- External Secrets Operator (ESO) integration
- AWS Secrets Manager / GCP Secret Manager / Azure Key Vault
- Automatic secret rotation
- Audit trail for secret access

---

### FR-4: Service Definitions
**Description**: Services for database connectivity.

**Services**:
1. **Headless Service** (for StatefulSet):
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: postgresql
   spec:
     clusterIP: None  # Headless
     selector:
       app.kubernetes.io/component: database
     ports:
     - port: 5432
       targetPort: postgres
   ```

2. **ClusterIP Service** (for clients):
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: postgresql-rw  # Read-write endpoint
   spec:
     type: ClusterIP
     selector:
       app.kubernetes.io/component: database
     ports:
     - port: 5432
       targetPort: postgres
   ```

3. **Exporter Service** (for Prometheus scraping):
   ```yaml
   apiVersion: v1
   kind: Service
   metadata:
     name: postgresql-exporter
     labels:
       app.kubernetes.io/component: database-exporter
   spec:
     ports:
     - port: 9187
       targetPort: metrics
       name: metrics
     selector:
       app.kubernetes.io/component: database
   ```

---

### FR-5: PodDisruptionBudget
**Description**: Ensure availability during voluntary disruptions.

**Configuration**:
```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: postgresql-pdb
spec:
  minAvailable: 1  # Keep at least 1 pod running
  selector:
    matchLabels:
      app.kubernetes.io/component: database
```

**Behavior**:
- Prevents kubectl drain / node eviction if it would violate minAvailable
- Allows rolling updates to proceed safely
- Protects during cluster upgrades

---

### FR-6: Backup & Restore
**Description**: Automated backup/restore via pgBackRest or Velero.

**pgBackRest Integration**:
```yaml
backup:
  enabled: true
  tool: pgbackrest  # or pg_basebackup, velero

  # Storage configuration
  storage:
    type: s3
    bucket: alertmanager-backups
    region: us-east-1
    endpoint: https://s3.amazonaws.com
    path: postgresql/alerthistory

  # Backup schedules
  schedules:
    full: "0 2 * * 0"  # Weekly full backup (Sunday 2 AM)
    differential: "0 2 * * 1-6"  # Daily differential
    incremental: "*/15 * * * *"  # 15-min WAL archiving (PITR)

  # Retention policy
  retention:
    full: 4  # Keep 4 weekly full backups (1 month)
    differential: 7  # Keep 7 daily differentials
    incremental: 168  # Keep 7 days of 15-min increments

  # Verification
  verify:
    enabled: true
    schedule: "0 3 * * 0"  # Weekly restore test
    notification:
      slack: "#alerts-ops"
      email: ops@company.com
```

**Backup CronJob**:
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgresql-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: pgbackrest
            image: pgbackrest/pgbackrest:latest
            command: ["/scripts/backup.sh"]
            env:
            - name: PGBACKREST_STANZA
              value: alerthistory
            - name: PGBACKREST_TYPE
              value: full
            volumeMounts:
            - name: backup-config
              mountPath: /etc/pgbackrest
            - name: aws-credentials
              mountPath: /root/.aws
```

---

### FR-7: Monitoring & Alerting
**Description**: Prometheus metrics collection and Grafana dashboards.

**PostgreSQL Exporter Sidecar**:
```yaml
- name: postgres-exporter
  image: quay.io/prometheuscommunity/postgres-exporter:v0.15.0
  ports:
  - name: metrics
    containerPort: 9187
  env:
  - name: DATA_SOURCE_NAME
    value: "postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable"
  - name: PG_EXPORTER_AUTO_DISCOVER_DATABASES
    value: "true"
  - name: PG_EXPORTER_EXTEND_QUERY_PATH
    value: /etc/postgres-exporter/queries.yaml
  volumeMounts:
  - name: exporter-config
    mountPath: /etc/postgres-exporter
```

**Custom Queries** (`queries.yaml`):
```yaml
pg_alert_ingestion_rate:
  query: |
    SELECT COUNT(*) as alerts_per_minute
    FROM alerts
    WHERE created_at > NOW() - INTERVAL '1 minute'
  metrics:
    - alerts_per_minute:
        usage: GAUGE
        description: Alert ingestion rate (alerts/min)

pg_database_growth_rate:
  query: |
    SELECT
      pg_database_size(current_database()) as size_bytes,
      pg_database_size(current_database()) - lag(pg_database_size(current_database()))
        OVER (ORDER BY now()) as growth_bytes_per_hour
    FROM pg_stat_database
    WHERE datname = current_database()
  metrics:
    - size_bytes:
        usage: GAUGE
    - growth_bytes_per_hour:
        usage: GAUGE
```

**Grafana Dashboard Panels** (20+):
1. Connection pool usage (gauge)
2. Transaction rate (graph)
3. Alert ingestion rate (graph)
4. Cache hit ratio (gauge)
5. Disk usage (gauge + forecast)
6. Query latency p50/p95/p99 (graph)
7. Slow queries (table)
8. Replication lag (graph)
9. WAL generation rate (graph)
10. Checkpoint frequency (graph)
11. Table bloat (table)
12. Index usage (table)
13. Lock contention (graph)
14. Background writer stats (graph)
15. Autovacuum activity (graph)
16. Database size growth (graph + forecast)
17. Top 10 slowest queries (table)
18. Connection states (stacked graph)
19. Buffer cache hit ratio by table (table)
20. Transaction wraparound risk (gauge)

---

## 🚀 Non-Functional Requirements

### NFR-1: Performance
**Target Metrics**:
- Alert ingestion: >10,000 alerts/sec
- Query latency: p95 <10ms (simple queries), p95 <100ms (complex)
- Connection establishment: <5ms
- Checkpoint duration: <30s
- VACUUM duration: <5min (per table)

**Load Testing**:
```bash
# k6 load test
k6 run --vus 100 --duration 5m scripts/postgresql-load-test.js

# pgbench
pgbench -i -s 100 alerthistory  # Initialize with 100x scale
pgbench -c 50 -j 10 -T 300 alerthistory  # 50 clients, 10 threads, 5 min
```

---

### NFR-2: Scalability
**Targets**:
- Vertical: Support up to 16 cores, 64GB RAM
- Horizontal: Support multi-replica (TN-99 future)
- Data volume: 1TB+ database size
- Connections: 250 concurrent connections

---

### NFR-3: Availability
**Targets**:
- Uptime: 99.95% (4.4 hours/year downtime)
- Failover time: <30s (future HA mode)
- Zero-downtime updates: Yes (via rolling updates)

---

### NFR-4: Reliability
**Targets**:
- Data durability: 99.999% (WAL archiving + backups)
- Backup success rate: 99.9%
- Mean Time To Recovery (MTTR): <4 hours
- Recovery Point Objective (RPO): <5 minutes

---

### NFR-5: Security
**Requirements**:
- Encryption at rest: Yes (via PVC encryption)
- Encryption in transit: Yes (TLS 1.3)
- Authentication: scram-sha-256
- Authorization: Role-based (least privilege)
- Audit logging: All DDL/DML operations
- Compliance: CIS PostgreSQL Benchmark Level 1

---

## 📋 Acceptance Criteria

### Core Functionality (Baseline 100%)
- [ ] StatefulSet deploys successfully in Standard Profile
- [ ] StatefulSet DOES NOT deploy in Lite Profile
- [ ] PostgreSQL 16 boots and accepts connections
- [ ] PVC provisioned and PGDATA persisted
- [ ] ConfigMap applied correctly
- [ ] Secret mounted and password works
- [ ] Probes pass (startup, liveness, readiness)
- [ ] Services reachable (postgresql, postgresql-rw)
- [ ] PodDisruptionBudget enforced

### 150% Quality Targets
- [ ] **+10%**: Prometheus exporter sidecar functional
- [ ] **+10%**: Grafana dashboard with 20+ panels
- [ ] **+10%**: 15+ alerting rules configured
- [ ] **+10%**: Automated backups to S3/GCS
- [ ] **+10%**: PITR capability demonstrated
- [ ] **+10%**: Backup verification CronJob
- [ ] **+10%**: DR runbook comprehensive (10+ pages)
- [ ] **+10%**: Security hardening (TLS, Pod Security Standards)
- [ ] **+10%**: Performance benchmarks (>10K inserts/sec)
- [ ] **+10%**: Multi-environment configs (dev/staging/prod)
- [ ] **+10%**: Helm unit tests (10+ test cases)
- [ ] **+10%**: Documentation excellence (5,000+ LOC)
- [ ] **+10%**: Zero technical debt
- [ ] **+10%**: Production deployment guide
- [ ] **+10%**: Operational runbook (20+ scenarios)

**Total Possible**: 250% (100% baseline + 150% enhancements)
**Target**: 150% (deliver 10 of 15 enhancements)

---

## 🔍 Success Metrics

### Quantitative Metrics
| Metric | Baseline | Target (150%) | Measurement |
|--------|----------|---------------|-------------|
| Deployment success rate | 95% | 99.9% | CI/CD stats |
| MTTR | 8h | <4h | Incident logs |
| Backup success rate | 95% | 99.9% | Backup logs |
| Alert ingestion rate | 5K/sec | 10K/sec | pgbench |
| Query p95 latency | 20ms | <10ms | pg_stat_statements |
| Uptime | 99.9% | 99.95% | Monitoring |
| Connection pool efficiency | 60% | >80% | pg_stat_database |

### Qualitative Metrics
- [ ] Operations team confident in DR procedures
- [ ] Security audit passes with zero high-severity findings
- [ ] Performance benchmarks meet SLA requirements
- [ ] Documentation rated "Excellent" by 3+ reviewers
- [ ] Zero production incidents in first 30 days

---

## 🎯 Out of Scope

**Not Included in TN-098** (future tasks):
- Multi-replica HA setup (TN-99: PostgreSQL HA with Patroni)
- Connection pooling (TN-XXX: PgBouncer integration)
- Read replicas (TN-XXX: Read-only replicas for analytics)
- Cross-region replication (TN-XXX: Disaster recovery)
- PostgreSQL major version upgrades (separate runbook)
- Database schema migrations (handled by application)

---

## 📚 References

### Documentation
- [PostgreSQL 16 Documentation](https://www.postgresql.org/docs/16/)
- [Kubernetes StatefulSet Best Practices](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
- [CIS PostgreSQL Benchmark](https://www.cisecurity.org/benchmark/postgresql)
- [pgBackRest Documentation](https://pgbackrest.org/user-guide.html)

### Internal References
- TN-200: Deployment Profile Configuration
- TN-201: Storage Backend Selection Logic
- TN-97: HPA Configuration
- ROADMAP.md: Deployment Profiles section

---

**Status**: 📝 **REQUIREMENTS COMPLETE**
**Next**: Create design.md (architecture, components, implementation plan)
**Estimated**: 2-3 hours for comprehensive design document
