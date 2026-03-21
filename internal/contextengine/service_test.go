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

func assertMissingBlockKind(t *testing.T, blocks []Block, want string) {
	t.Helper()

	for _, block := range blocks {
		if block.Kind == want {
			t.Fatalf("unexpected block kind %q found in %#v", want, blocks)
		}
	}
}
