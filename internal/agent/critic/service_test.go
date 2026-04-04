package critic

import (
	"context"
	"fmt"
	"testing"

	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
	"opspilot-go/internal/llm"
	"opspilot-go/internal/retrieval"
)

func TestServiceReviewApprovesGroundedAnswer(t *testing.T) {
	svc := NewService()

	got, err := svc.Review(context.Background(), CriticInput{
		Plan: planner.ExecutionPlan{
			Intent:            planner.IntentKnowledgeQA,
			RequiresRetrieval: true,
		},
		Retrieval: &retrieval.RetrievalResult{
			EvidenceBlocks: []retrieval.EvidenceBlock{
				{EvidenceID: "evidence-1", CitationLabel: "[1]"},
			},
		},
		DraftAnswer: "Grounded answer [1]",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if got.Verdict != VerdictApprove {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, VerdictApprove)
	}
	if got.Groundedness < 0.70 {
		t.Fatalf("Groundedness = %.2f, want >= 0.70", got.Groundedness)
	}
	if got.CitationCoverage < 0.80 {
		t.Fatalf("CitationCoverage = %.2f, want >= 0.80", got.CitationCoverage)
	}
	if got.RiskLevel != RiskLevelLow {
		t.Fatalf("RiskLevel = %q, want %q", got.RiskLevel, RiskLevelLow)
	}
}

func TestServiceReviewRequestsRevisionWhenCitationsMissing(t *testing.T) {
	svc := NewService()

	got, err := svc.Review(context.Background(), CriticInput{
		Plan: planner.ExecutionPlan{
			Intent:            planner.IntentKnowledgeQA,
			RequiresRetrieval: true,
		},
		Retrieval: &retrieval.RetrievalResult{
			EvidenceBlocks: []retrieval.EvidenceBlock{
				{EvidenceID: "evidence-1", CitationLabel: "[1]"},
			},
		},
		DraftAnswer: "Answer without citations",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if got.Verdict != VerdictRevise {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, VerdictRevise)
	}
	if len(got.RevisionHints) == 0 {
		t.Fatal("RevisionHints is empty")
	}
}

func TestServiceReviewPromotesWorkflowOnApprovalRequiredTool(t *testing.T) {
	svc := NewService()

	got, err := svc.Review(context.Background(), CriticInput{
		Plan: planner.ExecutionPlan{
			Intent:           planner.IntentIncidentAssist,
			RequiresTool:     true,
			RequiresWorkflow: false,
		},
		ToolResults: []agenttool.ToolResult{
			{ToolName: "ticket_comment_create", Status: agenttool.StatusApprovalRequired, ApprovalRef: "approval-1"},
		},
		DraftAnswer: "Need approval",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}

	if got.Verdict != VerdictPromoteWorkflow {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, VerdictPromoteWorkflow)
	}
	if got.RiskLevel != RiskLevelMedium {
		t.Fatalf("RiskLevel = %q, want %q", got.RiskLevel, RiskLevelMedium)
	}
}

type mockCriticProvider struct {
	response string
	err      error
}

func (m *mockCriticProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	if m.err != nil {
		return llm.CompletionResponse{}, m.err
	}
	return llm.CompletionResponse{Content: m.response}, nil
}

func TestLLMCriticReturnsStructuredVerdict(t *testing.T) {
	provider := &mockCriticProvider{
		response: `{"verdict":"approve","groundedness":0.95,"citation_coverage":0.9,"tool_consistency":1.0,"risk_level":"low","reasoning":"well grounded"}`,
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Review(context.Background(), CriticInput{
		Plan:        planner.ExecutionPlan{Intent: planner.IntentKnowledgeQA},
		DraftAnswer: "A good answer.",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Verdict != VerdictApprove {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, VerdictApprove)
	}
	if got.Source != CriticSourceLLM {
		t.Fatalf("Source = %q, want %q", got.Source, CriticSourceLLM)
	}
	if got.Groundedness != 0.95 {
		t.Fatalf("Groundedness = %f, want 0.95", got.Groundedness)
	}
	if got.PromptVersion != CriticPromptVersion {
		t.Fatalf("PromptVersion = %q, want %q", got.PromptVersion, CriticPromptVersion)
	}
}

func TestLLMCriticFallsBackOnError(t *testing.T) {
	provider := &mockCriticProvider{err: fmt.Errorf("llm unavailable")}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Review(context.Background(), CriticInput{
		Plan: planner.ExecutionPlan{Intent: planner.IntentKnowledgeQA},
		Retrieval: &retrieval.RetrievalResult{
			EvidenceBlocks: []retrieval.EvidenceBlock{{CitationLabel: "[1]"}},
		},
		DraftAnswer: "Answer [1]",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Source != CriticSourceRule {
		t.Fatalf("Source = %q, want %q (should fallback to rules)", got.Source, CriticSourceRule)
	}
}

func TestLLMCriticFallsBackOnInvalidJSON(t *testing.T) {
	provider := &mockCriticProvider{response: "not json at all"}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Review(context.Background(), CriticInput{
		Plan:        planner.ExecutionPlan{Intent: planner.IntentKnowledgeQA},
		DraftAnswer: "Some answer",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Source != CriticSourceRule {
		t.Fatalf("Source = %q, want %q (should fallback)", got.Source, CriticSourceRule)
	}
}

func TestLLMCriticReviseVerdict(t *testing.T) {
	provider := &mockCriticProvider{
		response: `{"verdict":"revise","groundedness":0.3,"citation_coverage":0.2,"tool_consistency":1.0,"risk_level":"medium","revision_hints":["add citations"]}`,
	}
	svc := NewServiceWithLLM(provider)

	got, err := svc.Review(context.Background(), CriticInput{
		Plan:        planner.ExecutionPlan{Intent: planner.IntentKnowledgeQA},
		DraftAnswer: "Bad answer",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Verdict != VerdictRevise {
		t.Fatalf("Verdict = %q, want %q", got.Verdict, VerdictRevise)
	}
	if len(got.RevisionHints) != 1 {
		t.Fatalf("RevisionHints = %v, want 1 hint", got.RevisionHints)
	}
}

func TestRuleBasedCriticSetsSourceField(t *testing.T) {
	svc := NewService()
	got, err := svc.Review(context.Background(), CriticInput{
		Plan:        planner.ExecutionPlan{Intent: planner.IntentKnowledgeQA},
		DraftAnswer: "Simple answer",
	})
	if err != nil {
		t.Fatalf("Review() error = %v", err)
	}
	if got.Source != CriticSourceRule {
		t.Fatalf("Source = %q, want %q", got.Source, CriticSourceRule)
	}
}
