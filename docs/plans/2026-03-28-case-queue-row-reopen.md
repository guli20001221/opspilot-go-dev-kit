# 2026-03-28 Case Queue Row Reopen

## Goal
Add a row-level reopen action to `/admin/cases` so operators can return closed follow-up work to the open queue without opening the detail pane first.

## Scope
- reuse the canonical `POST /api/v1/cases/{case_id}/reopen` endpoint
- expose `Reopen from queue` on closed case rows
- keep queue state URL-driven and contract-first
- cover the workflow in the existing cases page runtime smoke test

## Non-goals
- new case queue endpoints
- extra reopen-specific admin APIs
- changes to case lifecycle semantics
