package cases

import (
	"context"
	"fmt"
	"sync"
)

type memoryStore struct {
	mu      sync.RWMutex
	records map[string]Case
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		records: map[string]Case{},
	}
}

func (s *memoryStore) Save(_ context.Context, item Case) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[item.ID] = item
	return item, nil
}

func (s *memoryStore) Get(_ context.Context, caseID string) (Case, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}

	return item, nil
}
