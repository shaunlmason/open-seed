#!/bin/sh
# Lint all orchestration artifacts (R9: a shipped convention and its validator
# are one deliverable). v1 scope: structural presence + the engine's spec
# lint. The full validator set (guardrails intersection, team files, plan
# pinning, receipts) lands with Phases 4-5 and runs here too.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
fail=0
say() { printf 'validate: %s\n' "$*"; }

for f in .seed/version .seed/engine.lock .seed/template.lock .seed/config.toml .seed/guardrails.yaml \
         .seed/port-schema/port.json .seed/port-schema/transitions.json \
         CODEOWNERS AGENTS.md CLAUDE.md Makefile; do
  if [ ! -f "$root/$f" ]; then
    say "MISSING $f"
    fail=1
  fi
done

# Engine-backed lints (spec, guardrails, teams, role variants, plans) when the
# engine can be bootstrapped; degrade with a warning otherwise (differentiator
# #4: the repo must work without the engine).
if out=$(cd "$root" && sh scripts/seed spec lint 2>&1); then
  say "$out"
  if out=$(cd "$root" && sh scripts/seed validate 2>&1); then
    say "$out"
  else
    say "FAIL: $out"
    fail=1
  fi
  if out=$(cd "$root" && sh scripts/seed sync --check 2>&1); then
    say "$out"
  else
    say "FAIL: $out"
    fail=1
  fi
  if out=$(cd "$root" && sh scripts/seed workflow validate --all 2>&1); then
    say "workflows ok"
  else
    say "FAIL: workflow validate: $out"
    fail=1
  fi
else
  say "WARNING: engine unavailable, skipped spec lint + validate ($out)"
fi

# Backend plugin contract tests (offline, fake substrates).
if command -v jq >/dev/null 2>&1; then
  for b in beads paperclip linear jira; do
    if out=$(sh "$root/.seed/backends/$b/test.sh" 2>&1); then
      say "$out"
    else
      say "FAIL: $b contract test: $out"
      fail=1
    fi
  done
else
  say "WARNING: jq unavailable, skipped backend contract tests"
fi

# Live beads validation (os-435d7b61): same corpus against a real bd
# install; self-skips (exit 0, explicit message) when bd is absent, so
# CI needs no new binaries.
if command -v jq >/dev/null 2>&1; then
  if out=$(sh "$root/.seed/backends/beads/live-test.sh" 2>&1); then
    say "$out"
  else
    say "FAIL: beads live test: $out"
    fail=1
  fi
fi

# Lifecycle shim structure (os-7792a002): every shim dir ships a README;
# fragments may reference only contract-defined hook paths; a README-only
# shim must declare the no-hook-point finding.
for d in "$root"/.seed/hooks/shims/*/; do
  [ -d "$d" ] || continue
  name=$(basename "$d")
  if [ ! -f "$d/README.md" ]; then
    say "FAIL: shim $name has no README.md"; fail=1; continue
  fi
  fragments=$(find "$d" -type f ! -name README.md)
  if [ -z "$fragments" ]; then
    grep -qi "no usable hook point" "$d/README.md"       || { say "FAIL: shim $name ships no fragment but its README does not declare the no-hook-point finding"; fail=1; }
    continue
  fi
  bad=$(grep -rho '\.seed/hooks/[A-Za-z0-9._-]*' $fragments | sort -u     | grep -v -E '^\.seed/hooks/(setup|run|teardown|post-create\.d|pre-merge\.d)$' || true)
  if [ -n "$bad" ]; then
    say "FAIL: shim $name references undefined hook paths: $bad"; fail=1
  fi
done

# Dispatcher routing contract (os-70028620): fake events through
# scripts/seed-dispatch-route: authorized cmd labels map to port
# invocations with label removal + provenance; unauthorized actors are
# refused; unknown commands are ignored; mirror label edits become
# requests, never state writes. Zero credentials.
if command -v jq >/dev/null 2>&1; then
  route="$root/scripts/seed-dispatch-route"
  ev() { printf '{"action":"labeled","label":{"name":"%s"},"sender":{"login":"alice"},"issue":{"number":7,"title":"card os-1234abcd","labels":[{"name":"seed:os-1234abcd"}]}}' "$1"; }
  out=$(ev cmd:promote | DISPATCH_SENDER_ASSOC=COLLABORATOR sh "$route") || { say "FAIL: dispatch-route authorized promote exited $?"; fail=1; }
  echo "$out" | grep -q '^scripts/seed task promote os-1234abcd --actor alice$' || { say "FAIL: dispatch-route promote invocation wrong: $out"; fail=1; }
  echo "$out" | grep -q -- '--remove-label cmd:promote' || { say "FAIL: dispatch-route one-shot label not removed"; fail=1; }
  echo "$out" | grep -q -- '--add-label by:agent' || { say "FAIL: dispatch-route provenance label missing"; fail=1; }
  echo "$out" | grep -q 'seed-dispatch' || { say "FAIL: dispatch-route sticky comment missing"; fail=1; }
  rc=0; out=$(ev cmd:promote | DISPATCH_SENDER_ASSOC=NONE sh "$route" 2>/dev/null) || rc=$?
  { [ "$rc" = 3 ] && [ -z "$out" ]; } || { say "FAIL: dispatch-route unauthorized actor not refused (rc=$rc out=$out)"; fail=1; }
  rc=0; out=$(ev cmd:frobnicate | DISPATCH_SENDER_ASSOC=OWNER sh "$route") || rc=$?
  { [ "$rc" = 0 ] && [ -z "$out" ]; } || { say "FAIL: dispatch-route unknown cmd not ignored (rc=$rc out=$out)"; fail=1; }
  out=$(ev state:done | DISPATCH_SENDER_ASSOC=OWNER sh "$route") || { say "FAIL: dispatch-route mirror edit errored"; fail=1; }
  echo "$out" | grep -q 'REQUEST' || { say "FAIL: dispatch-route mirror edit not treated as request"; fail=1; }
  echo "$out" | grep -q '^scripts/seed task' && { say "FAIL: dispatch-route mirror edit produced a state write"; fail=1; }
  rc=0; out=$(ev cmd:close | DISPATCH_SENDER_ASSOC=OWNER sh "$route") || rc=$?
  { [ "$rc" = 0 ] && echo "$out" | grep -q 'not label-routable' && ! echo "$out" | grep -q '^scripts/seed task'; } \
    || { say "FAIL: dispatch-route cmd:close must be refused with a comment, never routed (rc=$rc out=$out)"; fail=1; }
  evil='{"action":"labeled","label":{"name":"cmd:promote"},"sender":{"login":"alice"},"issue":{"number":7,"title":"x","labels":[{"name":"seed:os-12; rm -rf /"}]}}'
  rc=0; out=$(printf '%s' "$evil" | DISPATCH_SENDER_ASSOC=OWNER sh "$route" 2>/dev/null) || rc=$?
  { [ "$rc" = 3 ] && [ -z "$out" ]; } || { say "FAIL: dispatch-route crafted card id not refused (rc=$rc out=$out)"; fail=1; }
  say "dispatch-route: OK — authorized mapping, refusal, unknown-cmd ignore, mirror-edit-as-request, close-not-routable, injection refused"

  # Reviewer-lane conformance (D4.5): an app-identity approval satisfies
  # reviewer != implementer; a self-approval is refused; none = waiting.
  idcheck="$root/scripts/seed-review-identity"
  echo '[{"author":{"login":"seed-reviewer[bot]"},"state":"APPROVED"}]' | IMPLEMENTER=alice sh "$idcheck" >/dev/null     || { say "FAIL: review-identity rejected an app-identity approval"; fail=1; }
  rc=0; echo '[{"author":{"login":"alice"},"state":"APPROVED"}]' | IMPLEMENTER=alice sh "$idcheck" >/dev/null 2>&1 || rc=$?
  [ "$rc" = 3 ] || { say "FAIL: review-identity accepted a self-approval (rc=$rc)"; fail=1; }
  rc=0; echo '[]' | IMPLEMENTER=alice sh "$idcheck" >/dev/null 2>&1 || rc=$?
  [ "$rc" = 2 ] || { say "FAIL: review-identity wrong verdict on no reviews (rc=$rc)"; fail=1; }
  say "review-identity: OK — app approval passes, self-approval refused, none waits"
fi

[ "$fail" -eq 0 ] && say "ok" || exit 1
