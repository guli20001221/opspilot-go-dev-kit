package tracedetail

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	cases "opspilot-go/internal/case"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

type stubWorkflowReader struct {
	items map[string]workflow.Task
}

func (s stubWorkflowReader) GetTask(_ context.Context, taskID string) (workflow.Task, error) {
	item, ok := s.items[taskID]
	if !ok {
		return workflow.Task{}, workflow.ErrTaskNotFound
	}
	return item, nil
}

type stubReportReader struct {
	items map[string]report.Report
}

func (s stubReportReader) GetReport(_ context.Context, reportID string) (report.Report, error) {
	item, ok := s.items[reportID]
	if !ok {
		return report.Report{}, report.ErrReportNotFound
	}
	return item, nil
}

type stubCaseReader struct {
	items map[string]cases.Case
}

func (s stubCaseReader) GetCase(_ context.Context, caseID string) (cases.Case, error) {
	item, ok := s.items[caseID]
	if !ok {
		return cases.Case{}, cases.ErrCaseNotFound
	}
	return item, nil
}

func TestServiceLookupTaskResolvesTemporalAndWarnings(t *testing.T) {
	svc := NewService(
		stubWorkflowReader{items: map[string]workflow.Task{
			"task-1": {
				ID:        "task-1",
				RequestID: "req-1",
				TenantID:  "tenant-1",
				SessionID: "session-1",
				Status:    workflow.StatusSucceeded,
				AuditRef:  "temporal:workflow:task-1/run-1",
			},
		}},
		stubReportReader{},
		stubCaseReader{},
	)

	got, err := svc.Lookup(context.Background(), LookupInput{TaskID: "task-1"})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.Subject.Kind != SubjectTask || got.Subject.ID != "task-1" {
		t.Fatalf("Subject = %#v, want task-1", got.Subject)
	}
	if got.RequestID != "req-1" || got.SessionID != "session-1" {
		t.Fatalf("request/session = %#v, want req-1/session-1", got)
	}
	if got.Temporal == nil || got.Temporal.WorkflowID != "task-1" || got.Temporal.RunID != "run-1" {
		t.Fatalf("Temporal = %#v, want task-1/run-1", got.Temporal)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "trace_id is unavailable in the current skeleton" {
		t.Fatalf("Warnings = %#v, want trace_id warning", got.Warnings)
	}
}

func TestServiceLookupReportFallsBackToMetadataWhenTaskMissing(t *testing.T) {
	metadata := json.RawMessage(`{"request_id":"req-report","session_id":"session-report","audit_ref":"temporal:workflow:task-report/run-report","trace_id":"trace-report"}`)
	svc := NewService(
		stubWorkflowReader{},
		stubReportReader{items: map[string]report.Report{
			"report-1": {
				ID:           "report-1",
				TenantID:     "tenant-1",
				SourceTaskID: "task-report",
				ReportType:   report.TypeWorkflowSummary,
				Status:       report.StatusReady,
				MetadataJSON: metadata,
			},
		}},
		stubCaseReader{},
	)

	got, err := svc.Lookup(context.Background(), LookupInput{ReportID: "report-1"})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.Subject.Kind != SubjectReport || got.Lineage.ReportID != "report-1" || got.Lineage.TaskID != "task-report" {
		t.Fatalf("lineage = %#v, want report-1/task-report", got)
	}
	if got.RequestID != "req-report" || got.SessionID != "session-report" || got.TraceID != "trace-report" {
		t.Fatalf("metadata fallback fields = %#v", got)
	}
	if got.Temporal == nil || got.Temporal.WorkflowID != "task-report" || got.Temporal.RunID != "run-report" {
		t.Fatalf("Temporal = %#v, want task-report/run-report", got.Temporal)
	}
	if len(got.Warnings) != 1 || got.Warnings[0] != "source task lookup failed: workflow task not found" {
		t.Fatalf("Warnings = %#v, want source task lookup warning only", got.Warnings)
	}
}

func TestServiceLookupRejectsAmbiguousInput(t *testing.T) {
	svc := NewService(stubWorkflowReader{}, stubReportReader{}, stubCaseReader{})

	_, err := svc.Lookup(context.Background(), LookupInput{
		TaskID:   "task-1",
		ReportID: "report-1",
	})
	if err == nil {
		t.Fatal("Lookup() error = nil, want invalid lookup")
	}
	if !errors.Is(err, ErrInvalidLookup) {
		t.Fatalf("Lookup() error = %v, want ErrInvalidLookup", err)
	}
}

func TestServiceLookupCaseResolvesLineage(t *testing.T) {
	readyAt := time.Unix(1700006005, 0).UTC()
	svc := NewService(
		stubWorkflowReader{items: map[string]workflow.Task{
			"task-case-1": {
				ID:        "task-case-1",
				RequestID: "req-case-1",
				TenantID:  "tenant-case",
				SessionID: "session-case-1",
				Status:    workflow.StatusSucceeded,
				AuditRef:  "temporal:workflow:task-case-1/run-case-1",
			},
		}},
		stubReportReader{items: map[string]report.Report{
			"report-case-1": {
				ID:           "report-case-1",
				TenantID:     "tenant-case",
				SourceTaskID: "task-case-1",
				ReportType:   report.TypeWorkflowSummary,
				Status:       report.StatusReady,
				CreatedAt:    readyAt.Add(-time.Minute),
				ReadyAt:      &readyAt,
			},
		}},
		stubCaseReader{items: map[string]cases.Case{
			"case-1": {
				ID:             "case-1",
				TenantID:       "tenant-case",
				Status:         cases.StatusOpen,
				SourceTaskID:   "task-case-1",
				SourceReportID: "report-case-1",
			},
		}},
	)

	got, err := svc.Lookup(context.Background(), LookupInput{CaseID: "case-1"})
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}
	if got.Subject.Kind != SubjectCase || got.Subject.ID != "case-1" {
		t.Fatalf("Subject = %#v, want case-1", got.Subject)
	}
	if got.Lineage.TaskID != "task-case-1" || got.Lineage.ReportID != "report-case-1" || got.Lineage.CaseID != "case-1" {
		t.Fatalf("Lineage = %#v", got.Lineage)
	}
	if got.CaseStatus != cases.StatusOpen || got.ReportStatus != report.StatusReady || got.TaskStatus != workflow.StatusSucceeded {
		t.Fatalf("statuses = %#v", got)
	}
	if got.Temporal == nil || got.Temporal.WorkflowID != "task-case-1" {
		t.Fatalf("Temporal = %#v, want task-case-1", got.Temporal)
	}
}
