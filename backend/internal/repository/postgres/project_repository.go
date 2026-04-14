package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// projectRepository is the PostgreSQL implementation of schema.ProjectRepository
type projectRepository struct {
	db *sql.DB
}

// NewProjectRepository creates a new PostgreSQL-backed project repository
func NewProjectRepository(db *sql.DB) schema.ProjectRepository {
	return &projectRepository{db: db}
}

// Create inserts a new project
func (r *projectRepository) Create(ctx context.Context, project *schema.Project) error {
	q := `
		INSERT INTO projects (id, name, description, owner_id, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q,
		project.ID, project.Name, project.Description, project.OwnerID, project.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// GetByID fetches a project by UUID
func (r *projectRepository) GetByID(ctx context.Context, id uuid.UUID) (*schema.Project, error) {
	q := `SELECT id, name, description, owner_id, created_at FROM projects WHERE id = $1`
	p := &schema.Project{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, schema.ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project by id: %w", err)
	}
	return p, nil
}

// ListForUser returns projects owned by or having tasks assigned to the user (paginated)
func (r *projectRepository) ListForUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]schema.Project, int, error) {
	offset := (page - 1) * limit

	countQ := `
		SELECT COUNT(DISTINCT p.id) FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
	`
	var total int
	if err := r.db.QueryRowContext(ctx, countQ, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count projects: %w", err)
	}

	q := `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at
		FROM projects p
		LEFT JOIN tasks t ON t.project_id = p.id
		WHERE p.owner_id = $1 OR t.assignee_id = $1
		ORDER BY p.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.QueryContext(ctx, q, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []schema.Project
	for rows.Next() {
		var p schema.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.OwnerID, &p.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, p)
	}
	return projects, total, rows.Err()
}

// Update modifies an existing project
func (r *projectRepository) Update(ctx context.Context, project *schema.Project) error {
	q := `UPDATE projects SET name = $1, description = $2 WHERE id = $3`
	res, err := r.db.ExecContext(ctx, q, project.Name, project.Description, project.ID)
	if err != nil {
		return fmt.Errorf("update project: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return schema.ErrProjectNotFound
	}
	return nil
}

// Delete removes a project by UUID
func (r *projectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	q := `DELETE FROM projects WHERE id = $1`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return schema.ErrProjectNotFound
	}
	return nil
}

// UserHasAccess returns true if the user is the project owner OR has at least
// one task assigned to them in this project. One query, no N+1.
func (r *projectRepository) UserHasAccess(ctx context.Context, projectID uuid.UUID, userID uuid.UUID) (bool, error) {
	q := `
		SELECT EXISTS (
			SELECT 1 FROM projects WHERE id = $1 AND owner_id = $2
			UNION ALL
			SELECT 1 FROM tasks   WHERE project_id = $1 AND assignee_id = $2
			LIMIT 1
		)
	`
	var ok bool
	if err := r.db.QueryRowContext(ctx, q, projectID, userID).Scan(&ok); err != nil {
		return false, fmt.Errorf("check project access: %w", err)
	}
	return ok, nil
}

// GetStats returns task counts grouped by status and assignee for a project
func (r *projectRepository) GetStats(ctx context.Context, projectID uuid.UUID) (*schema.ProjectStats, error) {
	stats := &schema.ProjectStats{
		ProjectID:       projectID,
		TasksByStatus:   make(map[string]int),
		TasksByAssignee: make(map[string]int),
	}

	statusQ := `SELECT status, COUNT(*) FROM tasks WHERE project_id = $1 GROUP BY status`
	rows, err := r.db.QueryContext(ctx, statusQ, projectID)
	if err != nil {
		return nil, fmt.Errorf("get stats by status: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats.TasksByStatus[status] = count
	}

	assigneeQ := `
		SELECT COALESCE(u.name, 'unassigned'), COUNT(*) 
		FROM tasks t
		LEFT JOIN users u ON u.id = t.assignee_id
		WHERE t.project_id = $1
		GROUP BY u.name
	`
	rows2, err := r.db.QueryContext(ctx, assigneeQ, projectID)
	if err != nil {
		return nil, fmt.Errorf("get stats by assignee: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		var name string
		var count int
		if err := rows2.Scan(&name, &count); err != nil {
			return nil, err
		}
		stats.TasksByAssignee[name] = count
	}

	return stats, nil
}
