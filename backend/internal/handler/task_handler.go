package handler

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/middleware"
	"github.com/Sukhveer/taskflow/internal/schema"
)

// TaskHandler handles task-related HTTP requests
type TaskHandler struct {
	taskService schema.TaskService
	logger      *slog.Logger
}

// NewTaskHandler creates a new TaskHandler
func NewTaskHandler(taskService schema.TaskService, logger *slog.Logger) *TaskHandler {
	return &TaskHandler{taskService: taskService, logger: logger}
}

// ListTasks handles GET /projects/:id/tasks
// Supports ?status= and ?assignee= query filters, and ?page= / ?limit= pagination
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	page, limit := parsePagination(r)
	filter := schema.TaskFilter{Page: page, Limit: limit}

	if s := r.URL.Query().Get("status"); s != "" {
		status := schema.TaskStatus(s)
		if !status.IsValid() {
			respondValidationError(w, map[string]string{"status": "must be todo, in_progress, or done"})
			return
		}
		filter.Status = &status
	}

	if a := r.URL.Query().Get("assignee"); a != "" {
		assigneeID, err := uuid.Parse(a)
		if err != nil {
			respondValidationError(w, map[string]string{"assignee": "must be a valid UUID"})
			return
		}
		filter.AssigneeID = &assigneeID
	}

	tasks, total, err := h.taskService.ListByProject(r.Context(), projectID, filter, userID)
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	if tasks == nil {
		tasks = []schema.Task{}
	}

	respond(w, http.StatusOK, paginatedResponse{
		Data:  tasks,
		Total: total,
		Page:  page,
		Limit: limit,
	})
}

// createTaskRequest is the incoming body for POST /projects/:id/tasks
type createTaskRequest struct {
	Title       string              `json:"title"`
	Description *string             `json:"description"`
	Status      schema.TaskStatus   `json:"status"`
	Priority    schema.TaskPriority `json:"priority"`
	AssigneeID  *string             `json:"assignee_id"`
	DueDate     *string             `json:"due_date"`
}

// CreateTask handles POST /projects/:id/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	projectID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req createTaskRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Title == "" {
		respondValidationError(w, map[string]string{"title": "is required"})
		return
	}

	input := schema.CreateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		ProjectID:   projectID,
		DueDate:     req.DueDate,
		RequesterID: userID,
	}

	if req.AssigneeID != nil {
		parsed, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			respondValidationError(w, map[string]string{"assignee_id": "must be a valid UUID"})
			return
		}
		input.AssigneeID = &parsed
	}

	task, err := h.taskService.Create(r.Context(), input)
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusCreated, task)
}

// updateTaskRequest is the incoming body for PATCH /tasks/:id
type updateTaskRequest struct {
	Title       *string              `json:"title"`
	Description *string              `json:"description"`
	Status      *schema.TaskStatus   `json:"status"`
	Priority    *schema.TaskPriority `json:"priority"`
	AssigneeID  *string              `json:"assignee_id"`
	DueDate     *string              `json:"due_date"`
}

// UpdateTask handles PATCH /tasks/:id
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	taskID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var req updateTaskRequest
	if err := decode(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	input := schema.UpdateTaskInput{
		ID:          taskID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		RequesterID: userID,
	}

	if req.AssigneeID != nil {
		parsed, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			respondValidationError(w, map[string]string{"assignee_id": "must be a valid UUID"})
			return
		}
		input.AssigneeID = &parsed
	}

	task, err := h.taskService.Update(r.Context(), input)
	if err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusOK, task)
}

// DeleteTask handles DELETE /tasks/:id
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	taskID, err := parseUUID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.taskService.Delete(r.Context(), taskID, userID); err != nil {
		schemaErrorToHTTP(w, err)
		return
	}

	respond(w, http.StatusNoContent, nil)
}
