// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// SilenceUIRateLimiter provides rate limiting for Silence UI endpoints.
// Phase 13: Security Hardening enhancement.
type SilenceUIRateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int
	window   time.Duration
	logger   *slog.Logger
}

// NewSilenceUIRateLimiter creates a new rate limiter for Silence UI.
func NewSilenceUIRateLimiter(limit int, window time.Duration, logger *slog.Logger) *SilenceUIRateLimiter {
	rl := &SilenceUIRateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
		logger:   logger,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// cleanup removes old entries periodically.
func (rl *SilenceUIRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for ip, times := range rl.requests {
			// Remove old entries
			validTimes := make([]time.Time, 0)
			for _, t := range times {
				if now.Sub(t) < rl.window {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(rl.requests, ip)
			} else {
				rl.requests[ip] = validTimes
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed.
func (rl *SilenceUIRateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	times, exists := rl.requests[ip]

	if !exists {
		rl.requests[ip] = []time.Time{now}
		return true
	}

	// Remove old entries
	validTimes := make([]time.Time, 0)
	for _, t := range times {
		if now.Sub(t) < rl.window {
			validTimes = append(validTimes, t)
		}
	}

	// Check limit
	if len(validTimes) >= rl.limit {
		rl.logger.Warn("Rate limit exceeded",
			"ip", ip,
			"limit", rl.limit,
			"window", rl.window,
		)
		return false
	}

	// Add current request
	validTimes = append(validTimes, now)
	rl.requests[ip] = validTimes

	return true
}

// Middleware returns a middleware function that enforces rate limiting.
func (rl *SilenceUIRateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr
		// Extract IP from X-Forwarded-For if present
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			ip = forwarded
		}

		if !rl.Allow(ip) {
			// Record rate limit metric (Phase 14 enhancement)
			// Note: This would require access to metrics, can be added later

			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

// SetRateLimiter sets the rate limiter for the handler.
func (h *SilenceUIHandler) SetRateLimiter(limiter *SilenceUIRateLimiter) {
	h.rateLimiter = limiter
}

// RateLimitMiddleware wraps a handler with rate limiting.
func (h *SilenceUIHandler) RateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	if h.rateLimiter == nil {
		// Use default rate limiter if not set
		h.rateLimiter = NewSilenceUIRateLimiter(100, 1*time.Minute, h.logger)
	}
	return h.rateLimiter.Middleware(next)
}
