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

type createEvalRunRequest struct {
	TenantID  string `json:"tenant_id"`
	DatasetID string `json:"dataset_id"`
	CreatedBy string `json:"created_by,omitempty"`
}

type evalRunResponse struct {
	RunID            string                        `json:"run_id"`
	TenantID         string                        `json:"tenant_id"`
	DatasetID        string                        `json:"dataset_id"`
	DatasetName      string                        `json:"dataset_name"`
	DatasetItemCount int                           `json:"dataset_item_count"`
	ResultSummary    *evalRunResultSummaryResponse `json:"result_summary,omitempty"`
	Status           string                        `json:"status"`
	CreatedBy        string                        `json:"created_by"`
	ErrorReason      string                        `json:"error_reason,omitempty"`
	CreatedAt        string                        `json:"created_at"`
	UpdatedAt        string                        `json:"updated_at"`
	StartedAt        string                        `json:"started_at,omitempty"`
	FinishedAt       string                        `json:"finished_at,omitempty"`
	Events           []evalRunEventResponse        `json:"events,omitempty"`
	Items            []evalRunItemResponse         `json:"items,omitempty"`
	ItemResults      []evalRunItemResultResponse   `json:"item_results,omitempty"`
}

type listEvalRunsResponse struct {
	Runs       []evalRunResponse `json:"runs"`
	HasMore    bool              `json:"has_more"`
	NextOffset *int              `json:"next_offset,omitempty"`
}

type evalRunEventResponse struct {
	ID        int64  `json:"id"`
	Action    string `json:"action"`
	Actor     string `json:"actor,omitempty"`
	Detail    string `json:"detail,omitempty"`
	CreatedAt string `json:"created_at"`
}

type evalRunItemResponse struct {
	EvalCaseID     string `json:"eval_case_id"`
	Title          string `json:"title"`
	SourceCaseID   string `json:"source_case_id"`
	SourceTaskID   string `json:"source_task_id,omitempty"`
	SourceReportID string `json:"source_report_id,omitempty"`
	TraceID        string `json:"trace_id"`
	VersionID      string `json:"version_id,omitempty"`
}

type evalRunItemResultResponse struct {
	EvalCaseID   string          `json:"eval_case_id"`
	Status       string          `json:"status"`
	Verdict      string          `json:"verdict"`
	Detail       string          `json:"detail,omitempty"`
	Score        float64         `json:"score"`
	JudgeVersion string          `json:"judge_version"`
	JudgeOutput  json.RawMessage `json:"judge_output"`
	UpdatedAt    string          `json:"updated_at"`
}

type evalRunResultSummaryResponse struct {
	TotalItems      int `json:"total_items"`
	RecordedResults int `json:"recorded_results"`
	SucceededItems  int `json:"succeeded_items"`
	FailedItems     int `json:"failed_items"`
	MissingResults  int `json:"missing_results"`
}

func (a *appHandler) handleEvalRuns(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListEvalRuns(w, r)
	case http.MethodPost:
		a.handleCreateEvalRun(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *appHandler) handleEvalRunByID(w http.ResponseWriter, r *http.Request) {
	runID, action, ok := parseEvalRunPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	if action == "" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		a.handleGetEvalRun(w, r, runID)
		return
	}

	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	switch action {
	case "retry":
		a.handleRetryEvalRun(w, r, runID)
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found")
	}
}

func (a *appHandler) handleGetEvalRun(w http.ResponseWriter, r *http.Request, runID string) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}

	a.writeEvalRunDetailResponse(w, r, runID, tenantID, http.StatusOK)
}

func (a *appHandler) handleRetryEvalRun(w http.ResponseWriter, r *http.Request, runID string) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}

	item, err := a.evalRuns.GetRun(r.Context(), runID)
	if err != nil {
		if errors.Is(err, evalsvc.ErrEvalRunNotFound) {
			writeError(w, http.StatusNotFound, "eval_run_not_found", "eval run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "eval_run_lookup_failed", err.Error())
		return
	}
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_run_not_found", "eval run not found")
		return
	}

	retried, err := a.evalRuns.RetryRun(r.Context(), runID)
	if err != nil {
		switch {
		case errors.Is(err, evalsvc.ErrEvalRunNotFound):
			writeError(w, http.StatusNotFound, "eval_run_not_found", "eval run not found")
		case errors.Is(err, evalsvc.ErrInvalidEvalRunState):
			writeError(w, http.StatusConflict, "invalid_eval_run_state", err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "eval_run_retry_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newEvalRunResponse(retried, nil, nil, nil))
}

func (a *appHandler) handleCreateEvalRun(w http.ResponseWriter, r *http.Request) {
	var req createEvalRunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.DatasetID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_eval_run", "tenant_id and dataset_id are required")
		return
	}

	item, err := a.evalRuns.CreateRun(r.Context(), evalsvc.CreateRunInput{
		TenantID:  strings.TrimSpace(req.TenantID),
		DatasetID: strings.TrimSpace(req.DatasetID),
		CreatedBy: strings.TrimSpace(req.CreatedBy),
	})
	if err != nil {
		switch {
		case errors.Is(err, evalsvc.ErrEvalDatasetNotFound):
			writeError(w, http.StatusNotFound, "eval_dataset_not_found", "eval dataset not found")
		case errors.Is(err, evalsvc.ErrInvalidEvalDatasetState):
			writeError(w, http.StatusConflict, "invalid_eval_dataset_state", "eval dataset is not in a valid state for run kickoff")
		case errors.Is(err, evalsvc.ErrInvalidEvalDataset):
			writeError(w, http.StatusConflict, "invalid_eval_run", "eval run request is invalid for the current tenant scope")
		default:
			writeError(w, http.StatusInternalServerError, "eval_run_create_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, newEvalRunResponse(item, nil, nil, nil))
}

func (a *appHandler) handleListEvalRuns(w http.ResponseWriter, r *http.Request) {
	filter, err := parseEvalRunListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.evalRuns.ListRuns(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_run_list_failed", err.Error())
		return
	}

	resp := listEvalRunsResponse{
		Runs:    make([]evalRunResponse, 0, len(page.Runs)),
		HasMore: page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.Runs {
		resp.Runs = append(resp.Runs, newEvalRunResponse(item, nil, nil, nil))
	}

	writeJSON(w, http.StatusOK, resp)
}

func parseEvalRunListFilter(r *http.Request) (evalsvc.RunListFilter, error) {
	filter := evalsvc.RunListFilter{
		TenantID:  strings.TrimSpace(r.URL.Query().Get("tenant_id")),
		DatasetID: strings.TrimSpace(r.URL.Query().Get("dataset_id")),
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		Limit:     20,
	}
	if filter.TenantID == "" {
		return evalsvc.RunListFilter{}, errors.New("tenant_id is required")
	}
	if filter.Status != "" && filter.Status != evalsvc.RunStatusQueued && filter.Status != evalsvc.RunStatusRunning && filter.Status != evalsvc.RunStatusSucceeded && filter.Status != evalsvc.RunStatusFailed {
		return evalsvc.RunListFilter{}, errors.New("status must be queued, running, succeeded, or failed")
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return evalsvc.RunListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return evalsvc.RunListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}
	return filter, nil
}

func newEvalRunResponse(item evalsvc.EvalRun, events []evalsvc.EvalRunEvent, items []evalsvc.EvalRunItem, results []evalsvc.EvalRunItemResult) evalRunResponse {
	resp := evalRunResponse{
		RunID:            item.ID,
		TenantID:         item.TenantID,
		DatasetID:        item.DatasetID,
		DatasetName:      item.DatasetName,
		DatasetItemCount: item.DatasetItemCount,
		Status:           item.Status,
		CreatedBy:        item.CreatedBy,
		ErrorReason:      item.ErrorReason,
		CreatedAt:        item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:        item.UpdatedAt.Format(time.RFC3339Nano),
	}
	if item.ResultSummary != nil {
		resp.ResultSummary = &evalRunResultSummaryResponse{
			TotalItems:      item.ResultSummary.TotalItems,
			RecordedResults: item.ResultSummary.RecordedResults,
			SucceededItems:  item.ResultSummary.SucceededItems,
			FailedItems:     item.ResultSummary.FailedItems,
			MissingResults:  item.ResultSummary.MissingResults,
		}
	}
	if !item.StartedAt.IsZero() {
		resp.StartedAt = item.StartedAt.Format(time.RFC3339Nano)
	}
	if !item.FinishedAt.IsZero() {
		resp.FinishedAt = item.FinishedAt.Format(time.RFC3339Nano)
	}
	if len(events) > 0 {
		resp.Events = make([]evalRunEventResponse, 0, len(events))
		for _, event := range events {
			resp.Events = append(resp.Events, evalRunEventResponse{
				ID:        event.ID,
				Action:    event.Action,
				Actor:     event.Actor,
				Detail:    event.Detail,
				CreatedAt: event.CreatedAt.Format(time.RFC3339Nano),
			})
		}
	}
	if len(items) > 0 {
		resp.Items = make([]evalRunItemResponse, 0, len(items))
		for _, item := range items {
			resp.Items = append(resp.Items, evalRunItemResponse{
				EvalCaseID:     item.EvalCaseID,
				Title:          item.Title,
				SourceCaseID:   item.SourceCaseID,
				SourceTaskID:   item.SourceTaskID,
				SourceReportID: item.SourceReportID,
				TraceID:        item.TraceID,
				VersionID:      item.VersionID,
			})
		}
	}
	if len(results) > 0 {
		resp.ItemResults = make([]evalRunItemResultResponse, 0, len(results))
		for _, result := range results {
			resp.ItemResults = append(resp.ItemResults, evalRunItemResultResponse{
				EvalCaseID:   result.EvalCaseID,
				Status:       result.Status,
				Verdict:      result.Verdict,
				Detail:       result.Detail,
				Score:        result.Score,
				JudgeVersion: result.JudgeVersion,
				JudgeOutput:  result.JudgeOutput,
				UpdatedAt:    result.UpdatedAt.Format(time.RFC3339Nano),
			})
		}
	}
	return resp
}

func (a *appHandler) writeEvalRunDetailResponse(w http.ResponseWriter, r *http.Request, runID string, tenantID string, statusCode int) {
	detail, err := a.evalRuns.GetRunDetail(r.Context(), runID)
	if err != nil {
		if errors.Is(err, evalsvc.ErrEvalRunNotFound) {
			writeError(w, http.StatusNotFound, "eval_run_not_found", "eval run not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "eval_run_lookup_failed", err.Error())
		return
	}
	item := detail.Run
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_run_not_found", "eval run not found")
		return
	}

	writeJSON(w, statusCode, newEvalRunResponse(item, detail.Events, detail.Items, detail.ItemResults))
}

func parseEvalRunPath(path string) (runID string, action string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/api/v1/eval-runs/")
	if trimmed == "" {
		return "", "", false
	}

	parts := strings.Split(trimmed, "/")
	switch len(parts) {
	case 1:
		if parts[0] == "" {
			return "", "", false
		}
		return parts[0], "", true
	case 2:
		if parts[0] == "" || parts[1] == "" {
			return "", "", false
		}
		return parts[0], parts[1], true
	default:
		return "", "", false
	}
}
