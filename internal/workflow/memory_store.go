package workflow

import (
	"context"
	"fmt"
	"sync"
)

// MemoryStore stores workflow task records in memory for tests and offline use.
type MemoryStore struct {
	mu    sync.RWMutex
	tasks map[string]Task
}

// NewMemoryStore constructs an in-memory workflow task store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		tasks: make(map[string]Task),
	}
}

// SaveTask writes the task into the in-memory store.
func (s *MemoryStore) SaveTask(_ context.Context, task Task) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks[task.ID] = task
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
