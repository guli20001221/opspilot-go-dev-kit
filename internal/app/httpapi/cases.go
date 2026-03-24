package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
	AssignedTo     string `json:"assigned_to,omitempty"`
	AssignedAt     string `json:"assigned_at,omitempty"`
	ClosedBy       string `json:"closed_by,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
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

func (a *appHandler) handleListCases(w http.ResponseWriter, r *http.Request) {
	filter, err := parseCaseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
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

	writeJSON(w, http.StatusOK, newCaseResponse(item))
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

func parseCaseListFilter(r *http.Request) (casesvc.ListFilter, error) {
	filter := casesvc.ListFilter{
		TenantID:       r.URL.Query().Get("tenant_id"),
		Status:         r.URL.Query().Get("status"),
		SourceTaskID:   r.URL.Query().Get("source_task_id"),
		SourceReportID: r.URL.Query().Get("source_report_id"),
		Limit:          20,
	}
	if strings.TrimSpace(filter.TenantID) == "" {
		return casesvc.ListFilter{}, errors.New("tenant_id is required")
	}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return casesvc.ListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return casesvc.ListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	return filter, nil
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
		AssignedTo:     item.AssignedTo,
		AssignedAt:     formatOptionalTime(item.AssignedAt),
		ClosedBy:       item.ClosedBy,
		CreatedAt:      item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:      item.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func formatOptionalTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}

	return value.Format(time.RFC3339Nano)
}
