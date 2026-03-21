---
name: agent-runtime-responses
description: Implement provider adapters, structured planner/tool/critic orchestration, prompt versioning, and the core agent runtime.
---

# agent-runtime-responses

## Goal
Keep the multi-agent runtime explicit, typed, and maintainable while still taking advantage of model capabilities.

## Use this skill when
- adding or changing planner, tool, critic, or response orchestration
- integrating or swapping model providers
- adding structured outputs or tool schemas
- moving prompt logic out of handlers into runtime modules

## Inputs to collect first
- task flow to implement
- tool contracts and risk levels
- expected structured output schemas
- prompt ownership and versioning needs
- eval cases impacted by the change

## Likely files and directories
- `internal/agent/**`
- `internal/model/**`
- `internal/tools/**`
- `eval/prompts/**`
- `internal/eval/**` when behavior changes affect regressions

## Standard workflow
1. Model the task as explicit planner, retrieval, tool, and critic stages.
2. Keep orchestration state in Go structs, not only in prompt text.
3. Define provider interfaces and adapters before adding vendor-specific code.
4. Prefer structured outputs for planner and critic stages.
5. Normalize tool inputs and outputs through typed schemas.
6. Version prompts in code, alongside relevant eval cases.
7. Add unit tests for orchestration decisions and integration tests for happy-path flows.

## Output contract
When you finish, always report:
- summary of implementation
- key decisions
- files changed
- commands run
- risks or follow-ups

## Done checklist
- runtime stages are explicit and typed
- provider-specific code is isolated
- prompts are versioned and discoverable
- tool schemas are normalized
- regression cases exist for changed behavior
- handlers do not contain hidden orchestration logic

## Guardrails
- no magic prompt strings driving critical control flow alone
- no direct vendor SDK leakage across the codebase
- no prompt change without evaluating impact
- no critic stage that cannot explain its decision basis
