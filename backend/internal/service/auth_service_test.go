package service

import (
	"context"
	"testing"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthService_Register_Advanced(t *testing.T) {
	uRepo := new(mockUserRepo)
	tSvc := new(mockTokenService)
	svc := NewAuthService(uRepo, tSvc, discardLogger())
	ctx := context.Background()

	tests := []struct {
		name    string
		input   schema.RegisterInput
		setup   func()
		wantErr error
	}{
		{
			name:    "fail - empty name",
			input:   schema.RegisterInput{Name: " ", Email: "a@b.com", Password: "password123"},
			wantErr: schema.ErrNameRequired,
		},
		{
			name:    "fail - invalid email format",
			input:   schema.RegisterInput{Name: "John", Email: "not-an-email", Password: "password123"},
			wantErr: schema.ErrInvalidEmail,
		},
		{
			name:    "fail - password too short",
			input:   schema.RegisterInput{Name: "John", Email: "john@test.com", Password: "short"},
			wantErr: schema.ErrPasswordTooShort,
		},
		{
			name:  "fail - email already exists",
			input: schema.RegisterInput{Name: "John", Email: "taken@test.com", Password: "password123"},
			setup: func() {
				uRepo.On("ExistsByEmail", ctx, "taken@test.com").Return(true, nil).Once()
			},
			wantErr: schema.ErrEmailAlreadyTaken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}
			_, err := svc.Register(ctx, tt.input)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestAuthService_Login_Advanced(t *testing.T) {
	uRepo := new(mockUserRepo)
	tSvc := new(mockTokenService)
	svc := NewAuthService(uRepo, tSvc, discardLogger())
	ctx := context.Background()

	t.Run("fail - user not found returns generic error", func(t *testing.T) {
		uRepo.On("GetByEmail", ctx, "missing@test.com").Return(nil, schema.ErrUserNotFound).Once()
		_, err := svc.Login(ctx, schema.LoginInput{Email: "missing@test.com", Password: "any"})
		assert.ErrorIs(t, err, schema.ErrInvalidPassword) // Security: don't leak user existence
	})

	t.Run("fail - wrong password", func(t *testing.T) {
		hash, _ := bcrypt.GenerateFromPassword([]byte("correct-pass"), 12)
		uRepo.On("GetByEmail", ctx, "user@test.com").Return(&schema.User{Password: string(hash)}, nil).Once()

		_, err := svc.Login(ctx, schema.LoginInput{Email: "user@test.com", Password: "wrong-pass"})
		assert.ErrorIs(t, err, schema.ErrInvalidPassword)
	})
}
