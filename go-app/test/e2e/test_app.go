//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/vitaliisemenov/alert-history/test/integration"
)

// TestApplication represents minimal application server for E2E tests
type TestApplication struct {
	Server        *httptest.Server
	Infrastructure *integration.TestInfrastructure
	mu            sync.Mutex
	webhooks      []map[string]interface{}
}

// StartTestApplication creates and starts test application server
func StartTestApplication(ctx context.Context, infra *integration.TestInfrastructure) (*TestApplication, error) {
	app := &TestApplication{
		Infrastructure: infra,
		webhooks:       make([]map[string]interface{}, 0),
	}

	// Run database migrations
	if err := app.runMigrations(ctx); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Create HTTP handler
	mux := http.NewServeMux()

	// Webhook endpoint - main E2E target
	mux.HandleFunc("/api/v2/webhook", app.handleWebhook)
	mux.HandleFunc("/webhook", app.handleWebhook) // Alias

	// Health endpoint
	mux.HandleFunc("/healthz", app.handleHealth)
	mux.HandleFunc("/health", app.handleHealth) // Alias

	// Metrics endpoint
	mux.HandleFunc("/metrics", app.handleMetrics)

	// History API endpoints
	mux.HandleFunc("/api/v2/history", app.handleHistory)
	mux.HandleFunc("/api/v2/history/", app.handleHistoryDetail)

	// Publishing endpoints
	mux.HandleFunc("/api/v2/publishing/targets", app.handlePublishingTargets)
	mux.HandleFunc("/api/v2/publishing/stats", app.handlePublishingStats)

	// Start test server
	app.Server = httptest.NewServer(mux)

	// Update infrastructure BaseURL to point to test server
	infra.BaseURL = app.Server.URL

	return app, nil
}

// handleWebhook processes Alertmanager webhook
func (app *TestApplication) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		fmt.Printf("[TEST_APP] Failed to decode JSON: %v\n", err)
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	fmt.Printf("[TEST_APP] Received webhook payload: %+v\n", payload)

	// Store webhook for verification
	app.mu.Lock()
	app.webhooks = append(app.webhooks, payload)
	app.mu.Unlock()

	// Extract alerts from payload
	alerts, ok := payload["alerts"].([]interface{})
	if !ok || len(alerts) == 0 {
		fmt.Printf("[TEST_APP] No alerts in payload\n")
		http.Error(w, "No alerts in payload", http.StatusBadRequest)
		return
	}

	fmt.Printf("[TEST_APP] Processing %d alerts\n", len(alerts))

	// Process each alert
	fingerprints := make([]string, 0, len(alerts))
	for idx, alertData := range alerts {
		alert, ok := alertData.(map[string]interface{})
		if !ok {
			fmt.Printf("[TEST_APP] Alert %d: invalid type\n", idx)
			continue
		}

		// Generate fingerprint
		fingerprint := fmt.Sprintf("fp_%d", time.Now().UnixNano())
		fingerprints = append(fingerprints, fingerprint)

		// Extract alert name
		alertName := "UnknownAlert"
		if labels, ok := alert["labels"].(map[string]interface{}); ok {
			if name, ok := labels["alertname"].(string); ok {
				alertName = name
			}
		}

		fmt.Printf("[TEST_APP] Alert %d: name=%s, fingerprint=%s\n", idx, alertName, fingerprint)

		// Insert into database
		query := `
			INSERT INTO alerts (fingerprint, alert_name, status, labels, annotations, starts_at, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		labelsJSON, _ := json.Marshal(alert["labels"])
		annotationsJSON, _ := json.Marshal(alert["annotations"])
		status := "firing"
		if s, ok := alert["status"].(string); ok {
			status = s
		}

		startsAt := time.Now()
		if sa, ok := alert["startsAt"].(string); ok {
			if parsed, err := time.Parse(time.RFC3339, sa); err == nil {
				startsAt = parsed
			}
		}

		fmt.Printf("[TEST_APP] Inserting into DB: fingerprint=%s, alertName=%s\n", fingerprint, alertName)
		_, err := app.Infrastructure.DB.ExecContext(r.Context(), query,
			fingerprint, alertName, status, labelsJSON, annotationsJSON, startsAt, time.Now())
		if err != nil {
			fmt.Printf("[TEST_APP] Database error: %v\n", err)
			http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
			return
		}

		fmt.Printf("[TEST_APP] DB insert successful\n")

		// Simulate LLM classification (sync in test environment for predictability)
		app.classifyAlert(r.Context(), fingerprint, alertName)
	}

	// Return response
	response := map[string]interface{}{
		"status":       "ok",
		"fingerprints": fingerprints,
		"count":        len(fingerprints),
	}

	// If single alert, return fingerprint directly
	if len(fingerprints) == 1 {
		response["fingerprint"] = fingerprints[0]
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// classifyAlert simulates LLM classification
func (app *TestApplication) classifyAlert(ctx context.Context, fingerprint, alertName string) {
	// Call mock LLM
	req := &integration.ClassificationRequest{
		AlertName: alertName,
		Labels:    map[string]string{"alertname": alertName},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return
	}

	resp, err := http.Post(
		app.Infrastructure.MockLLMServer.URL()+"/classify",
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil || resp.StatusCode != http.StatusOK {
		// Fallback classification
		return
	}
	defer resp.Body.Close()

	var classification integration.ClassificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&classification); err != nil {
		return
	}

	// Store classification in database
	query := `
		INSERT INTO alert_classifications (alert_fingerprint, severity, confidence, reasoning, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, _ = app.Infrastructure.DB.ExecContext(ctx, query,
		fingerprint, classification.Severity, classification.Confidence,
		classification.Reasoning, time.Now())
}

// handleHealth returns health status
func (app *TestApplication) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// handleMetrics returns Prometheus-style metrics
func (app *TestApplication) handleMetrics(w http.ResponseWriter, r *http.Request) {
	app.mu.Lock()
	webhookCount := len(app.webhooks)
	app.mu.Unlock()

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)

	// Minimal metrics for E2E tests
	fmt.Fprintf(w, "# HELP webhooks_total Total number of webhooks received\n")
	fmt.Fprintf(w, "# TYPE webhooks_total counter\n")
	fmt.Fprintf(w, "webhooks_total %d\n\n", webhookCount)

	fmt.Fprintf(w, "# HELP classification_l1_cache_hits_total L1 cache hits\n")
	fmt.Fprintf(w, "# TYPE classification_l1_cache_hits_total counter\n")
	fmt.Fprintf(w, "classification_l1_cache_hits_total 0\n\n")

	fmt.Fprintf(w, "# HELP classification_l2_cache_hits_total L2 cache hits\n")
	fmt.Fprintf(w, "# TYPE classification_l2_cache_hits_total counter\n")
	fmt.Fprintf(w, "classification_l2_cache_hits_total 0\n\n")

	fmt.Fprintf(w, "# HELP llm_errors_total LLM errors\n")
	fmt.Fprintf(w, "# TYPE llm_errors_total counter\n")
	fmt.Fprintf(w, "llm_errors_total 0\n")
}

// handleHistory returns alert history
func (app *TestApplication) handleHistory(w http.ResponseWriter, r *http.Request) {
	query := `
		SELECT fingerprint, alert_name, status, labels, annotations, starts_at, created_at
		FROM alerts
		ORDER BY created_at DESC
		LIMIT 100
	`

	rows, err := app.Infrastructure.DB.QueryContext(r.Context(), query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	alerts := make([]map[string]interface{}, 0)
	for rows.Next() {
		var fingerprint, alertName, status string
		var labelsJSON, annotationsJSON []byte
		var startsAt, createdAt time.Time

		if err := rows.Scan(&fingerprint, &alertName, &status, &labelsJSON, &annotationsJSON, &startsAt, &createdAt); err != nil {
			continue
		}

		var labels, annotations map[string]interface{}
		json.Unmarshal(labelsJSON, &labels)
		json.Unmarshal(annotationsJSON, &annotations)

		alerts = append(alerts, map[string]interface{}{
			"fingerprint": fingerprint,
			"alertName":   alertName,
			"status":      status,
			"labels":      labels,
			"annotations": annotations,
			"startsAt":    startsAt,
			"createdAt":   createdAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alerts": alerts,
		"total":  len(alerts),
	})
}

// handleHistoryDetail returns single alert details
func (app *TestApplication) handleHistoryDetail(w http.ResponseWriter, r *http.Request) {
	// Extract fingerprint from path
	// For simplicity, just return empty for now
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"alert": nil,
	})
}

// handlePublishingTargets returns publishing targets
func (app *TestApplication) handlePublishingTargets(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"targets": []map[string]interface{}{
			{"name": "slack", "type": "slack", "enabled": true},
			{"name": "pagerduty", "type": "pagerduty", "enabled": true},
		},
	})
}

// handlePublishingStats returns publishing statistics
func (app *TestApplication) handlePublishingStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_published": 0,
		"success_rate":    1.0,
		"targets":         []map[string]interface{}{},
	})
}

// runMigrations creates minimal database schema for tests
func (app *TestApplication) runMigrations(ctx context.Context) error {
	schema := `
		-- Minimal schema for E2E tests
		CREATE TABLE IF NOT EXISTS alerts (
			id BIGSERIAL PRIMARY KEY,
			fingerprint VARCHAR(64) NOT NULL,
			alert_name VARCHAR(255) NOT NULL,
			namespace VARCHAR(255),
			status VARCHAR(20) NOT NULL DEFAULT 'firing',
			labels JSONB NOT NULL DEFAULT '{}',
			annotations JSONB NOT NULL DEFAULT '{}',
			starts_at TIMESTAMP WITH TIME ZONE,
			ends_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_alerts_fingerprint ON alerts(fingerprint);
		CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at DESC);

		-- LLM Classifications table
		CREATE TABLE IF NOT EXISTS alert_classifications (
			id BIGSERIAL PRIMARY KEY,
			alert_fingerprint VARCHAR(64) NOT NULL,
			severity VARCHAR(20) NOT NULL,
			confidence DECIMAL(4,3) NOT NULL,
			reasoning TEXT,
			recommendations JSONB DEFAULT '[]',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_classifications_fingerprint ON alert_classifications(alert_fingerprint);
	`

	_, err := app.Infrastructure.DB.ExecContext(ctx, schema)
	return err
}

// Close stops the test application
func (app *TestApplication) Close() {
	if app.Server != nil {
		app.Server.Close()
	}
}

// GetWebhooks returns received webhooks for verification
func (app *TestApplication) GetWebhooks() []map[string]interface{} {
	app.mu.Lock()
	defer app.mu.Unlock()
	return append([]map[string]interface{}{}, app.webhooks...)
}
