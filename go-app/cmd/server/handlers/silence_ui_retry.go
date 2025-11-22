// Package handlers provides HTTP handlers for the Alert History Service.
package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// RetryConfig defines retry configuration for API calls.
// Phase 12: Error Handling enhancement.
type RetryConfig struct {
	MaxAttempts      int
	InitialBackoff   time.Duration
	MaxBackoff       time.Duration
	BackoffMultiplier float64
}

// DefaultRetryConfig returns default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:         5 * time.Second,
		BackoffMultiplier: 2.0,
	}
}

// RetryableError indicates if an error is retryable.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network/timeout errors
	errStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"network",
		"temporary",
		"503",
		"502",
		"504",
	}

	for _, pattern := range retryablePatterns {
		if containsError(errStr, pattern) {
			return true
		}
	}

	return false
}

// containsError checks if an error message contains a substring (case-insensitive).
func containsError(errStr, substr string) bool {
	return len(errStr) >= len(substr) && strings.Contains(strings.ToLower(errStr), strings.ToLower(substr))
}

// RetryableFunc is a function that can be retried.
type RetryableFunc func() error

// RetryWithBackoff executes a function with exponential backoff retry logic.
func (h *SilenceUIHandler) RetryWithBackoff(
	ctx context.Context,
	fn RetryableFunc,
	config RetryConfig,
) error {
	var lastErr error
	backoff := config.InitialBackoff

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute function
		err := fn()
		if err == nil {
			// Success
			if attempt > 0 {
				h.logger.Debug("Retry succeeded",
					"attempt", attempt+1,
					"max_attempts", config.MaxAttempts,
				)
			}
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if !isRetryableError(err) {
			h.logger.Debug("Error is not retryable, aborting",
				"attempt", attempt+1,
				"error", err,
			)
			return err
		}

		// Don't retry on last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Wait before retry
		h.logger.Debug("Retrying after backoff",
			"attempt", attempt+1,
			"backoff_ms", backoff.Milliseconds(),
			"error", err,
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}

		// Increase backoff for next attempt
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	return fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}

// RetryableHTTPRequest executes an HTTP request with retry logic.
func (h *SilenceUIHandler) RetryableHTTPRequest(
	ctx context.Context,
	req *http.Request,
	client *http.Client,
	config RetryConfig,
) (*http.Response, error) {
	var lastResp *http.Response
	var lastErr error

	backoff := config.InitialBackoff

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Execute request
		resp, err := client.Do(req.WithContext(ctx))
		if err == nil && resp.StatusCode < 500 {
			// Success or client error (don't retry 4xx)
			if attempt > 0 {
				h.logger.Debug("HTTP request retry succeeded",
					"attempt", attempt+1,
					"status_code", resp.StatusCode,
				)
			}
			return resp, nil
		}

		if resp != nil {
			lastResp = resp
		}
		lastErr = err

		// Check if error is retryable
		if resp != nil && resp.StatusCode < 500 {
			// Client error, don't retry
			return resp, nil
		}

		// Don't retry on last attempt
		if attempt == config.MaxAttempts-1 {
			break
		}

		// Wait before retry
		h.logger.Debug("Retrying HTTP request after backoff",
			"attempt", attempt+1,
			"backoff_ms", backoff.Milliseconds(),
			"error", lastErr,
			"status_code", func() int {
				if lastResp != nil {
					return lastResp.StatusCode
				}
				return 0
			}(),
		)

		// Close response body if exists
		if lastResp != nil && lastResp.Body != nil {
			lastResp.Body.Close()
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}

		// Increase backoff
		backoff = time.Duration(float64(backoff) * config.BackoffMultiplier)
		if backoff > config.MaxBackoff {
			backoff = config.MaxBackoff
		}
	}

	if lastResp != nil {
		return lastResp, lastErr
	}
	return nil, fmt.Errorf("max retry attempts (%d) exceeded: %w", config.MaxAttempts, lastErr)
}
