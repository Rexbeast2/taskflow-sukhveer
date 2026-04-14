package schema

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// TaskStatus represents the current state of a task
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusDone       TaskStatus = "done"
)

// TaskPriority represents the importance level of a task
type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
)

// Task represents a task entity
type Task struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	Title       string       `json:"title" db:"title"`
	Description *string      `json:"description,omitempty" db:"description"`
	Status      TaskStatus   `json:"status" db:"status"`
	Priority    TaskPriority `json:"priority" db:"priority"`
	ProjectID   uuid.UUID    `json:"project_id" db:"project_id"`
	AssigneeID  *uuid.UUID   `json:"assignee_id,omitempty" db:"assignee_id"`
	DueDate     *time.Time   `json:"due_date,omitempty" db:"due_date"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
}

// TaskFilter holds optional filters for listing tasks
type TaskFilter struct {
	Status     *TaskStatus
	AssigneeID *uuid.UUID
	Page       int
	Limit      int
}

// Validation errors
var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskTitleRequired = errors.New("task title is required")
	ErrInvalidStatus     = errors.New("invalid task status")
	ErrInvalidPriority   = errors.New("invalid task priority")
)

// Validate performs schema-level validation on a Task
func (t *Task) Validate() error {
	if t.Title == "" {
		return ErrTaskTitleRequired
	}
	if !t.Status.IsValid() {
		return ErrInvalidStatus
	}
	if !t.Priority.IsValid() {
		return ErrInvalidPriority
	}
	return nil
}

// IsValid checks if a TaskStatus is one of the allowed values
func (s TaskStatus) IsValid() bool {
	switch s {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusDone:
		return true
	}
	return false
}

// IsValid checks if a TaskPriority is one of the allowed values
func (p TaskPriority) IsValid() bool {
	switch p {
	case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh:
		return true
	}
	return false
}
