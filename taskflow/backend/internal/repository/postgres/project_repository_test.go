package postgres

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProjectRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewProjectRepository(db)
	ctx := context.Background()
	projectID := uuid.New()
	userID := uuid.New()

	t.Run("Create Success", func(t *testing.T) {
		p := &schema.Project{
			ID:        projectID,
			Name:      "New Project",
			OwnerID:   userID,
			CreatedAt: time.Now(),
		}
		mock.ExpectExec("INSERT INTO projects").
			WithArgs(p.ID, p.Name, p.Description, p.OwnerID, p.CreatedAt).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.Create(ctx, p)
		assert.NoError(t, err)
	})

	t.Run("GetByID - Not Found", func(t *testing.T) {
		mock.ExpectQuery("SELECT (.+) FROM projects WHERE id =").
			WithArgs(projectID).
			WillReturnError(sql.ErrNoRows)

		res, err := repo.GetByID(ctx, projectID)
		assert.ErrorIs(t, err, schema.ErrProjectNotFound)
		assert.Nil(t, res)
	})

	t.Run("UserHasAccess - True", func(t *testing.T) {
		mock.ExpectQuery("SELECT EXISTS").
			WithArgs(projectID, userID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		ok, err := repo.UserHasAccess(ctx, projectID, userID)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("GetStats Success", func(t *testing.T) {
		mock.ExpectQuery("SELECT status, COUNT").WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"status", "count"}).AddRow("todo", 2))

		mock.ExpectQuery("SELECT COALESCE").WithArgs(projectID).
			WillReturnRows(sqlmock.NewRows([]string{"name", "count"}).AddRow("Alice", 1))

		stats, err := repo.GetStats(ctx, projectID)
		assert.NoError(t, err)
		assert.Equal(t, 2, stats.TasksByStatus["todo"])
		assert.Equal(t, 1, stats.TasksByAssignee["Alice"])
	})
}
