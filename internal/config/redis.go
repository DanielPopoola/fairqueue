package config

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

func (c *RedisConfig) Client() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", c.Host, c.Port),
		Password: c.Password,
		DB:       c.DB,
	})
}
