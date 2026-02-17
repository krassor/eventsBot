package sites

import (
	"context"

	"eventsBot/internal/models/domain"
)

// ScrapeFunc — тип функции скрапера для конкретного сайта.
// Принимает контекст и URL, возвращает список событий.
type ScrapeFunc func(ctx context.Context, url string, shutdownChan <-chan struct{}) ([]domain.Event, error)
