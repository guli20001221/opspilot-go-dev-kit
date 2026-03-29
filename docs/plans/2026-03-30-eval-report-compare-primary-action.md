## Goal
Move the main `/admin/eval-report-compare` left/right follow-up buttons onto one canonical backend-owned action field per side.

## Scope
- add per-side `preferred_primary_action` to the compare read contract
- prefer linked-case reuse or linked-case queue before compare-queue or create
- update `/admin/eval-report-compare` to consume the new field for its main buttons
- keep compare-queue links as secondary handoff
- sync OpenAPI, docs, and skills

## Notes
- additive only; no endpoint changes
- do not change compare-origin case dedupe semantics
