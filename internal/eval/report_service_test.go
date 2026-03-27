package eval

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestEvalReportServiceMaterializesAggregatesFromRunDetail(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	runService := NewRunServiceWithStore(store, nil)
	reportService := NewEvalReportServiceWithDependencies(store, runService)

	finishedAt := time.Unix(1700042000, 0).UTC()
	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-aggregate",
		TenantID:         "tenant-eval",
		DatasetID:        "eval-dataset-aggregate",
		DatasetName:      "Aggregate Dataset",
		DatasetItemCount: 2,
		Status:           RunStatusSucceeded,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700041900, 0).UTC(),
		UpdatedAt:        finishedAt,
		StartedAt:        time.Unix(1700041950, 0).UTC(),
		FinishedAt:       finishedAt,
	}, EvalRunItem{
		EvalCaseID:     "eval-case-pass",
		Title:          "Pass item",
		SourceCaseID:   "case-pass",
		SourceTaskID:   "task-pass",
		SourceReportID: "report-pass",
		TraceID:        "trace-pass",
		VersionID:      "version-pass",
	}, EvalRunItem{
		EvalCaseID:     "eval-case-fail",
		Title:          "Fail item",
		SourceCaseID:   "case-fail",
		SourceTaskID:   "task-fail",
		SourceReportID: "report-fail",
		TraceID:        "trace-fail",
		VersionID:      "version-fail",
	})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	store.runItemResults[run.ID] = []EvalRunItemResult{
		{
			EvalCaseID:   "eval-case-pass",
			Status:       RunItemResultSucceeded,
			Verdict:      RunItemVerdictPass,
			Detail:       "pass rationale",
			Score:        1,
			JudgeVersion: "http_json/judge-demo/placeholder-eval-judge-v1",
			JudgeOutput:  json.RawMessage(`{"judge_prompt_path":"eval/prompts/placeholder-eval-judge-v1.md"}`),
			UpdatedAt:    finishedAt,
		},
		{
			EvalCaseID:   "eval-case-fail",
			Status:       RunItemResultFailed,
			Verdict:      RunItemVerdictFail,
			Detail:       "fail rationale",
			Score:        0.25,
			JudgeVersion: "http_json/judge-demo/placeholder-eval-judge-v1",
			JudgeOutput:  json.RawMessage(`{"judge_prompt_path":"eval/prompts/placeholder-eval-judge-v1.md"}`),
			UpdatedAt:    finishedAt,
		},
	}

	report, err := reportService.MaterializeRunReport(ctx, run.ID)
	if err != nil {
		t.Fatalf("MaterializeRunReport() error = %v", err)
	}
	if report.ID != "eval-report-eval-run-aggregate" {
		t.Fatalf("ID = %q, want %q", report.ID, "eval-report-eval-run-aggregate")
	}
	if report.TotalItems != 2 || report.RecordedResults != 2 {
		t.Fatalf("report totals = %#v, want two recorded results", report)
	}
	if report.PassedItems != 1 || report.FailedItems != 1 {
		t.Fatalf("report pass/fail = %#v, want one pass and one fail", report)
	}
	if report.AverageScore != 0.625 {
		t.Fatalf("AverageScore = %v, want 0.625", report.AverageScore)
	}
	if report.JudgeVersion != "http_json/judge-demo/placeholder-eval-judge-v1" {
		t.Fatalf("JudgeVersion = %q, want provider-backed judge version", report.JudgeVersion)
	}
	if len(report.BadCases) != 1 {
		t.Fatalf("len(BadCases) = %d, want 1", len(report.BadCases))
	}
	if report.BadCases[0].EvalCaseID != "eval-case-fail" || report.BadCases[0].SourceCaseID != "case-fail" {
		t.Fatalf("bad case = %#v, want failed eval-case lineage", report.BadCases[0])
	}
}

func TestEvalReportServiceCompareEvalReportsBuildsOperatorFacingSummary(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	reportService := NewEvalReportServiceWithDependencies(store, nil)

	left := EvalReport{
		ID:              "eval-report-left",
		TenantID:        "tenant-eval-compare",
		RunID:           "eval-run-left",
		DatasetID:       "eval-dataset-shared",
		DatasetName:     "Shared Dataset",
		RunStatus:       RunStatusSucceeded,
		Status:          EvalReportStatusReady,
		Summary:         "1 failed / 3 passed / 4 total",
		TotalItems:      4,
		RecordedResults: 4,
		PassedItems:     3,
		FailedItems:     1,
		MissingResults:  0,
		AverageScore:    0.88,
		JudgeVersion:    "judge-v1",
		MetadataJSON:    json.RawMessage(`{"judge_prompt_path":"eval/prompts/judge-v1.md","version_ids":["version-a"]}`),
		BadCases: []EvalReportBadCase{
			{EvalCaseID: "eval-case-shared", Title: "Shared bad case", Verdict: RunItemVerdictFail, Score: 0.1},
			{EvalCaseID: "eval-case-left-only", Title: "Left-only bad case", Verdict: RunItemVerdictFail, Score: 0.2},
		},
		CreatedAt: time.Unix(1700042900, 0).UTC(),
		UpdatedAt: time.Unix(1700043000, 0).UTC(),
		ReadyAt:   time.Unix(1700043000, 0).UTC(),
	}
	right := EvalReport{
		ID:              "eval-report-right",
		TenantID:        "tenant-eval-compare",
		RunID:           "eval-run-right",
		DatasetID:       "eval-dataset-other",
		DatasetName:     "Other Dataset",
		RunStatus:       RunStatusFailed,
		Status:          EvalReportStatusReady,
		Summary:         "2 failed / 2 passed / 4 total",
		TotalItems:      4,
		RecordedResults: 4,
		PassedItems:     2,
		FailedItems:     2,
		MissingResults:  0,
		AverageScore:    0.41,
		JudgeVersion:    "judge-v2",
		MetadataJSON:    json.RawMessage(`{"judge_prompt_path":"eval/prompts/judge-v2.md","version_ids":["version-b"]}`),
		BadCases: []EvalReportBadCase{
			{EvalCaseID: "eval-case-shared", Title: "Shared bad case", Verdict: RunItemVerdictFail, Score: 0.1},
			{EvalCaseID: "eval-case-right-only", Title: "Right-only bad case", Verdict: RunItemVerdictFail, Score: 0.05},
		},
		CreatedAt: time.Unix(1700042910, 0).UTC(),
		UpdatedAt: time.Unix(1700043015, 0).UTC(),
		ReadyAt:   time.Unix(1700043015, 0).UTC(),
	}
	for _, item := range []EvalReport{left, right} {
		if _, err := store.SaveEvalReport(ctx, item); err != nil {
			t.Fatalf("SaveEvalReport(%s) error = %v", item.ID, err)
		}
	}

	got, err := reportService.CompareEvalReports(ctx, left.ID, right.ID)
	if err != nil {
		t.Fatalf("CompareEvalReports() error = %v", err)
	}
	if got.Left.ID != left.ID || got.Right.ID != right.ID {
		t.Fatalf("comparison IDs = %#v, want %q and %q", got, left.ID, right.ID)
	}
	if !got.Summary.SameTenant {
		t.Fatalf("SameTenant = false, want true")
	}
	if got.Summary.SameDataset {
		t.Fatalf("SameDataset = true, want false")
	}
	if got.Summary.SameRunStatus {
		t.Fatalf("SameRunStatus = true, want false")
	}
	if !got.Summary.JudgeVersionChanged || !got.Summary.MetadataChanged {
		t.Fatalf("judge/metadata flags = %#v, want changed", got.Summary)
	}
	if diff := got.Summary.AverageScoreDelta - (-0.47); diff < -0.000001 || diff > 0.000001 {
		t.Fatalf("AverageScoreDelta = %v, want approximately -0.47", got.Summary.AverageScoreDelta)
	}
	if got.Summary.FailedItemsDelta != 1 {
		t.Fatalf("FailedItemsDelta = %d, want 1", got.Summary.FailedItemsDelta)
	}
	if got.Summary.BadCaseCountDelta != 0 {
		t.Fatalf("BadCaseCountDelta = %d, want 0", got.Summary.BadCaseCountDelta)
	}
	if got.Summary.BadCaseOverlapCount != 1 {
		t.Fatalf("BadCaseOverlapCount = %d, want 1", got.Summary.BadCaseOverlapCount)
	}
	if got.Summary.ReadyAtDeltaSecond != 15 {
		t.Fatalf("ReadyAtDeltaSecond = %d, want 15", got.Summary.ReadyAtDeltaSecond)
	}
}
