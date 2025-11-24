package parser

import (
	"testing"
)

// ================================================================================
// Parser Benchmarks (TN-151 Phase 2E)
// ================================================================================
// Performance benchmarks for JSON/YAML parsers
//
// Target: < 10ms p95 for typical configs
// Quality Target: 150% (Grade A+ EXCEPTIONAL)

var (
	smallConfig = []byte(`{
  "global": {"resolve_timeout": "5m"},
  "route": {"receiver": "default"},
  "receivers": [{"name": "default"}]
}`)

	mediumConfig = []byte(`{
  "global": {
    "resolve_timeout": "5m",
    "http_config": {"follow_redirects": true},
    "smtp_smarthost": "smtp.example.com:587",
    "smtp_from": "alertmanager@example.com"
  },
  "route": {
    "receiver": "default",
    "group_by": ["alertname", "cluster"],
    "group_wait": "30s",
    "group_interval": "5m",
    "repeat_interval": "4h",
    "routes": [
      {"receiver": "team-a", "matchers": ["team=team-a"]},
      {"receiver": "team-b", "matchers": ["team=team-b"]}
    ]
  },
  "receivers": [
    {"name": "default", "webhook_configs": [{"url": "https://example.com/webhook"}]},
    {"name": "team-a", "slack_configs": [{"api_url": "https://hooks.slack.com/services/T/A", "channel": "#alerts"}]},
    {"name": "team-b", "email_configs": [{"to": "team-b@example.com"}]}
  ],
  "inhibit_rules": [
    {"source_matchers": ["severity=critical"], "target_matchers": ["severity=warning"], "equal": ["alertname"]}
  ]
}`)

	largeConfig = []byte(`{
  "global": {
    "resolve_timeout": "5m",
    "http_config": {"follow_redirects": true},
    "smtp_smarthost": "smtp.example.com:587",
    "smtp_from": "alertmanager@example.com",
    "smtp_require_tls": true,
    "slack_api_url": "https://hooks.slack.com/services/ORG/KEY",
    "pagerduty_url": "https://events.pagerduty.com/v2/enqueue",
    "opsgenie_api_url": "https://api.opsgenie.com/"
  },
  "route": {
    "receiver": "default",
    "group_by": ["alertname", "cluster", "service"],
    "group_wait": "30s",
    "group_interval": "5m",
    "repeat_interval": "4h",
    "routes": [
      {
        "receiver": "critical-alerts",
        "matchers": ["severity=critical"],
        "group_wait": "10s",
        "repeat_interval": "1h"
      },
      {
        "receiver": "database-team",
        "matchers": ["service=~database.*"],
        "group_by": ["alertname", "instance"],
        "routes": [
          {"receiver": "dba-oncall", "matchers": ["severity=critical"]},
          {"receiver": "dba-team", "matchers": ["severity=warning"]}
        ]
      },
      {
        "receiver": "frontend-team",
        "matchers": ["service=~frontend.*"],
        "group_by": ["alertname", "instance"]
      },
      {
        "receiver": "backend-team",
        "matchers": ["service=~backend.*"],
        "group_by": ["alertname", "instance"]
      }
    ]
  },
  "receivers": [
    {"name": "default", "webhook_configs": [{"url": "https://example.com/webhook"}]},
    {"name": "critical-alerts", "pagerduty_configs": [{"service_key": "key1"}], "slack_configs": [{"api_url": "https://hooks.slack.com/critical", "channel": "#critical"}]},
    {"name": "database-team", "slack_configs": [{"api_url": "https://hooks.slack.com/db", "channel": "#database"}]},
    {"name": "dba-oncall", "pagerduty_configs": [{"service_key": "dba-key"}]},
    {"name": "dba-team", "slack_configs": [{"api_url": "https://hooks.slack.com/dba", "channel": "#dba"}]},
    {"name": "frontend-team", "slack_configs": [{"api_url": "https://hooks.slack.com/fe", "channel": "#frontend"}], "email_configs": [{"to": "frontend@example.com"}]},
    {"name": "backend-team", "slack_configs": [{"api_url": "https://hooks.slack.com/be", "channel": "#backend"}], "email_configs": [{"to": "backend@example.com"}]}
  ],
  "inhibit_rules": [
    {"source_matchers": ["severity=critical"], "target_matchers": ["severity=warning"], "equal": ["alertname", "instance"]},
    {"source_matchers": ["alertname=HostDown"], "target_matchers": ["alertname=HostNetworkDown"], "equal": ["instance"]},
    {"source_matchers": ["alertname=ServiceDown"], "target_matchers": ["alertname=HighErrorRate"], "equal": ["service", "instance"]}
  ]
}`)
)

// Benchmark JSON Parser with small config
func BenchmarkJSONParser_Small(b *testing.B) {
	parser := NewJSONParser(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(smallConfig)
	}
}

// Benchmark JSON Parser with medium config
func BenchmarkJSONParser_Medium(b *testing.B) {
	parser := NewJSONParser(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(mediumConfig)
	}
}

// Benchmark JSON Parser with large config
func BenchmarkJSONParser_Large(b *testing.B) {
	parser := NewJSONParser(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(largeConfig)
	}
}

// Benchmark YAML Parser with small config (convert JSON to YAML-like for testing)
func BenchmarkYAMLParser_Small(b *testing.B) {
	yamlConfig := []byte(`
global:
  resolve_timeout: 5m
route:
  receiver: default
receivers:
  - name: default
`)
	parser := NewYAMLParser(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(yamlConfig)
	}
}

// Benchmark Multi-Format Parser (JSON auto-detection)
func BenchmarkMultiFormatParser_JSON(b *testing.B) {
	parser := NewMultiFormatParser(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(mediumConfig)
	}
}

// Benchmark Multi-Format Parser (YAML auto-detection)
func BenchmarkMultiFormatParser_YAML(b *testing.B) {
	yamlConfig := []byte(`
global:
  resolve_timeout: 5m
route:
  receiver: default
  group_by: [alertname, cluster]
receivers:
  - name: default
    webhook_configs:
      - url: https://example.com/webhook
`)
	parser := NewMultiFormatParser(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(yamlConfig)
	}
}

// Benchmark memory allocations
func BenchmarkJSONParser_Allocations(b *testing.B) {
	parser := NewJSONParser(false)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parser.Parse(mediumConfig)
	}
}

