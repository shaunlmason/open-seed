# `plans/` — gated work authorizations (D3)

One file per task at `plans/<task-id>.md`, forever (no archival moves). A plan
lands via its own PR from branch `seed/<task-id>-plan`, touching only that one
file; task PRs may not touch `plans/**` at all (the purity rule). The approved
plan for a task PR is **the blob at the PR's merge-base with the default
branch** — the merge-base copy is what gates and CI execute, never the PR
head's.

Grammar: `## Steps`, `## File Scope`, `## Acceptance Criteria`, and
`## Validation Commands` (executed mechanically by the loop runner, the
pre-merge gate, and CI verify).
