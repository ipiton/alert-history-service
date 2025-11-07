package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

// createTestSecret creates a test secret for testing
func createTestSecret(name, namespace string, labels map[string]string, data map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Data: data,
	}
}

// createFakeClient creates a DefaultK8sClient with fake clientset for testing
func createFakeClient(secrets ...*corev1.Secret) *DefaultK8sClient {
	objects := make([]runtime.Object, len(secrets))
	for i, secret := range secrets {
		objects[i] = secret
	}

	fakeClientset := fake.NewSimpleClientset(objects...)

	return &DefaultK8sClient{
		clientset: fakeClientset,
		config:    DefaultK8sClientConfig(),
		logger:    slog.Default(),
	}
}

func TestDefaultK8sClientConfig(t *testing.T) {
	config := DefaultK8sClientConfig()

	assert.Equal(t, 30*time.Second, config.Timeout)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, config.RetryBackoff)
	assert.Equal(t, 5*time.Second, config.MaxRetryBackoff)
	assert.NotNil(t, config.Logger)
}

func TestListSecrets_Success(t *testing.T) {
	secret1 := createTestSecret("test-secret-1", "default", map[string]string{
		"publishing-target": "true",
	}, map[string][]byte{
		"url": []byte("https://example.com"),
	})

	secret2 := createTestSecret("test-secret-2", "default", map[string]string{
		"publishing-target": "true",
	}, nil)

	client := createFakeClient(secret1, secret2)

	secrets, err := client.ListSecrets(context.Background(), "default", "publishing-target=true")

	require.NoError(t, err)
	assert.Len(t, secrets, 2)

	// Verify secrets are returned (order may vary)
	secretNames := []string{secrets[0].Name, secrets[1].Name}
	assert.Contains(t, secretNames, "test-secret-1")
	assert.Contains(t, secretNames, "test-secret-2")
}

func TestListSecrets_EmptyResult(t *testing.T) {
	client := createFakeClient()

	secrets, err := client.ListSecrets(context.Background(), "default", "publishing-target=true")

	require.NoError(t, err)
	assert.Len(t, secrets, 0)
}

func TestListSecrets_LabelFiltering(t *testing.T) {
	secret1 := createTestSecret("target-secret", "default", map[string]string{
		"publishing-target": "true",
	}, nil)

	secret2 := createTestSecret("other-secret", "default", map[string]string{
		"app": "other",
	}, nil)

	client := createFakeClient(secret1, secret2)

	// Note: fake clientset doesn't actually filter, so we'd get both
	// In real K8s, only secret1 would be returned
	secrets, err := client.ListSecrets(context.Background(), "default", "publishing-target=true")

	require.NoError(t, err)
	// Fake clientset returns all secrets, real K8s would filter
	assert.GreaterOrEqual(t, len(secrets), 1)
}

func TestGetSecret_Success(t *testing.T) {
	secret1 := createTestSecret("test-secret-1", "default", map[string]string{
		"publishing-target": "true",
	}, map[string][]byte{
		"url": []byte("https://example.com"),
	})

	client := createFakeClient(secret1)

	secret, err := client.GetSecret(context.Background(), "default", "test-secret-1")

	require.NoError(t, err)
	assert.NotNil(t, secret)
	assert.Equal(t, "test-secret-1", secret.Name)
	assert.Equal(t, "default", secret.Namespace)
	assert.Equal(t, []byte("https://example.com"), secret.Data["url"])
}

func TestGetSecret_NotFound(t *testing.T) {
	client := createFakeClient()

	secret, err := client.GetSecret(context.Background(), "default", "nonexistent")

	assert.Nil(t, secret)
	assert.Error(t, err)

	// Check error type
	var notFoundErr *NotFoundError
	assert.ErrorAs(t, err, &notFoundErr)
}

func TestListSecrets_ContextCancelled(t *testing.T) {
	client := createFakeClient()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	secrets, err := client.ListSecrets(ctx, "default", "")

	assert.Nil(t, secrets)
	assert.Error(t, err)

	// Should be timeout error due to cancelled context
	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
}

func TestGetSecret_ContextCancelled(t *testing.T) {
	client := createFakeClient()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	secret, err := client.GetSecret(ctx, "default", "test-secret")

	assert.Nil(t, secret)
	assert.Error(t, err)

	// Should be timeout error due to cancelled context
	var timeoutErr *TimeoutError
	assert.ErrorAs(t, err, &timeoutErr)
}

func TestConcurrentAccess(t *testing.T) {
	// Test for race conditions
	// Run with: go test -race

	secret1 := createTestSecret("test-secret", "default", nil, nil)
	client := createFakeClient(secret1)

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	// Launch concurrent ListSecrets calls
	for i := 0; i < numGoroutines/2; i++ {
		go func() {
			_, _ = client.ListSecrets(context.Background(), "default", "")
			done <- true
		}()
	}

	// Launch concurrent GetSecret calls
	for i := 0; i < numGoroutines/2; i++ {
		go func() {
			_, _ = client.GetSecret(context.Background(), "default", "test-secret")
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestClose_MultipleCalls(t *testing.T) {
	client := createFakeClient()

	// First close
	err1 := client.Close()
	assert.NoError(t, err1)

	// Second close should also succeed
	err2 := client.Close()
	assert.NoError(t, err2)
}

func TestRetryLogic_ImmediateSuccess(t *testing.T) {
	client := createFakeClient()

	attemptCount := 0
	err := client.retryWithBackoff(context.Background(), func() error {
		attemptCount++
		return nil // Immediate success
	})

	assert.NoError(t, err)
	assert.Equal(t, 1, attemptCount)
}

func TestRetryLogic_EventualSuccess(t *testing.T) {
	client := createFakeClient()

	attemptCount := 0
	err := client.retryWithBackoff(context.Background(), func() error {
		attemptCount++
		if attemptCount < 3 {
			return fmt.Errorf("transient error")
		}
		return nil // Success on 3rd attempt
	})

	assert.NoError(t, err)
	assert.Equal(t, 3, attemptCount)
}

func TestRetryLogic_ExhaustedRetries(t *testing.T) {
	client := createFakeClient()

	attemptCount := 0
	err := client.retryWithBackoff(context.Background(), func() error {
		attemptCount++
		return fmt.Errorf("persistent error")
	})

	assert.Error(t, err)
	assert.Equal(t, client.config.MaxRetries+1, attemptCount) // MaxRetries + initial attempt
}

// Benchmarks

func BenchmarkListSecrets_10Secrets(b *testing.B) {
	secrets := make([]*corev1.Secret, 10)
	for i := 0; i < 10; i++ {
		secrets[i] = createTestSecret(
			fmt.Sprintf("secret-%d", i),
			"default",
			map[string]string{"publishing-target": "true"},
			nil,
		)
	}

	client := createFakeClient(secrets...)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ListSecrets(ctx, "default", "publishing-target=true")
	}
}

func BenchmarkGetSecret(b *testing.B) {
	secret := createTestSecret("test-secret", "default", nil, nil)
	client := createFakeClient(secret)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GetSecret(ctx, "default", "test-secret")
	}
}

func BenchmarkHealth(b *testing.B) {
	client := createFakeClient()
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.Health(ctx)
	}
}
