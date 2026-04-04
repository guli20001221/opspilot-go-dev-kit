// Package tracing provides OpenTelemetry initialization and span helpers for OpsPilot.
package tracing

import (
	"context"
	"log/slog"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "opspilot-go"

// Tracer returns the global OpsPilot tracer.
func Tracer() trace.Tracer {
	return otel.Tracer(tracerName)
}

// InitStdout initializes a stdout trace exporter for local development.
// Returns a shutdown function that should be deferred.
func InitStdout() func(context.Context) error {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		slog.Error("failed to create stdout trace exporter", slog.Any("error", err))
		return func(context.Context) error { return nil }
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),                 // Syncer for stdout (immediate output); use WithBatcher for OTLP
		sdktrace.WithSampler(sdktrace.AlwaysSample()), // TODO: use configurable sampler for production
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown
}

// StartSpan starts a new child span with the given operation name.
func StartSpan(ctx context.Context, operation string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	return Tracer().Start(ctx, operation, trace.WithAttributes(attrs...))
}

// Common attribute keys for OpsPilot spans.
var (
	AttrRequestID = attribute.Key("opspilot.request_id")
	AttrTenantID  = attribute.Key("opspilot.tenant_id")
	AttrSessionID = attribute.Key("opspilot.session_id")
	AttrPlanID    = attribute.Key("opspilot.plan_id")
	AttrToolName  = attribute.Key("opspilot.tool_name")
	AttrModel     = attribute.Key("opspilot.model")
	AttrProvider  = attribute.Key("opspilot.provider")
	AttrTokensIn  = attribute.Key("opspilot.tokens.input")
	AttrTokensOut = attribute.Key("opspilot.tokens.output")
	AttrIntent    = attribute.Key("opspilot.intent")
	AttrVerdict   = attribute.Key("opspilot.verdict")
	AttrSource    = attribute.Key("opspilot.source")
)

// RecordError marks the span as errored and records the error details.
func RecordError(span trace.Span, err error) {
	if err == nil || span == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}
