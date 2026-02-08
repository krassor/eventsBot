package main

import (
	"app/main.go/internal/config"
	"app/main.go/internal/graceful"
	"app/main.go/internal/openrouter"
	"app/main.go/internal/orchestrator"
	"app/main.go/internal/repositories"
	"app/main.go/internal/scraper"
	telegramBot "app/main.go/internal/telegram"
	"app/main.go/internal/utils/logger/handlers/slogpretty"
	"context"
	"log/slog"
	"os"
	"time"
	//openai "app/main.go/internal/openai"
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
		},
		log,
	)

	go aiService.Start()
	go scraperService.Start()
	go orchestratorService.Start()
	go tgBot.Start(30)

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
