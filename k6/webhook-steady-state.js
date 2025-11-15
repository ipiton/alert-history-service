/**
 * TN-061: Webhook Endpoint - Steady State Load Test
 * 
 * Scenario: Sustained load at target throughput
 * Target: 10,000 req/s for 10 minutes
 * Purpose: Validate system can handle production load
 * 
 * Performance Targets (150% Quality):
 * - p95 latency < 5ms
 * - p99 latency < 10ms
 * - Error rate < 0.01%
 * - Throughput > 10,000 req/s
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const successRate = new Rate('success_rate');
const webhookDuration = new Trend('webhook_duration');
const payloadSize = new Counter('payload_size');

// Test configuration
export const options = {
  scenarios: {
    steady_state: {
      executor: 'constant-arrival-rate',
      rate: 10000, // 10K requests per second
      timeUnit: '1s',
      duration: '10m',
      preAllocatedVUs: 100,
      maxVUs: 500,
    },
  },
  thresholds: {
    // Performance targets (150% quality)
    'http_req_duration{type:webhook}': ['p(95)<5', 'p(99)<10'],
    'error_rate': ['rate<0.0001'], // < 0.01%
    'success_rate': ['rate>0.9999'], // > 99.99%
    'http_req_failed': ['rate<0.0001'],
    
    // Response times
    'http_req_waiting': ['p(95)<3', 'p(99)<7'],
    'http_req_connecting': ['p(95)<1'],
    
    // Throughput
    'http_reqs': ['rate>9500'], // Allow 5% margin
  },
};

// Test data: Alertmanager-style webhook payload
const webhookPayload = JSON.stringify({
  receiver: 'webhook',
  status: 'firing',
  alerts: [
    {
      status: 'firing',
      labels: {
        alertname: 'HighCPU',
        severity: 'critical',
        instance: 'server-1',
        job: 'api',
        datacenter: 'us-east-1',
      },
      annotations: {
        summary: 'High CPU usage detected',
        description: 'CPU usage is above 90% for 5 minutes',
        runbook_url: 'https://runbooks.example.com/HighCPU',
      },
      startsAt: new Date().toISOString(),
      endsAt: '0001-01-01T00:00:00Z',
      generatorURL: 'http://prometheus:9090/graph',
      fingerprint: '7c7d3d3e7f8e9a0b',
    },
  ],
  groupLabels: {
    alertname: 'HighCPU',
  },
  commonLabels: {
    severity: 'critical',
  },
  commonAnnotations: {
    summary: 'High CPU usage detected',
  },
  externalURL: 'http://alertmanager:9093',
  version: '4',
  groupKey: '{}:{alertname="HighCPU"}',
});

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

export default function () {
  const headers = {
    'Content-Type': 'application/json',
  };

  // Add API key if configured
  if (API_KEY) {
    headers['X-API-Key'] = API_KEY;
  }

  const startTime = new Date().getTime();

  // POST webhook request
  const response = http.post(
    `${BASE_URL}/webhook`,
    webhookPayload,
    {
      headers: headers,
      tags: { type: 'webhook' },
    }
  );

  const endTime = new Date().getTime();
  const duration = endTime - startTime;

  // Record metrics
  webhookDuration.add(duration);
  payloadSize.add(webhookPayload.length);

  // Validate response
  const success = check(response, {
    'status is 200 or 207': (r) => r.status === 200 || r.status === 207,
    'response time < 10ms': (r) => r.timings.duration < 10,
    'response has request_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.request_id !== undefined && body.request_id !== '';
      } catch (e) {
        return false;
      }
    },
    'response is JSON': (r) => r.headers['Content-Type']?.includes('application/json'),
  });

  // Record success/error rates
  if (success) {
    successRate.add(1);
    errorRate.add(0);
  } else {
    successRate.add(0);
    errorRate.add(1);
    console.error(`Request failed: ${response.status} - ${response.body}`);
  }

  // Small sleep to avoid overwhelming the system
  sleep(0.001); // 1ms
}

// Setup function (runs once at start)
export function setup() {
  console.log('=== Webhook Steady State Load Test ===');
  console.log(`Target: 10,000 req/s for 10 minutes`);
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`API Key: ${API_KEY ? 'Configured' : 'Not configured'}`);
  console.log(`Payload size: ${webhookPayload.length} bytes`);
  console.log('');
  
  // Warmup request
  const response = http.get(`${BASE_URL}/healthz`);
  console.log(`Health check: ${response.status}`);
  
  return { startTime: new Date().toISOString() };
}

// Teardown function (runs once at end)
export function teardown(data) {
  console.log('');
  console.log('=== Test Complete ===');
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);
}

