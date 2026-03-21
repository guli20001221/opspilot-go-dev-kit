package critic

import (
	"context"
	"strings"

	agenttool "opspilot-go/internal/agent/tool"
)

// Service evaluates the current draft and runtime artifacts.
type Service struct{}

// NewService constructs the critic service.
func NewService() *Service {
	return &Service{}
}

// Review returns a deterministic verdict for the current draft.
func (s *Service) Review(_ context.Context, input CriticInput) (CriticVerdict, error) {
	verdict := CriticVerdict{
		Groundedness:     1,
		CitationCoverage: 1,
		ToolConsistency:  1,
		RiskLevel:        RiskLevelLow,
	}

	for _, toolResult := range input.ToolResults {
		switch toolResult.Status {
		case agenttool.StatusApprovalRequired:
			verdict.Verdict = VerdictPromoteWorkflow
			verdict.RiskLevel = RiskLevelMedium
			verdict.ToolConsistency = 0.5
			verdict.BlockingReasons = append(verdict.BlockingReasons, "tool approval required")
			return verdict, nil
		case agenttool.StatusFailed:
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
