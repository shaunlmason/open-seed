# amux shim

amux reads `.amux/workspaces.json` (inspirations/07 §4): `setup-workspace`
runs once at workspace creation, `run` is the toggleable dev command,
`archive` runs best-effort just before deletion.

## Install

Merge `workspaces.json` from this directory into `.amux/workspaces.json`
at the repo root.

**Trust prompts**: amux hash-pins the file (`~/.amux/trusted-scripts.json`,
fail-closed) — any byte change re-gates and the user must re-approve on
next use. Expect a prompt after installing or editing the shim.

## Fidelity

| Contract point | Supported | Via |
|---|---|---|
| setup + post-create.d | yes | `setup-workspace` (runs once at creation) |
| run | yes | `run` |
| teardown | best-effort | `archive` — ≤2 min, failure never blocks deletion, may run twice; make teardown idempotent |
| pre-merge.d (blocking) | **no** | amux has no merge hook; CI is the merge authority (R11) |
