---
id: os-a7b5dffb
title: 'shims: a herdr shim for the v1 template (.seed/hooks/shims/herdr/, protected surface)'
state: backlog
priority: P2
squad: core
created_at: "2026-09-05T18:48:19Z"
---

docs/next-build-plan.md §3 "Borrowed from surveyed tools", card 4. v1 work, not next/**: .seed/** is protected surface, so this lands under ordinary protected-path review. Contents per the shim convention (.seed/hooks/README.md): a README with the fidelity table (setup yes via a workspace-create script; run yes via `herdr agent start`; post-create and teardown no, herdr manages panes not worktrees; blocking pre-merge no, CI remains the merge authority per R11), a sample launch script opening one pane per lane running scripts/loop.sh with a distinct --actor, and a `seed mail nudge` path for herdr beside the tmux-only one (content-free; the message stays in the mail file). Source: github.com/herdrdev/herdr. Above L1: plan first.
