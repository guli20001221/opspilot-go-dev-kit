package critic

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"opspilot-go/internal/llm"

	agenttool "opspilot-go/internal/agent/tool"
)

// Service evaluates the current draft and runtime artifacts.
type Service struct {
	llmProvider llm.Provider
}

// NewService constructs the critic service with deterministic rule-based evaluation.
func NewService() *Service {
	return &Service{}
}

// NewServiceWithLLM constructs the critic service with LLM-backed evaluation.
func NewServiceWithLLM(provider llm.Provider) *Service {
	return &Service{llmProvider: provider}
}

// Review returns a verdict for the current draft. Uses LLM when configured,
// with graceful fallback to deterministic rules.
func (s *Service) Review(ctx context.Context, input CriticInput) (CriticVerdict, error) {
	// Check for tool approval requirement first (deterministic, always applies)
	for _, toolResult := range input.ToolResults {
		if toolResult.Status == agenttool.StatusApprovalRequired {
			return CriticVerdict{
				Verdict:          VerdictPromoteWorkflow,
				Groundedness:     1,
				CitationCoverage: 1,
				ToolConsistency:  0.5,
				RiskLevel:        RiskLevelMedium,
				BlockingReasons:  []string{"tool approval required"},
				Source:           CriticSourceRule,
			}, nil
		}
	}

	// Try LLM-backed review
	if s.llmProvider != nil {
		if _, isPlaceholder := s.llmProvider.(*llm.PlaceholderProvider); !isPlaceholder {
			verdict, err := s.reviewWithLLM(ctx, input)
			if err != nil {
				slog.Warn("llm critic failed, falling back to rules",
					slog.Any("error", err),
				)
			} else {
				return verdict, nil
			}
		}
	}

	// Deterministic rule-based fallback
	return s.reviewWithRules(ctx, input)
}

func (s *Service) reviewWithLLM(ctx context.Context, input CriticInput) (CriticVerdict, error) {
	callCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	temp := llm.TemperaturePtr(0)
	resp, err := s.llmProvider.Complete(callCtx, llm.CompletionRequest{
		SystemPrompt:   criticSystemPrompt,
		Messages:       []llm.Message{{Role: "user", Content: buildCriticUserMessage(input)}},
		MaxTokens:      512,
		Temperature:    temp,
		ResponseFormat: llm.ResponseFormatJSON,
	})
	if err != nil {
		return CriticVerdict{}, err
	}

	parsed, err := parseCriticResponse(resp.Content)
	if err != nil {
		return CriticVerdict{}, err
	}

	if err := validateCriticResponse(parsed); err != nil {
		return CriticVerdict{}, err
	}

	verdict := toLLMVerdict(parsed)

	slog.Info("llm critic verdict",
		slog.String("verdict", verdict.Verdict),
		slog.Float64("groundedness", verdict.Groundedness),
		slog.Float64("citation_coverage", verdict.CitationCoverage),
		slog.String("risk_level", verdict.RiskLevel),
		slog.String("prompt_version", verdict.PromptVersion),
	)

	return verdict, nil
}

func (s *Service) reviewWithRules(_ context.Context, input CriticInput) (CriticVerdict, error) {
	verdict := CriticVerdict{
		Groundedness:     1,
		CitationCoverage: 1,
		ToolConsistency:  1,
		RiskLevel:        RiskLevelLow,
		Source:           CriticSourceRule,
	}

	for _, toolResult := range input.ToolResults {
		if toolResult.Status == agenttool.StatusFailed {
			verdict.Verdict = VerdictRevise
			verdict.RiskLevel = RiskLevelMedium
			verdict.ToolConsistency = 0.3
			verdict.RevisionHints = append(verdict.RevisionHints, "resolve failed tool execution")
		}
	}

	if input.Plan.RequiresRetrieval {
		if input.Retrieval == nil || len(input.Retrieval.EvidenceBlocks) == 0 {
			verdict.Groundedness = 0.4
			verdict.CitationCoverage = 0
			verdict.Verdict = VerdictRevise
			verdict.MissingItems = append(verdict.MissingItems, "retrieval_evidence")
			verdict.RevisionHints = append(verdict.RevisionHints, "add grounded evidence before answering")
			return verdict, nil
		}

		citationMatches := 0
		for _, block := range input.Retrieval.EvidenceBlocks {
			if block.CitationLabel != "" && strings.Contains(input.DraftAnswer, block.CitationLabel) {
				citationMatches++
			}
		}

		verdict.CitationCoverage = float64(citationMatches) / float64(len(input.Retrieval.EvidenceBlocks))
		if verdict.CitationCoverage < 0.80 {
			verdict.Verdict = VerdictRevise
			verdict.RevisionHints = append(verdict.RevisionHints, "add citations for supporting evidence")
			return verdict, nil
		}
	}

	if verdict.Verdict == "" {
		verdict.Verdict = VerdictApprove
	}

	return verdict, nil
}
