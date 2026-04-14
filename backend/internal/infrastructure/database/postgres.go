package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"github.com/Sukhveer/taskflow/internal/config"
)

// New creates and verifies a PostgreSQL connection pool
func New(cfg config.DatabaseConfig) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)

		err = db.PingContext(ctx)
		cancel()

		if err == nil {
			return db, nil
		}

		// Exponential backoff
		backoff := time.Duration(1<<attempt) * cfg.BaseDelay

		// Cap delay
		if backoff > cfg.MaxDelay {
			backoff = cfg.MaxDelay
		}

		// Jitter
		jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
		delay := backoff/2 + jitter

		slog.Error(
			"DB not ready",
			"attempt", attempt,
			"max_attempts", cfg.MaxAttempts,
			"error", err,
			"retry_in", delay.String(),
		)

		time.Sleep(delay)
	}

	return nil, fmt.Errorf("database connection failed after %d attempts: %w", cfg.MaxAttempts, err)
}

// RunMigrations executes pending migrations from the given path
func RunMigrations(db *sql.DB, migrationsPath string) error {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("create migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	version, dirty, err := m.Version()
	if err != nil && err != migrate.ErrNilVersion {
		return fmt.Errorf("get migration version: %w", err)
	}

	slog.Info("migrations applied", "version", version, "dirty", dirty)
	return nil
}
