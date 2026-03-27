package eval

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type failingJudge struct {
	err error
}

func (j failingJudge) BuildItemResults(_ context.Context, items []EvalRunItem, status string, detail string, updatedAt time.Time) ([]EvalRunItemResult, error) {
	return nil, j.err
}

func (j failingJudge) Version() string {
	return "failing"
}

func (j failingJudge) PromptPath() string {
	return "eval/prompts/failing.md"
}

func TestRunnerFallsBackToPlaceholderFailureWhenJudgeFailsAfterSuccessfulExecution(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	service := NewRunServiceWithDependencies(store, nil, failingJudge{err: errors.New("judge upstream unavailable")})

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-provider-fallback",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-provider-fallback",
		DatasetName:      "Published baseline",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030100, 0).UTC(),
		UpdatedAt:        time.Unix(1700030100, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-a", Title: "Eval A", SourceCaseID: "case-a", TraceID: "trace-a"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	runner := NewRunner(service, NewPlaceholderRunExecutor())
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	got, err := service.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.Status != RunStatusFailed {
		t.Fatalf("Status = %q, want %q", got.Status, RunStatusFailed)
	}
	if !strings.Contains(got.ErrorReason, "eval judge failed after successful execution") {
		t.Fatalf("ErrorReason = %q, want judge failure summary", got.ErrorReason)
	}

	detail, err := service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.ItemResults) != 1 {
		t.Fatalf("len(detail.ItemResults) = %d, want 1", len(detail.ItemResults))
	}
	if detail.ItemResults[0].JudgeVersion != PlaceholderJudgeVersion {
		t.Fatalf("JudgeVersion = %q, want placeholder fallback", detail.ItemResults[0].JudgeVersion)
	}
	if detail.ItemResults[0].Status != RunItemResultFailed {
		t.Fatalf("Status = %q, want %q", detail.ItemResults[0].Status, RunItemResultFailed)
	}
}

func TestRunnerPersistsHTTPJudgeResultsOnSuccessfulExecution(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want %q", r.Method, http.MethodPost)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"verdict":   RunItemVerdictPass,
			"score":     0.9,
			"rationale": "provider-backed eval passed",
		})
	}))
	defer server.Close()

	store := newMemoryStore()
	service := NewRunServiceWithDependencies(store, nil, NewHTTPJSONJudge(HTTPJSONJudgeOptions{
		BaseURL:    server.URL,
		Model:      "judge-demo",
		PromptPath: PlaceholderJudgePromptPath,
	}))

	run, err := store.CreateRun(ctx, EvalRun{
		ID:               "eval-run-provider-success",
		TenantID:         "tenant-run",
		DatasetID:        "eval-dataset-provider-success",
		DatasetName:      "Published baseline",
		DatasetItemCount: 1,
		Status:           RunStatusQueued,
		CreatedBy:        "operator",
		CreatedAt:        time.Unix(1700030200, 0).UTC(),
		UpdatedAt:        time.Unix(1700030200, 0).UTC(),
	}, EvalRunItem{EvalCaseID: "eval-case-http", Title: "Eval HTTP", SourceCaseID: "case-http", TraceID: "trace-http"})
	if err != nil {
		t.Fatalf("CreateRun() error = %v", err)
	}

	runner := NewRunner(service, NewPlaceholderRunExecutor())
	processed, err := runner.ProcessNextBatch(ctx, 10)
	if err != nil {
		t.Fatalf("ProcessNextBatch() error = %v", err)
	}
	if processed != 1 {
		t.Fatalf("processed = %d, want 1", processed)
	}

	got, err := service.GetRun(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRun() error = %v", err)
	}
	if got.Status != RunStatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, RunStatusSucceeded)
	}

	detail, err := service.GetRunDetail(ctx, run.ID)
	if err != nil {
		t.Fatalf("GetRunDetail() error = %v", err)
	}
	if len(detail.ItemResults) != 1 {
		t.Fatalf("len(detail.ItemResults) = %d, want 1", len(detail.ItemResults))
	}
	if detail.ItemResults[0].JudgeVersion != "http_json/judge-demo/placeholder-eval-judge-v1" {
		t.Fatalf("JudgeVersion = %q, want provider-backed version", detail.ItemResults[0].JudgeVersion)
	}
	if detail.ItemResults[0].Verdict != RunItemVerdictPass {
		t.Fatalf("Verdict = %q, want %q", detail.ItemResults[0].Verdict, RunItemVerdictPass)
	}
	if detail.ItemResults[0].Score != 0.9 {
		t.Fatalf("Score = %v, want 0.9", detail.ItemResults[0].Score)
	}
}
