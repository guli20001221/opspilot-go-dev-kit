---
name: eval-datasets-regression
description: Create and maintain datasets, judge prompts, regression runs, experiment reports, and failure-case promotion workflows.
---

# eval-datasets-regression

## Goal
Turn prompt, routing, retrieval, and tool changes into measurable, repeatable quality checks.

## Use this skill when
- creating or updating datasets
- building regression jobs
- changing judge rubrics or prompts
- comparing model or prompt versions
- promoting failed production cases into eval coverage

## Inputs to collect first
- quality goals
- representative cases or failures
- scoring dimensions
- prompt or model versions being compared
- report consumers and required drill-down depth

## Likely files and directories
- `eval/datasets/**`
- `eval/prompts/**`
- `eval/reports/**`
- `internal/eval/**`
- `docs/runbooks/eval-operations.md`

## Standard workflow
1. Define the evaluation dimensions before collecting scores.
2. Build or update datasets from representative success and failure cases.
3. Separate production prompts from judge prompts.
4. Make scoring semantics explicit and versioned.
5. Run comparisons across the changed dimension only when possible.
6. Preserve raw judge output and normalized scores.
7. Summarize both top-line metrics and drill-down bad cases.
8. Update baselines when intentional behavior changes are accepted.
9. When promoting production failures from durable operator cases, preserve stable lineage such as `source_case_id`, `source_task_id`, `source_report_id`, `trace_id`, `version_id`, and operator note instead of relying on frontend-only bookmarks.
10. Once durable eval-case promotion exists, prefer a tenant-scoped `GET /api/v1/eval-cases` queue before introducing dataset or regression-run mutation surfaces.
11. The first dataset mutation surface should create a durable draft dataset directly from durable eval cases, preserving explicit membership instead of inferring datasets from tags or bookmarks.
12. Once durable draft datasets exist, add a tenant-scoped `GET /api/v1/eval-datasets` plus a shared `/admin/eval-datasets` lane so operators can revisit drafts without reconstructing them from eval-case bookmarks.
13. Keep dataset list rows lightweight, such as `dataset_id`, `name`, `status`, `created_by`, `updated_at`, and `item_count`; reserve full membership lineage for `GET /api/v1/eval-datasets/{id}`.
14. The first incremental curation surface should be `POST /api/v1/eval-datasets/{dataset_id}/items`, append-only and idempotent for the same eval case, instead of introducing remove/reorder flows too early.
15. Once draft creation and append exist, add an explicit `POST /api/v1/eval-datasets/{dataset_id}/publish` transition that freezes the draft into an immutable baseline before introducing eval-run execution.
16. Once published datasets exist, add a canonical `POST /api/v1/eval-runs` plus tenant-scoped `GET /api/v1/eval-runs` and `GET /api/v1/eval-runs/{id}` so run kickoff becomes durable before judge execution is wired.
17. The first eval-run execution slice should reuse that durable run record and advance it through `queued -> running -> succeeded|failed` with placeholder worker execution before introducing judge prompts, score aggregation, or eval reports.
18. After that first execution slice lands, add `POST /api/v1/eval-runs/{run_id}/retry` as the minimal operator recovery surface so failed runs can be re-queued on the same durable record before per-item scoring or judge prompts exist.
19. Once retry exists on the same durable run record, preserve prior failure context with append-only eval-run events on detail reads instead of introducing a second run-attempt model too early.
20. Before adding judge prompts or per-item scores, snapshot the published dataset membership into durable eval-run items so each run detail can stand on its own lineage without reconstructing membership from dataset state later.
21. Before judge scoring lands, prefer durable placeholder `item_results` on the single-run detail contract, written when the canonical run reaches a terminal state and cleared when retry re-queues that same run.
22. Once durable placeholder `item_results` exist, prefer adding a lightweight terminal-only `result_summary` on canonical eval-run reads so list and operator lanes can scan pass/fail totals without promoting full per-item payloads into every response.
23. When placeholder judge fields become structured, extract them behind a replaceable judge runtime with a stable version ID and prompt artifact path before wiring any external provider.
24. The first provider-backed judge slice should be env-gated and reuse the existing durable `item_results` contract, so local development can stay on the placeholder path until explicit credentials or an HTTP judge service are supplied.
25. When the first external judge call is introduced, preserve a canonical terminal fallback path so provider errors do not strand eval runs in `running`.
26. Once provider-backed judging exists, materialize terminal eval runs into a durable aggregated eval-report artifact before building comparison-heavy UI, so metrics and bad-case references have a canonical backend source of truth.
27. Once durable aggregated eval reports exist, expose canonical `GET /api/v1/eval-reports` and `GET /api/v1/eval-reports/{report_id}` reads before building eval-report-heavy operator views, keeping the list lightweight and the single-report detail as the drill-down surface.
28. Once durable eval reports need side-by-side regression review, prefer a narrow canonical compare contract over two report IDs before building an eval-report comparison page, so score deltas and bad-case overlap stay reproducible and backend-owned.
29. Once a durable eval-report compare lane exists, prefer handing regressions into the canonical case lifecycle through `POST /api/v1/cases` before inventing a separate regression backlog or browser-only follow-up state.
30. If operators need to revisit promoted eval regressions, prefer canonical case filters such as `source_eval_report_id` or `eval_backed_only=true` over duplicating that queue inside eval-only storage.
31. On eval-report detail surfaces, prefer showing linked follow-up cases by querying the canonical case list with `source_eval_report_id` rather than copying case state into eval-report storage.
32. When a durable case detail needs source eval-regression context, prefer reading the canonical eval-report detail by `source_eval_report_id` and keep the case usable if that report row is missing or was only partially recovered.
33. When eval-report list consumers need backlog pressure signals, prefer adding durable follow-up case summary fields such as total/open counts to `GET /api/v1/eval-reports` rather than issuing per-row case-list reads or duplicating case state into eval-report storage.
34. When operators need to isolate unresolved eval regressions, prefer a canonical boolean filter such as `needs_follow_up=true` on `GET /api/v1/eval-reports` before adding any second queue contract.
35. When unresolved regressions need immediate operator handoff, prefer carrying `latest_follow_up_case_id` on eval-report list items so admin lanes can jump directly into the freshest linked case.
36. When one bad case inside a durable eval report needs its own follow-up, prefer anchoring the canonical case to both `source_eval_report_id` and `source_eval_case_id` so precise bad-case triage remains distinct from report-level follow-up.
37. Once canonical eval-case reads carry follow-up summary fields, prefer using those fields as the first eval-triage signal on `/admin/evals` before introducing any eval-only follow-up queue or duplicated case summary store.
38. Once canonical eval-case reads expose `needs_follow_up`, prefer filtering unresolved follow-up through that backend-owned field before introducing any separate eval-follow-up queue contract.
39. Once the canonical case contract accepts standalone `source_eval_case_id`, prefer creating or reusing precise follow-up directly from `/admin/evals` instead of routing every eval-case action through an eval-report-level case flow.
40. Once canonical eval-report detail carries per-bad-case follow-up summary, prefer exposing those fields directly on bad-case drill-down surfaces so operators can reuse existing case lineage instead of opening duplicate follow-up from the same failing eval case.
41. Once canonical eval-report detail carries typed `preferred_primary_action` on each bad case, prefer consuming that backend-owned field for the main bad-case follow-up button instead of mixing follow-up and linked-case heuristics in admin pages.
42. Once canonical eval-report detail supports a `bad_case_needs_follow_up` filter, prefer using that backend-owned slice for unresolved bad-case triage rather than re-implementing follow-up filtering inside admin pages.
43. When operators need unresolved bad-case pressure at eval-report list scope, prefer durable list fields such as `bad_case_without_open_follow_up_count` and a canonical `bad_case_needs_follow_up` filter on `GET /api/v1/eval-reports` before introducing any second unresolved-regression queue.
44. When eval-report comparison needs unresolved bad-case pressure, prefer extending the canonical compare item with `bad_case_without_open_follow_up_count` and hand off into the existing unresolved report slice instead of inventing a compare-only follow-up model.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- datasets are versioned and reproducible
- rubrics and judge prompts are explicit
- reports support comparison and drill-down
- changed behavior has matching regression coverage
- raw judge output is preserved
- runbooks explain how to rerun the evals

## Guardrails
- no silent rubric changes
- no prompt change without considering eval impact
- no report that only shows aggregate pass rate without bad-case evidence
- no hand-wavy claims of quality improvement without comparison data
- when eval-report comparison needs compare-origin follow-up pressure, prefer backend-owned per-side compare queue summary and handoff into canonical case filters instead of reconstructing compare-derived work in the browser
- when compare-created follow-up is retriggered for the same side of the same comparison, prefer backend deduplication on exact compare lineage instead of relying on the page to suppress duplicate operator clicks
