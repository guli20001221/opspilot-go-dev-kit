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
	EvalCaseID                      string                                          `json:"eval_case_id"`
	Title                           string                                          `json:"title"`
	SourceCaseID                    string                                          `json:"source_case_id"`
	SourceTaskID                    string                                          `json:"source_task_id,omitempty"`
	SourceReportID                  string                                          `json:"source_report_id,omitempty"`
	TraceID                         string                                          `json:"trace_id,omitempty"`
	VersionID                       string                                          `json:"version_id,omitempty"`
	LinkedCaseSummary               evalReportLinkedCaseSummaryResponse             `json:"linked_case_summary"`
	PreferredFollowUpAction         evalCaseFollowUpActionResponse                  `json:"preferred_follow_up_action"`
	PreferredPrimaryAction          evalCaseFollowUpActionResponse                  `json:"preferred_primary_action"`
	PreferredLinkedCaseAction       evalCaseFollowUpActionResponse                  `json:"preferred_linked_case_action"`
	PreferredSourceCaseProvenance   evalReportBadCaseSourceCaseProvenanceResponse   `json:"preferred_source_case_provenance"`
	PreferredSourceReportProvenance evalReportBadCaseSourceReportProvenanceResponse `json:"preferred_source_report_provenance"`
	PreferredSourceTaskProvenance   evalReportBadCaseSourceTaskProvenanceResponse   `json:"preferred_source_task_provenance"`
	PreferredTraceProvenance        evalReportBadCaseTraceProvenanceResponse        `json:"preferred_trace_provenance"`
	PreferredVersionProvenance      evalReportBadCaseVersionProvenanceResponse      `json:"preferred_version_provenance"`
	PreferredEvalProvenance         evalReportBadCaseEvalProvenanceResponse         `json:"preferred_eval_provenance"`
	PreferredFollowUpSliceAction    evalReportBadCaseFollowUpSliceActionResponse    `json:"preferred_follow_up_slice_action"`
}

type evalDatasetResponse struct {
	DatasetID                       string                                    `json:"dataset_id"`
	TenantID                        string                                    `json:"tenant_id"`
	Name                            string                                    `json:"name"`
	Description                     string                                    `json:"description,omitempty"`
	Status                          string                                    `json:"status"`
	CreatedBy                       string                                    `json:"created_by"`
	CreatedAt                       string                                    `json:"created_at"`
	UpdatedAt                       string                                    `json:"updated_at"`
	PublishedBy                     string                                    `json:"published_by,omitempty"`
	PublishedAt                     string                                    `json:"published_at,omitempty"`
	LatestRunID                     string                                    `json:"latest_run_id,omitempty"`
	LatestRunStatus                 string                                    `json:"latest_run_status,omitempty"`
	LatestReportID                  string                                    `json:"latest_report_id,omitempty"`
	LatestReportStatus              string                                    `json:"latest_report_status,omitempty"`
	PreferredLatestActivityAction   evalDatasetRecentRunPrimaryActionResponse `json:"preferred_latest_activity_action"`
	UnresolvedFollowUpCount         int                                       `json:"unresolved_follow_up_count"`
	NeedsFollowUp                   bool                                      `json:"needs_follow_up"`
	PreferredFollowUpAction         evalDatasetFollowUpActionResponse         `json:"preferred_follow_up_action"`
	FollowUpCaseSummary             evalDatasetFollowUpCaseSummaryResponse    `json:"follow_up_case_summary"`
	LinkedCaseSummary               evalDatasetLinkedCaseSummaryResponse      `json:"linked_case_summary"`
	RunBackedCaseSummary            evalDatasetLinkedCaseSummaryResponse      `json:"run_backed_case_summary"`
	PreferredRunBackedCaseAction    evalDatasetCaseQueueActionResponse        `json:"preferred_run_backed_case_action"`
	OpenFollowUpCaseCount           int                                       `json:"open_follow_up_case_count"`
	PreferredCaseQueueAction        evalDatasetCaseQueueActionResponse        `json:"preferred_case_queue_action"`
	DatasetFollowUpCaseSummary      evalDatasetFollowUpCaseSummaryResponse    `json:"dataset_follow_up_case_summary"`
	DatasetOpenFollowUpCaseCount    int                                       `json:"dataset_open_follow_up_case_count"`
	PreferredDatasetCaseQueueAction evalDatasetCaseQueueActionResponse        `json:"preferred_dataset_case_queue_action"`
	PreferredCaseHandoffAction      evalDatasetCaseQueueActionResponse        `json:"preferred_case_handoff_action"`
	RecentRuns                      []evalDatasetRecentRunResponse            `json:"recent_runs"`
	Items                           []evalDatasetItemResponse                 `json:"items"`
}

type evalDatasetSummaryResponse struct {
	DatasetID                       string                                    `json:"dataset_id"`
	TenantID                        string                                    `json:"tenant_id"`
	Name                            string                                    `json:"name"`
	Status                          string                                    `json:"status"`
	CreatedBy                       string                                    `json:"created_by"`
	CreatedAt                       string                                    `json:"created_at"`
	UpdatedAt                       string                                    `json:"updated_at"`
	ItemCount                       int                                       `json:"item_count"`
	LatestRunID                     string                                    `json:"latest_run_id,omitempty"`
	LatestRunStatus                 string                                    `json:"latest_run_status,omitempty"`
	LatestReportID                  string                                    `json:"latest_report_id,omitempty"`
	LatestReportStatus              string                                    `json:"latest_report_status,omitempty"`
	PreferredLatestActivityAction   evalDatasetRecentRunPrimaryActionResponse `json:"preferred_latest_activity_action"`
	UnresolvedFollowUpCount         int                                       `json:"unresolved_follow_up_count"`
	NeedsFollowUp                   bool                                      `json:"needs_follow_up"`
	PreferredFollowUpAction         evalDatasetFollowUpActionResponse         `json:"preferred_follow_up_action"`
	PreferredPrimaryAction          evalDatasetPrimaryActionResponse          `json:"preferred_primary_action"`
	OpenFollowUpCaseCount           int                                       `json:"open_follow_up_case_count"`
	LinkedCaseSummary               evalDatasetLinkedCaseSummaryResponse      `json:"linked_case_summary"`
	RunBackedCaseSummary            evalDatasetLinkedCaseSummaryResponse      `json:"run_backed_case_summary"`
	PreferredRunBackedCaseAction    evalDatasetCaseQueueActionResponse        `json:"preferred_run_backed_case_action"`
	PreferredCaseQueueAction        evalDatasetCaseQueueActionResponse        `json:"preferred_case_queue_action"`
	DatasetFollowUpCaseSummary      evalDatasetFollowUpCaseSummaryResponse    `json:"dataset_follow_up_case_summary"`
	DatasetOpenFollowUpCaseCount    int                                       `json:"dataset_open_follow_up_case_count"`
	PreferredDatasetCaseQueueAction evalDatasetCaseQueueActionResponse        `json:"preferred_dataset_case_queue_action"`
	PreferredCaseHandoffAction      evalDatasetCaseQueueActionResponse        `json:"preferred_case_handoff_action"`
}

type evalDatasetPrimaryActionResponse struct {
	Mode     string `json:"mode"`
	CaseID   string `json:"case_id,omitempty"`
	RunID    string `json:"run_id,omitempty"`
	ReportID string `json:"report_id,omitempty"`
}

type evalDatasetFollowUpActionResponse struct {
	Mode            string `json:"mode"`
	SourceDatasetID string `json:"source_dataset_id"`
	RunID           string `json:"run_id,omitempty"`
	ReportID        string `json:"report_id,omitempty"`
}

type evalDatasetCaseQueueActionResponse struct {
	Mode                string `json:"mode"`
	CaseID              string `json:"case_id,omitempty"`
	SourceEvalDatasetID string `json:"source_eval_dataset_id,omitempty"`
	SourceEvalReportID  string `json:"source_eval_report_id,omitempty"`
	SourceEvalRunID     string `json:"source_eval_run_id,omitempty"`
}

type evalDatasetFollowUpCaseSummaryResponse struct {
	FollowUpCaseCount        int    `json:"follow_up_case_count"`
	OpenFollowUpCaseCount    int    `json:"open_follow_up_case_count"`
	ClosedFollowUpCaseCount  int    `json:"closed_follow_up_case_count"`
	LatestFollowUpCaseID     string `json:"latest_follow_up_case_id,omitempty"`
	LatestFollowUpCaseStatus string `json:"latest_follow_up_case_status,omitempty"`
}

type evalDatasetLinkedCaseSummaryResponse struct {
	TotalCaseCount   int    `json:"total_case_count"`
	OpenCaseCount    int    `json:"open_case_count"`
	LatestCaseID     string `json:"latest_case_id,omitempty"`
	LatestCaseStatus string `json:"latest_case_status,omitempty"`
	LatestAssignedTo string `json:"latest_assigned_to,omitempty"`
}

type evalDatasetRecentRunResponse struct {
	RunID                        string                                    `json:"run_id"`
	Status                       string                                    `json:"status"`
	CreatedAt                    string                                    `json:"created_at"`
	UpdatedAt                    string                                    `json:"updated_at"`
	FinishedAt                   string                                    `json:"finished_at,omitempty"`
	ItemWithoutOpenFollowUpCount int                                       `json:"item_without_open_follow_up_count"`
	NeedsFollowUp                bool                                      `json:"needs_follow_up"`
	ReportID                     string                                    `json:"report_id,omitempty"`
	ReportStatus                 string                                    `json:"report_status,omitempty"`
	PreferredPrimaryAction       evalDatasetRecentRunPrimaryActionResponse `json:"preferred_primary_action"`
	PreferredFollowUpAction      evalDatasetFollowUpActionResponse         `json:"preferred_follow_up_action"`
	LinkedCaseSummary            evalDatasetLinkedCaseSummaryResponse      `json:"linked_case_summary"`
	PreferredCaseAction          evalDatasetCaseQueueActionResponse        `json:"preferred_case_action"`
}

type evalDatasetRecentRunPrimaryActionResponse struct {
	Mode     string `json:"mode"`
	CaseID   string `json:"case_id,omitempty"`
	RunID    string `json:"run_id,omitempty"`
	ReportID string `json:"report_id,omitempty"`
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

	writeJSON(w, http.StatusCreated, newEvalDatasetResponse(item, evalDatasetLatestRunSummary{}, evalDatasetLinkedCaseSummaryResponse{}, evalDatasetLinkedCaseSummaryResponse{}, nil, nil))
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
	linkedCaseSummary, err := a.evalDatasetLinkedCaseSummary(r.Context(), item.TenantID, latestRun)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_linked_case_summary_failed", err.Error())
		return
	}
	runBackedCaseSummary, err := a.evalDatasetRunBackedCaseSummary(r.Context(), item.TenantID, latestRun)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_run_backed_case_summary_failed", err.Error())
		return
	}
	recentRuns, err := a.evalDatasetRecentRuns(r.Context(), item.TenantID, item.ID, 5)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_recent_runs_failed", err.Error())
		return
	}
	itemFollowUpSummaries, err := a.evalDatasetItemFollowUpSummaries(r.Context(), item.TenantID, item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_dataset_item_linked_case_summary_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item, latestRun, linkedCaseSummary, runBackedCaseSummary, recentRuns, itemFollowUpSummaries))
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

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item, evalDatasetLatestRunSummary{}, evalDatasetLinkedCaseSummaryResponse{}, evalDatasetLinkedCaseSummaryResponse{}, nil, nil))
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

	writeJSON(w, http.StatusOK, newEvalDatasetResponse(item, evalDatasetLatestRunSummary{}, evalDatasetLinkedCaseSummaryResponse{}, evalDatasetLinkedCaseSummaryResponse{}, nil, nil))
}

func newEvalDatasetResponse(item evalsvc.EvalDataset, latestRun evalDatasetLatestRunSummary, linkedCaseSummary evalDatasetLinkedCaseSummaryResponse, runBackedCaseSummary evalDatasetLinkedCaseSummaryResponse, recentRuns []evalDatasetRecentRunResponse, itemFollowUpSummaries map[string]casesvc.EvalCaseFollowUpSummary) evalDatasetResponse {
	resp := evalDatasetResponse{
		DatasetID:                       item.ID,
		TenantID:                        item.TenantID,
		Name:                            item.Name,
		Description:                     item.Description,
		Status:                          item.Status,
		CreatedBy:                       item.CreatedBy,
		CreatedAt:                       item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                       item.UpdatedAt.Format(time.RFC3339Nano),
		LatestRunID:                     latestRun.LatestRunID,
		LatestRunStatus:                 latestRun.LatestRunStatus,
		LatestReportID:                  latestRun.LatestReportID,
		LatestReportStatus:              latestRun.LatestReportStatus,
		PreferredLatestActivityAction:   newEvalDatasetLatestActivityActionResponse(latestRun),
		UnresolvedFollowUpCount:         latestRun.UnresolvedFollowUpCount,
		NeedsFollowUp:                   latestRun.NeedsFollowUp,
		PreferredFollowUpAction:         newEvalDatasetFollowUpActionResponse(item.ID, latestRun),
		FollowUpCaseSummary:             newEvalDatasetFollowUpCaseSummaryResponse(latestRun),
		LinkedCaseSummary:               linkedCaseSummary,
		RunBackedCaseSummary:            runBackedCaseSummary,
		PreferredRunBackedCaseAction:    newEvalDatasetRunBackedCaseActionResponse(latestRun, runBackedCaseSummary),
		OpenFollowUpCaseCount:           latestRun.OpenFollowUpCaseCount,
		PreferredCaseQueueAction:        newEvalDatasetCaseQueueActionResponse(latestRun),
		DatasetFollowUpCaseSummary:      newEvalDatasetDatasetFollowUpCaseSummaryResponse(latestRun),
		DatasetOpenFollowUpCaseCount:    latestRun.DatasetOpenFollowUpCaseCount,
		PreferredDatasetCaseQueueAction: newEvalDatasetDatasetCaseQueueActionResponse(item.ID, latestRun),
		PreferredCaseHandoffAction:      newEvalDatasetPreferredCaseHandoffActionResponse(item.ID, latestRun),
		RecentRuns:                      make([]evalDatasetRecentRunResponse, 0, len(recentRuns)),
		Items:                           make([]evalDatasetItemResponse, 0, len(item.Items)),
	}
	resp.RecentRuns = append(resp.RecentRuns, recentRuns...)
	if item.PublishedBy != "" {
		resp.PublishedBy = item.PublishedBy
	}
	if !item.PublishedAt.IsZero() {
		resp.PublishedAt = item.PublishedAt.Format(time.RFC3339Nano)
	}
	for _, member := range item.Items {
		followUpSummary := itemFollowUpSummaries[member.EvalCaseID]
		resp.Items = append(resp.Items, evalDatasetItemResponse{
			EvalCaseID:     member.EvalCaseID,
			Title:          member.Title,
			SourceCaseID:   member.SourceCaseID,
			SourceTaskID:   member.SourceTaskID,
			SourceReportID: member.SourceReportID,
			TraceID:        member.TraceID,
			VersionID:      member.VersionID,
			LinkedCaseSummary: evalReportLinkedCaseSummaryResponse{
				TotalCaseCount:   followUpSummary.FollowUpCaseCount,
				OpenCaseCount:    followUpSummary.OpenFollowUpCaseCount,
				LatestCaseID:     followUpSummary.LatestFollowUpCaseID,
				LatestCaseStatus: followUpSummary.LatestFollowUpCaseStatus,
			},
			PreferredFollowUpAction: newEvalCaseFollowUpActionResponseFromSummary(
				member.EvalCaseID,
				followUpSummary.OpenFollowUpCaseCount,
				followUpSummary.LatestFollowUpCaseID,
			),
			PreferredPrimaryAction: newEvalDatasetItemPrimaryActionResponse(
				member.EvalCaseID,
				followUpSummary,
			),
			PreferredLinkedCaseAction: newEvalCaseLinkedCaseActionResponseFromSummary(
				member.EvalCaseID,
				followUpSummary.FollowUpCaseCount,
				followUpSummary.OpenFollowUpCaseCount,
				followUpSummary.LatestFollowUpCaseID,
				followUpSummary.LatestFollowUpCaseStatus,
			),
			PreferredSourceCaseProvenance:   datasetItemSourceCaseProvenance(member.SourceCaseID),
			PreferredSourceReportProvenance: datasetItemSourceReportProvenance(member.SourceReportID),
			PreferredSourceTaskProvenance:   datasetItemSourceTaskProvenance(member.SourceTaskID),
			PreferredTraceProvenance:        datasetItemTraceProvenance(member.TraceID),
			PreferredVersionProvenance:      datasetItemVersionProvenance(member.VersionID),
			PreferredEvalProvenance:         datasetItemEvalProvenance(member.EvalCaseID),
			PreferredFollowUpSliceAction:    datasetItemFollowUpSliceAction(member.EvalCaseID),
		})
	}

	return resp
}

func newEvalDatasetItemPrimaryActionResponse(evalCaseID string, summary casesvc.EvalCaseFollowUpSummary) evalCaseFollowUpActionResponse {
	linkedAction := newEvalCaseLinkedCaseActionResponseFromSummary(
		evalCaseID,
		summary.FollowUpCaseCount,
		summary.OpenFollowUpCaseCount,
		summary.LatestFollowUpCaseID,
		summary.LatestFollowUpCaseStatus,
	)
	if linkedAction.Mode != "none" {
		return linkedAction
	}
	return newEvalCaseFollowUpActionResponseFromSummary(
		evalCaseID,
		summary.OpenFollowUpCaseCount,
		summary.LatestFollowUpCaseID,
	)
}

func (a *appHandler) evalDatasetItemFollowUpSummaries(ctx context.Context, tenantID string, item evalsvc.EvalDataset) (map[string]casesvc.EvalCaseFollowUpSummary, error) {
	summaries := make(map[string]casesvc.EvalCaseFollowUpSummary, len(item.Items))
	if a.cases == nil || len(item.Items) == 0 {
		return summaries, nil
	}

	evalCaseIDs := make([]string, 0, len(item.Items))
	seen := make(map[string]struct{}, len(item.Items))
	for _, member := range item.Items {
		if member.EvalCaseID == "" {
			continue
		}
		if _, ok := seen[member.EvalCaseID]; ok {
			continue
		}
		seen[member.EvalCaseID] = struct{}{}
		evalCaseIDs = append(evalCaseIDs, member.EvalCaseID)
	}
	if len(evalCaseIDs) == 0 {
		return summaries, nil
	}

	return a.cases.SummarizeBySourceEvalCaseIDs(ctx, tenantID, evalCaseIDs)
}

type evalDatasetLatestRunSummary struct {
	LatestRunID                     string
	LatestRunStatus                 string
	LatestReportID                  string
	LatestReportStatus              string
	UnresolvedFollowUpCount         int
	NeedsFollowUp                   bool
	FollowUpCaseCount               int
	OpenFollowUpCaseCount           int
	LatestFollowUpCaseID            string
	LatestFollowUpCaseStatus        string
	DatasetFollowUpCaseCount        int
	DatasetOpenFollowUpCaseCount    int
	DatasetLatestFollowUpCaseID     string
	DatasetLatestFollowUpCaseStatus string
	DatasetLatestFollowUpCaseOwner  string
}

func newEvalDatasetSummaryResponse(item evalsvc.EvalDatasetSummary, latestRun evalDatasetLatestRunSummary, linkedCaseSummary evalDatasetLinkedCaseSummaryResponse, runBackedCaseSummary evalDatasetLinkedCaseSummaryResponse) evalDatasetSummaryResponse {
	return evalDatasetSummaryResponse{
		DatasetID:                       item.ID,
		TenantID:                        item.TenantID,
		Name:                            item.Name,
		Status:                          item.Status,
		CreatedBy:                       item.CreatedBy,
		CreatedAt:                       item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                       item.UpdatedAt.Format(time.RFC3339Nano),
		ItemCount:                       item.ItemCount,
		LatestRunID:                     latestRun.LatestRunID,
		LatestRunStatus:                 latestRun.LatestRunStatus,
		LatestReportID:                  latestRun.LatestReportID,
		LatestReportStatus:              latestRun.LatestReportStatus,
		PreferredLatestActivityAction:   newEvalDatasetLatestActivityActionResponse(latestRun),
		UnresolvedFollowUpCount:         latestRun.UnresolvedFollowUpCount,
		NeedsFollowUp:                   latestRun.NeedsFollowUp,
		PreferredFollowUpAction:         newEvalDatasetFollowUpActionResponse(item.ID, latestRun),
		PreferredPrimaryAction:          newEvalDatasetPrimaryActionResponse(item.ID, latestRun),
		OpenFollowUpCaseCount:           latestRun.OpenFollowUpCaseCount,
		LinkedCaseSummary:               linkedCaseSummary,
		RunBackedCaseSummary:            runBackedCaseSummary,
		PreferredRunBackedCaseAction:    newEvalDatasetRunBackedCaseActionResponse(latestRun, runBackedCaseSummary),
		PreferredCaseQueueAction:        newEvalDatasetCaseQueueActionResponse(latestRun),
		DatasetFollowUpCaseSummary:      newEvalDatasetDatasetFollowUpCaseSummaryResponse(latestRun),
		DatasetOpenFollowUpCaseCount:    latestRun.DatasetOpenFollowUpCaseCount,
		PreferredDatasetCaseQueueAction: newEvalDatasetDatasetCaseQueueActionResponse(item.ID, latestRun),
		PreferredCaseHandoffAction:      newEvalDatasetPreferredCaseHandoffActionResponse(item.ID, latestRun),
	}
}

func newEvalDatasetPrimaryActionResponse(datasetID string, latestRun evalDatasetLatestRunSummary) evalDatasetPrimaryActionResponse {
	action := evalDatasetPrimaryActionResponse{
		Mode:  "none",
		RunID: latestRun.LatestRunID,
	}
	caseAction := newEvalDatasetPreferredCaseHandoffActionResponse(datasetID, latestRun)
	if caseAction.Mode == "open_existing_case" && caseAction.CaseID != "" {
		action.Mode = "open_existing_case"
		action.CaseID = caseAction.CaseID
		return action
	}
	if caseAction.Mode == "open_existing_queue" {
		action.Mode = "open_existing_queue"
		return action
	}

	followUpAction := newEvalDatasetFollowUpActionResponse(datasetID, latestRun)
	switch followUpAction.Mode {
	case "open_latest_report_queue":
		action.Mode = "open_latest_report_queue"
		action.ReportID = followUpAction.ReportID
		return action
	case "open_latest_run_queue":
		action.Mode = "open_latest_run_queue"
		return action
	}
	if latestRun.LatestReportID != "" {
		action.Mode = "open_report"
		action.ReportID = latestRun.LatestReportID
		return action
	}
	if latestRun.LatestRunID != "" {
		action.Mode = "open_run"
		return action
	}
	return action
}

func newEvalDatasetLatestActivityActionResponse(latestRun evalDatasetLatestRunSummary) evalDatasetRecentRunPrimaryActionResponse {
	action := evalDatasetRecentRunPrimaryActionResponse{
		Mode:  "none",
		RunID: latestRun.LatestRunID,
	}
	if latestRun.LatestReportID != "" {
		action.Mode = "open_report"
		action.ReportID = latestRun.LatestReportID
		return action
	}
	if latestRun.LatestRunID != "" {
		action.Mode = "open_run"
	}
	return action
}

func newEvalDatasetFollowUpActionResponse(datasetID string, latestRun evalDatasetLatestRunSummary) evalDatasetFollowUpActionResponse {
	action := evalDatasetFollowUpActionResponse{
		Mode:            "none",
		SourceDatasetID: datasetID,
	}
	if !latestRun.NeedsFollowUp {
		return action
	}
	if latestRun.LatestReportID != "" {
		action.Mode = "open_latest_report_queue"
		action.ReportID = latestRun.LatestReportID
		return action
	}
	if latestRun.LatestRunID != "" {
		action.Mode = "open_latest_run_queue"
		action.RunID = latestRun.LatestRunID
	}
	return action
}

func newEvalDatasetFollowUpCaseSummaryResponse(latestRun evalDatasetLatestRunSummary) evalDatasetFollowUpCaseSummaryResponse {
	closedFollowUpCaseCount := latestRun.FollowUpCaseCount - latestRun.OpenFollowUpCaseCount
	if closedFollowUpCaseCount < 0 {
		closedFollowUpCaseCount = 0
	}

	return evalDatasetFollowUpCaseSummaryResponse{
		FollowUpCaseCount:        latestRun.FollowUpCaseCount,
		OpenFollowUpCaseCount:    latestRun.OpenFollowUpCaseCount,
		ClosedFollowUpCaseCount:  closedFollowUpCaseCount,
		LatestFollowUpCaseID:     latestRun.LatestFollowUpCaseID,
		LatestFollowUpCaseStatus: latestRun.LatestFollowUpCaseStatus,
	}
}

func newEvalDatasetDatasetFollowUpCaseSummaryResponse(latestRun evalDatasetLatestRunSummary) evalDatasetFollowUpCaseSummaryResponse {
	closedFollowUpCaseCount := latestRun.DatasetFollowUpCaseCount - latestRun.DatasetOpenFollowUpCaseCount
	if closedFollowUpCaseCount < 0 {
		closedFollowUpCaseCount = 0
	}

	return evalDatasetFollowUpCaseSummaryResponse{
		FollowUpCaseCount:        latestRun.DatasetFollowUpCaseCount,
		OpenFollowUpCaseCount:    latestRun.DatasetOpenFollowUpCaseCount,
		ClosedFollowUpCaseCount:  closedFollowUpCaseCount,
		LatestFollowUpCaseID:     latestRun.DatasetLatestFollowUpCaseID,
		LatestFollowUpCaseStatus: latestRun.DatasetLatestFollowUpCaseStatus,
	}
}

func newEvalDatasetLinkedCaseSummaryResponse(latestRun evalDatasetLatestRunSummary) evalDatasetLinkedCaseSummaryResponse {
	return evalDatasetLinkedCaseSummaryResponse{
		TotalCaseCount:   latestRun.DatasetFollowUpCaseCount,
		OpenCaseCount:    latestRun.DatasetOpenFollowUpCaseCount,
		LatestCaseID:     latestRun.DatasetLatestFollowUpCaseID,
		LatestCaseStatus: latestRun.DatasetLatestFollowUpCaseStatus,
		LatestAssignedTo: latestRun.DatasetLatestFollowUpCaseOwner,
	}
}

func newEvalDatasetCaseQueueActionResponse(latestRun evalDatasetLatestRunSummary) evalDatasetCaseQueueActionResponse {
	action := evalDatasetCaseQueueActionResponse{Mode: "none"}
	if latestRun.LatestReportID == "" || latestRun.OpenFollowUpCaseCount <= 0 {
		return action
	}
	action.SourceEvalReportID = latestRun.LatestReportID
	if latestRun.LatestFollowUpCaseID != "" && latestRun.LatestFollowUpCaseStatus == casesvc.StatusOpen {
		action.Mode = "open_existing_case"
		action.CaseID = latestRun.LatestFollowUpCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalDatasetDatasetCaseQueueActionResponse(datasetID string, latestRun evalDatasetLatestRunSummary) evalDatasetCaseQueueActionResponse {
	action := evalDatasetCaseQueueActionResponse{Mode: "none"}
	if datasetID == "" || latestRun.DatasetOpenFollowUpCaseCount <= 0 {
		return action
	}
	action.SourceEvalDatasetID = datasetID
	if latestRun.DatasetLatestFollowUpCaseID != "" && latestRun.DatasetLatestFollowUpCaseStatus == casesvc.StatusOpen {
		action.Mode = "open_existing_case"
		action.CaseID = latestRun.DatasetLatestFollowUpCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalDatasetPreferredCaseHandoffActionResponse(datasetID string, latestRun evalDatasetLatestRunSummary) evalDatasetCaseQueueActionResponse {
	datasetAction := newEvalDatasetDatasetCaseQueueActionResponse(datasetID, latestRun)
	if datasetAction.Mode != "none" {
		return datasetAction
	}
	return newEvalDatasetCaseQueueActionResponse(latestRun)
}

func newEvalDatasetRunBackedCaseActionResponse(latestRun evalDatasetLatestRunSummary, summary evalDatasetLinkedCaseSummaryResponse) evalDatasetCaseQueueActionResponse {
	action := evalDatasetCaseQueueActionResponse{Mode: "none"}
	if latestRun.LatestRunID == "" || summary.OpenCaseCount <= 0 {
		return action
	}

	action.SourceEvalRunID = latestRun.LatestRunID
	if summary.LatestCaseID != "" && summary.LatestCaseStatus == casesvc.StatusOpen {
		action.Mode = "open_existing_case"
		action.CaseID = summary.LatestCaseID
		return action
	}

	action.Mode = "open_existing_queue"
	return action
}

func (a *appHandler) evalDatasetRecentRuns(ctx context.Context, tenantID string, datasetID string, limit int) ([]evalDatasetRecentRunResponse, error) {
	if a.evalRuns == nil || limit <= 0 {
		return nil, nil
	}

	page, err := a.evalRuns.ListRuns(ctx, evalsvc.RunListFilter{
		TenantID:  tenantID,
		DatasetID: datasetID,
		Limit:     limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list recent eval runs for dataset %q: %w", datasetID, err)
	}

	rows := make([]evalDatasetRecentRunResponse, 0, len(page.Runs))
	for _, run := range page.Runs {
		resp := evalDatasetRecentRunResponse{
			RunID:     run.ID,
			Status:    run.Status,
			CreatedAt: run.CreatedAt.Format(time.RFC3339Nano),
			UpdatedAt: run.UpdatedAt.Format(time.RFC3339Nano),
		}
		if !run.FinishedAt.IsZero() {
			resp.FinishedAt = run.FinishedAt.Format(time.RFC3339Nano)
		}
		if run.Status == evalsvc.RunStatusSucceeded || run.Status == evalsvc.RunStatusFailed {
			followUpSummary, err := a.evalRunFollowUpSummary(ctx, run.ID, tenantID)
			if err != nil {
				return nil, fmt.Errorf("summarize follow-up for eval run %q: %w", run.ID, err)
			}
			resp.ItemWithoutOpenFollowUpCount = followUpSummary.ItemWithoutOpenFollowUpCount
			resp.NeedsFollowUp = followUpSummary.ItemWithoutOpenFollowUpCount > 0
			resp.LinkedCaseSummary = evalDatasetLinkedCaseSummaryResponse{
				TotalCaseCount:   followUpSummary.RunSummary.FollowUpCaseCount,
				OpenCaseCount:    followUpSummary.RunSummary.OpenFollowUpCaseCount,
				LatestCaseID:     followUpSummary.RunSummary.LatestFollowUpCaseID,
				LatestCaseStatus: followUpSummary.RunSummary.LatestFollowUpCaseStatus,
				LatestAssignedTo: followUpSummary.RunSummary.LatestFollowUpAssignedTo,
			}
			resp.PreferredCaseAction = newEvalDatasetRecentRunCaseActionResponse(run.ID, resp.LinkedCaseSummary)
			if a.evalReports != nil {
				reportID := evalsvc.EvalReportIDFromRunID(run.ID)
				reportItem, err := a.evalReports.GetEvalReport(ctx, reportID)
				switch {
				case err == nil && reportItem.TenantID == tenantID:
					resp.ReportID = reportItem.ID
					resp.ReportStatus = reportItem.Status
				case err == nil:
					// Ignore cross-tenant report rows.
				case errors.Is(err, evalsvc.ErrEvalReportNotFound):
					// Run can exist before durable report materialization.
				default:
					return nil, fmt.Errorf("lookup eval report for run %q: %w", run.ID, err)
				}
			}
			resp.PreferredFollowUpAction = newEvalDatasetRecentRunFollowUpActionResponse(run.ID, resp.ReportID, resp.NeedsFollowUp)
		}
		resp.PreferredPrimaryAction = newEvalDatasetRecentRunPrimaryActionResponse(run.ID, resp.ReportID, resp.PreferredCaseAction, resp.PreferredFollowUpAction)
		rows = append(rows, resp)
	}

	return rows, nil
}

func newEvalDatasetRecentRunFollowUpActionResponse(runID string, reportID string, needsFollowUp bool) evalDatasetFollowUpActionResponse {
	action := evalDatasetFollowUpActionResponse{Mode: "none"}
	if !needsFollowUp {
		return action
	}
	if reportID != "" {
		action.Mode = "open_latest_report_queue"
		action.ReportID = reportID
		return action
	}
	if runID != "" {
		action.Mode = "open_latest_run_queue"
		action.RunID = runID
	}
	return action
}

func newEvalDatasetRecentRunCaseActionResponse(runID string, summary evalDatasetLinkedCaseSummaryResponse) evalDatasetCaseQueueActionResponse {
	action := evalDatasetCaseQueueActionResponse{Mode: "none"}
	if runID == "" || summary.OpenCaseCount <= 0 {
		return action
	}

	action.SourceEvalRunID = runID
	if summary.LatestCaseID != "" && summary.LatestCaseStatus == casesvc.StatusOpen {
		action.Mode = "open_existing_case"
		action.CaseID = summary.LatestCaseID
		return action
	}

	action.Mode = "open_existing_queue"
	return action
}

func newEvalDatasetRecentRunPrimaryActionResponse(runID string, reportID string, caseAction evalDatasetCaseQueueActionResponse, followUpAction evalDatasetFollowUpActionResponse) evalDatasetRecentRunPrimaryActionResponse {
	action := evalDatasetRecentRunPrimaryActionResponse{
		Mode:  "none",
		RunID: runID,
	}
	if caseAction.Mode == "open_existing_case" && caseAction.CaseID != "" {
		action.Mode = "open_existing_case"
		action.CaseID = caseAction.CaseID
		return action
	}
	if caseAction.Mode == "open_existing_queue" {
		action.Mode = "open_existing_queue"
		return action
	}
	switch followUpAction.Mode {
	case "open_latest_report_queue":
		action.Mode = "open_latest_report_queue"
		action.ReportID = followUpAction.ReportID
		return action
	case "open_latest_run_queue":
		action.Mode = "open_latest_run_queue"
		return action
	}
	if reportID != "" {
		action.Mode = "open_report"
		action.ReportID = reportID
		return action
	}
	if runID != "" {
		action.Mode = "open_run"
		return action
	}
	return action
}

func datasetItemSourceCaseProvenance(sourceCaseID string) evalReportBadCaseSourceCaseProvenanceResponse {
	if sourceCaseID != "" {
		return evalReportBadCaseSourceCaseProvenanceResponse{Mode: "open", CaseID: sourceCaseID}
	}
	return evalReportBadCaseSourceCaseProvenanceResponse{Mode: "none"}
}

func datasetItemSourceReportProvenance(sourceReportID string) evalReportBadCaseSourceReportProvenanceResponse {
	if sourceReportID != "" {
		return evalReportBadCaseSourceReportProvenanceResponse{Mode: "open_api", ReportID: sourceReportID}
	}
	return evalReportBadCaseSourceReportProvenanceResponse{Mode: "none"}
}

func datasetItemSourceTaskProvenance(sourceTaskID string) evalReportBadCaseSourceTaskProvenanceResponse {
	if sourceTaskID != "" {
		return evalReportBadCaseSourceTaskProvenanceResponse{Mode: "open_api", TaskID: sourceTaskID}
	}
	return evalReportBadCaseSourceTaskProvenanceResponse{Mode: "none"}
}

func datasetItemTraceProvenance(traceID string) evalReportBadCaseTraceProvenanceResponse {
	if traceID != "" {
		return evalReportBadCaseTraceProvenanceResponse{Mode: "open", TraceID: traceID}
	}
	return evalReportBadCaseTraceProvenanceResponse{Mode: "none"}
}

func datasetItemVersionProvenance(versionID string) evalReportBadCaseVersionProvenanceResponse {
	if versionID != "" {
		return evalReportBadCaseVersionProvenanceResponse{Mode: "open", VersionID: versionID}
	}
	return evalReportBadCaseVersionProvenanceResponse{Mode: "none"}
}

func datasetItemEvalProvenance(evalCaseID string) evalReportBadCaseEvalProvenanceResponse {
	if evalCaseID != "" {
		return evalReportBadCaseEvalProvenanceResponse{Mode: "open", EvalCaseID: evalCaseID}
	}
	return evalReportBadCaseEvalProvenanceResponse{Mode: "none"}
}

func datasetItemFollowUpSliceAction(evalCaseID string) evalReportBadCaseFollowUpSliceActionResponse {
	if evalCaseID != "" {
		return evalReportBadCaseFollowUpSliceActionResponse{Mode: "open", SourceEvalCaseID: evalCaseID}
	}
	return evalReportBadCaseFollowUpSliceActionResponse{Mode: "none"}
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
		linkedCaseSummaries, err := a.evalDatasetLinkedCaseSummaries(ctx, filter.TenantID, latestRunSummaries)
		if err != nil {
			return listEvalDatasetsResponse{}, err
		}
		runBackedCaseSummaries, err := a.evalDatasetRunBackedCaseSummaries(ctx, filter.TenantID, latestRunSummaries)
		if err != nil {
			return listEvalDatasetsResponse{}, err
		}
		resp := listEvalDatasetsResponse{
			Datasets: buildEvalDatasetListRows(page.Datasets, latestRunSummaries, linkedCaseSummaries, runBackedCaseSummaries),
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
	linkedCaseSummaries, err := a.evalDatasetLinkedCaseSummaries(ctx, filter.TenantID, latestRunSummaries)
	if err != nil {
		return listEvalDatasetsResponse{}, err
	}
	runBackedCaseSummaries, err := a.evalDatasetRunBackedCaseSummaries(ctx, filter.TenantID, latestRunSummaries)
	if err != nil {
		return listEvalDatasetsResponse{}, err
	}
	rows := buildEvalDatasetListRows(candidates, latestRunSummaries, linkedCaseSummaries, runBackedCaseSummaries)
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

func buildEvalDatasetListRows(items []evalsvc.EvalDatasetSummary, latestRunSummaries map[string]evalDatasetLatestRunSummary, linkedCaseSummaries map[string]evalDatasetLinkedCaseSummaryResponse, runBackedCaseSummaries map[string]evalDatasetLinkedCaseSummaryResponse) []evalDatasetSummaryResponse {
	rows := make([]evalDatasetSummaryResponse, 0, len(items))
	for _, item := range items {
		rows = append(rows, newEvalDatasetSummaryResponse(item, latestRunSummaries[item.ID], linkedCaseSummaries[item.ID], runBackedCaseSummaries[item.ID]))
	}
	return rows
}

func (a *appHandler) evalDatasetLinkedCaseSummaries(ctx context.Context, tenantID string, latestRunSummaries map[string]evalDatasetLatestRunSummary) (map[string]evalDatasetLinkedCaseSummaryResponse, error) {
	summaries := make(map[string]evalDatasetLinkedCaseSummaryResponse, len(latestRunSummaries))
	for datasetID, latestRun := range latestRunSummaries {
		summary, err := a.evalDatasetLinkedCaseSummary(ctx, tenantID, latestRun)
		if err != nil {
			return nil, fmt.Errorf("build linked case summary for dataset %q: %w", datasetID, err)
		}
		summaries[datasetID] = summary
	}
	return summaries, nil
}

func (a *appHandler) evalDatasetRunBackedCaseSummaries(ctx context.Context, tenantID string, latestRunSummaries map[string]evalDatasetLatestRunSummary) (map[string]evalDatasetLinkedCaseSummaryResponse, error) {
	summaries := make(map[string]evalDatasetLinkedCaseSummaryResponse, len(latestRunSummaries))
	if a.cases == nil || tenantID == "" || len(latestRunSummaries) == 0 {
		return summaries, nil
	}

	runIDs := make([]string, 0, len(latestRunSummaries))
	runIDSet := make(map[string]struct{}, len(latestRunSummaries))
	for _, latestRun := range latestRunSummaries {
		if latestRun.LatestRunID == "" {
			continue
		}
		if _, ok := runIDSet[latestRun.LatestRunID]; ok {
			continue
		}
		runIDSet[latestRun.LatestRunID] = struct{}{}
		runIDs = append(runIDs, latestRun.LatestRunID)
	}
	if len(runIDs) == 0 {
		return summaries, nil
	}

	runSummaries, err := a.cases.SummarizeBySourceEvalRunIDs(ctx, tenantID, runIDs)
	if err != nil {
		return nil, fmt.Errorf("summarize run-backed cases for tenant %q: %w", tenantID, err)
	}

	for datasetID, latestRun := range latestRunSummaries {
		runSummary := runSummaries[latestRun.LatestRunID]
		summaries[datasetID] = evalDatasetLinkedCaseSummaryResponse{
			TotalCaseCount:   runSummary.FollowUpCaseCount,
			OpenCaseCount:    runSummary.OpenFollowUpCaseCount,
			LatestCaseID:     runSummary.LatestFollowUpCaseID,
			LatestCaseStatus: runSummary.LatestFollowUpCaseStatus,
			LatestAssignedTo: runSummary.LatestFollowUpAssignedTo,
		}
	}
	return summaries, nil
}

func (a *appHandler) evalDatasetLinkedCaseSummary(ctx context.Context, tenantID string, latestRun evalDatasetLatestRunSummary) (evalDatasetLinkedCaseSummaryResponse, error) {
	summary := newEvalDatasetLinkedCaseSummaryResponse(latestRun)
	if a.cases == nil || tenantID == "" || latestRun.DatasetLatestFollowUpCaseID == "" {
		return summary, nil
	}

	latestCase, err := a.cases.GetCase(ctx, latestRun.DatasetLatestFollowUpCaseID)
	if err != nil {
		if errors.Is(err, casesvc.ErrCaseNotFound) {
			return summary, nil
		}
		return evalDatasetLinkedCaseSummaryResponse{}, fmt.Errorf("get linked case %q: %w", latestRun.DatasetLatestFollowUpCaseID, err)
	}
	if latestCase.TenantID != tenantID {
		return summary, nil
	}
	summary.LatestAssignedTo = latestCase.AssignedTo
	return summary, nil
}

func (a *appHandler) evalDatasetRunBackedCaseSummary(ctx context.Context, tenantID string, latestRun evalDatasetLatestRunSummary) (evalDatasetLinkedCaseSummaryResponse, error) {
	if a.cases == nil || tenantID == "" || latestRun.LatestRunID == "" {
		return evalDatasetLinkedCaseSummaryResponse{}, nil
	}

	summaries, err := a.cases.SummarizeBySourceEvalRunIDs(ctx, tenantID, []string{latestRun.LatestRunID})
	if err != nil {
		return evalDatasetLinkedCaseSummaryResponse{}, fmt.Errorf("summarize run-backed cases for run %q: %w", latestRun.LatestRunID, err)
	}
	runSummary := summaries[latestRun.LatestRunID]
	return evalDatasetLinkedCaseSummaryResponse{
		TotalCaseCount:   runSummary.FollowUpCaseCount,
		OpenCaseCount:    runSummary.OpenFollowUpCaseCount,
		LatestCaseID:     runSummary.LatestFollowUpCaseID,
		LatestCaseStatus: runSummary.LatestFollowUpCaseStatus,
		LatestAssignedTo: runSummary.LatestFollowUpAssignedTo,
	}, nil
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
		reportIDs := make([]string, 0, len(reportsByDatasetID))
		for _, reportItem := range reportsByDatasetID {
			reports = append(reports, reportItem)
			reportIDs = append(reportIDs, reportItem.ID)
		}
		unresolvedCounts, err := a.evalReportBadCaseWithoutOpenFollowUpCounts(ctx, tenantID, reports)
		if err != nil {
			return nil, fmt.Errorf("summarize eval-report follow-up for tenant %q: %w", tenantID, err)
		}
		reportFollowUpSummaries := map[string]casesvc.EvalReportFollowUpSummary{}
		if a.cases != nil {
			reportFollowUpSummaries, err = a.cases.SummarizeBySourceEvalReportIDs(ctx, tenantID, reportIDs)
			if err != nil {
				return nil, fmt.Errorf("summarize dataset report case queue for tenant %q: %w", tenantID, err)
			}
		}
		for datasetID, reportItem := range reportsByDatasetID {
			summary := summaries[datasetID]
			summary.UnresolvedFollowUpCount = unresolvedCounts[reportItem.ID]
			summary.NeedsFollowUp = summary.UnresolvedFollowUpCount > 0
			reportFollowUpSummary := reportFollowUpSummaries[reportItem.ID]
			summary.FollowUpCaseCount = reportFollowUpSummary.FollowUpCaseCount
			summary.OpenFollowUpCaseCount = reportFollowUpSummary.OpenFollowUpCaseCount
			summary.LatestFollowUpCaseID = reportFollowUpSummary.LatestFollowUpCaseID
			summary.LatestFollowUpCaseStatus = reportFollowUpSummary.LatestFollowUpCaseStatus
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

	datasetCaseSummaries, err := a.evalDatasetCaseQueueSummaries(ctx, tenantID, targetDatasetIDs)
	if err != nil {
		return nil, err
	}
	for datasetID, queueSummary := range datasetCaseSummaries {
		summary := summaries[datasetID]
		summary.DatasetFollowUpCaseCount = queueSummary.FollowUpCaseCount
		summary.DatasetOpenFollowUpCaseCount = queueSummary.OpenFollowUpCaseCount
		summary.DatasetLatestFollowUpCaseID = queueSummary.LatestFollowUpCaseID
		summary.DatasetLatestFollowUpCaseStatus = queueSummary.LatestFollowUpCaseStatus
		summaries[datasetID] = summary
	}

	return summaries, nil
}

func (a *appHandler) evalDatasetCaseQueueSummaries(ctx context.Context, tenantID string, targetDatasetIDs map[string]struct{}) (map[string]evalDatasetFollowUpCaseSummaryResponse, error) {
	summaries := make(map[string]evalDatasetFollowUpCaseSummaryResponse, len(targetDatasetIDs))
	if a.cases == nil || a.evalReports == nil || len(targetDatasetIDs) == 0 {
		return summaries, nil
	}

	reportIDsByDataset, err := a.collectEvalReportIDsByDataset(ctx, tenantID, targetDatasetIDs)
	if err != nil {
		return nil, err
	}
	allReportIDs := make([]string, 0)
	seenReportIDs := make(map[string]struct{})
	for _, reportIDs := range reportIDsByDataset {
		for _, reportID := range reportIDs {
			if _, ok := seenReportIDs[reportID]; ok {
				continue
			}
			seenReportIDs[reportID] = struct{}{}
			allReportIDs = append(allReportIDs, reportID)
		}
	}
	reportSummaries, err := a.cases.SummarizeBySourceEvalReportIDs(ctx, tenantID, allReportIDs)
	if err != nil {
		return nil, fmt.Errorf("summarize dataset case queues for tenant %q: %w", tenantID, err)
	}

	for datasetID := range targetDatasetIDs {
		reportIDs := reportIDsByDataset[datasetID]
		summary := evalDatasetFollowUpCaseSummaryResponse{}
		for _, reportID := range reportIDs {
			reportSummary := reportSummaries[reportID]
			summary.FollowUpCaseCount += reportSummary.FollowUpCaseCount
			summary.OpenFollowUpCaseCount += reportSummary.OpenFollowUpCaseCount
		}
		if len(reportIDs) > 0 {
			page, err := a.cases.ListCases(ctx, casesvc.ListFilter{
				TenantID:            tenantID,
				SourceEvalReportIDs: reportIDs,
				Limit:               1,
			})
			if err != nil {
				return nil, fmt.Errorf("load latest dataset follow-up case for dataset %q: %w", datasetID, err)
			}
			if len(page.Cases) > 0 {
				summary.LatestFollowUpCaseID = page.Cases[0].ID
				summary.LatestFollowUpCaseStatus = page.Cases[0].Status
			}
		}
		closedFollowUpCaseCount := summary.FollowUpCaseCount - summary.OpenFollowUpCaseCount
		if closedFollowUpCaseCount < 0 {
			closedFollowUpCaseCount = 0
		}
		summary.ClosedFollowUpCaseCount = closedFollowUpCaseCount
		summaries[datasetID] = summary
	}

	return summaries, nil
}

func (a *appHandler) collectEvalReportIDsByDataset(ctx context.Context, tenantID string, targetDatasetIDs map[string]struct{}) (map[string][]string, error) {
	reportIDsByDataset := make(map[string][]string, len(targetDatasetIDs))
	if a.evalReports == nil || len(targetDatasetIDs) == 0 {
		return reportIDsByDataset, nil
	}

	filter := evalsvc.EvalReportListFilter{
		TenantID: tenantID,
		Limit:    100,
	}
	for {
		page, err := a.evalReports.ListEvalReports(ctx, filter)
		if err != nil {
			return nil, fmt.Errorf("list eval reports for tenant %q: %w", tenantID, err)
		}
		for _, item := range page.Reports {
			if _, ok := targetDatasetIDs[item.DatasetID]; !ok {
				continue
			}
			reportIDsByDataset[item.DatasetID] = append(reportIDsByDataset[item.DatasetID], item.ID)
		}
		if !page.HasMore {
			break
		}
		filter.Offset = page.NextOffset
	}

	return reportIDsByDataset, nil
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
