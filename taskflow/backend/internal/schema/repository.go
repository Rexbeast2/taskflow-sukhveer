package schema

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines the persistence contract for User entities.
// Depends on abstractions, not concretions (Dependency Inversion Principle).
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
}

// ProjectRepository defines the persistence contract for Project entities.
type ProjectRepository interface {
	Create(ctx context.Context, project *Project) error
	GetByID(ctx context.Context, id uuid.UUID) (*Project, error)
	// ListForUser returns projects where user is owner OR has assigned tasks
	ListForUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]Project, int, error)
	Update(ctx context.Context, project *Project) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetStats(ctx context.Context, projectID uuid.UUID) (*ProjectStats, error)
	// UserHasAccess returns true if the user is the owner OR has at least one
	// task assigned to them in this project. Single efficient SQL query.
	UserHasAccess(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (bool, error)
}

// TaskRepository defines the persistence contract for Task entities.
type TaskRepository interface {
	Create(ctx context.Context, task *Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*Task, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, filter TaskFilter) ([]Task, int, error)
	Update(ctx context.Context, task *Task) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByProjectID(ctx context.Context, projectID uuid.UUID) error
}
