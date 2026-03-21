# AGENTS.md

## 0. Canonical instruction sources
This repository is optimized for both Codex and Claude Code.

Repository-wide governance lives here in `AGENTS.md`.
Claude-specific loading starts from `CLAUDE.md`, which imports this file.
Task-specific playbooks live under `.claude/skills/<skill-name>/SKILL.md`.
Path-specific local rules live in `AGENTS.override.md` files close to the code they govern.

Treat this file as the repository-level router and policy layer.
Do not duplicate full skill bodies into AGENTS files.
Use AGENTS files for routing, guardrails, and non-negotiable engineering standards.

## 1. Purpose
This repository builds OpsPilot-Go, a production-grade Golang multi-agent platform for enterprise knowledge, ticket, and workflow orchestration.

Optimize for:
- correctness
- safety
- observability
- evaluation and regression
- maintainability
- small, reviewable diffs

Do not optimize for:
- demo-only UX
- framework-heavy abstractions
- speculative microservices
- hidden prompt logic in handlers
- unreviewed side-effecting automation

## 2. Product boundaries
The system must support:
- conversational and task-execution modes
- Planner, Retrieval, Tool, and Critic agents
- a structured context engine
- synchronous request paths and asynchronous workflow paths
- trace, report, and evaluation capabilities
- human approval for high-risk actions
- multi-tenant isolation, auditability, and explicit operator controls

Non-goals:
- microservices by default
- shipping UI before backend contracts stabilize
- storing raw full history everywhere
- silent prompt or rubric changes without versioning
- direct write access to production systems without approval paths

## 3. Skills registry
Canonical skill implementations live under `.claude/skills/<skill-name>/SKILL.md`.

When a task touches one of these areas, consult the matching skill before changing code:

- repo bootstrap / local dev / repo restructuring -> `repo-bootstrap-go`
- PostgreSQL / migrations / sqlc / pgvector -> `postgres-sqlc-pgvector`
- API contracts / SSE / middleware / error envelopes -> `api-contract-sse`
- ingestion / chunking / embeddings / citations -> `retrieval-ingest-provenance`
- planner / tool orchestration / critic / provider adapters -> `agent-runtime-responses`
- async jobs / approvals / retries / worker execution -> `workflow-temporal-approval`
- traces / metrics / logs / LLM observability -> `otel-langfuse-observability`
- datasets / experiments / judges / regression reports -> `eval-datasets-regression`
- external tools / MCP adapters / HTTP tool wrappers -> `mcp-adapter-factory`
- RBAC / tenancy / audit / secrets / approvals -> `security-tenancy-audit`
- web admin / reports / version comparison views -> `admin-console-reports`
- README / ADR / architecture docs / runbooks -> `docs-adr-runbook`

## 4. Skill invocation policy
Before any non-trivial change:
1. inspect the current code and adjacent modules
2. identify which subsystem is touched
3. consult the matching skill(s)
4. make the smallest coherent change
5. run validations
6. update docs and skill guidance if the workflow changed

If a task spans multiple areas, compose skills in this order:
1. `repo-bootstrap-go` for skeleton or repo hygiene
2. data/model skills
3. runtime or workflow skills
4. observability and eval skills
5. admin/docs skills

## 5. Repository shape
Keep or evolve toward this layout:

```text
.
├── AGENTS.md
├── CLAUDE.md
├── Makefile
├── README.md
├── .env.example
├── .claude/
│   ├── skills/
│   ├── agents/
│   └── hooks/
├── cmd/
│   ├── api/
│   └── worker/
├── config/
├── db/
│   ├── migrations/
│   └── queries/
├── docs/
│   ├── adr/
│   ├── runbooks/
│   └── architecture.md
├── eval/
│   ├── datasets/
│   ├── prompts/
│   └── reports/
├── internal/
│   ├── agent/
│   │   ├── planner/
│   │   ├── retrieval/
│   │   ├── tool/
│   │   └── critic/
│   ├── auth/
│   ├── contextengine/
│   ├── eval/
│   ├── model/
│   ├── observability/
│   ├── retrieval/
│   ├── session/
│   ├── storage/
│   ├── tools/
│   │   ├── http/
│   │   ├── mcp/
│   │   └── registry/
│   └── workflow/
├── pkg/
├── scripts/
└── web/
    ├── admin/
    └── shared/
```

Rules:
- domain logic must not depend on HTTP handlers, CLI entrypoints, or vendor SDKs
- avoid circular dependencies
- prefer a modular monolith until operational pressure proves otherwise
- keep prompts, rubrics, and datasets versioned as code

## 6. Delivery order
When building from scratch, work in this order:

1. foundation
   - config loading
   - logger
   - health/readiness
   - migrations
   - local dev stack
   - make targets

2. baseline retrieval
   - document ingestion
   - chunking
   - embeddings
   - retrieval
   - citations
   - tenant scoping

3. agent runtime
   - planner
   - retrieval agent
   - tool agent
   - critic agent
   - typed orchestration
   - prompt versioning

4. workflow path
   - async jobs
   - approval flow
   - report generation
   - retries and timeouts
   - worker lifecycle

5. eval and agentops
   - traces
   - metrics
   - datasets
   - judges
   - regression reports

6. admin UX
   - task views
   - case management
   - report comparison
   - trace deep links

If a task conflicts with this order, explain why and choose the smallest vertical slice that still fits the architecture.

## 7. Preferred stack
Preferred unless the task explicitly requires something else:

- Go as the primary language
- HTTP API with SSE streaming for long-running user-visible responses
- PostgreSQL + pgvector for operational data and retrieval metadata
- Redis only when coordination or caching materially benefits latency or throughput
- Temporal for long-running, retryable, approval-gated workflows
- OpenTelemetry for traces and metrics
- `slog` for structured logs
- `pgx` + `sqlc` for database access
- manual dependency injection via constructors
- provider adapters behind internal interfaces
- HTTP and MCP adapters for tool integrations

Ask before adding heavy frameworks, ORMs, or new infrastructure services.

## 8. Coding rules
- Write idiomatic Go.
- Prefer explicit types over reflection, magic registries, or `map[string]any`.
- Use `context.Context` as the first parameter for request-scoped operations.
- Exported identifiers must have doc comments.
- Keep interfaces small and define them near the consumer.
- Favor composition over inheritance-like abstractions.
- Prefer pure functions for routing, scoring, ranking, and prompt assembly.
- Never panic in request or worker paths; return wrapped errors with `%w`.
- Use structured logs only; no committed `fmt.Println` debugging.
- No global mutable state except carefully scoped caches or configuration singletons.
- Add timeouts, retry policy, and idempotency considerations at all I/O boundaries.
- If a package name needs `common`, `misc`, or `utils`, reconsider the design.

## 9. Agent runtime rules
### 9.1 Planner
- planner outputs must be typed, auditable, and replayable
- orchestration must remain visible in Go control flow
- separate intent classification from execution planning as complexity grows

### 9.2 Retrieval
- never dump full chat history into retrieval
- always build retrieval input from a structured query object
- retrieval results must carry provenance:
  - source id
  - chunk id
  - score
  - timestamp or version
  - permissions scope
- keep re-ranking deterministic where feasible

### 9.3 Context engine
Context is layered, not a raw transcript append.

Maintain at least:
- recent turns
- task scratchpad
- user or tenant profile
- long-term retrieval context

Rules:
- summaries are derived artifacts and must be replaceable
- token budgets must be explicit
- drop the least valuable context first
- record why each context block was included

### 9.4 Tool execution
- classify tools as read-only or side-effecting
- side-effecting tools require approval, dry-run mode, or an explicit safeguard
- normalize tool outputs into typed structs
- persist tool-call audit records for every external action

### 9.5 Critic / judge
- critic validates answer quality, citation completeness, consistency, and policy risk
- judge prompts and production prompts must be versioned separately
- prompt or routing changes without matching eval updates are incomplete work

## 10. Workflow and reliability rules
- use synchronous request handling only for short operations
- use Temporal for long-running, retryable, or approval-gated tasks
- workflows orchestrate; activities perform I/O
- activities must be idempotent or guarded by idempotency keys
- prefer additive DB migrations
- destructive migrations require a rollback plan
- every user-visible async job needs:
  - status
  - error reason
  - retry surface
  - audit trail

## 11. API and contract rules
- external APIs are REST-first
- public endpoints must be documented with OpenAPI
- stream long responses over SSE unless there is a strong reason not to
- use a consistent error envelope with machine-readable codes
- thread `request_id`, `trace_id`, `user_id`, and `tenant_id` through logs and traces
- preserve backward compatibility once an endpoint or event is used outside the service

## 12. Data and storage rules
- keep operational data, eval data, and audit data logically separated
- store prompts, prompt versions, rubrics, and eval fixtures in version control
- never bury critical business logic in SQL migrations or ad-hoc scripts
- prefer soft-delete or append-only patterns for audit-heavy entities
- every retrieval table must preserve provenance and tenant scope

## 13. Observability and evaluation rules
- every LLM call, tool call, retrieval pass, and workflow step must be traceable
- add metrics for latency, error rate, token usage, tool success rate, and evaluation scores
- new agent behaviors should land with at least one regression case
- failed production cases should be promotable into eval datasets
- do not ship major prompt or routing changes without updating evaluation baselines or reports

## 14. Testing and validation
Before marking work complete, run the smallest relevant set first, then broader validation:

- targeted `go test` for touched packages
- `go test ./...`
- lint and format checks
- integration tests when storage, workflow, or external contract behavior changes

Preferred repo targets:
- `make fmt`
- `make lint`
- `make test`
- `make check`
- `make dev-up`
- `make dev-down`

If these targets do not exist yet, create them instead of relying on undocumented local commands.
Never claim tests passed unless they were actually run.

## 15. Documentation rules
- update README when setup, commands, or architecture assumptions change
- add ADRs for major choices:
  - provider boundary
  - retrieval design
  - workflow orchestration
  - evaluation strategy
  - tenancy and security
- new subsystems need a short package README or `doc.go`
- keep example env files and local bootstrap docs current

## 16. Skill sync rule
If a change affects subsystem workflow, setup steps, test commands, contracts, or operational guidance, update the corresponding skill in the same task.

Skill-only edits are allowed when improving playbooks, but the change report must state that the code path was unchanged.

## 17. Path-specific instruction policy
Use `AGENTS.override.md` in subsystem directories when local rules differ from repository-wide guidance.

Planned local override zones:
- `internal/agent/AGENTS.override.md`
- `internal/retrieval/AGENTS.override.md`
- `internal/workflow/AGENTS.override.md`
- `internal/eval/AGENTS.override.md`
- `web/admin/AGENTS.override.md`

Do not duplicate full skills into local AGENTS files.
Keep local AGENTS files short and focused on code that actually lives there.

## 18. Change protocol for coding agents
For any non-trivial task:
1. inspect the current code and adjacent modules before editing
2. state assumptions when requirements are ambiguous
3. propose a brief plan before large multi-file changes
4. make the smallest coherent diff
5. run validations
6. report changed files, why they changed, what was tested, and what remains

Ask before:
- adding new top-level dependencies
- changing public API contracts
- introducing new infrastructure services
- writing destructive migrations
- enabling network calls in tests
- automating side-effecting external actions

## 19. What "done" means here
A task is not done unless most of the following are true:
- code builds
- relevant tests pass
- lint and format pass
- logs, errors, and traces are sensible
- docs are updated
- prompt and eval changes are versioned together
- no hidden TODOs for core behavior remain without documentation

## 20. Response style for Codex and Claude Code
When finishing a task, report:
- summary of the implementation
- key design decisions
- files changed
- commands run
- risks or follow-ups

Be honest about uncertainty.
Do not invent outputs, metrics, or test results.
