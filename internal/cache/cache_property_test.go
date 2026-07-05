package cache

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"testing/quick"
	"time"
)

// **Validates: Requirements 4.10**
// Property: cache.Stats().Size <= maxSize always holds after any number of Set operations.
func TestProperty_SizeInvariant(t *testing.T) {
	f := func(maxSize uint8, numOps uint8) bool {
		// Constrain maxSize to [1, 200] to keep tests fast
		size := int(maxSize)%200 + 1
		ops := int(numOps)%500 + 1

		c := NewQueryCache(size, 10*time.Second)

		for i := 0; i < ops; i++ {
			key := fmt.Sprintf("key:%d", i)
			c.Set(key, i)
		}

		stats := c.Stats()
		return stats.Size <= size
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatalf("size invariant violated: %v", err)
	}
}

// **Validates: Requirements 4.10**
// Property: after inserting N unique keys into a cache of maxSize M, the cache contains min(N, M) entries.
func TestProperty_SizeBounds(t *testing.T) {
	f := func(maxSize uint8, numKeys uint8) bool {
		size := int(maxSize)%100 + 1
		n := int(numKeys)%300 + 1

		c := NewQueryCache(size, 10*time.Second)

		for i := 0; i < n; i++ {
			c.Set(fmt.Sprintf("k%d", i), i)
		}

		stats := c.Stats()
		expected := n
		if expected > size {
			expected = size
		}
		return stats.Size == expected
	}

	if err := quick.Check(f, &quick.Config{MaxCount: 200}); err != nil {
		t.Fatalf("size bounds property violated: %v", err)
	}
}

// **Validates: Requirements 4.10**
// Property: after sleeping past TTL, all entries should expire (lazy eviction on Get).
func TestProperty_TTLExpiry(t *testing.T) {
	// Use a short TTL and insert random number of entries, then verify all miss after TTL.
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for trial := 0; trial < 20; trial++ {
		numEntries := rng.Intn(50) + 1
		ttl := 30 * time.Millisecond

		c := NewQueryCache(100, ttl)

		for i := 0; i < numEntries; i++ {
			c.Set(fmt.Sprintf("entry:%d", i), i)
		}

		// Wait past TTL + margin
		time.Sleep(ttl + 20*time.Millisecond)

		// All entries should miss (lazy eviction)
		for i := 0; i < numEntries; i++ {
			_, ok := c.Get(fmt.Sprintf("entry:%d", i))
			if ok {
				t.Fatalf("trial %d: entry:%d should have expired after TTL", trial, i)
			}
		}
	}
}

// **Validates: Requirements 4.10**
// Property: the evicted entry is always the least recently accessed.
func TestProperty_LRUOrdering(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for trial := 0; trial < 50; trial++ {
		maxSize := rng.Intn(10) + 2 // [2, 11]
		c := NewQueryCache(maxSize, 10*time.Second)

		// Fill the cache completely
		keys := make([]string, maxSize)
		for i := 0; i < maxSize; i++ {
			keys[i] = fmt.Sprintf("key:%d", i)
			c.Set(keys[i], i)
		}

		// Access a random subset to make them recently used
		// Pick one key NOT to access — that should be the LRU and get evicted
		lruIdx := rng.Intn(maxSize)
		for i := 0; i < maxSize; i++ {
			if i != lruIdx {
				c.Get(keys[i])
			}
		}

		// Insert one more entry to trigger eviction
		c.Set("new-entry", 999)

		// The LRU key should have been evicted
		_, ok := c.Get(keys[lruIdx])
		if ok {
			t.Fatalf("trial %d: expected key %q (idx %d) to be evicted as LRU",
				trial, keys[lruIdx], lruIdx)
		}

		// All other original keys should still exist
		for i := 0; i < maxSize; i++ {
			if i == lruIdx {
				continue
			}
			_, ok := c.Get(keys[i])
			if !ok {
				t.Fatalf("trial %d: expected key %q to still exist", trial, keys[i])
			}
		}
	}
}

// **Validates: Requirements 4.10**
// Property: concurrent reads, writes, and invalidations do not cause panics or data races.
// Run with -race flag to detect race conditions.
func TestProperty_ConcurrentSafety(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for trial := 0; trial < 10; trial++ {
		maxSize := rng.Intn(50) + 10
		c := NewQueryCache(maxSize, 100*time.Millisecond)

		var wg sync.WaitGroup
		goroutines := rng.Intn(50) + 20

		// Launch writers
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := fmt.Sprintf("k:%d:%d", id, j%20)
					c.Set(key, j)
				}
			}(i)
		}

		// Launch readers
		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 100; j++ {
					key := fmt.Sprintf("k:%d:%d", id, j%20)
					c.Get(key)
				}
			}(i)
		}

		// Launch invalidators
		for i := 0; i < goroutines/5; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < 20; j++ {
					c.Invalidate(fmt.Sprintf("k:%d:%d", id, j))
					c.InvalidatePrefix(fmt.Sprintf("k:%d:", id))
				}
			}(i)
		}

		// Launch stats readers
		for i := 0; i < goroutines/5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for j := 0; j < 50; j++ {
					stats := c.Stats()
					if stats.Size < 0 || stats.Size > maxSize {
						t.Errorf("size invariant violated under concurrency: %d (max %d)", stats.Size, maxSize)
					}
				}
			}()
		}

		wg.Wait()

		// Final invariant check
		stats := c.Stats()
		if stats.Size < 0 || stats.Size > maxSize {
			t.Fatalf("trial %d: final size %d exceeds maxSize %d", trial, stats.Size, maxSize)
		}
	}
}
