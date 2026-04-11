package testhelpers

import (
	"context"
	"log/slog"

	"github.com/DanielPopoola/fairqueue/internal/infra/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrationsOnPool applies the schema to the test database.
// Returns an error instead of calling t.Fatal so it can be
// used in TestMain where no *testing.T is available.
func RunMigrationsOnPool(ctx context.Context, pool *pgxpool.Pool) error {
	return migrate.Run(ctx, pool, slog.Default())
}
