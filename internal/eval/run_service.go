package eval

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var evalRunIDSequence atomic.Uint64

type runStore interface {
	CreateRun(ctx context.Context, item EvalRun) (EvalRun, error)
	GetRun(ctx context.Context, runID string) (EvalRun, error)
	ListRuns(ctx context.Context, filter RunListFilter) (RunListPage, error)
}

type datasetReader interface {
	GetDataset(ctx context.Context, datasetID string) (EvalDataset, error)
}

// RunService manages durable eval-run kickoff records.
type RunService struct {
	store    runStore
	datasets datasetReader
}

// NewRunService constructs the eval-run service with memory-backed defaults.
func NewRunService(datasets datasetReader) *RunService {
	return NewRunServiceWithStore(nil, datasets)
}

// NewRunServiceWithStore constructs the eval-run service with caller-provided storage.
func NewRunServiceWithStore(store runStore, datasets datasetReader) *RunService {
	if store == nil {
		store = newMemoryStore()
	}

	return &RunService{
		store:    store,
		datasets: datasets,
	}
}

// CreateRun creates one durable eval run from a published dataset.
func (s *RunService) CreateRun(ctx context.Context, input CreateRunInput) (EvalRun, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.DatasetID) == "" {
		return EvalRun{}, ErrInvalidEvalDataset
	}

	dataset, err := s.datasets.GetDataset(ctx, strings.TrimSpace(input.DatasetID))
	if err != nil {
		if err == ErrEvalDatasetNotFound {
			return EvalRun{}, ErrEvalDatasetNotFound
		}
		return EvalRun{}, err
	}
	if dataset.TenantID != strings.TrimSpace(input.TenantID) {
		return EvalRun{}, ErrEvalDatasetNotFound
	}
	if dataset.Status != DatasetStatusPublished {
		return EvalRun{}, ErrInvalidEvalDatasetState
	}

	now := time.Now().UTC()
	return s.store.CreateRun(ctx, EvalRun{
		ID:               newEvalRunID(now),
		TenantID:         dataset.TenantID,
		DatasetID:        dataset.ID,
		DatasetName:      dataset.Name,
		DatasetItemCount: len(dataset.Items),
		Status:           RunStatusQueued,
		CreatedBy:        fallbackString(strings.TrimSpace(input.CreatedBy), "operator"),
		CreatedAt:        now,
		UpdatedAt:        now,
	})
}

// GetRun returns one durable eval run by ID.
func (s *RunService) GetRun(ctx context.Context, runID string) (EvalRun, error) {
	return s.store.GetRun(ctx, runID)
}

// ListRuns returns one durable eval-run page.
func (s *RunService) ListRuns(ctx context.Context, filter RunListFilter) (RunListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	return s.store.ListRuns(ctx, filter)
}

func newEvalRunID(now time.Time) string {
	return fmt.Sprintf("eval-run-%d-%d", now.UnixNano(), evalRunIDSequence.Add(1))
}
