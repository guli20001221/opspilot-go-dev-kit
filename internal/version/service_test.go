package version

import (
	"context"
	"testing"
	"time"
)

func TestServiceCurrentVersionIsDurable(t *testing.T) {
	svc := NewService()

	got, err := svc.CurrentVersion(context.Background())
	if err != nil {
		t.Fatalf("CurrentVersion() error = %v", err)
	}
	if got.ID != DefaultVersionID {
		t.Fatalf("CurrentVersion().ID = %q, want %q", got.ID, DefaultVersionID)
	}
	if got.RuntimeVersion == "" || got.PromptBundle == "" {
		t.Fatalf("CurrentVersion() = %#v, want populated runtime and prompt versions", got)
	}

	loaded, err := svc.GetVersion(context.Background(), DefaultVersionID)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if loaded.ID != got.ID {
		t.Fatalf("GetVersion().ID = %q, want %q", loaded.ID, got.ID)
	}
}

func TestServiceListVersionsIncludesCurrentVersion(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	custom := Version{
		ID:                  "version-custom-v1",
		RuntimeVersion:      "runtime-custom-v1",
		PromptBundle:        "prompt-custom-v1",
		PlannerVersion:      "planner-custom-v1",
		RetrievalVersion:    "retrieval-custom-v1",
		ToolRegistryVersion: "tool-custom-v1",
		CriticVersion:       "critic-custom-v1",
		WorkflowVersion:     "workflow-custom-v1",
		Notes:               "Custom test version.",
		CreatedAt:           time.Date(2026, time.March, 25, 0, 0, 0, 0, time.UTC),
	}
	if _, err := svc.store.Save(ctx, custom); err != nil {
		t.Fatalf("Save(custom) error = %v", err)
	}

	page, err := svc.ListVersions(ctx, ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if len(page.Versions) != 2 {
		t.Fatalf("len(ListVersions().Versions) = %d, want %d", len(page.Versions), 2)
	}
	if page.Versions[0].ID != custom.ID {
		t.Fatalf("ListVersions().Versions[0].ID = %q, want %q", page.Versions[0].ID, custom.ID)
	}
	if page.Versions[1].ID != DefaultVersionID {
		t.Fatalf("ListVersions().Versions[1].ID = %q, want %q", page.Versions[1].ID, DefaultVersionID)
	}
}

func TestServiceCurrentVersionDoesNotOverwriteExistingDefaultRow(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	customDefault := defaultVersion()
	customDefault.Provider = "openai"
	customDefault.Model = "gpt-test"
	customDefault.Notes = "Existing durable row."
	customDefault.CreatedAt = time.Date(2027, time.January, 15, 10, 0, 0, 0, time.UTC)
	if _, err := svc.store.Save(ctx, customDefault); err != nil {
		t.Fatalf("Save(default) error = %v", err)
	}

	got, err := svc.GetVersion(ctx, DefaultVersionID)
	if err != nil {
		t.Fatalf("GetVersion() error = %v", err)
	}
	if got.Provider != customDefault.Provider || got.Model != customDefault.Model {
		t.Fatalf("GetVersion() = %#v, want existing provider/model preserved", got)
	}
	if !got.CreatedAt.Equal(customDefault.CreatedAt) {
		t.Fatalf("GetVersion().CreatedAt = %v, want %v", got.CreatedAt, customDefault.CreatedAt)
	}

	if _, err := svc.CurrentVersionID(ctx); err != nil {
		t.Fatalf("CurrentVersionID() error = %v", err)
	}
	page, err := svc.ListVersions(ctx, ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("ListVersions() error = %v", err)
	}
	if len(page.Versions) != 1 {
		t.Fatalf("len(ListVersions().Versions) = %d, want %d", len(page.Versions), 1)
	}
	if page.Versions[0].Provider != customDefault.Provider || page.Versions[0].Model != customDefault.Model {
		t.Fatalf("ListVersions().Versions[0] = %#v, want existing provider/model preserved", page.Versions[0])
	}
}
