package service

import (
	"context"
	"testing"

	"github.com/Sukhveer/taskflow/internal/schema"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskService_Create_Advanced(t *testing.T) {
	tRepo := new(mockTaskRepo)
	pRepo := new(mockProjectRepo)
	svc := NewTaskService(tRepo, pRepo, discardLogger())
	ctx := context.Background()
	pID, uID := uuid.New(), uuid.New()

	t.Run("fail - invalid due date format", func(t *testing.T) {
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID, OwnerID: uID}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(true, nil).Once()

		badDate := "01-01-2024" // Wrong format
		_, err := svc.Create(ctx, schema.CreateTaskInput{
			Title: "Task", ProjectID: pID, RequesterID: uID, DueDate: &badDate,
		})
		assert.Contains(t, err.Error(), "invalid due_date format")
	})
}

func TestTaskService_Update_Permissions(t *testing.T) {
	tRepo := new(mockTaskRepo)
	pRepo := new(mockProjectRepo)
	svc := NewTaskService(tRepo, pRepo, discardLogger())
	ctx := context.Background()
	tID, pID, uID := uuid.New(), uuid.New(), uuid.New()

	t.Run("success - current assignee can update task status", func(t *testing.T) {
		task := &schema.Task{ID: tID, ProjectID: pID, AssigneeID: &uID}
		// uID is NOT the owner
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID, OwnerID: uuid.New()}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(true, nil).Once()
		tRepo.On("GetByID", ctx, tID).Return(task, nil).Once()
		tRepo.On("Update", ctx, mock.Anything).Return(nil).Once()

		status := schema.TaskStatusInProgress
		res, err := svc.Update(ctx, schema.UpdateTaskInput{ID: tID, RequesterID: uID, Status: &status})
		assert.NoError(t, err)
		assert.Equal(t, schema.TaskStatusInProgress, res.Status)
	})

	t.Run("fail - title cannot be updated to empty string", func(t *testing.T) {
		task := &schema.Task{ID: tID, ProjectID: pID, AssigneeID: &uID}
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID, OwnerID: uID}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(true, nil).Once()
		tRepo.On("GetByID", ctx, tID).Return(task, nil).Once()

		emptyTitle := "  "
		_, err := svc.Update(ctx, schema.UpdateTaskInput{ID: tID, RequesterID: uID, Title: &emptyTitle})
		assert.ErrorIs(t, err, schema.ErrTaskTitleRequired)
	})
}

func TestTaskService_ListByProject_Pagination(t *testing.T) {
	tRepo := new(mockTaskRepo)
	pRepo := new(mockProjectRepo)
	svc := NewTaskService(tRepo, pRepo, discardLogger())
	ctx := context.Background()
	pID, uID := uuid.New(), uuid.New()

	t.Run("defaults pagination values if zero or negative", func(t *testing.T) {
		pRepo.On("GetByID", ctx, pID).Return(&schema.Project{ID: pID}, nil).Once()
		pRepo.On("UserHasAccess", ctx, pID, uID).Return(true, nil).Once()

		// Expect page 1, limit 20
		tRepo.On("ListByProject", ctx, pID, mock.MatchedBy(func(f schema.TaskFilter) bool {
			return f.Page == 1 && f.Limit == 20
		})).Return([]schema.Task{}, 0, nil).Once()

		_, _, err := svc.ListByProject(ctx, pID, schema.TaskFilter{Page: 0, Limit: -5}, uID)
		assert.NoError(t, err)
	})
}
