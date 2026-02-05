package dto

import (
	"app/main.go/internal/models/domain"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EventStructuredResponseSchema - структура для получения данных о концертах и мероприятиях
type EventStructuredResponseSchema struct {
	Name                string `json:"name" description:"Название мероприятия"`
	Photo               string `json:"photo" description:"Ссылка на фото мероприятия"`
	Description         string `json:"description" description:"Описание мероприятия"`
	Date                string `json:"date" description:"Дата и время мероприятия"`
	Price               string `json:"price" description:"Цена билета на мероприятие"`
	Currency            string `json:"currency" description:"Валюта цены (например: EUR, USD, RUB)"`
	EventLink           string `json:"event_link" description:"Ссылка на страницу мероприятия"`
	MapLink             string `json:"map_link" description:"Ссылка на местоположение на карте"`
	CalendarLinkIOS     string `json:"calendar_link_ios" description:"Ссылка для добавления в календарь iPhone"`
	CalendarLinkAndroid string `json:"calendar_link_android" description:"Ссылка для добавления в календарь Android"`
	Tag                 string `json:"tag" description:"Тег мероприятия (например: концерт, выставка, фестиваль)"`
}

func (e EventStructuredResponseSchema) ToDomain() domain.Event {
	// Парсинг цены: убираем лишние пробелы и возможные символы валюты, если они попали в строку
	priceStr := strings.TrimSpace(e.Price)
	priceStr = strings.ReplaceAll(priceStr, ",", ".")
	price, _ := strconv.ParseFloat(priceStr, 64)

	// Парсинг даты: пробуем несколько форматов
	layouts := []string{
		"02.01.2006 15:04",
		"2006-01-02 15:04",
		time.RFC3339,
	}

	var eventDate time.Time
	for _, layout := range layouts {
		if t, err := time.Parse(layout, e.Date); err == nil {
			eventDate = t
			break
		}
	}

	return domain.Event{
		ID:                  uuid.New(),
		Name:                e.Name,
		Photo:               e.Photo,
		Description:         e.Description,
		Date:                eventDate,
		Price:               price,
		Currency:            e.Currency,
		EventLink:           e.EventLink,
		MapLink:             e.MapLink,
		CalendarLinkIOS:     e.CalendarLinkIOS,
		CalendarLinkAndroid: e.CalendarLinkAndroid,
		Tag:                 e.Tag,
	}
}

// ApplyToEvent применяет данные из AI-ответа к существующему событию.
// Обновляет только те поля, которые AI вернул непустыми.
func (e EventStructuredResponseSchema) ApplyToEvent(event domain.Event) domain.Event {
	// Обновляем описание, если AI вернул непустое
	if strings.TrimSpace(e.Description) != "" {
		event.Description = e.Description
	}

	// Обновляем тег
	if strings.TrimSpace(e.Tag) != "" {
		event.Tag = e.Tag
	}

	// Обновляем ссылку на карту
	if strings.TrimSpace(e.MapLink) != "" {
		event.MapLink = e.MapLink
	}

	// Обновляем ссылки на календари
	if strings.TrimSpace(e.CalendarLinkIOS) != "" {
		event.CalendarLinkIOS = e.CalendarLinkIOS
	}
	if strings.TrimSpace(e.CalendarLinkAndroid) != "" {
		event.CalendarLinkAndroid = e.CalendarLinkAndroid
	}

	// Если AI обновил название, применяем
	if strings.TrimSpace(e.Name) != "" {
		event.Name = e.Name
	}

	return event
}
