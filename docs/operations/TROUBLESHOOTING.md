# Alertmanager++ Troubleshooting Guide

**Version**: 2.0.0
**Last Updated**: 2025-11-30
**Audience**: SRE, Operations, Support Engineers

---

## 📋 Table of Contents

1. [Diagnostic Tools](#diagnostic-tools)
2. [Pod & Container Issues](#pod--container-issues)
3. [Database Issues](#database-issues)
4. [Cache & Redis Issues](#cache--redis-issues)
5. [Performance Problems](#performance-problems)
6. [Network & Connectivity](#network--connectivity)
7. [Configuration Issues](#configuration-issues)
8. [Log Analysis](#log-analysis)
9. [Quick Reference](#quick-reference)

---

## Diagnostic Tools

### Essential Commands

```bash
# Pod status
kubectl get pods -n alertmanager-plus -o wide

# Pod logs
kubectl logs -n alertmanager-plus <pod-name> --tail=100 --follow

# Pod events
kubectl get events -n alertmanager-plus --sort-by='.lastTimestamp' | tail -20

# Pod description
kubectl describe pod -n alertmanager-plus <pod-name>

# Resource usage
kubectl top pods -n alertmanager-plus

# Execute command in pod
kubectl exec -it -n alertmanager-plus <pod-name> -- /bin/sh

# Port forward for local testing
kubectl port-forward -n alertmanager-plus svc/alert-history 8080:80
```

### Health Check Script

```bash
#!/bin/bash
# comprehensive-health-check.sh

NS="alertmanager-plus"

echo "=== Comprehensive Health Check ==="

# 1. Pod Status
echo -e "\n1. Pod Status:"
kubectl get pods -n $NS -o custom-columns=\
NAME:.metadata.name,STATUS:.status.phase,RESTARTS:.status.containerStatuses[0].restartCount,AGE:.metadata.creationTimestamp

# 2. Health Endpoint
echo -e "\n2. Health Endpoint:"
kubectl run test-health --image=curlimages/curl:latest --rm -i --restart=Never -- \
  curl -s http://alert-history.$NS.svc.cluster.local/health | jq

# 3. Database Connectivity
echo -e "\n3. Database Connectivity:"
kubectl exec -n $NS deployment/alert-history -- \
  sh -c 'timeout 5 nc -zv $DATABASE_HOST $DATABASE_PORT 2>&1'

# 4. Redis Connectivity
echo -e "\n4. Redis Connectivity:"
kubectl exec -n $NS deployment/alert-history -- \
  sh -c 'timeout 5 redis-cli -h $REDIS_ADDR ping 2>&1 || echo "Redis not configured"'

# 5. Recent Errors
echo -e "\n5. Recent Errors (last 5min):"
kubectl logs -n $NS deployment/alert-history --since=5m --tail=500 | \
  grep -i "error\|fatal\|panic" | tail -10

# 6. Metrics Snapshot
echo -e "\n6. Key Metrics:"
kubectl run test-metrics --image=curlimages/curl:latest --rm -i --restart=Never -- \
  curl -s http://alert-history.$NS.svc.cluster.local/metrics | \
  grep -E "alert_history_(business_alerts_received_total|technical_errors_total)"

echo -e "\n=== Health Check Complete ==="
```

---

## Pod & Container Issues

### Issue: Pod Stuck in `Pending` State

**Symptoms**:
- Pod status shows `Pending`
- Events show scheduling failures

**Diagnosis**:
```bash
# Check pod events
kubectl describe pod -n alertmanager-plus <pod-name>

# Check node resources
kubectl top nodes

# Check PVC binding (Lite profile)
kubectl get pvc -n alertmanager-plus
```

**Common Causes & Solutions**:

1. **Insufficient cluster resources**
   ```bash
   # Reduce resource requests
   kubectl edit deployment alert-history -n alertmanager-plus
   # Adjust resources.requests
   ```

2. **PVC not bound** (Lite profile)
   ```bash
   # Check storage class
   kubectl get storageclass

   # Check PVC status
   kubectl get pvc -n alertmanager-plus

   # Create PVC manually if needed
   kubectl apply -f pvc.yaml
   ```

3. **Node selectors/affinity conflicts**
   ```bash
   # Remove node selectors temporarily
   kubectl edit deployment alert-history -n alertmanager-plus
   ```

---

### Issue: Pod Stuck in `CrashLoopBackOff`

**Symptoms**:
- Pod repeatedly restarts
- Status shows `CrashLoopBackOff`

**Diagnosis**:
```bash
# Check recent logs
kubectl logs -n alertmanager-plus <pod-name> --previous

# Check exit code
kubectl describe pod -n alertmanager-plus <pod-name> | grep "Exit Code"
```

**Common Causes & Solutions**:

1. **Configuration errors**
   ```bash
   # Check environment variables
   kubectl get deployment alert-history -n alertmanager-plus -o yaml | grep -A 20 "env:"

   # Verify secrets
   kubectl get secret -n alertmanager-plus postgresql-secret -o yaml

   # Test configuration
   kubectl exec -it -n alertmanager-plus <pod-name> -- env | grep DATABASE
   ```

2. **Database connection failure**
   ```bash
   # Test connectivity
   kubectl exec -n alertmanager-plus <pod-name> -- \
     nc -zv $DATABASE_HOST $DATABASE_PORT

   # Check PostgreSQL status
   kubectl get pods -n alertmanager-plus -l app=postgresql

   # Review database logs
   kubectl logs -n alertmanager-plus postgresql-0 --tail=100
   ```

3. **Missing dependencies**
   ```bash
   # For Standard profile, ensure PostgreSQL is running
   kubectl get statefulset -n alertmanager-plus postgresql

   # For Redis/Valkey
   kubectl get statefulset -n alertmanager-plus valkey
   ```

---

### Issue: Pod Out of Memory (OOMKilled)

**Symptoms**:
- Pod status shows `OOMKilled`
- High memory usage before crash

**Diagnosis**:
```bash
# Check pod events
kubectl describe pod -n alertmanager-plus <pod-name> | grep -A 5 "State:"

# Check memory limits
kubectl get pod -n alertmanager-plus <pod-name> -o yaml | grep -A 5 "resources:"

# Check memory usage before crash
kubectl logs -n alertmanager-plus <pod-name> --previous | grep -i memory
```

**Solutions**:

1. **Increase memory limits**
   ```bash
   kubectl patch deployment alert-history -n alertmanager-plus -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"alert-history","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
   ```

2. **Enable Redis cache to reduce memory** (Standard profile)
   ```bash
   helm upgrade alert-history alertmanager-plus/alert-history \
     --namespace alertmanager-plus \
     --reuse-values \
     --set valkey.enabled=true
   ```

3. **Investigate memory leaks**
   ```bash
   # Enable memory profiling
   kubectl port-forward -n alertmanager-plus <pod-name> 6060:6060 &
   go tool pprof http://localhost:6060/debug/pprof/heap
   ```

---

## Database Issues

### Issue: Connection Pool Exhaustion

**Symptoms**:
- Errors: "too many connections" or "connection pool exhausted"
- High DB connection count

**Diagnosis**:
```bash
# Check active connections
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT count(*), state FROM pg_stat_activity GROUP BY state;"

# Check max_connections setting
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c "SHOW max_connections;"

# Check per-pod connection usage
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT application_name, count(*) FROM pg_stat_activity
   WHERE application_name != '' GROUP BY application_name;"
```

**Solutions**:

1. **Immediate: Kill idle connections**
   ```bash
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c \
     "SELECT pg_terminate_backend(pid) FROM pg_stat_activity
      WHERE state = 'idle' AND state_change < NOW() - INTERVAL '15 minutes';"
   ```

2. **Short-term: Scale down replicas**
   ```bash
   kubectl scale deployment/alert-history -n alertmanager-plus --replicas=2
   ```

3. **Long-term: Increase max_connections**
   ```bash
   # Update ConfigMap
   kubectl edit configmap postgresql-config -n alertmanager-plus
   # Change max_connections to 300

   # Restart PostgreSQL
   kubectl rollout restart statefulset/postgresql -n alertmanager-plus
   ```

4. **Long-term: Optimize connection pool per pod**
   ```bash
   # Reduce connections per pod
   kubectl set env deployment/alert-history -n alertmanager-plus \
     DB_POOL_SIZE=15  # Reduced from 20
   ```

---

### Issue: Slow Database Queries

**Symptoms**:
- API responses slow (>5s)
- Database CPU high
- Timeout errors

**Diagnosis**:
```bash
# Check slow queries
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT query, mean_exec_time, calls
   FROM pg_stat_statements
   ORDER BY mean_exec_time DESC LIMIT 10;"

# Check active queries
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT pid, now() - query_start AS duration, query
   FROM pg_stat_activity
   WHERE state = 'active' ORDER BY duration DESC;"

# Check table sizes
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT schemaname, tablename,
   pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
   FROM pg_tables WHERE schemaname = 'public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"
```

**Solutions**:

1. **Check indexes**
   ```bash
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c "\di+"
   ```

2. **Run VACUUM ANALYZE**
   ```bash
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c "VACUUM ANALYZE;"
   ```

3. **Add missing indexes**
   ```sql
   -- Example: Add index on frequently queried fields
   CREATE INDEX CONCURRENTLY idx_alert_history_severity
     ON alert_history(severity) WHERE status = 'firing';
   ```

4. **Increase PostgreSQL cache**
   ```bash
   kubectl edit configmap postgresql-config -n alertmanager-plus
   # Increase shared_buffers, effective_cache_size
   ```

---

### Issue: Database Disk Full

**Symptoms**:
- Errors: "no space left on device"
- Write operations fail

**Diagnosis**:
```bash
# Check PVC usage
kubectl exec -n alertmanager-plus postgresql-0 -- df -h

# Check table sizes
kubectl exec -n alertmanager-plus postgresql-0 -- \
  psql -U alerthistory -d alerthistory -c \
  "SELECT pg_size_pretty(pg_database_size('alerthistory'));"
```

**Solutions**:

1. **Delete old alerts**
   ```bash
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c \
     "DELETE FROM alert_history WHERE created_at < NOW() - INTERVAL '90 days';"

   # Reclaim space
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c "VACUUM FULL;"
   ```

2. **Increase PVC size**
   ```bash
   # Edit PVC (if storage class supports expansion)
   kubectl edit pvc postgresql-data-postgresql-0 -n alertmanager-plus
   # Increase storage: 100Gi (was 50Gi)

   # Verify expansion
   kubectl get pvc -n alertmanager-plus
   ```

3. **Archive old data**
   ```bash
   # Export old alerts to S3
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     pg_dump -U alerthistory -d alerthistory \
     --table=alert_history \
     --where="created_at < NOW() - INTERVAL '180 days'" | \
     gzip | aws s3 cp - s3://archive/alerts-$(date +%Y%m%d).sql.gz

   # Delete archived alerts
   kubectl exec -n alertmanager-plus postgresql-0 -- \
     psql -U alerthistory -d alerthistory -c \
     "DELETE FROM alert_history WHERE created_at < NOW() - INTERVAL '180 days';"
   ```

---

## Cache & Redis Issues

### Issue: Low Cache Hit Rate

**Symptoms**:
- Cache hit rate < 70%
- High LLM API costs
- Slow classification

**Diagnosis**:
```bash
# Check cache stats
curl http://localhost:8080/api/v2/cache/stats | jq

# Check Redis keys
kubectl exec -n alertmanager-plus valkey-0 -- redis-cli DBSIZE

# Check Redis memory
kubectl exec -n alertmanager-plus valkey-0 -- redis-cli INFO memory
```

**Solutions**:

1. **Increase cache TTL**
   ```bash
   kubectl set env deployment/alert-history -n alertmanager-plus \
     CACHE_TTL=7200  # 2 hours (was 1 hour)
   ```

2. **Increase cache size**
   ```bash
   kubectl set env deployment/alert-history -n alertmanager-plus \
     CACHE_SIZE=5000  # (was 1000)
   ```

3. **Increase Redis maxmemory**
   ```bash
   kubectl edit configmap valkey-config -n alertmanager-plus
   # Increase maxmemory to 512mb

   kubectl rollout restart statefulset/valkey -n alertmanager-plus
   ```

---

### Issue: Redis Connection Failures

**Symptoms**:
- Errors: "connection refused" or "i/o timeout"
- Fallback to L1 cache only
- Classification slower

**Diagnosis**:
```bash
# Check Redis pod status
kubectl get pods -n alertmanager-plus -l app=valkey

# Test connectivity from app pod
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  redis-cli -h valkey -p 6379 ping

# Check Redis logs
kubectl logs -n alertmanager-plus valkey-0 --tail=100
```

**Solutions**:

1. **Restart Redis**
   ```bash
   kubectl rollout restart statefulset/valkey -n alertmanager-plus
   ```

2. **Verify service**
   ```bash
   kubectl get svc -n alertmanager-plus valkey

   # Ensure service exists and has endpoints
   kubectl get endpoints -n alertmanager-plus valkey
   ```

3. **Check network policies**
   ```bash
   kubectl get networkpolicy -n alertmanager-plus

   # Ensure policy allows traffic from app to Redis
   ```

---

## Performance Problems

### Issue: High API Latency

**Symptoms**:
- API responses slow (>5s)
- p95 latency high

**Diagnosis**:
```bash
# Check metrics
curl http://localhost:8080/metrics | grep alert_processing_duration

# Check slow endpoints
kubectl logs -n alertmanager-plus deployment/alert-history --tail=500 | \
  grep "duration" | sort -k8 -rn | head -20

# Check resource usage
kubectl top pods -n alertmanager-plus
```

**Root Causes & Solutions**:

1. **Database slow queries** → See [Slow Database Queries](#issue-slow-database-queries)

2. **Cache misses**
   ```bash
   # Check cache hit rate
   curl http://localhost:8080/api/v2/cache/stats

   # Warm up cache
   curl -X POST http://localhost:8080/api/v2/cache/warmup
   ```

3. **LLM timeout**
   ```bash
   # Check LLM latency
   kubectl logs -n alertmanager-plus deployment/alert-history --tail=100 | \
     grep "llm_classification_duration"

   # Increase LLM timeout
   kubectl set env deployment/alert-history -n alertmanager-plus \
     LLM_TIMEOUT=15s  # (was 10s)
   ```

4. **High CPU usage**
   ```bash
   # Check CPU
   kubectl top pods -n alertmanager-plus

   # Scale horizontally
   kubectl scale deployment/alert-history -n alertmanager-plus --replicas=5
   ```

---

### Issue: High Memory Usage

**Symptoms**:
- Memory usage > 85%
- OOMKill events
- Slow performance

**Diagnosis**:
```bash
# Check memory usage
kubectl top pods -n alertmanager-plus

# Check for memory leaks
kubectl logs -n alertmanager-plus <pod-name> --tail=500 | \
  grep -i "memory\|heap\|gc"

# Memory profiling
kubectl port-forward -n alertmanager-plus <pod-name> 6060:6060 &
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof -http=:8081 heap.prof
```

**Solutions**:

1. **Increase memory limits**
   ```bash
   kubectl patch deployment alert-history -n alertmanager-plus -p \
     '{"spec":{"template":{"spec":{"containers":[{"name":"alert-history","resources":{"limits":{"memory":"4Gi"}}}]}}}}'
   ```

2. **Enable Redis cache** (reduces in-memory caching)
   ```bash
   helm upgrade alert-history alertmanager-plus/alert-history \
     --namespace alertmanager-plus \
     --reuse-values \
     --set valkey.enabled=true
   ```

3. **Reduce cache size**
   ```bash
   kubectl set env deployment/alert-history -n alertmanager-plus \
     CACHE_SIZE=1000  # Reduced from 5000
   ```

4. **Force garbage collection**
   ```bash
   kubectl exec -n alertmanager-plus <pod-name> -- \
     curl -X POST http://localhost:6060/debug/freeOSMemory
   ```

---

### Issue: Slow Alert Processing

**Symptoms**:
- Alerts delayed in history
- High processing latency
- Queue backlog

**Diagnosis**:
```bash
# Check queue stats
curl http://localhost:8080/api/v2/publishing/queue/stats | jq

# Check processing metrics
curl http://localhost:8080/metrics | \
  grep alert_history_business_alert_processing_duration_seconds

# Check goroutine count (potential deadlock)
curl http://localhost:6060/debug/pprof/goroutine?debug=1 | grep "goroutine profile:"
```

**Solutions**:

1. **Scale horizontally**
   ```bash
   kubectl scale deployment/alert-history -n alertmanager-plus --replicas=5
   ```

2. **Increase worker pool**
   ```bash
   kubectl set env deployment/alert-history -n alertmanager-plus \
     WORKER_POOL_SIZE=20  # (was 10)
   ```

3. **Optimize database queries** → See [Database Issues](#database-issues)

---

## Network & Connectivity

### Issue: Cannot Reach External Services (LLM, Publishing Targets)

**Symptoms**:
- Publishing failures
- LLM classification failures
- Timeout errors

**Diagnosis**:
```bash
# Test external connectivity from pod
kubectl exec -it -n alertmanager-plus deployment/alert-history -- /bin/sh

# Inside pod:
curl -I https://api.openai.com/v1/models
curl -I https://hooks.slack.com/services/TEST
nc -zv api.pagerduty.com 443

# Check network policies
kubectl get networkpolicy -n alertmanager-plus

# Check DNS resolution
kubectl exec -n alertmanager-plus deployment/alert-history -- \
  nslookup api.openai.com
```

**Solutions**:

1. **Allow egress traffic**
   ```yaml
   # network-policy.yaml
   apiVersion: networking.k8s.io/v1
   kind: NetworkPolicy
   metadata:
     name: allow-egress
     namespace: alertmanager-plus
   spec:
     podSelector:
       matchLabels:
         app: alert-history
     policyTypes:
     - Egress
     egress:
     - to:
       - namespaceSelector: {}
     - to:
       - podSelector: {}
     - ports:
       - protocol: TCP
         port: 443  # HTTPS
       - protocol: TCP
         port: 53   # DNS
       - protocol: UDP
         port: 53   # DNS
   ```

2. **Configure HTTP proxy** (if required)
   ```bash
   kubectl set env deployment/alert-history -n alertmanager-plus \
     HTTP_PROXY=http://proxy.example.com:3128 \
     HTTPS_PROXY=http://proxy.example.com:3128 \
     NO_PROXY=localhost,127.0.0.1,.svc,.cluster.local
   ```

3. **Verify certificates** (if TLS errors)
   ```bash
   kubectl exec -n alertmanager-plus deployment/alert-history -- \
     openssl s_client -connect api.openai.com:443 -showcerts
   ```

---

## Configuration Issues

### Issue: Invalid Configuration

**Symptoms**:
- Pod fails validation
- Configuration errors in logs

**Diagnosis**:
```bash
# Check configuration
kubectl get configmap -n alertmanager-plus alert-history-config -o yaml

# Validate configuration locally
helm template alert-history ./helm/alert-history \
  --values production-values.yaml \
  --debug

# Check environment variables
kubectl exec -n alertmanager-plus deployment/alert-history -- env | sort
```

**Solutions**:

1. **Validate Helm values**
   ```bash
   helm lint ./helm/alert-history --values production-values.yaml
   ```

2. **Use default configuration**
   ```bash
   helm upgrade alert-history alertmanager-plus/alert-history \
     --namespace alertmanager-plus \
     --reset-values
   ```

3. **Rollback configuration**
   ```bash
   kubectl rollout undo deployment/alert-history -n alertmanager-plus
   ```

---

### Issue: Secrets Not Found

**Symptoms**:
- Errors: "secret not found"
- Missing environment variables

**Diagnosis**:
```bash
# List secrets
kubectl get secrets -n alertmanager-plus

# Check secret content (base64 encoded)
kubectl get secret postgresql-secret -n alertmanager-plus -o yaml

# Verify secret mounted in pod
kubectl describe pod -n alertmanager-plus <pod-name> | grep -A 10 "Mounts:"
```

**Solutions**:

1. **Create missing secret**
   ```bash
   kubectl create secret generic postgresql-secret \
     --from-literal=username=alerthistory \
     --from-literal=password='STRONG_PASSWORD' \
     -n alertmanager-plus
   ```

2. **Update secret**
   ```bash
   kubectl delete secret postgresql-secret -n alertmanager-plus
   kubectl create secret generic postgresql-secret \
     --from-literal=username=alerthistory \
     --from-literal=password='NEW_PASSWORD' \
     -n alertmanager-plus

   # Restart pods to pick up new secret
   kubectl rollout restart deployment/alert-history -n alertmanager-plus
   ```

---

## Log Analysis

### Useful Log Queries

```bash
# 1. Errors in last hour
kubectl logs -n alertmanager-plus deployment/alert-history \
  --since=1h --tail=10000 | grep -i "error\|fatal\|panic"

# 2. Slow operations (>1s)
kubectl logs -n alertmanager-plus deployment/alert-history \
  --since=1h --tail=10000 | grep "duration.*[1-9][0-9]\{3,\}ms"

# 3. Database errors
kubectl logs -n alertmanager-plus deployment/alert-history \
  --since=1h --tail=10000 | grep -i "database\|postgres\|sql"

# 4. LLM classification errors
kubectl logs -n alertmanager-plus deployment/alert-history \
  --since=1h --tail=10000 | grep -i "llm\|classification"

# 5. Publishing failures
kubectl logs -n alertmanager-plus deployment/alert-history \
  --since=1h --tail=10000 | grep -i "publish\|slack\|pagerduty\|rootly"

# 6. Top error types
kubectl logs -n alertmanager-plus deployment/alert-history \
  --since=6h --tail=50000 | grep "ERROR" | \
  cut -d' ' -f5- | sort | uniq -c | sort -rn | head -20

# 7. Request IDs for tracing
kubectl logs -n alertmanager-plus deployment/alert-history \
  --tail=1000 | grep "request_id=<UUID>"
```

### Log Levels

| Level | Usage | Example |
|-------|-------|---------|
| **DEBUG** | Development only | Detailed execution traces |
| **INFO** | Normal operations | "Alert received", "Processing complete" |
| **WARN** | Recoverable issues | "Cache miss", "Retry attempt 2/3" |
| **ERROR** | Errors requiring attention | "Database connection failed" |
| **FATAL** | Critical failures | "Cannot start service" |

### Change Log Level

```bash
# Temporarily increase verbosity
kubectl set env deployment/alert-history -n alertmanager-plus \
  LOG_LEVEL=debug

# Restore to info
kubectl set env deployment/alert-history -n alertmanager-plus \
  LOG_LEVEL=info
```

---

## Quick Reference

### Emergency Commands

```bash
# Restart all pods
kubectl rollout restart deployment/alert-history -n alertmanager-plus

# Scale to 0 and back (last resort)
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=0
sleep 10
kubectl scale deployment/alert-history -n alertmanager-plus --replicas=3

# Force delete stuck pod
kubectl delete pod -n alertmanager-plus <pod-name> --force --grace-period=0

# Rollback to previous version
kubectl rollout undo deployment/alert-history -n alertmanager-plus
```

### Useful Queries

```sql
-- PostgreSQL Diagnostics

-- Active connections by user
SELECT usename, count(*) FROM pg_stat_activity GROUP BY usename;

-- Long-running queries
SELECT pid, now() - query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND query_start < NOW() - INTERVAL '30 seconds'
ORDER BY duration DESC;

-- Table bloat
SELECT tablename,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS total_size,
       pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) AS table_size,
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) AS index_size
FROM pg_tables WHERE schemaname = 'public' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;

-- Unused indexes
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0 AND indexrelname NOT LIKE '%_pkey';
```

### Escalation Decision Tree

```
Issue Detected
    │
    ├─ Service Down? → P0 (Page on-call immediately)
    │
    ├─ High Error Rate (>5%)? → P1 (Notify team, investigate within 15min)
    │
    ├─ Performance Degraded (>2x latency)? → P2 (Create ticket, investigate within 1h)
    │
    └─ Minor issue? → P3 (Add to backlog)
```

---

## Additional Resources

- **Operations Runbook**: [RUNBOOK.md](./RUNBOOK.md)
- **Deployment Guide**: [../deployment/DEPLOYMENT_GUIDE.md](../deployment/DEPLOYMENT_GUIDE.md)
- **API Documentation**: [../api/openapi.yaml](../api/openapi.yaml)
- **Architecture**: [../architecture/ARCHITECTURE.md](../architecture/ARCHITECTURE.md)
- **GitHub Issues**: https://github.com/ipiton/alert-history-service/issues

---

**Still Having Issues?**

1. Search existing [GitHub Issues](https://github.com/ipiton/alert-history-service/issues)
2. Check [GitHub Discussions](https://github.com/ipiton/alert-history-service/discussions)
3. Review [Operations Runbook](./RUNBOOK.md) for incident playbooks
4. Escalate to on-call engineer (see [RUNBOOK.md](./RUNBOOK.md#escalation-contacts))
