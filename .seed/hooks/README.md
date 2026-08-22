# `.seed/hooks/` — the worktree lifecycle contract (D6)

Runner-agnostic hooks any orchestrator (or a bare human) invokes at the same
points. Run them via `seed hooks run <name>` (portable fallback, R2) or
directly. All hooks run with the worktree as cwd.

| Hook | When | Contract |
|---|---|---|
| `setup` | once per clone | install deps, prepare the machine |
| `post-create.d/*` | after a task worktree is created | run in lexical order; propagate `.worktreeinclude` files, install deps |
| `run` | to start an agent session in a worktree | optional convenience entry |
| `pre-merge.d/*` | before merging a task branch | **blocking**: any non-zero exit stops the merge (the local gate — a fast pre-check; CI is the authority, R11) |
| `teardown` | before a worktree is removed | flush state, stop processes |

Keep hooks fast and deterministic. They are user-editable convention, not
control logic — the port and CI gates do the enforcing.
