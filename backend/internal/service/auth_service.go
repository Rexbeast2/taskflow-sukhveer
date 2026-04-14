package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/Sukhveer/taskflow/internal/schema"
)

// authService implements schema.AuthService
type authService struct {
	userRepo     schema.UserRepository
	tokenService schema.TokenService
	logger       *slog.Logger
}

// NewAuthService constructs an authService with its dependencies injected
func NewAuthService(
	userRepo schema.UserRepository,
	tokenService schema.TokenService,
	logger *slog.Logger,
) schema.AuthService {
	return &authService{
		userRepo:     userRepo,
		tokenService: tokenService,
		logger:       logger,
	}
}

// Register validates input, hashes the password, persists the user, and returns a JWT
func (s *authService) Register(ctx context.Context, input schema.RegisterInput) (*schema.AuthOutput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	if input.Name == "" {
		return nil, schema.ErrNameRequired
	}
	if !isValidEmail(input.Email) {
		return nil, schema.ErrInvalidEmail
	}
	if len(input.Password) < 8 {
		return nil, schema.ErrPasswordTooShort
	}

	exists, err := s.userRepo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("check email: %w", err)
	}
	if exists {
		return nil, schema.ErrEmailAlreadyTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &schema.User{
		ID:        uuid.New(),
		Name:      input.Name,
		Email:     input.Email,
		Password:  string(hashed),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.tokenService.Generate(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	s.logger.Info("user registered", "user_id", user.ID, "email", user.Email)

	return &schema.AuthOutput{Token: token, User: user}, nil
}

// Login verifies credentials and returns a JWT
func (s *authService) Login(ctx context.Context, input schema.LoginInput) (*schema.AuthOutput, error) {
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))

	user, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err != nil {
		// Do not reveal whether email exists — return generic error
		return nil, schema.ErrInvalidPassword
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, schema.ErrInvalidPassword
	}

	token, err := s.tokenService.Generate(user.ID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	s.logger.Info("user logged in", "user_id", user.ID)

	return &schema.AuthOutput{Token: token, User: user}, nil
}

// isValidEmail performs a simple email format check
func isValidEmail(email string) bool {
	at := strings.Index(email, "@")
	if at < 1 {
		return false
	}
	dot := strings.LastIndex(email[at:], ".")
	return dot > 1
}
