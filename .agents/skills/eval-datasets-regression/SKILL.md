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
