package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
)

func TestCaseStoreRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	want := casesvc.Case{
		ID:                 "case-roundtrip-1",
		TenantID:           "tenant-1",
		Status:             casesvc.StatusOpen,
		Title:              "Review generated report",
		Summary:            "Operator wants a durable follow-up case.",
		SourceTaskID:       "task-source-1",
		SourceReportID:     "report-source-1",
		SourceEvalReportID: "eval-report-roundtrip-1",
		CompareOrigin: casesvc.CompareOrigin{
			LeftEvalReportID:  "eval-report-roundtrip-1",
			RightEvalReportID: "eval-report-roundtrip-2",
			SelectedSide:      "left",
		},
		CreatedBy: "operator-1",
	}

	if _, err := pool.Exec(ctx, `
INSERT INTO workflow_tasks (
    id, request_id, tenant_id, session_id, task_type, tool_name, tool_arguments,
    status, reason, error_reason, audit_ref, requires_approval, created_at, updated_at
) VALUES (
    'task-source-1', 'req-source-1', 'tenant-1', 'session-1', 'report_generation', '', '{}'::jsonb,
    'succeeded', 'workflow_required', '', '', false, NOW(), NOW()
)`); err != nil {
		t.Fatalf("insert workflow task error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO reports (
    id, tenant_id, source_task_id, report_type, status, title, summary, content_uri, metadata_json, created_by, created_at, ready_at
) VALUES (
    'report-source-1', 'tenant-1', 'task-source-1', 'workflow_summary', 'ready', 'Title', 'Summary', '', '{}'::jsonb, 'worker', NOW(), NOW()
)`); err != nil {
		t.Fatalf("insert report error = %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO eval_reports (
    id, tenant_id, run_id, dataset_id, dataset_name, run_status, status, summary,
    total_items, recorded_results, passed_items, failed_items, missing_results,
    average_score, judge_version, metadata_json, created_at, updated_at, ready_at
) VALUES
(
    'eval-report-roundtrip-1', 'tenant-1', 'eval-run-roundtrip-1', 'dataset-roundtrip-1', 'Dataset Roundtrip',
    'failed', 'ready', 'first compare side',
    1, 1, 0, 1, 0, 0, 'judge-a', '{}'::jsonb, NOW(), NOW(), NOW()
),
(
    'eval-report-roundtrip-2', 'tenant-1', 'eval-run-roundtrip-2', 'dataset-roundtrip-1', 'Dataset Roundtrip',
    'succeeded', 'ready', 'second compare side',
    1, 1, 1, 0, 0, 1, 'judge-a', '{}'::jsonb, NOW(), NOW(), NOW()
)`); err != nil {
		t.Fatalf("insert eval reports error = %v", err)
	}

	saved, err := store.Save(ctx, want)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if saved.ID != want.ID {
		t.Fatalf("Save().ID = %q, want %q", saved.ID, want.ID)
	}

	got, err := store.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Title != want.Title {
		t.Fatalf("Get().Title = %q, want %q", got.Title, want.Title)
	}
	if got.SourceTaskID != want.SourceTaskID {
		t.Fatalf("Get().SourceTaskID = %q, want %q", got.SourceTaskID, want.SourceTaskID)
	}
	if got.SourceReportID != want.SourceReportID {
		t.Fatalf("Get().SourceReportID = %q, want %q", got.SourceReportID, want.SourceReportID)
	}
	if got.SourceEvalReportID != want.SourceEvalReportID {
		t.Fatalf("Get().SourceEvalReportID = %q, want %q", got.SourceEvalReportID, want.SourceEvalReportID)
	}
	if got.CompareOrigin.LeftEvalReportID != want.CompareOrigin.LeftEvalReportID {
		t.Fatalf("Get().CompareOrigin.LeftEvalReportID = %q, want %q", got.CompareOrigin.LeftEvalReportID, want.CompareOrigin.LeftEvalReportID)
	}
	if got.CompareOrigin.RightEvalReportID != want.CompareOrigin.RightEvalReportID {
		t.Fatalf("Get().CompareOrigin.RightEvalReportID = %q, want %q", got.CompareOrigin.RightEvalReportID, want.CompareOrigin.RightEvalReportID)
	}
	if got.CompareOrigin.SelectedSide != want.CompareOrigin.SelectedSide {
		t.Fatalf("Get().CompareOrigin.SelectedSide = %q, want %q", got.CompareOrigin.SelectedSide, want.CompareOrigin.SelectedSide)
	}
}

func TestCaseStoreListSupportsFiltersAndOffset(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002000, 0).UTC()
	for _, item := range []casesvc.Case{
		{
			ID:        "case-list-1",
			TenantID:  "tenant-1",
			Status:    casesvc.StatusOpen,
			Title:     "First case",
			CreatedBy: "operator-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			ID:        "case-list-2",
			TenantID:  "tenant-1",
			Status:    casesvc.StatusOpen,
			Title:     "Second case",
			CreatedBy: "operator-1",
			CreatedAt: now.Add(time.Second),
			UpdatedAt: now.Add(time.Second),
		},
		{
			ID:        "case-list-3",
			TenantID:  "tenant-2",
			Status:    casesvc.StatusOpen,
			Title:     "Third case",
			CreatedBy: "operator-1",
			CreatedAt: now.Add(2 * time.Second),
			UpdatedAt: now.Add(2 * time.Second),
		},
	} {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.List(ctx, casesvc.ListFilter{
		TenantID: "tenant-1",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(List().Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].ID != "case-list-2" {
		t.Fatalf("List().Cases[0].ID = %q, want %q", page.Cases[0].ID, "case-list-2")
	}
	if !page.HasMore {
		t.Fatal("List().HasMore = false, want true")
	}

	nextPage, err := store.List(ctx, casesvc.ListFilter{
		TenantID: "tenant-1",
		Limit:    1,
		Offset:   page.NextOffset,
	})
	if err != nil {
		t.Fatalf("List(nextPage) error = %v", err)
	}
	if len(nextPage.Cases) != 1 {
		t.Fatalf("len(nextPage.Cases) = %d, want %d", len(nextPage.Cases), 1)
	}
	if nextPage.Cases[0].ID != "case-list-1" {
		t.Fatalf("nextPage.Cases[0].ID = %q, want %q", nextPage.Cases[0].ID, "case-list-1")
	}
}

func TestCaseStoreListSupportsAssignedToFilter(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002050, 0).UTC()
	for _, item := range []casesvc.Case{
		{
			ID:         "case-assignee-1",
			TenantID:   "tenant-1",
			Status:     casesvc.StatusOpen,
			Title:      "Mine",
			CreatedBy:  "operator-1",
			AssignedTo: "cases-operator",
			AssignedAt: now.Add(time.Second),
			CreatedAt:  now,
			UpdatedAt:  now.Add(time.Second),
		},
		{
			ID:         "case-assignee-2",
			TenantID:   "tenant-1",
			Status:     casesvc.StatusOpen,
			Title:      "Other",
			CreatedBy:  "operator-1",
			AssignedTo: "other-operator",
			AssignedAt: now.Add(2 * time.Second),
			CreatedAt:  now.Add(2 * time.Second),
			UpdatedAt:  now.Add(2 * time.Second),
		},
	} {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.List(ctx, casesvc.ListFilter{
		TenantID:   "tenant-1",
		Status:     casesvc.StatusOpen,
		AssignedTo: "cases-operator",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(List().Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].ID != "case-assignee-1" {
		t.Fatalf("List().Cases[0].ID = %q, want %q", page.Cases[0].ID, "case-assignee-1")
	}
}

func TestCaseStoreListSupportsUnassignedOnlyFilter(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002060, 0).UTC()
	unassigned, err := store.Save(ctx, casesvc.Case{
		ID:        "case-unassigned-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusOpen,
		Title:     "Unassigned",
		CreatedBy: "operator-1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Save(unassigned) error = %v", err)
	}
	if _, err := store.Save(ctx, casesvc.Case{
		ID:         "case-assigned-1",
		TenantID:   "tenant-1",
		Status:     casesvc.StatusOpen,
		Title:      "Assigned",
		CreatedBy:  "operator-1",
		AssignedTo: "cases-operator",
		AssignedAt: now.Add(time.Second),
		CreatedAt:  now.Add(time.Second),
		UpdatedAt:  now.Add(time.Second),
	}); err != nil {
		t.Fatalf("Save(assigned) error = %v", err)
	}

	page, err := store.List(ctx, casesvc.ListFilter{
		TenantID:       "tenant-1",
		Status:         casesvc.StatusOpen,
		UnassignedOnly: true,
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(List().Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].ID != unassigned.ID {
		t.Fatalf("List().Cases[0].ID = %q, want %q", page.Cases[0].ID, unassigned.ID)
	}
}

func TestCaseStoreListSupportsEvalReportFilters(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE eval_reports, case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval_reports and lineage tables error = %v", err)
	}

	reportStore := NewEvalReportStore(pool)
	now := time.Unix(1700002070, 0).UTC()
	for _, item := range []evalsvc.EvalReport{
		{
			ID:              "eval-report-filter-1",
			TenantID:        "tenant-1",
			RunID:           "eval-run-filter-1",
			DatasetID:       "dataset-filter-1",
			DatasetName:     "Dataset One",
			RunStatus:       evalsvc.RunStatusFailed,
			Status:          evalsvc.EvalReportStatusReady,
			Summary:         "first eval report",
			TotalItems:      1,
			RecordedResults: 1,
			PassedItems:     0,
			FailedItems:     1,
			MissingResults:  0,
			AverageScore:    0,
			JudgeVersion:    "judge-a",
			MetadataJSON:    []byte(`{}`),
			CreatedAt:       now,
			UpdatedAt:       now,
			ReadyAt:         now,
		},
		{
			ID:              "eval-report-filter-2",
			TenantID:        "tenant-1",
			RunID:           "eval-run-filter-2",
			DatasetID:       "dataset-filter-1",
			DatasetName:     "Dataset One",
			RunStatus:       evalsvc.RunStatusFailed,
			Status:          evalsvc.EvalReportStatusReady,
			Summary:         "second eval report",
			TotalItems:      1,
			RecordedResults: 1,
			PassedItems:     0,
			FailedItems:     1,
			MissingResults:  0,
			AverageScore:    0,
			JudgeVersion:    "judge-a",
			MetadataJSON:    []byte(`{}`),
			CreatedAt:       now.Add(time.Second),
			UpdatedAt:       now.Add(time.Second),
			ReadyAt:         now.Add(time.Second),
		},
	} {
		if _, err := reportStore.SaveEvalReport(ctx, item); err != nil {
			t.Fatalf("SaveEvalReport(%s) error = %v", item.ID, err)
		}
	}

	store := NewCaseStore(pool)
	for _, item := range []casesvc.Case{
		{
			ID:                 "case-eval-filter-1",
			TenantID:           "tenant-1",
			Status:             casesvc.StatusOpen,
			Title:              "Eval-backed one",
			SourceEvalReportID: "eval-report-filter-1",
			CreatedBy:          "operator-1",
			CreatedAt:          now,
			UpdatedAt:          now,
		},
		{
			ID:                 "case-eval-filter-2",
			TenantID:           "tenant-1",
			Status:             casesvc.StatusOpen,
			Title:              "Eval-backed two",
			SourceEvalReportID: "eval-report-filter-2",
			CreatedBy:          "operator-1",
			CreatedAt:          now.Add(time.Second),
			UpdatedAt:          now.Add(time.Second),
		},
		{
			ID:        "case-non-eval-filter-1",
			TenantID:  "tenant-1",
			Status:    casesvc.StatusOpen,
			Title:     "No eval linkage",
			CreatedBy: "operator-1",
			CreatedAt: now.Add(2 * time.Second),
			UpdatedAt: now.Add(2 * time.Second),
		},
	} {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	exactPage, err := store.List(ctx, casesvc.ListFilter{
		TenantID:           "tenant-1",
		SourceEvalReportID: "eval-report-filter-1",
		Limit:              10,
	})
	if err != nil {
		t.Fatalf("List(exactPage) error = %v", err)
	}
	if len(exactPage.Cases) != 1 {
		t.Fatalf("len(exactPage.Cases) = %d, want %d", len(exactPage.Cases), 1)
	}
	if exactPage.Cases[0].ID != "case-eval-filter-1" {
		t.Fatalf("exactPage.Cases[0].ID = %q, want %q", exactPage.Cases[0].ID, "case-eval-filter-1")
	}

	evalPage, err := store.List(ctx, casesvc.ListFilter{
		TenantID:       "tenant-1",
		EvalBackedOnly: true,
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("List(evalPage) error = %v", err)
	}
	if len(evalPage.Cases) != 2 {
		t.Fatalf("len(evalPage.Cases) = %d, want %d", len(evalPage.Cases), 2)
	}
	if evalPage.Cases[0].ID != "case-eval-filter-2" {
		t.Fatalf("evalPage.Cases[0].ID = %q, want %q", evalPage.Cases[0].ID, "case-eval-filter-2")
	}
	if evalPage.Cases[1].ID != "case-eval-filter-1" {
		t.Fatalf("evalPage.Cases[1].ID = %q, want %q", evalPage.Cases[1].ID, "case-eval-filter-1")
	}

	comparePage, err := store.List(ctx, casesvc.ListFilter{
		TenantID:          "tenant-1",
		CompareOriginOnly: true,
		Limit:             10,
	})
	if err != nil {
		t.Fatalf("List(comparePage) error = %v", err)
	}
	if len(comparePage.Cases) != 1 {
		t.Fatalf("len(comparePage.Cases) = %d, want %d", len(comparePage.Cases), 1)
	}
	if comparePage.Cases[0].ID != "case-eval-filter-1" {
		t.Fatalf("comparePage.Cases[0].ID = %q, want %q", comparePage.Cases[0].ID, "case-eval-filter-1")
	}
}

func TestCaseStorePersistsClosedBy(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002100, 0).UTC()
	item := casesvc.Case{
		ID:        "case-closed-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusClosed,
		Title:     "Closed case",
		CreatedBy: "operator-1",
		ClosedBy:  "operator-2",
		CreatedAt: now,
		UpdatedAt: now.Add(time.Second),
	}

	if _, err := store.Save(ctx, item); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(ctx, item.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ClosedBy != "operator-2" {
		t.Fatalf("Get().ClosedBy = %q, want %q", got.ClosedBy, "operator-2")
	}
}

func TestCaseStoreCloseAndReopenRoundTrip(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002150, 0).UTC()
	saved, err := store.Save(ctx, casesvc.Case{
		ID:        "case-reopen-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusOpen,
		Title:     "Reopenable case",
		CreatedBy: "operator-1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	closed, err := store.Close(ctx, saved.ID, "operator-2", now.Add(time.Second))
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if closed.Status != casesvc.StatusClosed {
		t.Fatalf("Close().Status = %q, want %q", closed.Status, casesvc.StatusClosed)
	}
	if closed.ClosedBy != "operator-2" {
		t.Fatalf("Close().ClosedBy = %q, want %q", closed.ClosedBy, "operator-2")
	}

	reopened, err := store.Reopen(ctx, saved.ID, "operator-3", now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("Reopen() error = %v", err)
	}
	if reopened.Status != casesvc.StatusOpen {
		t.Fatalf("Reopen().Status = %q, want %q", reopened.Status, casesvc.StatusOpen)
	}
	if reopened.ClosedBy != "" {
		t.Fatalf("Reopen().ClosedBy = %q, want empty", reopened.ClosedBy)
	}

	notes, err := store.ListNotes(ctx, saved.ID, 10)
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(ListNotes()) = %d, want %d", len(notes), 1)
	}
	if notes[0].Body != "case reopened by operator-3" {
		t.Fatalf("notes[0].Body = %q, want %q", notes[0].Body, "case reopened by operator-3")
	}
}

func TestCaseStorePersistsAssignee(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002200, 0).UTC()
	item := casesvc.Case{
		ID:         "case-assigned-1",
		TenantID:   "tenant-1",
		Status:     casesvc.StatusOpen,
		Title:      "Assigned case",
		CreatedBy:  "operator-1",
		AssignedTo: "owner-1",
		AssignedAt: now.Add(time.Second),
		CreatedAt:  now,
		UpdatedAt:  now.Add(2 * time.Second),
	}

	if _, err := store.Save(ctx, item); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(ctx, item.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.AssignedTo != "owner-1" {
		t.Fatalf("Get().AssignedTo = %q, want %q", got.AssignedTo, "owner-1")
	}
	if got.AssignedAt.IsZero() {
		t.Fatal("Get().AssignedAt is zero")
	}
}

func TestCaseStoreAssignRejectsStaleWrite(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002300, 0).UTC()
	item := casesvc.Case{
		ID:        "case-stale-assign-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusOpen,
		Title:     "Stale assign case",
		CreatedBy: "operator-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	saved, err := store.Save(ctx, item)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	first, err := store.Assign(ctx, saved.ID, "owner-1", now.Add(time.Second), saved.UpdatedAt)
	if err != nil {
		t.Fatalf("Assign(first) error = %v", err)
	}
	if _, err := store.Assign(ctx, saved.ID, "owner-2", now.Add(2*time.Second), saved.UpdatedAt); !errors.Is(err, casesvc.ErrCaseConflict) {
		t.Fatalf("Assign(second) error = %v, want %v", err, casesvc.ErrCaseConflict)
	}
	if first.AssignedTo != "owner-1" {
		t.Fatalf("first.AssignedTo = %q, want %q", first.AssignedTo, "owner-1")
	}
}

func TestCaseStoreUnassignRejectsStaleWrite(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002310, 0).UTC()
	saved, err := store.Save(ctx, casesvc.Case{
		ID:        "case-stale-unassign-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusOpen,
		Title:     "Stale unassign case",
		CreatedBy: "operator-1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	assigned, err := store.Assign(ctx, saved.ID, "owner-1", now.Add(time.Second), saved.UpdatedAt)
	if err != nil {
		t.Fatalf("Assign() error = %v", err)
	}
	if _, err := store.Unassign(ctx, saved.ID, "operator-2", now.Add(2*time.Second), saved.UpdatedAt); !errors.Is(err, casesvc.ErrCaseConflict) {
		t.Fatalf("Unassign(stale) error = %v, want %v", err, casesvc.ErrCaseConflict)
	}
	unassigned, err := store.Unassign(ctx, saved.ID, "operator-2", now.Add(3*time.Second), assigned.UpdatedAt)
	if err != nil {
		t.Fatalf("Unassign() error = %v", err)
	}
	if unassigned.AssignedTo != "" {
		t.Fatalf("Unassign().AssignedTo = %q, want empty", unassigned.AssignedTo)
	}
	if !unassigned.AssignedAt.IsZero() {
		t.Fatal("Unassign().AssignedAt should be zero")
	}

	notes, err := store.ListNotes(ctx, saved.ID, 10)
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(ListNotes()) = %d, want %d", len(notes), 1)
	}
	if notes[0].Body != "case returned to queue by operator-2" {
		t.Fatalf("notes[0].Body = %q, want %q", notes[0].Body, "case returned to queue by operator-2")
	}
	if notes[0].CreatedBy != "operator-2" {
		t.Fatalf("notes[0].CreatedBy = %q, want %q", notes[0].CreatedBy, "operator-2")
	}
}

func TestCaseStoreUnassignRejectsAlreadyUnassignedCase(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002090, 0).UTC()
	item, err := store.Save(ctx, casesvc.Case{
		ID:        "case-unassign-open-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusOpen,
		Title:     "Already unassigned",
		CreatedBy: "operator-1",
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	if _, err := store.Unassign(ctx, item.ID, "operator-2", now.Add(time.Second), item.UpdatedAt); !errors.Is(err, casesvc.ErrInvalidCaseState) {
		t.Fatalf("Unassign() error = %v, want %v", err, casesvc.ErrInvalidCaseState)
	}
}

func TestCaseStoreAppendAndListNotes(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE case_notes, cases, reports, workflow_task_events, workflow_tasks error = %v", err)
	}

	store := NewCaseStore(pool)
	now := time.Unix(1700002400, 0).UTC()
	item := casesvc.Case{
		ID:        "case-note-1",
		TenantID:  "tenant-1",
		Status:    casesvc.StatusOpen,
		Title:     "Case note test",
		CreatedBy: "operator-1",
		CreatedAt: now,
		UpdatedAt: now,
	}
	if _, err := store.Save(ctx, item); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	first, err := store.AppendNote(ctx, casesvc.Note{
		ID:        "case-note-row-1",
		TenantID:  "tenant-1",
		CaseID:    item.ID,
		Body:      "first note",
		CreatedBy: "operator-a",
		CreatedAt: now.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("AppendNote(first) error = %v", err)
	}
	second, err := store.AppendNote(ctx, casesvc.Note{
		ID:        "case-note-row-2",
		TenantID:  "tenant-1",
		CaseID:    item.ID,
		Body:      "second note",
		CreatedBy: "operator-b",
		CreatedAt: now.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("AppendNote(second) error = %v", err)
	}

	notes, err := store.ListNotes(ctx, item.ID, 20)
	if err != nil {
		t.Fatalf("ListNotes() error = %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("len(ListNotes()) = %d, want %d", len(notes), 2)
	}
	if notes[0].ID != second.ID {
		t.Fatalf("notes[0].ID = %q, want %q", notes[0].ID, second.ID)
	}
	if notes[1].ID != first.ID {
		t.Fatalf("notes[1].ID = %q, want %q", notes[1].ID, first.ID)
	}

	refreshed, err := store.Get(ctx, item.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !refreshed.UpdatedAt.Equal(second.CreatedAt) {
		t.Fatalf("Get().UpdatedAt = %v, want %v", refreshed.UpdatedAt, second.CreatedAt)
	}
}

func TestCaseStoreSummarizeBySourceEvalReportIDs(t *testing.T) {
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
	if _, err := pool.Exec(ctx, "TRUNCATE case_notes, cases, eval_reports, reports, workflow_task_events, workflow_tasks RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("TRUNCATE eval-report lineage tables error = %v", err)
	}

	reportStore := NewEvalReportStore(pool)
	reportNow := time.Unix(1700002700, 0).UTC()
	for _, item := range []evalsvc.EvalReport{
		{
			ID:              "eval-report-summary-1",
			TenantID:        "tenant-1",
			RunID:           "eval-run-summary-1",
			DatasetID:       "dataset-summary-1",
			DatasetName:     "Dataset One",
			RunStatus:       evalsvc.RunStatusFailed,
			Status:          evalsvc.EvalReportStatusReady,
			Summary:         "summary one",
			TotalItems:      1,
			RecordedResults: 1,
			PassedItems:     0,
			FailedItems:     1,
			MissingResults:  0,
			AverageScore:    0,
			JudgeVersion:    "judge-a",
			MetadataJSON:    []byte(`{}`),
			CreatedAt:       reportNow,
			UpdatedAt:       reportNow,
			ReadyAt:         reportNow,
		},
		{
			ID:              "eval-report-summary-2",
			TenantID:        "tenant-1",
			RunID:           "eval-run-summary-2",
			DatasetID:       "dataset-summary-1",
			DatasetName:     "Dataset One",
			RunStatus:       evalsvc.RunStatusFailed,
			Status:          evalsvc.EvalReportStatusReady,
			Summary:         "summary two",
			TotalItems:      1,
			RecordedResults: 1,
			PassedItems:     0,
			FailedItems:     1,
			MissingResults:  0,
			AverageScore:    0,
			JudgeVersion:    "judge-a",
			MetadataJSON:    []byte(`{}`),
			CreatedAt:       reportNow.Add(time.Second),
			UpdatedAt:       reportNow.Add(time.Second),
			ReadyAt:         reportNow.Add(time.Second),
		},
	} {
		if _, err := reportStore.SaveEvalReport(ctx, item); err != nil {
			t.Fatalf("SaveEvalReport(%s) error = %v", item.ID, err)
		}
	}

	store := NewCaseStore(pool)
	for _, item := range []casesvc.Case{
		{
			ID:                 "case-summary-1",
			TenantID:           "tenant-1",
			Status:             casesvc.StatusOpen,
			Title:              "Open follow-up",
			SourceEvalReportID: "eval-report-summary-1",
			CreatedBy:          "operator-1",
			CreatedAt:          reportNow,
			UpdatedAt:          reportNow,
		},
		{
			ID:                 "case-summary-2",
			TenantID:           "tenant-1",
			Status:             casesvc.StatusClosed,
			Title:              "Closed follow-up",
			SourceEvalReportID: "eval-report-summary-1",
			CreatedBy:          "operator-2",
			ClosedBy:           "operator-3",
			CreatedAt:          reportNow.Add(2 * time.Second),
			UpdatedAt:          reportNow.Add(3 * time.Second),
		},
		{
			ID:                 "case-summary-other-tenant",
			TenantID:           "tenant-2",
			Status:             casesvc.StatusOpen,
			Title:              "Ignored follow-up",
			SourceEvalReportID: "eval-report-summary-1",
			CreatedBy:          "operator-1",
			CreatedAt:          reportNow.Add(4 * time.Second),
			UpdatedAt:          reportNow.Add(4 * time.Second),
		},
	} {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	summaries, err := store.SummarizeBySourceEvalReportIDs(ctx, "tenant-1", []string{"eval-report-summary-1", "eval-report-summary-2"})
	if err != nil {
		t.Fatalf("SummarizeBySourceEvalReportIDs() error = %v", err)
	}

	got := summaries["eval-report-summary-1"]
	if got.SourceEvalReportID != "eval-report-summary-1" {
		t.Fatalf("SourceEvalReportID = %q, want %q", got.SourceEvalReportID, "eval-report-summary-1")
	}
	if got.FollowUpCaseCount != 2 {
		t.Fatalf("FollowUpCaseCount = %d, want %d", got.FollowUpCaseCount, 2)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want %d", got.OpenFollowUpCaseCount, 1)
	}
	if got.LatestFollowUpCaseID != "case-summary-2" {
		t.Fatalf("LatestFollowUpCaseID = %q, want %q", got.LatestFollowUpCaseID, "case-summary-2")
	}
	if got.LatestFollowUpCaseStatus != casesvc.StatusClosed {
		t.Fatalf("LatestFollowUpCaseStatus = %q, want %q", got.LatestFollowUpCaseStatus, casesvc.StatusClosed)
	}

	empty := summaries["eval-report-summary-2"]
	if empty.SourceEvalReportID != "eval-report-summary-2" {
		t.Fatalf("empty.SourceEvalReportID = %q, want %q", empty.SourceEvalReportID, "eval-report-summary-2")
	}
	if empty.FollowUpCaseCount != 0 {
		t.Fatalf("empty.FollowUpCaseCount = %d, want %d", empty.FollowUpCaseCount, 0)
	}
	if empty.OpenFollowUpCaseCount != 0 {
		t.Fatalf("empty.OpenFollowUpCaseCount = %d, want %d", empty.OpenFollowUpCaseCount, 0)
	}
	if empty.LatestFollowUpCaseID != "" {
		t.Fatalf("empty.LatestFollowUpCaseID = %q, want empty", empty.LatestFollowUpCaseID)
	}
	if empty.LatestFollowUpCaseStatus != "" {
		t.Fatalf("empty.LatestFollowUpCaseStatus = %q, want empty", empty.LatestFollowUpCaseStatus)
	}
}
