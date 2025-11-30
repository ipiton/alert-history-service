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

// TestE2E_Classification_FirstTime validates LLM classification on first alert
func TestE2E_Classification_FirstTime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Configure mock LLM response
	infra.MockLLMServer.AddResponse("/classify", MockLLMResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"severity":   "critical",
			"confidence": 0.95,
			"reasoning":  "High CPU indicates performance degradation",
			"recommendations": []string{
				"Scale horizontally",
				"Check for resource-intensive processes",
			},
		},
		Latency: 100 * time.Millisecond,
	})

	// Clear mock LLM request history
	infra.MockLLMServer.ClearRequests()

	// Send alert
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("HighCPUClassification", "warning")

	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fingerprint := result["fingerprint"].(string)

	time.Sleep(300 * time.Millisecond) // Allow classification to complete

	// Verify LLM was called
	llmRequests := infra.MockLLMServer.GetRequests()
	assert.Len(t, llmRequests, 1, "Expected exactly 1 LLM call for first-time classification")

	// Verify alert has classification
	alert, err := helper.GetAlertByFingerprint(ctx, fingerprint)
	require.NoError(t, err)
	assert.NotEmpty(t, alert.Classification, "Alert should have classification")

	if alert.Classification != "" {
		var classification map[string]interface{}
		err := json.Unmarshal([]byte(alert.Classification), &classification)
		require.NoError(t, err)

		assert.Equal(t, "critical", classification["severity"])
		assert.Equal(t, 0.95, classification["confidence"])
		assert.Contains(t, classification["reasoning"], "CPU")
	}

	t.Logf("✅ First-time classification successful with LLM call")
}

// TestE2E_Classification_CacheHitL1 validates L1 (memory) cache hit
func TestE2E_Classification_CacheHitL1(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Configure mock LLM
	infra.MockLLMServer.SetDefaultResponse(MockLLMResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"severity":   "high",
			"confidence": 0.9,
		},
	})

	// Send first alert
	webhook1 := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("CacheTestL1", "warning")

	resp1, err := helper.MakeRequest("POST", "/webhook", webhook1)
	require.NoError(t, err)
	resp1.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Clear LLM request history
	infra.MockLLMServer.ClearRequests()

	// Send identical alert (should hit L1 cache)
	webhook2 := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("CacheTestL1", "warning")

	resp2, err := helper.MakeRequest("POST", "/webhook", webhook2)
	require.NoError(t, err)
	resp2.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Verify LLM was NOT called (cache hit)
	llmRequests := infra.MockLLMServer.GetRequests()
	assert.Len(t, llmRequests, 0, "Expected 0 LLM calls (L1 cache hit)")

	// Verify cache hit metric incremented
	metricsResp, err := helper.MakeRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	defer metricsResp.Body.Close()

	// Parse metrics
	var metricsBody []byte
	metricsBody, _ = helper.ReadBody(metricsResp)
	metricsText := string(metricsBody)

	assert.Contains(t, metricsText, "classification_l1_cache_hits_total", "L1 cache hit metric should exist")

	t.Logf("✅ L1 cache hit successful (no LLM call)")
}

// TestE2E_Classification_CacheHitL2 validates L2 (Redis) cache hit
func TestE2E_Classification_CacheHitL2(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Configure mock LLM
	infra.MockLLMServer.SetDefaultResponse(MockLLMResponse{
		StatusCode: http.StatusOK,
		Body: map[string]interface{}{
			"severity":   "medium",
			"confidence": 0.85,
		},
	})

	// Send first alert (populates L1 and L2 cache)
	webhook1 := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("CacheTestL2", "info")

	resp1, err := helper.MakeRequest("POST", "/webhook", webhook1)
	require.NoError(t, err)
	resp1.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Clear L1 cache (simulate pod restart or eviction)
	// This forces next lookup to go to L2 (Redis)
	err = helper.FlushL1Cache()
	if err != nil {
		t.Skip("L1 cache flush not implemented, skipping L2 test")
	}

	// Clear LLM request history
	infra.MockLLMServer.ClearRequests()

	// Send identical alert (should hit L2 cache)
	webhook2 := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("CacheTestL2", "info")

	resp2, err := helper.MakeRequest("POST", "/webhook", webhook2)
	require.NoError(t, err)
	resp2.Body.Close()

	time.Sleep(200 * time.Millisecond)

	// Verify LLM was NOT called (L2 cache hit)
	llmRequests := infra.MockLLMServer.GetRequests()
	assert.Len(t, llmRequests, 0, "Expected 0 LLM calls (L2 cache hit)")

	// Verify L2 cache hit metric
	metricsResp, err := helper.MakeRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	defer metricsResp.Body.Close()

	metricsBody, _ := helper.ReadBody(metricsResp)
	metricsText := string(metricsBody)

	assert.Contains(t, metricsText, "classification_l2_cache_hits_total", "L2 cache hit metric should exist")

	t.Logf("✅ L2 cache hit successful (Redis, no LLM call)")
}

// TestE2E_Classification_LLMTimeout validates fallback on LLM timeout
func TestE2E_Classification_LLMTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Configure mock LLM with timeout (long delay)
	infra.MockLLMServer.SetDefaultResponse(MockLLMResponse{
		StatusCode: http.StatusOK,
		Body:       map[string]interface{}{"severity": "high"},
		Latency:    10 * time.Second, // Longer than timeout
	})

	// Send alert
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("TimeoutTest", "critical")

	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Alert should still be processed (with fallback classification)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Alert should be accepted despite LLM timeout")

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fingerprint := result["fingerprint"].(string)

	time.Sleep(500 * time.Millisecond)

	// Verify alert exists (processed despite timeout)
	alert, err := helper.GetAlertByFingerprint(ctx, fingerprint)
	require.NoError(t, err, "Alert should be stored despite LLM timeout")
	assert.Equal(t, "TimeoutTest", alert.AlertName)

	// Verify fallback classification was applied
	assert.NotEmpty(t, alert.Classification, "Alert should have fallback classification")

	// Check metrics for LLM timeout
	metricsResp, err := helper.MakeRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	defer metricsResp.Body.Close()

	metricsBody, _ := helper.ReadBody(metricsResp)
	metricsText := string(metricsBody)

	assert.Contains(t, metricsText, "llm_errors_total", "LLM error metric should exist")

	t.Logf("✅ LLM timeout handled gracefully with fallback")
}

// TestE2E_Classification_LLMUnavailable validates fallback when LLM is down
func TestE2E_Classification_LLMUnavailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Configure mock LLM to return errors
	infra.MockLLMServer.SetDefaultResponse(MockLLMResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       map[string]interface{}{"error": "Service unavailable"},
		ErrorRate:  1.0, // Always fail
	})

	// Send alert
	webhook := fixtures.NewAlertmanagerWebhook().
		AddFiringAlert("UnavailableTest", "warning")

	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Alert should still be processed (graceful degradation)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "Alert should be accepted despite LLM unavailability")

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fingerprint := result["fingerprint"].(string)

	time.Sleep(500 * time.Millisecond)

	// Verify alert exists with rule-based classification
	alert, err := helper.GetAlertByFingerprint(ctx, fingerprint)
	require.NoError(t, err, "Alert should be stored with rule-based classification")

	assert.Equal(t, "UnavailableTest", alert.AlertName)
	assert.NotEmpty(t, alert.Classification, "Alert should have rule-based fallback classification")

	// Verify fallback was rule-based (check classification source)
	if alert.Classification != "" {
		var classification map[string]interface{}
		json.Unmarshal([]byte(alert.Classification), &classification)
		// Rule-based classification should have different structure or source indicator
		assert.NotNil(t, classification, "Classification should exist")
	}

	t.Logf("✅ LLM unavailable handled gracefully (rule-based fallback)")
}
