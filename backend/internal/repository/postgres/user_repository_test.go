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

func TestUserRepository(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)
	ctx := context.Background()
	now := time.Now() // Use a real time object

	t.Run("GetByEmail - Success", func(t *testing.T) {
		email := "find@me.com"

		// FIX: Pass 'now' instead of 'nil' for the last column
		rows := sqlmock.NewRows([]string{"id", "name", "email", "password", "created_at"}).
			AddRow(uuid.New(), "User", email, "hash", now)

		mock.ExpectQuery("SELECT (.+) FROM users WHERE email =").
			WithArgs(email).
			WillReturnRows(rows)

		res, err := repo.GetByEmail(ctx, email)

		// Check error first to avoid nil pointer panic
		assert.NoError(t, err)
		if assert.NotNil(t, res) {
			assert.Equal(t, email, res.Email)
		}
	})

	t.Run("GetByID - Not Found", func(t *testing.T) {
		id := uuid.New()
		mock.ExpectQuery("SELECT (.+) FROM users WHERE id =").
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		res, err := repo.GetByID(ctx, id)
		assert.ErrorIs(t, err, schema.ErrUserNotFound)
		assert.Nil(t, res)
	})
}
