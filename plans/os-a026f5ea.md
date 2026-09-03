# Plan: next — the Phase 10 exit record (os-a026f5ea)

Phase 10 (qualification and evaluation) is complete: every numbered item
and the tier-vocabulary card have merged PRs, and this card is its exit
record — the paragraph in `next/docs/progress.md` that the Phase 5, 6,
7, 8 and 9 records wrote, confirming the build plan's exit line against
what shipped, row by row, by citation to named drills on `main`.

**Why plan-first for a docs-only change.** The same reason the Phase 9
record gave: the exit record is the document the next phase orients
from. This one carries more than the last did — it corrects four
ledger lines that still read "in review" after their merges, one line
that two merges interleaved into nonsense, and a Frontier section
that has grown by accretion until it contradicts the ledger above it.
A record that rewrites the resume point deserves a reviewer before it
becomes the thing everyone reads instead of the items.

## What the tree actually shows

Every Phase 10 item has an implementation on `main`, cited so the
record cites rather than asserts:

| item | what | plan | landed |
|---|---|---|---|
| 1 | runtime tuples in grants; adapters report the provisioned tuple; drift is out of grant | #215 | #216 (os-8e53ffd9) |
| 2 | eval contracts through the production machinery; grants cite passing tuples; spot-checks; suspension | #217 | #221 (os-03e47abb) |
| — | the tier vocabulary and its table (the residual item 1b found) | #219 | #222 (os-be12ac16) |
| 3 | independence levels L2/L3 declared per tier, computed and recorded in verdicts | #223 | #233 (os-99829835) |
| 4 | rubric verdicts; the human deferral; calibration with authority suspension on drift | #225 | #238 (os-2e34f66a) |
| 5 | trajectory-prefix regression harness; re-triage and unedited-approval rates | #227 | #239 (os-6bd9ffff) |
| — | out of item: the client's git dir and the verifier's clone arm auto-gc | #224 | #232 (os-711b3028) |

**The exit line's four terms** — "charter III.E (tuples) + III.G
(levels, calibration) + III.O eval items + III.J row 3" — each backed
by drills a reviewer can `grep -n "^func <name>"` for:

- *III.E tuples* (rows 6 and 7): `TestRunStartDeclaresTheTupleAndDriftIsOutOfGrant`,
  `TestDriftIsPerFieldAgainstTheHolder`,
  `TestSmallTeamQualifiedWorkerIsOfferedAndHeldToItsConfiguration`
  (#216); `TestDueOffersMintsAndDisqualifies`,
  `TestDueSpotChecksAgeFromTheDeclaredInstant`,
  `TestEvalLifecycleMintsDisqualifiesAndReTests`,
  `TestSmallTeamEvalQualifiesAndDisqualifiesThroughTheProductionMachinery`
  (#221).
- *III.G levels* (row 6): `TestLevelsAreOrderedAndTiered`,
  `TestLevelClaimedMustEqualTheLevelSupported`,
  `TestLevelShortOfTheTierRefuses`,
  `TestLevelsAreReappliedAlongTheMergeChain`,
  `TestSmallTeamCriticalContractReachesDoneAtL2`,
  `TestFleetExecutableCriticalContractReachesDoneAtL3` (#233, over
  #222's `TestTierTableMirrorsSpec`).
- *III.G calibration* (row 8): `TestRubricVerdictsAtTheTerminal`,
  `TestHumanVerdictAndTheDeferral`, `TestScorecardDerivationAtAdmission`,
  `TestCalibrationQualifiesTheVerifierAndBindsItsRenders`,
  `TestCalibrationOwesMintsDriftAndTheDefect`,
  `TestSmallTeamRubricContractsReachDone` (#238).
- *III.O eval items* (rows 1 and 2, and row 5's recorded half):
  `TestEvalCheckProvesTheKnownVerdict`, `TestCheckProvesTheKnownVerdict`
  (#221); the calibration drills above (#238);
  `TestTrajectoryCorpusIsTheRecorderScenario`,
  `TestTrajectoryCorpusReplaysGreenAndPlantedRowsDiverge`,
  `TestReplayClassifiesEveryDivergence` (#239).
- *III.J row 3*: `TestReportLanesSection`,
  `TestReportLanesRatesAreNullOverNothing`,
  `TestRespecificationActivatesAtSeed4`,
  `TestPlanDigestsThroughTheBoundary` (#239) — the metrics half. The
  policy clause is the exit's one carve-out, below.

## Charter III.E, walked

The Phase 8 record's posture, kept by Phase 9: the exit line's terms
decide the exit; every row of each pillar the exit line names is
walked regardless; an unmet row is recorded as **UNMET** and routed,
never glossed, and never given as a fraction.

| row | status | by |
|---|---|---|
| 1. Identity, credential, principal and runtime distinct in the schemas and the threat model; no user-facing claim exceeds what signatures prove | **met** | #100 (`actors.md`: attribution is not trust; kind and name are the enrolling operator's assertions); #216 (`internal/tuple`: principal and runtime are fields of the declared tuple, distinct from the signing key) |
| 2. Every actor a keypair; enrollment, grants, suspension, revocation events; keyring a projection; signatures verified on every proposal | **met** | #100, #102 (the Phase 3 exit) |
| 3. Kind documented as an operator assertion | **met** | #100 (`actors.md`'s conformance line) |
| 4. Agent enrollment is exactly an identity plus a scoped credential; no inbound connectivity or registration server | **met** | #100 (`actor.enrolled` carries the public key and the operator's assertion, nothing else); #145 (the wakeless poll-only drill: a worker needs no inbound path) |
| 5. Grants checked at admission; out-of-grant structural; operator-only verbs refuse non-operator keys; no self-approval by key disjointness | **met** | #102 (exit 14), #135 (L1, exit 17), #137 (the chain) |
| 6. Qualification binds to the runtime tuple; grants cite tuples; adapters report the provisioned tuple; drifted tuples out of grant | **met** | #216; #221's mints feed the same rule (`qualification.md`, `evals.md`) |
| 7. Scheduled spot-checks re-test active tuples; failures suspend grants attributably without operator ceremony | **met** | #221 (`Due`, `seed eval act` under the supervisor's own grant); #238 extends it to verdict qualifications. "Scheduled" is what a routine invoking `seed eval act` supplies, as the maintenance pass's schedule is the deployment's (`evals.md`) |
| 8. Rotation and revocation drilled: revocation ends standing, **reaps its claims**, preserves attribution, triggers sealed-keyring rotation for verifier keys | **UNMET, not claimed** | ends standing and preserves attribution: #104. Sealed rotation after a verifier key's revocation: #139 (the rotate drill: revoked identity locked out, the current keyring able to unseal). Reaps its claims: **nothing**. The Phase 3 record routed the reap to Phase 5, and Phase 5's exit did not land it; the maintenance reap (#205) needs an interrupt answered by silence, and a revoked holder's `no_data` stream carries no reap path at all. Routed: D2(a) |
| 9. Humans and machines distinguished in the roster; agent-only guardrails and human/agent metrics read the distinction | **UNMET, not claimed** | distinguished: #100 (`kind`), the roster projection (#117) and the cache's roster table (#119). Read by nothing: no guardrail and no metric consumes `kind` — the tier table's human-review column (#238) routes to operator standing, not to kind. The Phase 3 record routed it to Phases 8 and 11; neither consumed it. Routed: D2(b) |

## Charter III.G, walked

| row | status | by |
|---|---|---|
| 1. Done only through the reconciliation chain, each step its own event | **met** | #137 (#141's override path stays uncollapsed) |
| 2. Verdict/merge divergence detected, surfaced, reconciled; each induced in CI | **met** | #137, extended by #139 and #141 |
| 3. Verdict keys provably disjoint from implementing keys; override its own verb | **met** | #135, #141 |
| 4. Clean per-run isolation; parallel verdicts never collide; cleanup pass or fail; inputs enumerable and self-executed | **met** | #135 (origin-stripped per-run workspaces; the runner profile named in every receipt) |
| 5. Receipts bind contract, plan hash at merge-base, diff hash, inventory, visible and sealed transcripts, environment fingerprint; verification recomputes | **met** | #135, #139 (exit 21) |
| 6. Levels L1–L3 defined, declared per tier, enforced at verdict time, recorded; high-consequence tiers require L2 or L3 | **met** | #233 with #222's tier table (`verdicts.md`, `tiers.md`) |
| 7. A red verdict is unmergeable and locks out self-approval until a new submission | **met** | #141 (exit 25, `contract.returned`) |
| 8. Rubric verdicts per item with cited evidence and explicit uncertainty; low-confidence items to human verdict; calibration against a human gold set with authority suspension on drift | **met** | #238 (`verdicts.md`, `evals.md`). Residual stated, not hidden: no calibration definition ships, because the gold set is held outside the tree by design and committed to by digest |
| 9. Verifier code, rubrics, thresholds, sealed keyring, admission rules on the protected surface; governance root and its change process named in config; the capability audit proves agent-key disjointness in CI | **UNMET, not claimed** | the audit half is met: #139 (sealer disjointness both directions, the implementer-cannot-decrypt drills), #102, #234 (curate disjointness). The config half is not: no config under `next/` enumerates the protected surface or names the root's change process — the root is named by genesis (#83), and `next/**` is guarded today only by v1's guardrails as ordinary paths. Same substance as III.L row 2. Routed: D2(b) |
| 10. Evidence, receipts and verdicts queryable by contract, actor, time and outcome | **UNMET, not claimed** | contract, actor and outcome: #119's cache (`contracts` carries subject, position, verb, actor and the payload verbatim; `contract_state` the verdict columns) and the artifact store. Time: no table carries the event's `ts`; position is order, not a clock. A gap in the tree, so a card, not a line: **os-74ce2261** (D5) |

## Charter III.O and III.J row 3, walked

| row | status | by |
|---|---|---|
| O.1 eval contracts through production machinery against fixture repos gate tuple qualification; spot-checks scheduled | **met** | #221 (the shipped `fix-the-check` definition; the mint gated on a recomputed receipt) |
| O.2 verifier calibration scheduled against a human gold set; automatic authority suspension | **met** | #238 |
| O.3 the compromised-actor drill in CI | not this phase's | Phase 12 item 1, planned as #241 |
| O.4 standing drills in CI | **met** | projection rebuild #109; checkpoint verification #205; packet-resume with dead-end assertions #124/#127; claim race storms #123; halt including the raw-git bypass under enforced #84/#99; key revocation with keyring rotation #104/#139; verdict/merge divergence #137; the hostile classification corpus #80; budget reservation races #149; curator poisoning #236 — all under `make check` |
| O.5 trajectory-prefix regression covers lane decision points; simulation mode runs the whole system credential-free | **UNMET, not claimed** | the recorded half: #239. Simulation mode is **Phase 12 item 6's** by the build plan; `trajectories.md` says "Phase 13's" three times and numbers this row 3 (the compromised-actor drill's), both corrected here (D6). Routed: D2(c) |
| J.3 dispatcher re-triage rate and planner unedited-approval rate tracked; the planner receives the strongest tuples by policy | **UNMET, not claimed** | the metrics half: #239 (`trajectories.md`, the report's `lanes` section). The policy clause: #216 landed the offer's `tuples` scope as the scheduling input, `qualification.md` calls that "the whole of 'strongest tuples by policy' the tree can honestly hold until item 2's eval results exist to rank them", and #221 then deferred ranking by name (`evals.md`, "Deferred, by name"); no later Phase 10 item picked it up, so nothing ranks. Routed: D2(d), with the Phase 10 criterion revised to match: D2(f) |

## Design decisions (binding for this task)

- **D1 — the record follows the Phase 9 shape exactly, and claims the
  exit line as this record revises it, never a contradiction.** One
  bold "Phase 10 exit: met" opening that names the exit line's four
  terms as revised by D2(f) — III.E tuples, III.G levels and
  calibration, III.O eval items, III.J row 3's metrics half — and the
  drills backing each, **with the revision in the first sentence, not
  the last**: the exit line as written named III.J row 3 whole, the
  row's policy clause was deferred by name in item 2's `evals.md` and
  never re-homed by any Phase 10 item, and the record says which of
  the two honest courses it took and why. Refused: the scoped-exit
  posture of the Phase 2, 3, 5, 6 and 7 records as a cover for this —
  those exit lines carved their subsets out in their own words
  ("except sealed checks", "minus L2/L3 levels"), and Phase 10's does
  not, so a record that said "met" over a criterion it classifies
  UNMET would be the contradiction the review of #242 named. Then
  III.E and III.G walked row by row with "met by" citations, then
  III.O and III.J row 3, then the unmet rows with their routing, then
  the closing sentence naming this card as the record's task PR.

- **D2 — every unmet row is routed in the build plan's own text, and
  each joins its landing phase's exit line.** The Phase 8 and 9 move,
  for the same reason: an unmet row nobody is assigned stays unmet.
  Seven hunks in `docs/next-build-plan.md`, and nothing else in it:

  *(a) III.E row 8's reap arm → Phase 12 item 1.* The compromised-actor
  consequence has two halves: the ceiling the drill asserts, and
  "revocation (detection ends it)". Item 1 gains the clause that
  revoking the compromised key reaps its open claims **on the
  revocation alone** — the record proves the holder can never exit its
  window, the one case where the ledger rather than the observation
  channel corroborates a reap — with packets, so the work is
  re-offered. #241's plan does not absorb it (a reap corroborated by
  the revocation record is a lifecycle change with its own boundary
  rule and drills, not a drill over the boundary as it stands) and
  names it as a follow-up card; the build plan is where the obligation
  lives either way, so the clause names the card as item 1's.

  *(b) III.E row 9's consumer half and III.G row 9's config half →
  Phase 12 item 4.* The preseed declares config, guardrails, teams,
  protections and posture; it gains: the guardrails it declares
  include the agent-only ones, read off the roster's `kind`, and the
  report's lane rates split by kind (III.E row 9); and it enumerates
  the protected surface and names the governance root and its change
  process (III.G row 9; III.L row 2). III.E rows 8–9 and III.G row 9
  join Phase 12's exit line.

  *(c) III.O row 5's simulation half → Phase 12 item 6*, which already
  names simulation mode; the item gains "(III.O row 5's second half)"
  and the exit line gains III.O row 5.

  *(d) III.J row 3's policy clause → Phase 13 item 7 (new).* The plan
  reserves Phase 13 for "the items Part III requires that Phases 0–12
  deliberately defer", and item 2's D9 deferral is exactly that. The
  item: eval results (mints, spot-checks, calibration agreement) rank
  qualified tuples, and the supervisor's planner offers carry the
  strongest into the `tuples` scope item 1 landed as the input. It is
  a quality policy, not a safety one, so it belongs after promotion
  (§5's reasoning for what Phases 10 and 11 must deliver before
  cutover does not reach it). III.J row 3 joins Phase 13's exit line
  beside row 2.

  *(e) The pillars no exit line owns.* Walking III.G row 9 reaches
  III.L row 2, and the plan names III.C, III.L, III.M and III.Q on no
  exit line at all — every row of theirs already has a landing item
  (observations #131 and metering #151 for III.C; tiers #222, the
  injection defense #192, plan lint #126 and Phase 12 items 2 and 4
  for III.L; the flywheel #240 over the v1 engine for III.M; `make
  check`, the coverage floor, `decisions.md` and Phase 12 item 6 for
  III.Q), but no exit record will ever walk them. Phase 12 is `deps:
  all`, so its exit line gains one sentence: those four pillars are
  walked at that exit, each row met by citation or routed to Phase 13.

  *(f) The Phase 10 criterion is revised to match.* Two honest courses
  exist once a row on an exit line is found unmet: hold the phase open
  until it lands, or revise the criterion in the plan's own text and
  say so (review finding on #242: routing alone neither satisfies nor
  removes a criterion). The first is refused here: it would block
  Phase 12 — the release gate, the migration promotion needs — on a
  quality policy that §5's reasoning for what must precede cutover
  does not reach. So Phase 10 item 1's clause "the planner lane
  receives the strongest tuples by policy" is revised to what landed
  (the offer's `tuples` scope as the scheduling input, the ranking
  policy Phase 13 item 7's), and Phase 10's exit line reads "+ III.J
  row 3's metrics half (its policy clause is Phase 13 item 7's)". The
  record states this revision as its one act of judgment, in its
  opening sentence, and the row itself stays recorded UNMET.

  Refused: routing III.E row 8's reap to a new maintenance card
  outside the plan. The reap is the response half of the very
  invariant Phase 12 item 1 drills, and a card the plan does not name
  is a card the frontier cannot find.

  Refused: recording III.J row 3 "met, metrics half". A conjunctive
  criterion half met is unmet (the Phase 9 record's rule), and the
  exit line naming the row is the reason to say so loudly rather than
  quietly.

- **D3 — the Frontier section is re-derived, and the stale narrative
  is removed with a map.** Today's Frontier still says "Phase 9 was
  under way", "the frontier is item 2's merge, then item 3" and
  "**Next action: Phase 10 item 1**", three paragraphs after the
  ledger records all of those merged. The section becomes one current
  statement: the phase-roll paragraph gains "Phase 10 is done and
  closed" in the shape of the Phase 5–9 sentences; a Phase 11
  paragraph (items 1–4 merged, item 5 in review as #240, the Phase 11
  record to follow its merge); a Phase 12 paragraph (item 1 planned as
  #241 and under revision for two review findings; items 2–6
  uncarded; the phase gate and decision 0003's draft-PR clause); the
  promotion paragraph and the "red PR first" rule unchanged. Every
  paragraph removed is one whose facts the ledger already records,
  and the task PR body carries the map — paragraph → ledger line — so
  the reviewer can check that nothing is lost rather than take it on
  trust. The Phase 9 record's two carried lessons stay where they are,
  in that record's paragraph.

- **D4 — the ledger is brought to the truth.** Items 2, 3 and 5 and
  the auto-gc card read "in review" after their merges and are
  corrected to done with their PR numbers; the item 3 line, which two
  merges interleaved with the tier-vocabulary line, is rewritten as
  two clean lines carrying the same facts. Phase 11's lines are
  corrected the same way (item 3 merged #236, item 4 merged #237,
  item 5's plan merged #231 with its task PR #240 in review), because
  a PR that touches `progress.md` leaves no line it knows to be false;
  the Phase 11 record will re-derive them again when it is written.

- **D5 — gaps in the tree are cards.** The cache's missing `ts`
  column (III.G row 10) was filed at planning as os-74ce2261 and the
  record cites it. Nothing else the walk found is a tree gap: rows 8
  and 9 of III.E and row 9 of III.G were routed by earlier records to
  phases that did not land them, so they are build-plan routing (D2),
  not new cards.

- **D6 — spec edits are corrections only, of numbers the walk reads.**
  `trajectories.md` says simulation mode is Phase 13's in three places
  (the build plan puts it in Phase 12 item 6) and calls the trajectory
  row "III.O row 3" in the same three places, where row 3 is the
  compromised-actor drill and the trajectory-and-simulation row is 5
  (review finding on #242). Walking III.G by number found the same
  class twice more: `verdicts.md` maps the levels to "III.G row 1"
  (row 6) and the rubric to "III.G row 7" (row 8), and `lanes.md`
  repeats "III.G row 7" and "III.O row 3" for the same two rows. A
  conformance mapping that points at the wrong normative row is
  misinformation the next reader acts on, so every one of these is
  corrected — phase and row numbers only, no sentence otherwise
  touched. No other spec changes; no code, no test.

- **D7 — scope guard.** The files are `next/docs/progress.md`,
  `docs/next-build-plan.md` (the D2 edits only), `next/spec/trajectories.md`,
  `next/spec/lanes.md` and `next/spec/verdicts.md` (the D6 number
  corrections only) and the receipt. If confirming a row turns up
  anything else, it is a card.

## Steps

1. `docs/next-build-plan.md` — the seven D2 hunks: Phase 10 item 1's
   clause and Phase 10's exit line (f); Phase 12 item 1's reap clause;
   Phase 12 item 4's two clauses; Phase 12 item 6's parenthetical;
   Phase 12's exit line (III.E rows 8–9, III.G row 9, III.O row 5, and
   the four-pillar sentence); Phase 13 item 7 and its exit line.
2. `next/spec/trajectories.md`, `next/spec/lanes.md`,
   `next/spec/verdicts.md` — the D6 number corrections.
3. `next/docs/progress.md` — the Phase 10 ledger corrections (D4), the
   Phase 11 state words (D4), the exit paragraph (D1, D2, D5) inserted
   after the Phase 10 ledger where the Phase 9 record sits relative to
   its phase, and the Frontier section (D3).
4. Receipt; evidence (PR, receipt, the scope check); review.

## File Scope

- `next/docs/progress.md`
- `docs/next-build-plan.md` (the D2 edits only)
- `next/spec/trajectories.md`, `next/spec/lanes.md`,
  `next/spec/verdicts.md` (the D6 number corrections only)
- `receipts/os-a026f5ea.json`

Nothing else. No Go file, no test, no other spec.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. The record opens with the exit line's four terms as D2(f) revises
   them, states the revision and the refused alternative in its first
   sentences, and names for each term the drills on `main` that back
   it; every drill named exists under that name
   (`grep -n "^func <name>"`).
2. All nine III.E rows, all ten III.G rows, all five III.O rows and
   III.J row 3 appear, each with a status and a citation; III.E rows 8
   and 9, III.G rows 9 and 10, III.O row 5 and III.J row 3 say
   **UNMET** in those letters, each with its routing (D2) or its card
   (D5). No row's status is a fraction.
3. `docs/next-build-plan.md` carries exactly the seven D2 hunks: Phase
   10 item 1 and its exit line, Phase 12 items 1, 4 and 6, Phase 12's
   exit line, Phase 13 item 7 with its exit line — and changes nowhere
   else. After the edit no Phase 10 text names III.J row 3 whole.
4. The three spec files change only phase and row numbers:
   `trajectories.md`'s three "Phase 13" → "Phase 12 item 6" and three
   "III.O row 3" → "row 5"; `lanes.md`'s "III.G row 7" → "row 8" and
   "III.O row 3" → "row 5"; `verdicts.md`'s "III.G row 1" → "row 6"
   and "III.G row 7" → "row 8". `git diff --word-diff` on each shows
   numbers only.
5. The Phase 10 ledger reads done with a PR number on every line;
   item 3's line is one clean line; Phase 11's lines carry the merged
   numbers; the Frontier section names the next action (Phase 12
   items 2–6 carded and planned; item 1's revision in flight; the
   Phase 11 record after #240) and states its derivation from the
   ledger; no paragraph in it contradicts a ledger line.
6. The task PR body carries the D3 map: each removed Frontier
   paragraph → the ledger line that holds its facts.

**Retention set (existing, shown unharmed):**

- `make check` green; `scripts/seed validate` green; the scope check
  below passes on exactly the four files.
- No Go file, no test, no spec other than the three named in the
  diff; every drill named in the record still exists on `main`.
- The Phase 5 through 9 exit records are byte-for-byte unchanged.

## Validation Commands

```sh
export PATH=/home/shaun/go-toolchain/go/bin:$PATH
make check
scripts/seed validate
git diff --name-only origin/main | grep -vE '^next/docs/progress.md$|^docs/next-build-plan.md$|^next/spec/(trajectories|lanes|verdicts).md$|^receipts/os-a026f5ea.json$' | grep -q . && echo SCOPE-VIOLATION || echo scope ok
```

## Expected diff shape

Modified: `next/docs/progress.md` (the Phase 10 ledger and exit
record, the Phase 11 state words, the Frontier section),
`docs/next-build-plan.md` (seven hunks in Phases 10, 12 and 13),
`next/spec/trajectories.md`, `next/spec/lanes.md`,
`next/spec/verdicts.md` (number-only corrections). New:
`receipts/os-a026f5ea.json`. Nothing else.
