package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	casesvc "opspilot-go/internal/case"
	evalsvc "opspilot-go/internal/eval"
	"opspilot-go/internal/report"
	"opspilot-go/internal/workflow"
)

type createCaseRequest struct {
	TenantID           string                    `json:"tenant_id"`
	Title              string                    `json:"title"`
	Summary            string                    `json:"summary"`
	SourceTaskID       string                    `json:"source_task_id,omitempty"`
	SourceReportID     string                    `json:"source_report_id,omitempty"`
	SourceEvalReportID string                    `json:"source_eval_report_id,omitempty"`
	SourceEvalCaseID   string                    `json:"source_eval_case_id,omitempty"`
	SourceEvalRunID    string                    `json:"source_eval_run_id,omitempty"`
	CompareOrigin      *caseCompareOriginRequest `json:"compare_origin,omitempty"`
	CreatedBy          string                    `json:"created_by,omitempty"`
}

type caseResponse struct {
	CaseID                          string                             `json:"case_id"`
	TenantID                        string                             `json:"tenant_id"`
	Status                          string                             `json:"status"`
	Title                           string                             `json:"title"`
	Summary                         string                             `json:"summary"`
	SourceTaskID                    string                             `json:"source_task_id,omitempty"`
	SourceReportID                  string                             `json:"source_report_id,omitempty"`
	SourceEvalReportID              string                             `json:"source_eval_report_id,omitempty"`
	SourceEvalCaseID                string                             `json:"source_eval_case_id,omitempty"`
	SourceEvalRunID                 string                             `json:"source_eval_run_id,omitempty"`
	CompareOrigin                   *caseCompareOriginResponse         `json:"compare_origin,omitempty"`
	PreferredSourceTaskProvenance   caseSourceTaskProvenanceResponse   `json:"preferred_source_task_provenance"`
	PreferredSourceReportProvenance caseSourceReportProvenanceResponse `json:"preferred_source_report_provenance"`
	PreferredEvalReportProvenance   caseEvalReportProvenanceResponse   `json:"preferred_eval_report_provenance"`
	PreferredEvalCaseProvenance     caseEvalCaseProvenanceResponse     `json:"preferred_eval_case_provenance"`
	PreferredEvalRunProvenance      caseEvalRunProvenanceResponse      `json:"preferred_eval_run_provenance"`
	PreferredTraceDetailAction      caseTraceDetailActionResponse      `json:"preferred_trace_detail_action"`
	PreferredCompareAction          caseCompareActionResponse          `json:"preferred_compare_action"`
	CreatedBy                       string                             `json:"created_by"`
	AssignedTo                      string                             `json:"assigned_to,omitempty"`
	AssignedAt                      string                             `json:"assigned_at,omitempty"`
	ClosedBy                        string                             `json:"closed_by,omitempty"`
	Notes                           []caseNoteResponse                 `json:"notes,omitempty"`
	CreatedAt                       string                             `json:"created_at"`
	UpdatedAt                       string                             `json:"updated_at"`
}

type caseNoteResponse struct {
	NoteID    string `json:"note_id"`
	CaseID    string `json:"case_id"`
	TenantID  string `json:"tenant_id"`
	Body      string `json:"body"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type listCasesResponse struct {
	Cases      []caseResponse `json:"cases"`
	HasMore    bool           `json:"has_more"`
	NextOffset *int           `json:"next_offset,omitempty"`
}

type closeCaseRequest struct {
	ClosedBy string `json:"closed_by,omitempty"`
}

type assignCaseRequest struct {
	AssignedTo string `json:"assigned_to,omitempty"`
}

type unassignCaseRequest struct {
	UnassignedBy string `json:"unassigned_by,omitempty"`
}

type reopenCaseRequest struct {
	ReopenedBy string `json:"reopened_by,omitempty"`
}

type createCaseNoteRequest struct {
	Body      string `json:"body"`
	CreatedBy string `json:"created_by,omitempty"`
}

type caseCompareOriginRequest struct {
	LeftEvalReportID  string `json:"left_eval_report_id"`
	RightEvalReportID string `json:"right_eval_report_id"`
	SelectedSide      string `json:"selected_side"`
}

type caseCompareOriginResponse struct {
	LeftEvalReportID  string `json:"left_eval_report_id"`
	RightEvalReportID string `json:"right_eval_report_id"`
	SelectedSide      string `json:"selected_side"`
}

type caseSourceTaskProvenanceResponse struct {
	Mode   string `json:"mode"`
	TaskID string `json:"task_id,omitempty"`
}

type caseSourceReportProvenanceResponse struct {
	Mode     string `json:"mode"`
	ReportID string `json:"report_id,omitempty"`
}

type caseEvalReportProvenanceResponse struct {
	Mode     string `json:"mode"`
	ReportID string `json:"report_id,omitempty"`
}

type caseEvalCaseProvenanceResponse struct {
	Mode       string `json:"mode"`
	EvalCaseID string `json:"eval_case_id,omitempty"`
}

type caseEvalRunProvenanceResponse struct {
	Mode  string `json:"mode"`
	RunID string `json:"run_id,omitempty"`
}

type caseTraceDetailActionResponse struct {
	Mode   string `json:"mode"`
	CaseID string `json:"case_id,omitempty"`
}

type caseCompareActionResponse struct {
	Mode              string `json:"mode"`
	LeftEvalReportID  string `json:"left_eval_report_id,omitempty"`
	RightEvalReportID string `json:"right_eval_report_id,omitempty"`
}

func (a *appHandler) handleCases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListCases(w, r)
	case http.MethodPost:
		a.handleCreateCase(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *appHandler) handleCreateCase(w http.ResponseWriter, r *http.Request) {
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
	if req.SourceEvalCaseID != "" {
		existing, ok, err := a.cases.FindOpenCaseBySourceEvalCase(r.Context(), req.TenantID, req.SourceEvalCaseID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "case_lookup_failed", err.Error())
			return
		}
		if ok {
			writeJSON(w, http.StatusOK, newCaseResponse(existing))
			return
		}
	}
	if req.SourceEvalReportID != "" && req.CompareOrigin != nil {
		existing, ok, err := a.cases.FindOpenCaseByCompareOrigin(r.Context(), req.TenantID, req.SourceEvalReportID, newCaseCompareOriginModel(req.CompareOrigin))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "case_lookup_failed", err.Error())
			return
		}
		if ok {
			writeJSON(w, http.StatusOK, newCaseResponse(existing))
			return
		}
	}
	if req.SourceEvalReportID != "" && req.SourceEvalCaseID == "" && req.CompareOrigin == nil {
		existing, ok, err := a.cases.FindOpenCaseBySourceEvalReport(r.Context(), req.TenantID, req.SourceEvalReportID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "case_lookup_failed", err.Error())
			return
		}
		if ok {
			writeJSON(w, http.StatusOK, newCaseResponse(existing))
			return
		}
	}

	item, created, err := a.cases.CreateCaseWithOutcome(r.Context(), casesvc.CreateInput{
		TenantID:           req.TenantID,
		Title:              req.Title,
		Summary:            req.Summary,
		SourceTaskID:       req.SourceTaskID,
		SourceReportID:     req.SourceReportID,
		SourceEvalReportID: req.SourceEvalReportID,
		SourceEvalCaseID:   req.SourceEvalCaseID,
		SourceEvalRunID:    req.SourceEvalRunID,
		CompareOrigin:      newCaseCompareOriginModel(req.CompareOrigin),
		CreatedBy:          req.CreatedBy,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "case_create_failed", err.Error())
		return
	}

	statusCode := http.StatusCreated
	if !created {
		statusCode = http.StatusOK
	}
	writeJSON(w, statusCode, newCaseResponse(item))
}

func (a *appHandler) handleListCases(w http.ResponseWriter, r *http.Request) {
	filter, sourceEvalDatasetID, err := parseCaseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	if sourceEvalDatasetID != "" {
		reportIDs, err := a.collectEvalReportIDsForDataset(r.Context(), filter.TenantID, sourceEvalDatasetID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "case_list_failed", err.Error())
			return
		}
		if len(reportIDs) == 0 {
			writeJSON(w, http.StatusOK, listCasesResponse{Cases: []caseResponse{}, HasMore: false})
			return
		}
		filter.SourceEvalReportIDs = reportIDs
	}

	page, err := a.cases.ListCases(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "case_list_failed", err.Error())
		return
	}

	resp := listCasesResponse{
		Cases:   make([]caseResponse, 0, len(page.Cases)),
		HasMore: page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.Cases {
		resp.Cases = append(resp.Cases, newCaseResponse(item))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleCaseByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/v1/cases/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if strings.HasSuffix(path, "/close") {
		caseID := strings.TrimSuffix(path, "/close")
		caseID = strings.TrimSuffix(caseID, "/")
		a.handleCloseCase(w, r, caseID)
		return
	}
	if strings.HasSuffix(path, "/assign") {
		caseID := strings.TrimSuffix(path, "/assign")
		caseID = strings.TrimSuffix(caseID, "/")
		a.handleAssignCase(w, r, caseID)
		return
	}
	if strings.HasSuffix(path, "/unassign") {
		caseID := strings.TrimSuffix(path, "/unassign")
		caseID = strings.TrimSuffix(caseID, "/")
		a.handleUnassignCase(w, r, caseID)
		return
	}
	if strings.HasSuffix(path, "/reopen") {
		caseID := strings.TrimSuffix(path, "/reopen")
		caseID = strings.TrimSuffix(caseID, "/")
		a.handleReopenCase(w, r, caseID)
		return
	}
	if strings.HasSuffix(path, "/notes") {
		caseID := strings.TrimSuffix(path, "/notes")
		caseID = strings.TrimSuffix(caseID, "/")
		a.handleAddCaseNote(w, r, caseID)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	caseID := path
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		return
	}

	notes, err := a.cases.ListCaseNotes(r.Context(), caseID, 20)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "case_notes_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, newCaseResponse(item, notes))
}

func (a *appHandler) handleCloseCase(w http.ResponseWriter, r *http.Request, caseID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		return
	}

	var req closeCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	closed, err := a.cases.CloseCase(r.Context(), caseID, req.ClosedBy)
	if err != nil {
		switch {
		case errors.Is(err, casesvc.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		case errors.Is(err, casesvc.ErrInvalidCaseState):
			writeError(w, http.StatusConflict, "invalid_case_state", "case is not in a valid state for close")
		default:
			writeError(w, http.StatusInternalServerError, "case_close_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newCaseResponse(closed))
}

func (a *appHandler) handleAssignCase(w http.ResponseWriter, r *http.Request, caseID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		return
	}

	var req assignCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.AssignedTo) == "" {
		writeError(w, http.StatusBadRequest, "invalid_case", "assigned_to is required")
		return
	}

	assigned, err := a.cases.AssignCase(r.Context(), item, req.AssignedTo)
	if err != nil {
		switch {
		case errors.Is(err, casesvc.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		case errors.Is(err, casesvc.ErrCaseConflict):
			writeError(w, http.StatusConflict, "case_conflict", "case assignment is stale; reload and retry")
		case errors.Is(err, casesvc.ErrInvalidCaseState):
			writeError(w, http.StatusConflict, "invalid_case_state", "case is not in a valid state for assign")
		default:
			writeError(w, http.StatusInternalServerError, "case_assign_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newCaseResponse(assigned))
}

func (a *appHandler) handleUnassignCase(w http.ResponseWriter, r *http.Request, caseID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		return
	}

	var req unassignCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	unassigned, err := a.cases.UnassignCase(r.Context(), item, req.UnassignedBy)
	if err != nil {
		switch {
		case errors.Is(err, casesvc.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		case errors.Is(err, casesvc.ErrCaseConflict):
			writeError(w, http.StatusConflict, "case_conflict", "case assignment is stale; reload and retry")
		case errors.Is(err, casesvc.ErrInvalidCaseState):
			writeError(w, http.StatusConflict, "invalid_case_state", "case is not in a valid state for unassign")
		default:
			writeError(w, http.StatusInternalServerError, "case_unassign_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newCaseResponse(unassigned))
}

func (a *appHandler) handleReopenCase(w http.ResponseWriter, r *http.Request, caseID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		return
	}

	var req reopenCaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	reopened, err := a.cases.ReopenCase(r.Context(), caseID, req.ReopenedBy)
	if err != nil {
		switch {
		case errors.Is(err, casesvc.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		case errors.Is(err, casesvc.ErrInvalidCaseState):
			writeError(w, http.StatusConflict, "invalid_case_state", "case is not in a valid state for reopen")
		default:
			writeError(w, http.StatusInternalServerError, "case_reopen_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newCaseResponse(reopened))
}

func (a *appHandler) handleAddCaseNote(w http.ResponseWriter, r *http.Request, caseID string) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if caseID == "" || strings.Contains(caseID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		return
	}

	var req createCaseNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}

	note, err := a.cases.AddNote(r.Context(), item, req.Body, req.CreatedBy)
	if err != nil {
		switch {
		case errors.Is(err, casesvc.ErrInvalidNote):
			writeError(w, http.StatusBadRequest, "invalid_note", "body is required")
		case errors.Is(err, casesvc.ErrCaseNotFound):
			writeError(w, http.StatusNotFound, "case_not_found", "case not found")
		default:
			writeError(w, http.StatusInternalServerError, "case_note_create_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, newCaseNoteResponse(note))
}

func validateCreateCaseRequest(req createCaseRequest) error {
	switch {
	case req.TenantID == "":
		return errors.New("tenant_id is required")
	case req.Title == "":
		return errors.New("title is required")
	case req.CompareOrigin != nil:
		if req.CompareOrigin.LeftEvalReportID == "" || req.CompareOrigin.RightEvalReportID == "" {
			return errors.New("compare_origin.left_eval_report_id and compare_origin.right_eval_report_id are required")
		}
		if req.SourceTaskID != "" || req.SourceReportID != "" || req.SourceEvalCaseID != "" || req.SourceEvalRunID != "" {
			return errors.New("compare_origin cannot be combined with source_task_id, source_report_id, source_eval_case_id, or source_eval_run_id")
		}
		if req.CompareOrigin.SelectedSide != "left" && req.CompareOrigin.SelectedSide != "right" {
			return errors.New("compare_origin.selected_side must be left or right")
		}
		expectedSourceEvalReportID := req.CompareOrigin.LeftEvalReportID
		if req.CompareOrigin.SelectedSide == "right" {
			expectedSourceEvalReportID = req.CompareOrigin.RightEvalReportID
		}
		if req.SourceEvalReportID != expectedSourceEvalReportID {
			return errors.New("source_eval_report_id must match compare_origin.selected_side")
		}
		return nil
	default:
		return nil
	}
}

func parseCaseListFilter(r *http.Request) (casesvc.ListFilter, string, error) {
	filter := casesvc.ListFilter{
		TenantID:           r.URL.Query().Get("tenant_id"),
		Status:             r.URL.Query().Get("status"),
		AssignedTo:         r.URL.Query().Get("assigned_to"),
		UnassignedOnly:     false,
		EvalBackedOnly:     false,
		RunBackedOnly:      false,
		CompareOriginOnly:  false,
		SourceTaskID:       r.URL.Query().Get("source_task_id"),
		SourceReportID:     r.URL.Query().Get("source_report_id"),
		SourceEvalReportID: r.URL.Query().Get("source_eval_report_id"),
		SourceEvalCaseID:   r.URL.Query().Get("source_eval_case_id"),
		SourceEvalRunID:    r.URL.Query().Get("source_eval_run_id"),
		Limit:              20,
	}
	sourceEvalDatasetID := strings.TrimSpace(r.URL.Query().Get("source_eval_dataset_id"))
	if strings.TrimSpace(filter.TenantID) == "" {
		return casesvc.ListFilter{}, "", errors.New("tenant_id is required")
	}
	if rawUnassignedOnly := r.URL.Query().Get("unassigned_only"); rawUnassignedOnly != "" {
		value, err := strconv.ParseBool(rawUnassignedOnly)
		if err != nil {
			return casesvc.ListFilter{}, "", fmt.Errorf("unassigned_only must be a boolean")
		}
		filter.UnassignedOnly = value
	}
	if rawEvalBackedOnly := r.URL.Query().Get("eval_backed_only"); rawEvalBackedOnly != "" {
		value, err := strconv.ParseBool(rawEvalBackedOnly)
		if err != nil {
			return casesvc.ListFilter{}, "", fmt.Errorf("eval_backed_only must be a boolean")
		}
		filter.EvalBackedOnly = value
	}
	if rawRunBackedOnly := r.URL.Query().Get("run_backed_only"); rawRunBackedOnly != "" {
		value, err := strconv.ParseBool(rawRunBackedOnly)
		if err != nil {
			return casesvc.ListFilter{}, "", fmt.Errorf("run_backed_only must be a boolean")
		}
		filter.RunBackedOnly = value
	}
	if rawCompareOriginOnly := r.URL.Query().Get("compare_origin_only"); rawCompareOriginOnly != "" {
		value, err := strconv.ParseBool(rawCompareOriginOnly)
		if err != nil {
			return casesvc.ListFilter{}, "", fmt.Errorf("compare_origin_only must be a boolean")
		}
		filter.CompareOriginOnly = value
	}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return casesvc.ListFilter{}, "", errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return casesvc.ListFilter{}, "", errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	return filter, sourceEvalDatasetID, nil
}

func (a *appHandler) collectEvalReportIDsForDataset(ctx context.Context, tenantID string, datasetID string) ([]string, error) {
	if a.evalReports == nil || datasetID == "" {
		return nil, nil
	}

	filter := evalsvc.EvalReportListFilter{
		TenantID:  tenantID,
		DatasetID: datasetID,
		Limit:     100,
	}
	reportIDs := make([]string, 0, 8)
	seen := make(map[string]struct{})
	for {
		page, err := a.evalReports.ListEvalReports(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("list eval reports for dataset %q: %w", datasetID, err)
		}
		for _, item := range page.Reports {
			if item.ID == "" {
				continue
			}
			if _, ok := seen[item.ID]; ok {
				continue
			}
			seen[item.ID] = struct{}{}
			reportIDs = append(reportIDs, item.ID)
		}
		if !page.HasMore {
			break
		}
		filter.Offset = page.NextOffset
	}

	return reportIDs, nil
}

func (a *appHandler) validateCaseSources(r *http.Request, req createCaseRequest) (report.Report, error) {
	reportItem := evalsvc.EvalReport{}

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

	if req.SourceEvalReportID != "" {
		item, err := a.evalReports.GetEvalReport(r.Context(), req.SourceEvalReportID)
		if err != nil {
			return report.Report{}, err
		}
		if item.TenantID != req.TenantID {
			return report.Report{}, errInvalidCaseSource
		}
		reportItem = item
	}
	if req.SourceEvalCaseID != "" {
		item, err := a.evalCases.GetEvalCase(r.Context(), req.SourceEvalCaseID)
		if err != nil {
			return report.Report{}, err
		}
		if item.TenantID != req.TenantID {
			return report.Report{}, errInvalidCaseSource
		}
		if req.SourceEvalReportID != "" && !evalReportContainsBadCase(req.SourceEvalCaseID, reportItem) {
			return report.Report{}, errInvalidEvalCaseSource
		}
	}
	if req.SourceEvalRunID != "" {
		item, err := a.evalRuns.GetRun(r.Context(), req.SourceEvalRunID)
		if err != nil {
			return report.Report{}, err
		}
		if item.TenantID != req.TenantID {
			return report.Report{}, errInvalidCaseSource
		}
	}
	if req.CompareOrigin != nil {
		for _, reportID := range []string{req.CompareOrigin.LeftEvalReportID, req.CompareOrigin.RightEvalReportID} {
			item, err := a.evalReports.GetEvalReport(r.Context(), reportID)
			if err != nil {
				return report.Report{}, err
			}
			if item.TenantID != req.TenantID {
				return report.Report{}, errInvalidCaseSource
			}
		}
	}

	return report.Report{}, nil
}

var errInvalidCaseSource = errors.New("case source tenant mismatch")
var errInvalidEvalCaseSource = errors.New("source eval case does not belong to source eval report")

func writeCaseSourceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, workflow.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "task_not_found", "task not found")
	case errors.Is(err, report.ErrReportNotFound):
		writeError(w, http.StatusNotFound, "report_not_found", "report not found")
	case errors.Is(err, evalsvc.ErrEvalReportNotFound):
		writeError(w, http.StatusNotFound, "eval_report_not_found", "eval report not found")
	case errors.Is(err, evalsvc.ErrEvalCaseNotFound):
		writeError(w, http.StatusNotFound, "eval_case_not_found", "eval case not found")
	case errors.Is(err, evalsvc.ErrEvalRunNotFound):
		writeError(w, http.StatusNotFound, "eval_run_not_found", "eval run not found")
	case errors.Is(err, errInvalidCaseSource):
		writeError(w, http.StatusConflict, "invalid_case_source", err.Error())
	case errors.Is(err, errInvalidEvalCaseSource):
		writeError(w, http.StatusConflict, "invalid_case_source", "source eval case does not belong to source eval report")
	default:
		writeError(w, http.StatusInternalServerError, "case_source_lookup_failed", err.Error())
	}
}

func newCaseResponse(item casesvc.Case, notes ...[]casesvc.Note) caseResponse {
	resp := caseResponse{
		CaseID:                          item.ID,
		TenantID:                        item.TenantID,
		Status:                          item.Status,
		Title:                           item.Title,
		Summary:                         item.Summary,
		SourceTaskID:                    item.SourceTaskID,
		SourceReportID:                  item.SourceReportID,
		SourceEvalReportID:              item.SourceEvalReportID,
		SourceEvalCaseID:                item.SourceEvalCaseID,
		SourceEvalRunID:                 item.SourceEvalRunID,
		CompareOrigin:                   newCaseCompareOriginResponse(item.CompareOrigin),
		PreferredSourceTaskProvenance:   newCaseSourceTaskProvenance(item.SourceTaskID),
		PreferredSourceReportProvenance: newCaseSourceReportProvenance(item.SourceReportID),
		PreferredEvalReportProvenance:   newCaseEvalReportProvenance(item.SourceEvalReportID),
		PreferredEvalCaseProvenance:     newCaseEvalCaseProvenance(item.SourceEvalCaseID),
		PreferredEvalRunProvenance:      newCaseEvalRunProvenance(item.SourceEvalRunID),
		PreferredTraceDetailAction:      newCaseTraceDetailAction(item.ID),
		PreferredCompareAction:          newCaseCompareAction(item.CompareOrigin),
		CreatedBy:                       item.CreatedBy,
		AssignedTo:                      item.AssignedTo,
		AssignedAt:                      formatOptionalTime(item.AssignedAt),
		ClosedBy:                        item.ClosedBy,
		CreatedAt:                       item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                       item.UpdatedAt.Format(time.RFC3339Nano),
	}
	if len(notes) > 0 && len(notes[0]) > 0 {
		resp.Notes = make([]caseNoteResponse, 0, len(notes[0]))
		for _, note := range notes[0] {
			resp.Notes = append(resp.Notes, newCaseNoteResponse(note))
		}
	}

	return resp
}

func evalReportContainsBadCase(evalCaseID string, item evalsvc.EvalReport) bool {
	for _, badCase := range item.BadCases {
		if badCase.EvalCaseID == evalCaseID {
			return true
		}
	}

	return false
}

func newCaseCompareOriginModel(req *caseCompareOriginRequest) casesvc.CompareOrigin {
	if req == nil {
		return casesvc.CompareOrigin{}
	}

	return casesvc.CompareOrigin{
		LeftEvalReportID:  req.LeftEvalReportID,
		RightEvalReportID: req.RightEvalReportID,
		SelectedSide:      req.SelectedSide,
	}
}

func newCaseCompareOriginResponse(item casesvc.CompareOrigin) *caseCompareOriginResponse {
	if item.LeftEvalReportID == "" && item.RightEvalReportID == "" && item.SelectedSide == "" {
		return nil
	}

	return &caseCompareOriginResponse{
		LeftEvalReportID:  item.LeftEvalReportID,
		RightEvalReportID: item.RightEvalReportID,
		SelectedSide:      item.SelectedSide,
	}
}

func newCaseSourceTaskProvenance(taskID string) caseSourceTaskProvenanceResponse {
	if taskID != "" {
		return caseSourceTaskProvenanceResponse{Mode: "open", TaskID: taskID}
	}
	return caseSourceTaskProvenanceResponse{Mode: "none"}
}

func newCaseSourceReportProvenance(reportID string) caseSourceReportProvenanceResponse {
	if reportID != "" {
		return caseSourceReportProvenanceResponse{Mode: "open", ReportID: reportID}
	}
	return caseSourceReportProvenanceResponse{Mode: "none"}
}

func newCaseEvalReportProvenance(reportID string) caseEvalReportProvenanceResponse {
	if reportID != "" {
		return caseEvalReportProvenanceResponse{Mode: "open", ReportID: reportID}
	}
	return caseEvalReportProvenanceResponse{Mode: "none"}
}

func newCaseEvalCaseProvenance(evalCaseID string) caseEvalCaseProvenanceResponse {
	if evalCaseID != "" {
		return caseEvalCaseProvenanceResponse{Mode: "open", EvalCaseID: evalCaseID}
	}
	return caseEvalCaseProvenanceResponse{Mode: "none"}
}

func newCaseEvalRunProvenance(runID string) caseEvalRunProvenanceResponse {
	if runID != "" {
		return caseEvalRunProvenanceResponse{Mode: "open", RunID: runID}
	}
	return caseEvalRunProvenanceResponse{Mode: "none"}
}

func newCaseTraceDetailAction(caseID string) caseTraceDetailActionResponse {
	if caseID != "" {
		return caseTraceDetailActionResponse{Mode: "open", CaseID: caseID}
	}
	return caseTraceDetailActionResponse{Mode: "none"}
}

func newCaseCompareAction(origin casesvc.CompareOrigin) caseCompareActionResponse {
	if origin.LeftEvalReportID != "" && origin.RightEvalReportID != "" {
		return caseCompareActionResponse{
			Mode:              "open",
			LeftEvalReportID:  origin.LeftEvalReportID,
			RightEvalReportID: origin.RightEvalReportID,
		}
	}
	return caseCompareActionResponse{Mode: "none"}
}

func newCaseNoteResponse(note casesvc.Note) caseNoteResponse {
	return caseNoteResponse{
		NoteID:    note.ID,
		CaseID:    note.CaseID,
		TenantID:  note.TenantID,
		Body:      note.Body,
		CreatedBy: note.CreatedBy,
		CreatedAt: note.CreatedAt.Format(time.RFC3339Nano),
	}
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(time.RFC3339Nano)
}
