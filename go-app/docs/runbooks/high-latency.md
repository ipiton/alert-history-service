# Runbook: High Latency in History API

**Severity**: P1 (High)  
**Last Updated**: 2025-11-16

---

## Symptoms

- p95 latency > 10ms
- p99 latency > 50ms
- User complaints about slow responses
- Alert: `HistoryAPIHighLatency`

---

## Investigation Steps

### 1. Check Metrics

```promql
# Current p95 latency
histogram_quantile(0.95, rate(alert_history_api_history_http_request_duration_seconds_bucket[5m]))

# Current p99 latency
histogram_quantile(0.99, rate(alert_history_api_history_http_request_duration_seconds_bucket[5m]))

# Cache hit rate
rate(alert_history_api_history_cache_hits_total[5m]) / 
(rate(alert_history_api_history_cache_hits_total[5m]) + rate(alert_history_api_history_cache_misses_total[5m]))

# Query duration
histogram_quantile(0.95, rate(alert_history_api_history_query_duration_seconds_bucket[5m]))
```

### 2. Check Cache Performance

- **Cache Hit Rate < 80%**: Cache not working effectively
- **Cache Latency High**: Redis connection issues
- **Cache Size High**: Cache eviction happening

### 3. Check Database Performance

```sql
-- Check slow queries
SELECT query, mean_exec_time, calls
FROM pg_stat_statements
WHERE query LIKE '%alerts%'
ORDER BY mean_exec_time DESC
LIMIT 10;

-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE tablename = 'alerts'
ORDER BY idx_scan DESC;
```

### 4. Check Application Logs

```bash
# Check for slow queries
grep "duration_ms" /var/log/alert-history/api.log | awk '$NF > 10'

# Check for cache misses
grep "Cache miss" /var/log/alert-history/api.log | tail -100
```

---

## Resolution Steps

### Step 1: Verify Cache is Working

```bash
# Check cache status
curl http://localhost:9090/metrics | grep alert_history_api_history_cache

# Expected: cache hit rate > 90%
```

**If cache hit rate is low:**
1. Check Redis connectivity
2. Verify cache TTL settings
3. Enable cache warming
4. Increase cache size

### Step 2: Check Database Indexes

```sql
-- Verify indexes exist
\d alerts

-- Check index usage
SELECT * FROM pg_stat_user_indexes WHERE tablename = 'alerts';

-- Reindex if needed
REINDEX INDEX CONCURRENTLY idx_alerts_status_severity_time;
```

**If indexes are missing:**
1. Run migration: `20251116160000_tn63_history_performance_indexes.sql`
2. Analyze table: `ANALYZE alerts;`
3. Verify query plans: `EXPLAIN ANALYZE <query>`

### Step 3: Optimize Queries

**Check for expensive queries:**
- Large time ranges (> 30 days)
- Deep pagination (> 1000 pages)
- Complex filter combinations
- Missing indexes

**Optimize:**
- Limit time ranges to 24-48 hours
- Use appropriate page sizes (50-100)
- Combine filters efficiently
- Verify indexes are used

### Step 4: Scale Resources

**If latency persists:**
1. Increase database connections
2. Add read replicas
3. Scale application instances
4. Increase cache size

---

## Prevention

### Monitoring

- **Alert**: p95 latency > 10ms for 5 minutes
- **Dashboard**: History API Performance
- **Metrics**: Request duration, cache hit rate, query duration

### Best Practices

1. **Cache Configuration**:
   - L1: 10K entries, 5min TTL
   - L2: 1M entries, 1h TTL
   - Cache warming enabled

2. **Database Optimization**:
   - All indexes created
   - Regular VACUUM and ANALYZE
   - Connection pooling configured

3. **Query Optimization**:
   - Use indexed filters
   - Limit time ranges
   - Appropriate pagination

---

## Escalation

If latency remains high after all steps:
1. Contact Database Team (index issues)
2. Contact Infrastructure Team (resource scaling)
3. Contact Development Team (query optimization)

---

## Related Runbooks

- [High Error Rate](./high-error-rate.md)
- [Low Cache Hit Rate](./low-cache-hit-rate.md)
- [Database Performance Issues](./database-performance.md)

