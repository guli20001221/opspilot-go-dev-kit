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
	CreateTaskWithEvent(ctx context.Context, task Task, event AuditEvent) (Task, error)
	GetTask(ctx context.Context, taskID string) (Task, error)
	ClaimQueuedTasks(ctx context.Context, limit int) ([]Task, error)
	UpdateTask(ctx context.Context, task Task) (Task, error)
	UpdateTaskWithEvent(ctx context.Context, task Task, event AuditEvent) (Task, error)
	AppendTaskEvent(ctx context.Context, event AuditEvent) (AuditEvent, error)
	ListTaskEvents(ctx context.Context, taskID string) ([]AuditEvent, error)
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

	created, err := s.store.CreateTaskWithEvent(ctx, task, AuditEvent{
		TaskID:    task.ID,
		Action:    AuditActionCreated,
		Actor:     "api",
		Detail:    task.Status,
		CreatedAt: now,
	})
	if err != nil {
		return Task{}, err
	}

	return created, nil
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

	task.UpdatedAt = time.Now().UTC()
	updated, err := s.store.UpdateTaskWithEvent(ctx, task, AuditEvent{
		TaskID:    task.ID,
		Action:    AuditActionApproved,
		Actor:     approvedBy,
		Detail:    task.Status,
		CreatedAt: task.UpdatedAt,
	})
	if err != nil {
		return Task{}, err
	}

	return updated, nil
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

	task.UpdatedAt = time.Now().UTC()
	updated, err := s.store.UpdateTaskWithEvent(ctx, task, AuditEvent{
		TaskID:    task.ID,
		Action:    AuditActionRetried,
		Actor:     retriedBy,
		Detail:    task.Status,
		CreatedAt: task.UpdatedAt,
	})
	if err != nil {
		return Task{}, err
	}

	return updated, nil
}

// ListTaskEvents returns the structured audit history for a task.
func (s *Service) ListTaskEvents(ctx context.Context, taskID string) ([]AuditEvent, error) {
	return s.store.ListTaskEvents(ctx, taskID)
}
