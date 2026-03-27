package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	evalsvc "opspilot-go/internal/eval"
)

func TestEvalReportStoreRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_reports, eval_run_item_results, eval_run_items, eval_run_events, eval_runs, eval_datasets, eval_dataset_items, eval_cases, version_refs, versions, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval report tables error = %v", err)
	}

	store := NewEvalReportStore(pool)
	readyAt := time.Unix(1700042500, 0).UTC()
	want := evalsvc.EvalReport{
		ID:              "eval-report-eval-run-1",
		TenantID:        "tenant-eval",
		RunID:           "eval-run-1",
		DatasetID:       "eval-dataset-1",
		DatasetName:     "Dataset One",
		RunStatus:       evalsvc.RunStatusSucceeded,
		Status:          evalsvc.EvalReportStatusReady,
		Summary:         "1 failed / 1 passed / 2 total (avg score 0.625)",
		TotalItems:      2,
		RecordedResults: 2,
		PassedItems:     1,
		FailedItems:     1,
		MissingResults:  0,
		AverageScore:    0.625,
		JudgeVersion:    "http_json/judge-demo/placeholder-eval-judge-v1",
		MetadataJSON:    json.RawMessage(`{"judge_prompt_paths":["eval/prompts/placeholder-eval-judge-v1.md"]}`),
		BadCases: []evalsvc.EvalReportBadCase{{
			EvalCaseID:     "eval-case-fail",
			Title:          "Fail item",
			SourceCaseID:   "case-fail",
			SourceTaskID:   "task-fail",
			SourceReportID: "report-fail",
			TraceID:        "trace-fail",
			VersionID:      "version-fail",
			Verdict:        evalsvc.RunItemVerdictFail,
			Detail:         "fail rationale",
			Score:          0.25,
		}},
		CreatedAt: time.Unix(1700042400, 0).UTC(),
		UpdatedAt: readyAt,
		ReadyAt:   readyAt,
	}

	if _, err := store.SaveEvalReport(ctx, want); err != nil {
		t.Fatalf("SaveEvalReport() error = %v", err)
	}

	got, err := store.GetEvalReport(ctx, want.ID)
	if err != nil {
		t.Fatalf("GetEvalReport() error = %v", err)
	}
	if got.RunID != want.RunID {
		t.Fatalf("RunID = %q, want %q", got.RunID, want.RunID)
	}
	if got.AverageScore != want.AverageScore {
		t.Fatalf("AverageScore = %v, want %v", got.AverageScore, want.AverageScore)
	}
	if got.JudgeVersion != want.JudgeVersion {
		t.Fatalf("JudgeVersion = %q, want %q", got.JudgeVersion, want.JudgeVersion)
	}
	if len(got.BadCases) != 1 || got.BadCases[0].EvalCaseID != "eval-case-fail" {
		t.Fatalf("BadCases = %#v, want persisted failed case reference", got.BadCases)
	}
	if !jsonEqual(got.MetadataJSON, want.MetadataJSON) {
		t.Fatalf("MetadataJSON = %s, want %s", string(got.MetadataJSON), string(want.MetadataJSON))
	}
}

func TestEvalReportStoreListSupportsFiltersAndPagination(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_reports, eval_run_item_results, eval_run_items, eval_run_events, eval_runs, eval_datasets, eval_dataset_items, eval_cases, version_refs, versions, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval report tables error = %v", err)
	}

	store := NewEvalReportStore(pool)
	firstReady := time.Unix(1700042600, 0).UTC()
	secondReady := time.Unix(1700042700, 0).UTC()
	fixtures := []evalsvc.EvalReport{
		{
			ID:              "eval-report-eval-run-1",
			TenantID:        "tenant-eval",
			RunID:           "eval-run-1",
			DatasetID:       "eval-dataset-1",
			DatasetName:     "Dataset One",
			RunStatus:       evalsvc.RunStatusSucceeded,
			Status:          evalsvc.EvalReportStatusReady,
			Summary:         "0 failed / 1 passed / 1 total (avg score 1.000)",
			TotalItems:      1,
			RecordedResults: 1,
			PassedItems:     1,
			CreatedAt:       firstReady.Add(-time.Minute),
			UpdatedAt:       firstReady,
			ReadyAt:         firstReady,
		},
		{
			ID:              "eval-report-eval-run-2",
			TenantID:        "tenant-eval",
			RunID:           "eval-run-2",
			DatasetID:       "eval-dataset-2",
			DatasetName:     "Dataset Two",
			RunStatus:       evalsvc.RunStatusFailed,
			Status:          evalsvc.EvalReportStatusReady,
			Summary:         "1 failed / 0 passed / 1 total (avg score 0.000)",
			TotalItems:      1,
			RecordedResults: 1,
			FailedItems:     1,
			CreatedAt:       secondReady.Add(-time.Minute),
			UpdatedAt:       secondReady,
			ReadyAt:         secondReady,
		},
	}
	for _, item := range fixtures {
		if _, err := store.SaveEvalReport(ctx, item); err != nil {
			t.Fatalf("SaveEvalReport(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.ListEvalReports(ctx, evalsvc.EvalReportListFilter{
		TenantID: "tenant-eval",
		Status:   evalsvc.EvalReportStatusReady,
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListEvalReports() error = %v", err)
	}
	if len(page.Reports) != 1 || page.Reports[0].ID != "eval-report-eval-run-2" {
		t.Fatalf("page.Reports = %#v, want newest eval report first", page.Reports)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}

	filtered, err := store.ListEvalReports(ctx, evalsvc.EvalReportListFilter{
		TenantID:  "tenant-eval",
		DatasetID: "eval-dataset-1",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("ListEvalReports(filtered) error = %v", err)
	}
	if len(filtered.Reports) != 1 || filtered.Reports[0].ID != "eval-report-eval-run-1" {
		t.Fatalf("filtered.Reports = %#v, want dataset-scoped report", filtered.Reports)
	}
}
