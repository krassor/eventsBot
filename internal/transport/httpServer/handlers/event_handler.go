package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"eventsBot/internal/models/domain"
	"eventsBot/internal/transport/httpServer/handlers/dto"
	"eventsBot/internal/utils"
	"eventsBot/internal/utils/logger/sl"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type EventHandler struct {
	repository        EventRepository
	eventOrchestrator EventOrchestrator
	log               *slog.Logger
}

func NewEventHandler(log *slog.Logger, repo EventRepository, eventOrchestrator EventOrchestrator) *EventHandler {
	return &EventHandler{
		repository:        repo,
		eventOrchestrator: eventOrchestrator,
		log:               log,
	}
}

// GetEvents обрабатывает GET /api/v1/events?status=...
// Если параметр status не задан или пустой — возвращаются все события.
func (h *EventHandler) GetEvents(w http.ResponseWriter, r *http.Request) {
	op := "httpServer.handlers.EventHandler.GetEvents()"
	log := h.log.With(slog.String("op", op))

	status := r.URL.Query().Get("status")
	ctx := r.Context()

	var events []domain.Event
	var err error

	if status != "" {
		if !isValidStatus(status) {
			h.respondError(log, fmt.Errorf("invalid status filter: %s", status), w, http.StatusBadRequest)
			return
		}
		events, err = h.repository.FindEventsByStatus(ctx, domain.EventStatus(status))
	} else {
		events, err = h.repository.ReadAllEvents(ctx)
	}

	if err != nil {
		h.respondError(log, fmt.Errorf("failed to get events: %w", err), w, http.StatusInternalServerError)
		return
	}

	response := dto.MapDomainToEventResponseList(events)

	if err := utils.Json(w, http.StatusOK, response); err != nil {
		log.Error("error encoding response", sl.Err(err))
	}
}

// ChangeEvent обрабатывает PUT /api/v1/events/{eventId}
// Полностью заменяет событие переданными данными.
func (h *EventHandler) ChangeEvent(w http.ResponseWriter, r *http.Request) {
	op := "httpServer.handlers.EventHandler.ChangeEvent()"
	log := h.log.With(slog.String("op", op))

	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		h.respondError(log, fmt.Errorf("empty eventId"), w, http.StatusBadRequest)
		return
	}

	parsedID, err := uuid.Parse(eventID)
	if err != nil {
		h.respondError(log, fmt.Errorf("invalid eventId: %w", err), w, http.StatusBadRequest)
		return
	}

	var req dto.ChangeEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(log, fmt.Errorf("cannot decode json: %w", err), w, http.StatusBadRequest)
		return
	}

	if req.Status != "" && !isValidStatus(req.Status) {
		h.respondError(log, fmt.Errorf("invalid status: %s", req.Status), w, http.StatusBadRequest)
		return
	}

	event := dto.MapEventRequestToDomain(req, parsedID)

	log.Info("changing event", slog.String("eventID", eventID))

	ctx := r.Context()
	updated, err := h.repository.UpdateEvent(ctx, event)
	if err != nil {
		h.respondError(log, fmt.Errorf("failed to update event: %w", err), w, http.StatusInternalServerError)
		return
	}

	response := dto.MapDomainToEventResponse(updated)

	if err := utils.Json(w, http.StatusOK, response); err != nil {
		log.Error("error encoding response", sl.Err(err))
	}
}

// UpdateStatus обрабатывает PUT /api/v1/events/{eventId}/status
func (h *EventHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	op := "httpServer.handlers.EventHandler.UpdateStatus()"
	log := h.log.With(slog.String("op", op))

	eventID := chi.URLParam(r, "eventId")
	if eventID == "" {
		h.respondError(log, fmt.Errorf("empty eventId"), w, http.StatusBadRequest)
		return
	}

	var req dto.UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(log, fmt.Errorf("cannot decode json: %w", err), w, http.StatusBadRequest)
		return
	}

	// Валидация статуса
	if !isValidStatus(req.Status) {
		h.respondError(log, fmt.Errorf("invalid status: %s", req.Status), w, http.StatusBadRequest)
		return
	}

	log.Info("updating event status",
		slog.String("eventID", eventID),
		slog.String("status", req.Status),
	)

	ctx := r.Context()
	event, err := h.repository.FindEventByID(ctx, uuid.Must(uuid.Parse(eventID)))
	if err != nil {
		h.respondError(log, fmt.Errorf("failed to get event: %w", err), w, http.StatusInternalServerError)
		return
	}
	oldStatus := event.Status

	event.Status = domain.EventStatus(req.Status)
	err = h.repository.UpdateEventStatus(ctx, event.ID, string(event.Status))
	if err != nil {
		h.respondError(log, fmt.Errorf("failed to update event status: %w", err), w, http.StatusInternalServerError)
		return
	}

	if req.Status == string(domain.EventStatusReadyToApprove) {
		err = h.eventOrchestrator.SendEventToTelegram(&event)
		if err != nil {
			event.Status = domain.EventStatus(oldStatus)
			err = h.repository.UpdateEventStatus(ctx, event.ID, string(event.Status))
			if err != nil {
				h.respondError(log, fmt.Errorf("failed to update event status: %w", err), w, http.StatusInternalServerError)
				return
			}
			h.respondError(log, fmt.Errorf("failed to send event to Telegram: %w", err), w, http.StatusInternalServerError)
			return
		}
	}

	if err := utils.Json(w, http.StatusOK, map[string]string{"status": "ok"}); err != nil {
		log.Error("error encoding response", sl.Err(err))
	}
}

func (h *EventHandler) respondError(log *slog.Logger, err error, w http.ResponseWriter, status int) {
	log.Error("handler error", sl.Err(err))
	if httpErr := utils.Err(w, status, err); httpErr != nil {
		log.Error("error sending http response", sl.Err(httpErr))
	}
}

// isValidStatus проверяет, является ли переданный статус допустимым.
func isValidStatus(status string) bool {
	switch domain.EventStatus(status) {
	case domain.EventStatusNew,
		domain.EventStatusAIEnriched,
		domain.EventStatusReadyToApprove,
		domain.EventStatusApproved,
		domain.EventStatusRejected:
		return true
	default:
		return false
	}
}
