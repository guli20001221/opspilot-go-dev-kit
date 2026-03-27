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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, `
INSERT INTO cases (
    id, tenant_id, title, status, reason, source_task_id, source_report_id, created_by, created_at, updated_at
) VALUES (
    $1, $2, $3, 'open', 'workflow_required', '', '', 'operator', $4, $4
)`,
		"case-run-roundtrip",
		want.TenantID,
		"Roundtrip case",
		time.Unix(1700019800, 0).UTC(),
	); err != nil {
		t.Fatalf("seed cases error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_cases (
    id, tenant_id, source_case_id, source_task_id, source_report_id, trace_id, version_id, title, summary, operator_note, created_by, created_at
) VALUES (
    $1, $2, $3, '', '', 'trace-roundtrip', 'version-roundtrip', $4, '', '', 'operator', $5
)`,
		"eval-case-roundtrip",
		want.TenantID,
		"case-run-roundtrip",
		"Roundtrip eval case",
		time.Unix(1700019850, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_cases error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_dataset_items (dataset_id, eval_case_id, position, created_at)
VALUES ($1, $2, 0, $3)`,
		want.DatasetID,
		"eval-case-roundtrip",
		time.Unix(1700019900, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_dataset_items error = %v", err)
	}

	created, err := store.CreateRun(ctx, want, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-roundtrip",
		Title:        "Roundtrip eval case",
		SourceCaseID: "case-run-roundtrip",
		TraceID:      "trace-roundtrip",
		VersionID:    "version-roundtrip",
	})
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

	detail, err := store.GetRunDetail(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.Items) != 1 {
		t.Fatalf("len(detail.Items) = %d, want 1", len(detail.Items))
	}
	if detail.Items[0].EvalCaseID != "eval-case-roundtrip" {
		t.Fatalf("detail.Items[0].EvalCaseID = %q, want %q", detail.Items[0].EvalCaseID, "eval-case-roundtrip")
	}
	if detail.Items[0].TraceID != "trace-roundtrip" || detail.Items[0].VersionID != "version-roundtrip" {
		t.Fatalf("detail.Items[0] = %#v, want trace/version lineage", detail.Items[0])
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
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
	if _, err := pool.Exec(ctx, `
INSERT INTO cases (
    id, tenant_id, title, status, reason, source_task_id, source_report_id, created_by, created_at, updated_at
) VALUES (
    $1, $2, $3, 'open', 'workflow_required', '', '', 'operator', $4, $4
)`,
		"case-events",
		"tenant-run",
		"Events case",
		time.Unix(1700019890, 0).UTC(),
	); err != nil {
		t.Fatalf("seed cases error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_cases (
    id, tenant_id, source_case_id, source_task_id, source_report_id, trace_id, version_id, title, summary, operator_note, created_by, created_at
) VALUES (
    $1, $2, $3, '', '', 'trace-events', 'version-events', $4, '', '', 'operator', $5
)`,
		"eval-case-events",
		"tenant-run",
		"case-events",
		"Events eval case",
		time.Unix(1700019895, 0).UTC(),
	); err != nil {
		t.Fatalf("seed eval_cases error = %v", err)
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
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-events",
		Title:        "Events eval case",
		SourceCaseID: "case-events",
		TraceID:      "trace-events",
		VersionID:    "version-events",
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
	failedResults := []evalsvc.EvalRunItemResult{{
		EvalCaseID: "eval-case-events",
		Status:     evalsvc.RunItemResultFailed,
		Detail:     "fault injection",
		UpdatedAt:  time.Unix(1700020200, 0).UTC(),
	}}
	if _, err := store.MarkRunFailed(ctx, run.ID, "fault injection", time.Unix(1700020200, 0).UTC(), failedResults); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}
	if _, err := store.RetryRun(ctx, run.ID, time.Unix(1700020300, 0).UTC()); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	if _, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700020400, 0).UTC()); err != nil {
		t.Fatalf("ClaimQueuedRuns(second) error = %v", err)
	}
	succeededResults := []evalsvc.EvalRunItemResult{{
		EvalCaseID: "eval-case-events",
		Status:     evalsvc.RunItemResultSucceeded,
		Detail:     "placeholder eval passed",
		UpdatedAt:  time.Unix(1700020500, 0).UTC(),
	}}
	if _, err := store.MarkRunSucceeded(ctx, run.ID, time.Unix(1700020500, 0).UTC(), succeededResults); err != nil {
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

func TestEvalRunStoreRunDetailIncludesAndClearsItemResults(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	publishedAt := time.Unix(1700021000, 0).UTC()
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator', $5, $5, 'operator', $5
)`,
		"eval-dataset-results",
		"tenant-run",
		"Dataset Results",
		evalsvc.DatasetStatusPublished,
		publishedAt,
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cases (
    id, tenant_id, title, status, reason, source_task_id, source_report_id, created_by, created_at, updated_at
) VALUES
    ('case-results-a', 'tenant-run', 'Results case A', 'open', 'workflow_required', '', '', 'operator', $1, $1),
    ('case-results-b', 'tenant-run', 'Results case B', 'open', 'workflow_required', '', '', 'operator', $1, $1)
`, publishedAt); err != nil {
		t.Fatalf("seed cases error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_cases (
    id, tenant_id, source_case_id, source_task_id, source_report_id, trace_id, version_id, title, summary, operator_note, created_by, created_at
) VALUES
    ('eval-case-results-a', 'tenant-run', 'case-results-a', '', '', 'trace-results-a', 'version-results-a', 'Results eval case A', '', '', 'operator', $1),
    ('eval-case-results-b', 'tenant-run', 'case-results-b', '', '', 'trace-results-b', 'version-results-b', 'Results eval case B', '', '', 'operator', $1)
`, publishedAt); err != nil {
		t.Fatalf("seed eval_cases error = %v", err)
	}

	store := NewEvalRunStore(pool)
	run, err := store.CreateRun(ctx, evalsvc.EvalRun{
		ID:               "eval-run-results",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-results",
		DatasetName:      "Dataset Results",
		DatasetItemCount: 2,
		Status:           evalsvc.RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700021010, 0).UTC(),
		UpdatedAt:        time.Unix(1700021010, 0).UTC(),
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-results-a",
		Title:        "Results eval case A",
		SourceCaseID: "case-results-a",
		TraceID:      "trace-results-a",
		VersionID:    "version-results-a",
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-results-b",
		Title:        "Results eval case B",
		SourceCaseID: "case-results-b",
		TraceID:      "trace-results-b",
		VersionID:    "version-results-b",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if _, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700021020, 0).UTC()); err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	results := []evalsvc.EvalRunItemResult{
		{
			EvalCaseID: "eval-case-results-b",
			Status:     evalsvc.RunItemResultSucceeded,
			Detail:     "placeholder eval passed",
			UpdatedAt:  time.Unix(1700021030, 0).UTC(),
		},
		{
			EvalCaseID: "eval-case-results-a",
			Status:     evalsvc.RunItemResultSucceeded,
			Detail:     "placeholder eval passed",
			UpdatedAt:  time.Unix(1700021030, 0).UTC(),
		},
	}
	if _, err := store.MarkRunSucceeded(ctx, run.ID, time.Unix(1700021030, 0).UTC(), results); err != nil {
		t.Fatalf("MarkRunSucceeded() error = %v", err)
	}

	detail, err := store.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() after success error = %v", err)
	}
	if len(detail.ItemResults) != 2 {
		t.Fatalf("len(detail.ItemResults) = %d, want 2", len(detail.ItemResults))
	}
	if detail.ItemResults[0].EvalCaseID != "eval-case-results-a" || detail.ItemResults[1].EvalCaseID != "eval-case-results-b" {
		t.Fatalf("detail.ItemResults = %#v, want ordered run item results", detail.ItemResults)
	}

	if _, err := store.RetryRun(ctx, run.ID, time.Unix(1700021040, 0).UTC()); err != nil {
		t.Fatalf("RetryRun() error = %v", err)
	}
	detail, err = store.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() after retry error = %v", err)
	}
	if len(detail.ItemResults) != 0 {
		t.Fatalf("len(detail.ItemResults) after retry = %d, want 0", len(detail.ItemResults))
	}
}

func TestEvalRunStoreListRunsIncludesResultSummary(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	publishedAt := time.Unix(1700021100, 0).UTC()
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator', $5, $5, 'operator', $5
)`,
		"eval-dataset-summary",
		"tenant-run",
		"Dataset Summary",
		evalsvc.DatasetStatusPublished,
		publishedAt,
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cases (
    id, tenant_id, title, status, reason, source_task_id, source_report_id, created_by, created_at, updated_at
) VALUES
    ('case-summary-a', 'tenant-run', 'Summary case A', 'open', 'workflow_required', '', '', 'operator', $1, $1),
    ('case-summary-b', 'tenant-run', 'Summary case B', 'open', 'workflow_required', '', '', 'operator', $1, $1)
`, publishedAt); err != nil {
		t.Fatalf("seed cases error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_cases (
    id, tenant_id, source_case_id, source_task_id, source_report_id, trace_id, version_id, title, summary, operator_note, created_by, created_at
) VALUES
    ('eval-case-summary-a', 'tenant-run', 'case-summary-a', '', '', 'trace-summary-a', 'version-summary-a', 'Summary eval case A', '', '', 'operator', $1),
    ('eval-case-summary-b', 'tenant-run', 'case-summary-b', '', '', 'trace-summary-b', 'version-summary-b', 'Summary eval case B', '', '', 'operator', $1)
`, publishedAt); err != nil {
		t.Fatalf("seed eval_cases error = %v", err)
	}

	store := NewEvalRunStore(pool)
	run, err := store.CreateRun(ctx, evalsvc.EvalRun{
		ID:               "eval-run-summary",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-summary",
		DatasetName:      "Dataset Summary",
		DatasetItemCount: 2,
		Status:           evalsvc.RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700021110, 0).UTC(),
		UpdatedAt:        time.Unix(1700021110, 0).UTC(),
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-summary-a",
		Title:        "Summary eval case A",
		SourceCaseID: "case-summary-a",
		TraceID:      "trace-summary-a",
		VersionID:    "version-summary-a",
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-summary-b",
		Title:        "Summary eval case B",
		SourceCaseID: "case-summary-b",
		TraceID:      "trace-summary-b",
		VersionID:    "version-summary-b",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if _, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700021120, 0).UTC()); err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if _, err := store.MarkRunFailed(ctx, run.ID, "fault injection", time.Unix(1700021130, 0).UTC(), []evalsvc.EvalRunItemResult{
		{EvalCaseID: "eval-case-summary-a", Status: evalsvc.RunItemResultFailed, Detail: "fault injection", UpdatedAt: time.Unix(1700021130, 0).UTC()},
		{EvalCaseID: "eval-case-summary-b", Status: evalsvc.RunItemResultFailed, Detail: "fault injection", UpdatedAt: time.Unix(1700021130, 0).UTC()},
	}); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}

	page, err := store.ListRuns(ctx, evalsvc.RunListFilter{
		TenantID: "tenant-run",
		Status:   evalsvc.RunStatusFailed,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(page.Runs) != 1 {
		t.Fatalf("len(page.Runs) = %d, want 1", len(page.Runs))
	}
	if page.Runs[0].ResultSummary == nil {
		t.Fatal("ResultSummary = nil, want counts")
	}
	if page.Runs[0].ResultSummary.TotalItems != 2 {
		t.Fatalf("TotalItems = %d, want 2", page.Runs[0].ResultSummary.TotalItems)
	}
	if page.Runs[0].ResultSummary.RecordedResults != 2 || page.Runs[0].ResultSummary.MissingResults != 0 {
		t.Fatalf("ResultSummary = %#v, want fully recorded terminal results", page.Runs[0].ResultSummary)
	}
	if page.Runs[0].ResultSummary.FailedItems != 2 || page.Runs[0].ResultSummary.SucceededItems != 0 {
		t.Fatalf("ResultSummary = %#v, want two failures and zero successes", page.Runs[0].ResultSummary)
	}
}

func TestEvalRunStoreListRunsIncludesMissingResultCounts(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_run_item_results, eval_run_events, eval_runs, eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval run and lineage tables error = %v", err)
	}

	publishedAt := time.Unix(1700022100, 0).UTC()
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_datasets (
    id, tenant_id, name, description, status, created_by, created_at, updated_at, published_by, published_at
) VALUES (
    $1, $2, $3, '', $4, 'operator', $5, $5, 'operator', $5
)`,
		"eval-dataset-summary-missing",
		"tenant-run",
		"Dataset Summary Missing",
		evalsvc.DatasetStatusPublished,
		publishedAt,
	); err != nil {
		t.Fatalf("seed eval_datasets error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO cases (
    id, tenant_id, title, status, reason, source_task_id, source_report_id, created_by, created_at, updated_at
) VALUES
    ('case-summary-missing-a', 'tenant-run', 'Summary missing case A', 'open', 'workflow_required', '', '', 'operator', $1, $1),
    ('case-summary-missing-b', 'tenant-run', 'Summary missing case B', 'open', 'workflow_required', '', '', 'operator', $1, $1)
`, publishedAt); err != nil {
		t.Fatalf("seed cases error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_cases (
    id, tenant_id, source_case_id, source_task_id, source_report_id, trace_id, version_id, title, summary, operator_note, created_by, created_at
) VALUES
    ('eval-case-summary-missing-a', 'tenant-run', 'case-summary-missing-a', '', '', 'trace-summary-missing-a', 'version-summary-missing-a', 'Summary missing eval case A', '', '', 'operator', $1),
    ('eval-case-summary-missing-b', 'tenant-run', 'case-summary-missing-b', '', '', 'trace-summary-missing-b', 'version-summary-missing-b', 'Summary missing eval case B', '', '', 'operator', $1)
`, publishedAt); err != nil {
		t.Fatalf("seed eval_cases error = %v", err)
	}

	store := NewEvalRunStore(pool)
	run, err := store.CreateRun(ctx, evalsvc.EvalRun{
		ID:               "eval-run-summary-missing",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-summary-missing",
		DatasetName:      "Dataset Summary Missing",
		DatasetItemCount: 2,
		Status:           evalsvc.RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700022110, 0).UTC(),
		UpdatedAt:        time.Unix(1700022110, 0).UTC(),
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-summary-missing-a",
		Title:        "Summary missing eval case A",
		SourceCaseID: "case-summary-missing-a",
		TraceID:      "trace-summary-missing-a",
		VersionID:    "version-summary-missing-a",
	}, evalsvc.EvalRunItem{
		EvalCaseID:   "eval-case-summary-missing-b",
		Title:        "Summary missing eval case B",
		SourceCaseID: "case-summary-missing-b",
		TraceID:      "trace-summary-missing-b",
		VersionID:    "version-summary-missing-b",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}
	if _, err := store.ClaimQueuedRuns(ctx, 1, time.Unix(1700022120, 0).UTC()); err != nil {
		t.Fatalf("ClaimQueuedRuns() error = %v", err)
	}
	if _, err := store.MarkRunFailed(ctx, run.ID, "partial fault injection", time.Unix(1700022130, 0).UTC(), []evalsvc.EvalRunItemResult{
		{EvalCaseID: "eval-case-summary-missing-a", Status: evalsvc.RunItemResultFailed, Detail: "partial fault injection", UpdatedAt: time.Unix(1700022130, 0).UTC()},
	}); err != nil {
		t.Fatalf("MarkRunFailed() error = %v", err)
	}

	page, err := store.ListRuns(ctx, evalsvc.RunListFilter{
		TenantID: "tenant-run",
		Status:   evalsvc.RunStatusFailed,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(page.Runs) != 1 {
		t.Fatalf("len(page.Runs) = %d, want 1", len(page.Runs))
	}
	if page.Runs[0].ResultSummary == nil {
		t.Fatal("ResultSummary = nil, want counts")
	}
	if page.Runs[0].ResultSummary.TotalItems != 2 || page.Runs[0].ResultSummary.RecordedResults != 1 {
		t.Fatalf("ResultSummary = %#v, want two total items and one recorded result", page.Runs[0].ResultSummary)
	}
	if page.Runs[0].ResultSummary.MissingResults != 1 {
		t.Fatalf("MissingResults = %d, want 1", page.Runs[0].ResultSummary.MissingResults)
	}
}
