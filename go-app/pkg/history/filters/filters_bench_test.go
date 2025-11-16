package filters

import (
	"testing"

	"github.com/vitaliisemenov/alert-history/pkg/history/query"
)

// BenchmarkStatusFilter_ApplyToQuery benchmarks StatusFilter query application
func BenchmarkStatusFilter_ApplyToQuery(b *testing.B) {
	filter, _ := NewStatusFilter(map[string]interface{}{
		"values": []string{"firing"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := query.NewBuilder()
		_ = filter.ApplyToQuery(qb)
	}
}

// BenchmarkSeverityFilter_ApplyToQuery benchmarks SeverityFilter query application
func BenchmarkSeverityFilter_ApplyToQuery(b *testing.B) {
	filter, _ := NewSeverityFilter(map[string]interface{}{
		"values": []string{"critical", "warning"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := query.NewBuilder()
		_ = filter.ApplyToQuery(qb)
	}
}

// BenchmarkFilterRegistry_Create benchmarks filter creation
func BenchmarkFilterRegistry_Create(b *testing.B) {
	registry := NewRegistry(nil)
	params := map[string]interface{}{
		"values": []string{"firing"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.Create(FilterTypeStatus, params)
	}
}

// BenchmarkFilterRegistry_CreateFromQueryParams benchmarks query parameter parsing
func BenchmarkFilterRegistry_CreateFromQueryParams(b *testing.B) {
	registry := NewRegistry(nil)
	queryParams := map[string][]string{
		"status":   {"firing"},
		"severity": {"critical"},
		"namespace": {"production"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = registry.CreateFromQueryParams(queryParams)
	}
}
