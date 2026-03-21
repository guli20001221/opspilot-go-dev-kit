# AGENTS.override.md

Local scope: `internal/retrieval/**`

Priorities here:
- provenance on every retrieved item
- deterministic filtering and ranking where possible
- tenant scoping on every query
- context assembly driven by structured query objects

Extra rules:
- never retrieve against raw transcripts without query normalization
- never return citationless results to upstream agents
- preserve chunk metadata needed for audit and report views
- benchmark index and query changes before claiming improvements
