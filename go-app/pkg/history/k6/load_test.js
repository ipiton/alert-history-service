import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const historyLatency = new Trend('history_latency');

export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 50 },    // Ramp up to 50 users
    { duration: '2m', target: 100 },  // Stay at 100 users
    { duration: '1m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'], // 95% of requests < 100ms
    http_req_failed: ['rate<0.01'],    // Error rate < 1%
    errors: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
  // Test 1: GET /api/v2/history - Basic query
  const historyParams = {
    page: 1,
    per_page: 50,
    status: 'firing',
  };

  const historyUrl = `${BASE_URL}/api/v2/history`;
  const historyRes = http.get(historyUrl, {
    params: historyParams,
    headers: {
      'Authorization': `ApiKey ${API_KEY}`,
      'X-Request-ID': `k6-${__VU}-${__ITER}`,
    },
  });

  const historyCheck = check(historyRes, {
    'history status is 200': (r) => r.status === 200,
    'history has alerts': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.alerts !== undefined;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!historyCheck);
  historyLatency.add(historyRes.timings.duration);

  sleep(1);

  // Test 2: GET /api/v2/history/top - Top alerts
  const topRes = http.get(`${BASE_URL}/api/v2/history/top`, {
    params: { limit: 10 },
    headers: {
      'Authorization': `ApiKey ${API_KEY}`,
      'X-Request-ID': `k6-${__VU}-${__ITER}`,
    },
  });

  check(topRes, {
    'top status is 200': (r) => r.status === 200,
  });

  sleep(1);

  // Test 3: GET /api/v2/history/recent - Recent alerts
  const recentRes = http.get(`${BASE_URL}/api/v2/history/recent`, {
    params: { limit: 20 },
    headers: {
      'Authorization': `ApiKey ${API_KEY}`,
      'X-Request-ID': `k6-${__VU}-${__ITER}`,
    },
  });

  check(recentRes, {
    'recent status is 200': (r) => r.status === 200,
  });

  sleep(1);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'summary.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  return `
  ============================================
  K6 Load Test Summary
  ============================================
  Total Requests: ${data.metrics.http_reqs.values.count}
  Failed Requests: ${data.metrics.http_req_failed.values.rate * 100}%
  P95 Latency: ${data.metrics.http_req_duration.values['p(95)']}ms
  P99 Latency: ${data.metrics.http_req_duration.values['p(99)']}ms
  ============================================
  `;
}
