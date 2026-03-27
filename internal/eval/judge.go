package eval

import (
	"context"
	"encoding/json"
	"time"
)

// PlaceholderJudgePromptPath identifies the versioned built-in judge prompt artifact.
const PlaceholderJudgePromptPath = "eval/prompts/placeholder-eval-judge-v1.md"

// RunJudge builds per-item judged results for one eval run.
type RunJudge interface {
	BuildItemResults(ctx context.Context, items []EvalRunItem, status string, detail string, updatedAt time.Time) ([]EvalRunItemResult, error)
	Version() string
	PromptPath() string
}

// PlaceholderJudge is the current deterministic built-in eval judge.
type PlaceholderJudge struct{}

// NewPlaceholderJudge constructs the built-in deterministic eval judge.
func NewPlaceholderJudge() *PlaceholderJudge {
	return &PlaceholderJudge{}
}

// Version returns the stable version identifier for the built-in judge.
func (j *PlaceholderJudge) Version() string {
	return PlaceholderJudgeVersion
}

// PromptPath returns the versioned prompt artifact path for the built-in judge.
func (j *PlaceholderJudge) PromptPath() string {
	return PlaceholderJudgePromptPath
}

// BuildItemResults constructs one judged result per eval-run item.
func (j *PlaceholderJudge) BuildItemResults(_ context.Context, items []EvalRunItem, status string, detail string, updatedAt time.Time) ([]EvalRunItemResult, error) {
	results := make([]EvalRunItemResult, 0, len(items))
	for _, item := range items {
		results = append(results, newPlaceholderRunItemResult(j, item.EvalCaseID, status, detail, updatedAt))
	}
	return results, nil
}

func newPlaceholderRunItemResult(judge *PlaceholderJudge, evalCaseID string, status string, detail string, updatedAt time.Time) EvalRunItemResult {
	result := EvalRunItemResult{
		EvalCaseID:   evalCaseID,
		Status:       status,
		Detail:       detail,
		JudgeVersion: judge.Version(),
		UpdatedAt:    updatedAt,
	}
	switch status {
	case RunItemResultSucceeded:
		result.Verdict = RunItemVerdictPass
		result.Score = 1
	default:
		result.Verdict = RunItemVerdictFail
		result.Score = 0
	}
	result.JudgeOutput = mustMarshalPlaceholderJudgeOutput(judge, result.Verdict, result.Score, detail)
	return result
}

func mustMarshalPlaceholderJudgeOutput(judge *PlaceholderJudge, verdict string, score float64, rationale string) json.RawMessage {
	payload, err := json.Marshal(struct {
		JudgeKind       string  `json:"judge_kind"`
		JudgeVersion    string  `json:"judge_version"`
		JudgePromptPath string  `json:"judge_prompt_path"`
		Verdict         string  `json:"verdict"`
		Score           float64 `json:"score"`
		Rationale       string  `json:"rationale"`
	}{
		JudgeKind:       PlaceholderJudgeKind,
		JudgeVersion:    judge.Version(),
		JudgePromptPath: judge.PromptPath(),
		Verdict:         verdict,
		Score:           score,
		Rationale:       rationale,
	})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return payload
}
