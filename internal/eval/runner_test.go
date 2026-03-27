package eval

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type cancelingRunExecutor struct {
	cancel context.CancelFunc
	err    error
}

func (e cancelingRunExecutor) ExecuteRun(_ context.Context, _ EvalRun) error {
	e.cancel()
	return e.err
}

func TestRunnerProcessesQueuedRunToSucceeded(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-success",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-success",
		DatasetName:      "Published baseline",
		DatasetItemCount: 2,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030100, 0).UTC(),
		UpdatedAt:        time.Unix(1700030100, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-a", Title: "Eval A", SourceCaseID: "case-a", TraceID: "trace-a"}, EvalRunItem{EvalCaseID: "eval-case-b", Title: "Eval B", SourceCaseID: "case-b", TraceID: "trace-b"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	runner := NewRunner(service, NewPlaceholderRunExecutor())
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	got, err := service.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.Status != RunStatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, RunStatusSucceeded)
	}
	if got.StartedAt.IsZero() {
		t.Fatal("StartedAt is zero")
	}
	if got.FinishedAt.IsZero() {
		t.Fatal("FinishedAt is zero")
	}
	if got.ErrorReason != "" {
		t.Fatalf("ErrorReason = %q, want empty", got.ErrorReason)
	}
	detail, err := service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.ItemResults) != 2 {
		t.Fatalf("len(detail.ItemResults) = %d, want 2", len(detail.ItemResults))
	}
	for _, result := range detail.ItemResults {
		if result.Status != RunItemResultSucceeded {
			t.Fatalf("result.Status = %q, want %q", result.Status, RunItemResultSucceeded)
		}
		if result.Verdict != "pass" {
			t.Fatalf("result.Verdict = %q, want %q", result.Verdict, "pass")
		}
		if result.Score != 1 {
			t.Fatalf("result.Score = %v, want 1", result.Score)
		}
		if result.JudgeVersion == "" {
			t.Fatal("result.JudgeVersion is empty")
		}
		if len(result.JudgeOutput) == 0 {
			t.Fatal("result.JudgeOutput is empty")
		}
	}
}

func TestRunnerProcessesQueuedRunToFailed(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-failed",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-failed",
		DatasetName:      "Published baseline",
		DatasetItemCount: 2,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030100, 0).UTC(),
		UpdatedAt:        time.Unix(1700030100, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-a", Title: "Eval A", SourceCaseID: "case-a", TraceID: "trace-a"}, EvalRunItem{EvalCaseID: "eval-case-b", Title: "Eval B", SourceCaseID: "case-b", TraceID: "trace-b"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	executor := NewPlaceholderRunExecutor()
	executor.FailAll = true
	runner := NewRunner(service, executor)
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	got, err := service.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.Status != RunStatusFailed {
		t.Fatalf("Status = %q, want %q", got.Status, RunStatusFailed)
	}
	if got.StartedAt.IsZero() {
		t.Fatal("StartedAt is zero")
	}
	if got.FinishedAt.IsZero() {
		t.Fatal("FinishedAt is zero")
	}
	if got.ErrorReason == "" {
		t.Fatal("ErrorReason is empty")
	}
	detail, err := service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.ItemResults) != 2 {
		t.Fatalf("len(detail.ItemResults) = %d, want 2", len(detail.ItemResults))
	}
	for _, result := range detail.ItemResults {
		if result.Status != RunItemResultFailed {
			t.Fatalf("result.Status = %q, want %q", result.Status, RunItemResultFailed)
		}
		if result.Detail == "" {
			t.Fatal("result.Detail is empty")
		}
		if result.Verdict != "fail" {
			t.Fatalf("result.Verdict = %q, want %q", result.Verdict, "fail")
		}
		if result.Score != 0 {
			t.Fatalf("result.Score = %v, want 0", result.Score)
		}
		if result.JudgeVersion == "" {
			t.Fatal("result.JudgeVersion is empty")
		}
		if len(result.JudgeOutput) == 0 {
			t.Fatal("result.JudgeOutput is empty")
		}
	}
}

func TestRunnerFinalizesRunAfterExecutionContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-canceled",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-canceled",
		DatasetName:      "Published baseline",
		DatasetItemCount: 2,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030100, 0).UTC(),
		UpdatedAt:        time.Unix(1700030100, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-a", Title: "Eval A", SourceCaseID: "case-a", TraceID: "trace-a"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	runner := NewRunner(service, cancelingRunExecutor{
		cancel: cancel,
		err:    errors.New("context canceled during execution"),
	})
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	got, err := service.GetRun(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.Status != RunStatusFailed {
		t.Fatalf("Status = %q, want %q", got.Status, RunStatusFailed)
	}
	if got.FinishedAt.IsZero() {
		t.Fatal("FinishedAt is zero")
	}
	if got.ErrorReason == "" {
		t.Fatal("ErrorReason is empty")
	}
}

func TestRunnerProcessesRetriedRunToSucceeded(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithStore(store, nil)

	failedAt := time.Unix(1700030200, 0).UTC()
	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-retried-success",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-success",
		DatasetName:      "Published baseline",
		DatasetItemCount: 2,
		Status:           RunStatusFailed,
		CreatedBy:        "operator",
		ErrorReason:      "fault injection",
		CreatedAt:        time.Unix(1700030100, 0).UTC(),
		UpdatedAt:        failedAt,
		StartedAt:        time.Unix(1700030150, 0).UTC(),
		FinishedAt:       failedAt,
	}, EvalRunItem{EvalCaseID: "eval-case-a", Title: "Eval A", SourceCaseID: "case-a", TraceID: "trace-a"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if _, err := service.RetryRun(ctx, run.ID); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	detail, err := service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail(after retry) error = %v", err)
	}
	if len(detail.ItemResults) != 0 {
		t.Fatalf("len(detail.ItemResults) = %d, want 0 after retry", len(detail.ItemResults))
	}

	runner := NewRunner(service, NewPlaceholderRunExecutor())
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	got, err := service.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.Status != RunStatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, RunStatusSucceeded)
	}
	if got.ErrorReason != "" {
		t.Fatalf("ErrorReason = %q, want empty", got.ErrorReason)
	}
	if got.StartedAt.IsZero() {
		t.Fatal("StartedAt is zero")
	}
	if got.FinishedAt.IsZero() {
		t.Fatal("FinishedAt is zero")
	}
	detail, err = service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() after retry success error = %v", err)
	}
	var judgeOutput map[string]any
	if err := json.Unmarshal(detail.ItemResults[0].JudgeOutput, &judgeOutput); err != nil {
		t.Fatalf("Unmarshal(JudgeOutput) error = %v", err)
	}
	if judgeOutput["judge_kind"] != "placeholder" {
		t.Fatalf("judge_kind = %#v, want %q", judgeOutput["judge_kind"], "placeholder")
	}
}

func TestRunnerMaterializesEvalReportAfterSucceededRun(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	runService := NewRunServiceWithStore(store, nil)
	reportService := NewEvalReportServiceWithDependencies(store, runService)

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-report-success",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-success",
		DatasetName:      "Published baseline",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030300, 0).UTC(),
		UpdatedAt:        time.Unix(1700030300, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-a", Title: "Eval A", SourceCaseID: "case-a", TraceID: "trace-a"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	runner := NewRunnerWithReports(runService, NewPlaceholderRunExecutor(), reportService)
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	report, err := reportService.GetEvalReport(ctx, EvalReportIDFromRunID(run.ID))
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if report.RunID != run.ID {
		t.Fatalf("RunID = %q, want %q", report.RunID, run.ID)
	}
	if report.PassedItems != 1 || report.FailedItems != 0 {
		t.Fatalf("report counts = %#v, want one passed item", report)
	}
}

func TestRunnerMaterializesEvalReportAfterFailedRun(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	runService := NewRunServiceWithStore(store, nil)
	reportService := NewEvalReportServiceWithDependencies(store, runService)

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-report-failed",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-failed",
		DatasetName:      "Published baseline",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030400, 0).UTC(),
		UpdatedAt:        time.Unix(1700030400, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-fail", Title: "Eval Fail", SourceCaseID: "case-fail", TraceID: "trace-fail"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	executor := NewPlaceholderRunExecutor()
	executor.FailAll = true
	runner := NewRunnerWithReports(runService, executor, reportService)
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	report, err := reportService.GetEvalReport(ctx, EvalReportIDFromRunID(run.ID))
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if report.RunStatus != RunStatusFailed {
		t.Fatalf("RunStatus = %q, want %q", report.RunStatus, RunStatusFailed)
	}
	if report.PassedItems != 0 || report.FailedItems != 1 {
		t.Fatalf("report counts = %#v, want one failed item", report)
	}
	if len(report.BadCases) != 1 || report.BadCases[0].EvalCaseID != "eval-case-fail" {
		t.Fatalf("BadCases = %#v, want failed case lineage", report.BadCases)
	}
}
