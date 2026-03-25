package eval

import (
	"context"
	"fmt"
	"sort"
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

func (s *memoryStore) List(_ context.Context, filter ListFilter) (ListPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]EvalCase, 0, len(s.byID))
	for _, item := range s.byID {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.SourceCaseID != "" && item.SourceCaseID != filter.SourceCaseID {
			continue
		}
		if filter.SourceTaskID != "" && item.SourceTaskID != filter.SourceTaskID {
			continue
		}
		if filter.SourceReportID != "" && item.SourceReportID != filter.SourceReportID {
			continue
		}
		if filter.VersionID != "" && item.VersionID != filter.VersionID {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i int, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID > items[j].ID
	})

	start := filter.Offset
	if start > len(items) {
		start = len(items)
	}
	end := start + filter.Limit
	page := ListPage{}
	if end < len(items) {
		page.HasMore = true
		page.NextOffset = end
	} else {
		end = len(items)
	}
	page.EvalCases = append(page.EvalCases, items[start:end]...)

	return page, nil
}
