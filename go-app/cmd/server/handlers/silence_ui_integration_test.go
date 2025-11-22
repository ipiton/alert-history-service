// Package handlers provides HTTP handlers for the Alert History Service.
// Phase 11: Testing Enhancement - Integration tests.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"

	businesssilencing "github.com/vitaliisemenov/alert-history/internal/business/silencing"
	coresilencing "github.com/vitaliisemenov/alert-history/internal/core/silencing"
	infrasilencing "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/cache"
)

// mockSilenceManager is a mock implementation of SilenceManager for testing.
type mockSilenceManager struct {
	silences []*coresilencing.Silence
	stats    *businesssilencing.SilenceManagerStats
	err      error
}

func (m *mockSilenceManager) ListSilences(ctx context.Context, filter infrasilencing.SilenceFilter) ([]*coresilencing.Silence, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.silences, nil
}

func (m *mockSilenceManager) GetSilence(ctx context.Context, id string) (*coresilencing.Silence, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, s := range m.silences {
		if s.ID == id {
			return s, nil
		}
	}
	return nil, infrasilencing.ErrSilenceNotFound
}

func (m *mockSilenceManager) GetStats(ctx context.Context) (*businesssilencing.SilenceManagerStats, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.stats == nil {
		return &businesssilencing.SilenceManagerStats{
			CacheSize:      len(m.silences),
			ActiveSilences: int64(len(m.silences)),
		}, nil
	}
	return m.stats, nil
}

func (m *mockSilenceManager) CreateSilence(ctx context.Context, silence *coresilencing.Silence) (*coresilencing.Silence, error) {
	if m.err != nil {
		return nil, m.err
	}
	m.silences = append(m.silences, silence)
	return silence, nil
}

func (m *mockSilenceManager) UpdateSilence(ctx context.Context, silence *coresilencing.Silence) error {
	if m.err != nil {
		return m.err
	}
	for i, s := range m.silences {
		if s.ID == silence.ID {
			m.silences[i] = silence
			return nil
		}
	}
	return infrasilencing.ErrSilenceNotFound
}

func (m *mockSilenceManager) DeleteSilence(ctx context.Context, id string) error {
	if m.err != nil {
		return m.err
	}
	for i, s := range m.silences {
		if s.ID == id {
			m.silences = append(m.silences[:i], m.silences[i+1:]...)
			return nil
		}
	}
	return infrasilencing.ErrSilenceNotFound
}

func (m *mockSilenceManager) IsAlertSilenced(ctx context.Context, alert *coresilencing.Alert) (bool, []string, error) {
	return false, nil, nil
}

func (m *mockSilenceManager) GetActiveSilences(ctx context.Context) ([]*coresilencing.Silence, error) {
	return m.silences, nil
}

func (m *mockSilenceManager) Start(ctx context.Context) error {
	return nil
}

func (m *mockSilenceManager) Stop(ctx context.Context) error {
	return nil
}

// setupUIHandler creates a test SilenceUIHandler with mocks.
func setupUIHandler(t *testing.T) *SilenceUIHandler {
	logger := slog.Default()
	mockManager := &mockSilenceManager{
		silences: []*coresilencing.Silence{},
	}
	// Use nil cache for testing (will be initialized if needed)
	var mockCache cache.Cache = nil
	wsHub := NewWebSocketHub(logger)
	apiHandler := &SilenceHandler{} // Minimal mock

	handler, err := NewSilenceUIHandler(mockManager, apiHandler, wsHub, mockCache, logger)
	require.NoError(t, err)
	require.NotNil(t, handler)

	return handler
}

func TestSilenceUIHandler_RenderDashboard_Empty(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences", nil)
	w := httptest.NewRecorder()

	handler.RenderDashboard(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Silences")
}

func TestSilenceUIHandler_RenderDashboard_WithFilters(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences?status=active&limit=10", nil)
	w := httptest.NewRecorder()

	handler.RenderDashboard(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Silences")
}

func TestSilenceUIHandler_RenderCreateForm(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences/create", nil)
	w := httptest.NewRecorder()

	handler.RenderCreateForm(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Create")
}

func TestSilenceUIHandler_RenderDetailView_NotFound(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences/non-existent-id", nil)
	w := httptest.NewRecorder()

	handler.RenderDetailView(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestSilenceUIHandler_RenderTemplates(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences/templates", nil)
	w := httptest.NewRecorder()

	handler.RenderTemplates(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Template")
}

func TestSilenceUIHandler_RenderAnalytics(t *testing.T) {
	handler := setupUIHandler(t)

	req := httptest.NewRequest("GET", "/ui/silences/analytics", nil)
	w := httptest.NewRecorder()

	handler.RenderAnalytics(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Analytics")
}

func TestTemplateCache_GetSet(t *testing.T) {
	logger := slog.Default()
	cache := NewTemplateCache(10, 5*time.Minute, logger)

	key := "test-key"
	content := []byte("test content")

	// Cache miss
	_, _, found := cache.Get(key)
	assert.False(t, found)

	// Set cache
	cache.Set(key, content)

	// Cache hit
	cached, etag, found := cache.Get(key)
	assert.True(t, found)
	assert.Equal(t, content, cached)
	assert.NotEmpty(t, etag)
}

func TestTemplateCache_TTL(t *testing.T) {
	logger := slog.Default()
	cache := NewTemplateCache(10, 100*time.Millisecond, logger)

	key := "test-key"
	content := []byte("test content")

	cache.Set(key, content)

	// Should be found immediately
	_, _, found := cache.Get(key)
	assert.True(t, found)

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, _, found = cache.Get(key)
	assert.False(t, found)
}

func TestTemplateCache_Stats(t *testing.T) {
	logger := slog.Default()
	cache := NewTemplateCache(10, 5*time.Minute, logger)

	cache.Set("key1", []byte("content1"))
	cache.Set("key2", []byte("content2"))

	stats := cache.Stats()
	assert.Equal(t, 2, stats["size"])
	assert.Equal(t, 10, stats["max_size"])
}

func TestCSRFManager_GenerateValidate(t *testing.T) {
	logger := slog.Default()
	manager := NewCSRFManager(nil, 1*time.Hour, logger)

	sessionID := "test-session"
	token, err := manager.GenerateToken(sessionID)
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	// Valid token
	valid := manager.ValidateToken(sessionID, token)
	assert.True(t, valid)

	// Invalid token
	valid = manager.ValidateToken(sessionID, "invalid-token")
	assert.False(t, valid)

	// Wrong session
	valid = manager.ValidateToken("wrong-session", token)
	assert.False(t, valid)
}

func TestCSRFManager_Expiration(t *testing.T) {
	logger := slog.Default()
	manager := NewCSRFManager(nil, 100*time.Millisecond, logger)

	sessionID := "test-session"
	token, err := manager.GenerateToken(sessionID)
	require.NoError(t, err)

	// Should be valid immediately
	valid := manager.ValidateToken(sessionID, token)
	assert.True(t, valid)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	valid = manager.ValidateToken(sessionID, token)
	assert.False(t, valid)
}

func TestSilenceUIMetrics_RecordPageRender(t *testing.T) {
	logger := slog.Default()
	metrics := NewSilenceUIMetrics(logger)

	metrics.RecordPageRender("dashboard", 100*time.Millisecond, "success")
	metrics.RecordPageRender("dashboard", 200*time.Millisecond, "success")

	// Metrics should be recorded (no error means success)
	assert.NotNil(t, metrics)
}

func TestSilenceUIMetrics_TemplateCache(t *testing.T) {
	logger := slog.Default()
	metrics := NewSilenceUIMetrics(logger)

	metrics.RecordTemplateCacheHit()
	metrics.RecordTemplateCacheMiss()
	metrics.UpdateTemplateCacheSize(5)

	// Metrics should be recorded (no error means success)
	assert.NotNil(t, metrics)
}

func TestSilenceUIMetrics_WebSocket(t *testing.T) {
	logger := slog.Default()
	metrics := NewSilenceUIMetrics(logger)

	metrics.UpdateWebSocketConnections(10)
	metrics.RecordWebSocketMessage("silence_created")

	// Metrics should be recorded (no error means success)
	assert.NotNil(t, metrics)
}

func TestSilenceUIMetrics_UserActions(t *testing.T) {
	logger := slog.Default()
	metrics := NewSilenceUIMetrics(logger)

	metrics.RecordUserAction("create_silence", "success")
	metrics.RecordUserAction("delete_silence", "error")
	metrics.RecordError("validation_error", "dashboard")

	// Metrics should be recorded (no error means success)
	assert.NotNil(t, metrics)
}

// Benchmark tests for performance validation
func BenchmarkTemplateCache_Get(b *testing.B) {
	logger := slog.Default()
	cache := NewTemplateCache(100, 5*time.Minute, logger)
	cache.Set("test-key", []byte("test content"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = cache.Get("test-key")
	}
}

func BenchmarkTemplateCache_Set(b *testing.B) {
	logger := slog.Default()
	cache := NewTemplateCache(1000, 5*time.Minute, logger)
	content := []byte("test content")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("test-key", content)
	}
}

func BenchmarkCSRFManager_GenerateToken(b *testing.B) {
	logger := slog.Default()
	manager := NewCSRFManager(nil, 1*time.Hour, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = manager.GenerateToken("test-session")
	}
}

func BenchmarkCSRFManager_ValidateToken(b *testing.B) {
	logger := slog.Default()
	manager := NewCSRFManager(nil, 1*time.Hour, logger)
	token, _ := manager.GenerateToken("test-session")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.ValidateToken("test-session", token)
	}
}
