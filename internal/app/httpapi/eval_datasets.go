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

type publishEvalDatasetRequest struct {
	TenantID    string `json:"tenant_id"`
	PublishedBy string `json:"published_by,omitempty"`
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
	DatasetID               string                    `json:"dataset_id"`
	TenantID                string                    `json:"tenant_id"`
	Name                    string                    `json:"name"`
	Description             string                    `json:"description,omitempty"`
	Status                  string                    `json:"status"`
	CreatedBy               string                    `json:"created_by"`
	CreatedAt               string                    `json:"created_at"`
	UpdatedAt               string                    `json:"updated_at"`
	PublishedBy             string                    `json:"published_by,omitempty"`
	PublishedAt             string                    `json:"published_at,omitempty"`
	LatestRunID             string                    `json:"latest_run_id,omitempty"`
	LatestRunStatus         string                    `json:"latest_run_status,omitempty"`
	LatestReportID          string                    `json:"latest_report_id,omitempty"`
	LatestReportStatus      string                    `json:"latest_report_status,omitempty"`
	UnresolvedFollowUpCount int                       `json:"unresolved_follow_up_count"`
	NeedsFollowUp           bool                      `json:"needs_follow_up"`
	Items                   []evalDatasetItemResponse `json:"items"`
}

type evalDatasetSummaryResponse struct {
	DatasetID               string `json:"dataset_id"`
	TenantID                string `json:"tenant_id"`
	Name                    string `json:"name"`
	Status                  string `json:"status"`
	CreatedBy               string `json:"created_by"`
	CreatedAt               string `json:"created_at"`
	UpdatedAt               string `json:"updated_at"`
	ItemCount               int    `json:"item_count"`
	LatestRunID             string `json:"latest_run_id,omitempty"`
	LatestRunStatus         string `json:"latest_run_status,omitempty"`
	LatestReportID          string `json:"latest_report_id,omitempty"`
	LatestReportStatus      string `json:"latest_report_status,omitempty"`
	UnresolvedFollowUpCount int    `json:"unresolved_follow_up_count"`
	NeedsFollowUp           bool   `json:"needs_follow_up"`
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
	filter, needsFollowUp, err := parseEvalDatasetListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := a.listEvalDatasetsResponse(r.Context(), filter, needsFollowUp)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_list_failed", err.Error())
		return
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

	writeJSON(w, http.StatusCreated, newEvalDatasetResponse(item, evalDatasetLatestRunSummary{}))
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

	switch action {
	case "items":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		a.handleAddEvalDatasetItem(w, r, datasetID)
	case "publish":
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		a.handlePublishEvalDataset(w, r, datasetID)
	default:
		writeError(w, http.StatusNotFound, "not_found", "not found")
	}
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

	latestRunSummaries, err := a.evalDatasetLatestRunSummaries(r.Context(), item.TenantID, []evalsvc.EvalDatasetSummary{{
		ID:       item.ID,
		TenantID: item.TenantID,
	}})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_latest_run_summary_failed", err.Error())
		return
	}
	latestRun := latestRunSummaries[item.ID]

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item, latestRun))
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

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item, evalDatasetLatestRunSummary{}))
}

func (a *appHandler) handlePublishEvalDataset(w http.ResponseWriter, r *http.Request, datasetID string) {
	var req publishEvalDatasetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" {
		writeError(w, http.StatusBadRequest, "invalid_eval_dataset", "tenant_id is required")
		return
	}

	item, err := a.evalDatasets.PublishDataset(r.Context(), datasetID, evalsvc.PublishDatasetInput{
		TenantID:    strings.TrimSpace(req.TenantID),
		PublishedBy: strings.TrimSpace(req.PublishedBy),
	})
	if err != nil {
		switch {
		case errors.Is(err, evalsvc.ErrEvalDatasetNotFound):
			writeError(w, http.StatusNotFound, "eval_dataset_not_found", "eval dataset not found")
		case errors.Is(err, evalsvc.ErrInvalidEvalDatasetState):
			writeError(w, http.StatusConflict, "invalid_eval_dataset_state", "eval dataset is not in a valid state for publish")
		case errors.Is(err, evalsvc.ErrInvalidEvalDataset):
			writeError(w, http.StatusConflict, "invalid_eval_dataset", "eval dataset request is invalid for the current tenant scope")
		default:
			writeError(w, http.StatusInternalServerError, "eval_dataset_publish_failed", err.Error())
		}
		return
	}

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item, evalDatasetLatestRunSummary{}))
}

func newEvalDatasetResponse(item evalsvc.EvalDataset, latestRun evalDatasetLatestRunSummary) evalDatasetResponse {
	resp := evalDatasetResponse{
		DatasetID:               item.ID,
		TenantID:                item.TenantID,
		Name:                    item.Name,
		Description:             item.Description,
		Status:                  item.Status,
		CreatedBy:               item.CreatedBy,
		CreatedAt:               item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:               item.UpdatedAt.Format(time.RFC3339Nano),
		LatestRunID:             latestRun.LatestRunID,
		LatestRunStatus:         latestRun.LatestRunStatus,
		LatestReportID:          latestRun.LatestReportID,
		LatestReportStatus:      latestRun.LatestReportStatus,
		UnresolvedFollowUpCount: latestRun.UnresolvedFollowUpCount,
		NeedsFollowUp:           latestRun.NeedsFollowUp,
		Items:                   make([]evalDatasetItemResponse, 0, len(item.Items)),
	}
	if item.PublishedBy != "" {
		resp.PublishedBy = item.PublishedBy
	}
	if !item.PublishedAt.IsZero() {
		resp.PublishedAt = item.PublishedAt.Format(time.RFC3339Nano)
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

type evalDatasetLatestRunSummary struct {
	LatestRunID             string
	LatestRunStatus         string
	LatestReportID          string
	LatestReportStatus      string
	UnresolvedFollowUpCount int
	NeedsFollowUp           bool
}

func newEvalDatasetSummaryResponse(item evalsvc.EvalDatasetSummary, latestRun evalDatasetLatestRunSummary) evalDatasetSummaryResponse {
	return evalDatasetSummaryResponse{
		DatasetID:               item.ID,
		TenantID:                item.TenantID,
		Name:                    item.Name,
		Status:                  item.Status,
		CreatedBy:               item.CreatedBy,
		CreatedAt:               item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:               item.UpdatedAt.Format(time.RFC3339Nano),
		ItemCount:               item.ItemCount,
		LatestRunID:             latestRun.LatestRunID,
		LatestRunStatus:         latestRun.LatestRunStatus,
		LatestReportID:          latestRun.LatestReportID,
		LatestReportStatus:      latestRun.LatestReportStatus,
		UnresolvedFollowUpCount: latestRun.UnresolvedFollowUpCount,
		NeedsFollowUp:           latestRun.NeedsFollowUp,
	}
}

func parseEvalDatasetListFilter(r *http.Request) (evalsvc.DatasetListFilter, *bool, error) {
	filter := evalsvc.DatasetListFilter{
		TenantID:  strings.TrimSpace(r.URL.Query().Get("tenant_id")),
		Status:    strings.TrimSpace(r.URL.Query().Get("status")),
		CreatedBy: strings.TrimSpace(r.URL.Query().Get("created_by")),
		Limit:     20,
	}
	if filter.TenantID == "" {
		return evalsvc.DatasetListFilter{}, nil, errors.New("tenant_id is required")
	}
	if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return evalsvc.DatasetListFilter{}, nil, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := strings.TrimSpace(r.URL.Query().Get("offset")); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return evalsvc.DatasetListFilter{}, nil, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}
	var needsFollowUp *bool
	if rawNeedsFollowUp := strings.TrimSpace(r.URL.Query().Get("needs_follow_up")); rawNeedsFollowUp != "" {
		switch rawNeedsFollowUp {
		case "true":
			value := true
			needsFollowUp = &value
		case "false":
			value := false
			needsFollowUp = &value
		default:
			return evalsvc.DatasetListFilter{}, nil, errors.New("needs_follow_up must be true or false")
		}
	}

	return filter, needsFollowUp, nil
}

func (a *appHandler) listEvalDatasetsResponse(ctx context.Context, filter evalsvc.DatasetListFilter, needsFollowUp *bool) (listEvalDatasetsResponse, error) {
	if needsFollowUp == nil {
		page, err := a.evalDatasets.ListDatasets(ctx, filter)
		if err != nil {
			return listEvalDatasetsResponse{}, err
		}
		latestRunSummaries, err := a.evalDatasetLatestRunSummaries(ctx, filter.TenantID, page.Datasets)
		if err != nil {
			return listEvalDatasetsResponse{}, err
		}
		resp := listEvalDatasetsResponse{
			Datasets: buildEvalDatasetListRows(page.Datasets, latestRunSummaries),
			HasMore:  page.HasMore,
		}
		if page.HasMore {
			resp.NextOffset = &page.NextOffset
		}
		return resp, nil
	}

	scanFilter := filter
	scanFilter.Offset = 0
	if scanFilter.Limit < 50 {
		scanFilter.Limit = 50
	}

	candidates := make([]evalsvc.EvalDatasetSummary, 0)
	for {
		page, err := a.evalDatasets.ListDatasets(ctx, scanFilter)
		if err != nil {
			return listEvalDatasetsResponse{}, err
		}
		candidates = append(candidates, page.Datasets...)
		if !page.HasMore {
			break
		}
		scanFilter.Offset = page.NextOffset
	}

	latestRunSummaries, err := a.evalDatasetLatestRunSummaries(ctx, filter.TenantID, candidates)
	if err != nil {
		return listEvalDatasetsResponse{}, err
	}
	rows := buildEvalDatasetListRows(candidates, latestRunSummaries)
	resp := listEvalDatasetsResponse{Datasets: make([]evalDatasetSummaryResponse, 0, filter.Limit)}
	matchedCount := 0
	for _, row := range rows {
		if row.NeedsFollowUp != *needsFollowUp {
			continue
		}
		if matchedCount < filter.Offset {
			matchedCount++
			continue
		}
		if len(resp.Datasets) < filter.Limit {
			resp.Datasets = append(resp.Datasets, row)
			matchedCount++
			continue
		}
		resp.HasMore = true
		nextOffset := filter.Offset + filter.Limit
		resp.NextOffset = &nextOffset
		return resp, nil
	}

	return resp, nil
}

func buildEvalDatasetListRows(items []evalsvc.EvalDatasetSummary, latestRunSummaries map[string]evalDatasetLatestRunSummary) []evalDatasetSummaryResponse {
	rows := make([]evalDatasetSummaryResponse, 0, len(items))
	for _, item := range items {
		rows = append(rows, newEvalDatasetSummaryResponse(item, latestRunSummaries[item.ID]))
	}
	return rows
}

func (a *appHandler) evalDatasetLatestRunSummaries(ctx context.Context, tenantID string, items []evalsvc.EvalDatasetSummary) (map[string]evalDatasetLatestRunSummary, error) {
	summaries := make(map[string]evalDatasetLatestRunSummary, len(items))
	if len(items) == 0 || a.evalRuns == nil {
		return summaries, nil
	}

	targetDatasetIDs := make(map[string]struct{}, len(items))
	for _, item := range items {
		targetDatasetIDs[item.ID] = struct{}{}
	}

	latestRunsByDataset, err := a.collectLatestEvalRunsByDataset(ctx, tenantID, targetDatasetIDs)
	if err != nil {
		return nil, err
	}

	reportsByDatasetID := make(map[string]evalsvc.EvalReport, len(items))
	fallbackRunIDs := make(map[string]evalsvc.EvalRun)
	for _, item := range items {
		latestRun, ok := latestRunsByDataset[item.ID]
		if !ok {
			summaries[item.ID] = evalDatasetLatestRunSummary{}
			continue
		}

		summary := evalDatasetLatestRunSummary{
			LatestRunID:     latestRun.ID,
			LatestRunStatus: latestRun.Status,
		}
		if a.evalReports != nil && (latestRun.Status == evalsvc.RunStatusSucceeded || latestRun.Status == evalsvc.RunStatusFailed) {
			reportID := evalsvc.EvalReportIDFromRunID(latestRun.ID)
			reportItem, err := a.evalReports.GetEvalReport(ctx, reportID)
			switch {
			case err == nil && reportItem.TenantID == tenantID:
				summary.LatestReportID = reportItem.ID
				summary.LatestReportStatus = reportItem.Status
				reportsByDatasetID[item.ID] = reportItem
			case err == nil:
				// Ignore cross-tenant report rows.
			case errors.Is(err, evalsvc.ErrEvalReportNotFound):
				fallbackRunIDs[item.ID] = latestRun
			default:
				return nil, fmt.Errorf("lookup eval report for dataset %q: %w", item.ID, err)
			}
		}
		summaries[item.ID] = summary
	}

	if len(reportsByDatasetID) > 0 {
		reports := make([]evalsvc.EvalReport, 0, len(reportsByDatasetID))
		for _, reportItem := range reportsByDatasetID {
			reports = append(reports, reportItem)
		}
		unresolvedCounts, err := a.evalReportBadCaseWithoutOpenFollowUpCounts(ctx, tenantID, reports)
		if err != nil {
			return nil, fmt.Errorf("summarize eval-report follow-up for tenant %q: %w", tenantID, err)
		}
		for datasetID, reportItem := range reportsByDatasetID {
			summary := summaries[datasetID]
			summary.UnresolvedFollowUpCount = unresolvedCounts[reportItem.ID]
			summary.NeedsFollowUp = summary.UnresolvedFollowUpCount > 0
			summaries[datasetID] = summary
		}
	}

	for datasetID, latestRun := range fallbackRunIDs {
		followUpSummary, err := a.evalRunFollowUpSummary(ctx, latestRun.ID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("summarize eval-run follow-up for dataset %q: %w", datasetID, err)
		}
		summary := summaries[datasetID]
		summary.UnresolvedFollowUpCount = followUpSummary.ItemWithoutOpenFollowUpCount
		summary.NeedsFollowUp = summary.UnresolvedFollowUpCount > 0
		summaries[datasetID] = summary
	}

	return summaries, nil
}

func (a *appHandler) collectLatestEvalRunsByDataset(ctx context.Context, tenantID string, targetDatasetIDs map[string]struct{}) (map[string]evalsvc.EvalRun, error) {
	latestRuns := make(map[string]evalsvc.EvalRun, len(targetDatasetIDs))
	if a.evalRuns == nil || len(targetDatasetIDs) == 0 {
		return latestRuns, nil
	}

	filter := evalsvc.RunListFilter{
		TenantID: tenantID,
		Limit:    100,
	}
	for {
		page, err := a.evalRuns.ListRuns(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("list eval runs for tenant %q: %w", tenantID, err)
		}
		for _, item := range page.Runs {
			if _, ok := targetDatasetIDs[item.DatasetID]; !ok {
				continue
			}
			if _, seen := latestRuns[item.DatasetID]; seen {
				continue
			}
			latestRuns[item.DatasetID] = item
			if len(latestRuns) == len(targetDatasetIDs) {
				return latestRuns, nil
			}
		}
		if !page.HasMore {
			return latestRuns, nil
		}
		filter.Offset = page.NextOffset
	}
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
