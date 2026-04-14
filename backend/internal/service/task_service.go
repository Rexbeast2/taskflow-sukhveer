package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// taskService implements schema.TaskService
type taskService struct {
	taskRepo    schema.TaskRepository
	projectRepo schema.ProjectRepository
	logger      *slog.Logger
}

// NewTaskService constructs a taskService with its dependencies injected
func NewTaskService(
	taskRepo schema.TaskRepository,
	projectRepo schema.ProjectRepository,
	logger *slog.Logger,
) schema.TaskService {
	return &taskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		logger:      logger,
	}
}

// requireProjectAccess returns the project if the user has read access,
// or ErrAccessDenied (→ 404) otherwise. Prevents project ID enumeration.
func (s *taskService) requireProjectAccess(ctx context.Context, projectID, userID uuid.UUID) (*schema.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, schema.ErrAccessDenied
	}
	ok, err := s.projectRepo.UserHasAccess(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("check project access: %w", err)
	}
	if !ok {
		return nil, schema.ErrAccessDenied
	}
	return project, nil
}

// requireProjectOwner returns the project if the user is the owner.
// Returns ErrAccessDenied (404) for no-visibility, ErrNotProjectOwner (403) for non-owners.
func (s *taskService) requireProjectOwner(ctx context.Context, projectID, userID uuid.UUID) (*schema.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, schema.ErrAccessDenied
	}
	ok, err := s.projectRepo.UserHasAccess(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("check project access: %w", err)
	}
	if !ok {
		return nil, schema.ErrAccessDenied
	}
	if !project.IsOwnedBy(userID) {
		return nil, schema.ErrNotProjectOwner
	}
	return project, nil
}

// Create validates, checks ownership, and persists a new task.
// Only the project owner can add tasks to a project.
func (s *taskService) Create(ctx context.Context, input schema.CreateTaskInput) (*schema.Task, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return nil, schema.ErrTaskTitleRequired
	}

	if input.Status == "" {
		input.Status = schema.TaskStatusTodo
	}
	if input.Priority == "" {
		input.Priority = schema.TaskPriorityMedium
	}
	if !input.Status.IsValid() {
		return nil, schema.ErrInvalidStatus
	}
	if !input.Priority.IsValid() {
		return nil, schema.ErrInvalidPriority
	}

	// Only project owner can create tasks
	if _, err := s.requireProjectOwner(ctx, input.ProjectID, input.RequesterID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	task := &schema.Task{
		ID:          uuid.New(),
		Title:       input.Title,
		Description: input.Description,
		Status:      input.Status,
		Priority:    input.Priority,
		ProjectID:   input.ProjectID,
		AssigneeID:  input.AssigneeID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if input.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format, use YYYY-MM-DD: %w", err)
		}
		task.DueDate = &parsed
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	s.logger.Info("task created", "task_id", task.ID, "project_id", input.ProjectID)
	return task, nil
}

// ListByProject returns paginated, filtered tasks.
// Requires the requester to be the project owner OR have an assigned task in it.
func (s *taskService) ListByProject(ctx context.Context, projectID uuid.UUID, filter schema.TaskFilter, requesterID uuid.UUID) ([]schema.Task, int, error) {
	if _, err := s.requireProjectAccess(ctx, projectID, requesterID); err != nil {
		return nil, 0, err
	}

	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 20
	}

	return s.taskRepo.ListByProject(ctx, projectID, filter)
}

// Update applies partial updates to a task.
// Allowed for: project owner OR the task's current assignee.
// All others receive 403; a task from an invisible project receives 404.
func (s *taskService) Update(ctx context.Context, input schema.UpdateTaskInput) (*schema.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, input.ID)
	if err != nil {
		return nil, err // ErrTaskNotFound → 404
	}

	// Check requester has visibility of the parent project first
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		return nil, schema.ErrAccessDenied
	}

	ok, err := s.projectRepo.UserHasAccess(ctx, task.ProjectID, input.RequesterID)
	if err != nil {
		return nil, fmt.Errorf("check project access: %w", err)
	}
	if !ok {
		// Project exists but user has zero visibility — 404
		return nil, schema.ErrAccessDenied
	}

	// Within a visible project: only owner OR current assignee may update
	isOwner := project.IsOwnedBy(input.RequesterID)
	isAssignee := task.AssigneeID != nil && *task.AssigneeID == input.RequesterID
	if !isOwner && !isAssignee {
		return nil, schema.ErrNotProjectOwner
	}

	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		if trimmed == "" {
			return nil, schema.ErrTaskTitleRequired
		}
		task.Title = trimmed
	}
	if input.Description != nil {
		task.Description = input.Description
	}
	if input.Status != nil {
		if !input.Status.IsValid() {
			return nil, schema.ErrInvalidStatus
		}
		task.Status = *input.Status
	}
	if input.Priority != nil {
		if !input.Priority.IsValid() {
			return nil, schema.ErrInvalidPriority
		}
		task.Priority = *input.Priority
	}
	if input.AssigneeID != nil {
		task.AssigneeID = input.AssigneeID
	}
	if input.DueDate != nil {
		parsed, err := time.Parse("2006-01-02", *input.DueDate)
		if err != nil {
			return nil, fmt.Errorf("invalid due_date format, use YYYY-MM-DD: %w", err)
		}
		task.DueDate = &parsed
	}

	task.UpdatedAt = time.Now().UTC()

	if err := s.taskRepo.Update(ctx, task); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}

	s.logger.Info("task updated", "task_id", task.ID, "requester", input.RequesterID)
	return task, nil
}

// Delete removes a task. Project owner only.
// Returns 404 for invisible projects, 403 for non-owners.
func (s *taskService) Delete(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err // ErrTaskNotFound → 404
	}

	if _, err := s.requireProjectOwner(ctx, task.ProjectID, requesterID); err != nil {
		return err
	}

	if err := s.taskRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}

	s.logger.Info("task deleted", "task_id", id, "requester", requesterID)
	return nil
}
