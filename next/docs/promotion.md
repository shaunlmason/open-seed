# The promotion evidence packet

This is the document the operator reads at the promotion gate
(`docs/next-build-plan.md` §5; plans/os-98ce6f8a.md). The build plan
defines promotion as two human cutovers, self-hosting and then
distribution, gated by seven criteria, and says what agents do at the
gate: drive the work up to it, present the evidence, and stop. This
packet presents. It maps each criterion to the evidence on `main` by
drill name and file, states the criterion's status in a closed
vocabulary, writes the cutover and its rollback down, proposes the
shadow run as a protocol the operator can accept or amend, names the
two cutovers as the reserved escalations they are, and keeps the
ledger of the measurements charter III.R waits on. Nothing here
decides anything.

**How to read it.** Each numbered section is one of the build plan's
criteria, in its order. The section opens with `Status:` and one of
four words: `met` (the evidence on `main` satisfies the criterion),
`partial` (part of it does, and the section says which part is
missing), `not started` (nothing on `main` addresses it), `reserved`
(a decision the packet presents and cannot make). Every `met` or
`partial` criterion cites its evidence as rows of drill, file (under
`next/`) and the pull request that landed it; `internal/promotion`
holds every cited drill to the tree under `make check`
(`TestPacketCitesRealDrills`), so a citation the tree no longer
declares fails the build rather than standing at the gate. A status
that is not `met` carries a `Missing:` sentence; a `reserved` one
carries the `Question:` the operator answers.

**The criteria at a glance.**

| # | criterion | status |
|---|---|---|
| 1 | loop-completeness | met |
| 2 | lanes operable | met |
| 3 | migration proven | met |
| 4 | shadow run | reserved |
| 5 | cutover and rollback written down | met |
| 6 | core conformance | met |
| 7 | the compromised-actor drill green in CI before the cutover | met |

The gate is not open: criterion 4 waits on the one thing only the
operator can supply, the server the enforced self-hosted posture
needs. Everything an agent can do before that is done and cited
below.

## 1. Loop-completeness

Status: met

A lane runs poll, claim, plan-gate, work, meter, submit, verdict,
merge-observe and a deliberate exit, plus escalation and messages,
entirely through Seed verbs, orienting from one position-stamped read
rather than hand-assembling ledger payloads and hand-computing fences.
Phase 9 item 5 landed the three parts: the obligations projection and
the situation read (#171), the loop verbs that derive every argument
the system can derive and refuse before signing what the boundary
would refuse after (#173, with #175's three post-merge fixes), and the
situation read carrying the caller's messages with "unread" derived
from the cited position (#211). The worker loop runs those verbs end
to end against a real ledger (#191), and the two mode fixtures drive
every lane through the whole loop on a remote with no wake channel
(#207). The lane fragments name the one read each lane orients from,
and `internal/lane` refuses a fragment that instructs a bare
heartbeat (#188).

| drill | file | PR |
|---|---|---|
| `TestLoopVerbsDeriveAndDischarge` | `cmd/seed/loop_cli_test.go` | #173 |
| `TestWorkerLoopRunsEndToEndAgainstARealLedger` | `cmd/seed/loop_e2e_test.go` | #191 |
| `TestSituationRead` | `cmd/seed/situation_cli_test.go` | #171 |
| `TestSituationSinceIsApplicable` | `cmd/seed/situation_cli_test.go` | #171 |
| `TestSituationCarriesTheCallersMessages` | `cmd/seed/message_cli_test.go` | #211 |
| `TestSevenActs` | `internal/loopverb/loopverb_test.go` | #188 |
| `TestSpecNamesTheSameActs` | `internal/loopverb/loopverb_test.go` | #188 |
| `TestOnlyTheExclusiveActIsRemoteOnly` | `internal/loopverb/loopverb_test.go` | #188 |
| `TestLoopSurfaceIsWakeless` | `internal/loop/loop_test.go` | #191 |
| `TestEscalationRaiseAndAnswerThroughTheCLI` | `cmd/seed/escalation_cli_test.go` | #200 |
| `TestSmallTeamModeReachesDone` | `cmd/seed/modes_e2e_test.go` | #207 |
| `TestFleetModeConvergesAndReachesDone` | `cmd/seed/modes_e2e_test.go` | #207 |

## 2. Lanes operable

Status: met

Phase 9 is complete and its exit record (`next/docs/progress.md`,
card os-e6cdb3d9) walks charter III.J row by row: the six lane role
fragments as grants plus conventions, ordered and validated (#188,
#212 making "six" enforced by name); the dispatcher's least-capability
posture as an allowlist and the injection conformance suite over
intents and tool output (#192), the mirror arm closed by Phase 13 item
4 (#270); the worker loop with exhaustion parking at the real budget
refusal (#191); escalation with packet, question and decision, and
the age and resolution latency derived from the chain (#200); the
maintenance loop runnable unattended and audited as an ordinary actor
(#205); and the small-team and fleet modes running the full loop in
CI with every refusal converging within one retry and the blind retry
pinned as the forbidden outcome (#207).

| drill | file | PR |
|---|---|---|
| `TestTheCharterSixAreClosed` | `internal/lane/lane_test.go` | #188 |
| `TestEveryRequiredCapabilityIsGrantedBySomeManifest` | `internal/lane/lane_test.go` | #212 |
| `TestFragmentInstructingABareHeartbeatIsAFinding` | `internal/lane/lane_test.go` | #188 |
| `TestDispatcherPostureIsAnAllowlist` | `internal/lane/lane_test.go` | #188 |
| `TestNoHostileTextWidensTheDispatcherSet` | `internal/admit/injection_test.go` | #192 |
| `TestNoHostileRequestWidensTheDispatcherSet` | `internal/admit/injection_request_test.go` | #270 |
| `TestWorkerLoopParksOnRealBudgetExhaustion` | `cmd/seed/loop_e2e_test.go` | #191 |
| `TestMaintainHoldsNoPrivatePowers` | `cmd/seed/maintain_cli_test.go` | #205 |
| `TestMaintainFilesDefectsAndRaisesNoEscalation` | `cmd/seed/maintain_cli_test.go` | #205 |
| `TestMaintainCheckpointIsStartableByAFreshReader` | `cmd/seed/maintain_cli_test.go` | #205 |
| `TestBlindRetryDetector` | `cmd/seed/modes_e2e_test.go` | #207 |
| `TestModeGrantsComeFromTheShippedManifests` | `cmd/seed/modes_e2e_test.go` | #212 |

## 3. Migration proven

Status: met

`seed import --from-open-seed` (Phase 12 item 5, #255) is drilled
against a real export of this repository's v1 state, not only a
synthetic one: `next/fixtures/import/open-seed/` is this repository's
export at `seed-anchor/20260903T014125Z` (251 files, 1214 run-log
entries, imported as 1345 records), and `TestRealFixtureImports` folds
every one of its contracts to the state its card holds. `make
fixture-import` regenerates the fixture from the live repository at
the newest anchor, so the drill stays real as the v1 history grows,
and the cutover procedure below re-imports at the final anchor rather
than trusting a snapshot. The four refusals precede every write.

| drill | file | PR |
|---|---|---|
| `TestRealFixtureImports` | `internal/importer/fixture_test.go` | #255 |
| `TestAnchorRefusalsPrecedeEveryWrite` | `internal/importer/drills_test.go` | #255 |
| `TestNonEmptyLedgerRefuses` | `internal/importer/drills_test.go` | #255 |
| `TestUnmappedVerbRefuses` | `internal/importer/drills_test.go` | #255 |
| `TestSyntheticPredecessorImports` | `internal/importer/drills_test.go` | #255 |
| `TestImportCommandEnvelopes` | `cmd/seed/import_cli_test.go` | #255 |

## 4. Shadow run

Status: reserved

Missing: the run itself. Seed has coordinated no card of this repository beside v1: no deployment for this repository exists, no slice is declared, no window is stated, and no divergence has been reconciled because nothing has run.

Question: which POSIX git server hosts the shadow ledger with the `seed-admit` pre-receive hook, so the run happens at the enforced self-hosted posture the criteria name?

The build plan evaluates every criterion "at the enforced self-hosted
posture" and reserves the two cutovers, not the shadow run; so the
protocol below runs under enforced self-hosted admission, a bare
remote on a POSIX git server executing the hook and hosting the
ledger ref alone while the code stays on GitHub (charter III.N row
2: the loop runs on any git remote supporting the declared posture).
The one thing about the run that only the operator can supply is
that server: this repository lives on github.com, where no server
executes the hook (`next/spec/platform.md`). Everything else in the
protocol is written and blocks on nothing.

The other two postures are stated for what they are, not as options.
Forge-hosted needs `seed-admit serve` deployed under a credential the
operator holds, both reserved by the autonomy contract, and is the
posture of a deployment whose forge is the server, not a shadow of
this one. Cooperative costs nothing to stand up and forfeits exactly
what the run must demonstrate: the doctor prints that the security
invariant does not hold and protocol rules are advisory against a
hostile credential. A shadow run under either produces no criterion-4
evidence, and choosing one would be the distinct supervised milestone
the build plan says must name what it trades away and be accepted as
the deviation it is; the packet does not propose it.

The protocol is in the section "The shadow run, as a protocol" below,
in the build plan's own words for criterion 4; the packet starts
nothing and flips nothing.

## 5. Cutover and rollback written down

Status: met

The build plan asks for three things written down: which entry point
flips when, what stays authoritative where during the window, and the
documented path back. The section "The cutover and the rollback"
below answers the three by name, and the packet's own drill holds the
section to that shape, so the criterion is met by this document
existing in the tree rather than by a claim about it.

| drill | file | PR |
|---|---|---|
| `TestPacketWritesTheCutoverDown` | `internal/promotion/promotion_test.go` | #294 |
| `TestPacketCitesRealDrills` | `internal/promotion/promotion_test.go` | #294 |

## 6. Core conformance

Status: met

Phases 0 through 12 are complete and recorded, and the doctor reports
exactly which Phase 13 rows remain open: the conformance report
(os-83bc3d84, #289) checks in Part III as a table held to the
charter row for row, renders it under the docs drift gate, and gives
`seed doctor --repo .` a `conformance` section that counts the rows by
status, lists every row not yet met by pillar, row and status, sets
the enforced-only rows aside at the cooperative posture and names the
mixed rows there, and reports `complete` only when every applicable
row is met. The rows it lists as open today are Phase 13's, flipped
by the Phase 13 exit record (os-d63c7441) once III.R's measurements
exist, which is the promotion critical path in the build plan's own
words.

| drill | file | PR |
|---|---|---|
| `TestTableIsTheCharterRowForRow` | `internal/conformance/conformance_test.go` | #289 |
| `TestTableDriftFromTheCharterIsRefused` | `internal/conformance/conformance_test.go` | #289 |
| `TestVocabularyHolds` | `internal/conformance/conformance_test.go` | #289 |
| `TestAssessJudgesAtThePosture` | `internal/conformance/conformance_test.go` | #289 |
| `TestConformanceRendersFromTheTableWithoutAClock` | `internal/docs/docs_test.go` | #289 |
| `TestDoctorReportsConformanceAtThePosture` | `cmd/seed/doctor_test.go` | #289 |

Phases 0 through 12 each closed with an exit record that walks the
pillars its exit line names, row by row, with drills on `main` cited
by name (`next/docs/progress.md`). One drill per phase stands here as
the pointer into that record; the record carries the rest.

| drill | file | PR |
|---|---|---|
| `TestCorruptionsAreDetectedDistinctly` | `internal/ledger/corruption_test.go` | #79 |
| `TestHostileCorpusRefuses` | `internal/classify/classify_test.go` | #80 |
| `TestDrillRawAdversaryPerPosture` | `cmd/seed-admit/drill_test.go` | #99 |
| `TestDrillKillAndReplace` | `cmd/seed-admit/drill_test.go` | #99 |
| `TestDrillKeyRotation` | `cmd/seed-admit/rotation_test.go` | #104 |
| `TestRebuildByteIdenticalAndStamped` | `internal/project/project_test.go` | #117 |
| `TestClaimRaceStorm` | `internal/gitref/gitref_test.go` | #123 |
| `TestPacketResumeDrill` | `internal/packet/resume_test.go` | #124 |
| `TestPlanGateAboveTrivialTier` | `internal/admit/plan_test.go` | #126 |
| `TestSubjectClassifiesInducedDivergences` | `internal/reconcile/reconcile_test.go` | #137 |
| `TestWakelessPollOnlyRun` | `cmd/seed/offer_cli_test.go` | #145 |
| `TestReservationRaceAndStatus` | `cmd/seed/budget_cli_test.go` | #149 |
| `TestDisposabilityDrill` | `cmd/seed/run_cli_test.go` | #151 |
| `TestAffordanceRegressionClass` | `internal/admit/soundness_test.go` | #163 |
| `TestHaltRefusesEverythingButLift` | `internal/halt/halt_test.go` | #84 |
| `TestSmallTeamEvalQualifiesAndDisqualifiesThroughTheProductionMachinery` | `cmd/seed/modes_e2e_test.go` | #221 |
| `TestSmallTeamCriticalContractReachesDoneAtL2` | `cmd/seed/level_modes_e2e_test.go` | #233 |
| `TestSmallTeamRubricContractsReachDone` | `cmd/seed/rubric_modes_e2e_test.go` | #238 |
| `TestSmallTeamPromotionDeliversLessonsAtClaimTime` | `cmd/seed/lessons_e2e_test.go` | #235 |
| `TestPoisonsRefuseAtTheTerminal` | `cmd/seed/modes_e2e_test.go` | #236 |
| `TestEveryPoisonFailsAtBothEnds` | `internal/admit/poisoning_test.go` | #236 |
| `TestSmallTeamRetirementAndRevalidationAtClaimTime` | `cmd/seed/retirement_e2e_test.go` | #237 |
| `TestSmallTeamChoreWorkedThreeTimesConverts` | `cmd/seed/flywheel_e2e_test.go` | #240 |
| `TestServiceAgreesWithTheBoundaryAndTheHook` | `cmd/seed-admit/serve_test.go` | #252 |
| `TestGitHubAdapterReconciles` | `internal/protections/github_test.go` | #252 |
| `TestRunReMeasuresColdOnce` | `internal/perfgate/perfgate_test.go` | #253 |
| `TestInitPreseedIsIdempotentAndDriftRefuses` | `cmd/seed/preseed_cli_test.go` | #254 |
| `TestAgentCeilingReadsTheRosterKind` | `internal/admit/policy_test.go` | #254 |
| `TestGeneratedContentIsFromTheTables` | `internal/docs/docs_test.go` | #272 |
| `TestSimulateReachesDoneEnforced` | `cmd/seed/simulate_cli_test.go` | #272 |
| `TestSimulateAcceleratedBacklog` | `cmd/seed/simulate_cli_test.go` | #272 |
| `TestAuditCatchesSilentAbandonment` | `internal/simulate/audit_test.go` | #272 |

Phase 13 is not a precondition of promotion (build plan §5, "what is
not required"), and its items are on `main` regardless: request
ingress and federation (#270), the cross-organization boundary
(#279), the machine-protocol surface and the platform matrix (#273),
the Forgejo adapter (#281), the remaining executor adapters (#282),
tuple ranking (#286), racing (#269). Their drills are cited in the
conformance table, not repeated here.

## 7. The compromised-actor drill green in CI before the cutover

Status: met

Phase 12 item 1 (#250) is the release gate: `internal/redteam` asserts
the charter's §I.2 ceiling item by item against an enforced fixture
with a valid key, a credential and raw git, the ceiling and the
residuals pinned as tables both ways, and the code-ref rules proven
load-bearing. It runs under `make check`, which
`.github/workflows/check-validate.yml` runs on every push to `main`,
every pull request and every merge group, so no commit exists that
the drill has not gated, and the cutover commit will be one of them.
The consequence's second half, revocation reaping the revoked
holder's open claims on the revocation alone, landed in #267.

| drill | file | PR |
|---|---|---|
| `TestCeilingHoldsAtThePush` | `internal/redteam/redteam_test.go` | #250 |
| `TestOneDerivationLedgerAgrees` | `internal/redteam/redteam_test.go` | #250 |
| `TestCoverageBothWays` | `internal/redteam/redteam_test.go` | #250 |
| `TestResidualsArePinned` | `internal/redteam/redteam_test.go` | #250 |
| `TestTablesValidate` | `internal/redteam/redteam_test.go` | #250 |
| `TestCodeRefRulesAreLoadBearing` | `cmd/seed-admit/mutation_test.go` | #250 |
| `TestDrillCompromisedKeyCutPerPosture` | `cmd/seed-admit/rotation_test.go` | #104 |
| `TestRevokedHolderReapsOnTheRevocationAlone` | `internal/admit/revoked_test.go` | #267 |

## The shadow run, as a protocol

A proposal for criterion 4, in the build plan's words: "Seed
coordinates a declared slice of this repository's own cards beside v1
for a stated window, with any divergence reconciled and recorded."
Every line below is a default the operator can amend; none of it is
in force until the operator says so. This section presents; it does
not schedule. No window opens, no slice is declared and no deployment
is created until the operator accepts or amends what is written here,
which is what build plan §5 asks of the packet: present the evidence
and stop.

**The deployment.** A declaration for this repository, kept under
`next/deploy/` and passed to every verb by `--config` (or
`SEED_CONFIG`), in the shape `seed init --preseed` reads
(`next/spec/postures.md`, "The preseed"). The proposed content:

```json
{
  "posture": "enforced-self-hosted",
  "protocol": "seed/7",
  "governance": {
    "root": "declared-at-init",
    "owners": ["@shaunlmason"],
    "change_process": "pr+owner-review"
  },
  "protected": [
    "next/spec", "next/internal/admit", "next/internal/transition",
    "next/internal/keyring", "next/internal/verdict", "next/internal/seal",
    "next/internal/eval", "next/evals", "next/internal/curation",
    "next/knowledge/lessons", "next/lanes", "next/cmd/seed-admit",
    "next/cmd/covergate", "Makefile", ".github/workflows", "scripts"
  ],
  "checkpoints": {"trust": "signers"},
  "guardrails": {
    "squads": {"core": {"default": "standard", "max_agent": "standard"}},
    "paths": [
      {"prefix": "next/internal/admit", "min": "critical"},
      {"prefix": "next/spec/transitions.json", "min": "critical"}
    ]
  },
  "teams": {
    "squads": [{"name": "core", "lanes": ["dispatcher", "planner", "implementer", "verifier", "curator", "maintenance", "supervisor", "observer"]}]
  }
}
```

The ledger remote is a bare repository on the server the operator
names, its `pre-receive` hook the `seed-admit` binary built from this
tree, the ledger ref `refs/seed/ledger`; the code stays on GitHub. The
genesis names the operator's key as the governance root; one key per
lane is enrolled for the identities that will act (the implementer
lane for the sessions that work cards, the dispatcher lane for the
session that files intents, the verifier lane for the reviewing
identity, the maintenance lane for the scheduled pass, an observer
for `merge.observed`).

**The slice.** This workstream's own `next:` cards open at the
window's start: the backlog cards the build plan's §3 names
(os-a00d3f34, os-7953612b, os-f17567a6), the coverage card
(os-f262585a) and the exit record (os-d63c7441). Each is filed on the
ledger as an intent whose subject is the v1 card id, so every
divergence is a diff between two records of one thing.

**The window.** Seven days from the ledger position the operator
records as the start, the length charter III.R row 5 asks for, ended
by a second recorded position.

**The dual-run rule.** v1 stays authoritative for every card in the
slice for the whole window. Every v1 transition on a sliced card is
mirrored to the ledger by the same actor in the same session:
`claim.taken` beside `seed task claim`, the plan's approval beside
the plan PR's merge, `submission.made` beside the task PR,
`merge.observed` by the observer when the PR merges, the deliberate
exits beside the card's park, release or review. The ledger's
projections are read on every wake, as the lane fragments say, and
never acted on alone.

**Divergence.** Once a day the folded contract states
(`contracts.json`) are diffed against the v1 card states of the slice
and the diff is appended to the log at the end of this packet: date,
position, card, v1 state, ledger state, the reconciliation. Every
divergence is reconciled toward v1 by an admitted act on the ledger,
never by editing history, and a divergence that recurs is a defect
card.

**The evidence at the end.** The five-bar audit over the real chain
(`simulate.Audit`, `internal/simulate/audit.go`, run over any ledger
an operator points at by `seed ledger audit`, os-7599c27d), the
report's `lanes` section for the two rates, the shape of every
`escalation.raised` payload, the receipts' independence on the
happy-path submissions. Each feeds the measurement ledger below
through a follow-up card that revises this packet.

## The cutover and the rollback

Criterion 5, in the build plan's three clauses.

### Which entry point flips when

Today `scripts/seed` (v1, the pinned engine) is the only coordination
entry point, and the build plan's ground rules keep it so "until
spin-out". The flip is one change, made after the shadow window has
closed with its divergences reconciled and the seven criteria all
`met`: the root `AGENTS.md` section "How work happens" is rewritten
around the Seed loop verbs (`seed situation`, `seed claim take`,
`seed submission make`, `seed claim release|park`, `seed escalation
raise`, `seed message read`, and mail sent as a `message.sent` append
through `seed ledger append --verb message.sent`, the one loop act
without a verb of its own today: the cutover pull request names that
form or adds the verb, and this packet assumes neither) and the lane
fragments under
`next/lanes/`, the Seed binary built into `next/bin/seed` becomes the
verb every role file names, and `scripts/seed task` is retired from
every role file and the dispatch and maintenance workflows in the
same pull request. That pull request is the cutover; its merge is the
moment authority moves, and it is the escalated decision the build
plan reserves. Before it merges, the v1 state ref is anchored one last
time (`scripts/seed state anchor`) and re-imported into the ledger at
that anchor (`seed import --from-open-seed`), so the ledger holds the
whole history at the flip.

### What stays authoritative where during the window

During the shadow window, v1 is authoritative for every card,
sliced or not; the ledger is a shadow that records the slice and is
read for orientation only. From the cutover's merge, the ledger is
authoritative for every contract filed after it and for every
imported contract, and v1's `seed-state` ref is frozen at its final
anchor and kept read-only for history, never written again. CI's
receipt gate (`receipt verify`, the plan-at-merge-base rule and the
reviewer-identity check) is unchanged through both: it is a property
of pull requests, not of the queue. The plan and task PR
conventions (`plans/<id>.md` merged first, `seed/<id>` never touching
`plans/**`) carry over unchanged, with the card id now a ledger
subject.

### The path back

The ledger is append-only, so rollback never rewrites it. The path
back is the flip reversed: the cutover pull request is reverted, the
frozen `seed-state` ref is unfrozen (its anchor tag is the position
to resume from), and every contract the ledger admitted after the
cutover that has no v1 card is listed from the contracts projection
(`seed project` at the rollback position) and filed as a card by the
dispatcher lane, with the ledger's positions cited in the card body.
Claims in flight on the ledger are released with packets, so the
work resumes from the packet on v1 exactly as a preempted worker
resumes. The ledger keeps running as a shadow, so the rollback loses
no history and the next cutover attempt starts from a longer record.

## The two cutovers are escalations

Neither cutover is autonomously decidable (build plan §5): spin-out
is the entry-point switch, and renaming the later publish does not
authorize the earlier authority switch. Agents drive the work up to
each gate, present this packet, and stop.

**Self-hosting.** Question: does this repository's own development move to Seed at the position the shadow window closed, on the terms in "The cutover and the rollback"?

Its preconditions are the seven criteria all `met`, the shadow window
closed with every divergence reconciled, and the compromised-actor
drill green on the commit that carries the cutover.

**Distribution.** Question: does Seed become what new users clone, and from which repository?

Its preconditions are self-hosting held for a stated period without a
rollback, a released Seed binary with checksums and provenance
(charter III.P row 1's one residual today: the binary is built from
source and is not yet a released artifact), and a README a team that
has never spoken to the authors can adopt from in under an hour (III.R
row 7).

## The III.R measurement ledger

The conformance table (`next/spec/conformance.json`, os-83bc3d84)
routes each row of charter III.R to a measurement and says the row
flips to `met` when this packet records it. No measurement exists
before the shadow run, so every row is `not measured`; a follow-up
card revises this table with the reading, the position it was read
at, and the surface it was read from, and flips the conformance row
with the packet as evidence. III.R stays open by construction until
the reserved steps run: the shadow run supplies R.1 through R.5, the
cutovers R.6, the distribution step R.7, and the Phase 13 exit record
(plans/os-d63c7441.md) does not close while any row is outstanding.
`not measured` is the packet re-deriving the frontier toward
promotion, not an omission in it.

| row | measure | surface | status |
|---|---|---|---|
| R.1 | one-sentence intents become routed contracts whose draft acceptance specs survive human review: the dispatcher's re-triage rate over the shadow window and a human-review sample of the draft acceptance specs it filed | `report.json` lanes.dispatcher.retriage_rate; the sample recorded in this packet | not measured |
| R.2 | planner plan PRs pass human review above 80% unedited and implementers reach verdict-passed submissions on the happy path: lanes.planner.unedited_rate above 0.800 and the receipts' independence on the window's happy-path submissions | `report.json` lanes.planner.unedited_rate; the verdict records' receipts | not measured |
| R.3 | the verifier lane holds quality alone on low tiers and humans review only high-tier plans: the tiers' independence levels and the verifiers' calibration agreement over the window | `next/spec/tiers.md` levels per tier; the calibration harness's agreement | not measured |
| R.4 | every escalation is one packet, one question and one decision, a transcript dump filed as a defect: the shape of every escalation.raised payload in the window | the chain's escalation.raised payloads through `seed decision` | not measured |
| R.5 | the system runs unattended for a week on a real backlog with zero chain violations, zero lost updates, zero silent abandonments, zero guardrail breaches and zero unreserved spend: the five-bar audit over the real chain at the window's end | `seed ledger audit` over the shadow ledger (`simulate.Audit`) | not measured |
| R.6 | the flywheel demonstrably compounds over a quarter: chore-to-workflow conversions, the packet-resume rate and the cost per contract over the quarter after the self-hosting cutover | `report.json` flywheel and knowledge sections; the budget records | not measured |
| R.7 | a team that has never spoken to the authors adopts from the README in under an hour on its own forge and reaches a verifier-passed, human-reviewed PR the same day: the first external adoption after the distribution step | the adopting team's report, recorded in this packet | not measured |

## The divergence log

Empty until the shadow window opens. Each entry: date, position,
card, v1 state, ledger state, the reconciliation.
