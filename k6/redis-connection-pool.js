/*
 * redis-connection-pool.js
 * k6 load test for Redis connection pool (TN-99)
 *
 * Tests:
 * - Connection pool capacity (500 concurrent connections)
 * - SET/GET operations under load
 * - Latency under stress (target: <50ms p95)
 * - Success rate (target: >99%)
 *
 * Usage:
 *   k6 run k6/redis-connection-pool.js
 *
 * Prerequisites:
 *   - k6 installed (https://k6.io/docs/getting-started/installation/)
 *   - xk6-redis extension (https://github.com/grafana/xk6-redis)
 *   - Redis accessible at alerthistory-redis:6379
 */

import redis from 'k6/x/redis';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const successRate = new Rate('redis_success_rate');
const setDuration = new Trend('redis_set_duration');
const getDuration = new Trend('redis_get_duration');

// Test configuration
export let options = {
  stages: [
    // Ramp up to 500 connections over 1 minute
    { duration: '1m', target: 500 },
    // Hold 500 connections for 5 minutes
    { duration: '5m', target: 500 },
    // Ramp down to 0 over 1 minute
    { duration: '1m', target: 0 },
  ],
  thresholds: {
    // Success rate must be >99%
    'redis_success_rate': ['rate>0.99'],
    // p95 latency must be <50ms
    'redis_set_duration': ['p(95)<50'],
    'redis_get_duration': ['p(95)<50'],
    // No errors allowed
    'checks': ['rate>0.99'],
  },
};

// Redis client configuration
const redisAddr = __ENV.REDIS_ADDR || 'alerthistory-redis:6379';
const redisPassword = __ENV.REDIS_PASSWORD || '';

// Create Redis client (one per VU)
const client = redis.newClient({
  addrs: [redisAddr],
  password: redisPassword,
});

export default function () {
  // Generate unique key for this VU and iteration
  const key = `test-key-${__VU}-${__ITER}`;
  const value = `test-value-${__VU}-${__ITER}-${Date.now()}`;

  // Test SET operation
  const setStart = Date.now();
  const setResult = client.set(key, value);
  const setEnd = Date.now();
  setDuration.add(setEnd - setStart);

  const setSuccess = check(setResult, {
    'SET successful': (r) => r === 'OK',
  });
  successRate.add(setSuccess);

  // Small delay between SET and GET
  sleep(0.001); // 1ms

  // Test GET operation
  const getStart = Date.now();
  const getValue = client.get(key);
  const getEnd = Date.now();
  getDuration.add(getEnd - getStart);

  const getSuccess = check(getValue, {
    'GET successful': (v) => v === value,
  });
  successRate.add(getSuccess);

  // Test TTL (optional)
  client.expire(key, 3600); // 1 hour TTL

  // Small delay before next iteration
  sleep(0.1); // 100ms think time
}

export function teardown(data) {
  // Cleanup: Close Redis client
  client.close();
}

/*
 * Expected Results:
 * - 500 concurrent connections handled successfully
 * - Success rate: >99%
 * - p95 latency: <50ms (SET and GET)
 * - No connection rejections (redis_rejected_connections_total = 0)
 * - Redis memory usage stable (<90% of maxmemory)
 *
 * Monitoring:
 * - Watch Grafana Redis dashboard during test
 * - Check Prometheus metrics:
 *   - redis_connected_clients (should peak at ~500)
 *   - redis_commands_processed_total (should increase linearly)
 *   - redis_rejected_connections_total (should stay 0)
 *   - redis_memory_used_bytes (should stay <345MB, 90% of 384MB)
 *
 * Troubleshooting:
 * - If success rate <99%: Check network, Redis logs
 * - If p95 >50ms: Check Redis slow query log, memory fragmentation
 * - If connections rejected: Increase maxclients or reduce VUs
 */
