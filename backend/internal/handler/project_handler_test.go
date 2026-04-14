package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/Sukhveer/taskflow/internal/middleware"
	"github.com/Sukhveer/taskflow/internal/schema"
)

func withUserContext(r *http.Request, userID uuid.UUID) *http.Request {
	ctx := middleware.SetUserContext(r.Context(), userID, "test@example.com")
	return r.WithContext(ctx)
}

func TestProjectHandler_CreateProject(t *testing.T) {
	svc := new(mockProjectService)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewProjectHandler(svc, logger)

	userID := uuid.New()

	t.Run("successfully creates project", func(t *testing.T) {
		input := createProjectRequest{Name: "New Project"}
		body, _ := json.Marshal(input)

		req := httptest.NewRequest(http.MethodPost, "/projects", bytes.NewReader(body))
		req = withUserContext(req, userID)
		rr := httptest.NewRecorder()

		svc.On("Create", mock.Anything, schema.CreateProjectInput{
			Name:    "New Project",
			OwnerID: userID,
		}).Return(&schema.Project{Name: "New Project", OwnerID: userID}, nil).Once()

		h.CreateProject(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
		svc.AssertExpectations(t)
	})

}

func TestProjectHandler_GetProject(t *testing.T) {
	svc := new(mockProjectService)
	h := NewProjectHandler(svc, discardLogger()) // assume discardLogger helper exists

	userID := uuid.New()
	projectID := uuid.New()

	t.Run("success returns project", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String(), nil)
		req = withUserContext(req, userID)

		// Manually inject chi URL param for test
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", projectID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()

		svc.On("GetByID", mock.Anything, projectID, userID).
			Return(&schema.ProjectWithTasks{Project: schema.Project{ID: projectID}}, nil).Once()

		h.GetProject(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp schema.ProjectWithTasks
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, projectID, resp.ID)
	})

	t.Run("returns 404 when service returns access denied", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects/"+projectID.String(), nil)
		req = withUserContext(req, userID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", projectID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

		rr := httptest.NewRecorder()

		// Note: schemaErrorToHTTP must handle schema.ErrAccessDenied -> 404
		svc.On("GetByID", mock.Anything, projectID, userID).
			Return(nil, schema.ErrAccessDenied).Once()

		h.GetProject(rr, req)

		assert.Equal(t, http.StatusNotFound, rr.Code)
	})
}

func TestProjectHandler_ListProjects(t *testing.T) {
	svc := new(mockProjectService)
	h := NewProjectHandler(svc, discardLogger())
	userID := uuid.New()

	t.Run("success with pagination params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/projects?page=2&limit=10", nil)
		req = withUserContext(req, userID)
		rr := httptest.NewRecorder()

		projects := []schema.Project{{ID: uuid.New(), Name: "P1"}}
		svc.On("List", mock.Anything, userID, 2, 10).Return(projects, 1, nil).Once()

		h.ListProjects(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		var resp paginatedResponse
		json.Unmarshal(rr.Body.Bytes(), &resp)
		assert.Equal(t, 2, resp.Page)
		assert.Equal(t, 10, resp.Limit)
		assert.NotEmpty(t, resp.Data)
	})
}

func TestProjectHandler_DeleteProject(t *testing.T) {
	svc := new(mockProjectService)
	h := NewProjectHandler(svc, discardLogger())
	userID := uuid.New()
	projectID := uuid.New()

	t.Run("success returns 204", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, "/projects/"+projectID.String(), nil)
		req = withUserContext(req, userID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", projectID.String())
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rr := httptest.NewRecorder()

		svc.On("Delete", mock.Anything, projectID, userID).Return(nil).Once()

		h.DeleteProject(rr, req)

		assert.Equal(t, http.StatusNoContent, rr.Code)
	})
}
