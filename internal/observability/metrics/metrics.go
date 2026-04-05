// Package metrics provides OpenTelemetry metric instruments for OpsPilot agent runtime.
//
// Instruments cover the six core subsystems: planner, retrieval, tool execution,
// LLM provider, critic, and dynamic replanning. Each instrument is initialized once
// at startup and safe for concurrent use across request goroutines.
//
// Call InitStdout (or configure your own MeterProvider) BEFORE NewInstruments,
// because OTel meters resolve the provider at creation time.
package metrics

import (
	"context"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const meterName = "opspilot-go"

// agentLatencyBuckets defines histogram boundaries tuned for agent subsystem
// latencies that typically range 10ms–10s.
var agentLatencyBuckets = []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

// Instruments holds all pre-registered metric instruments for the agent runtime.
// Create once via NewInstruments and pass to subsystems that need to record metrics.
// A nil *Instruments is safe — all Record* methods are no-ops on nil receiver.
type Instruments struct {
	// Planner
	PlannerLatency     metric.Float64Histogram // seconds; labels: intent, source, tenant_id
	PlannerIntentCount metric.Int64Counter     // labels: intent, source

	// Retrieval (covers full pipeline: HyDE + search + rerank + CRAG + reorder)
	RetrievalPipelineLatency metric.Float64Histogram // seconds
	RetrievalEvidenceSize    metric.Int64Histogram   // evidence block count per query

	// Tool execution
	ToolExecutionLatency metric.Float64Histogram // seconds; labels: tool_name
	ToolExecutionCount   metric.Int64Counter     // labels: tool_name, tool_status

	// LLM provider
	LLMLatency   metric.Float64Histogram // seconds; labels: model
	LLMTokensIn  metric.Int64Counter     // labels: model
	LLMTokensOut metric.Int64Counter     // labels: model

	// Critic
	CriticLatency      metric.Float64Histogram // seconds; labels: source
	CriticVerdictCount metric.Int64Counter     // labels: verdict, source

	// Replanning
	ReplanCount metric.Int64Counter // labels: reason
}

// NewInstruments registers all agent runtime metric instruments on the global meter.
// Must be called after the global MeterProvider is configured (e.g. after InitStdout).
func NewInstruments() *Instruments {
	meter := otel.Meter(meterName)
	m := &Instruments{}

	var err error

	m.PlannerLatency, err = meter.Float64Histogram("opspilot.planner.latency",
		metric.WithDescription("Planner execution latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(agentLatencyBuckets...))
	logRegistrationError("opspilot.planner.latency", err)

	m.PlannerIntentCount, err = meter.Int64Counter("opspilot.planner.intent.count",
		metric.WithDescription("Plan count by classified intent"))
	logRegistrationError("opspilot.planner.intent.count", err)

	m.RetrievalPipelineLatency, err = meter.Float64Histogram("opspilot.retrieval.pipeline.latency",
		metric.WithDescription("Full retrieval pipeline latency (HyDE + search + rerank + CRAG + reorder) in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(agentLatencyBuckets...))
	logRegistrationError("opspilot.retrieval.pipeline.latency", err)

	m.RetrievalEvidenceSize, err = meter.Int64Histogram("opspilot.retrieval.evidence_size",
		metric.WithDescription("Number of evidence blocks returned per retrieval query"))
	logRegistrationError("opspilot.retrieval.evidence_size", err)

	m.ToolExecutionLatency, err = meter.Float64Histogram("opspilot.tool.latency",
		metric.WithDescription("Tool execution latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(agentLatencyBuckets...))
	logRegistrationError("opspilot.tool.latency", err)

	m.ToolExecutionCount, err = meter.Int64Counter("opspilot.tool.execution.count",
		metric.WithDescription("Tool execution count by tool name and status"))
	logRegistrationError("opspilot.tool.execution.count", err)

	m.LLMLatency, err = meter.Float64Histogram("opspilot.llm.latency",
		metric.WithDescription("LLM completion latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(agentLatencyBuckets...))
	logRegistrationError("opspilot.llm.latency", err)

	m.LLMTokensIn, err = meter.Int64Counter("opspilot.llm.tokens.input",
		metric.WithDescription("Total LLM input tokens"))
	logRegistrationError("opspilot.llm.tokens.input", err)

	m.LLMTokensOut, err = meter.Int64Counter("opspilot.llm.tokens.output",
		metric.WithDescription("Total LLM output tokens"))
	logRegistrationError("opspilot.llm.tokens.output", err)

	m.CriticLatency, err = meter.Float64Histogram("opspilot.critic.latency",
		metric.WithDescription("Critic review latency in seconds"),
		metric.WithUnit("s"),
		metric.WithExplicitBucketBoundaries(agentLatencyBuckets...))
	logRegistrationError("opspilot.critic.latency", err)

	m.CriticVerdictCount, err = meter.Int64Counter("opspilot.critic.verdict.count",
		metric.WithDescription("Critic verdict count by verdict type"))
	logRegistrationError("opspilot.critic.verdict.count", err)

	m.ReplanCount, err = meter.Int64Counter("opspilot.replan.count",
		metric.WithDescription("Dynamic replan trigger count"))
	logRegistrationError("opspilot.replan.count", err)

	return m
}

func logRegistrationError(name string, err error) {
	if err != nil {
		slog.Warn("failed to register metric instrument",
			slog.String("instrument", name),
			slog.Any("error", err))
	}
}

// Common attribute helpers for recording metrics with consistent labels.
// Source values must come from bounded enums (e.g. PlanSourceLLM, PlanSourceKeyword).
var (
	AttrIntent     = attribute.Key("intent")
	AttrToolName   = attribute.Key("tool_name")
	AttrToolStatus = attribute.Key("tool_status")
	AttrVerdict    = attribute.Key("verdict")
	AttrSource     = attribute.Key("source")
	AttrModel      = attribute.Key("model")
	AttrTenantID   = attribute.Key("tenant_id")
)

// RecordPlannerLatency records the planner execution duration and intent.
func (m *Instruments) RecordPlannerLatency(ctx context.Context, duration time.Duration, intent, source, tenantID string) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(
		AttrIntent.String(intent),
		AttrSource.String(source),
		AttrTenantID.String(tenantID))
	m.PlannerLatency.Record(ctx, duration.Seconds(), attrs)
	m.PlannerIntentCount.Add(ctx, 1, attrs)
}

// RecordRetrievalLatency records the full retrieval pipeline duration and result size.
func (m *Instruments) RecordRetrievalLatency(ctx context.Context, duration time.Duration, evidenceCount int, tenantID string) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(AttrTenantID.String(tenantID))
	m.RetrievalPipelineLatency.Record(ctx, duration.Seconds(), attrs)
	m.RetrievalEvidenceSize.Record(ctx, int64(evidenceCount), attrs)
}

// RecordToolExecution records one tool execution with its outcome.
func (m *Instruments) RecordToolExecution(ctx context.Context, duration time.Duration, toolName, status, tenantID string) {
	if m == nil {
		return
	}
	m.ToolExecutionLatency.Record(ctx, duration.Seconds(),
		metric.WithAttributes(AttrToolName.String(toolName), AttrTenantID.String(tenantID)))
	m.ToolExecutionCount.Add(ctx, 1,
		metric.WithAttributes(AttrToolName.String(toolName), AttrToolStatus.String(status), AttrTenantID.String(tenantID)))
}

// RecordLLMCall records one LLM completion with token counts.
func (m *Instruments) RecordLLMCall(ctx context.Context, duration time.Duration, model string, tokensIn, tokensOut int) {
	if m == nil {
		return
	}
	attrs := metric.WithAttributes(AttrModel.String(model))
	m.LLMLatency.Record(ctx, duration.Seconds(), attrs)
	m.LLMTokensIn.Add(ctx, int64(tokensIn), attrs)
	m.LLMTokensOut.Add(ctx, int64(tokensOut), attrs)
}

// RecordCriticVerdict records one critic review result.
func (m *Instruments) RecordCriticVerdict(ctx context.Context, duration time.Duration, verdict, source string) {
	if m == nil {
		return
	}
	m.CriticLatency.Record(ctx, duration.Seconds(),
		metric.WithAttributes(AttrSource.String(source)))
	m.CriticVerdictCount.Add(ctx, 1,
		metric.WithAttributes(AttrVerdict.String(verdict), AttrSource.String(source)))
}

// RecordReplan records a dynamic replanning event.
func (m *Instruments) RecordReplan(ctx context.Context, reason string) {
	if m == nil {
		return
	}
	m.ReplanCount.Add(ctx, 1, metric.WithAttributes(attribute.String("reason", reason)))
}

// InitStdout initializes a stdout metric exporter for local development.
// Must be called BEFORE NewInstruments — OTel meters resolve the provider at creation time.
// Returns a shutdown function that should be deferred.
func InitStdout() func(context.Context) error {
	exporter, err := stdoutmetric.New()
	if err != nil {
		slog.Error("failed to create stdout metric exporter", slog.Any("error", err))
		return func(context.Context) error { return nil }
	}

	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(30*time.Second))
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	otel.SetMeterProvider(mp)

	return mp.Shutdown
}
