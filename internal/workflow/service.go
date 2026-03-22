package workflow

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrTaskNotFound identifies missing workflow task records.
var ErrTaskNotFound = errors.New("workflow task not found")

// ErrInvalidTaskTransition identifies unsupported task state changes.
var ErrInvalidTaskTransition = errors.New("invalid workflow task transition")

// TaskStore persists workflow task records.
type TaskStore interface {
	SaveTask(ctx context.Context, task Task) (Task, error)
	GetTask(ctx context.Context, taskID string) (Task, error)
	ClaimQueuedTasks(ctx context.Context, limit int) ([]Task, error)
	UpdateTask(ctx context.Context, task Task) (Task, error)
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

// ClaimQueuedTasks marks queued tasks as running and returns them to a worker.
func (s *Service) ClaimQueuedTasks(ctx context.Context, limit int) ([]Task, error) {
	return s.store.ClaimQueuedTasks(ctx, limit)
}

// UpdateTask persists task state after worker processing.
func (s *Service) UpdateTask(ctx context.Context, task Task) (Task, error) {
	task.UpdatedAt = time.Now().UTC()
	return s.store.UpdateTask(ctx, task)
}

// ApproveTask resumes a waiting-approval task into the queued state.
func (s *Service) ApproveTask(ctx context.Context, taskID string, approvedBy string) (Task, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if task.Status != StatusWaitingApproval {
		return Task{}, fmt.Errorf("%w: approve from %s", ErrInvalidTaskTransition, task.Status)
	}

	task.Status = StatusQueued
	task.ErrorReason = ""
	task.AuditRef = fmt.Sprintf("approval:%s", approvedBy)

	return s.UpdateTask(ctx, task)
}

// RetryTask re-queues a failed task for another worker attempt.
func (s *Service) RetryTask(ctx context.Context, taskID string, retriedBy string) (Task, error) {
	task, err := s.store.GetTask(ctx, taskID)
	if err != nil {
		return Task{}, err
	}
	if task.Status != StatusFailed {
		return Task{}, fmt.Errorf("%w: retry from %s", ErrInvalidTaskTransition, task.Status)
	}

	task.Status = StatusQueued
	task.ErrorReason = ""
	task.AuditRef = fmt.Sprintf("retry:%s", retriedBy)

	return s.UpdateTask(ctx, task)
}
