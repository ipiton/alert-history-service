/**
 * TN-061: Webhook Endpoint - Spike Test
 *
 * Scenario: Sudden traffic spike to test elasticity
 * Pattern: 1K → 20K → 1K req/s
 * Purpose: Validate system handles sudden load increases
 *
 * Performance Targets (150% Quality):
 * - System remains stable during spike
 * - Recovery to normal after spike
 * - Error rate < 0.1% during spike
 * - No lingering effects after spike
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const spikeErrors = new Rate('spike_errors');
const recoveryErrors = new Rate('recovery_errors');
const webhookDuration = new Trend('webhook_duration');

// Test configuration
export const options = {
  scenarios: {
    spike_test: {
      executor: 'ramping-arrival-rate',
      startRate: 1000, // Start at 1K req/s
      timeUnit: '1s',
      preAllocatedVUs: 50,
      maxVUs: 1000,
      stages: [
        { duration: '2m', target: 1000 },   // Baseline: 1K req/s for 2 min
        { duration: '30s', target: 20000 }, // Spike: ramp to 20K in 30s
        { duration: '1m', target: 20000 },  // Peak: hold 20K for 1 min
        { duration: '30s', target: 1000 },  // Recovery: ramp down in 30s
        { duration: '2m', target: 1000 },   // Stabilize: 1K for 2 min
      ],
    },
  },
  thresholds: {
    // During spike, allow slightly higher error rate
    'http_req_duration': ['p(95)<15', 'p(99)<30'],
    'error_rate': ['rate<0.001'], // < 0.1%
    'http_req_failed': ['rate<0.001'],

    // Spike-specific thresholds
    'spike_errors': ['rate<0.005'], // < 0.5% during spike
    'recovery_errors': ['rate<0.0001'], // < 0.01% after recovery
  },
};

// Test data
const webhookPayload = JSON.stringify({
  alerts: [
    {
      status: 'firing',
      labels: {
        alertname: 'SpikeTest',
        severity: 'warning',
      },
      annotations: {
        summary: 'Spike test alert',
      },
      startsAt: new Date().toISOString(),
    },
  ],
});

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

// Track test phase
let testPhase = 'baseline';

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
      tags: {
        type: 'webhook',
        phase: testPhase,
      },
    }
  );

  const endTime = new Date().getTime();
  const duration = endTime - startTime;

  webhookDuration.add(duration, { phase: testPhase });

  const success = check(response, {
    'status is 200 or 207 or 429': (r) =>
      r.status === 200 || r.status === 207 || r.status === 429,
    'response time < 30ms': (r) => r.timings.duration < 30,
  });

  // Track errors by phase
  if (!success || response.status >= 400) {
    errorRate.add(1);

    if (testPhase === 'spike') {
      spikeErrors.add(1);
    } else if (testPhase === 'recovery') {
      recoveryErrors.add(1);
    }
  } else {
    errorRate.add(0);

    if (testPhase === 'spike') {
      spikeErrors.add(0);
    } else if (testPhase === 'recovery') {
      recoveryErrors.add(0);
    }
  }

  // Update test phase based on elapsed time
  const elapsed = (__ITER * 1000) / __ENV.K6_VUS || 0;
  if (elapsed < 120000) {
    testPhase = 'baseline';
  } else if (elapsed < 150000) {
    testPhase = 'spike';
  } else if (elapsed < 210000) {
    testPhase = 'peak';
  } else if (elapsed < 240000) {
    testPhase = 'recovery';
  } else {
    testPhase = 'stabilize';
  }

  sleep(0.001);
}

export function setup() {
  console.log('=== Webhook Spike Test ===');
  console.log('Pattern: 1K → 20K → 1K req/s');
  console.log('Duration: 7 minutes total');
  console.log(`Base URL: ${BASE_URL}`);
  console.log('');
  console.log('Phases:');
  console.log('  0-2m: Baseline (1K req/s)');
  console.log('  2-2.5m: Ramp up (1K → 20K)');
  console.log('  2.5-3.5m: Peak (20K req/s)');
  console.log('  3.5-4m: Ramp down (20K → 1K)');
  console.log('  4-6m: Stabilize (1K req/s)');
  console.log('');

  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log('');
  console.log('=== Spike Test Complete ===');
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);
  console.log('');
  console.log('Expected results:');
  console.log('  ✓ System handles 20x spike');
  console.log('  ✓ Quick recovery after spike');
  console.log('  ✓ No lingering performance issues');
}
