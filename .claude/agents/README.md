# Claude subagents (recommended later)

Suggested future subagents for this repository:

- `architecture-reviewer`
  - reviews module boundaries, package cohesion, interface placement, and orchestration clarity
- `security-reviewer`
  - reviews tenant isolation, approvals, auditability, secret handling, and unsafe write paths
- `eval-judge`
  - independently reviews answer quality, citation completeness, and rubric alignment

Keep subagents read-only by default unless there is a strong reason to grant write access.
Use them for review, challenge, or arbitration tasks rather than mainline implementation.
