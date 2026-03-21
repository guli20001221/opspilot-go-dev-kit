package tool

import (
	"context"
	"encoding/json"
	"testing"

	toolregistry "opspilot-go/internal/tools/registry"
)

func TestServiceExecuteReadOnlyTool(t *testing.T) {
	registry := toolregistry.New()
	registry.Register(toolregistry.Definition{
		Name:             "ticket_search",
		ActionClass:      ActionClassRead,
		ReadOnly:         true,
		RequiresApproval: false,
		StubResponse: map[string]any{
			"matches": []map[string]string{
				{"ticket_id": "INC-100", "summary": "database incident"},
			},
		},
	})

	svc := NewService(registry)
	got, err := svc.Execute(context.Background(), ToolInvocation{
		RequestID:        "req-1",
		TraceID:          "trace-1",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		PlanID:           "plan-1",
		StepID:           "step-1",
		ToolName:         "ticket_search",
		ActionClass:      ActionClassRead,
		RequiresApproval: false,
		Arguments:        json.RawMessage(`{"query":"database incident"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, StatusSucceeded)
	}
	if got.ToolName != "ticket_search" {
		t.Fatalf("ToolName = %q, want %q", got.ToolName, "ticket_search")
	}
	if got.ToolCallID == "" || got.AuditRef == "" {
		t.Fatalf("missing audit ids in %#v", got)
	}
	if len(got.StructuredData) == 0 {
		t.Fatal("StructuredData is empty")
	}
}

func TestServiceExecuteWriteToolRequiresApproval(t *testing.T) {
	registry := toolregistry.New()
	registry.Register(toolregistry.Definition{
		Name:             "ticket_comment_create",
		ActionClass:      ActionClassWrite,
		ReadOnly:         false,
		RequiresApproval: true,
	})

	svc := NewService(registry)
	got, err := svc.Execute(context.Background(), ToolInvocation{
		RequestID:        "req-2",
		TraceID:          "trace-2",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		PlanID:           "plan-2",
		StepID:           "step-2",
		ToolName:         "ticket_comment_create",
		ActionClass:      ActionClassWrite,
		RequiresApproval: true,
		Arguments:        json.RawMessage(`{"ticket_id":"INC-100","comment":"please review"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status != StatusApprovalRequired {
		t.Fatalf("Status = %q, want %q", got.Status, StatusApprovalRequired)
	}
	if got.ApprovalRef == "" {
		t.Fatal("ApprovalRef is empty")
	}
	if len(got.StructuredData) != 0 {
		t.Fatalf("StructuredData = %s, want empty", string(got.StructuredData))
	}
}
