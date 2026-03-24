package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opspilot-go/internal/report"
)

type reportResponse struct {
	ReportID     string          `json:"report_id"`
	TenantID     string          `json:"tenant_id"`
	SourceTaskID string          `json:"source_task_id"`
	ReportType   string          `json:"report_type"`
	Status       string          `json:"status"`
	Title        string          `json:"title"`
	Summary      string          `json:"summary"`
	ContentURI   string          `json:"content_uri,omitempty"`
	Metadata     json.RawMessage `json:"metadata,omitempty"`
	CreatedBy    string          `json:"created_by,omitempty"`
	CreatedAt    string          `json:"created_at"`
	ReadyAt      string          `json:"ready_at,omitempty"`
}

type listReportsResponse struct {
	Reports    []reportResponse `json:"reports"`
	HasMore    bool             `json:"has_more"`
	NextOffset *int             `json:"next_offset,omitempty"`
}

type reportComparisonSummaryResponse struct {
	SameTenant         bool  `json:"same_tenant"`
	SameReportType     bool  `json:"same_report_type"`
	SourceTaskChanged  bool  `json:"source_task_changed"`
	TitleChanged       bool  `json:"title_changed"`
	SummaryChanged     bool  `json:"summary_changed"`
	ContentURIChanged  bool  `json:"content_uri_changed"`
	MetadataChanged    bool  `json:"metadata_changed"`
	CreatedAtChanged   bool  `json:"created_at_changed"`
	ReadyAtChanged     bool  `json:"ready_at_changed"`
	ReadyAtDeltaSecond int64 `json:"ready_at_delta_second"`
}

type reportComparisonResponse struct {
	Left    reportResponse                  `json:"left"`
	Right   reportResponse                  `json:"right"`
	Summary reportComparisonSummaryResponse `json:"summary"`
}

func (a *appHandler) handleReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	filter, err := parseReportListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.reports.ListReports(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "report_list_failed", err.Error())
		return
	}

	resp := listReportsResponse{
		Reports: make([]reportResponse, 0, len(page.Reports)),
		HasMore: page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.Reports {
		resp.Reports = append(resp.Reports, newReportResponse(item))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleReportCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	leftReportID := r.URL.Query().Get("left_report_id")
	if leftReportID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "left_report_id is required")
		return
	}
	rightReportID := r.URL.Query().Get("right_report_id")
	if rightReportID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "right_report_id is required")
		return
	}

	comparison, err := a.reports.CompareReports(r.Context(), leftReportID, rightReportID)
	if err != nil {
		if errors.Is(err, report.ErrReportNotFound) {
			writeError(w, http.StatusNotFound, "report_not_found", "report not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "report_compare_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, reportComparisonResponse{
		Left:  newReportResponse(comparison.Left),
		Right: newReportResponse(comparison.Right),
		Summary: reportComparisonSummaryResponse{
			SameTenant:         comparison.Summary.SameTenant,
			SameReportType:     comparison.Summary.SameReportType,
			SourceTaskChanged:  comparison.Summary.SourceTaskChanged,
			TitleChanged:       comparison.Summary.TitleChanged,
			SummaryChanged:     comparison.Summary.SummaryChanged,
			ContentURIChanged:  comparison.Summary.ContentURIChanged,
			MetadataChanged:    comparison.Summary.MetadataChanged,
			CreatedAtChanged:   comparison.Summary.CreatedAtChanged,
			ReadyAtChanged:     comparison.Summary.ReadyAtChanged,
			ReadyAtDeltaSecond: comparison.Summary.ReadyAtDeltaSecond,
		},
	})
}

func (a *appHandler) handleReportByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	reportID := strings.TrimPrefix(r.URL.Path, "/api/v1/reports/")
	if reportID == "" || strings.Contains(reportID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	item, err := a.reports.GetReport(r.Context(), reportID)
	if err != nil {
		if errors.Is(err, report.ErrReportNotFound) {
			writeError(w, http.StatusNotFound, "report_not_found", "report not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "report_lookup_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, newReportResponse(item))
}

func parseReportListFilter(r *http.Request) (report.ListFilter, error) {
	filter := report.ListFilter{
		TenantID:     r.URL.Query().Get("tenant_id"),
		Status:       r.URL.Query().Get("status"),
		ReportType:   r.URL.Query().Get("report_type"),
		SourceTaskID: r.URL.Query().Get("source_task_id"),
		Limit:        20,
	}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return report.ListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return report.ListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}
	if filter.Status != "" && filter.Status != report.StatusReady {
		return report.ListFilter{}, errors.New("status must be ready")
	}
	if filter.ReportType != "" && filter.ReportType != report.TypeWorkflowSummary {
		return report.ListFilter{}, errors.New("report_type must be workflow_summary")
	}

	return filter, nil
}

func newReportResponse(item report.Report) reportResponse {
	resp := reportResponse{
		ReportID:     item.ID,
		TenantID:     item.TenantID,
		SourceTaskID: item.SourceTaskID,
		ReportType:   item.ReportType,
		Status:       item.Status,
		Title:        item.Title,
		Summary:      item.Summary,
		ContentURI:   item.ContentURI,
		Metadata:     item.MetadataJSON,
		CreatedBy:    item.CreatedBy,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339Nano),
	}
	if item.ReadyAt != nil {
		resp.ReadyAt = item.ReadyAt.Format(time.RFC3339Nano)
	}

	return resp
}
