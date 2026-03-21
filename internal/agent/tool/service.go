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
func (s *Service) Execute(_ context.Context, inv ToolInvocation) (ToolResult, error) {
	def, ok := s.registry.Lookup(inv.ToolName)
	if !ok {
		return ToolResult{}, fmt.Errorf("tool %q not found", inv.ToolName)
	}

	result := ToolResult{
		ToolCallID: fmt.Sprintf("toolcall-%s-%s", inv.PlanID, inv.StepID),
		ToolName:   inv.ToolName,
		AuditRef:   fmt.Sprintf("audit-%s-%s", inv.PlanID, inv.StepID),
	}

	if inv.RequiresApproval || def.RequiresApproval || def.ActionClass == ActionClassWrite || def.ActionClass == ActionClassAdmin {
		result.Status = StatusApprovalRequired
		result.OutputSummary = "approval required before tool execution"
		result.ApprovalRef = fmt.Sprintf("approval-%s-%s", inv.PlanID, inv.StepID)
		return result, nil
	}

	data, err := json.Marshal(def.StubResponse)
	if err != nil {
		return ToolResult{}, fmt.Errorf("marshal stub response: %w", err)
	}

	result.Status = StatusSucceeded
	result.OutputSummary = "stub tool execution completed"
	result.StructuredData = data
	return result, nil
}
