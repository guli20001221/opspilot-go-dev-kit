package cases

import (
	"context"
	"testing"
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
