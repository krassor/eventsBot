package sites

import (
	"context"

	"app/main.go/internal/models/domain"
)

// ScrapeFunc — тип функции скрапера для конкретного сайта.
// Принимает контекст и URL, возвращает список событий.
type ScrapeFunc func(ctx context.Context, url string, shutdownChan <-chan struct{}) ([]domain.Event, error)
