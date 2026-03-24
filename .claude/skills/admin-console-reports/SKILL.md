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
30. For high-frequency autonomous-lane triage, prefer adding quick-view presets that write back into the existing `requires_approval` filter over inventing a separate no-approval dashboard.
31. For high-frequency approval-failure triage, prefer composing the existing `status` and `requires_approval` filters in a quick-view preset over inventing a separate failed-approval dashboard.
32. For high-frequency report-output triage, prefer composing the existing `status` and `task_type` filters in a quick-view preset over inventing a separate succeeded-reports dashboard.
33. For early report-focused admin pages, prefer deriving a dedicated `/admin/reports` view from the existing task-board read model and single-task detail contract before introducing a report-specific backend API.
34. When a report lane has both a list and a detail pane, keep the selected report row visually synced with the current detail target and support adjacent navigation within the visible slice before adding a second report queue view.
35. For report-lane monitoring, prefer lightweight polling against the existing admin read model before introducing report-specific watch or subscription contracts.
36. For report-lane handoff, prefer copy actions that derive report links and compact summaries from the existing single-task detail response before introducing report-specific export endpoints or backend formatters.
37. Once successful report tasks persist a durable report entity, prefer wiring report-focused pages to the stable report read endpoint before introducing a separate case or comparison surface.
38. When a report page needs both artifact metadata and execution provenance, prefer reading title/summary/status from the report endpoint and keeping audit timeline or Temporal drill-down on the existing task detail path.
39. Once a durable artifact list endpoint exists, prefer sourcing report-lane tables from that canonical artifact list contract rather than continuing to derive the list from task-board slices.
40. When a report page consumes the canonical artifact list, keep task detail as a provenance drill-down path instead of forcing the artifact list contract to carry workflow audit history.
41. When a stable artifact read endpoint exists, prefer exposing its raw JSON on the page for troubleshooting before inventing an admin-only debug contract.
42. When an operator page depends on a derived artifact row that may be missing for legacy or partially recovered tasks, degrade to the surviving task provenance instead of failing the entire detail panel.
43. Before building a case page, land a durable case contract that can reference existing task and report IDs, so operator handoff is rooted in backend state instead of frontend-only bookmarks.
44. Once a durable case contract exists, prefer adding `Create case` actions to existing task/report detail panes that reuse `POST /api/v1/cases` and deep-link into the case page over inventing admin-only write endpoints.
45. Once durable cases have an explicit lifecycle mutation, prefer wiring close or reopen controls to the canonical case endpoint, such as `POST /api/v1/cases/{case_id}/close`, instead of inventing admin-only write surfaces.
46. For case handoff, prefer copy actions and deep links that reuse the canonical case detail response over inventing a separate case-export contract.
47. Once cases have explicit ownership, prefer wiring assign or claim controls to the canonical case endpoint, such as `POST /api/v1/cases/{case_id}/assign`, instead of inventing admin-only ownership state.
48. Prefer append-only case collaboration through `POST /api/v1/cases/{case_id}/notes` and `GET /api/v1/cases/{case_id}` note reads instead of inventing admin-only comment stores.
49. When cases become operator-owned, prefer queue views built on canonical `GET /api/v1/cases` filters such as `status=open`, `assigned_to=<actor>`, and `unassigned_only=true` instead of frontend-only task lists.
50. When a case page becomes the primary operator queue, prefer promoting current-actor and unassigned slices plus provenance badges in the existing case list over inventing a separate queue-specific backend contract.
51. Once durable report artifacts exist, prefer a narrow read-only compare contract over two report IDs plus a dedicated `/admin/report-compare` page instead of diffing report payloads ad hoc inside the report lane.

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
