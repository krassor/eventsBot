package scraper

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"app/main.go/internal/config"
	"app/main.go/internal/models/domain"
	"app/main.go/internal/scraper/sites"

	"github.com/google/uuid"
)

type Repository interface {
	CreateEvent(ctx context.Context, event domain.Event) (domain.Event, error)
	FindEventByLinkAndDate(ctx context.Context, link string, date time.Time) (domain.Event, error)
}

// Job представляет задачу, передаваемую в воркер.
type Job struct {
	requestID uuid.UUID     // Уникальный идентификатор запроса
	siteName  string        // Название сайта для выбора скрапера
	url       string        // URL для скрапинга
	Done      chan struct{} // Канал для сигнала завершения
}

// Scraper — структура, управляющая скрапингом событий.
type Scraper struct {
	logger              *slog.Logger
	cfg                 *config.Config
	repository          Repository
	scrapers            map[string]sites.ScrapeFunc // Регистр site-specific скраперов
	jobs                chan Job
	CompletedEventsChan chan domain.Event // Канал для завершённых событий (для передачи в AI)
	shutdownChannel     chan struct{}
	wg                  *sync.WaitGroup
}

// New создаёт новый экземпляр Scraper.
func New(
	logger *slog.Logger,
	cfg *config.Config,
	repository Repository,
) *Scraper {
	op := "Scraper.New()"
	log := logger.With(
		slog.String("op", op),
	)

	log.Info("Creating scraper client")

	s := &Scraper{
		logger:              logger,
		cfg:                 cfg,
		repository:          repository,
		scrapers:            make(map[string]sites.ScrapeFunc),
		jobs:                make(chan Job, cfg.ScraperConfig.JobBufferSize),
		CompletedEventsChan: make(chan domain.Event, 100),
		shutdownChannel:     make(chan struct{}),
		wg:                  &sync.WaitGroup{},
	}

	// Регистрация скраперов
	s.scrapers["lococlub"] = sites.ScrapeLococlub

	return s
}

// Start запускает воркеры для обработки задач.
func (s *Scraper) Start() {
	op := "Scraper.Start()"
	log := s.logger.With(
		slog.String("op", op),
	)
	for i := 0; i < s.cfg.ScraperConfig.WorkersCount; i++ {
		s.wg.Add(1)
		go s.handleJob(i)
	}
	log.Info("scraper service started", slog.Int("workers", s.cfg.ScraperConfig.WorkersCount))

	s.wg.Wait()
}

// AddJob добавляет новую задачу в очередь на обработку.
func (s *Scraper) AddJob(requestID uuid.UUID, siteName string, url string) (chan struct{}, error) {
	newJob := Job{
		requestID: requestID,
		siteName:  siteName,
		url:       url,
		Done:      make(chan struct{}),
	}
	select {
	case <-s.shutdownChannel:
		return nil, fmt.Errorf("service is shutting down")
	default:
		if len(s.jobs) < s.cfg.ScraperConfig.JobBufferSize {
			s.jobs <- newJob
			return newJob.Done, nil
		} else {
			return nil, fmt.Errorf("job buffer is full")
		}
	}
}

// handleJob — воркер, обрабатывающий задачи из канала.
func (s *Scraper) handleJob(id int) {
	defer s.wg.Done()
	op := "Scraper.handleJob()"
	log := s.logger.With(
		slog.String("op", op),
		slog.Int("workerId", id),
	)

	log.Info("start scraper job handler")

	for {
		select {
		case <-s.shutdownChannel:
			return
		case job, ok := <-s.jobs:
			if !ok {
				log.Error("jobs channel closed")
				return
			}

			joblog := log.With(
				slog.String("requestID", job.requestID.String()),
				slog.String("siteName", job.siteName),
			)

			// Получаем скрапер по имени сайта
			scrapeFunc, exists := s.scrapers[job.siteName]
			if !exists {
				joblog.Error("scraper not found for site", slog.String("siteName", job.siteName))
				close(job.Done)
				continue
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.ScraperConfig.Timeout)*time.Second)

			events, err := scrapeFunc(ctx, job.url, s.shutdownChannel)
			cancel()

			if err != nil {
				joblog.Error("scraping failed", slog.String("error", err.Error()))
				close(job.Done)
				continue
			}

			// Обрабатываем каждое событие
			for _, event := range events {
				// Проверяем, есть ли уже такое событие в БД
				existing, err := s.repository.FindEventByLinkAndDate(ctx, event.EventLink, event.Date)
				if err == nil && existing.ID != uuid.Nil {
					joblog.Debug("event already exists", slog.String("link", event.EventLink))
					continue
				}

				// Сохраняем событие со статусом NEW
				event.ID = uuid.New()
				event.Status = domain.EventStatusNew
				savedEvent, err := s.repository.CreateEvent(ctx, event)
				if err != nil {
					joblog.Error("failed to create event", slog.String("error", err.Error()))
					continue
				}

				joblog.Debug("event created", slog.String("name", savedEvent.Name))

				// Отправляем в канал для обработки AI
				select {
				case s.CompletedEventsChan <- savedEvent:
				default:
					joblog.Warn("CompletedEventsChan is full, skipping AI enrichment")
				}
			}

			close(job.Done)

			joblog.Info("scraping completed", slog.Int("eventsCount", len(events)))
		}
	}
}

// Shutdown корректно завершает работу сервиса.
func (s *Scraper) Shutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("force exit scraper: %w", ctx.Err())
	default:
		close(s.shutdownChannel)
		close(s.jobs)
		close(s.CompletedEventsChan)
		return nil
	}
}
