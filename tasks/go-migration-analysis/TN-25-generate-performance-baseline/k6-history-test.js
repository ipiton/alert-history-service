// k6 Load Test Script for History API Endpoint
// This script tests the /history endpoint with various query patterns

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const historyDuration = new Trend('history_duration');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Constant load - 500 RPS for 2 minutes
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 500, // 500 requests per second
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 30,
      maxVUs: 100,
      tags: { scenario: 'constant_load' },
    },

    // Scenario 2: Ramp up test - gradually increase to 750 RPS
    ramp_up: {
      executor: 'ramping-arrival-rate',
      startRate: 50,
      timeUnit: '1s',
      preAllocatedVUs: 30,
      maxVUs: 150,
      stages: [
        { duration: '30s', target: 250 },  // Ramp up to 250 RPS
        { duration: '1m', target: 500 },   // Ramp up to 500 RPS
        { duration: '1m', target: 750 },   // Ramp up to 750 RPS
        { duration: '30s', target: 500 },  // Ramp down to 500 RPS
        { duration: '30s', target: 0 },    // Ramp down to 0
      ],
      tags: { scenario: 'ramp_up' },
    },

    // Scenario 3: Pagination stress test
    pagination_stress: {
      executor: 'ramping-arrival-rate',
      startRate: 200,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 200,
      stages: [
        { duration: '30s', target: 200 },  // Normal load
        { duration: '1m', target: 1000 },  // High load for pagination
        { duration: '30s', target: 200 },  // Return to normal
      ],
      tags: { scenario: 'pagination_stress' },
    },
  },

  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be below 500ms
    http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
    errors: ['rate<0.01'],            // Custom error rate should be less than 1%
  },
};

// Base URL - can be overridden with environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Query patterns for realistic testing
const queryPatterns = [
  // Basic pagination
  { page: 1, page_size: 50 },
  { page: 2, page_size: 50 },
  { page: 1, page_size: 100 },
  { page: 5, page_size: 20 },

  // Status filtering
  { page: 1, page_size: 50, status: 'firing' },
  { page: 1, page_size: 50, status: 'resolved' },
  { page: 2, page_size: 30, status: 'firing' },

  // Alert name filtering
  { page: 1, page_size: 50, alertname: 'HighCPUUsage' },
  { page: 1, page_size: 50, alertname: 'HighMemoryUsage' },
  { page: 1, page_size: 50, alertname: 'DiskSpaceLow' },
  { page: 1, page_size: 50, alertname: 'ServiceDown' },

  // Combined filters
  { page: 1, page_size: 50, status: 'firing', alertname: 'HighCPUUsage' },
  { page: 2, page_size: 25, status: 'resolved', alertname: 'DiskSpaceLow' },

  // Large page sizes (stress test)
  { page: 1, page_size: 500 },
  { page: 1, page_size: 1000 },

  // Deep pagination
  { page: 50, page_size: 50 },
  { page: 100, page_size: 20 },
  { page: 200, page_size: 10 },
];

export default function () {
  // Select random query pattern
  const queryPattern = queryPatterns[Math.floor(Math.random() * queryPatterns.length)];

  // Build query string
  const queryParams = new URLSearchParams();
  Object.keys(queryPattern).forEach(key => {
    if (queryPattern[key] !== undefined) {
      queryParams.append(key, queryPattern[key].toString());
    }
  });

  const url = `${BASE_URL}/history?${queryParams.toString()}`;

  const params = {
    headers: {
      'Accept': 'application/json',
    },
    tags: {
      endpoint: 'history',
      page: queryPattern.page.toString(),
      page_size: queryPattern.page_size.toString(),
      has_status_filter: queryPattern.status ? 'true' : 'false',
      has_alertname_filter: queryPattern.alertname ? 'true' : 'false',
    },
  };

  const startTime = Date.now();
  const response = http.get(url, params);
  const duration = Date.now() - startTime;

  // Record custom metrics
  historyDuration.add(duration);
  errorRate.add(response.status !== 200);

  // Validate response
  const result = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 1000ms': (r) => r.timings.duration < 1000,
    'response has alerts array': (r) => {
      try {
        const body = JSON.parse(r.body);
        return Array.isArray(body.alerts);
      } catch (e) {
        return false;
      }
    },
    'response has pagination info': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.total !== undefined &&
               body.page !== undefined &&
               body.page_size !== undefined;
      } catch (e) {
        return false;
      }
    },
    'response has timestamp': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.timestamp !== undefined;
      } catch (e) {
        return false;
      }
    },
    'alerts have required fields': (r) => {
      try {
        const body = JSON.parse(r.body);
        if (!Array.isArray(body.alerts) || body.alerts.length === 0) {
          return true; // Empty results are valid
        }

        const firstAlert = body.alerts[0];
        return firstAlert.id !== undefined &&
               firstAlert.alertname !== undefined &&
               firstAlert.status !== undefined &&
               firstAlert.labels !== undefined &&
               firstAlert.annotations !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  // Log errors for debugging
  if (response.status !== 200) {
    console.error(`Request failed: ${response.status} - ${response.body}`);
    console.error(`URL: ${url}`);
  }

  // Validate response data structure for large responses
  if (response.status === 200) {
    try {
      const body = JSON.parse(response.body);

      // Check if page size is respected
      if (body.alerts.length > queryPattern.page_size) {
        console.warn(`Page size exceeded: expected max ${queryPattern.page_size}, got ${body.alerts.length}`);
      }

      // Check pagination consistency
      if (queryPattern.page > 1 && body.alerts.length === 0 && body.total > 0) {
        console.warn(`Empty page ${queryPattern.page} but total is ${body.total}`);
      }

    } catch (e) {
      console.error(`Failed to parse response: ${e.message}`);
    }
  }

  // Small random sleep to simulate realistic usage patterns
  sleep(Math.random() * 0.2); // 0-200ms random sleep
}

// Setup function - runs once before the test
export function setup() {
  console.log(`Starting history API load test against ${BASE_URL}`);

  // Verify the service is running
  const healthCheck = http.get(`${BASE_URL}/healthz`);
  if (healthCheck.status !== 200) {
    throw new Error(`Service health check failed: ${healthCheck.status}`);
  }

  // Test basic history endpoint functionality
  const basicHistoryTest = http.get(`${BASE_URL}/history?page=1&page_size=10`);
  if (basicHistoryTest.status !== 200) {
    throw new Error(`History endpoint test failed: ${basicHistoryTest.status}`);
  }

  try {
    const body = JSON.parse(basicHistoryTest.body);
    if (!Array.isArray(body.alerts)) {
      throw new Error('History endpoint does not return alerts array');
    }
    console.log(`History endpoint working, returned ${body.alerts.length} alerts out of ${body.total} total`);
  } catch (e) {
    throw new Error(`History endpoint response validation failed: ${e.message}`);
  }

  console.log('Service health check and history endpoint validation passed');
  return { baseUrl: BASE_URL };
}

// Teardown function - runs once after the test
export function teardown(data) {
  console.log('History API load test completed');
  console.log(`Test ran against: ${data.baseUrl}`);
}
