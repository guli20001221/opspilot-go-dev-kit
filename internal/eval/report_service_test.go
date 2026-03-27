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
