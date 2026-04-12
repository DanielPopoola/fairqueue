// Package migrate runs database migrations embedded at compile time.
// SQL files are baked into the binary — no filesystem dependency at runtime.
package migrate

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

// Run applies all *.up.sql files in the embedded migrations directory
// in lexicographic order. Down migrations are skipped.
// Safe to call on every startup — Postgres will error on duplicate
// object creation, so migrations must use IF NOT EXISTS or be truly
// idempotent. The standard pattern is to wrap each migration in a
// transaction with a version check table (see note below).
//
// For MVP, migrations are applied directly. A proper migration tool
// (golang-migrate) should replace this before production.
func Run(ctx context.Context, pool *pgxpool.Pool, logger *slog.Logger) error {
	entries, err := fs.ReadDir(migrationsFS, "sql")
	if err != nil {
		return fmt.Errorf("reading embedded migrations: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		if filepath.Ext(name) != ".sql" {
			continue
		}

		// Skip down migrations
		if strings.Contains(name, ".down.") {
			continue
		}

		sql, err := migrationsFS.ReadFile("sql/" + name)
		if err != nil {
			return fmt.Errorf("reading %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("applying %s: %w", name, err)
		}

		logger.Info("migration applied", "file", name)
	}

	return nil
}
