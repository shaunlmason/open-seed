# dmux shim

dmux resolves single executable files from `.dmux-hooks/<name>`
(version-controlled, highest priority: inspirations/07 §5).

## Install

```sh
mkdir -p .dmux-hooks
cp .seed/hooks/shims/dmux/worktree_created .dmux-hooks/
cp .seed/hooks/shims/dmux/before_worktree_remove .dmux-hooks/
cp .seed/hooks/shims/dmux/pre_merge .dmux-hooks/
chmod +x .dmux-hooks/worktree_created .dmux-hooks/before_worktree_remove .dmux-hooks/pre_merge
```

## Fidelity

| Contract point | Supported | Via |
|---|---|---|
| post-create.d | yes | `worktree_created` (cd `$DMUX_WORKTREE_PATH`) |
| teardown | yes | `before_worktree_remove` (worktree still exists) |
| pre-merge.d (blocking) | **declared caveat** | `pre_merge` runs the gates, but dmux spawns every hook detached and non-blocking: "hook errors are logged but don't stop dmux". **dmux cannot veto a merge**; the CI verify gate is the real backstop (R11). |
| setup / run | no | dmux's `run_test`/`run_dev` report over HTTP, not exit codes, no clean mapping |
