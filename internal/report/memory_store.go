package report

import (
	"context"
	"fmt"
	"slices"
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

func (s *memoryStore) List(_ context.Context, filter ListFilter) (ListPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	items := make([]Report, 0, len(s.records))
	for _, item := range s.records {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.ReportType != "" && item.ReportType != filter.ReportType {
			continue
		}
		if filter.SourceTaskID != "" && item.SourceTaskID != filter.SourceTaskID {
			continue
		}
		items = append(items, item)
	}

	slices.SortFunc(items, func(left Report, right Report) int {
		leftReady := left.CreatedAt
		if left.ReadyAt != nil {
			leftReady = *left.ReadyAt
		}
		rightReady := right.CreatedAt
		if right.ReadyAt != nil {
			rightReady = *right.ReadyAt
		}
		if cmp := rightReady.Compare(leftReady); cmp != 0 {
			return cmp
		}
		if cmp := right.CreatedAt.Compare(left.CreatedAt); cmp != 0 {
			return cmp
		}
		switch {
		case left.ID < right.ID:
			return 1
		case left.ID > right.ID:
			return -1
		default:
			return 0
		}
	})

	if filter.Offset >= len(items) {
		return ListPage{
			Reports: make([]Report, 0),
		}, nil
	}

	end := filter.Offset + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}

	page := ListPage{
		Reports: slices.Clone(items[filter.Offset:end]),
		HasMore: hasMore,
	}
	if hasMore {
		page.NextOffset = filter.Offset + len(page.Reports)
	}

	return page, nil
}
