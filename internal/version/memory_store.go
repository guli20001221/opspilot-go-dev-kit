package version

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type memoryStore struct {
	mu       sync.RWMutex
	versions map[string]Version
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		versions: make(map[string]Version),
	}
}

func (s *memoryStore) Save(_ context.Context, item Version) (Version, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.versions[item.ID] = item
	return item, nil
}

func (s *memoryStore) Get(_ context.Context, versionID string) (Version, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.versions[versionID]
	if !ok {
		return Version{}, fmt.Errorf("%w: %s", ErrVersionNotFound, versionID)
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

	items := make([]Version, 0, len(s.versions))
	for _, item := range s.versions {
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})

	if offset >= len(items) {
		return ListPage{Versions: []Version{}}, nil
	}

	end := offset + limit
	hasMore := end < len(items)
	if end > len(items) {
		end = len(items)
	}

	page := make([]Version, end-offset)
	copy(page, items[offset:end])

	result := ListPage{
		Versions: page,
		HasMore:  hasMore,
	}
	if hasMore {
		result.NextOffset = end
	}

	return result, nil
}
