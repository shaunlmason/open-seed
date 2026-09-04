# Plan: next — the promotion evidence packet (os-98ce6f8a)

Build plan §5 defines promotion as two human cutovers (self-hosting,
then distribution) gated by seven criteria, and says what agents do
at that gate: "drive the work up to each gate, present the evidence,
and stop." Nothing in the tree presents it. This card lands the
packet the operator reads there: one document mapping each of the
seven criteria to the evidence on `main` by name, stating each
criterion's status in a fixed vocabulary, writing down the cutover
and its rollback (criterion 5 is a document, so the packet is where
it is written), proposing the shadow run as a protocol the operator
can accept or amend, and naming the two cutovers as the reserved
escalations they are. It presents; it does not decide. Charter III.Q
row 7 (self-hosting is total once bootstrapped) and III.R (the
autonomy end-state) both route here through the conformance table
(plans/os-83bc3d84.md D5), and the Phase 13 exit record
(plans/os-d63c7441.md D4) re-derives the Frontier from this packet.
Deps: Phases 0 through 12 on `main` (their drills are the
evidence; build plan §5 puts promotion before Phase 13, so no Phase
13 item is a prerequisite of the packet) and the conformance report
(os-83bc3d84) for criterion 6's doctor half. Tier: trivial in code,
plan-first because the packet is what the promotion gate reads.

## What the tree actually shows

- **Six of the seven criteria have evidence on `main` and no
  document names it.** Loop-completeness (criterion 1) is Phase 9
  item 5: `TestLoopVerbsDeriveAndDischarge` and
  `TestWorkerLoopRunsEndToEndAgainstARealLedger`
  (`cmd/seed/loop_cli_test.go`, `cmd/seed/loop_e2e_test.go`),
  `TestSituationRead` and `TestSituationCarriesTheCallersMessages`,
  `TestSevenActs` and `TestSpecNamesTheSameActs`
  (`internal/loopverb`). Lanes operable (2) is the Phase 9 exit
  record: `TestSmallTeamModeReachesDone`,
  `TestFleetModeConvergesAndReachesDone`, `TestBlindRetryDetector`
  (`cmd/seed/modes_e2e_test.go`), the injection corpus, the worker
  loop's exhaustion parking, the escalation verbs, the three
  maintenance drills. Migration (3) is #255:
  `TestRealFixtureImports` over this repository's own export at
  `seed-anchor/20260903T014125Z`, refreshed by `make fixture-import`.
  Core conformance (6) is the Phase 12 exit record plus the
  conformance report's doctor section (os-83bc3d84, #289, in
  review). The compromised-actor drill (7) is #250 under `make check`
  on every push. Criterion 5 (cutover and rollback written down) has
  no text anywhere. Criterion 4 (the shadow run) has not started and
  cannot start on an agent's own authority: it needs a deployment
  for this repository, which is a posture decision and, for the
  enforced postures, a server or a service credential.
- **III.R's rows are routed to measurements nobody has scheduled.**
  `next/spec/conformance.json` (#289) routes each R row to a named
  measure (the dispatcher's re-triage rate at the shadow run,
  `lanes.planner.unedited_rate` above 0.800, the escalation packets'
  shape, the five-bar audit over the real week, the flywheel over a
  quarter, the first external team) and says a row flips when "the
  promotion evidence card records its measurement". That card is
  this one; the measurements need the shadow run and the cutovers
  first, so what this card can record now is the measure, where it
  is read from, and that it is not yet measured.
- **The five-bar audit is a library, not a verb.** `simulate.Audit`
  (`internal/simulate/audit.go`) audits any record slice, and only
  `seed simulate` calls it; the shadow run's evidence needs it over a
  real ledger, which is a small follow-up rather than this card's.
- **The postures this repository can run are named.**
  `next/spec/platform.md`: a bare checkout runs cooperative or
  forge-hosted; the enforced self-hosted posture needs a POSIX git
  server executing the hook, which the ledger can live on without
  the code moving (III.N row 2). The criteria say "at the enforced
  self-hosted posture", so the packet must say what that costs here
  rather than quietly choosing cooperative.

## Design decisions (binding for this task)

- **D1 — one document, a fixed vocabulary.** `next/docs/promotion.md`,
  one section per criterion in the build plan's numbering, each
  opening with a status from exactly `met`, `partial`, `not started`
  or `reserved` (a human decision the packet presents but cannot
  make), followed by the evidence as rows of `drill | file | PR` and
  the sentence saying what is missing when the status is not `met`.
  No other status word; no score, weight or percentage.
- **D2 — citations are held to the tree.** `internal/promotion`
  (new) parses the packet's evidence rows and checks each cited
  drill is declared in the cited file (`func <name>(`) and each cited
  path exists; `TestPacketCitesRealDrills` runs it over the committed
  packet under `make check`, and a planted citation of a drill that
  does not exist fails by name. A packet that cites what the tree no
  longer holds is a stale claim at the gate, which is the rot this
  drill exists to refuse.
- **D3 — the packet presents and never decides, and the shadow run
  runs at the posture the criteria name.** Build plan §5 evaluates
  every criterion "at the enforced self-hosted posture" and reserves
  the two cutovers, not the shadow run; so the protocol the packet
  proposes runs under enforced self-hosted admission (a POSIX git
  server executing the `seed-admit` hook, hosting the ledger ref
  alone while the code stays on GitHub, charter III.N row 2), and
  the one reserved thing about the run is what only the operator
  can supply: that server. The packet states the other two postures
  for what they are, not as options: forge-hosted needs a deployment
  and a credential the autonomy contract reserves, and cooperative
  forfeits the invariant the run must demonstrate, so a shadow run
  under either produces no criterion-4 evidence and would be the
  distinct supervised milestone the build plan says must be accepted
  as the deviation it is. The two cutovers stay `reserved` with their
  questions. The protocol itself (the declaration for this repository
  under `next/deploy/`, the slice: this workstream's own `next:`
  cards, the window: seven days, the reconciliation: v1 stays
  authoritative, each v1 transition on a sliced card mirrored to the
  ledger by the same actor, a daily diff of v1 card states against
  the folded contract states recorded in the packet, every
  divergence reconciled toward v1 and written down, the evidence:
  the five-bar audit over the real chain) is a proposal in the build
  plan's own words for what criterion 4 requires, and it blocks only
  on the server.
- **D4 — criterion 5 is met by writing it.** The packet carries the
  cutover-and-rollback section in the build plan's three clauses:
  which entry point flips when (`scripts/seed` to the Seed binary,
  AGENTS.md's loop rewritten to the loop verbs, v1's `seed-state`
  frozen at a final anchor and kept read-only), what stays
  authoritative where during the window (v1 for every card until the
  flip, the ledger for every contract after it, CI's receipt gate
  unchanged through both), and the path back (the ledger is
  append-only, so rollback is the entry-point flip reversed, the
  contracts filed during the window listed from the contracts
  projection and re-filed as cards, the frozen state ref unfrozen).
  Meeting it by writing it is what the criterion asks; the status
  says so.
- **D5 — the III.R measurement ledger.** One table, one row per R
  row: the measure in the conformance table's words, the surface it
  is read from (the report section, the receipt field, the audit),
  and its status, `not measured` until the shadow run and cutovers
  supply it. A later card that records a measurement revises this
  table and flips the conformance row with the packet as evidence;
  this card records none, because none exists.
- **D6 — criterion 6 is stated against what has merged.** The
  packet cites the Phase 12 exit record for Phases 0 through 12 and
  the doctor's conformance section for the open rows. os-83bc3d84
  merged (#289, 2026-09-04T01:13Z), so the status is `met`, citing
  #289's drills on `main`; the rule the earlier hedge applied stands:
  never `met` on a claim about a PR in review.
- **D7 — bounds.** No admission change, no verb, no projection, no
  deployment created: the survey helper reads the tree and nothing
  else, and the packet's proposed declaration is prose in the packet,
  not a file the CLI would load. The handbook gains one paragraph
  pointing at the packet.

## Steps

1. `internal/promotion`: the packet parser (status lines, evidence
   rows), the citation check, the two drills (the committed packet
   passes; a planted bogus citation fails by name).
2. `next/docs/promotion.md`: the seven criteria with status and
   evidence rows, the shadow-run protocol, the cutover and rollback,
   the two reserved escalations with their questions, the III.R
   measurement ledger.
3. The handbook paragraph; `next/docs/progress.md` (the packet's
   line, the Frontier pointing at the gate), `next/docs/decisions.md`,
   `memory/LEARNINGS.md`; receipt; evidence; review.

## File Scope

- `next/internal/promotion/**` (new)
- `next/docs/promotion.md` (new), `next/docs/handbook.md`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-98ce6f8a.json`

Nothing else. NOT `next/spec/**`, NOT `next/internal/admit/**`, NOT
`next/cmd/**`, NOT `docs/next-build-plan.md`, NOT `.seed/**`.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. **Every criterion has a status and its evidence.** The packet
   carries the seven criteria in the build plan's order, each with
   a status from the D1 vocabulary; every `met` and `partial` cites
   at least one drill by name and file, and every status that is not
   `met` says in one sentence what is missing.
2. **Citations are real.** `TestPacketCitesRealDrills` passes over
   the committed packet; a planted citation of a drill that does not
   exist, or of a file that does not exist, fails naming the
   citation.
3. **The reserved decisions are questions, not choices.** The
   shadow run and the two cutovers are `reserved`, each with one
   question; the shadow run's question is the server the enforced
   self-hosted posture needs, never a choice of posture, and the
   packet contains no sentence that starts the run or flips an entry
   point.
4. **Criterion 5 is written.** The cutover and rollback section
   answers the build plan's three clauses by name.
5. **The measurement ledger covers III.R.** Seven rows, one per R
   row, each naming the measure, its surface and `not measured`.
6. `make check` green with coverage measured cold; no model
   identifiers in any committed artifact.

**Retention set (existing, shown unharmed):**

- Every existing generated document renders byte-identically and
  `docs check` on the committed tree is clean; no admission,
  transition or projection drill changes.

## Validation Commands

- Boundary: `cd next && go test ./internal/promotion/ -count=1`
- Retention: `make check` (exit checked separately from any pipe)

## Expected diff shape

New: `next/internal/promotion/` (the parser, the check, the drills),
`next/docs/promotion.md`. Modified: `next/docs/handbook.md` (one
paragraph), `next/docs/progress.md`, `next/docs/decisions.md`,
`memory/LEARNINGS.md`, the receipt. No admission, transition, verb or
projection change.
