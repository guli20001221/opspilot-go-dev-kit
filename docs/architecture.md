# Architecture summary

OpsPilot-Go is a production-oriented Golang multi-agent platform with these major layers:

- API / gateway
- application services under `internal/app/*`
- context engine
- Planner / Retrieval / Tool / Critic runtime
- workflow and approval layer
- retrieval and storage
- eval and observability
- admin console

The current foundation slice also includes a local development stack:

- PostgreSQL for application data and migrations
- Redis for future coordination and caching flows
- Temporal plus Temporal UI for workflow development
- API and worker processes bootstrapped through the same local Compose topology
- local Compose now starts prebuilt runtime images for the Go services instead of bind-mounting source and calling `go run` inside containers

The current Milestone 1 slice adds:

- `internal/session` for in-memory session and message persistence
- `internal/app/chat` as the application boundary for the synchronous chat skeleton
- `internal/app/admin/taskboard` as the first admin read model that converts workflow task pages into operator-facing task board summaries for future `web/admin` flows
- `web/admin` as the home for embedded operator pages, starting with the task board served directly by the API process
- the embedded task board page now drills into `GET /api/v1/tasks/{task_id}` for audit history and failure context instead of duplicating detail logic in the browser
- the same page now reuses `POST /api/v1/tasks/{task_id}/approve` and `POST /api/v1/tasks/{task_id}/retry` for operator actions, so the admin UI does not fork workflow mutation contracts
- the same embedded admin surface now includes `/admin/reports`, a report-lane view backed by the durable report list contract while still reusing task detail for execution provenance
- `internal/report` now holds the first durable report read model, emitted from successful `report_generation` workflow completion rather than inferred only from task audit history
- `internal/case` now holds the first durable operator case read model, so follow-up work can reference a source task, a source report, or both through stable IDs
- the same `internal/case` package now also supports filtered list reads for operator-facing case slices
- `internal/contextengine` for deterministic block assembly and assembly logging
- `internal/agent/planner` for deterministic typed execution plans
- `internal/retrieval` for deterministic structured-query retrieval and provenance-bearing evidence blocks
- `internal/agent/tool`, `internal/tools/registry`, and `internal/tools/http` for deterministic typed tool execution, request validation, and approval gating
- `cmd/ticketapi` plus `internal/tools/http/tickets.NewFakeHandler` for the dev-only fake ticket API used to validate the HTTP adapter path locally
- `internal/agent/critic` for deterministic structured verdicts over draft answers, retrieval, and tool results
- `internal/workflow` for store-backed promoted task records and the current Temporal bridge layer
- approval-gated workflow tasks now carry an internal tool payload for worker-side approved execution
- `internal/storage/postgres` for the current PostgreSQL task repository and connection pool wiring
- `internal/app/httpapi` as a thin transport layer over the session and chat services
- `cmd/api` for task creation plus Temporal-backed approval-workflow initialization
- `cmd/worker` plus `internal/workflow.Runner` for PostgreSQL-backed task claiming, Temporal report execution, and Temporal approval-workflow continuation and recovery
- the worker can optionally enable a dev-only approved-tool fault-injection path through configuration to verify failure and retry recovery

The current synchronous chat stream now surfaces internal runtime milestones over SSE:

- `plan` when the execution plan is derived
- `retrieval` when retrieval runs
- `tool` for each executed tool step
- `task_promoted` when the internal workflow layer creates an async task

The current HTTP layer also exposes the same PostgreSQL-backed workflow records over REST:

- `POST /api/v1/tasks` for explicit async task creation
- `GET /api/v1/tasks` for operator-facing filtered task listing with offset pagination metadata, including approval, promotion-reason, created-at, and updated-at window filters
- `GET /api/v1/tasks/{task_id}` for task status lookup
- `POST /api/v1/tasks/{task_id}/approve` to resume approval-gated tasks
- `POST /api/v1/tasks/{task_id}/retry` to re-queue failed tasks
- `GET /api/v1/admin/task-board` for the first admin read-model endpoint, which keeps visible-slice summaries and pagination metadata on the backend for future `web/admin` task pages
- `GET /admin/task-board` as the first operator page that renders summary cards, filters, and task rows directly from the admin read-model endpoint
- the board's quick-view presets now cover pending queues, queue-oriented slices, terminal success slices, report-success slices, approval-failure slices, reason slices, approval-lane slices, and task-type slices while still writing back into the same filter form and URL state
- the same page now exposes a task detail panel backed by the existing single-task API, keeping board, audit views, adjacent-task navigation, and existing task actions on one operator surface
- when a task `audit_ref` points at `temporal:workflow:<workflow_id>/<run_id>`, the detail panel now derives a direct Temporal UI history link without expanding the backend contract
- `GET /admin/reports` is the first report-focused operator page, fixed to `status=ready` and `report_type=workflow_summary` while consuming the canonical report list endpoint and the single-task detail endpoint for provenance
- the same report lane now keeps the selected report row visually synced with the detail pane and supports previous/next navigation within the current visible slice
- the same report lane now also supports lightweight polling against the existing admin read model, so no report-specific watch contract is needed for basic operator monitoring
- the same report lane now also supports copyable report summaries and shareable report links derived from the current task detail response, so operator handoff still reuses canonical task contracts
- `GET /api/v1/reports` now exposes the durable report artifact list, separate from workflow task queues
- `GET /api/v1/reports/{report_id}` now exposes the durable report artifact emitted by a successful report task without forcing clients to parse task audit history
- `internal/version` now holds the durable runtime version registry, so reproducibility metadata such as planner, retrieval, tool, critic, and workflow bundle versions have a stable backend contract
- `GET /api/v1/versions` and `GET /api/v1/versions/{version_id}` now expose that runtime version registry directly
- `GET /api/v1/report-compare` now exposes a narrow read-only comparison contract over two durable report IDs, so regression-style operator review does not require the frontend to diff report artifacts on its own
- `GET /admin/report-compare` now exposes the first report-comparison page, wired directly to the durable compare contract and report read endpoints
- `internal/observability/tracedetail` now exposes a narrow read-only lineage resolver over durable tasks, reports, and cases, so operator pages can share one trace drill-down path without each page re-deriving provenance
- `GET /api/v1/trace-drilldown` and `GET /admin/trace-detail` now form the first shared trace drill-down surface across task, report, comparison, and case pages
- `GET /admin/version-detail` now provides the shared version-drill-down surface, and the reports, report-compare, and trace-detail pages hand off into it using durable `version_id` references instead of duplicating runtime metadata in the browser
- `POST /api/v1/cases` and `GET /api/v1/cases/{case_id}` now expose the durable operator case contract, separate from task/report runtime status
- `GET /api/v1/cases` now exposes the first operator-facing case list with tenant, status, source-task, and source-report filters plus offset pagination
- the same case list now supports explicit `assigned_to` and `unassigned_only` filtering so queue views can map cleanly onto owned and shared operator lanes without inventing frontend-only state
- `POST /api/v1/cases/{case_id}/close` now provides the first case lifecycle mutation, recording `closed_by` while keeping case status transitions explicit and REST-first
- `POST /api/v1/cases/{case_id}/reopen` now provides the inverse lifecycle mutation, returning a closed case to the open queue and appending a durable operator note for the reopen action
- `POST /api/v1/cases/{case_id}/assign` now provides the first case ownership mutation, recording `assigned_to` and `assigned_at` while keeping ownership explicit and REST-first
- `POST /api/v1/cases/{case_id}/notes` now provides append-only case collaboration, and `GET /api/v1/cases/{case_id}` returns recent notes without introducing a separate admin-only comment surface
- the same report lane now reads report title, summary, and readiness metadata from the durable report endpoint while still reusing task detail for audit timeline and Temporal links
- the same report lane now reads durable version metadata from the canonical version endpoint, so report provenance can jump from report artifact to runtime snapshot without treating task JSON as the source of truth
- `GET /admin/cases` is the first case-focused operator page, backed directly by the durable case contract and existing task/report detail endpoints, and now supports the minimal close action for open cases
- the same case page now also supports copyable case summaries, shareable case links, and a direct jump into the canonical case-detail JSON without any admin-only debug contract
- the same case page now also surfaces and updates assignment, so ownership stays in the canonical case contract instead of drifting into frontend-only state
- `internal/eval` now holds the first durable eval-case read model, so failure-case promotion is rooted in canonical case lineage instead of frontend-only bookmarks
- `POST /api/v1/eval-cases` and `GET /api/v1/eval-cases/{eval_case_id}` now expose durable eval-case promotion derived from canonical case, task, report, trace, and version state
- `GET /api/v1/eval-cases` now exposes the first tenant-scoped eval queue contract, so promoted coverage can be browsed as an operator lane instead of remaining write-only
- the same canonical eval-case list/detail contract now also carries follow-up case summary fields and `latest_follow_up_case_id`, so eval triage can inspect linked operator work without an extra case query layer
- the same canonical eval-case list now also supports `needs_follow_up=true|false`, so unresolved eval follow-up can be filtered on the backend instead of derived in the browser
- the canonical case contract now also accepts standalone `source_eval_case_id`, so precise eval-case follow-up can be created or reused without routing through an eval-report-level contract first
- the same `/admin/cases` page now also supports `Promote to eval`, keeping the operator action on the canonical case surface instead of introducing an admin-only eval write path
- `/admin/evals` now exposes the first eval-focused operator lane, reusing durable eval-case list/detail reads and canonical handoff links into case/task/report/version/trace surfaces
- `POST /api/v1/eval-datasets`, `GET /api/v1/eval-datasets`, and `GET /api/v1/eval-datasets/{dataset_id}` now expose the first durable dataset-draft contract plus its canonical lightweight browse surface
- the same `/admin/evals` page now also supports `Create dataset draft`, so operators can turn one promoted failure into a reusable regression asset and then hand off into the shared dataset lane
- `POST /api/v1/eval-datasets/{dataset_id}/items` now exposes the first incremental dataset-curation contract, so operators can grow a draft dataset over time instead of recreating it per eval case
- the same `/admin/evals` page now also supports `Add to dataset`, reusing the canonical append-membership contract instead of inventing an eval-page-only saved-view mutation
- the same `/admin/evals` page now also surfaces canonical eval-case follow-up summary and handoff links into the latest follow-up case or the full follow-up slice, instead of re-deriving that state in the browser
- the same `/admin/evals` page now also exposes a `Needs follow-up` quick view backed by that canonical eval-case filter, turning follow-up pressure into a real operator lane
- the same `/admin/evals` page now also supports direct case creation from one durable eval case, reusing `POST /api/v1/cases` with standalone `source_eval_case_id` and handing off into `/admin/cases`
- when that eval-case follow-up already has open work, the same `/admin/evals` primary action now flips to `Open existing case` or `Open existing queue`, so reuse is visible in the operator flow before any write is attempted
- `/admin/eval-datasets` now exposes the first dataset-focused operator lane, keeping dataset list rows lightweight while reusing canonical dataset detail and source-lineage handoff paths
- `POST /api/v1/eval-datasets/{dataset_id}/publish` now turns a durable draft into an immutable published baseline, recording `published_by` and `published_at` so later regression work can target stable dataset state instead of a moving draft
- `internal/eval` now also holds the first durable eval-run kickoff model, which snapshots published dataset metadata into a queued run row before judge execution is connected
- `POST /api/v1/eval-runs`, `GET /api/v1/eval-runs`, and `GET /api/v1/eval-runs/{run_id}` now expose that tenant-scoped run-kickoff contract
- `/admin/eval-runs` is the first eval-run operator lane, and `/admin/eval-datasets` now hands published baselines into it through `Run dataset`
- `/admin/eval-reports` is the first eval-report operator lane, reusing the canonical eval-report list/detail contracts instead of reconstructing aggregated artifacts from run detail in the browser
- the same eval-report lane now also carries follow-up case summary directly on canonical list rows, so queue-level triage does not need a second case-list fetch per row
- the same canonical eval-report list now also supports a `needs_follow_up` filter, so operator lanes can pull unresolved-regression slices without inventing a second queue contract
- that canonical eval-report list also carries `latest_follow_up_case_id`, so operator lanes can hand off directly into the freshest linked follow-up case without an extra per-row lookup
- the eval-report detail pane reuses that same canonical field for its `Open latest case` action instead of inventing a separate handoff endpoint
- the eval-report compare contract now also carries `latest_follow_up_case_id` per side, so compare-time triage can hand off into canonical cases without a second lookup surface
- that same compare contract now also carries per-side follow-up summary fields, keeping “is this already being worked?” on the same screen as the compare decision
- the same eval-report lane now also reuses the canonical case list filter `source_eval_report_id`, so operators can see linked durable follow-up cases for the selected regression without a second backend surface
- the same eval-report lane now also reuses `POST /api/v1/cases` with `source_eval_report_id`, so an operator can create a durable follow-up directly from the canonical eval-report detail without detouring through another page
- when that eval-report follow-up lineage already has open work, the canonical `POST /api/v1/cases` path now reuses the newest open case for the same `tenant_id + source_eval_report_id` instead of minting duplicate regression cases
- when that open eval-report follow-up already exists, the same `/admin/eval-reports` primary action now flips to `Open existing case` or `Open existing queue`, so reuse is visible in the operator flow before any write is attempted
- the same canonical eval-report detail now also decorates each bad case with follow-up summary and `latest_follow_up_case_id`, so bad-case triage can hand off into durable case state without adding another backend surface
- when a bad case already has open follow-up work, the same `/admin/eval-reports` row-level action now flips to `Open existing bad-case case` or `Open bad-case queue`, so operators reuse the canonical bad-case slice before opening another regression case
- that same canonical eval-report detail now also supports a `bad_case_needs_follow_up` filter, so bad-case triage slices stay backend-owned instead of becoming a browser-only filter over already-loaded rows
- that same canonical eval-report detail now also carries stable `bad_case_count`, so report-level case handoff is not distorted by a filtered bad-case drill-down
- the same canonical eval-report list now also carries `bad_case_without_open_follow_up_count`, so unresolved bad-case pressure becomes a durable queue signal instead of a detail-only inference
- that same canonical eval-report list now also supports `bad_case_needs_follow_up=true|false`, and `/admin/eval-reports` uses it for an unresolved-bad-case slice without inventing a second queue contract
- the same eval-report lane now also supports bad-case-specific follow-up through `source_eval_case_id`, so one failing eval case can promote to its own canonical case without collapsing back into the broader report-level follow-up
- `/admin/eval-report-compare` is the first eval-report comparison lane, reusing a narrow canonical compare contract instead of diffing eval reports ad hoc in the browser
- the same compare lane now also hands regression findings into the canonical case lifecycle by reusing side-specific `POST /api/v1/cases` actions and deep-linking to `/admin/cases`, instead of inventing an admin-only follow-up store
- the same canonical compare contract now also carries per-side `bad_case_without_open_follow_up_count`, so compare can surface unresolved bad-case pressure without sending operators back to the report lane first
- the same compare lane now also hands a side straight into `/admin/eval-reports?bad_case_needs_follow_up=true&report_id=...`, so unresolved bad-case triage stays on canonical report state instead of becoming compare-only browser logic
- the durable case contract now also carries `source_eval_report_id`, so compare-created follow-up work keeps a canonical pointer back to the originating eval report and `/admin/cases` can hand operators back into the eval-report lane without parsing summary text
- the durable case contract now also carries `source_eval_case_id`, so bad-case follow-up can point at one canonical eval case and `/admin/cases` can hand operators back into the exact eval-case artifact
- the canonical case list contract now also supports `source_eval_report_id` and `eval_backed_only`, so `/admin/cases` can expose a true eval-backed queue without front-end-only provenance filtering
- the same `/admin/cases` detail pane now also reads source eval-report metadata from the canonical eval-report detail endpoint and degrades to surviving case provenance if that report row is missing
- the same canonical case contract now also persists compare-origin fields for left/right eval reports plus the selected side, so case detail can hand operators back into `/admin/eval-report-compare` without scraping that context out of notes or summaries
- the canonical case list contract now also supports `compare_origin_only`, so `/admin/cases` can expose a true compare-follow-up queue without client-side provenance filtering
- when compare provenance is already present on case list rows, `/admin/cases` should hand operators straight back into `/admin/eval-report-compare` from the queue instead of forcing a detail drill-down first
- when a case queue is already backed by the canonical case contract, prefer row-level claim actions that reuse `POST /api/v1/cases/{case_id}/assign` instead of forcing every claim through detail-only controls
- when an operator queue is already filtered to open cases, prefer row-level resolution actions that reuse `POST /api/v1/cases/{case_id}/close` instead of forcing a detail-only close workflow
- when a closed case is already visible in the canonical case queue, prefer row-level recovery actions that reuse `POST /api/v1/cases/{case_id}/reopen` instead of forcing a detail-only reopen workflow
- when a claimed open case needs to return to the shared backlog, prefer row-level or detail actions that reuse `POST /api/v1/cases/{case_id}/unassign` and append a durable operator note instead of encoding release as an empty assign payload
- the worker now also claims queued eval runs and advances them through `queued -> running -> succeeded|failed` with placeholder execution, persisting `started_at`, `finished_at`, and `error_reason` on the canonical run record
- failed eval runs now have an explicit retry path on that same canonical record through `POST /api/v1/eval-runs/{run_id}/retry`, and the `/admin/eval-runs` lane reuses it directly for operator recovery
- eval runs now also emit append-only `created`, `claimed`, `failed`, `retried`, and `succeeded` events on the same `run_id`, and detail reads surface that timeline while the list contract stays lightweight
- eval runs now also snapshot immutable `items` copied from the published dataset membership at kickoff time, so future judging and operator drill-down can rely on run-local provenance instead of re-reading mutable dataset detail
- eval-run detail now also carries durable terminal `item_results`, so placeholder per-item outcomes stay attached to the same run record and are cleared when retry re-queues the canonical run
- those same durable `item_results` now also carry structured placeholder judge metadata such as `verdict`, `score`, `judge_version`, and raw `judge_output`
- the built-in placeholder judge now lives behind a dedicated `RunJudge` runtime boundary and points at a versioned prompt artifact under `eval/prompts`, so later provider-backed judging can swap the execution body without redesigning the run-result contract
- that same `RunJudge` boundary now also supports an env-gated HTTP provider implementation, while the runner still degrades to placeholder failure results if the external judge call cannot finalize the canonical run
- completed eval runs now also materialize a durable aggregated eval report read model, so top-line metrics, bad-case provenance, and judge metadata live in a canonical backend artifact instead of being derived ad hoc from run detail
- `GET /api/v1/eval-reports` and `GET /api/v1/eval-reports/{report_id}` now expose that durable aggregated eval-report artifact directly, with a lightweight list contract and heavier single-report drill-down
- `GET /api/v1/eval-report-compare` now exposes a narrow read-only compare contract over two durable eval reports, carrying top-line metric deltas, metadata drift, and bad-case overlap for operator review
- the same compare contract now also carries compare-derived follow-up queue summary per side, so compare pages can hand operators into the canonical compare-origin case lane without browser-side inference
- canonical case creation now deduplicates exact compare-origin lineage on the backend, so repeated compare handoff for the same left/right/selected-side follows the existing open case instead of creating duplicate regression work
- canonical eval-run reads now also attach a lightweight `result_summary` on terminal runs, letting list and detail consumers scan placeholder pass/fail totals without moving the heavier `item_results` payload onto create/list/retry responses
- the same case page now also shows and appends recent notes, so operator handoff context lives on the case instead of being implied by task/report provenance
- the same case page now defaults into an open-case queue view, adds `My open cases` and `Unassigned` shortcuts, and computes age/staleness from canonical `updated_at`
- the same case page now also foregrounds operator queue slices by highlighting `My open cases` and `Unassigned`, and it surfaces task-only versus report-backed provenance directly from the canonical case contract
- the same case page now also lets an operator reopen a closed case and return it directly to the open queue, instead of forcing a new case record for follow-up
- the existing task-board and report-lane detail panes can now create durable cases by reusing `POST /api/v1/cases`, keeping case creation on canonical task/report surfaces instead of inventing admin-only write APIs
- the task-board handoff now preserves durable report lineage for successful report tasks only when the durable report row actually exists, and otherwise degrades to a task-only case if the durable report lookup is missing or temporarily unavailable; the report-lane fallback path keeps case creation disabled until that row is present
- the same report lane can now surface and copy the raw durable report JSON directly from the report endpoint, so artifact troubleshooting stays contract-first too
- the same report lane now falls back to task provenance when a legacy or partially recovered successful report task has no durable report row, so operator drill-down remains readable
- the worker now finalizes report success and durable report persistence together, so report `ready_at` and `metadata.audit_ref` match the final task success surface
- the same page can optionally auto-refresh against the existing board and task-detail endpoints, so operator monitoring does not require manual reload loops
- the board now also includes quick-view presets for common operator slices, but those presets still flow through the same existing filter fields and backend read model
- the detail panel can now expose the raw single-task JSON payload from the existing detail endpoint, keeping debugging and escalation views contract-first as well
- the same detail panel now exposes handoff actions that stay contract-first: copy the selected board URL or open the canonical task-detail JSON in a separate tab
- the same detail panel can now derive a compact audit-summary string from the selected task response and its audit events, giving operators a contract-first handoff artifact without a new backend surface
- the same detail panel now supports previous/next navigation within the current board slice and derives execution/timeline digest cards from the selected task response without introducing extra backend aggregation
- the same detail panel can now reapply the board filters to the selected task lane by writing back into the existing tenant/task-type/reason/requires-approval filters rather than introducing a separate frontend query model
- the same detail panel can now also reapply the selected task queue back into the existing board filter form by combining `status` and `requires_approval`, keeping queue-based triage inside the same query model
- the same detail panel can now also reapply the selected task type back into the existing board filter form, keeping task-type triage inside the same query model
- the same detail panel can now also reapply the selected task approval lane back into the existing board filter form, keeping approval-lane triage inside the same query model
- the same detail panel can now also reapply the selected task reason back into the existing board filter form, keeping reason-based triage inside the same query model
- the board now also keeps the selected row highlighted and scroll-synced with the detail panel, so navigation across the current slice does not break table context
- the same detail panel can also reapply the selected task status back into the existing board filter form, keeping status-based triage inside the same query model
- `audit_events` embedded in task responses as the current structured operator audit view
- the list endpoint intentionally omits `audit_events`, so the summary surface stays cheap while the single-task endpoint remains the detailed audit drill-down, and it returns `has_more` plus `next_offset` for simple operator pagination
- `error_reason` normalized to an operator-facing summary while deep Temporal detail remains in worker logs

Within the current PostgreSQL-backed workflow runtime, task-state changes and their matching
audit-event inserts now commit in the same transaction for create, claim, approve, retry,
success, and failure paths.

The current worker path advances supported queued tasks through:

- `queued -> running -> succeeded` for report generation, with the execution body now running inside a Temporal workflow and activity
- `waiting_approval -> queued -> running -> succeeded` for approved tool execution, with the waiting phase and resume signal tracked in Temporal
- `waiting_approval -> queued -> running -> failed -> queued -> running -> succeeded` for approved tool execution recovery, where a failed approval run closes and retry starts a new Temporal run for the same task ID
- approval tasks promoted from chat carry the selected tool name and typed arguments; legacy tasks without payload keep the older placeholder-compatible execution path
- the default ticket adapters now execute through typed request/response contracts, so approved-tool runs can reject invalid payloads instead of silently succeeding on fixed stub output
- registry construction is now config-driven: without a ticket API base URL it uses deterministic local adapters, and with one it switches both API and worker to the HTTP ticket adapter through the same typed executor hook
- task success audit events now carry execution summaries from the executor path, which gives operators a concise description of what completed without changing the task response schema
- task failure audit events now use categorized detail prefixes while leaving `error_reason` as the shorter root-cause string
- `queued -> running -> failed` for unsupported task types

This file is intentionally brief in the AI development kit.
Promote it to the main repository and expand it as implementation begins.
