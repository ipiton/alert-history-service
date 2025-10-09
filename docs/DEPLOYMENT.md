# Alert History Service - Production Deployment Guide

Полное руководство по развертыванию Alert History Service в production окружении с Kubernetes, Helm, PostgreSQL, Redis и мониторингом.

## 📋 Предварительные требования

### Infrastructure Requirements
- **Kubernetes cluster**: v1.20+
- **Helm**: v3.8+
- **Persistent storage**: ReadWriteOnce (для PostgreSQL)
- **Load balancer**: Nginx Ingress или эквивалент
- **Monitoring**: Prometheus + Grafana (опционально)

### Resource Requirements

| Component | CPU | Memory | Storage |
|-----------|-----|--------|---------|
| Alert History Service | 200m-1000m | 256Mi-1Gi | - |
| PostgreSQL | 100m-500m | 256Mi-1Gi | 10Gi |
| Redis | 50m-200m | 128Mi-512Mi | - |

---

## 🚀 Step-by-Step Deployment

### 1. Prepare Kubernetes Namespace

```bash
# Create namespace
kubectl create namespace alert-history

# Create namespace for publishing targets
kubectl create namespace alert-targets

# Set default namespace
kubectl config set-context --current --namespace=alert-history
```

### 2. Configure Database Secrets

```bash
# Create PostgreSQL secret
kubectl create secret generic postgresql-auth \
  --from-literal=postgres-password='your-secure-password' \
  --from-literal=password='your-secure-password' \
  --from-literal=username='alerthistory'

# Create application database secret
kubectl create secret generic alert-history-db \
  --from-literal=DATABASE_URL='postgresql://alerthistory:your-secure-password@alert-history-postgresql:5432/alerthistory'
```

### 3. Configure Publishing Targets

#### Rootly Integration
```bash
kubectl create secret generic rootly-config \
  --namespace=alert-targets \
  --from-literal=url='https://api.rootly.com' \
  --from-literal=api_key='your-rootly-api-key' \
  --from-literal=organization_id='your-org-id' \
  --from-literal=service_id='your-service-id'

kubectl label secret rootly-config \
  --namespace=alert-targets \
  alert-history.io/target=true \
  alert-history.io/format=rootly
```

#### Slack Integration
```bash
kubectl create secret generic slack-webhook \
  --namespace=alert-targets \
  --from-literal=webhook_url='https://hooks.slack.com/services/YOUR/SLACK/WEBHOOK'

kubectl label secret slack-webhook \
  --namespace=alert-targets \
  alert-history.io/target=true \
  alert-history.io/format=slack
```

#### PagerDuty Integration
```bash
kubectl create secret generic pagerduty-config \
  --namespace=alert-targets \
  --from-literal=integration_key='your-pagerduty-integration-key' \
  --from-literal=routing_key='your-routing-key'

kubectl label secret pagerduty-config \
  --namespace=alert-targets \
  alert-history.io/target=true \
  alert-history.io/format=pagerduty
```

### 4. Configure LLM Integration (Optional)

```bash
# LLM Proxy integration
kubectl create secret generic llm-config \
  --from-literal=LLM_PROXY_URL='http://llm-proxy.llm-namespace:8080' \
  --from-literal=LLM_MODEL='gpt-4' \
  --from-literal=LLM_API_KEY='your-api-key'
```

### 5. Deploy with Helm

#### Basic Deployment
```bash
# Add Helm repository (if using OCI registry)
helm repo add alert-history https://ghcr.io/ipiton/alert-history-service

# Install with basic configuration
helm install alert-history alert-history/alert-history \
  --namespace alert-history \
  --set postgresql.enabled=true \
  --set redis.enabled=true \
  --set autoscaling.enabled=true \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=alert-history.your-domain.com
```

#### Production Deployment with Custom Values
```yaml
# values-production.yaml
image:
  repository: ghcr.io/ipiton/alert-history-service
  tag: "latest"
  pullPolicy: IfNotPresent

replicaCount: 3

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 200m
    memory: 256Mi

env:
  - name: ENVIRONMENT
    value: "production"
  - name: LOG_LEVEL
    value: "info"
  - name: ENRICHMENT_MODE
    value: "enriched"
  - name: PUBLISHING_ENABLED
    value: "true"
  - name: TARGET_DISCOVERY_ENABLED
    value: "true"
  - name: TARGET_DISCOVERY_NAMESPACE
    value: "alert-targets"
  - name: DATABASE_URL
    valueFrom:
      secretKeyRef:
        name: alert-history-db
        key: DATABASE_URL

service:
  type: ClusterIP
  port: 8080

ingress:
  enabled: true
  className: "nginx"
  annotations:
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    nginx.ingress.kubernetes.io/force-ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: alert-history.your-domain.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: alert-history-tls
      hosts:
        - alert-history.your-domain.com

postgresql:
  enabled: true
  auth:
    existingSecret: postgresql-auth
    secretKeys:
      adminPasswordKey: postgres-password
      userPasswordKey: password
      replicationPasswordKey: password
    username: alerthistory
    database: alerthistory
  primary:
    persistence:
      enabled: true
      size: 20Gi
      storageClass: "fast-ssd"
    resources:
      limits:
        cpu: 500m
        memory: 1Gi
      requests:
        cpu: 100m
        memory: 256Mi

redis:
  enabled: true
  architecture: standalone
  auth:
    enabled: false
  master:
    persistence:
      enabled: false
    resources:
      limits:
        cpu: 200m
        memory: 512Mi
      requests:
        cpu: 50m
        memory: 128Mi

rbac:
  create: true
  rules:
    - apiGroups: [""]
      resources: ["secrets"]
      verbs: ["get", "list", "watch"]
    - apiGroups: [""]
      resources: ["configmaps"]
      verbs: ["get", "list", "watch"]

serviceAccount:
  create: true
  annotations: {}

monitoring:
  serviceMonitor:
    enabled: true
    interval: 30s
    path: /metrics
```

```bash
# Deploy with production values
helm install alert-history alert-history/alert-history \
  --namespace alert-history \
  --values values-production.yaml
```

### 6. Configure RBAC for Target Discovery

```yaml
# rbac-target-discovery.yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: alert-history-target-discovery
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: alert-history-target-discovery
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: alert-history-target-discovery
subjects:
- kind: ServiceAccount
  name: alert-history
  namespace: alert-history
```

```bash
kubectl apply -f rbac-target-discovery.yaml
```

---

## 🔧 Post-Deployment Configuration

### 1. Verify Deployment

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=alert-history

# Check service endpoints
kubectl get svc

# Check ingress
kubectl get ingress

# Test health endpoints
curl https://alert-history.your-domain.com/healthz
curl https://alert-history.your-domain.com/readyz
```

### 2. Configure Alertmanager Integration

```yaml
# alertmanager.yml
global:
  resolve_timeout: 5m

receivers:
  - name: 'alert-history-proxy'
    webhook_configs:
      - url: 'https://alert-history.your-domain.com/webhook/proxy'
        send_resolved: true
        http_config:
          follow_redirects: true

route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'alert-history-proxy'
```

### 3. Test Publishing Targets

```bash
# Check discovered targets
curl https://alert-history.your-domain.com/publishing/targets

# Check publishing mode
curl https://alert-history.your-domain.com/publishing/mode

# Test enrichment mode
curl https://alert-history.your-domain.com/enrichment/mode

# Switch to transparent mode for testing
curl -X POST https://alert-history.your-domain.com/enrichment/mode \
  -H "Content-Type: application/json" \
  -d '{"mode": "transparent"}'
```

### 4. Setup Monitoring

#### Import Grafana Dashboard
1. Скачайте `alert_history_grafana_dashboard_v3_enrichment.json`
2. Import в Grafana: **+ → Import → Upload JSON file**
3. Configure Prometheus datasource

#### Configure Prometheus Recording Rules
```yaml
# prometheus-rules.yaml
apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: alert-history-recording-rules
  namespace: alert-history
spec:
  groups:
  - name: alert_history_aggregation
    interval: 30s
    rules:
    - record: alert_history:enrichment_mode_current
      expr: max(alert_history_enrichment_mode_status)
    - record: alert_history:enrichment_efficiency
      expr: |
        sum(rate(alert_history_enrichment_enriched_alerts_total[5m])) /
        (sum(rate(alert_history_enrichment_transparent_alerts_total[5m])) + sum(rate(alert_history_enrichment_enriched_alerts_total[5m])))
    - record: alert_history:publishing_success_rate
      expr: |
        sum(rate(alert_history_publishing_total{status="success"}[5m])) by (target) /
        sum(rate(alert_history_publishing_total[5m])) by (target)
```

---

## 🔒 Security Considerations

### 1. Network Policies

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: alert-history-netpol
  namespace: alert-history
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: alert-history
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
    - namespaceSelector:
        matchLabels:
          name: alert-history
    ports:
    - protocol: TCP
      port: 5432  # PostgreSQL
    - protocol: TCP
      port: 6379  # Redis
  - to:
    - namespaceSelector:
        matchLabels:
          name: alert-targets
    ports:
    - protocol: TCP
      port: 443   # HTTPS для external APIs
```

### 2. Pod Security Standards

```yaml
# pod-security-policy.yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: alert-history-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  volumes:
    - 'configMap'
    - 'emptyDir'
    - 'projected'
    - 'secret'
    - 'downwardAPI'
    - 'persistentVolumeClaim'
  runAsUser:
    rule: 'MustRunAsNonRoot'
  seLinux:
    rule: 'RunAsAny'
  fsGroup:
    rule: 'RunAsAny'
```

---

## 📊 Monitoring & Alerting

### Key Metrics to Monitor

1. **Application Health**
   - `up{job="alert-history"}` — service availability
   - `alert_history_request_latency_seconds` — request latency
   - `alert_history_webhook_errors_total` — error rate

2. **Enrichment Mode**
   - `alert_history_enrichment_mode_status` — current mode
   - `alert_history_enrichment_efficiency` — enrichment efficiency
   - `alert_history_enrichment_mode_switches_total` — mode changes

3. **Publishing**
   - `alert_history_publishing_success_rate` — publishing success rate
   - `alert_history_publishing_targets_discovered` — discovered targets
   - `alert_history_publishing_queue_size` — queue size

### Recommended Alerts

```yaml
# alerts.yaml
groups:
- name: alert-history
  rules:
  - alert: AlertHistoryDown
    expr: up{job="alert-history"} == 0
    for: 5m
    labels:
      severity: critical
    annotations:
      summary: "Alert History Service is down"

  - alert: AlertHistoryHighErrorRate
    expr: rate(alert_history_webhook_errors_total[5m]) > 0.1
    for: 5m
    labels:
      severity: warning
    annotations:
      summary: "High error rate in Alert History Service"

  - alert: AlertHistoryPublishingFailed
    expr: alert_history:publishing_success_rate < 0.8
    for: 10m
    labels:
      severity: warning
    annotations:
      summary: "Publishing success rate is below 80%"
```

---

## 🔄 Backup & Recovery

### Database Backup

```bash
# Create backup job
kubectl create job alert-history-backup-$(date +%Y%m%d) \
  --from=cronjob/postgresql-backup

# Manual backup
kubectl exec -it deployment/alert-history-postgresql -- \
  pg_dump -U alerthistory alerthistory > backup.sql
```

### Disaster Recovery

1. **Database Recovery**
   ```bash
   kubectl exec -i deployment/alert-history-postgresql -- \
     psql -U alerthistory -d alerthistory < backup.sql
   ```

2. **Configuration Recovery**
   ```bash
   # Restore secrets
   kubectl apply -f secrets-backup/

   # Restart deployment
   kubectl rollout restart deployment/alert-history
   ```

---

## 🚀 Scaling & Performance

### Horizontal Scaling

```bash
# Manual scaling
kubectl scale deployment alert-history --replicas=5

# Configure HPA
kubectl autoscale deployment alert-history \
  --cpu-percent=70 \
  --min=3 \
  --max=10
```

### Performance Tuning

1. **Database Optimization**
   ```sql
   -- Create indexes for performance
   CREATE INDEX CONCURRENTLY idx_alerts_timestamp ON alerts(timestamp);
   CREATE INDEX CONCURRENTLY idx_alerts_alertname ON alerts(alertname);
   CREATE INDEX CONCURRENTLY idx_alerts_namespace ON alerts(namespace);
   ```

2. **Redis Configuration**
   ```yaml
   redis:
     master:
       configuration: |
         maxmemory 512mb
         maxmemory-policy allkeys-lru
         save ""  # Disable persistence for cache-only usage
   ```

---

## 🐛 Troubleshooting

### Common Issues

1. **Service не поднимается**
   ```bash
   kubectl describe pod <pod-name>
   kubectl logs <pod-name> --previous
   ```

2. **Target discovery не работает**
   ```bash
   # Check RBAC permissions
   kubectl auth can-i get secrets --as=system:serviceaccount:alert-history:alert-history

   # Check secret labels
   kubectl get secrets -n alert-targets -l alert-history.io/target=true
   ```

3. **Publishing fails**
   ```bash
   # Check network connectivity
   kubectl exec -it deployment/alert-history -- curl -I https://api.rootly.com

   # Check secret content
   kubectl get secret rootly-config -n alert-targets -o yaml
   ```

4. **High memory usage**
   ```bash
   # Check metrics
   kubectl top pods

   # Increase memory limits
   kubectl patch deployment alert-history -p '{"spec":{"template":{"spec":{"containers":[{"name":"alert-history","resources":{"limits":{"memory":"2Gi"}}}]}}}}'
   ```

### Debug Commands

```bash
# Enable debug logging
kubectl patch deployment alert-history -p '{"spec":{"template":{"spec":{"containers":[{"name":"alert-history","env":[{"name":"LOG_LEVEL","value":"debug"}]}]}}}}'

# Check application logs
kubectl logs -f deployment/alert-history

# Access application metrics
kubectl port-forward deployment/alert-history 8080:8080
curl http://localhost:8080/metrics

# Check database connectivity
kubectl exec -it deployment/alert-history -- python -c "
import asyncpg
import asyncio
async def test():
    conn = await asyncpg.connect('postgresql://alerthistory:password@alert-history-postgresql:5432/alerthistory')
    print('Database connection OK')
    await conn.close()
asyncio.run(test())
"
```

---

## 📋 Maintenance

### Regular Tasks

1. **Weekly**
   - Review error logs
   - Check disk usage
   - Update secrets rotation

2. **Monthly**
   - Database maintenance
   - Performance review
   - Security updates

3. **Quarterly**
   - Capacity planning
   - Disaster recovery testing
   - Documentation updates

### Upgrade Process

```bash
# Check current version
helm list -n alert-history

# Upgrade to new version
helm upgrade alert-history alert-history/alert-history \
  --namespace alert-history \
  --values values-production.yaml \
  --version 2.0.0

# Rollback if needed
helm rollback alert-history 1 -n alert-history
```

---

Для получения дополнительной поддержки обращайтесь к [документации проекта](../README.md) или создавайте [GitHub Issues](https://github.com/ipiton/alert-history-service/issues).
