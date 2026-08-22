#!/bin/sh
# Contract test for the beads plugin against the fake bd (offline, CI-safe).
# Exercises every required verb's envelope shape and exit-code mapping.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d); trap 'rm -rf "$work"' EXIT
export PATH="$work/bin:$PATH"
mkdir -p "$work/bin" && cp "$dir/testdata/fake-bd" "$work/bin/bd" && chmod +x "$work/bin/bd"
export BD_FAKE_STATE="$work/state.json"
sb="$dir/bin/seed-backend"
say() { printf 'beads-test: %s\n' "$*"; }
die() { say "FAIL: $*"; exit 1; }
ok() { echo "$1" | jq -e '.ok == true and .schema_version == "1.0"' >/dev/null || die "$2: bad envelope: $1"; }

out=$("$sb" create --title "Test card" --body "the work" --priority P1 --squad core --actor a --json); ok "$out" create
id=$(echo "$out" | jq -r .task); [ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"

out=$("$sb" ready --actor a --json); ok "$out" ready
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "backlog card visible in ready"

out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
out=$("$sb" ready --actor a --json)
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"
[ "$(echo "$out" | jq -r '.tasks[0].priority')" = "P1" ] || die "priority mapping"

out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
[ "$(echo "$out" | jq -r .state)" = "in_progress" ] || die "claim state"

"$sb" claim "$id" --actor agent-2 --json > "$work/c2" 2>&1 && die "rival claim succeeded"
rc=$?; [ "$rc" = "2" ] || die "rival claim exit $rc, want 2"
[ "$(jq -r .holder "$work/c2")" = "agent-1" ] || die "contention holder"

"$sb" transition "$id" --to review --actor agent-2 --json >/dev/null 2>&1 && die "wrong-actor transition passed"
rc=$?; [ "$rc" = "6" ] || die "fence exit $rc, want 6"

out=$("$sb" comment "$id" --body "progress" --actor agent-1 --json); ok "$out" comment
out=$("$sb" attach-evidence "$id" --kind pr --ref "https://x/pr/1" --actor agent-1 --json); ok "$out" evidence
out=$("$sb" lease-renew "$id" --actor agent-1 --json); ok "$out" lease-renew

out=$("$sb" transition "$id" --to review --actor agent-1 --json); ok "$out" review
out=$("$sb" get "$id" --json); ok "$out" get
[ "$(echo "$out" | jq -r .state)" = "review" ] || die "review state mapping (label)"

out=$("$sb" reject "$id" --actor lead --resolution "not yet" --json); ok "$out" reject
[ "$("$sb" get "$id" --json | jq -r .state)" = "ready" ] || die "reject → ready"

out=$("$sb" claim "$id" --actor agent-2 --json); ok "$out" reclaim
out=$("$sb" transition "$id" --to review --actor agent-2 --json); ok "$out" re-review
out=$("$sb" close "$id" --actor lead --resolution "merged" --json); ok "$out" close
[ "$("$sb" get "$id" --json | jq -r .state)" = "done" ] || die "close → done"

out=$("$sb" create --title "Cancel me" --actor a --json); id2=$(echo "$out" | jq -r .task)
out=$("$sb" cancel "$id2" --actor lead --json); ok "$out" cancel
[ "$("$sb" get "$id2" --json | jq -r .state)" = "cancelled" ] || die "cancel mapping (label)"
out=$("$sb" reinstate "$id2" --actor lead --json); ok "$out" reinstate
[ "$("$sb" get "$id2" --json | jq -r .state)" = "backlog" ] || die "reinstate → backlog"

out=$("$sb" event-append --body "out-of-band" --json); ok "$out" event-append
out=$("$sb" list --json); ok "$out" list

"$sb" get bd-nope --json >/dev/null 2>&1 && die "missing card found"
rc=$?; [ "$rc" = "4" ] || die "not_found exit $rc, want 4"

say "OK: all required verbs round-trip with valid envelopes and mapped exit codes"
