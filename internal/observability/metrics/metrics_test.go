package metrics

import (
	"context"
	"testing"
	"time"
)

func TestNewInstrumentsReturnsPopulatedStruct(t *testing.T) {
	m := NewInstruments()
	if m == nil {
		t.Fatal("NewInstruments() returned nil")
	}
	if m.PlannerLatency == nil {
		t.Fatal("PlannerLatency is nil")
	}
	if m.PlannerIntentCount == nil {
		t.Fatal("PlannerIntentCount is nil")
	}
	if m.RetrievalPipelineLatency == nil {
		t.Fatal("RetrievalPipelineLatency is nil")
	}
	if m.ToolExecutionLatency == nil {
		t.Fatal("ToolExecutionLatency is nil")
	}
	if m.ToolExecutionCount == nil {
		t.Fatal("ToolExecutionCount is nil")
	}
	if m.LLMLatency == nil {
		t.Fatal("LLMLatency is nil")
	}
	if m.LLMTokensIn == nil {
		t.Fatal("LLMTokensIn is nil")
	}
	if m.LLMTokensOut == nil {
		t.Fatal("LLMTokensOut is nil")
	}
	if m.CriticLatency == nil {
		t.Fatal("CriticLatency is nil")
	}
	if m.CriticVerdictCount == nil {
		t.Fatal("CriticVerdictCount is nil")
	}
	if m.ReplanCount == nil {
		t.Fatal("ReplanCount is nil")
	}
}

func TestNilInstrumentsRecordMethodsDoNotPanic(t *testing.T) {
	var m *Instruments
	ctx := context.Background()
	d := 100 * time.Millisecond

	// All Record* methods must be safe on nil receiver
	m.RecordPlannerLatency(ctx, d, "knowledge_qa", "llm", "tenant-1")
	m.RecordRetrievalLatency(ctx, d, 5, "tenant-1")
	m.RecordToolExecution(ctx, d, "ticket_search", "succeeded", "tenant-1")
	m.RecordLLMCall(ctx, d, "test-model", 100, 50)
	m.RecordCriticVerdict(ctx, d, "approve", "llm")
	m.RecordReplan(ctx, "tool_failure")
}

func TestRecordMethodsDoNotPanicWithInitializedInstruments(t *testing.T) {
	m := NewInstruments()
	ctx := context.Background()
	d := 50 * time.Millisecond

	m.RecordPlannerLatency(ctx, d, "incident_assist", "keyword", "tenant-2")
	m.RecordRetrievalLatency(ctx, d, 3, "tenant-2")
	m.RecordToolExecution(ctx, d, "ticket_comment_create", "approval_required", "tenant-2")
	m.RecordLLMCall(ctx, d, "doubao-seed", 200, 100)
	m.RecordCriticVerdict(ctx, d, "revise", "rule")
	m.RecordReplan(ctx, "tool_failure")
}
