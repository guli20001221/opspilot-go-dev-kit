package report

import (
	"context"
	"fmt"
	"sync"
)

type memoryStore struct {
	mu      sync.RWMutex
	records map[string]Report
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		records: map[string]Report{},
	}
}

func (s *memoryStore) Save(_ context.Context, item Report) (Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records[item.ID] = item
	return item, nil
}

func (s *memoryStore) Get(_ context.Context, reportID string) (Report, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.records[reportID]
	if !ok {
		return Report{}, fmt.Errorf("%w: %s", ErrReportNotFound, reportID)
	}

	return item, nil
}
