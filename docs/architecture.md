# Architecture

## System Overview

OpsPilot-Go is a modular monolith with two entrypoints (`cmd/api`, `cmd/worker`) that share domain packages under `internal/`. All services use constructor injection — no reflection, no magic registries, no global mutable state.

## Request Flow

```
HTTP Request (SSE)
  │
  ├─ PolicyLoader.LoadPolicy(org, tenant, user) → merged TenantPolicy
  │
  ├─ Context Engine
  │    ├─ maybeSummarize (ConversationSummaryBuffer)
  │    ├─ candidateBlocks (6 block kinds)
  │    ├─ ImportanceScorer.ScoreBlocks (dynamic priority)
  │    └─ filterByKinds + applyBudget (per-stage)
  │
  ├─ Planner.Plan (LLM structured JSON → keyword fallback)
  │    ├─ validateLLMPlan (structural)
  │    ├─ validatePlanPolicy (tenant + registry invariants)
  │    └─ workflow-shape invariant
  │
  ├─ Retrieval Pipeline
  │    ├─ HyDE → Hybrid Search (Dense + BM25 + RRF)
  │    ├─ Parent-Child Expansion
  │    ├─ LLM Reranking
  │    ├─ Contextual Compression
  │    ├─ CRAG Filtering
  │    └─ Lost-in-the-Middle Reordering
  │
  ├─ Tool Execution (with write-tool safety boundary)
  │    ├─ executeToolSteps
  │    ├─ on failure → Replan (LLM) → re-execute revised plan
  │    └─ post-replan retrieval if revised plan requires it
  │
  ├─ LLM Completion (streaming via SSE when available)
  │
  ├─ Critic.Review (LLM → rule fallback)
  │
  └─ Workflow Promotion (if required)
```

## Key Design Decisions

### 1. Dynamic Replanning (Plan-and-Execute with Feedback Loop)

**Problem**: Static plans fail when tool execution produces unexpected results.

**Decision**: After tool failure, the planner produces a revised plan that accounts for what already happened. Four completeness paths ensure the revised plan flows through the entire downstream pipeline:

1. **Tool re-execution**: revised plan's tool steps execute
2. **Critic + Workflow**: use `activePlan` (not stale original)
3. **Post-replan retrieval**: if revised plan requires retrieval, run full pipeline
4. **Audit trail**: failed tool attempt preserved in ToolResults and SSE events

**Tradeoff**: Max 1 replan attempt prevents infinite loops but limits recovery depth. Keyword plans cannot replan (requires LLM reasoning).

### 2. Hierarchical Policy (org → tenant → user)

**Problem**: Different tenants need different tool access, approval requirements, and step limits.

**Decision**: Three-level policy inheritance stored as JSONB in PostgreSQL. Merge rules designed for security:

- `ForbiddenTools`: additive union (can never be un-forbidden by child)
- `RequireApprovalForWrite`: escalation-only (true at any level = always true)
- `AllowToolUse`: child overrides parent (only when explicitly set via `AllowToolUseExplicit`)
- `AllowedTools`: child replaces parent list (narrowing)

**Tradeoff**: `AllowToolUseExplicit` flag adds complexity but prevents partial policy rows from accidentally disabling tools.

### 3. Write-Tool Argument Safety

**Problem**: Heuristic argument construction dumps raw user text into tool parameters.

**Decision**: Write (side-effecting) tools require structured `ToolArguments` from the planner. Heuristic fallback restricted to read-only tools. Enforced in both primary and replan execution paths.

**Tradeoff**: Keyword planner cannot produce write-tool arguments, so write tools via keyword path are now rejected. This is acceptable — write tools should only execute with LLM-backed structured planning.

### 4. Context Engine: Layered, Not Raw Transcript

**Problem**: Naively appending full conversation history wastes tokens and includes irrelevant context.

**Decision**: Four best-practice patterns compose into a single pipeline:
- **ConversationSummaryBuffer**: older turns compressed by LLM, rolling re-compression prevents unbounded growth
- **Stage-aware filtering**: planner sees different blocks than critic
- **Per-snippet evidence blocks**: individual evidence items with position-based priority for fine-grained eviction
- **Dynamic importance scoring**: keyword/embedding similarity adjusts priority before budget eviction

### 5. RAG: Defense in Depth

**Problem**: No single retrieval technique is sufficient for production quality.

**Decision**: 10 techniques chained in a specific order where each stage improves the next:
1. HyDE improves semantic matching for the dense search
2. Hybrid search (dense + BM25 + RRF) combines semantic and keyword precision
3. Parent-child expansion provides richer context for reranking
4. Reranking orders by LLM-judged relevance
5. Contextual compression extracts only query-relevant content (feeds cleaner input to CRAG)
6. CRAG makes binary keep/discard decisions on compressed passages
7. Lost-in-the-middle places strongest evidence at optimal context positions

### 6. Eval: Closed-Loop Regression Detection

**Problem**: Prompt/routing changes can silently degrade quality.

**Decision**: Full regression loop from datasets to operator triage:
- `DetectRegression()` compares baseline vs candidate with configurable thresholds
- New bad cases automatically identified via set difference
- `PromoteRegressionCases()` auto-creates follow-up cases
- API supports `auto_promote=true` (POST only for HTTP safety)

## Package Dependencies

```
cmd/api ──► httpapi ──► chat ──► planner
                    │        ├──► critic
                    │        ├──► tool ──► registry
                    │        │            └──► mcp
                    │        ├──► retrieval (HyDE, CRAG, rerank, compress)
                    │        ├──► contextengine
                    │        └──► workflow
                    ├──► eval (reports, regression, cases)
                    └──► storage/postgres
```

No circular dependencies. Domain packages do not import HTTP handlers or CLI entrypoints.
