package eval

import (
	"context"
	"errors"
	"testing"
	"time"

	casesvc "opspilot-go/internal/case"
)

func TestRunServiceCreateRunRequiresPublishedDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)
	runService := NewRunService(datasetService)

	sourceCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Source",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	evalCase, _, err := evalService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: sourceCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase() error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-run",
		EvalCaseIDs: []string{evalCase.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	_, err = runService.CreateRun(ctx, CreateRunInput{
		TenantID:  "tenant-run",
		DatasetID: dataset.ID,
	})
	if !errors.Is(err, ErrInvalidEvalDatasetState) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalDatasetState)
	}
}

func TestRunServiceCreateRunFromPublishedDataset(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)
	runService := NewRunService(datasetService)

	firstCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "First source",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	firstEval, _, err := evalService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: firstCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(first) error = %v", err)
	}
	secondCase, err := caseService.CreateCase(ctx, casesvc.CreateInput{
		TenantID: "tenant-run",
		Title:    "Second source",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	secondEval, _, err := evalService.PromoteCase(ctx, CreateInput{
		TenantID:     "tenant-run",
		SourceCaseID: secondCase.ID,
	})
	if err != nil {
		t.Fatalf("PromoteCase(second) error = %v", err)
	}
	dataset, err := datasetService.CreateDataset(ctx, CreateDatasetInput{
		TenantID:    "tenant-run",
		Name:        "Published baseline",
		EvalCaseIDs: []string{firstEval.ID, secondEval.ID},
	})
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if _, err := datasetService.PublishDataset(ctx, dataset.ID, PublishDatasetInput{
		TenantID:    "tenant-run",
		PublishedBy: "operator-publish",
	}); err != nil {
		t.Fatalf("PublishDataset() error = %v", err)
	}

	run, err := runService.CreateRun(ctx, CreateRunInput{
		TenantID:  "tenant-run",
		DatasetID: dataset.ID,
		CreatedBy: "operator-run",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if run.Status != RunStatusQueued {
		t.Fatalf("Status = %q, want %q", run.Status, RunStatusQueued)
	}
	if run.DatasetName != "Published baseline" {
		t.Fatalf("DatasetName = %q, want %q", run.DatasetName, "Published baseline")
	}
	if run.DatasetItemCount != 2 {
		t.Fatalf("DatasetItemCount = %d, want %d", run.DatasetItemCount, 2)
	}
	if run.CreatedBy != "operator-run" {
		t.Fatalf("CreatedBy = %q, want %q", run.CreatedBy, "operator-run")
	}

	detail, err := runService.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.Items) != 2 {
		t.Fatalf("len(detail.Items) = %d, want 2", len(detail.Items))
	}
	if detail.Items[0].EvalCaseID != firstEval.ID || detail.Items[1].EvalCaseID != secondEval.ID {
		t.Fatalf("detail.Items = %#v, want published dataset membership order", detail.Items)
	}
	if detail.Items[0].Title != "First source" || detail.Items[1].Title != "Second source" {
		t.Fatalf("detail.Items titles = %#v, want source case titles", detail.Items)
	}
	if detail.Items[0].SourceCaseID != firstCase.ID || detail.Items[1].SourceCaseID != secondCase.ID {
		t.Fatalf("detail.Items source_case_id = %#v, want source case lineage", detail.Items)
	}
}

func TestRunServiceDetailKeepsSnappedItemsAfterDatasetDrift(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	reader := &stubRunDatasetReader{
		dataset: EvalDataset{
			ID:          "eval-dataset-snap",
			TenantID:    "tenant-run",
			Name:        "Published baseline",
			Status:      DatasetStatusPublished,
			CreatedBy:   "operator",
			CreatedAt:   time.Unix(1700031000, 0).UTC(),
			UpdatedAt:   time.Unix(1700031000, 0).UTC(),
			PublishedBy: "operator",
			PublishedAt: time.Unix(1700031000, 0).UTC(),
			Items: []EvalDatasetItem{
				{
					EvalCaseID:   "eval-case-a",
					Title:        "Original title",
					SourceCaseID: "case-a",
					TraceID:      "trace-a",
					VersionID:    "version-a",
				},
			},
		},
	}
	service := NewRunServiceWithStore(store, reader)

	run, err := service.CreateRun(ctx, CreateRunInput{
		TenantID:  "tenant-run",
		DatasetID: "eval-dataset-snap",
		CreatedBy: "operator-run",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	reader.dataset.Items[0].Title = "Mutated title"
	reader.dataset.Items = append(reader.dataset.Items, EvalDatasetItem{
		EvalCaseID:   "eval-case-b",
		Title:        "Late item",
		SourceCaseID: "case-b",
		TraceID:      "trace-b",
		VersionID:    "version-b",
	})

	detail, err := service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("len(detail.Items) = %d, want 1", len(detail.Items))
	}
	if detail.Items[0].Title != "Original title" {
		t.Fatalf("detail.Items[0].Title = %q, want %q", detail.Items[0].Title, "Original title")
	}
}

func TestRunServiceListRunsSupportsFilters(t *testing.T) {
	ctx := context.Background()
	caseService := casesvc.NewService()
	evalService := NewService(caseService, nil)
	datasetService := NewDatasetService(evalService)
	runService := NewRunService(datasetService)

	firstCase, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-run", Title: "First"})
	firstEval, _, _ := evalService.PromoteCase(ctx, CreateInput{TenantID: "tenant-run", SourceCaseID: firstCase.ID})
	secondCase, _ := caseService.CreateCase(ctx, casesvc.CreateInput{TenantID: "tenant-run", Title: "Second"})
	secondEval, _, _ := evalService.PromoteCase(ctx, CreateInput{TenantID: "tenant-run", SourceCaseID: secondCase.ID})

	firstDataset, _ := datasetService.CreateDataset(ctx, CreateDatasetInput{TenantID: "tenant-run", Name: "A", EvalCaseIDs: []string{firstEval.ID}})
	secondDataset, _ := datasetService.CreateDataset(ctx, CreateDatasetInput{TenantID: "tenant-run", Name: "B", EvalCaseIDs: []string{secondEval.ID}})
	_, _ = datasetService.PublishDataset(ctx, firstDataset.ID, PublishDatasetInput{TenantID: "tenant-run"})
	_, _ = datasetService.PublishDataset(ctx, secondDataset.ID, PublishDatasetInput{TenantID: "tenant-run"})

	secondRun, err := runService.CreateRun(ctx, CreateRunInput{TenantID: "tenant-run", DatasetID: secondDataset.ID})
	if err != nil {
		t.Fatalf("CreateRun(second) error = %v", err)
	}
	if _, err := runService.CreateRun(ctx, CreateRunInput{TenantID: "tenant-run", DatasetID: firstDataset.ID}); err != nil {
		t.Fatalf("CreateRun(first) error = %v", err)
	}

	page, err := runService.ListRuns(ctx, RunListFilter{
		TenantID:  "tenant-run",
		DatasetID: secondDataset.ID,
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(page.Runs) != 1 || page.Runs[0].ID != secondRun.ID {
		t.Fatalf("Runs = %#v, want only %q", page.Runs, secondRun.ID)
	}
}

type stubRunDatasetReader struct {
	dataset EvalDataset
}

func (s *stubRunDatasetReader) GetDataset(_ context.Context, datasetID string) (EvalDataset, error) {
	if s.dataset.ID != datasetID {
		return EvalDataset{}, ErrEvalDatasetNotFound
	}
	return s.dataset, nil
}

func TestRunServiceClaimAndFinalize(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	if _, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-claim",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-claim",
		DatasetName:      "Published baseline",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030000, 0).UTC(),
		UpdatedAt:        time.Unix(1700030000, 0).UTC(),
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	claimed, err := service.ClaimQueuedRuns(ctx, 10)
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("len(claimed) = %d, want 1", len(claimed))
	}
	if claimed[0].Status != RunStatusRunning {
		t.Fatalf("Status = %q, want %q", claimed[0].Status, RunStatusRunning)
	}
	if claimed[0].StartedAt.IsZero() {
		t.Fatal("StartedAt is zero")
	}

	succeeded, err := service.MarkRunSucceeded(ctx, claimed[0].ID)
	if err != nil {
		t.Fatalf("MarkRunSucceeded() error = %v", err)
	}
	if succeeded.Status != RunStatusSucceeded {
		t.Fatalf("Status = %q, want %q", succeeded.Status, RunStatusSucceeded)
	}
	if succeeded.FinishedAt.IsZero() {
		t.Fatal("FinishedAt is zero")
	}
}

func TestRunServiceMarkRunFailedRequiresRunningState(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	if _, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-state",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-state",
		DatasetName:      "Published baseline",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030000, 0).UTC(),
		UpdatedAt:        time.Unix(1700030000, 0).UTC(),
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	_, err := service.MarkRunFailed(ctx, "eval-run-state", "boom")
	if !errors.Is(err, ErrInvalidEvalRunState) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalRunState)
	}
}

func TestRunServiceClaimQueuedRunsUsesFIFOOrder(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	for _, item := range []EvalRun{
		{
			ID:               "eval-run-oldest",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-a",
			DatasetName:      "Dataset A",
			DatasetItemCount: 1,
			Status:           RunStatusQueued,
			CreatedBy:        "operator",
			CreatedAt:        time.Unix(1700030000, 0).UTC(),
			UpdatedAt:        time.Unix(1700030000, 0).UTC(),
		},
		{
			ID:               "eval-run-newest",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-b",
			DatasetName:      "Dataset B",
			DatasetItemCount: 1,
			Status:           RunStatusQueued,
			CreatedBy:        "operator",
			CreatedAt:        time.Unix(1700030100, 0).UTC(),
			UpdatedAt:        time.Unix(1700030100, 0).UTC(),
		},
	} {
		if _, err := store.CreateRun(ctx, item); err != nil {
			t.Fatalf("CreateRun(%s) error = %v", item.ID, err)
		}
	}

	claimed, err := service.ClaimQueuedRuns(ctx, 2)
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("len(claimed) = %d, want 2", len(claimed))
	}
	if claimed[0].ID != "eval-run-oldest" || claimed[1].ID != "eval-run-newest" {
		t.Fatalf("claim order = %#v, want oldest-first", claimed)
	}
}

func TestRunServiceListRunsUsesLatestUpdatedFirstOrder(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	for _, item := range []EvalRun{
		{
			ID:               "eval-run-older",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-a",
			DatasetName:      "Dataset A",
			DatasetItemCount: 1,
			Status:           RunStatusQueued,
			CreatedBy:        "operator-a",
			CreatedAt:        time.Unix(1700030000, 0).UTC(),
			UpdatedAt:        time.Unix(1700030005, 0).UTC(),
		},
		{
			ID:               "eval-run-newer",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-b",
			DatasetName:      "Dataset B",
			DatasetItemCount: 1,
			Status:           RunStatusRunning,
			CreatedBy:        "operator-b",
			CreatedAt:        time.Unix(1700030010, 0).UTC(),
			UpdatedAt:        time.Unix(1700030020, 0).UTC(),
		},
	} {
		if _, err := store.CreateRun(ctx, item); err != nil {
			t.Fatalf("CreateRun(%s) error = %v", item.ID, err)
		}
	}

	page, err := service.ListRuns(ctx, RunListFilter{
		TenantID: "tenant-run",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(page.Runs) != 2 {
		t.Fatalf("len(page.Runs) = %d, want 2", len(page.Runs))
	}
	if page.Runs[0].ID != "eval-run-newer" || page.Runs[1].ID != "eval-run-older" {
		t.Fatalf("run order = %#v, want latest-updated-first", page.Runs)
	}
}

func TestRunServiceRetryRunRequeuesFailedRun(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	failedAt := time.Unix(1700030200, 0).UTC()
	if _, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-retry",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-retry",
		DatasetName:      "Dataset Retry",
		DatasetItemCount: 2,
		Status:           RunStatusFailed,
		CreatedBy:        "operator",
		ErrorReason:      "fault injection",
		CreatedAt:        time.Unix(1700030000, 0).UTC(),
		UpdatedAt:        failedAt,
		StartedAt:        time.Unix(1700030100, 0).UTC(),
		FinishedAt:       failedAt,
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	retried, err := service.RetryRun(ctx, "eval-run-retry")
	if err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if retried.Status != RunStatusQueued {
		t.Fatalf("Status = %q, want %q", retried.Status, RunStatusQueued)
	}
	if retried.ErrorReason != "" {
		t.Fatalf("ErrorReason = %q, want empty", retried.ErrorReason)
	}
	if !retried.StartedAt.IsZero() {
		t.Fatalf("StartedAt = %v, want zero", retried.StartedAt)
	}
	if !retried.FinishedAt.IsZero() {
		t.Fatalf("FinishedAt = %v, want zero", retried.FinishedAt)
	}
	if !retried.UpdatedAt.After(failedAt) {
		t.Fatalf("UpdatedAt = %v, want after %v", retried.UpdatedAt, failedAt)
	}
}

func TestRunServiceRetryRunRequiresFailedState(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	if _, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-not-failed",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-retry",
		DatasetName:      "Dataset Retry",
		DatasetItemCount: 1,
		Status:           RunStatusSucceeded,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030000, 0).UTC(),
		UpdatedAt:        time.Unix(1700030100, 0).UTC(),
		StartedAt:        time.Unix(1700030050, 0).UTC(),
		FinishedAt:       time.Unix(1700030100, 0).UTC(),
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	_, err := service.RetryRun(ctx, "eval-run-not-failed")
	if !errors.Is(err, ErrInvalidEvalRunState) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalRunState)
	}
}

func TestRunServiceRetryRunRejectsRunClaimedAfterRetry(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	if _, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-race",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-race",
		DatasetName:      "Dataset Race",
		DatasetItemCount: 1,
		Status:           RunStatusFailed,
		CreatedBy:        "operator",
		ErrorReason:      "fault injection",
		CreatedAt:        time.Unix(1700030000, 0).UTC(),
		UpdatedAt:        time.Unix(1700030100, 0).UTC(),
		StartedAt:        time.Unix(1700030050, 0).UTC(),
		FinishedAt:       time.Unix(1700030100, 0).UTC(),
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if _, err := service.RetryRun(ctx, "eval-run-race"); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if _, err := service.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}

	_, err := service.RetryRun(ctx, "eval-run-race")
	if !errors.Is(err, ErrInvalidEvalRunState) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidEvalRunState)
	}
}

func TestRunServiceListRunEventsPreservesRetryHistory(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-events",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-events",
		DatasetName:      "Dataset Events",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030000, 0).UTC(),
		UpdatedAt:        time.Unix(1700030000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if _, err := service.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns(first) error = %v", err)
	}
	if _, err := service.MarkRunFailed(ctx, run.ID, "fault injection"); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}
	if _, err := service.RetryRun(ctx, run.ID); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if _, err := service.ClaimQueuedRuns(ctx, 10); err != nil {
		t.Fatalf("ClaimQueuedRuns(second) error = %v", err)
	}
	if _, err := service.MarkRunSucceeded(ctx, run.ID); err != nil {
		t.Fatalf("MarkRunSucceeded() error = %v", err)
	}

	events, err := service.ListRunEvents(ctx, run.ID)
	if err != nil {
		t.Fatalf("ListRunEvents() error = %v", err)
	}
	if len(events) != 6 {
		t.Fatalf("len(events) = %d, want 6", len(events))
	}

	actions := make([]string, 0, len(events))
	for _, event := range events {
		actions = append(actions, event.Action)
	}
	want := []string{
		RunEventCreated,
		RunEventClaimed,
		RunEventFailed,
		RunEventRetried,
		RunEventClaimed,
		RunEventSucceeded,
	}
	for i := range want {
		if actions[i] != want[i] {
			t.Fatalf("actions[%d] = %q, want %q (all=%#v)", i, actions[i], want[i], actions)
		}
	}
	if events[2].Detail != "fault injection" {
		t.Fatalf("failed detail = %q, want %q", events[2].Detail, "fault injection")
	}
}
