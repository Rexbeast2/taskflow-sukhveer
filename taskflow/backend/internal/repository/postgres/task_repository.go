package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// taskRepository is the PostgreSQL implementation of schema.TaskRepository
type taskRepository struct {
	db *sql.DB
}

// NewTaskRepository creates a new PostgreSQL-backed task repository
func NewTaskRepository(db *sql.DB) schema.TaskRepository {
	return &taskRepository{db: db}
}

// Create inserts a new task record
func (r *taskRepository) Create(ctx context.Context, task *schema.Task) error {
	q := `
		INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, q,
		task.ID, task.Title, task.Description, task.Status, task.Priority,
		task.ProjectID, task.AssigneeID, task.DueDate, task.CreatedAt, task.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	return nil
}

// GetByID fetches a task by UUID
func (r *taskRepository) GetByID(ctx context.Context, id uuid.UUID) (*schema.Task, error) {
	q := `
		SELECT id, title, description, status, priority, project_id, assignee_id, due_date, created_at, updated_at
		FROM tasks WHERE id = $1
	`
	t := &schema.Task{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
		&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, schema.ErrTaskNotFound
		}
		return nil, fmt.Errorf("get task by id: %w", err)
	}
	return t, nil
}

// ListByProject fetches tasks for a project with optional filters and pagination
func (r *taskRepository) ListByProject(ctx context.Context, projectID uuid.UUID, filter schema.TaskFilter) ([]schema.Task, int, error) {
	args := []interface{}{projectID}
	conditions := []string{"project_id = $1"}
	argIdx := 2

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.AssigneeID != nil {
		conditions = append(conditions, fmt.Sprintf("assignee_id = $%d", argIdx))
		args = append(args, *filter.AssigneeID)
		argIdx++
	}

	where := "WHERE " + strings.Join(conditions, " AND ")

	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM tasks %s`, where)
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	args = append(args, limit, offset)
	q := fmt.Sprintf(`
		SELECT id, title, description, status, priority, project_id, assignee_id, due_date, created_at, updated_at
		FROM tasks %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []schema.Task
	for rows.Next() {
		var t schema.Task
		if err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.Status, &t.Priority,
			&t.ProjectID, &t.AssigneeID, &t.DueDate, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	return tasks, total, rows.Err()
}

// Update modifies an existing task
func (r *taskRepository) Update(ctx context.Context, task *schema.Task) error {
	q := `
		UPDATE tasks
		SET title=$1, description=$2, status=$3, priority=$4, assignee_id=$5, due_date=$6, updated_at=$7
		WHERE id=$8
	`
	res, err := r.db.ExecContext(ctx, q,
		task.Title, task.Description, task.Status, task.Priority,
		task.AssigneeID, task.DueDate, task.UpdatedAt, task.ID,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return schema.ErrTaskNotFound
	}
	return nil
}

// Delete removes a task by UUID
func (r *taskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM tasks WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return schema.ErrTaskNotFound
	}
	return nil
}

// DeleteByProjectID removes all tasks belonging to a project
func (r *taskRepository) DeleteByProjectID(ctx context.Context, projectID uuid.UUID) error {
	q := `DELETE FROM tasks WHERE project_id = $1`
	if _, err := r.db.ExecContext(ctx, q, projectID); err != nil {
		return fmt.Errorf("delete tasks by project: %w", err)
	}
	return nil
}
