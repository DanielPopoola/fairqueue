package config

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func (c *DatabaseConfig) Pool(ctx context.Context) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)

	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parsing pool config: %w", err)
	}

	cfg.MaxConns = int32(c.MaxOpenConns) //nolint:gosec // simple ints are fine
	cfg.MinConns = int32(c.MaxIdleConns) //nolint:gosec // simple ints are fine
	cfg.MaxConnLifetime = c.ConnMaxLifetime
	cfg.MaxConnIdleTime = c.ConnMaxIdleTime
	cfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("creating pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}
