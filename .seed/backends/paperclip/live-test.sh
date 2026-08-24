#!/bin/sh
# Live contract test: the SAME corpus as test.sh, driven through the
# adapter against a real Paperclip instance (plan os-2c0c474c).
#
# Two modes, both credential-free:
#   PAPERCLIP_API_URL=<url> sh live-test.sh    use an instance you run
#   sh live-test.sh --onboard                  boot one via npx
# Self-skips with exit 0 and an explicit message when neither is
# available, so `make check` stays green and offline.
#
# A quickstart instance runs deploymentMode=local_trusted, which serves
# the API unauthenticated over loopback: no browser login, no device-code
# flow, no API key. The bootstrap-CEO invite is required only in
# `authenticated` mode.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)

PINNED_PAPERCLIP="2026.817.0"
PREFIX="paperclip-live"
say() { printf '%s: %s\n' "$PREFIX" "$*"; }

onboard=0
[ "${1:-}" = "--onboard" ] && onboard=1

command -v jq >/dev/null 2>&1 || { say "SKIP: jq unavailable"; exit 0; }
command -v curl >/dev/null 2>&1 || { say "SKIP: curl unavailable"; exit 0; }

work=""
srv=""
srvgroup=""
# npx execs the server as a CHILD process that does not die with the
# wrapper, so killing $srv alone leaves a live Paperclip (and an
# embedded Postgres) behind: fatal in CI, rude on a workstation. Kill the
# whole process group where the platform allows it, and fall back to
# reaping children plus a pattern scoped to OUR data dir, never a bare
# "paperclip" match that could hit somebody else's instance.
cleanup() {
  [ -n "$srvgroup" ] && kill -TERM "-$srvgroup" 2>/dev/null || true
  if [ -n "$srv" ]; then
    pkill -P "$srv" 2>/dev/null || true
    kill "$srv" 2>/dev/null || true
  fi
  [ -n "$work" ] && pkill -f "paperclipai@$PINNED_PAPERCLIP.*$work" 2>/dev/null || true
  # Do not remove the tree until the server has let go of it.
  i=0
  while [ "$i" -lt 20 ]; do
    curl -sf -m 1 "http://127.0.0.1:3100/api/health" >/dev/null 2>&1 || break
    i=$((i + 1)); sleep 1
  done
  [ -n "$work" ] && rm -rf "$work" 2>/dev/null || true
}
trap cleanup EXIT

if [ -z "${PAPERCLIP_API_URL:-}" ]; then
  if [ "$onboard" = "0" ]; then
    say "SKIP: no PAPERCLIP_API_URL and --onboard not requested."
    say "SKIP: to run it: sh $dir/live-test.sh --onboard  (needs Node >= 20 and network;"
    say "SKIP: boots paperclipai@$PINNED_PAPERCLIP with embedded Postgres on loopback)"
    exit 0
  fi
  command -v node >/dev/null 2>&1 || { say "SKIP: --onboard needs Node >= 20, none on PATH"; exit 0; }
  nodemaj=$(node -p 'process.versions.node.split(".")[0]' 2>/dev/null || echo 0)
  [ "$nodemaj" -ge 20 ] 2>/dev/null || { say "SKIP: --onboard needs Node >= 20 (found $nodemaj)"; exit 0; }
  work=$(mktemp -d)
  say "onboarding paperclipai@$PINNED_PAPERCLIP (embedded Postgres, this takes a minute)"
  if command -v setsid >/dev/null 2>&1; then
    setsid env HOME="$work/home" PAPERCLIP_HOME="$work/pc" \
      npx -y "paperclipai@$PINNED_PAPERCLIP" onboard --yes --data-dir "$work/pc" >"$work/server.log" 2>&1 &
    srv=$!
    srvgroup=$srv
  else
    env HOME="$work/home" PAPERCLIP_HOME="$work/pc" \
      npx -y "paperclipai@$PINNED_PAPERCLIP" onboard --yes --data-dir "$work/pc" >"$work/server.log" 2>&1 &
    srv=$!
  fi
  PAPERCLIP_API_URL="http://127.0.0.1:3100"
  i=0
  while [ "$i" -lt 180 ]; do
    curl -sf -m 2 "$PAPERCLIP_API_URL/api/health" >/dev/null 2>&1 && break
    kill -0 "$srv" 2>/dev/null || { say "SKIP: onboard exited early (no network?); see $work/server.log"; exit 0; }
    i=$((i + 1)); sleep 1
  done
  curl -sf -m 2 "$PAPERCLIP_API_URL/api/health" >/dev/null 2>&1 \
    || { say "SKIP: instance did not become healthy within 180s"; exit 0; }
fi
export PAPERCLIP_API_URL

health=$(curl -sf -m 5 "$PAPERCLIP_API_URL/api/health" 2>/dev/null) \
  || { say "SKIP: no Paperclip at $PAPERCLIP_API_URL"; exit 0; }
ver=$(printf '%s' "$health" | jq -r '.version // "unknown"')
mode=$(printf '%s' "$health" | jq -r '.deploymentMode // "unknown"')
say "paperclip $ver ($mode) at $PAPERCLIP_API_URL (validated pin: $PINNED_PAPERCLIP)"
[ "$ver" = "$PINNED_PAPERCLIP" ] || say "WARNING: $ver differs from the validated pin $PINNED_PAPERCLIP: running anyway to surface drift"

pc() { # METHOD PATH [BODY]
  m=$1; p=$2; d=${3:-}
  curl -sS -X "$m" -H "Authorization: Bearer ${PAPERCLIP_API_KEY:-local}" \
    -H 'Content-Type: application/json' ${d:+-d "$d"} "$PAPERCLIP_API_URL/api$p"
}

# Fresh company per run: the corpus assumes an empty board, and a live
# instance may already carry work.
stamp=$(date -u +%Y%m%d%H%M%S 2>/dev/null || echo now)
PAPERCLIP_COMPANY_ID=$(pc POST /companies "$(jq -nc --arg n "seed-corpus-$stamp" '{name: $n}')" | jq -r '.id // empty')
[ -n "$PAPERCLIP_COMPANY_ID" ] || { say "FAIL: could not create a company"; exit 1; }
PAPERCLIP_DEFAULT_GOAL_ID=$(pc POST "/companies/$PAPERCLIP_COMPANY_ID/goals" '{"title":"seed corpus goal"}' | jq -r '.id // empty')
[ -n "$PAPERCLIP_DEFAULT_GOAL_ID" ] || { say "FAIL: could not create the default goal"; exit 1; }
PAPERCLIP_API_KEY="${PAPERCLIP_API_KEY:-local}"
export PAPERCLIP_COMPANY_ID PAPERCLIP_DEFAULT_GOAL_ID PAPERCLIP_API_KEY

sb="$dir/bin/seed-backend"
. "$dir/corpus.sh"

# One agent row per corpus actor, named after the actor: `claim` resolves
# --actor against this roster by name/urlKey, so the suite exercises the
# production lookup rather than a test-only bypass. `role` is a server
# enum; `engineer` is a member of it.
# Each agent is then PAUSED, which is what keeps the corpus runnable at
# all (F5). Paperclip's checkout is not a passive lock: assigning an
# issue wakes the agent, its adapter runs and fails (no runtime is
# configured here), and `recovery.reconcile_stranded_assigned_issue`
# moves the issue in_progress -> blocked within ~10s. A paused agent is
# not dispatched to, so the claim holds. This is semantically right, not
# a trick: seed drives the work, so Paperclip must not also dispatch it.
# `status` is ignored on create, so the pause is a second request.
for a in $CORPUS_ACTORS; do
  got=$(pc POST "/companies/$PAPERCLIP_COMPANY_ID/agents" "$(jq -nc --arg n "$a" '{name: $n, role: "engineer"}')" | jq -r '.id // empty')
  [ -n "$got" ] || { say "FAIL: could not provision agent '$a'"; exit 1; }
  paused=$(pc PATCH "/agents/$got" '{"status":"paused"}' | jq -r '.status // empty')
  [ "$paused" = "paused" ] || { say "FAIL: could not pause agent '$a' (got '${paused:-none}')"; exit 1; }
done
say "provisioned $(printf '%s' "$CORPUS_ACTORS" | wc -w | tr -d ' ') corpus agents in company $PAPERCLIP_COMPANY_ID"

# --- quiescence gate (F5) ---------------------------------------------
# The platform moves issues on its own: a checked-out issue was observed
# going in_progress -> blocked seconds after checkout, with the agent's
# own runtimeConfig.heartbeat.enabled already false. The corpus holds a
# claim across release, reclaim, rotation, and review, so a live run that
# does not first establish quiescence reports flakes as contract
# failures. This gate measures rather than assumes: it holds a real claim
# for a settle window and refuses to run the corpus if the platform moved
# it underneath us.
SETTLE=${PAPERCLIP_SETTLE_SECONDS:-20}
q=$("$sb" create --title "quiescence probe" --actor a --json | jq -r .task)
"$sb" promote "$q" --actor lead --json >/dev/null
qtok=$("$sb" claim "$q" --actor agent-1 --json | jq -r .claim_token)
before=$("$sb" get "$q" --json | jq -r .state)
sleep "$SETTLE"
after=$("$sb" get "$q" --json | jq -r .state)
if [ "$before" != "$after" ]; then
  say "FAIL: platform moved a held claim $before -> $after within ${SETTLE}s."
  say "FAIL: the corpus cannot hold in_progress on this instance. Neutralize the"
  say "FAIL: mover (pause the corpus agents, or disable the instance heartbeat)"
  say "FAIL: and re-run; do not paper over it by relaxing the corpus."
  exit 1
fi
"$sb" release "$q" --actor agent-1 --token "$qtok" --json >/dev/null
"$sb" cancel "$q" --actor lead --json >/dev/null
say "quiescence ok: a held claim survived ${SETTLE}s unmoved"

run_corpus

say "OK: full corpus green against real paperclip $ver"
