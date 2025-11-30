# Alertmanager++ OSS Operations Runbook

**Version**: 2.0.0
**Last Updated**: 2025-11-30
**Audience**: SRE, Platform Engineers, Operations Teams

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Daily Operations](#daily-operations)
3. [Monitoring & Alerting](#monitoring--alerting)
4. [Incident Response](#incident-response)
5. [Backup & Recovery](#backup--recovery)
6. [Scaling Operations](#scaling-operations)
7. [Maintenance Procedures](#maintenance-procedures)
8. [Performance Tuning](#performance-tuning)
9. [Common Issues & Solutions](#common-issues--solutions)

---

## Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Alertmanager++                           │
│                                                             │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  │
│  │  Pod 1   │  │  Pod 2   │  │  Pod 3   │  │  Pod N   │  │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  │
│       │             │             │             │         │
│       └─────────────┴─────────────┴─────────────┘         │
│                         │                                  │
└─────────────────────────┼──────────────────────────────────┘
                          │
        ┌─────────────────┼─────────────────┐
        │                 │                 │
   ┌────▼─────┐    ┌─────▼──────┐   ┌─────▼──────┐
   │PostgreSQL│    │Redis/Valkey│   │ LLM Service│
   │(HA Setup)│    │  (Cache)   │   │ (Optional) │
   └──────────┘    └────────────┘   └────────────┘
```

### Deployment Profiles

| Profile | Use Case | Dependencies | Replicas | Auto-Scale |
|---------|----------|--------------|----------|------------|
| **Lite** | Dev/Test, <1K alerts/day | SQLite (PVC) | 1 | No |
| **Standard** | Production, >1K alerts/day | PostgreSQL, Redis | 2-10 | Yes (HPA) |

### Key Components

- **Alert Ingestion**: `/webhook`, `/webhook/proxy`
- **Classification**: LLM-powered alert analysis
- **History Storage**: PostgreSQL (Standard) or SQLite (Lite)
- **Caching**: Redis/Valkey (Standard) or Memory (Lite)
- **Publishing**: Multi-target (Slack, PagerDuty, Rootly, Webhooks)

---

## Daily Operations

### Morning Checklist

```bash
#!/bin/bash
# Daily health check script

echo "=== Alertmanager++ Daily Health Check ==="

# 1. Check pod status
echo "1. Pod Status:"
kubectl get pods -n alertmanager-plus -o wide

# 2. Check HPA status (Standard profile only)
echo -e "\n2. HPA Status:"
kubectl get hpa -n alertmanager-plus

# 3. Check recent restarts
echo -e "\n3. Recent Restarts:"
kubectl get pods -n alertmanager-plus -o json | \
  jq -r '.items[] | select(.status.containerStatuses[].restartCount > 0) |
  "\(.metadata.name): \(.status.containerStatuses[].restartCount) restarts"'

# 4. Check resource usage
echo -e "\n4. Resource Usage:"
kubectl top pods -n alertmanager-plus

# 5. Check database connections (Standard profile)
echo -e "\n5. Database Connections:"
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT count(*) as active_connections FROM pg_stat_activity WHERE state = 'active';"

# 6. Check alert processing rate (last hour)
echo -e "\n6. Alert Processing Rate:"
curl -s http://localhost:8080/api/v2/history/stats?interval=1h | jq

# 7. Check error rate
echo -e "\n7. Error Rate (last 5min):"
kubectl logs -n alertmanager-plus deployment/alert-history --tail=100 --since=5m | \
  grep -i error | wc -l

echo -e "\n=== Health Check Complete ==="
```

### Key Metrics to Monitor

| Metric | Threshold | Action |
|--------|-----------|--------|
| Pod restarts | >3 in 1h | Investigate logs, check resources |
| CPU usage | >80% sustained | Scale up or optimize |
| Memory usage | >85% | Check for memory leaks, scale up |
| Alert processing latency | >5s p95 | Check database performance |
| Database connections | >200/250 (80%) | Scale replicas or increase pool |
| Cache hit rate | <70% | Increase cache TTL or size |
| Error rate | >1% | Check logs, investigate errors |

### Daily Tasks

1. **Review Metrics** (9:00 AM daily)
   - Check Grafana dashboards
   - Review alert processing rate
   - Verify no alerts stuck in queue

2. **Log Review** (10:00 AM daily)
   ```bash
   # Check for errors
   kubectl logs -n alertmanager-plus deployment/alert-history \
     --since=24h --tail=1000 | grep -i "error\|fatal\|panic"

   # Check slow queries
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c \
     "SELECT query, mean_exec_time FROM pg_stat_statements
      ORDER BY mean_exec_time DESC LIMIT 10;"
   ```

3. **Capacity Planning** (Weekly)
   - Review 7-day trends
   - Check disk usage (PostgreSQL, PVC)
   - Forecast scaling needs

---

## Monitoring & Alerting

### Prometheus Metrics

#### Key Metrics to Monitor

```promql
# 1. Alert Processing Rate
rate(alert_history_business_alerts_received_total[5m])

# 2. Alert Processing Latency (p95)
histogram_quantile(0.95,
  rate(alert_history_business_alert_processing_duration_seconds_bucket[5m]))

# 3. Classification Success Rate
rate(alert_history_business_llm_classifications_total{status="success"}[5m]) /
rate(alert_history_business_llm_classifications_total[5m])

# 4. Database Connection Pool Usage
alert_history_infra_db_connections_active /
alert_history_infra_db_connections_max

# 5. Cache Hit Rate
rate(alert_history_business_classification_l1_cache_hits_total[5m]) /
(rate(alert_history_business_classification_l1_cache_hits_total[5m]) +
 rate(alert_history_business_classification_cache_misses_total[5m]))

# 6. Publishing Success Rate
rate(alert_history_business_publishing_success_total[5m]) /
rate(alert_history_business_publishing_attempts_total[5m])

# 7. Error Rate
rate(alert_history_technical_errors_total[5m])

# 8. Pod CPU Usage
rate(container_cpu_usage_seconds_total{pod=~"alert-history-.*"}[5m])

# 9. Pod Memory Usage
container_memory_working_set_bytes{pod=~"alert-history-.*"}

# 10. Database Slow Queries
pg_stat_statements_mean_exec_time_seconds > 1
```

### Alerting Rules

Save as `prometheus-alerts.yaml`:

```yaml
groups:
  - name: alertmanager-plus
    interval: 30s
    rules:
      # Critical Alerts
      - alert: AlertManagerPlusDown
        expr: up{job="alert-history"} == 0
        for: 5m
        labels:
          severity: critical
          component: alertmanager-plus
        annotations:
          summary: "Alertmanager++ is down"
          description: "Pod {{ $labels.pod }} has been down for more than 5 minutes"

      - alert: HighErrorRate
        expr: |
          rate(alert_history_technical_errors_total[5m]) > 10
        for: 5m
        labels:
          severity: critical
          component: alertmanager-plus
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value }} errors/sec (threshold: 10)"

      - alert: DatabaseConnectionPoolExhaustion
        expr: |
          (alert_history_infra_db_connections_active /
           alert_history_infra_db_connections_max) > 0.9
        for: 5m
        labels:
          severity: critical
          component: database
        annotations:
          summary: "Database connection pool near exhaustion"
          description: "Connection pool usage at {{ $value | humanizePercentage }}"

      # Warning Alerts
      - alert: HighMemoryUsage
        expr: |
          (container_memory_working_set_bytes{pod=~"alert-history-.*"} /
           container_spec_memory_limit_bytes{pod=~"alert-history-.*"}) > 0.85
        for: 10m
        labels:
          severity: warning
          component: alertmanager-plus
        annotations:
          summary: "High memory usage on {{ $labels.pod }}"
          description: "Memory usage at {{ $value | humanizePercentage }}"

      - alert: HighCPUUsage
        expr: |
          rate(container_cpu_usage_seconds_total{pod=~"alert-history-.*"}[5m]) > 0.8
        for: 10m
        labels:
          severity: warning
          component: alertmanager-plus
        annotations:
          summary: "High CPU usage on {{ $labels.pod }}"
          description: "CPU usage at {{ $value | humanizePercentage }}"

      - alert: LowCacheHitRate
        expr: |
          (rate(alert_history_business_classification_l1_cache_hits_total[10m]) /
          (rate(alert_history_business_classification_l1_cache_hits_total[10m]) +
           rate(alert_history_business_classification_cache_misses_total[10m]))) < 0.7
        for: 15m
        labels:
          severity: warning
          component: cache
        annotations:
          summary: "Low cache hit rate"
          description: "Cache hit rate at {{ $value | humanizePercentage }} (threshold: 70%)"

      - alert: SlowAlertProcessing
        expr: |
          histogram_quantile(0.95,
            rate(alert_history_business_alert_processing_duration_seconds_bucket[5m])) > 5
        for: 10m
        labels:
          severity: warning
          component: alertmanager-plus
        annotations:
          summary: "Slow alert processing detected"
          description: "p95 processing time: {{ $value }}s (threshold: 5s)"

      - alert: PublishingFailureRate
        expr: |
          (rate(alert_history_business_publishing_failed_total[10m]) /
           rate(alert_history_business_publishing_attempts_total[10m])) > 0.05
        for: 10m
        labels:
          severity: warning
          component: publishing
        annotations:
          summary: "High publishing failure rate"
          description: "Failure rate: {{ $value | humanizePercentage }} (threshold: 5%)"

      - alert: PostgreSQLSlowQueries
        expr: |
          pg_stat_statements_mean_exec_time_seconds > 1
        for: 5m
        labels:
          severity: warning
          component: database
        annotations:
          summary: "Slow PostgreSQL queries detected"
          description: "Query {{ $labels.query }} taking {{ $value }}s on average"
```

### Grafana Dashboards

#### Main Dashboard Panels

1. **Overview**
   - Total alerts received (24h)
   - Alert processing rate (5m)
   - Error rate (5m)
   - Cache hit rate (5m)

2. **Performance**
   - Alert processing latency (p50, p95, p99)
   - Database query latency
   - LLM classification latency
   - Publishing latency

3. **Resource Usage**
   - CPU usage per pod
   - Memory usage per pod
   - Network I/O
   - Disk I/O (PostgreSQL)

4. **Business Metrics**
   - Alerts by severity
   - Alerts by namespace
   - Top firing alerts
   - Classification accuracy

5. **Database Health**
   - Connection pool usage
   - Active connections
   - Slow query count
   - Table sizes

6. **Cache Performance**
   - L1 cache hits/misses
   - L2 (Redis) cache hits/misses
   - Cache size
   - Eviction rate

---

## Incident Response

### Severity Levels

| Severity | Response Time | Escalation |
|----------|---------------|------------|
| **P0 - Critical** | Immediate | Page on-call engineer |
| **P1 - High** | <15min | Notify team channel |
| **P2 - Medium** | <1h | Create ticket |
| **P3 - Low** | <24h | Backlog |

### Incident Playbooks

#### P0: Service Down

```bash
# 1. Assess impact
kubectl get pods -n alertmanager-plus

# 2. Check recent events
kubectl get events -n alertmanager-plus --sort-by='.lastTimestamp' | tail -20

# 3. Check logs
kubectl logs -n alertmanager-plus deployment/alert-history --tail=100

# 4. Describe failing pods
kubectl describe pod -n alertmanager-plus <pod-name>

# 5. Check dependencies
kubectl get pods -n alertmanager-plus -l app=postgresql
kubectl get pods -n alertmanager-plus -l app=valkey

# 6. Quick fixes
# Option A: Restart deployment
kubectl rollout restart deployment/alert-history -n alertmanager-plus

# Option B: Scale to 0 and back
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=0
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=3

# 7. Verify recovery
kubectl get pods -n alertmanager-plus -w
curl http://localhost:8080/health
```

#### P0: Database Connection Pool Exhaustion

```bash
# 1. Confirm issue
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT count(*) FROM pg_stat_activity;"

# 2. Check active connections by state
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT state, count(*) FROM pg_stat_activity GROUP BY state;"

# 3. Kill idle connections (if safe)
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
   WHERE state = 'idle' AND state_change < NOW() - INTERVAL '10 minutes';"

# 4. Immediate: Scale up app replicas to reduce load per pod
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=2

# 5. Long-term: Increase max_connections
kubectl edit configmap postgresql-config -n alertmanager-plus
# Change max_connections to 300, then restart PostgreSQL
```

#### P1: High Memory Usage

```bash
# 1. Identify pod with high memory
kubectl top pods -n alertmanager-plus --sort-by=memory

# 2. Check for memory leaks
kubectl logs -n alertmanager-plus <pod-name> | grep -i "memory\|oom"

# 3. Check cache size
curl http://localhost:8080/api/v2/cache/stats

# 4. Restart high-memory pod
kubectl delete pod -n alertmanager-plus <pod-name>

# 5. If persistent, increase memory limits
kubectl edit deployment alert-history -n alertmanager-plus
# Update resources.limits.memory

# 6. Monitor recovery
kubectl get pods -n alertmanager-plus -w
kubectl top pods -n alertmanager-plus
```

#### P1: High Error Rate

```bash
# 1. Check error types
kubectl logs -n alertmanager-plus deployment/alert-history --tail=200 | \
  grep -i error | cut -d' ' -f5- | sort | uniq -c | sort -rn

# 2. Check specific error details
kubectl logs -n alertmanager-plus deployment/alert-history --tail=500 | grep "ERROR"

# 3. Check external dependencies
# PostgreSQL
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  sh -c 'nc -zv $DATABASE_HOST $DATABASE_PORT'

# Redis
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  sh -c 'redis-cli -h $REDIS_ADDR ping'

# LLM Service
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  sh -c 'curl -s -o /dev/null -w "%{http_code}" $LLM_BASE_URL/health'

# 4. Review recent changes
kubectl rollout history deployment/alert-history -n alertmanager-plus

# 5. Rollback if recent deployment
kubectl rollout undo deployment/alert-history -n alertmanager-plus
```

### Escalation Contacts

| Role | Contact | Escalation |
|------|---------|------------|
| **On-Call SRE** | Slack: #sre-oncall | PagerDuty |
| **Platform Lead** | Email: platform-lead@example.com | Phone |
| **Database Admin** | Slack: #dba-oncall | PagerDuty |
| **Engineering Manager** | Phone: +1-XXX-XXX-XXXX | Email |

---

## Backup & Recovery

### Backup Strategy

#### PostgreSQL Backup (Standard Profile)

```bash
# Daily automated backup (CronJob)
apiVersion: batch/v1
kind: CronJob
metadata:
  name: postgresql-backup
  namespace: alertmanager-plus
spec:
  schedule: "0 2 * * *"  # 2 AM daily
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:16-alpine
            command:
            - /bin/sh
            - -c
            - |
              BACKUP_FILE="/backup/alerthistory-$(date +%Y%m%d-%H%M%S).sql.gz"
              pg_dump -h postgresql -U alerthistory alerthistory | gzip > $BACKUP_FILE
              echo "Backup complete: $BACKUP_FILE"

              # Keep last 30 days
              find /backup -name "alerthistory-*.sql.gz" -mtime +30 -delete
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: postgresql-secret
                  key: password
            volumeMounts:
            - name: backup
              mountPath: /backup
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: postgresql-backup-pvc
          restartPolicy: OnFailure
```

#### Manual Backup

```bash
# Create manual backup
kubectl exec -n alertmanager-plus postgresql-0 -- \
  pg_dump -U alerthistory alerthistory | \
  gzip > backup-$(date +%Y%m%d-%H%M%S).sql.gz

# Upload to S3 (optional)
aws s3 cp backup-$(date +%Y%m%d-%H%M%S).sql.gz \
  s3://my-backups/alertmanager-plus/
```

### Recovery Procedures

#### Restore from Backup

```bash
# 1. Stop application
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=0

# 2. Connect to PostgreSQL
kubectl port-forward -n alertmanager-plus postgresql-0 5432:5432 &

# 3. Drop and recreate database
psql -h localhost -U alerthistory -d postgres -c "DROP DATABASE IF EXISTS alerthistory;"
psql -h localhost -U alerthistory -d postgres -c "CREATE DATABASE alerthistory;"

# 4. Restore from backup
gunzip -c backup-20251130-020000.sql.gz | \
  psql -h localhost -U alerthistory -d alerthistory

# 5. Verify restoration
psql -h localhost -U alerthistory -d alerthistory -c \
  "SELECT count(*) FROM alert_history;"

# 6. Restart application
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=3
```

#### Disaster Recovery

**RPO (Recovery Point Objective)**: 24 hours (daily backups)
**RTO (Recovery Time Objective)**: 2 hours

**DR Checklist**:
1. [ ] Latest backup available
2. [ ] PostgreSQL cluster restored
3. [ ] Redis/Valkey cluster restored
4. [ ] Application deployed
5. [ ] Health checks passing
6. [ ] Monitoring configured
7. [ ] DNS/Ingress configured
8. [ ] Smoke tests passed

---

## Scaling Operations

### Horizontal Scaling

#### Manual Scaling

```bash
# Scale up
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=5

# Scale down
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=2

# Verify
kubectl get pods -n alertmanager-plus
kubectl get hpa -n alertmanager-plus
```

#### HPA Configuration

```yaml
# Standard profile - already configured
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80
```

### Vertical Scaling

#### Increase Resources

```bash
# Edit deployment
kubectl edit deployment alert-history -n alertmanager-plus

# Update resources:
#   resources:
#     requests:
#       cpu: 1000m    # was 500m
#       memory: 2Gi   # was 1Gi
#     limits:
#       cpu: 3000m    # was 2000m
#       memory: 6Gi   # was 4Gi

# Restart to apply
kubectl rollout restart deployment/alert-history -n alertmanager-plus
```

### Database Scaling

```bash
# Increase PostgreSQL connections
kubectl edit configmap postgresql-config -n alertmanager-plus

# Update max_connections: 300 (was 250)
# Restart PostgreSQL
kubectl rollout restart statefulset/postgresql -n alertmanager-plus
```

---

## Maintenance Procedures

### Routine Maintenance

#### Weekly Tasks

1. **Log Rotation**
   ```bash
   # Check log sizes
   kubectl exec -n alertmanager-plus deployment/alert-history -- du -sh /var/log/*

   # Rotate if needed (automatic in container)
   ```

2. **Database Vacuum**
   ```bash
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c "VACUUM ANALYZE;"
   ```

3. **Cache Cleanup**
   ```bash
   # Redis/Valkey cleanup (if needed)
   kubectl exec -n alertmanager-plus valkey-0 -- redis-cli FLUSHDB
   ```

#### Monthly Tasks

1. **Database Index Maintenance**
   ```bash
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c "REINDEX DATABASE alerthistory;"
   ```

2. **Storage Cleanup**
   ```bash
   # Delete old alerts (>90 days)
   kubectl exec -n alertmanager-plus deployment/alert-history -- \
     sh -c 'psql -h $DATABASE_HOST -U $DATABASE_USER -d $DATABASE_NAME -c \
     "DELETE FROM alert_history WHERE created_at < NOW() - INTERVAL '\''90 days'\'';"'
   ```

3. **Backup Verification**
   ```bash
   # Test restore of latest backup
   ./test-restore-backup.sh
   ```

### Planned Downtime

```bash
# Maintenance window template

# 1. Announce maintenance
echo "Alertmanager++ maintenance starting at $(date)"

# 2. Enable maintenance mode (optional)
# Set up redirect or maintenance page

# 3. Scale down application
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=0

# 4. Perform maintenance
# - Database upgrades
# - Schema migrations
# - Configuration changes

# 5. Verify changes
kubectl exec -n alertmanager-plus postgresql-0 -- psql -U alerthistory -d alerthistory -c "SELECT version();"

# 6. Restore application
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=3

# 7. Verify health
kubectl get pods -n alertmanager-plus -w
curl http://localhost:8080/health

# 8. Announce completion
echo "Alertmanager++ maintenance complete at $(date)"
```

---

## Performance Tuning

### Application Tuning

1. **Increase Cache Size**
   ```yaml
   # values.yaml
   cache:
     enabled: true
     size: 5000  # Increased from 1000
     ttl: 3600   # 1 hour
   ```

2. **Optimize Database Pool**
   ```yaml
   env:
     - name: DB_POOL_SIZE
       value: "30"  # Per pod
     - name: DB_POOL_MAX_IDLE
       value: "10"
     - name: DB_POOL_MAX_LIFETIME
       value: "1h"
   ```

3. **Enable Compression**
   ```yaml
   env:
     - name: HTTP_COMPRESSION
       value: "true"
   ```

### Database Tuning

```sql
-- PostgreSQL tuning parameters
ALTER SYSTEM SET shared_buffers = '1GB';
ALTER SYSTEM SET effective_cache_size = '3GB';
ALTER SYSTEM SET maintenance_work_mem = '256MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;
ALTER SYSTEM SET work_mem = '16MB';
ALTER SYSTEM SET min_wal_size = '1GB';
ALTER SYSTEM SET max_wal_size = '4GB';

-- Restart PostgreSQL
SELECT pg_reload_conf();
```

---

## Common Issues & Solutions

| Issue | Symptoms | Solution |
|-------|----------|----------|
| **High latency** | Slow API responses | Check DB queries, increase cache, scale horizontally |
| **Memory leaks** | OOM kills, increasing memory | Restart pods, investigate logs, update version |
| **Connection exhaustion** | "too many connections" errors | Scale down replicas, increase max_connections |
| **Cache misses** | Low hit rate | Increase TTL, increase cache size |
| **Disk full** | PostgreSQL errors | Clean old alerts, increase PVC size |
| **HPA not scaling** | High CPU, no scaling | Check metrics-server, verify HPA config |

---

## Additional Resources

- **Deployment Guide**: `/docs/deployment/DEPLOYMENT_GUIDE.md`
- **API Documentation**: `/docs/api/openapi.yaml`
- **Troubleshooting**: `/docs/operations/TROUBLESHOOTING.md` (TN-119)
- **Architecture**: `/docs/architecture/ARCHITECTURE.md` (TN-120)
- **GitHub Issues**: https://github.com/ipiton/alert-history-service/issues

---

**Need Help?**

- 🐛 [Troubleshooting Guide](./TROUBLESHOOTING.md)
- 💬 [GitHub Discussions](https://github.com/ipiton/alert-history-service/discussions)
- 🚨 [Report Issues](https://github.com/ipiton/alert-history-service/issues)
