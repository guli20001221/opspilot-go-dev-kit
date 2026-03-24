package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	casesvc "opspilot-go/internal/case"
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
		ID:             "case-roundtrip-1",
		TenantID:       "tenant-1",
		Status:         casesvc.StatusOpen,
		Title:          "Review generated report",
		Summary:        "Operator wants a durable follow-up case.",
		SourceTaskID:   "task-source-1",
		SourceReportID: "report-source-1",
		CreatedBy:      "operator-1",
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
