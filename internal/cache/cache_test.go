package cache

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestGetSet(t *testing.T) {
	c := NewQueryCache(10, 5*time.Second)

	// Miss on empty cache
	val, ok := c.Get("key1")
	if ok || val != nil {
		t.Fatal("expected miss on empty cache")
	}

	// Set and get
	c.Set("key1", "value1")
	val, ok = c.Get("key1")
	if !ok || val != "value1" {
		t.Fatalf("expected hit with value1, got ok=%v val=%v", ok, val)
	}

	// Overwrite
	c.Set("key1", "value2")
	val, ok = c.Get("key1")
	if !ok || val != "value2" {
		t.Fatalf("expected updated value2, got ok=%v val=%v", ok, val)
	}
}

func TestTTLExpiry(t *testing.T) {
	c := NewQueryCache(10, 50*time.Millisecond)

	c.Set("key1", "value1")

	// Immediately available
	val, ok := c.Get("key1")
	if !ok || val != "value1" {
		t.Fatal("expected hit immediately after set")
	}

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	val, ok = c.Get("key1")
	if ok || val != nil {
		t.Fatal("expected miss after TTL expiry")
	}

	// Entry should be removed from cache
	c.mu.RLock()
	size := c.order.Len()
	c.mu.RUnlock()
	if size != 0 {
		t.Fatalf("expected cache size 0 after lazy eviction, got %d", size)
	}
}

func TestLRUEviction(t *testing.T) {
	c := NewQueryCache(3, 10*time.Second)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	// Access "a" to make it recently used
	c.Get("a")

	// Add "d" — should evict "b" (least recently used)
	c.Set("d", 4)

	// "b" should be evicted
	_, ok := c.Get("b")
	if ok {
		t.Fatal("expected 'b' to be evicted")
	}

	// "a", "c", "d" should still exist
	for _, key := range []string{"a", "c", "d"} {
		_, ok := c.Get(key)
		if !ok {
			t.Fatalf("expected key %q to exist", key)
		}
	}
}

func TestLRUEvictionOrder(t *testing.T) {
	c := NewQueryCache(3, 10*time.Second)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3)

	// No access to any — "a" is LRU (added first, never accessed since)
	c.Set("d", 4)

	_, ok := c.Get("a")
	if ok {
		t.Fatal("expected 'a' to be evicted as LRU")
	}

	// "b", "c", "d" should exist
	for _, key := range []string{"b", "c", "d"} {
		_, ok := c.Get(key)
		if !ok {
			t.Fatalf("expected key %q to exist", key)
		}
	}
}

func TestInvalidate(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	c.Set("key1", "v1")
	c.Set("key2", "v2")
	c.Set("key3", "v3")

	c.Invalidate("key1", "key3")

	_, ok := c.Get("key1")
	if ok {
		t.Fatal("expected key1 invalidated")
	}
	_, ok = c.Get("key3")
	if ok {
		t.Fatal("expected key3 invalidated")
	}

	val, ok := c.Get("key2")
	if !ok || val != "v2" {
		t.Fatal("expected key2 to still exist")
	}
}

func TestInvalidatePrefix(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	c.Set("stats:dashboard", "d1")
	c.Set("stats:users", "d2")
	c.Set("plans:list", "p1")
	c.Set("nodes:list", "n1")

	c.InvalidatePrefix("stats:")

	_, ok := c.Get("stats:dashboard")
	if ok {
		t.Fatal("expected stats:dashboard invalidated")
	}
	_, ok = c.Get("stats:users")
	if ok {
		t.Fatal("expected stats:users invalidated")
	}

	// Non-matching prefix should remain
	val, ok := c.Get("plans:list")
	if !ok || val != "p1" {
		t.Fatal("expected plans:list to still exist")
	}
	val, ok = c.Get("nodes:list")
	if !ok || val != "n1" {
		t.Fatal("expected nodes:list to still exist")
	}
}

func TestStats(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	c.Set("a", 1)
	c.Set("b", 2)

	// 2 hits
	c.Get("a")
	c.Get("b")
	// 1 miss
	c.Get("nonexistent")

	stats := c.Stats()
	if stats.Hits != 2 {
		t.Fatalf("expected 2 hits, got %d", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Fatalf("expected 1 miss, got %d", stats.Misses)
	}
	if stats.Size != 2 {
		t.Fatalf("expected size 2, got %d", stats.Size)
	}
	expectedRate := 2.0 / 3.0
	if stats.HitRate < expectedRate-0.01 || stats.HitRate > expectedRate+0.01 {
		t.Fatalf("expected hit rate ~%.3f, got %.3f", expectedRate, stats.HitRate)
	}
}

func TestStatsEvictions(t *testing.T) {
	c := NewQueryCache(2, 10*time.Second)

	c.Set("a", 1)
	c.Set("b", 2)
	c.Set("c", 3) // evicts "a"
	c.Set("d", 4) // evicts "b"

	stats := c.Stats()
	if stats.Evictions != 2 {
		t.Fatalf("expected 2 evictions, got %d", stats.Evictions)
	}
	if stats.Size != 2 {
		t.Fatalf("expected size 2, got %d", stats.Size)
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := NewQueryCache(100, 5*time.Second)

	var wg sync.WaitGroup
	// Concurrent writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key:%d", i)
			c.Set(key, i)
		}(i)
	}
	// Concurrent readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key:%d", i)
			c.Get(key)
		}(i)
	}
	// Concurrent invalidators
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c.InvalidatePrefix(fmt.Sprintf("key:%d", i))
		}(i)
	}

	wg.Wait()

	// Should not panic or deadlock — just verify stats are consistent
	stats := c.Stats()
	if stats.Size < 0 || stats.Size > 100 {
		t.Fatalf("unexpected cache size: %d", stats.Size)
	}
}

func TestNewQueryCacheDefaultSize(t *testing.T) {
	// Zero or negative maxSize defaults to 100
	c := NewQueryCache(0, time.Second)
	if c.maxSize != 100 {
		t.Fatalf("expected default maxSize 100, got %d", c.maxSize)
	}

	c = NewQueryCache(-5, time.Second)
	if c.maxSize != 100 {
		t.Fatalf("expected default maxSize 100, got %d", c.maxSize)
	}
}

func TestSetUpdatesExpiresAt(t *testing.T) {
	c := NewQueryCache(10, 100*time.Millisecond)

	c.Set("key1", "v1")

	// Wait half the TTL
	time.Sleep(60 * time.Millisecond)

	// Re-set to refresh TTL
	c.Set("key1", "v1-updated")

	// Wait another 60ms (110ms total since first set — past original TTL)
	time.Sleep(60 * time.Millisecond)

	// Should still be available because Set refreshed the TTL
	val, ok := c.Get("key1")
	if !ok || val != "v1-updated" {
		t.Fatal("expected key1 to be available after TTL refresh via Set")
	}
}

func TestInvalidateNonExistentKey(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	c.Set("key1", "v1")

	// Should not panic when invalidating non-existent keys
	c.Invalidate("nonexistent", "also-missing")

	// Original key should still be accessible
	val, ok := c.Get("key1")
	if !ok || val != "v1" {
		t.Fatal("expected key1 to still exist after invalidating non-existent keys")
	}

	stats := c.Stats()
	if stats.Size != 1 {
		t.Fatalf("expected size 1, got %d", stats.Size)
	}
}

func TestInvalidatePrefixNoMatches(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	c.Set("alpha:1", "a1")
	c.Set("alpha:2", "a2")
	c.Set("beta:1", "b1")

	// Should not panic when prefix matches nothing
	c.InvalidatePrefix("gamma:")

	// All entries should still exist
	for _, key := range []string{"alpha:1", "alpha:2", "beta:1"} {
		_, ok := c.Get(key)
		if !ok {
			t.Fatalf("expected key %q to still exist", key)
		}
	}

	stats := c.Stats()
	if stats.Size != 3 {
		t.Fatalf("expected size 3, got %d", stats.Size)
	}
}

func TestGetEmptyCache(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	// Get on empty cache
	val, ok := c.Get("anything")
	if ok {
		t.Fatal("expected miss on empty cache")
	}
	if val != nil {
		t.Fatalf("expected nil value on miss, got %v", val)
	}

	stats := c.Stats()
	if stats.Misses != 1 {
		t.Fatalf("expected 1 miss, got %d", stats.Misses)
	}
}

func TestStatsZeroState(t *testing.T) {
	c := NewQueryCache(10, 10*time.Second)

	stats := c.Stats()
	if stats.Hits != 0 {
		t.Fatalf("expected 0 hits, got %d", stats.Hits)
	}
	if stats.Misses != 0 {
		t.Fatalf("expected 0 misses, got %d", stats.Misses)
	}
	if stats.Size != 0 {
		t.Fatalf("expected size 0, got %d", stats.Size)
	}
	if stats.Evictions != 0 {
		t.Fatalf("expected 0 evictions, got %d", stats.Evictions)
	}
	if stats.HitRate != 0 {
		t.Fatalf("expected hit rate 0, got %f", stats.HitRate)
	}
}

func TestSetSameKeyMultipleTimes(t *testing.T) {
	c := NewQueryCache(5, 10*time.Second)

	// Set same key multiple times — should update in place, not increase size
	for i := 0; i < 10; i++ {
		c.Set("key", fmt.Sprintf("value-%d", i))
	}

	stats := c.Stats()
	if stats.Size != 1 {
		t.Fatalf("expected size 1 after repeated sets on same key, got %d", stats.Size)
	}

	val, ok := c.Get("key")
	if !ok || val != "value-9" {
		t.Fatalf("expected last value 'value-9', got ok=%v val=%v", ok, val)
	}
}
