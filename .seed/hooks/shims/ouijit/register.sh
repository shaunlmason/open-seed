#!/bin/sh
# One-time ouijit hook registration (hooks are DB records, not files).
# Each command translates the OUIJIT_* context into the portable SEED_*
# variables and treats an absent optional hook as a successful no-op.
set -eu
command -v ouijit >/dev/null 2>&1 || { echo "ouijit not on PATH"; exit 1; }
seed_env='export SEED_WORKTREE="$OUIJIT_WORKTREE_PATH" SEED_REPO_ROOT="$OUIJIT_PROJECT_PATH" SEED_BRANCH="$OUIJIT_TASK_BRANCH" SEED_TASK_TITLE="$OUIJIT_TASK_NAME" SEED_TASK_DESCRIPTION="$OUIJIT_TASK_DESCRIPTION"'
ouijit hook set start --name seed-setup \
  --command "$seed_env"'; cd "$OUIJIT_WORKTREE_PATH" && { if [ -x .seed/hooks/setup ]; then .seed/hooks/setup; fi; for h in .seed/hooks/post-create.d/*; do [ -x "$h" ] && "$h"; done; :; }'
ouijit hook set run --name seed-run \
  --command "$seed_env"'; cd "$OUIJIT_WORKTREE_PATH" && { if [ -x .seed/hooks/run ]; then .seed/hooks/run; fi; }'
ouijit hook set done --name seed-teardown \
  --command "$seed_env"'; cd "$OUIJIT_WORKTREE_PATH" && { if [ -x .seed/hooks/teardown ]; then .seed/hooks/teardown; fi; }'
echo "seed hooks registered with ouijit (start/run/done)"
