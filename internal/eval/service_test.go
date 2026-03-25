package eval

import (
	"context"
	"errors"
	"testing"
	"time"

	casesvc "opspilot-go/internal/case"
	"opspilot-go/internal/observability/tracedetail"
)

func TestServicePromoteCaseBuildsLineageFromCaseAndTrace(t *testing.T) {
	sourceCase := casesvc.Case{
		ID:             "case-1",
		TenantID:       "tenant-1",
		Title:          "Investigate workflow failure",
		Summary:        "Failure promoted for eval coverage.",
		SourceTaskID:   "task-1",
		SourceReportID: "report-1",
	}
	service := NewServiceWithStore(nil,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			if caseID != sourceCase.ID {
				return casesvc.Case{}, casesvc.ErrCaseNotFound
			}
			return sourceCase, nil
		}),
		traceLookupFunc(func(_ context.Context, input tracedetail.LookupInput) (tracedetail.Result, error) {
			if input.CaseID != sourceCase.ID {
				return tracedetail.Result{}, tracedetail.ErrInvalidLookup
			}
			return tracedetail.Result{
				Lineage: tracedetail.Lineage{
					TaskID:   sourceCase.SourceTaskID,
					ReportID: sourceCase.SourceReportID,
				},
				TraceID:   "trace-1",
				VersionID: "version-1",
			}, nil
		}),
	)

	got, created, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
		OperatorNote: "promote this failure into regression coverage",
		CreatedBy:    "operator-1",
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	if !created {
		t.Fatal("created = false, want true")
	}
	if got.SourceTaskID != sourceCase.SourceTaskID {
		t.Fatalf("SourceTaskID = %q, want %q", got.SourceTaskID, sourceCase.SourceTaskID)
	}
	if got.SourceReportID != sourceCase.SourceReportID {
		t.Fatalf("SourceReportID = %q, want %q", got.SourceReportID, sourceCase.SourceReportID)
	}
	if got.TraceID != "trace-1" {
		t.Fatalf("TraceID = %q, want %q", got.TraceID, "trace-1")
	}
	if got.VersionID != "version-1" {
		t.Fatalf("VersionID = %q, want %q", got.VersionID, "version-1")
	}
}

func TestServicePromoteCaseIsIdempotentBySourceCase(t *testing.T) {
	sourceCase := casesvc.Case{
		ID:       "case-2",
		TenantID: "tenant-1",
		Title:    "Case already promoted",
	}
	service := NewServiceWithStore(nil,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			return sourceCase, nil
		}),
		traceLookupFunc(func(_ context.Context, input tracedetail.LookupInput) (tracedetail.Result, error) {
			return tracedetail.Result{}, nil
		}),
	)

	first, created, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}
	if !created {
		t.Fatal("created(first) = false, want true")
	}

	second, created, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}
	if created {
		t.Fatal("created(second) = true, want false")
	}
	if second.ID != first.ID {
		t.Fatalf("second.ID = %q, want %q", second.ID, first.ID)
	}
}

func TestServicePromoteCaseRejectsTenantMismatch(t *testing.T) {
	service := NewServiceWithStore(nil,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			return casesvc.Case{
				ID:       caseID,
				TenantID: "tenant-a",
				Title:    "Cross tenant",
			}, nil
		}),
		nil,
	)

	_, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-b",
		SourceCaseID: "case-cross-tenant",
	})
	if !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidSource)
	}
}

func TestServicePromoteCaseRejectsCrossTenantExistingRecord(t *testing.T) {
	sourceCase := casesvc.Case{
		ID:       "case-cross-tenant-existing",
		TenantID: "tenant-a",
		Title:    "Cross tenant existing eval",
	}
	store := newMemoryStore()
	if _, err := store.Save(context.Background(), EvalCase{
		ID:           "eval-existing",
		TenantID:     "tenant-a",
		SourceCaseID: sourceCase.ID,
		Title:        sourceCase.Title,
		Summary:      "existing",
		CreatedBy:    "operator-a",
		CreatedAt:    time.Now().UTC(),
	}); err != nil {
		t.Fatalf("store.Save() error = %v", err)
	}
	service := NewServiceWithStore(store,
		caseReaderFunc(func(_ context.Context, caseID string) (casesvc.Case, error) {
			return sourceCase, nil
		}),
		nil,
	)

	_, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-b",
		SourceCaseID: sourceCase.ID,
	})
	if !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidSource)
	}
}

func TestMemoryStoreListSupportsFiltersAndPagination(t *testing.T) {
	store := newMemoryStore()
	fixtures := []EvalCase{
		{
			ID:             "eval-1",
			TenantID:       "tenant-a",
			SourceCaseID:   "case-1",
			SourceTaskID:   "task-1",
			SourceReportID: "report-1",
			VersionID:      "version-a",
			Title:          "Eval 1",
			Summary:        "First",
			CreatedBy:      "operator-1",
			CreatedAt:      time.Unix(1700000001, 0).UTC(),
		},
		{
			ID:             "eval-2",
			TenantID:       "tenant-a",
			SourceCaseID:   "case-2",
			SourceTaskID:   "task-2",
			SourceReportID: "report-2",
			VersionID:      "version-b",
			Title:          "Eval 2",
			Summary:        "Second",
			CreatedBy:      "operator-2",
			CreatedAt:      time.Unix(1700000002, 0).UTC(),
		},
		{
			ID:             "eval-3",
			TenantID:       "tenant-b",
			SourceCaseID:   "case-3",
			SourceTaskID:   "task-3",
			SourceReportID: "report-3",
			VersionID:      "version-b",
			Title:          "Eval 3",
			Summary:        "Third",
			CreatedBy:      "operator-3",
			CreatedAt:      time.Unix(1700000003, 0).UTC(),
		},
	}
	for _, item := range fixtures {
		if _, err := store.Save(context.Background(), item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.List(context.Background(), ListFilter{
		TenantID:  "tenant-a",
		VersionID: "version-b",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.EvalCases) != 1 || page.EvalCases[0].ID != "eval-2" {
		t.Fatalf("EvalCases = %#v, want eval-2", page.EvalCases)
	}

	page, err = store.List(context.Background(), ListFilter{
		TenantID: "tenant-a",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("List(paginated) error = %v", err)
	}
	if len(page.EvalCases) != 1 {
		t.Fatalf("len(EvalCases) = %d, want 1", len(page.EvalCases))
	}
	if page.EvalCases[0].ID != "eval-2" {
		t.Fatalf("first EvalCase ID = %q, want %q", page.EvalCases[0].ID, "eval-2")
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}
}

func TestDatasetServiceListSupportsFiltersAndPagination(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)

	firstCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "First source",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	firstEval, _, err := evalService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: firstCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}

	secondCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Second source",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	secondEval, _, err := evalService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: secondCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}

	if _, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		Name:        "Older dataset",
		EvalCaseIDs: []string{firstEval.ID},
		CreatedBy:   "operator-a",
	}); err != nil {
		t.Fatalf("CreateDataset(first) error = %v", err)
	}
	secondDataset, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		Name:        "Newer dataset",
		EvalCaseIDs: []string{secondEval.ID},
		CreatedBy:   "operator-b",
	})
	if err != nil {
		t.Fatalf("CreateDataset(second) error = %v", err)
	}

	page, err := datasetService.ListDatasets(ctx, DatasetListFilter{
		TenantID:  "tenant-dataset",
		CreatedBy: "operator-b",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListDatasets(filtered) error = %v", err)
	}
	if len(page.Datasets) != 1 || page.Datasets[0].ID != secondDataset.ID {
		t.Fatalf("Datasets = %#v, want only %q", page.Datasets, secondDataset.ID)
	}

	page, err = datasetService.ListDatasets(ctx, DatasetListFilter{
		TenantID: "tenant-dataset",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListDatasets(paginated) error = %v", err)
	}
	if len(page.Datasets) != 1 {
		t.Fatalf("len(Datasets) = %d, want 1", len(page.Datasets))
	}
	if page.Datasets[0].ID != secondDataset.ID {
		t.Fatalf("first dataset ID = %q, want %q", page.Datasets[0].ID, secondDataset.ID)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}
}

func TestDatasetServiceCreateRejectsDuplicateEvalCaseIDs(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Duplicate source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-dataset",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}

	_, err = datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID, evalCase.ID},
	})
	if !errors.Is(err, ErrInvalidEvalDataset) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalDataset)
	}
}

type caseReaderFunc func(ctx context.Context, caseID string) (casesvc.Case, error)

func (fn caseReaderFunc) GetCase(ctx context.Context, caseID string) (casesvc.Case, error) {
	return fn(ctx, caseID)
}

type traceLookupFunc func(ctx context.Context, input tracedetail.LookupInput) (tracedetail.Result, error)

func (fn traceLookupFunc) Lookup(ctx context.Context, input tracedetail.LookupInput) (tracedetail.Result, error) {
	return fn(ctx, input)
}
