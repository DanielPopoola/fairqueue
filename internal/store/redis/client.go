// Package redis contains Redis-backed store implementations.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Client wraps a Redis client with shared helpers.
type Client struct {
	rdb *redis.Client
}

// NewClient creates a Client and verifies connectivity.
func NewClient(ctx context.Context, rdb *redis.Client) (*Client, error) {
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("pinging redis: %w", err)
	}
	return &Client{rdb: rdb}, nil
}

// SetEX sets a key with a TTL. Used by OTPStore.
func (c *Client) SetEX(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := c.rdb.Set(ctx, key, value, ttl).Err(); err != nil {
		return fmt.Errorf("setting key %s: %w", key, err)
	}
	return nil
}

// Get returns the string value for a key.
// Returns redis.Nil wrapped in a descriptive error on cache miss.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", fmt.Errorf("key not found: %s", key)
		}
		return "", fmt.Errorf("getting key %s: %w", key, err)
	}
	return val, nil
}

// Del deletes one or more keys. Used by OTPStore after successful verification.
func (c *Client) Del(ctx context.Context, keys ...string) error {
	if err := c.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("deleting keys: %w", err)
	}
	return nil
}
