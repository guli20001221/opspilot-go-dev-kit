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
