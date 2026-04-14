package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/middleware"
	"github.com/Sukhveer/taskflow/internal/schema"
)

// ProjectHandler handles project-related HTTP requests
type ProjectHandler struct {
	projectService schema.ProjectService
	logger         *slog.Logger
}

// NewProjectHandler creates a new ProjectHandler
func NewProjectHandler(projectService schema.ProjectService, logger *slog.Logger) *ProjectHandler {
	return &ProjectHandler{projectService: projectService, logger: logger}
}

// paginatedResponse wraps a list result with metadata
type paginatedResponse struct {
	Data  any `json:"data"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

// ListProjects handles GET /projects
func (h *ProjectHandler) ListProjects(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	page, limit := parsePagination(r)

	projects, total, err := h.projectService.List(r.Context(), userID, page, limit)
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	// Return empty array rather than null
	if projects == nil {
		projects = []schema.Project{}
	}

	respond(w, http.StatusOK, paginatedResponse{
		Data:  projects,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// createProjectRequest is the incoming body for POST /projects
type createProjectRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// CreateProject handles POST /projects
func (h *ProjectHandler) CreateProject(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())

	var req createProjectRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondValidationError(w, map[string]string{"name": "is required"})
		return
	}

	project, err := h.projectService.Create(r.Context(), schema.CreateProjectInput{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     userID,
	})
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusCreated, project)
}

// GetProject handles GET /projects/:id
func (h *ProjectHandler) GetProject(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	project, err := h.projectService.GetByID(r.Context(), id, userID)
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusOK, project)
}

// updateProjectRequest is the incoming body for PATCH /projects/:id
type updateProjectRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

// UpdateProject handles PATCH /projects/:id
func (h *ProjectHandler) UpdateProject(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req updateProjectRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	project, err := h.projectService.Update(r.Context(), schema.UpdateProjectInput{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		RequesterID: userID,
	})
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusOK, project)
}

// DeleteProject handles DELETE /projects/:id
func (h *ProjectHandler) DeleteProject(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.projectService.Delete(r.Context(), id, userID); err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusNoContent, nil)
}

// GetProjectStats handles GET /projects/:id/stats
func (h *ProjectHandler) GetProjectStats(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	stats, err := h.projectService.GetStats(r.Context(), id, userID)
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusOK, stats)
}

// parseUUID extracts and validates a UUID from a chi URL parameter
func parseUUID(r *http.Request, param string) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, param))
}

// parsePagination reads ?page= and ?limit= query params with safe defaults
func parsePagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return
}
