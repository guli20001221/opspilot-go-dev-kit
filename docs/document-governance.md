# Document Governance

This document defines the single-source-of-truth order for repository guidance and planning artifacts in OpsPilot-Go.

## Purpose

Use this file to answer four questions quickly:

1. Which document defines repository-wide rules?
2. Which document is authoritative for implementation details?
3. Which document explains the project at the blueprint level?
4. Which document should an agent or developer consult for a task-specific workflow?

## Authority Order

When documents overlap or conflict, resolve them in this order:

1. `AGENTS.md`
2. `implementation-spec.md`
3. `plan.md`
4. `.claude/skills/*/SKILL.md`
5. path-local `AGENTS.override.md`

Notes:
- `AGENTS.override.md` only applies within its local path scope.
- A local override may refine or tighten repository guidance for that path, but it must not contradict repository-wide policy.
- Skills define execution workflow for a task area; they do not redefine system truth.

## Source Matrix

| Source | Role | Owns | Does not own |
|---|---|---|---|
| `AGENTS.md` | Repository-wide policy layer | engineering rules, architecture guardrails, delivery protocol, skill routing, definition of done | endpoint field lists, DTO truth, table-level implementation detail |
| `implementation-spec.md` | Canonical implementation spec | module contracts, state machines, DTOs, DDL drafts, milestone implementation order, acceptance criteria | business narrative, interview packaging, broad project storytelling |
| `plan.md` | High-level blueprint | background, goals, scope, end-to-end system framing, milestone narrative, risk framing, reporting language | field-level implementation truth, final API schemas, low-level module contracts |
| `.claude/skills/*/SKILL.md` | Task playbooks | workflow for a task type, required inputs, sequencing, guardrails, completion checklist | project-wide policy, shared canonical contracts, cross-system source of truth |
| `AGENTS.override.md` | Path-local refinement | tighter guidance for a subsystem directory | repository-wide policy replacement |

## Practical Usage Order

Use the documentation stack in this order during implementation:

1. Read `AGENTS.md` for repository-wide constraints.
2. Read `implementation-spec.md` for concrete implementation truth.
3. Read `plan.md` for high-level intent, milestones, and presentation context.
4. Read the matching `.claude/skills/*/SKILL.md` before changing a subsystem.
5. Read the nearest `AGENTS.override.md` when working inside a scoped directory.

## Conflict Rules

### `AGENTS.md` vs other documents

`AGENTS.md` wins on repository-wide engineering policy, delivery protocol, and architectural guardrails.

### `implementation-spec.md` vs `plan.md`

`implementation-spec.md` wins on:
- module boundaries
- input and output contracts
- state machines
- DTO and field definitions
- implementation sequencing
- acceptance criteria

`plan.md` remains the source for:
- background and rationale
- high-level goals and non-goals
- milestone framing
- risk framing
- demo, resume, and defense packaging

### Skills vs top-level documents

Skills must follow the top-level sources. If a skill workflow drifts from `AGENTS.md` or `implementation-spec.md`, update the skill rather than treating the skill as authoritative.

### Path-local overrides

`AGENTS.override.md` applies only to its directory subtree. It is intended to make local constraints more explicit, not to create a second architecture.

## Recommended Wording for `plan.md`

To avoid ambiguity, treat the following interpretation as canonical:

> `plan.md` defines project blueprint, scope, and phase goals. Concrete interfaces, state machines, field definitions, and implementation contracts are governed by `AGENTS.md` and `implementation-spec.md`.

If `plan.md` is edited later, keep that distinction explicit.

## Change Management

Update this file when any of the following changes:
- a new top-level planning or governance document is introduced
- authority order changes
- a new local override zone is added with special handling rules
- skills are reorganized in a way that changes how contributors should route work

When updating this file, also check:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/architecture.md`
- `docs/skills/README.md`

## Summary

Use the stack like this:

- `AGENTS.md` says what is allowed and required.
- `implementation-spec.md` says what to build and how it is shaped.
- `plan.md` says why the project exists and how to explain it.
- `SKILL.md` says how to execute a class of work safely.
- `AGENTS.override.md` says what is stricter in one subtree.
