//go:build integration
// +build integration

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
		ctx:        context.Background(),
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
	Fingerprint string            `json:"fingerprint"`
	AlertName   string            `json:"alert_name"`
	Status      string            `json:"status"`
	Severity    string            `json:"severity"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    time.Time         `json:"starts_at"`
	EndsAt      *time.Time        `json:"ends_at,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
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
