# 2026-03-22 Ticket HTTP Adapter Wiring

## Goal
Allow the default ticket tools to cross a real HTTP boundary when configured, while preserving deterministic local adapters and the existing public API.

## Scope
- add optional ticket API configuration
- add an HTTP-backed ticket adapter using the standard library client
- make API and worker share the same config-driven tool registry wiring
- preserve deterministic fallback when no ticket API base URL is configured

## Non-goals
- new public REST fields
- live external calls in routine tests
- changing approval workflow semantics

## Validation
- unit tests for the HTTP ticket adapter with `httptest`
- unit tests proving configured registry paths no longer use local fallback
- targeted chat/workflow regression tests
- full `go test ./...`
- repo `check` command
