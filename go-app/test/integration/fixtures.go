//go:build integration
// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Fixtures provides access to test data
type Fixtures struct {
	basePath string
}

// NewFixtures creates fixtures loader
func NewFixtures() *Fixtures {
	// Get fixtures path relative to test files
	basePath := filepath.Join("../fixtures")
	return &Fixtures{basePath: basePath}
}

// LoadAlerts loads alerts from fixtures file
func (f *Fixtures) LoadAlerts(filename string) ([]*Alert, error) {
	path := filepath.Join(f.basePath, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixtures file %s: %w", path, err)
	}

	var alerts []*Alert
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alerts: %w", err)
	}

	return alerts, nil
}

// LoadSilences loads silences from fixtures file
func (f *Fixtures) LoadSilences(filename string) ([]*Silence, error) {
	path := filepath.Join(f.basePath, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixtures file %s: %w", path, err)
	}

	var silences []*Silence
	if err := json.Unmarshal(data, &silences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal silences: %w", err)
	}

	return silences, nil
}

// LoadWebhookPayload loads webhook payload from fixtures
func (f *Fixtures) LoadWebhookPayload(filename string) (map[string]interface{}, error) {
	path := filepath.Join(f.basePath, filename)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read fixtures file %s: %w", path, err)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal webhook payload: %w", err)
	}

	return payload, nil
}

// --- Builder Functions for Test Data ---

// NewTestAlert creates a test alert with default values
func NewTestAlert(name string) *Alert {
	now := time.Now()
	return &Alert{
		Fingerprint: fmt.Sprintf("fp_%s_%d", name, now.Unix()),
		AlertName:   name,
		Status:      "firing",
		Severity:    "warning",
		Namespace:   "default",
		Labels: map[string]string{
			"alertname": name,
			"severity":  "warning",
		},
		Annotations: map[string]string{
			"summary": fmt.Sprintf("Test alert %s", name),
		},
		StartsAt:  now,
		CreatedAt: now,
	}
}

// WithStatus sets alert status
func (a *Alert) WithStatus(status string) *Alert {
	a.Status = status
	return a
}

// WithSeverity sets alert severity
func (a *Alert) WithSeverity(severity string) *Alert {
	a.Severity = severity
	a.Labels["severity"] = severity
	return a
}

// WithNamespace sets alert namespace
func (a *Alert) WithNamespace(namespace string) *Alert {
	a.Namespace = namespace
	a.Labels["namespace"] = namespace
	return a
}

// WithLabel adds a label to alert
func (a *Alert) WithLabel(key, value string) *Alert {
	a.Labels[key] = value
	return a
}

// WithAnnotation adds an annotation to alert
func (a *Alert) WithAnnotation(key, value string) *Alert {
	a.Annotations[key] = value
	return a
}

// WithEndsAt sets alert end time
func (a *Alert) WithEndsAt(endsAt time.Time) *Alert {
	a.EndsAt = &endsAt
	return a
}

// NewTestSilence creates a test silence with default values
func NewTestSilence() *Silence {
	now := time.Now()
	return &Silence{
		ID:        fmt.Sprintf("silence_%d", now.Unix()),
		CreatedBy: "test@example.com",
		Comment:   "Test silence",
		StartsAt:  now,
		EndsAt:    now.Add(1 * time.Hour),
		Matchers: []map[string]string{
			{"name": "alertname", "value": "TestAlert", "isRegex": "false"},
		},
	}
}

// WithMatcher adds a matcher to silence
func (s *Silence) WithMatcher(name, value string, isRegex bool) *Silence {
	s.Matchers = append(s.Matchers, map[string]string{
		"name":    name,
		"value":   value,
		"isRegex": fmt.Sprintf("%t", isRegex),
	})
	return s
}

// WithComment sets silence comment
func (s *Silence) WithComment(comment string) *Silence {
	s.Comment = comment
	return s
}

// WithDuration sets silence duration
func (s *Silence) WithDuration(duration time.Duration) *Silence {
	s.EndsAt = s.StartsAt.Add(duration)
	return s
}

// --- Alertmanager Webhook Builder ---

// AlertmanagerWebhook creates Alertmanager webhook payload
type AlertmanagerWebhook struct {
	Alerts []AlertmanagerAlert `json:"alerts"`
}

// AlertmanagerAlert represents alert in Alertmanager format
type AlertmanagerAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"startsAt"`
	EndsAt      *time.Time        `json:"endsAt,omitempty"`
	Status      string            `json:"status"`
}

// NewAlertmanagerWebhook creates Alertmanager webhook payload
func NewAlertmanagerWebhook() *AlertmanagerWebhook {
	return &AlertmanagerWebhook{
		Alerts: make([]AlertmanagerAlert, 0),
	}
}

// AddAlert adds alert to webhook
func (w *AlertmanagerWebhook) AddAlert(alert AlertmanagerAlert) *AlertmanagerWebhook {
	w.Alerts = append(w.Alerts, alert)
	return w
}

// AddFiringAlert adds firing alert to webhook
func (w *AlertmanagerWebhook) AddFiringAlert(name, severity string) *AlertmanagerWebhook {
	now := time.Now()
	w.Alerts = append(w.Alerts, AlertmanagerAlert{
		Labels: map[string]string{
			"alertname": name,
			"severity":  severity,
			"namespace": "default",
		},
		Annotations: map[string]string{
			"summary": fmt.Sprintf("Test alert %s", name),
		},
		StartsAt: now,
		Status:   "firing",
	})
	return w
}

// AddResolvedAlert adds resolved alert to webhook
func (w *AlertmanagerWebhook) AddResolvedAlert(name, severity string) *AlertmanagerWebhook {
	now := time.Now()
	endsAt := now.Add(1 * time.Minute)
	w.Alerts = append(w.Alerts, AlertmanagerAlert{
		Labels: map[string]string{
			"alertname": name,
			"severity":  severity,
			"namespace": "default",
		},
		Annotations: map[string]string{
			"summary": fmt.Sprintf("Test alert %s", name),
		},
		StartsAt: now.Add(-10 * time.Minute),
		EndsAt:   &endsAt,
		Status:   "resolved",
	})
	return w
}

// --- Prometheus v2 Webhook Builder ---

// PrometheusWebhook creates Prometheus webhook payload
type PrometheusWebhook struct {
	Data PrometheusData `json:"data"`
}

// PrometheusData represents Prometheus data format
type PrometheusData struct {
	Alerts []PrometheusAlert `json:"alerts"`
}

// PrometheusAlert represents alert in Prometheus format
type PrometheusAlert struct {
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	State       string            `json:"state"`
	ActiveAt    time.Time         `json:"activeAt"`
	Value       string            `json:"value"`
}

// NewPrometheusWebhook creates Prometheus webhook payload
func NewPrometheusWebhook() *PrometheusWebhook {
	return &PrometheusWebhook{
		Data: PrometheusData{
			Alerts: make([]PrometheusAlert, 0),
		},
	}
}

// AddFiringAlert adds firing alert
func (w *PrometheusWebhook) AddFiringAlert(name, severity string) *PrometheusWebhook {
	w.Data.Alerts = append(w.Data.Alerts, PrometheusAlert{
		Labels: map[string]string{
			"alertname": name,
			"severity":  severity,
		},
		Annotations: map[string]string{
			"summary": fmt.Sprintf("Test alert %s", name),
		},
		State:    "firing",
		ActiveAt: time.Now(),
		Value:    "1",
	})
	return w
}

// --- Common Test Scenarios ---

// GetCriticalAlertsScenario returns multiple critical alerts
func GetCriticalAlertsScenario() []*Alert {
	return []*Alert{
		NewTestAlert("HighMemoryUsage").WithSeverity("critical").WithNamespace("production"),
		NewTestAlert("DiskSpaceLow").WithSeverity("critical").WithNamespace("production"),
		NewTestAlert("ServiceDown").WithSeverity("critical").WithNamespace("production"),
	}
}

// GetMixedAlertsScenario returns alerts with mixed severities
func GetMixedAlertsScenario() []*Alert {
	return []*Alert{
		NewTestAlert("HighCPU").WithSeverity("warning"),
		NewTestAlert("HighMemory").WithSeverity("critical"),
		NewTestAlert("NetworkLatency").WithSeverity("info"),
		NewTestAlert("DiskFull").WithSeverity("critical"),
		NewTestAlert("SlowQuery").WithSeverity("warning"),
	}
}

// GetResolvedAlertsScenario returns resolved alerts
func GetResolvedAlertsScenario() []*Alert {
	now := time.Now()
	endsAt := now.Add(-5 * time.Minute)
	return []*Alert{
		NewTestAlert("PastAlert1").WithStatus("resolved").WithEndsAt(endsAt),
		NewTestAlert("PastAlert2").WithStatus("resolved").WithEndsAt(endsAt),
	}
}
