---
name: mcp-adapter-factory
description: Wrap external systems as MCP or HTTP tools with typed contracts, safety controls, and audit-friendly behavior.
---

# mcp-adapter-factory

## Goal
Integrate external systems cleanly without leaking vendor-specific chaos into the agent core.

## Use this skill when
- integrating GitHub, ticketing, files, internal APIs, or search systems
- designing new tool wrappers
- deciding between MCP and direct HTTP adapters
- normalizing tool IO and approval behavior

## Inputs to collect first
- external system capability and risk profile
- read-only versus side-effecting classification
- auth and secret handling model
- expected input/output schema
- failure and retry semantics

## Likely files and directories
- `internal/tools/**`
- `internal/tools/mcp/**`
- `internal/tools/http/**`
- `internal/tools/registry/**`
- `docs/runbooks/**` for operational caveats

## Standard workflow
1. Classify the tool as read-only or side-effecting.
2. Define a typed request and response contract.
3. Choose MCP when protocol reuse or remote tool exposure is valuable; choose direct HTTP when the integration is narrower and internal.
4. Normalize external errors into internal categories.
5. Add approval or dry-run controls for write-capable tools.
6. Emit audit records for external actions.
7. Start with deterministic typed adapters when the real external system is not wired yet; validate request payloads instead of returning fixed success blobs.
8. When a real external boundary is added, wire it behind the same typed executor contract and keep a deterministic fallback for local development when feasible.
9. Add tests with fakes or local harnesses instead of live network calls when possible.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- tool contract is typed and documented
- deterministic local adapters validate arguments before any live integration exists
- config-driven adapter selection does not leak vendor behavior into the agent runtime
- read-only versus write-capable behavior is explicit
- errors are normalized
- approvals exist where needed
- audit logging exists for external actions
- tests cover nominal and failure paths

## Guardrails
- no raw external SDK structs beyond adapter boundaries
- no write-capable tool without approval or safeguard path
- no secrets embedded in code or tests
- no live network dependency in routine test runs
