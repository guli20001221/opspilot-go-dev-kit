package report

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"opspilot-go/internal/workflow"
)

func TestServiceRecordGeneratedReport(t *testing.T) {
	svc := NewService()
	task := workflow.Task{
		ID:        "task-report-1",
		RequestID: "req-1",
		TenantID:  "tenant-1",
		SessionID: "session-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		AuditRef:  "temporal:workflow:task-report-1/run-1",
		CreatedAt: time.Unix(1700001000, 0).UTC(),
		UpdatedAt: time.Unix(1700001001, 0).UTC(),
	}

	reportID, err := svc.RecordGeneratedReport(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated:task-report-1",
	})
	if err != nil {
		t.Fatalf("RecordGeneratedReport() error = %v", err)
	}
	if reportID != "report-task-report-1" {
		t.Fatalf("reportID = %q, want %q", reportID, "report-task-report-1")
	}

	got, err := svc.GetReport(context.Background(), reportID)
	if err != nil {
		t.Fatalf("GetReport() error = %v", err)
	}
	if got.SourceTaskID != task.ID {
		t.Fatalf("SourceTaskID = %q, want %q", got.SourceTaskID, task.ID)
	}
	if got.Status != StatusReady {
		t.Fatalf("Status = %q, want %q", got.Status, StatusReady)
	}
	if got.Summary != "generated:task-report-1" {
		t.Fatalf("Summary = %q, want execution detail", got.Summary)
	}

	var metadata map[string]any
	if err := json.Unmarshal(got.MetadataJSON, &metadata); err != nil {
		t.Fatalf("Unmarshal(metadata) error = %v", err)
	}
	if metadata["task_id"] != task.ID {
		t.Fatalf("metadata task_id = %v, want %q", metadata["task_id"], task.ID)
	}
}

func TestServiceSupportsAtomicFinalizationDependsOnStore(t *testing.T) {
	if NewService().SupportsAtomicFinalization() {
		t.Fatal("SupportsAtomicFinalization() = true, want false for default memory store")
	}
}

type fakeCurrentVersionSource struct {
	versionID string
}

func (s fakeCurrentVersionSource) CurrentVersionID(context.Context) (string, error) {
	return s.versionID, nil
}

type finalizingStore struct {
	Store
	lastTask   workflow.Task
	lastReport Report
}

func (s *finalizingStore) FinalizeSucceededTaskWithReport(ctx context.Context, task workflow.Task, _ workflow.AuditEvent, item Report) (Report, workflow.Task, error) {
	s.lastTask = task
	s.lastReport = item

	saved, err := s.Store.Save(ctx, item)
	if err != nil {
		return Report{}, workflow.Task{}, err
	}

	return saved, task, nil
}

func TestServiceFinalizeGeneratedReportTaskPropagatesFallbackVersionToTask(t *testing.T) {
	store := &finalizingStore{Store: newMemoryStore()}
	svc := NewServiceWithDependencies(store, fakeCurrentVersionSource{versionID: "version-fallback-v1"})

	task := workflow.Task{
		ID:        "task-report-fallback-version",
		TenantID:  "tenant-1",
		TaskType:  workflow.TaskTypeReportGeneration,
		Status:    workflow.StatusSucceeded,
		Reason:    workflow.PromotionReasonWorkflowRequired,
		CreatedAt: time.Unix(1700005000, 0).UTC(),
		UpdatedAt: time.Unix(1700005001, 0).UTC(),
	}

	updated, reportID, err := svc.FinalizeGeneratedReportTask(context.Background(), task, workflow.ExecutionResult{
		Detail: "generated:task-report-fallback-version",
	}, workflow.AuditEvent{
		TaskID:    task.ID,
		Action:    workflow.AuditActionSucceeded,
		Actor:     "worker",
		Detail:    "generated:task-report-fallback-version",
		CreatedAt: task.UpdatedAt,
	})
	if err != nil {
		t.Fatalf("FinalizeGeneratedReportTask() error = %v", err)
	}

	if updated.VersionID != "version-fallback-v1" {
		t.Fatalf("updated.VersionID = %q, want %q", updated.VersionID, "version-fallback-v1")
	}
	if store.lastTask.VersionID != "version-fallback-v1" {
		t.Fatalf("finalizer task.VersionID = %q, want %q", store.lastTask.VersionID, "version-fallback-v1")
	}
	if store.lastReport.VersionID != "version-fallback-v1" {
		t.Fatalf("finalizer report.VersionID = %q, want %q", store.lastReport.VersionID, "version-fallback-v1")
	}
	if reportID != ReportIDFromTaskID(task.ID) {
		t.Fatalf("reportID = %q, want %q", reportID, ReportIDFromTaskID(task.ID))
	}
}

func TestServiceListReportsAppliesFiltersAndPagination(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	readyOne := time.Unix(1700002003, 0).UTC()
	readyTwo := time.Unix(1700002005, 0).UTC()
	readyThree := time.Unix(1700002004, 0).UTC()
	items := []Report{
		{
			ID:           "report-a",
			TenantID:     "tenant-a",
			SourceTaskID: "task-a",
			ReportType:   TypeWorkflowSummary,
			Status:       StatusReady,
			Title:        "A",
			Summary:      "A",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002000, 0).UTC(),
			ReadyAt:      &readyOne,
		},
		{
			ID:           "report-b",
			TenantID:     "tenant-a",
			SourceTaskID: "task-b",
			ReportType:   TypeWorkflowSummary,
			Status:       StatusReady,
			Title:        "B",
			Summary:      "B",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002001, 0).UTC(),
			ReadyAt:      &readyTwo,
		},
		{
			ID:           "report-c",
			TenantID:     "tenant-b",
			SourceTaskID: "task-c",
			ReportType:   TypeWorkflowSummary,
			Status:       StatusReady,
			Title:        "C",
			Summary:      "C",
			CreatedBy:    "worker",
			CreatedAt:    time.Unix(1700002002, 0).UTC(),
			ReadyAt:      &readyThree,
		},
	}

	for _, item := range items {
		if _, err := svc.store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := svc.ListReports(ctx, ListFilter{
		TenantID:   "tenant-a",
		Status:     StatusReady,
		ReportType: TypeWorkflowSummary,
		Limit:      1,
	})
	if err != nil {
		t.Fatalf("ListReports() error = %v", err)
	}
	if len(page.Reports) != 1 {
		t.Fatalf("len(Reports) = %d, want 1", len(page.Reports))
	}
	if page.Reports[0].ID != "report-b" {
		t.Fatalf("Reports[0].ID = %q, want report-b", page.Reports[0].ID)
	}
	if !page.HasMore || page.NextOffset != 1 {
		t.Fatalf("pagination = %#v, want has_more with next_offset=1", page)
	}

	nextPage, err := svc.ListReports(ctx, ListFilter{
		TenantID:   "tenant-a",
		Status:     StatusReady,
		ReportType: TypeWorkflowSummary,
		Limit:      1,
		Offset:     1,
	})
	if err != nil {
		t.Fatalf("ListReports(offset) error = %v", err)
	}
	if len(nextPage.Reports) != 1 || nextPage.Reports[0].ID != "report-a" {
		t.Fatalf("next page = %#v, want report-a", nextPage.Reports)
	}
	if nextPage.HasMore {
		t.Fatalf("HasMore = true, want false on final page")
	}
}

func TestServiceCompareReportsBuildsOperatorFacingSummary(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	leftReady := time.Unix(1700004000, 0).UTC()
	rightReady := time.Unix(1700004012, 0).UTC()
	left := Report{
		ID:           "report-compare-left",
		TenantID:     "tenant-compare",
		SourceTaskID: "task-left",
		ReportType:   TypeWorkflowSummary,
		Status:       StatusReady,
		Title:        "Left Report",
		Summary:      "left summary",
		ContentURI:   "s3://reports/left",
		MetadataJSON: json.RawMessage(`{"version":"v1"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700003990, 0).UTC(),
		ReadyAt:      &leftReady,
	}
	right := Report{
		ID:           "report-compare-right",
		TenantID:     "tenant-compare",
		SourceTaskID: "task-right",
		ReportType:   TypeWorkflowSummary,
		Status:       StatusReady,
		Title:        "Right Report",
		Summary:      "right summary",
		ContentURI:   "s3://reports/right",
		MetadataJSON: json.RawMessage(`{"version":"v2"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700003995, 0).UTC(),
		ReadyAt:      &rightReady,
	}
	for _, item := range []Report{left, right} {
		if _, err := svc.store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	got, err := svc.CompareReports(ctx, left.ID, right.ID)
	if err != nil {
		t.Fatalf("CompareReports() error = %v", err)
	}
	if got.Left.ID != left.ID || got.Right.ID != right.ID {
		t.Fatalf("comparison IDs = %#v, want %q and %q", got, left.ID, right.ID)
	}
	if !got.Summary.SameTenant || !got.Summary.SameReportType {
		t.Fatalf("summary tenant/type = %#v, want same tenant and type", got.Summary)
	}
	if !got.Summary.SourceTaskChanged || !got.Summary.TitleChanged || !got.Summary.SummaryChanged {
		t.Fatalf("summary diff flags = %#v, want source/title/summary changes", got.Summary)
	}
	if !got.Summary.ContentURIChanged || !got.Summary.MetadataChanged || !got.Summary.ReadyAtChanged {
		t.Fatalf("summary content/metadata/ready flags = %#v, want changes", got.Summary)
	}
	if got.Summary.ReadyAtDeltaSecond != 12 {
		t.Fatalf("ReadyAtDeltaSecond = %d, want 12", got.Summary.ReadyAtDeltaSecond)
	}
}

func TestServiceCompareReportsIgnoresMetadataKeyOrder(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	left := Report{
		ID:           "report-compare-meta-left",
		TenantID:     "tenant-compare",
		SourceTaskID: "task-meta-left",
		ReportType:   TypeWorkflowSummary,
		Status:       StatusReady,
		Title:        "Same",
		Summary:      "Same",
		MetadataJSON: json.RawMessage(`{"version":"v1","dataset":"incidents"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700004100, 0).UTC(),
	}
	right := Report{
		ID:           "report-compare-meta-right",
		TenantID:     "tenant-compare",
		SourceTaskID: "task-meta-right",
		ReportType:   TypeWorkflowSummary,
		Status:       StatusReady,
		Title:        "Same",
		Summary:      "Same",
		MetadataJSON: json.RawMessage(`{"dataset":"incidents","version":"v1"}`),
		CreatedBy:    "worker",
		CreatedAt:    time.Unix(1700004100, 0).UTC(),
	}
	for _, item := range []Report{left, right} {
		if _, err := svc.store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	got, err := svc.CompareReports(ctx, left.ID, right.ID)
	if err != nil {
		t.Fatalf("CompareReports() error = %v", err)
	}
	if got.Summary.MetadataChanged {
		t.Fatalf("MetadataChanged = true, want false for semantically identical JSON: %#v", got.Summary)
	}
}
