package eval

import (
	"context"
	"errors"
	"testing"

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
