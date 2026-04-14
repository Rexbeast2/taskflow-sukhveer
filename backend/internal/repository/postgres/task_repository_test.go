package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestTaskRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewTaskRepository(db)
	ctx := context.Background()

	t.Run("Create Task Success", func(t *testing.T) {
		taskID := uuid.New()
		projectID := uuid.New()
		now := time.Now()
		desc := "Test Description"

		task := &schema.Task{
			ID:          taskID,
			Title:       "Test Task",
			Description: &desc,
			Status:      schema.TaskStatusTodo,
			Priority:    schema.TaskPriorityMedium,
			ProjectID:   projectID,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		mock.ExpectExec("INSERT INTO tasks").
			WithArgs(
				task.ID, task.Title, task.Description, task.Status, task.Priority,
				task.ProjectID, task.AssigneeID, task.DueDate, task.CreatedAt, task.UpdatedAt,
			).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, task)
		assert.NoError(t, err)
	})

	t.Run("GetByID Success", func(t *testing.T) {
		id := uuid.New()
		rows := sqlmock.NewRows([]string{"id", "title", "description", "status", "priority", "project_id", "assignee_id", "due_date", "created_at", "updated_at"}).
			AddRow(id, "Title", nil, "todo", "high", uuid.New(), nil, nil, time.Now(), time.Now())

		mock.ExpectQuery("SELECT (.+) FROM tasks WHERE id =").
			WithArgs(id).
			WillReturnRows(rows)

		task, err := repo.GetByID(ctx, id)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, id, task.ID)
	})
	t.Run("Update Task Success", func(t *testing.T) {
		task := &schema.Task{
			ID:        uuid.New(),
			Title:     "Updated",
			Status:    schema.TaskStatusDone,
			UpdatedAt: time.Now(),
		}

		mock.ExpectExec("UPDATE tasks SET").
			WithArgs(task.Title, task.Description, task.Status, task.Priority, task.AssigneeID, task.DueDate, task.UpdatedAt, task.ID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Update(ctx, task)
		assert.NoError(t, err)
	})

	t.Run("Update Task Not Found", func(t *testing.T) {
		task := &schema.Task{ID: uuid.New()}

		mock.ExpectExec("UPDATE tasks SET").
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.Update(ctx, task)
		assert.ErrorIs(t, err, schema.ErrTaskNotFound)
	})

	t.Run("Delete Success", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectExec("DELETE FROM tasks WHERE id =").
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Delete(ctx, id)
		assert.NoError(t, err)
	})

	t.Run("DeleteByProjectID Success", func(t *testing.T) {
		pid := uuid.New()
		mock.ExpectExec("DELETE FROM tasks WHERE project_id =").
			WithArgs(pid).
			WillReturnResult(sqlmock.NewResult(0, 5))

		err := repo.DeleteByProjectID(ctx, pid)
		assert.NoError(t, err)
	})
}
