# Shared contract corpus for the beads adapter (plan os-435d7b61).
# Sourced, never executed, by test.sh (fake bd) and live-test.sh (real
# bd), so the two suites cannot drift apart silently. Callers set:
#   sb      path to bin/seed-backend
#   PREFIX  log tag (beads-test / beads-live)
# and guarantee a `bd` on PATH with an empty store.

say() { printf '%s: %s\n' "${PREFIX:-beads-corpus}" "$*"; }
die() { say "FAIL: $*"; exit 1; }
ok() { echo "$1" | jq -e '.ok == true and .schema_version == "1.0"' >/dev/null || die "$2: bad envelope: $1"; }
expect_rc() { # rc_want desc cmd...
  want=$1; desc=$2; shift 2
  rc=0; "$@" >/dev/null 2>&1 || rc=$?
  [ "$rc" = "$want" ] || die "$desc: exit $rc, want $want"
}
# Comment counting is shape-agnostic: `bd comments` lists them on real bd
# (show carries only comment_count there) and the fake mirrors it.
comment_count() { bd comments "$1" --json 2>/dev/null | jq 'length'; }

run_corpus() {
  out=$("$sb" create --title "Test card" --body "the work" --priority P1 --squad core --actor a --json); ok "$out" create
  id=$(echo "$out" | jq -r .task); [ "$(echo "$out" | jq -r .state)" = "backlog" ] || die "create state"

  out=$("$sb" promote "$id" --actor lead --json); ok "$out" promote
  out=$("$sb" ready --actor a --json)
  [ "$(echo "$out" | jq -r '.tasks[0].task')" = "$id" ] || die "promoted card not ready"

  out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" claim
  tok=$(echo "$out" | jq -r .claim_token)
  case "$tok" in tok:?*) ;; *) die "token shape: $tok" ;; esac

  expect_rc 2 "rival claim" "$sb" claim "$id" --actor agent-2 --json
  # The fence: right actor + wrong/missing token must exit 6 (P1 finding).
  expect_rc 6 "no-token transition" "$sb" transition "$id" --to review --actor agent-1 --json
  expect_rc 6 "wrong-token transition" "$sb" transition "$id" --to review --actor agent-1 --token bogus --json
  expect_rc 6 "wrong-actor transition" "$sb" transition "$id" --to review --actor agent-2 --token "$tok" --json
  expect_rc 6 "no-token comment on claimed card" "$sb" comment "$id" --body x --actor agent-1 --json
  # Close is review→done only (P1 finding): refuse from in_progress.
  expect_rc 3 "close from in_progress" "$sb" close "$id" --actor lead --resolution x --json

  out=$("$sb" comment "$id" --body "progress" --actor agent-1 --token "$tok" --json); ok "$out" comment
  [ -n "$(echo "$out" | jq -r '.comment_id // empty')" ] || die "comment_id missing from envelope"
  out=$("$sb" attach-evidence "$id" --kind commit --ref abc123 --actor agent-1 --token "$tok" --json); ok "$out" attach-evidence
  [ -n "$(echo "$out" | jq -r '.evidence_id // empty')" ] || die "evidence_id missing from envelope"
  out=$("$sb" lease-renew "$id" --actor agent-1 --token "$tok" --json); ok "$out" lease-renew

  # Plan parking persists blocked_on (P1 finding), plan-unblock releases it.
  out=$("$sb" transition "$id" --to blocked --actor agent-1 --token "$tok" --blocked-on plan:41 --json); ok "$out" park
  [ "$("$sb" get "$id" --json | jq -r '.card.blocked_on[0]')" = "plan:41" ] || die "blocked_on not persisted"
  expect_rc 3 "plan-unblock wrong pr" "$sb" plan-unblock "$id" --pr 99 --actor lead --json
  out=$("$sb" plan-unblock "$id" --pr 41 --actor lead --json); ok "$out" plan-unblock
  [ "$(echo "$out" | jq -r .state)" = "ready" ] || die "plan-unblock did not release"

  stale_tok=$tok
  out=$("$sb" claim "$id" --actor agent-1 --json); ok "$out" reclaim
  tok=$(echo "$out" | jq -r .claim_token)
  # Rotation: the pre-reclaim token is dead even for the same actor.
  [ "$stale_tok" != "$tok" ] || die "claim token did not rotate on reclaim"
  expect_rc 6 "stale pre-reclaim token" "$sb" comment "$id" --body late --actor agent-1 --token "$stale_tok" --json
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

  # Cancel cascades like close: a dependent parked on the cancelled
  # blocker is released (the spec attaches cascade to every
  # nonterminal→cancelled edge).
  out=$("$sb" create --title "Cancel me" --actor a --json); id2=$(echo "$out" | jq -r .task)
  "$sb" promote "$id2" --actor lead --json >/dev/null
  out=$("$sb" create --title "Blocked on cancelled" --actor a --json); dep2=$(echo "$out" | jq -r .task)
  "$sb" promote "$dep2" --actor lead --json >/dev/null
  "$sb" block "$dep2" --actor lead --blocked-on "dep:$id2" --json >/dev/null 2>&1 || true
  out=$("$sb" cancel "$id2" --actor lead --json); ok "$out" cancel
  echo "$out" | jq -e --arg d "$dep2" '.cascaded | index($d)' >/dev/null || die "cancel did not cascade to dependent"
  [ "$("$sb" get "$dep2" --json | jq -r .state)" = "ready" ] || die "cancel cascade did not release dependent"
  expect_rc 3 "cancel from terminal" "$sb" cancel "$id2" --actor lead --json
  out=$("$sb" reinstate "$id2" --actor lead --json); ok "$out" reinstate
  [ "$("$sb" get "$id2" --json | jq -r .state)" = "backlog" ] || die "reinstate → backlog"

  # Out-of-band events persist to the audit issue (P2 finding).
  out=$("$sb" event-append --event '{"ts":"t","actor":"x","verb":"probe"}' --json); ok "$out" event-append
  audit=$(bd list --json | jq -r '.[] | select((.labels // []) | index("seed:audit")) | .id')
  [ -n "$audit" ] || die "no audit issue created"
  [ "$(comment_count "$audit")" = "1" ] || die "event not persisted to audit issue"

  out=$("$sb" list --json); ok "$out" list
  # The list verb sees terminal cards too (bd list --all; default bd list
  # hides closed: a declared v1.2.2 variance).
  [ "$(echo "$out" | jq --arg id "$id" '[.tasks[] | select(.task == $id)] | length')" = "1" ] || die "closed card missing from list verb"
  # --state filters on the port state: done includes the closed card,
  # ready excludes it.
  out=$("$sb" list --state done --json); ok "$out" list-done
  [ "$(echo "$out" | jq --arg id "$id" '[.tasks[] | select(.task == $id)] | length')" = "1" ] || die "list --state done missing the closed card"
  [ "$(echo "$out" | jq '[.tasks[] | select(.state != "done")] | length')" = "0" ] || die "list --state done leaked other states"
  out=$("$sb" list --state ready --json)
  [ "$(echo "$out" | jq --arg id "$id" '[.tasks[] | select(.task == $id)] | length')" = "0" ] || die "list --state ready leaked the closed card"
  expect_rc 4 "missing card" "$sb" get bd-nope --json
}
