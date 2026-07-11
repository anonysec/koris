// Package redis provides a thin optional Redis client wrapper.
//
// Redis is optional: if REDIS_ADDR is unset the backends that consume this
// client fall back to their in-memory implementations, so a default single-instance
// deployment works unchanged. If REDIS_ADDR is set but the server is unreachable
// at startup, NewClient returns an error and the caller also falls back locally.
package redis

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// NewClient creates a Redis client from the REDIS_ADDR / REDIS_PASSWORD / REDIS_DB
// environment variables. It returns (nil, nil) when REDIS_ADDR is empty, and
// (nil, err) if the server cannot be pinged. Callers should treat a nil client as
// "use the local in-memory implementation".
func NewClient() (*redis.Client, error) {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		return nil, nil
	}

	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       redisDBFromEnv(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("[redis] cannot reach %s, disabling Redis features (falling back to in-memory): %v", addr, err)
		_ = client.Close()
		return nil, err
	}
	return client, nil
}

func redisDBFromEnv() int {
	v := os.Getenv("REDIS_DB")
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil {
		return n
	}
	return 0
}
