package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	evalsvc "opspilot-go/internal/eval"
)

func TestEvalDatasetStoreRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval dataset and lineage tables error = %v", err)
	}

	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-a", "report-task-eval-a", "case-eval-a", "version-a", "trace-a", time.Unix(1700012900, 0).UTC())
	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-b", "report-task-eval-b", "case-eval-b", "version-b", "trace-b", time.Unix(1700012910, 0).UTC())
	evalStore := NewEvalCaseStore(pool)
	for _, item := range []evalsvc.EvalCase{
		{
			ID:             "eval-case-a",
			TenantID:       "tenant-eval",
			SourceCaseID:   "case-eval-a",
			SourceTaskID:   "task-eval-a",
			SourceReportID: "report-task-eval-a",
			TraceID:        "trace-a",
			VersionID:      "version-a",
			Title:          "Eval A",
			Summary:        "First",
			CreatedBy:      "operator-a",
			CreatedAt:      time.Unix(1700013000, 0).UTC(),
		},
		{
			ID:             "eval-case-b",
			TenantID:       "tenant-eval",
			SourceCaseID:   "case-eval-b",
			SourceTaskID:   "task-eval-b",
			SourceReportID: "report-task-eval-b",
			TraceID:        "trace-b",
			VersionID:      "version-b",
			Title:          "Eval B",
			Summary:        "Second",
			CreatedBy:      "operator-b",
			CreatedAt:      time.Unix(1700013010, 0).UTC(),
		},
	} {
		if _, err := evalStore.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	store := NewEvalDatasetStore(pool)
	want := evalsvc.EvalDataset{
		ID:          "eval-dataset-roundtrip",
		TenantID:    "tenant-eval",
		Name:        "Regression draft",
		Description: "Seed from durable eval coverage",
		Status:      evalsvc.DatasetStatusDraft,
		CreatedBy:   "operator-1",
		CreatedAt:   time.Unix(1700013100, 0).UTC(),
		UpdatedAt:   time.Unix(1700013100, 0).UTC(),
		Items: []evalsvc.EvalDatasetItem{
			{EvalCaseID: "eval-case-b"},
			{EvalCaseID: "eval-case-a"},
		},
	}

	created, err := store.CreateDataset(ctx, want)
	if err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}
	if created.Name != want.Name {
		t.Fatalf("Name = %q, want %q", created.Name, want.Name)
	}
	if len(created.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(created.Items))
	}
	if created.Items[0].EvalCaseID != "eval-case-b" || created.Items[1].EvalCaseID != "eval-case-a" {
		t.Fatalf("Items = %#v, want ordered dataset items", created.Items)
	}

	got, err := store.GetDataset(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetDataset() error = %v", err)
	}
	if got.Status != evalsvc.DatasetStatusDraft {
		t.Fatalf("Status = %q, want %q", got.Status, evalsvc.DatasetStatusDraft)
	}
	if len(got.Items) != 2 || got.Items[0].VersionID != "version-b" {
		t.Fatalf("Items = %#v, want durable eval-case metadata", got.Items)
	}
}

func TestEvalDatasetStoreListSupportsFiltersAndPagination(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval dataset and lineage tables error = %v", err)
	}

	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-a", "report-task-eval-a", "case-eval-a", "version-a", "trace-a", time.Unix(1700012900, 0).UTC())
	seedEvalLineageCase(t, ctx, pool, "tenant-eval", "task-eval-b", "report-task-eval-b", "case-eval-b", "version-b", "trace-b", time.Unix(1700012910, 0).UTC())
	evalStore := NewEvalCaseStore(pool)
	for _, item := range []evalsvc.EvalCase{
		{
			ID:             "eval-case-a",
			TenantID:       "tenant-eval",
			SourceCaseID:   "case-eval-a",
			SourceTaskID:   "task-eval-a",
			SourceReportID: "report-task-eval-a",
			TraceID:        "trace-a",
			VersionID:      "version-a",
			Title:          "Eval A",
			Summary:        "First",
			CreatedBy:      "operator-a",
			CreatedAt:      time.Unix(1700013000, 0).UTC(),
		},
		{
			ID:             "eval-case-b",
			TenantID:       "tenant-eval",
			SourceCaseID:   "case-eval-b",
			SourceTaskID:   "task-eval-b",
			SourceReportID: "report-task-eval-b",
			TraceID:        "trace-b",
			VersionID:      "version-b",
			Title:          "Eval B",
			Summary:        "Second",
			CreatedBy:      "operator-b",
			CreatedAt:      time.Unix(1700013010, 0).UTC(),
		},
	} {
		if _, err := evalStore.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	store := NewEvalDatasetStore(pool)
	if _, err := store.CreateDataset(ctx, evalsvc.EvalDataset{
		ID:        "eval-dataset-a",
		TenantID:  "tenant-eval",
		Name:      "Dataset A",
		Status:    evalsvc.DatasetStatusDraft,
		CreatedBy: "operator-a",
		CreatedAt: time.Unix(1700013100, 0).UTC(),
		UpdatedAt: time.Unix(1700013100, 0).UTC(),
		Items:     []evalsvc.EvalDatasetItem{{EvalCaseID: "eval-case-a"}},
	}); err != nil {
		t.Fatalf("CreateDataset(first) error = %v", err)
	}
	second, err := store.CreateDataset(ctx, evalsvc.EvalDataset{
		ID:        "eval-dataset-b",
		TenantID:  "tenant-eval",
		Name:      "Dataset B",
		Status:    evalsvc.DatasetStatusDraft,
		CreatedBy: "operator-b",
		CreatedAt: time.Unix(1700013110, 0).UTC(),
		UpdatedAt: time.Unix(1700013110, 0).UTC(),
		Items:     []evalsvc.EvalDatasetItem{{EvalCaseID: "eval-case-b"}},
	})
	if err != nil {
		t.Fatalf("CreateDataset(second) error = %v", err)
	}

	page, err := store.ListDatasets(ctx, evalsvc.DatasetListFilter{
		TenantID:  "tenant-eval",
		CreatedBy: "operator-b",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListDatasets(filtered) error = %v", err)
	}
	if len(page.Datasets) != 1 || page.Datasets[0].ID != second.ID {
		t.Fatalf("Datasets = %#v, want only %q", page.Datasets, second.ID)
	}
	if page.Datasets[0].ItemCount != 1 {
		t.Fatalf("ItemCount = %d, want 1", page.Datasets[0].ItemCount)
	}

	page, err = store.ListDatasets(ctx, evalsvc.DatasetListFilter{
		TenantID: "tenant-eval",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListDatasets(paginated) error = %v", err)
	}
	if len(page.Datasets) != 1 || page.Datasets[0].ID != second.ID {
		t.Fatalf("first page = %#v, want dataset %q", page.Datasets, second.ID)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}
}

func TestEvalDatasetStoreFiltersCrossTenantMembershipsFromDetailAndCounts(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_dataset_items, eval_datasets, eval_cases, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval dataset and lineage tables error = %v", err)
	}

	seedEvalLineageCase(t, ctx, pool, "tenant-a", "task-eval-a", "report-task-eval-a", "case-eval-a", "version-a", "trace-a", time.Unix(1700012900, 0).UTC())
	seedEvalLineageCase(t, ctx, pool, "tenant-b", "task-eval-b", "report-task-eval-b", "case-eval-b", "version-b", "trace-b", time.Unix(1700012910, 0).UTC())
	evalStore := NewEvalCaseStore(pool)
	for _, item := range []evalsvc.EvalCase{
		{
			ID:             "eval-case-a",
			TenantID:       "tenant-a",
			SourceCaseID:   "case-eval-a",
			SourceTaskID:   "task-eval-a",
			SourceReportID: "report-task-eval-a",
			TraceID:        "trace-a",
			VersionID:      "version-a",
			Title:          "Eval A",
			Summary:        "First",
			CreatedBy:      "operator-a",
			CreatedAt:      time.Unix(1700013000, 0).UTC(),
		},
		{
			ID:             "eval-case-b",
			TenantID:       "tenant-b",
			SourceCaseID:   "case-eval-b",
			SourceTaskID:   "task-eval-b",
			SourceReportID: "report-task-eval-b",
			TraceID:        "trace-b",
			VersionID:      "version-b",
			Title:          "Eval B",
			Summary:        "Second",
			CreatedBy:      "operator-b",
			CreatedAt:      time.Unix(1700013010, 0).UTC(),
		},
	} {
		if _, err := evalStore.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	store := NewEvalDatasetStore(pool)
	if _, err := store.CreateDataset(ctx, evalsvc.EvalDataset{
		ID:        "eval-dataset-a",
		TenantID:  "tenant-a",
		Name:      "Dataset A",
		Status:    evalsvc.DatasetStatusDraft,
		CreatedBy: "operator-a",
		CreatedAt: time.Unix(1700013100, 0).UTC(),
		UpdatedAt: time.Unix(1700013100, 0).UTC(),
		Items:     []evalsvc.EvalDatasetItem{{EvalCaseID: "eval-case-a"}},
	}); err != nil {
		t.Fatalf("CreateDataset() error = %v", err)
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO eval_dataset_items (dataset_id, eval_case_id, position, created_at)
VALUES ('eval-dataset-a', 'eval-case-b', 1, $1)
ON CONFLICT DO NOTHING
`, time.Unix(1700013110, 0).UTC()); err != nil {
		t.Fatalf("inject cross-tenant membership error = %v", err)
	}

	got, err := store.GetDataset(ctx, "eval-dataset-a")
	if err != nil {
		t.Fatalf("GetDataset() error = %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].EvalCaseID != "eval-case-a" {
		t.Fatalf("Items = %#v, want only tenant-a membership", got.Items)
	}

	page, err := store.ListDatasets(ctx, evalsvc.DatasetListFilter{
		TenantID: "tenant-a",
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("ListDatasets() error = %v", err)
	}
	if len(page.Datasets) != 1 || page.Datasets[0].ItemCount != 1 {
		t.Fatalf("Datasets = %#v, want one tenant-safe item count", page.Datasets)
	}
}
