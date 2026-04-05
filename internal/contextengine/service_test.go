package contextengine

import (
	"context"
	"testing"
)

func TestServiceBuildIncludesCoreBlocksAndAssemblyLog(t *testing.T) {
	svc := NewService(Config{
		MaxBlocks: 8,
		Budget:    128,
	})

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID: "req-1",
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Mode:      "chat",
		RecentTurns: []Turn{
			{Role: "user", Content: "first"},
			{Role: "assistant", Content: "second"},
		},
		SessionSummary: "summary text",
		TaskScratchpad: "scratchpad",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if len(got.Planner.Blocks) == 0 {
		t.Fatal("Planner.Blocks is empty")
	}
	if len(got.Retrieval.Blocks) == 0 {
		t.Fatal("Retrieval.Blocks is empty")
	}
	if len(got.Critic.Blocks) == 0 {
		t.Fatal("Critic.Blocks is empty")
	}
	if got.Log.RequestID != "req-1" {
		t.Fatalf("Log.RequestID = %q, want %q", got.Log.RequestID, "req-1")
	}
	if len(got.Log.IncludedBlocks) == 0 {
		t.Fatal("Log.IncludedBlocks is empty")
	}
	if got.Log.BudgetLimit != 128 {
		t.Fatalf("Log.BudgetLimit = %d, want %d", got.Log.BudgetLimit, 128)
	}

	assertHasBlockKind(t, got.Planner.Blocks, BlockKindUserProfile)
	assertHasBlockKind(t, got.Planner.Blocks, BlockKindRecentTurns)
	assertHasBlockKind(t, got.Planner.Blocks, BlockKindSessionSummary)
	assertHasBlockKind(t, got.Planner.Blocks, BlockKindTaskScratchpad)
}

func TestServiceBuildDropsLowestPriorityBlocksWhenBudgetExceeded(t *testing.T) {
	svc := NewService(Config{
		MaxBlocks: 8,
		Budget:    18,
	})

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID:      "req-2",
		TenantID:       "tenant-1",
		UserID:         "user-1",
		Mode:           "chat",
		SessionSummary: "summary block that should be dropped first",
		TaskScratchpad: "scratchpad block that should also be dropped",
		RecentTurns: []Turn{
			{Role: "user", Content: "short"},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	assertHasBlockKind(t, got.Planner.Blocks, BlockKindUserProfile)
	assertHasBlockKind(t, got.Planner.Blocks, BlockKindRecentTurns)
	assertMissingBlockKind(t, got.Planner.Blocks, BlockKindSessionSummary)
	assertMissingBlockKind(t, got.Planner.Blocks, BlockKindTaskScratchpad)
	if len(got.Log.DroppedBlocks) == 0 {
		t.Fatal("Log.DroppedBlocks is empty")
	}
}

func assertHasBlockKind(t *testing.T, blocks []Block, want string) {
	t.Helper()

	for _, block := range blocks {
		if block.Kind == want {
			return
		}
	}

	t.Fatalf("block kind %q not found in %#v", want, blocks)
}

func TestServiceBuildStageSpecificFiltering(t *testing.T) {
	svc := NewService(Config{Budget: 4096})

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID:  "req-stage",
		TenantID:   "tenant-1",
		UserID:     "user-1",
		Mode:       "chat",
		RecentTurns: []Turn{{Role: "user", Content: "hello"}},
		RetrievalResults: []EvidenceSnippet{
			{SourceTitle: "Doc A", Snippet: "evidence text", CitationLabel: "[1]", Score: 0.9},
		},
		ToolResults: []ToolResultSnippet{
			{ToolName: "ticket_search", Status: "succeeded", OutputSummary: "found 3 matches"},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Planner should NOT have retrieval evidence or tool results
	assertMissingBlockKind(t, got.Planner.Blocks, BlockKindRetrievalEvidence)
	assertMissingBlockKind(t, got.Planner.Blocks, BlockKindToolResult)
	assertHasBlockKind(t, got.Planner.Blocks, BlockKindRecentTurns)

	// Retrieval should NOT have evidence or tool results (it provides context for query)
	assertMissingBlockKind(t, got.Retrieval.Blocks, BlockKindRetrievalEvidence)
	assertMissingBlockKind(t, got.Retrieval.Blocks, BlockKindToolResult)

	// Critic SHOULD have everything including evidence and tool results
	assertHasBlockKind(t, got.Critic.Blocks, BlockKindRetrievalEvidence)
	assertHasBlockKind(t, got.Critic.Blocks, BlockKindToolResult)
	assertHasBlockKind(t, got.Critic.Blocks, BlockKindRecentTurns)
}

func TestServiceBuildPerStageBudgets(t *testing.T) {
	svc := NewService(Config{
		Budget:        4096,
		PlannerBudget: 20, // tight budget for planner
		CriticBudget:  4096,
	})

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID:      "req-budgets",
		TenantID:       "tenant-1",
		UserID:         "user-1",
		Mode:           "chat",
		RecentTurns:    []Turn{{Role: "user", Content: "this is a longer message that should push the planner over budget"}},
		SessionSummary: "detailed session summary that adds more tokens",
		TaskScratchpad: "task notes with additional context",
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Planner has tight budget — should drop lower-priority blocks
	if len(got.Planner.Blocks) >= 4 {
		t.Fatalf("Planner.Blocks = %d, want < 4 (tight budget should drop some)", len(got.Planner.Blocks))
	}

	// Critic has large budget — should keep all
	if len(got.Critic.Blocks) < len(got.Planner.Blocks) {
		t.Fatalf("Critic.Blocks (%d) < Planner.Blocks (%d), want critic to have more with larger budget",
			len(got.Critic.Blocks), len(got.Planner.Blocks))
	}
}

func TestServiceBuildRetrievalEvidenceFormatting(t *testing.T) {
	svc := NewService(Config{Budget: 4096})

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID: "req-evidence",
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Mode:      "chat",
		RetrievalResults: []EvidenceSnippet{
			{SourceTitle: "Password Reset Guide", Snippet: "Navigate to Settings...", CitationLabel: "[1]", Score: 0.95},
			{SourceTitle: "Account Recovery", Snippet: "Contact support...", CitationLabel: "[2]", Score: 0.82},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Critic should have evidence block with formatted content
	for _, block := range got.Critic.Blocks {
		if block.Kind == BlockKindRetrievalEvidence {
			if block.Priority != 80 {
				t.Fatalf("evidence Priority = %d, want 80", block.Priority)
			}
			if block.EstimatedTokens <= 0 {
				t.Fatal("evidence EstimatedTokens = 0")
			}
			return
		}
	}
	t.Fatal("retrieval_evidence block not found in critic context")
}

func assertMissingBlockKind(t *testing.T, blocks []Block, want string) {
	t.Helper()

	for _, block := range blocks {
		if block.Kind == want {
			t.Fatalf("unexpected block kind %q found in %#v", want, blocks)
		}
	}
}
