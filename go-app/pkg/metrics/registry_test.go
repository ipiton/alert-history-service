package metrics

import (
	"sync"
	"testing"
)

func TestDefaultRegistry_Singleton(t *testing.T) {
	// Test that DefaultRegistry returns the same instance
	registry1 := DefaultRegistry()
	registry2 := DefaultRegistry()

	if registry1 != registry2 {
		t.Error("DefaultRegistry() should return singleton instance")
	}
}

func TestDefaultRegistry_ConcurrentAccess(t *testing.T) {
	// Test thread-safety of singleton pattern
	var wg sync.WaitGroup
	registries := make([]*MetricsRegistry, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			registries[index] = DefaultRegistry()
		}(i)
	}

	wg.Wait()

	// All should be the same instance
	first := registries[0]
	for i := 1; i < len(registries); i++ {
		if registries[i] != first {
			t.Errorf("Registry at index %d is not the same instance", i)
		}
	}
}

func TestNewMetricsRegistry(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		expected  string
	}{
		{
			name:      "with custom namespace",
			namespace: "test_service",
			expected:  "test_service",
		},
		{
			name:      "with empty namespace (should default)",
			namespace: "",
			expected:  "alert_history",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewMetricsRegistry(tt.namespace)
			if registry.Namespace() != tt.expected {
				t.Errorf("Namespace() = %q, want %q", registry.Namespace(), tt.expected)
			}
		})
	}
}

func TestMetricsRegistry_Business(t *testing.T) {
	registry := NewMetricsRegistry("test_reg_biz")

	// First call should initialize
	business1 := registry.Business()
	if business1 == nil {
		t.Fatal("Business() returned nil")
	}

	// Second call should return same instance
	business2 := registry.Business()
	if business1 != business2 {
		t.Error("Business() should return same instance on subsequent calls")
	}

	// Check that metrics are initialized
	if business1.AlertsProcessedTotal == nil {
		t.Error("AlertsProcessedTotal not initialized")
	}
	if business1.LLMConfidenceScore == nil {
		t.Error("LLMConfidenceScore not initialized")
	}
	if business1.PublishingDurationSeconds == nil {
		t.Error("PublishingDurationSeconds not initialized")
	}
}

func TestMetricsRegistry_Technical(t *testing.T) {
	registry := NewMetricsRegistry("test_reg_tech")

	// First call should initialize
	technical1 := registry.Technical()
	if technical1 == nil {
		t.Fatal("Technical() returned nil")
	}

	// Second call should return same instance
	technical2 := registry.Technical()
	if technical1 != technical2 {
		t.Error("Technical() should return same instance on subsequent calls")
	}

	// Check that subsystems are initialized
	if technical1.HTTP == nil {
		t.Error("HTTP metrics not initialized")
	}
	if technical1.Filter == nil {
		t.Error("Filter metrics not initialized")
	}
	if technical1.Enrichment == nil {
		t.Error("Enrichment metrics not initialized")
	}
}

func TestMetricsRegistry_Infra(t *testing.T) {
	registry := NewMetricsRegistry("test_reg_infra")

	// First call should initialize
	infra1 := registry.Infra()
	if infra1 == nil {
		t.Fatal("Infra() returned nil")
	}

	// Second call should return same instance
	infra2 := registry.Infra()
	if infra1 != infra2 {
		t.Error("Infra() should return same instance on subsequent calls")
	}

	// Check that subsystems are initialized
	if infra1.DB == nil {
		t.Error("DB metrics not initialized")
	}
	if infra1.Cache == nil {
		t.Error("Cache metrics not initialized")
	}
	if infra1.Repository == nil {
		t.Error("Repository metrics not initialized")
	}
}

func TestMetricsRegistry_LazyInitialization(t *testing.T) {
	registry := NewMetricsRegistry("test_lazy_init_unique")

	// Initially, category managers should be nil (lazy init)
	if registry.business != nil {
		t.Error("Business should be nil before first access")
	}
	if registry.infra != nil {
		t.Error("Infra should be nil before first access")
	}

	// Access Business - only Business should be initialized
	_ = registry.Business()
	if registry.business == nil {
		t.Error("Business should be initialized after access")
	}
	// Infra should still be nil (independent lazy init)
	if registry.infra != nil {
		t.Error("Infra should still be nil (not accessed yet)")
	}

	// Access Infra - should be initialized now
	_ = registry.Infra()
	if registry.infra == nil {
		t.Error("Infra should be initialized after access")
	}

	// Note: Technical test skipped due to HTTPMetrics using fixed namespace
	// (already tested separately in TestMetricsRegistry_Technical)
}

func BenchmarkDefaultRegistry(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = DefaultRegistry()
	}
}

func BenchmarkMetricsRegistry_Business(b *testing.B) {
	registry := DefaultRegistry()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = registry.Business()
	}
}

func BenchmarkMetricsRegistry_AllCategories(b *testing.B) {
	registry := DefaultRegistry()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = registry.Business()
		_ = registry.Technical()
		_ = registry.Infra()
	}
}
