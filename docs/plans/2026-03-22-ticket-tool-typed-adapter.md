# 2026-03-22 Ticket Tool Typed Adapter

## Goal
Replace fixed stub-only ticket tools with deterministic typed adapters so approved tool workflows execute argument-dependent behavior and reject invalid payloads.

## Scope
- add executor hooks to the internal tool registry
- implement deterministic typed ticket search and ticket comment adapters
- keep approval gating in `internal/agent/tool`
- preserve legacy fallback behavior for approval tasks without stored payload

## Non-goals
- live ticketing system calls
- new public API fields
- changing workflow task promotion semantics

## Validation
- unit tests proving approved ticket comments echo typed arguments
- unit tests proving invalid approval payloads are rejected
- full `go test ./...`
- repo `check` command
