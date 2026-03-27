package eval

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"time"
)

type evalReportStore interface {
	SaveEvalReport(ctx context.Context, item EvalReport) (EvalReport, error)
	GetEvalReport(ctx context.Context, reportID string) (EvalReport, error)
	ListEvalReports(ctx context.Context, filter EvalReportListFilter) (EvalReportListPage, error)
}

type evalRunDetailReader interface {
	GetRunDetail(ctx context.Context, runID string) (EvalRunDetail, error)
}

// EvalReportService materializes completed eval runs into durable aggregated reports.
type EvalReportService struct {
	store evalReportStore
	runs  evalRunDetailReader
}

// NewEvalReportService constructs the eval-report service with memory-backed defaults.
func NewEvalReportService() *EvalReportService {
	store := newMemoryStore()
	return NewEvalReportServiceWithDependencies(store, NewRunServiceWithStore(store, nil))
}

// NewEvalReportServiceWithDependencies constructs the eval-report service with caller-provided storage and run reader.
func NewEvalReportServiceWithDependencies(store evalReportStore, runs evalRunDetailReader) *EvalReportService {
	if store == nil {
		store = newMemoryStore()
	}
	if runs == nil {
		runs = NewRunServiceWithStore(nil, nil)
	}

	return &EvalReportService{
		store: store,
		runs:  runs,
	}
}

// GetEvalReport returns one durable eval report.
func (s *EvalReportService) GetEvalReport(ctx context.Context, reportID string) (EvalReport, error) {
	return s.store.GetEvalReport(ctx, reportID)
}

// ListEvalReports returns a durable eval-report page.
func (s *EvalReportService) ListEvalReports(ctx context.Context, filter EvalReportListFilter) (EvalReportListPage, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}

	return s.store.ListEvalReports(ctx, filter)
}

// CompareEvalReports returns two durable eval reports plus an operator-facing summary
// of the differences that matter for triage and regression review.
func (s *EvalReportService) CompareEvalReports(ctx context.Context, leftReportID string, rightReportID string) (EvalReportComparison, error) {
	if leftReportID == "" {
		return EvalReportComparison{}, errors.New("left_report_id is required")
	}
	if rightReportID == "" {
		return EvalReportComparison{}, errors.New("right_report_id is required")
	}

	left, err := s.store.GetEvalReport(ctx, leftReportID)
	if err != nil {
		return EvalReportComparison{}, err
	}
	right, err := s.store.GetEvalReport(ctx, rightReportID)
	if err != nil {
		return EvalReportComparison{}, err
	}

	return EvalReportComparison{
		Left:    left,
		Right:   right,
		Summary: buildEvalReportComparisonSummary(left, right),
	}, nil
}

// MaterializeRunReport builds and saves the canonical eval report for one completed run.
func (s *EvalReportService) MaterializeRunReport(ctx context.Context, runID string) (EvalReport, error) {
	detail, err := s.runs.GetRunDetail(ctx, runID)
	if err != nil {
		return EvalReport{}, err
	}

	report, err := buildEvalReport(detail)
	if err != nil {
		return EvalReport{}, err
	}

	return s.store.SaveEvalReport(ctx, report)
}

func buildEvalReport(detail EvalRunDetail) (EvalReport, error) {
	run := detail.Run
	if run.Status != RunStatusSucceeded && run.Status != RunStatusFailed {
		return EvalReport{}, fmt.Errorf("eval report requires terminal run state")
	}

	summary := run.ResultSummary
	if summary == nil {
		summary = summarizeRunResultsForTotal(run.DatasetItemCount, detail.ItemResults)
	}
	if summary == nil {
		return EvalReport{}, fmt.Errorf("eval report requires item results")
	}

	itemByCaseID := make(map[string]EvalRunItem, len(detail.Items))
	for _, item := range detail.Items {
		itemByCaseID[item.EvalCaseID] = item
	}

	badCases := make([]EvalReportBadCase, 0)
	judgeVersions := make([]string, 0)
	promptPaths := make([]string, 0)
	versionIDs := make([]string, 0)
	totalScore := 0.0
	for _, result := range detail.ItemResults {
		totalScore += result.Score
		if result.JudgeVersion != "" {
			judgeVersions = appendIfMissing(judgeVersions, result.JudgeVersion)
		}
		if promptPath := judgePromptPathFromOutput(result.JudgeOutput); promptPath != "" {
			promptPaths = appendIfMissing(promptPaths, promptPath)
		}
		if item, ok := itemByCaseID[result.EvalCaseID]; ok {
			if item.VersionID != "" {
				versionIDs = appendIfMissing(versionIDs, item.VersionID)
			}
			if result.Verdict == RunItemVerdictFail || result.Status == RunItemResultFailed {
				badCases = append(badCases, EvalReportBadCase{
					EvalCaseID:     result.EvalCaseID,
					Title:          item.Title,
					SourceCaseID:   item.SourceCaseID,
					SourceTaskID:   item.SourceTaskID,
					SourceReportID: item.SourceReportID,
					TraceID:        item.TraceID,
					VersionID:      item.VersionID,
					Verdict:        result.Verdict,
					Detail:         result.Detail,
					Score:          result.Score,
				})
			}
		}
	}

	averageScore := 0.0
	if summary.RecordedResults > 0 {
		averageScore = totalScore / float64(summary.RecordedResults)
	}

	readyAt := run.FinishedAt
	if readyAt.IsZero() {
		readyAt = run.UpdatedAt
	}
	metadata, err := json.Marshal(map[string]any{
		"run_status":         run.Status,
		"judge_versions":     judgeVersions,
		"judge_prompt_paths": promptPaths,
		"version_ids":        versionIDs,
		"dataset_item_count": run.DatasetItemCount,
		"recorded_results":   summary.RecordedResults,
		"bad_case_count":     len(badCases),
		"ready_at":           readyAt.Format(time.RFC3339Nano),
		"result_summary":     summary,
	})
	if err != nil {
		return EvalReport{}, fmt.Errorf("marshal eval report metadata: %w", err)
	}

	return EvalReport{
		ID:              EvalReportIDFromRunID(run.ID),
		TenantID:        run.TenantID,
		RunID:           run.ID,
		DatasetID:       run.DatasetID,
		DatasetName:     run.DatasetName,
		RunStatus:       run.Status,
		Status:          EvalReportStatusReady,
		Summary:         buildEvalReportSummary(*summary, averageScore),
		TotalItems:      summary.TotalItems,
		RecordedResults: summary.RecordedResults,
		PassedItems:     summary.SucceededItems,
		FailedItems:     summary.FailedItems,
		MissingResults:  summary.MissingResults,
		AverageScore:    averageScore,
		JudgeVersion:    collapseSingle(judgeVersions),
		MetadataJSON:    metadata,
		BadCases:        badCases,
		CreatedAt:       run.CreatedAt,
		UpdatedAt:       readyAt,
		ReadyAt:         readyAt,
	}, nil
}

func buildEvalReportSummary(summary EvalRunResultSummary, averageScore float64) string {
	return fmt.Sprintf("%d failed / %d passed / %d total (avg score %.3f)", summary.FailedItems, summary.SucceededItems, summary.TotalItems, averageScore)
}

func collapseSingle(values []string) string {
	if len(values) == 1 {
		return values[0]
	}
	return ""
}

func appendIfMissing(values []string, candidate string) []string {
	if candidate == "" || slices.Contains(values, candidate) {
		return values
	}
	return append(values, candidate)
}

func judgePromptPathFromOutput(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	value, _ := payload["judge_prompt_path"].(string)
	return value
}

func buildEvalReportComparisonSummary(left EvalReport, right EvalReport) EvalReportComparisonSummary {
	return EvalReportComparisonSummary{
		SameTenant:           left.TenantID == right.TenantID,
		SameDataset:          left.DatasetID == right.DatasetID,
		SameRunStatus:        left.RunStatus == right.RunStatus,
		JudgeVersionChanged:  left.JudgeVersion != right.JudgeVersion,
		MetadataChanged:      !equalEvalJSON(left.MetadataJSON, right.MetadataJSON),
		TotalItemsDelta:      right.TotalItems - left.TotalItems,
		RecordedResultsDelta: right.RecordedResults - left.RecordedResults,
		PassedItemsDelta:     right.PassedItems - left.PassedItems,
		FailedItemsDelta:     right.FailedItems - left.FailedItems,
		MissingResultsDelta:  right.MissingResults - left.MissingResults,
		AverageScoreDelta:    right.AverageScore - left.AverageScore,
		BadCaseCountDelta:    len(right.BadCases) - len(left.BadCases),
		BadCaseOverlapCount:  overlapEvalBadCases(left.BadCases, right.BadCases),
		ReadyAtDeltaSecond:   int64(right.ReadyAt.Sub(left.ReadyAt).Seconds()),
	}
}

func overlapEvalBadCases(left []EvalReportBadCase, right []EvalReportBadCase) int {
	leftIDs := make(map[string]struct{}, len(left))
	for _, item := range left {
		leftIDs[item.EvalCaseID] = struct{}{}
	}

	count := 0
	for _, item := range right {
		if _, ok := leftIDs[item.EvalCaseID]; ok {
			count++
		}
	}

	return count
}

func equalEvalJSON(left json.RawMessage, right json.RawMessage) bool {
	if len(left) == 0 && len(right) == 0 {
		return true
	}

	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return string(left) == string(right)
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return string(left) == string(right)
	}

	return reflect.DeepEqual(leftValue, rightValue)
}
