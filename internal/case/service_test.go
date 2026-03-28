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

func TestServiceCreateAndGetCasePreservesCompareOrigin(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID:           "tenant-1",
		Title:              "Compare-derived regression",
		SourceEvalReportID: "eval-report-right",
		CompareOrigin: CompareOrigin{
			LeftEvalReportID:  "eval-report-left",
			RightEvalReportID: "eval-report-right",
			SelectedSide:      "right",
		},
		CreatedBy: "operator-1",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	got, err := svc.GetCase(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("GetCase() error = %v", err)
	}
	if got.CompareOrigin.LeftEvalReportID != "eval-report-left" {
		t.Fatalf("CompareOrigin.LeftEvalReportID = %q, want %q", got.CompareOrigin.LeftEvalReportID, "eval-report-left")
	}
	if got.CompareOrigin.RightEvalReportID != "eval-report-right" {
		t.Fatalf("CompareOrigin.RightEvalReportID = %q, want %q", got.CompareOrigin.RightEvalReportID, "eval-report-right")
	}
	if got.CompareOrigin.SelectedSide != "right" {
		t.Fatalf("CompareOrigin.SelectedSide = %q, want %q", got.CompareOrigin.SelectedSide, "right")
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

func TestServiceListCasesSupportsUnassignedOnlyFilter(t *testing.T) {
	svc := NewService()

	unassigned, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Unassigned",
	})
	if err != nil {
		t.Fatalf("CreateCase(unassigned) error = %v", err)
	}
	assigned, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Assigned",
	})
	if err != nil {
		t.Fatalf("CreateCase(assigned) error = %v", err)
	}
	if _, err := svc.AssignCase(context.Background(), assigned, "cases-operator"); err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}

	page, err := svc.ListCases(context.Background(), ListFilter{
		TenantID:       "tenant-1",
		Status:         StatusOpen,
		UnassignedOnly: true,
		Limit:          10,
	})
	if err != nil {
		t.Fatalf("ListCases() error = %v", err)
	}
	if len(page.Cases) != 1 {
		t.Fatalf("len(ListCases().Cases) = %d, want %d", len(page.Cases), 1)
	}
	if page.Cases[0].ID != unassigned.ID {
		t.Fatalf("ListCases().Cases[0].ID = %q, want %q", page.Cases[0].ID, unassigned.ID)
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

func TestServiceReopenCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Reopen me",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	if _, err := svc.CloseCase(context.Background(), created.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	reopened, err := svc.ReopenCase(context.Background(), created.ID, "operator-2")
	if err != nil {
		t.Fatalf("ReopenCase() error = %v", err)
	}
	if reopened.Status != StatusOpen {
		t.Fatalf("ReopenCase().Status = %q, want %q", reopened.Status, StatusOpen)
	}
	if reopened.ClosedBy != "" {
		t.Fatalf("ReopenCase().ClosedBy = %q, want empty", reopened.ClosedBy)
	}

	notes, err := svc.ListCaseNotes(context.Background(), created.ID, 10)
	if err != nil {
		t.Fatalf("ListCaseNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(ListCaseNotes()) = %d, want %d", len(notes), 1)
	}
	if notes[0].Body != "case reopened by operator-2" {
		t.Fatalf("notes[0].Body = %q, want %q", notes[0].Body, "case reopened by operator-2")
	}
}

func TestServiceReopenCaseRejectsOpenCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Already open",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	if _, err := svc.ReopenCase(context.Background(), created.ID, "operator-1"); !errors.Is(err, ErrInvalidCaseState) {
		t.Fatalf("ReopenCase() error = %v, want %v", err, ErrInvalidCaseState)
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

func TestServiceUnassignCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Release me",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	assigned, err := svc.AssignCase(context.Background(), created, "owner-1")
	if err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}

	unassigned, err := svc.UnassignCase(context.Background(), assigned, "operator-2")
	if err != nil {
		t.Fatalf("UnassignCase() error = %v", err)
	}
	if unassigned.AssignedTo != "" {
		t.Fatalf("UnassignCase().AssignedTo = %q, want empty", unassigned.AssignedTo)
	}
	if !unassigned.AssignedAt.IsZero() {
		t.Fatal("UnassignCase().AssignedAt should be zero")
	}

	notes, err := svc.ListCaseNotes(context.Background(), assigned.ID, 10)
	if err != nil {
		t.Fatalf("ListCaseNotes() error = %v", err)
	}
	if len(notes) != 1 {
		t.Fatalf("len(ListCaseNotes()) = %d, want %d", len(notes), 1)
	}
	if notes[0].Body != "case returned to queue by operator-2" {
		t.Fatalf("notes[0].Body = %q, want %q", notes[0].Body, "case returned to queue by operator-2")
	}
	if notes[0].CreatedBy != "operator-2" {
		t.Fatalf("notes[0].CreatedBy = %q, want %q", notes[0].CreatedBy, "operator-2")
	}
}

func TestServiceUnassignCaseRejectsClosedCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Closed before unassign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	assigned, err := svc.AssignCase(context.Background(), created, "owner-1")
	if err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}
	if _, err := svc.CloseCase(context.Background(), assigned.ID, "operator-1"); err != nil {
		t.Fatalf("CloseCase() error = %v", err)
	}

	if _, err := svc.UnassignCase(context.Background(), assigned, "operator-2"); !errors.Is(err, ErrInvalidCaseState) {
		t.Fatalf("UnassignCase() error = %v, want %v", err, ErrInvalidCaseState)
	}
}

func TestServiceUnassignCaseRejectsAlreadyUnassignedCase(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Already unassigned",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}

	if _, err := svc.UnassignCase(context.Background(), created, "operator-2"); !errors.Is(err, ErrInvalidCaseState) {
		t.Fatalf("UnassignCase() error = %v, want %v", err, ErrInvalidCaseState)
	}
}

func TestServiceUnassignCaseRejectsStaleWrite(t *testing.T) {
	svc := NewService()

	created, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID: "tenant-1",
		Title:    "Stale unassign",
	})
	if err != nil {
		t.Fatalf("CreateCase() error = %v", err)
	}
	assigned, err := svc.AssignCase(context.Background(), created, "owner-1")
	if err != nil {
		t.Fatalf("AssignCase() error = %v", err)
	}
	stale := assigned
	stale.UpdatedAt = stale.UpdatedAt.Add(-time.Nanosecond)

	if _, err := svc.UnassignCase(context.Background(), stale, "operator-2"); !errors.Is(err, ErrCaseConflict) {
		t.Fatalf("UnassignCase() error = %v, want %v", err, ErrCaseConflict)
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

func TestServiceSummarizeBySourceEvalReportIDs(t *testing.T) {
	svc := NewService()
	now := time.Unix(1700002600, 0).UTC()

	first, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID:           "tenant-1",
		Title:              "First follow-up",
		SourceEvalReportID: "eval-report-1",
		CreatedBy:          "operator-1",
	})
	if err != nil {
		t.Fatalf("CreateCase(first) error = %v", err)
	}
	first.CreatedAt = now
	first.UpdatedAt = now
	if _, err := svc.store.Save(context.Background(), first); err != nil {
		t.Fatalf("store.Save(first) error = %v", err)
	}

	second, err := svc.CreateCase(context.Background(), CreateInput{
		TenantID:           "tenant-1",
		Title:              "Second follow-up",
		SourceEvalReportID: "eval-report-1",
		CreatedBy:          "operator-2",
	})
	if err != nil {
		t.Fatalf("CreateCase(second) error = %v", err)
	}
	second.CreatedAt = now.Add(time.Second)
	second.UpdatedAt = now.Add(time.Second)
	if _, err := svc.store.Save(context.Background(), second); err != nil {
		t.Fatalf("store.Save(second) error = %v", err)
	}

	if _, err := svc.CloseCase(context.Background(), first.ID, "operator-3"); err != nil {
		t.Fatalf("CloseCase(first) error = %v", err)
	}

	summaries, err := svc.SummarizeBySourceEvalReportIDs(context.Background(), "tenant-1", []string{"eval-report-1", "eval-report-2"})
	if err != nil {
		t.Fatalf("SummarizeBySourceEvalReportIDs() error = %v", err)
	}

	got := summaries["eval-report-1"]
	if got.SourceEvalReportID != "eval-report-1" {
		t.Fatalf("SourceEvalReportID = %q, want %q", got.SourceEvalReportID, "eval-report-1")
	}
	if got.FollowUpCaseCount != 2 {
		t.Fatalf("FollowUpCaseCount = %d, want %d", got.FollowUpCaseCount, 2)
	}
	if got.OpenFollowUpCaseCount != 1 {
		t.Fatalf("OpenFollowUpCaseCount = %d, want %d", got.OpenFollowUpCaseCount, 1)
	}
	if got.LatestFollowUpCaseID != first.ID {
		t.Fatalf("LatestFollowUpCaseID = %q, want %q", got.LatestFollowUpCaseID, first.ID)
	}
	if got.LatestFollowUpCaseStatus != StatusClosed {
		t.Fatalf("LatestFollowUpCaseStatus = %q, want %q", got.LatestFollowUpCaseStatus, StatusClosed)
	}

	empty := summaries["eval-report-2"]
	if empty.SourceEvalReportID != "eval-report-2" {
		t.Fatalf("empty.SourceEvalReportID = %q, want %q", empty.SourceEvalReportID, "eval-report-2")
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
