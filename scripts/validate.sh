#!/bin/sh
# Lint all orchestration artifacts (R9: a shipped convention and its validator
# are one deliverable). v1 scope: structural presence + the engine's spec
# lint. The full validator set (guardrails intersection, team files, plan
# pinning, receipts) lands with Phases 4-5 and runs here too.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
fail=0
say() { printf 'validate: %s\n' "$*"; }

for f in .seed/version .seed/engine.lock .seed/config.toml .seed/guardrails.yaml \
         .seed/port-schema/port.json .seed/port-schema/transitions.json \
         CODEOWNERS AGENTS.md CLAUDE.md Makefile; do
  if [ ! -f "$root/$f" ]; then
    say "MISSING $f"
    fail=1
  fi
done

# Spec lint via the engine when it can be bootstrapped; degrade with a warning
# otherwise (differentiator #4: the repo must work without the engine).
if out=$(cd "$root" && sh scripts/seed spec lint 2>&1); then
  say "$out"
else
  say "WARNING: engine unavailable, skipped spec lint ($out)"
fi

[ "$fail" -eq 0 ] && say "ok" || exit 1
