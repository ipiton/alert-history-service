package inhibition

import (
	"context"
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// --- Mock ActiveAlertCache for testing ---

type mockCache struct {
	firingAlerts []*core.Alert
	err          error
}

func (m *mockCache) GetFiringAlerts(ctx context.Context) ([]*core.Alert, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.firingAlerts, nil
}

func (m *mockCache) AddFiringAlert(ctx context.Context, alert *core.Alert) error {
	m.firingAlerts = append(m.firingAlerts, alert)
	return nil
}

func (m *mockCache) RemoveAlert(ctx context.Context, fingerprint string) error {
	return nil
}

// --- Test Helpers ---

func createTestAlert(name, severity, node, cluster string) *core.Alert {
	return &core.Alert{
		AlertName:   name,
		Fingerprint: fmt.Sprintf("fp-%s-%s", name, node),
		Labels: map[string]string{
			"alertname": name,
			"severity":  severity,
			"node":      node,
			"cluster":   cluster,
		},
		Status: "firing",
	}
}

func createTestRule(name string) InhibitionRule {
	rule := InhibitionRule{
		Name: name,
		SourceMatch: map[string]string{
			"alertname": "NodeDown",
			"severity":  "critical",
		},
		TargetMatch: map[string]string{
			"alertname": "InstanceDown",
		},
		Equal: []string{"node", "cluster"},
	}
	rule.compiledSourceRE = make(map[string]*regexp.Regexp)
	rule.compiledTargetRE = make(map[string]*regexp.Regexp)
	return rule
}

// --- Happy Path Tests ---

func TestShouldInhibit_BasicMatch(t *testing.T) {
	// Setup
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	// Execute
	result, err := matcher.ShouldInhibit(context.Background(), targetAlert)

	// Assert
	if err != nil {
		t.Fatalf("ShouldInhibit() error = %v", err)
	}

	if !result.Matched {
		t.Error("Expected match, got no match")
	}

	if result.InhibitedBy == nil {
		t.Error("Expected InhibitedBy to be set")
	}

	if result.InhibitedBy.AlertName != "NodeDown" {
		t.Errorf("Expected inhibitor to be NodeDown, got %s", result.InhibitedBy.AlertName)
	}

	if result.Rule == nil {
		t.Error("Expected Rule to be set")
	}
}

func TestShouldInhibit_NoMatch_DifferentNode(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node2", "prod") // Different node

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	result, err := matcher.ShouldInhibit(context.Background(), targetAlert)

	if err != nil {
		t.Fatalf("ShouldInhibit() error = %v", err)
	}

	if result.Matched {
		t.Error("Expected no match (different node), got match")
	}
}

func TestShouldInhibit_NoMatch_DifferentAlertName(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("DiskFull", "warning", "node1", "prod") // Different alertname

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	result, err := matcher.ShouldInhibit(context.Background(), targetAlert)

	if err != nil {
		t.Fatalf("ShouldInhibit() error = %v", err)
	}

	if result.Matched {
		t.Error("Expected no match (different alertname), got match")
	}
}

func TestShouldInhibit_NoFiringAlerts(t *testing.T) {
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		firingAlerts: []*core.Alert{}, // Empty
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	result, err := matcher.ShouldInhibit(context.Background(), targetAlert)

	if err != nil {
		t.Fatalf("ShouldInhibit() error = %v", err)
	}

	if result.Matched {
		t.Error("Expected no match (no firing alerts), got match")
	}
}

func TestShouldInhibit_CacheError(t *testing.T) {
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		err: fmt.Errorf("cache error"),
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	_, err := matcher.ShouldInhibit(context.Background(), targetAlert)

	if err == nil {
		t.Error("Expected error from cache, got nil")
	}
}

func TestShouldInhibit_MultipleRules_FirstMatch(t *testing.T) {
	sourceAlert1 := createTestAlert("NodeDown", "critical", "node1", "prod")
	sourceAlert2 := createTestAlert("ClusterDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert1, sourceAlert2},
	}

	rule1 := createTestRule("rule1")
	rule2 := createTestRule("rule2")
	rule2.SourceMatch = map[string]string{"alertname": "ClusterDown"}

	matcher := NewMatcher(cache, []InhibitionRule{rule1, rule2}, nil)

	result, err := matcher.ShouldInhibit(context.Background(), targetAlert)

	if err != nil {
		t.Fatalf("ShouldInhibit() error = %v", err)
	}

	if !result.Matched {
		t.Error("Expected match, got no match")
	}

	// Should match first rule (NodeDown)
	if result.InhibitedBy.AlertName != "NodeDown" {
		t.Errorf("Expected inhibitor to be NodeDown (first match), got %s", result.InhibitedBy.AlertName)
	}
}

func TestFindInhibitors_MultipleMatches(t *testing.T) {
	sourceAlert1 := createTestAlert("NodeDown", "critical", "node1", "prod")
	sourceAlert2 := createTestAlert("NodeDown", "critical", "node1", "prod") // Duplicate
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert1, sourceAlert2},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	results, err := matcher.FindInhibitors(context.Background(), targetAlert)

	if err != nil {
		t.Fatalf("FindInhibitors() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 inhibitors, got %d", len(results))
	}
}

func TestFindInhibitors_NoMatches(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node2", "prod") // Different node

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	results, err := matcher.FindInhibitors(context.Background(), targetAlert)

	if err != nil {
		t.Fatalf("FindInhibitors() error = %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 inhibitors, got %d", len(results))
	}
}

// --- MatchRule Tests ---

func TestMatchRule_ExactMatch(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	rule := createTestRule("test-rule")
	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if !matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected rule to match")
	}
}

func TestMatchRule_SourceMismatch(t *testing.T) {
	sourceAlert := createTestAlert("DiskFull", "warning", "node1", "prod") // Wrong alertname
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	rule := createTestRule("test-rule")
	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected no match (source mismatch)")
	}
}

func TestMatchRule_TargetMismatch(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("DiskFull", "warning", "node1", "prod") // Wrong alertname

	rule := createTestRule("test-rule")
	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected no match (target mismatch)")
	}
}

func TestMatchRule_EqualLabelMismatch(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node2", "prod") // Different node

	rule := createTestRule("test-rule")
	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected no match (equal label mismatch)")
	}
}

func TestMatchRule_MissingEqualLabel(t *testing.T) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := &core.Alert{
		AlertName:   "InstanceDown",
		Fingerprint: "fp-test",
		Labels: map[string]string{
			"alertname": "InstanceDown",
			// Missing "node" label
			"cluster": "prod",
		},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected no match (missing equal label)")
	}
}

func TestMatchRule_RegexMatch(t *testing.T) {
	sourceAlert := &core.Alert{
		AlertName:   "ServiceDown",
		Fingerprint: "fp-service",
		Labels: map[string]string{
			"alertname": "ServiceDown",
			"service":   "api-backend", // Matches ^api.*
			"cluster":   "prod",
		},
	}

	targetAlert := &core.Alert{
		AlertName:   "HighLatency",
		Fingerprint: "fp-latency",
		Labels: map[string]string{
			"alertname": "HighLatency",
			"severity":  "warning", // Matches warning|info
			"cluster":   "prod",
		},
	}

	rule := InhibitionRule{
		Name:          "regex-rule",
		SourceMatchRE: map[string]string{"service": "^api.*"},
		TargetMatchRE: map[string]string{"severity": "warning|info"},
		Equal:         []string{"cluster"},
	}

	// Compile regex
	rule.setCompiledSourceRE("service", regexp.MustCompile("^api.*"))
	rule.setCompiledTargetRE("severity", regexp.MustCompile("warning|info"))

	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if !matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected regex rule to match")
	}
}

func TestMatchRule_RegexNoMatch(t *testing.T) {
	sourceAlert := &core.Alert{
		AlertName:   "ServiceDown",
		Fingerprint: "fp-service",
		Labels: map[string]string{
			"alertname": "ServiceDown",
			"service":   "database", // Does NOT match ^api.*
			"cluster":   "prod",
		},
	}

	targetAlert := &core.Alert{
		AlertName:   "HighLatency",
		Fingerprint: "fp-latency",
		Labels: map[string]string{
			"alertname": "HighLatency",
			"severity":  "warning",
			"cluster":   "prod",
		},
	}

	rule := InhibitionRule{
		Name:          "regex-rule",
		SourceMatchRE: map[string]string{"service": "^api.*"},
		TargetMatchRE: map[string]string{"severity": "warning|info"},
		Equal:         []string{"cluster"},
	}

	rule.setCompiledSourceRE("service", regexp.MustCompile("^api.*"))
	rule.setCompiledTargetRE("severity", regexp.MustCompile("warning|info"))

	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	if matcher.MatchRule(&rule, sourceAlert, targetAlert) {
		t.Error("Expected no match (regex mismatch)")
	}
}

// --- Performance Tests ---

func TestShouldInhibit_Performance(t *testing.T) {
	// Create 100 firing alerts
	firingAlerts := make([]*core.Alert, 100)
	for i := 0; i < 100; i++ {
		firingAlerts[i] = createTestAlert(fmt.Sprintf("Alert%d", i), "warning", fmt.Sprintf("node%d", i), "prod")
	}

	cache := &mockCache{
		firingAlerts: firingAlerts,
	}

	// Create 10 rules
	rules := make([]InhibitionRule, 10)
	for i := 0; i < 10; i++ {
		rules[i] = createTestRule(fmt.Sprintf("rule%d", i))
	}

	matcher := NewMatcher(cache, rules, nil)

	targetAlert := createTestAlert("TargetAlert", "info", "node99", "prod")

	start := time.Now()
	_, err := matcher.ShouldInhibit(context.Background(), targetAlert)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("ShouldInhibit() error = %v", err)
	}

	// Target: <1ms
	if duration > time.Millisecond {
		t.Errorf("Performance target missed: took %v (target < 1ms)", duration)
	}

	t.Logf("Performance: %v for 100 alerts x 10 rules", duration)
}

// --- Benchmarks ---

func BenchmarkShouldInhibit_SingleRule(b *testing.B) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = matcher.ShouldInhibit(ctx, targetAlert)
	}
}

func BenchmarkShouldInhibit_100Alerts_10Rules(b *testing.B) {
	// Create 100 firing alerts
	firingAlerts := make([]*core.Alert, 100)
	for i := 0; i < 100; i++ {
		firingAlerts[i] = createTestAlert(fmt.Sprintf("Alert%d", i), "warning", fmt.Sprintf("node%d", i), "prod")
	}

	cache := &mockCache{
		firingAlerts: firingAlerts,
	}

	// Create 10 rules
	rules := make([]InhibitionRule, 10)
	for i := 0; i < 10; i++ {
		rules[i] = createTestRule(fmt.Sprintf("rule%d", i))
	}

	matcher := NewMatcher(cache, rules, nil)
	targetAlert := createTestAlert("TargetAlert", "info", "node99", "prod")

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = matcher.ShouldInhibit(ctx, targetAlert)
	}
}

func BenchmarkMatchRule(b *testing.B) {
	sourceAlert := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	rule := createTestRule("test-rule")
	matcher := NewMatcher(&mockCache{}, []InhibitionRule{rule}, nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = matcher.MatchRule(&rule, sourceAlert, targetAlert)
	}
}

func BenchmarkFindInhibitors(b *testing.B) {
	sourceAlert1 := createTestAlert("NodeDown", "critical", "node1", "prod")
	sourceAlert2 := createTestAlert("NodeDown", "critical", "node1", "prod")
	targetAlert := createTestAlert("InstanceDown", "warning", "node1", "prod")

	cache := &mockCache{
		firingAlerts: []*core.Alert{sourceAlert1, sourceAlert2},
	}

	rule := createTestRule("test-rule")
	matcher := NewMatcher(cache, []InhibitionRule{rule}, nil)

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = matcher.FindInhibitors(ctx, targetAlert)
	}
}
