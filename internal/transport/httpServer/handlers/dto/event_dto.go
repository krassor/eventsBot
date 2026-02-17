package dto

import (
	"time"

	"eventsBot/internal/models/domain"

	"github.com/google/uuid"
)

// EventResponse — DTO для ответа с данными события.
type EventResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Photo               string    `json:"photo"`
	Description         string    `json:"description"`
	Date                time.Time `json:"date"`
	Price               float64   `json:"price"`
	Currency            string    `json:"currency"`
	EventLink           string    `json:"event_link"`
	MapLink             string    `json:"map_link"`
	VideoURL            string    `json:"video_url"`
	CalendarLinkIOS     string    `json:"calendar_link_ios"`
	CalendarLinkAndroid string    `json:"calendar_link_android"`
	Tag                 string    `json:"tag"`
	Status              string    `json:"status"`
}

// ChangeEventRequest — DTO для запроса на полное обновление события.
type ChangeEventRequest struct {
	Name                string    `json:"name"`
	Photo               string    `json:"photo"`
	Description         string    `json:"description"`
	Date                time.Time `json:"date"`
	Price               float64   `json:"price"`
	Currency            string    `json:"currency"`
	EventLink           string    `json:"event_link"`
	MapLink             string    `json:"map_link"`
	VideoURL            string    `json:"video_url"`
	CalendarLinkIOS     string    `json:"calendar_link_ios"`
	CalendarLinkAndroid string    `json:"calendar_link_android"`
	Tag                 string    `json:"tag"`
	Status              string    `json:"status"`
}

// UpdateStatusRequest — DTO для запроса на изменение статуса события.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// MapDomainToEventResponse конвертирует доменную модель Event в EventResponse DTO.
func MapDomainToEventResponse(e domain.Event) EventResponse {
	return EventResponse{
		ID:                  e.ID,
		Name:                e.Name,
		Photo:               e.Photo,
		Description:         e.Description,
		Date:                e.Date,
		Price:               e.Price,
		Currency:            e.Currency,
		EventLink:           e.EventLink,
		MapLink:             e.MapLink,
		VideoURL:            e.VideoURL,
		CalendarLinkIOS:     e.CalendarLinkIOS,
		CalendarLinkAndroid: e.CalendarLinkAndroid,
		Tag:                 e.Tag,
		Status:              string(e.Status),
	}
}

// MapDomainToEventResponseList конвертирует слайс доменных моделей в слайс DTO.
func MapDomainToEventResponseList(events []domain.Event) []EventResponse {
	result := make([]EventResponse, len(events))
	for i, e := range events {
		result[i] = MapDomainToEventResponse(e)
	}
	return result
}

// MapEventRequestToDomain конвертирует ChangeEventRequest DTO в доменную модель Event.
func MapEventRequestToDomain(req ChangeEventRequest, id uuid.UUID) domain.Event {
	return domain.Event{
		ID:                  id,
		Name:                req.Name,
		Photo:               req.Photo,
		Description:         req.Description,
		Date:                req.Date,
		Price:               req.Price,
		Currency:            req.Currency,
		EventLink:           req.EventLink,
		MapLink:             req.MapLink,
		VideoURL:            req.VideoURL,
		CalendarLinkIOS:     req.CalendarLinkIOS,
		CalendarLinkAndroid: req.CalendarLinkAndroid,
		Tag:                 req.Tag,
		Status:              domain.EventStatus(req.Status),
	}
}
