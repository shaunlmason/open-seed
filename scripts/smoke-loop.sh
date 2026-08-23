#!/bin/bash
# End-to-end loop smoke (build plan Phase 6 done-when): instantiate the
# template into a temp repo, pre-merge a plan, create+promote a card, and run
# loop.sh --once with a deterministic fake harness. Asserts: card reaches
# review, receipt committed on the task branch, gate green, memory appended.
# No model, no secrets: safe for CI.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
say() { printf 'smoke-loop: %s\n' "$*"; }

export GIT_AUTHOR_NAME=smoke GIT_AUTHOR_EMAIL=s@s GIT_COMMITTER_NAME=smoke GIT_COMMITTER_EMAIL=s@s

# Instantiate: tracked template files into a fresh repo with a local origin.
inst="$work/inst"; mkdir -p "$inst"
git -C "$root" archive HEAD | tar -x -C "$inst"
git init -q --bare "$work/origin.git"
cd "$inst"
git init -q --initial-branch=main
git remote add origin "$work/origin.git"

# The task: a plan already merged on main (D3: plan gate before work).
id="os-aaaa0001"
cat > "plans/$id.md" <<'EOF'
# Plan

## Steps

1. Create src/hello.txt containing "hello from the loop".

## File Scope

- src/

## Acceptance Criteria

- src/hello.txt exists with the expected content.

## Validation Commands

- `test -f src/hello.txt`
- `grep -q "hello from the loop" src/hello.txt`
EOF
git add -A && git commit -q -m "template + plan $id"
git push -q -u origin main
scripts/seed init >/dev/null

# Card in ready.
created=$(scripts/seed task create --title "Say hello" --actor shaunlmason \
  --body "Create the greeting file per the plan.")
real_id=$(echo "$created" | jq -r .task)
# The plan path must match the card id: move the plan to the minted id.
git mv "plans/$id.md" "plans/$real_id.md"
git commit -q -m "plan for $real_id" && git push -q origin main
scripts/seed task promote "$real_id" --actor shaunlmason >/dev/null
id="$real_id"

# Deterministic fake harness: implements the plan, appends memory, commits.
cat > "$work/fake-harness" <<'EOF'
#!/bin/sh
set -eu
cat >/dev/null  # consume the prompt
mkdir -p src
echo "hello from the loop" > src/hello.txt
printf -- "- %s: the loop smoke ran (%s)\n" "$(date -u +%F)" "$SEED_TASK" >> memory/LEARNINGS.md
git add -A && git commit -q -m "implement $SEED_TASK per plan"
printf '{"result":"implemented per plan"}\n'
EOF
chmod +x "$work/fake-harness"

say "running loop.sh --once (unattended, tier L2)"
SEED_HARNESS_CMD="$work/fake-harness" bash scripts/loop.sh --actor smoke-agent --once

# Assertions.
state=$(scripts/seed task get "$id" | jq -r .state)
[ "$state" = "review" ] || { say "FAIL: card state is $state, want review"; exit 1; }
git fetch -q origin "seed/$id"
git cat-file -e "origin/seed/$id:receipts/$id.json" || { say "FAIL: receipt missing on task branch"; exit 1; }
git cat-file -p "origin/seed/$id:memory/LEARNINGS.md" | grep -q "loop smoke ran" \
  || { say "FAIL: memory append missing"; exit 1; }
git cat-file -p "origin/seed/$id:src/hello.txt" | grep -q "hello from the loop" \
  || { say "FAIL: implementation missing"; exit 1; }
evidence=$(scripts/seed task get "$id" | jq -r .card.body)
echo "$evidence" | grep -q "Evidence" || { say "FAIL: evidence not attached"; exit 1; }

# Builtin id conformance (os-61967950): comment and attach-evidence must
# return the ids verbs.json declares, resolvable in the card body.
cid=$(scripts/seed task comment "$id" --actor smoke-agent --body "smoke comment probe" | jq -r '.comment_id // empty')
[ -n "$cid" ] || { say "FAIL: comment_id missing from envelope"; exit 1; }
eid=$(scripts/seed task attach-evidence "$id" --actor smoke-agent --kind log --ref smoke-probe | jq -r '.evidence_id // empty')
[ -n "$eid" ] || { say "FAIL: evidence_id missing from envelope"; exit 1; }
body=$(scripts/seed task get "$id" | jq -r .card.body)
echo "$body" | grep -q "$cid" || { say "FAIL: comment_id not stamped into the card body"; exit 1; }
echo "$body" | grep -q "$eid" || { say "FAIL: evidence_id not stamped into the card body"; exit 1; }

# Two-squad routing (os-10c10aae): a second team file activates §6:
# explicit/label routing, ready --squad filtering, and the overlap
# refusal, all against the live instantiation.
say "two-squad scenario: routing + ready --squad + overlap refusal"
mkdir -p web
cat > .seed/teams/web.yaml <<'TEOF'
name: web
lead: smoke-lead
members:
  - human: smoke-lead
scope: ["web/**"]
backlog: {labels: [frontend]}
priority: 2
tier: L2
review: codeowners
TEOF
scripts/seed validate >/dev/null || { say "FAIL: core ** + web squad should validate (fallback exemption)"; exit 1; }
wid=$(scripts/seed task create --title "Web card" --actor shaunlmason --label frontend | jq -r .task)
fid=$(scripts/seed task create --title "Fallback card" --actor shaunlmason | jq -r .task)
scripts/seed task promote "$wid" --actor shaunlmason >/dev/null
scripts/seed task promote "$fid" --actor shaunlmason >/dev/null
[ "$(scripts/seed task get "$wid" | jq -r .squad)" = "web" ] || { say "FAIL: labeled card did not route to web"; exit 1; }
[ "$(scripts/seed task get "$fid" | jq -r .squad)" = "core" ] || { say "FAIL: unmatched card did not fall back to core"; exit 1; }
webq=$(scripts/seed task ready --actor smoke-agent-2 --squad web)
echo "$webq" | jq -e --arg t "$wid" '.tasks | map(.task) | index($t)' >/dev/null || { say "FAIL: ready --squad web missing the web card"; exit 1; }
echo "$webq" | jq -e --arg t "$fid" '.tasks | map(.task) | index($t) | not' >/dev/null || { say "FAIL: ready --squad web leaked the core card"; exit 1; }
cat > .seed/teams/api.yaml <<'TEOF'
name: api
lead: smoke-lead
members:
  - human: smoke-lead
scope: ["web/shared/**"]
backlog: {}
priority: 3
tier: L2
review: codeowners
TEOF
if scripts/seed validate >/dev/null 2>&1; then
  say "FAIL: specific scope overlap without shared_scope accepted"; exit 1
fi
rm .seed/teams/api.yaml

say "OK: $id ready→review unattended — implementation, receipt, memory append, evidence + record ids all present; two-squad routing + overlap refusal enforced"
