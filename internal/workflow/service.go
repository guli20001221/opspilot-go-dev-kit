package workflow

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrTaskNotFound identifies missing workflow task records.
var ErrTaskNotFound = errors.New("workflow task not found")

// Service stores promoted tasks in memory for the current skeleton.
type Service struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

// NewService constructs the workflow promotion service.
func NewService() *Service {
	return &Service{
		tasks: make(map[string]Task),
	}
}

// Promote creates a new async task record from the current synchronous request.
func (s *Service) Promote(_ context.Context, req PromoteRequest) (Task, error) {
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
	}
	if req.RequiresApproval {
		task.Status = StatusWaitingApproval
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task

	return task, nil
}

// GetTask returns a promoted task by ID.
func (s *Service) GetTask(_ context.Context, taskID string) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return Task{}, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	return task, nil
}
