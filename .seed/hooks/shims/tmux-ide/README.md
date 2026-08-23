# tmux-ide shim

tmux-ide reads `.tmux-ide/workspace.yml` (inspirations/07 §7): `before` is
a pre-launch shell hook; panes carry their own commands.

## Install

Merge `workspace.yml` from this directory into `.tmux-ide/workspace.yml`
(or start from it) and adjust pane layout to taste.

## Fidelity

| Contract point | Supported | Via |
|---|---|---|
| setup | yes | `before` (pre-launch, once per session start) |
| run | yes | a pane whose `command` is the run hook |
| post-create.d | no | tmux-ide manages sessions, not worktrees, no create event |
| teardown | no | no session-end hook surface |
| pre-merge.d (blocking) | **no** | no merge concept; CI is the merge authority (R11) |

The agent-status protocol (`@agent_state` pane option) is an observation
surface, not a hook point.
