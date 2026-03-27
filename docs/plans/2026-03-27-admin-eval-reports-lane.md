# 2026-03-27 Admin Eval Reports Lane

## Goal
Add a dedicated operator page for durable eval-report artifacts without changing backend contracts.

## Scope
- Add embedded `/admin/eval-reports`
- Read list from `GET /api/v1/eval-reports`
- Read detail from `GET /api/v1/eval-reports/{report_id}`
- Keep drill-down contract-first:
  - bad cases
  - metadata
  - raw JSON
  - handoff to eval runs, datasets, evals, trace, and version pages

## Non-goals
- No new eval-report write path
- No admin-only backend API
- No compare or case mutations in this slice

## Notes
- The list must stay lightweight.
- Heavy fields stay in the detail request.
- Detail navigation should stay inside the current visible slice.
