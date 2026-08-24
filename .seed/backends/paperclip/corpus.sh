# Shared contract corpus for the paperclip adapter (plan os-2c0c474c).
# Sourced, never executed, by test.sh (fake server) and live-test.sh (a
# real Paperclip instance), so the two suites cannot drift apart silently.
# Callers set:
#   sb      path to bin/seed-backend
#   PREFIX  log tag (paperclip-test / paperclip-live)
# and guarantee a reachable API with an empty company, the four
# requires_env variables exported, and one agent row per CORPUS_ACTORS
# entry named after the actor: claim resolves --actor against that roster
# by name/urlKey (the production contract), so the suites exercise the
# same lookup a deployment uses rather than a test-only bypass.

# Every actor the corpus claims as. Harnesses provision one agent row per
# entry; the resolution contract turns the name back into the UUID that
# Paperclip's checkout endpoint requires.
CORPUS_ACTORS="a lead agent-1 agent-2 fresh-agent"

say() { printf '%s: %s\n' "${PREFIX:-paperclip-corpus}" "$*"; }
die() { say "FAIL: $*"; exit 1; }
ok() { echo "$1" | jq -e '.ok == true and .schema_version == "1.0"' >/dev/null || die "$2: bad envelope: $1"; }
expect_rc() { # rc_want desc cmd...
  want=$1; desc=$2; shift 2
  rc=0; "$@" >/dev/null 2>&1 || rc=$?
  [ "$rc" = "$want" ] || die "$desc: exit $rc, want $want"
}
# Native API helper: the corpus reaches past the port only to create the
# platform-native conditions the port cannot express (native dependency
# edges). Same routes against fake and real, which is the point.
pc_api() { # METHOD PATH [BODY]
  m=$1; p=$2; d=${3:-}
  curl -sS -X "$m" -H "Authorization: Bearer $PAPERCLIP_API_KEY" \
    -H 'Content-Type: application/json' ${d:+-d "$d"} "$PAPERCLIP_API_URL/api$p"
}
# F2 guard: a transition is only real if re-reading the issue shows it.
# Paperclip returns 200 for a PATCH naming a field it does not have, so
# asserting on the response alone let the `state` vs `status` drift pass
# unnoticed. Every state assertion in this corpus re-reads.
assert_state() { # id want desc
  got=$("$sb" get "$1" --json | jq -r .state)
  [ "$got" = "$2" ] || die "$3: state is $got, want $2 (re-read)"
}

run_corpus() {
  # Mandatory ancestry: the default goal covers parentless creates; with
  # no goal source at all, create refuses with remediation.
  out=$("$sb" create --title "Card A" --body work --priority P1 --actor a --json); ok "$out" create
  id=$(echo "$out" | jq -r .task); [ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"
  assert_state "$id" backlog "create"
  PAPERCLIP_DEFAULT_GOAL_ID= expect_rc 5 "parentless create without goal" "$sb" create --title "No goal" --actor a --json

  # --parent is a TASK id: the child inherits the parent's goal (no
  # default needed) and keeps the parent link; a bogus parent is
  # not_found, and ancestry walks child → parent → goal.
  out=$(PAPERCLIP_DEFAULT_GOAL_ID= "$sb" create --title "Child" --actor a --parent "$id" --json); ok "$out" create-child
  child=$(echo "$out" | jq -r .task)
  expect_rc 4 "create with bogus parent" env PAPERCLIP_DEFAULT_GOAL_ID= "$sb" create --title "Orphan" --actor a --parent PAP-999 --json
  out=$("$sb" ancestry "$child" --json); ok "$out" ancestry
  [ "$(echo "$out" | jq -r '.ancestors[0].id')" = "$id" ] || die "ancestry parent"
  [ "$(echo "$out" | jq -r '.ancestors[-1].kind')" = "goal" ] || die "ancestry does not end at a goal"

  # budget: the optional capability must answer for a real issue and
  # report a port-vocabulary state (ok/alert/paused).
  out=$("$sb" budget "$child" --json); ok "$out" budget
  case "$(echo "$out" | jq -r '.budget.state')" in ok|alert|paused) ;; *) die "budget state not in port vocabulary" ;; esac

  out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
  assert_state "$id" ready "promote"
  out=$("$sb" ready --actor agent-1 --json); ok "$out" ready
  [ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"

  out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
  tok1=$(echo "$out" | jq -r .claim_token)
  case "$tok1" in c-*) ;; *) die "minted token shape: $tok1" ;; esac
  assert_state "$id" in_progress "claim"

  # Checkout + mint is one exclusive claim: a second claim by the SAME
  # actor is contention, never a silent token rotation under the first.
  expect_rc 2 "same-actor duplicate claim" "$sb" claim "$id" --actor agent-1 --json

  # Actor resolution (F4): an actor with no agent row refuses with
  # remediation rather than inventing an id, and never silently claims.
  expect_rc 5 "claim as unregistered actor" "$sb" claim "$id" --actor nobody-here --json

  # Assignment is routing, not a claim: an issue ROUTED to an actor must
  # be both listed in that actor's ready set and claimable by them. The
  # two have to agree, or routed work is advertised and then refused.
  out=$("$sb" create --title "Routed" --actor a --json); routed=$(echo "$out" | jq -r .task)
  "$sb" promote "$routed" --actor lead --json >/dev/null
  ragent=$(pc_api GET "/companies/$PAPERCLIP_COMPANY_ID/agents" \
    | jq -r '[.[] | select(.name == "agent-2") | .id] | .[0]')
  pc_api PATCH "/issues/$routed" "$(jq -nc --arg a "$ragent" '{assigneeAgentId: $a}')" >/dev/null
  out=$("$sb" ready --actor agent-2 --json)
  echo "$out" | jq -e --arg t "$routed" '[.tasks[].task] | index($t)' >/dev/null \
    || die "issue routed to agent-2 missing from its ready set"
  out=$("$sb" claim "$routed" --actor agent-2 --json); ok "$out" claim-routed
  rtok=$(echo "$out" | jq -r .claim_token)
  "$sb" release "$routed" --actor agent-2 --token "$rtok" --json >/dev/null
  "$sb" cancel "$routed" --actor lead --json >/dev/null

  # Checkout-aware ready: a checked-out issue is not claimable work.
  out=$("$sb" ready --actor agent-2 --json)
  [ "$(echo "$out" | jq '.tasks | length')" = "0" ] || die "checked-out issue still in ready"
  expect_rc 2 "rival claim" "$sb" claim "$id" --actor agent-2 --json
  expect_rc 6 "no-token transition" "$sb" transition "$id" --to review --actor agent-1 --json
  expect_rc 6 "wrong-token transition" "$sb" transition "$id" --to review --actor agent-1 --token c-bogus --json
  expect_rc 3 "close from in_progress" "$sb" close "$id" --actor lead --json

  # The rotation case (D1): release (reap stand-in), same actor
  # reclaims: the old token must be dead though the identity is unchanged.
  out=$("$sb" release "$id" --actor agent-1 --token "$tok1" --json); ok "$out" release
  assert_state "$id" ready "release"
  out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" reclaim
  tok2=$(echo "$out" | jq -r .claim_token)
  [ "$tok1" != "$tok2" ] || die "token did not rotate on reclaim"
  expect_rc 6 "stale token after reclaim (same actor)" "$sb" comment "$id" --body late --actor agent-1 --token "$tok1" --json

  # comment and attach-evidence must return the ids verbs.json declares
  # (F10): the port's callers record them, so a missing id is a
  # conformance failure, not a cosmetic gap.
  out=$("$sb" comment "$id" --body progress --actor agent-1 --token "$tok2" --json); ok "$out" comment
  [ -n "$(echo "$out" | jq -r '.comment_id // empty')" ] || die "comment_id missing from envelope"
  out=$("$sb" attach-evidence "$id" --kind commit --ref abc123 --actor agent-1 --token "$tok2" --json); ok "$out" attach-evidence
  [ -n "$(echo "$out" | jq -r '.evidence_id // empty')" ] || die "evidence_id missing from envelope"
  out=$("$sb" lease-renew "$id" --actor agent-1 --token "$tok2" --json); ok "$out" lease-renew

  # Park on a plan, then state-shaped unpark.
  out=$("$sb" transition "$id" --to blocked --actor agent-1 --token "$tok2" --blocked-on plan:7 --json); ok "$out" park
  assert_state "$id" blocked "park"
  [ "$("$sb" get "$id" --json | jq -r '.card.blocked_on[0]')" = "plan:7" ] || die "blocked_on not persisted"
  expect_rc 3 "plan-unblock wrong pr" "$sb" plan-unblock "$id" --pr 99 --actor lead --json
  out=$("$sb" plan-unblock "$id" --pr 7 --actor lead --json); ok "$out" plan-unblock
  assert_state "$id" ready "plan-unblock"

  out=$("$sb" claim "$id" --actor agent-1 --json); tok3=$(echo "$out" | jq -r .claim_token)
  out=$("$sb" transition "$id" --to review --actor agent-1 --token "$tok3" --json); ok "$out" review
  assert_state "$id" review "review mapping (in_review)"

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
  assert_state "$id" done "close"
  assert_state "$dep" ready "cascade release"
  expect_rc 3 "close from done (assertTransition)" "$sb" close "$id" --actor lead --json

  out=$("$sb" create --title "Cancel me" --actor a --json); id2=$(echo "$out" | jq -r .task)
  out=$("$sb" cancel "$id2" --actor lead --json); ok "$out" cancel
  out=$("$sb" reinstate "$id2" --actor lead --json); ok "$out" reinstate
  assert_state "$id2" backlog "reinstate"

  # Cancel cascades like close: a dependent blocked solely on a cancelled
  # blocker must be released, not stay blocked forever.
  out=$("$sb" create --title "Blocker" --actor a --json); blk=$(echo "$out" | jq -r .task)
  out=$("$sb" create --title "Dependent of blocker" --actor a --blocked-by "$blk" --json); dep2=$(echo "$out" | jq -r .task)
  "$sb" promote "$dep2" --actor lead --json >/dev/null
  "$sb" block "$dep2" --actor lead --blocked-on "dep:$blk" --json >/dev/null
  out=$("$sb" cancel "$blk" --actor lead --json); ok "$out" cancel-cascade
  echo "$out" | jq -e --arg d "$dep2" '.cascaded | index($d)' >/dev/null || die "cancel did not report cascade"
  assert_state "$dep2" ready "cancel cascade release"

  # NATIVE Paperclip dependencies gate ready too: a todo issue whose
  # native blocker edge is nonterminal is not claimable work, even with
  # no seed bookkeeping entries (e.g. deps created in the Paperclip UI,
  # invisible to seed).
  n1=$(pc_api POST "/companies/$PAPERCLIP_COMPANY_ID/issues" \
    "{\"title\":\"native blocker\",\"goalId\":\"$PAPERCLIP_DEFAULT_GOAL_ID\",\"status\":\"backlog\"}" | jq -r .id)
  n2=$(pc_api POST "/companies/$PAPERCLIP_COMPANY_ID/issues" \
    "{\"title\":\"natively blocked\",\"goalId\":\"$PAPERCLIP_DEFAULT_GOAL_ID\",\"status\":\"backlog\",\"blockedByIssueIds\":[\"$n1\"]}" | jq -r .id)
  "$sb" promote "$n2" --actor lead --json >/dev/null
  out=$("$sb" ready --actor fresh-agent --json)
  echo "$out" | jq -e --arg t "$n2" '[.tasks[].task] | index($t) | not' >/dev/null || die "natively blocked issue in ready"
  "$sb" cancel "$n1" --actor lead --json >/dev/null
  out=$("$sb" ready --actor fresh-agent --json)
  echo "$out" | jq -e --arg t "$n2" '[.tasks[].task] | index($t)' >/dev/null || die "terminal native blocker still gates ready"

  # ready normalizes priorities to P0-P3 and sorts by them: loop.sh claims
  # the FIRST returned task, so a P0 must precede lower priorities.
  out=$("$sb" create --title "Low prio" --priority P3 --actor a --json); lo=$(echo "$out" | jq -r .task)
  out=$("$sb" create --title "Critical" --priority P0 --actor a --json); hi=$(echo "$out" | jq -r .task)
  "$sb" promote "$lo" --actor lead --json >/dev/null
  "$sb" promote "$hi" --actor lead --json >/dev/null
  out=$("$sb" ready --actor fresh-agent --json); ok "$out" ready-sorted
  [ "$(echo "$out" | jq -r '.tasks[0].task')" = "$hi" ] || die "P0 not first in ready"
  [ "$(echo "$out" | jq -r '.tasks[0].priority')" = "P0" ] || die "priority not normalized to P0"
  [ "$(echo "$out" | jq -r '.tasks[-1].priority')" = "P3" ] || die "priority not normalized to P3"

  # event-append in BOTH shapes (F7). verbs.json makes `task` optional, so
  # a taskless event must land somewhere durable too: comment-on-task
  # cannot serve it, and a silent success would be a lie.
  out=$("$sb" event-append --event "{\"ts\":\"t\",\"verb\":\"probe\",\"task\":\"$lo\"}" --json); ok "$out" event-append-task
  out=$("$sb" event-append --event '{"ts":"t","verb":"probe"}' --json); ok "$out" event-append-taskless
  [ -n "$(echo "$out" | jq -r '.sink // empty')" ] || die "taskless event-append does not name its sink"

  out=$("$sb" list --json); ok "$out" list
  expect_rc 4 "missing issue" "$sb" get PAP-999 --json
}
