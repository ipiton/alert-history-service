//go:build e2e
// +build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_History_Pagination validates pagination (limit/offset)
func TestE2E_History_Pagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Insert 25 alerts
	totalAlerts := 25
	for i := 0; i < totalAlerts; i++ {
		webhook := fixtures.NewAlertmanagerWebhook().
			AddFiringAlert(fmt.Sprintf("PaginationTest%d", i), "info")

		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		resp.Body.Close()
	}

	time.Sleep(1 * time.Second) // Allow all alerts to be processed

	// Test pagination with limit=10
	page1Resp, err := helper.MakeRequest("GET", "/api/v2/history?limit=10&offset=0", nil)
	require.NoError(t, err)
	defer page1Resp.Body.Close()
	assert.Equal(t, http.StatusOK, page1Resp.StatusCode)

	var page1Result map[string]interface{}
	json.NewDecoder(page1Resp.Body).Decode(&page1Result)

	page1Alerts, ok := page1Result["alerts"].([]interface{})
	require.True(t, ok, "Response should contain alerts array")
	assert.Len(t, page1Alerts, 10, "First page should have 10 alerts")

	// Get page 2
	page2Resp, err := helper.MakeRequest("GET", "/api/v2/history?limit=10&offset=10", nil)
	require.NoError(t, err)
	defer page2Resp.Body.Close()

	var page2Result map[string]interface{}
	json.NewDecoder(page2Resp.Body).Decode(&page2Result)

	page2Alerts, ok := page2Result["alerts"].([]interface{})
	require.True(t, ok)
	assert.Len(t, page2Alerts, 10, "Second page should have 10 alerts")

	// Get page 3
	page3Resp, err := helper.MakeRequest("GET", "/api/v2/history?limit=10&offset=20", nil)
	require.NoError(t, err)
	defer page3Resp.Body.Close()

	var page3Result map[string]interface{}
	json.NewDecoder(page3Resp.Body).Decode(&page3Result)

	page3Alerts, ok := page3Result["alerts"].([]interface{})
	require.True(t, ok)
	assert.Len(t, page3Alerts, 5, "Third page should have 5 alerts (remaining)")

	// Verify total count
	if total, ok := page1Result["total"].(float64); ok {
		assert.GreaterOrEqual(t, int(total), totalAlerts, "Total count should be at least %d", totalAlerts)
	}

	t.Logf("✅ Pagination successful (3 pages, %d total alerts)", totalAlerts)
}

// TestE2E_History_Filtering validates filtering by severity and namespace
func TestE2E_History_Filtering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Insert alerts with different severities and namespaces
	testData := []struct {
		name      string
		severity  string
		namespace string
	}{
		{"Critical1", "critical", "production"},
		{"Critical2", "critical", "production"},
		{"Warning1", "warning", "staging"},
		{"Warning2", "warning", "staging"},
		{"Info1", "info", "development"},
	}

	for _, td := range testData {
		webhook := fixtures.NewAlertmanagerWebhook()
		// Need to add namespace label manually
		alert := AlertmanagerAlert{
			Labels: map[string]string{
				"alertname": td.name,
				"severity":  td.severity,
				"namespace": td.namespace,
			},
			Annotations: map[string]string{
				"summary": fmt.Sprintf("Test alert %s", td.name),
			},
			StartsAt: time.Now(),
			Status:   "firing",
		}
		webhook.AddAlert(alert)

		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		resp.Body.Close()
	}

	time.Sleep(1 * time.Second)

	// Test filter by severity=critical
	criticalResp, err := helper.MakeRequest("GET", "/api/v2/history?severity=critical", nil)
	require.NoError(t, err)
	defer criticalResp.Body.Close()
	assert.Equal(t, http.StatusOK, criticalResp.StatusCode)

	var criticalResult map[string]interface{}
	json.NewDecoder(criticalResp.Body).Decode(&criticalResult)

	criticalAlerts, ok := criticalResult["alerts"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(criticalAlerts), 2, "Should have at least 2 critical alerts")

	// Verify all returned alerts have critical severity
	for _, alertInterface := range criticalAlerts {
		alert := alertInterface.(map[string]interface{})
		if severity, ok := alert["severity"].(string); ok {
			assert.Equal(t, "critical", severity, "All alerts should be critical")
		}
	}

	// Test filter by namespace=production
	prodResp, err := helper.MakeRequest("GET", "/api/v2/history?namespace=production", nil)
	require.NoError(t, err)
	defer prodResp.Body.Close()

	var prodResult map[string]interface{}
	json.NewDecoder(prodResp.Body).Decode(&prodResult)

	prodAlerts, ok := prodResult["alerts"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(prodAlerts), 2, "Should have at least 2 production alerts")

	// Test combined filters (severity=warning AND namespace=staging)
	combinedResp, err := helper.MakeRequest("GET", "/api/v2/history?severity=warning&namespace=staging", nil)
	require.NoError(t, err)
	defer combinedResp.Body.Close()

	var combinedResult map[string]interface{}
	json.NewDecoder(combinedResp.Body).Decode(&combinedResult)

	combinedAlerts, ok := combinedResult["alerts"].([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(combinedAlerts), 2, "Should have at least 2 warning+staging alerts")

	t.Logf("✅ Filtering successful (severity, namespace, combined)")
}

// TestE2E_History_Aggregation validates aggregation queries (stats, top alerts)
func TestE2E_History_Aggregation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	infra, err := SetupTestInfrastructure(ctx)
	require.NoError(t, err)
	defer infra.Teardown(ctx)

	helper := NewAPITestHelper(infra)
	fixtures := NewFixtures()

	// Insert alerts with some repeating (for "top alerts" query)
	alertNames := []string{"FrequentAlert", "FrequentAlert", "FrequentAlert", "RareAlert", "OtherAlert"}
	for _, name := range alertNames {
		webhook := fixtures.NewAlertmanagerWebhook().
			AddFiringAlert(name, "info")

		resp, err := helper.MakeRequest("POST", "/webhook", webhook)
		require.NoError(t, err)
		resp.Body.Close()

		time.Sleep(100 * time.Millisecond) // Small delay between alerts
	}

	time.Sleep(1 * time.Second)

	// Test GET /api/v2/history/stats
	statsResp, err := helper.MakeRequest("GET", "/api/v2/history/stats", nil)
	require.NoError(t, err)
	defer statsResp.Body.Close()
	assert.Equal(t, http.StatusOK, statsResp.StatusCode)

	var statsResult map[string]interface{}
	json.NewDecoder(statsResp.Body).Decode(&statsResult)

	// Verify stats structure
	assert.Contains(t, statsResult, "total", "Stats should contain total")

	if total, ok := statsResult["total"].(float64); ok {
		assert.GreaterOrEqual(t, int(total), len(alertNames), "Total should be at least %d", len(alertNames))
	}

	// Stats by severity
	if bySeverity, ok := statsResult["by_severity"].(map[string]interface{}); ok {
		assert.NotEmpty(t, bySeverity, "Should have severity breakdown")
	}

	// Test GET /api/v2/history/top (top N most frequent alerts)
	topResp, err := helper.MakeRequest("GET", "/api/v2/history/top?limit=3", nil)
	require.NoError(t, err)
	defer topResp.Body.Close()
	assert.Equal(t, http.StatusOK, topResp.StatusCode)

	var topResult map[string]interface{}
	json.NewDecoder(topResp.Body).Decode(&topResult)

	topAlerts, ok := topResult["top_alerts"].([]interface{})
	require.True(t, ok, "Response should contain top_alerts array")
	assert.LessOrEqual(t, len(topAlerts), 3, "Should return at most 3 top alerts")

	// Verify "FrequentAlert" is at the top
	if len(topAlerts) > 0 {
		topAlert := topAlerts[0].(map[string]interface{})
		if name, ok := topAlert["alert_name"].(string); ok {
			assert.Equal(t, "FrequentAlert", name, "Most frequent alert should be FrequentAlert")
		}
		if count, ok := topAlert["count"].(float64); ok {
			assert.GreaterOrEqual(t, int(count), 3, "FrequentAlert should appear at least 3 times")
		}
	}

	t.Logf("✅ Aggregation queries successful (stats + top alerts)")
}
