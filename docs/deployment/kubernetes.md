# Kubernetes Deployment Guide

**Service**: Alert History - Intelligent Proxy Webhook
**Target**: Production Kubernetes Cluster
**Difficulty**: Intermediate
**Time**: 30 minutes

---

## Prerequisites

### Required

- ✅ Kubernetes cluster 1.23+ (with RBAC enabled)
- ✅ kubectl configured and connected
- ✅ Helm 3.x installed
- ✅ PostgreSQL database (for alert storage)
- ✅ Redis cluster (for classification cache)

### Optional

- ⚪ LLM service endpoint (for classification)
- ⚪ Prometheus operator (for metrics)
- ⚪ Cert-manager (for TLS)
- ⚪ Ingress controller (for external access)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────┐
│ Kubernetes Cluster                                       │
│                                                          │
│  ┌──────────────────┐      ┌────────────────┐         │
│  │ Ingress (TLS)    │─────▶│ Service        │         │
│  │ alert-history    │      │ alert-history  │         │
│  └──────────────────┘      └────────┬───────┘         │
│                                     │                   │
│           ┌─────────────────────────┼─────────┐        │
│           │                         │         │        │
│      ┌────▼────┐  ┌─────────────┐  ▼         │        │
│      │ Pod 1   │  │ Pod 2       │  Pod 3      │        │
│      │ proxy   │  │ proxy       │  proxy      │        │
│      └────┬────┘  └─────┬───────┘  └─────┬───┘        │
│           │             │                 │            │
│  ┌────────▼─────────────▼─────────────────▼────┐      │
│  │ External Dependencies                        │      │
│  ├──────────────────────────────────────────────┤      │
│  │ • PostgreSQL (alerts storage)                │      │
│  │ • Redis (classification cache)               │      │
│  │ • LLM Service (classification)               │      │
│  │ • Publishing Targets (Rootly, PagerDuty)     │      │
│  └──────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────┘
```

---

## Step 1: Prepare Configuration (5 minutes)

### 1.1 Create Namespace

```bash
kubectl create namespace alert-history
kubectl config set-context --current --namespace=alert-history
```

### 1.2 Create Secrets

```bash
# API keys for authentication
kubectl create secret generic alert-history-api-keys \
  --from-literal=api-key-1=$(openssl rand -hex 32) \
  --from-literal=api-key-2=$(openssl rand -hex 32) \
  --namespace=alert-history

# Database credentials
kubectl create secret generic alert-history-db \
  --from-literal=username=alerthistory \
  --from-literal=password=$(openssl rand -hex 32) \
  --from-literal=host=postgres.db.svc.cluster.local \
  --from-literal=port=5432 \
  --from-literal=database=alerthistory \
  --namespace=alert-history

# Redis credentials
kubectl create secret generic alert-history-redis \
  --from-literal=host=redis.cache.svc.cluster.local \
  --from-literal=port=6379 \
  --from-literal=password=$(openssl rand -hex 32) \
  --namespace=alert-history

# LLM service credentials (optional)
kubectl create secret generic alert-history-llm \
  --from-literal=endpoint=https://llm-service.company.com/v1 \
  --from-literal=api-key=sk_your_llm_api_key_here \
  --namespace=alert-history

# Publishing targets (optional)
kubectl create secret generic alert-history-publishing \
  --from-literal=rootly-api-key=your_rootly_key \
  --from-literal=pagerduty-routing-key=your_pagerduty_key \
  --from-literal=slack-webhook-url=https://hooks.slack.com/services/... \
  --namespace=alert-history
```

### 1.3 Create ConfigMap

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: alert-history-config
  namespace: alert-history
data:
  config.yaml: |
    server:
      port: 8080
      host: "0.0.0.0"
      read_timeout: 30s
      write_timeout: 30s
      shutdown_timeout: 10s

    proxy:
      enabled: true

      http:
        max_request_size: 10485760  # 10MB
        request_timeout: 30s
        max_alerts_per_req: 100

      classification:
        enabled: true
        timeout: 5s
        cache_ttl: 15m
        fallback_enabled: true
        fallback_category: "unknown"
        fallback_severity: "medium"

      filtering:
        enabled: true
        default_action: "allow"
        rules_file: "/etc/config/filter-rules.yaml"

      publishing:
        enabled: true
        parallel: true
        timeout_per_target: 5s
        max_targets: 10
        retry_enabled: true
        retry_max_attempts: 3
        retry_initial_interval: 1s
        continue_on_error: true

    middleware:
      rate_limiting:
        enabled: true
        per_ip_limit: 100
        global_limit: 1000
        burst: 50

      authentication:
        enabled: true
        type: "api_key"  # or "jwt"

      security_headers:
        enabled: true

      metrics:
        enabled: true
        path: "/metrics"

    logging:
      level: "info"  # debug, info, warn, error
      format: "json"  # json, text

    metrics:
      enabled: true
      namespace: "alert_history"

  filter-rules.yaml: |
    filters:
      - name: "ignore_test_alerts"
        type: "label"
        action: "deny"
        config:
          label: "alertname"
          regex: "^Test.*"

      - name: "only_production"
        type: "label"
        action: "deny"
        config:
          label: "environment"
          regex: "^(dev|staging)$"

      - name: "severity_threshold"
        type: "severity"
        action: "allow"
        config:
          min_severity: "warning"
EOF
```

---

## Step 2: Deploy Application (10 minutes)

### 2.1 Using Helm (Recommended)

```bash
# Add Helm repository
helm repo add alert-history https://charts.alerthistory.io
helm repo update

# Install with custom values
cat <<EOF > values.yaml
replicaCount: 3

image:
  repository: alerthistory/alert-history
  tag: "1.0.0"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080

resources:
  requests:
    memory: 256Mi
    cpu: 250m
  limits:
    memory: 512Mi
    cpu: 500m

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

ingress:
  enabled: true
  className: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: api.alerthistory.io
      paths:
        - path: /v1/webhook/proxy
          pathType: Prefix
  tls:
    - secretName: alert-history-tls
      hosts:
        - api.alerthistory.io

serviceMonitor:
  enabled: true
  interval: 30s

podDisruptionBudget:
  enabled: true
  minAvailable: 2

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
      - weight: 100
        podAffinityTerm:
          labelSelector:
            matchExpressions:
              - key: app
                operator: In
                values:
                  - alert-history
          topologyKey: kubernetes.io/hostname
EOF

# Install
helm install alert-history alert-history/alert-history \
  --namespace alert-history \
  --values values.yaml
```

### 2.2 Using kubectl (Alternative)

```bash
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: alert-history
  namespace: alert-history
spec:
  replicas: 3
  selector:
    matchLabels:
      app: alert-history
  template:
    metadata:
      labels:
        app: alert-history
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "8080"
        prometheus.io/path: "/metrics"
    spec:
      containers:
      - name: alert-history
        image: alerthistory/alert-history:1.0.0
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: CONFIG_FILE
          value: /etc/config/config.yaml
        - name: DB_HOST
          valueFrom:
            secretKeyRef:
              name: alert-history-db
              key: host
        - name: DB_PORT
          valueFrom:
            secretKeyRef:
              name: alert-history-db
              key: port
        - name: DB_USERNAME
          valueFrom:
            secretKeyRef:
              name: alert-history-db
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: alert-history-db
              key: password
        - name: REDIS_HOST
          valueFrom:
            secretKeyRef:
              name: alert-history-redis
              key: host
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: alert-history-redis
              key: password
        - name: LLM_ENDPOINT
          valueFrom:
            secretKeyRef:
              name: alert-history-llm
              key: endpoint
              optional: true
        - name: LLM_API_KEY
          valueFrom:
            secretKeyRef:
              name: alert-history-llm
              key: api-key
              optional: true
        volumeMounts:
        - name: config
          mountPath: /etc/config
        resources:
          requests:
            memory: 256Mi
            cpu: 250m
          limits:
            memory: 512Mi
            cpu: 500m
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
      volumes:
      - name: config
        configMap:
          name: alert-history-config
---
apiVersion: v1
kind: Service
metadata:
  name: alert-history
  namespace: alert-history
spec:
  selector:
    app: alert-history
  ports:
  - port: 8080
    targetPort: 8080
    name: http
  type: ClusterIP
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: alert-history
  namespace: alert-history
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.alerthistory.io
    secretName: alert-history-tls
  rules:
  - host: api.alerthistory.io
    http:
      paths:
      - path: /v1/webhook/proxy
        pathType: Prefix
        backend:
          service:
            name: alert-history
            port:
              number: 8080
EOF
```

---

## Step 3: Verify Deployment (5 minutes)

### 3.1 Check Pod Status

```bash
# Check pods
kubectl get pods -l app=alert-history

# Expected output:
# NAME                             READY   STATUS    RESTARTS   AGE
# alert-history-5d7c8f9b6d-abc12   1/1     Running   0          2m
# alert-history-5d7c8f9b6d-def34   1/1     Running   0          2m
# alert-history-5d7c8f9b6d-ghi56   1/1     Running   0          2m
```

### 3.2 Check Logs

```bash
# Check logs for errors
kubectl logs -l app=alert-history --tail=100

# Should see:
# {"level":"info","msg":"Starting Alert History Service","version":"1.0.0"}
# {"level":"info","msg":"Server listening","address":"0.0.0.0:8080"}
```

### 3.3 Test Health Endpoint

```bash
# Port-forward for local testing
kubectl port-forward svc/alert-history 8080:8080 &

# Health check
curl http://localhost:8080/health

# Expected: {"status":"healthy","timestamp":"2025-11-16T10:00:00Z"}
```

### 3.4 Test Proxy Endpoint

```bash
# Get API key
API_KEY=$(kubectl get secret alert-history-api-keys -o jsonpath='{.data.api-key-1}' | base64 -d)

# Send test alert
curl -X POST http://localhost:8080/v1/webhook/proxy \
  -H "X-API-Key: $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "receiver": "test",
    "status": "firing",
    "alerts": [{
      "status": "firing",
      "labels": {
        "alertname": "TestAlert",
        "severity": "critical"
      },
      "annotations": {
        "summary": "Test alert"
      },
      "startsAt": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"
    }]
  }'

# Expected: {"status":"success",...}
```

---

## Step 4: Configure Monitoring (5 minutes)

### 4.1 Service Monitor (Prometheus Operator)

```bash
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: alert-history
  namespace: alert-history
  labels:
    app: alert-history
spec:
  selector:
    matchLabels:
      app: alert-history
  endpoints:
  - port: http
    interval: 30s
    path: /metrics
EOF
```

### 4.2 Prometheus Rules

```bash
cat <<EOF | kubectl apply -f -
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: alert-history-proxy-alerts
  namespace: alert-history
spec:
  groups:
  - name: proxy_webhook
    interval: 30s
    rules:
    - alert: ProxyWebhookHighErrorRate
      expr: rate(alert_history_proxy_http_errors_total[5m]) > 10
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "High error rate on proxy webhook"

    - alert: ProxyWebhookHighLatency
      expr: histogram_quantile(0.95, rate(alert_history_proxy_http_request_duration_seconds_bucket[5m])) > 1.0
      for: 5m
      labels:
        severity: critical
      annotations:
        summary: "High p95 latency on proxy webhook"
EOF
```

---

## Step 5: Production Hardening (5 minutes)

### 5.1 Pod Disruption Budget

```bash
cat <<EOF | kubectl apply -f -
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: alert-history-pdb
  namespace: alert-history
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: alert-history
EOF
```

### 5.2 Network Policy

```bash
cat <<EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: alert-history-netpol
  namespace: alert-history
spec:
  podSelector:
    matchLabels:
      app: alert-history
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 443  # HTTPS (LLM, publishing targets)
EOF
```

### 5.3 Resource Quotas

```bash
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: ResourceQuota
metadata:
  name: alert-history-quota
  namespace: alert-history
spec:
  hard:
    requests.cpu: "5"
    requests.memory: 8Gi
    limits.cpu: "10"
    limits.memory: 16Gi
    pods: "20"
EOF
```

---

## Troubleshooting

### Pods Not Starting

```bash
# Check events
kubectl describe pod -l app=alert-history

# Check if secrets exist
kubectl get secrets

# Check if config exists
kubectl get configmap alert-history-config
```

### Health Check Failing

```bash
# Check pod logs
kubectl logs -l app=alert-history --tail=100

# Common issues:
# - Database connection failed
# - Redis connection failed
# - Config file parse error
```

### Ingress Not Working

```bash
# Check ingress
kubectl get ingress alert-history

# Check ingress controller logs
kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller

# Test from inside cluster
kubectl run curl --image=curlimages/curl -it --rm -- curl http://alert-history:8080/health
```

---

## Rollback

```bash
# Using Helm
helm rollback alert-history

# Using kubectl
kubectl rollout undo deployment/alert-history

# Check rollout status
kubectl rollout status deployment/alert-history
```

---

## Scaling

```bash
# Manual scaling
kubectl scale deployment alert-history --replicas=5

# Enable autoscaling
kubectl autoscale deployment alert-history \
  --min=3 \
  --max=10 \
  --cpu-percent=70
```

---

## Maintenance

### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap alert-history-config

# Restart pods to pick up changes
kubectl rollout restart deployment/alert-history
```

### Update Secrets

```bash
# Update secret
kubectl create secret generic alert-history-api-keys \
  --from-literal=api-key-1=new_key \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart
kubectl rollout restart deployment/alert-history
```

### Upgrade Version

```bash
# Using Helm
helm upgrade alert-history alert-history/alert-history \
  --set image.tag=1.1.0

# Using kubectl
kubectl set image deployment/alert-history alert-history=alerthistory/alert-history:1.1.0
```

---

## Next Steps

1. ✅ **Configure Alertmanager** - Point webhooks to your endpoint
2. ✅ **Set up Grafana** - Import dashboard for visualization
3. ✅ **Configure Alerts** - Set up PagerDuty/Slack notifications
4. ✅ **Test Load** - Run k6 load tests
5. ✅ **Monitor** - Watch metrics and logs

---

## Support

- **Documentation**: https://docs.alerthistory.io/deployment
- **Helm Charts**: https://github.com/alerthistory/charts
- **Issues**: https://github.com/alerthistory/alert-history/issues
- **Community**: https://slack.alerthistory.io
