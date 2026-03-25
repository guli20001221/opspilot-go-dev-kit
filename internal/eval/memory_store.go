package eval

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type memoryStore struct {
	mu           sync.RWMutex
	byID         map[string]EvalCase
	bySourceCase map[string]string
	datasets     map[string]EvalDataset
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		byID:         make(map[string]EvalCase),
		bySourceCase: make(map[string]string),
		datasets:     make(map[string]EvalDataset),
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

func (s *memoryStore) CreateDataset(_ context.Context, item EvalDataset) (EvalDataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.datasets[item.ID] = item

	return item, nil
}

func (s *memoryStore) GetDataset(_ context.Context, datasetID string) (EvalDataset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.datasets[datasetID]
	if !ok {
		return EvalDataset{}, fmt.Errorf("%w: %s", ErrEvalDatasetNotFound, datasetID)
	}

	return item, nil
}

func (s *memoryStore) ListDatasets(_ context.Context, filter DatasetListFilter) (DatasetListPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]EvalDatasetSummary, 0, len(s.datasets))
	for _, item := range s.datasets {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		if filter.CreatedBy != "" && item.CreatedBy != filter.CreatedBy {
			continue
		}
		items = append(items, EvalDatasetSummary{
			ID:        item.ID,
			TenantID:  item.TenantID,
			Name:      item.Name,
			Status:    item.Status,
			CreatedBy: item.CreatedBy,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			ItemCount: len(item.Items),
		})
	}

	sort.Slice(items, func(i int, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		return items[i].ID > items[j].ID
	})

	start := filter.Offset
	if start > len(items) {
		start = len(items)
	}
	end := start + filter.Limit
	page := DatasetListPage{}
	if end < len(items) {
		page.HasMore = true
		page.NextOffset = end
	} else {
		end = len(items)
	}
	page.Datasets = append(page.Datasets, items[start:end]...)

	return page, nil
}

func (s *memoryStore) AddDatasetItem(_ context.Context, datasetID string, item EvalDatasetItem, updatedAt time.Time) (EvalDataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dataset, ok := s.datasets[datasetID]
	if !ok {
		return EvalDataset{}, fmt.Errorf("%w: %s", ErrEvalDatasetNotFound, datasetID)
	}
	for _, existing := range dataset.Items {
		if existing.EvalCaseID == item.EvalCaseID {
			return dataset, nil
		}
	}

	dataset.Items = append(dataset.Items, item)
	dataset.UpdatedAt = updatedAt
	s.datasets[datasetID] = dataset

	return dataset, nil
}
