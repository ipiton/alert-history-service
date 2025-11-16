# TN-63 History Endpoint Performance Optimization Guide

## Overview

This guide documents performance optimizations implemented for the GET /history endpoint to achieve p95 latency < 10ms target.

## Database Optimizations

### Indexes Created (8 new indexes)

1. **idx_alerts_status_severity_time** - Composite index for status + severity + time
   - Use case: `WHERE status = 'firing' AND labels->>'severity' = 'critical' AND starts_at >= ?`
   - Expected improvement: 50-100x faster

2. **idx_alerts_namespace_status_time** - Composite index for namespace + status + time
   - Use case: `WHERE labels->>'namespace' = 'production' AND status = 'firing' AND starts_at >= ?`
   - Expected improvement: 30-50x faster

3. **idx_alerts_ends_at** - Index for duration calculations
   - Use case: `WHERE ends_at IS NOT NULL AND EXTRACT(EPOCH FROM (ends_at - starts_at)) >= ?`
   - Expected improvement: 20-30x faster

4. **idx_alerts_generator_url** - Index for generator URL filtering
   - Use case: `WHERE generator_url = ?`
   - Expected improvement: 10-20x faster

5. **idx_alerts_resolved** - Partial index for resolved alerts
   - Use case: `WHERE status = 'resolved' AND ends_at IS NOT NULL`
   - Expected improvement: 20-30x faster

6. **idx_alerts_fingerprint_timeline** - Composite index for timeline queries
   - Use case: `WHERE fingerprint = ? ORDER BY starts_at DESC`
   - Expected improvement: 10-20x faster

7. **idx_alerts_alert_name_pattern** - Index for pattern matching
   - Use case: `WHERE alert_name LIKE 'pattern%'`
   - Expected improvement: 5-10x faster

8. **idx_alerts_alert_name_trgm** - Trigram index for full-text search (optional)
   - Requires: `CREATE EXTENSION IF NOT EXISTS pg_trgm;`
   - Use case: `WHERE alert_name ILIKE '%query%'`
   - Expected improvement: 5-10x faster

### Index Maintenance

```sql
-- Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
WHERE tablename = 'alerts'
ORDER BY idx_scan DESC;

-- Check index bloat
SELECT pg_size_pretty(pg_relation_size('idx_alerts_status_severity_time')) AS index_size;

-- Reindex if needed (CONCURRENTLY to avoid locks)
REINDEX INDEX CONCURRENTLY idx_alerts_status_severity_time;
```

## Application Optimizations

### Query Optimization

- Use QueryOptimizer to analyze and optimize queries
- Prefer indexed columns for filtering and sorting
- Use partial indexes for common filter combinations
- Limit pagination depth (max page 1000)

### Cache Optimization

- L1 Cache: 10K entries, 5min TTL (in-memory)
- L2 Cache: 1M entries, 1h TTL (Redis)
- Cache warming for popular queries
- Monitor cache hit rate (target > 90%)

### Profiling

- Enable CPU profiling for performance analysis
- Monitor memory usage and GC pauses
- Track goroutine count
- Collect Prometheus metrics

## Performance Targets

- **p95 Latency**: < 10ms
- **p99 Latency**: < 50ms
- **Cache Hit Rate**: > 90%
- **Query Time**: < 5ms (with indexes)
- **Throughput**: > 1000 req/s

## Monitoring

### Key Metrics

- `alert_history_api_history_cache_hits_total` - Cache hits
- `alert_history_api_history_cache_misses_total` - Cache misses
- `alert_history_infra_repository_query_duration_seconds` - Query duration
- `alert_history_performance_goroutines` - Goroutine count
- `alert_history_performance_memory_alloc_bytes` - Memory allocation

### Alerts

- p95 latency > 10ms
- Cache hit rate < 80%
- Query duration > 5ms
- Memory usage > 80%

## Troubleshooting

### Slow Queries

1. Check if indexes are being used: `EXPLAIN ANALYZE <query>`
2. Verify index exists: `\d alerts` in psql
3. Check index statistics: `ANALYZE alerts;`
4. Consider adding missing indexes

### High Memory Usage

1. Reduce L1 cache size
2. Reduce L2 cache TTL
3. Enable cache compression
4. Monitor GC pauses

### Low Cache Hit Rate

1. Increase cache TTL
2. Enable cache warming
3. Increase cache size
4. Analyze popular queries

