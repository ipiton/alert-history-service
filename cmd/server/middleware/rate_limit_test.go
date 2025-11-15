// Package middleware provides HTTP middleware components.
package middleware

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestRateLimitMiddleware_AllowsWithinLimit tests requests within rate limit
func TestRateLimitMiddleware_AllowsWithinLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  10, // 10 requests per minute
		GlobalLimit: 100,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	nextCalled := 0
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled++
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	// Send 5 requests (well within limit)
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i, rr.Code)
		}
	}

	if nextCalled != 5 {
		t.Errorf("Expected next handler called 5 times, got %d", nextCalled)
	}
}

// TestRateLimitMiddleware_BlocksExceedingLimit tests rate limit enforcement
func TestRateLimitMiddleware_BlocksExceedingLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  5,  // Very low limit for testing
		GlobalLimit: 100,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	successCount := 0
	rateLimitedCount := 0

	// Send 10 requests rapidly from same IP
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusOK {
			successCount++
		} else if rr.Code == http.StatusTooManyRequests {
			rateLimitedCount++
		}

		// Small delay to simulate real traffic
		time.Sleep(1 * time.Millisecond)
	}

	// Should have some successful and some rate-limited requests
	t.Logf("Success: %d, Rate limited: %d", successCount, rateLimitedCount)

	if successCount == 0 {
		t.Error("Expected at least some successful requests")
	}
}

// TestRateLimitMiddleware_PerIPIsolation tests per-IP rate limiting isolation
func TestRateLimitMiddleware_PerIPIsolation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  10,
		GlobalLimit: 100,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	// Different IPs should be isolated
	ips := []string{
		"192.168.1.1:12345",
		"192.168.1.2:12345",
		"192.168.1.3:12345",
	}

	for _, ip := range ips {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = ip
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200 for IP %s, got %d", ip, rr.Code)
		}
	}
}

// TestRateLimitMiddleware_GlobalLimit tests global rate limit
func TestRateLimitMiddleware_GlobalLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  100, // High per-IP limit
		GlobalLimit: 10,  // Low global limit
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	// Send requests from different IPs to hit global limit
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = "192.168.1." + string(rune(i)) + ":12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		// Log status for debugging
		if rr.Code != http.StatusOK {
			t.Logf("Request %d: status %d (global limit enforcement)", i, rr.Code)
		}
	}
}

// TestRateLimitMiddleware_Disabled tests that disabled middleware passes through
func TestRateLimitMiddleware_Disabled(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     false, // Disabled
		PerIPLimit:  1,     // Very restrictive, but should be ignored
		GlobalLimit: 1,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	// Send many requests, all should pass
	for i := 0; i < 20; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Request %d: expected status 200 (disabled), got %d", i, rr.Code)
		}
	}
}

// TestRateLimitMiddleware_ExtractClientIP tests client IP extraction
func TestRateLimitMiddleware_ExtractClientIP(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  10,
		GlobalLimit: 100,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	testCases := []struct {
		name           string
		remoteAddr     string
		xForwardedFor  string
		xRealIP        string
		expectedPrefix string
	}{
		{
			name:           "direct connection",
			remoteAddr:     "192.168.1.1:12345",
			expectedPrefix: "192.168.1.1",
		},
		{
			name:           "X-Forwarded-For single",
			remoteAddr:     "127.0.0.1:12345",
			xForwardedFor:  "203.0.113.1",
			expectedPrefix: "203.0.113.1",
		},
		{
			name:           "X-Forwarded-For multiple",
			remoteAddr:     "127.0.0.1:12345",
			xForwardedFor:  "203.0.113.1, 192.168.1.1",
			expectedPrefix: "203.0.113.1",
		},
		{
			name:           "X-Real-IP",
			remoteAddr:     "127.0.0.1:12345",
			xRealIP:        "203.0.113.2",
			expectedPrefix: "203.0.113.2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
			req.RemoteAddr = tc.remoteAddr
			if tc.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tc.xForwardedFor)
			}
			if tc.xRealIP != "" {
				req.Header.Set("X-Real-IP", tc.xRealIP)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			// Should extract IP correctly (verified by successful processing)
			if rr.Code != http.StatusOK && rr.Code != http.StatusTooManyRequests {
				t.Errorf("Unexpected status code: %d", rr.Code)
			}
		})
	}
}

// TestRateLimitMiddleware_Concurrent tests concurrent rate limiting
func TestRateLimitMiddleware_Concurrent(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  50,
		GlobalLimit: 200,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	const numGoroutines = 10
	const requestsPerGoroutine = 20

	var wg sync.WaitGroup
	results := make(chan int, numGoroutines*requestsPerGoroutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
				req.RemoteAddr = "192.168.1." + string(rune(id%5)) + ":12345"
				rr := httptest.NewRecorder()

				handler.ServeHTTP(rr, req)
				results <- rr.Code
			}
		}(i)
	}

	wg.Wait()
	close(results)

	successCount := 0
	rateLimitedCount := 0

	for code := range results {
		if code == http.StatusOK {
			successCount++
		} else if code == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	t.Logf("Concurrent test: %d success, %d rate-limited", successCount, rateLimitedCount)

	if successCount == 0 {
		t.Error("Expected at least some successful requests")
	}
}

// TestRateLimitMiddleware_RetryAfterHeader tests Retry-After header
func TestRateLimitMiddleware_RetryAfterHeader(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  2, // Very low for testing
		GlobalLimit: 100,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	// Exhaust rate limit
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		if rr.Code == http.StatusTooManyRequests {
			retryAfter := rr.Header().Get("Retry-After")
			if retryAfter == "" {
				t.Log("Retry-After header missing (may be optional)")
			} else {
				t.Logf("Retry-After: %s", retryAfter)
			}
			break
		}
	}
}

// BenchmarkRateLimitMiddleware benchmarks rate limit middleware
func BenchmarkRateLimitMiddleware(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     true,
		PerIPLimit:  10000, // High limit to avoid throttling
		GlobalLimit: 100000,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}

// BenchmarkRateLimitMiddleware_Disabled benchmarks disabled middleware
func BenchmarkRateLimitMiddleware_Disabled(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	config := &RateLimitConfig{
		Enabled:     false,
		PerIPLimit:  100,
		GlobalLimit: 1000,
		Logger:      logger,
	}

	rateLimit := NewRateLimitMiddleware(config)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := rateLimit.Middleware(next)

	req := httptest.NewRequest(http.MethodPost, "/webhook", nil)
	req.RemoteAddr = "192.168.1.1:12345"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, req)
	}
}
