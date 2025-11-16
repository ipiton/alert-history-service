package cache

import (
	"testing"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// BenchmarkL1Cache_Get benchmarks L1 cache Get operation
func BenchmarkL1Cache_Get(b *testing.B) {
	cache := NewL1Cache(1000, 5*60*1000*1000*1000) // 5 minutes in nanoseconds
	key := "test-key"
	value := &core.HistoryResponse{Total: 10}
	cache.Set(key, value)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get(key)
	}
}

// BenchmarkL1Cache_Set benchmarks L1 cache Set operation
func BenchmarkL1Cache_Set(b *testing.B) {
	cache := NewL1Cache(1000, 5*60*1000*1000*1000)
	value := &core.HistoryResponse{Total: 10}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set(string(rune(i)), value)
	}
}

// BenchmarkManager_GenerateCacheKey benchmarks cache key generation
func BenchmarkManager_GenerateCacheKey(b *testing.B) {
	cfg := DefaultConfig()
	cfg.L1Enabled = true
	cfg.L2Enabled = false
	manager, _ := NewManager(cfg, nil)
	defer manager.Close()
	
	req := &core.HistoryRequest{
		Pagination: &core.Pagination{
			Page:    1,
			PerPage: 50,
		},
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manager.GenerateCacheKey(req)
	}
}

