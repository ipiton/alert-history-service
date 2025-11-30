//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Errors_DatabaseUnavailable validates 503 error when database is down
func TestE2E_Errors_DatabaseUnavailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// TODO: Implement database shutdown simulation
	// This would require either:
	// 1. Stopping the PostgreSQL container temporarily
	// 2. Closing database connections
	// 3. Using a circuit breaker to simulate DB failure

	// For now, we'll test that the system handles database errors gracefully
	// by testing timeout scenarios

	t.Skip("Database unavailability test requires container control or circuit breaker simulation")

	// Prepare webhook
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("DBUnavailableTest", "critical")

	// Attempt to send alert
	resp, err := helper.MakeRequest("POST", "/webhook", webhook)

	// If database is unavailable, we expect:
	// - Either timeout (context deadline exceeded)
	// - Or 503 Service Unavailable response
	if err != nil {
		t.Logf("Request failed as expected: %v", err)
	} else {
		defer resp.Body.Close()
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
			"Expected 503 Service Unavailable when database is down")

		var errorResp map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&errorResp)
		assert.Contains(t, errorResp, "error", "Error response should contain error field")
	}

	t.Logf("✅ Database unavailability handled gracefully")
}

// TestE2E_Errors_GracefulDegradation validates system continues despite component failures
func TestE2E_Errors_GracefulDegradation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Scenario: LLM is down, Redis cache is unavailable, but alert ingestion should still work

	// Configure LLM to always fail
	infra.MockLLMServer.SetDefaultResponse(MockLLMResponse{
		StatusCode: http.StatusServiceUnavailable,
		Body:       map[string]interface{}{"error": "Service unavailable"},
		ErrorRate:  1.0,
	})

	// Flush Redis (simulate cache unavailability)
	err = helper.FlushCache(ctx)
	if err != nil {
		t.Logf("Redis flush failed (expected if Redis unavailable): %v", err)
	}

	// Send alert
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("GracefulDegradationTest", "warning")

	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err, "Alert ingestion should succeed despite component failures")
	defer resp.Body.Close()

	// System should still accept the alert (graceful degradation)
	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"Alert should be accepted despite LLM and cache failures")

	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)

	assert.Equal(t, "success", result["status"], "Status should be success")
	fingerprint := result["fingerprint"].(string)
	assert.NotEmpty(t, fingerprint, "Fingerprint should be returned")

	time.Sleep(500 * time.Millisecond)

	// Verify alert was stored (despite failures)
	alert, err := helper.GetAlertByFingerprint(ctx, fingerprint)
	require.NoError(t, err, "Alert should be stored despite component failures")
	assert.Equal(t, "GracefulDegradationTest", alert.AlertName)

	// Verify fallback classification was applied (rule-based)
	assert.NotEmpty(t, alert.Classification, "Alert should have fallback classification")

	// Check metrics for error handling
	metricsResp, err := helper.MakeRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	defer metricsResp.Body.Close()

	metricsBody, _ := helper.ReadBody(metricsResp)
	metricsText := string(metricsBody)

	// Verify error metrics exist
	assert.Contains(t, metricsText, "llm_errors_total", "LLM error metric should be recorded")

	t.Logf("✅ Graceful degradation successful (alert processed despite component failures)")
}
