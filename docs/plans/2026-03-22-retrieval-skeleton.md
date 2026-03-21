# Retrieval Skeleton Plan

**Goal:** add a typed retrieval package with provenance-bearing evidence results and wire it into the synchronous chat application flow when the planner requires retrieval.

**Scope:**
- define `RetrievalRequest`, `RetrievalResult`, and `EvidenceBlock`
- enforce structured-query-driven retrieval rather than transcript dumping
- preserve tenant scope and provenance fields on every evidence item
- implement deterministic in-memory matching as a stub before storage-backed retrieval lands
- expose retrieval output on the internal chat application result

**Out of scope:**
- ingestion
- embeddings
- pgvector
- re-ranking models
- public SSE `retrieval` and `citation` events

**Verification:**
- `go test ./internal/retrieval -count=1`
- `go test ./internal/app/chat -count=1`
- `go test ./internal/app/httpapi -count=1`
- `go test ./...`
