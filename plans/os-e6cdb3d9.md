# Plan: next — the Phase 9 exit record (os-e6cdb3d9)

Phase 9 (lanes, escalation, maintenance) is complete once #212 merges,
and this card is its exit record: the paragraph in
`next/docs/progress.md` that the Phase 5, 6, 7 and 8 records wrote,
confirming charter III.J against what shipped, row by row, by citation.

**Why plan-first for a docs-only change.** The exit record is the
document the next phase orients from, and this phase's frontier line
has already been wrong once: a revision claimed the phase complete when
item 5(b) had no implementation, while two other paragraphs in the same
file said so. A record that states the phase's final shape deserves a
reviewer before it becomes the thing everyone reads instead of the
items.

## What the tree actually shows

Every numbered item has an implementation on `main`, cited here so the
record cites rather than asserts:

| item | what | landed |
|---|---|---|
| 1 | six lane fragments; dispatcher least-capability; injection corpus; worker loop with exhaustion parking | #188, #191, #192; review fixes os-378e44f3 |
| 2 | escalation with packet, question, minimal decision; age surfaced | #200 |
| 3 | unattended maintenance loop: reap, reconcile, rebuild, checkpoint, lints | #205 |
| 4 | small-team and fleet end-to-end fixtures, wakeless, every convergence arm | #207 |
| 5(a) | obligations projection | #171 |
| 5(b) | the situation read carries the caller's mail; `seed message read` | #211 |
| 5(c) | loop verbs deriving every derivable argument | #173; post-merge defects os-9b3f3ef3 |
| — | the role-grant gap #207 found: supervisor and observer roles, sealer on the verifier, the capability-coverage drill | #212 (pending merge at planning) |

**The exit line's three criteria**, each backed by a named drill on
`main`:

- *both modes run the full loop in CI* —
  `TestSmallTeamModeReachesDone` and
  `TestFleetModeConvergesAndReachesDone` (`cmd/seed/modes_e2e_test.go`),
  remote posture, no wake channel, every refusal converging within one
  retry, with `TestBlindRetryDetector` pinning the forbidden fourth
  outcome.
- *injection corpus green* — the eight corpus files under
  `internal/admit/testdata/injection/` and
  `TestNoHostileTextWidensTheDispatcherSet`, plus the containment
  sweep `TestIntentProseReachesNoDownstreamReadAutomatically`, which
  since #211 plants its marker in a message payload and a message
  subject as well as in an intent.
- *maintenance runs unattended in the fixture* —
  `TestMaintainHoldsNoPrivatePowers`,
  `TestMaintainFilesDefectsAndRaisesNoEscalation` and
  `TestMaintainCheckpointIsStartableByAFreshReader`
  (`cmd/seed/maintain_cli_test.go`): one pass, no scheduler, no wake,
  audited as an ordinary actor.

## Charter III.J, walked

The Phase 8 record's posture: the exit line's criteria decide the exit;
every charter row is walked regardless, and an unmet row is recorded and
**routed**, never glossed.

| row | status | by |
|---|---|---|
| 1. Role definitions for all six lanes as grants + conventions, composable from ordered fragments, checked by validation | **met** | #188 (manifests + fragments + `internal/lane`); #212 adds the `kind` field and refuses a seventh lane by name, so "six" is enforced rather than counted |
| 2. Dispatcher least standing capability; injection conformance suite (intents, mirrors, tool output quoted as data) | **met, two-thirds** | #192: intents and tool output covered; the mirror arm cannot be met because `request.*` has zero transition rows. Recorded in `lanes.md` and routed already: whichever card lands `request.*` inherits the corpus |
| 3. Dispatcher re-triage rate and planner unedited-approval rate tracked; planner receives the strongest tuples by policy | **UNMET, not claimed** | nothing in the tree: no rate is computed anywhere, and "strongest tuples" presupposes Phase 10's tuple system. Routed by this card, see D2 |
| 4. Maintenance runs green unattended and is audited as an ordinary actor | **met** | #205, with `TestMaintainHoldsNoPrivatePowers` as the audit-posture pin |
| 5. Escalations carry packet + question + minimal decision; waiting ones surface with age; resolution latency tracked | **met** | #200: `escalation.pending` with the raising `ts`; `seed decision record` reports `resolved_after_seconds`, derived from the chain and stored nowhere (`escalation.md`) |
| 6. Small-team and fleet modes both run the full loop in CI | **met** | #207, with #212 making both modes buildable from the shipped role set alone rather than from identities the fixture invented |

## Design decisions (binding for this task)

- **D1 — the record follows the Phase 8 shape exactly.** One bold
  "Phase 9 exit: met" opening naming the exit line's criteria and the
  drills that back them; then III.J walked row by row with "met by"
  citations; then the unmet row with its routing; then the closing
  sentence naming this card as the record's task PR. A record that
  looks like the last four is a record a reader can diff.

- **D2 — row 3 is routed to Phase 10, in the build plan's own text,
  and III.J row 3 joins Phase 10's exit line.** This is the move the
  Phase 8 record made for III.I rows 3 and 5, and the reason is the
  same: an unmet row that nobody is assigned is a row that stays unmet.
  "Strongest tuples by policy" is Phase 10 item 1 already in substance
  (runtime tuples in grants); the two rates are lane-quality metrics
  that are meaningless without the eval harness, so they land beside
  Phase 10 item 5's trajectory-prefix regression harness for lane
  decision points, which is where a dispatcher's re-triage and a
  planner's unedited approval become observable. The edit is one
  sentence in each item and one clause on the exit line.

  Refused: claiming row 3 on the strength of the report projection's
  existing refusal-rate section. A refusal rate is an affordance-gap
  metric (III.I row 4), not a lane-quality one, and reading one as the
  other is how a row gets ticked by a paragraph.

- **D3 — the frontier line is re-derived, not edited.** The next-action
  line says Phase 10 item 1, and the record states the derivation in
  one sentence: items 1 through 5(c) and the gap card, each with a
  merged PR. The correction the last frontier needed is kept in the
  record (it is already in `progress.md` from #211), because a record
  that erases its own correction invites the same mistake.

- **D4 — what the phase learned about its drills is carried forward
  as a sentence in the record, pointing at `memory/LEARNINGS.md`.**
  Three cards in a row found hand-listed counts wrong: the exit-code
  table and the refusal-site matrix (os-d03bde01), the capability
  coverage gap (os-d6a52784, where the derived drill found six verbs
  against a card that named two capabilities). The record names the
  rule — when a criterion says "all N", N is derived — and does not
  restate the three cases.

- **D5 — scope guard.** No code, no spec, no test. If confirming a row
  turns up a gap in the tree, it is a card, not a line here. The only
  files are `next/docs/progress.md`, `docs/next-build-plan.md` (the D2
  routing sentences), and the receipt.

- **D6 — this card blocks on #212.** The record cites #212 as merged;
  it cannot be written truthfully before that. If #212 is not merged
  when this plan merges, the task PR waits.

## Steps

1. `docs/next-build-plan.md` — Phase 10 item 1 gains the clause "the
   planner lane receives the strongest tuples by policy (III.J row
   3)"; item 5 gains "dispatcher re-triage rate and planner
   unedited-approval rate tracked (III.J row 3)"; Phase 10's exit line
   gains "+ III.J row 3" (D2).
2. `next/docs/progress.md` — the Phase 9 exit paragraph (D1, D3, D4),
   inserted where the Phase 8 record sits relative to its phase.
3. Receipt; evidence; review.

## File Scope

- `next/docs/progress.md`
- `docs/next-build-plan.md` (the three D2 edits only)
- `receipts/os-e6cdb3d9.json`

Nothing else. No file under `next/**` other than `progress.md`.

## Acceptance Criteria

1. The record opens with the exit line's three criteria, each naming
   the drill on `main` that backs it, and every drill named exists
   under that name (a reviewer can `grep -n "^func <name>"`).
2. All six III.J rows appear, each with a status and a PR citation, and
   row 3 says **UNMET** in those letters with its Phase 10 routing.
3. `docs/next-build-plan.md` names III.J row 3 in Phase 10 items 1
   and 5 and on Phase 10's exit line, and nowhere else changes.
4. The frontier line names Phase 10 item 1 and states its derivation.
5. `make check` green; `scripts/seed validate` green. No Go file, no
   spec file, no test file in the diff.
6. #212 is merged before the task PR opens, and the record cites its
   merged number.

## Validation Commands

```sh
make check
scripts/seed validate
git diff --stat origin/main -- next/ docs/ | grep -vE 'progress.md|next-build-plan.md' | grep -q . && echo SCOPE-VIOLATION || echo scope ok
```
