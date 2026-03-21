# AGENTS.override.md

Local scope: `internal/eval/**`

Priorities here:
- reproducible datasets
- explicit rubrics
- versioned judge prompts
- comparable reports across model, prompt, and tool versions

Extra rules:
- do not silently change scoring semantics
- every rubric or judge update needs example cases
- preserve raw judge outputs alongside normalized scores
- report both pass/fail summaries and drill-down artifacts
