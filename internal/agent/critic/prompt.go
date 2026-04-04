package critic

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// CriticPromptVersion is the current versioned critic prompt identifier.
	CriticPromptVersion = "critic-v1"
)

const criticSystemPrompt = `You are a quality assurance critic for an enterprise AI assistant. Evaluate the draft answer against the provided context and tool results.

Score each dimension from 0.0 to 1.0:
- groundedness: Is every claim in the answer supported by the provided context? (1.0 = fully grounded, 0.0 = fabricated)
- citation_coverage: Does the answer reference the provided evidence appropriately? (1.0 = all evidence cited, 0.0 = no citations)
- tool_consistency: Are tool results correctly reflected in the answer? (1.0 = consistent, 0.0 = contradicts tools)

Determine the risk level: "low", "medium", or "high"
- low: answer is safe, accurate, well-grounded
- medium: minor gaps or uncertain claims
- high: contains potentially harmful, fabricated, or policy-violating content

Determine the verdict: "approve", "revise", "promote_workflow", or "reject"
- approve: answer is ready to send
- revise: answer needs improvements (provide revision_hints)
- promote_workflow: requires async human review
- reject: answer should be blocked (provide blocking_reasons)

Respond in JSON:
{
  "verdict": "approve|revise|promote_workflow|reject",
  "groundedness": 0.0-1.0,
  "citation_coverage": 0.0-1.0,
  "tool_consistency": 0.0-1.0,
  "risk_level": "low|medium|high",
  "missing_items": ["..."],
  "revision_hints": ["..."],
  "blocking_reasons": ["..."],
  "reasoning": "brief explanation"
}`

type llmCriticResponse struct {
	Verdict          string   `json:"verdict"`
	Groundedness     float64  `json:"groundedness"`
	CitationCoverage float64  `json:"citation_coverage"`
	ToolConsistency  float64  `json:"tool_consistency"`
	RiskLevel        string   `json:"risk_level"`
	MissingItems     []string `json:"missing_items"`
	RevisionHints    []string `json:"revision_hints"`
	BlockingReasons  []string `json:"blocking_reasons"`
	Reasoning        string   `json:"reasoning"`
}

func buildCriticUserMessage(input CriticInput) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("## Query Context\nPlan intent: %s", input.Plan.Intent))

	if input.Retrieval != nil && len(input.Retrieval.EvidenceBlocks) > 0 {
		parts = append(parts, "\n## Retrieved Evidence")
		for _, block := range input.Retrieval.EvidenceBlocks {
			parts = append(parts, fmt.Sprintf("- %s %s: %s", block.CitationLabel, block.SourceTitle, block.Snippet))
		}
	} else {
		parts = append(parts, "\n## Retrieved Evidence\nNo evidence was retrieved.")
	}

	if len(input.ToolResults) > 0 {
		parts = append(parts, "\n## Tool Results")
		for _, tr := range input.ToolResults {
			parts = append(parts, fmt.Sprintf("- %s: status=%s output=%s", tr.ToolName, tr.Status, tr.OutputSummary))
		}
	}

	parts = append(parts, fmt.Sprintf("\n## Draft Answer\n%s", input.DraftAnswer))

	return strings.Join(parts, "\n")
}

func parseCriticResponse(content string) (llmCriticResponse, error) {
	cleaned := stripCodeFencesCritic(content)
	var resp llmCriticResponse
	if err := json.Unmarshal([]byte(cleaned), &resp); err != nil {
		return llmCriticResponse{}, fmt.Errorf("parse critic JSON: %w", err)
	}
	return resp, nil
}

func validateCriticResponse(resp llmCriticResponse) error {
	switch resp.Verdict {
	case VerdictApprove, VerdictRevise, VerdictPromoteWorkflow, VerdictReject:
		// ok
	default:
		return fmt.Errorf("invalid verdict %q", resp.Verdict)
	}
	switch resp.RiskLevel {
	case RiskLevelLow, RiskLevelMedium, RiskLevelHigh:
		// ok
	case "":
		// allow empty, default to low
	default:
		return fmt.Errorf("invalid risk level %q", resp.RiskLevel)
	}
	return nil
}

func toLLMVerdict(resp llmCriticResponse) CriticVerdict {
	riskLevel := resp.RiskLevel
	if riskLevel == "" {
		riskLevel = RiskLevelLow
	}
	return CriticVerdict{
		Verdict:          resp.Verdict,
		Groundedness:     clamp01(resp.Groundedness),
		CitationCoverage: clamp01(resp.CitationCoverage),
		ToolConsistency:  clamp01(resp.ToolConsistency),
		RiskLevel:        riskLevel,
		MissingItems:     resp.MissingItems,
		RevisionHints:    resp.RevisionHints,
		BlockingReasons:  resp.BlockingReasons,
		Source:           CriticSourceLLM,
		PromptVersion:    CriticPromptVersion,
	}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func stripCodeFencesCritic(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
	}
	if strings.HasSuffix(s, "```") {
		s = s[:len(s)-3]
	}
	return strings.TrimSpace(s)
}
