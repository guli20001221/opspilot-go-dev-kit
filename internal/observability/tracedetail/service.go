package tracedetail

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	cases "opspilot-go/internal/case"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

// ErrInvalidLookup identifies invalid drill-down query combinations.
var ErrInvalidLookup = errors.New("invalid trace drill-down lookup")

type workflowReader interface {
	GetTask(ctx context.Context, taskID string) (workflow.Task, error)
}

type reportReader interface {
	GetReport(ctx context.Context, reportID string) (report.Report, error)
}

type caseReader interface {
	GetCase(ctx context.Context, caseID string) (cases.Case, error)
}

// Service resolves durable task, report, and case objects into a narrow trace drill-down view.
type Service struct {
	workflows workflowReader
	reports   reportReader
	cases     caseReader
}

// NewService constructs the trace drill-down service.
func NewService(workflows workflowReader, reports reportReader, cases caseReader) *Service {
	return &Service{
		workflows: workflows,
		reports:   reports,
		cases:     cases,
	}
}

// Lookup resolves one durable task, report, or case identifier into a trace drill-down result.
func (s *Service) Lookup(ctx context.Context, input LookupInput) (Result, error) {
	setCount := 0
	if input.TaskID != "" {
		setCount++
	}
	if input.ReportID != "" {
		setCount++
	}
	if input.CaseID != "" {
		setCount++
	}
	if setCount != 1 {
		return Result{}, fmt.Errorf("%w: exactly one of task_id, report_id, case_id is required", ErrInvalidLookup)
	}

	switch {
	case input.TaskID != "":
		return s.lookupTask(ctx, input.TaskID)
	case input.ReportID != "":
		return s.lookupReport(ctx, input.ReportID)
	default:
		return s.lookupCase(ctx, input.CaseID)
	}
}

func (s *Service) lookupTask(ctx context.Context, taskID string) (Result, error) {
	task, err := s.workflows.GetTask(ctx, taskID)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Subject: Subject{
			Kind:     SubjectTask,
			ID:       task.ID,
			TenantID: task.TenantID,
		},
		Lineage: Lineage{
			TaskID: task.ID,
		},
		VersionID:  task.VersionID,
		RequestID:  task.RequestID,
		SessionID:  task.SessionID,
		AuditRef:   task.AuditRef,
		TaskStatus: task.Status,
	}
	applyTemporalRef(&result)
	addTraceWarning(&result)

	return result, nil
}

func (s *Service) lookupReport(ctx context.Context, reportID string) (Result, error) {
	item, err := s.reports.GetReport(ctx, reportID)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Subject: Subject{
			Kind:     SubjectReport,
			ID:       item.ID,
			TenantID: item.TenantID,
		},
		Lineage: Lineage{
			ReportID: item.ID,
			TaskID:   item.SourceTaskID,
		},
		ReportType:   item.ReportType,
		ReportStatus: item.Status,
	}
	applyReportMetadata(&result, item)
	if item.SourceTaskID != "" {
		if task, taskErr := s.workflows.GetTask(ctx, item.SourceTaskID); taskErr == nil {
			applyTask(&result, task)
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("source task lookup failed: %v", taskErr))
		}
	}
	applyTemporalRef(&result)
	addTraceWarning(&result)

	return result, nil
}

func (s *Service) lookupCase(ctx context.Context, caseID string) (Result, error) {
	item, err := s.cases.GetCase(ctx, caseID)
	if err != nil {
		return Result{}, err
	}

	result := Result{
		Subject: Subject{
			Kind:     SubjectCase,
			ID:       item.ID,
			TenantID: item.TenantID,
		},
		Lineage: Lineage{
			CaseID:   item.ID,
			TaskID:   item.SourceTaskID,
			ReportID: item.SourceReportID,
		},
		CaseStatus: item.Status,
	}
	if item.SourceReportID != "" {
		if reportItem, reportErr := s.reports.GetReport(ctx, item.SourceReportID); reportErr == nil {
			applyReportMetadata(&result, reportItem)
			result.ReportType = reportItem.ReportType
			result.ReportStatus = reportItem.Status
			if result.Lineage.TaskID == "" {
				result.Lineage.TaskID = reportItem.SourceTaskID
			}
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("source report lookup failed: %v", reportErr))
		}
	}
	if result.Lineage.TaskID != "" {
		if task, taskErr := s.workflows.GetTask(ctx, result.Lineage.TaskID); taskErr == nil {
			applyTask(&result, task)
		} else {
			result.Warnings = append(result.Warnings, fmt.Sprintf("source task lookup failed: %v", taskErr))
		}
	}
	applyTemporalRef(&result)
	addTraceWarning(&result)

	return result, nil
}

func applyTask(result *Result, task workflow.Task) {
	result.Lineage.TaskID = task.ID
	if result.VersionID == "" {
		result.VersionID = task.VersionID
	}
	result.RequestID = task.RequestID
	result.SessionID = task.SessionID
	result.AuditRef = task.AuditRef
	result.TaskStatus = task.Status
}

func applyReportMetadata(result *Result, item report.Report) {
	result.Lineage.ReportID = item.ID
	if result.VersionID == "" {
		result.VersionID = item.VersionID
	}
	result.ReportType = item.ReportType
	result.ReportStatus = item.Status
	if result.AuditRef == "" || result.RequestID == "" || result.SessionID == "" {
		var metadata map[string]any
		if err := json.Unmarshal(item.MetadataJSON, &metadata); err == nil {
			if result.RequestID == "" {
				result.RequestID = stringValue(metadata["request_id"])
			}
			if result.SessionID == "" {
				result.SessionID = stringValue(metadata["session_id"])
			}
			if result.AuditRef == "" {
				result.AuditRef = stringValue(metadata["audit_ref"])
			}
			if result.TraceID == "" {
				result.TraceID = stringValue(metadata["trace_id"])
			}
		}
	}
}

func applyTemporalRef(result *Result) {
	workflowID, runID, ok := parseTemporalRef(result.AuditRef)
	if ok {
		result.Temporal = &TemporalRef{
			WorkflowID: workflowID,
			RunID:      runID,
		}
		return
	}
	if result.AuditRef != "" {
		result.Warnings = append(result.Warnings, "audit_ref does not point at a Temporal workflow run")
	}
}

func addTraceWarning(result *Result) {
	if result.TraceID == "" {
		result.Warnings = append(result.Warnings, "trace_id is unavailable in the current skeleton")
	}
}

func parseTemporalRef(auditRef string) (string, string, bool) {
	const prefix = "temporal:workflow:"
	if !strings.HasPrefix(auditRef, prefix) {
		return "", "", false
	}
	payload := strings.TrimPrefix(auditRef, prefix)
	parts := strings.SplitN(payload, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}

	return parts[0], parts[1], true
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}
