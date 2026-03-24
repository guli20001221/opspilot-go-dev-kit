package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"opspilot-go/internal/version"
)

func TestVersionStoreRoundTrip(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "DELETE FROM versions WHERE id <> $1", version.DefaultVersionID); err != nil {
		t.Fatalf("DELETE versions error = %v", err)
	}

	store := NewVersionStore(pool)
	want := version.Version{
		ID:                  "version-postgres-roundtrip",
		RuntimeVersion:      "runtime-v2",
		Provider:            "openai",
		Model:               "gpt-test",
		PromptBundle:        "prompt-v2",
		PlannerVersion:      "planner-v2",
		RetrievalVersion:    "retrieval-v2",
		ToolRegistryVersion: "tools-v2",
		CriticVersion:       "critic-v2",
		WorkflowVersion:     "workflow-v2",
		Notes:               "postgres roundtrip",
		CreatedAt:           time.Unix(1700009000, 0).UTC(),
	}

	if _, err := store.Save(ctx, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Get(ctx, want.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != want.ID {
		t.Fatalf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.ToolRegistryVersion != want.ToolRegistryVersion {
		t.Fatalf("ToolRegistryVersion = %q, want %q", got.ToolRegistryVersion, want.ToolRegistryVersion)
	}
}

func TestVersionStoreListReturnsNewestFirst(t *testing.T) {
	dsn := os.Getenv("OPSPILOT_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("OPSPILOT_TEST_POSTGRES_DSN not set")
	}

	ctx := context.Background()
	pool, err := OpenPool(ctx, dsn)
	if err != nil {
		t.Fatalf("OpenPool() error = %v", err)
	}
	defer pool.Close()

	applyMigration(t, ctx, pool)
	if _, err := pool.Exec(ctx, "DELETE FROM versions WHERE id <> $1", version.DefaultVersionID); err != nil {
		t.Fatalf("DELETE versions error = %v", err)
	}

	store := NewVersionStore(pool)
	for _, item := range []version.Version{
		{
			ID:                  "version-a",
			RuntimeVersion:      "runtime-a",
			PromptBundle:        "prompt-a",
			PlannerVersion:      "planner-a",
			RetrievalVersion:    "retrieval-a",
			ToolRegistryVersion: "tools-a",
			CriticVersion:       "critic-a",
			WorkflowVersion:     "workflow-a",
			CreatedAt:           time.Unix(1800009100, 0).UTC(),
		},
		{
			ID:                  "version-b",
			RuntimeVersion:      "runtime-b",
			PromptBundle:        "prompt-b",
			PlannerVersion:      "planner-b",
			RetrievalVersion:    "retrieval-b",
			ToolRegistryVersion: "tools-b",
			CriticVersion:       "critic-b",
			WorkflowVersion:     "workflow-b",
			CreatedAt:           time.Unix(1800009200, 0).UTC(),
		},
	} {
		if _, err := store.Save(ctx, item); err != nil {
			t.Fatalf("Save(%s) error = %v", item.ID, err)
		}
	}

	page, err := store.List(ctx, version.ListFilter{Limit: 3})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Versions) != 3 {
		t.Fatalf("len(Versions) = %d, want %d", len(page.Versions), 3)
	}
	if page.Versions[0].ID != "version-b" {
		t.Fatalf("Versions[0].ID = %q, want %q", page.Versions[0].ID, "version-b")
	}
	if page.Versions[1].ID != "version-a" {
		t.Fatalf("Versions[1].ID = %q, want %q", page.Versions[1].ID, "version-a")
	}
	if page.Versions[2].ID != version.DefaultVersionID {
		t.Fatalf("Versions[2].ID = %q, want %q", page.Versions[2].ID, version.DefaultVersionID)
	}
	if page.HasMore {
		t.Fatalf("HasMore = true, want false")
	}
}
