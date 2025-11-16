package query

import (
	"testing"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// BenchmarkBuilder_Build benchmarks query building
func BenchmarkBuilder_Build(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb := NewBuilder()
		qb.AddWhere("status = ?", "firing")
		qb.SetLimit(50)
		qb.SetOffset(0)
		qb.AddOrderBy("starts_at", core.SortOrderDesc)
		_, _ = qb.Build()
	}
}

// BenchmarkBuilder_AddWhere benchmarks WHERE clause addition
func BenchmarkBuilder_AddWhere(b *testing.B) {
	qb := NewBuilder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb.AddWhere("status = ?", "firing")
	}
}

// BenchmarkBuilder_SetLimitOffset benchmarks limit/offset setting
func BenchmarkBuilder_SetLimitOffset(b *testing.B) {
	qb := NewBuilder()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		qb.SetLimit(50)
		qb.SetOffset(0)
	}
}

