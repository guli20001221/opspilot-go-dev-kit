---
name: retrieval-ingest-provenance
description: Implement document ingestion, chunking, embeddings, retrieval, re-ranking, provenance, and citation-aware context assembly.
---

# retrieval-ingest-provenance

## Goal
Build a retrieval stack that is explainable, auditable, and useful for downstream planner and critic agents.

## Use this skill when
- adding ingestion pipelines
- changing chunking or embedding strategy
- implementing retrieval, re-ranking, or citation assembly
- wiring context blocks from retrieved content

## Inputs to collect first
- source document types and metadata
- chunking constraints
- embedding model strategy
- retrieval filters such as tenant, tag, time, or version
- citation and provenance requirements for final answers

## Likely files and directories
- `internal/retrieval/**`
- `internal/contextengine/**`
- `internal/storage/**`
- `db/migrations/**`
- `eval/datasets/**` if retrieval changes need regression cases

## Standard workflow
1. Normalize source documents and preserve source-level metadata.
2. Chunk content in a way that keeps meaning and reconstructability.
3. Store chunk metadata with source ids, versions, and permissions scope.
4. Build retrieval from a structured query object, not a raw transcript dump.
5. Apply filtering before or alongside ranking as appropriate.
6. Re-rank results deterministically where possible.
7. Assemble context blocks with provenance and token budgeting.
8. Ensure downstream answers can cite returned evidence precisely.
9. Add regression cases for broken citations or poor recall patterns.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- ingestion preserves source metadata
- retrieval is query-object driven
- provenance exists on every retrieved item
- context assembly includes rationale and budgeting
- citation generation is testable
- evaluation cases exist for key retrieval behaviors

## Guardrails
- never dump full chat history directly into retrieval
- never return evidence without provenance
- never lose tenant or permission scope during retrieval
- do not claim quality improvements without benchmark or eval evidence
