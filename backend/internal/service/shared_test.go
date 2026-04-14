package service

import (
	"context"
	"io"
	"log/slog"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

// --- USER REPO MOCK ---
type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, u *schema.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *mockUserRepo) GetByEmail(ctx context.Context, e string) (*schema.User, error) {
	args := m.Called(ctx, e)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}
func (m *mockUserRepo) ExistsByEmail(ctx context.Context, e string) (bool, error) {
	args := m.Called(ctx, e)
	return args.Bool(0), args.Error(1)
}
func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*schema.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.User), args.Error(1)
}

// --- TOKEN SERVICE MOCK ---
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

// --- PROJECT REPO MOCK ---
type mockProjectRepo struct{ mock.Mock }

func (m *mockProjectRepo) Create(ctx context.Context, p *schema.Project) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockProjectRepo) GetByID(ctx context.Context, id uuid.UUID) (*schema.Project, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Project), args.Error(1)
}
func (m *mockProjectRepo) UserHasAccess(ctx context.Context, pID, uID uuid.UUID) (bool, error) {
	args := m.Called(ctx, pID, uID)
	return args.Bool(0), args.Error(1)
}
func (m *mockProjectRepo) ListForUser(ctx context.Context, uID uuid.UUID, p, l int) ([]schema.Project, int, error) {
	args := m.Called(ctx, uID, p, l)
	return args.Get(0).([]schema.Project), args.Int(1), args.Error(2)
}
func (m *mockProjectRepo) Update(ctx context.Context, p *schema.Project) error {
	return m.Called(ctx, p).Error(0)
}
func (m *mockProjectRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockProjectRepo) GetStats(ctx context.Context, id uuid.UUID) (*schema.ProjectStats, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.ProjectStats), args.Error(1)
}

// --- TASK REPO MOCK ---
type mockTaskRepo struct{ mock.Mock }

func (m *mockTaskRepo) Create(ctx context.Context, t *schema.Task) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTaskRepo) GetByID(ctx context.Context, id uuid.UUID) (*schema.Task, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.Task), args.Error(1)
}
func (m *mockTaskRepo) ListByProject(ctx context.Context, id uuid.UUID, f schema.TaskFilter) ([]schema.Task, int, error) {
	args := m.Called(ctx, id, f)
	return args.Get(0).([]schema.Task), args.Int(1), args.Error(2)
}
func (m *mockTaskRepo) Update(ctx context.Context, t *schema.Task) error {
	return m.Called(ctx, t).Error(0)
}
func (m *mockTaskRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}
func (m *mockTaskRepo) DeleteByProjectID(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
