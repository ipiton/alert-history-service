//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWebhookIngestion_Alertmanager tests Alertmanager webhook ingestion end-to-end
func TestWebhookIngestion_Alertmanager(t *testing.T) {
	ctx := context.Background()

	// Setup infrastructure
	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err, "failed to setup test infrastructure")
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Reset database
	err = infra.ResetDatabase(ctx)
	require.NoError(t, err)

	t.Run("single firing alert", func(t *testing.T) {
		// Create Alertmanager webhook payload
		webhook := NewAlertmanagerWebhook().
			AddFiringAlert("HighMemoryUsage", "critical")

		// Send webhook
		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Assert response
		helper.AssertResponse(t, resp, 200)

		// Wait for processing
		time.Sleep(100 * time.Millisecond)

		// Verify alert stored in database
		count, err := helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "should have 1 alert in database")
	})

	t.Run("multiple alerts", func(t *testing.T) {
		infra.ResetDatabase(ctx)

		webhook := NewAlertmanagerWebhook().
			AddFiringAlert("HighCPU", "warning").
			AddFiringAlert("HighMemory", "critical").
			AddResolvedAlert("OldAlert", "info")

		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		helper.AssertResponse(t, resp, 200)

		time.Sleep(150 * time.Millisecond)

		count, err := helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 3, count, "should have 3 alerts")
	})

	t.Run("invalid payload", func(t *testing.T) {
		// Send invalid JSON
		resp, err := helper.MakeRequestWithHeaders("POST", "/webhook",
			map[string]interface{}{"invalid": "payload"},
			map[string]string{"Content-Type": "application/json"})

		require.NoError(t, err)
		defer resp.Body.Close()

		// Should return 4xx error
		assert.True(t, resp.StatusCode >= 400 && resp.StatusCode < 500,
			"should return 4xx for invalid payload")
	})

	t.Run("empty alerts array", func(t *testing.T) {
		webhook := NewAlertmanagerWebhook() // empty

		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should accept but process 0 alerts
		helper.AssertResponse(t, resp, 200)
	})
}

// TestWebhookIngestion_Prometheus tests Prometheus webhook format
func TestWebhookIngestion_Prometheus(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	t.Run("prometheus v2 format", func(t *testing.T) {
		webhook := NewPrometheusWebhook().
			AddFiringAlert("PrometheusAlert", "warning")

		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		helper.AssertResponse(t, resp, 200)

		time.Sleep(100 * time.Millisecond)

		count, err := helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 1, "should have at least 1 alert")
	})
}

// TestWebhookProxy_WithClassification tests intelligent proxy with LLM classification
func TestWebhookProxy_WithClassification(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	// Configure mock LLM server
	helper.MockLLM.SetDefaultResponses()

	t.Run("classification with cache miss", func(t *testing.T) {
		// Reset LLM request tracking
		helper.MockLLM.Reset()
		helper.MockLLM.SetDefaultResponses()

		webhook := NewAlertmanagerWebhook().
			AddFiringAlert("HighMemoryUsage", "critical")

		resp, err := helper.MakeRequest("POST", "/webhook/proxy", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		helper.AssertResponse(t, resp, 200)

		time.Sleep(200 * time.Millisecond)

		// Verify LLM was called (cache miss)
		assert.GreaterOrEqual(t, helper.MockLLM.GetRequestCount(), 1,
			"LLM should be called on cache miss")
	})

	t.Run("classification with cache hit", func(t *testing.T) {
		// Second request should hit cache
		initialCount := helper.MockLLM.GetRequestCount()

		webhook := NewAlertmanagerWebhook().
			AddFiringAlert("HighMemoryUsage", "critical")

		resp, err := helper.MakeRequest("POST", "/webhook/proxy", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		helper.AssertResponse(t, resp, 200)

		time.Sleep(100 * time.Millisecond)

		// LLM request count should not increase (cache hit)
		finalCount := helper.MockLLM.GetRequestCount()
		assert.Equal(t, initialCount, finalCount,
			"LLM should not be called on cache hit")
	})

	t.Run("LLM timeout fallback", func(t *testing.T) {
		helper.MockLLM.Reset()
		helper.MockLLM.SetLatency(10 * time.Second) // Simulate timeout

		webhook := NewAlertmanagerWebhook().
			AddFiringAlert("TimeoutAlert", "warning")

		resp, err := helper.MakeRequest("POST", "/webhook/proxy", webhook)
		require.NoError(t, err)
		defer resp.Body.Close()

		// Should still succeed with rule-based fallback
		helper.AssertResponse(t, resp, 200)

		helper.MockLLM.SetLatency(0) // Reset
	})
}

// TestWebhook_RequestIDPropagation tests request ID is properly propagated
func TestWebhook_RequestIDPropagation(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	webhook := NewAlertmanagerWebhook().AddFiringAlert("TestAlert", "info")

	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	// Check for X-Request-ID header in response
	requestID := resp.Header.Get("X-Request-ID")
	assert.NotEmpty(t, requestID, "response should contain X-Request-ID header")
}

// TestWebhook_MetricsRecording tests Prometheus metrics are recorded
func TestWebhook_MetricsRecording(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Send webhook
	webhook := NewAlertmanagerWebhook().AddFiringAlert("MetricsTest", "info")

	resp, err := helper.MakeRequest("POST", "/webhook", webhook)
	require.NoError(t, err)
	defer resp.Body.Close()

	helper.AssertResponse(t, resp, 200)

	// TODO: Verify metrics via /metrics endpoint
	// This would require starting the actual server or mocking metrics registry
}

// TestWebhook_ConcurrentRequests tests concurrent webhook processing
func TestWebhook_ConcurrentRequests(t *testing.T) {
	t.Skip("Requires running server - implement in Phase 5")

	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	infra.ResetDatabase(ctx)

	// Send 10 concurrent webhook requests
	const numRequests = 10
	done := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(n int) {
			webhook := NewAlertmanagerWebhook().
				AddFiringAlert("ConcurrentAlert", "info")

			resp, err := helper.MakeRequest("POST", "/webhook", webhook)
			if err != nil {
				done <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				done <- assert.AnError
				return
			}
			done <- nil
		}(i)
	}

	// Wait for all requests
	for i := 0; i < numRequests; i++ {
		err := <-done
		assert.NoError(t, err, "concurrent request should succeed")
	}

	time.Sleep(200 * time.Millisecond)

	// Verify all alerts stored
	count, err := helper.GetAlertsCount(ctx)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, numRequests,
		"should have at least %d alerts", numRequests)
}
