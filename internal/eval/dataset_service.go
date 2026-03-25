package eval

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

var evalDatasetIDSequence atomic.Uint64

type datasetStore interface {
	CreateDataset(ctx context.Context, item EvalDataset) (EvalDataset, error)
	GetDataset(ctx context.Context, datasetID string) (EvalDataset, error)
	ListDatasets(ctx context.Context, filter DatasetListFilter) (DatasetListPage, error)
	AddDatasetItem(ctx context.Context, datasetID string, item EvalDatasetItem, updatedAt time.Time) (EvalDataset, error)
	PublishDataset(ctx context.Context, datasetID string, publishedBy string, publishedAt time.Time) (EvalDataset, error)
}

type evalCaseReader interface {
	GetEvalCase(ctx context.Context, evalCaseID string) (EvalCase, error)
}

// DatasetService manages durable eval dataset drafts and reads.
type DatasetService struct {
	store     datasetStore
	evalCases evalCaseReader
}

// NewDatasetService constructs the dataset service with in-memory defaults.
func NewDatasetService(evalCases evalCaseReader) *DatasetService {
	return NewDatasetServiceWithStore(nil, evalCases)
}

// NewDatasetServiceWithStore constructs the dataset service with caller-provided storage.
func NewDatasetServiceWithStore(store datasetStore, evalCases evalCaseReader) *DatasetService {
	if store == nil {
		store = newMemoryStore()
	}

	return &DatasetService{
		store:     store,
		evalCases: evalCases,
	}
}

// CreateDataset creates one durable dataset draft from promoted eval cases.
func (s *DatasetService) CreateDataset(ctx context.Context, input CreateDatasetInput) (EvalDataset, error) {
	if strings.TrimSpace(input.TenantID) == "" || len(input.EvalCaseIDs) == 0 {
		return EvalDataset{}, ErrInvalidEvalDataset
	}

	items := make([]EvalDatasetItem, 0, len(input.EvalCaseIDs))
	seenEvalCaseIDs := make(map[string]struct{}, len(input.EvalCaseIDs))
	for _, evalCaseID := range input.EvalCaseIDs {
		trimmedEvalCaseID := strings.TrimSpace(evalCaseID)
		if trimmedEvalCaseID == "" {
			return EvalDataset{}, ErrInvalidEvalDataset
		}
		if _, seen := seenEvalCaseIDs[trimmedEvalCaseID]; seen {
			return EvalDataset{}, ErrInvalidEvalDataset
		}
		seenEvalCaseIDs[trimmedEvalCaseID] = struct{}{}

		item, err := s.evalCases.GetEvalCase(ctx, trimmedEvalCaseID)
		if err != nil {
			return EvalDataset{}, err
		}
		if item.TenantID != input.TenantID {
			return EvalDataset{}, ErrInvalidEvalDataset
		}
		items = append(items, EvalDatasetItem{
			EvalCaseID:     item.ID,
			Title:          item.Title,
			SourceCaseID:   item.SourceCaseID,
			SourceTaskID:   item.SourceTaskID,
			SourceReportID: item.SourceReportID,
			TraceID:        item.TraceID,
			VersionID:      item.VersionID,
		})
	}

	now := time.Now().UTC()
	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = fmt.Sprintf("Dataset seeded from %s", items[0].EvalCaseID)
	}

	return s.store.CreateDataset(ctx, EvalDataset{
		ID:          newEvalDatasetID(now),
		TenantID:    input.TenantID,
		Name:        name,
		Description: strings.TrimSpace(input.Description),
		Status:      DatasetStatusDraft,
		CreatedBy:   fallbackString(strings.TrimSpace(input.CreatedBy), "operator"),
		CreatedAt:   now,
		UpdatedAt:   now,
		Items:       items,
	})
}

// GetDataset returns one durable eval dataset by ID.
func (s *DatasetService) GetDataset(ctx context.Context, datasetID string) (EvalDataset, error) {
	return s.store.GetDataset(ctx, datasetID)
}

// ListDatasets returns one durable eval-dataset page.
func (s *DatasetService) ListDatasets(ctx context.Context, filter DatasetListFilter) (DatasetListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return s.store.ListDatasets(ctx, filter)
}

// AddDatasetItem appends one durable eval case into an existing draft dataset.
func (s *DatasetService) AddDatasetItem(ctx context.Context, datasetID string, input AddDatasetItemInput) (EvalDataset, error) {
	if strings.TrimSpace(input.TenantID) == "" || strings.TrimSpace(input.EvalCaseID) == "" {
		return EvalDataset{}, ErrInvalidEvalDataset
	}

	dataset, err := s.store.GetDataset(ctx, datasetID)
	if err != nil {
		return EvalDataset{}, err
	}
	if dataset.TenantID != strings.TrimSpace(input.TenantID) {
		return EvalDataset{}, ErrEvalDatasetNotFound
	}
	if dataset.Status != DatasetStatusDraft {
		return EvalDataset{}, ErrInvalidEvalDatasetState
	}

	evalCase, err := s.evalCases.GetEvalCase(ctx, strings.TrimSpace(input.EvalCaseID))
	if err != nil {
		return EvalDataset{}, err
	}
	if evalCase.TenantID != dataset.TenantID {
		return EvalDataset{}, ErrEvalCaseNotFound
	}
	for _, member := range dataset.Items {
		if member.EvalCaseID == evalCase.ID {
			return dataset, nil
		}
	}

	return s.store.AddDatasetItem(ctx, dataset.ID, EvalDatasetItem{
		EvalCaseID:     evalCase.ID,
		Title:          evalCase.Title,
		SourceCaseID:   evalCase.SourceCaseID,
		SourceTaskID:   evalCase.SourceTaskID,
		SourceReportID: evalCase.SourceReportID,
		TraceID:        evalCase.TraceID,
		VersionID:      evalCase.VersionID,
	}, time.Now().UTC())
}

// PublishDataset freezes a durable dataset draft into an immutable published baseline.
func (s *DatasetService) PublishDataset(ctx context.Context, datasetID string, input PublishDatasetInput) (EvalDataset, error) {
	if strings.TrimSpace(input.TenantID) == "" {
		return EvalDataset{}, ErrInvalidEvalDataset
	}

	dataset, err := s.store.GetDataset(ctx, datasetID)
	if err != nil {
		return EvalDataset{}, err
	}
	if dataset.TenantID != strings.TrimSpace(input.TenantID) {
		return EvalDataset{}, ErrEvalDatasetNotFound
	}
	if dataset.Status != DatasetStatusDraft {
		return EvalDataset{}, ErrInvalidEvalDatasetState
	}

	return s.store.PublishDataset(
		ctx,
		dataset.ID,
		fallbackString(strings.TrimSpace(input.PublishedBy), "operator"),
		time.Now().UTC(),
	)
}

func newEvalDatasetID(now time.Time) string {
	return fmt.Sprintf("eval-dataset-%d-%d", now.UnixNano(), evalDatasetIDSequence.Add(1))
}
