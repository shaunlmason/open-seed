#!/bin/sh
# Contract test for the paperclip plugin against the fake server (offline,
# CI-safe). Covers every required verb, the minted-token fence (incl. the
# reap-then-reclaim rotation case), checkout contention, checkout-aware
# ready, mandatory goal ancestry, server-arbitrated transitions, the
# reviewer lockout, and the dep cascade.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
command -v python3 >/dev/null 2>&1 || { echo "paperclip-test: SKIP (python3 unavailable)"; exit 0; }
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
python3 "$dir/testdata/fake-paperclip" "$port" & srv=$!
trap 'kill $srv 2>/dev/null || true' EXIT
export PAPERCLIP_API_URL="http://127.0.0.1:$port" PAPERCLIP_API_KEY=test PAPERCLIP_COMPANY_ID=co-1 PAPERCLIP_DEFAULT_GOAL_ID=goal-1
for i in $(seq 1 50); do curl -sf "$PAPERCLIP_API_URL/api/companies/co-1/issues" >/dev/null 2>&1 && break; sleep 0.1; done

sb="$dir/bin/seed-backend"
say() { printf 'paperclip-test: %s\n' "$*"; }
die() { say "FAIL: $*"; exit 1; }
ok() { echo "$1" | jq -e '.ok == true and .schema_version == "1.0"' >/dev/null || die "$2: bad envelope: $1"; }
expect_rc() { want=$1; desc=$2; shift 2; rc=0; "$@" >/dev/null 2>&1 || rc=$?; [ "$rc" = "$want" ] || die "$desc: exit $rc, want $want"; }

# Mandatory ancestry: default goal covers parentless creates; without any
# goal source, create refuses with remediation.
out=$("$sb" create --title "Card A" --body work --priority P1 --actor a --json); ok "$out" create
id=$(echo "$out" | jq -r .task); [ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"
PAPERCLIP_DEFAULT_GOAL_ID= expect_rc 5 "parentless create without goal" "$sb" create --title "No goal" --actor a --json

out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
out=$("$sb" ready --actor agent-1 --json); ok "$out" ready
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"

out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
tok1=$(echo "$out" | jq -r .claim_token)
case "$tok1" in c-*) ;; *) die "minted token shape: $tok1" ;; esac

# Checkout-aware ready: a checked-out issue is not claimable work.
out=$("$sb" ready --actor agent-2 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "checked-out issue still in ready"
expect_rc 2 "rival claim" "$sb" claim "$id" --actor agent-2 --json
expect_rc 6 "no-token transition" "$sb" transition "$id" --to review --actor agent-1 --json
expect_rc 6 "wrong-token transition" "$sb" transition "$id" --to review --actor agent-1 --token c-bogus --json
expect_rc 3 "close from in_progress" "$sb" close "$id" --actor lead --json

# The rotation case (D1): release (reap stand-in), same actor reclaims —
# the old token must be dead even though the identity is unchanged.
out=$("$sb" release "$id" --actor agent-1 --token "$tok1" --json); ok "$out" release
out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" reclaim
tok2=$(echo "$out" | jq -r .claim_token)
[ "$tok1" != "$tok2" ] || die "token did not rotate on reclaim"
expect_rc 6 "stale token after reclaim (same actor)" "$sb" comment "$id" --body late --actor agent-1 --token "$tok1" --json

out=$("$sb" comment "$id" --body progress --actor agent-1 --token "$tok2" --json); ok "$out" comment
out=$("$sb" lease-renew "$id" --actor agent-1 --token "$tok2" --json); ok "$out" lease-renew

# Park on a plan, then state-shaped unpark.
out=$("$sb" transition "$id" --to blocked --actor agent-1 --token "$tok2" --blocked-on plan:7 --json); ok "$out" park
[ "$("$sb" get "$id" --json | jq -r '.card.blocked_on[0]')" = "plan:7" ] || die "blocked_on not persisted"
expect_rc 3 "plan-unblock wrong pr" "$sb" plan-unblock "$id" --pr 99 --actor lead --json
out=$("$sb" plan-unblock "$id" --pr 7 --actor lead --json); ok "$out" plan-unblock
[ "$(echo "$out" | jq -r .state)" = "ready" ] || die "plan-unblock did not release"

out=$("$sb" claim "$id" --actor agent-1 --json); tok3=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-1 --token "$tok3" --json); ok "$out" review
[ "$("$sb" get "$id" --json | jq -r .state)" = "review" ] || die "review mapping (in_review)"

# Reject → lockout; a clean actor proceeds.
out=$("$sb" reject "$id" --actor lead --resolution "not yet" --json); ok "$out" reject
expect_rc 2 "rejected author reclaim" "$sb" claim "$id" --actor agent-1 --json
out=$("$sb" ready --actor agent-1 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "rejected author still sees card"
out=$("$sb" claim "$id" --actor agent-2 --json); tok4=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-2 --token "$tok4" --json); ok "$out" re-review

# Cascade: a dependent created --blocked-by releases on the blocker's close.
out=$("$sb" create --title "Dependent" --actor a --blocked-by "$id" --json); dep=$(echo "$out" | jq -r .task)
"$sb" promote "$dep" --actor lead --json >/dev/null
"$sb" block "$dep" --actor lead --blocked-on "dep:$id" --json >/dev/null 2>&1 || true
out=$("$sb" close "$id" --actor lead --resolution merged --json); ok "$out" close
[ "$("$sb" get "$id" --json | jq -r .state)" = "done" ] || die "close → done"
[ "$("$sb" get "$dep" --json | jq -r .state)" = "ready" ] || die "cascade did not release dependent"
expect_rc 3 "close from done (assertTransition)" "$sb" close "$id" --actor lead --json

out=$("$sb" create --title "Cancel me" --actor a --json); id2=$(echo "$out" | jq -r .task)
out=$("$sb" cancel "$id2" --actor lead --json); ok "$out" cancel
out=$("$sb" reinstate "$id2" --actor lead --json); ok "$out" reinstate
[ "$("$sb" get "$id2" --json | jq -r .state)" = "backlog" ] || die "reinstate → backlog"

out=$("$sb" event-append --event '{"ts":"t","verb":"probe"}' --json); ok "$out" event-append
out=$("$sb" list --json); ok "$out" list
expect_rc 4 "missing issue" "$sb" get PAP-999 --json

say "OK: all required verbs; minted-token fence (incl. rotation), checkout-aware ready, ancestry, server-arbitrated transitions, lockout, cascade all enforced"
