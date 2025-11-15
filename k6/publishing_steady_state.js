// publishing_steady_state.js - Steady state load test for Publishing System
// Target: Baseline performance under sustained load
// Duration: 5 minutes
// VUs: 100
// Expected: p95 < 10ms, 0 errors

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const healthCheckDuration = new Trend('health_check_duration');
const metricsCheckDuration = new Trend('metrics_check_duration');
const targetDiscoveryDuration = new Trend('target_discovery_duration');
const successfulRequests = new Counter('successful_requests');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 100 },  // Ramp up to 100 VUs
    { duration: '4m', target: 100 },   // Stay at 100 VUs for 4 minutes
    { duration: '30s', target: 0 },    // Ramp down to 0 VUs
  ],
  thresholds: {
    'http_req_duration': ['p(95)<10'],     // 95% of requests must complete below 10ms
    'http_req_failed': ['rate<0.01'],      // Error rate must be below 1%
    'errors': ['rate<0.01'],               // Custom error rate below 1%
    'health_check_duration': ['p(95)<10'], // Health checks below 10ms
    'metrics_check_duration': ['p(95)<10'], // Metrics below 10ms
  },
};

// Base URL (adjust for your environment)
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test data
const testTargets = [
  'rootly-prod',
  'pagerduty-oncall',
  'slack-alerts',
  'webhook-default',
];

export default function () {
  // Scenario 1: Check health of all targets (50% of requests)
  if (Math.random() < 0.5) {
    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/v2/publishing/targets/health`);
    const duration = Date.now() - startTime;

    const success = check(response, {
      'health check status is 200': (r) => r.status === 200,
      'health check has targets': (r) => {
        try {
          const body = JSON.parse(r.body);
          return Array.isArray(body) && body.length > 0;
        } catch (e) {
          return false;
        }
      },
    });

    healthCheckDuration.add(duration);
    errorRate.add(!success);
    if (success) successfulRequests.add(1);
  }

  // Scenario 2: Check specific target health (30% of requests)
  else if (Math.random() < 0.6) {
    const target = testTargets[Math.floor(Math.random() * testTargets.length)];
    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/v2/publishing/targets/health/${target}`);
    const duration = Date.now() - startTime;

    const success = check(response, {
      'target health status is 200 or 404': (r) => r.status === 200 || r.status === 404,
      'target health has valid structure': (r) => {
        if (r.status === 404) return true;
        try {
          const body = JSON.parse(r.body);
          return body.target_name && body.status;
        } catch (e) {
          return false;
        }
      },
    });

    healthCheckDuration.add(duration);
    errorRate.add(!success);
    if (success) successfulRequests.add(1);
  }

  // Scenario 3: Get publishing metrics (20% of requests)
  else {
    const startTime = Date.now();
    const response = http.get(`${BASE_URL}/api/v2/publishing/metrics`);
    const duration = Date.now() - startTime;

    const success = check(response, {
      'metrics status is 200': (r) => r.status === 200,
      'metrics has data': (r) => {
        try {
          const body = JSON.parse(r.body);
          return body.metrics && typeof body.metrics === 'object';
        } catch (e) {
          return false;
        }
      },
    });

    metricsCheckDuration.add(duration);
    errorRate.add(!success);
    if (success) successfulRequests.add(1);
  }

  // Small sleep to simulate realistic user behavior
  sleep(0.1);
}

export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'summary_steady_state.json': JSON.stringify(data),
  };
}

function textSummary(data, options) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = '\n';
  summary += indent + '='.repeat(60) + '\n';
  summary += indent + 'Publishing System - Steady State Load Test Results\n';
  summary += indent + '='.repeat(60) + '\n\n';

  summary += indent + 'Test Configuration:\n';
  summary += indent + '  Duration: 5 minutes\n';
  summary += indent + '  VUs: 100 (sustained)\n';
  summary += indent + '  Target: 1000 req/s\n\n';

  summary += indent + 'Performance Metrics:\n';
  summary += indent + `  Total Requests: ${data.metrics.http_reqs.values.count}\n`;
  summary += indent + `  Success Rate: ${(100 - data.metrics.http_req_failed.values.rate * 100).toFixed(2)}%\n`;
  summary += indent + `  Avg Duration: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += indent + `  P95 Duration: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += indent + `  P99 Duration: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n\n`;

  summary += indent + 'Thresholds:\n';
  const thresholds = data.metrics.http_req_duration.thresholds;
  for (const [name, result] of Object.entries(thresholds || {})) {
    const status = result.ok ? '✓ PASS' : '✗ FAIL';
    summary += indent + `  ${status}: ${name}\n`;
  }

  return summary;
}

