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
