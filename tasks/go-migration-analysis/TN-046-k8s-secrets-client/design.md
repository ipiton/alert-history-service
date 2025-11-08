# TN-046: Kubernetes Client - Technical Design

## 1. Архитектурное решение

### 1.1 Overall Architecture

Реализуем **Adapter Pattern** для k8s.io/client-go с упрощённым interface для нужд Publishing System.

```
┌─────────────────────────────────────────────────────┐
│         Publishing System Components                │
│  (TargetDiscoveryManager, HealthMonitor)            │
└──────────────────────┬──────────────────────────────┘
                       │
                       │ uses
                       ▼
          ┌────────────────────────┐
          │   K8sClient Interface  │ ◄─── OUR ABSTRACTION
          │  (simplified API)      │
          └────────────┬───────────┘
                       │
                       │ implements
                       ▼
          ┌────────────────────────┐
          │  DefaultK8sClient      │
          │  (adapter)             │
          └────────────┬───────────┘
                       │
                       │ uses
                       ▼
          ┌────────────────────────┐
          │  k8s.io/client-go      │
          │  (kubernetes.Interface)│
          └────────────────────────┘
                       │
                       ▼
          ┌────────────────────────┐
          │   Kubernetes API       │
          │   (REST endpoints)     │
          └────────────────────────┘
```

**Преимущества**:
- Decoupled от сложности client-go
- Easy mocking для tests
- Interface segregation (только нужные методы)
- Future-proof (можно поменять implementation без breaking changes)

### 1.2 Component Diagram

```
┌─────────────────────────────────────────────────────────┐
│              internal/infrastructure/k8s/               │
│                                                         │
│  ┌───────────────┐          ┌─────────────────┐        │
│  │    client.go  │          │   errors.go     │        │
│  │               │          │                 │        │
│  │ + K8sClient   │          │ + K8sError      │        │
│  │   interface   │          │ + ConnectionErr │        │
│  │               │          │ + AuthError     │        │
│  │ + DefaultK8s  │          │ + NotFoundErr   │        │
│  │   Client      │          │ + TimeoutError  │        │
│  │   struct      │          └─────────────────┘        │
│  │               │                                       │
│  │ + NewK8sClient()                                     │
│  │ + ListSecrets()                                      │
│  │ + GetSecret()                                        │
│  │ + Health()                                           │
│  │ + Close()                                            │
│  └───────────────┘                                      │
│                                                         │
│  ┌───────────────────────────────────┐                 │
│  │        client_test.go             │                 │
│  │                                   │                 │
│  │  + TestNewK8sClient()             │                 │
│  │  + TestListSecrets_Success()      │                 │
│  │  + TestListSecrets_Error()        │                 │
│  │  + TestGetSecret_Success()        │                 │
│  │  + TestGetSecret_NotFound()       │                 │
│  │  + TestHealth_Success()           │                 │
│  │  + TestHealth_Error()             │                 │
│  │  + TestRetryLogic()               │                 │
│  │  + TestConcurrentAccess()         │                 │
│  │  + BenchmarkListSecrets()         │                 │
│  └───────────────────────────────────┘                 │
└─────────────────────────────────────────────────────────┘
```

## 2. Interface Design

### 2.1 K8sClient Interface

```go
package k8s

import (
	"context"

	corev1 "k8s.io/api/core/v1"
)

// K8sClient defines the interface for Kubernetes operations needed by Publishing System
type K8sClient interface {
	// ListSecrets returns secrets from namespace matching label selector
	ListSecrets(ctx context.Context, namespace string, labelSelector string) ([]corev1.Secret, error)

	// GetSecret returns a specific secret by name
	GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error)

	// Health checks if K8s API is accessible
	Health(ctx context.Context) error

	// Close cleans up resources
	Close() error
}
```

**Design Rationale**:
- **Context-first**: Все methods принимают context для cancellation/timeout
- **Standard types**: Используем `corev1.Secret` из k8s.io/api (standard)
- **Simple interface**: Только 4 метода (не 50+ от client-go)
- **Error returns**: Idiomatic Go error handling
- **Closeable**: Resource cleanup support

### 2.2 Configuration Structure

```go
// K8sClientConfig holds configuration for K8s client
type K8sClientConfig struct {
	// Timeout for K8s API requests (default 30s)
	Timeout time.Duration

	// MaxRetries for transient errors (default 3)
	MaxRetries int

	// RetryBackoff initial backoff duration (default 100ms)
	RetryBackoff time.Duration

	// MaxRetryBackoff maximum backoff duration (default 5s)
	MaxRetryBackoff time.Duration

	// Logger for structured logging
	Logger *slog.Logger
}

// DefaultK8sClientConfig returns config with sensible defaults
func DefaultK8sClientConfig() *K8sClientConfig {
	return &K8sClientConfig{
		Timeout:         30 * time.Second,
		MaxRetries:      3,
		RetryBackoff:    100 * time.Millisecond,
		MaxRetryBackoff: 5 * time.Second,
		Logger:          slog.Default(),
	}
}
```

## 3. Implementation Design

### 3.1 DefaultK8sClient Structure

```go
// DefaultK8sClient implements K8sClient using k8s.io/client-go
type DefaultK8sClient struct {
	clientset kubernetes.Interface
	config    *K8sClientConfig
	logger    *slog.Logger
	mu        sync.RWMutex // For thread-safe configuration updates
}

// NewK8sClient creates a new K8s client with in-cluster configuration
func NewK8sClient(config *K8sClientConfig) (K8sClient, error) {
	if config == nil {
		config = DefaultK8sClientConfig()
	}

	// Load in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, NewConnectionError("failed to load in-cluster config", err)
	}

	// Apply timeout from config
	k8sConfig.Timeout = config.Timeout

	// Create clientset
	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, NewConnectionError("failed to create K8s clientset", err)
	}

	client := &DefaultK8sClient{
		clientset: clientset,
		config:    config,
		logger:    config.Logger,
	}

	// Verify connection
	if err := client.Health(context.Background()); err != nil {
		return nil, fmt.Errorf("K8s API health check failed: %w", err)
	}

	return client, nil
}
```

**Key Design Points**:
- In-cluster config only (production-first)
- Health check сразу после creation (fail-fast)
- Thread-safe через sync.RWMutex
- Structured logging (slog)

### 3.2 ListSecrets Implementation

```go
func (c *DefaultK8sClient) ListSecrets(ctx context.Context, namespace string, labelSelector string) ([]corev1.Secret, error) {
	c.logger.Debug("Listing K8s secrets",
		"namespace", namespace,
		"label_selector", labelSelector,
	)

	var secrets []corev1.Secret
	err := c.retryWithBackoff(ctx, func() error {
		listOptions := metav1.ListOptions{
			LabelSelector: labelSelector,
			Limit:         1000, // Pagination support
		}

		secretList, err := c.clientset.CoreV1().Secrets(namespace).List(ctx, listOptions)
		if err != nil {
			return err
		}

		secrets = secretList.Items

		// Handle pagination if needed
		if secretList.Continue != "" {
			c.logger.Warn("Secrets list truncated, pagination not implemented",
				"namespace", namespace,
				"continue_token", secretList.Continue,
			)
		}

		return nil
	})

	if err != nil {
		c.logger.Error("Failed to list secrets",
			"namespace", namespace,
			"error", err,
		)
		return nil, wrapK8sError("list secrets", err)
	}

	c.logger.Info("Successfully listed secrets",
		"namespace", namespace,
		"count", len(secrets),
	)

	return secrets, nil
}
```

**Features**:
- Retry logic через `retryWithBackoff()`
- Pagination awareness (logs warning если truncated)
- Structured logging на всех уровнях
- Context cancellation support
- Error wrapping для typed errors

### 3.3 GetSecret Implementation

```go
func (c *DefaultK8sClient) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	c.logger.Debug("Getting K8s secret",
		"namespace", namespace,
		"name", name,
	)

	var secret *corev1.Secret
	err := c.retryWithBackoff(ctx, func() error {
		s, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		secret = s
		return nil
	})

	if err != nil {
		// Check for NotFound error (no retry needed)
		if errors.IsNotFound(err) {
			return nil, NewNotFoundError(fmt.Sprintf("secret %s/%s not found", namespace, name))
		}

		c.logger.Error("Failed to get secret",
			"namespace", namespace,
			"name", name,
			"error", err,
		)
		return nil, wrapK8sError("get secret", err)
	}

	c.logger.Debug("Successfully got secret",
		"namespace", namespace,
		"name", name,
	)

	return secret, nil
}
```

**NotFound Handling**:
- Separate error type для NotFound (no retry)
- Typed error checking (k8serrors.IsNotFound)

### 3.4 Health Check Implementation

```go
func (c *DefaultK8sClient) Health(ctx context.Context) error {
	// Short timeout for health checks
	healthCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Lightweight API call to check connectivity
	// Using Discovery().ServerVersion() is standard health check
	_, err := c.clientset.Discovery().ServerVersion()
	if err != nil {
		c.logger.Warn("K8s health check failed", "error", err)
		return NewConnectionError("K8s API unavailable", err)
	}

	return nil
}
```

**Health Check Strategy**:
- Lightweight call (ServerVersion, ~10ms)
- Short timeout (5s, independent от config.Timeout)
- No retry (health должен быть fast)
- Used for readiness probes

### 3.5 Retry Logic with Exponential Backoff

```go
func (c *DefaultK8sClient) retryWithBackoff(ctx context.Context, operation func() error) error {
	backoff := c.config.RetryBackoff

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		// Check context cancellation before attempt
		select {
		case <-ctx.Done():
			return NewTimeoutError("operation cancelled", ctx.Err())
		default:
		}

		err := operation()
		if err == nil {
			return nil // Success
		}

		// Check if error is retryable
		if !isRetryableError(err) {
			return err // Permanent error, no retry
		}

		// Last attempt - return error
		if attempt == c.config.MaxRetries {
			return err
		}

		// Log retry
		c.logger.Warn("Retrying K8s operation",
			"attempt", attempt+1,
			"max_retries", c.config.MaxRetries,
			"backoff", backoff,
			"error", err,
		)

		// Wait with backoff
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return NewTimeoutError("operation cancelled during backoff", ctx.Err())
		}

		// Exponential backoff
		backoff *= 2
		if backoff > c.config.MaxRetryBackoff {
			backoff = c.config.MaxRetryBackoff
		}
	}

	return fmt.Errorf("operation failed after %d retries", c.config.MaxRetries)
}

func isRetryableError(err error) bool {
	// Retryable: network errors, timeouts, 5xx server errors, rate limiting
	if errors.IsTimeout(err) || errors.IsServerTimeout(err) {
		return true
	}
	if errors.IsInternalError(err) || errors.IsServiceUnavailable(err) {
		return true
	}
	if errors.IsTooManyRequests(err) {
		return true
	}

	// Not retryable: auth errors, not found, invalid input
	if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
		return false
	}
	if errors.IsNotFound(err) || errors.IsInvalid(err) {
		return false
	}

	// Default: retry for unknown errors (conservative)
	return true
}
```

**Retry Strategy**:
- Exponential backoff: 100ms → 200ms → 400ms → 800ms → ...
- Max backoff: 5s (prevents long waits)
- Context-aware (cancellation between retries)
- Typed error checking (k8serrors.Is*)
- Smart retry decision (permanent vs transient)

### 3.6 Close Implementation

```go
func (c *DefaultK8sClient) Close() error {
	c.logger.Info("Closing K8s client")

	// client-go's clientset doesn't have explicit Close()
	// But we can nil out references for GC
	c.mu.Lock()
	defer c.mu.Unlock()

	c.clientset = nil

	c.logger.Info("K8s client closed")
	return nil
}
```

**Note**: client-go doesn't require explicit cleanup, но мы предоставляем Close() для consistency с другими clients (DB, Redis).

## 4. Error Design

### 4.1 Error Types

```go
// errors.go

package k8s

import "fmt"

// K8sError is the base error type for K8s client errors
type K8sError struct {
	Op      string // Operation name (e.g., "list secrets", "get secret")
	Message string // Human-readable message
	Err     error  // Underlying error
}

func (e *K8sError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("k8s %s: %s: %v", e.Op, e.Message, e.Err)
	}
	return fmt.Sprintf("k8s %s: %s", e.Op, e.Message)
}

func (e *K8sError) Unwrap() error {
	return e.Err
}

// Specific error types

type ConnectionError struct {
	*K8sError
}

func NewConnectionError(message string, err error) *ConnectionError {
	return &ConnectionError{
		K8sError: &K8sError{
			Op:      "connection",
			Message: message,
			Err:     err,
		},
	}
}

type AuthError struct {
	*K8sError
}

func NewAuthError(message string, err error) *AuthError {
	return &AuthError{
		K8sError: &K8sError{
			Op:      "authentication",
			Message: message,
			Err:     err,
		},
	}
}

type NotFoundError struct {
	*K8sError
}

func NewNotFoundError(message string) *NotFoundError {
	return &NotFoundError{
		K8sError: &K8sError{
			Op:      "not_found",
			Message: message,
		},
	}
}

type TimeoutError struct {
	*K8sError
}

func NewTimeoutError(message string, err error) *TimeoutError {
	return &TimeoutError{
		K8sError: &K8sError{
			Op:      "timeout",
			Message: message,
			Err:     err,
		},
	}
}

// Helper function to wrap k8s API errors into our types
func wrapK8sError(operation string, err error) error {
	if errors.IsUnauthorized(err) || errors.IsForbidden(err) {
		return NewAuthError("insufficient permissions", err)
	}
	if errors.IsNotFound(err) {
		return NewNotFoundError(operation + " not found")
	}
	if errors.IsTimeout(err) || errors.IsServerTimeout(err) {
		return NewTimeoutError("request timed out", err)
	}

	// Generic K8s error
	return &K8sError{
		Op:      operation,
		Message: "operation failed",
		Err:     err,
	}
}
```

**Error Design Principles**:
- Typed errors для different scenarios
- Error wrapping (supports errors.Is/As)
- Operation context в error message
- No sensitive data в errors (tokens, passwords)

## 5. Testing Strategy

### 5.1 Unit Tests Structure

```go
// client_test.go

package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListSecrets_Success(t *testing.T) {
	// Create fake clientset with test secrets
	fakeClientset := fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret-1",
				Namespace: "default",
				Labels: map[string]string{
					"publishing-target": "true",
				},
			},
			Data: map[string][]byte{
				"url": []byte("https://example.com"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret-2",
				Namespace: "default",
				Labels: map[string]string{
					"publishing-target": "true",
				},
			},
		},
	)

	client := &DefaultK8sClient{
		clientset: fakeClientset,
		config:    DefaultK8sClientConfig(),
		logger:    slog.Default(),
	}

	secrets, err := client.ListSecrets(context.Background(), "default", "publishing-target=true")

	require.NoError(t, err)
	assert.Len(t, secrets, 2)
	assert.Equal(t, "test-secret-1", secrets[0].Name)
}

func TestGetSecret_NotFound(t *testing.T) {
	fakeClientset := fake.NewSimpleClientset()

	client := &DefaultK8sClient{
		clientset: fakeClientset,
		config:    DefaultK8sClientConfig(),
		logger:    slog.Default(),
	}

	secret, err := client.GetSecret(context.Background(), "default", "nonexistent")

	assert.Nil(t, secret)
	assert.Error(t, err)

	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestConcurrentAccess(t *testing.T) {
	// Test for race conditions
	// Run with: go test -race

	fakeClientset := fake.NewSimpleClientset(/* test secrets */)
	client := &DefaultK8sClient{
		clientset: fakeClientset,
		config:    DefaultK8sClientConfig(),
		logger:    slog.Default(),
	}

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, _ = client.ListSecrets(context.Background(), "default", "")
			done <- true
		}()
	}

	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}
```

### 5.2 Test Coverage Goals

- **Happy Paths**: 40%
  - ListSecrets с results
  - GetSecret success
  - Health check success

- **Error Handling**: 30%
  - Connection errors
  - Auth errors (403 Forbidden)
  - Not found (404)
  - Timeout errors
  - Retry exhaustion

- **Edge Cases**: 20%
  - Empty namespace
  - Empty label selector
  - Context cancellation
  - Concurrent operations

- **Benchmarks**: 10%
  - ListSecrets performance
  - GetSecret performance

### 5.3 Benchmarks

```go
func BenchmarkListSecrets(b *testing.B) {
	fakeClientset := fake.NewSimpleClientset(
		// Create 100 test secrets
	)

	client := &DefaultK8sClient{
		clientset: fakeClientset,
		config:    DefaultK8sClientConfig(),
		logger:    slog.Default(),
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListSecrets(ctx, "default", "publishing-target=true")
	}
}
```

## 6. Performance Targets

| Operation | Target (p95) | Expected | Buffer |
|-----------|-------------|----------|--------|
| ListSecrets (10 secrets) | < 500ms | ~300ms | 1.7x |
| GetSecret | < 200ms | ~100ms | 2x |
| Health check | < 100ms | ~50ms | 2x |
| NewK8sClient | < 1s | ~500ms | 2x |

## 7. Integration Points

### 7.1 Usage Example (для TN-047)

```go
// In TN-047: Target Discovery Manager

import (
	"github.com/vitaliisemenov/alert-history/internal/infrastructure/k8s"
)

type TargetDiscoveryManager struct {
	k8sClient k8s.K8sClient
	namespace string
}

func (m *TargetDiscoveryManager) DiscoverTargets(ctx context.Context) error {
	// List secrets with label publishing-target=true
	secrets, err := m.k8sClient.ListSecrets(ctx, m.namespace, "publishing-target=true")
	if err != nil {
		return fmt.Errorf("failed to discover targets: %w", err)
	}

	// Parse secrets into PublishingTarget models
	for _, secret := range secrets {
		target, err := parseSecretToTarget(&secret)
		if err != nil {
			m.logger.Warn("Failed to parse secret", "name", secret.Name, "error", err)
			continue
		}

		m.targets[target.Name] = target
	}

	return nil
}
```

### 7.2 Configuration Integration

Add to `internal/config/config.go`:

```go
type K8sConfig struct {
	Enabled         bool          `mapstructure:"enabled" default:"true"`
	Timeout         time.Duration `mapstructure:"timeout" default:"30s"`
	MaxRetries      int           `mapstructure:"max_retries" default:"3"`
	RetryBackoff    time.Duration `mapstructure:"retry_backoff" default:"100ms"`
	MaxRetryBackoff time.Duration `mapstructure:"max_retry_backoff" default:"5s"`
}
```

## 8. Security Considerations

### 8.1 RBAC Requirements (будет в TN-050)

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: alert-history-service
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: alert-history-secrets-reader
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get", "list"]
  # Optionally restrict to specific label selectors
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: alert-history-secrets-reader-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: alert-history-secrets-reader
subjects:
- kind: ServiceAccount
  name: alert-history-service
```

### 8.2 Token Rotation

- ServiceAccount tokens automatically rotated by K8s
- client-go automatically reloads token от file
- No manual handling needed

### 8.3 TLS Validation

- Always validate K8s API server certificates
- Use CA cert от ServiceAccount mount
- No InsecureSkipVerify

## 9. Observability

### 9.1 Logging Events

- **DEBUG**: All API calls (ListSecrets, GetSecret) с parameters
- **INFO**: Successful operations с results count
- **WARN**: Retries, pagination truncation
- **ERROR**: Failed operations после всех retries

### 9.2 Log Format Example

```json
{
  "time": "2025-11-07T10:00:00Z",
  "level": "INFO",
  "msg": "Successfully listed secrets",
  "namespace": "default",
  "label_selector": "publishing-target=true",
  "count": 5,
  "duration_ms": 250
}
```

### 9.3 Metrics (Future - TN-057)

Metrics будут добавлены в TN-057, но design готов:

- `alert_history_k8s_requests_total{operation, status}`
- `alert_history_k8s_request_duration_seconds{operation}`
- `alert_history_k8s_retries_total{operation}`

## 10. Migration Path & Rollback

- **No Migration Needed**: New component
- **Rollback**: Remove K8s client, fallback to static configuration (if implemented)
- **Backward Compatibility**: N/A (new feature)

## 11. Future Enhancements (Out of Scope)

- Watch API для real-time secret changes (instead of polling)
- Multi-cluster support
- Custom Resource Definitions (CRDs) для PublishingTarget
- Admission webhooks для secret validation

---

**Document Version**: 1.0
**Created**: 2025-11-07
**Author**: AI Assistant (Phase 5 Implementation)
**Status**: APPROVED FOR IMPLEMENTATION
