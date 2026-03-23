package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	casesvc "opspilot-go/internal/case"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

type createCaseRequest struct {
	TenantID       string `json:"tenant_id"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	SourceTaskID   string `json:"source_task_id,omitempty"`
	SourceReportID string `json:"source_report_id,omitempty"`
	CreatedBy      string `json:"created_by,omitempty"`
}

type caseResponse struct {
	CaseID         string `json:"case_id"`
	TenantID       string `json:"tenant_id"`
	Status         string `json:"status"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	SourceTaskID   string `json:"source_task_id,omitempty"`
	SourceReportID string `json:"source_report_id,omitempty"`
	CreatedBy      string `json:"created_by"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func (a *appHandler) handleCases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	var req createCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if err := validateCreateCaseRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_case", err.Error())
		return
	}

	reportItem, err := a.validateCaseSources(r, req)
	if err != nil {
		writeCaseSourceError(w, err)
		return
	}
	if reportItem.ID != "" && req.SourceTaskID != "" && reportItem.SourceTaskID != req.SourceTaskID {
		writeError(w, http.StatusConflict, "invalid_case_source", "source report does not belong to source task")
		return
	}

	item, err := a.cases.CreateCase(r.Context(), casesvc.CreateInput{
		TenantID:       req.TenantID,
		Title:          req.Title,
		Summary:        req.Summary,
		SourceTaskID:   req.SourceTaskID,
		SourceReportID: req.SourceReportID,
		CreatedBy:      req.CreatedBy,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "case_create_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, newCaseResponse(item))
}

func (a *appHandler) handleCaseByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	caseID := strings.TrimPrefix(r.URL.Path, "/api/v1/cases/")
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	item, err := a.cases.GetCase(r.Context(), caseID)
	if err != nil {
		if errors.Is(err, casesvc.ErrCaseNotFound) {
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "case_lookup_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, newCaseResponse(item))
}

func validateCreateCaseRequest(req createCaseRequest) error {
	switch {
	case req.TenantID == "":
		return errors.New("tenant_id is required")
	case req.Title == "":
		return errors.New("title is required")
	default:
		return nil
	}
}

func (a *appHandler) validateCaseSources(r *http.Request, req createCaseRequest) (report.Report, error) {
	if req.SourceTaskID != "" {
		task, err := a.workflows.GetTask(r.Context(), req.SourceTaskID)
		if err != nil {
			return report.Report{}, err
		}
		if task.TenantID != req.TenantID {
			return report.Report{}, errInvalidCaseSource
		}
	}

	if req.SourceReportID != "" {
		item, err := a.reports.GetReport(r.Context(), req.SourceReportID)
		if err != nil {
			return report.Report{}, err
		}
		if item.TenantID != req.TenantID {
			return report.Report{}, errInvalidCaseSource
		}
		return item, nil
	}

	return report.Report{}, nil
}

var errInvalidCaseSource = errors.New("case source tenant mismatch")

func writeCaseSourceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workflow.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task_not_found", "task not found")
	case errors.Is(err, report.ErrReportNotFound):
		writeError(w, http.StatusNotFound, "report_not_found", "report not found")
	case errors.Is(err, errInvalidCaseSource):
		writeError(w, http.StatusConflict, "invalid_case_source", err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "case_source_lookup_failed", err.Error())
	}
}

func newCaseResponse(item casesvc.Case) caseResponse {
	return caseResponse{
		CaseID:         item.ID,
		TenantID:       item.TenantID,
		Status:         item.Status,
		Title:          item.Title,
		Summary:        item.Summary,
		SourceTaskID:   item.SourceTaskID,
		SourceReportID: item.SourceReportID,
		CreatedBy:      item.CreatedBy,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339Nano),
	}
}
