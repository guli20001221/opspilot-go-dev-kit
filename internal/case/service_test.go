package cases

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestServiceCreateAndGetCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID:       "tenant-1",
		Title:          "Investigate report mismatch",
		Summary:        "Operator needs to review a successful report task.",
		SourceTaskID:   "task-1",
		SourceReportID: "report-task-1",
		CreatedBy:      "operator-1",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("CreateCase().ID is empty")
	}
	if created.Status != StatusOpen {
		t.Fatalf("CreateCase().Status = %q, want %q", created.Status, StatusOpen)
	}
	if created.CreatedBy != "operator-1" {
		t.Fatalf("CreateCase().CreatedBy = %q, want %q", created.CreatedBy, "operator-1")
	}

	got, err := svc.GetCase(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetCase() error = %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetCase().ID = %q, want %q", got.ID, created.ID)
	}
	if got.SourceReportID != "report-task-1" {
		t.Fatalf("GetCase().SourceReportID = %q, want %q", got.SourceReportID, "report-task-1")
	}
}

func TestServiceDefaultsCreatedBy(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Manual case",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if created.CreatedBy != "operator" {
		t.Fatalf("CreateCase().CreatedBy = %q, want %q", created.CreatedBy, "operator")
	}
}

func TestServiceListCasesSupportsFilterAndOffset(t *testing.T) {
	svc := NewService()

	first, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Case 1",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	second, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID:       "tenant-1",
		Title:          "Case 2",
		SourceTaskID:   "task-2",
		SourceReportID: "report-2",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	if _, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-2",
		Title:    "Case 3",
	}); err != nil {
		t.Fatalf("CreateCase(third) error = %v", err)
	}

	page, err := svc.ListCases(context.Background(), ListFilter{
		TenantID: "tenant-1",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(ListCases().Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].ID != second.ID {
		t.Fatalf("ListCases().Cases[0].ID = %q, want %q", page.Cases[0].ID, second.ID)
	}
	if !page.HasMore {
		t.Fatal("ListCases().HasMore = false, want true")
	}

	nextPage, err := svc.ListCases(context.Background(), ListFilter{
		TenantID: "tenant-1",
		Limit:    1,
		Offset:   page.NextOffset,
	})
	if err != nil {
		t.Fatalf("ListCases(nextPage) error = %v", err)
	}
	if len(nextPage.Cases) != 1 {
		t.Fatalf("len(nextPage.Cases) = %d, want %d", len(nextPage.Cases), 1)
	}
	if nextPage.Cases[0].ID != first.ID {
		t.Fatalf("nextPage.Cases[0].ID = %q, want %q", nextPage.Cases[0].ID, first.ID)
	}
}

func TestServiceListCasesSupportsAssignedToFilter(t *testing.T) {
	svc := NewService()

	first, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Assigned to me",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	second, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Assigned elsewhere",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	if _, err := svc.AssignCase(context.Background(), first, "cases-operator"); err != nil {
		t.Fatalf("AssignCase(first) error = %v", err)
	}
	if _, err := svc.AssignCase(context.Background(), second, "other-operator"); err != nil {
		t.Fatalf("AssignCase(second) error = %v", err)
	}

	page, err := svc.ListCases(context.Background(), ListFilter{
		TenantID:   "tenant-1",
		Status:     StatusOpen,
		AssignedTo: "cases-operator",
		Limit:      10,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(ListCases().Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].ID != first.ID {
		t.Fatalf("ListCases().Cases[0].ID = %q, want %q", page.Cases[0].ID, first.ID)
	}
}

func TestServiceCloseCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Close me",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	closed, err := svc.CloseCase(context.Background(), created.ID, "operator-2")
	if err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}
	if closed.Status != StatusClosed {
		t.Fatalf("CloseCase().Status = %q, want %q", closed.Status, StatusClosed)
	}
	if closed.ClosedBy != "operator-2" {
		t.Fatalf("CloseCase().ClosedBy = %q, want %q", closed.ClosedBy, "operator-2")
	}
	if closed.UpdatedAt.Before(created.UpdatedAt) {
		t.Fatal("CloseCase().UpdatedAt regressed")
	}
}

func TestServiceCloseCaseRejectsClosedCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Already closed",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := svc.CloseCase(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase(first) error = %v", err)
	}

	if _, err := svc.CloseCase(context.Background(), created.ID, "operator-2"); !errors.Is(err, ErrInvalidCaseState) {
		t.Fatalf("CloseCase(second) error = %v, want %v", err, ErrInvalidCaseState)
	}
}

func TestServiceCloseCaseAllowsOnlyOneConcurrentCloser(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Concurrent close",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, actor := range []string{"operator-a", "operator-b"} {
		wg.Add(1)
		go func(actor string) {
			defer wg.Done()
			<-start
			_, err := svc.CloseCase(context.Background(), created.ID, actor)
			results <- err
		}(actor)
	}

	close(start)
	wg.Wait()
	close(results)

	var successCount int
	var invalidCount int
	for err := range results {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, ErrInvalidCaseState):
			invalidCount++
		default:
			t.Fatalf("CloseCase(concurrent) unexpected error = %v", err)
		}
	}

	if successCount != 1 {
		t.Fatalf("successCount = %d, want %d", successCount, 1)
	}
	if invalidCount != 1 {
		t.Fatalf("invalidCount = %d, want %d", invalidCount, 1)
	}
}

func TestServiceAssignCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Assign me",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	assigned, err := svc.AssignCase(context.Background(), created, "owner-1")
	if err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}
	if assigned.AssignedTo != "owner-1" {
		t.Fatalf("AssignCase().AssignedTo = %q, want %q", assigned.AssignedTo, "owner-1")
	}
	if assigned.AssignedAt.IsZero() {
		t.Fatal("AssignCase().AssignedAt is zero")
	}
}

func TestServiceAssignCaseRejectsClosedCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Closed before assign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := svc.CloseCase(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	if _, err := svc.AssignCase(context.Background(), created, "owner-1"); !errors.Is(err, ErrInvalidCaseState) {
		t.Fatalf("AssignCase() error = %v, want %v", err, ErrInvalidCaseState)
	}
}

func TestServiceAssignCaseRejectsStaleWrite(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Stale assign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := svc.AssignCase(context.Background(), created, "owner-1"); err != nil {
		t.Fatalf("AssignCase(first) error = %v", err)
	}
	stale := created
	stale.UpdatedAt = stale.UpdatedAt.Add(-time.Nanosecond)

	if _, err := svc.AssignCase(context.Background(), stale, "owner-2"); !errors.Is(err, ErrCaseConflict) {
		t.Fatalf("AssignCase(second) error = %v, want %v", err, ErrCaseConflict)
	}
}

func TestServiceAddAndListCaseNotes(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Case with notes",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	first, err := svc.AddNote(context.Background(), created, "first note", "operator-a")
	if err != nil {
		t.Fatalf("AddNote(first) error = %v", err)
	}
	second, err := svc.AddNote(context.Background(), created, "second note", "operator-b")
	if err != nil {
		t.Fatalf("AddNote(second) error = %v", err)
	}

	notes, err := svc.ListCaseNotes(context.Background(), created.ID, 20)
	if err != nil {
		t.Fatalf("ListCaseNotes() error = %v", err)
	}
	if len(notes) != 2 {
		t.Fatalf("len(ListCaseNotes()) = %d, want %d", len(notes), 2)
	}
	if notes[0].ID != second.ID {
		t.Fatalf("notes[0].ID = %q, want %q", notes[0].ID, second.ID)
	}
	if notes[1].ID != first.ID {
		t.Fatalf("notes[1].ID = %q, want %q", notes[1].ID, first.ID)
	}

	refreshed, err := svc.GetCase(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetCase() error = %v", err)
	}
	if !refreshed.UpdatedAt.Equal(second.CreatedAt) {
		t.Fatalf("GetCase().UpdatedAt = %v, want %v", refreshed.UpdatedAt, second.CreatedAt)
	}
}

func TestServiceAddNoteRejectsEmptyBody(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Case with invalid note",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	if _, err := svc.AddNote(context.Background(), created, "   ", "operator-a"); !errors.Is(err, ErrInvalidNote) {
		t.Fatalf("AddNote() error = %v, want %v", err, ErrInvalidNote)
	}
}
