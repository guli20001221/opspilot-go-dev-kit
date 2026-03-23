---
name: admin-console-reports
description: Build the admin console for tasks, cases, reports, versions, and trace drill-down without leaking business logic into the frontend.
---

# admin-console-reports

## Goal
Provide a practical operator UI for evaluation and runtime analysis while keeping backend contracts authoritative.

## Use this skill when
- building or changing admin pages
- adding task management, case management, report views, or version comparison UI
- wiring trace deep links and report drill-down

## Inputs to collect first
- backend contracts and page goals
- required filters and drill-down flows
- operator personas
- reproducibility and audit data needed on-screen

## Likely files and directories
- `web/admin/**`
- API client packages
- related backend endpoints when contract gaps are discovered
- `docs/architecture.md` or frontend docs if the page model changes

## Standard workflow
1. Start from stable backend contracts.
2. Design the minimum page flow for operators: tasks, cases, reports, version comparison, trace links.
3. Prefer backend read models, such as task-board aggregations or `/api/v1/admin/task-board`, over recomputing status summaries in the UI.
4. For early slices, prefer embedded pages under `web/admin` or similarly low-overhead delivery before introducing a separate frontend toolchain.
5. Keep frontend state simple and derived from backend data where possible.
6. Surface reproducibility data such as prompt version, model version, dataset id, and trace ids.
7. Add empty, loading, and failure states.
8. Prefer drill-down that reuses existing detail endpoints over inventing parallel admin-only detail contracts.
9. Prefer operator actions that call existing workflow/task endpoints over introducing admin-only write surfaces.
10. If contract changes are necessary, change backend and docs first, then UI.
11. When execution provenance already exists in fields such as `audit_ref`, prefer deriving trace or workflow deep links in the UI over adding redundant backend fields.
12. For operator monitoring, prefer lightweight polling against existing read models before inventing new backend watch or subscription contracts.
13. For high-frequency operator slices, prefer transparent quick-view presets that write back into the existing filter model over a separate frontend-only query state.
14. When operators need payload-level troubleshooting, prefer exposing the existing detail response as raw JSON over inventing a second admin-only debug contract.
15. For handoff flows, prefer links and copy actions that point back to existing board or detail contracts over generating separate export endpoints.
16. When adding operator handoff summaries, derive them from the current detail response and timeline data already on-screen before considering any new backend formatter.
17. When operators triage multiple items in the same slice, prefer next/previous navigation within the current visible board results over adding a second list or modal stack.
18. When execution or audit detail already exists in the task timeline, promote it into a compact digest on the page before adding new backend aggregation fields.
19. When a selected item should drive related-list triage, prefer writing its attributes back into the existing filter model over inventing a second "similar items" query state.
20. When detail drill-down and list navigation coexist on one page, keep the selected list row visually synced with the current detail target so operators do not lose table context.
21. When operators need to pivot from one task into a broader state-based queue, prefer reusing the existing status filter and URL state over adding a separate queue view contract.
22. When operators need to pivot from one task into a broader reason-based slice, prefer reusing the existing reason filter and URL state over adding a separate reason dashboard or frontend-only query state.
23. When operators need to pivot from one task into a broader approval lane, prefer reusing the existing `requires_approval` filter and URL state over adding a separate approval queue dashboard.
24. When operators need to pivot from one task into a broader task-type slice, prefer reusing the existing `task_type` filter and URL state over adding a separate report-only or tool-only queue dashboard.
25. When operators need to pivot from one task into a broader operational queue, prefer composing the existing `status` and `requires_approval` filters over inventing a separate queue-specific backend contract.
26. For high-frequency task-type triage, prefer adding quick-view presets that write back into the existing `task_type` filter over introducing a second frontend-only browse mode.
27. For high-frequency terminal-state triage, prefer adding quick-view presets that write back into the existing `status` filter over inventing a separate success-only dashboard.
28. For high-frequency pending-work triage, prefer adding quick-view presets that write back into the existing `status` filter over inventing a separate queued-work dashboard.
29. For high-frequency routing triage, prefer adding quick-view presets that write back into the existing `reason` filter over inventing a separate routing dashboard.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- UI matches backend contracts
- page flows for tasks, cases, and reports are coherent
- reproducibility metadata is visible
- trace drill-down paths exist where useful
- no business logic is stranded in the frontend
- docs or screenshots are updated when needed

## Guardrails
- do not invent API fields in the frontend
- do not move domain logic into the UI for speed
- do not hide failure reasons behind generic toasts only
- do not build polished dashboards before core operator workflows work
