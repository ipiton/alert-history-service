# Runbook: High Error Rate in History API

**Severity**: P1 (High)  
**Last Updated**: 2025-11-16

---

## Symptoms

- Error rate > 0.1 errors/sec
- HTTP 4xx/5xx responses increasing
- User complaints about errors
- Alert: `HistoryAPIHighErrorRate`

---

## Investigation Steps

### 1. Check Error Types

```promql
# Error rate by type
rate(alert_history_api_history_http_errors_total[5m])

# Error rate by endpoint
sum by (endpoint) (rate(alert_history_api_history_http_errors_total[5m]))
```

### 2. Check Query Errors

```promql
# Query errors
rate(alert_history_api_history_query_errors_total[5m])

# Filter errors
rate(alert_history_api_history_filters_errors_total[5m])
```

### 3. Check Application Logs

```bash
# Check for errors
grep "ERROR" /var/log/alert-history/api.log | tail -100

# Check for specific error types
grep "VALIDATION_ERROR" /var/log/alert-history/api.log | tail -50
grep "INTERNAL_ERROR" /var/log/alert-history/api.log | tail -50
```

### 4. Check Database Connectivity

```bash
# Check database connection pool
curl http://localhost:9090/metrics | grep pgx_pool

# Test database connection
psql -h localhost -U alert_history -d alert_history -c "SELECT 1"
```

---

## Resolution Steps

### Step 1: Identify Error Type

**Validation Errors (400)**:
- Check input validation rules
- Review recent API changes
- Check for malformed requests

**Authentication Errors (401)**:
- Verify API keys are valid
- Check authentication middleware
- Review rate limiting

**Database Errors (500)**:
- Check database connectivity
- Review query errors
- Check connection pool

### Step 2: Fix Validation Errors

**Common causes:**
- Invalid query parameters
- Malformed time formats
- Exceeding limits (page, per_page)

**Fix:**
- Review error messages
- Update client code
- Adjust validation rules if needed

### Step 3: Fix Database Errors

**Common causes:**
- Connection pool exhausted
- Query timeout
- Database overload

**Fix:**
1. Check connection pool size
2. Increase query timeout
3. Optimize slow queries
4. Scale database resources

### Step 4: Fix Authentication Errors

**Common causes:**
- Invalid API keys
- Rate limiting too strict
- Authentication middleware issues

**Fix:**
1. Verify API keys
2. Adjust rate limits
3. Check authentication config

---

## Prevention

### Monitoring

- **Alert**: Error rate > 0.1 errors/sec for 5 minutes
- **Dashboard**: History API Errors
- **Metrics**: Error rate by type, endpoint, status code

### Best Practices

1. **Input Validation**:
   - Validate all parameters
   - Provide clear error messages
   - Log validation failures

2. **Error Handling**:
   - Graceful degradation
   - Retry logic for transient errors
   - Proper error responses

3. **Database**:
   - Connection pooling
   - Query timeouts
   - Error handling

---

## Escalation

If error rate remains high:
1. Contact Development Team (code issues)
2. Contact Database Team (database issues)
3. Contact Infrastructure Team (infrastructure issues)

---

## Related Runbooks

- [High Latency](./high-latency.md)
- [Database Performance Issues](./database-performance.md)
- [Authentication Failures](./authentication-failures.md)

