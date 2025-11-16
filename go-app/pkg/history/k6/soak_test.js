import http from 'k6/http';
import { check, sleep } from 'k6';

// Soak test - sustained load over long period
export const options = {
  stages: [
    { duration: '5m', target: 50 },  // Ramp up
    { duration: '30m', target: 50 },  // Sustained load
    { duration: '5m', target: 0 },    // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<100'],
    http_req_failed: ['rate<0.01'],
    // Memory leak detection
    'http_req_duration{status:200}': ['p(95)<100'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
  // Rotate through different endpoints
  const endpoints = [
    { path: '/api/v2/history', params: { page: 1, per_page: 50 } },
    { path: '/api/v2/history/top', params: { limit: 10 } },
    { path: '/api/v2/history/recent', params: { limit: 20 } },
  ];

  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const res = http.get(`${BASE_URL}${endpoint.path}`, {
    params: endpoint.params,
    headers: {
      'Authorization': `ApiKey ${API_KEY}`,
      'X-Request-ID': `k6-soak-${__VU}-${__ITER}`,
    },
  });

  check(res, {
    'status is 200': (r) => r.status === 200,
  });

  sleep(1);
}
