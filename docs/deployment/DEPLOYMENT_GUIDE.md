# Alertmanager++ OSS Deployment Guide

**Version**: 2.0.0
**Last Updated**: 2025-11-30
**Deployment Profiles**: Lite (Single-node) | Standard (HA-ready)

---

## 📋 Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start - Lite Profile](#quick-start-lite-profile)
3. [Production Deployment - Standard Profile](#production-deployment-standard-profile)
4. [Configuration Reference](#configuration-reference)
5. [Deployment Profiles](#deployment-profiles)
6. [Verification & Health Checks](#verification--health-checks)
7. [Troubleshooting](#troubleshooting)
8. [Upgrade & Rollback](#upgrade--rollback)

---

## Prerequisites

### Required

- **Kubernetes Cluster**: v1.25+ (tested on v1.28)
- **Helm**: v3.12+ ([install instructions](https://helm.sh/docs/intro/install/))
- **kubectl**: Compatible with your cluster version
- **Storage Class**: For persistent volumes (Lite profile only)

### Optional (Standard Profile)

- **PostgreSQL**: v15+ (external or StatefulSet)
- **Redis/Valkey**: v7+ (for caching)
- **Prometheus**: For metrics scraping
- **Grafana**: For dashboards

### Resource Requirements

| Profile | CPU | Memory | Storage | Replicas |
|---------|-----|--------|---------|----------|
| **Lite** | 250m | 512Mi | 10Gi PVC | 1 (single-node) |
| **Standard** | 500m | 1Gi | External DB | 2-10 (auto-scale) |

---

## Quick Start - Lite Profile

**Use Case**: Development, testing, small-scale production (<1K alerts/day)

### Step 1: Add Helm Repository

```bash
# Add Alertmanager++ Helm repository
helm repo add alertmanager-plus https://ipiton.github.io/alert-history-service
helm repo update
```

### Step 2: Install Lite Profile

```bash
# Create namespace
kubectl create namespace alertmanager-plus

# Install with Lite profile (default)
helm install alert-history alertmanager-plus/alert-history \
  --namespace alertmanager-plus \
  --set profile=lite \
  --set storage.backend=filesystem \
  --set storage.filesystemPath=/data/alerthistory.db \
  --set cache.enabled=false \
  --set persistence.enabled=true \
  --set persistence.size=10Gi
```

### Step 3: Verify Deployment

```bash
# Check pod status
kubectl get pods -n alertmanager-plus

# Check service
kubectl get svc -n alertmanager-plus

# Port-forward for local access
kubectl port-forward -n alertmanager-plus svc/alert-history 8080:80

# Test health endpoint
curl http://localhost:8080/health
```

**Expected Output**:
```json
{
  "status": "healthy",
  "components": {
    "database": {"status": "healthy", "message": "SQLite connection OK"},
    "cache": {"status": "healthy", "message": "Memory cache operational"}
  }
}
```

---

## Production Deployment - Standard Profile

**Use Case**: Production environments, high-volume (>1K alerts/day), HA requirements

### Step 1: Prerequisites Setup

#### 1.1 Deploy PostgreSQL (if not external)

```bash
# Option A: Use Helm subchart (included)
helm install alert-history alertmanager-plus/alert-history \
  --set profile=standard \
  --set postgresql.enabled=true \
  --set postgresql.replicaCount=3 \
  --set postgresql.persistence.size=50Gi

# Option B: Use external PostgreSQL
# Skip postgresql subchart, provide connection details
```

#### 1.2 Deploy Redis/Valkey (optional but recommended)

```bash
# Option A: Use Helm subchart (Valkey)
helm install alert-history alertmanager-plus/alert-history \
  --set profile=standard \
  --set valkey.enabled=true \
  --set valkey.replicaCount=3

# Option B: Use external Redis
# Skip valkey subchart, provide connection details
```

### Step 2: Create Configuration Values

Create `production-values.yaml`:

```yaml
# Production Configuration for Standard Profile
profile: standard

# Application Configuration
replicaCount: 3  # HA setup

image:
  repository: ghcr.io/ipiton/alert-history-service
  tag: "2.0.0"
  pullPolicy: IfNotPresent

# Resource Limits (production-grade)
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: 2000m
    memory: 4Gi

# Autoscaling (HPA)
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
  # Custom metrics (requires Prometheus Adapter)
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80

# Storage Configuration
storage:
  backend: postgres  # Required for Standard profile

# PostgreSQL Configuration
postgresql:
  enabled: true  # Use Helm subchart
  replicaCount: 3  # HA setup
  auth:
    username: alerthistory
    password: "CHANGE_ME"  # Use K8s Secret in production!
    database: alerthistory
  persistence:
    enabled: true
    size: 50Gi
    storageClass: fast-ssd  # Use your storage class
  resources:
    requests:
      cpu: 500m
      memory: 2Gi
    limits:
      cpu: 2000m
      memory: 8Gi
  config:
    maxConnections: 250  # For 10 replicas * 20 conns/pod = 200
    sharedBuffers: "512MB"
    effectiveCacheSize: "2GB"
    maintenanceWorkMem: "128MB"
    walBuffers: "16MB"

# Redis/Valkey Configuration (optional, recommended)
valkey:
  enabled: true
  replicaCount: 3
  persistence:
    enabled: true
    size: 10Gi
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 1Gi
  valkeyConfig:
    maxmemory: "256mb"
    maxmemory-policy: "allkeys-lru"
    appendonly: "yes"
    appendfsync: "everysec"

# Ingress Configuration
ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
    nginx.ingress.kubernetes.io/rate-limit: "100"
  hosts:
    - host: alertmanager-plus.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: alertmanager-plus-tls
      hosts:
        - alertmanager-plus.example.com

# Service Configuration
service:
  type: ClusterIP
  port: 80
  targetPort: 8080

# Security Context
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  capabilities:
    drop:
      - ALL

# Pod Disruption Budget
podDisruptionBudget:
  enabled: true
  minAvailable: 1  # Always keep at least 1 pod running

# Monitoring & Observability
prometheus:
  enabled: true
  serviceMonitor:
    enabled: true
    interval: 30s
    scrapeTimeout: 10s

# LLM Configuration (optional)
llm:
  enabled: true
  baseURL: "https://api.openai.com/v1"
  apiKey: ""  # Set via K8s Secret
  model: "gpt-4"
  timeout: 10s

# Environment Variables
env:
  - name: LOG_LEVEL
    value: "info"
  - name: ENVIRONMENT
    value: "production"
  - name: PROFILE
    value: "standard"
```

### Step 3: Deploy to Production

```bash
# Create namespace
kubectl create namespace alertmanager-plus

# Create secrets (PostgreSQL, Redis, LLM)
kubectl create secret generic postgresql-secret \
  --from-literal=password='YOUR_STRONG_PASSWORD' \
  -n alertmanager-plus

kubectl create secret generic llm-secret \
  --from-literal=api-key='YOUR_LLM_API_KEY' \
  -n alertmanager-plus

# Deploy with production values
helm install alert-history alertmanager-plus/alert-history \
  --namespace alertmanager-plus \
  --values production-values.yaml \
  --timeout 10m \
  --wait

# Verify deployment
kubectl get pods -n alertmanager-plus -w
```

### Step 4: Post-Deployment Checks

```bash
# 1. Check all pods are running
kubectl get pods -n alertmanager-plus

# 2. Check HPA status
kubectl get hpa -n alertmanager-plus

# 3. Check services
kubectl get svc -n alertmanager-plus

# 4. Check ingress
kubectl get ingress -n alertmanager-plus

# 5. Test health endpoint
kubectl port-forward -n alertmanager-plus svc/alert-history 8080:80 &
curl http://localhost:8080/health

# 6. Check metrics
curl http://localhost:8080/metrics

# 7. Check logs
kubectl logs -n alertmanager-plus deployment/alert-history --tail=50
```

---

## Configuration Reference

### Deployment Profiles

#### Lite Profile

**Characteristics**:
- Single-node deployment (no HA)
- Embedded storage (SQLite)
- Memory-only cache
- Zero external dependencies
- PVC for persistence
- Use case: <1K alerts/day

**Configuration**:
```yaml
profile: lite
replicaCount: 1
autoscaling:
  enabled: false
storage:
  backend: filesystem
  filesystemPath: /data/alerthistory.db
persistence:
  enabled: true
  size: 10Gi
cache:
  enabled: false  # Memory-only
postgresql:
  enabled: false
valkey:
  enabled: false
```

#### Standard Profile

**Characteristics**:
- Multi-replica (2-10 pods with HPA)
- PostgreSQL database
- Redis/Valkey cache (optional)
- Horizontal auto-scaling
- HA-ready
- Use case: >1K alerts/day

**Configuration**:
```yaml
profile: standard
replicaCount: 3
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
storage:
  backend: postgres
postgresql:
  enabled: true
  replicaCount: 3
valkey:
  enabled: true
  replicaCount: 3
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PROFILE` | `standard` | Deployment profile (`lite` or `standard`) |
| `STORAGE_BACKEND` | `postgres` | Storage backend (`filesystem` or `postgres`) |
| `STORAGE_FILESYSTEM_PATH` | `/data/alerthistory.db` | SQLite DB path (Lite only) |
| `DATABASE_HOST` | `localhost` | PostgreSQL host (Standard) |
| `DATABASE_PORT` | `5432` | PostgreSQL port |
| `DATABASE_NAME` | `alerthistory` | Database name |
| `DATABASE_USER` | `alerthistory` | Database user |
| `DATABASE_PASSWORD` | `` | Database password (from secret) |
| `REDIS_ADDR` | `` | Redis address (optional) |
| `REDIS_PASSWORD` | `` | Redis password (from secret) |
| `LLM_BASE_URL` | `` | LLM API endpoint (optional) |
| `LLM_API_KEY` | `` | LLM API key (from secret) |
| `LOG_LEVEL` | `info` | Logging level (`debug`, `info`, `warn`, `error`) |
| `ENVIRONMENT` | `production` | Environment name |

---

## Verification & Health Checks

### Health Endpoints

```bash
# Overall health
curl http://localhost:8080/health

# Readiness (K8s probe)
curl http://localhost:8080/ready

# Liveness (K8s probe)
curl http://localhost:8080/live

# Metrics (Prometheus)
curl http://localhost:8080/metrics
```

### Database Connectivity

```bash
# Check PostgreSQL connection
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  sh -c 'psql -h $DATABASE_HOST -U $DATABASE_USER -d $DATABASE_NAME -c "SELECT 1"'

# Check Redis connection
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  sh -c 'redis-cli -h $REDIS_ADDR ping'
```

### Smoke Tests

```bash
# 1. Send test alert (Alertmanager format)
curl -X POST http://localhost:8080/webhook \
  -H "Content-Type: application/json" \
  -d '{
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "TestAlert",
        "severity": "warning"
      },
      "annotations": {
        "summary": "Test alert for deployment verification"
      },
      "startsAt": "2025-11-30T12:00:00Z"
    }]
  }'

# 2. Query alert history
curl http://localhost:8080/api/v2/history?limit=10

# 3. Check enrichment mode
curl http://localhost:8080/api/v2/enrichment/mode

# 4. Verify metrics
curl http://localhost:8080/metrics | grep alert_history_business
```

---

## Troubleshooting

### Common Issues

#### Pod Not Starting

```bash
# Check pod events
kubectl describe pod -n alertmanager-plus <pod-name>

# Check logs
kubectl logs -n alertmanager-plus <pod-name>

# Common causes:
# - Image pull errors: Check image tag and registry access
# - Resource limits: Adjust CPU/memory requests
# - Volume mount issues: Check PVC status
```

#### Database Connection Errors

```bash
# Check PostgreSQL pod status
kubectl get pods -n alertmanager-plus -l app=postgresql

# Check connection from app pod
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  nc -zv $DATABASE_HOST $DATABASE_PORT

# Verify credentials
kubectl get secret postgresql-secret -n alertmanager-plus -o yaml

# Check PostgreSQL logs
kubectl logs -n alertmanager-plus statefulset/postgresql --tail=100
```

#### HPA Not Scaling

```bash
# Check HPA status
kubectl describe hpa -n alertmanager-plus alert-history

# Check metrics-server
kubectl get deployment metrics-server -n kube-system

# Check pod CPU/memory usage
kubectl top pods -n alertmanager-plus

# Common causes:
# - metrics-server not installed
# - Resource requests not set
# - Metrics collection lag
```

#### High Memory Usage

```bash
# Check pod memory
kubectl top pods -n alertmanager-plus

# Check for memory leaks in logs
kubectl logs -n alertmanager-plus deployment/alert-history | grep -i "memory\|oom"

# Solutions:
# - Increase memory limits
# - Enable Redis cache to reduce in-memory caching
# - Review cache TTL settings
```

---

## Upgrade & Rollback

### Upgrade Process

```bash
# 1. Backup database (if using PostgreSQL)
kubectl exec -n alertmanager-plus postgresql-0 -- \
  pg_dump -U alerthistory alerthistory > backup-$(date +%Y%m%d).sql

# 2. Update Helm repository
helm repo update

# 3. Check new version
helm search repo alertmanager-plus/alert-history --versions

# 4. Upgrade (dry-run first)
helm upgrade alert-history alertmanager-plus/alert-history \
  --namespace alertmanager-plus \
  --values production-values.yaml \
  --version 2.1.0 \
  --dry-run

# 5. Perform upgrade
helm upgrade alert-history alertmanager-plus/alert-history \
  --namespace alertmanager-plus \
  --values production-values.yaml \
  --version 2.1.0 \
  --timeout 10m \
  --wait

# 6. Verify upgrade
kubectl rollout status deployment/alert-history -n alertmanager-plus
```

### Rollback

```bash
# 1. Check release history
helm history alert-history -n alertmanager-plus

# 2. Rollback to previous version
helm rollback alert-history <revision> -n alertmanager-plus

# 3. Verify rollback
kubectl get pods -n alertmanager-plus
kubectl logs -n alertmanager-plus deployment/alert-history --tail=50
```

---

## Production Checklist

### Pre-Deployment

- [ ] Review and customize `production-values.yaml`
- [ ] Set strong passwords for PostgreSQL
- [ ] Configure LLM API keys (if using classification)
- [ ] Set up external PostgreSQL (if not using subchart)
- [ ] Set up external Redis (if not using subchart)
- [ ] Configure Ingress with TLS
- [ ] Set up monitoring (Prometheus, Grafana)
- [ ] Define resource limits and requests
- [ ] Configure PodDisruptionBudget
- [ ] Set up backup strategy for PostgreSQL

### Post-Deployment

- [ ] Verify all pods are running
- [ ] Test health endpoints
- [ ] Send test alerts
- [ ] Verify metrics collection
- [ ] Set up Grafana dashboards
- [ ] Configure alerting rules
- [ ] Test HPA behavior
- [ ] Perform load testing
- [ ] Document custom configuration
- [ ] Train operations team

### Security

- [ ] Use K8s Secrets for sensitive data
- [ ] Enable network policies
- [ ] Configure RBAC
- [ ] Set security contexts (non-root)
- [ ] Enable TLS for Ingress
- [ ] Review pod security policies
- [ ] Enable audit logging
- [ ] Rotate credentials regularly

---

## Additional Resources

- **Helm Chart Repository**: https://github.com/ipiton/alert-history-service/tree/main/helm
- **API Documentation**: `/docs/api/openapi.yaml`
- **Operations Runbook**: `/docs/operations/RUNBOOK.md` (TN-118)
- **Troubleshooting Guide**: `/docs/operations/TROUBLESHOOTING.md` (TN-119)
- **Architecture Documentation**: `/docs/architecture/ARCHITECTURE.md` (TN-120)
- **GitHub Issues**: https://github.com/ipiton/alert-history-service/issues

---

**Need Help?**

- 📖 Read the [Operations Runbook](../operations/RUNBOOK.md)
- 🐛 Check [Troubleshooting Guide](../operations/TROUBLESHOOTING.md)
- 💬 Ask in [GitHub Discussions](https://github.com/ipiton/alert-history-service/discussions)
- 🚨 Report bugs in [GitHub Issues](https://github.com/ipiton/alert-history-service/issues)
