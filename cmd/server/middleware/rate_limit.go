package middleware

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RateLimitMiddleware enforces rate limits (simplified in-memory version)
// Production version should use Redis for distributed rate limiting
func RateLimitMiddleware(config *RateLimitConfig) Middleware {
	// Initialize in-memory rate limiters
	perIPLimiter := newInMemoryRateLimiter(config.PerIPLimit, time.Minute)
	globalLimiter := newFixedWindowLimiter(config.GlobalLimit, time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract client IP
			clientIP := extractClientIP(r)

			// Check global rate limit first (fast check)
			if !globalLimiter.Allow() {
				if config.Logger != nil {
					config.Logger.Warn("Global rate limit exceeded",
						"request_id", GetRequestID(r.Context()),
						"client_ip", clientIP,
						"limit", config.GlobalLimit,
					)
				}

				writeRateLimitError(w, r, "global", config.GlobalLimit)
				return
			}

			// Check per-IP rate limit
			if !perIPLimiter.Allow(clientIP) {
				if config.Logger != nil {
					config.Logger.Warn("Per-IP rate limit exceeded",
						"request_id", GetRequestID(r.Context()),
						"client_ip", clientIP,
						"limit", config.PerIPLimit,
					)
				}

				writeRateLimitError(w, r, "per_ip", config.PerIPLimit)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractClientIP extracts client IP from request headers
func extractClientIP(r *http.Request) string {
	// Try X-Forwarded-For header first (behind proxy)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Try X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// writeRateLimitError writes 429 rate limit error response
func writeRateLimitError(w http.ResponseWriter, r *http.Request, limitType string, limit int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", GetRequestID(r.Context()))
	w.Header().Set("Retry-After", "60")
	w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
	w.Header().Set("X-RateLimit-Remaining", "0")
	w.WriteHeader(http.StatusTooManyRequests)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "rate_limited",
		"message":     "Too many requests. Please retry after 60 seconds",
		"limit_type":  limitType,
		"limit":       limit,
		"retry_after": 60,
		"request_id":  GetRequestID(r.Context()),
	})
}

// inMemoryRateLimiter implements simple in-memory rate limiting per key (IP)
type inMemoryRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*tokenBucket
	limit    int
	window   time.Duration
}

func newInMemoryRateLimiter(limit int, window time.Duration) *inMemoryRateLimiter {
	return &inMemoryRateLimiter{
		limiters: make(map[string]*tokenBucket),
		limit:    limit,
		window:   window,
	}
}

func (rl *inMemoryRateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = newTokenBucket(rl.limit, rl.window)
		rl.limiters[key] = limiter
	}

	return limiter.Allow()
}

// tokenBucket implements simple token bucket algorithm
type tokenBucket struct {
	tokens    int
	maxTokens int
	refillAt  time.Time
	window    time.Duration
}

func newTokenBucket(maxTokens int, window time.Duration) *tokenBucket {
	return &tokenBucket{
		tokens:    maxTokens,
		maxTokens: maxTokens,
		refillAt:  time.Now().Add(window),
		window:    window,
	}
}

func (tb *tokenBucket) Allow() bool {
	now := time.Now()

	// Refill if window has passed
	if now.After(tb.refillAt) {
		tb.tokens = tb.maxTokens
		tb.refillAt = now.Add(tb.window)
	}

	// Check if tokens available
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}

	return false
}

// fixedWindowLimiter implements global fixed window rate limiting
type fixedWindowLimiter struct {
	mu       sync.Mutex
	count    int
	limit    int
	windowEnd time.Time
	window   time.Duration
}

func newFixedWindowLimiter(limit int, window time.Duration) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		count:    0,
		limit:    limit,
		windowEnd: time.Now().Add(window),
		window:   window,
	}
}

func (fl *fixedWindowLimiter) Allow() bool {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	now := time.Now()

	// Reset counter if window has passed
	if now.After(fl.windowEnd) {
		fl.count = 0
		fl.windowEnd = now.Add(fl.window)
	}

	// Check if under limit
	if fl.count < fl.limit {
		fl.count++
		return true
	}

	return false
}
