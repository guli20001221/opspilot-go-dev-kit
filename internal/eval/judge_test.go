package eval

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlaceholderJudgeBuildsVersionedResults(t *testing.T) {
	judge := NewPlaceholderJudge()
	item := EvalRunItem{EvalCaseID: "eval-case-a"}
	results := judge.BuildItemResults([]EvalRunItem{item}, RunItemResultSucceeded, "placeholder eval passed", time.Unix(1700040000, 0).UTC())
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].JudgeVersion != PlaceholderJudgeVersion {
		t.Fatalf("JudgeVersion = %q, want %q", results[0].JudgeVersion, PlaceholderJudgeVersion)
	}
	if results[0].Verdict != RunItemVerdictPass {
		t.Fatalf("Verdict = %q, want %q", results[0].Verdict, RunItemVerdictPass)
	}

	var payload map[string]any
	if err := json.Unmarshal(results[0].JudgeOutput, &payload); err != nil {
		t.Fatalf("Unmarshal(JudgeOutput) error = %v", err)
	}
	if payload["judge_kind"] != PlaceholderJudgeKind {
		t.Fatalf("judge_kind = %#v, want %q", payload["judge_kind"], PlaceholderJudgeKind)
	}
	if payload["judge_version"] != PlaceholderJudgeVersion {
		t.Fatalf("judge_version = %#v, want %q", payload["judge_version"], PlaceholderJudgeVersion)
	}
	if payload["judge_prompt_path"] != PlaceholderJudgePromptPath {
		t.Fatalf("judge_prompt_path = %#v, want %q", payload["judge_prompt_path"], PlaceholderJudgePromptPath)
	}
}

func TestPlaceholderJudgePromptArtifactPathIsVersioned(t *testing.T) {
	if !strings.HasPrefix(PlaceholderJudgePromptPath, "eval/prompts/") {
		t.Fatalf("PlaceholderJudgePromptPath = %q, want eval/prompts/*", PlaceholderJudgePromptPath)
	}
	if filepath.Ext(PlaceholderJudgePromptPath) != ".md" {
		t.Fatalf("PlaceholderJudgePromptPath = %q, want markdown prompt artifact", PlaceholderJudgePromptPath)
	}
}
