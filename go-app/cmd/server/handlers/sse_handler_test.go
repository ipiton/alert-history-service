// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vitaliisemenov/alert-history/internal/realtime"
	"log/slog"
)

func TestSSEHandler_ServeHTTP(t *testing.T) {
	metrics := realtime.NewRealtimeMetrics("test")
	eventBus := realtime.NewEventBus(slog.Default(), metrics)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := eventBus.Start(ctx)
	require.NoError(t, err)
	defer eventBus.Stop(context.Background())

	handler := NewSSEHandler(eventBus, slog.Default(), metrics)

	req, err := http.NewRequest("GET", "/api/v2/events/stream", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	// Start handler in goroutine (it will block)
	done := make(chan bool)
	go func() {
		handler.ServeHTTP(rr, req)
		done <- true
	}()

	// Wait a bit for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Check headers
	assert.Equal(t, "text/event-stream", rr.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rr.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rr.Header().Get("Connection"))

	// Cancel request context to stop handler
	cancel()
	<-done
}

func TestSSEHandler_KeepAlive(t *testing.T) {
	metrics := realtime.NewRealtimeMetrics("test")
	eventBus := realtime.NewEventBus(slog.Default(), metrics)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := eventBus.Start(ctx)
	require.NoError(t, err)
	defer eventBus.Stop(context.Background())

	handler := NewSSEHandler(eventBus, slog.Default(), metrics)

	req, err := http.NewRequest("GET", "/api/v2/events/stream", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	// Start handler in goroutine
	done := make(chan bool)
	go func() {
		handler.ServeHTTP(rr, req)
		done <- true
	}()

	// Wait for keep-alive ping (should be sent every 30s, but we'll wait 1s)
	time.Sleep(1100 * time.Millisecond)

	// Check that keep-alive ping was sent
	body := rr.Body.String()
	assert.Contains(t, body, ": ping")

	// Cancel request context
	cancel()
	<-done
}

func TestSSEHandler_EventSending(t *testing.T) {
	metrics := realtime.NewRealtimeMetrics("test")
	eventBus := realtime.NewEventBus(slog.Default(), metrics)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := eventBus.Start(ctx)
	require.NoError(t, err)
	defer eventBus.Stop(context.Background())

	handler := NewSSEHandler(eventBus, slog.Default(), metrics)

	req, err := http.NewRequest("GET", "/api/v2/events/stream", nil)
	require.NoError(t, err)

	rr := httptest.NewRecorder()

	// Start handler in goroutine
	done := make(chan bool)
	go func() {
		handler.ServeHTTP(rr, req)
		done <- true
	}()

	// Wait for connection to establish
	time.Sleep(100 * time.Millisecond)

	// Publish an event
	event := realtime.NewEvent("test_event", map[string]interface{}{"key": "value"}, "test_source")
	err = eventBus.Publish(*event)
	require.NoError(t, err)

	// Wait for event to be sent
	time.Sleep(200 * time.Millisecond)

	// Check that event was sent in SSE format
	body := rr.Body.String()
	assert.Contains(t, body, "data:")
	assert.Contains(t, body, "test_event")
	assert.Contains(t, body, "key")
	assert.Contains(t, body, "value")

	// Cancel request context
	cancel()
	<-done
}

func TestSSEHandler_CORS(t *testing.T) {
	metrics := realtime.NewRealtimeMetrics("test")
	eventBus := realtime.NewEventBus(slog.Default(), metrics)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err := eventBus.Start(ctx)
	require.NoError(t, err)
	defer eventBus.Stop(context.Background())

	handler := NewSSEHandler(eventBus, slog.Default(), metrics)

	req, err := http.NewRequest("GET", "/api/v2/events/stream", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://example.com")

	rr := httptest.NewRecorder()

	// Start handler in goroutine
	done := make(chan bool)
	go func() {
		handler.ServeHTTP(rr, req)
		done <- true
	}()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Check CORS headers
	assert.Equal(t, "https://example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))

	// Cancel request context
	cancel()
	<-done
}
