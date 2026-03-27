package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	// HTTPJSONJudgeKind identifies the HTTP-backed eval judge implementation.
	HTTPJSONJudgeKind = "http_json"
	// PlaceholderJudgeProvider identifies the deterministic built-in judge.
	PlaceholderJudgeProvider = "placeholder"
)

// JudgeOptions configures eval-judge selection.
type JudgeOptions struct {
	Provider   string
	BaseURL    string
	APIKey     string
	Model      string
	PromptPath string
	Timeout    time.Duration
	Client     *http.Client
}

// HTTPJSONJudgeOptions configures the HTTP-backed eval judge.
type HTTPJSONJudgeOptions struct {
	BaseURL    string
	APIKey     string
	Model      string
	PromptPath string
	Timeout    time.Duration
	Client     *http.Client
}

// HTTPJSONJudge evaluates one eval-run item through an external HTTP judge.
type HTTPJSONJudge struct {
	baseURL    string
	apiKey     string
	model      string
	promptPath string
	client     *http.Client
}

// NewHTTPJSONJudge constructs an HTTP-backed eval judge.
func NewHTTPJSONJudge(options HTTPJSONJudgeOptions) *HTTPJSONJudge {
	client := options.Client
	if client == nil {
		timeout := options.Timeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	return &HTTPJSONJudge{
		baseURL:    strings.TrimRight(strings.TrimSpace(options.BaseURL), "/"),
		apiKey:     strings.TrimSpace(options.APIKey),
		model:      strings.TrimSpace(options.Model),
		promptPath: fallbackString(strings.TrimSpace(options.PromptPath), PlaceholderJudgePromptPath),
		client:     client,
	}
}

// NewConfiguredJudge builds the requested eval-judge runtime or falls back to the deterministic judge.
func NewConfiguredJudge(options JudgeOptions) (RunJudge, error) {
	switch strings.TrimSpace(options.Provider) {
	case "", PlaceholderJudgeProvider:
		return NewPlaceholderJudge(), nil
	case HTTPJSONJudgeKind:
		if strings.TrimSpace(options.BaseURL) == "" {
			return nil, fmt.Errorf("http_json judge requires base URL")
		}
		if strings.TrimSpace(options.Model) == "" {
			return nil, fmt.Errorf("http_json judge requires model")
		}
		return NewHTTPJSONJudge(HTTPJSONJudgeOptions{
			BaseURL:    options.BaseURL,
			APIKey:     options.APIKey,
			Model:      options.Model,
			PromptPath: fallbackString(strings.TrimSpace(options.PromptPath), PlaceholderJudgePromptPath),
			Timeout:    options.Timeout,
			Client:     options.Client,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported eval judge provider %q", options.Provider)
	}
}

// Version returns the stable version identifier for this HTTP-backed judge configuration.
func (j *HTTPJSONJudge) Version() string {
	promptBase := strings.TrimSuffix(path.Base(j.promptPath), path.Ext(j.promptPath))
	return fmt.Sprintf("%s/%s/%s", HTTPJSONJudgeKind, fallbackString(j.model, "default-model"), promptBase)
}

// PromptPath returns the versioned prompt artifact path used by the judge.
func (j *HTTPJSONJudge) PromptPath() string {
	return j.promptPath
}

// BuildItemResults evaluates each snapped run item through the HTTP judge.
func (j *HTTPJSONJudge) BuildItemResults(ctx context.Context, items []EvalRunItem, status string, detail string, updatedAt time.Time) ([]EvalRunItemResult, error) {
	prompt, err := readJudgePromptArtifact(j.promptPath)
	if err != nil {
		return nil, fmt.Errorf("read judge prompt artifact: %w", err)
	}

	results := make([]EvalRunItemResult, 0, len(items))
	for _, item := range items {
		decision, rawResponse, err := j.evaluateItem(ctx, prompt, item, status, detail)
		if err != nil {
			return nil, fmt.Errorf("evaluate item %s: %w", item.EvalCaseID, err)
		}
		results = append(results, EvalRunItemResult{
			EvalCaseID:   item.EvalCaseID,
			Status:       status,
			Verdict:      decision.Verdict,
			Detail:       decision.Rationale,
			Score:        decision.Score,
			JudgeVersion: j.Version(),
			JudgeOutput:  mustMarshalHTTPJudgeOutput(j, decision, rawResponse),
			UpdatedAt:    updatedAt,
		})
	}
	return results, nil
}

type httpJudgeRequest struct {
	Model      string      `json:"model,omitempty"`
	PromptPath string      `json:"prompt_path"`
	Prompt     string      `json:"prompt"`
	RunStatus  string      `json:"run_status"`
	RunDetail  string      `json:"run_detail"`
	Item       EvalRunItem `json:"item"`
}

type httpJudgeDecision struct {
	Verdict   string  `json:"verdict"`
	Score     float64 `json:"score"`
	Rationale string  `json:"rationale"`
}

func (j *HTTPJSONJudge) evaluateItem(ctx context.Context, prompt string, item EvalRunItem, status string, detail string) (httpJudgeDecision, json.RawMessage, error) {
	payload, err := json.Marshal(httpJudgeRequest{
		Model:      j.model,
		PromptPath: j.promptPath,
		Prompt:     prompt,
		RunStatus:  status,
		RunDetail:  detail,
		Item:       item,
	})
	if err != nil {
		return httpJudgeDecision{}, nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, j.baseURL+"/judge", bytes.NewReader(payload))
	if err != nil {
		return httpJudgeDecision{}, nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if j.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+j.apiKey)
	}

	resp, err := j.client.Do(req)
	if err != nil {
		return httpJudgeDecision{}, nil, fmt.Errorf("request provider: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return httpJudgeDecision{}, nil, fmt.Errorf("read provider response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return httpJudgeDecision{}, nil, fmt.Errorf("provider status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var decision httpJudgeDecision
	if err := json.Unmarshal(body, &decision); err != nil {
		return httpJudgeDecision{}, nil, fmt.Errorf("decode provider response: %w", err)
	}
	if decision.Verdict != RunItemVerdictPass && decision.Verdict != RunItemVerdictFail {
		return httpJudgeDecision{}, nil, fmt.Errorf("invalid verdict %q", decision.Verdict)
	}
	if decision.Rationale == "" {
		return httpJudgeDecision{}, nil, fmt.Errorf("missing rationale")
	}

	return decision, json.RawMessage(body), nil
}

func mustMarshalHTTPJudgeOutput(judge *HTTPJSONJudge, decision httpJudgeDecision, rawResponse json.RawMessage) json.RawMessage {
	payload, err := json.Marshal(struct {
		JudgeKind        string          `json:"judge_kind"`
		JudgeVersion     string          `json:"judge_version"`
		JudgePromptPath  string          `json:"judge_prompt_path"`
		Verdict          string          `json:"verdict"`
		Score            float64         `json:"score"`
		Rationale        string          `json:"rationale"`
		ProviderResponse json.RawMessage `json:"provider_response"`
	}{
		JudgeKind:        HTTPJSONJudgeKind,
		JudgeVersion:     judge.Version(),
		JudgePromptPath:  judge.PromptPath(),
		Verdict:          decision.Verdict,
		Score:            decision.Score,
		Rationale:        decision.Rationale,
		ProviderResponse: rawResponse,
	})
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return payload
}

func readJudgePromptArtifact(promptPath string) (string, error) {
	candidates := make([]string, 0, 16)
	appendCandidates := func(base string) {
		current := base
		for range 8 {
			candidates = append(candidates, filepath.Join(current, filepath.FromSlash(promptPath)))
			parent := filepath.Dir(current)
			if parent == current {
				break
			}
			current = parent
		}
	}

	appendCandidates(".")
	if cwd, err := os.Getwd(); err == nil {
		appendCandidates(cwd)
	}
	if exe, err := os.Executable(); err == nil {
		appendCandidates(filepath.Dir(exe))
	}

	var lastErr error
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return string(data), nil
		}
		lastErr = err
	}

	if lastErr == nil {
		lastErr = os.ErrNotExist
	}
	return "", fmt.Errorf("%s: %w", promptPath, lastErr)
}
