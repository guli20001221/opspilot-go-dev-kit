# Critic Skeleton Plan

**Goal:** add a typed critic stage under `internal/agent/critic` and wire it into the synchronous chat application flow after retrieval and tool execution.

**Scope:**
- define `CriticInput` and `CriticVerdict`
- score groundedness, citation coverage, tool consistency, and simple risk
- return deterministic `approve`, `revise`, and `promote_workflow` verdicts
- expose the critic verdict on the internal chat application result

**Out of scope:**
- model-backed judging
- persistence of critic outputs
- public SSE `critic` events
- eval dataset promotion

**Verification:**
- `go test ./internal/agent/critic -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
