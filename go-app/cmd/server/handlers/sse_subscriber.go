// Package handlers provides HTTP handlers for the Alert History Service.
// TN-78: Real-time Updates (SSE/WebSocket) - SSE Subscriber Implementation
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/vitaliisemenov/alert-history/internal/realtime"
)

// SSESubscriber implements EventSubscriber for SSE connections.
type SSESubscriber struct {
	id        string
	writer    http.ResponseWriter
	ctx       context.Context
	eventChan chan realtime.Event
	logger    *slog.Logger
	mu        sync.Mutex
	closed    bool
}

// NewSSESubscriber creates a new SSE subscriber.
func NewSSESubscriber(w http.ResponseWriter, ctx context.Context, logger *slog.Logger) *SSESubscriber {
	subscriberID := uuid.New().String()
	return &SSESubscriber{
		id:        subscriberID,
		writer:    w,
		ctx:       ctx,
		eventChan: make(chan realtime.Event, 10), // Buffered channel
		logger:    logger.With("component", "sse_subscriber", "subscriber_id", subscriberID),
		closed:    false,
	}
}

// ID returns the subscriber ID.
func (s *SSESubscriber) ID() string {
	return s.id
}

// Send sends an event to the subscriber.
func (s *SSESubscriber) Send(event realtime.Event) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return realtime.ErrSubscriberClosed
	}
	s.mu.Unlock()

	select {
	case s.eventChan <- event:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		// Channel full, drop event
		s.logger.Warn("SSE subscriber channel full, dropping event",
			"subscriber_id", s.id,
			"event_type", event.Type,
		)
		return fmt.Errorf("subscriber channel full")
	}
}

// EventChan returns the event channel (for reading events).
func (s *SSESubscriber) EventChan() <-chan realtime.Event {
	return s.eventChan
}

// Close closes the subscriber.
func (s *SSESubscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.eventChan)
	return nil
}

// Context returns the subscriber context.
func (s *SSESubscriber) Context() context.Context {
	return s.ctx
}
