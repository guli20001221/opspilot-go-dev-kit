package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
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
