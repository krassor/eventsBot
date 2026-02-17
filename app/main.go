package main

import (
	"context"
	"eventsBot/internal/config"
	"eventsBot/internal/graceful"
	"eventsBot/internal/openrouter"
	"eventsBot/internal/orchestrator"
	"eventsBot/internal/repositories"
	"eventsBot/internal/scraper"
	telegramBot "eventsBot/internal/telegram"
	"eventsBot/internal/transport/httpServer"
	"eventsBot/internal/transport/httpServer/handlers"
	"eventsBot/internal/transport/httpServer/routers"
	"eventsBot/internal/utils/logger/handlers/slogpretty"
	"log/slog"
	"os"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

var Version = "0.1"

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	cfg.ReadPromptFromFile()

	log.Info(
		"starting events bot",
		slog.String("env", cfg.Env),
		slog.String("version", Version),
	)

	repositoryService := repositories.New(log, cfg)
	aiService := openrouter.NewClient(log, cfg, repositoryService)
	scraperService := scraper.New(log, cfg, repositoryService)
	tgBot := telegramBot.New(log, cfg, repositoryService)
	orchestratorService := orchestrator.New(log, cfg, scraperService, aiService, repositoryService, tgBot, scraperService.CompletedEventsChan)

	// HTTP Server
	eventHandler := handlers.NewEventHandler(log, repositoryService, orchestratorService)
	router := routers.NewRouter(eventHandler)
	httpSrv := httpServer.NewHttpServer(log, router, cfg)

	maxSecond := 15 * time.Second
	waitShutdown := graceful.GracefulShutdown(
		context.Background(),
		maxSecond,
		map[string]graceful.Operation{
			"Scraper service": func(ctx context.Context) error {
				return scraperService.Shutdown(ctx)
			},
			"AI service": func(ctx context.Context) error {
				return aiService.Shutdown(ctx)
			},
			"Repository service": func(ctx context.Context) error {
				return repositoryService.Shutdown(ctx)
			},
			"Telegram bot": func(ctx context.Context) error {
				return tgBot.Shutdown(ctx)
			},
			"Orchestrator service": func(ctx context.Context) error {
				return orchestratorService.Shutdown(ctx)
			},
			"HTTP server": func(ctx context.Context) error {
				return httpSrv.Shutdown(ctx)
			},
		},
		log,
	)

	go aiService.Start()
	go scraperService.Start()
	go orchestratorService.Start()
	go tgBot.Start(30)
	go httpSrv.Listen()

	<-waitShutdown
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		// log = slog.New(
		// 	slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		// )
		log = setupPrettySlogProd()
	default: // If env config is invalid, set prod settings by default due to security
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}

func setupPrettySlogProd() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelInfo,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
