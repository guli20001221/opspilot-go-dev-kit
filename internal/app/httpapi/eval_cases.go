package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
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
	EvalCaseID                      string                                          `json:"eval_case_id"`
	TenantID                        string                                          `json:"tenant_id"`
	SourceCaseID                    string                                          `json:"source_case_id"`
	SourceTaskID                    string                                          `json:"source_task_id,omitempty"`
	SourceReportID                  string                                          `json:"source_report_id,omitempty"`
	FollowUpCaseCount               int                                             `json:"follow_up_case_count"`
	OpenFollowUpCaseCount           int                                             `json:"open_follow_up_case_count"`
	LatestFollowUpCaseID            string                                          `json:"latest_follow_up_case_id,omitempty"`
	LatestFollowUpCaseStatus        string                                          `json:"latest_follow_up_case_status,omitempty"`
	LinkedCaseSummary               evalReportLinkedCaseSummaryResponse             `json:"linked_case_summary"`
	PreferredFollowUpAction         evalCaseFollowUpActionResponse                  `json:"preferred_follow_up_action"`
	PreferredPrimaryAction          evalCaseFollowUpActionResponse                  `json:"preferred_primary_action"`
	PreferredLinkedCaseAction       evalCaseFollowUpActionResponse                  `json:"preferred_linked_case_action"`
	PreferredSourceCaseProvenance   evalReportBadCaseSourceCaseProvenanceResponse   `json:"preferred_source_case_provenance"`
	PreferredSourceReportProvenance evalReportBadCaseSourceReportProvenanceResponse `json:"preferred_source_report_provenance"`
	PreferredSourceTaskProvenance   evalReportBadCaseSourceTaskProvenanceResponse   `json:"preferred_source_task_provenance"`
	PreferredTraceProvenance        evalReportBadCaseTraceProvenanceResponse        `json:"preferred_trace_provenance"`
	PreferredVersionProvenance      evalReportBadCaseVersionProvenanceResponse      `json:"preferred_version_provenance"`
	PreferredFollowUpSliceAction    evalReportBadCaseFollowUpSliceActionResponse    `json:"preferred_follow_up_slice_action"`
	TraceID                         string                                          `json:"trace_id,omitempty"`
	VersionID                       string                                          `json:"version_id,omitempty"`
	Title                           string                                          `json:"title"`
	Summary                         string                                          `json:"summary"`
	OperatorNote                    string                                          `json:"operator_note,omitempty"`
	CreatedBy                       string                                          `json:"created_by"`
	CreatedAt                       string                                          `json:"created_at"`
}

type evalCaseFollowUpActionResponse struct {
	Mode             string `json:"mode"`
	CaseID           string `json:"case_id,omitempty"`
	SourceEvalCaseID string `json:"source_eval_case_id,omitempty"`
}

type listEvalCasesResponse struct {
	EvalCases  []evalCaseResponse `json:"eval_cases"`
	HasMore    bool               `json:"has_more"`
	NextOffset *int               `json:"next_offset,omitempty"`
}

func (a *appHandler) handleEvalCases(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListEvalCases(w, r)
	case http.MethodPost:
		a.handleCreateEvalCase(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *appHandler) handleListEvalCases(w http.ResponseWriter, r *http.Request) {
	filter, err := parseEvalCaseListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.evalCases.ListEvalCases(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_case_list_failed", err.Error())
		return
	}

	resp := listEvalCasesResponse{
		EvalCases: make([]evalCaseResponse, 0, len(page.EvalCases)),
		HasMore:   page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.EvalCases {
		resp.EvalCases = append(resp.EvalCases, newEvalCaseResponse(item))
	}

	writeJSON(w, http.StatusOK, resp)
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
		EvalCaseID:               item.ID,
		TenantID:                 item.TenantID,
		SourceCaseID:             item.SourceCaseID,
		SourceTaskID:             item.SourceTaskID,
		SourceReportID:           item.SourceReportID,
		FollowUpCaseCount:        item.FollowUpCaseCount,
		OpenFollowUpCaseCount:    item.OpenFollowUpCaseCount,
		LatestFollowUpCaseID:     item.LatestFollowUpCaseID,
		LatestFollowUpCaseStatus: item.LatestFollowUpCaseStatus,
		LinkedCaseSummary: evalReportLinkedCaseSummaryResponse{
			TotalCaseCount:   item.FollowUpCaseCount,
			OpenCaseCount:    item.OpenFollowUpCaseCount,
			LatestCaseID:     item.LatestFollowUpCaseID,
			LatestCaseStatus: item.LatestFollowUpCaseStatus,
		},
		PreferredFollowUpAction:         newEvalCaseFollowUpActionResponse(item),
		PreferredPrimaryAction:          newEvalCasePrimaryActionResponse(item),
		PreferredLinkedCaseAction:       newEvalCaseLinkedCaseActionResponse(item),
		PreferredSourceCaseProvenance:   evalCaseSourceCaseProvenance(item.SourceCaseID),
		PreferredSourceReportProvenance: evalCaseSourceReportProvenance(item.SourceReportID),
		PreferredSourceTaskProvenance:   evalCaseSourceTaskProvenance(item.SourceTaskID),
		PreferredTraceProvenance:        evalCaseTraceProvenance(item.TraceID),
		PreferredVersionProvenance:      evalCaseVersionProvenance(item.VersionID),
		PreferredFollowUpSliceAction:    evalCaseFollowUpSliceAction(item.ID),
		TraceID:                         item.TraceID,
		VersionID:                       item.VersionID,
		Title:                           item.Title,
		Summary:                         item.Summary,
		OperatorNote:                    item.OperatorNote,
		CreatedBy:                       item.CreatedBy,
		CreatedAt:                       item.CreatedAt.Format(time.RFC3339Nano),
	}
}

func newEvalCaseFollowUpActionResponse(item evalsvc.EvalCase) evalCaseFollowUpActionResponse {
	return newEvalCaseFollowUpActionResponseFromSummary(item.ID, item.OpenFollowUpCaseCount, item.LatestFollowUpCaseID)
}

func newEvalCaseLinkedCaseActionResponse(item evalsvc.EvalCase) evalCaseFollowUpActionResponse {
	return newEvalCaseLinkedCaseActionResponseFromSummary(
		item.ID,
		item.FollowUpCaseCount,
		item.OpenFollowUpCaseCount,
		item.LatestFollowUpCaseID,
		item.LatestFollowUpCaseStatus,
	)
}

func newEvalCasePrimaryActionResponse(item evalsvc.EvalCase) evalCaseFollowUpActionResponse {
	linkedAction := newEvalCaseLinkedCaseActionResponse(item)
	if linkedAction.Mode != "none" {
		return linkedAction
	}
	return newEvalCaseFollowUpActionResponse(item)
}

func evalCaseSourceCaseProvenance(sourceCaseID string) evalReportBadCaseSourceCaseProvenanceResponse {
	if sourceCaseID != "" {
		return evalReportBadCaseSourceCaseProvenanceResponse{Mode: "open", CaseID: sourceCaseID}
	}
	return evalReportBadCaseSourceCaseProvenanceResponse{Mode: "none"}
}

func evalCaseSourceReportProvenance(sourceReportID string) evalReportBadCaseSourceReportProvenanceResponse {
	if sourceReportID != "" {
		return evalReportBadCaseSourceReportProvenanceResponse{Mode: "open_api", ReportID: sourceReportID}
	}
	return evalReportBadCaseSourceReportProvenanceResponse{Mode: "none"}
}

func evalCaseSourceTaskProvenance(sourceTaskID string) evalReportBadCaseSourceTaskProvenanceResponse {
	if sourceTaskID != "" {
		return evalReportBadCaseSourceTaskProvenanceResponse{Mode: "open_api", TaskID: sourceTaskID}
	}
	return evalReportBadCaseSourceTaskProvenanceResponse{Mode: "none"}
}

func evalCaseTraceProvenance(traceID string) evalReportBadCaseTraceProvenanceResponse {
	if traceID != "" {
		return evalReportBadCaseTraceProvenanceResponse{Mode: "open", TraceID: traceID}
	}
	return evalReportBadCaseTraceProvenanceResponse{Mode: "none"}
}

func evalCaseVersionProvenance(versionID string) evalReportBadCaseVersionProvenanceResponse {
	if versionID != "" {
		return evalReportBadCaseVersionProvenanceResponse{Mode: "open", VersionID: versionID}
	}
	return evalReportBadCaseVersionProvenanceResponse{Mode: "none"}
}

func evalCaseFollowUpSliceAction(evalCaseID string) evalReportBadCaseFollowUpSliceActionResponse {
	if evalCaseID != "" {
		return evalReportBadCaseFollowUpSliceActionResponse{Mode: "open", SourceEvalCaseID: evalCaseID}
	}
	return evalReportBadCaseFollowUpSliceActionResponse{Mode: "none"}
}

func newEvalCaseLinkedCaseActionResponseFromSummary(evalCaseID string, followUpCaseCount int, openFollowUpCaseCount int, latestFollowUpCaseID string, latestFollowUpCaseStatus string) evalCaseFollowUpActionResponse {
	action := evalCaseFollowUpActionResponse{
		Mode:             "none",
		SourceEvalCaseID: evalCaseID,
	}
	if followUpCaseCount <= 0 {
		return action
	}
	if openFollowUpCaseCount > 0 && latestFollowUpCaseID != "" && latestFollowUpCaseStatus == casesvc.StatusOpen {
		action.Mode = "open_existing_case"
		action.CaseID = latestFollowUpCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalCaseFollowUpActionResponseFromSummary(evalCaseID string, openFollowUpCaseCount int, latestFollowUpCaseID string) evalCaseFollowUpActionResponse {
	action := evalCaseFollowUpActionResponse{
		Mode:             "create",
		SourceEvalCaseID: evalCaseID,
	}
	if openFollowUpCaseCount <= 0 {
		return action
	}
	if latestFollowUpCaseID != "" {
		action.Mode = "open_existing_case"
		action.CaseID = latestFollowUpCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func parseEvalCaseListFilter(r *http.Request) (evalsvc.ListFilter, error) {
	filter := evalsvc.ListFilter{
		TenantID:       strings.TrimSpace(r.URL.Query().Get("tenant_id")),
		SourceCaseID:   strings.TrimSpace(r.URL.Query().Get("source_case_id")),
		SourceTaskID:   strings.TrimSpace(r.URL.Query().Get("source_task_id")),
		SourceReportID: strings.TrimSpace(r.URL.Query().Get("source_report_id")),
		VersionID:      strings.TrimSpace(r.URL.Query().Get("version_id")),
		Limit:          20,
	}
	if filter.TenantID == "" {
		return evalsvc.ListFilter{}, errors.New("tenant_id is required")
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return evalsvc.ListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return evalsvc.ListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}
	if rawNeedsFollowUp := strings.TrimSpace(r.URL.Query().Get("needs_follow_up")); rawNeedsFollowUp != "" {
		needsFollowUp, err := strconv.ParseBool(rawNeedsFollowUp)
		if err != nil {
			return evalsvc.ListFilter{}, errors.New("needs_follow_up must be a boolean")
		}
		filter.NeedsFollowUp = &needsFollowUp
	}

	return filter, nil
}
