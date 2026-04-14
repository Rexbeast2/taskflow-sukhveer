package handler

import (
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

func TestNewRouter(t *testing.T) {
	// 1. Setup all dependencies
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	mAuthSvc := new(mockAuthService)
	mProjSvc := new(mockProjectService)
	mTaskSvc := new(mockTaskService)
	mTokenSvc := new(mockTokenService) // From shared_test.go

	authH := NewAuthHandler(mAuthSvc, logger)
	projH := NewProjectHandler(mProjSvc, logger)
	// Assuming TaskHandler exists similar to others
	taskH := &TaskHandler{taskService: mTaskSvc, logger: logger}

	router := NewRouter(authH, projH, taskH, mTokenSvc, logger)

	t.Run("Health check is public", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Body.String(), "ok")
	})

	// t.Run("Auth routes are public", func(t *testing.T) {
	// 	// We don't need to test full logic, just that it reaches the handler
	// 	mAuthSvc.On("Login", mock.Anything, mock.Anything).
	// 		Return(&schema.AuthOutput{Token: "t"}, nil).Once()

	// 	loginJSON := `{"email":"test@test.com","password":"password123"}`
	// 	req := httptest.NewRequest(http.MethodPost, "/auth/login", io.NopCloser([]byte(loginJSON)))
	// 	rr := httptest.NewRecorder()
	// 	router.ServeHTTP(rr, req)

	// 	assert.Equal(t, http.StatusOK, rr.Code)
	// })

	t.Run("Protected routes return 401 without token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		// middleware.Authenticate should reject this
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
	})

	t.Run("Protected routes work with valid token", func(t *testing.T) {
		userID := uuid.New()
		token := "valid-token"

		// 1. Mock token validation
		mTokenSvc.On("Validate", token).Return(&schema.TokenClaims{
			UserID: userID,
			Email:  "user@test.com",
		}, nil).Once()

		// 2. Mock the service called by the handler
		mProjSvc.On("List", mock.Anything, userID, 1, 20).
			Return([]schema.Project{}, 0, nil).Once()

		req := httptest.NewRequest(http.MethodGet, "/projects", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("Nested routes parse ID correctly", func(t *testing.T) {
		userID := uuid.New()
		projectID := uuid.New()
		token := "valid-token"

		mTokenSvc.On("Validate", token).Return(&schema.TokenClaims{UserID: userID}, nil).Once()

		// Verify that the ID in the URL is passed correctly to the service
		mProjSvc.On("GetByID", mock.Anything, projectID, userID).
			Return(&schema.ProjectWithTasks{}, nil).Once()

		url := "/projects/" + projectID.String()
		req := httptest.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rr := httptest.NewRecorder()

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		mProjSvc.AssertExpectations(t)
	})
}
