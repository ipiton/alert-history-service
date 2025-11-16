import http from 'k6/http';
import { check, sleep } from 'k6';

// Stress test - gradually increase load until system breaks
export const options = {
  stages: [
    { duration: '1m', target: 50 },
    { duration: '2m', target: 100 },
    { duration: '2m', target: 200 },
    { duration: '2m', target: 300 },
    { duration: '2m', target: 400 },
    { duration: '2m', target: 500 },
    { duration: '5m', target: 0 }, // Recovery
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // Relaxed for stress test
    http_req_failed: ['rate<0.05'],   // 5% error rate acceptable
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
  // Mix of different endpoints
  const endpoints = [
    '/api/v2/history?page=1&per_page=50',
    '/api/v2/history/top?limit=10',
    '/api/v2/history/recent?limit=20',
    '/api/v2/history/stats',
  ];
  
  const endpoint = endpoints[Math.floor(Math.random() * endpoints.length)];
  const res = http.get(`${BASE_URL}${endpoint}`, {
    headers: {
      'Authorization': `ApiKey ${API_KEY}`,
      'X-Request-ID': `k6-stress-${__VU}-${__ITER}`,
    },
  });
  
  check(res, {
    'status is 200 or 429': (r) => r.status === 200 || r.status === 429,
  });
  
  sleep(0.5);
}

