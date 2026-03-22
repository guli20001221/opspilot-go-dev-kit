package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"
)

// ErrTaskNotFound identifies missing workflow task records.
var ErrTaskNotFound = errors.New("workflow task not found")

// ErrInvalidTaskTransition identifies unsupported task state changes.
var ErrInvalidTaskTransition = errors.New("invalid workflow task transition")

var taskIDSequence atomic.Uint64

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
	ListTasks(ctx context.Context, filter TaskListFilter) (TaskListPage, error)
}

// TaskStarter starts external workflow execution for tasks that must be
// initialized before worker-side processing.
type TaskStarter interface {
	StartTask(ctx context.Context, task Task) error
}

// Service persists promoted tasks through a caller-provided store.
type Service struct {
	store   TaskStore
	starter TaskStarter
}

// NewService constructs the workflow promotion service.
func NewService() *Service {
	return NewServiceWithStore(nil)
}

// NewServiceWithStore constructs the workflow promotion service with a
// caller-provided task store.
func NewServiceWithStore(store TaskStore) *Service {
	return NewServiceWithHooks(store, nil)
}

// NewServiceWithHooks constructs the workflow promotion service with optional
// runtime hooks.
func NewServiceWithHooks(store TaskStore, starter TaskStarter) *Service {
	if store == nil {
		store = NewMemoryStore()
	}

	return &Service{
		store:   store,
		starter: starter,
	}
}

// Promote creates a new async task record from the current synchronous request.
func (s *Service) Promote(ctx context.Context, req PromoteRequest) (Task, error) {
	now := time.Now().UTC()
	task := Task{
		ID:               newTaskID(now),
		RequestID:        req.RequestID,
		TenantID:         req.TenantID,
		SessionID:        req.SessionID,
		TaskType:         req.TaskType,
		ToolName:         req.ToolName,
		ToolArguments:    req.ToolArguments,
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
	if shouldStartTaskOnPromote(created) && s.starter != nil {
		if err := s.starter.StartTask(ctx, created); err != nil {
			created.Status = StatusFailed
			created.ErrorReason = err.Error()
			created.AuditRef = "temporal:start_failed"
			created.UpdatedAt = time.Now().UTC()

			failed, updateErr := s.store.UpdateTaskWithEvent(ctx, created, AuditEvent{
				TaskID:    created.ID,
				Action:    AuditActionFailed,
				Actor:     "api",
				Detail:    created.ErrorReason,
				CreatedAt: created.UpdatedAt,
			})
			if updateErr != nil {
				return Task{}, updateErr
			}

			return failed, nil
		}
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

// ListTasks returns operator-facing task rows for the provided filter.
func (s *Service) ListTasks(ctx context.Context, filter TaskListFilter) (TaskListPage, error) {
	return s.store.ListTasks(ctx, filter)
}

func shouldStartTaskOnPromote(task Task) bool {
	return task.TaskType == TaskTypeApprovedToolExecution && task.RequiresApproval
}

func newTaskID(now time.Time) string {
	return fmt.Sprintf("task-%d-%d", now.UnixNano(), taskIDSequence.Add(1))
}
