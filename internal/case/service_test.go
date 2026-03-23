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
