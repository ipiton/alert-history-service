/**
 * TN-061: Webhook Endpoint - Stress Test
 * 
 * Scenario: Find breaking point of the system
 * Pattern: Gradually increase load until failure
 * Purpose: Identify maximum capacity and failure modes
 * 
 * Performance Targets (150% Quality):
 * - Find maximum sustainable throughput
 * - Graceful degradation (no crashes)
 * - Identify bottlenecks
 * - System recovers after stress
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('error_rate');
const timeoutRate = new Rate('timeout_rate');
const rateLimitRate = new Rate('rate_limit_rate');
const webhookDuration = new Trend('webhook_duration');
const maxThroughput = new Counter('max_throughput');

// Test configuration
export const options = {
  scenarios: {
    stress_test: {
      executor: 'ramping-arrival-rate',
      startRate: 1000,
      timeUnit: '1s',
      preAllocatedVUs: 100,
      maxVUs: 2000,
      stages: [
        { duration: '2m', target: 1000 },   // Warmup: 1K req/s
        { duration: '2m', target: 5000 },   // 5K req/s
        { duration: '2m', target: 10000 },  // 10K req/s (target)
        { duration: '2m', target: 15000 },  // 15K req/s (150%)
        { duration: '2m', target: 20000 },  // 20K req/s (200%)
        { duration: '2m', target: 30000 },  // 30K req/s (300%)
        { duration: '2m', target: 40000 },  // 40K req/s (400%)
        { duration: '1m', target: 50000 },  // 50K req/s (500%)
        { duration: '2m', target: 1000 },   // Recovery: back to 1K
      ],
    },
  },
  thresholds: {
    // Relaxed thresholds for stress test
    'http_req_duration': ['p(95)<50', 'p(99)<100'],
    'error_rate': ['rate<0.5'], // Allow 50% error rate at peak
    
    // Track specific error types
    'timeout_rate': ['rate<0.1'],
    'rate_limit_rate': ['rate<0.4'],
  },
};

// Test data: Varied payload sizes
const payloads = [
  // Small payload (1 alert)
  JSON.stringify({
    alerts: [{
      status: 'firing',
      labels: { alertname: 'StressTest1', severity: 'info' },
      startsAt: new Date().toISOString(),
    }],
  }),
  
  // Medium payload (5 alerts)
  JSON.stringify({
    alerts: Array(5).fill(null).map((_, i) => ({
      status: 'firing',
      labels: { alertname: `StressTest${i}`, severity: 'warning' },
      startsAt: new Date().toISOString(),
    })),
  }),
  
  // Large payload (20 alerts)
  JSON.stringify({
    alerts: Array(20).fill(null).map((_, i) => ({
      status: 'firing',
      labels: { alertname: `StressTest${i}`, severity: 'critical' },
      annotations: { summary: `Stress test alert ${i}` },
      startsAt: new Date().toISOString(),
    })),
  }),
];

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

let currentStage = 0;
let maxSuccessfulRate = 0;

export default function () {
  const headers = {
    'Content-Type': 'application/json',
  };

  if (API_KEY) {
    headers['X-API-Key'] = API_KEY;
  }

  // Randomly select payload size
  const payload = payloads[Math.floor(Math.random() * payloads.length)];

  const response = http.post(
    `${BASE_URL}/webhook`,
    payload,
    {
      headers: headers,
      timeout: '30s',
      tags: { type: 'webhook', stage: `stage_${currentStage}` },
    }
  );

  webhookDuration.add(response.timings.duration, { stage: currentStage });

  // Categorize response
  const isSuccess = response.status === 200 || response.status === 207;
  const isRateLimit = response.status === 429;
  const isTimeout = response.status === 0 || response.timings.duration > 30000;
  const isError = !isSuccess && !isRateLimit && !isTimeout;

  // Record metrics
  errorRate.add(isError ? 1 : 0);
  rateLimitRate.add(isRateLimit ? 1 : 0);
  timeoutRate.add(isTimeout ? 1 : 0);

  if (isSuccess) {
    maxThroughput.add(1);
  }

  // Detailed logging for failures
  if (!isSuccess && Math.random() < 0.01) { // Sample 1% of failures
    console.warn(`Failure at stage ${currentStage}: ${response.status} - ${response.body?.substring(0, 100)}`);
  }

  // Track max successful rate
  const currentRate = __ITER / (__VU || 1);
  if (isSuccess && currentRate > maxSuccessfulRate) {
    maxSuccessfulRate = currentRate;
  }

  sleep(0.001);
}

export function setup() {
  console.log('=== Webhook Stress Test ===');
  console.log('Purpose: Find system breaking point');
  console.log('Pattern: 1K → 50K req/s (gradual increase)');
  console.log(`Base URL: ${BASE_URL}`);
  console.log('');
  console.log('Load stages:');
  console.log('  Stage 0: 1K req/s (baseline)');
  console.log('  Stage 1: 5K req/s');
  console.log('  Stage 2: 10K req/s (target)');
  console.log('  Stage 3: 15K req/s (150%)');
  console.log('  Stage 4: 20K req/s (200%)');
  console.log('  Stage 5: 30K req/s (300%)');
  console.log('  Stage 6: 40K req/s (400%)');
  console.log('  Stage 7: 50K req/s (500%)');
  console.log('  Stage 8: 1K req/s (recovery)');
  console.log('');
  
  return { startTime: new Date().toISOString() };
}

export function teardown(data) {
  console.log('');
  console.log('=== Stress Test Complete ===');
  console.log(`Started: ${data.startTime}`);
  console.log(`Ended: ${new Date().toISOString()}`);
  console.log(`Max successful rate observed: ~${Math.round(maxSuccessfulRate)} req/s`);
  console.log('');
  console.log('Analysis:');
  console.log('  - Identify stage where errors increase significantly');
  console.log('  - Check for graceful degradation (429 vs 500/timeout)');
  console.log('  - Verify recovery in final stage');
  console.log('  - Review metrics for bottlenecks');
}

