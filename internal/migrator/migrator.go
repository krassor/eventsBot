package migrator

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrator управляет миграциями базы данных
type Migrator struct {
	db  *sqlx.DB
	log *slog.Logger
}

// NewMigrator создает новый экземпляр мигратора
func NewMigrator(db *sqlx.DB, log *slog.Logger) *Migrator {
	return &Migrator{
		db:  db,
		log: log,
	}
}

// Run выполняет все миграции при запуске приложения
func (m *Migrator) Run() error {
	op := "migrator.Run"
	m.log.Info("starting database migrations")

	// создаем таблицу для отслеживания миграций
	if err := m.createMigrationsTable(); err != nil {
		return fmt.Errorf("%s: failed to create migrations table: %w", op, err)
	}

	// получаем список всех миграций
	migrations, err := m.getMigrationFiles()
	if err != nil {
		return fmt.Errorf("%s: failed to get migration files: %w", op, err)
	}

	// выполняем каждую миграцию
	for _, migration := range migrations {
		if err := m.runMigration(migration); err != nil {
			return fmt.Errorf("%s: failed to run migration %s: %w", op, migration, err)
		}
	}

	m.log.Info("database migrations completed successfully")
	return nil
}

// createMigrationsTable создает таблицу для отслеживания выполненных миграций
func (m *Migrator) createMigrationsTable() error {
	// создаем схему если она еще не существует
	schemaQuery := `CREATE SCHEMA IF NOT EXISTS profreport`
	if _, err := m.db.Exec(schemaQuery); err != nil {
		return err
	}

	// создаем таблицу миграций в схеме profreport
	query := `
		CREATE TABLE IF NOT EXISTS profreport.schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := m.db.Exec(query)
	return err
}

// getMigrationFiles возвращает отсортированный список файлов миграций
func (m *Migrator) getMigrationFiles() ([]string, error) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	var migrations []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			migrations = append(migrations, entry.Name())
		}
	}

	sort.Strings(migrations)
	return migrations, nil
}

// isMigrationApplied проверяет, была ли миграция уже применена
func (m *Migrator) isMigrationApplied(version string) (bool, error) {
	var count int
	query := `SELECT COUNT(*) FROM profreport.schema_migrations WHERE version = $1`
	err := m.db.Get(&count, query, version)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// runMigration выполняет одну миграцию
func (m *Migrator) runMigration(filename string) error {
	version := strings.TrimSuffix(filename, ".sql")

	// проверяем, была ли миграция уже применена
	applied, err := m.isMigrationApplied(version)
	if err != nil {
		return err
	}

	if applied {
		m.log.Debug("migration already applied", slog.String("version", version))
		return nil
	}

	m.log.Info("applying migration", slog.String("version", version))

	// читаем содержимое файла миграции
	content, err := migrationsFS.ReadFile("migrations/" + filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file: %w", err)
	}

	// выполняем миграцию в транзакции
	tx, err := m.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	// выполняем SQL из файла миграции
	if _, err = tx.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute migration: %w", err)
	}

	// записываем информацию о применении миграции
	insertQuery := `INSERT INTO profreport.schema_migrations (version) VALUES ($1)`
	if _, err = tx.Exec(insertQuery, version); err != nil {
		return fmt.Errorf("failed to record migration: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	m.log.Info("migration applied successfully", slog.String("version", version))
	return nil
}

// Rollback откатывает последнюю миграцию (для будущего использования)
func (m *Migrator) Rollback() error {
	return fmt.Errorf("rollback not implemented yet")
}

// GetAppliedMigrations возвращает список примененных миграций
func (m *Migrator) GetAppliedMigrations() ([]string, error) {
	var versions []string
	query := `SELECT version FROM profreport.schema_migrations ORDER BY applied_at DESC`
	err := m.db.Select(&versions, query)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	return versions, nil
}
