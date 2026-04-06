// Package redis contains Redis-backed store implementations.
package redis

import (
	"context"
	"fmt"

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
