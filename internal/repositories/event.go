package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"app/main.go/internal/models/domain"
	"app/main.go/internal/models/repositories"

	"github.com/google/uuid"
)

func (r *Repository) CreateEvent(ctx context.Context, event domain.Event) (domain.Event, error) {
	op := "repository.CreateEvent()"

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}

	repoEvent := mapToRepo(event)

	insertQuery := `INSERT INTO events (
		id, name, photo, description, date, price, currency, 
		event_link, map_link, calendar_link_ios, calendar_link_android, tag, status,
		created_at, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	_, err := r.DB.ExecContext(ctx, insertQuery,
		repoEvent.ID,
		repoEvent.Name,
		repoEvent.Photo,
		repoEvent.Description,
		repoEvent.Date,
		repoEvent.Price,
		repoEvent.Currency,
		repoEvent.EventLink,
		repoEvent.MapLink,
		repoEvent.CalendarLinkIOS,
		repoEvent.CalendarLinkAndroid,
		repoEvent.Tag,
		repoEvent.Status,
	)
	if err != nil {
		return domain.Event{}, fmt.Errorf("%s: %w", op, err)
	}

	return event, nil
}

func (r *Repository) FindEventByID(ctx context.Context, id uuid.UUID) (domain.Event, error) {
	var repoEvent repositories.Event
	query := `SELECT id, name, photo, description, date, price, currency, event_link, map_link, calendar_link_ios, calendar_link_android, tag, status, created_at, updated_at 
	          FROM events WHERE id = $1 LIMIT 1`

	err := r.DB.GetContext(ctx, &repoEvent, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Event{}, fmt.Errorf("event not found with id: %s", id)
		}
		return domain.Event{}, fmt.Errorf("error in FindEventByID(): %w", err)
	}

	return mapToDomain(repoEvent), nil
}

func (r *Repository) FindEventByLinkAndDate(ctx context.Context, link string, date time.Time) (domain.Event, error) {
	var repoEvent repositories.Event
	query := `SELECT id, name, photo, description, date, price, currency, event_link, map_link, calendar_link_ios, calendar_link_android, tag, status, created_at, updated_at 
	          FROM events WHERE event_link = $1 AND date = $2 LIMIT 1`

	err := r.DB.GetContext(ctx, &repoEvent, query, link, date)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Event{}, fmt.Errorf("event not found with link: %s and date: %s", link, date)
		}
		return domain.Event{}, fmt.Errorf("error in FindEventByLinkAndDate(): %w", err)
	}

	return mapToDomain(repoEvent), nil
}

func (r *Repository) UpdateEvent(ctx context.Context, event domain.Event) (domain.Event, error) {
	repoEvent := mapToRepo(event)

	updateQuery := `UPDATE events SET 
		name = $1, photo = $2, description = $3, date = $4, price = $5, currency = $6, 
		event_link = $7, map_link = $8, calendar_link_ios = $9, calendar_link_android = $10, tag = $11, status = $12,
		updated_at = CURRENT_TIMESTAMP
		WHERE id = $13`

	result, err := r.DB.ExecContext(ctx, updateQuery,
		repoEvent.Name,
		repoEvent.Photo,
		repoEvent.Description,
		repoEvent.Date,
		repoEvent.Price,
		repoEvent.Currency,
		repoEvent.EventLink,
		repoEvent.MapLink,
		repoEvent.CalendarLinkIOS,
		repoEvent.CalendarLinkAndroid,
		repoEvent.Tag,
		repoEvent.Status,
		repoEvent.ID,
	)
	if err != nil {
		return domain.Event{}, fmt.Errorf("error in UpdateEvent(): %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return domain.Event{}, fmt.Errorf("error checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return domain.Event{}, fmt.Errorf("event with id %s not found", event.ID)
	}

	return event, nil
}

func (r *Repository) DeleteEvent(ctx context.Context, id uuid.UUID) error {
	deleteQuery := `DELETE FROM events WHERE id = $1`

	result, err := r.DB.ExecContext(ctx, deleteQuery, id)
	if err != nil {
		return fmt.Errorf("error in DeleteEvent(): %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error checking rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event with id %s not found", id)
	}

	return nil
}

func (r *Repository) ListEvents(ctx context.Context) ([]domain.Event, error) {
	var repoEvents []repositories.Event
	query := `SELECT id, name, photo, description, date, price, currency, event_link, map_link, calendar_link_ios, calendar_link_android, tag, status, created_at, updated_at 
	          FROM events ORDER BY date ASC`

	err := r.DB.SelectContext(ctx, &repoEvents, query)
	if err != nil {
		return nil, fmt.Errorf("error in ListEvents(): %w", err)
	}

	result := make([]domain.Event, len(repoEvents))
	for i, e := range repoEvents {
		result[i] = mapToDomain(e)
	}

	return result, nil
}

func mapToRepo(e domain.Event) repositories.Event {
	return repositories.Event{
		BaseModel: repositories.BaseModel{
			ID: e.ID,
		},
		Name:                e.Name,
		Photo:               e.Photo,
		Description:         e.Description,
		Date:                e.Date,
		Price:               e.Price,
		Currency:            e.Currency,
		EventLink:           e.EventLink,
		MapLink:             e.MapLink,
		CalendarLinkIOS:     e.CalendarLinkIOS,
		CalendarLinkAndroid: e.CalendarLinkAndroid,
		Tag:                 e.Tag,
		Status:              string(e.Status),
	}
}

func mapToDomain(e repositories.Event) domain.Event {
	return domain.Event{
		ID:                  e.ID,
		Name:                e.Name,
		Photo:               e.Photo,
		Description:         e.Description,
		Date:                e.Date,
		Price:               e.Price,
		Currency:            e.Currency,
		EventLink:           e.EventLink,
		MapLink:             e.MapLink,
		CalendarLinkIOS:     e.CalendarLinkIOS,
		CalendarLinkAndroid: e.CalendarLinkAndroid,
		Tag:                 e.Tag,
		Status:              domain.EventStatus(e.Status),
	}
}
