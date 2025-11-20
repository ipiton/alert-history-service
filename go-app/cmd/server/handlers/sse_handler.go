// Package handlers provides HTTP handlers for the Alert History Service.
// TN-78: Real-time Updates (SSE/WebSocket) - SSE Handler Implementation
package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/realtime"
)

// SSEHandler handles Server-Sent Events connections.
// GET /api/v2/events/stream
type SSEHandler struct {
	eventBus *realtime.DefaultEventBus
	logger   *slog.Logger
	metrics  *realtime.RealtimeMetrics
}

// NewSSEHandler creates a new SSE handler.
func NewSSEHandler(eventBus *realtime.DefaultEventBus, logger *slog.Logger, metrics *realtime.RealtimeMetrics) *SSEHandler {
	return &SSEHandler{
		eventBus: eventBus,
		logger:   logger.With("component", "sse_handler"),
		metrics:  metrics,
	}
}

// ServeHTTP handles GET /api/v2/events/stream
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// CORS headers (if needed)
	if origin := r.Header.Get("Origin"); origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Create SSE subscriber
	subscriber := NewSSESubscriber(w, r.Context(), h.logger)
	if err := h.eventBus.Subscribe(subscriber); err != nil {
		h.logger.Error("Failed to subscribe SSE client", "error", err)
		http.Error(w, "Failed to establish connection", http.StatusInternalServerError)
		return
	}
	defer h.eventBus.Unsubscribe(subscriber)

	h.logger.Info("SSE client connected",
		"remote_addr", r.RemoteAddr,
		"subscriber_id", subscriber.ID(),
	)

	// Start keep-alive ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// Event loop
	for {
		select {
		case <-r.Context().Done():
			h.logger.Debug("SSE client disconnected",
				"subscriber_id", subscriber.ID(),
				"reason", "context cancelled",
			)
			return

		case <-ticker.C:
			// Send keep-alive ping
			if _, err := fmt.Fprintf(w, ": ping\n\n"); err != nil {
				h.logger.Warn("Failed to send SSE ping",
					"subscriber_id", subscriber.ID(),
					"error", err,
				)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}

		case event := <-subscriber.EventChan():
			// Send event in SSE format
			if err := h.sendSSEEvent(w, event); err != nil {
				h.logger.Warn("Failed to send SSE event",
					"subscriber_id", subscriber.ID(),
					"event_type", event.Type,
					"error", err,
				)
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}

// sendSSEEvent sends an event in SSE format: "data: {...}\n\n"
func (h *SSEHandler) sendSSEEvent(w http.ResponseWriter, event realtime.Event) error {
	// Marshal event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Write SSE format: "data: {...}\n\n"
	_, err = fmt.Fprintf(w, "data: %s\n\n", data)
	return err
}
