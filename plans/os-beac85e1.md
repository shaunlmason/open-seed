# Plan: next Phase 1 exit — frontier update (os-beac85e1)

Records the Phase 1 exit in the frontier file. Design authority:
[`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 1 *Exit*
(charter III.A) and the [`AGENTS.md`](../AGENTS.md) rule that
`next/docs/progress.md` stays the accurate frontier fresh agents resume
from. All seven Phase 1 items are merged and their cards closed
(#76, #79, #83, #86, #84, #80, #85); the file still shows 1.4/1.5/1.7 in
review, so the frontier is stale exactly where a fresh agent would read
it. Docs-only: no code surface changes.

## Steps

1. **Flip the finished rows.** In `next/docs/progress.md`, mark 1.4
   (task PR #86), 1.5 (task PR #84), and 1.7 (task PR #85) **done**
   (merged, review outcomes noted, cards closed), matching the wording of
   the already-done rows.
2. **Record the Phase 1 exit.** One short paragraph tying the exit
   criteria to their evidence: one-command verification (#85), corrupted
   fixtures detected with positioned reasons (#79/#83/#85), the hostile
   classification corpus (#80), and the race drill green on main (#86).
3. **Open the Phase 2 ledger.** List the filed Phase 2 admission cards
   (2.1 os-3898f232, 2.2 os-895bf828, 2.3 os-d3591e09, 2.4 os-3c72f93f,
   2.5 os-028dda91) with their dependency chain — one line per build-plan
   item, and only those five: this administrative card is recorded in the
   exit prose, never as a Phase 2 ledger row — and rewrite the Frontier
   paragraph to point at 2.1 (`internal/admit`, plan-first) and the
   Phase 2 exit subset.
4. **Durable insight.** Append the parallel-PR conflict pattern to
   `memory/LEARNINGS.md`: tail appends resolve by keeping both sides;
   in-flight PRs carry a byte-identical `progress.md`; receipts
   regenerate after every rebase.

## File Scope

- `next/docs/progress.md`
- `memory/LEARNINGS.md`

## Acceptance Criteria

**Boundary set (new, shown working):**

- Every Phase 1 row reads **done** with its task PR and closed card; no
  row claims an open review that has merged.
- The Phase 1 exit paragraph names all four exit criteria with the PRs
  that evidence them.
- The Phase 2 section lists exactly the five admission cards with their
  deps (this card appears only in the exit prose), and the Frontier
  paragraph's next action is 2.1's plan.
- The frontier-content validation command below fails on any of: a Phase
  1 row not marked done, a missing Phase 2 card id, or a missing exit
  paragraph — so the receipt cannot certify another stale frontier.

**Retention set (existing, shown unharmed):**

- No file outside the two named docs changes; `plans/**` untouched by the
  task PR.
- The repo-wide gate stays green (`make check`).

## Validation Commands

- `make check-next`
- `make check`
- `sh -c 'p=$(tr "\n" " " < next/docs/progress.md | tr -s " "); for pr in 76 79 80 83 84 85 86; do echo "$p" | grep -q "task PR #$pr — \*\*done\*\*" || { echo "PR $pr row not done"; exit 1; }; done; for id in os-3898f232 os-895bf828 os-d3591e09 os-3c72f93f os-028dda91; do echo "$p" | grep -q "$id" || { echo "$id missing"; exit 1; }; done; echo "$p" | grep -q "Phase 1 exit (charter III.A)" || { echo "exit paragraph missing"; exit 1; }'`
