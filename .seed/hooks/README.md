# `.seed/hooks/`: the worktree lifecycle contract (D6)

Runner-agnostic hooks any orchestrator (or a bare human) invokes at the same
points. Run them via `seed hooks run <name>` (portable fallback, R2) or
directly. All hooks run with the worktree as cwd.

| Hook | When | Contract |
|---|---|---|
| `setup` | once per clone | install deps, prepare the machine |
| `post-create.d/*` | after a task worktree is created | run in lexical order; propagate `.worktreeinclude` files, install deps |
| `run` | to start an agent session in a worktree | optional convenience entry |
| `pre-merge.d/*` | before merging a task branch | **blocking**: any non-zero exit stops the merge (the local gate: a fast pre-check; CI is the authority, R11) |
| `teardown` | before a worktree is removed | flush state, stop processes |

Keep hooks fast and deterministic. They are user-editable convention, not
control logic: the port and CI gates do the enforcing.

Portable context: runners and shims export what they know of
`SEED_WORKTREE`, `SEED_REPO_ROOT`, `SEED_BRANCH`, `SEED_TARGET_BRANCH`
(merge hooks), `SEED_TASK`, `SEED_TASK_TITLE`, `SEED_TASK_DESCRIPTION`,
and `SEED_PORT` before invoking a hook, so a hook reads one env surface
under every tool. Absent values stay unset: hooks must tolerate that.
(`scripts/loop.sh` additionally exports `SEED_MAIN_CHECKOUT`, its
pre-shim name for the main checkout path.)

## Tool shims (`shims/`)

Checked-in integration fragments wiring external worktree tools onto this
contract: one directory per tool, each with a README stating install
steps and a fidelity table (which contract points the tool can honor,
which it cannot: declared, never silent). A tool with no usable hook
point gets a README-only entry instead of a dead file.

| Shim | Post-create | Teardown | Blocking pre-merge |
|---|---|---|---|
| `shims/octomux/` | yes | approximate | no (fire-and-forget hooks) |
| `shims/amux/` | yes | best-effort | no |
| `shims/dmux/` | yes | yes | **no: detached hooks; declared caveat** |
| `shims/tmux-ide/` | no | no | no |
| `shims/ouijit/` | via `start` | approximate | no |
| `shims/parallel-code/` | README-only: no hook surface | n/a | n/a |

v1 guidance for superset, agent-deck, and vibe-tree lives in the handbook's
lifecycle section. On every tool the rule is the same: local hooks are
convenience, the CI verify gate is the merge authority (R11).
