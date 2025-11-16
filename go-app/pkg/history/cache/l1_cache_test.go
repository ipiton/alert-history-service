package cache

import (
	"testing"
	"time"
	
	"github.com/vitaliisemenov/alert-history/internal/core"
)

// TestL1Cache_GetSet tests basic Get/Set operations
func TestL1Cache_GetSet(t *testing.T) {
	cache := NewL1Cache(100, 5*time.Minute)
	
	key := "test-key"
	value := &core.HistoryResponse{
		Alerts: []*core.Alert{},
		Total:  10,
		Page:  1,
		PerPage: 50,
	}
	
	// Test Set
	cache.Set(key, value)
	
	// Test Get
	got, found := cache.Get(key)
	if !found {
		t.Error("Get() returned false, want true")
	}
	if got.Total != value.Total {
		t.Errorf("Get() Total = %v, want %v", got.Total, value.Total)
	}
}

// TestL1Cache_Expiration tests cache expiration
func TestL1Cache_Expiration(t *testing.T) {
	cache := NewL1Cache(100, 100*time.Millisecond)
	
	key := "test-key"
	value := &core.HistoryResponse{
		Total: 10,
	}
	
	cache.Set(key, value)
	
	// Should be found immediately
	_, found := cache.Get(key)
	if !found {
		t.Error("Get() returned false immediately after Set")
	}
	
	// Wait for expiration
	time.Sleep(150 * time.Millisecond)
	
	// Should not be found after expiration
	_, found = cache.Get(key)
	if found {
		t.Error("Get() returned true after expiration, want false")
	}
}

// TestL1Cache_Eviction tests LRU eviction when cache is full
func TestL1Cache_Eviction(t *testing.T) {
	cache := NewL1Cache(2, 5*time.Minute)
	
	// Fill cache to capacity
	cache.Set("key1", &core.HistoryResponse{Total: 1})
	cache.Set("key2", &core.HistoryResponse{Total: 2})
	
	// Add one more - should evict oldest
	cache.Set("key3", &core.HistoryResponse{Total: 3})
	
	// key1 should be evicted
	_, found := cache.Get("key1")
	if found {
		t.Error("key1 should be evicted")
	}
	
	// key2 and key3 should still be present
	_, found = cache.Get("key2")
	if !found {
		t.Error("key2 should still be present")
	}
	_, found = cache.Get("key3")
	if !found {
		t.Error("key3 should still be present")
	}
}

// TestL1Cache_Delete tests Delete operation
func TestL1Cache_Delete(t *testing.T) {
	cache := NewL1Cache(100, 5*time.Minute)
	
	key := "test-key"
	value := &core.HistoryResponse{Total: 10}
	
	cache.Set(key, value)
	cache.Delete(key)
	
	_, found := cache.Get(key)
	if found {
		t.Error("Get() returned true after Delete, want false")
	}
}

// TestL1Cache_Clear tests Clear operation
func TestL1Cache_Clear(t *testing.T) {
	cache := NewL1Cache(100, 5*time.Minute)
	
	cache.Set("key1", &core.HistoryResponse{Total: 1})
	cache.Set("key2", &core.HistoryResponse{Total: 2})
	
	cache.Clear()
	
	_, found := cache.Get("key1")
	if found {
		t.Error("Get() returned true after Clear")
	}
	_, found = cache.Get("key2")
	if found {
		t.Error("Get() returned true after Clear")
	}
}

// TestL1Cache_Stats tests Stats functionality
func TestL1Cache_Stats(t *testing.T) {
	cache := NewL1Cache(100, 5*time.Minute)
	
	cache.Set("key1", &core.HistoryResponse{Total: 1})
	cache.Set("key2", &core.HistoryResponse{Total: 2})
	
	stats := cache.Stats()
	
	if stats["entries"].(int) != 2 {
		t.Errorf("Stats() entries = %v, want 2", stats["entries"])
	}
	if stats["max_entries"].(int64) != 100 {
		t.Errorf("Stats() max_entries = %v, want 100", stats["max_entries"])
	}
}

