package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	"github.com/jackc/pgx/v5"
)

type EventStore struct {
	exec Executor
}

func NewEventStore(db *DB) *EventStore {
	return &EventStore{exec: db.Pool}
}

func (s *EventStore) Create(ctx context.Context, event *domain.Event) error {
	query := `
		INSERT INTO events (
			id, organizer_id, name, total_inventory,
			price_kobo, status, sale_start, sale_end,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.exec.Exec(ctx, query,
		event.ID,
		event.OrganizerID,
		event.Name,
		event.TotalInventory,
		event.Price,
		event.Status,
		event.SaleStart,
		event.SaleEnd,
		event.CreatedAt,
		event.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}
	return nil
}

func (s *EventStore) GetByID(ctx context.Context, id string) (*domain.Event, error) {
	query := `
		SELECT id, organizer_id, name, total_inventory,
            price_kobo, status, sale_start, sale_end,
            created_at, updated_at
		FROM events
		WHERE id = $1
	`

	var e domain.Event
	err := s.exec.QueryRow(ctx, query, id).Scan(
		&e.ID,
		&e.OrganizerID,
		&e.Name,
		&e.TotalInventory,
		&e.Price,
		&e.Status,
		&e.SaleStart,
		&e.SaleEnd,
		&e.CreatedAt,
		&e.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("getting event: %w", err)
	}

	return &e, nil
}

func (s *EventStore) GetByStatus(ctx context.Context, status domain.EventStatus) ([]domain.Event, error) {
	query := `
		SELECT id, organizer_id, name, total_inventory,
		       price_kobo, status, sale_start, sale_end,
		       created_at, updated_at
		FROM events
		WHERE status = $1
	`
	rows, err := s.exec.Query(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("querying events by status: %w", err)
	}
	defer rows.Close()

	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Event, error) {
		var e domain.Event
		err := row.Scan(
			&e.ID, &e.OrganizerID, &e.Name, &e.TotalInventory,
			&e.Price, &e.Status, &e.SaleStart, &e.SaleEnd,
			&e.CreatedAt, &e.UpdatedAt,
		)
		return e, err
	})
}

func (s *EventStore) UpdateStatus(ctx context.Context, id string, status domain.EventStatus) error {
	query := `
        UPDATE events
        SET status = $1, updated_at = $2
        WHERE id = $3`

	result, err := s.exec.Exec(ctx, query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("updating event status: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrEventNotFound
	}
	return nil
}
