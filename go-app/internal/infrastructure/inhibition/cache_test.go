package inhibition

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vitaliisemenov/alert-history/internal/core"
)

// --- Mock Redis Cache ---

type mockRedisCache struct {
	data  map[string]string
	fail  bool
}

func (m *mockRedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	if m.fail {
		return fmt.Errorf("redis unavailable")
	}
	if m.data == nil {
		return fmt.Errorf("not found")
	}
	val, exists := m.data[key]
	if !exists {
		return fmt.Errorf("not found")
	}
	// Simplified deserialization
	if strPtr, ok := dest.(*string); ok {
		*strPtr = val
	}
	return nil
}

func (m *mockRedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if m.fail {
		return fmt.Errorf("redis unavailable")
	}
	if m.data == nil {
		m.data = make(map[string]string)
	}
	// Simplified serialization
	if str, ok := value.(string); ok {
		m.data[key] = str
	}
	return nil
}

func (m *mockRedisCache) Delete(ctx context.Context, key string) error {
	if m.fail {
		return fmt.Errorf("redis unavailable")
	}
	if m.data != nil {
		delete(m.data, key)
	}
	return nil
}

func (m *mockRedisCache) Exists(ctx context.Context, key string) (bool, error) {
	if m.fail {
		return false, fmt.Errorf("redis unavailable")
	}
	if m.data == nil {
		return false, nil
	}
	_, exists := m.data[key]
	return exists, nil
}

func (m *mockRedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if m.fail {
		return 0, fmt.Errorf("redis unavailable")
	}
	return 5 * time.Minute, nil
}

func (m *mockRedisCache) Expire(ctx context.Context, key string, ttl time.Duration) error {
	return nil // No-op for mock
}

func (m *mockRedisCache) HealthCheck(ctx context.Context) error {
	if m.fail {
		return fmt.Errorf("redis unavailable")
	}
	return nil
}

func (m *mockRedisCache) Ping(ctx context.Context) error {
	if m.fail {
		return fmt.Errorf("redis unavailable")
	}
	return nil
}

func (m *mockRedisCache) Flush(ctx context.Context) error {
	if m.fail {
		return fmt.Errorf("redis unavailable")
	}
	m.data = make(map[string]string)
	return nil
}

// --- Test Helpers ---

func createCacheTestAlert(name, fingerprint string) *core.Alert {
	now := time.Now()
	return &core.Alert{
		AlertName:   name,
		Fingerprint: fingerprint,
		Status:      "firing",
		StartsAt:    now,
		EndsAt:      nil, // Not ended
		Labels: map[string]string{
			"alertname": name,
		},
	}
}

// --- Happy Path Tests ---

func TestTwoTierAlertCache_AddAndGet(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil) // L1 only
	defer cache.Stop()

	alert := createCacheTestAlert("TestAlert", "fp-test-1")

	// Add alert
	err := cache.AddFiringAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("AddFiringAlert() error = %v", err)
	}

	// Get alerts
	alerts, err := cache.GetFiringAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetFiringAlerts() error = %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert, got %d", len(alerts))
	}

	if alerts[0].Fingerprint != "fp-test-1" {
		t.Errorf("Expected fingerprint fp-test-1, got %s", alerts[0].Fingerprint)
	}
}

func TestTwoTierAlertCache_AddMultiple(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	// Add 10 alerts
	for i := 0; i < 10; i++ {
		alert := createCacheTestAlert("Alert", fmt.Sprintf("fp-%d", i))
		_ = cache.AddFiringAlert(context.Background(), alert)
	}

	alerts, _ := cache.GetFiringAlerts(context.Background())

	if len(alerts) != 10 {
		t.Errorf("Expected 10 alerts, got %d", len(alerts))
	}
}

func TestTwoTierAlertCache_RemoveAlert(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	alert := createCacheTestAlert("TestAlert", "fp-test")
	_ = cache.AddFiringAlert(context.Background(), alert)

	// Remove alert
	err := cache.RemoveAlert(context.Background(), "fp-test")
	if err != nil {
		t.Fatalf("RemoveAlert() error = %v", err)
	}

	// Verify removal
	alerts, _ := cache.GetFiringAlerts(context.Background())
	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts after removal, got %d", len(alerts))
	}
}

func TestTwoTierAlertCache_WithRedis(t *testing.T) {
	redis := &mockRedisCache{}
	cache := NewTwoTierAlertCache(redis, nil)
	defer cache.Stop()

	alert := createCacheTestAlert("TestAlert", "fp-redis-test")

	// Add alert (should go to Redis)
	err := cache.AddFiringAlert(context.Background(), alert)
	if err != nil {
		t.Fatalf("AddFiringAlert() error = %v", err)
	}

	// Verify Redis was called
	if len(redis.data) == 0 {
		t.Error("Expected alert to be added to Redis")
	}

	// Remove alert (should remove from Redis)
	_ = cache.RemoveAlert(context.Background(), "fp-redis-test")

	if len(redis.data) != 0 {
		t.Error("Expected alert to be removed from Redis")
	}
}

func TestTwoTierAlertCache_RedisFallback(t *testing.T) {
	redis := &mockRedisCache{fail: true} // Redis unavailable
	cache := NewTwoTierAlertCache(redis, nil)
	defer cache.Stop()

	alert := createCacheTestAlert("TestAlert", "fp-fallback")

	// Add should succeed despite Redis failure (L1 fallback)
	err := cache.AddFiringAlert(context.Background(), alert)
	if err != nil {
		t.Errorf("Expected AddFiringAlert to succeed with Redis fallback, got error: %v", err)
	}

	// Get should succeed (from L1)
	alerts, err := cache.GetFiringAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetFiringAlerts() error = %v", err)
	}

	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert from L1 cache, got %d", len(alerts))
	}
}

func TestTwoTierAlertCache_Capacity(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	// Add more than capacity (1000)
	for i := 0; i < 1100; i++ {
		alert := createCacheTestAlert("Alert", fmt.Sprintf("fp-%d", i))
		_ = cache.AddFiringAlert(context.Background(), alert)
	}

	alerts, _ := cache.GetFiringAlerts(context.Background())

	// Should not exceed capacity
	if len(alerts) > 1000 {
		t.Errorf("Expected max 1000 alerts (capacity), got %d", len(alerts))
	}
}

func TestTwoTierAlertCache_Cleanup(t *testing.T) {
	// Use fast cleanup interval for testing (avoid data race)
	opts := &AlertCacheOptions{
		CleanupInterval: 100 * time.Millisecond,
	}
	cache := NewTwoTierAlertCacheWithOptions(nil, nil, opts)
	defer cache.Stop()

	// Add expired alert
	now := time.Now()
	endsAt := now.Add(-5 * time.Minute)
	expiredAlert := &core.Alert{
		AlertName:   "ExpiredAlert",
		Fingerprint: "fp-expired",
		Status:      "firing",
		StartsAt:    now.Add(-10 * time.Minute), // Old
		EndsAt:      &endsAt,                      // Ended
		Labels:      map[string]string{"alertname": "ExpiredAlert"},
	}

	_ = cache.AddFiringAlert(context.Background(), expiredAlert)

	// Add fresh alert
	freshAlert := createCacheTestAlert("FreshAlert", "fp-fresh")
	_ = cache.AddFiringAlert(context.Background(), freshAlert)

	// Wait for cleanup
	time.Sleep(200 * time.Millisecond)

	alerts, _ := cache.GetFiringAlerts(context.Background())

	// Expired alert should be removed, fresh alert should remain
	if len(alerts) != 1 {
		t.Errorf("Expected 1 alert after cleanup, got %d", len(alerts))
	}

	if len(alerts) > 0 && alerts[0].Fingerprint != "fp-fresh" {
		t.Errorf("Expected fresh alert to remain, got %s", alerts[0].Fingerprint)
	}
}

func TestTwoTierAlertCache_AddNilAlert(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	err := cache.AddFiringAlert(context.Background(), nil)
	if err == nil {
		t.Error("Expected error when adding nil alert")
	}
}

func TestTwoTierAlertCache_EmptyCache(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	alerts, err := cache.GetFiringAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetFiringAlerts() error = %v", err)
	}

	if len(alerts) != 0 {
		t.Errorf("Expected 0 alerts from empty cache, got %d", len(alerts))
	}
}

func TestTwoTierAlertCache_OnlyFiringAlerts(t *testing.T) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	// Add firing alert
	firingAlert := createCacheTestAlert("FiringAlert", "fp-firing")
	_ = cache.AddFiringAlert(context.Background(), firingAlert)

	// Add resolved alert
	resolvedAlert := &core.Alert{
		AlertName:   "ResolvedAlert",
		Fingerprint: "fp-resolved",
		Status:      "resolved", // Not firing
		StartsAt:    time.Now(),
		Labels:      map[string]string{"alertname": "ResolvedAlert"},
	}
	_ = cache.AddFiringAlert(context.Background(), resolvedAlert)

	alerts, _ := cache.GetFiringAlerts(context.Background())

	// Should only return firing alert
	if len(alerts) != 1 {
		t.Errorf("Expected 1 firing alert, got %d", len(alerts))
	}

	if alerts[0].Status != "firing" {
		t.Errorf("Expected firing alert, got status: %s", alerts[0].Status)
	}
}

// --- Benchmarks ---

func BenchmarkTwoTierAlertCache_AddFiringAlert(b *testing.B) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	alert := createCacheTestAlert("BenchAlert", "fp-bench")
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cache.AddFiringAlert(ctx, alert)
	}
}

func BenchmarkTwoTierAlertCache_GetFiringAlerts(b *testing.B) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	// Populate with 100 alerts
	for i := 0; i < 100; i++ {
		alert := createCacheTestAlert("Alert", fmt.Sprintf("fp-bench-%d", i))
		_ = cache.AddFiringAlert(context.Background(), alert)
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = cache.GetFiringAlerts(ctx)
	}
}

func BenchmarkTwoTierAlertCache_RemoveAlert(b *testing.B) {
	cache := NewTwoTierAlertCache(nil, nil)
	defer cache.Stop()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Add and remove
		alert := createCacheTestAlert("Alert", fmt.Sprintf("fp-%d", i))
		_ = cache.AddFiringAlert(ctx, alert)
		_ = cache.RemoveAlert(ctx, alert.Fingerprint)
	}
}
