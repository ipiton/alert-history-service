# TN-99: Redis/Valkey StatefulSet - Technical Design

**Task ID**: TN-99
**Document Type**: Technical Design
**Status**: 📐 **DESIGN COMPLETE**
**Target Quality**: **150% (Grade A+ EXCEPTIONAL)**
**Version**: 1.0
**Last Updated**: 2025-11-30

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [StatefulSet Design](#2-statefulset-design)
3. [Configuration Management](#3-configuration-management)
4. [Service Architecture](#4-service-architecture)
5. [Persistent Storage Design](#5-persistent-storage-design)
6. [Monitoring & Observability](#6-monitoring--observability)
7. [Security Architecture](#7-security-architecture)
8. [Deployment Workflow](#8-deployment-workflow)
9. [Integration Points](#9-integration-points)
10. [Testing Strategy](#10-testing-strategy)
11. [Operational Procedures](#11-operational-procedures)
12. [Design Decisions](#12-design-decisions)

---

## 1. Architecture Overview

### 1.1 High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         Standard Profile Architecture                        │
│                                                                               │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                     Application Layer (HPA 2-10)                      │  │
│  │                                                                        │  │
│  │  ┌─────────┐  ┌─────────┐  ┌─────────┐       ┌─────────┐            │  │
│  │  │ App-0   │  │ App-1   │  │ App-2   │  ...  │ App-9   │            │  │
│  │  │         │  │         │  │         │       │         │            │  │
│  │  │ L1: LRU │  │ L1: LRU │  │ L1: LRU │       │ L1: LRU │            │  │
│  │  │ 1K items│  │ 1K items│  │ 1K items│       │ 1K items│            │  │
│  │  │ ~5MB    │  │ ~5MB    │  │ ~5MB    │       │ ~5MB    │            │  │
│  │  └────┬────┘  └────┬────┘  └────┬────┘       └────┬────┘            │  │
│  │       │            │            │                  │                 │  │
│  │       │            │            │                  │                 │  │
│  │       └────────────┴────────────┴──────────────────┘                 │  │
│  │                           │                                           │  │
│  └───────────────────────────┼───────────────────────────────────────────┘  │
│                              │                                               │
│                              ▼                                               │
│               ┌──────────────────────────────┐                              │
│               │   Redis Service (ClusterIP)  │                              │
│               │   alerthistory-redis:6379    │                              │
│               └──────────────┬───────────────┘                              │
│                              │                                               │
│                              ▼                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                      Redis/Valkey StatefulSet                          │  │
│  │                                                                         │  │
│  │  ┌────────────────────────────────────────────────────────────────┐   │  │
│  │  │  Pod: alerthistory-redis-0 (PRIMARY)                           │   │  │
│  │  │                                                                 │   │  │
│  │  │  ┌────────────────┐         ┌──────────────────┐              │   │  │
│  │  │  │  Redis         │         │ redis-exporter   │              │   │  │
│  │  │  │  Container     │         │ (sidecar)        │              │   │  │
│  │  │  │                │         │                  │              │   │  │
│  │  │  │  Port: 6379    │         │  Port: 9121      │              │   │  │
│  │  │  │  Memory: 512Mi │         │  Memory: 128Mi   │              │   │  │
│  │  │  │  CPU: 500m     │         │  CPU: 100m       │              │   │  │
│  │  │  │                │         │                  │              │   │  │
│  │  │  │  ┌──────────┐  │         │  ┌────────────┐  │              │   │  │
│  │  │  │  │ L2 Cache │  │         │  │  50+ Metrics│  │              │   │  │
│  │  │  │  │ 384MB    │  │         │  │  Exporter   │  │              │   │  │
│  │  │  │  │ LRU      │  │         │  └────────────┘  │              │   │  │
│  │  │  │  └──────────┘  │         └──────────────────┘              │   │  │
│  │  │  └────────┬───────┘                                            │   │  │
│  │  │           │                                                    │   │  │
│  │  │           ▼                                                    │   │  │
│  │  │  ┌────────────────────────────────────┐                       │   │  │
│  │  │  │  Persistent Volume Claim (PVC)     │                       │   │  │
│  │  │  │  alerthistory-redis-data-0         │                       │   │  │
│  │  │  │                                    │                       │   │  │
│  │  │  │  Size: 5Gi (RWO)                   │                       │   │  │
│  │  │  │  Mount: /data                      │                       │   │  │
│  │  │  │                                    │                       │   │  │
│  │  │  │  ┌──────────────┐ ┌──────────────┐ │                       │   │  │
│  │  │  │  │ appendonly.  │ │   dump.rdb   │ │                       │   │  │
│  │  │  │  │    aof       │ │   (snapshot) │ │                       │   │  │
│  │  │  │  │  (AOF log)   │ │              │ │                       │   │  │
│  │  │  │  └──────────────┘ └──────────────┘ │                       │   │  │
│  │  │  └────────────────────────────────────┘                       │   │  │
│  │  └─────────────────────────────────────────────────────────────────┘   │  │
│  │                                                                         │  │
│  │  ┌─────────────────────────────────────────────────────────────────┐   │  │
│  │  │  Future: Redis Sentinel HA (Expandable to 3 replicas)          │   │  │
│  │  │  - redis-1 (REPLICA) with PVC-1                                 │   │  │
│  │  │  - redis-2 (REPLICA) with PVC-2                                 │   │  │
│  │  │  - Automatic failover via Sentinel quorum                       │   │  │
│  │  └─────────────────────────────────────────────────────────────────┘   │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        Monitoring Stack                                │  │
│  │                                                                         │  │
│  │  ┌─────────────┐      ┌──────────────┐      ┌──────────────┐         │  │
│  │  │ redis-      │ ───▶ │  Prometheus  │ ───▶ │   Grafana    │         │  │
│  │  │ exporter    │      │  (scrape     │      │  (Dashboard  │         │  │
│  │  │ :9121       │      │   30s)       │      │   ID: 11835) │         │  │
│  │  │             │      │              │      │              │         │  │
│  │  │ 50+ metrics │      │ 10 alerts    │      │ 12 panels    │         │  │
│  │  └─────────────┘      └──────────────┘      └──────────────┘         │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │                        Security Layer                                  │  │
│  │                                                                         │  │
│  │  ┌────────────────┐    ┌───────────────┐    ┌─────────────────┐      │  │
│  │  │ NetworkPolicy  │    │    Secret     │    │      RBAC       │      │  │
│  │  │ (pod isolation)│    │  (password)   │    │ (minimal perms) │      │  │
│  │  │                │    │               │    │                 │      │  │
│  │  │ Allow: app pods│    │ Base64 encoded│    │ get/list secrets│      │  │
│  │  │ Deny: external │    │ Auto-rotation │    │                 │      │  │
│  │  └────────────────┘    └───────────────┘    └─────────────────┘      │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
```

### 1.2 Component Responsibilities

| Component | Responsibility | Dependencies |
|-----------|---------------|--------------|
| **Application Pods** | L1 cache, business logic, API | Redis Service |
| **Redis StatefulSet** | L2 cache, persistence, state | PVC, ConfigMap, Secret |
| **Redis Service (ClusterIP)** | Load balancing to primary | StatefulSet Pod 0 |
| **Headless Service** | StatefulSet DNS discovery | StatefulSet |
| **Metrics Service** | Prometheus scraping | redis-exporter sidecar |
| **redis-exporter** | Metrics collection (50+) | Redis container |
| **PersistentVolume** | Data durability (AOF + RDB) | StorageClass |
| **ConfigMap** | Redis configuration | None |
| **Secret** | Password storage | External Secrets Operator (future) |
| **NetworkPolicy** | Pod isolation | None |
| **ServiceMonitor** | Prometheus auto-discovery | Prometheus Operator |
| **PrometheusRule** | Alerting rules (10 alerts) | Prometheus |

### 1.3 Data Flow

**Write Path** (App → Redis):
```
1. App pod receives alert
2. Classification service generates fingerprint
3. LLM classification result received
4. Check L1 cache (memory)
   - Hit: Return immediately (<5ms)
   - Miss: Continue to L2
5. Check L2 cache (Redis)
   - Hit: Populate L1, return (<10ms)
   - Miss: Continue to LLM
6. LLM API call (~500ms)
7. Store in L2 cache (Redis SET, 1h TTL)
8. Store in L1 cache (memory, 1h TTL)
9. Return result to caller
```

**Read Path** (App → Redis):
```
1. App pod queries classification
2. Check L1 cache (memory LRU)
   - Hit (90%): Return immediately (~5ms) ✅
   - Miss: Continue to L2
3. Check L2 cache (Redis GET)
   - Hit (3%): Populate L1, return (~10ms) ✅
   - Miss (7%): Return cache miss
4. Caller triggers LLM API (~500ms)
5. Result stored via write path
```

**Persistence Path** (Redis → Disk):
```
1. Redis receives SET command
2. Write to memory (in-memory data structure)
3. Append to AOF file (/data/appendonly.aof)
4. fsync to disk (everysec = every 1 second)
5. Periodic RDB snapshot (15min/5min/1min triggers)
6. RDB written to /data/dump.rdb
7. Both files persisted to PVC
```

### 1.4 Failure Modes & Recovery

| Failure Mode | Detection | Recovery Action | RTO | RPO |
|--------------|-----------|-----------------|-----|-----|
| **Redis pod crash** | kubelet (immediate) | Restart pod → AOF replay | <30s | <1s |
| **Node failure** | K8s scheduler (<30s) | Reschedule pod → PVC reattach → AOF replay | <60s | <1s |
| **Volume corruption** | Manual (monitoring alerts) | Restore from RDB snapshot | <5min | <15min |
| **Complete data loss** | Manual (all backups lost) | Rebuild cache from PostgreSQL classification history | <10min | Full rebuild |
| **Network partition** | Connection timeout (5s) | App falls back to memory-only | <5s | N/A (cache) |
| **Memory exhaustion** | Redis OOM (maxmemory) | LRU eviction (automatic) | <1s | 0 (oldest keys) |

---

## 2. StatefulSet Design

### 2.1 StatefulSet Manifest Structure

**File**: `helm/alert-history/templates/redis-statefulset.yaml`

```yaml
{{- if eq .Values.profile "standard" }}
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis
    app.kubernetes.io/part-of: cache-layer
spec:
  # StatefulSet Configuration
  replicas: {{ .Values.valkey.replicas | default 1 }}
  serviceName: {{ include "alerthistory.fullname" . }}-redis-headless
  podManagementPolicy: OrderedReady  # Sequential startup

  # Update Strategy
  updateStrategy:
    type: RollingUpdate
    rollingUpdate:
      partition: 0  # Update all pods

  # Selector
  selector:
    matchLabels:
      {{- include "alerthistory.selectorLabels" . | nindent 6 }}
      app.kubernetes.io/component: redis

  # Pod Template
  template:
    metadata:
      labels:
        {{- include "alerthistory.selectorLabels" . | nindent 8 }}
        app.kubernetes.io/component: redis
      annotations:
        checksum/config: {{ include (print $.Template.BasePath "/redis-config.yaml") . | sha256sum }}
        prometheus.io/scrape: "true"
        prometheus.io/port: "9121"
        prometheus.io/path: "/metrics"
    spec:
      # Service Account
      serviceAccountName: {{ include "alerthistory.serviceAccountName" . }}

      # Security Context (Pod-level)
      securityContext:
        fsGroup: 999  # redis user
        runAsNonRoot: true
        runAsUser: 999

      # Init Containers
      initContainers:
      - name: config-init
        image: busybox:1.36
        command:
        - sh
        - -c
        - |
          # Copy redis.conf to writable location
          cp /tmp/redis/redis.conf /data/redis.conf

          # Set password from secret
          if [ -f /tmp/secret/password ]; then
            echo "requirepass $(cat /tmp/secret/password)" >> /data/redis.conf
          fi

          # Ensure correct permissions
          chmod 644 /data/redis.conf
        volumeMounts:
        - name: config
          mountPath: /tmp/redis
        - name: redis-data
          mountPath: /data
        - name: redis-secret
          mountPath: /tmp/secret

      # Main Containers
      containers:
      # Redis Container
      - name: redis
        image: {{ .Values.valkey.image.repository }}:{{ .Values.valkey.image.tag | default "7-alpine" }}
        imagePullPolicy: {{ .Values.valkey.image.pullPolicy | default "IfNotPresent" }}

        # Command
        command:
        - redis-server
        - /data/redis.conf

        # Ports
        ports:
        - name: redis
          containerPort: 6379
          protocol: TCP

        # Environment Variables
        env:
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "alerthistory.fullname" . }}-redis-secret
              key: password

        # Resource Limits
        resources:
          {{- toYaml .Values.valkey.resources | nindent 10 }}

        # Liveness Probe
        livenessProbe:
          exec:
            command:
            - sh
            - -c
            - redis-cli -a $REDIS_PASSWORD ping | grep PONG
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3

        # Readiness Probe
        readinessProbe:
          exec:
            command:
            - sh
            - -c
            - redis-cli -a $REDIS_PASSWORD ping | grep PONG
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3

        # Startup Probe
        startupProbe:
          exec:
            command:
            - sh
            - -c
            - redis-cli -a $REDIS_PASSWORD ping | grep PONG
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 30

        # Volume Mounts
        volumeMounts:
        - name: redis-data
          mountPath: /data
        - name: config
          mountPath: /usr/local/etc/redis
          readOnly: true

        # Security Context (Container-level)
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: false  # Redis needs to write to /data
          runAsNonRoot: true
          runAsUser: 999

      # redis-exporter Sidecar
      - name: redis-exporter
        image: {{ .Values.valkey.exporter.image }}:{{ .Values.valkey.exporter.tag | default "v1.55.0" }}
        imagePullPolicy: {{ .Values.valkey.exporter.pullPolicy | default "IfNotPresent" }}

        # Environment Variables
        env:
        - name: REDIS_ADDR
          value: "localhost:6379"
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: {{ include "alerthistory.fullname" . }}-redis-secret
              key: password

        # Ports
        ports:
        - name: metrics
          containerPort: 9121
          protocol: TCP

        # Resource Limits
        resources:
          {{- toYaml .Values.valkey.exporter.resources | nindent 10 }}

        # Liveness Probe
        livenessProbe:
          httpGet:
            path: /metrics
            port: 9121
          initialDelaySeconds: 30
          periodSeconds: 10

        # Readiness Probe
        readinessProbe:
          httpGet:
            path: /metrics
            port: 9121
          initialDelaySeconds: 5
          periodSeconds: 5

        # Security Context
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          readOnlyRootFilesystem: true
          runAsNonRoot: true
          runAsUser: 999

      # Volumes
      volumes:
      - name: config
        configMap:
          name: {{ include "alerthistory.fullname" . }}-redis-config
      - name: redis-secret
        secret:
          secretName: {{ include "alerthistory.fullname" . }}-redis-secret

      # Node Affinity (optional)
      {{- with .Values.nodeSelector }}
      nodeSelector:
        {{- toYaml . | nindent 8 }}
      {{- end }}

      # Tolerations (optional)
      {{- with .Values.tolerations }}
      tolerations:
        {{- toYaml . | nindent 8 }}
      {{- end }}

      # Affinity (pod anti-affinity recommended)
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
          - weight: 100
            podAffinityTerm:
              labelSelector:
                matchExpressions:
                - key: app.kubernetes.io/component
                  operator: In
                  values:
                  - redis
              topologyKey: kubernetes.io/hostname

  # Volume Claim Templates
  volumeClaimTemplates:
  - metadata:
      name: redis-data
      labels:
        {{- include "alerthistory.labels" . | nindent 8 }}
    spec:
      accessModes:
      - ReadWriteOnce
      {{- if .Values.valkey.storage.className }}
      storageClassName: {{ .Values.valkey.storage.className }}
      {{- end }}
      resources:
        requests:
          storage: {{ .Values.valkey.storage.requestedSize | default "5Gi" }}
{{- end }}
```

### 2.2 Pod Lifecycle

**Startup Sequence**:
```
1. StatefulSet Controller creates Pod 0 (alerthistory-redis-0)
2. PVC provisioner creates PersistentVolume (5Gi)
3. Volume mounts to pod at /data
4. Init container (config-init) runs:
   - Copies redis.conf from ConfigMap to /data/redis.conf
   - Injects password from Secret
   - Sets correct permissions (644)
5. Redis container starts:
   - Loads redis.conf from /data/redis.conf
   - Loads AOF file if exists (appendonly.aof replay)
   - Loads RDB snapshot if AOF missing (dump.rdb restore)
   - Binds to port 6379
   - Ready to accept connections
6. redis-exporter sidecar starts:
   - Connects to localhost:6379
   - Starts metrics scraping
   - Exposes metrics on port 9121
7. Probes execute:
   - Startup probe (30 attempts × 5s = 150s max)
   - Liveness probe (every 10s)
   - Readiness probe (every 5s)
8. Pod marked Ready when all probes pass
```

**Graceful Shutdown Sequence**:
```
1. K8s sends SIGTERM to Redis container
2. Redis initiates shutdown:
   - Stops accepting new connections
   - Waits for in-flight commands to complete
   - Flushes AOF buffer to disk (fsync)
   - Creates final RDB snapshot
   - Closes all connections
   - Exits (exit code 0)
3. K8s waits up to terminationGracePeriodSeconds (30s)
4. If not exited, K8s sends SIGKILL (force kill)
5. redis-exporter sidecar receives SIGTERM
6. Sidecar shuts down gracefully
7. Pod marked Terminated
```

**Update Strategy** (RollingUpdate):
```
1. New StatefulSet revision created (spec change)
2. StatefulSet controller updates pods in reverse order:
   - For 1 replica: Only Pod 0
   - For 3 replicas: Pod 2 → Pod 1 → Pod 0
3. For each pod:
   - Mark pod as unready (remove from Service endpoints)
   - Send SIGTERM (graceful shutdown)
   - Wait for termination (max 30s)
   - Delete old pod
   - Create new pod with new spec
   - Wait for Ready status
   - Proceed to next pod
4. All pods updated successfully
```

### 2.3 Resource Requirements

**Redis Container**:
```yaml
resources:
  limits:
    cpu: 500m       # 0.5 CPU cores
    memory: 512Mi   # 512 MiB
  requests:
    cpu: 250m       # 0.25 CPU cores minimum
    memory: 256Mi   # 256 MiB minimum
```

**redis-exporter Sidecar**:
```yaml
resources:
  limits:
    cpu: 100m       # 0.1 CPU cores
    memory: 128Mi   # 128 MiB
  requests:
    cpu: 50m        # 0.05 CPU cores minimum
    memory: 64Mi    # 64 MiB minimum
```

**Total Pod Resources**:
```
CPU:
  Requests: 250m + 50m = 300m
  Limits: 500m + 100m = 600m

Memory:
  Requests: 256Mi + 64Mi = 320Mi
  Limits: 512Mi + 128Mi = 640Mi

Storage:
  PVC: 5Gi per pod (persistent)
```

**Scaling Considerations**:
```
Single replica (current):
  - CPU requests: 300m
  - Memory requests: 320Mi
  - Storage: 5Gi

Three replicas (future HA):
  - CPU requests: 900m (3 × 300m)
  - Memory requests: 960Mi (3 × 320Mi)
  - Storage: 15Gi (3 × 5Gi)
```

---

## 3. Configuration Management

### 3.1 Redis Configuration File

**File**: `helm/alert-history/templates/redis-config.yaml`

```yaml
{{- if eq .Values.profile "standard" }}
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-config
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-config
data:
  redis.conf: |
    # ===========================================
    # Redis Configuration for Alertmanager++
    # Profile: Standard (HA-Ready)
    # Generated: {{ now | date "2006-01-02" }}
    # ===========================================

    # NETWORK CONFIGURATION
    # ---------------------
    bind 0.0.0.0
    port 6379
    protected-mode yes
    tcp-backlog 511
    timeout 0
    tcp-keepalive 300

    # GENERAL CONFIGURATION
    # ---------------------
    daemonize no
    pidfile /var/run/redis.pid
    loglevel {{ .Values.valkey.settings.loglevel | default "notice" }}
    logfile ""
    databases 1

    # SNAPSHOTTING (RDB)
    # ------------------
    # Save the DB on disk:
    #   save <seconds> <changes>
    save 900 1      # After 900 sec (15 min) if at least 1 key changed
    save 300 10     # After 300 sec (5 min) if at least 10 keys changed
    save 60 10000   # After 60 sec if at least 10000 keys changed

    stop-writes-on-bgsave-error yes
    rdbcompression yes
    rdbchecksum yes
    dbfilename dump.rdb
    dir /data

    # REPLICATION (Future HA)
    # -----------------------
    # slave-serve-stale-data yes
    # slave-read-only yes
    # repl-diskless-sync no
    # repl-diskless-sync-delay 5
    # repl-disable-tcp-nodelay no
    # slave-priority 100

    # SECURITY
    # --------
    # requirepass will be injected by init container from Secret
    # requirepass <password>

    # Rename dangerous commands
    rename-command CONFIG ""
    # Keep FLUSHDB/FLUSHALL for testing (optional: disable in production)
    # rename-command FLUSHDB ""
    # rename-command FLUSHALL ""

    # MEMORY MANAGEMENT
    # -----------------
    maxmemory {{ .Values.valkey.settings.maxmemory | default "384mb" }}
    maxmemory-policy {{ .Values.valkey.settings.maxmemoryPolicy | default "allkeys-lru" }}
    maxmemory-samples 5

    # LAZY FREEING
    # ------------
    lazyfree-lazy-eviction yes
    lazyfree-lazy-expire yes
    lazyfree-lazy-server-del yes
    replica-lazy-flush yes

    # APPEND ONLY FILE (AOF)
    # ----------------------
    appendonly {{ .Values.valkey.settings.appendonly | default "yes" }}
    appendfilename "appendonly.aof"
    appendfsync {{ .Values.valkey.settings.appendfsync | default "everysec" }}
    no-appendfsync-on-rewrite no
    auto-aof-rewrite-percentage 100
    auto-aof-rewrite-min-size 64mb
    aof-load-truncated yes
    aof-use-rdb-preamble yes

    # LUA SCRIPTING
    # -------------
    lua-time-limit 5000

    # SLOW LOG
    # --------
    slowlog-log-slower-than {{ .Values.valkey.settings.slowlogThreshold | default "10000" }}
    slowlog-max-len 128

    # LATENCY MONITOR
    # ---------------
    latency-monitor-threshold 100

    # EVENT NOTIFICATION
    # ------------------
    notify-keyspace-events ""

    # ADVANCED CONFIG
    # ---------------
    hash-max-ziplist-entries 512
    hash-max-ziplist-value 64
    list-max-ziplist-size -2
    list-compress-depth 0
    set-max-intset-entries 512
    zset-max-ziplist-entries 128
    zset-max-ziplist-value 64
    hll-sparse-max-bytes 3000
    stream-node-max-bytes 4096
    stream-node-max-entries 100
    activerehashing yes
    client-output-buffer-limit normal 0 0 0
    client-output-buffer-limit replica 256mb 64mb 60
    client-output-buffer-limit pubsub 32mb 8mb 60
    hz 10
    dynamic-hz yes
    aof-rewrite-incremental-fsync yes
    rdb-save-incremental-fsync yes
{{- end }}
```

### 3.2 Configuration Parameters

| Parameter | Value | Rationale |
|-----------|-------|-----------|
| **maxmemory** | 384mb | 75% of 512Mi limit, leaves headroom |
| **maxmemory-policy** | allkeys-lru | Evict least recently used keys when full |
| **appendonly** | yes | Enable AOF for durability |
| **appendfsync** | everysec | Balance performance (fsync every 1s) |
| **save** | 900/300/60 | RDB snapshots at 15min/5min/1min |
| **tcp-keepalive** | 300 | Detect dead connections after 5 minutes |
| **databases** | 1 | Single database to save memory |
| **slowlog-threshold** | 10000 | Log queries >10ms for optimization |
| **protected-mode** | yes | Reject external connections |
| **bind** | 0.0.0.0 | Listen on all interfaces (within pod) |

### 3.3 values.yaml Integration

```yaml
# Redis/Valkey Configuration (Standard Profile Only)
valkey:
  # Deployment
  enabled: true
  replicas: 1  # Expandable to 3 for Sentinel HA

  # Container Image
  image:
    repository: redis
    tag: "7-alpine"
    pullPolicy: IfNotPresent

  # Resource Limits
  resources:
    limits:
      cpu: 500m
      memory: 512Mi
    requests:
      cpu: 250m
      memory: 256Mi

  # Persistent Storage
  storage:
    className: ""  # Use default StorageClass
    requestedSize: 5Gi

  # Redis Settings
  settings:
    maxmemory: 384mb
    maxmemoryPolicy: allkeys-lru
    appendonly: "yes"
    appendfsync: everysec
    loglevel: notice
    slowlogThreshold: 10000

  # redis-exporter Sidecar
  exporter:
    enabled: true
    image: quay.io/oliver006/redis_exporter
    tag: v1.55.0
    pullPolicy: IfNotPresent
    resources:
      limits:
        cpu: 100m
        memory: 128Mi
      requests:
        cpu: 50m
        memory: 64Mi

  # Security
  password:
    existingSecret: ""  # Use existing secret (optional)
    secretKey: password
```

---

## 4. Service Architecture

### 4.1 Headless Service (StatefulSet DNS)

**File**: `helm/alert-history/templates/redis-service.yaml`

```yaml
{{- if eq .Values.profile "standard" }}
---
# Headless Service for StatefulSet DNS
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-headless
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-headless
spec:
  type: ClusterIP
  clusterIP: None  # Headless service
  selector:
    {{- include "alerthistory.selectorLabels" . | nindent 4 }}
    app.kubernetes.io/component: redis
  ports:
  - name: redis
    port: 6379
    targetPort: 6379
    protocol: TCP
  publishNotReadyAddresses: true  # Include not-ready pods in DNS
{{- end }}
```

**DNS Records Created**:
```
# Service DNS
alerthistory-redis-headless.default.svc.cluster.local

# Pod DNS (individual pods)
alerthistory-redis-0.alerthistory-redis-headless.default.svc.cluster.local
alerthistory-redis-1.alerthistory-redis-headless.default.svc.cluster.local  # Future HA
alerthistory-redis-2.alerthistory-redis-headless.default.svc.cluster.local  # Future HA
```

**Use Cases**:
- StatefulSet pod discovery
- Future Sentinel mode (requires direct pod access)
- Direct connection to specific replica

### 4.2 ClusterIP Service (Application Connections)

```yaml
{{- if eq .Values.profile "standard" }}
---
# ClusterIP Service for Application Connections
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis
spec:
  type: ClusterIP
  selector:
    {{- include "alerthistory.selectorLabels" . | nindent 4 }}
    app.kubernetes.io/component: redis
    statefulset.kubernetes.io/pod-name: {{ include "alerthistory.fullname" . }}-redis-0  # Primary only
  ports:
  - name: redis
    port: 6379
    targetPort: 6379
    protocol: TCP
  sessionAffinity: None  # No session affinity (stateless operations)
{{- end }}
```

**Service Configuration**:
```
Service Name: alerthistory-redis
Type: ClusterIP
Port: 6379
Selector: alerthistory-redis-0 (primary pod only)
DNS: alerthistory-redis.default.svc.cluster.local
```

**Use Cases**:
- Application connection string: `redis://alerthistory-redis:6379`
- Load balanced to primary replica
- Future: Can selector match all ready replicas for read distribution

### 4.3 Metrics Service (Prometheus Scraping)

```yaml
{{- if eq .Values.profile "standard" }}
---
# Metrics Service for Prometheus Scraping
apiVersion: v1
kind: Service
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-metrics
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-metrics
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9121"
    prometheus.io/path: "/metrics"
spec:
  type: ClusterIP
  selector:
    {{- include "alerthistory.selectorLabels" . | nindent 4 }}
    app.kubernetes.io/component: redis
  ports:
  - name: metrics
    port: 9121
    targetPort: 9121
    protocol: TCP
{{- end }}
```

**Metrics Endpoint**:
```
Endpoint: http://alerthistory-redis-metrics:9121/metrics
Format: Prometheus text format
Metrics: 50+ metrics from redis-exporter
Scrape Interval: 30s (configurable in ServiceMonitor)
```

---

## 5. Persistent Storage Design

### 5.1 Volume Claim Template

```yaml
volumeClaimTemplates:
- metadata:
    name: redis-data
    labels:
      {{- include "alerthistory.labels" . | nindent 6 }}
      app.kubernetes.io/component: redis-storage
  spec:
    accessModes:
    - ReadWriteOnce
    {{- if .Values.valkey.storage.className }}
    storageClassName: {{ .Values.valkey.storage.className }}
    {{- end }}
    resources:
      requests:
        storage: {{ .Values.valkey.storage.requestedSize | default "5Gi" }}
```

### 5.2 Storage Layout

**Directory Structure** (inside `/data`):
```
/data/
├── redis.conf               # Redis configuration (from init container)
├── appendonly.aof           # AOF log file (append-only)
├── dump.rdb                 # RDB snapshot file
├── appendonly.aof.manifest  # AOF manifest (Redis 7+)
└── temp-*.aof               # Temporary AOF files during rewrite
```

### 5.3 Persistence Mechanisms

**AOF (Append-Only File)**:
```
Purpose: Write-ahead log for all write operations
Fsync Policy: everysec (fsync every 1 second)
Data Loss: Max 1 second of writes on crash
File Size: Grows continuously until rewrite
Rewrite Trigger: 2x file size (auto-aof-rewrite-percentage 100)
Rewrite Min Size: 64MB (auto-aof-rewrite-min-size)
Recovery: Replay all commands from AOF (deterministic)
Performance: <5% CPU overhead for fsync
```

**RDB (Redis Database Snapshot)**:
```
Purpose: Point-in-time snapshot of entire dataset
Triggers:
  - 900s (15 min) with 1+ key changes
  - 300s (5 min) with 10+ key changes
  - 60s (1 min) with 10,000+ key changes
  - Manual: redis-cli SAVE/BGSAVE
Data Loss: Up to 15 minutes (worst case)
File Size: Compressed binary format (~50% of memory size)
Recovery: Fast load (faster than AOF replay)
Performance: Fork process for background save (BGSAVE)
```

**Hybrid Strategy** (AOF + RDB):
```
Primary: AOF (durability, RPO <1s)
Backup: RDB (fast recovery, smaller file)
Recovery Process:
  1. If AOF exists: Replay AOF (primary)
  2. Else: Load RDB snapshot (fallback)
  3. Else: Start empty (complete data loss)
```

### 5.4 Storage Sizing Calculation

```
Data Requirements:
  Classification Cache: 200MB (100K alerts × 2KB)
  Timer Persistence: 0.5MB (1K groups × 500B)
  Inhibition State: 10MB (10K inhibitions × 1KB)
  Overhead (20%): 42MB
  Total Data: 252.5MB

Persistence Files:
  AOF File: ~300MB (1.2x data size)
  RDB Snapshot: ~150MB (0.6x data size)
  Temp Files: ~100MB (rewrite operations)
  Total Files: 550MB

Safety Margin: 10x (5GB)
Provisioned: 5Gi (5,120MB)
Utilization: 550MB / 5,120MB = 10.7%
Headroom: 4,570MB (8.3x current usage) ✅
```

---

## 6. Monitoring & Observability

### 6.1 redis-exporter Sidecar

**Purpose**: Collect and expose Redis metrics in Prometheus format

**Configuration**:
```yaml
containers:
- name: redis-exporter
  image: quay.io/oliver006/redis_exporter:v1.55.0
  env:
  - name: REDIS_ADDR
    value: "localhost:6379"
  - name: REDIS_PASSWORD
    valueFrom:
      secretKeyRef:
        name: alerthistory-redis-secret
        key: password
  ports:
  - name: metrics
    containerPort: 9121
```

**Metrics Exported** (50+ total):

**Connection Metrics** (10):
- `redis_connected_clients` - Number of client connections
- `redis_connected_slaves` - Number of connected replicas
- `redis_rejected_connections_total` - Rejected connections (maxclients)
- `redis_blocked_clients` - Clients blocked on BLPOP/BRPOP
- `redis_client_recent_max_input_buffer_bytes` - Largest input buffer
- `redis_client_recent_max_output_buffer_bytes` - Largest output buffer
- `redis_connected_clients_max_age_seconds` - Oldest connection age
- `redis_tracking_clients` - Number of tracking clients
- `redis_commands_total` - Total commands processed
- `redis_commands_duration_seconds_total` - Command execution time

**Memory Metrics** (12):
- `redis_memory_used_bytes` - Total memory used
- `redis_memory_max_bytes` - maxmemory setting
- `redis_memory_used_rss_bytes` - RSS memory (resident set size)
- `redis_memory_used_peak_bytes` - Peak memory usage
- `redis_memory_used_overhead_bytes` - Memory overhead (non-data)
- `redis_memory_used_dataset_bytes` - Actual data memory
- `redis_memory_used_lua_bytes` - Lua scripts memory
- `redis_memory_fragmentation_ratio` - Fragmentation ratio
- `redis_memory_fragmentation_bytes` - Fragmentation overhead
- `redis_allocator_allocated_bytes` - Allocator stats
- `redis_allocator_active_bytes` - Active memory
- `redis_allocator_resident_bytes` - Resident memory

**Persistence Metrics** (8):
- `redis_rdb_last_save_timestamp_seconds` - Last RDB save time
- `redis_rdb_changes_since_last_save` - Changes since last RDB
- `redis_rdb_last_bgsave_duration_sec` - Last BGSAVE duration
- `redis_rdb_last_bgsave_status` - Last BGSAVE status (1=success)
- `redis_aof_enabled` - AOF enabled (1=yes)
- `redis_aof_rewrite_in_progress` - AOF rewrite in progress
- `redis_aof_last_rewrite_duration_sec` - Last AOF rewrite duration
- `redis_aof_current_size_bytes` - Current AOF file size

**Performance Metrics** (10):
- `redis_commands_processed_total` - Total commands processed
- `redis_keyspace_hits_total` - Cache hits
- `redis_keyspace_misses_total` - Cache misses
- `redis_keyspace_hit_ratio` - Hit rate (calculated)
- `redis_latest_fork_usec` - Last fork duration (microseconds)
- `redis_slowlog_length` - Slow query log length
- `redis_slowlog_last_id` - Last slow query ID
- `redis_instantaneous_ops_per_sec` - Current ops/sec
- `redis_instantaneous_input_kbps` - Current input bandwidth
- `redis_instantaneous_output_kbps` - Current output bandwidth

**Keyspace Metrics** (10):
- `redis_db_keys` - Number of keys (by database)
- `redis_db_keys_expiring` - Keys with TTL
- `redis_db_avg_ttl_seconds` - Average TTL
- `redis_expired_keys_total` - Expired keys (evicted by TTL)
- `redis_evicted_keys_total` - Evicted keys (by maxmemory policy)
- `redis_keyspace_read_hits_total` - Read hits
- `redis_keyspace_read_misses_total` - Read misses
- `redis_keyspace_write_hits_total` - Write hits
- `redis_keyspace_write_misses_total` - Write misses
- `redis_keys_total` - Total keys across all databases

### 6.2 ServiceMonitor CRD

**File**: `helm/alert-history/templates/redis-servicemonitor.yaml`

```yaml
{{- if and (eq .Values.profile "standard") .Values.monitoring.prometheusEnabled }}
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-monitoring
spec:
  selector:
    matchLabels:
      {{- include "alerthistory.selectorLabels" . | nindent 6 }}
      app.kubernetes.io/component: redis-metrics
  endpoints:
  - port: metrics
    interval: 30s
    scrapeTimeout: 10s
    path: /metrics
    scheme: http
    relabelings:
    - sourceLabels: [__meta_kubernetes_pod_name]
      targetLabel: pod
    - sourceLabels: [__meta_kubernetes_pod_node_name]
      targetLabel: node
{{- end }}
```

### 6.3 Prometheus Alerting Rules

**File**: `helm/alert-history/templates/redis-prometheusrule.yaml`

```yaml
{{- if and (eq .Values.profile "standard") .Values.monitoring.prometheusEnabled }}
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-alerts
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-alerts
spec:
  groups:
  - name: redis.critical
    interval: 30s
    rules:
    # CRITICAL ALERT 1: Redis Down
    - alert: RedisDown
      expr: redis_up{job="alerthistory-redis"} == 0
      for: 1m
      labels:
        severity: critical
        component: redis
      annotations:
        summary: "Redis is down"
        description: "Redis instance {{ $labels.instance }} is not responding to PING"
        runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-down"

    # CRITICAL ALERT 2: Redis Out of Memory
    - alert: RedisOutOfMemory
      expr: |
        redis_memory_used_bytes{job="alerthistory-redis"}
        / redis_memory_max_bytes{job="alerthistory-redis"} > 0.9
      for: 5m
      labels:
        severity: critical
        component: redis
      annotations:
        summary: "Redis memory usage critical (>90%)"
        description: "Redis instance {{ $labels.instance }} is using {{ $value | humanizePercentage }} of maxmemory"
        runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-oom"

    # CRITICAL ALERT 3: Too Many Connections
    - alert: RedisTooManyConnections
      expr: redis_connected_clients{job="alerthistory-redis"} > 8000
      for: 5m
      labels:
        severity: critical
        component: redis
      annotations:
        summary: "Redis has too many connections (>8000)"
        description: "Redis instance {{ $labels.instance }} has {{ $value }} connections (maxclients: 10000)"
        runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-connections"

    # CRITICAL ALERT 4: Rejected Connections
    - alert: RedisRejectedConnections
      expr: |
        increase(redis_rejected_connections_total{job="alerthistory-redis"}[5m]) > 0
      labels:
        severity: critical
        component: redis
      annotations:
        summary: "Redis rejected connections detected"
        description: "Redis instance {{ $labels.instance }} rejected {{ $value }} connections in last 5min"
        runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-rejected"

    # CRITICAL ALERT 5: Persistence Failure
    - alert: RedisPersistenceFailure
      expr: redis_rdb_last_bgsave_status{job="alerthistory-redis"} == 0
      for: 10m
      labels:
        severity: critical
        component: redis
      annotations:
        summary: "Redis RDB persistence failed"
        description: "Redis instance {{ $labels.instance }} RDB save failed"
        runbook_url: "https://docs.alertmanager-plus-plus.io/runbooks/redis-persistence"

  - name: redis.warning
    interval: 30s
    rules:
    # WARNING ALERT 6: High Memory Usage
    - alert: RedisHighMemoryUsage
      expr: |
        redis_memory_used_bytes{job="alerthistory-redis"}
        / redis_memory_max_bytes{job="alerthistory-redis"} > 0.75
      for: 10m
      labels:
        severity: warning
        component: redis
      annotations:
        summary: "Redis memory usage high (>75%)"
        description: "Redis instance {{ $labels.instance }} is using {{ $value | humanizePercentage }} of maxmemory"

    # WARNING ALERT 7: High Connection Usage
    - alert: RedisHighConnectionUsage
      expr: redis_connected_clients{job="alerthistory-redis"} > 6000
      for: 10m
      labels:
        severity: warning
        component: redis
      annotations:
        summary: "Redis connection usage high (>60%)"
        description: "Redis instance {{ $labels.instance }} has {{ $value }} connections"

    # WARNING ALERT 8: Slow Queries
    - alert: RedisSlowQueries
      expr: redis_slowlog_length{job="alerthistory-redis"} > 10
      for: 5m
      labels:
        severity: warning
        component: redis
      annotations:
        summary: "Redis slow queries detected"
        description: "Redis instance {{ $labels.instance }} has {{ $value }} slow queries in log"

    # WARNING ALERT 9: Replication Lag (Future HA)
    - alert: RedisReplicationLag
      expr: redis_slave_repl_offset{job="alerthistory-redis"} - redis_master_repl_offset{job="alerthistory-redis"} > 1000
      for: 5m
      labels:
        severity: warning
        component: redis
      annotations:
        summary: "Redis replication lag high"
        description: "Redis replica {{ $labels.instance }} is lagging by {{ $value }} bytes"

    # WARNING ALERT 10: Low Cache Hit Rate
    - alert: RedisLowHitRate
      expr: |
        rate(redis_keyspace_hits_total{job="alerthistory-redis"}[5m])
        / (rate(redis_keyspace_hits_total{job="alerthistory-redis"}[5m])
           + rate(redis_keyspace_misses_total{job="alerthistory-redis"}[5m])) < 0.8
      for: 15m
      labels:
        severity: warning
        component: redis
      annotations:
        summary: "Redis cache hit rate low (<80%)"
        description: "Redis instance {{ $labels.instance }} has {{ $value | humanizePercentage }} hit rate"
{{- end }}
```

### 6.4 Grafana Dashboard

**Dashboard ID**: 11835 (Redis Dashboard for Prometheus Redis Exporter)
**Import URL**: https://grafana.com/grafana/dashboards/11835

**Panels** (12 total):
1. **Redis Uptime** - Time series (uptime_in_seconds)
2. **Connected Clients** - Gauge (redis_connected_clients)
3. **Memory Usage** - Bar gauge (redis_memory_used_bytes / redis_memory_max_bytes)
4. **Commands Per Second** - Time series (rate(redis_commands_total[1m]))
5. **Hit Rate** - Time series (cache hit rate calculation)
6. **Network I/O** - Time series (input/output kbps)
7. **Keyspace** - Time series (redis_db_keys)
8. **Evicted/Expired Keys** - Time series (redis_evicted_keys_total, redis_expired_keys_total)
9. **Persistence** - Stat (last RDB save time, AOF status)
10. **Slow Queries** - Table (slowlog entries)
11. **Memory Fragmentation** - Time series (redis_memory_fragmentation_ratio)
12. **Connection Age** - Histogram (connection age distribution)

---

## 7. Security Architecture

### 7.1 Network Security

**NetworkPolicy**:

**File**: `helm/alert-history/templates/redis-networkpolicy.yaml`

```yaml
{{- if and (eq .Values.profile "standard") .Values.valkey.networkPolicy.enabled }}
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-network-policy
spec:
  podSelector:
    matchLabels:
      {{- include "alerthistory.selectorLabels" . | nindent 6 }}
      app.kubernetes.io/component: redis
  policyTypes:
  - Ingress
  - Egress

  ingress:
  # Allow traffic from application pods
  - from:
    - podSelector:
        matchLabels:
          {{- include "alerthistory.selectorLabels" . | nindent 10 }}
          app.kubernetes.io/component: application
    ports:
    - protocol: TCP
      port: 6379

  # Allow Prometheus scraping
  - from:
    - namespaceSelector:
        matchLabels:
          name: monitoring
      podSelector:
        matchLabels:
          app.kubernetes.io/name: prometheus
    ports:
    - protocol: TCP
      port: 9121

  egress:
  # Allow DNS resolution
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

  # Allow egress to API server (for future Sentinel)
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 6443
{{- end }}
```

### 7.2 Secret Management

**Manual Secret** (default):

**File**: `helm/alert-history/templates/redis-secret.yaml`

```yaml
{{- if eq .Values.profile "standard" }}
{{- if not .Values.valkey.password.existingSecret }}
apiVersion: v1
kind: Secret
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-secret
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
    app.kubernetes.io/component: redis-secret
type: Opaque
data:
  password: {{ .Values.valkey.password.value | default (randAlphaNum 32) | b64enc | quote }}
{{- end }}
{{- end }}
```

**External Secrets Operator** (future - TN-100):

```yaml
{{- if and (eq .Values.profile "standard") .Values.valkey.password.existingSecret .Values.externalSecrets.enabled }}
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: {{ include "alerthistory.fullname" . }}-redis-secret
  namespace: {{ .Release.Namespace }}
  labels:
    {{- include "alerthistory.labels" . | nindent 4 }}
spec:
  secretStoreRef:
    name: {{ .Values.externalSecrets.secretStore }}
    kind: {{ .Values.externalSecrets.secretStoreKind | default "SecretStore" }}
  refreshInterval: {{ .Values.externalSecrets.refreshInterval | default "1h" }}
  target:
    name: {{ include "alerthistory.fullname" . }}-redis-secret
    creationPolicy: Owner
  data:
  - secretKey: password
    remoteRef:
      key: {{ .Values.externalSecrets.keyPath }}/redis-password
{{- end }}
```

### 7.3 RBAC

**ServiceAccount** (reuse existing):

```yaml
# Defined in main ServiceAccount manifest
# helm/alert-history/templates/serviceaccount.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: {{ include "alerthistory.serviceAccountName" . }}
  namespace: {{ .Release.Namespace }}
```

**No Additional RBAC Required** for Redis pod (StatefulSet doesn't need K8s API access)

---

## 8. Deployment Workflow

### 8.1 Initial Deployment

**Helm Install**:
```bash
# Standard Profile (Redis enabled)
helm install alerthistory ./helm/alert-history \
  --set profile=standard \
  --set valkey.enabled=true \
  --set valkey.storage.requestedSize=5Gi \
  --set valkey.password.value=<secure-password>
```

**Deployment Sequence**:
```
1. Helm template rendering
   - Profile check: standard ✅
   - Redis templates included

2. Create ConfigMap (redis-config)
   - redis.conf generated

3. Create Secret (redis-secret)
   - Password generated or provided

4. Create Services (3 services)
   - Headless service
   - ClusterIP service
   - Metrics service

5. Create StatefulSet
   - PVC provisioner triggered
   - PVC bound (5Gi)
   - Pod 0 created

6. Init container runs
   - Copy redis.conf
   - Inject password

7. Redis container starts
   - Load configuration
   - AOF replay (if exists)
   - Ready to accept connections

8. redis-exporter starts
   - Connect to Redis
   - Start metrics scraping

9. Probes execute
   - Startup probe passes
   - Liveness probe passes
   - Readiness probe passes

10. Pod marked Ready
    - Added to Service endpoints
    - Application connects successfully
```

### 8.2 Upgrade Workflow

**Helm Upgrade**:
```bash
# Upgrade with configuration changes
helm upgrade alerthistory ./helm/alert-history \
  --set profile=standard \
  --set valkey.settings.maxmemory=512mb \
  --reuse-values
```

**Upgrade Sequence** (RollingUpdate):
```
1. Helm computes diff
   - ConfigMap changed: Yes (maxmemory)
   - StatefulSet spec changed: Yes (ConfigMap checksum)

2. Update ConfigMap
   - New redis.conf with maxmemory=512mb
   - ConfigMap checksum changes

3. StatefulSet rolling update
   - Pod 0 marked for update (checksum mismatch)
   - Remove from Service endpoints (not Ready)
   - Send SIGTERM to containers
   - Wait for graceful shutdown (max 30s)
   - Delete pod
   - Create new pod with new spec
   - Wait for Ready status
   - Add back to Service endpoints

4. Verify upgrade
   - Check maxmemory: redis-cli CONFIG GET maxmemory
   - Expected: 536870912 (512MB in bytes)
```

### 8.3 Rollback Strategy

**Helm Rollback**:
```bash
# Rollback to previous release
helm rollback alerthistory

# Rollback to specific revision
helm rollback alerthistory 2
```

**Rollback Sequence**:
```
1. Helm retrieves previous release manifest
2. Apply previous StatefulSet spec
3. Pod recreated with old spec
4. AOF replay ensures data consistency
5. Service restored
```

**Data Safety**:
- PVC retained during rollback (data preserved)
- AOF ensures no data loss
- Rollback time: <2 minutes

---

## 9. Integration Points

### 9.1 Application Integration

**Go Application** (already implemented):

```go
// go-app/cmd/server/main.go:359-409
if cfg.Profile == appconfig.ProfileStandard && cfg.Redis.Addr != "" {
    cacheConfig := cache.CacheConfig{
        Addr:                  cfg.Redis.Addr,  // "alerthistory-redis:6379"
        Password:              cfg.Redis.Password,
        DB:                    cfg.Redis.DB,
        PoolSize:              cfg.Redis.PoolSize,  // 50
        MinIdleConns:          cfg.Redis.MinIdleConns,  // 1
        DialTimeout:           cfg.Redis.DialTimeout,  // 5s
        ReadTimeout:           cfg.Redis.ReadTimeout,  // 3s
        WriteTimeout:          cfg.Redis.WriteTimeout,  // 3s
        MaxRetries:            cfg.Redis.MaxRetries,  // 3
        MinRetryBackoff:       cfg.Redis.MinRetryBackoff,  // 8ms
        MaxRetryBackoff:       cfg.Redis.MaxRetryBackoff,  // 512ms
        CircuitBreakerEnabled: true,
        MetricsEnabled:        cfg.Metrics.Enabled,
    }

    redisCache, err = cache.NewRedisCache(&cacheConfig, appLogger)
    if err != nil {
        slog.Warn("Failed to initialize Redis, fallback to memory-only", "error", err)
        redisCache = nil  // Graceful fallback
    }
}
```

**Environment Variables** (ConfigMap):
```yaml
env:
- name: REDIS_ADDR
  value: "alerthistory-redis:6379"
- name: REDIS_PASSWORD
  valueFrom:
    secretKeyRef:
      name: alerthistory-redis-secret
      key: password
- name: REDIS_DB
  value: "0"
- name: REDIS_POOL_SIZE
  value: "50"
```

### 9.2 Helm Chart Integration

**values.yaml** (profile conditional):
```yaml
profile: "standard"  # Triggers Redis deployment

valkey:
  enabled: true  # Managed by profile
  # ... configuration ...
```

**Template Conditional**:
```yaml
{{- if eq .Values.profile "standard" }}
# Redis templates included
{{- end }}
```

### 9.3 Monitoring Integration

**Prometheus** (ServiceMonitor auto-discovery):
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
# Prometheus Operator discovers and scrapes automatically
```

**Grafana** (dashboard import):
```bash
# Import Redis dashboard
curl -X POST http://grafana:3000/api/dashboards/import \
  -H "Content-Type: application/json" \
  -d '{
    "dashboard": {
      "id": 11835
    }
  }'
```

---

## 10. Testing Strategy

### 10.1 Helm Template Tests

```bash
#!/bin/bash
# helm-template-test.sh

# Test 1: Template renders for Standard Profile
helm template alerthistory ./helm/alert-history \
  --set profile=standard | grep -A 10 "kind: StatefulSet"

# Test 2: No Redis for Lite Profile
helm template alerthistory ./helm/alert-history \
  --set profile=lite | grep -c "kind: StatefulSet"
# Expected: 0 (PostgreSQL only)

# Test 3: ConfigMap rendered correctly
helm template alerthistory ./helm/alert-history \
  --set profile=standard | grep -A 5 "maxmemory"

# Test 4: Services created
helm template alerthistory ./helm/alert-history \
  --set profile=standard | grep -c "kind: Service"
# Expected: 6 (3 app + 3 Redis)
```

### 10.2 Load Testing (K6)

```javascript
// k6-connection-pool.js

import redis from 'k6/x/redis';
import { check } from 'k6';

export let options = {
  stages: [
    { duration: '1m', target: 500 },  // Ramp up to 500 connections
    { duration: '5m', target: 500 },  // Hold 500 connections
    { duration: '1m', target: 0 },    // Ramp down
  ],
};

const client = redis.newClient('redis://alerthistory-redis:6379');

export default function () {
  // Test SET operation
  let setResult = client.set('test-key', 'test-value');
  check(setResult, { 'SET successful': (r) => r === 'OK' });

  // Test GET operation
  let getValue = client.get('test-key');
  check(getValue, { 'GET successful': (v) => v === 'test-value' });
}
```

### 10.3 Failover Test

```bash
#!/bin/bash
# failover-test.sh

echo "1. Writing test data to Redis..."
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD SET test-key test-value

echo "2. Verify data written..."
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD GET test-key

echo "3. Delete pod (simulate crash)..."
kubectl delete pod alerthistory-redis-0

echo "4. Wait for pod recreation..."
kubectl wait --for=condition=ready pod/alerthistory-redis-0 --timeout=60s

echo "5. Verify data persisted..."
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD GET test-key
# Expected: test-value (data preserved via AOF)

echo "6. Verify AOF replay..."
kubectl logs alerthistory-redis-0 | grep "Loading DB"
```

### 10.4 Persistence Test

```bash
#!/bin/bash
# persistence-test.sh

echo "1. Check AOF enabled..."
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD CONFIG GET appendonly
# Expected: appendonly yes

echo "2. Write test data..."
for i in {1..1000}; do
  kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD SET test-$i value-$i
done

echo "3. Verify AOF file created..."
kubectl exec alerthistory-redis-0 -- ls -lh /data/appendonly.aof

echo "4. Force RDB snapshot..."
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD SAVE

echo "5. Verify RDB file created..."
kubectl exec alerthistory-redis-0 -- ls -lh /data/dump.rdb

echo "6. Verify both files exist..."
kubectl exec alerthistory-redis-0 -- ls -lh /data/
# Expected: appendonly.aof + dump.rdb
```

---

## 11. Operational Procedures

### 11.1 Backup Procedures

**Manual Backup** (RDB snapshot):
```bash
# Trigger RDB snapshot
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD SAVE

# Copy RDB file to local machine
kubectl cp alerthistory-redis-0:/data/dump.rdb ./backup-$(date +%Y%m%d).rdb

# Verify backup
ls -lh backup-*.rdb
```

**Manual Backup** (AOF file):
```bash
# Copy AOF file to local machine
kubectl cp alerthistory-redis-0:/data/appendonly.aof ./backup-aof-$(date +%Y%m%d).aof

# Verify backup
ls -lh backup-aof-*.aof
```

**Automated Backup** (CronJob - future):
```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: redis-backup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: redis:7-alpine
            command:
            - sh
            - -c
            - |
              redis-cli -h alerthistory-redis -a $REDIS_PASSWORD SAVE
              kubectl cp alerthistory-redis-0:/data/dump.rdb /backup/dump-$(date +%Y%m%d).rdb
```

### 11.2 Restore Procedures

**Restore from RDB**:
```bash
# 1. Scale down application (stop writes)
kubectl scale deployment alerthistory --replicas=0

# 2. Delete Redis pod
kubectl delete pod alerthistory-redis-0

# 3. Wait for pod deletion
kubectl wait --for=delete pod/alerthistory-redis-0 --timeout=60s

# 4. Copy backup RDB to PVC
kubectl cp backup-20251130.rdb alerthistory-redis-0:/data/dump.rdb

# 5. Pod will auto-restart and load RDB
kubectl wait --for=condition=ready pod/alerthistory-redis-0 --timeout=60s

# 6. Verify data restored
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD DBSIZE

# 7. Scale up application
kubectl scale deployment alerthistory --replicas=2
```

### 11.3 Maintenance Procedures

**Memory Optimization**:
```bash
# Check memory fragmentation
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD INFO memory | grep fragmentation

# If fragmentation >1.5, restart pod (triggers defragmentation on load)
kubectl delete pod alerthistory-redis-0
```

**AOF Compaction** (manual):
```bash
# Trigger AOF rewrite
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD BGREWRITEAOF

# Monitor rewrite progress
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD INFO persistence | grep aof_rewrite
```

**Slow Query Analysis**:
```bash
# View slow queries
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD SLOWLOG GET 10

# Reset slow query log
kubectl exec alerthistory-redis-0 -- redis-cli -a $REDIS_PASSWORD SLOWLOG RESET
```

---

## 12. Design Decisions

### 12.1 Redis vs Valkey

**Decision**: Support both Redis and Valkey (Redis-compatible fork)

**Rationale**:
- Valkey is OSS fork of Redis with full API compatibility
- Configuration and behavior identical
- Allows users to choose based on licensing preferences
- Implementation: Same StatefulSet, different image

**Configuration**:
```yaml
valkey:
  image:
    repository: redis  # or valkey/valkey
    tag: "7-alpine"
```

### 12.2 Single Primary vs Sentinel HA

**Decision**: Start with single primary, design for future Sentinel HA

**Rationale**:
- Standard Profile (2-10 app replicas) doesn't require Redis HA initially
- Single primary sufficient for cache workload (not critical path)
- Sentinel HA adds complexity (3 Redis + 3 Sentinel pods = 6 pods total)
- Future upgrade path: 1 replica → 3 replicas + Sentinel quorum

**Trade-offs**:
- Single primary: 99.5% uptime (4h downtime/year acceptable for cache)
- Sentinel HA: 99.95% uptime (26min downtime/year) + 6x resource cost

### 12.3 AOF everysec vs always

**Decision**: `appendfsync everysec` (fsync every 1 second)

**Rationale**:
- **Performance**: <5% CPU overhead (vs 20-30% for `always`)
- **Durability**: <1s data loss acceptable for cache (not transactional data)
- **Recovery**: Fast AOF replay (<30s for 200MB)

**Alternative Considered**:
- `appendfsync always`: Fsync on every write (slow, 20-30% CPU overhead)
- `appendfsync no`: No fsync (fast, but max 30s data loss)

### 12.4 Connection Pool Sizing

**Decision**: 50 connections per app pod (default go-redis)

**Rationale**:
- Total connections: 10 pods × 50 = 500 connections
- Redis maxclients: 10,000 (default)
- Utilization: 5% (19x headroom)
- No need to tune (default sufficient)

**Monitoring**:
- Alert on >8,000 connections (80% utilization)
- Alert on rejected connections (maxclients reached)

### 12.5 maxmemory 384MB (75% of limit)

**Decision**: `maxmemory 384mb` (75% of 512Mi limit)

**Rationale**:
- Leave headroom for Redis overhead (metadata, buffers, etc.)
- Prevent OOM kills by K8s (memory limit 512Mi)
- LRU eviction handles memory pressure gracefully
- 52% buffer (131.5MB) above data requirement (252.5MB)

**Alternative Considered**:
- maxmemory=512mb: Risk of OOM (no overhead headroom)
- maxmemory=256mb: Waste of capacity (only 50% utilization)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-30
**Author**: Vitalii Semenov (AI-assisted)
**Status**: ✅ DESIGN COMPLETE - READY FOR IMPLEMENTATION
**Next**: tasks.md (Implementation Checklist)
