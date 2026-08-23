# octomux shim

octomux's `.d` hooks are **task-lifecycle events, not worktree-lifecycle
events** (survey correction, inspirations/07 §2). Its full surface is seven
task events; exactly two are mappable onto the seed contract, and this shim
maps only those: the rest of the contract is a declared gap, not an
invented mapping.

## Install

Copy the two event dirs into the repo-local hook home (or symlink them):

```sh
mkdir -p .octomux/hooks
cp -R .seed/hooks/shims/octomux/task_created.d .octomux/hooks/
cp -R .seed/hooks/shims/octomux/runtime_state_changed.d .octomux/hooks/
chmod +x .octomux/hooks/*.d/*
```

Scripts receive the `HookEnvelope` JSON (`{event, task, data}`) on stdin,
run with cwd = the task worktree when set, and are **advisory**: octomux
logs non-zero exits but never blocks on them.

## Fidelity

| Contract point | Supported | Via |
|---|---|---|
| post-create.d | yes | `task_created` (cwd = task worktree) |
| setup / teardown | approximate | `runtime_state_changed` dispatch: teardown fires on terminal runtime states, not on worktree removal |
| run | no | octomux owns the runtime; no hookable start point |
| pre-merge.d (blocking) | **no** | all octomux hooks are fire-and-forget; CI is the merge authority (R11) |

Loop budgets (`.octomux/loop-status.json`) are octomux-internal state, not
a hook surface.
