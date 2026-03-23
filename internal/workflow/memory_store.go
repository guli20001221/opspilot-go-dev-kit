package workflow

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MemoryStore stores workflow task records in memory for tests and offline use.
type MemoryStore struct {
	mu          sync.RWMutex
	tasks       map[string]Task
	events      map[string][]AuditEvent
	nextEventID int64
}

// NewMemoryStore constructs an in-memory workflow task store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks:  make(map[string]Task),
		events: make(map[string][]AuditEvent),
	}
}

// SaveTask writes the task into the in-memory store.
func (s *MemoryStore) SaveTask(_ context.Context, task Task) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	return task, nil
}

// CreateTaskWithEvent writes a task and its initial audit event atomically.
func (s *MemoryStore) CreateTaskWithEvent(_ context.Context, task Task, event AuditEvent) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
	s.nextEventID++
	event.ID = s.nextEventID
	s.events[event.TaskID] = append(s.events[event.TaskID], event)
	return task, nil
}

// GetTask loads a task from the in-memory store.
func (s *MemoryStore) GetTask(_ context.Context, taskID string) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return Task{}, fmt.Errorf("%w: %s", ErrTaskNotFound, taskID)
	}

	return task, nil
}

// ClaimQueuedTasks marks queued tasks as running and returns them in creation order.
func (s *MemoryStore) ClaimQueuedTasks(_ context.Context, limit int) ([]Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		return nil, nil
	}

	queued := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		if task.Status == StatusQueued {
			queued = append(queued, task)
		}
	}

	sort.Slice(queued, func(i, j int) bool {
		return queued[i].CreatedAt.Before(queued[j].CreatedAt)
	})

	if len(queued) > limit {
		queued = queued[:limit]
	}

	now := time.Now().UTC()
	for i := range queued {
		queued[i].Status = StatusRunning
		queued[i].UpdatedAt = now
		s.tasks[queued[i].ID] = queued[i]
		s.nextEventID++
		s.events[queued[i].ID] = append(s.events[queued[i].ID], AuditEvent{
			ID:        s.nextEventID,
			TaskID:    queued[i].ID,
			Action:    AuditActionClaimed,
			Actor:     "worker",
			Detail:    queued[i].Status,
			CreatedAt: queued[i].UpdatedAt,
		})
	}

	return queued, nil
}

// UpdateTask overwrites an existing task in the in-memory store.
func (s *MemoryStore) UpdateTask(_ context.Context, task Task) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[task.ID]; !ok {
		return Task{}, fmt.Errorf("%w: %s", ErrTaskNotFound, task.ID)
	}

	s.tasks[task.ID] = task
	return task, nil
}

// UpdateTaskWithEvent overwrites a task and appends an audit event atomically.
func (s *MemoryStore) UpdateTaskWithEvent(_ context.Context, task Task, event AuditEvent) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[task.ID]; !ok {
		return Task{}, fmt.Errorf("%w: %s", ErrTaskNotFound, task.ID)
	}

	s.tasks[task.ID] = task
	s.nextEventID++
	event.ID = s.nextEventID
	s.events[event.TaskID] = append(s.events[event.TaskID], event)
	return task, nil
}

// AppendTaskEvent appends a structured audit event to the in-memory store.
func (s *MemoryStore) AppendTaskEvent(_ context.Context, event AuditEvent) (AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.nextEventID++
	event.ID = s.nextEventID
	s.events[event.TaskID] = append(s.events[event.TaskID], event)
	return event, nil
}

// ListTaskEvents returns the in-memory audit history for a task.
func (s *MemoryStore) ListTaskEvents(_ context.Context, taskID string) ([]AuditEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := s.events[taskID]
	out := make([]AuditEvent, len(events))
	copy(out, events)
	return out, nil
}

// ListTasks returns filtered in-memory tasks ordered by newest update first.
func (s *MemoryStore) ListTasks(_ context.Context, filter TaskListFilter) (TaskListPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	out := make([]Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		if filter.TenantID != "" && task.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && task.Status != filter.Status {
			continue
		}
		if filter.TaskType != "" && task.TaskType != filter.TaskType {
			continue
		}
		if filter.Reason != "" && task.Reason != filter.Reason {
			continue
		}
		if filter.RequiresApproval != nil && task.RequiresApproval != *filter.RequiresApproval {
			continue
		}
		if filter.CreatedAfter != nil && !task.CreatedAt.After(*filter.CreatedAfter) {
			continue
		}
		if filter.CreatedBefore != nil && !task.CreatedAt.Before(*filter.CreatedBefore) {
			continue
		}
		if filter.UpdatedAfter != nil && !task.UpdatedAt.After(*filter.UpdatedAfter) {
			continue
		}
		if filter.UpdatedBefore != nil && !task.UpdatedAt.Before(*filter.UpdatedBefore) {
			continue
		}
		out = append(out, task)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			if out[i].CreatedAt.Equal(out[j].CreatedAt) {
				return out[i].ID > out[j].ID
			}
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	if offset >= len(out) {
		return TaskListPage{Tasks: []Task{}}, nil
	}

	end := offset + limit
	hasMore := end < len(out)
	if end > len(out) {
		end = len(out)
	}

	page := make([]Task, end-offset)
	copy(page, out[offset:end])

	result := TaskListPage{
		Tasks:   page,
		HasMore: hasMore,
	}
	if hasMore {
		result.NextOffset = end
	}

	return result, nil
}
