#!/bin/sh
# Contract test for the linear plugin against the fake GraphQL server
# (offline, CI-safe). Covers every required verb, the body round-trip, the
# minted-token fence (incl. rotation), the contention envelope (holder on
# record), plan parking, the reviewer lockout, close review-only, the
# cascade on close AND cancel, comment_id/evidence_id, event .task routing,
# the audit-issue exclusions, the deterministic lost-claim race, the
# compensated failed claim, the refused cascade release (skip-and-report),
# and the missing-workflow-state refusal → 5.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
command -v python3 >/dev/null 2>&1 || { echo "linear-test: SKIP (python3 unavailable)"; exit 0; }
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
python3 "$dir/testdata/fake-linear" "$port" & srv=$!
trap 'kill $srv 2>/dev/null || true' EXIT
export LINEAR_API_URL="http://127.0.0.1:$port" LINEAR_API_KEY=lin_test LINEAR_TEAM_ID=team-1
for i in $(seq 1 50); do curl -sf "$LINEAR_API_URL" >/dev/null 2>&1 && break; sleep 0.1; done

sb="$dir/bin/seed-backend"
say() { printf 'linear-test: %s\n' "$*"; }
die() { say "FAIL: $*"; exit 1; }
ok() { echo "$1" | jq -e '.ok == true and .schema_version == "1.0"' >/dev/null || die "$2: bad envelope: $1"; }
expect_rc() { want=$1; desc=$2; shift 2; rc=0; "$@" >/dev/null 2>&1 || rc=$?; [ "$rc" = "$want" ] || die "$desc: exit $rc, want $want"; }

out=$("$sb" create --title "Card A" --body work --priority P1 --actor a --json); ok "$out" create
id=$(echo "$out" | jq -r .task); [ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"
[ "$("$sb" get "$id" --json | jq -r .card.body)" = "work" ] || die "card body (description) not round-tripped"

out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
out=$("$sb" ready --actor agent-1 --json); ok "$out" ready
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"

# Priority ordering: loop.sh claims the FIRST returned task.
hi=$("$sb" create --title "Urgent" --priority P0 --actor a --json | jq -r .task)
"$sb" promote "$hi" --actor lead --json >/dev/null
out=$("$sb" ready --actor agent-1 --json)
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$hi" ] || die "P0 not first in ready"
"$sb" cancel "$hi" --actor lead --json >/dev/null

out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
tok1=$(echo "$out" | jq -r .claim_token)
case "$tok1" in c-*) ;; *) die "minted token shape: $tok1" ;; esac

# Claimed work leaves ready; a rival claim is refused with the holder.
out=$("$sb" ready --actor agent-2 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "claimed issue still in ready"
expect_rc 2 "rival claim" "$sb" claim "$id" --actor agent-2 --json
out=$("$sb" claim "$id" --actor agent-2 --json 2>/dev/null || true)
echo "$out" | jq -e '.error == "claim_contention" and .holder == "agent-1"' >/dev/null || die "contention envelope lacks claim_contention/holder: $out"
expect_rc 6 "no-token transition" "$sb" transition "$id" --to review --actor agent-1 --json
expect_rc 6 "wrong-token transition" "$sb" transition "$id" --to review --actor agent-1 --token c-bogus --json
expect_rc 6 "regex-metachar token cannot bypass the fence" "$sb" transition "$id" --to review --actor agent-1 --token '.*' --json
expect_rc 3 "close from in_progress" "$sb" close "$id" --actor lead --json

# Token rotation: release, same-actor reclaim — the old token must be dead.
out=$("$sb" release "$id" --actor agent-1 --token "$tok1" --json); ok "$out" release
out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" reclaim
tok2=$(echo "$out" | jq -r .claim_token)
[ "$tok1" != "$tok2" ] || die "token did not rotate on reclaim"
expect_rc 6 "stale token after reclaim (same actor)" "$sb" comment "$id" --body late --actor agent-1 --token "$tok1" --json

out=$("$sb" comment "$id" --body progress --actor agent-1 --token "$tok2" --json); ok "$out" comment
[ -n "$(echo "$out" | jq -r '.comment_id // ""')" ] || die "comment_id missing from envelope"
out=$("$sb" attach-evidence "$id" --kind commit --ref abc123 --actor agent-1 --token "$tok2" --json); ok "$out" attach-evidence
[ -n "$(echo "$out" | jq -r '.evidence_id // ""')" ] || die "evidence_id missing from envelope"
out=$("$sb" lease-renew "$id" --actor agent-1 --token "$tok2" --json); ok "$out" lease-renew

# Park on a plan, entry-by-entry release.
out=$("$sb" transition "$id" --to blocked --actor agent-1 --token "$tok2" --blocked-on plan:7 --json); ok "$out" park
[ "$("$sb" get "$id" --json | jq -r '.card.blocked_on[0]')" = "plan:7" ] || die "blocked_on not persisted"
expect_rc 3 "plan-unblock wrong pr" "$sb" plan-unblock "$id" --pr 99 --actor lead --json
out=$("$sb" plan-unblock "$id" --pr 7 --actor lead --json); ok "$out" plan-unblock
[ "$(echo "$out" | jq -r .state)" = "ready" ] || die "plan-unblock did not release"

out=$("$sb" claim "$id" --actor agent-1 --json); tok3=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-1 --token "$tok3" --json); ok "$out" review
[ "$("$sb" get "$id" --json | jq -r .state)" = "review" ] || die "review mapping (In Review)"

# Reject → lockout; a clean actor proceeds.
out=$("$sb" reject "$id" --actor lead --resolution "not yet" --json); ok "$out" reject
expect_rc 2 "rejected author reclaim" "$sb" claim "$id" --actor agent-1 --json
out=$("$sb" claim "$id" --actor agent-1 --json 2>/dev/null || true)
echo "$out" | jq -e '.error == "rejected_author"' >/dev/null || die "lockout cause is not rejected_author: $out"
out=$("$sb" ready --actor agent-1 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "rejected author still sees card"
out=$("$sb" claim "$id" --actor agent-2 --json); tok4=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-2 --token "$tok4" --json); ok "$out" re-review

# Cascade on CLOSE: a dependent created --blocked-by releases when the
# blocker is accepted.
dep=$("$sb" create --title "Dependent" --actor a --blocked-by "$id" --json | jq -r .task)
"$sb" promote "$dep" --actor lead --json >/dev/null
"$sb" block "$dep" --actor lead --blocked-on "dep:$id" --json >/dev/null
[ "$("$sb" get "$dep" --json | jq -r .state)" = "blocked" ] || die "cascade setup: dependent not blocked"
out=$("$sb" close "$id" --actor lead --resolution merged --json); ok "$out" close
[ "$("$sb" get "$id" --json | jq -r .state)" = "done" ] || die "close → done"
[ "$("$sb" get "$dep" --json | jq -r .state)" = "ready" ] || die "close cascade did not release dependent"
expect_rc 3 "close from done" "$sb" close "$id" --actor lead --json

# Cascade on CANCEL: the nonterminal→cancelled edge carries the same effect.
blk=$("$sb" create --title "Blocker B" --actor a --json | jq -r .task)
"$sb" promote "$blk" --actor lead --json >/dev/null
dep2=$("$sb" create --title "Dependent B" --actor a --blocked-by "$blk" --json | jq -r .task)
"$sb" promote "$dep2" --actor lead --json >/dev/null
"$sb" block "$dep2" --actor lead --blocked-on "dep:$blk" --json >/dev/null
[ "$("$sb" get "$dep2" --json | jq -r .state)" = "blocked" ] || die "cancel-cascade setup: dependent not blocked"
out=$("$sb" cancel "$blk" --actor lead --json); ok "$out" cancel
echo "$out" | jq -e --arg d "$dep2" '.cascaded | index($d)' >/dev/null || die "cancel cascade missing dependent"
[ "$("$sb" get "$dep2" --json | jq -r .state)" = "ready" ] || die "cancel cascade did not release dependent"

out=$("$sb" reinstate "$blk" --actor lead --json); ok "$out" reinstate
[ "$("$sb" get "$blk" --json | jq -r .state)" = "backlog" ] || die "reinstate → backlog"

# Cancelling in-progress work ends the claim: after reinstate + promote the
# card must be claimable, not stranded behind a dead holder.
"$sb" promote "$blk" --actor lead --json >/dev/null
"$sb" claim "$blk" --actor agent-1 --json >/dev/null
out=$("$sb" cancel "$blk" --actor lead --json); ok "$out" cancel-live-claim
"$sb" reinstate "$blk" --actor lead --json >/dev/null
"$sb" promote "$blk" --actor lead --json >/dev/null
out=$("$sb" claim "$blk" --actor agent-2 --json); ok "$out" "reclaim after cancel"

# Unfenced comment while no live claim exists (done card).
out=$("$sb" comment "$id" --body "post-review note" --actor lead --json); ok "$out" "unfenced comment"

# Bare operator unblock derives manual:<actor>; list applies --state.
mb=$("$sb" create --title "Manual block" --actor a --json | jq -r .task)
"$sb" promote "$mb" --actor lead --json >/dev/null
"$sb" block "$mb" --actor lead --blocked-on manual:lead --json >/dev/null
out=$("$sb" unblock "$mb" --actor lead --json); ok "$out" "bare unblock derives manual entry"
[ "$(echo "$out" | jq -r .state)" = "ready" ] || die "manual unblock did not release"
out=$("$sb" list --state ready --json); ok "$out" "list filtered"
[ "$(echo "$out" | jq '[.tasks[] | select(.state != "ready")] | length')" = "0" ] || die "list --state leaked other states"

out=$("$sb" event-append --event '{"ts":"t","verb":"probe"}' --json); ok "$out" event-append
out=$("$sb" event-append --event "{\"ts\":\"t\",\"verb\":\"probe\",\"task\":\"$id\"}" --json); ok "$out" "event-append with task in event JSON"
# The audit issue is the audit substrate, never a card: absent from ready,
# refused at claim.
out=$("$sb" ready --actor agent-1 --json)
echo "$out" | jq -e '[.tasks[].title] | index("seed: audit log") | not' >/dev/null || die "audit issue leaked into ready"
audit_id=$("$sb" list --json | jq -r '.tasks[] | select(.title == "seed: audit log") | .task')
expect_rc 3 "claiming the audit issue" "$sb" claim "$audit_id" --actor agent-1 --json

# Exact-match blocked_on release: removing manual:x must not strip manual:xy.
xb=$("$sb" create --title "Exact match" --actor a --json | jq -r .task)
"$sb" promote "$xb" --actor lead --json >/dev/null
"$sb" block "$xb" --actor lead --blocked-on manual:x --json >/dev/null
"$sb" block "$xb" --actor lead --blocked-on manual:xy --json >/dev/null
out=$("$sb" unblock "$xb" --actor lead --blocked-on manual:x --json); ok "$out" "exact unblock"
[ "$(echo "$out" | jq -r .state)" = "blocked" ] || die "prefix deletion stripped manual:xy too"
[ "$("$sb" get "$xb" --json | jq -r '.card.blocked_on[0]')" = "manual:xy" ] || die "manual:xy lost"
out=$("$sb" list --json); ok "$out" list
expect_rc 4 "missing issue" "$sb" get LIN-999 --json

# A lost claim interleave resolves to the substrate's actual holder: the
# rival's later write wins, our verification sees it, and we report
# contention instead of returning a dead token.
kill $srv 2>/dev/null || true; wait $srv 2>/dev/null || true
portr=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
FAKE_LINEAR_RACE_ACTOR=agent-2 python3 "$dir/testdata/fake-linear" "$portr" & srv=$!
export LINEAR_API_URL="http://127.0.0.1:$portr"
for i in $(seq 1 50); do curl -sf "$LINEAR_API_URL" >/dev/null 2>&1 && break; sleep 0.1; done
rc_card=$("$sb" create --title "Race card" --actor a --json | jq -r .task)
"$sb" promote "$rc_card" --actor lead --json >/dev/null
rc=0; out=$("$sb" claim "$rc_card" --actor agent-1 --json) || rc=$?
[ "$rc" = "2" ] || die "lost race: exit $rc, want 2"
echo "$out" | jq -e '.error == "claim_contention"' >/dev/null || die "lost race envelope: $out"
[ "$("$sb" get "$rc_card" --json | jq -r .card.holder)" = "agent-2" ] || die "race winner not recorded"

# A failed claim is compensated: the assign+move refusal rolls the minted
# token back and the issue is left unchanged — unheld, still claimable.
kill $srv 2>/dev/null || true; wait $srv 2>/dev/null || true
portf=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
FAKE_LINEAR_FAIL_ASSIGN=1 python3 "$dir/testdata/fake-linear" "$portf" & srv=$!
export LINEAR_API_URL="http://127.0.0.1:$portf"
for i in $(seq 1 50); do curl -sf "$LINEAR_API_URL" >/dev/null 2>&1 && break; sleep 0.1; done
fc=$("$sb" create --title "Failed claim" --actor a --json | jq -r .task)
"$sb" promote "$fc" --actor lead --json >/dev/null
expect_rc 5 "refused assign fails the claim" "$sb" claim "$fc" --actor agent-1 --json
[ "$("$sb" get "$fc" --json | jq -r .state)" = "ready" ] || die "failed claim moved the issue"
[ "$("$sb" get "$fc" --json | jq -r .card.holder)" = "" ] || die "failed claim left a holder"
out=$("$sb" ready --actor agent-2 --json)
echo "$out" | jq -e --arg t "$fc" '[.tasks[].task] | index($t)' >/dev/null || die "failed claim hid the card from ready"

# A refused cascade release keeps the dependency recorded (recoverable via
# operator unblock once the substrate allows it) — never a Blocked issue
# with no blocker on record.
kill $srv 2>/dev/null || true; wait $srv 2>/dev/null || true
port3=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
FAKE_LINEAR_NO_TODO_FROM_BLOCKED=1 python3 "$dir/testdata/fake-linear" "$port3" & srv=$!
export LINEAR_API_URL="http://127.0.0.1:$port3"
for i in $(seq 1 50); do curl -sf "$LINEAR_API_URL" >/dev/null 2>&1 && break; sleep 0.1; done
cb=$("$sb" create --title "Cascade blocker" --actor a --json | jq -r .task)
"$sb" promote "$cb" --actor lead --json >/dev/null
cd2=$("$sb" create --title "Cascade dependent" --actor a --blocked-by "$cb" --json | jq -r .task)
"$sb" promote "$cd2" --actor lead --json >/dev/null
"$sb" block "$cd2" --actor lead --blocked-on "dep:$cb" --json >/dev/null
[ "$("$sb" get "$cd2" --json | jq -r .state)" = "blocked" ] || die "refused-cascade setup: dependent not blocked"
tokx=$("$sb" claim "$cb" --actor agent-1 --json | jq -r .claim_token)
"$sb" transition "$cb" --to review --actor agent-1 --token "$tokx" --json >/dev/null
out=$("$sb" close "$cb" --actor lead --json); ok "$out" "close with refused cascade release"
echo "$out" | jq -e --arg d "$cd2" '.cascade_skipped | index($d)' >/dev/null || die "skipped dependent not reported"
[ "$("$sb" get "$cd2" --json | jq -r '.card.blocked_on[0]')" = "dep:$cb" ] || die "refused cascade dropped the dependency label"
expect_rc 5 "unblock refused by substrate keeps entry" "$sb" unblock "$cd2" --actor lead --blocked-on "dep:$cb" --json
[ "$("$sb" get "$cd2" --json | jq -r '.card.blocked_on[0]')" = "dep:$cb" ] || die "refused unblock dropped the entry"

# The required Blocked state is a declared convention: a workflow without it
# refuses with remediation at first use.
kill $srv 2>/dev/null || true; wait $srv 2>/dev/null || true
port2=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
FAKE_LINEAR_NO_BLOCKED=1 python3 "$dir/testdata/fake-linear" "$port2" & srv=$!
export LINEAR_API_URL="http://127.0.0.1:$port2"
for i in $(seq 1 50); do curl -sf "$LINEAR_API_URL" >/dev/null 2>&1 && break; sleep 0.1; done
nb=$("$sb" create --title "No blocked state" --actor a --json | jq -r .task)
expect_rc 5 "missing Blocked workflow state" "$sb" block "$nb" --actor lead --blocked-on manual:x --json

say "OK: all required verbs; body round-trip, minted-token fence (incl. rotation), contention with holder, compensated failed claim, lost-race verification, parking, lockout, close+cancel cascades with skip-and-report, comment/evidence ids, event .task routing, audit exclusions, missing-state refusal all enforced"
