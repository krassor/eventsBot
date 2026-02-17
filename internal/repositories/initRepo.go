package repositories

import (
	"context"
	"eventsBot/internal/config"
	"eventsBot/internal/migrator"
	"eventsBot/internal/models/domain"
	"eventsBot/internal/utils/logger/sl"
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Repository struct {
	DB  *sqlx.DB
	log *slog.Logger
}

// ListEvents implements [openrouter.Repository].
func (r *Repository) ListEvents(ctx context.Context) ([]domain.Event, error) {
	panic("unimplemented")
}

func New(logger *slog.Logger, cfg *config.Config) *Repository {
	op := "repositories.New()"
	log := logger.With(
		slog.String("op", op))

	username := cfg.DBConfig.User
	password := cfg.DBConfig.Password
	dbName := cfg.DBConfig.Name
	dbHost := cfg.DBConfig.Host
	dbPort := cfg.DBConfig.Port

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s sslmode=disable password=%s search_path=profreport",
		dbHost, dbPort, username, dbName, password)

	conn, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		log.Error("error connecting to database", sl.Err(err))
		panic("error connecting to database")
	}

	// проверка подключения
	if err := conn.Ping(); err != nil {
		log.Error("error pinging database", sl.Err(err))
		panic("error pinging database")
	}

	log.Debug("sqlx have connected to database")

	// выполнение миграций
	m := migrator.NewMigrator(conn, log)
	if err := m.Run(); err != nil {
		log.Error("error running database migrations", sl.Err(err))
		panic("error running database migrations")
	}

	return &Repository{
		DB:  conn,
		log: log,
	}
}

func (r *Repository) Shutdown(ctx context.Context) error {
	op := "Repository.Shutdown"
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("force exit %s: %w", op, ctx.Err())
		default:
			if err := r.DB.Close(); err != nil {
				return fmt.Errorf("error exit %s: %w", op, err)
			}
			return nil
		}
	}
}
