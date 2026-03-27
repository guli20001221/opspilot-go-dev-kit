package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPJSONJudgeBuildsVersionedResults(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("Method = %q, want %q", r.Method, http.MethodPost)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want bearer token", auth)
		}

		var request map[string]any
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("Decode(request) error = %v", err)
		}
		if request["prompt_path"] != PlaceholderJudgePromptPath {
			t.Fatalf("prompt_path = %#v, want %q", request["prompt_path"], PlaceholderJudgePromptPath)
		}
		if request["model"] != "judge-demo" {
			t.Fatalf("model = %#v, want %q", request["model"], "judge-demo")
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"verdict":   RunItemVerdictPass,
			"score":     0.8,
			"rationale": "provider accepted the eval item",
		})
	}))
	defer server.Close()

	judge := NewHTTPJSONJudge(HTTPJSONJudgeOptions{
		BaseURL:    server.URL,
		APIKey:     "test-token",
		Model:      "judge-demo",
		PromptPath: PlaceholderJudgePromptPath,
	})

	results, err := judge.BuildItemResults(context.Background(), []EvalRunItem{{
		EvalCaseID:   "eval-case-provider",
		Title:        "Provider-backed item",
		SourceCaseID: "case-provider",
	}}, RunItemResultSucceeded, "provider run passed", time.Unix(1700041000, 0).UTC())
	if err != nil {
		t.Fatalf("BuildItemResults() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].JudgeVersion != "http_json/judge-demo/placeholder-eval-judge-v1" {
		t.Fatalf("JudgeVersion = %q, want %q", results[0].JudgeVersion, "http_json/judge-demo/placeholder-eval-judge-v1")
	}
	if results[0].Verdict != RunItemVerdictPass {
		t.Fatalf("Verdict = %q, want %q", results[0].Verdict, RunItemVerdictPass)
	}
	if results[0].Score != 0.8 {
		t.Fatalf("Score = %v, want 0.8", results[0].Score)
	}

	var payload map[string]any
	if err := json.Unmarshal(results[0].JudgeOutput, &payload); err != nil {
		t.Fatalf("Unmarshal(JudgeOutput) error = %v", err)
	}
	if payload["judge_kind"] != "http_json" {
		t.Fatalf("judge_kind = %#v, want %q", payload["judge_kind"], "http_json")
	}
	if payload["judge_prompt_path"] != PlaceholderJudgePromptPath {
		t.Fatalf("judge_prompt_path = %#v, want %q", payload["judge_prompt_path"], PlaceholderJudgePromptPath)
	}
	if payload["rationale"] != "provider accepted the eval item" {
		t.Fatalf("rationale = %#v, want provider rationale", payload["rationale"])
	}
}
