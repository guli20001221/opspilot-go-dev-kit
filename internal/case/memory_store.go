package cases

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
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

func (s *memoryStore) List(_ context.Context, filter ListFilter) (ListPage, error) {
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

	items := make([]Case, 0, len(s.records))
	for _, item := range s.records {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.SourceTaskID != "" && item.SourceTaskID != filter.SourceTaskID {
			continue
		}
		if filter.SourceReportID != "" && item.SourceReportID != filter.SourceReportID {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.After(items[j].CreatedAt)
		}
		return items[i].ID > items[j].ID
	})

	if offset >= len(items) {
		return ListPage{Cases: []Case{}}, nil
	}

	end := offset + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}

	page := ListPage{
		Cases:   append([]Case(nil), items[offset:end]...),
		HasMore: hasMore,
	}
	if hasMore {
		page.NextOffset = end
	}

	return page, nil
}

func (s *memoryStore) Close(_ context.Context, caseID string, closedBy string, closedAt time.Time) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if item.Status == StatusClosed {
		return Case{}, ErrInvalidCaseState
	}

	item.Status = StatusClosed
	item.ClosedBy = closedBy
	item.UpdatedAt = closedAt
	s.records[caseID] = item

	return item, nil
}

func (s *memoryStore) Assign(_ context.Context, caseID string, assignedTo string, assignedAt time.Time, expectedUpdatedAt time.Time) (Case, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.records[caseID]
	if !ok {
		return Case{}, fmt.Errorf("%w: %s", ErrCaseNotFound, caseID)
	}
	if item.Status == StatusClosed {
		return Case{}, ErrInvalidCaseState
	}
	if !item.UpdatedAt.Equal(expectedUpdatedAt) {
		return Case{}, ErrCaseConflict
	}

	item.AssignedTo = assignedTo
	item.AssignedAt = assignedAt
	item.UpdatedAt = assignedAt
	s.records[caseID] = item

	return item, nil
}
