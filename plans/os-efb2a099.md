# Plan: next — the Phase 11 exit record (os-efb2a099)

Phase 11 (curator and flywheel) is complete once #240 merges, and this
card is its exit record: the paragraph in `next/docs/progress.md` that
the Phase 5 through 10 records wrote, confirming charter III.K against
what shipped, row by row, by citation.

**Why plan-first for a docs-only change.** The exit record is the
document the next phase orients from. Phase 9's frontier line was wrong
once, claiming the phase complete while an item had no implementation,
and the correction came from re-deriving the item list rather than
reading the summary; a record that states a phase's final shape deserves
a reviewer before it becomes the thing everyone reads instead of the
items.

## What the tree actually shows

Every numbered item has an implementation on `main` or in review, cited
here so the record cites rather than asserts:

| item | what | landed |
|---|---|---|
| 1 | staged curation stores with grant-gated boundaries; workers append candidates only | #234 (plan #226) |
| 2 | the promotion gate (support, applies-when, provenance, last-validated, the bound adversarial eval), the contested state, lessons at claim time | #235 (plan #228) |
| 3 | the poisoning drill: constructed trajectories fail to achieve promotion | #236 (plan #229) |
| 4 | expiry at an instant, retirement with evidence kept, rollback by revert, dead ends un-retired on environment change, staleness and bloat lints | #237 (plan #230) |
| 5 | flywheel v0: recurring shapes from the ledger, a drafted workflow, mock validation through the v1 engine, the proposal PR, the repair contract, the conversion rate | #240 (plan #231; in review at planning) |

**The exit line's two criteria**, each backed by a named drill:

- *the poisoning drill green* —
  `TestPoisonCorpusCoversEveryRegisteredGate`,
  `TestEveryPoisonFailsAtBothEnds` and `TestPoisonResidualsArePinned`
  (`internal/admit/poisoning_test.go`): the corpus derived from
  `curation.Gates()` and pinned against the spec table both ways, every
  poison scripted to a promotion attempt and asserted to fail at both
  ends, five residuals pinned by characterization; and the CLI arm
  `TestPoisonsRefuseAtTheTerminal` (`cmd/seed/modes_e2e_test.go`)
  running `worker-proposes` and `smuggled-role-lesson` through `seed
  knowledge` in the small-team fixture.
- *a real recurring chore in the fixture converts to a workflow through
  the gates* — `TestSmallTeamChoreWorkedThreeTimesConverts`
  (`cmd/seed/flywheel_e2e_test.go`): the chore worked three times, its
  shape recurring at the second occurrence, the draft validated by the
  v1 engine's `workflow validate` and `workflow run --mock`, proposed on
  `seed/flywheel-<shape>` and never on `main`, observed merged, the rate
  `1.000` in the report; and with a break planted in one step, the mock
  run failing, the repair contract filed under the dispatcher's key, the
  implementer's fix passing its verdict, the proposal admitting citing
  it, and one merge closing both.

## Charter III.K, walked

The Phase 8, 9 and 10 records' posture: the exit line's criteria decide
the exit; every charter row is walked regardless, and an unmet row is
recorded and **routed**, never glossed.

| row | status | by |
|---|---|---|
| 1. Online lanes append evidence only; conclusion-writing is grant-gated to the curator's proposal path; workers append candidate observations, never promoted lessons | **met** | #234: `curation.deadend.recorded` inside the holder's window; `curation.hypothesis.proposed` from `curate` alone, disjoint from `claim` and `operator` at the grant in both directions (`TestCuratorReadsAndCannotWrite`, `cmd/seed/modes_e2e_test.go`) |
| 2. The pipeline is staged with distinct storage and gates: observations → hypotheses → validated lessons → policy; no stage skips | **met** | #234's three facts and the lessons store; #235's promotion requiring an admitted, uncontested hypothesis whose support still passes; the `unbound` fold for a promotion citing nothing (`TestFoldRendersStagesAndCountsAnomalies`, `TestKnowledgeVerbsDriveTheStages`) |
| 3. Promotion requires applies-when; support from >1 non-failed trajectory (and >1 actor where the family allows); provenance links; last-validated stamp; adversarial evaluation for behavior-changing lessons | **met** | #235: `applies_when` as a predicate over record-derivable fields, the support rule's actor arm with `single_actor_family` recorded where waived, `carrier`/`adversarial`/`last_validated`/`digest` on the fact, the bound eval marker (`TestAppliesWhenIsAPredicateOverRecordFields`, `TestShapesRefuseAtRegisteredGates`) |
| 4. Trajectories are untrusted inputs; the poisoning drill fails to achieve promotion in CI | **met** | #236, the exit line's first criterion; #237 adds a poison per new gate so the corpus stays derived |
| 5. Conflicting evidence is a first-class contested state, never silently averaged; contested lessons do not surface | **met** | #235: `curation.hypothesis.contested` citing held-out observations, the fold's `contested` stage, removal from every delivery with the file and the facts kept (`TestSmallTeamPromotionDeliversLessonsAtClaimTime`, `cmd/seed/lessons_e2e_test.go`) |
| 6. Lessons expire for revalidation; retirement revokes conclusions and keeps evidence; a lesson implicated in a regression rolls back by reverting its PR | **met** | #237: expiry derived at a declared instant and never a fact, `curation.lesson.retired` with `regression` (the revert's `pr`), `superseded` and `expired`, the fold keeping file, hypothesis and observations (`TestExpiryIsDerivedAtAnInstant`, `TestRetirementShapesRefuseAtTheirGates`) |
| 7. Dead ends carry failure condition and environment and can be un-retired on environment change | **met** | #234 (the dead end's shape) and #237 (`curation.deadend.retired`/`unretired` on a changed environment, applicability by string equality with the run's declared environment) |
| 8. The flywheel closes through gates: recurring shapes → drafted workflows → mock validation → PR; repair roles propose patches as PRs; conversion rate is tracked | **met** | #240, the exit line's second criterion, with `TestRawChainsManufactureNoChore` and `TestRawRepairVerdictLeavesTheRepairOpen` (`internal/flywheel/flywheel_test.go`) pinning that only boundary-authentic completions and repair verdicts count |
| 9. Knowledge bloat is managed: dedup with provenance, staleness flags, structure lint | **met** | #237: `lint.duplicate` and `lint.structure` under `seed knowledge lint` and `make check`, `stale` at the projection's declared instant, the `lesson_stale` maintenance finding (`TestStructureAndDedupLints`); routed to Phase 11 item 4 by the build plan |

**The Phase 8 routing closes.** III.I row 5 (matching promoted lessons
surface in packets and envelopes at claim time), which the Phase 8 exit
record routed to Phase 11 item 2, is met by #235: the surfacing set on
`claim take --repo`, in the provisioned `.seed-run/lessons.json` and in
`seed situation --repo`, verified against the repository and reported
unverified where it is not. Full III.I conformance now waits on Phase
13 item 6 alone (the machine-protocol surface and platform parity),
and the record says so.

## Design decisions (binding for this task)

- **D1 — the record follows the Phase 9 and 10 shape exactly.** One
  bold "Phase 11 exit (charter III.K as docs/next-build-plan.md's exit
  line scopes it): met" opening naming the exit line's two criteria and
  the drills that back them; then III.K walked row by row with "met
  by" citations; then the Phase 8 routing closed; then the closing
  sentence naming this card as the record's task PR. A record that
  looks like the last six is a record a reader can diff.

- **D2 — no row is expected unmet, and the expectation is not the
  method.** The table above was walked against the specs' own
  conformance mappings (`curation.md`, `flywheel.md`, `lanes.md`) and
  the drills by name. If writing the record finds a row those mappings
  overstate — the record re-reads the drill, not the mapping — the row
  is recorded **UNMET** in those letters and routed in
  `docs/next-build-plan.md`'s own text to a landing phase, the move the
  Phase 8 and 9 records made. No fraction in the status column; a
  conjunctive row with an arm missing is unmet.

- **D3 — the frontier is re-derived, not edited.** The next-action
  line states its derivation in one sentence: items 1 through 5 each
  with a merged PR, so nothing in Phase 11 remains to claim; Phase 10's
  exit record (os-a026f5ea, #242) beside it; therefore Phase 12, which
  declares `deps: all`, is open, with item 1 planned in #241 (its
  follow-up os-32d06c65 carded) and items 2 through 6 carded as their
  ids stand at writing time. The record inserts after the Phase 10
  record and rebases onto its `progress.md` edits if #242's task PR
  lands first (the peer session rewrote the Frontier section there;
  the conflict is expected and resolved by keeping both records).

- **D4 — what the phase learned is carried forward as one sentence,
  pointing at `memory/LEARNINGS.md`.** Item 1's curation fold and item
  5's flywheel fold learned the same lesson twice: the lifecycle fold is
  tolerant by design, so reading a folded `done` reads what the table
  permits, never what the boundary admitted, and every consumer that
  counts completions (the support rule, the promotion gate, the
  reconciler, the flywheel's occurrences and repairs) re-judges
  authenticity at the record's own position through one derivation
  (`curation.AuthenticPass`). The record names the rule and does not
  restate the cases.

- **D5 — scope guard.** No code, no spec, no test. If confirming a row
  turns up a gap in the tree, it is a card, not a line here. The only
  files are `next/docs/progress.md`, `docs/next-build-plan.md` (only if
  D2 fires), and the receipt.

- **D6 — this card blocks on #240.** The record cites #240 as merged;
  it cannot be written truthfully before that. If #240 is not merged
  when this plan merges, the task PR waits.

## Steps

1. `next/docs/progress.md` — item 5's ledger line moves to **done**
   (merged #240); the Phase 11 exit paragraph (D1, D3, D4), inserted
   after the Phase 10 record; the Frontier section's Phase 11 sentences
   re-derived.
2. `docs/next-build-plan.md` — untouched unless D2 fires, and then only
   the routing sentence and the landing phase's exit line.
3. Receipt; evidence; review.

## File Scope

- `next/docs/progress.md`
- `docs/next-build-plan.md` (D2 only; expected untouched)
- `receipts/os-efb2a099.json`

Nothing else. No file under `next/**` other than `progress.md`.

## Acceptance Criteria

1. The record opens with the exit line's two criteria, each naming the
   drill on `main` that backs it, and every drill named exists under
   that name (a reviewer can `grep -n "^func <name>"`).
2. All nine III.K rows appear, each with a status and a PR citation;
   any unmet row says **UNMET** in those letters with its routing, and
   no row's status is a fraction.
3. III.I row 5 is recorded closed by #235, with III.I's remainder named
   as Phase 13 item 6.
4. The frontier names Phase 12 and states its derivation; item 5's
   ledger line reads **done** with #240's merge.
5. `make check` green; `scripts/seed validate` green. No Go file, no
   spec file, no test file in the diff.
6. #240 is merged before the task PR opens, and the record cites its
   merged number.

## Validation Commands

```sh
make check
scripts/seed validate
git diff --name-only origin/main -- next/ docs/ | grep -vE '^next/docs/progress.md$|^docs/next-build-plan.md$' | grep -q . && echo SCOPE-VIOLATION || echo scope ok
```

## Expected diff shape

`next/docs/progress.md`: one ledger line changed, one paragraph added,
the Frontier section's Phase 11 sentences re-derived. `docs/next-build-plan.md`:
no change expected. The receipt. Nothing else.
