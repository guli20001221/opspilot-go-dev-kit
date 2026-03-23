---
name: postgres-sqlc-pgvector
description: Design schemas, migrations, sqlc queries, pgx integration, and pgvector-backed retrieval storage.
---

# postgres-sqlc-pgvector

## Goal
Model operational, retrieval, and audit data in PostgreSQL using additive migrations and type-safe SQL access from Go.

## Use this skill when
- creating or changing schema
- adding migrations
- adding or changing sqlc queries
- wiring pgx pools or repository adapters
- building retrieval tables, chunk metadata, or vector indexes

## Inputs to collect first
- domain entities and relationships
- tenancy and audit requirements
- expected read and write patterns
- retention needs
- retrieval fields needed for provenance and filtering

## Likely files and directories
- `db/migrations/**`
- `db/queries/**`
- `sqlc.yaml`
- `internal/storage/**`
- `internal/retrieval/**`
- `docs/adr/**` when schema decisions are significant

## Standard workflow
1. Identify which tables are operational, eval-related, or audit-related.
2. Model the schema with tenant scope, timestamps, and provenance in mind.
3. Write additive migrations first.
4. Write SQL queries before repository wrappers.
5. Generate type-safe code with sqlc and adapt it behind storage interfaces.
6. Add indexes only after understanding query paths.
7. For vector-backed tables, preserve source metadata, chunk ids, versions, and permission scope.
8. Add integration tests for the query behavior that matters.
9. Workflow task tables should carry tenant scope, status, timestamps, and operator-facing failure fields from the start.
10. Audit-heavy workflow features should preserve structured event history, not only the latest summary field.
11. If a workflow write changes task state and appends a task event, prefer one DB transaction so operator state and audit history cannot diverge.
12. Workflow-generated durable artifacts such as reports should get their own table keyed by source task identity instead of being flattened into the task-status table.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- migrations are additive and reversible where practical
- sqlc queries compile and produce typed accessors
- repositories expose clear domain-shaped methods
- retrieval tables preserve provenance and tenant scope
- integration tests cover critical queries
- documentation notes any irreversible migration risks

## Guardrails
- no ORM introduction without approval
- no destructive migration without rollback plan
- no hidden business logic in SQL scripts
- no vector row without provenance metadata
- no cross-tenant query paths
