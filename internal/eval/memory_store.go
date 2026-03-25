package eval

import (
	"context"
	"fmt"
	"sync"
)

type memoryStore struct {
	mu           sync.RWMutex
	byID         map[string]EvalCase
	bySourceCase map[string]string
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		byID:         make(map[string]EvalCase),
		bySourceCase: make(map[string]string),
	}
}

func (s *memoryStore) Save(_ context.Context, item EvalCase) (EvalCase, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if existingID, ok := s.bySourceCase[item.SourceCaseID]; ok && existingID != item.ID {
		return EvalCase{}, fmt.Errorf("%w: %s", ErrEvalCaseExists, item.SourceCaseID)
	}

	s.byID[item.ID] = item
	s.bySourceCase[item.SourceCaseID] = item.ID

	return item, nil
}

func (s *memoryStore) Get(_ context.Context, evalCaseID string) (EvalCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.byID[evalCaseID]
	if !ok {
		return EvalCase{}, fmt.Errorf("%w: %s", ErrEvalCaseNotFound, evalCaseID)
	}

	return item, nil
}

func (s *memoryStore) GetBySourceCase(_ context.Context, sourceCaseID string) (EvalCase, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	evalCaseID, ok := s.bySourceCase[sourceCaseID]
	if !ok {
		return EvalCase{}, fmt.Errorf("%w: %s", ErrEvalCaseNotFound, sourceCaseID)
	}

	return s.byID[evalCaseID], nil
}
