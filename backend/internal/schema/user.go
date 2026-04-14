package schema

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User represents the core user entity
type User struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Validation errors
var (
	ErrUserNotFound      = errors.New("user not found")
	ErrEmailAlreadyTaken = errors.New("email already taken")
	ErrInvalidPassword   = errors.New("invalid password")
	ErrInvalidEmail      = errors.New("invalid email")
	ErrNameRequired      = errors.New("name is required")
	ErrPasswordTooShort  = errors.New("password must be at least 8 characters")
)

// Validate performs schema-level validation on a User
func (u *User) Validate() error {
	if u.Name == "" {
		return ErrNameRequired
	}
	if u.Email == "" {
		return ErrInvalidEmail
	}
	return nil
}
