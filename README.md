# OpsPilot-Go

A production-grade Go multi-agent platform for enterprise knowledge, ticket, and workflow orchestration. Built with cutting-edge RAG techniques, LLM-backed agent planning with dynamic replanning, hierarchical tenant policy governance, and evaluation-driven development.

## Architecture Overview

```
                         ┌─────────────────────────────────────────┐
                         │              HTTP API (SSE)             │
                         │   sessions · documents · cases · evals  │
                         └────────────────┬────────────────────────┘
                                          │
                    ┌─────────────────────┼─────────────────────┐
                    │                     │                     │
              ┌─────▼─────┐       ┌──────▼──────┐      ┌──────▼──────┐
              │  Context   │       │   Planner   │      │  PolicyLoader│
              │  Engine    │       │  (LLM/KW)   │      │ (org→tenant →│
              │            │       │             │      │    user)    │
              └─────┬──────┘       └──────┬──────┘      └─────────────┘
                    │                     │
         ┌──────────┼──────────┬──────────┼──────────┐
         │          │          │          │          │
    ┌────▼───┐ ┌───▼────┐ ┌───▼───┐ ┌───▼────┐ ┌───▼────┐
    │Retrieval│ │  Tool  │ │Critic │ │Workflow│ │ Replan │
    │Pipeline │ │ Agent  │ │(LLM)  │ │Promote │ │(LLM)   │
    └────┬────┘ └───┬────┘ └───────┘ └────────┘ └────────┘
         │          │
    ┌────▼──────────▼────┐
    │   PostgreSQL +     │
    │   pgvector         │
    └────────────────────┘
```

## Key Features

### Agent Runtime
- **LLM Planner** with keyword fallback and structured JSON plan output
- **Dynamic Replanning**: tool failure → LLM replan → re-execute (max 1 replan, 4 completeness paths: tools, critic, retrieval, audit trail)
- **LLM Critic** with rule-based fallback (groundedness, citation, risk scoring)
- **MCP Adapter Factory**: dynamic tool discovery from MCP-compliant servers via JSON-RPC 2.0
- **Write-tool safety boundary**: write tools require structured planner arguments, heuristic fallback restricted to read-only
- **Prompt versioning** (planner-v2) with eval prompt files

### RAG Pipeline (10 techniques)
```
Query → HyDE → Hybrid Search (Dense + BM25 + RRF) → Parent-Child Expansion
     → LLM Reranking → Contextual Compression → CRAG → Lost-in-the-Middle → LLM
```

| Technique | Description |
|-----------|-------------|
| Semantic Chunking | Embedding cosine similarity for chunk boundaries |
| Contextual Retrieval | LLM-generated context prefix per chunk (Anthropic 2024) |
| HyDE | Hypothetical document embedding for better semantic matching |
| Hybrid Search | Dense vector + BM25 full-text + Reciprocal Rank Fusion |
| Parent-Child Chunks | Fine-grained child retrieval expanded to parent context |
| LLM Cross-Encoder Reranking | Bounded concurrency, per-call timeout |
| Contextual Compression | Extract only query-relevant content from passages (LangChain pattern) |
| CRAG | Corrective RAG — classify passages as relevant/ambiguous/irrelevant |
| Lost-in-the-Middle | Place strongest evidence at start/end of context |
| RAGAS Metrics | Faithfulness, answer relevancy, context precision |

### Context Engine (4 best-practice patterns)
| Pattern | Reference | Implementation |
|---------|-----------|----------------|
| Stage-aware Assembly | Microsoft AutoGen | Planner/Retrieval/Critic get different block subsets with independent budgets |
| ConversationSummaryBuffer | LangChain | LLM compresses older turns, keeps recent N at full fidelity, rolling re-compression |
| Per-snippet Eviction | MemGPT | Individual evidence blocks with stable priority-based eviction |
| Dynamic Importance Scoring | MemGPT | Embedding cosine similarity + keyword overlap scoring adjusts block priority by query relevance |

### Hierarchical Tenant Policy (org → tenant → user)
- **DB-backed** JSONB policy storage with unique scope index
- **Merge rules**: AllowToolUse (child wins), ForbiddenTools (additive union), RequireApprovalForWrite (escalation-only)
- **Cached loader** with TTL, fail-stale on DB error, fail-closed when no cache
- **AllowToolUseExplicit** flag prevents partial policy rows from accidentally disabling tools

### Eval System (full regression loop)
```
Datasets → Runs → Reports → Regression Detection → Auto-Promote → Case Management → Operator Triage
```
- Threshold-based regression classification (regression/improvement/stable)
- Configurable score drop and new-bad-case thresholds
- Auto-promote new bad cases to case management on regression

### Observability
- **OTel Tracing**: spans for all critical paths (planner, retrieval, tools, critic, compression)
- **OTel Metrics**: 12 instruments across 6 subsystems (planner latency/intent, retrieval pipeline, tool execution, LLM tokens, critic verdicts, replan count)
- Explicit histogram buckets tuned for agent latencies (10ms–10s)
- Tenant-scoped metrics for multi-tenant observability

## Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.24 |
| Database | PostgreSQL + pgvector (4096-dim vectors) |
| Workflow | Temporal (optional, for async tasks) |
| LLM | OpenAI-compatible API (Doubao/豆包) |
| Observability | OpenTelemetry (tracing + metrics) |
| Streaming | SSE (Server-Sent Events) |
| Tools | MCP JSON-RPC 2.0 + HTTP adapters |

## Quick Start

```bash
# Prerequisites: Go 1.24+, Docker, Make

# Start local dev stack (Postgres, Redis, Temporal, API, Worker)
make dev-up

# Run all tests
make test

# Build all binaries
make build

# Full check (fmt + test + build)
make check
```

### Configuration

All config via `OPSPILOT_*` environment variables. Key vars:

| Variable | Default | Purpose |
|----------|---------|---------|
| `OPSPILOT_POSTGRES_DSN` | localhost DSN | Primary database |
| `OPSPILOT_LLM_PROVIDER` | `placeholder` | `placeholder` or `openai` |
| `OPSPILOT_LLM_BASE_URL` | — | OpenAI-compatible endpoint |
| `OPSPILOT_LLM_API_KEY` | — | API key |
| `OPSPILOT_TOOL_POLICY_ALLOW` | `true` | Global tool-use toggle |
| `OPSPILOT_TEMPORAL_ENABLED` | `false` | Enable Temporal workflows |

## Project Structure

```
cmd/
  api/          HTTP server (REST + SSE)
  worker/       Background task runner
  ticketapi/    Dev-only fake ticket system
internal/
  agent/
    planner/    LLM planner + keyword fallback + replanning + policy
    critic/     LLM critic + rule fallback
    tool/       Tool execution with approval gates
  contextengine/  Stage-aware assembly, summarizer, dynamic scoring
  retrieval/    HyDE, CRAG, reranker, compressor, RAGAS, embedder
  ingestion/    Semantic chunking, contextual prefixing, indexing
  eval/         Datasets, runs, reports, regression detection
  case/         Operator case management
  workflow/     Temporal workflows, task promotion
  storage/postgres/  pgx stores, pgvector, policy store
  tools/
    registry/   Typed tool registry with ParameterDef
    mcp/        MCP adapter factory (JSON-RPC 2.0)
    http/       HTTP tool adapters (tickets)
  observability/
    tracing/    OTel tracer + span helpers
    metrics/    12 agent runtime instruments
db/migrations/  30 PostgreSQL migrations
eval/prompts/   Versioned planner/critic prompts
```

## License

MIT License — see [LICENSE](LICENSE) for details.
