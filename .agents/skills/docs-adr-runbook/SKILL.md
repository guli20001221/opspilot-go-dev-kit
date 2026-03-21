---
name: docs-adr-runbook
description: Maintain README, ADRs, architecture docs, setup guides, and runbooks in lockstep with code changes.
---

# docs-adr-runbook

## Goal
Keep project knowledge explicit, current, and useful to both humans and coding agents.

## Use this skill when
- setup commands change
- architecture evolves
- a major dependency or subsystem is introduced
- operational procedures or recovery flows change
- project structure is reorganized

## Inputs to collect first
- code changes made
- actual commands run and validated
- decisions that deserve an ADR
- operator tasks that need a runbook
- diagrams or flows that need explanation

## Likely files and directories
- `README.md`
- `docs/adr/**`
- `docs/runbooks/**`
- `docs/architecture.md`
- relevant skill files when workflows change

## Standard workflow
1. Diff the user-facing or operator-facing behavior against the existing docs.
2. Update setup commands, environment expectations, and architecture notes.
3. Write ADRs for non-trivial decisions.
4. Write or update runbooks for workflows that operators must execute.
5. Keep examples and command snippets faithful to what was actually run.
6. Update related skill guidance if the workflow changed.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- docs reflect current reality
- setup commands are tested
- significant decisions have ADRs
- operators have runbooks for critical flows
- diagrams and tree docs match the repository
- related skill guidance stays in sync

## Guardrails
- do not claim unsupported capabilities
- do not leave stale commands in README or runbooks
- do not create ADRs with no decision or rationale
- do not skip docs for changes that alter workflow or operator behavior
