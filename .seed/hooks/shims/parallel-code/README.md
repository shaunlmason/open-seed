# parallel-code (README-only entry)

parallel-code exposes **no usable hook point** (inspirations/07 §9): its
only integration surface is `.claude/steps.json`, an agent-maintained
progress file the watcher reads (host-stamped timestamps, git-excluded).
There is no lifecycle event a repo-shipped fragment could attach to, so
this shim is deliberately README-only — no dead config file ships.

Interop that does work: agents running under parallel-code can keep the
seed checkpoint conventions (mailbox checks, `seed task` verbs) themselves,
and `awaiting_review` in steps.json is the human-gate signal to watch.
