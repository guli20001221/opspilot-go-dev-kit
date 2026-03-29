## Goal
Move the main `/admin/evals` follow-up button onto one canonical backend-owned action field.

## Scope
- add `preferred_primary_action` to eval-case list/detail reads
- prefer linked-case reuse or queue handoff before new case creation
- update `/admin/evals` row/detail primary actions to consume the new field
- sync OpenAPI, docs, and skills

## Notes
- additive only; no endpoint changes
- keep existing linked-case handoff as a secondary operator control
