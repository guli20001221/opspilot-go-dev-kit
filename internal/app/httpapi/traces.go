package httpapi

import (
	"errors"
	"net/http"

	cases "opspilot-go/internal/case"
	"opspilot-go/internal/observability/tracedetail"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

type traceDrilldownSubjectResponse struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	TenantID string `json:"tenant_id,omitempty"`
}

type traceDrilldownLineageResponse struct {
	TaskID   string `json:"task_id,omitempty"`
	ReportID string `json:"report_id,omitempty"`
	CaseID   string `json:"case_id,omitempty"`
}

type traceDrilldownTemporalResponse struct {
	WorkflowID string `json:"workflow_id"`
	RunID      string `json:"run_id"`
}

type traceDrilldownResponse struct {
	Subject      traceDrilldownSubjectResponse   `json:"subject"`
	Lineage      traceDrilldownLineageResponse   `json:"lineage"`
	RequestID    string                          `json:"request_id,omitempty"`
	SessionID    string                          `json:"session_id,omitempty"`
	TraceID      string                          `json:"trace_id,omitempty"`
	AuditRef     string                          `json:"audit_ref,omitempty"`
	TaskStatus   string                          `json:"task_status,omitempty"`
	ReportType   string                          `json:"report_type,omitempty"`
	ReportStatus string                          `json:"report_status,omitempty"`
	CaseStatus   string                          `json:"case_status,omitempty"`
	Temporal     *traceDrilldownTemporalResponse `json:"temporal,omitempty"`
	Warnings     []string                        `json:"warnings,omitempty"`
}

func (a *appHandler) handleTraceDrilldown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	result, err := a.traceDetails.Lookup(r.Context(), tracedetail.LookupInput{
		TaskID:   r.URL.Query().Get("task_id"),
		ReportID: r.URL.Query().Get("report_id"),
		CaseID:   r.URL.Query().Get("case_id"),
	})
	if err != nil {
		switch {
		case errors.Is(err, tracedetail.ErrInvalidLookup):
			writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		case errors.Is(err, workflow.ErrTaskNotFound), errors.Is(err, report.ErrReportNotFound), errors.Is(err, cases.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "trace_subject_not_found", "trace subject not found")
		default:
			writeError(w, http.StatusInternalServerError, "trace_lookup_failed", err.Error())
		}
		return
	}

	resp := traceDrilldownResponse{
		Subject: traceDrilldownSubjectResponse{
			Kind:     result.Subject.Kind,
			ID:       result.Subject.ID,
			TenantID: result.Subject.TenantID,
		},
		Lineage: traceDrilldownLineageResponse{
			TaskID:   result.Lineage.TaskID,
			ReportID: result.Lineage.ReportID,
			CaseID:   result.Lineage.CaseID,
		},
		RequestID:    result.RequestID,
		SessionID:    result.SessionID,
		TraceID:      result.TraceID,
		AuditRef:     result.AuditRef,
		TaskStatus:   result.TaskStatus,
		ReportType:   result.ReportType,
		ReportStatus: result.ReportStatus,
		CaseStatus:   result.CaseStatus,
		Warnings:     result.Warnings,
	}
	if result.Temporal != nil {
		resp.Temporal = &traceDrilldownTemporalResponse{
			WorkflowID: result.Temporal.WorkflowID,
			RunID:      result.Temporal.RunID,
		}
	}

	writeJSON(w, http.StatusOK, resp)
}
