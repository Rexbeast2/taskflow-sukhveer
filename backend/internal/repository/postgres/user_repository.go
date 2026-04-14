package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// userRepository is the PostgreSQL implementation of schema.UserRepository
type userRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new PostgreSQL-backed user repository
func NewUserRepository(db *sql.DB) schema.UserRepository {
	return &userRepository{db: db}
}

// Create inserts a new user record
func (r *userRepository) Create(ctx context.Context, user *User) error {
	q := `
		INSERT INTO users (id, name, email, password, created_at)
		VALUES ($1, $2, $3, $4, $5)
	`
	_, err := r.db.ExecContext(ctx, q,
		user.ID, user.Name, user.Email, user.Password, user.CreatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return schema.ErrEmailAlreadyTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetByID fetches a user by their UUID
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*schema.User, error) {
	q := `SELECT id, name, email, password, created_at FROM users WHERE id = $1`
	u := &schema.User{}
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&u.ID, &u.Name, &u.Email, &u.Password, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, schema.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return u, nil
}

// GetByEmail fetches a user by their email address
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*schema.User, error) {
	q := `SELECT id, name, email, password, created_at FROM users WHERE email = $1`
	u := &schema.User{}
	err := r.db.QueryRowContext(ctx, q, email).Scan(
		&u.ID, &u.Name, &u.Email, &u.Password, &u.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, schema.ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return u, nil
}

// ExistsByEmail checks if an email is already registered
func (r *userRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	q := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	var exists bool
	if err := r.db.QueryRowContext(ctx, q, email).Scan(&exists); err != nil {
		return false, fmt.Errorf("check email exists: %w", err)
	}
	return exists, nil
}

// User is a local alias to avoid import cycle in this file
type User = schema.User
