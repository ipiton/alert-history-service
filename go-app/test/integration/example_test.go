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

// TestInfrastructureSetup validates test infrastructure can be set up and torn down
func TestInfrastructureSetup(t *testing.T) {
	ctx := context.Background()

	// Setup infrastructure
	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err, "failed to setup test infrastructure")
	defer infra.Teardown(ctx)

	// Verify PostgreSQL is healthy
	err = infra.DB.PingContext(ctx)
	assert.NoError(t, err, "postgres should be reachable")

	// Verify Redis is healthy
	err = infra.RedisClient.Ping(ctx).Err()
	assert.NoError(t, err, "redis should be reachable")

	// Verify Mock LLM server is running
	assert.NotNil(t, infra.MockLLMServer, "mock LLM server should be initialized")
	assert.NotEmpty(t, infra.MockLLMServer.URL(), "mock LLM server should have URL")
}

// TestAPITestHelper validates helper functions work correctly
func TestAPITestHelper(t *testing.T) {
	ctx := context.Background()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)

	// Test database operations
	t.Run("database operations", func(t *testing.T) {
		// Get count (should be 0 initially)
		count, err := helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "should have 0 alerts initially")

		// Seed test data
		alerts := []*Alert{
			NewTestAlert("TestAlert1").WithSeverity("critical"),
			NewTestAlert("TestAlert2").WithSeverity("warning"),
		}
		err = helper.SeedTestData(ctx, alerts)
		require.NoError(t, err, "should seed test data successfully")

		// Verify count increased
		count, err = helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "should have 2 alerts after seeding")

		// Get alert from DB
		alert, err := helper.GetAlertFromDB(ctx, alerts[0].Fingerprint)
		require.NoError(t, err)
		assert.Equal(t, "TestAlert1", alert.AlertName)
		assert.Equal(t, "critical", alert.Severity)

		// Reset database
		err = infra.ResetDatabase(ctx)
		require.NoError(t, err)

		count, err = helper.GetAlertsCount(ctx)
		require.NoError(t, err)
		assert.Equal(t, 0, count, "should have 0 alerts after reset")
	})

	// Test Redis operations
	t.Run("redis operations", func(t *testing.T) {
		key := "test:key"
		value := "test value"

		// Set value
		err := helper.SetInRedis(ctx, key, value, 1*time.Minute)
		require.NoError(t, err)

		// Get value
		retrieved, err := helper.GetFromRedis(ctx, key)
		require.NoError(t, err)
		assert.Equal(t, value, retrieved)

		// Check existence
		exists, err := helper.RedisKeyExists(ctx, key)
		require.NoError(t, err)
		assert.True(t, exists)

		// Reset Redis
		err = infra.ResetRedis(ctx)
		require.NoError(t, err)

		exists, err = helper.RedisKeyExists(ctx, key)
		require.NoError(t, err)
		assert.False(t, exists, "key should not exist after reset")
	})
}

// TestMockLLMServer validates mock LLM server functionality
func TestMockLLMServer(t *testing.T) {
	mock := NewMockLLMServer()
	defer mock.Close()

	t.Run("default response", func(t *testing.T) {
		// Make request to mock server
		resp := &ClassificationResponse{}
		// Simulate HTTP call (in real test, would use http.Client)
		assert.NotEmpty(t, mock.URL())
	})

	t.Run("configured response", func(t *testing.T) {
		// Configure response
		mock.SetResponse("HighMemoryUsage", &ClassificationResponse{
			Severity:   "critical",
			Category:   "resource",
			Confidence: 0.95,
			Reasoning:  "Memory usage critically high",
		})

		// Verify response was set
		assert.Equal(t, 1, len(mock.responses))
	})

	t.Run("error simulation", func(t *testing.T) {
		mock.SetError(500, "Internal Server Error")
		assert.NotNil(t, mock.errorResp)

		mock.ClearError()
		assert.Nil(t, mock.errorResp)
	})

	t.Run("latency simulation", func(t *testing.T) {
		mock.SetLatency(100 * time.Millisecond)
		assert.Equal(t, 100*time.Millisecond, mock.latency)
	})

	t.Run("request tracking", func(t *testing.T) {
		mock.Reset()
		assert.Equal(t, 0, mock.GetRequestCount())
	})

	t.Run("default responses", func(t *testing.T) {
		mock.Reset()
		mock.SetDefaultResponses()
		// Verify default responses loaded
		assert.Greater(t, len(mock.responses), 0)
	})
}

// TestFixtures validates fixture loading
func TestFixtures(t *testing.T) {
	fixtures := NewFixtures()

	t.Run("load alerts", func(t *testing.T) {
		alerts, err := fixtures.LoadAlerts("alerts.json")
		require.NoError(t, err, "should load alerts fixture")
		assert.Greater(t, len(alerts), 0, "should have at least one alert")

		// Verify first alert
		if len(alerts) > 0 {
			alert := alerts[0]
			assert.NotEmpty(t, alert.Fingerprint)
			assert.NotEmpty(t, alert.AlertName)
			assert.NotEmpty(t, alert.Status)
		}
	})

	t.Run("alert builder", func(t *testing.T) {
		alert := NewTestAlert("TestAlert").
			WithSeverity("critical").
			WithNamespace("production").
			WithLabel("team", "platform").
			WithAnnotation("runbook", "https://example.com")

		assert.Equal(t, "TestAlert", alert.AlertName)
		assert.Equal(t, "critical", alert.Severity)
		assert.Equal(t, "production", alert.Namespace)
		assert.Equal(t, "platform", alert.Labels["team"])
		assert.Equal(t, "https://example.com", alert.Annotations["runbook"])
	})

	t.Run("webhook builders", func(t *testing.T) {
		// Alertmanager webhook
		webhook := NewAlertmanagerWebhook().
			AddFiringAlert("Alert1", "critical").
			AddResolvedAlert("Alert2", "warning")

		assert.Equal(t, 2, len(webhook.Alerts))
		assert.Equal(t, "firing", webhook.Alerts[0].Status)
		assert.Equal(t, "resolved", webhook.Alerts[1].Status)

		// Prometheus webhook
		promWebhook := NewPrometheusWebhook().
			AddFiringAlert("Alert3", "info")

		assert.Equal(t, 1, len(promWebhook.Data.Alerts))
		assert.Equal(t, "firing", promWebhook.Data.Alerts[0].State)
	})
}

// TestTestScenarios validates common test scenarios
func TestTestScenarios(t *testing.T) {
	t.Run("critical alerts scenario", func(t *testing.T) {
		alerts := GetCriticalAlertsScenario()
		assert.Equal(t, 3, len(alerts))
		for _, alert := range alerts {
			assert.Equal(t, "critical", alert.Severity)
			assert.Equal(t, "production", alert.Namespace)
		}
	})

	t.Run("mixed alerts scenario", func(t *testing.T) {
		alerts := GetMixedAlertsScenario()
		assert.Equal(t, 5, len(alerts))

		severities := make(map[string]int)
		for _, alert := range alerts {
			severities[alert.Severity]++
		}
		assert.Greater(t, len(severities), 1, "should have multiple severities")
	})

	t.Run("resolved alerts scenario", func(t *testing.T) {
		alerts := GetResolvedAlertsScenario()
		assert.Equal(t, 2, len(alerts))
		for _, alert := range alerts {
			assert.Equal(t, "resolved", alert.Status)
			assert.NotNil(t, alert.EndsAt)
		}
	})
}
