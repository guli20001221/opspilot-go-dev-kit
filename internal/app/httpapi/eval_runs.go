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
	RunID            string `json:"run_id"`
	TenantID         string `json:"tenant_id"`
	DatasetID        string `json:"dataset_id"`
	DatasetName      string `json:"dataset_name"`
	DatasetItemCount int    `json:"dataset_item_count"`
	Status           string `json:"status"`
	CreatedBy        string `json:"created_by"`
	ErrorReason      string `json:"error_reason,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
	StartedAt        string `json:"started_at,omitempty"`
	FinishedAt       string `json:"finished_at,omitempty"`
}

type listEvalRunsResponse struct {
	Runs       []evalRunResponse `json:"runs"`
	HasMore    bool              `json:"has_more"`
	NextOffset *int              `json:"next_offset,omitempty"`
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
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/eval-runs/")
	if runID == "" || strings.Contains(runID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

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

	writeJSON(w, http.StatusOK, newEvalRunResponse(item))
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

	writeJSON(w, http.StatusCreated, newEvalRunResponse(item))
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
		resp.Runs = append(resp.Runs, newEvalRunResponse(item))
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

func newEvalRunResponse(item evalsvc.EvalRun) evalRunResponse {
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
	if !item.StartedAt.IsZero() {
		resp.StartedAt = item.StartedAt.Format(time.RFC3339Nano)
	}
	if !item.FinishedAt.IsZero() {
		resp.FinishedAt = item.FinishedAt.Format(time.RFC3339Nano)
	}
	return resp
}
