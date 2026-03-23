package workflow

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreListTasksUsesStableIDTieBreakForPagination(t *testing.T) {
	store := NewMemoryStore()
	ts := time.Unix(1700000600, 0).UTC()

	for _, task := range []Task{
		{
			ID:        "task-page-1",
			RequestID: "req-page-1",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  TaskTypeReportGeneration,
			Status:    StatusSucceeded,
			Reason:    PromotionReasonWorkflowRequired,
			CreatedAt: ts,
			UpdatedAt: ts,
		},
		{
			ID:        "task-page-2",
			RequestID: "req-page-2",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  TaskTypeReportGeneration,
			Status:    StatusSucceeded,
			Reason:    PromotionReasonWorkflowRequired,
			CreatedAt: ts,
			UpdatedAt: ts,
		},
		{
			ID:        "task-page-3",
			RequestID: "req-page-3",
			TenantID:  "tenant-page",
			SessionID: "session-page",
			TaskType:  TaskTypeReportGeneration,
			Status:    StatusSucceeded,
			Reason:    PromotionReasonWorkflowRequired,
			CreatedAt: ts,
			UpdatedAt: ts,
		},
	} {
		if _, err := store.SaveTask(context.Background(), task); err != nil {
			t.Fatalf("SaveTask(%s) error = %v", task.ID, err)
		}
	}

	firstPage, err := store.ListTasks(context.Background(), TaskListFilter{
		TenantID: "tenant-page",
		Limit:    1,
	})
	if err != nil {
		t.Fatalf("ListTasks(firstPage) error = %v", err)
	}
	secondPage, err := store.ListTasks(context.Background(), TaskListFilter{
		TenantID: "tenant-page",
		Limit:    1,
		Offset:   1,
	})
	if err != nil {
		t.Fatalf("ListTasks(secondPage) error = %v", err)
	}

	if len(firstPage.Tasks) != 1 {
		t.Fatalf("len(firstPage.Tasks) = %d, want %d", len(firstPage.Tasks), 1)
	}
	if len(secondPage.Tasks) != 1 {
		t.Fatalf("len(secondPage.Tasks) = %d, want %d", len(secondPage.Tasks), 1)
	}
	if firstPage.Tasks[0].ID != "task-page-3" {
		t.Fatalf("firstPage.Tasks[0].ID = %q, want %q", firstPage.Tasks[0].ID, "task-page-3")
	}
	if secondPage.Tasks[0].ID != "task-page-2" {
		t.Fatalf("secondPage.Tasks[0].ID = %q, want %q", secondPage.Tasks[0].ID, "task-page-2")
	}
}
