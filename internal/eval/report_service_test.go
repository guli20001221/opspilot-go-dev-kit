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

// --- Regression detection tests ---

func TestDetectRegressionIdentifiesRegression(t *testing.T) {
	svc := NewEvalReportService()
	ctx := context.Background()

	baseline := seedEvalReport(t, svc, "baseline-1", "tenant-reg", 0.85, []EvalReportBadCase{
		{EvalCaseID: "case-a", Title: "Existing failure", Verdict: "fail", Score: 0.2},
	})
	candidate := seedEvalReport(t, svc, "candidate-1", "tenant-reg", 0.70, []EvalReportBadCase{
		{EvalCaseID: "case-a", Title: "Existing failure", Verdict: "fail", Score: 0.2},
		{EvalCaseID: "case-b", Title: "New failure", Verdict: "fail", Score: 0.1},
	})

	got, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, DefaultRegressionThresholds())
	if err != nil {
		t.Fatalf("DetectRegression() error = %v", err)
	}
	if got.Verdict != RegressionVerdictRegression {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, RegressionVerdictRegression)
	}
	if len(got.NewBadCases) != 1 || got.NewBadCases[0].EvalCaseID != "case-b" {
		t.Fatalf("NewBadCases = %v, want [case-b]", got.NewBadCases)
	}
	if len(got.ResolvedBadCases) != 0 {
		t.Fatalf("ResolvedBadCases = %v, want empty", got.ResolvedBadCases)
	}
	if got.AverageScoreDelta >= 0 {
		t.Fatalf("AverageScoreDelta = %f, want negative", got.AverageScoreDelta)
	}
}

func TestDetectRegressionIdentifiesImprovement(t *testing.T) {
	svc := NewEvalReportService()
	ctx := context.Background()

	baseline := seedEvalReport(t, svc, "baseline-2", "tenant-reg", 0.70, []EvalReportBadCase{
		{EvalCaseID: "case-a", Title: "Was failing", Verdict: "fail", Score: 0.2},
	})
	candidate := seedEvalReport(t, svc, "candidate-2", "tenant-reg", 0.90, []EvalReportBadCase{})

	got, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, DefaultRegressionThresholds())
	if err != nil {
		t.Fatalf("DetectRegression() error = %v", err)
	}
	if got.Verdict != RegressionVerdictImprovement {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, RegressionVerdictImprovement)
	}
	if len(got.ResolvedBadCases) != 1 || got.ResolvedBadCases[0].EvalCaseID != "case-a" {
		t.Fatalf("ResolvedBadCases = %v, want [case-a]", got.ResolvedBadCases)
	}
	if len(got.NewBadCases) != 0 {
		t.Fatalf("NewBadCases = %v, want empty", got.NewBadCases)
	}
}

func TestDetectRegressionIdentifiesStable(t *testing.T) {
	svc := NewEvalReportService()
	ctx := context.Background()

	baseline := seedEvalReport(t, svc, "baseline-3", "tenant-reg", 0.80, []EvalReportBadCase{
		{EvalCaseID: "case-a", Title: "Persistent failure", Verdict: "fail", Score: 0.3},
	})
	candidate := seedEvalReport(t, svc, "candidate-3", "tenant-reg", 0.81, []EvalReportBadCase{
		{EvalCaseID: "case-a", Title: "Persistent failure", Verdict: "fail", Score: 0.3},
	})

	got, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, DefaultRegressionThresholds())
	if err != nil {
		t.Fatalf("DetectRegression() error = %v", err)
	}
	if got.Verdict != RegressionVerdictStable {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, RegressionVerdictStable)
	}
}

func TestDetectRegressionCustomThresholds(t *testing.T) {
	svc := NewEvalReportService()
	ctx := context.Background()

	baseline := seedEvalReport(t, svc, "baseline-4", "tenant-reg", 0.80, []EvalReportBadCase{})
	candidate := seedEvalReport(t, svc, "candidate-4", "tenant-reg", 0.78, []EvalReportBadCase{
		{EvalCaseID: "case-new", Title: "Minor failure", Verdict: "fail", Score: 0.4},
	})

	// Lenient thresholds: allow 1 new bad case and 10% score drop
	lenient := RegressionThresholds{ScoreDropThreshold: 0.10, NewFailedCasesMax: 1}
	got, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, lenient)
	if err != nil {
		t.Fatalf("DetectRegression() error = %v", err)
	}
	if got.Verdict != RegressionVerdictStable {
		t.Fatalf("Verdict = %q, want %q (lenient thresholds)", got.Verdict, RegressionVerdictStable)
	}

	// Strict thresholds: 0 new bad cases allowed
	strict := RegressionThresholds{ScoreDropThreshold: 0.10, NewFailedCasesMax: 0}
	got2, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, strict)
	if err != nil {
		t.Fatalf("DetectRegression() strict error = %v", err)
	}
	if got2.Verdict != RegressionVerdictRegression {
		t.Fatalf("Verdict = %q, want %q (strict thresholds)", got2.Verdict, RegressionVerdictRegression)
	}
}

func TestDetectRegressionExactBoundaryIsRegression(t *testing.T) {
	svc := NewEvalReportService()
	ctx := context.Background()

	// Score drops by exactly the threshold (0.05) — should be regression, not stable
	baseline := seedEvalReport(t, svc, "baseline-boundary", "tenant-reg", 0.80, []EvalReportBadCase{})
	candidate := seedEvalReport(t, svc, "candidate-boundary", "tenant-reg", 0.75, []EvalReportBadCase{
		{EvalCaseID: "case-new", Title: "Boundary failure", Verdict: "fail", Score: 0.3},
	})

	got, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, DefaultRegressionThresholds())
	if err != nil {
		t.Fatalf("DetectRegression() error = %v", err)
	}
	if got.Verdict != RegressionVerdictRegression {
		t.Fatalf("Verdict = %q, want %q (exact boundary should trigger regression)", got.Verdict, RegressionVerdictRegression)
	}
}

func TestDetectRegressionRejectsEmptyIDs(t *testing.T) {
	svc := NewEvalReportService()
	ctx := context.Background()

	_, err := svc.DetectRegression(ctx, "", "some-id", DefaultRegressionThresholds())
	if err == nil {
		t.Fatal("want error for empty baseline_report_id")
	}
	_, err = svc.DetectRegression(ctx, "some-id", "", DefaultRegressionThresholds())
	if err == nil {
		t.Fatal("want error for empty candidate_report_id")
	}
}

func TestPromoteRegressionCasesCreatesFollowUps(t *testing.T) {
	creator := &mockCaseCreator{}
	svc := NewEvalReportServiceWithCases(nil, nil, creator)
	ctx := context.Background()

	baseline := seedEvalReport(t, svc, "baseline-promo", "tenant-promo", 0.85, []EvalReportBadCase{
		{EvalCaseID: "case-old", Title: "Existing failure"},
	})
	candidate := seedEvalReport(t, svc, "candidate-promo", "tenant-promo", 0.70, []EvalReportBadCase{
		{EvalCaseID: "case-old", Title: "Existing failure"},
		{EvalCaseID: "case-new-1", Title: "New regression A", Verdict: "fail", Score: 0.1},
		{EvalCaseID: "case-new-2", Title: "New regression B", Verdict: "fail", Score: 0.2},
	})

	result, err := svc.DetectRegression(ctx, baseline.ID, candidate.ID, DefaultRegressionThresholds())
	if err != nil {
		t.Fatalf("DetectRegression() error = %v", err)
	}
	if result.Verdict != RegressionVerdictRegression {
		t.Fatalf("Verdict = %q, want regression", result.Verdict)
	}

	promoted, err := svc.PromoteRegressionCases(ctx, result, "tenant-promo", "auto-test")
	if err != nil {
		t.Fatalf("PromoteRegressionCases() error = %v", err)
	}
	if promoted != 2 {
		t.Fatalf("promoted = %d, want 2", promoted)
	}
	if len(creator.created) != 2 {
		t.Fatalf("creator.created = %d, want 2", len(creator.created))
	}
	for _, c := range creator.created {
		if c.TenantID != "tenant-promo" {
			t.Fatalf("created case TenantID = %q, want %q", c.TenantID, "tenant-promo")
		}
		if c.SourceEvalReportID != candidate.ID {
			t.Fatalf("created case SourceEvalReportID = %q, want %q", c.SourceEvalReportID, candidate.ID)
		}
	}
}

func TestPromoteRegressionCasesSkipsNonRegression(t *testing.T) {
	creator := &mockCaseCreator{}
	svc := NewEvalReportServiceWithCases(nil, nil, creator)

	result := RegressionResult{Verdict: RegressionVerdictStable}
	promoted, err := svc.PromoteRegressionCases(context.Background(), result, "t", "op")
	if err != nil {
		t.Fatalf("error = %v", err)
	}
	if promoted != 0 {
		t.Fatalf("promoted = %d, want 0 for stable verdict", promoted)
	}
	if len(creator.created) != 0 {
		t.Fatalf("cases created = %d, want 0", len(creator.created))
	}
}

func TestPromoteRegressionCasesRequiresCaseCreator(t *testing.T) {
	svc := NewEvalReportService() // no case creator

	result := RegressionResult{Verdict: RegressionVerdictRegression, NewBadCases: []EvalReportBadCase{{EvalCaseID: "c1"}}}
	_, err := svc.PromoteRegressionCases(context.Background(), result, "t", "op")
	if err == nil {
		t.Fatal("want error when CaseCreator not configured")
	}
}

type mockCaseCreator struct {
	created []CaseCreateInput
}

func (m *mockCaseCreator) CreateCase(_ context.Context, input CaseCreateInput) error {
	m.created = append(m.created, input)
	return nil
}

// seedEvalReport creates and saves a minimal eval report for regression detection tests.
func seedEvalReport(t *testing.T, svc *EvalReportService, id, tenantID string, avgScore float64, badCases []EvalReportBadCase) EvalReport {
	t.Helper()
	passed := 10 - len(badCases)
	report := EvalReport{
		ID:              id,
		TenantID:        tenantID,
		RunID:           "run-" + id,
		DatasetID:       "dataset-" + id,
		Status:          EvalReportStatusReady,
		TotalItems:      10,
		RecordedResults: 10,
		PassedItems:     passed,
		FailedItems:     len(badCases),
		AverageScore:    avgScore,
		BadCases:        badCases,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ReadyAt:         time.Now(),
	}
	saved, err := svc.store.SaveEvalReport(context.Background(), report)
	if err != nil {
		t.Fatalf("seedEvalReport(%s) error = %v", id, err)
	}
	return saved
}
