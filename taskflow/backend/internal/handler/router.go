package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/Sukhveer/taskflow/internal/middleware"
	"github.com/Sukhveer/taskflow/internal/schema"
)

// NewRouter wires all routes and returns the root http.Handler.
// Each handler is injected — following the Dependency Inversion Principle.
func NewRouter(
	authHandler *AuthHandler,
	projectHandler *ProjectHandler,
	taskHandler *TaskHandler,
	tokenService schema.TokenService,
	logger *slog.Logger,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware stack (applied in order)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.Recoverer(logger))
	r.Use(middleware.RequestLogger(logger))

	// Health-check (no auth required)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	// ── Auth routes (public) ──────────────────────────────────────────────────
	r.Route("/auth", func(r chi.Router) {
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	// ── Protected routes ──────────────────────────────────────────────────────
	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(tokenService))

		// Projects
		r.Route("/projects", func(r chi.Router) {
			r.Get("/", projectHandler.ListProjects)
			r.Post("/", projectHandler.CreateProject)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/", projectHandler.GetProject)
				r.Patch("/", projectHandler.UpdateProject)
				r.Delete("/", projectHandler.DeleteProject)
				r.Get("/stats", projectHandler.GetProjectStats)

				// Tasks nested under project
				r.Get("/tasks", taskHandler.ListTasks)
				r.Post("/tasks", taskHandler.CreateTask)
			})
		})

		// Tasks (top-level for update/delete by task ID)
		r.Route("/tasks/{id}", func(r chi.Router) {
			r.Patch("/", taskHandler.UpdateTask)
			r.Delete("/", taskHandler.DeleteTask)
		})
	})

	return r
}
