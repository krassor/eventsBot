package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"eventsBot/internal/config"
	"eventsBot/internal/models/domain"
	"eventsBot/internal/models/dto"
	"eventsBot/internal/utils/logger/sl"

	"github.com/google/uuid"
	openrouter "github.com/revrost/go-openrouter"
	"github.com/revrost/go-openrouter/jsonschema"
)

const (
	// retryCount определяет количество попыток повторного запроса при ошибках.
	retryCount int = 10
	// retryDuration задаёт интервал между попытками повторного запроса.
	retryDuration time.Duration = 5 * time.Second
)

type Repository interface {
	UpdateEvent(ctx context.Context, event domain.Event) (domain.Event, error)
}

// Job представляет задачу, передаваемую в воркер.
type Job struct {
	requestID uuid.UUID     // Уникальный идентификатор запроса
	event     domain.Event  // Событие для обогащения AI
	Done      chan struct{} // Канал для сигнала завершения
}

// Openrouter — структура, управляющая взаимодействием с OpenRouter API.
// Содержит пул воркеров для асинхронной обработки запросов.
type Openrouter struct {
	logger          *slog.Logger       // Логгер с контекстом
	cfg             *config.Config     // Конфигурация приложения
	Client          *openrouter.Client // Клиент OpenRouter API
	repository      Repository
	jobs            chan Job        // Канал задач
	shutdownChannel chan struct{}   // Канал для сигнала завершения
	wg              *sync.WaitGroup // Группа для ожидания завершения воркеров
}

// NewClient создаёт новый экземпляр Openrouter.
//
// Параметры:
//   - logger: экземпляр *slog.Logger для логирования.
//   - cfg: конфигурация приложения.
//   - pdfService: реализация PdfService для генерации PDF.
//
// Возвращает указатель на инициализированный Openrouter.
func NewClient(
	logger *slog.Logger,
	cfg *config.Config,
	repository Repository,
) *Openrouter {
	op := "Openrouter.NewClient()"
	log := logger.With(
		slog.String("op", op),
	)

	client := openrouter.NewClient(
		cfg.BotConfig.AI.AIApiToken,
	)

	log.Info("Creating openrouter client")

	return &Openrouter{
		logger:          logger,
		cfg:             cfg,
		Client:          client,
		repository:      repository,
		jobs:            make(chan Job, cfg.BotConfig.AI.JobBufferSize),
		shutdownChannel: make(chan struct{}),
		wg:              &sync.WaitGroup{},
	}
}

// Start запускает воркеры для обработки задач.
// Количество воркеров задаётся в конфиге (WorkersCount).
// Метод блокируется до завершения всех воркеров.
func (s *Openrouter) Start() {
	op := "Openrouter.Start()"
	log := s.logger.With(
		slog.String("op", op),
	)
	for i := 0; i < s.cfg.BotConfig.AI.WorkersCount; i++ {
		s.wg.Add(1)
		go s.handleJob(i)
	}
	log.Info("openrouter service started")

	s.wg.Wait()
}

// AddJob добавляет новую задачу в очередь на обработку.
// Принимает событие для обогащения AI.
func (s *Openrouter) AddJob(requestID uuid.UUID, event domain.Event) (chan struct{}, error) {
	newJob := Job{
		requestID: requestID,
		event:     event,
		Done:      make(chan struct{}),
	}
	select {
	case <-s.shutdownChannel:
		return nil, fmt.Errorf("service is shutting down")
	default:
		if len(s.jobs) < s.cfg.BotConfig.AI.JobBufferSize {
			s.jobs <- newJob
			return newJob.Done, nil
		} else {
			return nil, fmt.Errorf("job buffer is full")
		}
	}
}

// handleJob — воркер, обрабатывающий задачи из канала.
// Получает событие, отправляет запрос в OpenRouter для обогащения, обновляет событие в БД.
func (s *Openrouter) handleJob(id int) {
	defer s.wg.Done()
	op := "Openrouter.handleJob()"
	log := s.logger.With(
		slog.String("op", op),
		slog.Int("workerId", id),
	)

	log.Info("start openrouter job handler")

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
				slog.String("eventName", job.event.Name),
			)

			ctx, cancel := context.WithTimeout(context.Background(), s.cfg.BotConfig.AI.GetTimeout())

			// Обогащаем событие через AI
			enrichedResponse, err := s.EnrichEventWithAI(ctx, joblog, job.requestID, job.event)

			if err != nil {
				cancel()
				joblog.Error("failed to enrich event", slog.String("error", err.Error()))
				close(job.Done)
				continue
			}

			// Обновляем событие с данными от AI
			updatedEvent := enrichedResponse.ApplyToEvent(job.event)
			updatedEvent.Status = domain.EventStatusAIEnriched

			_, err = s.repository.UpdateEvent(ctx, updatedEvent)
			cancel() // Освобождаем контекст после всех операций

			if err != nil {
				joblog.Error("failed to update event", slog.String("error", err.Error()))
				close(job.Done)
				continue
			}

			close(job.Done)

			joblog.Info("AI enrichment completed", slog.String("tag", updatedEvent.Tag))
		}
	}
}

// EnrichEventWithAI обогащает событие через AI.
// Принимает событие и возвращает структурированный ответ с обогащёнными данными.
func (s *Openrouter) EnrichEventWithAI(ctx context.Context, logger *slog.Logger, requestId uuid.UUID, event domain.Event) (dto.EventStructuredResponseSchema, error) {
	op := "openrouter.EnrichEventWithAI()"
	log := logger.With(
		slog.String("op", op),
		slog.String("requestID", requestId.String()),
	)
	log.Info("enriching event with AI")

	var responseSchema dto.EventStructuredResponseSchema
	var resp openrouter.ChatCompletionResponse
	var err error

	prompt := s.cfg.BotConfig.AI.SystemRolePrompt

	// Формируем сообщение для AI с данными события
	eventMessage := fmt.Sprintf(`Обогати следующее событие:
Название: %s
Описание: %s
Дата: %s
Цена: %.2f %s
Ссылка на событие: %s

Задачи:
1. Если описание меньше 50 символов, дополни его
2. Убери лишний мусорный текст и куски скриптов
3. Переведи описание на русский язык
4. Определи теги события
5. Сгенерируй ссылку на Google Maps (если есть адрес)
6. Сгенерируй ссылки на Google Calendar`,
		event.Name,
		event.Description,
		event.Date.Format("02.01.2006 15:04"),
		event.Price,
		event.Currency,
		event.EventLink,
	)

	for retry := range retryCount {
		var r openrouter.ChatCompletionResponse
		var e error
		select {
		case <-s.shutdownChannel:
			return dto.EventStructuredResponseSchema{}, fmt.Errorf("shutdown openrouter client")
		default:
			schema, err := jsonschema.GenerateSchemaForType(responseSchema)
			if err != nil {
				log.Error("GenerateSchemaForType error", sl.Err(err))
				return dto.EventStructuredResponseSchema{}, fmt.Errorf("GenerateSchemaForType error: %w", err)
			}

			r, e = s.Client.CreateChatCompletion(
				ctx,
				openrouter.ChatCompletionRequest{
					Model: s.cfg.BotConfig.AI.ModelName,
					Messages: []openrouter.ChatCompletionMessage{
						openrouter.SystemMessage(prompt),
						openrouter.UserMessage(eventMessage),
					},
					ResponseFormat: &openrouter.ChatCompletionResponseFormat{
						Type: "json_schema",
						JSONSchema: &openrouter.ChatCompletionResponseFormatJSONSchema{
							Name:   "eventStructuredResponseSchema",
							Strict: true,
							Schema: schema,
						},
					},
				},
			)
		}
		if e != nil && (isRateLimitError(e) || isEOFError(e)) {
			err = e
			log.Error("AI completion error", slog.String("error", err.Error()), slog.Int("retry", retry))
			time.Sleep(retryDuration)
			continue
		}
		resp = r
		err = e
		break
	}

	if err != nil {
		return dto.EventStructuredResponseSchema{}, fmt.Errorf("AI completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return dto.EventStructuredResponseSchema{}, fmt.Errorf("empty AI response")
	}

	// Очищаем ответ от markdown-разметки (```json ... ```)
	cleanedResponse := cleanJSONResponse(resp.Choices[0].Message.Content.Text)
	b := []byte(cleanedResponse)
	err = json.Unmarshal(b, &responseSchema)
	if err != nil {
		log.Error("error unmarshal response", sl.Err(err), slog.String("response", cleanedResponse))
		return dto.EventStructuredResponseSchema{}, fmt.Errorf("unmarshal error: %w", err)
	}

	log.Debug("AI enrichment response", slog.Any("schema", responseSchema))
	return responseSchema, nil
}

// isRateLimitError проверяет, связана ли ошибка с превышением лимита запросов (HTTP 429).
// Временное решение по анализу строки ошибки — менее надёжно, чем проверка кода.
func isRateLimitError(err error) bool {
	if err != nil {
		//return strings.Contains(err.Error(), "HTTP 429")
		return strings.Contains(err.Error(), "429")
	}
	return false
}

// isEOFError проверяет, связана ли ошибка с разрывом соединения (EOF).
// Используется для повтора запроса.
func isEOFError(err error) bool {
	if err != nil {
		return strings.Contains(err.Error(), "EOF")
	}
	return false
}

// cleanJSONResponse очищает ответ AI от markdown-разметки и лишнего текста.
// Некоторые модели (например, Claude) могут оборачивать JSON в ```json ... ```
// и добавлять текст после JSON объекта.
func cleanJSONResponse(response string) string {
	response = strings.TrimSpace(response)

	// Убираем markdown блок ```json или ``` в начале
	if after, ok := strings.CutPrefix(response, "```json"); ok {
		response = after
	} else if after0, ok0 := strings.CutPrefix(response, "```"); ok0 {
		response = after0
	}

	response = strings.TrimSpace(response)

	// Находим JSON объект: от первой { до соответствующей }
	startIdx := strings.Index(response, "{")
	if startIdx == -1 {
		return response
	}

	// Ищем закрывающую скобку, учитывая вложенность
	depth := 0
	endIdx := -1
	inString := false
	escaped := false

	for i := startIdx; i < len(response); i++ {
		c := response[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' && inString {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == '{' {
			depth++
		} else if c == '}' {
			depth--
			if depth == 0 {
				endIdx = i
				break
			}
		}
	}

	if endIdx != -1 {
		return response[startIdx : endIdx+1]
	}

	return response
}

// writeResponseInFile сохраняет текстовый ответ ИИ в файл.
//
// Параметры:
//   - requestId: идентификатор запроса (часть имени файла).
//   - data: содержимое для записи.
//   - fileType: расширение файла (например, "html").
//
// Использует filepath.Clean для защиты от path traversal.
// Устанавливает права 0644.
// Возвращает ошибку при неудачной записи.
func (s *Openrouter) writeResponseInFile(requestId string, data string, fileType string) error {
	if _, err := uuid.Parse(requestId); err != nil {
		return fmt.Errorf("invalid requestId")
	}
	bufWrite := []byte(data)
	filePath := filepath.Clean(fmt.Sprintf("%s%s.%s", s.cfg.BotConfig.AI.AiResponseFilePath, requestId, fileType))
	err := os.WriteFile(filePath, bufWrite, 0644)
	if err != nil {
		return fmt.Errorf("error write file \"%s\": %w", filePath, err)
	}
	return nil
}

// Shutdown корректно завершает работу сервиса.
//
// Параметры:
//   - ctx: контекст для отслеживания таймаута завершения.
//
// Действия:
//   - Закрывает канал shutdownChannel.
//   - Закрывает канал jobs для остановки воркеров.
//   - Возвращает ошибку, если контекст отменён.
//
// После вызова новые задачи не принимаются, обработка текущих завершается.
func (s *Openrouter) Shutdown(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("force exit AI client: %w", ctx.Err())
	default:
		close(s.shutdownChannel)
		close(s.jobs)
		return nil
	}
}
