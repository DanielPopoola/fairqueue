package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/jackc/pgx/v5"
)

type QueueStore struct {
	db *DB
}

func NewQueueStore(db *DB) *QueueStore {
	return &QueueStore{db: db}
}

func (s *QueueStore) Create(ctx context.Context, entry *domain.QueueEntry) error {
	query := `
        INSERT INTO queue_entries (
            id, event_id, customer_id, status,
            joined_at, admitted_at, updated_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.Pool.Exec(ctx, query,
		entry.ID,
		entry.EventID,
		entry.CustomerID,
		entry.Status,
		entry.JoinedAt,
		entry.AdmittedAt,
		entry.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting queue entry: %w", err)
	}
	return nil
}

func (s *QueueStore) GetByID(ctx context.Context, id string) (*domain.QueueEntry, error) {
	query := `
        SELECT id, event_id, customer_id, status,
               joined_at, admitted_at, updated_at
        FROM queue_entries
        WHERE id = $1`

	var q domain.QueueEntry
	err := s.db.Pool.QueryRow(ctx, query, id).Scan(
		&q.ID,
		&q.EventID,
		&q.CustomerID,
		&q.Status,
		&q.JoinedAt,
		&q.AdmittedAt,
		&q.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrQueueEntryNotFound
		}
		return nil, fmt.Errorf("getting queue entry: %w", err)
	}
	return &q, nil
}

func (s *QueueStore) GetByCustomerAndEvent(ctx context.Context, customerID, eventID string) (*domain.QueueEntry, error) {
	query := `
        SELECT id, event_id, customer_id, status,
               joined_at, admitted_at, updated_at
        FROM queue_entries
        WHERE customer_id = $1
        AND event_id = $2
        AND status IN ('WAITING', 'ADMITTED')`

	var q domain.QueueEntry
	err := s.db.Pool.QueryRow(ctx, query, customerID, eventID).Scan(
		&q.ID,
		&q.EventID,
		&q.CustomerID,
		&q.Status,
		&q.JoinedAt,
		&q.AdmittedAt,
		&q.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrQueueEntryNotFound
		}
		return nil, fmt.Errorf("getting queue entry by customer and event: %w", err)
	}
	return &q, nil
}

func (s *QueueStore) GetActiveByEvent(ctx context.Context, eventID string) ([]domain.QueueEntry, error) {
	query := `
        SELECT id, event_id, customer_id, status,
               joined_at, admitted_at, updated_at
        FROM queue_entries
        WHERE event_id = $1
        AND status IN ('WAITING', 'ADMITTED')
        ORDER BY joined_at ASC`

	rows, err := s.db.Pool.Query(ctx, query, eventID)
	if err != nil {
		return nil, fmt.Errorf("querying active queue entries: %w", err)
	}
	defer rows.Close()

	return scanQueueEntries(rows)
}

func (s *QueueStore) GetStaleEntries(
	ctx context.Context,
	waitingOlderThan time.Duration,
	admittedOlderThan time.Duration,
) ([]domain.QueueEntry, error) {
	query := `
        SELECT id, event_id, customer_id, status,
               joined_at, admitted_at, updated_at
        FROM queue_entries
        WHERE (
            status = 'WAITING'
            AND joined_at < $1
        ) OR (
            status = 'ADMITTED'
            AND admitted_at < $2
        )`

	rows, err := s.db.Pool.Query(ctx, query,
		time.Now().Add(-waitingOlderThan),
		time.Now().Add(-admittedOlderThan),
	)
	if err != nil {
		return nil, fmt.Errorf("querying stale queue entries: %w", err)
	}
	defer rows.Close()

	return scanQueueEntries(rows)
}

func (s *QueueStore) UpdateStatus(
	ctx context.Context,
	id string,
	newStatus domain.QueueEntryStatus,
	expectedStatus domain.QueueEntryStatus,
) error {
	query := `
        UPDATE queue_entries
        SET status = $1, updated_at = $2
        WHERE id = $3
        AND status = $4`

	result, err := s.db.Pool.Exec(ctx, query,
		newStatus,
		time.Now(),
		id,
		expectedStatus,
	)
	if err != nil {
		return fmt.Errorf("updating queue entry status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrQueueEntryNotFound
	}
	return nil
}

func (s *QueueStore) MarkAdmitted(ctx context.Context, id string) error {
	query := `
        UPDATE queue_entries
        SET status = 'ADMITTED',
            admitted_at = $1,
            updated_at = $1
        WHERE id = $2
        AND status = 'WAITING'`

	result, err := s.db.Pool.Exec(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("marking queue entry admitted: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrQueueEntryNotFound
	}
	return nil
}

// MarkAdmittedBatch updates all queue entries for the given
// customerIDs to ADMITTED in a single query.
func (s *QueueStore) MarkAdmittedBatch(ctx context.Context, eventID string, customerIDs []string) error {
	query := `
		UPDATE queue_entries
		SET status = 'ADMITTED',
		    admitted_at = $1,
		    updated_at  = $1
		WHERE customer_id = ANY($2)
		AND   event_id    = $3
		AND   status      = 'WAITING'
	`

	_, err := s.db.Pool.Exec(ctx, query, time.Now(), customerIDs, eventID)
	if err != nil {
		return fmt.Errorf("batch admitting queue entries: %w", err)
	}
	return nil
}

func (s *QueueStore) MarkExpiredBatch(ctx context.Context, eventID string, evicted []string) error {
	query := `
		UPDATE queue_entries
		SET status = 'EXPIRED',
			updated_at = $1
		WHERE customer_id = ANY($2)
		AND event_id = $3
		AND status = 'ADMITTED'
	`

	_, err := s.db.Pool.Exec(ctx, query, time.Now(), evicted, eventID)
	if err != nil {
		return fmt.Errorf("batch evicting admitted queue entries: %w", err)
	}
	return nil
}

func scanQueueEntries(rows pgx.Rows) ([]domain.QueueEntry, error) {
	results, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.QueueEntry, error) {
		var q domain.QueueEntry
		err := row.Scan(
			&q.ID, &q.EventID, &q.CustomerID, &q.Status, &q.JoinedAt, &q.AdmittedAt, &q.UpdatedAt,
		)
		return q, err
	})

	if err != nil {
		return nil, err
	}

	return results, nil
}
