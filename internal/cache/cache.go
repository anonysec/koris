package cache

import (
	"container/list"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// QueryCache provides in-memory caching for frequently-read, seldom-written queries.
// It uses an LRU eviction policy with per-entry TTL expiration.
type QueryCache struct {
	mu        sync.RWMutex
	entries   map[string]*list.Element
	order     *list.List // front = most recently used
	maxSize   int
	ttl       time.Duration
	hits      atomic.Int64
	misses    atomic.Int64
	evictions atomic.Int64
}

// cacheEntry is stored as the value in each list element.
type cacheEntry struct {
	key       string
	value     any
	expiresAt time.Time
}

// CacheStats contains cache performance metrics.
type CacheStats struct {
	Hits      int64   `json:"hits"`
	Misses    int64   `json:"misses"`
	Size      int     `json:"size"`
	Evictions int64   `json:"evictions"`
	HitRate   float64 `json:"hit_rate"`
}

// NewQueryCache creates a new LRU cache with the given maximum size and TTL.
// maxSize is the maximum number of entries. ttl is the time-to-live for each entry.
func NewQueryCache(maxSize int, ttl time.Duration) *QueryCache {
	if maxSize <= 0 {
		maxSize = 100
	}
	return &QueryCache{
		entries: make(map[string]*list.Element, maxSize),
		order:   list.New(),
		maxSize: maxSize,
		ttl:     ttl,
	}
}

// Get retrieves a value from the cache. Returns (value, true) on hit, (nil, false) on miss.
// Expired entries are lazily evicted on access.
func (c *QueryCache) Get(key string) (any, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		c.misses.Add(1)
		return nil, false
	}

	entry := elem.Value.(*cacheEntry)

	// Check TTL — lazy eviction
	if time.Now().After(entry.expiresAt) {
		c.removeLocked(elem)
		c.misses.Add(1)
		return nil, false
	}

	// Move to front (most recently used)
	c.order.MoveToFront(elem)
	c.hits.Add(1)
	return entry.value, true
}

// Set adds or updates a cache entry. If the cache is at capacity, the least
// recently used entry is evicted.
func (c *QueryCache) Set(key string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If key exists, update in place and move to front
	if elem, ok := c.entries[key]; ok {
		entry := elem.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(c.ttl)
		c.order.MoveToFront(elem)
		return
	}

	// Evict LRU if at capacity
	for c.order.Len() >= c.maxSize {
		c.evictLRU()
	}

	// Insert new entry at front
	entry := &cacheEntry{
		key:       key,
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	elem := c.order.PushFront(entry)
	c.entries[key] = elem
}

// Invalidate removes one or more specific keys from the cache.
func (c *QueryCache) Invalidate(keys ...string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, key := range keys {
		if elem, ok := c.entries[key]; ok {
			c.removeLocked(elem)
		}
	}
}

// InvalidatePrefix removes all entries whose key starts with the given prefix.
func (c *QueryCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key, elem := range c.entries {
		if strings.HasPrefix(key, prefix) {
			c.removeLocked(elem)
		}
	}
}

// Stats returns current cache performance metrics.
func (c *QueryCache) Stats() CacheStats {
	c.mu.RLock()
	size := c.order.Len()
	c.mu.RUnlock()

	hits := c.hits.Load()
	misses := c.misses.Load()
	evictions := c.evictions.Load()

	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}

	return CacheStats{
		Hits:      hits,
		Misses:    misses,
		Size:      size,
		Evictions: evictions,
		HitRate:   hitRate,
	}
}

// evictLRU removes the least recently used entry (back of the list).
// Caller must hold c.mu write lock.
func (c *QueryCache) evictLRU() {
	back := c.order.Back()
	if back == nil {
		return
	}
	c.removeLocked(back)
	c.evictions.Add(1)
}

// removeLocked removes an element from both the list and the map.
// Caller must hold c.mu write lock.
func (c *QueryCache) removeLocked(elem *list.Element) {
	entry := elem.Value.(*cacheEntry)
	delete(c.entries, entry.key)
	c.order.Remove(elem)
}
