package middleware

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock TokenService
type mockTokenService struct {
	mock.Mock
}

func (m *mockTokenService) Generate(userID uuid.UUID, email string) (string, error) {
	args := m.Called(userID, email)
	return args.String(0), args.Error(1)
}

func (m *mockTokenService) Validate(token string) (*schema.TokenClaims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*schema.TokenClaims), args.Error(1)
}

func TestAuthenticate(t *testing.T) {
	tokenSvc := new(mockTokenService)
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mw := Authenticate(tokenSvc)(nextHandler)

	t.Run("success - valid token", func(t *testing.T) {
		userID := uuid.New()
		tokenSvc.On("Validate", "valid-token").Return(&schema.TokenClaims{
			UserID: userID,
			Email:  "test@test.com",
		}, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rr := httptest.NewRecorder()

		mw.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		tokenSvc.AssertExpectations(t)
	})

	t.Run("fail - missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rr := httptest.NewRecorder()

		mw.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "missing authorization header")
	})

	t.Run("fail - invalid format", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Basic user:pass")
		rr := httptest.NewRecorder()

		mw.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid authorization header format")
	})

	t.Run("fail - token service error", func(t *testing.T) {
		tokenSvc.On("Validate", "bad-token").Return(nil, errors.New("invalid")).Once()

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", "Bearer bad-token")
		rr := httptest.NewRecorder()

		mw.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid or expired token")
	})
}

func TestContextHelpers(t *testing.T) {
	userID := uuid.New()
	email := "dev@taskflow.com"

	t.Run("UserIDFromContext - Success", func(t *testing.T) {
		ctx := SetUserContext(context.Background(), userID, email)
		extractedID := UserIDFromContext(ctx)
		assert.Equal(t, userID, extractedID)
	})

	t.Run("UserIDFromContext - Panic on missing", func(t *testing.T) {
		assert.Panics(t, func() {
			UserIDFromContext(context.Background())
		})
	})

	t.Run("EmailFromContext", func(t *testing.T) {
		ctx := SetUserContext(context.Background(), userID, email)
		assert.Equal(t, email, EmailFromContext(ctx))
	})
}

func TestRecoverer(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	mw := Recoverer(logger)(panicHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	assert.NotPanics(t, func() {
		mw.ServeHTTP(rr, req)
	})

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal server error")
}

func TestRequestLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	mw := RequestLogger(logger)(nextHandler)

	req := httptest.NewRequest(http.MethodPost, "/test-path", nil)
	rr := httptest.NewRecorder()

	mw.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusAccepted, rr.Code)
}
