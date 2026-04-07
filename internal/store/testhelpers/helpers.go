package testhelpers

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Truncate clears all tables between tests so each test
// starts with a clean database state.
func Truncate(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE
			payments,
			queue_entries,
			claims,
			events,
			customers,
			organizers
		RESTART IDENTITY CASCADE
	`)
	if err != nil {
		t.Fatalf("truncating tables: %v", err)
	}
}

// FlushRedis clears all keys between tests so each test
// starts with a clean Redis state.
func FlushRedis(ctx context.Context, t *testing.T, client *redis.Client) {
	t.Helper()

	if err := client.FlushAll(ctx).Err(); err != nil {
		t.Fatalf("flushing redis: %v", err)
	}
}

// testhelpers/helpers.go

func NewTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
