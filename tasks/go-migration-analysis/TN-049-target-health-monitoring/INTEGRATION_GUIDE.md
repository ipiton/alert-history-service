# TN-049: Target Health Monitoring - Integration Guide

**Status**: 🚀 READY FOR DEPLOYMENT
**Pre-requisites**: K8s cluster with TN-046/047 enabled
**Estimated Time**: 30 minutes

---

## Overview

Данный guide содержит step-by-step инструкции для полной интеграции Health Monitoring в production environment.

---

## Pre-Deployment Checklist

### ✅ Dependencies Verified

| Dependency | Status | Verification Command |
|------------|--------|---------------------|
| **TN-046** K8s Client | ✅ Complete | Check `go-app/internal/infrastructure/k8s/` |
| **TN-047** Target Discovery | ✅ Complete | Check `go-app/internal/business/publishing/discovery.go` |
| **TN-048** Target Refresh | ✅ Complete | Check `go-app/internal/business/publishing/refresh_manager.go` |
| **TN-021** Prometheus | ✅ Complete | Check `go-app/pkg/metrics/` |
| **TN-020** Logging | ✅ Complete | Check `slog` usage |

### ✅ Code Ready

| Component | Status | Location |
|-----------|--------|----------|
| HealthMonitor Interface | ✅ Ready | `health.go` (500 LOC) |
| Implementation | ✅ Ready | `health_impl.go` (500 LOC) |
| HTTP Checker | ✅ Ready | `health_checker.go` (310 LOC) |
| Background Worker | ✅ Ready | `health_worker.go` (280 LOC) |
| HTTP API | ✅ Ready | `handlers/publishing_health.go` (350 LOC) |
| Integration | ✅ Ready | `main.go` (lines 878-943, commented) |

### ✅ K8s Resources

| Resource | Status | File |
|----------|--------|------|
| ServiceAccount | 📝 To Create | `k8s/publishing/serviceaccount.yaml` |
| Role | 📝 To Create | `k8s/publishing/role.yaml` |
| RoleBinding | 📝 To Create | `k8s/publishing/rolebinding.yaml` |

---

## Step 1: Enable Integration in main.go

### Option A: Manual (5 mins)

1. Open `go-app/cmd/server/main.go`
2. Go to line 809 (start of Publishing System block)
3. **Uncomment lines 809-947** (TN-046/047/048/049 integration)
4. Save file

**Lines to uncomment**:
```go
// Line 809: Start of K8s Client initialization
// k8sClient, err := k8s.NewK8sClient(k8s.DefaultK8sClientConfig())

// ... (all lines until line 947)

// Line 947: End of Health Monitor integration
// }
```

### Option B: Automated Script (1 min)

**Create activation script**:

```bash
#!/bin/bash
# scripts/enable-health-monitoring.sh

set -euo pipefail

MAIN_FILE="go-app/cmd/server/main.go"

echo "🔧 Enabling TN-049 Health Monitoring integration..."

# Uncomment lines 809-947 (Publishing System + Health Monitor)
sed -i.bak '809,947s|^\([[:space:]]*\)// \(.*\)|\1\2|' "$MAIN_FILE"

echo "✅ Integration enabled in $MAIN_FILE"
echo "📋 Backup saved as ${MAIN_FILE}.bak"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff $MAIN_FILE"
echo "  2. Build: make build"
echo "  3. Deploy to K8s"
```

**Usage**:
```bash
chmod +x scripts/enable-health-monitoring.sh
./scripts/enable-health-monitoring.sh
```

### Option C: Git Branch Strategy (recommended)

```bash
# Create deployment branch
git checkout -b deploy/enable-health-monitoring

# Uncomment integration code
vim go-app/cmd/server/main.go  # uncomment lines 809-947

# Commit
git add go-app/cmd/server/main.go
git commit -m "deploy: Enable TN-049 Health Monitoring integration

- Uncommented K8s client initialization (TN-046)
- Uncommented Target Discovery Manager (TN-047)
- Uncommented Target Refresh Manager (TN-048)
- Uncommented Health Monitor (TN-049)
- Ready for K8s deployment"

# Merge to main when ready
git checkout main
git merge deploy/enable-health-monitoring
```

---

## Step 2: Create K8s RBAC Resources

### 2.1 ServiceAccount

**File**: `k8s/publishing/serviceaccount.yaml`

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: alert-history-publishing
  namespace: default  # Change to your namespace
  labels:
    app: alert-history
    component: publishing
    managed-by: TN-049
```

### 2.2 Role (Secrets Read Access)

**File**: `k8s/publishing/role.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
  namespace: default  # Change to your namespace
  labels:
    app: alert-history
    component: publishing
    managed-by: TN-049
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
  # Optional: Restrict to specific secrets with resourceNames
  # resourceNames: ["rootly-target", "pagerduty-target"]
```

### 2.3 RoleBinding

**File**: `k8s/publishing/rolebinding.yaml`

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
  namespace: default  # Change to your namespace
  labels:
    app: alert-history
    component: publishing
    managed-by: TN-049
subjects:
- kind: ServiceAccount
  name: alert-history-publishing
  namespace: default  # Change to your namespace
roleRef:
  kind: Role
  name: alert-history-secrets-reader
  apiGroup: rbac.authorization.k8s.io
```

### 2.4 Apply RBAC

```bash
# Create ServiceAccount
kubectl apply -f k8s/publishing/serviceaccount.yaml

# Create Role
kubectl apply -f k8s/publishing/role.yaml

# Create RoleBinding
kubectl apply -f k8s/publishing/rolebinding.yaml

# Verify
kubectl get serviceaccount alert-history-publishing
kubectl get role alert-history-secrets-reader
kubectl get rolebinding alert-history-secrets-reader-binding
```

---

## Step 3: Update Deployment Manifest

### 3.1 Add ServiceAccount to Deployment

**File**: `helm/alert-history/templates/deployment.yaml`

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "alert-history.fullname" . }}
spec:
  template:
    spec:
      # Add ServiceAccount
      serviceAccountName: alert-history-publishing

      containers:
      - name: alert-history
        image: {{ .Values.image.repository }}:{{ .Values.image.tag }}
        env:
        # Health Monitoring Configuration
        - name: TARGET_HEALTH_CHECK_INTERVAL
          value: "2m"
        - name: TARGET_HEALTH_CHECK_TIMEOUT
          value: "5s"
        - name: TARGET_HEALTH_FAILURE_THRESHOLD
          value: "3"
        - name: TARGET_HEALTH_MAX_CONCURRENT
          value: "10"

        # K8s Configuration (for TN-046)
        - name: K8S_NAMESPACE
          valueFrom:
            fieldRef:
              fieldPath: metadata.namespace

        # Target Discovery Configuration (TN-047)
        - name: TARGET_LABEL_SELECTOR
          value: "publishing-target=true"

        # Target Refresh Configuration (TN-048)
        - name: TARGET_REFRESH_INTERVAL
          value: "5m"
        - name: TARGET_REFRESH_TIMEOUT
          value: "30s"
```

### 3.2 Add Health Check Probes

```yaml
        # Health probes
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5

        readinessProbe:
          httpGet:
            path: /api/v2/publishing/targets/health/stats
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
```

---

## Step 4: Build and Deploy

### 4.1 Build Docker Image

```bash
# Build image
docker build -t alert-history:v1.0.0-health-monitoring .

# Tag for registry
docker tag alert-history:v1.0.0-health-monitoring \
  your-registry.com/alert-history:v1.0.0-health-monitoring

# Push to registry
docker push your-registry.com/alert-history:v1.0.0-health-monitoring
```

### 4.2 Deploy to K8s

```bash
# Using Helm
helm upgrade --install alert-history ./helm/alert-history \
  --set image.tag=v1.0.0-health-monitoring \
  --set serviceAccount.name=alert-history-publishing \
  --namespace default

# Or using kubectl
kubectl apply -f k8s/deployment.yaml
```

### 4.3 Verify Deployment

```bash
# Check pod status
kubectl get pods -l app=alert-history

# Check logs for health monitor startup
kubectl logs -f deployment/alert-history | grep "Health Monitor"

# Expected output:
# ✅ K8s Client initialized (TN-046)
# ✅ Target Discovery Manager initialized (TN-047)
# ✅ Refresh Manager started (TN-048)
# ✅ Health Monitor started (TN-049)
# ✅ Health Monitor API endpoints registered (TN-049)
```

---

## Step 5: Verify Health Monitoring

### 5.1 Check API Endpoints

```bash
# Port-forward to local
kubectl port-forward deployment/alert-history 8080:8080

# Test health endpoint
curl http://localhost:8080/api/v2/publishing/targets/health | jq

# Expected response:
# [
#   {
#     "target_name": "rootly-prod",
#     "target_type": "rootly",
#     "enabled": true,
#     "status": "healthy",
#     "latency_ms": 145,
#     "last_check": "2025-11-08T14:30:00Z",
#     ...
#   }
# ]
```

### 5.2 Check Prometheus Metrics

```bash
# Check metrics endpoint
curl http://localhost:8080/metrics | grep alert_history_health

# Expected metrics:
# alert_history_health_checks_total{target_name="rootly-prod",error_type="none",is_healthy="true"} 123
# alert_history_health_check_duration_seconds_sum{target_name="rootly-prod"} 18.5
# alert_history_targets_monitored_total 5
# alert_history_targets_healthy 4
# alert_history_targets_unhealthy 1
```

### 5.3 Trigger Manual Health Check

```bash
# Trigger manual check
curl -X POST http://localhost:8080/api/v2/publishing/targets/health/rootly-prod/check | jq

# Expected response (200 OK if healthy):
# {
#   "target_name": "rootly-prod",
#   "status": "healthy",
#   "latency_ms": 145,
#   "last_check": "2025-11-08T14:45:12Z"
# }

# Or 503 Service Unavailable if unhealthy:
# {
#   "target_name": "slack-ops",
#   "status": "unhealthy",
#   "error_message": "connection timeout after 5s",
#   "last_check": "2025-11-08T14:45:20Z"
# }
```

---

## Step 6: Set Up Monitoring

### 6.1 Import Grafana Dashboard

**File**: `grafana/health-monitoring-dashboard.json`

```json
{
  "dashboard": {
    "title": "Target Health Monitoring (TN-049)",
    "panels": [
      {
        "title": "Health Status Overview",
        "targets": [
          {
            "expr": "alert_history_targets_healthy"
          },
          {
            "expr": "alert_history_targets_unhealthy"
          }
        ]
      },
      {
        "title": "Success Rate by Target",
        "targets": [
          {
            "expr": "sum by (target_name) (rate(alert_history_health_checks_total{is_healthy=\"true\"}[5m])) / sum by (target_name) (rate(alert_history_health_checks_total[5m]))"
          }
        ]
      },
      {
        "title": "p95 Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, sum(rate(alert_history_health_check_duration_seconds_bucket[5m])) by (le, target_name))"
          }
        ]
      }
    ]
  }
}
```

### 6.2 Configure Alerting Rules

**File**: `prometheus/health-alerts.yaml`

```yaml
groups:
- name: target_health_monitoring
  interval: 30s
  rules:
  - alert: TargetUnhealthy
    expr: alert_history_targets_unhealthy > 0
    for: 5m
    labels:
      severity: warning
      component: publishing
    annotations:
      summary: "{{ $value }} publishing target(s) unhealthy"
      description: "One or more targets have been unhealthy for 5+ minutes"

  - alert: HighHealthCheckFailureRate
    expr: |
      sum by (target_name) (rate(alert_history_health_checks_total{is_healthy="false"}[5m]))
        / sum by (target_name) (rate(alert_history_health_checks_total[5m]))
      > 0.5
    for: 10m
    labels:
      severity: critical
      component: publishing
    annotations:
      summary: "{{ $labels.target_name }} failure rate {{ $value | humanizePercentage }}"
      description: "Target health checks failing >50% for 10+ minutes"

  - alert: SlowHealthChecks
    expr: |
      histogram_quantile(0.95, sum(rate(alert_history_health_check_duration_seconds_bucket[5m])) by (le, target_name))
      > 5
    for: 15m
    labels:
      severity: warning
      component: publishing
    annotations:
      summary: "{{ $labels.target_name }} p95 latency {{ $value }}s"
      description: "Health checks taking >5s (p95) for 15+ minutes"
```

### 6.3 Apply Prometheus Configuration

```bash
# Apply alerting rules
kubectl apply -f prometheus/health-alerts.yaml

# Reload Prometheus
kubectl exec -it prometheus-0 -- kill -HUP 1
```

---

## Step 7: Create Target Secrets (Example)

### 7.1 Rootly Target Secret

**File**: `k8s/publishing/secrets/rootly-prod.yaml`

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: rootly-prod
  namespace: default
  labels:
    publishing-target: "true"  # Required for discovery!
    target-type: rootly
type: Opaque
stringData:
  config: |
    {
      "name": "rootly-prod",
      "type": "rootly",
      "url": "https://api.rootly.com/v1",
      "enabled": true,
      "headers": {
        "Authorization": "Bearer YOUR_ROOTLY_API_TOKEN"
      },
      "format": "rootly"
    }
```

### 7.2 PagerDuty Target Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: pagerduty-prod
  namespace: default
  labels:
    publishing-target: "true"  # Required!
    target-type: pagerduty
type: Opaque
stringData:
  config: |
    {
      "name": "pagerduty-prod",
      "type": "pagerduty",
      "url": "https://api.pagerduty.com",
      "enabled": true,
      "headers": {
        "Authorization": "Token token=YOUR_PAGERDUTY_TOKEN",
        "Content-Type": "application/json"
      },
      "format": "pagerduty"
    }
```

### 7.3 Apply Secrets

```bash
# Create secrets
kubectl apply -f k8s/publishing/secrets/rootly-prod.yaml
kubectl apply -f k8s/publishing/secrets/pagerduty-prod.yaml

# Verify discovery
kubectl get secrets -l publishing-target=true

# Expected output:
# NAME             TYPE     DATA   AGE
# rootly-prod      Opaque   1      10s
# pagerduty-prod   Opaque   1      5s
```

---

## Step 8: Verify End-to-End

### 8.1 Check Target Discovery

```bash
# Check discovered targets
curl http://localhost:8080/api/v2/publishing/targets/discovery/stats | jq

# Expected response:
# {
#   "total_targets": 2,
#   "valid_targets": 2,
#   "invalid_targets": 0,
#   "targets_by_type": {
#     "rootly": 1,
#     "pagerduty": 1
#   }
# }
```

### 8.2 Check Health Status

```bash
# Wait 30s for warmup + first health check
sleep 30

# Check health status
curl http://localhost:8080/api/v2/publishing/targets/health | jq

# Expected: Both targets should be "healthy" (if configured correctly)
```

### 8.3 Check Logs

```bash
# Check logs for health check activity
kubectl logs -f deployment/alert-history | grep -E "(Health|target_name)"

# Expected log entries:
# [INFO] Health check succeeded target_name=rootly-prod latency_ms=145
# [INFO] Health check succeeded target_name=pagerduty-prod latency_ms=230
# [INFO] Health checks completed total_checked=2 successes=2 failures=0
```

---

## Troubleshooting

### Issue 1: Health Monitor Not Starting

**Symptoms**:
- No health check logs
- `/health` endpoint returns 404

**Causes**:
- Integration code still commented in main.go
- K8s client initialization failed
- RBAC permissions missing

**Solutions**:
```bash
# 1. Check if integration enabled
grep -n "Health Monitor started" go-app/cmd/server/main.go
# Should show uncommented line (not starting with //)

# 2. Check RBAC
kubectl auth can-i list secrets --as=system:serviceaccount:default:alert-history-publishing
# Should return "yes"

# 3. Check logs for errors
kubectl logs deployment/alert-history | grep -i error
```

---

### Issue 2: No Targets Discovered

**Symptoms**:
- Health endpoint returns empty array `[]`
- `targets_monitored_total` metric is 0

**Causes**:
- No secrets with label `publishing-target=true`
- Wrong namespace
- Secret format invalid

**Solutions**:
```bash
# 1. Check secrets
kubectl get secrets -l publishing-target=true

# 2. Check secret format
kubectl get secret rootly-prod -o jsonpath='{.data.config}' | base64 -d | jq

# 3. Check discovery logs
kubectl logs deployment/alert-history | grep "Target Discovery"
```

---

### Issue 3: Targets Always "Unhealthy"

**Symptoms**:
- All targets show `status: "unhealthy"`
- Error: "connection timeout after 5s"

**Causes**:
- Target URL unreachable from pod
- Invalid credentials
- Network policies blocking egress

**Solutions**:
```bash
# 1. Test connectivity from pod
kubectl exec -it deployment/alert-history -- curl -v https://api.rootly.com/v1

# 2. Check network policies
kubectl get networkpolicies

# 3. Increase timeout
kubectl set env deployment/alert-history TARGET_HEALTH_CHECK_TIMEOUT=10s
```

---

## Rollback Plan

### If Something Goes Wrong

```bash
# 1. Scale down deployment
kubectl scale deployment alert-history --replicas=0

# 2. Revert code changes
git checkout main -- go-app/cmd/server/main.go

# 3. Rebuild and redeploy
docker build -t alert-history:rollback .
kubectl set image deployment/alert-history alert-history=alert-history:rollback

# 4. Scale up
kubectl scale deployment alert-history --replicas=3

# 5. Verify
kubectl get pods -l app=alert-history
```

---

## Success Criteria

✅ **Deployment Successful** if:
1. Pod starts without errors
2. Health monitor logs `✅ Health Monitor started`
3. At least 1 target discovered
4. First health check completes within 1 minute
5. `/health` endpoint returns 200 OK
6. Prometheus metrics visible
7. Grafana dashboard shows data

---

## Post-Deployment Tasks

1. ✅ **Monitor for 24 hours**
   - Check error rates
   - Verify health checks running periodically
   - Monitor resource usage (CPU/memory)

2. ✅ **Set up alerting**
   - Import Prometheus rules
   - Configure PagerDuty/Slack notifications
   - Test alerts manually

3. ✅ **Complete Phase 7 (Testing)**
   - Write integration tests
   - Run load tests (100+ targets)
   - Verify race detector clean

4. ✅ **Document learnings**
   - Update runbook with production issues
   - Add troubleshooting tips
   - Share with team

---

## Next Steps

After successful deployment of TN-049:
- **TN-050**: RBAC documentation (already covered here)
- **TN-051**: Alert Formatter implementation
- **TN-052-060**: Publishing System completion

---

**Integration Owner**: Vitalii Semenov (@vitaliisemenov)
**Last Updated**: 2025-11-08
**Status**: 🚀 READY FOR DEPLOYMENT
