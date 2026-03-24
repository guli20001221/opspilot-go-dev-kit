# 2026-03-24 Case Unassigned Queue

## Goal

Extend the case queue slice so operators can jump directly from owned work to the shared unassigned backlog without inventing a separate frontend-only queue model.

## Scope

- add `unassigned_only` filtering to `GET /api/v1/cases`
- preserve existing `assigned_to` and `status` queue filters
- add an `Unassigned` quick view to `/admin/cases`
- keep the browser thin and reuse the canonical case list contract

## Validation

- targeted Go tests for service, PostgreSQL store, and HTTP endpoint filtering
- HTML render assertion for the new quick view
- browser smoke on `/admin/cases` confirming `Unassigned` narrows the queue slice and updates detail selection
