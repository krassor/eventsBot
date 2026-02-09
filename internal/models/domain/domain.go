package domain

import (
	"time"

	"github.com/google/uuid"
)

// EventStatus представляет статус события в пайплайне обработки
type EventStatus string

const (
	// EventStatusNew — событие создано после скрапинга
	EventStatusNew EventStatus = "NEW"
	// EventStatusAIEnriched — событие обогащено AI
	EventStatusAIEnriched EventStatus = "AI_ENRICHED"
	// EventStatusReadyToApprove — событие готово к модерации
	EventStatusReadyToApprove EventStatus = "READY_TO_APPROVE"
	// EventStatusApproved — событие одобрено
	EventStatusApproved EventStatus = "APPROVED"
	// EventStatusRejected — событие отклонено
	EventStatusRejected EventStatus = "REJECTED"
)

// Event - доменная модель мероприятия
type Event struct {
	ID                  uuid.UUID
	Name                string
	Photo               string
	Description         string
	Date                time.Time
	Price               float64
	Currency            string
	EventLink           string
	MapLink             string
	VideoURL            string
	CalendarLinkIOS     string
	CalendarLinkAndroid string
	Tag                 string
	Status              EventStatus
}

type EventURL string

func (e EventURL) String() string {
	return string(e)
}

func ToEventURL(url string) EventURL {
	return EventURL(url)
}
