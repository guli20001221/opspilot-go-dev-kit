package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
)

type createEvalCaseRequest struct {
	TenantID     string `json:"tenant_id"`
	SourceCaseID string `json:"source_case_id"`
	OperatorNote string `json:"operator_note,omitempty"`
	CreatedBy    string `json:"created_by,omitempty"`
}

type evalCaseResponse struct {
	EvalCaseID     string `json:"eval_case_id"`
	TenantID       string `json:"tenant_id"`
	SourceCaseID   string `json:"source_case_id"`
	SourceTaskID   string `json:"source_task_id,omitempty"`
	SourceReportID string `json:"source_report_id,omitempty"`
	TraceID        string `json:"trace_id,omitempty"`
	VersionID      string `json:"version_id,omitempty"`
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	OperatorNote   string `json:"operator_note,omitempty"`
	CreatedBy      string `json:"created_by"`
	CreatedAt      string `json:"created_at"`
}

func (a *appHandler) handleEvalCases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		a.handleCreateEvalCase(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *appHandler) handleCreateEvalCase(w http.ResponseWriter, r *http.Request) {
	var req createEvalCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.SourceCaseID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_eval_case", "tenant_id and source_case_id are required")
		return
	}

	item, created, err := a.evalCases.PromoteCase(r.Context(), evalsvc.CreateInput{
		TenantID:     strings.TrimSpace(req.TenantID),
		SourceCaseID: strings.TrimSpace(req.SourceCaseID),
		OperatorNote: req.OperatorNote,
		CreatedBy:    req.CreatedBy,
	})
	if err != nil {
		switch {
		case errors.Is(err, evalsvc.ErrInvalidSource):
			writeError(w, http.StatusConflict, "invalid_eval_case_source", "source case does not belong to tenant scope")
		case errors.Is(err, casesvc.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		default:
			writeError(w, http.StatusInternalServerError, "eval_case_create_failed", err.Error())
		}
		return
	}

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	writeJSON(w, status, newEvalCaseResponse(item))
}

func (a *appHandler) handleEvalCaseByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	evalCaseID := strings.TrimPrefix(r.URL.Path, "/api/v1/eval-cases/")
	if evalCaseID == "" || strings.Contains(evalCaseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}

	item, err := a.evalCases.GetEvalCase(r.Context(), evalCaseID)
	if err != nil {
		if errors.Is(err, evalsvc.ErrEvalCaseNotFound) {
			writeError(w, http.StatusNotFound, "eval_case_not_found", "eval case not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "eval_case_lookup_failed", err.Error())
		return
	}
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_case_not_found", "eval case not found")
		return
	}

	writeJSON(w, http.StatusOK, newEvalCaseResponse(item))
}

func newEvalCaseResponse(item evalsvc.EvalCase) evalCaseResponse {
	return evalCaseResponse{
		EvalCaseID:     item.ID,
		TenantID:       item.TenantID,
		SourceCaseID:   item.SourceCaseID,
		SourceTaskID:   item.SourceTaskID,
		SourceReportID: item.SourceReportID,
		TraceID:        item.TraceID,
		VersionID:      item.VersionID,
		Title:          item.Title,
		Summary:        item.Summary,
		OperatorNote:   item.OperatorNote,
		CreatedBy:      item.CreatedBy,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339Nano),
	}
}
