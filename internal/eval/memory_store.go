package eval

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type memoryStore struct {
	mu             sync.RWMutex
	byID           map[string]EvalCase
	bySourceCase   map[string]string
	datasets       map[string]EvalDataset
	runs           map[string]EvalRun
	runEvents      map[string][]EvalRunEvent
	runItems       map[string][]EvalRunItem
	runItemResults map[string][]EvalRunItemResult
	nextRunEvent   int64
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		byID:           make(map[string]EvalCase),
		bySourceCase:   make(map[string]string),
		datasets:       make(map[string]EvalDataset),
		runs:           make(map[string]EvalRun),
		runEvents:      make(map[string][]EvalRunEvent),
		runItems:       make(map[string][]EvalRunItem),
		runItemResults: make(map[string][]EvalRunItemResult),
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

func (s *memoryStore) PublishDataset(_ context.Context, datasetID string, publishedBy string, publishedAt time.Time) (EvalDataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dataset, ok := s.datasets[datasetID]
	if !ok {
		return EvalDataset{}, fmt.Errorf("%w: %s", ErrEvalDatasetNotFound, datasetID)
	}
	if dataset.Status != DatasetStatusDraft {
		return EvalDataset{}, ErrInvalidEvalDatasetState
	}

	dataset.Status = DatasetStatusPublished
	dataset.PublishedBy = publishedBy
	dataset.PublishedAt = publishedAt
	dataset.UpdatedAt = publishedAt
	s.datasets[datasetID] = dataset

	return dataset, nil
}

func (s *memoryStore) CreateRun(_ context.Context, item EvalRun, items ...EvalRunItem) (EvalRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.runs[item.ID] = item
	s.runItems[item.ID] = append([]EvalRunItem(nil), items...)
	s.appendRunEventLocked(EvalRunEvent{
		RunID:     item.ID,
		Action:    RunEventCreated,
		Actor:     item.CreatedBy,
		Detail:    item.Status,
		CreatedAt: item.CreatedAt,
	})
	return item, nil
}

func (s *memoryStore) GetRun(_ context.Context, runID string) (EvalRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.runs[runID]
	if !ok {
		return EvalRun{}, fmt.Errorf("%w: %s", ErrEvalRunNotFound, runID)
	}
	return item, nil
}

func (s *memoryStore) GetRunDetail(_ context.Context, runID string) (EvalRunDetail, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.runs[runID]
	if !ok {
		return EvalRunDetail{}, fmt.Errorf("%w: %s", ErrEvalRunNotFound, runID)
	}

	events := append([]EvalRunEvent(nil), s.runEvents[runID]...)
	items := append([]EvalRunItem(nil), s.runItems[runID]...)
	results := append([]EvalRunItemResult(nil), s.runItemResults[runID]...)
	return EvalRunDetail{
		Run:         item,
		Events:      events,
		Items:       items,
		ItemResults: results,
	}, nil
}

func (s *memoryStore) ListRuns(_ context.Context, filter RunListFilter) (RunListPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]EvalRun, 0, len(s.runs))
	for _, item := range s.runs {
		if filter.TenantID != "" && item.TenantID != filter.TenantID {
			continue
		}
		if filter.DatasetID != "" && item.DatasetID != filter.DatasetID {
			continue
		}
		if filter.Status != "" && item.Status != filter.Status {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
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
	page := RunListPage{}
	if end < len(items) {
		page.HasMore = true
		page.NextOffset = end
	} else {
		end = len(items)
	}
	page.Runs = append(page.Runs, items[start:end]...)

	return page, nil
}

func (s *memoryStore) ListRunEvents(_ context.Context, runID string) ([]EvalRunEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.runs[runID]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrEvalRunNotFound, runID)
	}
	events := append([]EvalRunEvent(nil), s.runEvents[runID]...)
	return events, nil
}

func (s *memoryStore) ClaimQueuedRuns(_ context.Context, limit int, startedAt time.Time) ([]EvalRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]EvalRun, 0, len(s.runs))
	for _, item := range s.runs {
		if item.Status != RunStatusQueued {
			continue
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		if !items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].CreatedAt.Before(items[j].CreatedAt)
		}
		return items[i].ID < items[j].ID
	})

	if limit > len(items) {
		limit = len(items)
	}
	claimed := make([]EvalRun, 0, limit)
	for _, item := range items[:limit] {
		item.Status = RunStatusRunning
		item.ErrorReason = ""
		item.UpdatedAt = startedAt
		if item.StartedAt.IsZero() {
			item.StartedAt = startedAt
		}
		s.runs[item.ID] = item
		s.appendRunEventLocked(EvalRunEvent{
			RunID:     item.ID,
			Action:    RunEventClaimed,
			Actor:     "worker",
			Detail:    item.Status,
			CreatedAt: startedAt,
		})
		claimed = append(claimed, item)
	}

	return claimed, nil
}

func (s *memoryStore) UpdateRun(_ context.Context, item EvalRun) (EvalRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.runs[item.ID]; !ok {
		return EvalRun{}, fmt.Errorf("%w: %s", ErrEvalRunNotFound, item.ID)
	}
	s.runs[item.ID] = item
	return item, nil
}

func (s *memoryStore) RetryRun(_ context.Context, runID string, updatedAt time.Time) (EvalRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.runs[runID]
	if !ok {
		return EvalRun{}, fmt.Errorf("%w: %s", ErrEvalRunNotFound, runID)
	}
	if item.Status != RunStatusFailed {
		return EvalRun{}, ErrInvalidEvalRunState
	}

	item.Status = RunStatusQueued
	item.ErrorReason = ""
	item.UpdatedAt = updatedAt
	item.StartedAt = time.Time{}
	item.FinishedAt = time.Time{}
	s.runs[runID] = item
	delete(s.runItemResults, runID)
	s.appendRunEventLocked(EvalRunEvent{
		RunID:     item.ID,
		Action:    RunEventRetried,
		Actor:     "operator",
		Detail:    item.Status,
		CreatedAt: updatedAt,
	})

	return item, nil
}

func (s *memoryStore) MarkRunSucceeded(_ context.Context, runID string, finishedAt time.Time, results []EvalRunItemResult) (EvalRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.runs[runID]
	if !ok {
		return EvalRun{}, fmt.Errorf("%w: %s", ErrEvalRunNotFound, runID)
	}
	if item.Status != RunStatusRunning {
		return EvalRun{}, ErrInvalidEvalRunState
	}

	item.Status = RunStatusSucceeded
	item.ErrorReason = ""
	item.UpdatedAt = finishedAt
	item.FinishedAt = finishedAt
	s.runs[runID] = item
	s.runItemResults[runID] = append([]EvalRunItemResult(nil), results...)
	s.appendRunEventLocked(EvalRunEvent{
		RunID:     item.ID,
		Action:    RunEventSucceeded,
		Actor:     "worker",
		Detail:    item.Status,
		CreatedAt: finishedAt,
	})

	return item, nil
}

func (s *memoryStore) MarkRunFailed(_ context.Context, runID string, reason string, finishedAt time.Time, results []EvalRunItemResult) (EvalRun, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, ok := s.runs[runID]
	if !ok {
		return EvalRun{}, fmt.Errorf("%w: %s", ErrEvalRunNotFound, runID)
	}
	if item.Status != RunStatusRunning {
		return EvalRun{}, ErrInvalidEvalRunState
	}

	item.Status = RunStatusFailed
	item.ErrorReason = reason
	item.UpdatedAt = finishedAt
	item.FinishedAt = finishedAt
	s.runs[runID] = item
	s.runItemResults[runID] = append([]EvalRunItemResult(nil), results...)
	s.appendRunEventLocked(EvalRunEvent{
		RunID:     item.ID,
		Action:    RunEventFailed,
		Actor:     "worker",
		Detail:    reason,
		CreatedAt: finishedAt,
	})

	return item, nil
}

func (s *memoryStore) appendRunEventLocked(event EvalRunEvent) EvalRunEvent {
	s.nextRunEvent++
	event.ID = s.nextRunEvent
	s.runEvents[event.RunID] = append(s.runEvents[event.RunID], event)
	return event
}
