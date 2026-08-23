# ouijit shim

ouijit hooks are per-project **database records, not files** (inspirations/07
§8): there is no checked-in config a repo can ship. The fragment here is a
one-time registration script instead.

## Install

```sh
sh .seed/hooks/shims/ouijit/register.sh
```

Hook commands run in ouijit PTYs with `OUIJIT_WORKTREE_PATH` et al. in env.

## Fidelity

| Contract point | Supported | Via |
|---|---|---|
| setup | yes | `start` hook |
| run | yes | `run` hook |
| teardown | approximate | `done` hook: fires on task completion, not worktree removal |
| post-create.d | folded into `start` | ouijit has no separate create event |
| pre-merge.d (blocking) | **no** | no merge hook type; CI is the merge authority (R11) |
