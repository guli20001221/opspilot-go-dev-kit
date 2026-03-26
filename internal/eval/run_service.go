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
	CreateRun(ctx context.Context, item EvalRun, items ...EvalRunItem) (EvalRun, error)
	GetRun(ctx context.Context, runID string) (EvalRun, error)
	GetRunDetail(ctx context.Context, runID string) (EvalRunDetail, error)
	ListRuns(ctx context.Context, filter RunListFilter) (RunListPage, error)
	ListRunEvents(ctx context.Context, runID string) ([]EvalRunEvent, error)
	ClaimQueuedRuns(ctx context.Context, limit int, startedAt time.Time) ([]EvalRun, error)
	MarkRunSucceeded(ctx context.Context, runID string, finishedAt time.Time) (EvalRun, error)
	MarkRunFailed(ctx context.Context, runID string, reason string, finishedAt time.Time) (EvalRun, error)
	RetryRun(ctx context.Context, runID string, updatedAt time.Time) (EvalRun, error)
	UpdateRun(ctx context.Context, item EvalRun) (EvalRun, error)
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
	items := make([]EvalRunItem, 0, len(dataset.Items))
	for _, item := range dataset.Items {
		items = append(items, EvalRunItem{
			EvalCaseID:     item.EvalCaseID,
			Title:          item.Title,
			SourceCaseID:   item.SourceCaseID,
			SourceTaskID:   item.SourceTaskID,
			SourceReportID: item.SourceReportID,
			TraceID:        item.TraceID,
			VersionID:      item.VersionID,
		})
	}

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
	}, items...)
}

// GetRun returns one durable eval run by ID.
func (s *RunService) GetRun(ctx context.Context, runID string) (EvalRun, error) {
	return s.store.GetRun(ctx, runID)
}

// GetRunDetail returns one durable eval run with a consistent snapshot of its timeline and membership.
func (s *RunService) GetRunDetail(ctx context.Context, runID string) (EvalRunDetail, error) {
	return s.store.GetRunDetail(ctx, runID)
}

// ListRuns returns one durable eval-run page.
func (s *RunService) ListRuns(ctx context.Context, filter RunListFilter) (RunListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	return s.store.ListRuns(ctx, filter)
}

// ListRunEvents returns the append-only lifecycle history for one eval run.
func (s *RunService) ListRunEvents(ctx context.Context, runID string) ([]EvalRunEvent, error) {
	return s.store.ListRunEvents(ctx, runID)
}

// ClaimQueuedRuns marks queued eval runs as running and returns them for worker execution.
func (s *RunService) ClaimQueuedRuns(ctx context.Context, limit int) ([]EvalRun, error) {
	if limit <= 0 {
		limit = 10
	}
	return s.store.ClaimQueuedRuns(ctx, limit, time.Now().UTC())
}

// MarkRunSucceeded finalizes a running eval run as succeeded.
func (s *RunService) MarkRunSucceeded(ctx context.Context, runID string) (EvalRun, error) {
	return s.store.MarkRunSucceeded(ctx, runID, time.Now().UTC())
}

// MarkRunFailed finalizes a running eval run as failed with a summarized error.
func (s *RunService) MarkRunFailed(ctx context.Context, runID string, reason string) (EvalRun, error) {
	return s.store.MarkRunFailed(ctx, runID, strings.TrimSpace(reason), time.Now().UTC())
}

// RetryRun re-queues a failed eval run for another worker attempt.
func (s *RunService) RetryRun(ctx context.Context, runID string) (EvalRun, error) {
	return s.store.RetryRun(ctx, runID, time.Now().UTC())
}

func newEvalRunID(now time.Time) string {
	return fmt.Sprintf("eval-run-%d-%d", now.UnixNano(), evalRunIDSequence.Add(1))
}
