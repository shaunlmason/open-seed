#!/bin/sh
# Contract test for the jira plugin against the fake REST v3 server
# (offline, CI-safe). Covers every required verb, create landing in Backlog
# despite a To-Do-initial workflow, the accountId mapping (unknown actor →
# 5), the minted-token fence (incl. rotation), contention, plan parking,
# the reviewer lockout, close review-only, the cascade on close AND cancel,
# transition-unavailable → 3, and the missing-Blocked-status refusal → 5.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
command -v python3 >/dev/null 2>&1 || { echo "jira-test: SKIP (python3 unavailable)"; exit 0; }
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
python3 "$dir/testdata/fake-jira" "$port" & srv=$!
trap 'kill $srv 2>/dev/null || true' EXIT
actors=$(mktemp)
trap 'kill $srv 2>/dev/null || true; rm -f "$actors"' EXIT
cat > "$actors" <<'EOF'
{"a": "acc-a", "lead": "acc-lead", "agent-1": "acc-1", "agent-2": "acc-2"}
EOF
export JIRA_BASE_URL="http://127.0.0.1:$port" JIRA_EMAIL=t@t JIRA_API_TOKEN=tok JIRA_PROJECT_KEY=SEED JIRA_ACTORS_FILE="$actors"
for i in $(seq 1 50); do curl -sf "$JIRA_BASE_URL/rest/api/3/search?jql=x" >/dev/null 2>&1 && break; sleep 0.1; done

sb="$dir/bin/seed-backend"
say() { printf 'jira-test: %s\n' "$*"; }
die() { say "FAIL: $*"; exit 1; }
ok() { echo "$1" | jq -e '.ok == true and .schema_version == "1.0"' >/dev/null || die "$2: bad envelope: $1"; }
expect_rc() { want=$1; desc=$2; shift 2; rc=0; "$@" >/dev/null 2>&1 || rc=$?; [ "$rc" = "$want" ] || die "$desc: exit $rc, want $want"; }

# Create must land in Backlog even though the fake workflow's initial
# status is To Do (the port contract: create returns backlog).
out=$("$sb" create --title "Card A" --body work --priority P1 --actor a --json); ok "$out" create
id=$(echo "$out" | jq -r .task)
[ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"
[ "$("$sb" get "$id" --json | jq -r .state)" = "backlog" ] || die "create did not actually land in Backlog"

out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
out=$("$sb" ready --actor agent-1 --json); ok "$out" ready
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"

# Priority ordering: the loop claims the FIRST returned task.
hi=$("$sb" create --title "Urgent" --priority P0 --actor a --json | jq -r .task)
"$sb" promote "$hi" --actor lead --json >/dev/null
out=$("$sb" ready --actor agent-1 --json)
[ "$(echo "$out" | jq -r '.tasks[0].task')" = "$hi" ] || die "P0 not first in ready"
"$sb" cancel "$hi" --actor lead --json >/dev/null

# Unknown actor: no accountId mapping → exit 5 with remediation.
expect_rc 5 "unmapped actor claim" "$sb" claim "$id" --actor nobody --json

out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
tok1=$(echo "$out" | jq -r .claim_token)
case "$tok1" in c-*) ;; *) die "minted token shape: $tok1" ;; esac

out=$("$sb" ready --actor agent-2 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "claimed issue still in ready"
expect_rc 2 "rival claim" "$sb" claim "$id" --actor agent-2 --json
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
out=$("$sb" ready --actor agent-1 --json)
[ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "rejected author still sees card"
out=$("$sb" claim "$id" --actor agent-2 --json); tok4=$(echo "$out" | jq -r .claim_token)
out=$("$sb" transition "$id" --to review --actor agent-2 --token "$tok4" --json); ok "$out" re-review

# Cascade on CLOSE.
dep=$("$sb" create --title "Dependent" --actor a --blocked-by "$id" --json | jq -r .task)
"$sb" promote "$dep" --actor lead --json >/dev/null
"$sb" block "$dep" --actor lead --blocked-on "dep:$id" --json >/dev/null 2>&1 || true
out=$("$sb" close "$id" --actor lead --resolution Done --json); ok "$out" close
[ "$("$sb" get "$id" --json | jq -r .state)" = "done" ] || die "close → done"
[ "$("$sb" get "$dep" --json | jq -r .state)" = "ready" ] || die "close cascade did not release dependent"
expect_rc 3 "close from done (workflow arbiter)" "$sb" close "$id" --actor lead --json

# Cascade on CANCEL; cancelled = Done + Won't Do.
blk=$("$sb" create --title "Blocker B" --actor a --json | jq -r .task)
"$sb" promote "$blk" --actor lead --json >/dev/null
dep2=$("$sb" create --title "Dependent B" --actor a --blocked-by "$blk" --json | jq -r .task)
"$sb" promote "$dep2" --actor lead --json >/dev/null
"$sb" block "$dep2" --actor lead --blocked-on "dep:$blk" --json >/dev/null 2>&1 || true
out=$("$sb" cancel "$blk" --actor lead --json); ok "$out" cancel
[ "$("$sb" get "$blk" --json | jq -r .state)" = "cancelled" ] || die "cancel → Done + Won't Do"
echo "$out" | jq -e --arg d "$dep2" '.cascaded | index($d)' >/dev/null || die "cancel cascade missing dependent"
[ "$("$sb" get "$dep2" --json | jq -r .state)" = "ready" ] || die "cancel cascade did not release dependent"

out=$("$sb" reinstate "$blk" --actor lead --json); ok "$out" reinstate
[ "$("$sb" get "$blk" --json | jq -r .state)" = "backlog" ] || die "reinstate → backlog"

# Cancelling in-progress work ends the claim: reinstate + promote must
# leave the card claimable, not stranded behind a dead holder.
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
out=$("$sb" list --json); ok "$out" list
expect_rc 4 "missing issue" "$sb" get SEED-999 --json

# The required Blocked status is a declared convention: a workflow without
# it refuses with remediation (exit 5, distinguished from an illegal move).
kill $srv 2>/dev/null || true; wait $srv 2>/dev/null || true
port2=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
FAKE_JIRA_NO_BLOCKED=1 python3 "$dir/testdata/fake-jira" "$port2" & srv=$!
export JIRA_BASE_URL="http://127.0.0.1:$port2"
for i in $(seq 1 50); do curl -sf "$JIRA_BASE_URL/rest/api/3/search?jql=x" >/dev/null 2>&1 && break; sleep 0.1; done
nb=$("$sb" create --title "No blocked status" --actor a --json | jq -r .task)
expect_rc 5 "missing Blocked workflow status" "$sb" block "$nb" --actor lead --blocked-on manual:x --json

say "OK: all required verbs; backlog-landing create, accountId mapping, minted-token fence (incl. rotation), contention, parking, lockout, close+cancel cascades, close review-only, workflow-arbitrated transitions, missing-status refusal all enforced"
