---
name: security-tenancy-audit
description: Enforce RBAC, tenant isolation, secret handling, approval policy, and auditability across the platform.
---

# security-tenancy-audit

## Goal
Keep the agent platform safe for enterprise-style use by making identity, authorization, and audit paths explicit.

## Use this skill when
- adding auth or authorization logic
- touching tenant-scoped storage or queries
- exposing write-capable tools or approval flows
- changing audit logging or secret handling

## Inputs to collect first
- actors, resources, and actions
- tenant model
- sensitive fields and redaction needs
- approval requirements for risky actions
- compliance or audit expectations

## Likely files and directories
- `internal/auth/**`
- `internal/storage/**`
- `internal/tools/**`
- `internal/workflow/**`
- middleware or API boundary packages
- `docs/runbooks/**` for security operations

## Standard workflow
1. Identify actors, resources, and allowed actions.
2. Enforce tenant scope at storage, service, and API boundaries.
3. Apply deny-by-default rules where possible.
4. Redact or hash sensitive values in logs and traces.
5. Persist audit events for approvals, tool calls, and admin actions.
6. Add tests for cross-tenant denial and role-based access.
7. Document operator procedures for approvals and security-sensitive failures.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- tenant scope is enforced at all relevant layers
- RBAC or equivalent authorization is explicit
- sensitive values are protected in logs and traces
- audit records exist for risky actions
- tests cover denial and approval paths
- runbooks describe operator controls

8. When a durable follow-up object such as a case is promoted into eval coverage, validate tenant scope from the canonical source object before copying lineage into the eval record.
9. Operator-facing eval list reads must remain tenant-scoped; do not expose cross-tenant browse surfaces or make `tenant_id` optional on queue endpoints.
10. Eval dataset draft creation must validate that every referenced eval case belongs to the caller's tenant before persisting membership rows.
11. Eval dataset list and detail reads must remain tenant-scoped; require `tenant_id` on browse/detail endpoints and never use dataset membership joins to leak cross-tenant lineage.
12. Eval dataset membership append must validate both tenant scope and mutable dataset state before persisting a new row; treat duplicate adds as idempotent rather than as implicit cross-tenant or server errors.
13. Eval dataset publish must validate both tenant scope and current lifecycle state; published datasets become immutable baselines and repeated publish attempts should fail as explicit invalid-state transitions.
14. Eval run kickoff must validate tenant scope and require a published dataset baseline; do not allow draft datasets or cross-tenant dataset references to create durable run records.
15. Eval run browse and detail reads must remain tenant-scoped; require `tenant_id` on list/detail endpoints and do not expose queued or terminal runs across tenants.

## Guardrails
- no cross-tenant read or write path
- no secret material in source control or tests
- no implicit superuser assumptions
- no side-effecting admin action without auditability
