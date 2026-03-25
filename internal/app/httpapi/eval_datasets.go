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

type createEvalDatasetRequest struct {
	TenantID    string   `json:"tenant_id"`
	Name        string   `json:"name,omitempty"`
	Description string   `json:"description,omitempty"`
	EvalCaseIDs []string `json:"eval_case_ids"`
	CreatedBy   string   `json:"created_by,omitempty"`
}

type addEvalDatasetItemRequest struct {
	TenantID   string `json:"tenant_id"`
	EvalCaseID string `json:"eval_case_id"`
	AddedBy    string `json:"added_by,omitempty"`
}

type evalDatasetItemResponse struct {
	EvalCaseID     string `json:"eval_case_id"`
	Title          string `json:"title"`
	SourceCaseID   string `json:"source_case_id"`
	SourceTaskID   string `json:"source_task_id,omitempty"`
	SourceReportID string `json:"source_report_id,omitempty"`
	TraceID        string `json:"trace_id,omitempty"`
	VersionID      string `json:"version_id,omitempty"`
}

type evalDatasetResponse struct {
	DatasetID   string                    `json:"dataset_id"`
	TenantID    string                    `json:"tenant_id"`
	Name        string                    `json:"name"`
	Description string                    `json:"description,omitempty"`
	Status      string                    `json:"status"`
	CreatedBy   string                    `json:"created_by"`
	CreatedAt   string                    `json:"created_at"`
	UpdatedAt   string                    `json:"updated_at"`
	Items       []evalDatasetItemResponse `json:"items"`
}

type evalDatasetSummaryResponse struct {
	DatasetID string `json:"dataset_id"`
	TenantID  string `json:"tenant_id"`
	Name      string `json:"name"`
	Status    string `json:"status"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	ItemCount int    `json:"item_count"`
}

type listEvalDatasetsResponse struct {
	Datasets   []evalDatasetSummaryResponse `json:"datasets"`
	HasMore    bool                         `json:"has_more"`
	NextOffset *int                         `json:"next_offset,omitempty"`
}

func (a *appHandler) handleEvalDatasets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.handleListEvalDatasets(w, r)
	case http.MethodPost:
		a.handleCreateEvalDataset(w, r)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

func (a *appHandler) handleListEvalDatasets(w http.ResponseWriter, r *http.Request) {
	filter, err := parseEvalDatasetListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.evalDatasets.ListDatasets(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_list_failed", err.Error())
		return
	}

	resp := listEvalDatasetsResponse{
		Datasets: make([]evalDatasetSummaryResponse, 0, len(page.Datasets)),
		HasMore:  page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.Datasets {
		resp.Datasets = append(resp.Datasets, newEvalDatasetSummaryResponse(item))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleCreateEvalDataset(w http.ResponseWriter, r *http.Request) {
	var req createEvalDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" || len(req.EvalCaseIDs) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_eval_dataset", "tenant_id and eval_case_ids are required")
		return
	}

	item, err := a.evalDatasets.CreateDataset(r.Context(), evalsvc.CreateDatasetInput{
		TenantID:    strings.TrimSpace(req.TenantID),
		Name:        strings.TrimSpace(req.Name),
		Description: strings.TrimSpace(req.Description),
		EvalCaseIDs: req.EvalCaseIDs,
		CreatedBy:   strings.TrimSpace(req.CreatedBy),
	})
	if err != nil {
		switch {
		case errors.Is(err, evalsvc.ErrEvalCaseNotFound):
			writeError(w, http.StatusNotFound, "eval_case_not_found", "eval case not found")
		case errors.Is(err, evalsvc.ErrInvalidEvalDataset):
			writeError(w, http.StatusConflict, "invalid_eval_dataset", "eval dataset request is invalid for the current tenant scope")
		default:
			writeError(w, http.StatusInternalServerError, "eval_dataset_create_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusCreated, newEvalDatasetResponse(item))
}

func (a *appHandler) handleEvalDatasetByID(w http.ResponseWriter, r *http.Request) {
	datasetID, action, ok := parseEvalDatasetPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	if action == "" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		a.handleGetEvalDataset(w, r, datasetID)
		return
	}

	if action != "items" {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	a.handleAddEvalDatasetItem(w, r, datasetID)
}

func (a *appHandler) handleGetEvalDataset(w http.ResponseWriter, r *http.Request, datasetID string) {
	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}

	item, err := a.evalDatasets.GetDataset(r.Context(), datasetID)
	if err != nil {
		if errors.Is(err, evalsvc.ErrEvalDatasetNotFound) {
			writeError(w, http.StatusNotFound, "eval_dataset_not_found", "eval dataset not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "eval_dataset_lookup_failed", err.Error())
		return
	}
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_dataset_not_found", "eval dataset not found")
		return
	}

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item))
}

func (a *appHandler) handleAddEvalDatasetItem(w http.ResponseWriter, r *http.Request, datasetID string) {
	var req addEvalDatasetItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.EvalCaseID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_eval_dataset", "tenant_id and eval_case_id are required")
		return
	}

	item, err := a.evalDatasets.AddDatasetItem(r.Context(), datasetID, evalsvc.AddDatasetItemInput{
		TenantID:   strings.TrimSpace(req.TenantID),
		EvalCaseID: strings.TrimSpace(req.EvalCaseID),
		AddedBy:    strings.TrimSpace(req.AddedBy),
	})
	if err != nil {
		switch {
		case errors.Is(err, evalsvc.ErrEvalDatasetNotFound):
			writeError(w, http.StatusNotFound, "eval_dataset_not_found", "eval dataset not found")
		case errors.Is(err, evalsvc.ErrEvalCaseNotFound):
			writeError(w, http.StatusNotFound, "eval_case_not_found", "eval case not found")
		case errors.Is(err, evalsvc.ErrInvalidEvalDatasetState):
			writeError(w, http.StatusConflict, "invalid_eval_dataset_state", "eval dataset is not in a valid state for append")
		case errors.Is(err, evalsvc.ErrInvalidEvalDataset):
			writeError(w, http.StatusConflict, "invalid_eval_dataset", "eval dataset request is invalid for the current tenant scope")
		default:
			writeError(w, http.StatusInternalServerError, "eval_dataset_update_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item))
}

func newEvalDatasetResponse(item evalsvc.EvalDataset) evalDatasetResponse {
	resp := evalDatasetResponse{
		DatasetID:   item.ID,
		TenantID:    item.TenantID,
		Name:        item.Name,
		Description: item.Description,
		Status:      item.Status,
		CreatedBy:   item.CreatedBy,
		CreatedAt:   item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   item.UpdatedAt.Format(time.RFC3339Nano),
		Items:       make([]evalDatasetItemResponse, 0, len(item.Items)),
	}
	for _, member := range item.Items {
		resp.Items = append(resp.Items, evalDatasetItemResponse{
			EvalCaseID:     member.EvalCaseID,
			Title:          member.Title,
			SourceCaseID:   member.SourceCaseID,
			SourceTaskID:   member.SourceTaskID,
			SourceReportID: member.SourceReportID,
			TraceID:        member.TraceID,
			VersionID:      member.VersionID,
		})
	}

	return resp
}

func newEvalDatasetSummaryResponse(item evalsvc.EvalDatasetSummary) evalDatasetSummaryResponse {
	return evalDatasetSummaryResponse{
		DatasetID: item.ID,
		TenantID:  item.TenantID,
		Name:      item.Name,
		Status:    item.Status,
		CreatedBy: item.CreatedBy,
		CreatedAt: item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: item.UpdatedAt.Format(time.RFC3339Nano),
		ItemCount: item.ItemCount,
	}
}

func parseEvalDatasetListFilter(r *http.Request) (evalsvc.DatasetListFilter, error) {
	filter := evalsvc.DatasetListFilter{
		TenantID:  strings.TrimSpace(r.URL.Query().Get("tenant_id")),
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		CreatedBy: strings.TrimSpace(r.URL.Query().Get("created_by")),
		Limit:     20,
	}
	if filter.TenantID == "" {
		return evalsvc.DatasetListFilter{}, errors.New("tenant_id is required")
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return evalsvc.DatasetListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return evalsvc.DatasetListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	return filter, nil
}

func parseEvalDatasetPath(path string) (datasetID string, action string, ok bool) {
	trimmed := strings.TrimPrefix(path, "/api/v1/eval-datasets/")
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
