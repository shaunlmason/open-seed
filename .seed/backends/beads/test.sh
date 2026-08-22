#!/bin/sh
# Contract test for the beads plugin against the fake bd (offline, CI-safe).
# The corpus itself lives in corpus.sh, shared verbatim with live-test.sh so
# the offline and live suites cannot drift apart; only fake-only scenarios
# (forced failures) live here.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
export PATH="$work/bin:$PATH"
mkdir -p "$work/bin" && cp "$dir/testdata/fake-bd" "$work/bin/bd" && chmod +x "$work/bin/bd"
export BD_FAKE_STATE="$work/state.json"
sb="$dir/bin/seed-backend"
PREFIX="beads-test"
. "$dir/corpus.sh"

# Fake-only: create rollback — a forced defer failure must not leave a
# claimable issue (the fake's failure injection has no live analogue).
FAKE_BD_FAIL_DEFER=1 expect_rc 5 "create with failed defer" "$sb" create --title "Doomed" --actor a --json
[ "$(bd list --json | jq '[.[] | select(.title == "Doomed" and .status != "closed")] | length')" = "0" ] || die "doomed issue not rolled back"

run_corpus

say "OK: all required verbs round-trip; fence, review-only close, blocked_on, lockout, cascade, audit all enforced"
