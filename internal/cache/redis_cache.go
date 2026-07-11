package cache

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache is a Cache implementation backed by Redis. Values are JSON-encoded
// and stored with a TTL. It is selected when Redis is configured, so that
// multiple panel instances share a single cache. Hit/miss counters are tracked
// locally (best-effort) since Redis does not expose them per-key.
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	hits   atomic.Int64
	misses atomic.Int64
}

// NewRedisCache builds a Redis-backed cache. Returns nil when client is nil so
// callers can fall back to the in-memory QueryCache.
func NewRedisCache(client *redis.Client, ttl time.Duration) *RedisCache {
	if client == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &RedisCache{client: client, ttl: ttl}
}

func (c *RedisCache) Get(key string) (any, bool) {
	data, err := c.client.Get(context.Background(), key).Bytes()
	if err != nil || len(data) == 0 {
		c.misses.Add(1)
		return nil, false
	}
	var val any
	if err := json.Unmarshal(data, &val); err != nil {
		c.misses.Add(1)
		return nil, false
	}
	c.hits.Add(1)
	return val, true
}

func (c *RedisCache) Set(key string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	c.client.Set(context.Background(), key, data, c.ttl)
}

// InvalidatePrefix removes every key matching the prefix using a SCAN + DEL
// iteration (best-effort — iteration errors are ignored).
func (c *RedisCache) InvalidatePrefix(prefix string) {
	ctx := context.Background()
	iter := c.client.Scan(ctx, 0, prefix+"*", 100).Iterator()
	for iter.Next(ctx) {
		c.client.Del(ctx, iter.Val())
	}
}

func (c *RedisCache) Stats() CacheStats {
	hits := c.hits.Load()
	misses := c.misses.Load()
	var hitRate float64
	total := hits + misses
	if total > 0 {
		hitRate = float64(hits) / float64(total)
	}
	return CacheStats{
		Hits:    hits,
		Misses:  misses,
		HitRate: hitRate,
	}
}
