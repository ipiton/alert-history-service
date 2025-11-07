package publishing

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// PublishingJob represents a single publishing task
type PublishingJob struct {
	EnrichedAlert *core.EnrichedAlert
	Target        *core.PublishingTarget
	RetryCount    int
	SubmittedAt   time.Time
}

// PublishingQueue manages async publishing with worker pool and retry logic
type PublishingQueue struct {
	jobs          chan *PublishingJob
	factory       *PublisherFactory
	maxRetries    int
	retryInterval time.Duration
	workerCount   int
	logger        *slog.Logger
	metrics       *PublishingMetrics
	wg            sync.WaitGroup
	ctx           context.Context
	cancel        context.CancelFunc
	circuitBreakers map[string]*CircuitBreaker
	mu            sync.RWMutex
}

// PublishingQueueConfig holds configuration for publishing queue
type PublishingQueueConfig struct {
	WorkerCount    int
	QueueSize      int
	MaxRetries     int
	RetryInterval  time.Duration
	CircuitTimeout time.Duration
}

// DefaultPublishingQueueConfig returns default configuration
func DefaultPublishingQueueConfig() PublishingQueueConfig {
	return PublishingQueueConfig{
		WorkerCount:    10,
		QueueSize:      1000,
		MaxRetries:     3,
		RetryInterval:  2 * time.Second,
		CircuitTimeout: 30 * time.Second,
	}
}

// NewPublishingQueue creates a new publishing queue
func NewPublishingQueue(factory *PublisherFactory, config PublishingQueueConfig, metrics *PublishingMetrics, logger *slog.Logger) *PublishingQueue {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = NewPublishingMetrics()
	}

	ctx, cancel := context.WithCancel(context.Background())

	queue := &PublishingQueue{
		jobs:            make(chan *PublishingJob, config.QueueSize),
		factory:         factory,
		maxRetries:      config.MaxRetries,
		retryInterval:   config.RetryInterval,
		workerCount:     config.WorkerCount,
		logger:          logger,
		metrics:         metrics,
		ctx:             ctx,
		cancel:          cancel,
		circuitBreakers: make(map[string]*CircuitBreaker),
	}

	// Initialize queue capacity metric
	queue.metrics.UpdateQueueMetrics(0, config.QueueSize)

	return queue
}

// Start starts the worker pool
func (q *PublishingQueue) Start() {
	q.logger.Info("Starting publishing queue", "workers", q.workerCount)

	for i := 0; i < q.workerCount; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// Stop gracefully stops the publishing queue
func (q *PublishingQueue) Stop(timeout time.Duration) error {
	q.logger.Info("Stopping publishing queue", "timeout", timeout)

	// Close job channel to signal workers
	close(q.jobs)

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		q.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		q.logger.Info("Publishing queue stopped gracefully")
		return nil
	case <-time.After(timeout):
		q.cancel() // Force cancel remaining jobs
		return fmt.Errorf("publishing queue stop timeout after %v", timeout)
	}
}

// Submit submits a job to the publishing queue
func (q *PublishingQueue) Submit(enrichedAlert *core.EnrichedAlert, target *core.PublishingTarget) error {
	job := &PublishingJob{
		EnrichedAlert: enrichedAlert,
		Target:        target,
		RetryCount:    0,
		SubmittedAt:   time.Now(),
	}

	select {
	case q.jobs <- job:
		q.metrics.RecordQueueSubmission(true)
		q.metrics.UpdateQueueMetrics(len(q.jobs), cap(q.jobs))
		return nil
	case <-q.ctx.Done():
		q.metrics.RecordQueueSubmission(false)
		return fmt.Errorf("publishing queue is shutting down")
	default:
		q.metrics.RecordQueueSubmission(false)
		return fmt.Errorf("publishing queue is full")
	}
}

// worker processes jobs from the queue
func (q *PublishingQueue) worker(id int) {
	defer q.wg.Done()

	q.logger.Debug("Worker started", "worker_id", id)

	for job := range q.jobs {
		q.processJob(job)
	}

	q.logger.Debug("Worker stopped", "worker_id", id)
}

// processJob processes a single publishing job with retry logic
func (q *PublishingQueue) processJob(job *PublishingJob) {
	// Check circuit breaker
	cb := q.getCircuitBreaker(job.Target.Name)
	if !cb.CanAttempt() {
		q.logger.Warn("Circuit breaker open, skipping publish",
			"target", job.Target.Name,
			"state", cb.State(),
		)
		return
	}

	// Create publisher
	publisher, err := q.factory.CreatePublisher(job.Target.Type)
	if err != nil {
		q.logger.Error("Failed to create publisher",
			"target", job.Target.Name,
			"type", job.Target.Type,
			"error", err,
		)
		cb.RecordFailure()
		return
	}

	// Attempt publish with retry
	startTime := time.Now()
	err = q.retryPublish(publisher, job)
	duration := time.Since(startTime).Seconds()

	if err != nil {
		q.logger.Error("Failed to publish after retries",
			"target", job.Target.Name,
			"fingerprint", job.EnrichedAlert.Alert.Fingerprint,
			"error", err,
		)
		cb.RecordFailure()
		q.metrics.RecordPublishError(job.Target.Name, job.Target.Type, "retry_exhausted")
	} else {
		q.logger.Info("Alert published successfully",
			"target", job.Target.Name,
			"fingerprint", job.EnrichedAlert.Alert.Fingerprint,
			"queue_time", time.Since(job.SubmittedAt),
		)
		cb.RecordSuccess()
		q.metrics.RecordPublishSuccess(job.Target.Name, job.Target.Type, duration)
	}

	// Update queue size metric
	q.metrics.UpdateQueueMetrics(len(q.jobs), cap(q.jobs))
}

// getCircuitBreaker gets or creates circuit breaker for target
func (q *PublishingQueue) getCircuitBreaker(targetName string) *CircuitBreaker {
	q.mu.RLock()
	cb, exists := q.circuitBreakers[targetName]
	q.mu.RUnlock()

	if exists {
		return cb
	}

	// Create new circuit breaker
	q.mu.Lock()
	defer q.mu.Unlock()

	// Double-check after acquiring write lock
	if cb, exists := q.circuitBreakers[targetName]; exists {
		return cb
	}

	cb = NewCircuitBreakerWithMetrics(
		CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 2,
			Timeout:          30 * time.Second,
		},
		targetName,
		q.metrics,
	)

	q.circuitBreakers[targetName] = cb
	q.logger.Debug("Created circuit breaker", "target", targetName)

	return cb
}

// GetQueueSize returns current queue size
func (q *PublishingQueue) GetQueueSize() int {
	return len(q.jobs)
}

// GetQueueCapacity returns queue capacity
func (q *PublishingQueue) GetQueueCapacity() int {
	return cap(q.jobs)
}

// retryPublish attempts to publish with exponential backoff retry
func (q *PublishingQueue) retryPublish(publisher AlertPublisher, job *PublishingJob) error {
	var lastErr error

	for attempt := 0; attempt <= q.maxRetries; attempt++ {
		// Try publish
		err := publisher.Publish(q.ctx, job.EnrichedAlert, job.Target)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't sleep after last attempt
		if attempt < q.maxRetries {
			// Exponential backoff: interval * 2^attempt
			backoff := time.Duration(math.Pow(2, float64(attempt))) * q.retryInterval
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			q.logger.Debug("Retrying publish",
				"attempt", attempt+1,
				"max_retries", q.maxRetries,
				"backoff", backoff,
			)

			select {
			case <-time.After(backoff):
				// Continue to next attempt
			case <-q.ctx.Done():
				return q.ctx.Err()
			}
		}
	}

	return fmt.Errorf("failed after %d retries: %w", q.maxRetries, lastErr)
}
