## Goal
Move the main `/admin/eval-reports` follow-up button onto one canonical backend-owned action field.

## Scope
- add `preferred_primary_action` to eval-report list/detail reads
- prefer linked-case reuse or queue handoff before new case creation
- update `/admin/eval-reports` row/detail primary actions to consume the new field
- sync OpenAPI, docs, and skills

## Notes
- keep the change additive and backward-compatible
- do not change case dedupe semantics
- preserve existing bad-case and linked-case secondary handoffs
