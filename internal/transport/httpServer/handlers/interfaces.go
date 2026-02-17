package handlers

import (
	"context"
	"eventsBot/internal/models/domain"

	"github.com/google/uuid"
)

// EventRepository — интерфейс для работы с событиями из хэндлеров.
type EventRepository interface {
	UpdateEventStatus(ctx context.Context, eventID uuid.UUID, status string) error
	FindEventByID(ctx context.Context, eventID uuid.UUID) (domain.Event, error)
	ReadAllEvents(ctx context.Context) ([]domain.Event, error)
	FindEventsByStatus(ctx context.Context, status domain.EventStatus) ([]domain.Event, error)
	UpdateEvent(ctx context.Context, event domain.Event) (domain.Event, error)
}

type EventOrchestrator interface {
	SendEventToTelegram(event *domain.Event) error
}
