package schema

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// Project represents a project entity
type Project struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description *string   `json:"description,omitempty" db:"description"`
	OwnerID     uuid.UUID `json:"owner_id" db:"owner_id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// ProjectWithTasks is a project with its associated tasks
type ProjectWithTasks struct {
	Project
	Tasks []Task `json:"tasks"`
}

// ProjectStats holds aggregated statistics for a project
type ProjectStats struct {
	ProjectID       uuid.UUID      `json:"project_id"`
	TasksByStatus   map[string]int `json:"tasks_by_status"`
	TasksByAssignee map[string]int `json:"tasks_by_assignee"`
}

// Validation errors
var (
	ErrProjectNotFound     = errors.New("project not found")
	ErrProjectNameRequired = errors.New("project name is required")
	ErrNotProjectOwner     = errors.New("only the project owner can perform this action")
	// ErrAccessDenied is returned as 404 (not 403) on read operations to prevent
	// project ID enumeration — the caller cannot distinguish "doesn't exist" from
	// "exists but you can't see it".
	ErrAccessDenied = errors.New("not found")
)

// Validate performs schema-level validation on a Project
func (p *Project) Validate() error {
	if p.Name == "" {
		return ErrProjectNameRequired
	}
	return nil
}

// IsOwnedBy checks if the project belongs to a given user
func (p *Project) IsOwnedBy(userID uuid.UUID) bool {
	return p.OwnerID == userID
}
