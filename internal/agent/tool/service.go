package tool

import (
	"context"
	"encoding/json"
	"fmt"

	toolregistry "opspilot-go/internal/tools/registry"
)

// Service executes registered tool definitions with deterministic stub behavior.
type Service struct {
	registry *toolregistry.Registry
}

// NewService constructs a tool execution service.
func NewService(registry *toolregistry.Registry) *Service {
	return &Service{registry: registry}
}

// Execute resolves a tool definition and returns a normalized execution result.
func (s *Service) Execute(ctx context.Context, inv ToolInvocation) (ToolResult, error) {
	def, ok := s.registry.Lookup(inv.ToolName)
	if !ok {
		return ToolResult{}, fmt.Errorf("tool %q not found", inv.ToolName)
	}

	result := ToolResult{
		ToolCallID: fmt.Sprintf("toolcall-%s-%s", inv.PlanID, inv.StepID),
		ToolName:   inv.ToolName,
		AuditRef:   fmt.Sprintf("audit-%s-%s", inv.PlanID, inv.StepID),
	}

	if !inv.ApprovalGranted && (inv.RequiresApproval || def.RequiresApproval || def.ActionClass == ActionClassWrite || def.ActionClass == ActionClassAdmin) {
		result.Status = StatusApprovalRequired
		result.OutputSummary = "approval required before tool execution"
		result.ApprovalRef = fmt.Sprintf("approval-%s-%s", inv.PlanID, inv.StepID)
		return result, nil
	}

	payload := any(def.StubResponse)
	summary := "stub tool execution completed"
	if def.Executor != nil {
		executed, err := def.Executor(ctx, inv.Arguments)
		if err != nil {
			return ToolResult{}, fmt.Errorf("execute %s: %w", inv.ToolName, err)
		}
		payload = executed
		summary = "tool execution completed"
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return ToolResult{}, fmt.Errorf("marshal tool response: %w", err)
	}

	result.Status = StatusSucceeded
	result.OutputSummary = summary
	result.StructuredData = data
	return result, nil
}
