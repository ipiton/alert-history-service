# TN-72: Troubleshooting Guide

## Common Issues and Solutions

### 1. Validation Errors (400 Bad Request)

#### Issue: "fingerprint is required"
**Cause**: Missing or empty fingerprint field
**Solution**: Ensure `alert.fingerprint` is present and non-empty

```json
{
  "alert": {
    "fingerprint": "alert-123",  // Required
    ...
  }
}
```

#### Issue: "alert_name is required"
**Cause**: Missing or empty alert_name field
**Solution**: Ensure `alert.alert_name` is present and non-empty

#### Issue: "status must be 'firing' or 'resolved'"
**Cause**: Invalid status value
**Solution**: Use only `"firing"` or `"resolved"` (lowercase)

#### Issue: "starts_at is required"
**Cause**: Missing or zero starts_at timestamp
**Solution**: Provide valid RFC3339 timestamp

```json
{
  "alert": {
    "starts_at": "2025-11-17T21:00:00Z"  // RFC3339 format
  }
}
```

#### Issue: "generator_url must be a valid HTTP/HTTPS URL"
**Cause**: Invalid generator_url format
**Solution**: Use valid HTTP/HTTPS URL or omit the field

```json
{
  "alert": {
    "generator_url": "https://prometheus.example.com"  // Valid URL
  }
}
```

### 2. Timeout Errors (504 Gateway Timeout)

#### Issue: "LLM classification request timed out"
**Cause**: LLM service is slow or unavailable
**Solution**:
1. Check LLM service health: `GET /api/v2/classification/stats`
2. Retry with exponential backoff
3. Use `force=false` to leverage cache
4. Check network connectivity to LLM service

**Retry Logic Example**:
```go
maxRetries := 3
backoff := 100 * time.Millisecond
for i := 0; i < maxRetries; i++ {
    resp, err := classifyAlert(alert)
    if err == nil {
        return resp, nil
    }
    if !isTimeoutError(err) {
        return nil, err
    }
    time.Sleep(backoff)
    backoff *= 2
}
```

### 3. Service Unavailable Errors (503)

#### Issue: "LLM service unavailable"
**Cause**: Circuit breaker is open or LLM service is down
**Solution**:
1. Check LLM service status
2. Wait for circuit breaker to close (usually 30-60 seconds)
3. Use fallback classification (automatic)
4. Check service logs for details

**Monitoring**:
```promql
# Circuit breaker status
alert_history_business_classification_errors_total{error_type="circuit_breaker"}
```

### 4. Cache Issues

#### Issue: Cache not working (always cache miss)
**Cause**: Cache service unavailable or misconfigured
**Solution**:
1. Check Redis connectivity
2. Verify cache configuration
3. Check cache metrics: `alert_history_business_classification_l1_cache_hits_total`
4. Review cache TTL settings

#### Issue: Force flag not invalidating cache
**Cause**: Cache invalidation failed (non-critical)
**Solution**: This is expected behavior - classification continues even if cache invalidation fails. Check logs for warnings.

### 5. Performance Issues

#### Issue: Slow response times
**Cause**: Multiple factors possible
**Solution**:

1. **Check cache hit rate**:
```promql
rate(alert_history_business_classification_l1_cache_hits_total[5m]) /
rate(api_http_requests_total{endpoint="/classification/classify"}[5m])
```

2. **Monitor LLM latency**:
```promql
histogram_quantile(0.95,
  alert_history_business_classification_duration_seconds{source="llm"})
```

3. **Check for high request rate**:
```promql
rate(api_http_requests_total{endpoint="/classification/classify"}[1m])
```

4. **Optimize usage**:
   - Use `force=false` when possible
   - Batch requests when appropriate
   - Monitor and adjust rate limits

### 6. Authentication Issues

#### Issue: 401 Unauthorized
**Cause**: Missing or invalid API key
**Solution**:
1. Verify API key is correct
2. Check API key format: `Authorization: Bearer <api-key>`
3. Verify API key has required permissions
4. Check API key expiration

### 7. Rate Limiting Issues

#### Issue: 429 Too Many Requests
**Cause**: Rate limit exceeded
**Solution**:
1. Check rate limit headers:
   - `X-RateLimit-Limit`: Maximum requests per window
   - `X-RateLimit-Remaining`: Remaining requests
   - `X-RateLimit-Reset`: Reset timestamp

2. Implement exponential backoff:
```go
if resp.StatusCode == 429 {
    resetTime := resp.Header.Get("X-RateLimit-Reset")
    waitTime := calculateWaitTime(resetTime)
    time.Sleep(waitTime)
    // Retry request
}
```

3. Reduce request rate or request limit increase

### 8. Response Format Issues

#### Issue: Unexpected response format
**Cause**: API version mismatch or malformed response
**Solution**:
1. Verify API version: `X-API-Version: v2`
2. Check response Content-Type: `application/json`
3. Validate JSON structure matches expected format
4. Check for API updates/changes

### 9. Metadata Issues

#### Issue: Model field missing in response
**Cause**: Classification result doesn't include model metadata
**Solution**: This is expected if:
- Classification used fallback (no LLM)
- LLM response doesn't include model info
- Metadata format changed

Check `result.metadata` field for available information.

### 10. Debugging Tips

#### Enable Debug Logging
Set log level to DEBUG to see detailed request/response logs:
```bash
export LOG_LEVEL=DEBUG
```

#### Use Request ID
Include `X-Request-ID` header for request tracking:
```bash
curl -H "X-Request-ID: debug-123" ...
```

Then search logs:
```bash
grep "debug-123" /var/log/alert-history.log
```

#### Check Prometheus Metrics
Query relevant metrics:
```promql
# Request rate
rate(api_http_requests_total{endpoint="/classification/classify"}[5m])

# Error rate
rate(api_http_requests_total{endpoint="/classification/classify",status=~"5.."}[5m])

# Cache hit rate
rate(alert_history_business_classification_l1_cache_hits_total[5m]) /
rate(api_http_requests_total{endpoint="/classification/classify"}[5m])

# P95 latency
histogram_quantile(0.95,
  api_http_request_duration_seconds{endpoint="/classification/classify"})
```

## Getting Help

If issues persist:
1. Check service logs: `/var/log/alert-history.log`
2. Review Prometheus metrics
3. Check service health: `GET /health`
4. Contact support with:
   - Request ID
   - Error message
   - Request payload (sanitized)
   - Timestamp

## Related Documentation

- [API Guide](./API_GUIDE.md)
- [Design Document](./design.md)
- [Requirements](./requirements.md)
