# 2026-03-31 Eval Run Row Primary Actions

## Goal
Promote the main row-level `/admin/eval-runs` handoff into a backend-owned contract.

## Scope
- add typed `preferred_primary_action` to canonical `GET /api/v1/eval-runs` rows
- reuse the same field on `GET /api/v1/eval-runs/{run_id}` for consistency
- wire `/admin/eval-runs` row-level primary button from that field
- keep per-item and per-result follow-up contracts unchanged

## Rules
- prefer linked-case reuse before report review
- prefer durable report review before falling back to plain run detail
- keep routing logic in backend read-model assembly, not browser heuristics
