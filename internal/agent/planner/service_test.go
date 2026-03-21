package planner

import (
	"context"
	"testing"

	"opspilot-go/internal/contextengine"
)

func TestServicePlanKnowledgeQuestion(t *testing.T) {
	svc := NewService()

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-1",
		TraceID:     "trace-1",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "what is the incident SOP?",
		Context: contextengine.PlannerContext{
			Blocks: []contextengine.Block{{Kind: contextengine.BlockKindRecentTurns, Content: "user: what is the incident SOP?"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentKnowledgeQA {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentKnowledgeQA)
	}
	if !got.RequiresRetrieval {
		t.Fatal("RequiresRetrieval = false, want true")
	}
	if got.RequiresTool {
		t.Fatal("RequiresTool = true, want false")
	}
	if got.RequiresWorkflow {
		t.Fatal("RequiresWorkflow = true, want false")
	}
	if got.MaxSteps > 6 {
		t.Fatalf("MaxSteps = %d, want <= 6", got.MaxSteps)
	}
	assertStepKinds(t, got.Steps, StepKindRetrieve, StepKindSynthesize, StepKindCritic)
}

func TestServicePlanTaskRequestPromotesWorkflow(t *testing.T) {
	svc := NewService()

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-2",
		TraceID:     "trace-2",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "task",
		UserMessage: "generate a report for last week's incidents",
		Context: contextengine.PlannerContext{
			Blocks: []contextengine.Block{{Kind: contextengine.BlockKindUserProfile, Content: "tenant_id=tenant-1"}},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if got.Intent != IntentReportRequest {
		t.Fatalf("Intent = %q, want %q", got.Intent, IntentReportRequest)
	}
	if !got.RequiresWorkflow {
		t.Fatal("RequiresWorkflow = false, want true")
	}
	if len(got.Steps) == 0 || got.Steps[0].Kind != StepKindPromoteWorkflow {
		t.Fatalf("first step = %#v, want promote_workflow", got.Steps)
	}
}

func TestServicePlanTicketSearchUsesReadOnlyTool(t *testing.T) {
	svc := NewService()

	got, err := svc.Plan(context.Background(), PlanInput{
		RequestID:   "req-3",
		TraceID:     "trace-3",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		Mode:        "chat",
		UserMessage: "search related ticket history",
		Context:     contextengine.PlannerContext{},
		AvailableTools: []ToolDescriptor{
			{Name: "ticket_search", ReadOnly: true},
		},
	})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if !got.RequiresTool {
		t.Fatal("RequiresTool = false, want true")
	}
	if got.RequiresApproval {
		t.Fatal("RequiresApproval = true, want false")
	}
	assertToolStep(t, got.Steps, "ticket_search", true)
}

func assertStepKinds(t *testing.T, steps []PlanStep, wantKinds ...string) {
	t.Helper()

	if len(steps) != len(wantKinds) {
		t.Fatalf("len(steps) = %d, want %d", len(steps), len(wantKinds))
	}
	for i, wantKind := range wantKinds {
		if steps[i].Kind != wantKind {
			t.Fatalf("steps[%d].Kind = %q, want %q", i, steps[i].Kind, wantKind)
		}
	}
}

func assertToolStep(t *testing.T, steps []PlanStep, wantTool string, wantReadOnly bool) {
	t.Helper()

	for _, step := range steps {
		if step.Kind != StepKindTool {
			continue
		}
		if step.ToolName != wantTool {
			t.Fatalf("ToolName = %q, want %q", step.ToolName, wantTool)
		}
		if step.ReadOnly != wantReadOnly {
			t.Fatalf("ReadOnly = %v, want %v", step.ReadOnly, wantReadOnly)
		}
		return
	}

	t.Fatalf("tool step for %q not found in %#v", wantTool, steps)
}
