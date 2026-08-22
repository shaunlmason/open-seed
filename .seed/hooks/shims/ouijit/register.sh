#!/bin/sh
# One-time ouijit hook registration (hooks are DB records, not files).
set -eu
command -v ouijit >/dev/null 2>&1 || { echo "ouijit not on PATH"; exit 1; }
ouijit hook set start --name seed-setup \
  --command 'cd "$OUIJIT_WORKTREE_PATH" && [ -x .seed/hooks/setup ] && .seed/hooks/setup; for h in .seed/hooks/post-create.d/*; do [ -x "$h" ] && "$h"; done'
ouijit hook set run --name seed-run \
  --command 'cd "$OUIJIT_WORKTREE_PATH" && [ -x .seed/hooks/run ] && .seed/hooks/run'
ouijit hook set done --name seed-teardown \
  --command 'cd "$OUIJIT_WORKTREE_PATH" && [ -x .seed/hooks/teardown ] && .seed/hooks/teardown'
echo "seed hooks registered with ouijit (start/run/done)"
