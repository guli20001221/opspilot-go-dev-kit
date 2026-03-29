---
name: api-contract-sse
description: Define and implement REST contracts, SSE streaming endpoints, middleware, error envelopes, and OpenAPI updates.
---

# api-contract-sse

## Goal
Keep the HTTP surface explicit, documented, stable, and friendly to streaming agent interactions.

## Use this skill when
- adding or changing public or internal HTTP endpoints
- implementing SSE response streams
- changing auth middleware, request validation, or error handling
- updating OpenAPI or handler contracts

## Inputs to collect first
- endpoint purpose and consumers
- request and response schemas
- auth and tenancy requirements
- whether the endpoint is synchronous, asynchronous, or streaming
- expected error cases and retry semantics

## Likely files and directories
- `cmd/api/**`
- `internal/app/http/**` or equivalent API packages
- `pkg/apierror/**`
- `docs/openapi/**` or generated spec targets
- `README.md` or API docs if commands change

## Standard workflow
1. Define the contract first: method, path, request, response, error envelope, and auth.
2. For long-running user-visible responses, prefer SSE.
3. Standardize correlation ids and machine-readable error codes.
4. Keep handlers thin; move domain logic to services.
5. Add request validation and tenancy enforcement at the boundary.
6. Update OpenAPI and handler tests together.
7. If asynchronous job creation is involved, return stable job ids and status endpoints.
8. If sync flows can promote work into async tasks, keep the task id stable across SSE events and REST lookup endpoints.
9. Task action endpoints such as approve or retry should surface invalid state transitions as explicit 409-style contract errors.
10. List endpoints for async jobs should document supported filters, including operator-centric booleans, reason enums, or created/updated time-window parameters when relevant, plus pagination semantics, and keep heavy per-item detail, such as audit history, off the summary response unless explicitly needed.
11. When async workflow execution produces a durable artifact such as a report, expose a stable artifact read endpoint separate from task status instead of forcing clients to reconstruct artifact metadata from task audit events.
12. When a durable follow-up object such as a case links existing tasks or reports, validate tenant-safe source references at the API boundary and surface lineage mismatches as explicit 409-style contract errors.
13. When exposing lifecycle actions for durable follow-up objects such as cases, keep each transition explicit as its own endpoint, return the updated object, and surface invalid state transitions as explicit 409-style contract errors.
14. When operators need to compare two durable artifacts such as reports, prefer a read-only compare endpoint with explicit left/right IDs and a typed summary instead of pushing diff logic into the client.
15. When multiple operator pages need the same provenance context, prefer one read-only trace-drilldown endpoint with explicit lookup keys instead of duplicating trace-resolution logic across several page-specific contracts.
16. When reproducibility depends on runtime bundle metadata, expose a stable version-registry read contract, such as `GET /api/v1/versions` and `GET /api/v1/versions/{version_id}`, instead of forcing clients to reconstruct planner, retrieval, tool, or workflow versions from task payloads.
17. Once a durable version registry exists, propagate `version_id` through related task, report, compare, or trace read contracts instead of duplicating full runtime-version payloads on every endpoint.
18. When operators promote a durable case into eval coverage, prefer a canonical `POST /api/v1/eval-cases` plus `GET /api/v1/eval-cases/{id}` contract that copies lineage from existing case, task, report, trace, and version state instead of storing frontend-only bookmarks.
19. When operators need to browse promoted eval coverage, expose a tenant-scoped `GET /api/v1/eval-cases` with the same `limit/offset/has_more/next_offset` pagination style used elsewhere instead of inventing an eval-only paging model.
20. When operator UIs need to decide between `create` and `open existing` follow-up actions for a durable eval artifact, prefer a typed backend-owned read-model field such as `preferred_follow_up_action` over re-deriving that decision from counts and IDs in the browser.
21. The first eval dataset contract should be a canonical `POST /api/v1/eval-datasets` plus `GET /api/v1/eval-datasets/{id}` backed by durable membership rows, not an admin-only batch action.
22. Once durable dataset drafts exist, expose a tenant-scoped `GET /api/v1/eval-datasets` with stable `updated_at DESC, id DESC` ordering and lightweight rows instead of returning full memberships from the list contract.
23. The first dataset-curation mutation should be an explicit `POST /api/v1/eval-datasets/{dataset_id}/items` append contract that is idempotent for the same eval case and returns the updated dataset detail.
24. Once dataset drafts can be curated incrementally, add an explicit `POST /api/v1/eval-datasets/{dataset_id}/publish` contract that returns the updated dataset detail and surfaces repeated publish attempts as a typed 409-style invalid-state error.
25. Once datasets can be published, add a canonical `POST /api/v1/eval-runs`, tenant-scoped `GET /api/v1/eval-runs`, and `GET /api/v1/eval-runs/{run_id}` contract that snapshots published dataset metadata into a durable queued run before execution is wired.
26. The first eval-run execution slice should keep `started_at`, `finished_at`, `status`, and `error_reason` on that same canonical run detail contract instead of inventing a second execution-status surface.
27. Recovery on eval runs should reuse that same canonical record; prefer `POST /api/v1/eval-runs/{run_id}/retry` with a typed 409 invalid-state error over creating a separate rerun resource or admin-only mutation path.
28. When retry on a durable eval run clears top-level failure fields, extend only the single-run detail contract with append-only lifecycle events; keep the list contract lightweight.
29. When per-run judging has not landed yet, prefer adding immutable `items` only on the single-run detail contract, copied from published dataset membership at kickoff time, and keep create/list/retry responses lightweight snapshots.
30. Before per-item judging lands, prefer adding durable placeholder `item_results` only on the single-run detail contract, keep create/list/retry responses lightweight, and clear stale item results when retry re-queues the same canonical run.
31. Once placeholder `item_results` exist, prefer a lightweight `result_summary` on terminal eval-run reads for operator scanning instead of leaking full per-item payloads into list, create, or retry responses.
32. Before a real judge provider is wired, keep placeholder eval item results structured: expose normalized verdict/score fields plus raw judge output on detail reads so later provider integration stays backward-compatible.
33. When durable follow-up objects such as cases link eval-run lineage, expose `source_eval_run_id` on the canonical case read and write contracts and prefer backend-owned queue filters such as `source_eval_run_id` or `run_backed_only=true` over browser-built run-follow-up slices.
34. Once `source_eval_run_id` is part of the canonical case contract, prefer `POST /api/v1/cases` to reuse the newest open `tenant_id + source_eval_run_id` case instead of creating duplicate run-backed follow-up rows from repeated operator clicks.
35. Once canonical eval-dataset reads already expose both latest-report and dataset-wide case queue actions, add a typed preferred case handoff field on that same contract instead of leaving the browser to compose dataset-versus-report queue priority on its own.
36. Once canonical eval-dataset reads already know the latest durable run ID, expose a lightweight run-backed case summary there instead of forcing operators to query `/api/v1/eval-runs` or `/api/v1/cases` just to see whether the latest run already has claimed follow-up work.
37. Once canonical eval-dataset reads already expose run-backed case summary for the latest durable run, add a typed preferred run-backed case handoff action on that same contract instead of leaving the browser to decide between an existing case and the open run-backed queue.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- contract is explicit and documented
- handlers are thin and testable
- SSE event format is consistent
- auth and tenancy checks are enforced
- error responses are uniform
- OpenAPI or equivalent docs are updated

## Guardrails
- no hidden breaking changes to consumed endpoints
- no business logic embedded in handlers
- no streaming format invented ad hoc per endpoint
- do not return internal stack traces to clients
- once canonical eval-dataset reads expose run-backed case summary, prefer adding a typed run-backed case handoff field on that same contract instead of leaving `/admin/eval-datasets` to route straight from `latest_case_id`
- once canonical eval-dataset detail already exposes `recent_runs[]` with unresolved pressure and report linkage, add a typed `preferred_follow_up_action` on those same rows instead of leaving `/admin/eval-datasets` to decide between report and run queues from `report_id` plus `needs_follow_up`
