package testhelpers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrationsOnPool applies the schema to the test database.
// Returns an error instead of calling t.Fatal so it can be
// used in TestMain where no *testing.T is available.
func RunMigrationsOnPool(ctx context.Context, pool *pgxpool.Pool) error {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "/../..")
	migrationsDir := filepath.Join(projectRoot, "migrations")

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("reading migrations directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if filepath.Ext(entry.Name()) != ".sql" {
			continue
		}

		path := filepath.Join(migrationsDir, entry.Name())
		sql, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("reading migration %s: %w", entry.Name(), err)
		}

		if _, err := pool.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("running migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}
