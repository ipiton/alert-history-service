//go:build integration || e2e
// +build integration e2e

package integration

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// APITestHelper provides utilities for API testing
type APITestHelper struct {
	BaseURL     string
	HTTPClient  *http.Client
	DB          *sql.DB
	Redis       *redis.Client
	MockLLM     *MockLLMServer
	ctx         context.Context
}

// NewAPITestHelper creates test helper with infrastructure
func NewAPITestHelper(infra *TestInfrastructure) *APITestHelper {
	return &APITestHelper{
		BaseURL:    infra.BaseURL,
		DB:         infra.DB,
		Redis:      infra.RedisClient,
		MockLLM:    infra.MockLLMServer,
		ctx:        infra.Context(), // Use context from infrastructure for proper propagation
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// MakeRequest performs HTTP request and returns response
func (h *APITestHelper) MakeRequest(method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := h.BaseURL + path
	req, err := http.NewRequestWithContext(h.ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return h.HTTPClient.Do(req)
}

// MakeRequestWithHeaders performs HTTP request with custom headers
func (h *APITestHelper) MakeRequestWithHeaders(method, path string, body interface{}, headers map[string]string) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonData)
	}

	url := h.BaseURL + path
	req, err := http.NewRequestWithContext(h.ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	if body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	return h.HTTPClient.Do(req)
}

// GetAlertByFingerprint retrieves alert from database by fingerprint
func (h *APITestHelper) GetAlertByFingerprint(ctx context.Context, fingerprint string) (*Alert, error) {
	query := `
		SELECT a.fingerprint, a.alert_name, a.status, a.namespace,
		       a.labels, a.annotations, a.starts_at, a.ends_at, a.created_at,
		       jsonb_build_object(
		           'severity', ac.severity,
		           'confidence', ac.confidence,
		           'reasoning', ac.reasoning,
		           'recommendations', ac.recommendations
		       ) as classification
		FROM alerts a
		LEFT JOIN alert_classifications ac ON a.fingerprint = ac.alert_fingerprint
		WHERE a.fingerprint = $1
		ORDER BY ac.created_at DESC
		LIMIT 1
	`

	var alert Alert
	var labelsJSON, annotationsJSON []byte
	var classificationJSON []byte
	var endsAt *time.Time
	var namespace sql.NullString

	err := h.DB.QueryRowContext(ctx, query, fingerprint).Scan(
		&alert.Fingerprint,
		&alert.AlertName,
		&alert.Status,
		&namespace,
		&labelsJSON,
		&annotationsJSON,
		&alert.StartsAt,
		&endsAt,
		&alert.CreatedAt,
		&classificationJSON,
	)

	if namespace.Valid {
		alert.Namespace = namespace.String
	}

	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if err := json.Unmarshal(labelsJSON, &alert.Labels); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(annotationsJSON, &alert.Annotations); err != nil {
		return nil, err
	}

	// Set classification as JSON string (if exists)
	if len(classificationJSON) > 0 && string(classificationJSON) != "null" {
		alert.Classification = string(classificationJSON)
	}

	// Set severity from classification or default
	if alert.Classification != "" {
		var classData map[string]interface{}
		if err := json.Unmarshal(classificationJSON, &classData); err == nil {
			if sev, ok := classData["severity"].(string); ok {
				alert.Severity = sev
			}
		}
	}

	alert.EndsAt = endsAt
	return &alert, nil
}

// AssertResponse validates HTTP response status code
func (h *APITestHelper) AssertResponse(t *testing.T, resp *http.Response, expectedStatus int) {
	t.Helper()
	assert.Equal(t, expectedStatus, resp.StatusCode, "unexpected status code")
}

// AssertJSONResponse validates response and decodes JSON
func (h *APITestHelper) AssertJSONResponse(t *testing.T, resp *http.Response, expectedStatus int, target interface{}) {
	t.Helper()
	require.Equal(t, expectedStatus, resp.StatusCode, "unexpected status code")

	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

// GetResponseBody reads response body as string
func (h *APITestHelper) GetResponseBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	return string(bodyBytes), nil
}

// ReadBody reads response body as byte array (alias for E2E compatibility)
func (h *APITestHelper) ReadBody(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}

// QueryAlerts queries alerts from database with optional filters
func (h *APITestHelper) QueryAlerts(ctx context.Context, filters map[string]string) ([]*Alert, error) {
	query := `SELECT fingerprint, alert_name, status, namespace, labels, annotations, starts_at, ends_at, created_at FROM alerts WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	// Build dynamic WHERE clause
	if status, ok := filters["status"]; ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if namespace, ok := filters["namespace"]; ok && namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIdx)
		args = append(args, namespace)
		argIdx++
	}
	if alertName, ok := filters["alert_name"]; ok && alertName != "" {
		query += fmt.Sprintf(" AND alert_name = $%d", argIdx)
		args = append(args, alertName)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	rows, err := h.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query alerts: %w", err)
	}
	defer rows.Close()

	var alerts []*Alert
	for rows.Next() {
		var alert Alert
		var labelsJSON, annotationsJSON []byte
		var endsAt *time.Time

		err := rows.Scan(
			&alert.Fingerprint,
			&alert.AlertName,
			&alert.Status,
			&alert.Namespace,
			&labelsJSON,
			&annotationsJSON,
			&alert.StartsAt,
			&endsAt,
			&alert.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan alert: %w", err)
		}

		if err := json.Unmarshal(labelsJSON, &alert.Labels); err != nil {
			return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
		}
		if err := json.Unmarshal(annotationsJSON, &alert.Annotations); err != nil {
			return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
		}

		alert.EndsAt = endsAt
		alerts = append(alerts, &alert)
	}

	return alerts, nil
}

// CountAlerts counts alerts in database with optional filters
func (h *APITestHelper) CountAlerts(ctx context.Context, filters map[string]string) (int, error) {
	query := `SELECT COUNT(*) FROM alerts WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	// Build dynamic WHERE clause
	if status, ok := filters["status"]; ok && status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	if namespace, ok := filters["namespace"]; ok && namespace != "" {
		query += fmt.Sprintf(" AND namespace = $%d", argIdx)
		args = append(args, namespace)
		argIdx++
	}

	var count int
	err := h.DB.QueryRowContext(ctx, query, args...).Scan(&count)
	return count, err
}

// FlushCache flushes all caches (L1 memory + L2 Redis)
func (h *APITestHelper) FlushCache(ctx context.Context) error {
	// Flush Redis (L2 cache)
	if err := h.Redis.FlushAll(ctx).Err(); err != nil {
		return fmt.Errorf("failed to flush Redis: %w", err)
	}
	// L1 cache will be cleared implicitly when service restarts or via FlushL1Cache()
	return nil
}

// WaitForCondition waits until condition is true or timeout
func (h *APITestHelper) WaitForCondition(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		<-ticker.C
	}
	t.Fatalf("timeout waiting for condition: %s", message)
}

// --- Database Helpers ---

// GetAlertFromDB retrieves alert from database by fingerprint
func (h *APITestHelper) GetAlertFromDB(ctx context.Context, fingerprint string) (*Alert, error) {
	query := `
		SELECT fingerprint, alert_name, status, severity, namespace,
		       labels, annotations, starts_at, ends_at, created_at
		FROM alert_history
		WHERE fingerprint = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	var alert Alert
	var labelsJSON, annotationsJSON []byte

	err := h.DB.QueryRowContext(ctx, query, fingerprint).Scan(
		&alert.Fingerprint,
		&alert.AlertName,
		&alert.Status,
		&alert.Severity,
		&alert.Namespace,
		&labelsJSON,
		&annotationsJSON,
		&alert.StartsAt,
		&alert.EndsAt,
		&alert.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Parse JSON fields
	if err := json.Unmarshal(labelsJSON, &alert.Labels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal labels: %w", err)
	}
	if err := json.Unmarshal(annotationsJSON, &alert.Annotations); err != nil {
		return nil, fmt.Errorf("failed to unmarshal annotations: %w", err)
	}

	return &alert, nil
}

// GetAlertsCount returns total number of alerts in database
func (h *APITestHelper) GetAlertsCount(ctx context.Context) (int, error) {
	var count int
	err := h.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM alert_history").Scan(&count)
	return count, err
}

// GetSilenceFromDB retrieves silence from database by ID
func (h *APITestHelper) GetSilenceFromDB(ctx context.Context, id string) (*Silence, error) {
	query := `
		SELECT id, created_by, comment, starts_at, ends_at, matchers
		FROM silences
		WHERE id = $1
	`

	var silence Silence
	var matchersJSON []byte

	err := h.DB.QueryRowContext(ctx, query, id).Scan(
		&silence.ID,
		&silence.CreatedBy,
		&silence.Comment,
		&silence.StartsAt,
		&silence.EndsAt,
		&matchersJSON,
	)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(matchersJSON, &silence.Matchers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal matchers: %w", err)
	}

	return &silence, nil
}

// --- Redis Helpers ---

// GetFromRedis retrieves value from Redis
func (h *APITestHelper) GetFromRedis(ctx context.Context, key string) (string, error) {
	return h.Redis.Get(ctx, key).Result()
}

// SetInRedis stores value in Redis
func (h *APITestHelper) SetInRedis(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return h.Redis.Set(ctx, key, value, expiration).Err()
}

// RedisKeyExists checks if key exists in Redis
func (h *APITestHelper) RedisKeyExists(ctx context.Context, key string) (bool, error) {
	count, err := h.Redis.Exists(ctx, key).Result()
	return count > 0, err
}

// FlushL1Cache flushes L1 (memory) cache via API endpoint (if available)
func (h *APITestHelper) FlushL1Cache() error {
	// This would require an admin endpoint like POST /admin/cache/flush
	// For now, we return an error indicating this feature is not implemented
	// E2E tests will skip L2 cache tests if this returns an error
	return fmt.Errorf("L1 cache flush not implemented - requires admin endpoint")
}

// --- Data Seeding Helpers ---

// SeedTestData inserts test alerts into database
func (h *APITestHelper) SeedTestData(ctx context.Context, alerts []*Alert) error {
	tx, err := h.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO alert_history
		(fingerprint, alert_name, status, severity, namespace, labels, annotations, starts_at, ends_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	for _, alert := range alerts {
		labelsJSON, err := json.Marshal(alert.Labels)
		if err != nil {
			return fmt.Errorf("failed to marshal labels: %w", err)
		}

		annotationsJSON, err := json.Marshal(alert.Annotations)
		if err != nil {
			return fmt.Errorf("failed to marshal annotations: %w", err)
		}

		_, err = tx.ExecContext(ctx, query,
			alert.Fingerprint,
			alert.AlertName,
			alert.Status,
			alert.Severity,
			alert.Namespace,
			labelsJSON,
			annotationsJSON,
			alert.StartsAt,
			alert.EndsAt,
			alert.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert alert: %w", err)
		}
	}

	return tx.Commit()
}

// SeedCache pre-populates Redis cache with test data
func (h *APITestHelper) SeedCache(ctx context.Context, key string, value interface{}) error {
	jsonData, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}
	return h.Redis.Set(ctx, key, jsonData, 1*time.Hour).Err()
}

// --- Test Data Models ---

// Alert represents alert in database
type Alert struct {
	Fingerprint    string            `json:"fingerprint"`
	AlertName      string            `json:"alert_name"`
	Status         string            `json:"status"`
	Severity       string            `json:"severity"`
	Namespace      string            `json:"namespace"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
	StartsAt       time.Time         `json:"starts_at"`
	EndsAt         *time.Time        `json:"ends_at,omitempty"`
	CreatedAt      time.Time         `json:"created_at"`
	Classification string            `json:"classification,omitempty"` // JSON string from alert_classifications table
}

// Silence represents silence in database
type Silence struct {
	ID        string              `json:"id"`
	CreatedBy string              `json:"created_by"`
	Comment   string              `json:"comment"`
	StartsAt  time.Time           `json:"starts_at"`
	EndsAt    time.Time           `json:"ends_at"`
	Matchers  []map[string]string `json:"matchers"`
}
