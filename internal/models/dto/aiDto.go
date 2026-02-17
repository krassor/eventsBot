package dto

import (
	"encoding/json"
	"eventsBot/internal/models/domain"
	"fmt"

	//"strconv"
	"strings"
	//"time"

	"github.com/google/uuid"
)

// FlexibleStringSlice — тип, который при десериализации принимает как строку, так и массив строк.
type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	// Пробуем как массив строк
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = arr
		return nil
	}

	// Пробуем как одну строку
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if s != "" {
			*f = []string{s}
		} else {
			*f = nil
		}
		return nil
	}

	return fmt.Errorf("tag: expected string or []string, got %s", string(data))
}

// EventStructuredResponseSchema - структура для получения данных о концертах и мероприятиях
type EventStructuredResponseSchema struct {
	Name string `json:"name" description:"Название мероприятия"`
	//Photo               string              `json:"photo" description:"Ссылка на фото мероприятия"`
	Description string `json:"description" description:"Описание мероприятия"`
	//Date                string              `json:"date" description:"Дата и время мероприятия"`
	//Price               string              `json:"price" description:"Цена билета на мероприятие"`
	//Currency            string              `json:"currency" description:"Валюта цены (например: EUR, USD, RUB)"`
	//EventLink           string              `json:"event_link" description:"Ссылка на страницу мероприятия"`
	MapLink      string              `json:"map_link" description:"Ссылка на местоположение на карте"`
	CalendarLink string              `json:"calendar_link" description:"Ссылка для добавления в календарь"`
	Tag          FlexibleStringSlice `json:"tag" description:"Теги мероприятия (например: концерт, выставка, фестиваль)"`
}

func (e EventStructuredResponseSchema) ToDomain() domain.Event {
	// Парсинг цены: убираем лишние пробелы и возможные символы валюты, если они попали в строку
	// priceStr := strings.TrimSpace(e.Price)
	// priceStr = strings.ReplaceAll(priceStr, ",", ".")
	// price, _ := strconv.ParseFloat(priceStr, 64)

	// Парсинг даты: пробуем несколько форматов
	// layouts := []string{
	// 	"02.01.2006 15:04",
	// 	"2006-01-02 15:04",
	// 	time.RFC3339,
	// }

	// var eventDate time.Time
	// for _, layout := range layouts {
	// 	if t, err := time.Parse(layout, e.Date); err == nil {
	// 		eventDate = t
	// 		break
	// 	}
	// }

	var tags strings.Builder
	for _, tag := range e.Tag {
		tag = strings.ReplaceAll(tag, " ", "")
		fmt.Fprintf(&tags, "#%s ", tag)
	}

	return domain.Event{
		ID:   uuid.New(),
		Name: e.Name,
		//Photo:               e.Photo,
		Description: e.Description,
		//Date:                eventDate,
		//Price:               price,
		//Currency:            e.Currency,
		//EventLink:           e.EventLink,
		MapLink:             e.MapLink,
		CalendarLinkAndroid: e.CalendarLink,
		Tag:                 tags.String(),
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
	if len(e.Tag) > 0 {
		var tags strings.Builder
		for _, tag := range e.Tag {
			tag = strings.ReplaceAll(tag, " ", "")
			fmt.Fprintf(&tags, "#%s ", tag)
		}
		event.Tag = tags.String()
	}

	// Обновляем ссылку на карту
	if strings.TrimSpace(e.MapLink) != "" {
		event.MapLink = e.MapLink
	}

	// Обновляем ссылки на календари
	if strings.TrimSpace(e.CalendarLink) != "" {
		event.CalendarLinkAndroid = e.CalendarLink
	}

	// Если AI обновил название, применяем
	if strings.TrimSpace(e.Name) != "" {
		event.Name = e.Name
	}

	return event
}
