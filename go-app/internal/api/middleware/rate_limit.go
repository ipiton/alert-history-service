package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements token bucket rate limiting per client
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit // Requests per second
	burst    int        // Burst capacity
}

// NewRateLimiter creates a new rate limiter
//
// Parameters:
//   - requestsPerMinute: Maximum requests per minute per client
//   - burst: Burst capacity (allows temporary spikes)
//
// Example:
//
//	limiter := NewRateLimiter(100, 20) // 100 req/min, burst 20
func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(float64(requestsPerMinute) / 60.0), // Convert to per-second
		burst:    burst,
	}
}

// GetLimiter returns or creates a limiter for the given client ID
func (rl *RateLimiter) GetLimiter(clientID string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[clientID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[clientID] = limiter
	}

	return limiter
}

// Cleanup removes stale limiters (full token bucket = inactive)
// Should be called periodically (e.g., every 5 minutes)
func (rl *RateLimiter) Cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for key, limiter := range rl.limiters {
		// If limiter has full tokens, it hasn't been used recently
		if limiter.TokensAt(now) == float64(rl.burst) {
			delete(rl.limiters, key)
		}
	}
}

// RateLimitMiddleware applies per-client rate limiting
//
// Rate limits are enforced per client (identified by API key or IP address).
// When rate limit is exceeded, returns 429 Too Many Requests with headers:
//   - X-RateLimit-Limit: Maximum requests per minute
//   - X-RateLimit-Remaining: Remaining requests
//   - X-RateLimit-Reset: Unix timestamp when limit resets
//   - Retry-After: Seconds until retry
func RateLimitMiddleware(requestsPerMinute, burst int) func(http.Handler) http.Handler {
	limiter := NewRateLimiter(requestsPerMinute, burst)

	// Start cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.Cleanup()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (API key or IP)
			clientID := getClientID(r)

			// Check rate limit
			if !limiter.GetLimiter(clientID).Allow() {
				// Rate limit exceeded
				w.Header().Set(RateLimitLimitHeader, fmt.Sprintf("%d", requestsPerMinute))
				w.Header().Set(RateLimitRemainingHeader, "0")
				w.Header().Set(RateLimitResetHeader, fmt.Sprintf("%d", time.Now().Add(time.Minute).Unix()))
				w.Header().Set("Retry-After", "60")

				http.Error(w, `{"error":{"code":"RATE_LIMIT_EXCEEDED","message":"Rate limit exceeded. Please retry after 60 seconds."}}`, http.StatusTooManyRequests)
				return
			}

			// Add rate limit headers to response
			w.Header().Set(RateLimitLimitHeader, fmt.Sprintf("%d", requestsPerMinute))
			// Note: RateLimitRemainingHeader would require tracking tokens, skipped for simplicity

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// getClientID extracts client identifier from request
// Priority: User API key > X-Forwarded-For > X-Real-IP > RemoteAddr
func getClientID(r *http.Request) string {
	// Try to get API key from context (set by AuthMiddleware)
	if user, ok := r.Context().Value(UserContextKey).(*User); ok && user != nil {
		return user.APIKey
	}

	// Fallback to IP address
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.Header.Get("X-Real-IP")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}
