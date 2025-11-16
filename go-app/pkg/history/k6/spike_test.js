import http from 'k6/http';
import { check, sleep } from 'k6';

// Spike test - sudden load increase
export const options = {
  stages: [
    { duration: '1m', target: 10 },   // Normal load
    { duration: '10s', target: 500 }, // Spike!
    { duration: '1m', target: 10 },  // Back to normal
    { duration: '10s', target: 500 }, // Another spike!
    { duration: '1m', target: 10 },  // Recovery
  ],
  thresholds: {
    http_req_duration: ['p(95)<200'],
    http_req_failed: ['rate<0.02'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || 'test-api-key';

export default function () {
  const res = http.get(`${BASE_URL}/api/v2/history`, {
    params: {
      page: 1,
      per_page: 50,
      status: 'firing',
    },
    headers: {
      'Authorization': `ApiKey ${API_KEY}`,
      'X-Request-ID': `k6-spike-${__VU}-${__ITER}`,
    },
  });
  
  check(res, {
    'status is 200': (r) => r.status === 200,
  });
  
  sleep(0.1);
}

