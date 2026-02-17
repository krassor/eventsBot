package routers

import (
	"eventsBot/internal/transport/httpServer/handlers"
	myMiddleware "eventsBot/internal/transport/httpServer/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Router struct {
	eventHandler *handlers.EventHandler
}

func NewRouter(eventHandler *handlers.EventHandler) *Router {
	return &Router{
		eventHandler: eventHandler,
	}
}

func (r *Router) Mount(mux *chi.Mux) {

	mux.Use(cors.AllowAll().Handler)
	mux.Use(myMiddleware.LoggerMiddleware)
	mux.Use(middleware.Heartbeat("/ping"))

	mux.Route("/api", func(mux chi.Router) {
		mux.Route("/v1", func(mux chi.Router) {
			mux.Route("/events", func(mux chi.Router) {
				mux.Get("/", r.eventHandler.GetEvents)
				mux.Put("/{eventId}", r.eventHandler.ChangeEvent)
				mux.Put("/{eventId}/status", r.eventHandler.UpdateStatus)
			})
		})
	})
}
