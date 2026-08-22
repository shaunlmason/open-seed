#!/bin/sh
# Live contract test: the SAME corpus as test.sh, driven through the
# adapter against a real bd install (plan os-435d7b61). Self-skips with
# exit 0 when bd is absent — CI stays green and credential-free; running
# against a version other than the validated pin warns but proceeds
# (drift discovery is the point).
set -eu
dir=$(cd "$(dirname "$0")" && pwd)

PINNED_BD="1.2.2"
if ! command -v bd >/dev/null 2>&1; then
  printf 'beads-live: SKIP — bd not on PATH. Validated install: go install github.com/steveyegge/beads/cmd/bd@v%s (CGO: apt install libicu-dev pkg-config; Go >= 1.26.2 auto-toolchains).\n' "$PINNED_BD"
  exit 0
fi
ver=$(bd --version 2>/dev/null | awk '{print $3}')
printf 'beads-live: bd %s on PATH (validated pin: v%s)\n' "${ver:-unknown}" "$PINNED_BD"
[ "$ver" = "$PINNED_BD" ] || printf 'beads-live: WARNING: bd %s differs from the validated pin v%s — running anyway to surface drift\n' "${ver:-unknown}" "$PINNED_BD"

# Scratch repo: bd stores per-repo, so the corpus starts from an empty
# database exactly like the fake run.
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
cd "$work"
git init -q --initial-branch=main .
bd init >/dev/null 2>&1 || { printf 'beads-live: FAIL: bd init refused in scratch repo\n'; exit 1; }

sb="$dir/bin/seed-backend"
PREFIX="beads-live"
. "$dir/corpus.sh"
run_corpus

say "OK: full corpus green against real bd ${ver:-unknown}"
