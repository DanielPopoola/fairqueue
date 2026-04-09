package postgres

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func newBufferLogger(buf *bytes.Buffer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func TestNewDBWithLogger_UsesProvidedLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := newBufferLogger(&buf)
	db := NewDBWithLogger(nil, logger)

	db.Close()

	output := buf.String()
	if !strings.Contains(output, "postgres.db") {
		t.Fatalf("expected component attribute in logs, got: %s", output)
	}
}

func TestNewDB_UsesDefaultLoggerWhenNil(t *testing.T) {
	db := NewDB(nil)
	if db.logger == nil {
		t.Fatal("expected default logger to be set")
	}
}

func TestWithTransaction_ReturnsErrorWhenPoolIsNil(t *testing.T) {
	var buf bytes.Buffer
	db := NewDBWithLogger(nil, newBufferLogger(&buf))

	err := db.WithTransaction(context.Background(), func(_ pgx.Tx) error {
		t.Fatal("transaction callback should not run when pool is nil")
		return nil
	})
	if err == nil {
		t.Fatal("expected error when pool is nil")
	}

	if !strings.Contains(buf.String(), "failed to begin transaction") {
		t.Fatalf("expected begin transaction error log, got: %s", buf.String())
	}
}

func TestBegin_ReturnsErrorWhenPoolIsNil(t *testing.T) {
	db := NewDBWithLogger(nil, slog.Default())

	tx, err := db.Begin(context.Background())
	if err == nil {
		t.Fatal("expected error when pool is nil")
	}
	if tx != nil {
		t.Fatal("expected nil transaction when pool is nil")
	}
}

func TestIsUniqueViolation(t *testing.T) {
	err := &pgconn.PgError{Code: "23505"}
	if !IsUniqueViolation(err) {
		t.Fatal("expected true for unique violation")
	}

	if IsUniqueViolation(errors.New("plain error")) {
		t.Fatal("expected false for non pg error")
	}
}
