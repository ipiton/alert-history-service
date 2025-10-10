// Package resilience provides reliability patterns for distributed systems.
//
// This package implements retry logic, circuit breakers, bulkheads, and other
// resilience patterns to handle transient failures and improve system reliability.
package resilience

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"
)

// RetryPolicy defines configuration for retry behavior with exponential backoff.
//
// Example usage:
//
//	policy := &RetryPolicy{
//	    MaxRetries:  3,
//	    BaseDelay:   100 * time.Millisecond,
//	    MaxDelay:    5 * time.Second,
//	    Multiplier:  2.0,
//	    Jitter:      true,
//	}
//	err := WithRetry(ctx, policy, func() error {
//	    return someOperation()
//	})
type RetryPolicy struct {
	// MaxRetries is the maximum number of retry attempts (0 = no retries)
	MaxRetries int

	// BaseDelay is the initial delay before the first retry
	BaseDelay time.Duration

	// MaxDelay is the maximum delay between retries
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases (exponential backoff)
	// Typical values: 1.5 to 3.0 (2.0 is common)
	Multiplier float64

	// Jitter adds randomness to delay to prevent thundering herd
	// If true, adds up to 10% random jitter to each delay
	Jitter bool

	// ErrorChecker determines which errors should trigger a retry
	// If nil, uses default checker (all non-nil errors are retryable)
	ErrorChecker RetryableErrorChecker

	// Logger for retry events (optional)
	// If nil, uses slog.Default()
	Logger *slog.Logger
}

// RetryableErrorChecker determines if an error should trigger a retry attempt.
//
// Implementations should return true for transient errors (network timeouts,
// temporary service unavailability) and false for permanent errors (invalid input,
// authorization failures).
type RetryableErrorChecker interface {
	IsRetryable(err error) bool
}

// DefaultRetryPolicy returns a sensible default retry policy.
//
// Configuration:
//   - MaxRetries: 3
//   - BaseDelay: 100ms
//   - MaxDelay: 5s
//   - Multiplier: 2.0 (exponential backoff)
//   - Jitter: true (10% randomness)
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries: 3,
		BaseDelay:  100 * time.Millisecond,
		MaxDelay:   5 * time.Second,
		Multiplier: 2.0,
		Jitter:     true,
	}
}

// WithRetry executes the given operation with retry logic according to the policy.
//
// The operation is retried on failure according to the retry policy. If the operation
// succeeds, returns nil. If all retry attempts are exhausted, returns the last error.
//
// Context cancellation is respected: if ctx is cancelled during a retry delay,
// WithRetry returns immediately with ctx.Err().
//
// Example:
//
//	policy := DefaultRetryPolicy()
//	err := WithRetry(ctx, policy, func() error {
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return err
//	    }
//	    defer resp.Body.Close()
//	    return processResponse(resp)
//	})
//	if err != nil {
//	    log.Fatal("Operation failed after retries:", err)
//	}
func WithRetry(ctx context.Context, policy *RetryPolicy, operation func() error) error {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	logger := policy.Logger
	if logger == nil {
		logger = slog.Default()
	}

	var lastErr error
	delay := policy.BaseDelay

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Execute the operation
		err := operation()
		if err == nil {
			// Success!
			if attempt > 0 {
				logger.Info("Operation succeeded after retry",
					"attempt", attempt+1,
					"total_attempts", attempt+1,
				)
			}
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if !shouldRetry(err, policy.ErrorChecker) {
			logger.Debug("Error is non-retryable, stopping retry loop",
				"error", err,
				"attempt", attempt+1,
			)
			return lastErr
		}

		// Check if we have more retries left
		if attempt >= policy.MaxRetries {
			logger.Error("Operation failed after all retries",
				"max_retries", policy.MaxRetries,
				"total_attempts", attempt+1,
				"error", lastErr,
			)
			break
		}

		// Log retry attempt
		logger.Warn("Operation failed, retrying",
			"attempt", attempt+1,
			"max_retries", policy.MaxRetries,
			"delay", delay,
			"error", err,
		)

		// Wait before next retry (respecting context cancellation)
		if !waitWithContext(ctx, delay) {
			logger.Debug("Context cancelled during retry delay",
				"attempt", attempt+1,
			)
			return ctx.Err()
		}

		// Calculate next delay with exponential backoff
		delay = calculateNextDelay(delay, policy)
	}

	return fmt.Errorf("operation failed after %d attempts: %w", policy.MaxRetries+1, lastErr)
}

// WithRetryFunc is like WithRetry but for operations that return a result.
//
// If the operation succeeds, returns (result, nil). If all retry attempts are
// exhausted, returns (last result, last error).
//
// Example:
//
//	policy := DefaultRetryPolicy()
//	data, err := WithRetryFunc(ctx, policy, func() ([]byte, error) {
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return nil, err
//	    }
//	    defer resp.Body.Close()
//	    return io.ReadAll(resp.Body)
//	})
//	if err != nil {
//	    log.Fatal("Failed to fetch data:", err)
//	}
func WithRetryFunc[T any](ctx context.Context, policy *RetryPolicy, operation func() (T, error)) (T, error) {
	if policy == nil {
		policy = DefaultRetryPolicy()
	}

	logger := policy.Logger
	if logger == nil {
		logger = slog.Default()
	}

	var lastResult T
	var lastErr error
	delay := policy.BaseDelay

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Execute the operation
		result, err := operation()
		if err == nil {
			// Success!
			if attempt > 0 {
				logger.Info("Operation succeeded after retry",
					"attempt", attempt+1,
					"total_attempts", attempt+1,
				)
			}
			return result, nil
		}

		lastResult = result
		lastErr = err

		// Check if we should retry this error
		if !shouldRetry(err, policy.ErrorChecker) {
			logger.Debug("Error is non-retryable, stopping retry loop",
				"error", err,
				"attempt", attempt+1,
			)
			return lastResult, lastErr
		}

		// Check if we have more retries left
		if attempt >= policy.MaxRetries {
			logger.Error("Operation failed after all retries",
				"max_retries", policy.MaxRetries,
				"total_attempts", attempt+1,
				"error", lastErr,
			)
			break
		}

		// Log retry attempt
		logger.Warn("Operation failed, retrying",
			"attempt", attempt+1,
			"max_retries", policy.MaxRetries,
			"delay", delay,
			"error", err,
		)

		// Wait before next retry (respecting context cancellation)
		if !waitWithContext(ctx, delay) {
			logger.Debug("Context cancelled during retry delay",
				"attempt", attempt+1,
			)
			var zero T
			return zero, ctx.Err()
		}

		// Calculate next delay with exponential backoff
		delay = calculateNextDelay(delay, policy)
	}

	return lastResult, fmt.Errorf("operation failed after %d attempts: %w", policy.MaxRetries+1, lastErr)
}

// shouldRetry determines if an error should trigger a retry attempt.
func shouldRetry(err error, checker RetryableErrorChecker) bool {
	if err == nil {
		return false
	}

	if checker != nil {
		return checker.IsRetryable(err)
	}

	// Default: all non-nil errors are retryable
	return true
}

// waitWithContext waits for the specified duration, respecting context cancellation.
// Returns true if the wait completed normally, false if context was cancelled.
func waitWithContext(ctx context.Context, delay time.Duration) bool {
	select {
	case <-time.After(delay):
		return true
	case <-ctx.Done():
		return false
	}
}

// calculateNextDelay calculates the next retry delay using exponential backoff.
// Applies jitter if enabled in the policy.
func calculateNextDelay(currentDelay time.Duration, policy *RetryPolicy) time.Duration {
	// Exponential backoff
	nextDelay := time.Duration(float64(currentDelay) * policy.Multiplier)

	// Cap at max delay
	if nextDelay > policy.MaxDelay {
		nextDelay = policy.MaxDelay
	}

	// Apply jitter if enabled (adds up to 10% randomness)
	if policy.Jitter {
		jitterFactor := 0.1
		jitterAmount := time.Duration(float64(nextDelay) * jitterFactor * rand.Float64())
		nextDelay += jitterAmount
	}

	return nextDelay
}
