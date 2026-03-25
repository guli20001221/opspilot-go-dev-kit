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
20. The first eval dataset contract should be a canonical `POST /api/v1/eval-datasets` plus `GET /api/v1/eval-datasets/{id}` backed by durable membership rows, not an admin-only batch action.
21. Once durable dataset drafts exist, expose a tenant-scoped `GET /api/v1/eval-datasets` with stable `updated_at DESC, id DESC` ordering and lightweight rows instead of returning full memberships from the list contract.
22. The first dataset-curation mutation should be an explicit `POST /api/v1/eval-datasets/{dataset_id}/items` append contract that is idempotent for the same eval case and returns the updated dataset detail.

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
