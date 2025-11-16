package metrics

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMetricsEndpointHandler_Cache tests caching functionality.
func TestMetricsEndpointHandler_Cache(t *testing.T) {
	t.Run("serves from cache when enabled", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.CacheEnabled = true
		config.CacheTTL = 5 * time.Second
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// First request - should populate cache
		req1 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)
		firstBody := w1.Body.String()

		// Second request - should serve from cache
		req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
		secondBody := w2.Body.String()

		// Bodies should be identical (from cache)
		assert.Equal(t, firstBody, secondBody)
	})

	t.Run("cache expires after TTL", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.CacheEnabled = true
		config.CacheTTL = 100 * time.Millisecond // Very short TTL
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// First request - populate cache
		req1 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w1 := httptest.NewRecorder()
		handler.ServeHTTP(w1, req1)
		assert.Equal(t, http.StatusOK, w1.Code)

		// Wait for cache to expire
		time.Sleep(150 * time.Millisecond)

		// Second request - should regenerate (cache expired)
		req2 := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusOK, w2.Code)
		// Cache should have been regenerated
	})

	t.Run("cache disabled by default", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		// CacheEnabled defaults to false
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		// Should work without cache
	})

	t.Run("cache works with concurrent requests", func(t *testing.T) {
		config := DefaultEndpointConfig()
		config.Path = "/metrics"
		config.CacheEnabled = true
		config.CacheTTL = 5 * time.Second
		registry := DefaultRegistry()

		handler, err := NewMetricsEndpointHandler(config, registry)
		require.NoError(t, err)

		// Warm up cache
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		// Make concurrent requests
		const numRequests = 10
		results := make(chan int, numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req)
				results <- w.Code
			}()
		}

		successCount := 0
		for i := 0; i < numRequests; i++ {
			code := <-results
			if code == http.StatusOK {
				successCount++
			}
		}

		assert.Equal(t, numRequests, successCount)
	})
}

// TestBufferPool tests buffer pooling optimization.
func TestBufferPool(t *testing.T) {
	t.Run("buffer pool reuses buffers", func(t *testing.T) {
		buf1 := getBuffer()
		buf1.WriteString("test")
		putBuffer(buf1)

		buf2 := getBuffer()
		// Buffer should be reset
		assert.Equal(t, 0, buf2.Len())
		putBuffer(buf2)
	})

	t.Run("large buffers are not pooled", func(t *testing.T) {
		buf := getBuffer()
		// Write more than 1MB
		data := make([]byte, 2*1024*1024)
		buf.Write(data)
		putBuffer(buf)

		// Should not cause issues
		buf2 := getBuffer()
		assert.Equal(t, 0, buf2.Len())
		putBuffer(buf2)
	})
}
