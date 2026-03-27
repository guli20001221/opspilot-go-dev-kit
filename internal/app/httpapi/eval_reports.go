package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	evalsvc "opspilot-go/internal/eval"
)

type evalReportBadCaseResponse struct {
	EvalCaseID     string  `json:"eval_case_id"`
	Title          string  `json:"title"`
	SourceCaseID   string  `json:"source_case_id"`
	SourceTaskID   string  `json:"source_task_id,omitempty"`
	SourceReportID string  `json:"source_report_id,omitempty"`
	TraceID        string  `json:"trace_id,omitempty"`
	VersionID      string  `json:"version_id,omitempty"`
	Verdict        string  `json:"verdict"`
	Detail         string  `json:"detail,omitempty"`
	Score          float64 `json:"score"`
}

type evalReportResponse struct {
	ReportID        string                      `json:"report_id"`
	TenantID        string                      `json:"tenant_id"`
	RunID           string                      `json:"run_id"`
	DatasetID       string                      `json:"dataset_id"`
	DatasetName     string                      `json:"dataset_name"`
	RunStatus       string                      `json:"run_status"`
	Status          string                      `json:"status"`
	Summary         string                      `json:"summary"`
	TotalItems      int                         `json:"total_items"`
	RecordedResults int                         `json:"recorded_results"`
	PassedItems     int                         `json:"passed_items"`
	FailedItems     int                         `json:"failed_items"`
	MissingResults  int                         `json:"missing_results"`
	AverageScore    float64                     `json:"average_score"`
	JudgeVersion    string                      `json:"judge_version,omitempty"`
	Metadata        json.RawMessage             `json:"metadata,omitempty"`
	BadCases        []evalReportBadCaseResponse `json:"bad_cases,omitempty"`
	CreatedAt       string                      `json:"created_at"`
	UpdatedAt       string                      `json:"updated_at"`
	ReadyAt         string                      `json:"ready_at"`
}

type listEvalReportsResponse struct {
	Reports    []evalReportResponse `json:"reports"`
	HasMore    bool                 `json:"has_more"`
	NextOffset *int                 `json:"next_offset,omitempty"`
}

func (a *appHandler) handleEvalReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	filter, err := parseEvalReportListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.evalReports.ListEvalReports(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_report_list_failed", err.Error())
		return
	}

	resp := listEvalReportsResponse{
		Reports: make([]evalReportResponse, 0, len(page.Reports)),
		HasMore: page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.Reports {
		resp.Reports = append(resp.Reports, newEvalReportResponse(item, false))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleEvalReportByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	reportID := strings.TrimPrefix(r.URL.Path, "/api/v1/eval-reports/")
	if reportID == "" || strings.Contains(reportID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	item, err := a.evalReports.GetEvalReport(r.Context(), reportID)
	if err != nil {
		if errors.Is(err, evalsvc.ErrEvalReportNotFound) {
			writeError(w, http.StatusNotFound, "eval_report_not_found", "eval report not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "eval_report_lookup_failed", err.Error())
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_report_not_found", "eval report not found")
		return
	}

	writeJSON(w, http.StatusOK, newEvalReportResponse(item, true))
}

func parseEvalReportListFilter(r *http.Request) (evalsvc.EvalReportListFilter, error) {
	filter := evalsvc.EvalReportListFilter{
		TenantID:  strings.TrimSpace(r.URL.Query().Get("tenant_id")),
		DatasetID: strings.TrimSpace(r.URL.Query().Get("dataset_id")),
		RunStatus: strings.TrimSpace(r.URL.Query().Get("run_status")),
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:     20,
	}
	if filter.TenantID == "" {
		return evalsvc.EvalReportListFilter{}, errors.New("tenant_id is required")
	}
	if filter.RunStatus != "" && filter.RunStatus != evalsvc.RunStatusSucceeded && filter.RunStatus != evalsvc.RunStatusFailed {
		return evalsvc.EvalReportListFilter{}, errors.New("run_status must be succeeded or failed")
	}
	if filter.Status != "" && filter.Status != evalsvc.EvalReportStatusReady {
		return evalsvc.EvalReportListFilter{}, errors.New("status must be ready")
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return evalsvc.EvalReportListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return evalsvc.EvalReportListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	return filter, nil
}

func newEvalReportResponse(item evalsvc.EvalReport, includeHeavy bool) evalReportResponse {
	resp := evalReportResponse{
		ReportID:        item.ID,
		TenantID:        item.TenantID,
		RunID:           item.RunID,
		DatasetID:       item.DatasetID,
		DatasetName:     item.DatasetName,
		RunStatus:       item.RunStatus,
		Status:          item.Status,
		Summary:         item.Summary,
		TotalItems:      item.TotalItems,
		RecordedResults: item.RecordedResults,
		PassedItems:     item.PassedItems,
		FailedItems:     item.FailedItems,
		MissingResults:  item.MissingResults,
		AverageScore:    item.AverageScore,
		JudgeVersion:    item.JudgeVersion,
		CreatedAt:       item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:       item.UpdatedAt.Format(time.RFC3339Nano),
		ReadyAt:         item.ReadyAt.Format(time.RFC3339Nano),
	}
	if includeHeavy {
		resp.Metadata = item.MetadataJSON
		if len(item.BadCases) > 0 {
			resp.BadCases = make([]evalReportBadCaseResponse, 0, len(item.BadCases))
			for _, badCase := range item.BadCases {
				resp.BadCases = append(resp.BadCases, evalReportBadCaseResponse{
					EvalCaseID:     badCase.EvalCaseID,
					Title:          badCase.Title,
					SourceCaseID:   badCase.SourceCaseID,
					SourceTaskID:   badCase.SourceTaskID,
					SourceReportID: badCase.SourceReportID,
					TraceID:        badCase.TraceID,
					VersionID:      badCase.VersionID,
					Verdict:        badCase.Verdict,
					Detail:         badCase.Detail,
					Score:          badCase.Score,
				})
			}
		}
	}

	return resp
}
