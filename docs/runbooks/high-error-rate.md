# Runbook: High Error Rate on Proxy Webhook

**Alert Name**: `ProxyWebhookHighErrorRate`
**Severity**: 🔴 Critical (P0)
**Service**: Alert History
**Component**: Proxy Webhook
**Last Updated**: 2025-11-16

---

## Alert Definition

```yaml
alert: ProxyWebhookHighErrorRate
expr: |
  rate(alert_history_proxy_http_errors_total[5m]) > 10
for: 5m
```

**Triggers when**: Error rate exceeds 10 errors/second for 5 consecutive minutes

---

## Symptoms

### What You'll Observe

1. **Prometheus Alert Firing**
   ```
   Alert: ProxyWebhookHighErrorRate
   Severity: critical
   Current Value: 15.3 errors/s
   Threshold: 10 errors/s
   ```

2. **PagerDuty Incident** (if configured)
   - Incident created automatically
   - Assigned to on-call engineer

3. **Grafana Dashboard** shows spike
   - Red line in "Error Rate" panel
   - Error count increasing rapidly

4. **User Reports** (likely)
   - Webhooks failing
   - Alerts not being processed
   - Customers reporting issues

---

## Impact

### User Impact

- **High**: Alerts are failing to process
- **Data Loss Risk**: Alerts may be dropped by Alertmanager
- **SLA Breach**: If sustained > 15 minutes

### System Impact

- **Processing Pipeline**: Partially or fully blocked
- **Downstream Systems**: May not receive alerts (Rootly, PagerDuty, Slack)
- **Database**: May be filling with error logs

---

## Triage (First 2 Minutes)

### 1. Check Current Error Rate

```bash
# Query current error rate
curl -s 'http://prometheus:9090/api/v1/query?query=rate(alert_history_proxy_http_errors_total[1m])' | jq '.data.result[0].value[1]'

# Expected: Should be > 10
```

### 2. Identify Error Types

```bash
# Get error breakdown by type
curl -s 'http://prometheus:9090/api/v1/query?query=sum by (error_type) (rate(alert_history_proxy_http_errors_total[5m]))' | jq -r '.data.result[] | "\(.metric.error_type): \(.value[1])"'

# Common types:
# - validation_error (400)
# - authentication_error (401)
# - rate_limit_error (429)
# - internal_error (500)
# - timeout_error (504)
```

### 3. Check Service Health

```bash
# Health check
curl -s https://api.alerthistory.io/v1/health | jq

# Expected: {"status": "healthy"}
# If unhealthy: {"status": "unhealthy", "errors": [...]}
```

### 4. Check Recent Logs

```bash
# Last 100 error logs
kubectl logs -l app=alert-history --tail=100 | grep -i error

# Look for patterns:
# - Repeated errors from same source
# - Specific error messages
# - Stack traces
```

---

## Diagnosis

### Scenario 1: Validation Errors (400)

**Symptoms**:
```bash
# High rate of validation errors
error_type=validation_error: 12.5/s
```

**Root Cause**: Invalid payloads from Alertmanager

**Investigation**:

```bash
# Check recent validation errors
kubectl logs -l app=alert-history --tail=500 | grep "validation failed"

# Example output:
# ERROR Request validation failed error="alerts[0].labels: required field missing"
# ERROR Request validation failed error="alerts[0].startsAt: required field missing"
```

**Common Causes**:
- Alertmanager misconfiguration
- Alertmanager version incompatibility
- Custom webhook format (not Alertmanager standard)

**Resolution**: See "Scenario 1 Resolution" below

---

### Scenario 2: Authentication Errors (401)

**Symptoms**:
```bash
# High rate of auth errors
error_type=authentication_error: 15.2/s
```

**Root Cause**: Invalid or expired API keys

**Investigation**:

```bash
# Check auth failures by source IP
kubectl logs -l app=alert-history --tail=500 | grep "authentication failed" | awk '{print $NF}' | sort | uniq -c | sort -rn

# Example output:
# 150 remote_addr=10.0.1.45
#  30 remote_addr=10.0.2.67
```

**Common Causes**:
- API key rotation (old key still in use)
- Leaked API key (revoked)
- Misconfigured Alertmanager (wrong key)

**Resolution**: See "Scenario 2 Resolution" below

---

### Scenario 3: Rate Limit Errors (429)

**Symptoms**:
```bash
# High rate of rate limit errors
error_type=rate_limit_error: 11.8/s
```

**Root Cause**: Too many requests from single source

**Investigation**:

```bash
# Check rate limit violations by IP
curl -s 'http://prometheus:9090/api/v1/query?query=sum by (remote_addr) (rate(alert_history_proxy_http_errors_total{error_type="rate_limit_error"}[5m]))' | jq

# Check current rate per IP
kubectl logs -l app=alert-history --tail=1000 | grep "rate limit exceeded" | awk '{print $8}' | sort | uniq -c
```

**Common Causes**:
- Alert storm (many alerts firing)
- Misconfigured group_interval in Alertmanager
- Single Alertmanager sending too much
- DDoS attack (rare)

**Resolution**: See "Scenario 3 Resolution" below

---

### Scenario 4: Internal Errors (500)

**Symptoms**:
```bash
# High rate of internal errors
error_type=internal_error: 8.5/s
```

**Root Cause**: Application bugs or infrastructure issues

**Investigation**:

```bash
# Check for panic/crash logs
kubectl logs -l app=alert-history --tail=500 | grep -E "(panic|fatal|FATAL)"

# Check resource usage
kubectl top pods -l app=alert-history

# Check if pods are restarting
kubectl get pods -l app=alert-history -w
```

**Common Causes**:
- Database connection issues
- Redis unavailable (classification cache)
- LLM service down
- Memory exhaustion
- Bug in code (panic/nil pointer)

**Resolution**: See "Scenario 4 Resolution" below

---

### Scenario 5: Timeout Errors (504)

**Symptoms**:
```bash
# High rate of timeout errors
error_type=timeout_error: 7.3/s
```

**Root Cause**: Requests taking > 30s (default timeout)

**Investigation**:

```bash
# Check p95 latency
curl -s 'http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95, rate(alert_history_proxy_http_request_duration_seconds_bucket[5m]))' | jq '.data.result[0].value[1]'

# Check slow pipelines
curl -s 'http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95, rate(alert_history_proxy_classification_duration_seconds_bucket[5m]))' | jq
```

**Common Causes**:
- LLM service slow/degraded
- Low cache hit rate (cold cache)
- Publishing targets slow
- Database queries slow

**Resolution**: See "Scenario 5 Resolution" below

---

## Resolution Steps

### Scenario 1 Resolution: Validation Errors

**Fix**: Update Alertmanager configuration

1. **Check Alertmanager config**:
   ```bash
   kubectl get configmap alertmanager-config -o yaml
   ```

2. **Verify webhook format**:
   ```yaml
   receivers:
     - name: 'alert-history'
       webhook_configs:
         - url: 'https://api.alerthistory.io/v1/webhook/proxy'
           send_resolved: true  # Important!
           http_config:
             bearer_token: 'ah_your_api_key'
   ```

3. **Test with valid payload**:
   ```bash
   # Send test alert
   amtool alert add test_alert severity=critical --alertmanager.url=http://alertmanager:9093
   ```

4. **Monitor error rate** (should drop):
   ```bash
   watch -n 1 'curl -s "http://prometheus:9090/api/v1/query?query=rate(alert_history_proxy_http_errors_total[1m])" | jq ".data.result[0].value[1]"'
   ```

---

### Scenario 2 Resolution: Authentication Errors

**Fix**: Update or rotate API keys

1. **Identify affected Alertmanagers**:
   ```bash
   # Get IPs with auth failures
   kubectl logs -l app=alert-history --tail=1000 | grep "authentication failed" | awk '{print $NF}' | sort -u
   ```

2. **Generate new API key** (if needed):
   ```bash
   # Create new API key
   kubectl create secret generic alertmanager-api-key \
     --from-literal=api-key=$(openssl rand -hex 32) \
     --dry-run=client -o yaml | kubectl apply -f -
   ```

3. **Update Alertmanager secrets**:
   ```bash
   # Update secret
   kubectl patch secret alertmanager-secrets \
     -p '{"data":{"api-key":"'"$(echo -n 'ah_new_key_here' | base64)"'"}}'

   # Restart Alertmanager
   kubectl rollout restart deployment/alertmanager
   ```

4. **Verify authentication working**:
   ```bash
   # Test with new key
   curl -H "X-API-Key: ah_new_key_here" https://api.alerthistory.io/v1/health
   ```

---

### Scenario 3 Resolution: Rate Limiting

**Fix**: Adjust rate limits or optimize Alertmanager

1. **Identify top sources**:
   ```bash
   # Get top IPs by request volume
   kubectl logs -l app=alert-history | awk '{print $5}' | sort | uniq -c | sort -rn | head -10
   ```

2. **Temporary fix** - Increase rate limit:
   ```bash
   # Edit config
   kubectl edit configmap alert-history-config

   # Update:
   rate_limiting:
     per_ip_limit: 200  # was: 100
     global_limit: 2000  # was: 1000

   # Restart
   kubectl rollout restart deployment/alert-history
   ```

3. **Long-term fix** - Optimize Alertmanager:
   ```yaml
   # alertmanager.yml
   route:
     group_wait: 30s        # was: 10s (batch alerts)
     group_interval: 5m     # was: 1m (reduce frequency)
     repeat_interval: 4h    # was: 1h (reduce repeats)
   ```

4. **Monitor new rate**:
   ```bash
   watch -n 1 'curl -s "http://prometheus:9090/api/v1/query?query=rate(alert_history_proxy_http_requests_total[1m])" | jq ".data.result[0].value[1]"'
   ```

---

### Scenario 4 Resolution: Internal Errors

**Fix**: Investigate and fix root cause

1. **Check pod status**:
   ```bash
   kubectl get pods -l app=alert-history

   # If CrashLoopBackOff:
   kubectl logs -l app=alert-history --previous
   ```

2. **Check dependencies**:
   ```bash
   # PostgreSQL
   kubectl exec -it postgres-0 -- psql -U postgres -c "SELECT 1"

   # Redis
   kubectl exec -it redis-0 -- redis-cli ping

   # LLM Service
   curl -s https://llm-service/health
   ```

3. **Scale up if resource issue**:
   ```bash
   # Increase replicas
   kubectl scale deployment alert-history --replicas=5

   # Increase resources
   kubectl edit deployment alert-history
   # Update:
   resources:
     requests:
       memory: 512Mi  # was: 256Mi
       cpu: 500m      # was: 250m
   ```

4. **Rollback if recent deployment**:
   ```bash
   # Check recent deployments
   kubectl rollout history deployment/alert-history

   # Rollback to previous
   kubectl rollout undo deployment/alert-history
   ```

---

### Scenario 5 Resolution: Timeout Errors

**Fix**: Optimize slow pipelines or increase timeout

1. **Identify slow pipeline**:
   ```bash
   # Classification
   curl -s 'http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95, rate(alert_history_proxy_classification_duration_seconds_bucket[5m]))'

   # Publishing
   curl -s 'http://prometheus:9090/api/v1/query?query=histogram_quantile(0.95, rate(alert_history_proxy_publishing_duration_seconds_bucket[5m]))'
   ```

2. **Check LLM service** (if classification slow):
   ```bash
   # LLM health
   curl -s https://llm-service/health

   # LLM latency
   curl -s 'http://prometheus:9090/api/v1/query?query=llm_request_duration_seconds'
   ```

3. **Check cache hit rate**:
   ```bash
   # Should be > 90%
   curl -s 'http://prometheus:9090/api/v1/query?query=rate(alert_history_proxy_classification_duration_seconds_bucket{cached="true"}[5m]) / rate(alert_history_proxy_classification_duration_seconds_bucket[5m])'
   ```

4. **Temporary fix** - Increase timeout:
   ```yaml
   # Edit config
   proxy:
     request_timeout: 60s  # was: 30s
     classification_timeout: 10s  # was: 5s
   ```

---

## Prevention

### Long-Term Solutions

1. **Monitoring & Alerting**
   - ✅ Alert configured (ProxyWebhookHighErrorRate)
   - ⚠️ Set up alerts for precursors:
     - Warn at 5 errors/s
     - Page at 10 errors/s

2. **Capacity Planning**
   - Monitor trends
   - Scale before hitting limits
   - Test with load testing (k6)

3. **Improved Validation**
   - Better error messages
   - Webhook validator tool
   - Documentation improvements

4. **Rate Limit Tuning**
   - Per-customer limits
   - Dynamic rate limiting
   - Burst allowance

5. **Reliability Improvements**
   - Add retries in Alertmanager
   - Dead letter queue
   - Circuit breaker for dependencies

---

## Related Information

### Metrics to Watch

```promql
# Error rate
rate(alert_history_proxy_http_errors_total[5m])

# Error rate by type
sum by (error_type) (rate(alert_history_proxy_http_errors_total[5m]))

# Success rate
rate(alert_history_proxy_alerts_processed_total{result="success"}[5m]) / rate(alert_history_proxy_alerts_processed_total[5m])

# Latency
histogram_quantile(0.95, rate(alert_history_proxy_http_request_duration_seconds_bucket[5m]))
```

### Dashboards

- **Main Dashboard**: https://grafana.company.com/d/proxy-webhook
- **Alertmanager**: https://grafana.company.com/d/alertmanager
- **Dependencies**: https://grafana.company.com/d/dependencies

### Logs

```bash
# Tail logs
kubectl logs -f -l app=alert-history --tail=100

# Search logs (last hour)
kubectl logs -l app=alert-history --since=1h | grep -i error

# Specific error type
kubectl logs -l app=alert-history | grep "validation failed"
```

### Related Runbooks

- [High Latency](./high-latency.md)
- [Publishing Failures](./publishing-failures.md)
- [Classification Issues](./classification-issues.md)

---

## Communication Template

### Incident Update (Slack/Status Page)

```
🔴 INCIDENT: High Error Rate on Alert History Proxy Webhook

Status: Investigating
Start Time: 2025-11-16 10:35 UTC
Duration: 8 minutes

Impact:
- Alerts may fail to process
- Webhooks returning errors
- Affecting ~15% of requests

Root Cause:
- [TBD - investigating]

Current Actions:
- Identified error type: [validation_error/auth_error/etc]
- Investigating source of errors
- ETA for fix: 15 minutes

Updates:
- Will provide update in 10 minutes
- Follow: https://status.alerthistory.io/incidents/12345
```

---

## Escalation

### When to Escalate

- Error rate remains > 10/s after 15 minutes
- Unable to identify root cause within 10 minutes
- Requires code changes or infrastructure changes
- Multiple services affected

### Who to Contact

1. **On-Call Engineer** (primary)
   - PagerDuty: Alert History On-Call
   - Slack: @alert-history-oncall

2. **Team Lead** (if needed)
   - Slack: @alert-history-lead
   - Email: lead@alerthistory.io

3. **Senior Engineer** (escalation)
   - PagerDuty: Senior On-Call
   - Phone: +1-XXX-XXX-XXXX

---

## Checklist

### Investigation

- [ ] Check error rate (current value)
- [ ] Identify error type(s)
- [ ] Check service health
- [ ] Review recent logs
- [ ] Check recent deployments
- [ ] Verify dependencies health

### Resolution

- [ ] Apply appropriate fix for scenario
- [ ] Verify error rate dropping
- [ ] Monitor for 10 minutes
- [ ] Update incident status
- [ ] Document root cause

### Post-Incident

- [ ] Write postmortem
- [ ] Identify improvements
- [ ] Create follow-up tickets
- [ ] Update runbook
- [ ] Share learnings with team

---

**Questions?** Contact: team@alerthistory.io
**Last Updated**: 2025-11-16
