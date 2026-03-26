package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	evalsvc "opspilot-go/internal/eval"
)

func TestEvalRunStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	store := NewEvalRunStore(pool)
	want := evalsvc.EvalRun{
		ID:               "eval-run-roundtrip",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-roundtrip",
		DatasetName:      "Published baseline",
		DatasetItemCount: 3,
		Status:           evalsvc.RunStatusQueued,
		CreatedBy:        "operator-run",
		CreatedAt:        time.Unix(1700020000, 0).UTC(),
		UpdatedAt:        time.Unix(1700020000, 0).UTC(),
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator-publish', $5, $5, 'operator-publish', $5
)`,
		want.DatasetID,
		want.TenantID,
		want.DatasetName,
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	created, err := store.CreateRun(ctx, want)
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if created.ID != want.ID {
		t.Fatalf("ID = %q, want %q", created.ID, want.ID)
	}

	got, err := store.GetRun(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.DatasetName != want.DatasetName {
		t.Fatalf("DatasetName = %q, want %q", got.DatasetName, want.DatasetName)
	}
	if got.DatasetItemCount != want.DatasetItemCount {
		t.Fatalf("DatasetItemCount = %d, want %d", got.DatasetItemCount, want.DatasetItemCount)
	}
}

func TestEvalRunStoreListRunsSupportsFiltersAndPagination(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES
    ('eval-dataset-a', 'tenant-run', 'Dataset A', '', $1, 'operator', $2, $2, 'operator', $2),
    ('eval-dataset-b', 'tenant-run', 'Dataset B', '', $1, 'operator', $3, $3, 'operator', $3)
`,
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
		time.Unix(1700019910, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	store := NewEvalRunStore(pool)
	for _, item := range []evalsvc.EvalRun{
		{
			ID:               "eval-run-a",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-a",
			DatasetName:      "Dataset A",
			DatasetItemCount: 1,
			Status:           evalsvc.RunStatusQueued,
			CreatedBy:        "operator-a",
			CreatedAt:        time.Unix(1700020000, 0).UTC(),
			UpdatedAt:        time.Unix(1700020000, 0).UTC(),
		},
		{
			ID:               "eval-run-b",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-b",
			DatasetName:      "Dataset B",
			DatasetItemCount: 2,
			Status:           evalsvc.RunStatusQueued,
			CreatedBy:        "operator-b",
			CreatedAt:        time.Unix(1700020010, 0).UTC(),
			UpdatedAt:        time.Unix(1700020010, 0).UTC(),
		},
	} {
		if _, err := store.CreateRun(ctx, item); err != nil {
			t.Fatalf("CreateRun(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.ListRuns(ctx, evalsvc.RunListFilter{
		TenantID:  "tenant-run",
		DatasetID: "eval-dataset-b",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListRuns(filtered) error = %v", err)
	}
	if len(page.Runs) != 1 || page.Runs[0].ID != "eval-run-b" {
		t.Fatalf("Runs = %#v, want only eval-run-b", page.Runs)
	}

	page, err = store.ListRuns(ctx, evalsvc.RunListFilter{
		TenantID: "tenant-run",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListRuns(paginated) error = %v", err)
	}
	if len(page.Runs) != 1 || page.Runs[0].ID != "eval-run-b" {
		t.Fatalf("first page = %#v, want eval-run-b", page.Runs)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}
}

func TestEvalRunStoreClaimAndUpdate(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator', $5, $5, 'operator', $5
)`,
		"eval-dataset-claim",
		"tenant-run",
		"Dataset claim",
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	store := NewEvalRunStore(pool)
	if _, err := store.CreateRun(ctx, evalsvc.EvalRun{
		ID:               "eval-run-claim",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-claim",
		DatasetName:      "Dataset claim",
		DatasetItemCount: 1,
		Status:           evalsvc.RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700020000, 0).UTC(),
		UpdatedAt:        time.Unix(1700020000, 0).UTC(),
	}); err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	claimed, err := store.ClaimQueuedRuns(ctx, 10, time.Unix(1700020100, 0).UTC())
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("len(claimed) = %d, want 1", len(claimed))
	}
	if claimed[0].Status != evalsvc.RunStatusRunning {
		t.Fatalf("Status = %q, want %q", claimed[0].Status, evalsvc.RunStatusRunning)
	}
	if claimed[0].StartedAt.IsZero() {
		t.Fatal("StartedAt is zero")
	}

	claimed[0].Status = evalsvc.RunStatusFailed
	claimed[0].ErrorReason = "fault injection"
	claimed[0].UpdatedAt = time.Unix(1700020200, 0).UTC()
	claimed[0].FinishedAt = time.Unix(1700020200, 0).UTC()
	updated, err := store.UpdateRun(ctx, claimed[0])
	if err != nil {
		t.Fatalf("UpdateRun() error = %v", err)
	}
	if updated.Status != evalsvc.RunStatusFailed {
		t.Fatalf("Status = %q, want %q", updated.Status, evalsvc.RunStatusFailed)
	}
	if updated.ErrorReason != "fault injection" {
		t.Fatalf("ErrorReason = %q, want %q", updated.ErrorReason, "fault injection")
	}
	if updated.FinishedAt.IsZero() {
		t.Fatal("FinishedAt is zero")
	}
}

func TestEvalRunStoreClaimQueuedRunsUsesFIFOOrder(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES
    ('eval-dataset-a', 'tenant-run', 'Dataset A', '', $1, 'operator', $2, $2, 'operator', $2),
    ('eval-dataset-b', 'tenant-run', 'Dataset B', '', $1, 'operator', $3, $3, 'operator', $3)
`,
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
		time.Unix(1700019910, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	store := NewEvalRunStore(pool)
	for _, item := range []evalsvc.EvalRun{
		{
			ID:               "eval-run-a-oldest",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-a",
			DatasetName:      "Dataset A",
			DatasetItemCount: 1,
			Status:           evalsvc.RunStatusQueued,
			CreatedBy:        "operator-a",
			CreatedAt:        time.Unix(1700020000, 0).UTC(),
			UpdatedAt:        time.Unix(1700020000, 0).UTC(),
		},
		{
			ID:               "eval-run-z-newest",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-b",
			DatasetName:      "Dataset B",
			DatasetItemCount: 1,
			Status:           evalsvc.RunStatusQueued,
			CreatedBy:        "operator-b",
			CreatedAt:        time.Unix(1700020010, 0).UTC(),
			UpdatedAt:        time.Unix(1700020010, 0).UTC(),
		},
	} {
		if _, err := store.CreateRun(ctx, item); err != nil {
			t.Fatalf("CreateRun(%s) error = %v", item.ID, err)
		}
	}

	claimed, err := store.ClaimQueuedRuns(ctx, 2, time.Unix(1700020100, 0).UTC())
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) != 2 {
		t.Fatalf("len(claimed) = %d, want 2", len(claimed))
	}
	if claimed[0].ID != "eval-run-a-oldest" || claimed[1].ID != "eval-run-z-newest" {
		t.Fatalf("claim order = %#v, want oldest-first", claimed)
	}
}

func TestEvalRunStoreListRunsUsesLatestUpdatedFirstOrder(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES
    ('eval-dataset-a', 'tenant-run', 'Dataset A', '', $1, 'operator', $2, $2, 'operator', $2),
    ('eval-dataset-b', 'tenant-run', 'Dataset B', '', $1, 'operator', $3, $3, 'operator', $3)
`,
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
		time.Unix(1700019910, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	store := NewEvalRunStore(pool)
	for _, item := range []evalsvc.EvalRun{
		{
			ID:               "eval-run-older",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-a",
			DatasetName:      "Dataset A",
			DatasetItemCount: 1,
			Status:           evalsvc.RunStatusQueued,
			CreatedBy:        "operator-a",
			CreatedAt:        time.Unix(1700020000, 0).UTC(),
			UpdatedAt:        time.Unix(1700020005, 0).UTC(),
		},
		{
			ID:               "eval-run-newer",
			TenantID:         "tenant-run",
			DatasetID:        "eval-dataset-b",
			DatasetName:      "Dataset B",
			DatasetItemCount: 1,
			Status:           evalsvc.RunStatusRunning,
			CreatedBy:        "operator-b",
			CreatedAt:        time.Unix(1700020010, 0).UTC(),
			UpdatedAt:        time.Unix(1700020020, 0).UTC(),
			StartedAt:        time.Unix(1700020020, 0).UTC(),
		},
	} {
		if _, err := store.CreateRun(ctx, item); err != nil {
			t.Fatalf("CreateRun(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.ListRuns(ctx, evalsvc.RunListFilter{
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

func TestEvalRunStoreUpdateAllowsRetryRequeue(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator', $5, $5, 'operator', $5
)`,
		"eval-dataset-retry",
		"tenant-run",
		"Dataset Retry",
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	store := NewEvalRunStore(pool)
	_, err = store.CreateRun(ctx, evalsvc.EvalRun{
		ID:               "eval-run-retry",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-retry",
		DatasetName:      "Dataset Retry",
		DatasetItemCount: 1,
		Status:           evalsvc.RunStatusFailed,
		CreatedBy:        "operator",
		ErrorReason:      "fault injection",
		CreatedAt:        time.Unix(1700020000, 0).UTC(),
		UpdatedAt:        time.Unix(1700020100, 0).UTC(),
		StartedAt:        time.Unix(1700020050, 0).UTC(),
		FinishedAt:       time.Unix(1700020100, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	retried, err := store.RetryRun(ctx, "eval-run-retry", time.Unix(1700020200, 0).UTC())
	if err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if retried.Status != evalsvc.RunStatusQueued {
		t.Fatalf("Status = %q, want %q", retried.Status, evalsvc.RunStatusQueued)
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

	claimed, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700020300, 0).UTC())
	if err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if len(claimed) != 1 || claimed[0].ID != "eval-run-retry" {
		t.Fatalf("claimed = %#v, want retried run to be claimable again", claimed)
	}

	_, err = store.RetryRun(ctx, "eval-run-retry", time.Unix(1700020400, 0).UTC())
	if !errors.Is(err, evalsvc.ErrInvalidEvalRunState) {
		t.Fatalf("error = %v, want %v", err, evalsvc.ErrInvalidEvalRunState)
	}
}

func TestEvalRunStoreListRunEventsPreservesRetryHistory(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator', $5, $5, 'operator', $5
)`,
		"eval-dataset-events",
		"tenant-run",
		"Dataset Events",
		evalsvc.DatasetStatusPublished,
		time.Unix(1700019900, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}

	store := NewEvalRunStore(pool)
	run, err := store.CreateRun(ctx, evalsvc.EvalRun{
		ID:               "eval-run-events",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-events",
		DatasetName:      "Dataset Events",
		DatasetItemCount: 1,
		Status:           evalsvc.RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700020000, 0).UTC(),
		UpdatedAt:        time.Unix(1700020000, 0).UTC(),
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	claimed, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700020100, 0).UTC())
	if err != nil {
		t.Fatalf("ClaimQueuedRuns(first) error = %v", err)
	}
	if len(claimed) != 1 {
		t.Fatalf("len(claimed) = %d, want 1", len(claimed))
	}
	if _, err := store.MarkRunFailed(ctx, run.ID, "fault injection", time.Unix(1700020200, 0).UTC()); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}
	if _, err := store.RetryRun(ctx, run.ID, time.Unix(1700020300, 0).UTC()); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if _, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700020400, 0).UTC()); err != nil {
		t.Fatalf("ClaimQueuedRuns(second) error = %v", err)
	}
	if _, err := store.MarkRunSucceeded(ctx, run.ID, time.Unix(1700020500, 0).UTC()); err != nil {
		t.Fatalf("MarkRunSucceeded() error = %v", err)
	}

	events, err := store.ListRunEvents(ctx, run.ID)
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
		evalsvc.RunEventCreated,
		evalsvc.RunEventClaimed,
		evalsvc.RunEventFailed,
		evalsvc.RunEventRetried,
		evalsvc.RunEventClaimed,
		evalsvc.RunEventSucceeded,
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
