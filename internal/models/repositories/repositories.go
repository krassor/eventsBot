package repositories

import (
	"time"

	"github.com/google/uuid"
)

type BaseModel struct {
	ID        uuid.UUID `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Event struct {
	BaseModel
	Name                string    `db:"name"`
	Photo               string    `db:"photo"`
	Description         string    `db:"description"`
	Date                time.Time `db:"date"`
	Price               float64   `db:"price"`
	Currency            string    `db:"currency"`
	EventLink           string    `db:"event_link"`
	MapLink             string    `db:"map_link"`
	VideoURL            string    `db:"video_url"`
	CalendarLinkIOS     string    `db:"calendar_link_ios"`
	CalendarLinkAndroid string    `db:"calendar_link_android"`
	Tag                 string    `db:"tag"`
	Status              string    `db:"status"`
}
