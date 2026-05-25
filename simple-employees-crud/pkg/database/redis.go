// Package database (redis.go) provides the Redis client and cache utilities.
// Redis is used by the service layer for short-lived caching of frequently
// read data (employee lists, individual employees) and for storing refresh
// token blacklist entries on logout.
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"employee-management/pkg/logger"
)

// RedisConfig holds connection parameters for the Redis client.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// NewRedisClient creates, tests, and returns a Redis client.
func NewRedisClient(ctx context.Context, cfg RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis: ping failed: %w", err)
	}

	logger.S().Infow("Redis client connected", "addr", cfg.Addr, "db", cfg.DB)
	return client, nil
}

// ─── Generic cache helpers ────────────────────────────────────────────────────

// CacheSet marshals v to JSON and writes it to Redis under key with ttl.
func CacheSet(ctx context.Context, client *redis.Client, key string, v any, ttl time.Duration) error {
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("cache: marshal error for key %q: %w", key, err)
	}
	return client.Set(ctx, key, data, ttl).Err()
}

// CacheGet retrieves key from Redis and unmarshals the JSON value into dest.
// Returns redis.Nil if the key does not exist (cache miss).
func CacheGet(ctx context.Context, client *redis.Client, key string, dest any) error {
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return err // callers check for redis.Nil
	}
	return json.Unmarshal(data, dest)
}

// CacheDel removes one or more keys from Redis.
func CacheDel(ctx context.Context, client *redis.Client, keys ...string) error {
	return client.Del(ctx, keys...).Err()
}

// CacheDelPattern removes all keys matching a glob pattern.
// Use sparingly — SCAN-based deletion can be slow on large key spaces.
func CacheDelPattern(ctx context.Context, client *redis.Client, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return fmt.Errorf("cache: scan error for pattern %q: %w", pattern, err)
		}
		if len(keys) > 0 {
			if err := client.Del(ctx, keys...).Err(); err != nil {
				return fmt.Errorf("cache: delete error: %w", err)
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}
