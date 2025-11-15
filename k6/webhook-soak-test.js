/**
 * TN-061: Webhook Endpoint - Soak Test
 *
 * Scenario: Sustained moderate load for extended period
 * Duration: 4 hours at 2K req/s
 * Purpose: Detect memory leaks, resource exhaustion, degradation
 *
 * Performance Targets (150% Quality):
 * - Stable performance throughout test
 * - No memory leaks
 * - No performance degradation over time
 * - Error rate < 0.01%
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend, Gauge } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const successRate = new Rate('success_rate');
const webhookDuration = new Trend('webhook_duration');
const currentThroughput = new Gauge('current_throughput');
const degradationScore = new Gauge('degradation_score');

// Tracking for degradation detection
let firstHourAvgLatency = 0;
let latencySamples = [];

// Test configuration
export const options = {
  scenarios: {
    soak_test: {
      executor: 'constant-arrival-rate',
      rate: 2000, // 2K req/s (moderate load)
      timeUnit: '1s',
      duration: '4h',
      preAllocatedVUs: 50,
      maxVUs: 200,
    },
  },
  thresholds: {
    // Strict thresholds for soak test
    'http_req_duration': ['p(95)<5', 'p(99)<10', 'p(99.9)<20'],
    'error_rate': ['rate<0.0001'], // < 0.01%
    'success_rate': ['rate>0.9999'], // > 99.99%
    'http_req_failed': ['rate<0.0001'],

    // Degradation detection
    'degradation_score': ['value<1.2'], // < 20% degradation

    // Memory/resource indicators (indirect)
    'http_req_connecting': ['p(95)<1'],
    'http_req_waiting': ['p(95)<3'],
  },
};

// Test data
const webhookPayload = JSON.stringify({
  receiver: 'webhook-soak',
  alerts: [
    {
      status: 'firing',
      labels: {
        alertname: 'SoakTest',
        severity: 'info',
        instance: 'soak-test-instance',
        job: 'soak-test-job',
      },
      annotations: {
        summary: 'Soak test alert',
        description: 'Long-running soak test to detect memory leaks and degradation',
      },
      startsAt: new Date().toISOString(),
    },
  ],
});

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

let requestCount = 0;
let successCount = 0;
let errorCount = 0;

export default function () {
  const headers = {
    'Content-Type': 'application/json',
  };

  if (API_KEY) {
    headers['X-API-Key'] = API_KEY;
  }

  const startTime = new Date().getTime();

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

  requestCount++;
  webhookDuration.add(duration);
  latencySamples.push(duration);

  // Keep only last 1000 samples for degradation calculation
  if (latencySamples.length > 1000) {
    latencySamples.shift();
  }

  const success = check(response, {
    'status is 200 or 207': (r) => r.status === 200 || r.status === 207,
    'response time < 10ms': (r) => r.timings.duration < 10,
    'response has request_id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.request_id !== undefined;
      } catch (e) {
        return false;
      }
    },
  });

  if (success) {
    successCount++;
    successRate.add(1);
    errorRate.add(0);
  } else {
    errorCount++;
    successRate.add(0);
    errorRate.add(1);

    // Log errors periodically
    if (errorCount % 100 === 0) {
      console.error(`Error #${errorCount}: ${response.status} - ${response.body?.substring(0, 100)}`);
    }
  }

  // Calculate current throughput
  currentThroughput.add(successCount / (requestCount / 2000));

  // Degradation detection (after first hour)
  if (requestCount === 7200000) { // After 1 hour at 2K req/s
    firstHourAvgLatency = latencySamples.reduce((a, b) => a + b, 0) / latencySamples.length;
    console.log(`First hour baseline latency: ${firstHourAvgLatency.toFixed(2)}ms`);
  }

  if (requestCount > 7200000 && firstHourAvgLatency > 0) {
    const currentAvgLatency = latencySamples.reduce((a, b) => a + b, 0) / latencySamples.length;
    const degradation = currentAvgLatency / firstHourAvgLatency;
    degradationScore.add(degradation);

    // Alert if significant degradation
    if (degradation > 1.5 && requestCount % 100000 === 0) {
      console.warn(`⚠️ Performance degradation detected: ${(degradation * 100).toFixed(1)}% increase in latency`);
    }
  }

  // Progress logging (every 10 minutes)
  if (requestCount % 1200000 === 0) {
    const elapsed = requestCount / 2000;
    const minutes = Math.floor(elapsed / 60);
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;

    console.log(`Progress: ${hours}h ${mins}m - Requests: ${requestCount.toLocaleString()}, Success rate: ${(successCount / requestCount * 100).toFixed(3)}%`);
  }

  sleep(0.001);
}

export function setup() {
  console.log('=== Webhook Soak Test ===');
  console.log('Duration: 4 hours');
  console.log('Load: 2,000 req/s (sustained)');
  console.log(`Base URL: ${BASE_URL}`);
  console.log(`Total requests: ~28.8 million`);
  console.log('');
  console.log('Monitoring for:');
  console.log('  - Memory leaks');
  console.log('  - Resource exhaustion');
  console.log('  - Performance degradation');
  console.log('  - Connection pool issues');
  console.log('  - Goroutine leaks');
  console.log('');
  console.log('⏰ This test will run for 4 hours. Grab a coffee! ☕');
  console.log('');

  // Health check
  const response = http.get(`${BASE_URL}/healthz`);
  console.log(`Initial health check: ${response.status}`);

  return {
    startTime: new Date().toISOString(),
    startTimestamp: Date.now(),
  };
}

export function teardown(data) {
  const elapsed = (Date.now() - data.startTimestamp) / 1000;
  const hours = Math.floor(elapsed / 3600);
  const minutes = Math.floor((elapsed % 3600) / 60);

  console.log('');
  console.log('=== Soak Test Complete ===');
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);
  console.log(`Duration: ${hours}h ${minutes}m`);
  console.log(`Total requests: ${requestCount.toLocaleString()}`);
  console.log(`Successful: ${successCount.toLocaleString()} (${(successCount / requestCount * 100).toFixed(3)}%)`);
  console.log(`Errors: ${errorCount.toLocaleString()} (${(errorCount / requestCount * 100).toFixed(3)}%)`);
  console.log('');
  console.log('Expected results:');
  console.log('  ✓ Stable latency throughout test');
  console.log('  ✓ No memory growth');
  console.log('  ✓ Success rate > 99.99%');
  console.log('  ✓ Degradation score < 1.2 (< 20% increase)');
  console.log('');
  console.log('📊 Review Prometheus metrics for:');
  console.log('  - Memory usage trend');
  console.log('  - Goroutine count');
  console.log('  - Connection pool stats');
  console.log('  - GC pause times');
}
