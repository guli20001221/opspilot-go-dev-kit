package critic

import (
	"context"
	"testing"

	"opspilot-go/internal/agent/planner"
	agenttool "opspilot-go/internal/agent/tool"
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
