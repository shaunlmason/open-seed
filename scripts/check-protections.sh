#!/bin/sh
# Read-back of the server-side protections (plans/os-7b6afa4d.md): the
# handbook §1 checklist applied per os-18135882, re-verified against the
# GitHub API so a later admin relaxation is flagged, not assumed.
#
# check-protections.sh [-repo owner/name] [-api-base URL] [--write-halt]
#
#   --write-halt  on drift, also write the HALT marker to the state ref
#                 (the engine's way: one commit on the ref head, seed
#                 identity, HALT file plus one run-log line), so every
#                 mutating verb refuses until `seed state resume`.
#
# Exit codes: 0 all pass (or a degraded case, named); 3 an assertion
# failed (findings printed, and HALT written with --write-halt); 1 a usage
# error (no repo, bad args). Degraded, per the plan's step 2: the check must
# not wall up a repo where it cannot see (the engine-absent posture), so a
# token that 401/402/403s on the protection or rulesets reads, a repo with
# no seed-state ref (fresh instantiation, `seed init` not run), and an API
# that cannot be read at all (no usable HTTP code) each print a named
# WARNING and exit 0. A missing token is not fatal either: a public repo's
# protection reads work unauthenticated, so an empty token proceeds and lets
# the read paths degrade. A ruleset name alone is never sufficient: enforcement and rule types are asserted, so a
# renamed-but-weakened ruleset still fails.

set -u

REPO="${GITHUB_REPOSITORY:-}"
API_BASE="${GITHUB_API_BASE:-https://api.github.com}"
WRITE_HALT=0

usage() {
  echo "usage: check-protections.sh [-repo owner/name] [-api-base URL] [--write-halt]" >&2
}

while [ $# -gt 0 ]; do
  case "$1" in
  -repo) REPO="${2:?}"; shift 2 ;;
  -api-base) API_BASE="${2:?}"; shift 2 ;;
  --write-halt) WRITE_HALT=1; shift ;;
  *) echo "check-protections: unknown argument $1"; usage; exit 1 ;;
  esac
done

if [ -z "$REPO" ]; then
  echo "check-protections: no repository (set GITHUB_REPOSITORY or -repo)"
  usage
  exit 1
fi

# The token is a preference, not a requirement: a public repo's protection
# and ruleset reads work unauthenticated, and an empty/unusable token or an
# API the token cannot read all degrade (named, exit 0) rather than hard-red
# the gate (the plan's step 2). gh uses GH_TOKEN, else GITHUB_TOKEN.
[ -n "${GH_TOKEN:-}" ] || GH_TOKEN="${GITHUB_TOKEN:-${SEED_GH_TOKEN:-}}"

# The API base is a real knob (GHES instances, test doubles): gh's
# --hostname is the host of the base, and anything other than
# api.github.com is addressed under /api/v3 (the GHES API path).
GH_HOST=""
GH_BASEPATH=""
case "$API_BASE" in
https://api.github.com|http://api.github.com) : ;;
https://*|http://*)
  GH_HOST=${API_BASE#*://}; GH_HOST=${GH_HOST%/}; GH_HOST=${GH_HOST%%/*}
  [ "$GH_HOST" = github.com ] || GH_BASEPATH=/api/v3/ ;;
esac

# api <endpoint>: writes the JSON body to $API_BODY_FILE and sets
# $API_CODE (empty when the request itself failed — network down — which
# the caller treats as an error, not a finding). A 403/404 is a finding,
# not a crash: gh prints an "HTTP <code>" line to stderr and exits
# non-zero, and the body is empty or the API's error object.
API_CODE=""
API_BODY_FILE=$(mktemp) || exit 1
API_ERR=$(mktemp) || exit 1
trap 'rm -f "$API_BODY_FILE" "$API_ERR"' EXIT
api() {
  : > "$API_BODY_FILE"
  if gh api ${GH_HOST:+--hostname "$GH_HOST"} "$GH_BASEPATH$1" -H "Accept: application/vnd.github+json" >"$API_BODY_FILE" 2>"$API_ERR"; then
    API_CODE=200
  else
    API_CODE=$(sed -n 's/.*HTTP \([0-9][0-9][0-9]\).*/\1/p' "$API_ERR" | tail -1)
  fi
}

# read_degraded <what> <how-not-to-see>: a read that came back with no
# usable HTTP code (gh failed before the request — no token gh can use,
# network down) degrades like a 403: the check cannot see, so it names the
# gap and exits 0. The plan's step 2: the check must not wall up a repo
# where it cannot see, mirroring the repo's engine-absent posture.
# (exit 1 is reserved for usage errors: no repo, no token, bad args.)
read_degraded() {
  echo "WARNING: $1 unreadable (cannot see the API: $2) — degraded, check not run"
  exit 0
}

failures=0
FAIL_LINES=""
say_ok()   { echo "ok: $1"; }
say_fail() {
  echo "FAIL: $1: $2"
  failures=$((failures + 1))
  FAIL_LINES="$FAIL_LINES$2\n"
}

# --- state ref presence -----------------------------------------------------
# The seed-state ruleset guards a ref; without the ref a fresh instantiation
# has nothing to guard (seed init not run, protections not yet applied). The
# check cannot see, so it names the gap and exits 0 (the plan's step 2): a
# repo still being set up must not be walled with a HALT. In the maintenance
# context the workflow gate already filters this out, but standalone runs hit
# it, so the check degrades on its own. (A repo that HAS the seed-state ref
# but is missing a protection is a real drift: that falls through to exit 3.)
state_ref_present() {
  # Against $REPO explicitly, never the caller's cwd origin: the check
  # must see the repo it is pointed at, wherever it runs from.
  git ls-remote --exit-code "https://x-oauth:${GH_TOKEN}@github.com/$REPO.git" seed-state >/dev/null 2>&1
}

if ! state_ref_present; then
  echo "WARNING: no seed-state ref on origin — degraded (fresh instantiation, seed init not run); protections not yet verifiable"
  exit 0
fi

# --- main branch protection -----------------------------------------------
api "repos/$REPO/branches/main/protection"
main=$(cat "$API_BODY_FILE" 2>/dev/null)
case "$API_CODE" in
401|402|403)
  echo "WARNING: the API answered $API_CODE on branches/main/protection (the token cannot read protections) — degraded, check not run"
  exit 0 ;;
404)
  say_fail "main" "no main branch protection (apply the handbook §1 checklist)"
  main=null ;;
'')
  read_degraded "branches/main/protection" "$(head -c 200 "$API_ERR" 2>/dev/null)" ;;
esac

if [ "$main" != "null" ] && [ -n "$main" ]; then
  main_ctx=$(printf '%s' "$main" | jq -r '.required_status_checks.contexts // [] | join(" ")')
  for want in check verify; do
    case " $main_ctx " in
    *" $want "*) say_ok "main: required check '$want'" ;;
    *) say_fail "main" "required check '$want' missing (contexts: $main_ctx)" ;;
    esac
  done
  [ "$(printf '%s' "$main" | jq '.allow_force_pushes.enabled')" = false ] \
    && say_ok "main: force pushes blocked" \
    || say_fail "main" "force pushes allowed (allow_force_pushes.enabled)"
  [ "$(printf '%s' "$main" | jq '.allow_deletions.enabled')" = false ] \
    && say_ok "main: branch deletion blocked" \
    || say_fail "main" "branch deletion allowed (allow_deletions.enabled)"
  [ "$(printf '%s' "$main" | jq '.required_conversation_resolution.enabled')" = true ] \
    && say_ok "main: conversation resolution required" \
    || say_fail "main" "conversation resolution not required"
  [ "$(printf '%s' "$main" | jq '.required_pull_request_reviews.require_code_owner_reviews')" = true ] \
    && say_ok "main: code-owner review required" \
    || say_fail "main" "code-owner review not required (require_code_owner_reviews)"
  [ "$(printf '%s' "$main" | jq '.enforce_admins.enabled')" = false ] \
    && say_ok "main: admins not enforced (scheduled jobs push via their token)" \
    || say_fail "main" "enforce_admins enabled — scheduled jobs would be walled"
fi

# --- rulesets -----------------------------------------------------------------
# The state ref exists (checked above), so a missing ruleset is a real
# drift, not a setup-pending case. Match by ref pattern, never by name.
api "repos/$REPO/rulesets?per_page=100"
  sets=$(cat "$API_BODY_FILE" 2>/dev/null)
  case "$API_CODE" in
  401|402|403)
    echo "WARNING: the API answered $API_CODE on the rulesets endpoint (the token needs Administration: read-only) — degraded, rulesets not asserted"
    exit 0 ;;
  404)
    say_fail "rulesets" "no rulesets exist on this repository"
    sets='[]' ;;
  '')
    read_degraded "the rulesets endpoint" "$(head -c 200 "$API_ERR" 2>/dev/null)" ;;
  *)
    [ -n "$sets" ] || sets='[]' ;;
  esac
  # The list endpoint carries only name/target/enforcement/id: the ref
  # patterns (conditions.ref_name.include) and the rule types live on the
  # per-ruleset detail endpoint, which is where "a renamed-but-weakened
  # ruleset still fails" is decided. Fetch every ruleset's detail; match
  # by ref pattern, never by name.
  details='[]'
  for id in $(printf '%s' "$sets" | jq -r '.[].id'); do
    api "repos/$REPO/rulesets/$id"
    d=$(cat "$API_BODY_FILE" 2>/dev/null)
    [ -n "$d" ] || continue
    details=$(printf '%s' "$details" | jq -c --argjson x "$d" '. + [$x]')
  done
  sets=$details

  # One jq over the details: the first ruleset whose ref include matches the
  # pattern. The payload's ref patterns live in conditions.ref_name.include
  # (there is no target_id field to match against).
  find_set() {
    printf '%s' "$sets" | jq -c --arg p "$1" '[.[] | select(.conditions.ref_name.include | index($p))] | first // empty'
  }
  set_ok() { # $1 label, $2 pattern, $3... required rule types
    label=$1; pattern=$2; shift 2
    st=$(find_set "$pattern")
    if [ -z "$st" ]; then
      say_fail "$label" "no ruleset matching '$pattern'"
      return
    fi
    enf=$(printf '%s' "$st" | jq -r .enforcement)
    rules=$(printf '%s' "$st" | jq -r '[.rules[].type] | sort | join(" ")')
    bad=""
    [ "$enf" = active ] || bad="enforcement is '$enf', not active"
    for want in "$@"; do
      case " $rules " in
      *" $want "*) : ;;
      *) bad="$bad rule type '$want' missing (has: $rules)"; break ;;
      esac
    done
    if [ -z "$bad" ]; then
      say_ok "$label: ruleset '$(printf '%s' "$st" | jq -r .name)' active on '$pattern' (rules: $rules)"
    else
      say_fail "$label" "${bad# }"
    fi
  }
  set_ok "seed-state" "refs/heads/seed-state" deletion non_fast_forward
  set_ok "seed-anchor" "refs/tags/seed-anchor/*" deletion non_fast_forward update
  set_ok "release-tags" "refs/tags/v*" deletion non_fast_forward update

  # the read-back summary for the job ----------------------------------------
  echo
  echo "protections read-back (rulesets):"
  printf '%s' "$sets" | jq -r 'if type == "array" and length > 0 then .[] | "  \(.name) [\(.enforcement)] \(.conditions.ref_name.include | join(",")): \([.rules[].type] | join(","))" else "  (none)" end'
  echo "protections read-back (main): required=[$(printf '%s' "$main" | jq -r '(.required_status_checks.contexts // []) | join(",")')] force=$(printf '%s' "$main" | jq -r '.allow_force_pushes.enabled | if . == null then "n/a" else tostring end') delete=$(printf '%s' "$main" | jq -r '.allow_deletions.enabled | if . == null then "n/a" else tostring end') convo=$(printf '%s' "$main" | jq -r '.required_conversation_resolution.enabled | if . == null then "n/a" else tostring end') codeowner=$(printf '%s' "$main" | jq -r '.required_pull_request_reviews.require_code_owner_reviews | if . == null then "n/a" else tostring end') admins=$(printf '%s' "$main" | jq -r '.enforce_admins.enabled | if . == null then "n/a" else tostring end')"


# --- the drift's write ---------------------------------------------------------
# The engine's way (gitx CommitTree + stateref Mutate): one commit on the
# state ref's head, seed identity, HALT file plus one run-log line in the
# same commit. The replay lint tolerates HALT-only commits (no card changed),
# and the halt event is the run-log's own verb. The commit's parent is the
# ref's current head, so the push is a fast-forward to the protected ref
# (the ruleset allows fast-forward, refuses force and delete).
write_halt() {
  # A self-contained temporary repository, not the caller's: fetch the
  # state ref INTO it (fetching in the caller's repo would leave an
  # origin/seed-state ref behind and root the commit in the wrong
  # repository's history).
  wt=$(mktemp -d) || { echo "check-protections: mktemp for the HALT worktree failed"; return 1; }
  if ! git -C "$wt" init -q; then
    echo "check-protections: cannot init the HALT worktree"; rm -rf "$wt"; return 1
  fi
  git -C "$wt" fetch -q "https://x-oauth:${GH_TOKEN}@github.com/$REPO.git" seed-state:refs/remotes/origin/seed-state 2>/dev/null \
    || { echo "check-protections: cannot fetch the state ref for the HALT write"; rm -rf "$wt"; return 1; }
  git -C "$wt" worktree add --detach "$wt/work" origin/seed-state >/dev/null 2>&1 \
    || { echo "check-protections: cannot open the state worktree for the HALT write"; rm -rf "$wt"; return 1; }
  wt="$wt/work"
  {
    echo "protections drift detected by seed-maintenance (check-protections):"
    printf '%b' "$FAIL_LINES" | while IFS= read -r line; do [ -n "$line" ] && printf -- '- %s\n' "$line"; done
  } > "$wt/HALT"
  printf '{"actor":"seed-maintenance","data":{"failures":%d},"task":"","ts":"%s","verb":"halt"}\n' \
    "$failures" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >> "$wt/run-log.jsonl"
  git -C "$wt" -c user.name=seed -c user.email=seed@open-seed add HALT run-log.jsonl
  git -C "$wt" -c user.name=seed -c user.email=seed@open-seed commit -q -m "protections drift: writing HALT"
  if ! git -C "$wt" push -q "https://x-oauth:${GH_TOKEN}@github.com/$REPO.git" HEAD:refs/heads/seed-state 2>/dev/null; then
    echo "check-protections: the HALT push was refused (contention or a rule) — inspect seed-state"
    rm -rf "$(dirname "$wt")"
    return 1
  fi
  git -C "$(dirname "$wt")" worktree remove "$wt" --force 2>/dev/null || rm -rf "$(dirname "$wt")"
  echo "check-protections: HALT written on seed-state (mutating verbs refuse until 'seed state resume')"
}

# --- verdict --------------------------------------------------------------------
if [ "$failures" -gt 0 ]; then
  if [ "$WRITE_HALT" = 1 ]; then
    write_halt || echo "check-protections: the HALT write failed (the findings above stand)"
  fi
  exit 3
fi
echo "check-protections: all protections verified on $REPO"
