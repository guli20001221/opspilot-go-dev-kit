package eval

import (
	"context"
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
	})
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
	})
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
	})
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
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	if _, err := service.RetryRun(ctx, run.ID); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
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
}
