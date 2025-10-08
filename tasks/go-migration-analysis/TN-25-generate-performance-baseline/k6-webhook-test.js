// k6 Load Test Script for Webhook Endpoint
// This script tests the /webhook endpoint with various load patterns

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const webhookDuration = new Trend('webhook_duration');

// Test configuration
export const options = {
  scenarios: {
    // Scenario 1: Constant load - 1000 RPS for 2 minutes
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 1000, // 1000 requests per second
      timeUnit: '1s',
      duration: '2m',
      preAllocatedVUs: 50,
      maxVUs: 200,
      tags: { scenario: 'constant_load' },
    },

    // Scenario 2: Ramp up test - gradually increase to 1500 RPS
    ramp_up: {
      executor: 'ramping-arrival-rate',
      startRate: 100,
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 300,
      stages: [
        { duration: '30s', target: 500 },  // Ramp up to 500 RPS
        { duration: '1m', target: 1000 },  // Ramp up to 1000 RPS
        { duration: '1m', target: 1500 },  // Ramp up to 1500 RPS
        { duration: '30s', target: 1000 }, // Ramp down to 1000 RPS
        { duration: '30s', target: 0 },    // Ramp down to 0
      ],
      tags: { scenario: 'ramp_up' },
    },

    // Scenario 3: Spike test - sudden load increase
    spike_test: {
      executor: 'ramping-arrival-rate',
      startRate: 500,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 500,
      stages: [
        { duration: '30s', target: 500 },  // Normal load
        { duration: '10s', target: 2000 }, // Spike to 2000 RPS
        { duration: '30s', target: 2000 }, // Maintain spike
        { duration: '10s', target: 500 },  // Return to normal
        { duration: '30s', target: 500 },  // Maintain normal
      ],
      tags: { scenario: 'spike_test' },
    },
  },

  thresholds: {
    http_req_duration: ['p(95)<100'], // 95% of requests should be below 100ms
    http_req_failed: ['rate<0.01'],   // Error rate should be less than 1%
    errors: ['rate<0.01'],            // Custom error rate should be less than 1%
  },
};

// Base URL - can be overridden with environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Sample webhook payloads for testing
const webhookPayloads = [
  {
    alertname: 'HighCPUUsage',
    status: 'firing',
    labels: {
      instance: 'server-01',
      job: 'node-exporter',
      severity: 'warning',
    },
    annotations: {
      summary: 'High CPU usage detected',
      description: 'CPU usage is above 80% for more than 5 minutes',
    },
    startsAt: new Date().toISOString(),
    endsAt: '',
    generatorURL: 'http://prometheus:9090/graph',
    fingerprint: 'abc123def456',
  },
  {
    alertname: 'HighMemoryUsage',
    status: 'firing',
    labels: {
      instance: 'server-02',
      job: 'node-exporter',
      severity: 'critical',
    },
    annotations: {
      summary: 'High memory usage detected',
      description: 'Memory usage is above 90% for more than 2 minutes',
    },
    startsAt: new Date().toISOString(),
    endsAt: '',
    generatorURL: 'http://prometheus:9090/graph',
    fingerprint: 'def456ghi789',
  },
  {
    alertname: 'DiskSpaceLow',
    status: 'resolved',
    labels: {
      instance: 'server-03',
      job: 'node-exporter',
      severity: 'warning',
      mountpoint: '/var',
    },
    annotations: {
      summary: 'Disk space is low',
      description: 'Available disk space is below 10%',
    },
    startsAt: new Date(Date.now() - 3600000).toISOString(), // 1 hour ago
    endsAt: new Date().toISOString(),
    generatorURL: 'http://prometheus:9090/graph',
    fingerprint: 'ghi789jkl012',
  },
];

export default function () {
  // Select random payload
  const payload = webhookPayloads[Math.floor(Math.random() * webhookPayloads.length)];

  // Add some randomization to make it more realistic
  payload.labels.instance = `server-${Math.floor(Math.random() * 100) + 1}`;
  payload.fingerprint = Math.random().toString(36).substring(2, 15);
  payload.startsAt = new Date().toISOString();

  const params = {
    headers: {
      'Content-Type': 'application/json',
    },
    tags: {
      endpoint: 'webhook',
    },
  };

  const startTime = Date.now();
  const response = http.post(`${BASE_URL}/webhook`, JSON.stringify(payload), params);
  const duration = Date.now() - startTime;

  // Record custom metrics
  webhookDuration.add(duration);
  errorRate.add(response.status !== 200);

  // Validate response
  const result = check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 200ms': (r) => r.timings.duration < 200,
    'response has alert_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.alert_id !== undefined;
      } catch (e) {
        return false;
      }
    },
    'response has success status': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.status === 'success';
      } catch (e) {
        return false;
      }
    },
  });

  // Log errors for debugging
  if (response.status !== 200) {
    console.error(`Request failed: ${response.status} - ${response.body}`);
  }

  // Small random sleep to simulate realistic usage patterns
  sleep(Math.random() * 0.1); // 0-100ms random sleep
}

// Setup function - runs once before the test
export function setup() {
  console.log(`Starting webhook load test against ${BASE_URL}`);

  // Verify the service is running
  const healthCheck = http.get(`${BASE_URL}/healthz`);
  if (healthCheck.status !== 200) {
    throw new Error(`Service health check failed: ${healthCheck.status}`);
  }

  console.log('Service health check passed');
  return { baseUrl: BASE_URL };
}

// Teardown function - runs once after the test
export function teardown(data) {
  console.log('Webhook load test completed');
  console.log(`Test ran against: ${data.baseUrl}`);
}
