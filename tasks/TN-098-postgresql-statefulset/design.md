# TN-098: PostgreSQL StatefulSet - Design Document

**Date**: 2025-11-30
**Version**: 1.0
**Status**: Draft → Review → **Approved**
**Target Quality**: 150% (Grade A+ EXCEPTIONAL)

---

## 📐 Architecture Overview

### System Context

```
┌──────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                        │
│                                                               │
│  ┌─────────────────────┐       ┌──────────────────────────┐ │
│  │ Alert History Pods  │       │   PostgreSQL StatefulSet  │ │
│  │  (2-10 replicas)    │──────▶│      (TN-098)            │ │
│  │                     │       │                          │ │
│  │  Connection Pool:   │       │  ┌──────────────────┐   │ │
│  │  20 conns/pod       │       │  │  PostgreSQL 16   │   │ │
│  └─────────────────────┘       │  │  (Primary)       │   │ │
│                                │  │                  │   │ │
│  ┌─────────────────────┐       │  │  Resources:      │   │ │
│  │ Prometheus          │──────▶│  │  - 2Gi RAM       │   │ │
│  │ (Metrics Scraping)  │       │  │  - 1000m CPU     │   │ │
│  └─────────────────────┘       │  │  - 10Gi PVC      │   │ │
│                                │  │                  │   │ │
│  ┌─────────────────────┐       │  └──────────────────┘   │ │
│  │ Backup CronJob      │──────▶│                          │ │
│  │ (Daily 2 AM)        │       │  postgres-exporter       │ │
│  └─────────────────────┘       │  (Sidecar)               │ │
│                                └──────────────────────────┘ │
│                                         │                    │
│                                         ▼                    │
│                                ┌─────────────────┐          │
│                                │ PersistentVolume│          │
│                                │  (10Gi SSD)     │          │
│                                └─────────────────┘          │
│                                         │                    │
│                                         ▼                    │
│                                ┌─────────────────┐          │
│                                │ S3/GCS/Azure    │          │
│                                │ (Backups)       │          │
│                                └─────────────────┘          │
└──────────────────────────────────────────────────────────────┘
```

---

## 🏗️ Component Design

### 1. StatefulSet Architecture

```yaml
# High-level structure
StatefulSet
├── metadata
│   ├── name: alert-history-postgresql
│   ├── labels: {...}
│   └── annotations:
│       ├── tn-97: "HPA support"
│       └── tn-98: "150% quality"
├── spec
│   ├── serviceName: alert-history-postgresql (headless)
│   ├── replicas: 1
│   ├── podManagementPolicy: OrderedReady
│   ├── updateStrategy: RollingUpdate
│   ├── selector: {...}
│   ├── template (Pod spec)
│   │   ├── containers
│   │   │   ├── postgres (main)
│   │   │   └── postgres-exporter (sidecar)
│   │   ├── volumes
│   │   │   ├── postgres-data (PVC)
│   │   │   ├── postgres-config (ConfigMap)
│   │   │   └── postgres-tls (Secret)
│   │   └── securityContext: {...}
│   └── volumeClaimTemplates
│       └── postgres-data (10Gi RWO)
```

#### Design Decisions

**Decision 1: Single-Replica Baseline**
- **Rationale**: Start simple, scale to multi-replica in TN-99
- **Trade-offs**: No automatic failover, but simpler operations
- **Future**: Patroni/CloudNativePG for HA

**Decision 2: OrderedReady Pod Management**
- **Rationale**: Ensures ordered startup (critical for future multi-replica)
- **Benefits**: Predictable pod naming (postgresql-0, postgresql-1, ...)

**Decision 3: RollingUpdate Strategy**
- **Rationale**: Zero-downtime updates
- **Mechanism**: Update pods one-by-one with readiness checks

---

### 2. Container Design

#### 2.1 Main Container (PostgreSQL 16)

```yaml
containers:
- name: postgres
  image: postgres:16  # Official image, LTS support until 2028
  imagePullPolicy: IfNotPresent

  # Resource Limits (TN-98: Production-ready)
  resources:
    limits:
      cpu: 1000m      # 1 core
      memory: 2Gi     # 2GB RAM
    requests:
      cpu: 500m       # 0.5 core baseline
      memory: 1Gi     # 1GB RAM baseline

  # Ports
  ports:
  - name: postgres
    containerPort: 5432
    protocol: TCP

  # Environment Variables
  env:
  - name: POSTGRES_DB
    value: alert_history
  - name: POSTGRES_USER
    value: alert_history
  - name: POSTGRES_PASSWORD
    valueFrom:
      secretKeyRef:
        name: postgresql-secret
        key: password
  - name: PGDATA
    value: /var/lib/postgresql/data/pgdata  # Subdirectory to avoid mount issues
  - name: POSTGRES_INITDB_ARGS
    value: "--encoding=UTF8 --locale=en_US.UTF-8"

  # Volume Mounts
  volumeMounts:
  - name: postgres-data
    mountPath: /var/lib/postgresql/data
  - name: postgres-config
    mountPath: /etc/postgresql
    readOnly: true
  - name: postgres-tls
    mountPath: /etc/postgresql/tls
    readOnly: true

  # Probes (see section 3 for details)
  startupProbe: {...}
  livenessProbe: {...}
  readinessProbe: {...}

  # Lifecycle Hooks
  lifecycle:
    preStop:
      exec:
        command:
        - /bin/sh
        - -c
        - |
          # Graceful shutdown: wait for active connections to close
          su - postgres -c "pg_ctl stop -D $PGDATA -m fast -w -t 60" || true

  # Security Context
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop: [ALL]
    readOnlyRootFilesystem: false  # PostgreSQL needs writable /tmp
    runAsNonRoot: true
    runAsUser: 999   # postgres user
    runAsGroup: 999
```

**Resource Sizing Justification**:
```
Memory:
- shared_buffers: 512MB (25% of 2GB)
- effective_cache_size: 1.5GB (75% of 2GB)
- work_mem: 4MB × 250 connections = 1GB max
- Overhead: 500MB (OS, pg processes, logs)
Total: 2GB limit

CPU:
- Baseline: 0.5 core (idle, low traffic)
- Peak: 1 core (10K alerts/sec ingestion)
- Burst: Allowed via limits
```

#### 2.2 Sidecar Container (Prometheus Exporter)

```yaml
- name: postgres-exporter
  image: quay.io/prometheuscommunity/postgres-exporter:v0.15.0
  imagePullPolicy: IfNotPresent

  # Resource Limits (Lightweight)
  resources:
    limits:
      cpu: 100m
      memory: 128Mi
    requests:
      cpu: 50m
      memory: 64Mi

  # Metrics Port
  ports:
  - name: metrics
    containerPort: 9187
    protocol: TCP

  # Configuration
  env:
  - name: DATA_SOURCE_NAME
    value: |
      postgresql://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable
  - name: PG_EXPORTER_AUTO_DISCOVER_DATABASES
    value: "true"
  - name: PG_EXPORTER_EXTEND_QUERY_PATH
    value: /etc/postgres-exporter/queries.yaml
  - name: PG_EXPORTER_DISABLE_DEFAULT_METRICS
    value: "false"
  - name: PG_EXPORTER_DISABLE_SETTINGS_METRICS
    value: "false"

  # Custom Queries Mount
  volumeMounts:
  - name: exporter-config
    mountPath: /etc/postgres-exporter
    readOnly: true

  # Probes
  livenessProbe:
    httpGet:
      path: /
      port: metrics
    initialDelaySeconds: 30
    periodSeconds: 10

  readinessProbe:
    httpGet:
      path: /
      port: metrics
    initialDelaySeconds: 10
    periodSeconds: 5

  # Security Context
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop: [ALL]
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    runAsUser: 65534  # nobody
```

**Custom Metrics** (`exporter-config` ConfigMap):
```yaml
# /etc/postgres-exporter/queries.yaml
pg_alert_ingestion_rate:
  query: |
    SELECT
      COALESCE(COUNT(*), 0) as alerts_inserted,
      COALESCE(SUM(pg_table_size('alerts'::regclass)), 0) as table_size_bytes
    FROM alerts
    WHERE created_at > NOW() - INTERVAL '1 minute'
  metrics:
    - alerts_inserted:
        usage: COUNTER
        description: "Total alerts inserted in last minute"
    - table_size_bytes:
        usage: GAUGE
        description: "Alerts table size in bytes"

pg_connection_pool_usage:
  query: |
    SELECT
      COUNT(*) FILTER (WHERE state = 'active') as active_connections,
      COUNT(*) FILTER (WHERE state = 'idle') as idle_connections,
      COUNT(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction,
      COUNT(*) as total_connections,
      (COUNT(*)::float / {{ .Values.postgresql.config.maxConnections }}::float * 100) as usage_percent
    FROM pg_stat_activity
    WHERE datname = current_database()
  metrics:
    - active_connections:
        usage: GAUGE
    - idle_connections:
        usage: GAUGE
    - idle_in_transaction:
        usage: GAUGE
    - total_connections:
        usage: GAUGE
    - usage_percent:
        usage: GAUGE

pg_slow_queries:
  query: |
    SELECT
      COUNT(*) as slow_queries_total,
      AVG(mean_exec_time)::int as avg_exec_time_ms,
      MAX(mean_exec_time)::int as max_exec_time_ms
    FROM pg_stat_statements
    WHERE mean_exec_time > 1000  # > 1 second
  metrics:
    - slow_queries_total:
        usage: GAUGE
    - avg_exec_time_ms:
        usage: GAUGE
    - max_exec_time_ms:
        usage: GAUGE
```

---

### 3. Probe Design

#### 3.1 Startup Probe (Initial Readiness)
```yaml
startupProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - |
      pg_isready -U {{ .Values.postgresql.username }} \
                 -d {{ .Values.postgresql.database }} \
                 -h 127.0.0.1
  initialDelaySeconds: 10  # Wait 10s before first check
  periodSeconds: 10        # Check every 10s
  timeoutSeconds: 5        # Timeout after 5s
  successThreshold: 1      # One success = ready
  failureThreshold: 30     # 30 failures × 10s = 5 min max startup time
```

**Design Rationale**:
- PostgreSQL 16 startup: ~30-60s (cold start with large shared_buffers)
- Max wait: 5 minutes (handles crash recovery, WAL replay)
- Use `pg_isready` (lightweight, doesn't need credentials)

#### 3.2 Liveness Probe (Container Health)
```yaml
livenessProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - |
      pg_isready -U {{ .Values.postgresql.username }} \
                 -d {{ .Values.postgresql.database }} \
                 -h 127.0.0.1
  initialDelaySeconds: 60  # Wait 1 min after startup
  periodSeconds: 10        # Check every 10s
  timeoutSeconds: 5        # Timeout after 5s
  successThreshold: 1      # One success = healthy
  failureThreshold: 6      # 6 failures × 10s = 60s grace period
```

**Design Rationale**:
- Detects PostgreSQL process crashes (rare, but possible)
- 60s grace period prevents flapping during heavy load
- Triggers pod restart if database is truly hung

#### 3.3 Readiness Probe (Traffic Routing)
```yaml
readinessProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - |
      pg_isready -U {{ .Values.postgresql.username }} \
                 -d {{ .Values.postgresql.database }} \
                 -h 127.0.0.1 &&
      psql -U {{ .Values.postgresql.username }} \
           -d {{ .Values.postgresql.database }} \
           -h 127.0.0.1 \
           -tAc "SELECT 1" > /dev/null
  initialDelaySeconds: 10  # Check soon after startup probe passes
  periodSeconds: 5         # Check every 5s
  timeoutSeconds: 5        # Timeout after 5s
  successThreshold: 1      # One success = ready for traffic
  failureThreshold: 3      # 3 failures × 5s = 15s grace period
```

**Design Rationale**:
- **Deep health check**: Not just process alive, but can execute queries
- **Fast failover**: 5s check interval ensures quick detection
- **Prevents routing to degraded pods**: If queries fail, remove from Service

---

### 4. Configuration Management

#### 4.1 ConfigMap Structure

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgresql-config
data:
  # Primary configuration file (110+ lines)
  postgresql.conf: |
    # Generated from values.yaml postgresql.config
    # See helm/alert-history/templates/postgresql-configmap.yaml

  # Client authentication (pg_hba.conf)
  pg_hba.conf: |
    # TYPE  DATABASE        USER            ADDRESS                 METHOD
    # Local connections (Unix domain sockets)
    local   all             all                                     trust
    local   replication     all                                     trust

    # IPv4 local connections
    host    all             all             127.0.0.1/32            trust

    # IPv4 cluster connections (TN-98: require auth)
    host    all             all             10.0.0.0/8              scram-sha-256

    # IPv6 cluster connections
    host    all             all             ::1/128                 trust

    # TLS connections (when SSL enabled)
    hostssl all             all             0.0.0.0/0               scram-sha-256

  # Initialization SQL (extensions, users)
  init.sql: |
    -- Extensions (TN-98: Monitoring & Auditing)
    CREATE EXTENSION IF NOT EXISTS pg_stat_statements;
    CREATE EXTENSION IF NOT EXISTS pg_audit;
    CREATE EXTENSION IF NOT EXISTS pgcrypto;  # For password hashing

    -- Create monitoring user (read-only)
    CREATE USER monitoring WITH PASSWORD '{{ .Values.postgresql.monitoring.password }}';
    GRANT CONNECT ON DATABASE alert_history TO monitoring;
    GRANT SELECT ON ALL TABLES IN SCHEMA public TO monitoring;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO monitoring;

    -- Create backup user (for pgBackRest)
    CREATE USER backup WITH REPLICATION;
    GRANT CONNECT ON DATABASE alert_history TO backup;
```

#### 4.2 Configuration Lifecycle

```
┌──────────────┐
│ values.yaml  │
│ .postgresql  │
│   .config    │
└──────┬───────┘
       │
       ▼
┌────────────────────────┐
│ Helm Template Engine   │
│ (postgresql-configmap) │
└──────┬─────────────────┘
       │
       ▼
┌───────────────────┐
│ ConfigMap Applied │
│ (checksum: abc123)│
└──────┬────────────┘
       │
       ▼
┌────────────────────────┐
│ StatefulSet Annotation │
│ checksum/config: abc123│
└──────┬─────────────────┘
       │
       ▼ (config changed)
┌───────────────────────┐
│ Rolling Update Trigger│
│ (pods restart)        │
└───────────────────────┘
```

**Hot Reload Support** (non-critical params):
```bash
# Values that can be reloaded with SIGHUP (no restart)
- work_mem
- maintenance_work_mem
- autovacuum settings
- logging settings
- statistics settings

# Values that require restart
- max_connections
- shared_buffers
- wal_level
- max_wal_senders
```

---

### 5. Storage Design

#### 5.1 PersistentVolumeClaim

```yaml
volumeClaimTemplates:
- metadata:
    name: postgres-data
    labels:
      app.kubernetes.io/component: database
  spec:
    accessModes:
    - ReadWriteOnce  # Single node access (StatefulSet requirement)

    resources:
      requests:
        storage: {{ .Values.postgresql.persistence.size | default "10Gi" }}

    {{- if .Values.postgresql.persistence.storageClass }}
    storageClassName: {{ .Values.postgresql.persistence.storageClass }}
    {{- end }}

    # Optional: Selector for pre-provisioned PVs
    {{- if .Values.postgresql.persistence.selector }}
    selector:
      matchLabels:
        {{- toYaml .Values.postgresql.persistence.selector | nindent 8 }}
    {{- end }}
```

#### 5.2 Storage Classes by Environment

```yaml
# Development (Fast, Ephemeral)
storageClass: local-path  # or hostPath
size: 5Gi
performance: ~500 IOPS

# Staging (Persistent, Good Performance)
storageClass: gp3  # AWS EBS gp3
size: 20Gi
iops: 3000
throughput: 125 MBps

# Production (High Performance, Persistent)
storageClass: io2  # AWS EBS io2 (provisioned IOPS)
size: 100Gi
iops: 10000  # Dedicated IOPS
throughput: 500 MBps
```

#### 5.3 Data Layout

```
/var/lib/postgresql/data/  (PVC mount point)
└── pgdata/  (PGDATA)
    ├── base/  (Database files)
    │   ├── 1/  (template1)
    │   ├── 16384/  (alert_history database)
    │   │   ├── 16385  (alerts table file)
    │   │   ├── 16385_fsm  (free space map)
    │   │   ├── 16385_vm  (visibility map)
    │   │   └── ...
    ├── global/  (Cluster-wide tables)
    ├── pg_wal/  (Write-Ahead Log, 16MB segments)
    │   ├── 000000010000000000000001
    │   ├── 000000010000000000000002
    │   └── ...
    ├── pg_xact/  (Transaction commit status)
    ├── pg_log/  (PostgreSQL logs, if logging_collector=on)
    ├── postgresql.conf  (Configuration)
    ├── pg_hba.conf  (Authentication)
    └── postmaster.pid  (Lock file, running instance)
```

**Capacity Planning**:
```
Alerts Table:
- Average alert size: 2KB (JSON payload)
- Daily alerts: 1M (high volume)
- Retention: 90 days
- Total data: 1M × 90 × 2KB = 180GB

Indexes (~30% overhead): 54GB
WAL files (max_wal_size=4GB): 4GB
Temporary files: 10GB
Total: 248GB

Recommendation: 500GB PVC (2x growth capacity)
```

---

### 6. Networking Design

#### 6.1 Service Topology

```
┌──────────────────────────────────────────────────┐
│                 Kubernetes Cluster                │
│                                                   │
│  ┌──────────────┐         ┌──────────────────┐  │
│  │ Client Pods  │────────▶│ Service          │  │
│  │              │         │ (postgresql-rw)  │  │
│  └──────────────┘         │ ClusterIP        │  │
│                           └─────────┬────────┘  │
│                                     │           │
│                                     ▼           │
│                           ┌──────────────────┐  │
│                           │ StatefulSet Pods │  │
│                           │ postgresql-0     │  │
│                           │ (Primary)        │  │
│                           └─────────┬────────┘  │
│                                     │           │
│                                     ▼           │
│                           ┌──────────────────┐  │
│                           │ Headless Service │  │
│                           │ (postgresql)     │  │
│                           │ ClusterIP: None  │  │
│                           └──────────────────┘  │
│                                                   │
│  ┌──────────────┐                                │
│  │ Prometheus   │────────────────────────────────┤
│  │              │   (ServiceMonitor scrapes      │
│  └──────────────┘    postgresql-exporter:9187)  │
└──────────────────────────────────────────────────┘
```

#### 6.2 Service Definitions

**Headless Service** (StatefulSet requirement):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgresql
  labels:
    app.kubernetes.io/component: database
spec:
  clusterIP: None  # Headless
  publishNotReadyAddresses: false
  selector:
    app.kubernetes.io/component: database
  ports:
  - name: postgres
    port: 5432
    targetPort: postgres
```

**ClusterIP Service** (client access):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgresql-rw  # Read-Write endpoint
  labels:
    app.kubernetes.io/component: database
spec:
  type: ClusterIP
  sessionAffinity: None  # Round-robin
  selector:
    app.kubernetes.io/component: database
  ports:
  - name: postgres
    port: 5432
    targetPort: postgres
    protocol: TCP
```

**Exporter Service** (metrics scraping):
```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgresql-exporter
  labels:
    app.kubernetes.io/component: database-exporter
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9187"
    prometheus.io/path: "/metrics"
spec:
  type: ClusterIP
  selector:
    app.kubernetes.io/component: database
  ports:
  - name: metrics
    port: 9187
    targetPort: metrics
```

---

### 7. Security Design

#### 7.1 Pod Security Context

```yaml
spec:
  securityContext:
    # Pod-level context
    runAsNonRoot: true
    fsGroup: 999  # postgres group
    supplementalGroups: [999]
    seccompProfile:
      type: RuntimeDefault

  containers:
  - name: postgres
    securityContext:
      # Container-level context (more restrictive)
      allowPrivilegeEscalation: false
      capabilities:
        drop: [ALL]
      readOnlyRootFilesystem: false  # PostgreSQL needs /tmp
      runAsUser: 999   # postgres user
      runAsGroup: 999
      runAsNonRoot: true
```

**Justification**:
- `runAsNonRoot`: Prevents root exploits
- `fsGroup=999`: Ensures PVC ownership for postgres user
- `capabilities.drop=[ALL]`: Minimal Linux capabilities
- `seccompProfile=RuntimeDefault`: Syscall filtering
- `readOnlyRootFilesystem=false`: PostgreSQL needs writable `/tmp` for temp files

#### 7.2 Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: postgresql-ingress
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/component: database

  policyTypes:
  - Ingress
  - Egress

  # Ingress Rules
  ingress:
  # Allow from application pods
  - from:
    - namespaceSelector:
        matchLabels:
          name: {{ .Release.Namespace }}
      podSelector:
        matchLabels:
          app.kubernetes.io/name: alert-history
    ports:
    - protocol: TCP
      port: 5432

  # Allow from Prometheus
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/name: prometheus
    ports:
    - protocol: TCP
      port: 9187  # Exporter

  # Egress Rules
  egress:
  # Allow DNS
  - to:
    - namespaceSelector:
        matchLabels:
          name: kube-system
      podSelector:
        matchLabels:
          k8s-app: kube-dns
    ports:
    - protocol: UDP
      port: 53

  # Allow backup to S3/GCS (egress to cloud storage)
  - to:
    - podSelector: {}  # Any pod (S3/GCS via HTTPS)
    ports:
    - protocol: TCP
      port: 443
```

#### 7.3 TLS Configuration (Optional)

```yaml
# When postgresql.tls.enabled=true
volumes:
- name: postgres-tls
  secret:
    secretName: {{ .Values.postgresql.tls.certificatesSecret }}
    defaultMode: 0400  # Read-only for postgres user

# Mount in container
volumeMounts:
- name: postgres-tls
  mountPath: /etc/postgresql/tls
  readOnly: true

# Configuration
postgresql.conf: |
  ssl = on
  ssl_cert_file = '/etc/postgresql/tls/tls.crt'
  ssl_key_file = '/etc/postgresql/tls/tls.key'
  ssl_ca_file = '/etc/postgresql/tls/ca.crt'
  ssl_ciphers = 'HIGH:MEDIUM:+3DES:!aNULL'
  ssl_prefer_server_ciphers = on
  ssl_min_protocol_version = 'TLSv1.3'
```

---

### 8. Backup & Restore Design

#### 8.1 Backup Strategy

```
┌────────────────────────────────────────────────────┐
│           Backup Strategy (3-2-1 Rule)             │
│                                                    │
│  3 Copies: Primary DB + Local Backup + Cloud      │
│  2 Media: PVC (local) + S3 (cloud)                │
│  1 Offsite: S3 in different region                │
└────────────────────────────────────────────────────┘

┌──────────────┐
│ PostgreSQL   │
│ (Primary)    │
└──────┬───────┘
       │
       ▼ Continuous WAL Archiving
┌──────────────────────┐
│ S3 Bucket            │
│ /alerthistory/       │
│   ├── base/          │  Full Backups (Weekly)
│   │   ├── 20251130/  │
│   │   └── 20251123/  │
│   └── wal/           │  WAL Archives (15-min)
│       ├── 000001.gz  │
│       └── 000002.gz  │
└──────────────────────┘
       │
       ▼ Replication (Optional)
┌──────────────────────┐
│ S3 Bucket (DR)       │
│ us-west-2            │
│ (Cross-region copy)  │
└──────────────────────┘
```

#### 8.2 Backup CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgresql-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM UTC
  concurrencyPolicy: Forbid  # Prevent overlapping backups
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 1

  jobTemplate:
    spec:
      backoffLimit: 2  # Retry twice on failure
      template:
        spec:
          restartPolicy: OnFailure

          # Service Account (with S3 access)
          serviceAccountName: postgresql-backup

          containers:
          - name: pgbackrest
            image: pgbackrest/pgbackrest:latest

            command:
            - /bin/bash
            - -c
            - |
              set -euo pipefail

              # Environment
              export PGBACKREST_STANZA=alerthistory
              export PGBACKREST_CONFIG=/etc/pgbackrest/pgbackrest.conf

              # Log
              echo "[$(date)] Starting backup..."

              # Create backup
              pgbackrest backup \
                --stanza=$PGBACKREST_STANZA \
                --type=full \
                --log-level-console=info

              # Verify
              pgbackrest info --stanza=$PGBACKREST_STANZA

              echo "[$(date)] Backup completed successfully"

            env:
            # PostgreSQL Connection
            - name: PGHOST
              value: postgresql-rw
            - name: PGPORT
              value: "5432"
            - name: PGDATABASE
              value: alert_history
            - name: PGUSER
              value: backup
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgresql-secret
                  key: backup_password

            # AWS Credentials (for S3)
            - name: AWS_ACCESS_KEY_ID
              valueFrom:
                secretKeyRef:
                  name: backup-credentials
                  key: access_key_id
            - name: AWS_SECRET_ACCESS_KEY
              valueFrom:
                secretKeyRef:
                  name: backup-credentials
                  key: secret_access_key
            - name: AWS_DEFAULT_REGION
              value: us-east-1

            volumeMounts:
            - name: pgbackrest-config
              mountPath: /etc/pgbackrest
              readOnly: true

          volumes:
          - name: pgbackrest-config
            configMap:
              name: pgbackrest-config
```

#### 8.3 WAL Archiving (PITR)

```yaml
# In postgresql.conf
archive_mode = on
archive_command = 'pgbackrest --stanza=alerthistory archive-push %p'
archive_timeout = 900  # Force WAL switch every 15 minutes

# Enables Point-in-Time Recovery to any timestamp within retention window
```

---

### 9. Monitoring Design

#### 9.1 Grafana Dashboard Layout

```
┌──────────────────────────────────────────────────────────────┐
│               PostgreSQL Dashboard (20 Panels)                │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  Row 1: Overview                                              │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐           │
│  │ Uptime      │ │ DB Size     │ │ Connections │           │
│  │ 99.95%      │ │ 245 GB      │ │ 156/250     │           │
│  └─────────────┘ └─────────────┘ └─────────────┘           │
│                                                               │
│  Row 2: Performance                                           │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Transaction Rate (graph)                                │ │
│  │ 10K/sec ████████████████████████                        │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Query Latency p50/p95/p99 (graph)                      │ │
│  │ p50: 5ms, p95: 12ms, p99: 45ms                         │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                               │
│  Row 3: Resource Usage                                        │
│  ┌────────────────────┐ ┌──────────────────────────────────┐│
│  │ Cache Hit Ratio    │ │ Disk I/O (read/write IOPS)      ││
│  │ 92% ████████       │ │ Read: 5K, Write: 2K             ││
│  └────────────────────┘ └──────────────────────────────────┘│
│                                                               │
│  Row 4: Tables & Indexes                                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ Top 10 Largest Tables (table)                          │ │
│  │ alerts: 180GB, silences: 5GB, ...                      │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                               │
│  Row 5: Operations                                            │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐        │
│  │ Autovacuum   │ │ Checkpoints  │ │ Locks        │        │
│  │ Active: 1    │ │ 8/hour       │ │ 12 active    │        │
│  └──────────────┘ └──────────────┘ └──────────────┘        │
└──────────────────────────────────────────────────────────────┘
```

#### 9.2 Alerting Rules

```yaml
groups:
- name: postgresql
  interval: 30s
  rules:

  # Connection Pool Saturation
  - alert: PostgreSQLConnectionPoolNearExhaustion
    expr: |
      (pg_stat_database_numbackends{datname="alert_history"}
       / {{ .Values.postgresql.config.maxConnections }}) > 0.8
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "PostgreSQL connection pool usage >80%"
      description: "{{ $value | humanizePercentage }} connections used"

  # Replication Lag (future HA mode)
  - alert: PostgreSQLReplicationLag
    expr: pg_replication_lag_seconds > 30
    for: 2m
    labels:
      severity: warning
    annotations:
      summary: "PostgreSQL replication lag >30s"

  # Disk Usage
  - alert: PostgreSQLDiskNearFull
    expr: |
      (kubelet_volume_stats_used_bytes{persistentvolumeclaim="postgres-data-postgresql-0"}
       / kubelet_volume_stats_capacity_bytes{persistentvolumeclaim="postgres-data-postgresql-0"}) > 0.85
    for: 10m
    labels:
      severity: critical
    annotations:
      summary: "PostgreSQL disk usage >85%"

  # Slow Queries
  - alert: PostgreSQLSlowQueries
    expr: |
      sum(rate(pg_stat_statements_mean_time_seconds{datname="alert_history"}[5m])) > 5
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "PostgreSQL has queries taking >5s"

  # Cache Hit Ratio
  - alert: PostgreSQLLowCacheHitRatio
    expr: |
      (pg_stat_database_blks_hit{datname="alert_history"}
       / (pg_stat_database_blks_hit{datname="alert_history"}
          + pg_stat_database_blks_read{datname="alert_history"})) < 0.9
    for: 15m
    labels:
      severity: warning
    annotations:
      summary: "PostgreSQL cache hit ratio <90%"

  # Transaction Wraparound Risk
  - alert: PostgreSQLTransactionWraparoundRisk
    expr: |
      (pg_database_age_datfrozenxid{datname="alert_history"}
       / 2000000000) > 0.9
    for: 1h
    labels:
      severity: critical
    annotations:
      summary: "PostgreSQL transaction wraparound risk >90%"
```

---

## 📋 Implementation Phases

### Phase 1: Baseline Enhancement (4h)
**Goal**: Upgrade existing StatefulSet to production-ready baseline

**Tasks**:
1. [ ] Enhance StatefulSet spec (affinity, tolerations, terminationGracePeriod)
2. [ ] Add postgres-exporter sidecar
3. [ ] Improve probes (readinessProbe with SELECT 1)
4. [ ] Add lifecycle hooks (preStop graceful shutdown)
5. [ ] Update ConfigMap with 150% quality config
6. [ ] Add pg_hba.conf and init.sql
7. [ ] Update Secret with TLS certificates support
8. [ ] Create Services (headless, ClusterIP, exporter)
9. [ ] Add PodDisruptionBudget
10. [ ] Update values.yaml with comprehensive defaults

**Deliverables**:
- Enhanced YAML templates (10 files)
- Unit tests (10+ test cases)
- Documentation updates

### Phase 2: Backup & DR (3h)
**Goal**: Implement automated backup/restore

**Tasks**:
1. [ ] Create backup CronJob (pgBackRest)
2. [ ] Configure WAL archiving
3. [ ] Create backup-credentials Secret
4. [ ] Create backup ConfigMap
5. [ ] Write backup verification CronJob
6. [ ] Create restore scripts
7. [ ] Document DR runbook

**Deliverables**:
- Backup infrastructure (5 files)
- DR runbook (20+ pages)
- Restore testing guide

### Phase 3: Monitoring & Alerts (3h)
**Goal**: Comprehensive observability

**Tasks**:
1. [ ] Create custom metrics ConfigMap (exporter-config)
2. [ ] Create Grafana dashboard JSON
3. [ ] Create PrometheusRule (15+ alerts)
4. [ ] Create ServiceMonitor (Prometheus scraping)
5. [ ] Document PromQL queries

**Deliverables**:
- Monitoring stack (4 files)
- Grafana dashboard (20+ panels)
- Alert runbook (15+ scenarios)

### Phase 4: Security Hardening (2h)
**Goal**: Enterprise-grade security

**Tasks**:
1. [ ] Implement TLS support (optional)
2. [ ] Create NetworkPolicy
3. [ ] Enhance Pod Security Standards
4. [ ] Create RBAC for backup ServiceAccount
5. [ ] Document security compliance

**Deliverables**:
- Security manifests (4 files)
- Security audit report
- Compliance checklist (CIS Benchmark)

### Phase 5: Testing & Validation (2h)
**Goal**: Comprehensive testing

**Tasks**:
1. [ ] Helm unit tests (helm unittest)
2. [ ] Integration tests (deploy + validate)
3. [ ] Performance benchmarks (pgbench)
4. [ ] Chaos testing (kill pods, delete PVC)
5. [ ] Backup/restore testing

**Deliverables**:
- Test suite (20+ tests)
- Benchmark results
- Test reports

### Phase 6: Documentation & Certification (2h)
**Goal**: 150% quality documentation

**Tasks**:
1. [ ] Update README.md
2. [ ] Create OPERATIONS_RUNBOOK.md
3. [ ] Create TROUBLESHOOTING.md
4. [ ] Create DEPLOYMENT_GUIDE.md
5. [ ] Create COMPLETION_REPORT.md
6. [ ] Certification checklist

**Deliverables**:
- Documentation (5 files, 5,000+ LOC)
- Completion report
- Grade A+ certification

---

## 🎯 Success Criteria

### Baseline (100%)
- [ ] StatefulSet deploys successfully
- [ ] PostgreSQL 16 accepts connections
- [ ] Data persists across pod restarts
- [ ] All probes pass
- [ ] Services reachable

### 150% Target
- [ ] All baseline + 10 of 15 enhancements (see requirements.md)
- [ ] Performance benchmarks pass (>10K inserts/sec)
- [ ] Security audit passes (zero high-severity)
- [ ] Documentation rated "Excellent"
- [ ] Grade A+ certification achieved

---

**Status**: 📝 **DESIGN COMPLETE**
**Next**: Create tasks.md (detailed implementation checklist)
**Estimated**: 1 hour for comprehensive tasks breakdown
