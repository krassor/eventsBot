package orchestrator

import (
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

// Orchestrator управляет пайплайном: scraper → AI.
type Orchestrator struct {
	logger              *slog.Logger
	cfg                 *config.Config
	scraper             Scraper
	ai                  AI
	completedEventsChan <-chan domain.Event
	doneChans           []chan struct{}
	mu                  sync.Mutex
	shutdownChan        chan struct{}
}

// New создаёт новый экземпляр Orchestrator.
func New(logger *slog.Logger, cfg *config.Config, scraper Scraper, ai AI, completedEventsChan <-chan domain.Event) *Orchestrator {
	op := "Orchestrator.New()"
	log := logger.With(slog.String("op", op))
	log.Info("Creating orchestrator")

	return &Orchestrator{
		logger:              logger,
		cfg:                 cfg,
		scraper:             scraper,
		ai:                  ai,
		completedEventsChan: completedEventsChan,
		doneChans:           make([]chan struct{}, 0),
		shutdownChan:        make(chan struct{}),
	}
}

// Start запускает оркестратор.
// Запускает горутину для обработки завершённых событий скрапера.
func (o *Orchestrator) Start() {
	op := "Orchestrator.Start()"
	log := o.logger.With(slog.String("op", op))
	log.Info("orchestrator started")

	// Горутина слушает CompletedEventsChan от скрапера и отправляет в AI
	go o.processCompletedEvents()
}

// processCompletedEvents слушает канал завершённых событий и отправляет их в AI.
func (o *Orchestrator) processCompletedEvents() {
	op := "Orchestrator.processCompletedEvents()"
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

// WaitAll ожидает завершения всех добавленных джоб скрапера.
func (o *Orchestrator) WaitAll() {
	op := "Orchestrator.WaitAll()"
	log := o.logger.With(slog.String("op", op))

	o.mu.Lock()
	chans := make([]chan struct{}, len(o.doneChans))
	copy(chans, o.doneChans)
	o.mu.Unlock()

	log.Info("waiting for all scraper jobs", slog.Int("count", len(chans)))

	var wg sync.WaitGroup
	for _, doneChan := range chans {
		wg.Add(1)
		go func(ch chan struct{}) {
			defer wg.Done()
			<-ch
		}(doneChan)
	}
	wg.Wait()

	o.mu.Lock()
	o.doneChans = make([]chan struct{}, 0)
	o.mu.Unlock()

	log.Info("all scraper jobs completed")
}

// Shutdown корректно завершает оркестратор.
func (o *Orchestrator) Shutdown() {
	close(o.shutdownChan)
}
