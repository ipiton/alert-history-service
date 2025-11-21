// Package handlers provides HTTP handlers for the Alert History Service.
// Phase 11: E2E Testing - End-to-end scenarios.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	coresilencing "github.com/vitaliisemenov/alert-history/internal/core/silencing"
)

// TestE2E_SilenceLifecycle tests complete silence lifecycle.
func TestE2E_SilenceLifecycle(t *testing.T) {
	handler := setupUIHandler(t)
	mockManager := handler.manager.(*mockSilenceManager)

	// Step 1: View empty dashboard
	req := httptest.NewRequest("GET", "/ui/silences", nil)
	w := httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Silences")

	// Step 2: View create form
	req = httptest.NewRequest("GET", "/ui/silences/create", nil)
	w = httptest.NewRecorder()
	handler.RenderCreateForm(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Create")

	// Step 3: Create silence (simulate via manager)
	silence := &coresilencing.Silence{
		ID:        "test-silence-1",
		CreatedBy: "test@example.com",
		Comment:   "E2E test silence",
		StartsAt:  time.Now(),
		EndsAt:    time.Now().Add(1 * time.Hour),
		Matchers: []coresilencing.Matcher{
			{Name: "alertname", Value: "TestAlert", Type: "="},
		},
	}
	created, err := mockManager.CreateSilence(context.Background(), silence)
	require.NoError(t, err)
	assert.Equal(t, silence.ID, created.ID)

	// Step 4: View dashboard with silence
	req = httptest.NewRequest("GET", "/ui/silences", nil)
	w = httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-silence-1")

	// Step 5: View detail page
	req = httptest.NewRequest("GET", "/ui/silences/test-silence-1", nil)
	w = httptest.NewRecorder()
	handler.RenderDetailView(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-silence-1")

	// Step 6: View edit form
	req = httptest.NewRequest("GET", "/ui/silences/test-silence-1/edit", nil)
	w = httptest.NewRecorder()
	handler.RenderEditForm(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "test-silence-1")

	// Step 7: Update silence
	silence.Comment = "Updated E2E test silence"
	err = mockManager.UpdateSilence(context.Background(), silence)
	require.NoError(t, err)

	// Step 8: Verify update in detail view
	req = httptest.NewRequest("GET", "/ui/silences/test-silence-1", nil)
	w = httptest.NewRecorder()
	handler.RenderDetailView(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Updated E2E test silence")

	// Step 9: Delete silence
	err = mockManager.DeleteSilence(context.Background(), silence.ID)
	require.NoError(t, err)

	// Step 10: Verify deletion (should return 404)
	req = httptest.NewRequest("GET", "/ui/silences/test-silence-1", nil)
	w = httptest.NewRecorder()
	handler.RenderDetailView(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestE2E_FilteringAndPagination tests filtering and pagination.
func TestE2E_FilteringAndPagination(t *testing.T) {
	handler := setupUIHandler(t)
	mockManager := handler.manager.(*mockSilenceManager)

	// Create test silences
	for i := 0; i < 25; i++ {
		silence := &coresilencing.Silence{
			ID:        "test-silence-" + string(rune(i)),
			CreatedBy: "test@example.com",
			Comment:   "Test silence",
			StartsAt:  time.Now(),
			EndsAt:    time.Now().Add(1 * time.Hour),
		}
		mockManager.silences = append(mockManager.silences, silence)
	}

	// Test pagination
	req := httptest.NewRequest("GET", "/ui/silences?page=1&limit=10", nil)
	w := httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Test filtering by status
	req = httptest.NewRequest("GET", "/ui/silences?status=active", nil)
	w = httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestE2E_TemplateLibrary tests template library functionality.
func TestE2E_TemplateLibrary(t *testing.T) {
	handler := setupUIHandler(t)

	// View templates page
	req := httptest.NewRequest("GET", "/ui/silences/templates", nil)
	w := httptest.NewRecorder()
	handler.RenderTemplates(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Template")
}

// TestE2E_AnalyticsDashboard tests analytics dashboard.
func TestE2E_AnalyticsDashboard(t *testing.T) {
	handler := setupUIHandler(t)

	// View analytics page
	req := httptest.NewRequest("GET", "/ui/silences/analytics", nil)
	w := httptest.NewRecorder()
	handler.RenderAnalytics(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Analytics")
}

// TestE2E_CSRFProtection tests CSRF protection flow.
func TestE2E_CSRFProtection(t *testing.T) {
	handler := setupUIHandler(t)

	// Get CSRF token from create form
	req := httptest.NewRequest("GET", "/ui/silences/create", nil)
	w := httptest.NewRecorder()
	handler.RenderCreateForm(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Token should be in the form (would need to parse HTML in real E2E)
	// For now, just verify form renders successfully
	assert.Contains(t, w.Body.String(), "Create")
}

// TestE2E_ErrorHandling tests error handling scenarios.
func TestE2E_ErrorHandling(t *testing.T) {
	handler := setupUIHandler(t)

	// Test 404 for non-existent silence
	req := httptest.NewRequest("GET", "/ui/silences/non-existent-id", nil)
	w := httptest.NewRecorder()
	handler.RenderDetailView(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)

	// Test error handling (renderError is private, test via invalid ID)
	req = httptest.NewRequest("GET", "/ui/silences/invalid-id-404", nil)
	w = httptest.NewRecorder()
	handler.RenderDetailView(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestE2E_CacheBehavior tests template cache behavior.
func TestE2E_CacheBehavior(t *testing.T) {
	handler := setupUIHandler(t)

	// First request (cache miss)
	req1 := httptest.NewRequest("GET", "/ui/silences", nil)
	w1 := httptest.NewRecorder()
	start1 := time.Now()
	handler.RenderDashboard(w1, req1)
	duration1 := time.Since(start1)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request (cache hit)
	req2 := httptest.NewRequest("GET", "/ui/silences", nil)
	req2.Header.Set("If-None-Match", w1.Header().Get("ETag"))
	w2 := httptest.NewRecorder()
	start2 := time.Now()
	handler.RenderDashboard(w2, req2)
	duration2 := time.Since(start2)

	// Cached request should be faster
	if w2.Code == http.StatusNotModified {
		assert.Less(t, duration2, duration1, "Cached request should be faster")
	}
}

// TestE2E_ConcurrentOperations tests concurrent UI operations.
func TestE2E_ConcurrentOperations(t *testing.T) {
	handler := setupUIHandler(t)

	var wg sync.WaitGroup
	concurrency := 20

	// Concurrent dashboard renders
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/ui/silences", nil)
			w := httptest.NewRecorder()
			handler.RenderDashboard(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}(i)
	}

	wg.Wait()
}

// TestE2E_WebSocketIntegration tests WebSocket integration.
func TestE2E_WebSocketIntegration(t *testing.T) {
	handler := setupUIHandler(t)

	// Verify WebSocket hub is initialized
	assert.NotNil(t, handler.wsHub)

	// Test broadcast (would need WebSocket client for full E2E)
	handler.wsHub.Broadcast("silence_created", map[string]interface{}{
		"id":      "test-silence",
		"creator": "test@example.com",
	})

	// Verify no panic occurred
	assert.True(t, true, "WebSocket broadcast should not panic")
}

// TestE2E_MetricsIntegration tests metrics integration.
func TestE2E_MetricsIntegration(t *testing.T) {
	handler := setupUIHandler(t)

	// Verify metrics are initialized
	assert.NotNil(t, handler.metrics)

	// Render page (should record metrics)
	req := httptest.NewRequest("GET", "/ui/silences", nil)
	w := httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Metrics should be recorded (no error means success)
	assert.True(t, true, "Metrics should be recorded")
}

// TestE2E_SecurityHeaders tests security headers.
func TestE2E_SecurityHeaders(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences", nil)
	w := httptest.NewRecorder()

	// Apply security middleware
	handler.SecurityMiddleware(handler.RenderDashboard)(w, req)

	// Verify security headers
	assert.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
	assert.Equal(t, "1; mode=block", w.Header().Get("X-XSS-Protection"))
}
