package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrTaskNotFound identifies missing workflow task records.
var ErrTaskNotFound = errors.New("workflow task not found")

// TaskStore persists workflow task records.
type TaskStore interface {
	SaveTask(ctx context.Context, task Task) (Task, error)
	GetTask(ctx context.Context, taskID string) (Task, error)
}

// Service persists promoted tasks through a caller-provided store.
type Service struct {
	store TaskStore
}

// NewService constructs the workflow promotion service.
func NewService() *Service {
	return NewServiceWithStore(NewMemoryStore())
}

// NewServiceWithStore constructs the workflow promotion service with a
// caller-provided task store.
func NewServiceWithStore(store TaskStore) *Service {
	if store == nil {
		store = NewMemoryStore()
	}

	return &Service{store: store}
}

// Promote creates a new async task record from the current synchronous request.
func (s *Service) Promote(ctx context.Context, req PromoteRequest) (Task, error) {
	now := time.Now().UTC()
	task := Task{
		ID:               fmt.Sprintf("task-%d", now.UnixNano()),
		RequestID:        req.RequestID,
		TenantID:         req.TenantID,
		SessionID:        req.SessionID,
		TaskType:         req.TaskType,
		Status:           StatusQueued,
		Reason:           req.Reason,
		RequiresApproval: req.RequiresApproval,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if req.RequiresApproval {
		task.Status = StatusWaitingApproval
	}

	return s.store.SaveTask(ctx, task)
}

// GetTask returns a promoted task by ID.
func (s *Service) GetTask(ctx context.Context, taskID string) (Task, error) {
	return s.store.GetTask(ctx, taskID)
}
