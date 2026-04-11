package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/DanielPopoola/fairqueue/internal/domain"
	postgres "github.com/DanielPopoola/fairqueue/internal/store/postgres"
)

// EventService handles event lifecycle operations.
// Organizer ownership is enforced here — handlers pass the organizer ID
// from context and this service validates it before any mutation.
type EventService struct {
	events *postgres.EventStore
	logger *slog.Logger
}

func NewEventService(events *postgres.EventStore, logger *slog.Logger) *EventService {
	return &EventService{events: events, logger: logger}
}

// Create validates input, converts NGN to kobo, and persists a new DRAFT event.
func (s *EventService) Create(ctx context.Context, organizerID string, req CreateEventRequest) (*domain.Event, error) {
	if req.TotalInventory < 1 {
		return nil, fmt.Errorf("total_inventory must be at least 1: %w", domain.ErrInvalidInput)
	}
	if req.PriceNGN < 1 {
		return nil, fmt.Errorf("price must be at least 1 NGN: %w", domain.ErrInvalidInput)
	}
	if !req.SaleEnd.After(req.SaleStart) {
		return nil, fmt.Errorf("sale_end must be after sale_start: %w", domain.ErrInvalidInput)
	}

	now := time.Now()
	event := &domain.Event{
		ID:             uuid.NewString(),
		OrganizerID:    organizerID,
		Name:           req.Name,
		TotalInventory: req.TotalInventory,
		Price:          req.PriceNGN * koboPerNaira, // NGN → kobo
		Status:         domain.EventStatusDraft,
		SaleStart:      req.SaleStart,
		SaleEnd:        req.SaleEnd,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.events.Create(ctx, event); err != nil {
		return nil, fmt.Errorf("creating event: %w", err)
	}

	return event, nil
}

// Activate transitions an event from DRAFT to ACTIVE.
// Enforces organizer ownership — only the creating organizer can activate.
func (s *EventService) Activate(ctx context.Context, organizerID, eventID string) (*domain.Event, error) {
	event, err := s.loadAndAuthorize(ctx, organizerID, eventID)
	if err != nil {
		return nil, err
	}

	if err := event.Activate(); err != nil {
		return nil, err // ErrInvalidTransition
	}

	if err := s.events.UpdateStatus(ctx, event.ID, domain.EventStatusActive); err != nil {
		return nil, fmt.Errorf("updating event status: %w", err)
	}

	s.logger.Info("event activated",
		"event_id", event.ID,
		"organizer_id", organizerID,
	)
	return event, nil
}

// End transitions an event from ACTIVE or SOLD_OUT to ENDED.
// Enforces organizer ownership.
func (s *EventService) End(ctx context.Context, organizerID, eventID string) (*domain.Event, error) {
	event, err := s.loadAndAuthorize(ctx, organizerID, eventID)
	if err != nil {
		return nil, err
	}

	if err := event.End(); err != nil {
		return nil, err // ErrInvalidTransition
	}

	if err := s.events.UpdateStatus(ctx, event.ID, domain.EventStatusEnded); err != nil {
		return nil, fmt.Errorf("updating event status: %w", err)
	}

	s.logger.Info("event ended",
		"event_id", event.ID,
		"organizer_id", organizerID,
	)
	return event, nil
}

// Get returns an event by ID. No auth check — events are public.
func (s *EventService) Get(ctx context.Context, eventID string) (*domain.Event, error) {
	return s.events.GetByID(ctx, eventID)
}

// loadAndAuthorize fetches the event and confirms the organizer owns it.
// Returns ErrForbidden if the organizer ID doesn't match.
func (s *EventService) loadAndAuthorize(ctx context.Context, organizerID, eventID string) (*domain.Event, error) {
	event, err := s.events.GetByID(ctx, eventID)
	if err != nil {
		if errors.Is(err, domain.ErrEventNotFound) {
			return nil, domain.ErrEventNotFound
		}
		return nil, fmt.Errorf("loading event: %w", err)
	}

	if event.OrganizerID != organizerID {
		return nil, domain.ErrForbidden
	}

	return event, nil
}

// koboPerNaira is the conversion factor. Defined here to keep the service
// layer self-contained — the API layer uses its own copy in helpers.go.
const koboPerNaira int64 = 100

// CreateEventRequest is the input t    o EventService.Create.
// Separate from the API request type so the service layer
// has no dependency on generated API types.
type CreateEventRequest struct {
	Name           string
	TotalInventory int
	PriceNGN       int64
	SaleStart      time.Time
	SaleEnd        time.Time
}
