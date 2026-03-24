---
name: otel-langfuse-observability
description: Instrument traces, metrics, and logs for LLM calls, tool calls, retrieval, and workflow steps; wire them to operator-friendly observability.
---

# otel-langfuse-observability

## Goal
Make every important runtime step inspectable, measurable, and debuggable without leaking sensitive data.

## Use this skill when
- adding or changing tracing
- instrumenting LLM calls or tool execution
- defining metrics and dashboards
- improving debugging for workflow or retrieval problems
- integrating LLM observability backends

## Inputs to collect first
- runtime steps to instrument
- required correlation ids
- metric consumers and alerting expectations
- sensitive data that must be redacted
- operator drill-down requirements

## Likely files and directories
- `internal/observability/**`
- `internal/model/**`
- `internal/tools/**`
- `internal/retrieval/**`
- `internal/workflow/**`
- `docs/runbooks/**`

## Standard workflow
1. Define a span taxonomy before scattering instrumentation.
2. Thread request, trace, user, and tenant identifiers consistently.
3. Instrument LLM calls, tool calls, retrieval, and workflow transitions.
4. Emit metrics for latency, errors, token use, and success rates.
5. Redact or hash sensitive fields before emitting logs or traces.
6. Ensure report and admin views can deep-link into trace contexts where possible.
7. Update runbooks with how to debug the new path.
8. When a full trace explorer is not available yet, prefer a narrow read-only trace drill-down contract that resolves lineage, request/session ids, audit refs, warnings, and Temporal pointers from durable runtime state.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- critical runtime steps are traced
- key metrics exist and are named consistently
- logs are structured and correlated
- sensitive fields are redacted
- runbooks explain how to inspect failures
- operator drill-down paths are clear

## Guardrails
- do not log secrets or raw sensitive payloads
- avoid uncontrolled high-cardinality metrics
- no external call without trace coverage on important paths
- do not claim observability is complete if workflow or retrieval spans are missing
