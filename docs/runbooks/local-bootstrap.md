# Local Bootstrap

## Scope

This runbook covers the current foundation slice only:

- Go module bootstrap
- API binary with `/healthz` and `/readyz`
- worker bootstrap
- local Docker Compose stack for PostgreSQL, Redis, Temporal, fake ticket API, API, and worker
- Make targets for format, test, build, and check

It does not yet wire real DB access from the app code or a real OpenTelemetry exporter.

## Prerequisites

- Go 1.24.2
- Optional: `make`
- PowerShell for the fallback script on Windows
- Docker Desktop with the daemon running

## Commands

1. Copy `.env.example` values into your local shell environment if you need overrides.
2. If `make` is installed, run `make test` and `make build`.
3. If `make` is not installed, run `powershell -File scripts/dev/tasks.ps1 test` and `powershell -File scripts/dev/tasks.ps1 build`.
4. Validate the Compose file with `docker compose config`.
5. Start the local stack with `make dev-up` or `powershell -File scripts/dev/tasks.ps1 dev-up`.
   This now runs `docker compose up -d --build`, so the app services start from prebuilt binaries rather than runtime `go run`.
6. Check `http://localhost:18080/healthz`.
7. Check `http://localhost:18080/readyz`.
8. Check Temporal UI at `http://localhost:8088`.
9. Check the fake ticket API at `http://localhost:19090/tickets/search?q=INC-100` with header `Authorization: Bearer local-dev-ticket-token` if you want to verify the HTTP adapter boundary directly.
10. Open `http://localhost:18080/admin/task-board` to inspect the embedded operator page against the local admin read model.
11. Open `http://localhost:18080/admin/reports` to inspect the first report-focused operator page against the same admin read model.
12. Open `http://localhost:18080/admin/eval-reports` to inspect the first eval-report-focused operator page against the durable eval-report contract.
13. Open `http://localhost:18080/admin/eval-report-compare` to compare two durable eval reports through the canonical compare contract and create follow-up cases from either side when needed.

Successful build artifacts are emitted under `bin/`.

## Current API surface

- `POST /api/v1/sessions`
- `GET /api/v1/sessions/{session_id}/messages`
- `POST /api/v1/tasks`
- `GET /api/v1/tasks`
- `GET /api/v1/tasks/{task_id}`
- `POST /api/v1/tasks/{task_id}/approve`
- `POST /api/v1/tasks/{task_id}/retry`
- `GET /api/v1/admin/task-board`
- `GET /admin/task-board`
- `GET /admin/cases`
- `GET /admin/reports`
- `GET /admin/eval-reports`
- `GET /admin/version-detail`
- `GET /api/v1/reports/{report_id}`
- `GET /api/v1/versions`
- `GET /api/v1/versions/{version_id}`
- `GET /api/v1/cases`
- `POST /api/v1/cases`
- `GET /api/v1/cases/{case_id}`
- `POST /api/v1/cases/{case_id}/close`
- `POST /api/v1/cases/{case_id}/assign`
- `POST /api/v1/cases/{case_id}/notes`
- `POST /api/v1/eval-cases`
- `GET /api/v1/eval-cases`
- `GET /api/v1/eval-cases/{eval_case_id}`
- `POST /api/v1/eval-datasets`
- `GET /api/v1/eval-datasets`
- `GET /api/v1/eval-datasets/{dataset_id}`
- `POST /api/v1/eval-datasets/{dataset_id}/items`
- `POST /api/v1/eval-datasets/{dataset_id}/publish`
- `POST /api/v1/eval-runs`
- `GET /api/v1/eval-runs`
- `GET /api/v1/eval-runs/{run_id}`
- `POST /api/v1/eval-runs/{run_id}/retry`
- `GET /api/v1/eval-reports`
- `GET /api/v1/eval-reports/{report_id}`
- `GET /api/v1/eval-report-compare`
- `POST /api/v1/chat/stream`

The current chat stream implementation is a Milestone 1 skeleton:
- session storage is in-memory
- task storage is PostgreSQL-backed in the API runtime
- the worker process polls queued tasks and advances supported task types to terminal states
- `report_generation` is executed through a Temporal workflow on the `opspilot-report-tasks` queue when Temporal is enabled
- successful `report_generation` runs now also persist a durable report row, addressable as `report-<task_id>` through `GET /api/v1/reports/{report_id}`
- `approved_tool_execution` now starts a waiting Temporal workflow at task creation time and is resumed by the worker after the approval action updates the task row
- if `approved_tool_execution` fails after approval, the current Temporal run closes, the task row moves to `failed`, and `POST /api/v1/tasks/{task_id}/retry` starts a new failed-only Temporal run for the same task
- set `OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE=true` on the worker to force the first approval attempt to fail while keeping retry successful
- approval tasks promoted from chat now carry an internal tool payload so worker-side approved execution can run the registered tool after approval; manually created approval tasks without payload still use the compatibility path
- the local compose stack now starts a fake ticket API and routes the default ticket tools through `http://ticket-api:8090`
- set `OPSPILOT_TICKET_API_BASE_URL` yourself only when you want to override that default and target a different ticket service; outside compose, leaving it empty keeps the deterministic local ticket adapters
- approval-gated tasks can be resumed through the approval action endpoint
- failed tasks can be re-queued through the retry action endpoint
- task responses now include structured `audit_events`
- `GET /api/v1/tasks` now supports `tenant_id`, `status`, `task_type`, `reason`, `requires_approval`, `created_after`, `created_before`, `updated_after`, `updated_before`, `limit`, and `offset` filters for operator listing, with the time filters parsed as RFC3339 values, and returns `has_more` plus `next_offset` while keeping per-task `audit_events` only on `GET /api/v1/tasks/{task_id}`
- `GET /api/v1/admin/task-board` reuses the same filters but returns a backend task-board read model with visible-slice summary counts for the current page
- `GET /admin/task-board` is the first embedded operator UI and mirrors the same filters in a simple browser form while keeping all summary logic on the backend
- the same page now supports task drill-down, so operators can inspect `audit_events`, `error_reason`, and `audit_ref` without leaving the board
- the detail panel also surfaces `Approve task` and `Retry task` controls when the current task state allows them, and those controls call the existing task action endpoints with the operator actor you enter on the page
- when a task has a Temporal-backed `audit_ref`, the same detail panel derives an `Open workflow history in Temporal UI` link so you can jump directly into the matching run
- enable `Auto refresh every 5s` on that same page when you want the board and selected task detail to keep tracking state changes without manual reload
- use the `Quick views` buttons on that page when you want common operator slices such as `Queued`, `Needs approval`, `Failed`, `Failed approvals`, `Running`, `Succeeded`, `Succeeded reports`, `Workflow required`, `Approval required`, `No approval`, `Report tasks`, or `Approved tools` without manually setting the same filters repeatedly
- use `Show raw JSON` in the same detail panel when you need the full task payload and audit structure for debugging, and `Copy raw JSON` when you want to paste that payload into an incident or handoff
- use `Copy task link` when you want to hand another operator the exact filtered board URL with the selected task in context, and `Open API detail` when you want the canonical JSON response in a separate tab
- use `Copy audit summary` when you want a compact text summary of task status, reason, audit reference, and timeline events for an incident note or shift handoff
- use `Previous visible` and `Next visible` when you want to triage neighboring tasks in the current filtered slice, and rely on the new execution/timeline digest cards at the top of the detail panel for a quick read before dropping into the full audit timeline
- use `Focus same lane` when you want the board to narrow to the selected task's tenant, task type, reason, and approval lane without manually re-entering those filters
- use `Focus same queue` when you want the board to narrow to the selected task state plus approval mode without re-entering both `status` and `requires_approval`
- use `Focus same task type` when you want the board to narrow to report-generation or approved-tool work in the current tenant without re-entering the `task_type` filter
- use `Focus approval lane` when you want the board to narrow to approval-gated or non-approval work in the current tenant without re-entering the `requires_approval` filter
- use `Focus same reason` when you want the board to narrow to the selected task reason, for example all `workflow_required` or `approval_required` work in the current tenant
- the selected task row is now highlighted in the table and follows detail navigation, so you can keep your place in the current slice while drilling between neighboring tasks
- use `Focus same status` when you want the board to narrow to the selected task state, for example all `waiting_approval`, `running`, or `failed` tasks in the current tenant
- `GET /admin/reports` fixes the lane to successful report-generation tasks and reuses the same backend contracts, so you can inspect report execution provenance without manually composing board filters each time
- use `Previous visible` and `Next visible` on `/admin/reports` when you want to step through the current visible report slice without bouncing back to the board list
- enable `Auto refresh every 5s` on `/admin/reports` when you want the report lane and selected report detail to track newly completed reports without manual reload
- use `Copy report summary` on `/admin/reports` when you need a compact, paste-ready handoff note for the selected successful report, and `Copy report link` when you want to share the exact filtered reports URL with the current report selected
- the `/admin/reports` detail panel now reads report title, summary, and ready time from `GET /api/v1/reports/{report_id}`, while still using task detail for audit timeline and Temporal history handoff
- open `http://localhost:18080/admin/version-detail` when you want the canonical runtime version registry and one-click handoff into a specific version snapshot
- use `GET /api/v1/versions` when you need the current durable runtime version registry, and `GET /api/v1/versions/{version_id}` when you need the canonical reproducibility record behind a task, report, or trace drill-down
- task, report, and trace responses now carry `version_id`, and the report, report-compare, and trace-detail admin pages hand off into `/admin/version-detail` using that stable reference instead of reconstructing runtime metadata in the browser
- use `Show raw report JSON` on `/admin/reports` when you need the exact durable report artifact, and `Copy raw report JSON` when you want to paste that artifact into an incident or escalation thread
- if a legacy successful report task has no durable report row yet, `/admin/reports` now falls back to task provenance and keeps the detail panel readable instead of failing the inspect flow
- use `GET /api/v1/reports/report-<task_id>` when you need the canonical report read model behind a successful report task, without parsing task audit history yourself
- use `POST /api/v1/cases` when you need a durable operator follow-up object that can point at a source task, a source report, or both
- use `GET /api/v1/cases` when you need to inspect the current durable case slice for a tenant, status, or source linkage
- use `GET /api/v1/cases/{case_id}` when you need the canonical case record for that follow-up object
- use `POST /api/v1/cases/{case_id}/close?tenant_id=<tenant>` when you need to close an open follow-up object and capture who closed it
- use `POST /api/v1/cases/{case_id}/assign?tenant_id=<tenant>` when you need to claim or reassign an open follow-up object and capture who owns it
- use `POST /api/v1/cases/{case_id}/notes?tenant_id=<tenant>` when you need to append a durable operator note to the case timeline
- use `POST /api/v1/eval-cases` when you need to promote a canonical case into durable eval coverage while preserving source case, task, report, trace, and version lineage
- use `GET /api/v1/eval-cases?tenant_id=<tenant>` when you need the first tenant-scoped queue of promoted eval coverage before creating datasets or regression runs
- use `POST /api/v1/eval-datasets` when you need to turn one or more durable eval cases into a draft dataset for later regression work
- use `GET /api/v1/eval-datasets?tenant_id=<tenant>` when you need the lightweight dataset lane without pulling full membership payloads into the list response
- use `POST /api/v1/eval-datasets/{dataset_id}/items` when you need to append another durable eval case into an existing draft dataset instead of creating a new draft
- use `POST /api/v1/eval-datasets/{dataset_id}/publish` when curation is complete and you need an immutable baseline for later eval runs
- use `POST /api/v1/eval-runs` when you need to create a durable queued eval run from a published dataset baseline
- use `GET /api/v1/eval-runs?tenant_id=<tenant>` when you need the tenant-scoped eval-run kickoff queue
- use `GET /api/v1/eval-runs/{run_id}?tenant_id=<tenant>` when you need the canonical run detail for one kickoff record
- the worker now advances queued eval runs through `running` to `succeeded` or `failed`, so `started_at`, `finished_at`, and `error_reason` on the canonical run record are now meaningful operator fields
- use `POST /api/v1/eval-runs/{run_id}/retry?tenant_id=<tenant>` when you need to re-queue a failed run without creating a second durable run row
- the same run detail now returns append-only `events`, so prior `failed` and `retried` history remains visible after retry clears the top-level failure fields
- the same run detail now also returns immutable `items`, so you can inspect the exact eval-case membership and case/task/report/trace/version lineage that were snapped onto the run at kickoff time
- the same run detail now also returns durable `item_results`, so placeholder terminal outcomes for each snapped eval case remain inspectable on the canonical run until retry clears them
- those same `item_results` now also expose structured placeholder judge fields such as `verdict`, `score`, `judge_version`, and raw `judge_output`
- the built-in placeholder judge is now emitted through a dedicated eval judge runtime and points at `eval/prompts/placeholder-eval-judge-v1.md`, so later provider-backed judging can replace the execution body without changing the current run-result contract
- set `OPSPILOT_EVAL_JUDGE_PROVIDER=http_json`, `OPSPILOT_EVAL_JUDGE_BASE_URL`, `OPSPILOT_EVAL_JUDGE_MODEL`, and optionally `OPSPILOT_EVAL_JUDGE_API_KEY` when you want the worker to call an external judge service while preserving the same canonical `item_results` shape
- if that external judge call fails during run finalization, the worker now records a canonical failed run with placeholder fallback `item_results` instead of leaving the run stuck in `running`
- once a run reaches a terminal state, the worker also materializes a durable aggregated eval report carrying top-line metrics, bad-case lineage, and judge metadata for later comparison/reporting slices
- use `GET /api/v1/eval-reports?tenant_id=<tenant>` when you need the lightweight tenant-scoped browse lane for those durable eval reports
- that same eval-report list now also includes `follow_up_case_count`, `open_follow_up_case_count`, and `latest_follow_up_case_status`, so `/admin/eval-reports` can surface regression follow-up pressure directly from the canonical list contract
- use `needs_follow_up=true` on `GET /api/v1/eval-reports` when you want only eval reports with at least one open follow-up case, or `needs_follow_up=false` when you want reports whose follow-up queue is already clear
- use `GET /api/v1/eval-reports/{report_id}?tenant_id=<tenant>` when you need the canonical aggregated eval report detail, including metadata and bad-case lineage
- open `http://localhost:18080/admin/eval-reports` when you want the first eval-report operator page, including bad-case drill-down plus run, dataset, eval, trace, and version handoff links
- use the `Needs follow-up` quick view on `/admin/eval-reports` when you want the unresolved-regression slice without manually entering `needs_follow_up=true`
- use `Open latest case` on `/admin/eval-reports` rows when you want to jump straight into the freshest linked follow-up case from the canonical list slice
- that same `Open latest case` handoff also appears inside the eval-report detail pane once a report is selected, so operators do not need to return to the table row to continue case triage
- use `Open left latest case` / `Open right latest case` on `/admin/eval-report-compare` when you need to inspect existing follow-up before deciding whether to create another case from the comparison
- check each compare card's follow-up summary on `/admin/eval-report-compare` when you need to know whether a side already has open regression work before creating another case
- use `Open left linked cases` / `Open right linked cases` on `/admin/eval-report-compare` when you need the full canonical case slice for one side's `source_eval_report_id`, not just the latest linked case
- use `Open linked cases` on `/admin/eval-reports` when you want to jump from one durable eval report straight into the canonical `/admin/cases?source_eval_report_id=<report_id>` slice
- use `GET /api/v1/eval-report-compare?tenant_id=<tenant>&left_report_id=<left>&right_report_id=<right>` when you need a canonical eval-report delta view with score change, metadata drift, and bad-case overlap
- open `http://localhost:18080/admin/eval-report-compare` when you want the first eval-report comparison page, including handoff into eval runs, version detail, and side-specific case creation
- terminal run reads now also expose `result_summary`, so `/api/v1/eval-runs` and `/admin/eval-runs` can show quick pass/fail totals without loading the full per-item payload first
- open `http://localhost:18080/admin/cases` when you want the first case-focused operator page, including source task/report handoff links and the minimal `Close case` action
- open `http://localhost:18080/admin/evals` when you want the first eval-focused operator page, including durable eval detail plus case/task/report/version/trace handoff links
- use `Create dataset draft` on `/admin/evals` when you want to seed a canonical dataset draft directly from the currently selected durable eval case
- use `Add to dataset` on `/admin/evals` when you want to append the currently selected durable eval case into an existing dataset draft by ID
- open `http://localhost:18080/admin/eval-datasets` when you want the first dataset-focused operator page, including dataset membership detail plus eval/case/task/report/version/trace handoff links
- use `Publish dataset` on `/admin/eval-datasets` when you want to freeze the selected draft and make the page read-only for that baseline
- use `Run dataset` on `/admin/eval-datasets` when you want to create a durable queued eval run from the selected published baseline and land on the matching `/admin/eval-runs` detail
- open `http://localhost:18080/admin/eval-runs` when you want the first eval-run operator page, including run detail plus dataset and eval handoff links
- set `OPSPILOT_EVAL_RUN_FAIL_ALL=true` on the worker when you want every claimed eval run to fail for local recovery and operator-surface testing
- use `Retry run` on `/admin/eval-runs` when you want to re-queue the selected failed run back into the worker lane from the same detail panel
- use the `Run timeline` card on `/admin/eval-runs` when you need the durable claim/fail/retry/succeed history for the selected run ID
- use the `Run items` card on `/admin/eval-runs` when you need the selected run's eval-case membership and provenance handoff links without leaving the run lane
- use the `Item results` card on `/admin/eval-runs` when you need the selected run's placeholder per-item terminal outcomes without unpacking the raw JSON payload
- the same `Item results` card now also surfaces structured placeholder judge fields, so you can inspect verdict, score, and judge version without reading the raw JSON pane
- use the `Results` column on `/admin/eval-runs` when you want a quick terminal pass/fail count before drilling into the selected run's full `item_results`
- use the `My open cases` shortcut on `/admin/cases` when you want a queue view for the current operator handle without manually composing `status=open&assigned_to=<actor>`
- use the `Unassigned` shortcut on `/admin/cases` when you want the shared open backlog without manually composing `status=open&unassigned_only=true`
- use the `Eval-backed cases` shortcut on `/admin/cases` when you want the durable follow-up slice created from eval regressions without manually composing `eval_backed_only=true`
- use `Copy case summary` on `/admin/cases` when you need a compact, paste-ready handoff note, `Copy case link` when you want to share the exact filtered case-board URL, and `Open case API detail` when you want the canonical case JSON in a separate tab
- use `Assign case` on `/admin/cases` when you need to put an open follow-up object into a named operator lane before continuing triage or handoff
- use `Add note` on `/admin/cases` when you need to capture operator progress without mutating the case lifecycle
- use `Promote to eval` on `/admin/cases` when you want to turn the current durable case into a durable eval artifact and then jump to the canonical eval-case API detail
- use `Create case` on `/admin/task-board` or `/admin/reports` when you want to promote the currently selected task/report into a durable follow-up object without hand-building the `POST /api/v1/cases` payload
- use `Create case from left` or `Create case from right` on `/admin/eval-report-compare` when a report-vs-report regression needs durable follow-up; the page reuses `POST /api/v1/cases`, anchors the new case to the selected side's report, and then deep-links into `/admin/cases`
- compare-created cases now persist that lineage as `source_eval_report_id`, and `/admin/cases` exposes direct handoff links back into `/admin/eval-reports` and `/api/v1/eval-reports/{report_id}`
- when a selected case carries `source_eval_report_id`, `/admin/cases` now also loads the canonical eval-report detail and shows dataset ID, run status, summary, and bad-case count inline; if that lookup fails, the case detail stays usable and keeps the existing handoff links
- when a case originated from `/admin/eval-report-compare`, `/admin/cases` now also shows the stored compare origin and exposes `Open compare origin`, which deep-links back into the canonical compare page with both report IDs intact
- use the `Compare follow-ups` quick view on `/admin/cases` when you want the durable queue of compare-derived regression cases without hand-composing `compare_origin_only=true`
- use the row-level `Open compare` action inside that `/admin/cases` compare queue when you need to jump straight back to the exact eval-report comparison for one case without opening detail first
- use the row-level `Assign to me` action in that same compare-follow-up queue when you want to claim an unassigned regression case without opening detail first
- successful `report_generation` tasks now finalize the durable report row and task `succeeded` transition together, so `ready_at` and report `metadata.audit_ref` line up with the final task state
- the local Compose app services now start from dedicated runtime images, which removes the previous startup dependence on downloading Go modules inside the running container
- the last successful `audit_event.detail` now carries an execution summary, such as which ticket comment was created
- failed `audit_event.detail` values now carry a coarse category prefix, such as `validation_error:` or `authorization_error:`
- failed tasks expose a summarized `error_reason` instead of the full wrapped Temporal error chain
- SSE always emits `meta`, `plan`, `state`, and `done`
- SSE may also emit `retrieval`, `tool`, and `task_promoted` depending on the internal runtime path
- assistant output is a fixed placeholder response
- the current HTTP contract is documented in `docs/openapi/openapi.yaml`

If your local PostgreSQL volume predates the workflow, report, or case migrations under `db/migrations/`, apply them manually before starting the API:

```powershell
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000002_workflow_tasks.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000003_workflow_task_events.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000004_workflow_task_payload.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000005_reports.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000006_cases.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000007_case_close_fields.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000008_case_assignment_fields.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000009_case_notes.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000010_versions.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000011_version_refs.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000012_eval_cases.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000013_eval_datasets.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000014_eval_dataset_publish_fields.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000015_eval_runs.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000018_eval_run_item_results.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000019_eval_run_item_judge_fields.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000020_eval_reports.sql
docker compose exec -T postgres psql -U opspilot -d opspilot -f /docker-entrypoint-initdb.d/000021_case_eval_report_source.sql
```

If you change Compose environment variables such as `OPSPILOT_POSTGRES_DSN`, `OPSPILOT_TEMPORAL_ENABLED`, or `OPSPILOT_WORKER_POLL_INTERVAL`, recreate the app containers instead of only restarting them:

```powershell
docker compose up -d --build --force-recreate api worker
```

To override the built-in fake ticket API and point both app processes at a different ticket API, recreate them with:

```powershell
$env:OPSPILOT_TICKET_API_BASE_URL = "http://host.docker.internal:19090"
$env:OPSPILOT_TICKET_API_TOKEN = "secret-token"
docker compose up -d --build --force-recreate api worker
```

If an approval-gated task fails after approval, recover it with:

```powershell
$task = Invoke-RestMethod -Method Post -Uri http://localhost:18080/api/v1/tasks/<task_id>/retry -ContentType application/json -Body '{"actor":"operator-1"}'
Invoke-RestMethod -Uri "http://localhost:18080/api/v1/tasks/$($task.task_id)"
```

The expected progression is:
- task status changes from `failed` back to `queued`
- the worker claims it again
- the Temporal run referenced by `audit_ref` changes to a new run ID for the same `task_id`
- `audit_events` grows with `retried`, `claimed`, and the terminal action

To force this path locally without changing code, recreate only the worker with:

```powershell
$env:OPSPILOT_APPROVED_TOOL_FAIL_ON_APPROVE = "true"
docker compose up -d --build --force-recreate worker
```

To force eval-run failures locally without changing code, recreate only the worker with:

```powershell
$env:OPSPILOT_EVAL_RUN_FAIL_ALL = "true"
docker compose up -d --build --force-recreate worker
```

## Current gaps

- In the current Windows shell, `make` may be unavailable; use `scripts/dev/tasks.ps1` as the verified fallback.
- Redis is still present only as future infrastructure; no runtime code path uses it yet.
- The API process still exposes PostgreSQL task rows as the external task-status surface even when Temporal is driving report execution and approval waiting.
- No trace exporter exists yet; only request-scoped IDs are logged.
