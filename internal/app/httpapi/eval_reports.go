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

type evalReportBadCaseResponse struct {
	EvalCaseID               string                         `json:"eval_case_id"`
	Title                    string                         `json:"title"`
	SourceCaseID             string                         `json:"source_case_id"`
	SourceTaskID             string                         `json:"source_task_id,omitempty"`
	SourceReportID           string                         `json:"source_report_id,omitempty"`
	TraceID                  string                         `json:"trace_id,omitempty"`
	VersionID                string                         `json:"version_id,omitempty"`
	Verdict                  string                         `json:"verdict"`
	Detail                   string                         `json:"detail,omitempty"`
	Score                    float64                        `json:"score"`
	FollowUpCaseCount        int                            `json:"follow_up_case_count"`
	OpenFollowUpCaseCount    int                            `json:"open_follow_up_case_count"`
	LatestFollowUpCaseID     string                         `json:"latest_follow_up_case_id,omitempty"`
	LatestFollowUpCaseStatus string                         `json:"latest_follow_up_case_status,omitempty"`
	PreferredFollowUpAction  evalCaseFollowUpActionResponse `json:"preferred_follow_up_action"`
}

type evalReportResponse struct {
	ReportID                        string                               `json:"report_id"`
	TenantID                        string                               `json:"tenant_id"`
	RunID                           string                               `json:"run_id"`
	DatasetID                       string                               `json:"dataset_id"`
	DatasetName                     string                               `json:"dataset_name"`
	RunStatus                       string                               `json:"run_status"`
	Status                          string                               `json:"status"`
	Summary                         string                               `json:"summary"`
	TotalItems                      int                                  `json:"total_items"`
	RecordedResults                 int                                  `json:"recorded_results"`
	PassedItems                     int                                  `json:"passed_items"`
	FailedItems                     int                                  `json:"failed_items"`
	MissingResults                  int                                  `json:"missing_results"`
	AverageScore                    float64                              `json:"average_score"`
	JudgeVersion                    string                               `json:"judge_version,omitempty"`
	BadCaseCount                    int                                  `json:"bad_case_count"`
	BadCaseWithoutOpenFollowUpCount int                                  `json:"bad_case_without_open_follow_up_count"`
	FollowUpCaseCount               int                                  `json:"follow_up_case_count"`
	OpenFollowUpCaseCount           int                                  `json:"open_follow_up_case_count"`
	LatestFollowUpCaseID            string                               `json:"latest_follow_up_case_id,omitempty"`
	LatestFollowUpCaseStatus        string                               `json:"latest_follow_up_case_status,omitempty"`
	PreferredFollowUpAction         evalReportFollowUpActionResponse     `json:"preferred_follow_up_action"`
	CompareFollowUpCaseCount        int                                  `json:"compare_follow_up_case_count"`
	OpenCompareFollowUpCaseCount    int                                  `json:"open_compare_follow_up_case_count"`
	LatestCompareFollowUpCaseID     string                               `json:"latest_compare_follow_up_case_id,omitempty"`
	LatestCompareFollowUpCaseStatus string                               `json:"latest_compare_follow_up_case_status,omitempty"`
	PreferredCompareFollowUpAction  evalReportCompareQueueActionResponse `json:"preferred_compare_follow_up_action"`
	LinkedCaseSummary               *evalReportLinkedCaseSummaryResponse `json:"linked_case_summary,omitempty"`
	PreferredLinkedCaseAction       evalReportLinkedCaseActionResponse   `json:"preferred_linked_case_action"`
	Metadata                        json.RawMessage                      `json:"metadata,omitempty"`
	BadCases                        []evalReportBadCaseResponse          `json:"bad_cases,omitempty"`
	CreatedAt                       string                               `json:"created_at"`
	UpdatedAt                       string                               `json:"updated_at"`
	ReadyAt                         string                               `json:"ready_at"`
}

type evalReportFollowUpActionResponse struct {
	Mode               string `json:"mode"`
	CaseID             string `json:"case_id,omitempty"`
	SourceEvalReportID string `json:"source_eval_report_id,omitempty"`
}

type evalReportLinkedCaseActionResponse struct {
	Mode               string `json:"mode"`
	CaseID             string `json:"case_id,omitempty"`
	SourceEvalReportID string `json:"source_eval_report_id,omitempty"`
}

type evalReportCompareQueueActionResponse struct {
	Mode               string `json:"mode"`
	SourceEvalReportID string `json:"source_eval_report_id,omitempty"`
}

type evalReportLinkedCaseSummaryResponse struct {
	TotalCaseCount   int    `json:"total_case_count"`
	OpenCaseCount    int    `json:"open_case_count"`
	LatestCaseID     string `json:"latest_case_id,omitempty"`
	LatestCaseStatus string `json:"latest_case_status,omitempty"`
	LatestAssignedTo string `json:"latest_assigned_to,omitempty"`
}

type listEvalReportsResponse struct {
	Reports    []evalReportResponse `json:"reports"`
	HasMore    bool                 `json:"has_more"`
	NextOffset *int                 `json:"next_offset,omitempty"`
}

type evalReportComparisonSummaryResponse struct {
	SameTenant           bool    `json:"same_tenant"`
	SameDataset          bool    `json:"same_dataset"`
	SameRunStatus        bool    `json:"same_run_status"`
	JudgeVersionChanged  bool    `json:"judge_version_changed"`
	MetadataChanged      bool    `json:"metadata_changed"`
	TotalItemsDelta      int     `json:"total_items_delta"`
	RecordedResultsDelta int     `json:"recorded_results_delta"`
	PassedItemsDelta     int     `json:"passed_items_delta"`
	FailedItemsDelta     int     `json:"failed_items_delta"`
	MissingResultsDelta  int     `json:"missing_results_delta"`
	AverageScoreDelta    float64 `json:"average_score_delta"`
	BadCaseCountDelta    int     `json:"bad_case_count_delta"`
	BadCaseOverlapCount  int     `json:"bad_case_overlap_count"`
	ReadyAtDeltaSecond   int64   `json:"ready_at_delta_second"`
}

type evalReportComparisonItemResponse struct {
	ReportID                        string                                  `json:"report_id"`
	TenantID                        string                                  `json:"tenant_id"`
	RunID                           string                                  `json:"run_id"`
	DatasetID                       string                                  `json:"dataset_id"`
	DatasetName                     string                                  `json:"dataset_name"`
	RunStatus                       string                                  `json:"run_status"`
	Status                          string                                  `json:"status"`
	Summary                         string                                  `json:"summary"`
	TotalItems                      int                                     `json:"total_items"`
	RecordedResults                 int                                     `json:"recorded_results"`
	PassedItems                     int                                     `json:"passed_items"`
	FailedItems                     int                                     `json:"failed_items"`
	MissingResults                  int                                     `json:"missing_results"`
	AverageScore                    float64                                 `json:"average_score"`
	JudgeVersion                    string                                  `json:"judge_version,omitempty"`
	VersionID                       string                                  `json:"version_id,omitempty"`
	BadCaseCount                    int                                     `json:"bad_case_count"`
	BadCaseWithoutOpenFollowUpCount int                                     `json:"bad_case_without_open_follow_up_count"`
	FollowUpCaseCount               int                                     `json:"follow_up_case_count"`
	OpenFollowUpCaseCount           int                                     `json:"open_follow_up_case_count"`
	LatestFollowUpCaseID            string                                  `json:"latest_follow_up_case_id,omitempty"`
	LatestFollowUpCaseStatus        string                                  `json:"latest_follow_up_case_status,omitempty"`
	LinkedCaseSummary               *evalReportLinkedCaseSummaryResponse    `json:"linked_case_summary,omitempty"`
	CompareFollowUpCaseCount        int                                     `json:"compare_follow_up_case_count"`
	OpenCompareFollowUpCaseCount    int                                     `json:"open_compare_follow_up_case_count"`
	LatestCompareFollowUpCaseID     string                                  `json:"latest_compare_follow_up_case_id,omitempty"`
	LatestCompareFollowUpCaseStatus string                                  `json:"latest_compare_follow_up_case_status,omitempty"`
	PreferredCompareFollowUpAction  evalReportCompareFollowUpActionResponse `json:"preferred_compare_follow_up_action"`
	CreatedAt                       string                                  `json:"created_at"`
	UpdatedAt                       string                                  `json:"updated_at"`
	ReadyAt                         string                                  `json:"ready_at"`
}

type evalReportCompareFollowUpActionResponse struct {
	Mode               string `json:"mode"`
	SourceEvalReportID string `json:"source_eval_report_id,omitempty"`
}

type evalReportComparisonResponse struct {
	Left    evalReportComparisonItemResponse    `json:"left"`
	Right   evalReportComparisonItemResponse    `json:"right"`
	Summary evalReportComparisonSummaryResponse `json:"summary"`
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
	needsFollowUp, err := parseEvalReportNeedsFollowUpFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}
	badCaseNeedsFollowUp, err := parseEvalReportBadCaseNeedsFollowUpFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	resp, err := a.listEvalReportsResponse(r.Context(), filter, needsFollowUp, badCaseNeedsFollowUp)
	if err != nil {
		code := "eval_report_list_failed"
		if errors.Is(err, errEvalReportFollowUpSummaryFailed) {
			code = "eval_report_follow_up_summary_failed"
		}
		writeError(w, http.StatusInternalServerError, code, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleEvalReportCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}
	leftReportID := strings.TrimSpace(r.URL.Query().Get("left_report_id"))
	if leftReportID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "left_report_id is required")
		return
	}
	rightReportID := strings.TrimSpace(r.URL.Query().Get("right_report_id"))
	if rightReportID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "right_report_id is required")
		return
	}

	comparison, err := a.evalReports.CompareEvalReports(r.Context(), leftReportID, rightReportID)
	if err != nil {
		if errors.Is(err, evalsvc.ErrEvalReportNotFound) {
			writeError(w, http.StatusNotFound, "eval_report_not_found", "eval report not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "eval_report_compare_failed", err.Error())
		return
	}
	if comparison.Left.TenantID != tenantID || comparison.Right.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_report_not_found", "eval report not found")
		return
	}

	followUpSummaries := map[string]casesvc.EvalReportFollowUpSummary{}
	badCaseWithoutOpenFollowUpCounts := map[string]int{}
	compareFollowUpSummaries := map[string]casesvc.EvalReportCompareFollowUpSummary{}
	linkedCaseSummaries := map[string]*evalReportLinkedCaseSummaryResponse{}
	if a.cases != nil {
		followUpSummaries, err = a.cases.SummarizeBySourceEvalReportIDs(r.Context(), tenantID, []string{comparison.Left.ID, comparison.Right.ID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "eval_report_follow_up_summary_failed", err.Error())
			return
		}
		compareFollowUpSummaries, err = a.cases.SummarizeCompareOriginBySourceEvalReportIDs(r.Context(), tenantID, []string{comparison.Left.ID, comparison.Right.ID})
		if err != nil {
			writeError(w, http.StatusInternalServerError, "eval_report_follow_up_summary_failed", err.Error())
			return
		}
		for _, report := range []evalsvc.EvalReport{comparison.Left, comparison.Right} {
			linkedSummary, linkedErr := a.evalReportLinkedCaseSummary(r.Context(), tenantID, report.ID, followUpSummaries[report.ID])
			if linkedErr != nil {
				writeError(w, http.StatusInternalServerError, "eval_report_linked_case_summary_failed", linkedErr.Error())
				return
			}
			linkedCaseSummaries[report.ID] = linkedSummary
		}
	}
	badCaseWithoutOpenFollowUpCounts, err = a.evalReportBadCaseWithoutOpenFollowUpCounts(r.Context(), tenantID, []evalsvc.EvalReport{comparison.Left, comparison.Right})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_report_follow_up_summary_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, evalReportComparisonResponse{
		Left:  newEvalReportComparisonItemResponse(comparison.Left, followUpSummaries[comparison.Left.ID], linkedCaseSummaries[comparison.Left.ID], compareFollowUpSummaries[comparison.Left.ID], badCaseWithoutOpenFollowUpCounts[comparison.Left.ID]),
		Right: newEvalReportComparisonItemResponse(comparison.Right, followUpSummaries[comparison.Right.ID], linkedCaseSummaries[comparison.Right.ID], compareFollowUpSummaries[comparison.Right.ID], badCaseWithoutOpenFollowUpCounts[comparison.Right.ID]),
		Summary: evalReportComparisonSummaryResponse{
			SameTenant:           comparison.Summary.SameTenant,
			SameDataset:          comparison.Summary.SameDataset,
			SameRunStatus:        comparison.Summary.SameRunStatus,
			JudgeVersionChanged:  comparison.Summary.JudgeVersionChanged,
			MetadataChanged:      comparison.Summary.MetadataChanged,
			TotalItemsDelta:      comparison.Summary.TotalItemsDelta,
			RecordedResultsDelta: comparison.Summary.RecordedResultsDelta,
			PassedItemsDelta:     comparison.Summary.PassedItemsDelta,
			FailedItemsDelta:     comparison.Summary.FailedItemsDelta,
			MissingResultsDelta:  comparison.Summary.MissingResultsDelta,
			AverageScoreDelta:    comparison.Summary.AverageScoreDelta,
			BadCaseCountDelta:    comparison.Summary.BadCaseCountDelta,
			BadCaseOverlapCount:  comparison.Summary.BadCaseOverlapCount,
			ReadyAtDeltaSecond:   comparison.Summary.ReadyAtDeltaSecond,
		},
	})
}

func (a *appHandler) handleEvalReportByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	if tenantID == "" {
		writeError(w, http.StatusBadRequest, "invalid_query", "tenant_id is required")
		return
	}

	reportID := strings.TrimPrefix(r.URL.Path, "/api/v1/eval-reports/")
	if reportID == "" || strings.Contains(reportID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}
	badCaseNeedsFollowUp, err := parseEvalReportBadCaseNeedsFollowUpFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
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
	if item.TenantID != tenantID {
		writeError(w, http.StatusNotFound, "eval_report_not_found", "eval report not found")
		return
	}
	originalBadCaseCount := len(item.BadCases)

	followUpSummaries, err := a.cases.SummarizeBySourceEvalReportIDs(r.Context(), tenantID, []string{reportID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_report_follow_up_summary_failed", err.Error())
		return
	}
	compareFollowUpSummaries, err := a.cases.SummarizeCompareOriginBySourceEvalReportIDs(r.Context(), tenantID, []string{reportID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_report_follow_up_summary_failed", err.Error())
		return
	}
	linkedCaseSummary, err := a.evalReportLinkedCaseSummary(r.Context(), tenantID, reportID, followUpSummaries[reportID])
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_report_linked_case_summary_failed", err.Error())
		return
	}

	badCaseFollowUpSummaries, err := a.evalCaseFollowUpSummaries(r.Context(), tenantID, item.BadCases)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "eval_case_follow_up_summary_failed", err.Error())
		return
	}
	badCaseWithoutOpenFollowUpCount := countEvalReportBadCasesWithoutOpenFollowUp(item.BadCases, badCaseFollowUpSummaries)

	if badCaseNeedsFollowUp != nil {
		item.BadCases = filterEvalReportBadCasesByNeedsFollowUp(item.BadCases, badCaseFollowUpSummaries, *badCaseNeedsFollowUp)
	}

	writeJSON(w, http.StatusOK, newEvalReportResponse(item, true, followUpSummaries[reportID], compareFollowUpSummaries[reportID], linkedCaseSummary, badCaseFollowUpSummaries, originalBadCaseCount, badCaseWithoutOpenFollowUpCount))
}

func parseEvalReportListFilter(r *http.Request) (evalsvc.EvalReportListFilter, error) {
	filter := evalsvc.EvalReportListFilter{
		ReportID:  strings.TrimSpace(r.URL.Query().Get("report_id")),
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

var errEvalReportFollowUpSummaryFailed = errors.New("eval report follow-up summary failed")

func parseEvalReportNeedsFollowUpFilter(r *http.Request) (*bool, error) {
	rawNeedsFollowUp := strings.TrimSpace(r.URL.Query().Get("needs_follow_up"))
	if rawNeedsFollowUp == "" {
		return nil, nil
	}

	value, err := strconv.ParseBool(rawNeedsFollowUp)
	if err != nil {
		return nil, errors.New("needs_follow_up must be true or false")
	}
	return &value, nil
}

func parseEvalReportBadCaseNeedsFollowUpFilter(r *http.Request) (*bool, error) {
	rawNeedsFollowUp := strings.TrimSpace(r.URL.Query().Get("bad_case_needs_follow_up"))
	if rawNeedsFollowUp == "" {
		return nil, nil
	}

	value, err := strconv.ParseBool(rawNeedsFollowUp)
	if err != nil {
		return nil, errors.New("bad_case_needs_follow_up must be true or false")
	}
	return &value, nil
}

func (a *appHandler) listEvalReportsResponse(ctx context.Context, filter evalsvc.EvalReportListFilter, needsFollowUp *bool, badCaseNeedsFollowUp *bool) (listEvalReportsResponse, error) {
	if needsFollowUp == nil && badCaseNeedsFollowUp == nil {
		page, err := a.evalReports.ListEvalReports(ctx, filter)
		if err != nil {
			return listEvalReportsResponse{}, err
		}
		return a.buildEvalReportListResponse(ctx, filter.TenantID, page)
	}

	collectorLimit := filter.Limit
	if collectorLimit < 50 {
		collectorLimit = 50
	}
	if collectorLimit < filter.Offset+filter.Limit+1 {
		collectorLimit = filter.Offset + filter.Limit + 1
	}

	baseFilter := filter
	baseFilter.Offset = 0
	baseFilter.Limit = collectorLimit

	resp := listEvalReportsResponse{Reports: make([]evalReportResponse, 0, filter.Limit)}
	matchedCount := 0
	for {
		page, err := a.evalReports.ListEvalReports(ctx, baseFilter)
		if err != nil {
			return listEvalReportsResponse{}, err
		}
		chunk, err := a.buildEvalReportListResponse(ctx, filter.TenantID, page)
		if err != nil {
			return listEvalReportsResponse{}, err
		}
		for _, item := range chunk.Reports {
			hasOpenFollowUp := item.OpenFollowUpCaseCount > 0
			if needsFollowUp != nil && hasOpenFollowUp != *needsFollowUp {
				continue
			}
			hasUncoveredBadCases := item.BadCaseWithoutOpenFollowUpCount > 0
			if badCaseNeedsFollowUp != nil && hasUncoveredBadCases != *badCaseNeedsFollowUp {
				continue
			}
			if matchedCount < filter.Offset {
				matchedCount++
				continue
			}
			if len(resp.Reports) < filter.Limit {
				resp.Reports = append(resp.Reports, item)
				matchedCount++
				continue
			}
			resp.HasMore = true
			nextOffset := filter.Offset + filter.Limit
			resp.NextOffset = &nextOffset
			return resp, nil
		}
		if !page.HasMore {
			return resp, nil
		}
		baseFilter.Offset = page.NextOffset
	}
}

func (a *appHandler) buildEvalReportListResponse(ctx context.Context, tenantID string, page evalsvc.EvalReportListPage) (listEvalReportsResponse, error) {
	resp := listEvalReportsResponse{
		Reports: make([]evalReportResponse, 0, len(page.Reports)),
		HasMore: page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	reportIDs := make([]string, 0, len(page.Reports))
	for _, item := range page.Reports {
		reportIDs = append(reportIDs, item.ID)
	}
	var err error
	followUpSummaries := map[string]casesvc.EvalReportFollowUpSummary{}
	compareFollowUpSummaries := map[string]casesvc.EvalReportCompareFollowUpSummary{}
	linkedCaseSummaries := map[string]*evalReportLinkedCaseSummaryResponse{}
	if a.cases != nil && len(reportIDs) > 0 {
		followUpSummaries, err = a.cases.SummarizeBySourceEvalReportIDs(ctx, tenantID, reportIDs)
		if err != nil {
			return listEvalReportsResponse{}, fmt.Errorf("%w: %v", errEvalReportFollowUpSummaryFailed, err)
		}
		compareFollowUpSummaries, err = a.cases.SummarizeCompareOriginBySourceEvalReportIDs(ctx, tenantID, reportIDs)
		if err != nil {
			return listEvalReportsResponse{}, fmt.Errorf("%w: %v", errEvalReportFollowUpSummaryFailed, err)
		}
		for _, item := range page.Reports {
			linkedSummary, linkedErr := a.evalReportLinkedCaseSummary(ctx, tenantID, item.ID, followUpSummaries[item.ID])
			if linkedErr != nil {
				return listEvalReportsResponse{}, fmt.Errorf("eval report %q: %w: %v", item.ID, errEvalReportFollowUpSummaryFailed, linkedErr)
			}
			linkedCaseSummaries[item.ID] = linkedSummary
		}
	}
	badCaseWithoutOpenFollowUpCounts, err := a.evalReportBadCaseWithoutOpenFollowUpCounts(ctx, tenantID, page.Reports)
	if err != nil {
		return listEvalReportsResponse{}, fmt.Errorf("%w: %v", errEvalReportFollowUpSummaryFailed, err)
	}
	for _, item := range page.Reports {
		summary := followUpSummaries[item.ID]
		compareSummary := compareFollowUpSummaries[item.ID]
		resp.Reports = append(resp.Reports, newEvalReportResponse(item, false, summary, compareSummary, linkedCaseSummaries[item.ID], nil, len(item.BadCases), badCaseWithoutOpenFollowUpCounts[item.ID]))
	}
	return resp, nil
}

func (a *appHandler) evalReportLinkedCaseSummary(ctx context.Context, tenantID, reportID string, followUpSummary casesvc.EvalReportFollowUpSummary) (*evalReportLinkedCaseSummaryResponse, error) {
	summary := &evalReportLinkedCaseSummaryResponse{
		TotalCaseCount:   followUpSummary.FollowUpCaseCount,
		OpenCaseCount:    followUpSummary.OpenFollowUpCaseCount,
		LatestCaseID:     followUpSummary.LatestFollowUpCaseID,
		LatestCaseStatus: followUpSummary.LatestFollowUpCaseStatus,
	}
	if a.cases == nil || tenantID == "" || followUpSummary.LatestFollowUpCaseID == "" {
		return summary, nil
	}

	latestCase, err := a.cases.GetCase(ctx, followUpSummary.LatestFollowUpCaseID)
	if err != nil {
		if errors.Is(err, casesvc.ErrCaseNotFound) {
			return summary, nil
		}
		return nil, fmt.Errorf("get linked case %q for eval report %q: %w", followUpSummary.LatestFollowUpCaseID, reportID, err)
	}
	if latestCase.TenantID != tenantID || latestCase.SourceEvalReportID != reportID {
		return summary, nil
	}
	summary.LatestAssignedTo = latestCase.AssignedTo
	return summary, nil
}

func (a *appHandler) evalCaseFollowUpSummaries(ctx context.Context, tenantID string, badCases []evalsvc.EvalReportBadCase) (map[string]casesvc.EvalCaseFollowUpSummary, error) {
	if a.cases == nil {
		return map[string]casesvc.EvalCaseFollowUpSummary{}, nil
	}
	if len(badCases) == 0 {
		return map[string]casesvc.EvalCaseFollowUpSummary{}, nil
	}

	evalCaseIDs := make([]string, 0, len(badCases))
	seen := make(map[string]struct{}, len(badCases))
	for _, badCase := range badCases {
		if badCase.EvalCaseID == "" {
			continue
		}
		if _, ok := seen[badCase.EvalCaseID]; ok {
			continue
		}
		seen[badCase.EvalCaseID] = struct{}{}
		evalCaseIDs = append(evalCaseIDs, badCase.EvalCaseID)
	}
	if len(evalCaseIDs) == 0 {
		return map[string]casesvc.EvalCaseFollowUpSummary{}, nil
	}

	return a.cases.SummarizeBySourceEvalCaseIDs(ctx, tenantID, evalCaseIDs)
}

func (a *appHandler) evalReportBadCaseWithoutOpenFollowUpCounts(ctx context.Context, tenantID string, reports []evalsvc.EvalReport) (map[string]int, error) {
	counts := make(map[string]int, len(reports))
	if len(reports) == 0 {
		return counts, nil
	}

	allBadCases := make([]evalsvc.EvalReportBadCase, 0)
	for _, report := range reports {
		allBadCases = append(allBadCases, report.BadCases...)
	}
	badCaseSummaries, err := a.evalCaseFollowUpSummaries(ctx, tenantID, allBadCases)
	if err != nil {
		return nil, err
	}
	for _, report := range reports {
		counts[report.ID] = countEvalReportBadCasesWithoutOpenFollowUp(report.BadCases, badCaseSummaries)
	}
	return counts, nil
}

func countEvalReportBadCasesWithoutOpenFollowUp(badCases []evalsvc.EvalReportBadCase, summaries map[string]casesvc.EvalCaseFollowUpSummary) int {
	count := 0
	for _, badCase := range badCases {
		if summaries[badCase.EvalCaseID].OpenFollowUpCaseCount == 0 {
			count++
		}
	}
	return count
}

func filterEvalReportBadCasesByNeedsFollowUp(badCases []evalsvc.EvalReportBadCase, summaries map[string]casesvc.EvalCaseFollowUpSummary, needsFollowUp bool) []evalsvc.EvalReportBadCase {
	if len(badCases) == 0 {
		return nil
	}

	filtered := make([]evalsvc.EvalReportBadCase, 0, len(badCases))
	for _, badCase := range badCases {
		summary := summaries[badCase.EvalCaseID]
		hasOpenFollowUp := summary.OpenFollowUpCaseCount > 0
		if hasOpenFollowUp != needsFollowUp {
			continue
		}
		filtered = append(filtered, badCase)
	}
	return filtered
}

func newEvalReportResponse(item evalsvc.EvalReport, includeHeavy bool, followUpSummary casesvc.EvalReportFollowUpSummary, compareFollowUpSummary casesvc.EvalReportCompareFollowUpSummary, linkedCaseSummary *evalReportLinkedCaseSummaryResponse, badCaseSummaries map[string]casesvc.EvalCaseFollowUpSummary, badCaseCount int, badCaseWithoutOpenFollowUpCount int) evalReportResponse {
	if badCaseCount < 0 {
		badCaseCount = len(item.BadCases)
	}
	resp := evalReportResponse{
		ReportID:                        item.ID,
		TenantID:                        item.TenantID,
		RunID:                           item.RunID,
		DatasetID:                       item.DatasetID,
		DatasetName:                     item.DatasetName,
		RunStatus:                       item.RunStatus,
		Status:                          item.Status,
		Summary:                         item.Summary,
		TotalItems:                      item.TotalItems,
		RecordedResults:                 item.RecordedResults,
		PassedItems:                     item.PassedItems,
		FailedItems:                     item.FailedItems,
		MissingResults:                  item.MissingResults,
		AverageScore:                    item.AverageScore,
		JudgeVersion:                    item.JudgeVersion,
		BadCaseCount:                    badCaseCount,
		BadCaseWithoutOpenFollowUpCount: badCaseWithoutOpenFollowUpCount,
		FollowUpCaseCount:               followUpSummary.FollowUpCaseCount,
		OpenFollowUpCaseCount:           followUpSummary.OpenFollowUpCaseCount,
		LatestFollowUpCaseID:            followUpSummary.LatestFollowUpCaseID,
		LatestFollowUpCaseStatus:        followUpSummary.LatestFollowUpCaseStatus,
		PreferredFollowUpAction:         newEvalReportFollowUpActionResponse(item.ID, followUpSummary),
		CompareFollowUpCaseCount:        compareFollowUpSummary.CompareFollowUpCaseCount,
		OpenCompareFollowUpCaseCount:    compareFollowUpSummary.OpenCompareFollowUpCaseCount,
		LatestCompareFollowUpCaseID:     compareFollowUpSummary.LatestCompareFollowUpCaseID,
		LatestCompareFollowUpCaseStatus: compareFollowUpSummary.LatestCompareFollowUpCaseStatus,
		PreferredCompareFollowUpAction:  newEvalReportCompareQueueActionResponse(item.ID, compareFollowUpSummary),
		LinkedCaseSummary:               linkedCaseSummary,
		PreferredLinkedCaseAction:       newEvalReportLinkedCaseActionResponse(item.ID, linkedCaseSummary),
		CreatedAt:                       item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                       item.UpdatedAt.Format(time.RFC3339Nano),
		ReadyAt:                         item.ReadyAt.Format(time.RFC3339Nano),
	}
	if includeHeavy {
		resp.Metadata = item.MetadataJSON
		if len(item.BadCases) > 0 {
			resp.BadCases = make([]evalReportBadCaseResponse, 0, len(item.BadCases))
			for _, badCase := range item.BadCases {
				badCaseSummary := badCaseSummaries[badCase.EvalCaseID]
				resp.BadCases = append(resp.BadCases, evalReportBadCaseResponse{
					EvalCaseID:               badCase.EvalCaseID,
					Title:                    badCase.Title,
					SourceCaseID:             badCase.SourceCaseID,
					SourceTaskID:             badCase.SourceTaskID,
					SourceReportID:           badCase.SourceReportID,
					TraceID:                  badCase.TraceID,
					VersionID:                badCase.VersionID,
					Verdict:                  badCase.Verdict,
					Detail:                   badCase.Detail,
					Score:                    badCase.Score,
					FollowUpCaseCount:        badCaseSummary.FollowUpCaseCount,
					OpenFollowUpCaseCount:    badCaseSummary.OpenFollowUpCaseCount,
					LatestFollowUpCaseID:     badCaseSummary.LatestFollowUpCaseID,
					LatestFollowUpCaseStatus: badCaseSummary.LatestFollowUpCaseStatus,
					PreferredFollowUpAction:  newEvalReportBadCaseFollowUpActionResponse(badCase.EvalCaseID, badCaseSummary),
				})
			}
		}
	}

	return resp
}

func newEvalReportFollowUpActionResponse(reportID string, followUpSummary casesvc.EvalReportFollowUpSummary) evalReportFollowUpActionResponse {
	action := evalReportFollowUpActionResponse{
		Mode:               "create",
		SourceEvalReportID: reportID,
	}
	if followUpSummary.OpenFollowUpCaseCount <= 0 {
		return action
	}
	if followUpSummary.LatestFollowUpCaseID != "" {
		action.Mode = "open_existing_case"
		action.CaseID = followUpSummary.LatestFollowUpCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalReportBadCaseFollowUpActionResponse(evalCaseID string, followUpSummary casesvc.EvalCaseFollowUpSummary) evalCaseFollowUpActionResponse {
	action := evalCaseFollowUpActionResponse{
		Mode:             "create",
		SourceEvalCaseID: evalCaseID,
	}
	if followUpSummary.OpenFollowUpCaseCount <= 0 {
		return action
	}
	if followUpSummary.LatestFollowUpCaseID != "" {
		action.Mode = "open_existing_case"
		action.CaseID = followUpSummary.LatestFollowUpCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalReportLinkedCaseActionResponse(reportID string, linkedCaseSummary *evalReportLinkedCaseSummaryResponse) evalReportLinkedCaseActionResponse {
	action := evalReportLinkedCaseActionResponse{
		Mode:               "none",
		SourceEvalReportID: reportID,
	}
	if linkedCaseSummary == nil || linkedCaseSummary.TotalCaseCount <= 0 {
		return action
	}
	if linkedCaseSummary.OpenCaseCount > 0 && linkedCaseSummary.LatestCaseID != "" && linkedCaseSummary.LatestCaseStatus == casesvc.StatusOpen {
		action.Mode = "open_existing_case"
		action.CaseID = linkedCaseSummary.LatestCaseID
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalReportCompareQueueActionResponse(reportID string, compareFollowUpSummary casesvc.EvalReportCompareFollowUpSummary) evalReportCompareQueueActionResponse {
	action := evalReportCompareQueueActionResponse{
		Mode:               "none",
		SourceEvalReportID: reportID,
	}
	if compareFollowUpSummary.OpenCompareFollowUpCaseCount <= 0 {
		return action
	}
	action.Mode = "open_existing_queue"
	return action
}

func newEvalReportComparisonItemResponse(item evalsvc.EvalReport, followUpSummary casesvc.EvalReportFollowUpSummary, linkedCaseSummary *evalReportLinkedCaseSummaryResponse, compareFollowUpSummary casesvc.EvalReportCompareFollowUpSummary, badCaseWithoutOpenFollowUpCount int) evalReportComparisonItemResponse {
	return evalReportComparisonItemResponse{
		ReportID:                        item.ID,
		TenantID:                        item.TenantID,
		RunID:                           item.RunID,
		DatasetID:                       item.DatasetID,
		DatasetName:                     item.DatasetName,
		RunStatus:                       item.RunStatus,
		Status:                          item.Status,
		Summary:                         item.Summary,
		TotalItems:                      item.TotalItems,
		RecordedResults:                 item.RecordedResults,
		PassedItems:                     item.PassedItems,
		FailedItems:                     item.FailedItems,
		MissingResults:                  item.MissingResults,
		AverageScore:                    item.AverageScore,
		JudgeVersion:                    item.JudgeVersion,
		VersionID:                       firstEvalReportVersionID(item.MetadataJSON),
		BadCaseCount:                    len(item.BadCases),
		BadCaseWithoutOpenFollowUpCount: badCaseWithoutOpenFollowUpCount,
		FollowUpCaseCount:               followUpSummary.FollowUpCaseCount,
		OpenFollowUpCaseCount:           followUpSummary.OpenFollowUpCaseCount,
		LatestFollowUpCaseID:            followUpSummary.LatestFollowUpCaseID,
		LatestFollowUpCaseStatus:        followUpSummary.LatestFollowUpCaseStatus,
		LinkedCaseSummary:               linkedCaseSummary,
		CompareFollowUpCaseCount:        compareFollowUpSummary.CompareFollowUpCaseCount,
		OpenCompareFollowUpCaseCount:    compareFollowUpSummary.OpenCompareFollowUpCaseCount,
		LatestCompareFollowUpCaseID:     compareFollowUpSummary.LatestCompareFollowUpCaseID,
		LatestCompareFollowUpCaseStatus: compareFollowUpSummary.LatestCompareFollowUpCaseStatus,
		PreferredCompareFollowUpAction:  newEvalReportCompareFollowUpActionResponse(item.ID, compareFollowUpSummary),
		CreatedAt:                       item.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:                       item.UpdatedAt.Format(time.RFC3339Nano),
		ReadyAt:                         item.ReadyAt.Format(time.RFC3339Nano),
	}
}

func newEvalReportCompareFollowUpActionResponse(reportID string, compareFollowUpSummary casesvc.EvalReportCompareFollowUpSummary) evalReportCompareFollowUpActionResponse {
	action := evalReportCompareFollowUpActionResponse{
		Mode:               "create",
		SourceEvalReportID: reportID,
	}
	if compareFollowUpSummary.OpenCompareFollowUpCaseCount > 0 {
		action.Mode = "open_existing_queue"
	}
	return action
}

func firstEvalReportVersionID(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	values, _ := payload["version_ids"].([]any)
	if len(values) == 0 {
		return ""
	}
	versionID, _ := values[0].(string)
	return versionID
}
