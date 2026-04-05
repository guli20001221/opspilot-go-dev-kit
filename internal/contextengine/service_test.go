package contextengine

import (
	"context"
	"fmt"
	"strings"
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

// --- Conversation Summarizer tests ---

type mockSummarizer struct {
	called    bool
	turnCount int
	result    string
	err       error
}

func (m *mockSummarizer) Summarize(_ context.Context, turns []Turn) (string, error) {
	m.called = true
	m.turnCount = len(turns)
	if m.err != nil {
		return "", m.err
	}
	return m.result, nil
}

func TestServiceBuildCompressesOlderTurnsWhenThresholdExceeded(t *testing.T) {
	summarizer := &mockSummarizer{result: "User asked about passwords and got reset instructions."}
	svc := NewServiceWithSummarizer(Config{
		Budget:               4096,
		SummaryTurnThreshold: 2, // keep last 2 turns, compress the rest
	}, summarizer)

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID: "req-summarize",
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Mode:      "chat",
		RecentTurns: []Turn{
			{Role: "user", Content: "How do I reset my password?"},
			{Role: "assistant", Content: "Go to Settings > Security > Reset Password."},
			{Role: "user", Content: "What about 2FA?"},
			{Role: "assistant", Content: "Enable in Security settings."},
			{Role: "user", Content: "Thanks, one more question about account recovery."},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if !summarizer.called {
		t.Fatal("Summarizer was not called")
	}
	if summarizer.turnCount != 3 {
		t.Fatalf("Summarizer received %d turns, want 3 (5 total - 2 kept)", summarizer.turnCount)
	}

	// Session summary block should contain the compressed summary
	assertHasBlockKind(t, got.Planner.Blocks, BlockKindSessionSummary)
	for _, block := range got.Planner.Blocks {
		if block.Kind == BlockKindSessionSummary {
			if block.Content != "User asked about passwords and got reset instructions." {
				t.Fatalf("summary content = %q, want compressed summary", block.Content)
			}
		}
		// Recent turns should only have the last 2 turns
		if block.Kind == BlockKindRecentTurns {
			if !strings.Contains(block.Content, "account recovery") {
				t.Fatal("recent turns should contain the last kept turn")
			}
			if strings.Contains(block.Content, "How do I reset") {
				t.Fatal("recent turns should NOT contain the compressed older turn")
			}
		}
	}
}

func TestServiceBuildSkipsSummarizationWhenBelowThreshold(t *testing.T) {
	summarizer := &mockSummarizer{result: "should not be called"}
	svc := NewServiceWithSummarizer(Config{
		Budget:               4096,
		SummaryTurnThreshold: 10, // threshold higher than turn count
	}, summarizer)

	_, err := svc.Build(context.Background(), BuildInput{
		RequestID:   "req-no-summarize",
		TenantID:    "tenant-1",
		UserID:      "user-1",
		Mode:        "chat",
		RecentTurns: []Turn{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	if summarizer.called {
		t.Fatal("Summarizer should not be called when below threshold")
	}
}

func TestServiceBuildGracefulOnSummarizerError(t *testing.T) {
	summarizer := &mockSummarizer{err: fmt.Errorf("LLM unavailable")}
	svc := NewServiceWithSummarizer(Config{
		Budget:               4096,
		SummaryTurnThreshold: 2,
	}, summarizer)

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID: "req-summarize-err",
		TenantID:  "tenant-1",
		UserID:    "user-1",
		Mode:      "chat",
		RecentTurns: []Turn{
			{Role: "user", Content: "turn 1"},
			{Role: "assistant", Content: "turn 2"},
			{Role: "user", Content: "turn 3"},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v, want nil (graceful fallback)", err)
	}

	// All turns should be preserved as-is since summarization failed
	for _, block := range got.Planner.Blocks {
		if block.Kind == BlockKindRecentTurns {
			if !strings.Contains(block.Content, "turn 1") {
				t.Fatal("all turns should be preserved when summarizer fails")
			}
		}
	}
}

func TestServiceBuildSummarizerAppendsToExistingSummary(t *testing.T) {
	summarizer := &mockSummarizer{result: "Compressed: user asked about 2FA."}
	svc := NewServiceWithSummarizer(Config{
		Budget:               4096,
		SummaryTurnThreshold: 2,
	}, summarizer)

	got, err := svc.Build(context.Background(), BuildInput{
		RequestID:      "req-append-summary",
		TenantID:       "tenant-1",
		UserID:         "user-1",
		Mode:           "chat",
		SessionSummary: "Previous: user asked about password reset.",
		RecentTurns: []Turn{
			{Role: "user", Content: "How do I enable 2FA?"},
			{Role: "assistant", Content: "Go to Security settings."},
			{Role: "user", Content: "What about recovery codes?"},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	for _, block := range got.Planner.Blocks {
		if block.Kind == BlockKindSessionSummary {
			want := "Previous: user asked about password reset.\n\nCompressed: user asked about 2FA."
			if block.Content != want {
				t.Fatalf("summary = %q, want concatenated %q", block.Content, want)
			}
			return
		}
	}
	t.Fatal("session_summary block not found")
}

func TestServiceBuildEmptyInput(t *testing.T) {
	svc := NewService(Config{Budget: 4096})
	got, err := svc.Build(context.Background(), BuildInput{RequestID: "req-empty"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(got.Planner.Blocks) != 0 {
		t.Fatalf("Planner.Blocks = %d, want 0 for empty input", len(got.Planner.Blocks))
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

	// Critic should have 2 evidence blocks with decaying priority
	var evidenceBlocks []Block
	for _, block := range got.Critic.Blocks {
		if block.Kind == BlockKindRetrievalEvidence {
			evidenceBlocks = append(evidenceBlocks, block)
		}
	}
	if len(evidenceBlocks) != 2 {
		t.Fatalf("evidence block count = %d, want 2", len(evidenceBlocks))
	}
	// First evidence (highest score after sort) should have priority 80
	if evidenceBlocks[0].Priority != 80 {
		t.Fatalf("first evidence Priority = %d, want 80", evidenceBlocks[0].Priority)
	}
	// Second evidence should have priority 79 (decayed by position)
	if evidenceBlocks[1].Priority != 79 {
		t.Fatalf("second evidence Priority = %d, want 79", evidenceBlocks[1].Priority)
	}
	if evidenceBlocks[0].EstimatedTokens <= 0 {
		t.Fatal("evidence EstimatedTokens = 0")
	}
}

func assertMissingBlockKind(t *testing.T, blocks []Block, want string) {
	t.Helper()

	for _, block := range blocks {
		if block.Kind == want {
			t.Fatalf("unexpected block kind %q found in %#v", want, blocks)
		}
	}
}
