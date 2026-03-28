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

func TestServiceListEvalCasesIncludesFollowUpSummary(t *testing.T) {
	caseService := casesvc.NewService()
	service := NewService(caseService, nil)

	sourceCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-1",
		Title:    "Promote for summary",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceCase) error = %v", err)
	}
	evalCase, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-1",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	followUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-1",
		Title:              "Follow-up",
		SourceEvalReportID: "eval-report-1",
		SourceEvalCaseID:   evalCase.ID,
		CreatedBy:          "operator-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(followUp) error = %v", err)
	}

	page, err := service.ListEvalCases(context.Background(), ListFilter{
		TenantID: "tenant-1",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListEvalCases() error = %v", err)
	}
	if len(page.EvalCases) != 1 {
		t.Fatalf("len(EvalCases) = %d, want 1", len(page.EvalCases))
	}
	got := page.EvalCases[0]
	if got.FollowUpCaseCount != 1 {
		t.Fatalf("FollowUpCaseCount = %d, want %d", got.FollowUpCaseCount, 1)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want %d", got.OpenFollowUpCaseCount, 1)
	}
	if got.LatestFollowUpCaseID != followUp.ID {
		t.Fatalf("LatestFollowUpCaseID = %q, want %q", got.LatestFollowUpCaseID, followUp.ID)
	}
	if got.LatestFollowUpCaseStatus != casesvc.StatusOpen {
		t.Fatalf("LatestFollowUpCaseStatus = %q, want %q", got.LatestFollowUpCaseStatus, casesvc.StatusOpen)
	}

	detail, err := service.GetEvalCase(context.Background(), evalCase.ID)
	if err != nil {
		t.Fatalf("GetEvalCase() error = %v", err)
	}
	if detail.LatestFollowUpCaseID != followUp.ID {
		t.Fatalf("GetEvalCase().LatestFollowUpCaseID = %q, want %q", detail.LatestFollowUpCaseID, followUp.ID)
	}
}

func TestServiceListEvalCasesSupportsNeedsFollowUpFilter(t *testing.T) {
	caseService := casesvc.NewService()
	service := NewService(caseService, nil)

	sourceWithoutFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-filter",
		Title:    "No follow-up",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceWithoutFollowUp) error = %v", err)
	}
	evalWithoutFollowUp, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-filter",
		SourceCaseID: sourceWithoutFollowUp.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalWithoutFollowUp) error = %v", err)
	}

	sourceWithOpenFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-filter",
		Title:    "Open follow-up",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceWithOpenFollowUp) error = %v", err)
	}
	evalWithOpenFollowUp, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-filter",
		SourceCaseID: sourceWithOpenFollowUp.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalWithOpenFollowUp) error = %v", err)
	}
	if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-filter",
		Title:              "Linked open follow-up",
		SourceEvalCaseID:   evalWithOpenFollowUp.ID,
		SourceEvalReportID: "eval-report-open",
	}); err != nil {
		t.Fatalf("CreateCase(open follow-up) error = %v", err)
	}

	sourceWithClosedFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID: "tenant-filter",
		Title:    "Closed follow-up",
	})
	if err != nil {
		t.Fatalf("CreateCase(sourceWithClosedFollowUp) error = %v", err)
	}
	evalWithClosedFollowUp, _, err := service.PromoteCase(context.Background(), CreateInput{
		TenantID:     "tenant-filter",
		SourceCaseID: sourceWithClosedFollowUp.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(evalWithClosedFollowUp) error = %v", err)
	}
	closedFollowUp, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
		TenantID:           "tenant-filter",
		Title:              "Linked closed follow-up",
		SourceEvalCaseID:   evalWithClosedFollowUp.ID,
		SourceEvalReportID: "eval-report-closed",
		CreatedBy:          "operator-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(closed follow-up) error = %v", err)
	}
	if _, err := caseService.CloseCase(context.Background(), closedFollowUp.ID, "operator-2"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	trueValue := true
	openOnly, err := service.ListEvalCases(context.Background(), ListFilter{
		TenantID:      "tenant-filter",
		NeedsFollowUp: &trueValue,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListEvalCases(openOnly) error = %v", err)
	}
	if len(openOnly.EvalCases) != 1 {
		t.Fatalf("len(openOnly.EvalCases) = %d, want 1", len(openOnly.EvalCases))
	}
	if openOnly.EvalCases[0].ID != evalWithOpenFollowUp.ID {
		t.Fatalf("openOnly.EvalCases[0].ID = %q, want %q", openOnly.EvalCases[0].ID, evalWithOpenFollowUp.ID)
	}

	falseValue := false
	clearOnly, err := service.ListEvalCases(context.Background(), ListFilter{
		TenantID:      "tenant-filter",
		NeedsFollowUp: &falseValue,
		Limit:         10,
	})
	if err != nil {
		t.Fatalf("ListEvalCases(clearOnly) error = %v", err)
	}
	if len(clearOnly.EvalCases) != 2 {
		t.Fatalf("len(clearOnly.EvalCases) = %d, want 2", len(clearOnly.EvalCases))
	}
	ids := []string{clearOnly.EvalCases[0].ID, clearOnly.EvalCases[1].ID}
	if !(containsString(ids, evalWithoutFollowUp.ID) && containsString(ids, evalWithClosedFollowUp.ID)) {
		t.Fatalf("clearOnly IDs = %#v, want %q and %q", ids, evalWithoutFollowUp.ID, evalWithClosedFollowUp.ID)
	}
}

func TestServiceListEvalCasesNeedsFollowUpFilterPreservesPagination(t *testing.T) {
	caseService := casesvc.NewService()
	service := NewService(caseService, nil)

	newEvalWithOpenFollowUp := func(title string, reportID string) EvalCase {
		t.Helper()

		sourceCase, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID: "tenant-follow-up-pagination",
			Title:    title,
		})
		if err != nil {
			t.Fatalf("CreateCase(%s) error = %v", title, err)
		}
		evalCase, _, err := service.PromoteCase(context.Background(), CreateInput{
			TenantID:     "tenant-follow-up-pagination",
			SourceCaseID: sourceCase.ID,
		})
		if err != nil {
			t.Fatalf("PromoteCase(%s) error = %v", title, err)
		}
		if _, err := caseService.CreateCase(context.Background(), casesvc.CreateInput{
			TenantID:           "tenant-follow-up-pagination",
			Title:              title + " follow-up",
			SourceEvalCaseID:   evalCase.ID,
			SourceEvalReportID: reportID,
		}); err != nil {
			t.Fatalf("CreateCase(%s follow-up) error = %v", title, err)
		}
		return evalCase
	}

	first := newEvalWithOpenFollowUp("First open", "eval-report-open-1")
	time.Sleep(2 * time.Millisecond)
	second := newEvalWithOpenFollowUp("Second open", "eval-report-open-2")

	trueValue := true
	pageOne, err := service.ListEvalCases(context.Background(), ListFilter{
		TenantID:      "tenant-follow-up-pagination",
		NeedsFollowUp: &trueValue,
		Limit:         1,
	})
	if err != nil {
		t.Fatalf("ListEvalCases(pageOne) error = %v", err)
	}
	if len(pageOne.EvalCases) != 1 {
		t.Fatalf("len(pageOne.EvalCases) = %d, want 1", len(pageOne.EvalCases))
	}
	if pageOne.EvalCases[0].ID != second.ID {
		t.Fatalf("pageOne.EvalCases[0].ID = %q, want %q", pageOne.EvalCases[0].ID, second.ID)
	}
	if !pageOne.HasMore || pageOne.NextOffset != 1 {
		t.Fatalf("pageOne pagination = %#v, want has_more with next_offset=1", pageOne)
	}

	pageTwo, err := service.ListEvalCases(context.Background(), ListFilter{
		TenantID:      "tenant-follow-up-pagination",
		NeedsFollowUp: &trueValue,
		Limit:         1,
		Offset:        1,
	})
	if err != nil {
		t.Fatalf("ListEvalCases(pageTwo) error = %v", err)
	}
	if len(pageTwo.EvalCases) != 1 {
		t.Fatalf("len(pageTwo.EvalCases) = %d, want 1", len(pageTwo.EvalCases))
	}
	if pageTwo.EvalCases[0].ID != first.ID {
		t.Fatalf("pageTwo.EvalCases[0].ID = %q, want %q", pageTwo.EvalCases[0].ID, first.ID)
	}
	if pageTwo.HasMore {
		t.Fatalf("pageTwo.HasMore = true, want false")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
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

func TestDatasetServiceAddDatasetItemAppendsAndIsIdempotent(t *testing.T) {
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

	created, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{firstEval.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	updated, err := datasetService.AddDatasetItem(ctx, created.ID, AddDatasetItemInput{
		TenantID:   "tenant-dataset",
		EvalCaseID: secondEval.ID,
	})
	if err != nil {
		t.Fatalf("AddDatasetItem() error = %v", err)
	}
	if len(updated.Items) != 2 || updated.Items[1].EvalCaseID != secondEval.ID {
		t.Fatalf("Items = %#v, want appended second eval case", updated.Items)
	}

	unchanged, err := datasetService.AddDatasetItem(ctx, created.ID, AddDatasetItemInput{
		TenantID:   "tenant-dataset",
		EvalCaseID: secondEval.ID,
	})
	if err != nil {
		t.Fatalf("AddDatasetItem(idempotent) error = %v", err)
	}
	if len(unchanged.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(unchanged.Items))
	}
}

func TestDatasetServiceAddDatasetItemRejectsPublishedDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Source",
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

	created, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, created.ID, PublishDatasetInput{
		TenantID: "tenant-dataset",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	_, err = datasetService.AddDatasetItem(ctx, created.ID, AddDatasetItemInput{
		TenantID:   "tenant-dataset",
		EvalCaseID: evalCase.ID,
	})
	if !errors.Is(err, ErrInvalidEvalDatasetState) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalDatasetState)
	}
}

func TestDatasetServicePublishDatasetTransitionsDraftToPublished(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Source",
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

	created, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	published, err := datasetService.PublishDataset(ctx, created.ID, PublishDatasetInput{
		TenantID:    "tenant-dataset",
		PublishedBy: "operator-publish",
	})
	if err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}
	if published.Status != DatasetStatusPublished {
		t.Fatalf("Status = %q, want %q", published.Status, DatasetStatusPublished)
	}
	if published.PublishedBy != "operator-publish" {
		t.Fatalf("PublishedBy = %q, want %q", published.PublishedBy, "operator-publish")
	}
	if published.PublishedAt.IsZero() {
		t.Fatal("PublishedAt is zero")
	}
}

func TestDatasetServicePublishDatasetRejectsRepublish(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-dataset",
		Title:    "Source",
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

	created, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-dataset",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, created.ID, PublishDatasetInput{
		TenantID: "tenant-dataset",
	}); err != nil {
		t.Fatalf("PublishDataset(first) error = %v", err)
	}
	_, err = datasetService.PublishDataset(ctx, created.ID, PublishDatasetInput{
		TenantID: "tenant-dataset",
	})
	if !errors.Is(err, ErrInvalidEvalDatasetState) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalDatasetState)
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
