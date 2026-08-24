#!/bin/sh
# Contract test for the paperclip plugin against the fake server
# (offline, CI-safe). The corpus itself lives in corpus.sh, shared
# verbatim with live-test.sh so the offline and live suites cannot drift
# apart; only fake-only scenarios live here.
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
command -v python3 >/dev/null 2>&1 || { echo "paperclip-test: SKIP (python3 unavailable)"; exit 0; }
command -v jq >/dev/null 2>&1 || { echo "paperclip-test: SKIP (jq unavailable)"; exit 0; }
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
python3 "$dir/testdata/fake-paperclip" "$port" & srv=$!
trap 'kill $srv 2>/dev/null || true' EXIT
export PAPERCLIP_API_URL="http://127.0.0.1:$port" PAPERCLIP_API_KEY=test PAPERCLIP_COMPANY_ID=co-1 PAPERCLIP_DEFAULT_GOAL_ID=goal-1
for i in $(seq 1 50); do curl -sf "$PAPERCLIP_API_URL/api/health" >/dev/null 2>&1 && break; sleep 0.1; done

sb="$dir/bin/seed-backend"
PREFIX="paperclip-test"
. "$dir/corpus.sh"

# One agent row per corpus actor, named after the actor: claim resolves
# --actor against this roster, the same production path live-test uses.
for a in $CORPUS_ACTORS; do
  curl -sS -X POST "$PAPERCLIP_API_URL/api/companies/co-1/agents" \
    -H 'Content-Type: application/json' \
    -d "$(jq -nc --arg n "$a" '{name: $n, role: "engineer"}')" >/dev/null
done

# Fake-only fidelity guards: these encode the live behaviours that let
# adapter drift pass unnoticed, so a future fake edit cannot quietly
# re-hide them.
probe=$("$sb" create --title "fidelity probe" --actor a --json | jq -r .task)
code=$(curl -sS -o /dev/null -w '%{http_code}' "$PAPERCLIP_API_URL/api/companies/co-1/issues/$probe")
[ "$code" = "404" ] || die "fake serves company-scoped single-issue GET ($code): live 404s, the fake must too"
code=$(curl -sS -o /dev/null -w '%{http_code}' -X PATCH "$PAPERCLIP_API_URL/api/companies/co-1/issues/$probe" \
  -H 'Content-Type: application/json' -d '{"status":"todo"}')
[ "$code" = "404" ] || die "fake serves company-scoped single-issue PATCH ($code): live 404s, the fake must too"
# The silent-drop behaviour itself: a PATCH naming `state` must 200 and
# change nothing, which is exactly why the corpus re-reads.
curl -sS -o /dev/null -X PATCH "$PAPERCLIP_API_URL/api/issues/$probe" \
  -H 'Content-Type: application/json' -d '{"state":"todo"}'
[ "$("$sb" get "$probe" --json | jq -r .state)" = "backlog" ] || die "fake honoured a 'state' PATCH: live ignores it"
# Documents are compare-and-swap: a blind update must be refused.
curl -sS -o /dev/null -X PUT "$PAPERCLIP_API_URL/api/issues/$probe/documents/seed" \
  -H 'Content-Type: application/json' -d '{"format":"markdown","body":"{}"}'
code=$(curl -sS -o /dev/null -w '%{http_code}' -X PUT "$PAPERCLIP_API_URL/api/issues/$probe/documents/seed" \
  -H 'Content-Type: application/json' -d '{"format":"markdown","body":"{\"x\":1}"}')
[ "$code" = "409" ] || die "fake allowed a document update without baseRevisionId ($code): live 409s"
"$sb" cancel "$probe" --actor lead --json >/dev/null

run_corpus

say "OK: all required verbs; atomic document fence (incl. rotation), checkout-aware ready, ancestry, server-arbitrated transitions, lockout, cascades, both event shapes"
