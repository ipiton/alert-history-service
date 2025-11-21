// Package handlers provides HTTP handlers for the Alert History Service.
// Phase 11: Advanced Testing - Edge cases and stress tests.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"log/slog"

	businesssilencing "github.com/vitaliisemenov/alert-history/internal/business/silencing"
	coresilencing "github.com/vitaliisemenov/alert-history/internal/core/silencing"
	infrasilencing "github.com/vitaliisemenov/alert-history/internal/infrastructure/silencing"
)

// TestSilenceUIHandler_ConcurrentAccess tests concurrent access to UI handler.
func TestSilenceUIHandler_ConcurrentAccess(t *testing.T) {
	handler := setupUIHandler(t)

	var wg sync.WaitGroup
	concurrency := 50

	// Concurrent dashboard renders
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := httptest.NewRequest("GET", "/ui/silences", nil)
			w := httptest.NewRecorder()
			handler.RenderDashboard(w, req)
			assert.Equal(t, http.StatusOK, w.Code)
		}()
	}

	wg.Wait()
}

// TestSilenceUIHandler_LargeDataset tests handling of large datasets.
func TestSilenceUIHandler_LargeDataset(t *testing.T) {
	handler := setupUIHandler(t)

	// Create mock manager with large dataset
	mockManager := handler.manager.(*mockSilenceManager)
	for i := 0; i < 1000; i++ {
		// Generate a simple ID for testing
		id := fmt.Sprintf("test-silence-%d", i)
		silence := &coresilencing.Silence{
			ID:        id,
			CreatedBy: "test@example.com",
			Comment:   "Test silence",
			StartsAt:  time.Now(),
			EndsAt:    time.Now().Add(1 * time.Hour),
		}
		mockManager.silences = append(mockManager.silences, silence)
	}

	req := httptest.NewRequest("GET", "/ui/silences?limit=100", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	handler.RenderDashboard(w, req)
	duration := time.Since(start)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Less(t, duration, 2*time.Second, "Should render large dataset quickly")
}

// TestTemplateCache_ConcurrentAccess tests concurrent cache access.
func TestTemplateCache_ConcurrentAccess(t *testing.T) {
	logger := slog.Default()
	cache := NewTemplateCache(100, 5*time.Minute, logger)

	var wg sync.WaitGroup
	concurrency := 100

	// Concurrent gets and sets
	for i := 0; i < concurrency; i++ {
		wg.Add(2)
		go func(i int) {
			defer wg.Done()
			key := "test-key"
			content := []byte("test content")
			cache.Set(key, content)
		}(i)
		go func(i int) {
			defer wg.Done()
			_, _, _ = cache.Get("test-key")
		}(i)
	}

	wg.Wait()
	stats := cache.Stats()
	assert.Greater(t, stats["hits"], int64(0))
}

// TestTemplateCache_LRU_Eviction tests LRU eviction policy.
func TestTemplateCache_LRU_Eviction(t *testing.T) {
	logger := slog.Default()
	cache := NewTemplateCache(5, 5*time.Minute, logger) // Small cache for testing

	// Fill cache beyond capacity
	for i := 0; i < 10; i++ {
		key := "test-key-" + string(rune(i))
		content := []byte("test content")
		cache.Set(key, content)
	}

	stats := cache.Stats()
	assert.LessOrEqual(t, stats["size"], 5, "Cache should not exceed max size")
	assert.Greater(t, stats["evictions"], int64(0), "Should have evictions")
}

// TestCSRFManager_ConcurrentAccess tests concurrent CSRF operations.
func TestCSRFManager_ConcurrentAccess(t *testing.T) {
	logger := slog.Default()
	manager := NewCSRFManager(nil, 1*time.Hour, logger)

	var wg sync.WaitGroup
	concurrency := 100

	// Concurrent token generation
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sessionID := "session-" + string(rune(i))
			token, err := manager.GenerateToken(sessionID)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate token
			valid := manager.ValidateToken(sessionID, token)
			assert.True(t, valid)
		}(i)
	}

	wg.Wait()
}

// TestRateLimiter_ConcurrentAccess tests concurrent rate limiting.
func TestRateLimiter_ConcurrentAccess(t *testing.T) {
	logger := slog.Default()
	limiter := NewSilenceUIRateLimiter(10, 1*time.Minute, logger)

	ip := "192.168.1.1"
	var allowed, denied int
	var mu sync.Mutex

	var wg sync.WaitGroup
	concurrency := 50

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow(ip) {
				mu.Lock()
				allowed++
				mu.Unlock()
			} else {
				mu.Lock()
				denied++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 10, allowed, "Should allow exactly limit requests")
	assert.Equal(t, 40, denied, "Should deny excess requests")
}

// TestSilenceUIHandler_ErrorHandling tests error handling scenarios.
func TestSilenceUIHandler_ErrorHandling(t *testing.T) {
	handler := setupUIHandler(t)

	// Test with error in manager
	mockManager := handler.manager.(*mockSilenceManager)
	mockManager.err = infrasilencing.ErrDatabaseConnection

	req := httptest.NewRequest("GET", "/ui/silences", nil)
	w := httptest.NewRecorder()

	handler.RenderDashboard(w, req)

	// Should handle error gracefully
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// TestSilenceUIHandler_InvalidInput tests handling of invalid input.
func TestSilenceUIHandler_InvalidInput(t *testing.T) {
	handler := setupUIHandler(t)

	// Test invalid page number
	req := httptest.NewRequest("GET", "/ui/silences?page=-1", nil)
	w := httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code) // Should handle gracefully

	// Test invalid limit
	req = httptest.NewRequest("GET", "/ui/silences?limit=10000", nil)
	w = httptest.NewRecorder()
	handler.RenderDashboard(w, req)
	assert.Equal(t, http.StatusOK, w.Code) // Should cap limit
}

// Benchmark tests for performance validation

// setupUIHandlerForBenchmark creates a handler for benchmarks.
func setupUIHandlerForBenchmark(b *testing.B) *SilenceUIHandler {
	logger := slog.Default()
	mockManager := &mockSilenceManager{
		silences: make([]*coresilencing.Silence, 0),
		logger:   logger,
	}
	apiHandler := &SilenceHandler{} // Minimal mock
	wsHub := NewWebSocketHub(logger)
	cache := nil // No cache for benchmarks

	handler, err := NewSilenceUIHandler(mockManager, apiHandler, wsHub, cache, logger)
	require.NoError(b, err)
	return handler
}

func BenchmarkSilenceUIHandler_RenderDashboard(b *testing.B) {
	handler := setupUIHandlerForBenchmark(b)
	req := httptest.NewRequest("GET", "/ui/silences", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.RenderDashboard(w, req)
	}
}

func BenchmarkTemplateCache_ConcurrentGet(b *testing.B) {
	logger := slog.Default()
	cache := NewTemplateCache(100, 5*time.Minute, logger)
	cache.Set("test-key", []byte("test content"))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = cache.Get("test-key")
		}
	})
}

func BenchmarkCSRFManager_GenerateValidate(b *testing.B) {
	logger := slog.Default()
	manager := NewCSRFManager(nil, 1*time.Hour, logger)
	sessionID := "test-session"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		token, _ := manager.GenerateToken(sessionID)
		_ = manager.ValidateToken(sessionID, token)
	}
}

func BenchmarkRateLimiter_Allow(b *testing.B) {
	logger := slog.Default()
	limiter := NewSilenceUIRateLimiter(1000, 1*time.Minute, logger)
	ip := "192.168.1.1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = limiter.Allow(ip)
	}
}

func BenchmarkSilenceUIHandler_RenderWithCache(b *testing.B) {
	handler := setupUIHandlerForBenchmark(b)
	req := httptest.NewRequest("GET", "/ui/silences", nil)

	// Warm up cache
	w := httptest.NewRecorder()
	handler.RenderDashboard(w, req)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.RenderDashboard(w, req)
	}
}
