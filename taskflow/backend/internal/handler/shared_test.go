package handler

import (
	"context"
	"io"
	"log/slog"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type mockProjectService struct {
	mock.Mock
}

type mockAuthService struct {
	mock.Mock
}
type mockTaskService struct {
	mock.Mock
} // Helper to provide a logger that writes nowhere
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// Ensure the TokenService mock is available in the handler package tests
type mockTokenService struct{ mock.Mock }

func (m *mockTokenService) Generate(id uuid.UUID, email string) (string, error) {
	args := m.Called(id, email)
	return args.String(0), args.Error(1)
}

func (m *mockTokenService) Validate(token string) (*schema.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.TokenClaims), args.Error(1)
}

func (m *mockTaskService) Create(ctx context.Context, i schema.CreateTaskInput) (*schema.Task, error) {
	args := m.Called(ctx, i)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Task), args.Error(1)
}

func (m *mockTaskService) ListByProject(ctx context.Context, pID uuid.UUID, f schema.TaskFilter, rID uuid.UUID) ([]schema.Task, int, error) {
	args := m.Called(ctx, pID, f, rID)
	return args.Get(0).([]schema.Task), args.Int(1), args.Error(2)
}

func (m *mockTaskService) Update(ctx context.Context, i schema.UpdateTaskInput) (*schema.Task, error) {
	args := m.Called(ctx, i)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Task), args.Error(1)
}

func (m *mockTaskService) Delete(ctx context.Context, id uuid.UUID, rID uuid.UUID) error {
	return m.Called(ctx, id, rID).Error(0)
}

func (m *mockAuthService) Register(ctx context.Context, input schema.RegisterInput) (*schema.AuthOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.AuthOutput), args.Error(1)
}

func (m *mockAuthService) Login(ctx context.Context, input schema.LoginInput) (*schema.AuthOutput, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.AuthOutput), args.Error(1)
}
func (m *mockProjectService) Create(ctx context.Context, input schema.CreateProjectInput) (*schema.Project, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Project), args.Error(1)
}

func (m *mockProjectService) GetByID(ctx context.Context, id uuid.UUID, reqID uuid.UUID) (*schema.ProjectWithTasks, error) {
	args := m.Called(ctx, id, reqID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.ProjectWithTasks), args.Error(1)
}

func (m *mockProjectService) List(ctx context.Context, uID uuid.UUID, p, l int) ([]schema.Project, int, error) {
	args := m.Called(ctx, uID, p, l)
	return args.Get(0).([]schema.Project), args.Int(1), args.Error(2)
}

func (m *mockProjectService) Update(ctx context.Context, input schema.UpdateProjectInput) (*schema.Project, error) {
	args := m.Called(ctx, input)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Project), args.Error(1)
}

func (m *mockProjectService) Delete(ctx context.Context, id uuid.UUID, reqID uuid.UUID) error {
	return m.Called(ctx, id, reqID).Error(0)
}

func (m *mockProjectService) GetStats(ctx context.Context, id uuid.UUID, reqID uuid.UUID) (*schema.ProjectStats, error) {
	args := m.Called(ctx, id, reqID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.ProjectStats), args.Error(1)
}
