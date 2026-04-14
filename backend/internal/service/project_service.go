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

// projectService implements schema.ProjectService
type projectService struct {
	projectRepo schema.ProjectRepository
	taskRepo    schema.TaskRepository
	logger      *slog.Logger
}

// NewProjectService constructs a projectService with its dependencies injected
func NewProjectService(
	projectRepo schema.ProjectRepository,
	taskRepo schema.TaskRepository,
	logger *slog.Logger,
) schema.ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
		logger:      logger,
	}
}

// assertAccess checks the project exists AND the requester has read access
// (owner or has an assigned task in the project).
//
// Returns ErrAccessDenied (→ 404) in BOTH cases:
//   - project does not exist
//   - project exists but user has no access
//
// This prevents project-ID enumeration: a stranger cannot distinguish
// "this ID doesn't exist" from "this ID exists but isn't yours".
func (s *projectService) assertAccess(ctx context.Context, projectID, userID uuid.UUID) (*schema.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, schema.ErrAccessDenied
	}

	ok, err := s.projectRepo.UserHasAccess(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("check access: %w", err)
	}
	if !ok {
		return nil, schema.ErrAccessDenied
	}

	return project, nil
}

// assertOwner verifies the requester is the project owner.
// Returns ErrAccessDenied (→ 404) when user has no visibility at all,
// or ErrNotProjectOwner (→ 403) when they can see it but aren't the owner.
func (s *projectService) assertOwner(ctx context.Context, projectID, userID uuid.UUID) (*schema.Project, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, schema.ErrAccessDenied
	}

	ok, err := s.projectRepo.UserHasAccess(ctx, projectID, userID)
	if err != nil {
		return nil, fmt.Errorf("check access: %w", err)
	}
	if !ok {
		// Completely invisible → 404 to prevent enumeration
		return nil, schema.ErrAccessDenied
	}

	if !project.IsOwnedBy(userID) {
		// Visible but not owner → 403
		return nil, schema.ErrNotProjectOwner
	}

	return project, nil
}

// Create validates and persists a new project
func (s *projectService) Create(ctx context.Context, input schema.CreateProjectInput) (*schema.Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, schema.ErrProjectNameRequired
	}

	project := &schema.Project{
		ID:          uuid.New(),
		Name:        input.Name,
		Description: input.Description,
		OwnerID:     input.OwnerID,
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}

	s.logger.Info("project created", "project_id", project.ID, "owner_id", input.OwnerID)
	return project, nil
}

// GetByID fetches a project with its tasks.
func (s *projectService) GetByID(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) (*schema.ProjectWithTasks, error) {
	project, err := s.assertAccess(ctx, id, requesterID)
	if err != nil {
		return nil, err
	}

	filter := schema.TaskFilter{Page: 1, Limit: 500}
	tasks, _, err := s.taskRepo.ListByProject(ctx, id, filter)
	if err != nil {
		return nil, fmt.Errorf("list tasks for project: %w", err)
	}

	if tasks == nil {
		tasks = []schema.Task{}
	}

	return &schema.ProjectWithTasks{
		Project: *project,
		Tasks:   tasks,
	}, nil
}

// List returns paginated projects visible to the requesting user.
func (s *projectService) List(ctx context.Context, userID uuid.UUID, page, limit int) ([]schema.Project, int, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.projectRepo.ListForUser(ctx, userID, page, limit)
}

// Update applies partial updates. Owner-only: 403 for members, 404 for strangers.
func (s *projectService) Update(ctx context.Context, input schema.UpdateProjectInput) (*schema.Project, error) {
	project, err := s.assertOwner(ctx, input.ID, input.RequesterID)
	if err != nil {
		return nil, err
	}

	if input.Name != nil {
		trimmed := strings.TrimSpace(*input.Name)
		if trimmed == "" {
			return nil, schema.ErrProjectNameRequired
		}
		project.Name = trimmed
	}
	if input.Description != nil {
		project.Description = input.Description
	}

	if err := s.projectRepo.Update(ctx, project); err != nil {
		return nil, fmt.Errorf("update project: %w", err)
	}

	s.logger.Info("project updated", "project_id", project.ID)
	return project, nil
}

// Delete removes a project and all its tasks. Owner-only.
func (s *projectService) Delete(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) error {
	if _, err := s.assertOwner(ctx, id, requesterID); err != nil {
		return err
	}

	if err := s.taskRepo.DeleteByProjectID(ctx, id); err != nil {
		return fmt.Errorf("delete project tasks: %w", err)
	}

	if err := s.projectRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}

	s.logger.Info("project deleted", "project_id", id)
	return nil
}

// GetStats returns task statistics. Requires read access (owner or assigned member).
func (s *projectService) GetStats(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) (*schema.ProjectStats, error) {
	if _, err := s.assertAccess(ctx, id, requesterID); err != nil {
		return nil, err
	}
	return s.projectRepo.GetStats(ctx, id)
}
