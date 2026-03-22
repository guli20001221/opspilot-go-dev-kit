package tool

import (
	"context"
	"encoding/json"
	"strings"
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
		StubResponse: map[string]any{
			"ticket_id": "INC-100",
			"status":    "comment_created",
		},
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

func TestServiceExecuteWriteToolAfterApproval(t *testing.T) {
	registry := toolregistry.New()
	registry.Register(toolregistry.Definition{
		Name:             "ticket_comment_create",
		ActionClass:      ActionClassWrite,
		ReadOnly:         false,
		RequiresApproval: true,
		StubResponse: map[string]any{
			"ticket_id": "INC-100",
			"status":    "comment_created",
		},
	})

	svc := NewService(registry)
	got, err := svc.Execute(context.Background(), ToolInvocation{
		RequestID:        "req-3",
		TraceID:          "trace-3",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskID:           "task-1",
		PlanID:           "plan-3",
		StepID:           "step-3",
		ToolName:         "ticket_comment_create",
		ActionClass:      ActionClassWrite,
		RequiresApproval: true,
		ApprovalGranted:  true,
		Arguments:        json.RawMessage(`{"ticket_id":"INC-100","comment":"approved"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if got.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, StatusSucceeded)
	}
	if got.ApprovalRef != "" {
		t.Fatalf("ApprovalRef = %q, want empty", got.ApprovalRef)
	}
	if len(got.StructuredData) == 0 {
		t.Fatal("StructuredData is empty")
	}
}

func TestServiceExecuteApprovedToolUsesTypedDefaultAdapter(t *testing.T) {
	svc := NewService(toolregistry.NewDefaultRegistry())

	got, err := svc.Execute(context.Background(), ToolInvocation{
		RequestID:        "req-4",
		TraceID:          "trace-4",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskID:           "task-typed-1",
		PlanID:           "plan-4",
		StepID:           "step-4",
		ToolName:         "ticket_comment_create",
		ActionClass:      ActionClassWrite,
		RequiresApproval: true,
		ApprovalGranted:  true,
		Arguments:        json.RawMessage(`{"ticket_id":"INC-200","comment":"approved typed comment"}`),
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if got.Status != StatusSucceeded {
		t.Fatalf("Status = %q, want %q", got.Status, StatusSucceeded)
	}

	var payload struct {
		TicketID string `json:"ticket_id"`
		Status   string `json:"status"`
		Comment  string `json:"comment"`
	}
	if err := json.Unmarshal(got.StructuredData, &payload); err != nil {
		t.Fatalf("json.Unmarshal(StructuredData) error = %v", err)
	}
	if payload.TicketID != "INC-200" {
		t.Fatalf("TicketID = %q, want %q", payload.TicketID, "INC-200")
	}
	if payload.Comment != "approved typed comment" {
		t.Fatalf("Comment = %q, want %q", payload.Comment, "approved typed comment")
	}
	if payload.Status != "comment_created" {
		t.Fatalf("Status = %q, want %q", payload.Status, "comment_created")
	}
}

func TestServiceExecuteApprovedToolRejectsInvalidArguments(t *testing.T) {
	svc := NewService(toolregistry.NewDefaultRegistry())

	_, err := svc.Execute(context.Background(), ToolInvocation{
		RequestID:        "req-5",
		TraceID:          "trace-5",
		TenantID:         "tenant-1",
		SessionID:        "session-1",
		TaskID:           "task-typed-2",
		PlanID:           "plan-5",
		StepID:           "step-5",
		ToolName:         "ticket_comment_create",
		ActionClass:      ActionClassWrite,
		RequiresApproval: true,
		ApprovalGranted:  true,
		Arguments:        json.RawMessage(`{"comment":"missing ticket id"}`),
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "ticket_id") {
		t.Fatalf("Execute() error = %v, want message containing %q", err, "ticket_id")
	}
}

func TestServiceExecuteReadOnlyToolUsesInjectedHTTPRegistry(t *testing.T) {
	registry := toolregistry.NewDefaultRegistryWithOptions(toolregistry.Options{
		TicketAPIBaseURL: "http://127.0.0.1:1",
	})

	svc := NewService(registry)
	_, err := svc.Execute(context.Background(), ToolInvocation{
		RequestID:   "req-6",
		TraceID:     "trace-6",
		TenantID:    "tenant-1",
		SessionID:   "session-1",
		PlanID:      "plan-6",
		StepID:      "step-6",
		ToolName:    "ticket_search",
		ActionClass: ActionClassRead,
		Arguments:   json.RawMessage(`{"query":"db issue"}`),
	})
	if err == nil {
		t.Fatal("Execute() error = nil, want non-nil from configured HTTP adapter")
	}
	if !strings.Contains(err.Error(), "ticket_search") {
		t.Fatalf("Execute() error = %v, want message containing %q", err, "ticket_search")
	}
}
