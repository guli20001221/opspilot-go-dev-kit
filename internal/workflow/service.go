package workflow

import (
	"context"
	"fmt"
	"sync"
	"time"
)

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
