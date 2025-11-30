//go:build e2e
// +build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Ingestion_HappyPath validates complete alert ingestion flow
func TestE2E_Ingestion_HappyPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Setup infrastructure
	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err, "Failed to setup test infrastructure")
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Prepare Alertmanager webhook payload
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("HighCPU", "critical")

	// Send POST /webhook
	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err, "Webhook request failed")
	defer resp.Body.Close()

	// Assert response
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err, "Failed to decode response")

	assert.Equal(t, "success", result["status"], "Expected success status")
	assert.NotEmpty(t, result["fingerprint"], "Fingerprint should be returned")

	fingerprint := result["fingerprint"].(string)

	// Verify alert stored in database
	time.Sleep(100 * time.Millisecond) // Small delay for async processing

	alerts, err := helper.QueryAlerts(ctx, map[string]string{
		"alertname": "HighCPU",
	})
	require.NoError(t, err, "Failed to query alerts")
	assert.Len(t, alerts, 1, "Expected 1 alert in database")

	if len(alerts) > 0 {
		alert := alerts[0]
		assert.Equal(t, "HighCPU", alert.AlertName)
		assert.Equal(t, "critical", alert.Severity)
		assert.Equal(t, "firing", alert.Status)
		assert.Equal(t, fingerprint, alert.Fingerprint)
	}

	t.Logf("✅ Alert ingestion happy path successful (fingerprint: %s)", fingerprint)
}

// TestE2E_Ingestion_DuplicateDetection validates deduplication
func TestE2E_Ingestion_DuplicateDetection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Create identical webhooks
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("DuplicateTest", "warning")

	// Send first request
	resp1, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp1.Body.Close()
	assert.Equal(t, http.StatusOK, resp1.StatusCode)

	var result1 map[string]interface{}
	json.NewDecoder(resp1.Body).Decode(&result1)
	fingerprint1 := result1["fingerprint"].(string)

	time.Sleep(100 * time.Millisecond)

	// Send duplicate request
	resp2, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var result2 map[string]interface{}
	json.NewDecoder(resp2.Body).Decode(&result2)
	fingerprint2 := result2["fingerprint"].(string)

	// Fingerprints should match
	assert.Equal(t, fingerprint1, fingerprint2, "Duplicate alerts should have same fingerprint")

	time.Sleep(100 * time.Millisecond)

	// Verify only 1 alert in database (updated, not duplicated)
	count, err := helper.CountAlerts(ctx, map[string]string{
		"alertname": "DuplicateTest",
	})
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Expected exactly 1 alert (deduplicated)")

	t.Logf("✅ Duplicate detection successful (fingerprint: %s)", fingerprint1)
}

// TestE2E_Ingestion_BatchIngestion validates multiple alerts in single request
func TestE2E_Ingestion_BatchIngestion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Create webhook with multiple alerts
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("Alert1", "critical").
		AddFiringAlert("Alert2", "warning").
		AddFiringAlert("Alert3", "info").
		AddResolvedAlert("Alert4", "critical")

	// Send batch request
	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected 200 OK for batch")

	time.Sleep(200 * time.Millisecond) // Allow processing time

	// Verify all alerts stored
	count, err := helper.CountAlerts(ctx, nil) // All alerts
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 4, "Expected at least 4 alerts")

	// Verify specific alerts
	for _, name := range []string{"Alert1", "Alert2", "Alert3", "Alert4"} {
		alerts, err := helper.QueryAlerts(ctx, map[string]string{"alertname": name})
		require.NoError(t, err, "Failed to query %s", name)
		assert.Len(t, alerts, 1, "Expected 1 %s", name)
	}

	t.Logf("✅ Batch ingestion successful (4 alerts)")
}

// TestE2E_Ingestion_InvalidFormat validates error handling for malformed JSON
func TestE2E_Ingestion_InvalidFormat(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Send malformed JSON
	malformedJSON := []byte(`{"alerts": [{"invalid": }]}`)
	req, err := http.NewRequestWithContext(ctx, "POST", infra.BaseURL+"/webhook", bytes.NewReader(malformedJSON))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := helper.HTTPClient.Do(req)
	require.NoError(t, err, "HTTP request should complete")
	defer resp.Body.Close()

	// Assert 400 Bad Request
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode, "Expected 400 for malformed JSON")

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	// Check error message
	if msg, ok := errorResp["message"].(string); ok {
		assert.Contains(t, msg, "invalid", "Error message should mention invalid format")
	} else if errField, ok := errorResp["error"].(string); ok {
		assert.Contains(t, errField, "invalid", "Error should mention invalid format")
	}

	t.Logf("✅ Invalid format handled correctly (400 response)")
}

// TestE2E_Ingestion_MissingRequiredFields validates validation of required fields
func TestE2E_Ingestion_MissingRequiredFields(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Send webhook with missing required fields
	incompleteWebhook := map[string]interface{}{
		"alerts": []map[string]interface{}{
			{
				// Missing labels (required)
				"annotations": map[string]string{
					"summary": "Test alert",
				},
				"startsAt": time.Now().Format(time.RFC3339),
				"status":   "firing",
			},
		},
	}

	resp, err := helper.MakeRequest("POST", "/webhook", incompleteWebhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Assert 422 Unprocessable Entity (validation error)
	// Or 400 Bad Request depending on implementation
	assert.Contains(t, []int{http.StatusBadRequest, http.StatusUnprocessableEntity}, resp.StatusCode,
		"Expected 400/422 for missing required fields")

	var errorResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&errorResp)

	// Error response should indicate validation issue
	assert.NotEmpty(t, errorResp, "Error response should not be empty")

	t.Logf("✅ Missing fields validation successful (%d response)", resp.StatusCode)
}
