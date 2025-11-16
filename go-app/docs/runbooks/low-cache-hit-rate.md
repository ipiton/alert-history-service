# Runbook: Low Cache Hit Rate

**Severity**: P2 (Medium)
**Last Updated**: 2025-11-16

---

## Symptoms

- Cache hit rate < 80%
- High database load
- Increased query latency
- Alert: `HistoryAPILowCacheHitRate`

---

## Investigation Steps

### 1. Check Cache Metrics

```promql
# Cache hit rate
rate(alert_history_api_history_cache_hits_total[5m]) /
(rate(alert_history_api_history_cache_hits_total[5m]) + rate(alert_history_api_history_cache_misses_total[5m]))

# Cache size
alert_history_api_history_cache_size_entries

# Cache evictions
rate(alert_history_api_history_cache_evictions_total[5m])
```

### 2. Check Cache Configuration

```bash
# Check cache config
curl http://localhost:9090/metrics | grep alert_history_api_history_cache

# Verify cache is enabled
grep "L1Enabled\|L2Enabled" /etc/alert-history/config.yaml
```

### 3. Check Query Patterns

- Are queries too diverse? (many unique cache keys)
- Are time ranges too specific? (few cache hits)
- Are filters too complex? (low cache reuse)

---

## Resolution Steps

### Step 1: Increase Cache TTL

**If cache TTL is too short:**
1. Increase L1 TTL: 5min → 10min
2. Increase L2 TTL: 1h → 2h
3. Monitor hit rate improvement

### Step 2: Enable Cache Warming

**If popular queries aren't cached:**
1. Enable cache warmer
2. Configure popular queries
3. Monitor cache population

### Step 3: Increase Cache Size

**If cache is evicting frequently:**
1. Increase L1 max entries: 10K → 50K
2. Increase L2 max entries: 1M → 5M
3. Monitor memory usage

### Step 4: Optimize Cache Keys

**If cache keys are too specific:**
1. Review cache key generation
2. Normalize query parameters
3. Consider cache key hashing

---

## Prevention

### Monitoring

- **Alert**: Cache hit rate < 80% for 10 minutes
- **Dashboard**: Cache Performance
- **Metrics**: Hit rate, size, evictions

### Best Practices

1. **Cache Configuration**:
   - L1: 10K entries, 5min TTL
   - L2: 1M entries, 1h TTL
   - Cache warming enabled

2. **Query Patterns**:
   - Use common time ranges
   - Standardize filter combinations
   - Avoid overly specific queries

3. **Cache Management**:
   - Monitor cache size
   - Review eviction patterns
   - Adjust TTL based on data freshness needs

---

## Escalation

If cache hit rate remains low:
1. Contact Development Team (cache optimization)
2. Contact Infrastructure Team (Redis scaling)

---

## Related Runbooks

- [High Latency](./high-latency.md)
- [Database Performance Issues](./database-performance.md)
