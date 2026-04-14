package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"

	"github.com/Sukhveer/taskflow/internal/config"
	"github.com/Sukhveer/taskflow/internal/handler"
	"github.com/Sukhveer/taskflow/internal/infrastructure"
	"github.com/Sukhveer/taskflow/internal/infrastructure/database"
	applogger "github.com/Sukhveer/taskflow/internal/infrastructure/logger"
	"github.com/Sukhveer/taskflow/internal/middleware"
	postgresrepo "github.com/Sukhveer/taskflow/internal/repository/postgres"
	"github.com/Sukhveer/taskflow/internal/service"
)

func main() {
	// Load .env if present (ignored in production where env vars are set directly)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	logger := applogger.New(cfg.Log.Level)

	// ── Database ──────────────────────────────────────────────────────────────
	db, err := database.New(cfg.Database)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// Run migrations automatically on start
	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		migrationsPath = "./migrations"
	}
	if err := database.RunMigrations(db, migrationsPath); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	// ── Infrastructure ────────────────────────────────────────────────────────
	tokenService := infrastructure.NewJWTTokenService(cfg.JWT)

	// ── Repositories (implements schema interfaces) ───────────────────────────
	userRepo := postgresrepo.NewUserRepository(db)
	projectRepo := postgresrepo.NewProjectRepository(db)
	taskRepo := postgresrepo.NewTaskRepository(db)

	// ── Services (business logic layer) ──────────────────────────────────────
	authSvc := service.NewAuthService(userRepo, tokenService, logger)
	projectSvc := service.NewProjectService(projectRepo, taskRepo, logger)
	taskSvc := service.NewTaskService(taskRepo, projectRepo, logger)

	// ── Handlers (HTTP layer) ─────────────────────────────────────────────────
	authHandler := handler.NewAuthHandler(authSvc, logger)
	projectHandler := handler.NewProjectHandler(projectSvc, logger)
	taskHandler := handler.NewTaskHandler(taskSvc, logger)

	// ── Router ────────────────────────────────────────────────────────────────
	_ = middleware.UserIDFromContext // imported for side effects / linker
	router := handler.NewRouter(authHandler, projectHandler, taskHandler, tokenService, logger)

	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: router,
	}

	// ── Graceful shutdown ─────────────────────────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		logger.Info("server starting", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	logger.Info("shutdown signal received")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}

// Ensure middleware package is used (avoids unused import if refactored)
var _ = slog.Default
