package httpServer

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"eventsBot/internal/config"
	"eventsBot/internal/utils/logger/sl"
	"log/slog"
	"net/http"

	"eventsBot/internal/transport/httpServer/routers"
)

type HttpServer struct {
	router     *routers.Router
	httpServer *http.Server
	cfg        *config.Config
	log        *slog.Logger
}

func NewHttpServer(log *slog.Logger, router *routers.Router, cfg *config.Config) *HttpServer {
	return &HttpServer{
		router: router,
		cfg:    cfg,
		log:    log,
	}
}

func (h *HttpServer) Listen() {
	op := "httpServer.Listen()"
	h.log.With(
		slog.String("op", op))

	mux := chi.NewRouter()
	h.router.Mount(mux)

	serverPort := h.cfg.HttpServer.Port
	serverAddress := h.cfg.HttpServer.Address

	h.httpServer = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", serverAddress, serverPort),
		Handler: mux,
	}
	h.log.Info("http server starts on ",
		slog.String("address", serverAddress),
		slog.String("port", serverPort))

	err := h.httpServer.ListenAndServe()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		h.log.Error("error start httpServer", sl.Err(err))
	}

	if errors.Is(err, http.ErrServerClosed) {
		h.log.Info("httpServer closed", sl.Err(err))
	}

}

func (h *HttpServer) Shutdown(ctx context.Context) error {
	if err := h.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return nil
}
