package service

import (
	"context"
	"testing"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestProjectService_AccessControl(t *testing.T) {
	pRepo := new(mockProjectRepo)
	tRepo := new(mockTaskRepo)
	svc := NewProjectService(pRepo, tRepo, discardLogger())
	ctx := context.Background()
	pID, uID := uuid.New(), uuid.New()

	t.Run("GetByID - fail - project exists but user has no access returns 404", func(t *testing.T) {
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(false, nil).Once()

		_, err := svc.GetByID(ctx, pID, uID)
		assert.ErrorIs(t, err, schema.ErrAccessDenied)
	})

	t.Run("Update - fail - user is member but not owner returns 403", func(t *testing.T) {
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID, OwnerID: uuid.New()}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(true, nil).Once() // User is in project

		newName := "New Name"
		_, err := svc.Update(ctx, schema.UpdateProjectInput{ID: pID, RequesterID: uID, Name: &newName})
		assert.ErrorIs(t, err, schema.ErrNotProjectOwner) // 403
	})
}

func TestProjectService_Stats(t *testing.T) {
	pRepo := new(mockProjectRepo)
	tRepo := new(mockTaskRepo)
	svc := NewProjectService(pRepo, tRepo, discardLogger())
	ctx := context.Background()
	pID, uID := uuid.New(), uuid.New()

	t.Run("GetStats success", func(t *testing.T) {
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(true, nil).Once()
		pRepo.On("GetStats", ctx, pID).Return(&schema.ProjectStats{ProjectID: pID}, nil).Once()

		stats, err := svc.GetStats(ctx, pID, uID)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})
}
