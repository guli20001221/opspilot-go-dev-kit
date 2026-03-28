# 2026-03-28 Case Unassign Action

## Goal
Add a canonical unassign action so `/admin/cases` can move claimed work back into the shared unassigned queue without abusing the assign contract.

## Scope
- add `POST /api/v1/cases/{case_id}/unassign`
- support optimistic-concurrency unassign in the case service and stores
- expose row-level and detail-level `Return to queue` on `/admin/cases`
- cover unassign in service, HTTP, store, and runtime smoke tests

## Non-goals
- new queue endpoints
- extra case lifecycle states
- admin-only write surfaces for ownership release
