package schema

import (
	"context"

	"github.com/google/uuid"
)

// RegisterInput holds the data needed to register a new user
type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

// LoginInput holds the data needed to authenticate a user
type LoginInput struct {
	Email    string
	Password string
}

// AuthOutput holds the result of a successful auth operation
type AuthOutput struct {
	Token string
	User  *User
}

// AuthService defines the authentication use-case contract
type AuthService interface {
	Register(ctx context.Context, input RegisterInput) (*AuthOutput, error)
	Login(ctx context.Context, input LoginInput) (*AuthOutput, error)
}

// CreateProjectInput holds data to create a new project
type CreateProjectInput struct {
	Name        string
	Description *string
	OwnerID     uuid.UUID
}

// UpdateProjectInput holds data to update an existing project
type UpdateProjectInput struct {
	ID          uuid.UUID
	Name        *string
	Description *string
	RequesterID uuid.UUID
}

// ProjectService defines the project use-case contract
type ProjectService interface {
	Create(ctx context.Context, input CreateProjectInput) (*Project, error)
	GetByID(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) (*ProjectWithTasks, error)
	List(ctx context.Context, userID uuid.UUID, page, limit int) ([]Project, int, error)
	Update(ctx context.Context, input UpdateProjectInput) (*Project, error)
	Delete(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) error
	GetStats(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) (*ProjectStats, error)
}

// CreateTaskInput holds data to create a new task
type CreateTaskInput struct {
	Title       string
	Description *string
	Status      TaskStatus
	Priority    TaskPriority
	ProjectID   uuid.UUID
	AssigneeID  *uuid.UUID
	DueDate     *string
	RequesterID uuid.UUID
}

// UpdateTaskInput holds data to update an existing task
type UpdateTaskInput struct {
	ID          uuid.UUID
	Title       *string
	Description *string
	Status      *TaskStatus
	Priority    *TaskPriority
	AssigneeID  *uuid.UUID
	DueDate     *string
	RequesterID uuid.UUID
}

// TaskService defines the task use-case contract
type TaskService interface {
	Create(ctx context.Context, input CreateTaskInput) (*Task, error)
	ListByProject(ctx context.Context, projectID uuid.UUID, filter TaskFilter, requesterID uuid.UUID) ([]Task, int, error)
	Update(ctx context.Context, input UpdateTaskInput) (*Task, error)
	Delete(ctx context.Context, id uuid.UUID, requesterID uuid.UUID) error
}

// TokenService defines the JWT token management contract
type TokenService interface {
	Generate(userID uuid.UUID, email string) (string, error)
	Validate(token string) (*TokenClaims, error)
}

// TokenClaims holds the parsed JWT claims
type TokenClaims struct {
	UserID uuid.UUID
	Email  string
}
