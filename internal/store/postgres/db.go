package postgres

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Executor defines the interface for executing database queries
type Executor interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// DB wraps a pgx pool.
type DB struct {
	Pool   *pgxpool.Pool
	logger *slog.Logger
}

func NewDB(pool *pgxpool.Pool) *DB {
	return NewDBWithLogger(pool, nil)
}

// NewDBWithLogger creates a DB with structured logging.
// When logger is nil, slog.Default() is used.
func NewDBWithLogger(pool *pgxpool.Pool, logger *slog.Logger) *DB {
	if logger == nil {
		logger = slog.Default()
	}

	return &DB{
		Pool:   pool,
		logger: logger.With("component", "postgres.db"),
	}
}

// Begin starts a new transaction.
func (db *DB) Begin(ctx context.Context) (pgx.Tx, error) {
	if db.Pool == nil {
		err := errors.New("postgres pool is nil")
		db.logger.Error("failed to begin transaction", "error", err)
		return nil, err
	}

	return db.Pool.Begin(ctx)
}

// WithTransaction executes fn within a transaction.
// Commits on success, rolls back on any error.
func (db *DB) WithTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	start := time.Now()
	if db.Pool == nil {
		err := errors.New("postgres pool is nil")
		db.logger.Error("failed to begin transaction", "error", err)
		return err
	}

	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		db.logger.Error("failed to begin transaction", "error", err)
		return err
	}
	db.logger.Debug("transaction started")

	if err := fn(tx); err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			db.logger.Error(
				"transaction rollback failed",
				"error", rollbackErr,
				"original_error", err,
				"duration", time.Since(start),
			)
			return fmt.Errorf("rolling back transaction after error: %w", errors.Join(err, rollbackErr))
		}

		db.logger.Warn("transaction rolled back", "error", err, "duration", time.Since(start))
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		db.logger.Error("failed to commit transaction", "error", err, "duration", time.Since(start))
		return err
	}

	db.logger.Debug("transaction committed", "duration", time.Since(start))
	return nil
}

func (db *DB) Close() {
	if db.Pool == nil {
		db.logger.Warn("close called with nil postgres pool")
		return
	}

	db.logger.Info("closing database connection pool")
	db.Pool.Close()
}

// IsUniqueViolation checks if an error is a Postgres unique constraint violation.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
