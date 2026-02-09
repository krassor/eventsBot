package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"app/main.go/internal/config"
	"app/main.go/internal/models/domain"

	"github.com/google/uuid"
)

// Scraper определяет интерфейс для взаимодействия со скрапером.
type Scraper interface {
	AddJob(requestID uuid.UUID, siteName string, url string) (chan struct{}, error)
}

// AI определяет интерфейс для взаимодействия с AI сервисом.
type AI interface {
	AddJob(requestID uuid.UUID, event domain.Event) (chan struct{}, error)
}

// Repository определяет интерфейс для взаимодействия с хранилищем данных.
type Repository interface {
	FindEventsByStatus(ctx context.Context, status domain.EventStatus) ([]domain.Event, error)
}

// TelegramBot определяет интерфейс для взаимодействия с Telegram ботом.
type TelegramBot interface {
	SendEvent(event *domain.Event, channelIDs []int64) error
}

// Orchestrator управляет пайплайном: scraper → AI.
type Orchestrator struct {
	logger              *slog.Logger
	cfg                 *config.Config
	scraper             Scraper
	ai                  AI
	repository          Repository
	telegramBot         TelegramBot
	completedEventsChan <-chan domain.Event
	doneChans           []chan struct{}
	mu                  sync.Mutex
	shutdownChan        chan struct{}
}

// New создаёт новый экземпляр Orchestrator.
func New(logger *slog.Logger, cfg *config.Config, scraper Scraper, ai AI, repository Repository, telegramBot TelegramBot, completedEventsChan <-chan domain.Event) *Orchestrator {
	op := "Orchestrator.New()"
	log := logger.With(slog.String("op", op))
	log.Info("Creating orchestrator")

	return &Orchestrator{
		logger:              logger,
		cfg:                 cfg,
		scraper:             scraper,
		ai:                  ai,
		repository:          repository,
		telegramBot:         telegramBot,
		completedEventsChan: completedEventsChan,
		doneChans:           make([]chan struct{}, 0),
		shutdownChan:        make(chan struct{}),
	}
}

// Start запускает оркестратор.
// Запускает горутину для обработки завершённых событий скрапера.
// Также добавляет все сайты из конфигурации в очередь скрапера.
func (o *Orchestrator) Start() {
	op := "Orchestrator.Start()"
	log := o.logger.With(slog.String("op", op))
	log.Info("orchestrator started")

	// Горутина слушает CompletedEventsChan от скрапера и отправляет в AI
	go o.processScrapedEvents()

	// Горутина слушает NewEventsChan от репозитория и отправляет в AI
	go o.processNewEvents()

	events, err := o.repository.FindEventsByStatus(context.Background(), domain.EventStatusReadyToApprove)
	if err != nil {
		log.Error("failed to find events", slog.String("error", err.Error()))
		return
	}

	for _, event := range events {
		o.telegramBot.SendEvent(&event, o.cfg.BotConfig.ChannelIDs)
	}

	// Добавляем сайты из конфигурации в очередь скрапера
	if err := o.EnqueueSites(); err != nil {
		log.Error("failed to enqueue sites", slog.String("error", err.Error()))
	}
}

// processNewEvents ищет все события в статусе NEW в репозитории и отправляет в AI
func (o *Orchestrator) processNewEvents() {
	op := "Orchestrator.processNewEvents()"
	log := o.logger.With(slog.String("op", op))

	events, err := o.repository.FindEventsByStatus(context.Background(), domain.EventStatusNew)
	if err != nil {
		log.Error("failed to find new events", slog.String("error", err.Error()))
		return
	}

	for _, event := range events {
		_, err := o.ai.AddJob(uuid.New(), event)
		if err != nil {
			log.Error("failed to add AI job", slog.String("error", err.Error()))
			continue
		}
		log.Debug("event sent to AI", slog.String("name", event.Name))
	}
}

// processCompletedEvents слушает канал завершённых событий и отправляет их в AI.
func (o *Orchestrator) processScrapedEvents() {
	op := "Orchestrator.processScrapedEvents()"
	log := o.logger.With(slog.String("op", op))

	for {
		select {
		case <-o.shutdownChan:
			log.Info("processCompletedEvents shutting down")
			return
		case event, ok := <-o.completedEventsChan:
			if !ok {
				log.Info("completedEventsChan closed")
				return
			}

			log.Debug("received completed event", slog.String("name", event.Name))

			// Отправляем событие в AI для обогащения
			_, err := o.ai.AddJob(uuid.New(), event)
			if err != nil {
				log.Error("failed to add AI job", slog.String("error", err.Error()))
				continue
			}

			log.Debug("event sent to AI", slog.String("name", event.Name))
		}
	}
}

// AddJob добавляет джобу в скрапер и сохраняет канал Done для ожидания.
func (o *Orchestrator) AddJob(siteName string, url string) error {
	op := "Orchestrator.AddJob()"
	log := o.logger.With(slog.String("op", op))

	requestID := uuid.New()
	doneChan, err := o.scraper.AddJob(requestID, siteName, url)
	if err != nil {
		log.Error("failed to add job",
			slog.String("siteName", siteName),
			slog.String("url", url),
			slog.String("error", err.Error()),
		)
		return err
	}

	o.mu.Lock()
	o.doneChans = append(o.doneChans, doneChan)
	o.mu.Unlock()

	log.Debug("job added",
		slog.String("requestID", requestID.String()),
		slog.String("siteName", siteName),
	)

	return nil
}

// EnqueueSites добавляет все сайты из конфигурации в очередь скрапера.
func (o *Orchestrator) EnqueueSites() error {
	op := "Orchestrator.EnqueueSites()"
	log := o.logger.With(slog.String("op", op))

	sites := o.cfg.ScraperConfig.Sites
	if len(sites) == 0 {
		log.Warn("no sites configured for scraping")
		return nil
	}

	log.Info("enqueueing sites for scraping", slog.Int("count", len(sites)))

	for _, site := range sites {
		if err := o.AddJob(site.Name, site.URL); err != nil {
			log.Error("failed to enqueue site",
				slog.String("name", site.Name),
				slog.String("url", site.URL),
				slog.String("error", err.Error()),
			)
			// Продолжаем добавлять остальные сайты
			continue
		}
		log.Debug("site enqueued", slog.String("name", site.Name), slog.String("url", site.URL))
	}

	return nil
}

// Shutdown корректно завершает оркестратор.
func (o *Orchestrator) Shutdown(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("exit tgBot: %w", ctx.Err())
		default:
			close(o.shutdownChan)
			return nil
		}
	}
}
