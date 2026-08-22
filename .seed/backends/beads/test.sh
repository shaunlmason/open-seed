#!/bin/sh
# Contract test for the beads plugin against the fake bd (offline, CI-safe).
# Exercises every required verb's envelope shape, exit-code mapping, the
# token fence, close-from-review-only, blocked_on persistence, and the
# reviewer lockout.
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
expect_rc() { # rc_want desc cmd...
  want=$1; desc=$2; shift 2
  rc=0; "$@" >/dev/null 2>&1 || rc=$?
  [ "$rc" = "$want" ] || die "$desc: exit $rc, want $want"
}

out=$("$sb" create --title "Test card" --body "the work" --priority P1 --squad core --actor a --json); ok "$out" create
id=$(echo "$out" | jq -r .task); [ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"

# Create rollback: defer failure must not leave a claimable issue.
FAKE_BD_FAIL_DEFER=1 expect_rc 5 "create with failed defer" "$sb" create --title "Doomed" --actor a --json
[ "$(bd list --json | jq '[.[] | select(.title == "Doomed" and .status != "closed")] | length')" = "0" ] || die "doomed issue not rolled back"

out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
out=$("$sb" ready --actor a --json)
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"

out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
tok=$(echo "$out" | jq -r .claim_token); [ "$tok" = "assignee:agent-1" ] || die "token shape"

expect_rc 2 "rival claim" "$sb" claim "$id" --actor agent-2 --json
# The fence: right actor + wrong/missing token must exit 6 (P1 finding).
expect_rc 6 "no-token transition" "$sb" transition "$id" --to review --actor agent-1 --json
expect_rc 6 "wrong-token transition" "$sb" transition "$id" --to review --actor agent-1 --token bogus --json
expect_rc 6 "wrong-actor transition" "$sb" transition "$id" --to review --actor agent-2 --token "$tok" --json
expect_rc 6 "no-token comment on claimed card" "$sb" comment "$id" --body x --actor agent-1 --json
# Close is review→done only (P1 finding): refuse from in_progress.
expect_rc 3 "close from in_progress" "$sb" close "$id" --actor lead --resolution x --json

out=$("$sb" comment "$id" --body "progress" --actor agent-1 --token "$tok" --json); ok "$out" comment
out=$("$sb" lease-renew "$id" --actor agent-1 --token "$tok" --json); ok "$out" lease-renew

# Plan parking persists blocked_on (P1 finding), plan-unblock releases it.
out=$("$sb" transition "$id" --to blocked --actor agent-1 --token "$tok" --blocked-on plan:41 --json); ok "$out" park
[ "$("$sb" get "$id" --json | jq -r '.card.blocked_on[0]')" = "plan:41" ] || die "blocked_on not persisted"
expect_rc 3 "plan-unblock wrong pr" "$sb" plan-unblock "$id" --pr 99 --actor lead --json
out=$("$sb" plan-unblock "$id" --pr 41 --actor lead --json); ok "$out" plan-unblock
[ "$(echo "$out" | jq -r .state)" = "ready" ] || die "plan-unblock did not release"

out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" reclaim
tok=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-1 --token "$tok" --json); ok "$out" review
[ "$("$sb" get "$id" --json | jq -r .state)" = "review" ] || die "review state"

# Reject records the implementer and locks them out (P1 finding).
out=$("$sb" reject "$id" --actor lead --resolution "not yet" --json); ok "$out" reject
expect_rc 2 "rejected author reclaim" "$sb" claim "$id" --actor agent-1 --json
out=$("$sb" ready --actor agent-1 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "rejected author still sees card in ready"
out=$("$sb" claim "$id" --actor agent-2 --json); ok "$out" claim2
tok2=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-2 --token "$tok2" --json); ok "$out" re-review

# Close from review succeeds and cascades port dep entries.
out=$("$sb" create --title "Dependent" --actor a --blocked-by "$id" --json); dep=$(echo "$out" | jq -r .task)
"$sb" promote "$dep" --actor lead --json >/dev/null
"$sb" block "$dep" --actor lead --blocked-on "dep:$id" --json >/dev/null 2>&1 || true
out=$("$sb" close "$id" --actor lead --resolution "merged" --json); ok "$out" close
[ "$("$sb" get "$id" --json | jq -r .state)" = "done" ] || die "close → done"
[ "$("$sb" get "$dep" --json | jq -r .state)" = "ready" ] || die "cascade did not release dependent"

out=$("$sb" create --title "Cancel me" --actor a --json); id2=$(echo "$out" | jq -r .task)
out=$("$sb" cancel "$id2" --actor lead --json); ok "$out" cancel
expect_rc 3 "cancel from terminal" "$sb" cancel "$id2" --actor lead --json
out=$("$sb" reinstate "$id2" --actor lead --json); ok "$out" reinstate
[ "$("$sb" get "$id2" --json | jq -r .state)" = "backlog" ] || die "reinstate → backlog"

# Out-of-band events persist to the audit issue (P2 finding).
out=$("$sb" event-append --event '{"ts":"t","actor":"x","verb":"probe"}' --json); ok "$out" event-append
audit=$(bd list --json | jq -r '.[] | select(.labels | index("seed:audit")) | .id')
[ -n "$audit" ] || die "no audit issue created"
[ "$(bd show "$audit" | jq '.comments | length')" = "1" ] || die "event not persisted to audit issue"

out=$("$sb" list --json); ok "$out" list
expect_rc 4 "missing card" "$sb" get bd-nope --json

say "OK: all required verbs round-trip; fence, review-only close, blocked_on, lockout, cascade, audit all enforced"
