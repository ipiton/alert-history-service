//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// PublishingTestHelper provides utilities for testing publishing flows
type PublishingTestHelper struct {
	DB          *sql.DB
	MockTargets map[string]*MockPublishingTarget
	ctx         context.Context
}

// NewPublishingTestHelper creates publishing test helper
func NewPublishingTestHelper(db *sql.DB, ctx context.Context) *PublishingTestHelper {
	return &PublishingTestHelper{
		DB:          db,
		MockTargets: make(map[string]*MockPublishingTarget),
		ctx:         ctx,
	}
}

// SetupMockTargets creates mock publishing targets
func (h *PublishingTestHelper) SetupMockTargets() {
	h.MockTargets["slack"] = NewMockSlackTarget("test-slack")
	h.MockTargets["pagerduty"] = NewMockPagerDutyTarget("test-pagerduty")
	h.MockTargets["rootly"] = NewMockRootlyTarget("test-rootly")
	h.MockTargets["webhook"] = NewMockGenericWebhook("test-webhook")
}

// TeardownMockTargets closes all mock targets
func (h *PublishingTestHelper) TeardownMockTargets() {
	for _, target := range h.MockTargets {
		target.Close()
	}
	h.MockTargets = make(map[string]*MockPublishingTarget)
}

// GetMockTarget retrieves a mock target by type
func (h *PublishingTestHelper) GetMockTarget(targetType string) *MockPublishingTarget {
	return h.MockTargets[targetType]
}

// ClearAllRequests clears request history for all mock targets
func (h *PublishingTestHelper) ClearAllRequests() {
	for _, target := range h.MockTargets {
		target.ClearRequests()
	}
}

// --- Publishing Verification ---

// VerifyPublished asserts alert was published to specified targets
func (h *PublishingTestHelper) VerifyPublished(t *testing.T, fingerprint string, expectedTargets ...string) {
	results, err := h.GetPublishingResults(h.ctx, fingerprint)
	require.NoError(t, err, "Failed to get publishing results")

	// Check each expected target
	for _, targetType := range expectedTargets {
		found := false
		for _, result := range results {
			if result.TargetType == targetType && result.Status == "success" {
				found = true
				break
			}
		}
		assert.True(t, found, "Alert not published to %s", targetType)
	}
}

// VerifyNotPublished asserts alert was NOT published to specified targets
func (h *PublishingTestHelper) VerifyNotPublished(t *testing.T, fingerprint string, unexpectedTargets ...string) {
	results, err := h.GetPublishingResults(h.ctx, fingerprint)
	require.NoError(t, err, "Failed to get publishing results")

	// Check each unexpected target
	for _, targetType := range unexpectedTargets {
		for _, result := range results {
			assert.NotEqual(t, targetType, result.TargetType, "Alert should not be published to %s", targetType)
		}
	}
}

// VerifyPublishingFailure asserts publishing to target failed
func (h *PublishingTestHelper) VerifyPublishingFailure(t *testing.T, fingerprint, targetType string) {
	results, err := h.GetPublishingResults(h.ctx, fingerprint)
	require.NoError(t, err, "Failed to get publishing results")

	found := false
	for _, result := range results {
		if result.TargetType == targetType && result.Status == "failed" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected publishing failure for %s", targetType)
}

// VerifyPublishingRetry asserts publishing was retried
func (h *PublishingTestHelper) VerifyPublishingRetry(t *testing.T, fingerprint, targetType string, minRetries int) {
	results, err := h.GetPublishingResults(h.ctx, fingerprint)
	require.NoError(t, err, "Failed to get publishing results")

	for _, result := range results {
		if result.TargetType == targetType {
			assert.GreaterOrEqual(t, result.Attempts, minRetries, "Expected at least %d retry attempts", minRetries)
			return
		}
	}
	t.Fatalf("No publishing result found for %s", targetType)
}

// --- Database Queries ---

// PublishingResult represents a publishing result from database
type PublishingResult struct {
	ID           int
	Fingerprint  string
	TargetType   string
	TargetName   string
	Status       string
	Attempts     int
	LastAttempt  time.Time
	ErrorMessage string
	ResponseData string
}

// GetPublishingResults retrieves publishing results for an alert
func (h *PublishingTestHelper) GetPublishingResults(ctx context.Context, fingerprint string) ([]PublishingResult, error) {
	query := `
		SELECT
			id,
			fingerprint,
			target_type,
			target_name,
			status,
			attempts,
			last_attempt_at,
			error_message,
			response_data
		FROM publishing_results
		WHERE fingerprint = $1
		ORDER BY last_attempt_at DESC
	`

	rows, err := h.DB.QueryContext(ctx, query, fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to query publishing results: %w", err)
	}
	defer rows.Close()

	var results []PublishingResult
	for rows.Next() {
		var r PublishingResult
		var errorMsg, responseData sql.NullString

		err := rows.Scan(
			&r.ID,
			&r.Fingerprint,
			&r.TargetType,
			&r.TargetName,
			&r.Status,
			&r.Attempts,
			&r.LastAttempt,
			&errorMsg,
			&responseData,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan publishing result: %w", err)
		}

		if errorMsg.Valid {
			r.ErrorMessage = errorMsg.String
		}
		if responseData.Valid {
			r.ResponseData = responseData.String
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// GetPublishingStatsByTarget returns publishing statistics by target type
func (h *PublishingTestHelper) GetPublishingStatsByTarget(ctx context.Context) (map[string]PublishingStats, error) {
	query := `
		SELECT
			target_type,
			COUNT(*) as total,
			SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as successful,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed,
			AVG(attempts) as avg_attempts
		FROM publishing_results
		GROUP BY target_type
	`

	rows, err := h.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query publishing stats: %w", err)
	}
	defer rows.Close()

	stats := make(map[string]PublishingStats)
	for rows.Next() {
		var targetType string
		var s PublishingStats

		err := rows.Scan(
			&targetType,
			&s.Total,
			&s.Successful,
			&s.Failed,
			&s.AvgAttempts,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan publishing stats: %w", err)
		}

		stats[targetType] = s
	}

	return stats, rows.Err()
}

// PublishingStats represents publishing statistics
type PublishingStats struct {
	Total       int
	Successful  int
	Failed      int
	AvgAttempts float64
}

// --- Mock Target Assertions ---

// AssertMockReceived asserts mock target received request
func (h *PublishingTestHelper) AssertMockReceived(t *testing.T, targetType string, expectedCount int) {
	mock := h.MockTargets[targetType]
	require.NotNil(t, mock, "Mock target %s not found", targetType)

	count := mock.GetRequestCount()
	assert.Equal(t, expectedCount, count, "Expected %d requests to %s, got %d", expectedCount, targetType, count)
}

// AssertMockNotReceived asserts mock target received no requests
func (h *PublishingTestHelper) AssertMockNotReceived(t *testing.T, targetType string) {
	h.AssertMockReceived(t, targetType, 0)
}

// AssertMockReceivedWithField asserts mock received request with specific field
func (h *PublishingTestHelper) AssertMockReceivedWithField(t *testing.T, targetType, field string, expectedValue interface{}) {
	mock := h.MockTargets[targetType]
	require.NotNil(t, mock, "Mock target %s not found", targetType)

	lastReq := mock.GetLastRequest()
	require.NotNil(t, lastReq, "No requests received by %s", targetType)

	err := lastReq.VerifyField(field, expectedValue)
	assert.NoError(t, err, "Field verification failed for %s", targetType)
}

// AssertParallelPublishing asserts requests were sent in parallel (within time window)
func (h *PublishingTestHelper) AssertParallelPublishing(t *testing.T, targetTypes []string, maxDelta time.Duration) {
	var timestamps []time.Time

	for _, targetType := range targetTypes {
		mock := h.MockTargets[targetType]
		require.NotNil(t, mock, "Mock target %s not found", targetType)

		lastReq := mock.GetLastRequest()
		require.NotNil(t, lastReq, "No requests received by %s", targetType)

		timestamps = append(timestamps, lastReq.ReceivedAt)
	}

	// Check all timestamps are within maxDelta of each other
	if len(timestamps) < 2 {
		return
	}

	earliest := timestamps[0]
	latest := timestamps[0]
	for _, ts := range timestamps[1:] {
		if ts.Before(earliest) {
			earliest = ts
		}
		if ts.After(latest) {
			latest = ts
		}
	}

	delta := latest.Sub(earliest)
	assert.LessOrEqual(t, delta, maxDelta, "Requests should be parallel (within %v), but delta was %v", maxDelta, delta)
}

// --- Webhook Payload Builders ---

// BuildSlackWebhookPayload creates Slack-formatted webhook payload
func BuildSlackWebhookPayload(alertName, severity, summary string) map[string]interface{} {
	return map[string]interface{}{
		"text": fmt.Sprintf("🚨 *%s*", alertName),
		"blocks": []map[string]interface{}{
			{
				"type": "section",
				"text": map[string]interface{}{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%s* (%s)\n%s", alertName, severity, summary),
				},
			},
		},
	}
}

// BuildPagerDutyEventPayload creates PagerDuty Events API payload
func BuildPagerDutyEventPayload(action, routingKey, dedupKey, summary string) map[string]interface{} {
	payload := map[string]interface{}{
		"routing_key":  routingKey,
		"event_action": action, // "trigger", "acknowledge", "resolve"
		"payload": map[string]interface{}{
			"summary":  summary,
			"severity": "critical",
			"source":   "alertmanager-plus-plus",
		},
	}

	if dedupKey != "" {
		payload["dedup_key"] = dedupKey
	}

	return payload
}

// BuildRootlyIncidentPayload creates Rootly Incidents API payload
func BuildRootlyIncidentPayload(title, severity, summary string) map[string]interface{} {
	return map[string]interface{}{
		"data": map[string]interface{}{
			"type": "incidents",
			"attributes": map[string]interface{}{
				"title":       title,
				"severity":    severity,
				"summary":     summary,
				"status":      "started",
				"environment": "production",
			},
		},
	}
}

// --- Utility Functions ---

// WaitForPublishing waits for publishing results to appear in database
func (h *PublishingTestHelper) WaitForPublishing(ctx context.Context, fingerprint string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			results, err := h.GetPublishingResults(ctx, fingerprint)
			if err != nil {
				return err
			}
			if len(results) > 0 {
				return nil
			}
			if time.Now().After(deadline) {
				return fmt.Errorf("timeout waiting for publishing results")
			}
		}
	}
}

// ParsePublishingResponse parses JSON response from database
func ParsePublishingResponse(responseData string) (map[string]interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(responseData), &data); err != nil {
		return nil, fmt.Errorf("failed to parse response data: %w", err)
	}
	return data, nil
}
