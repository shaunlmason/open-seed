# progress.md — the Seed implementation frontier

The single resume point (docs/next-build-plan.md §4): one line per plan item,
`phase.item — card id — PR — state`. A fresh agent resumes from this file
alone: read it top to bottom, verify the PR states it claims against the
forge, then take the frontier line's next action. Keep it truthful in every
PR that touches `next/**`; never start new work while it misstates the
frontier.

## Roster and conventions

- Implementer actor: `seed-next-implementer` (work verbs). Operator queue
  verbs for `next:` cards run under the session principal per
  `decisions/0003-next-loop-delegation.md`.
- Cards carry the `next:` title prefix; branches `seed/<card>` (task) and
  `seed/<card>-plan` (plan); receipts generated with `--run --write` before
  review.
- Phase gate: a phase starts when its dependencies' exit criteria are
  **merged** (docs/next-build-plan.md §2).

## Ledger

- 0.1 module scaffold + CI wiring (`make check-next`) — os-116ca9ac — plan
  PR #71, task PR #72 — **done** (merged; card closed)
- 0.2 spec skeleton (`next/spec/protocol.md`, `next/spec/envelope.md`) —
  os-116ca9ac — task PR #72 — **done** (merged; encodings and upgrade-safe
  version semantics per #71/#75 review)
- 0.3 decision log (`next/docs/decisions.md`; plus this frontier file) —
  os-116ca9ac — task PR #72 — **done** (merged)
- 1.1 event model + JCS + Ed25519 — os-aa146827 — plan PR #73, task PR
  #76 — **done** (merged; strict wire parsing + lowercase-only hex per
  review; card closed)
- 1.2 chain/segments/HEAD — os-ead12024 — plan PR #74 (merged, amended),
  task PR #79 — **done** (merged; card closed)
- 1.3 genesis via `seed init` — os-d636299d — plan PR #75 (merged,
  amended), task PR #83 — **done** (merged; genesis-payload version
  bootstrap + position-stamped init refusal per review; card closed)
- 1.4 push-race append loop — os-62e2aa1d — plan PR #81 (merged,
  amended: monotonic head + rollback drill), task PR #86 — **done**
  (merged; vanished-ref regression refusal + typed remote-rejection per
  review; card closed)
- 1.5 halt semantics in the rule set — os-bce3fb98 — plan PR #78 (merged,
  amended: exit code 7, reason-carrying state), task PR #84 — **done**
  (merged; halted refusal ordering + empty lift payload per review; card
  closed)
- 1.6 payload classification lint + hostile corpus — os-d6f81ec6 — plan
  PR #77 (merged, amended: aggregate free-text budget, embedded rules),
  task PR #80 — **done** (merged; narrowed anchor exemption + RFC 6901
  pointers per review; card closed)
- 1.7 CLI `seed ledger verify/append/show` — os-89412090 — plan PR #82
  (merged, amended: envelope v0 preserved, exit 9), task PR #85 —
  **done** (merged; position-stamped verify refusals, read-only show,
  active-version append per review; card closed)

**Phase 1 exit (charter III.A): met.** The chain verifies from genesis in
one command (`seed ledger verify`, #85); corrupted fixtures (reordered,
rewritten, forged-sig, bad-prev, lying HEAD, lying genesis) are detected
with positioned reasons (#79/#83/#85); the hostile classification corpus
passes (#80); the race drill (two concurrent appenders, no lost updates,
a real retry observed) is green on main (#86). This exit record is card
os-beac85e1's task PR (an administrative card, not a Phase 2 item).

## Phase 2 — Admission (docs/next-build-plan.md Phase 2; deps: 1 ✓)

- 2.1 admission rule set library (`internal/admit`) — os-3898f232 —
  plan PR #88 (merged, amended: ledger `WithObserver` + shared upgrade
  schema in scope), task PR #90 — **done** (merged; card closed)
- 2.2 cooperative posture (client self-validation) — os-895bf828 —
  plan PR #91 (merged, amended: gitref verify-option pass-through +
  update-phase race marker), task PR #93 — **done** (merged; state-dir
  lock + position-stamped refusals per review; card closed)
- 2.3 enforced posture, `seed-admit` pre-receive hook — os-d3591e09 —
  plan PR #92 (merged), task PR #94 — **done** (merged; tree-shape gate
  + exclusion-based rule selection per review; card closed)
- 2.4 posture declaration + `seed doctor` — os-3c72f93f — plan PR #95
  (merged, amended: exit 13 `posture_invalid` allocation per review),
  task PR #98 — **done** (merged; exit 66 `unreadable` for read
  failures + valid-postures wording per review; card closed)
- 2.5 admission drills (raw-git adversary; kill-and-replace) —
  os-028dda91 — plan PR #96 (merged, amended: fresh-clone
  kill-and-replace per review), task PR #99 — **done** (merged;
  shared adversary table under both postures + original host deleted
  with an independently installed hook copy, per review; card closed)

## Phase 3 — Identity and grants (docs/next-build-plan.md Phase 3; deps: 2)

- 3.1 actor events + keyring projection — os-52a2d688 — plan PR #97
  (merged, amended: root-liveness guard + seed/1 activation boundary
  per review), task PR #100 — **done** (merged; standing/grant rule
  split per review; card closed)
- 3.2 admission checks grants per verb — os-3979d48b — plan PR #101
  (merged, amended: versioning stance held, checkpoint accepts
  maintenance|operator, per review), task PR #102 — **done** (merged;
  vocabulary test parses the normative spec table per review; card
  closed)
- 3.3 key rotation/revocation drill — os-d1f35a8c — plan PR #103
  (merged, amended: exit record scoped to the III.E subset with the
  full unmet-remainder enumeration, per review), task PR #104 —
  **done** (merged; card closed)

**Phase 3 exit (the III.E subset docs/next-build-plan.md scopes): met.**
Signature and grant checks live in admission (#100: standing-aware
resolution behind the seed/1 boundary, root liveness, grandfathering;
#102: the capability vocabulary checked on every verb, operator-only
refusals structural, delegation via actor.granted proven end-to-end,
kind a drilled assertion), and the revocation drill is green (#104:
rotation with history attributed and every post-revocation proposal
refused at the rule set, the seed-admit boundary, and the CLI; the
compromised-key cut per posture with the cooperative consequence
observable; terminality, grants-die-with-standing, and root liveness
held at the boundary). Still-unmet III.E criteria, by landing phase:
implementer-disjoint self-approval and sealed-check keyring rotation on
verifier-key revocation (Phase 6); claim reaping on revocation (Phase
5); qualification tuples, grants citing them, and scheduled spot-check
suspension (Phase 10); the roster distinction consumed by agent-only
guardrails and human/agent metrics (with those surfaces, Phases 8 and
11).

**Phase 2 exit (charter III.B subset): met.** The validator is the
guarded ref's sole writer under enforced posture (#94: invalid-stream,
rewrite, truncation, force-update, and deletion refusals, tree shape
included); statelessness is drilled by kill-and-replace (#99: a fresh
host rebuilt from a bare clone makes identical decisions and the chain
verifies from genesis); the posture declaration prints the cooperative
consequence verbatim (#98); the raw-git adversary drill runs per posture
with the cooperative consequence made observable (#99); both postures
are selectable in fixtures through the shared declaration (#99).
Capability rules slot in at Phase 3, fences at Phase 5, reservations at
Phase 7, as added rules on the shared set.

## Phase 4 — Projections (docs/next-build-plan.md Phase 4; deps: 1 ✓)

- 4.1 projection engine (deterministic build, stamps, one-command
  rebuild) — os-4d5cacff — plan PR #105 (merged, amended: immutable
  builds + `CURRENT` pointer publication, roster includes genesis
  roots, overlap refusal, full-tree immutability and stale-`HEAD`
  fixtures, interrupted-publication drill, per review; second round:
  version-bearing build ids so derivation changes republish, and the
  superseded build survives one swap for in-flight readers), task PR
  #109 — **done** (merged; card closed)
- 4.2 standard projections (contract detail, ready queue, actor view,
  report skeleton, `seed project current`, exit 15 `stale`) —
  os-fecfb3f7 — plan PR #106 (merged, amended: the ready queue ships
  registered with `derivation: "none"` v0, per review), task PR
  #111 (stack-collapsed; its diff never reached main), re-landed as
  task PR #117 (merged, amended per review: registry-validated
  consumer verb, absence-4/damage-5 split refined so a layout
  missing its CURRENT pointer is damage, incomplete-stamp refusal,
  stale envelope position, unprivileged-cleanup test fix) —
  **done** (merged; card closed)
- 4.3 SQLite cache projection + mid-operation deletion drill —
  os-acc1ac78 — plan PR #108 (merged) + amendment PR #110 (merged:
  the cache is a registered projection, byte-identical like every
  view; the stamp table carries exactly the tree stamp's fields),
  task PR #119 — **done** (merged; card closed; Phase 4 complete)
- 4.4 write-boundary lint wired into check-next — os-8d5e9c45 — plan
  PR #107 (merged, amended: seam/write-separation lint + locked trees
  `0444`/`0555` with the engine unlock window, deletion via rebuild,
  per review), task PR #112 (stack-collapsed; its diff never reached
  main), re-landed as task PR #118 (merged, amended per review:
  openDirs partial-open rollback, lint vocabulary derived from the
  engine's declarations with a behavioral layout probe,
  unprivileged-cleanup lesson recorded) — **done** (merged; card
  closed)

## Phase 5 — Lifecycle, claims, packets (docs/next-build-plan.md Phase 5; deps: 3 ✓, 4 ✓)

- 5.1 transition table as data + lifecycle verbs — os-d69a6c91 —
  plan PR #113 (merged, amended: charter Appendix catalog vocabulary,
  merge.observed-only done, capability rows, completeness presence at
  the claimability transition, per review) — **done** (merged #122;
  card closed; the fold's seed/1 activation boundary rides follow-up
  #129, per the post-merge review round):
  transitions.json + self-validation, the lifecycle admission rule
  across rule set/hook/CLI, dispatch/claim capability lanes,
  contracts v2 (state + anomalies), queue v2 ("transitions/1"),
  cache generation 2, spec/lifecycle.md
- 5.2 claims with fences — os-5dc16a7c — plan PR #114 (merged,
  amended: prior claimants stay fenced, per review) — **done**
  (merged #123; card closed; the claimless-citation fence fix rides
  follow-up #128, per the post-merge round): the exclusive table
  flag, the claim fold (holder, fence, prior claimants), the fence
  rule between grant and lifecycle, structured contention, the
  online-only client seam, contracts v3 with the claim object, cache
  generation 3, the claim race storm and offline-boundary drills
- 5.3 four-part handoff packets — os-b07b0f59 — plan PR #115
  (merged, amended: packets on ALL four exits incl. submission; the
  3072 canonical bound fits the payload cap; the mandatory base
  range; combined anchors, per review) — **done** (merged #124;
  card closed; the resume-drill and packet-shape hardening landed
  via follow-up #127, per the post-merge round): internal/packet strict schema, the packet
  admission rule, the classifier's bare-range exemption, tolerant
  fold counting packetless/fence-violating exits, the A/B resume
  drill
- 5.4 acceptance-spec field + spec gate — os-73c00a50 — plan PR #116
  (merged, amended: no trivial-tier gate exemption; gate evidence
  bound to the acceptance revision, per review) — **done** (merged
  #125; card closed; the explicit executable marker and the -p 1
  coverage serialization landed in-PR per review): the structured acceptance field with
  the universal gate rule and ref/gate commit equality, the
  propose-vs-arm proposal shape rule, contracts v4 with the
  {ref, executable, gated} object, cache generation 4
- 5.5 falsifiable-plan lint + plan-gating — os-16c1d142 — plan PR
  #120 (merged, amended: two-layer plan binding with the Phase 6
  receipt closing the ancestry hole; classify ships as an invocable
  check, per review) — **done** (merged #126; card closed; the
  anchor-equality gate and labeled lint landed in-PR per review; the
  receipt's two plan-line rows go green with engine pin #130):
  internal/plan lint + classifier, seed plan lint/classify verbs,
  exit 16 plan_required, the submission plan gate over the fold's
  tier and plan.approved facts, plan.* capability rows
- 5.6 observation streams v0 + expiry vs. wedge — os-2ff8dbf1 — plan
  PR #121 (merged, amended per review: input-bearing build identity,
  fence-keyed streams, position throttle) — **done** (merged #131;
  card closed):
  internal/obs (fence-keyed JSONL streams, snapshot digest, pure
  classification), the engine Inputs seam with report Version "2"
  and the -i<digest12> build identity, milestone/wedge admission
  (monotonic counts, 25-position spacing, operator wedge facts),
  seed obs emit + rebuild declared inputs, spec/observations.md

**Phase 5 exit (the III.F subset docs/next-build-plan.md scopes): met.**
Met means the subset the plan's own Phase 5 item list and exit drills
assign to this phase, the scoped-exit posture of the Phase 2 and 3
records: the plan itself defers gate-before-run to Phase 6 (item 4's
named deferral), racing to the fixed-defaults backlog and Phase 13
(item 1 there names it as the III.F remainder), sealed checks to 6.3,
and the dependency-cascade row to the Phase 13 catch-all, so those
III.F rows are later phases' bound work, not this exit's.
The lifecycle vocabulary and
transition rules are self-validating data enforced at admission, and
claim is a transition, not a state (#122, with the fold honoring the
seed/1 activation boundary via #129). Claims are exclusive with
fencing granted only at admission: stale or missing citations refuse
exit 6 naming the cited fence, the active fence, and the holder;
contention returns the structured exit-2 envelope; prior claimants
stay fenced; and a claiming verb cannot smuggle a citation onto an
unheld subject (#123, #128). The claim race storm drill is green on
main, and offline claiming is impossible by construction — exclusive
verbs refuse the local append, drilled at the offline boundary
(#123). Every exit from `in_progress` is one of the four deliberate
exits and each carries a four-part, shape-linted, size-bounded
packet, so silent abandonment is impossible (#122, #124; null parts
refused via #127). Packet sufficiency is drilled end-to-end: a fresh
executor resumes from the packet alone, performs the recorded
unfinished work, lands it, and never re-tries recorded dead ends
(#124, hardened by #127). A contract cannot become claimable without
the structured acceptance field; executable content requires gate
evidence bound to the ref's revision at every tier, and outside text
can propose but never directly become acceptance content (#125).
Plans are falsifiable — boundary set, retention set, validation
commands, expected diff shape, missing retention fails lint — and
submission is plan-gated above the trivial tier with plan and task
PRs structurally disjoint (#126; the receipt's plan-line commands
execute under engine v0.15.1, pinned by #130). Observations v0 close
the phase: fence-keyed lossy streams, monotonic position-throttled
milestones, and expiry vs. wedge rendered as distinct states in the
report (#131). Still-unmet III.F criteria, by landing phase: sealed
checks and their honest-scope documentation (Phase 6, the exit
line's named carve-out); gate-before-run enforcement — the verifier
refusing ungated content (with verdicts, Phase 6); racing mode as
the per-squad opt-in (Phase 13 item 1); and the dependency-cascade
row (advisory wakes land with Phase 7's supervisor; holds,
initiative rollups, and goal-ancestry warnings have no named phase
item and fall to Phase 13's catch-all). This exit record is card
os-6e37b10e's task PR (an administrative card, not a Phase 6 item).

## Phase 6 — Verdict pipeline (docs/next-build-plan.md Phase 6; deps: 5 ✓)

- 6.1 submission binding, verifier workspace, receipt computation,
  verdict.rendered at L1 — os-f6d2c267 — plan PR #133 (merged,
  amended per review: sandbox runner profile with declared
  capability, immutable head binding, transcript-derived pass
  rule) — **done** (merged #135; card closed; the review round
  landed copied clone objects, the post-review check, and
  stored-evidence verification in-PR): internal/verdict (origin-stripped clone
  workspaces, the exec runner profile named in every receipt, JCS
  receipts with full immutable SHAs and plan-at-merge-base),
  internal/artifact, the verdict admission rule (review-only,
  submission binding, L1 independence exit 17) with the verdict-only
  capability row, gate-before-run exits 18/19, seed verdict
  receipt/render/check with the transcript-derived render rule
  (exit 20) and recompute-and-mismatch (exit 21), plan.Commands,
  spec/verdicts.md
- 6.2 reconciliation chain (merge.requested, merge.observed, done) +
  divergence drills — os-6cdc15be — plan PR #136 (merged, amended
  per review: target-ref observation replaces the unreachable
  two-sha signal; honest ancestry cases with attested_divergence as
  a surfaced state) — **done** (merged #137; card closed; the review
  round landed unlaunderable verdicts at both admitted chain steps
  plus reconcile.VerifyVerdicts/verdict_unverified, the
  coexisting-failure evidence walk, explicit-null chain fields, and
  the cache reconciliation row): merge.requested piped
  (pass-verdict citation, review-only, claim lane), merge.observed
  deepened to the observer's forge fact behind the full chain rule
  with the observer capability lane, fold chain facts, the
  internal/reconcile classifier (merge_without_verdict,
  chain_skipped, neutral unreconciled), contracts v6 / report v3
  reconciliation section / cache generation 5, and seed reconcile
  with the evidence-grade checks (attested_divergence,
  target_rewritten, evidence_missing) and the induced-divergence
  drills, spec/reconciliation.md
- 6.3 sealed checks (salted commitment, age-encrypted body, rotation
  re-encryption, capability audit) — os-3128535a — plan PR #138
  (merged, amended per review: the pre-claim window closes the
  release-path laundering hole, verdict check recomputes sealed
  transcripts, the sealer lane drops the operator fallback, empty
  seals refuse at both ends; the task-PR review round then pinned
  age to the plan's v1.2.1, filtered implementation-capable keys out
  of the recipient set, added the position-accurate authoring check
  at unseal plus the seal_unverified reconcile class, and validated
  the decrypted salt's shape) — **done** (merged #139; card
  closed): check.sealed
  admitted only in ready with no prior claim (one commitment per
  subject; raw seals outside the window fold as anomalies, never
  facts), the salted JCS envelope with the salt inside the
  ciphertext, internal/seal over filippo.io/age v1.2.1 (agessh
  recipients = the verdict-granted keys; the audit's header tag
  scan), the mutable sealed bucket keyed by commitment, the sealer
  capability with grant disjointness against claim and operator
  (root's implicit operator included) plus the seal-author claim
  refusal and the capability-audit decrypt drills, render's
  unseal-and-run with sealed transcripts in the receipt behind
  exits 22/23/24 (the above-trivial gate), check's sealed
  recompute-and-mismatch, seed seal create|rotate|audit (rotation
  re-encrypts open subjects without touching the ledger), the
  reconcile unsealed class, contracts v7 / report v4 / cache
  generation 6, spec/sealed-checks.md
- 6.4 red-verdict lockout + operator override verb — os-d2497eb7 —
  plan PR #140 (merged, amended per review: only boundary-validated
  fails lock or authorize, and the override requires a standing
  validated fail on the current submission — an escape hatch, never
  a bypass) — **done** (merged #141; card closed; the review round
  bound the lockout, the return, and the override to
  boundary-validated fails only, revalidated the override's citation
  at the chain steps, and taught the fold's anomaly check the
  override path): contract.returned resolves
  lifecycle.md's named extension point (review to ready, citing the
  authenticated fail, dispatch/operator lanes; prior facts and the
  seal survive); the lockout scans the whole submission window
  (SubmissionFails) so a raw later verdict never buries an authentic
  fail, refusing pass at admission and at render exit 25 red_locked
  until a new submission; merge.overridden as the operator-only
  attributable fact (strict reason + fail citation, one per window,
  folds as OverrideFact never a verdict) with the citation choice in
  merge.requested ({verdict} xor {override}) and the override-backed
  observed path, both steps validating the override signer;
  reconcile VerifyOverrides with the overridden (neutral, by name;
  override-backed done is never merge_without_verdict) and
  override_unverified classes; contracts v8 (override explicit-null),
  report v5, cache generation 7; lifecycle/verdicts/reconciliation/
  actors spec updates

**Phase 6 exit (the III.G subset docs/next-build-plan.md scopes): met.**
Met means the subset the plan's own Phase 6 item list and exit drills
assign to this phase, the scoped-exit posture of the Phase 2, 3, and
5 records: the plan itself defers both L2 runtime-tuple separation
and L3 deterministic-first verification (the per-tier level
declarations) and rubric-verdict calibration (per-item evidence, the
human gold set, drift-triggered authority suspension) to Phase 10,
whose exit names levels and calibration together, so those III.G
rows are Phase 10's bound work, not this exit's.
The exit line's three named drills are green on main: the
induced-divergence reconciliation drills (#137's
merge_without_verdict, chain_skipped, neutral unreconciled,
attested_divergence, target_rewritten, evidence_missing, and
verdict_unverified, extended by #139's unsealed and seal_unverified
and #141's overridden and override_unverified — every class induced
and detected); the receipt recompute-and-mismatch test (#135's exit
21, with #139 putting sealed transcripts inside the recompute
boundary so invented sealed outcomes fail the same way); and the
sealed-check audit (#139: sealer/claim and sealer/operator grant
disjointness both directions, the seal-author claim refusal, the
implementer-cannot-decrypt drills, the recipient-tag audit, and
rotation that touches no history). The III.G rows each merged PR
evidences: done is reachable only through the chain with each step
its own event and no code path collapsing them (#137; #141's
override-backed path stays uncollapsed); verdicts are signed by
verdict-granted keys provably disjoint from every implementing key,
and operator override is its own attributable verb, never a
disguised verdict (#135's L1 exit 17; #141's operator-only
merge.overridden folding as its own fact, surfaced by name);
the verifier executes in clean per-run isolation with cleanup firing
pass or fail and enumerable, exclusively self-executed inputs
(#135's origin-stripped workspaces); receipts bind contract id, plan
hash at merge-base, diff hash, changed-file inventory, visible and
sealed check transcripts, and environment fingerprint, and
verification recomputes everything from the submission head (#135,
#139); and a red verdict is unmergeable and locks the implementer
out of self-approval until a new submission (#137's chain-legality
half; #141's lockout exit 25 with contract.returned as the resolved
return path). The Phase 5 exit's named sealed-checks carve-out is
closed by #139, honest-scope documentation included. This exit
record is card os-600be59e's task PR (an administrative card, not a
Phase 6 item).

## Phase 7 — Supervisor, offers, budgets (docs/next-build-plan.md Phase 7; deps: 5 ✓)

- 7.1 offers (offer.published eligibility-scoped and expiring;
  workers pull and claim; the wakeless poll-only drill proving wake
  is advisory) — os-c61c3392 — **done** (merged #145; card closed;
  plan #144 with the #146 validation-command amendment; the
  supervise lane, claimed-or-expire liveness with the claim as
  consumption boundary, foreign offers inert at the list surface,
  the poll-only and race drills, and the review round's operator
  discoverability and last_claim byte-identity fixes; contracts v9 /
  report v6 / cache generation 8)
- 7.2 budgets (budget.reserve / settle / release; admission
  decrements reservations; the reservation race drill; per-adapter
  risk limits) — os-cecac5de — **done** (task PR #149 merged,
  against plan #147 as amended by #148; class-table capacity,
  holder-only reserves, derived closes with foreign facts inert on
  every surface, the two-drafts-one-view race drill, the empty
  spending gate, contracts v10 / report v7 / cache generation 9)
- 7.3 executor adapter interface + the local worktree adapter
  (provision / wake / meter / report-tuple; metering to the
  observation stream; run.settled aggregate) — os-1dad487d —
  **done** (task PR #151 merged against plan
  #150: the public next/executor package, the
  reserve/start/provision/meter/settle bracket with run.started
  filling the spending table, the SIGKILL disposability drill,
  contracts v11 / report v8 / cache generation 10; the merge raced
  the final review-round push, so two findings — Provision
  authenticating folded starts via the shared admit.RunStartValid,
  and worktree rollback on provisioning failure — landed in
  follow-up task PR #152, merged)
- 7.4 graceful preemption (safe-point park with packet; force reap
  packet) — os-0f718b4e — **done** (task PR #154 merged against
  plan #153; card closed: run.interrupted as a once-per-fence
  supervisory ledger fact with position-accurate validity shared by
  admission and polling workers, safe-point semantics specified as
  the worker contract, the graceful park drill and the force reap
  drill both completing elsewhere, contracts v12 / report v9 /
  cache generation 11; the review round hardened the validity
  helpers to re-run each verb's strict payload decode at the fact's
  own position, gated once-per-fence settles on boundary validity,
  and indexed the interrupts cache table)

**Phase 7 exit (the III.H subset docs/next-build-plan.md scopes): met.**
Met means the subset the plan's own Phase 7 item list and exit line
assign to this phase for the implemented adapter, the scoped-exit
posture of the Phase 2, 3, 5, and 6 records. The exit line's three
named drills are green on main: the wakeless poll-only run through
the whole loop (#145: wake is advisory transport whose total
failure costs only latency); the reservation race drill (#149: two
drafts admission-checked against one view, the second refusing at
append, so concurrent over-spend against one budget is structurally
impossible); and the disposability drill (#151: a real worker
subprocess SIGKILLed at a seeded randomized site after an admitted
synchronization, the chain verifying, the contract completing
elsewhere from the surviving ledger alone, and the loss window
exactly the post-sync observation lines, stated honestly). The
III.H rows the merged PRs evidence for the implemented adapter:
scheduling is offers-and-claims with eligibility-scoped, expiring
offers, workers pulling and claiming, and exclusivity settling at
admission so duplicate scheduling is impossible and no assignment
can orphan (#145); spending verbs require an admitted
budget.reserve with execution fenced to the reservation —
run.started fills the spending table, Provision refuses outside the
gate, and settle/release records actuals (#149, #151, with #152
authenticating folded starts position-accurately so fold presence
is never admission); metering flows on the observation channel and
settles to the ledger at run end via run.settled, drilled through
the full bracket (#151, #152) — with the row's universal half, "no
execution path is unmetered", recorded unmet at this exit, not
claimed: Meter is caller-optional and nothing structurally prevents
an unmetered caller of the public adapter interface, so what this
phase landed is the metered path and the drilled bracket, while the
universal half routes below (budget consumption stays enforced
upstream at the reservation gate either way, so an unmetered run
under-reports telemetry, never overspends); the adapter interface
is public with the local worktree adapter as its first
implementation (#151, #152); and
preemption is graceful-first with safe-point semantics specified as
the worker contract and a force kill still yielding an honest reap
packet (#154 — landed beyond the exit line's three named drills, so
the III.H preemption row closes with the phase rather than
waiting). The local worktree adapter can be stopped synchronously,
so the risk-limit posture next/spec/budgets.md declares has no
per-adapter declaration to make yet. The unmet III.H remainder
routes as obligations named in the landing phases' own build-plan
text: the container, cloud-session, and enrolled-remote adapters
with their per-adapter risk-limit declarations (Phase 13 item 2);
the remaining-reservation budget block in the worker's envelope
(Phase 8 item 1); exhaustion parking — the worker loop that answers
the budget gate's structured refusal by taking the 7.4 claim.parked
exit with its packet, consuming Phase 8's budget block — named in
Phase 9 item 1's worker-lane loop; the metering row's unmet
universal half in two named steps — detection as Phase 9 item 3's
unsettled-run lint (position-anchored: a started window lacking its
run.settled is flagged only once the subject takes a subsequent
claim window or reaches a terminal state), and the
full-conformance claim on Phase 13's exit line, which already
requires the named III.H criteria green; and scheduling inputs
completing with routing-as-data in the Phase 9 dispatcher and
qualification tuples in Phase 10 item 1 (cost classes exist today
in the filed budget class). This exit record is card os-c9e24032's
task PR (an administrative card, not a Phase 7 item).

## Phase 8 — Affordance envelopes (docs/next-build-plan.md Phase 8; deps: 5 ✓)

- 8.1 affordance envelope (affordances from the same internal/admit
  rule set; position; budget block; structured errors, exit codes)
  — os-f5551001 — **done** (#160 merged, card closed; plan #158 as
  amended:
  admit.Affordances probing every catalog verb with signed drafts
  through the enforcing Check, the completeness-pinned synthesizer
  table with the actor.enrolled carve-out, the budget block from
  BudgetViewAt, stamping on every append path success and refusal
  plus budget status --key, the lifecycle-walk and CLI stamp
  drills)
- 8.2 regression class: affordance-listed verb refused for legality
  at the same position = bug — os-148d3ba1 — **done** (#163 merged,
  card closed; plan #162: the walk's history extracted into the shared
  walkScript, the prefix-sweeping TestAffordanceRegressionClass
  re-drafting every listed verb through the enforcing Check for all
  seven enrolled-lane pairs at every position, and the CLI drill
  pinning that stamped position and stamped list agree with
  independent recomputation on success and refusal envelopes)
- 8.3 refusal-rate metric in the report — os-edf73d66 — **done**
  (#165 merged, card closed; plan #164 as amended: the attempts journal
  journaling both outcomes best-effort at every stamped
  admission-boundary seam, the journal as a declared digest-covered
  report input via --refusals, the nullable refusals section with
  one-population counts and the four-decimal rate, report v10,
  next/spec/refusals.md, and the D4 drills including the
  1-refusal-beside-100-admissions=0.0099 fixture; the review round
  hardened the journal against torn short writes — Note restores the
  previous length when the fragment is provably the tail, and Load
  treats the terminating newline as the commit marker so an
  uncommitted fragment cannot poison every future declaring build,
  while terminated lines stay strict)
- (out of item) ledger writeHead race fix — os-c6fb95ee — **done**
  (#161 merged, card closed: per-writer unique temps preserving the
  established HEAD mode, with the store-level contention regression
  test; the flake tripped TestGracefulPreemptionDrill on main)

**Phase 8 exit (the III.I subset docs/next-build-plan.md scopes):
met.** Met means the two criteria the plan's own exit line names,
the scoped-exit posture of the Phase 2, 3, 5, 6, and 7 records. The
same-rule-set property test is green on main (#163: every listed
verb independently re-drafted and run through the enforcing Check at
every prefix position of the shared walk scenario, for all seven
enrolled-lane pairs, with determinism and strictly ascending output
asserted, so a later split between computation and enforcement fails
as a named class rather than drifting), and the envelope schema is
stable and versioned (seed-envelope/0, its bump discipline stated in
next/spec/envelope.md, exit codes allocated in the spec table rather
than ad hoc). Charter III.I as a whole is NOT claimable at this
exit, so all five rows are walked rather than the two the exit line
scopes. Row 1 (versioned schema-stable envelope, structured errors,
meaningful exit codes, the verbs currently legal for this actor on
this subject, the position it was computed at): met by #160, with
three bounded carve-outs named rather than glossed — a response
carrying no actor-and-subject context (keyless read surfaces; probes
must be signed, so a fingerprint alone cannot compute a list) lists
the empty set, which is the honest answer where no such set exists;
actor.enrolled is listed only where the prober could supply the
subject's public key, which no fingerprint-holder can derive; and a
refusal that never opened a ledger carries a null position rather
than inventing one, which the spec sentence corrected in this PR now
states. Row 2 (one rule set for computation and enforcement;
listed-then-refused at the same position a bug class with a
regression test): met by #160's computation plus #163's sweep. Row 4
(refusal rates tracked as an affordance-gap metric): met by #165's
attempts journal — attempts, both outcomes, journaled best-effort at
every stamped admission-boundary seam — and report v10's refusals
section, whose rate draws numerator and denominator from that one
population rather than from the chain. Row 3 (the CLI is the
complete interface; a machine-protocol surface exposes identical
semantics; platform parity including Windows documented and tested)
and row 5 (matching promoted lessons surface in packets and
envelopes at claim time) are recorded UNMET, not claimed: no
machine-protocol surface and no platform-parity documentation or
test exist today, and row 5 presupposes the curator's promoted-lesson
store. Neither was routed anywhere in the build plan, so this task
extends the plan to name them in the landing phases' own text — row
3 as Phase 13's machine-protocol-and-parity item, with III.I added
to that phase's exit line, and row 5 in Phase 11 item 2's promotion
gate, beside the applies-when it already records. Full charter III.I
conformance is therefore claimable only once Phase 11 and Phase 13
both close. This exit record is card os-ef715d17's task PR (an
administrative card, not a Phase 8 item).

## Phase 9 — Lanes, escalation, maintenance (docs/next-build-plan.md Phase 9; deps: 6, 7, 8 ✓)

- 9.5 the lane-facing surface, parts (a) and (b) — os-52d5da3f —
  **done** (merged #171 against plan #170: internal/obligation
  deriving what is owed from the same fold admission enforces, with
  discharged_by read from the transition tables and a closed
  fact-shaped set for the verbs that change no state; the obligations
  projection registered alongside the others; seed situation
  rendering one position-stamped envelope whose --since is a complete
  change report with an explicit discharged list; and the
  dischargeability sweep over every prefix of the shared walk
  scenario, which caught a real modeling error on its first run)
- 9.5 part (c) loop verbs — os-7e197768 — **done** (merged #173
  against plan #172: the seven acts that close poll → claim → work →
  meter → submit → exit, each deriving the fence from the active
  window, the reservation from the shared budget view, the plan
  anchor from the approval, and the resume range from the
  repository, and each pre-flighting through the same admit.Check
  admission enforces so a refusal carries the boundary's own error
  beside the caller's affordances; claim take remote-only; the
  packet validated at the door; new next/spec/loop-verbs.md). The
  post-merge review found three defects, all live on main and carded
  as os-9b3f3ef3: derived arguments not re-derived across the
  optimistic retry, a remote refusal stamped from the stale view, and
  a JSON null packet panicking the CLI.
- **9.5 part (b) is MET** (os-8451d939, plan #209): the situation read
  carries the caller's messages, with "unread" derived from the cited
  `--since` position rather than stored read-state, so no
  `message.read` verb exists and `message.acked` stays unimplemented.
  It carries **notices, not bodies** — sender, contract, position,
  size — because `message.sent` needs no capability at all and the
  orienting read is taken on every wake, unbidden; the injection sweep
  now plants a hostile message addressed to the reading worker, and
  both the addressed and broadcast paths through the filter. The body
  is reached by `seed message read --at <position>`, the deliberate
  second act, which appends nothing and refuses a non-recipient with
  the same `not_found` a position holding no message gets. A `to` that
  is present and does not parse addresses NOBODY rather than everybody
  (review finding on #209): broadcasting a malformed address widens
  delivery from one intended recipient to every actor on an encoding
  slip.
- 9.1 lane role fragments, dispatcher least-capability, injection
  corpus, worker-lane loop with exhaustion parking — its
  role-definition text binds four ergonomic obligations (os-68ea0b2d):
  one position-stamped read named in each fragment, the loop acting
  through the item 5(c) loop verbs rather than the raw append seam,
  liveness riding the loop's own steps with no heartbeat verb, and the
  one-inbox doctrine. Too large for one plan, so split three ways the
  way item 5 was:
  - 1a the six lane fragments — os-cf1c9688 — **done** (merged #188;
    card closed: next/lanes/** as manifests plus ordered prose
    fragments, the four obligations as DECLARED FIELDS rather than
    paragraphs, and internal/lane checking each against an authority
    elsewhere — capabilities against internal/keyring, acts against the
    new internal/loopverb registry, and a lane's grants against what
    keyring.AcceptedCapabilities says each act's verb accepts. Plus
    `seed lane list|show|validate` and exit 26 lane_invalid)
  - 1b dispatcher least-capability drilled by the injection
    conformance suite (hostile corpus) — os-b779b4c7 — **done**
    (merged #192; card closed). III.J's second row is now
    **two-thirds met**, and the spec says so rather than reporting it
    closed: intents and tool output are covered, mirrors are not,
    because request.* has zero rows in the transition table.
    - Reachability is derived from admit.Affordances, which drafts a
      signed probe per catalog verb through the same Check pipeline
      admission enforces. NOT from keyring.AcceptedCapabilities: that
      table returns nil for the standing-only class, so a capability
      filter silently omits message.sent.
    - Three residuals named and pinned by characterization drills:
      the filed tier's "trivial" exempting both the plan gate and the
      sealed-checks lint (carded os-be12ac16 for Phase 10's tier
      system); claim.reaped consulting no liveness evidence, its two
      preconditions being freshness and attribution rather than
      authorization; and message.sent needing NO capability at all,
      which is the one that relays.
    - Tool output cannot inject, structurally: verdict.Transcript
      hashes output bytes at the boundary and drops them.
    - The projections carry every payload verbatim by design, which is
      where the unlanded mirror arm will land: whichever card adds
      request.* inherits an input already carrying hostile text.
  - 1c the worker loop made executable with exhaustion parking —
    os-abb206c8 — **done** (merged #191; card closed). Four review
    findings arrived after it merged and land as os-378e44f3: the
    actor derived from the key rather than supplied beside it (and
    re-derived each iteration, since a rotated key reopens the same
    mismatch through the filesystem), a claim reaped before the
    post-claim read ending the iteration idle rather than erroring,
    the fence adopted from the act that opened it so the claim is
    observable under its own window, and packet temp files unlinked
    on every path. It inherited
    1a's unfinished half and closed it: the loop emits as a side-effect
    of a declared liveness act that SUCCEEDED, keyed to its own actor
    and the fence its orienting read reports, so `liveness_from` is now
    what decides what happens rather than a label compared to another
    label. Three things the work turned up:
    - **The posture did not line up.** `claim take` refuses `--ledger`
      outright, while `offer list` and `situation` bound `--ledger`
      alone: in the only posture where a lane can claim it could
      neither poll nor orient. Both reads took the exclusive-or, and
      lane.SituationFlags follows with Posture beside Required.
    - **Exhaustion is the reserve, not the spending gate.**
      IsSpendingVerb holds only run.started, admitted from {supervise,
      operator}; the implementer holds claim, so that gate is the
      executor's and unreachable here. The drill asserts the capacity
      refusal by its own message so it cannot become a different one.
    - **The situation read withheld the acceptance anchor**, so a lane
      could not write the packet its own exit requires. The window now
      reports it.
- 9.2 escalation with packet, question and decision — backlog. An
  escalation answering a refusal carries that refusal's code and
  message, so the human is asked the boundary's own question (this
  card).
- 9.3 unattended maintenance, whose lint list carries the Phase 7
  exit's unsettled-run detection — backlog. 9.1's loop emitting
  observations from its own steps makes an expired classification
  better evidence, by removing forgotten bookkeeping as a cause of
  silence; it does NOT make silence proof, because the channel is
  lossy by declaration, so the reap stays a judgment needing
  corroboration and no_data carries no reap path at all. No heartbeat
  predicate is added either, because non-advancing observations are a
  legitimate long-running step (os-68ea0b2d).
- 9.4 small-team and fleet fixtures — backlog. Both run with no wake
  channel, and every refusal converges within one retry in one of
  three ways: an admitting act, a refreshed read showing the act is no
  longer owed, or an escalation carrying the refusal. The middle arm
  is the common case (a fleet-mode claim race means the loser
  re-orients), and what is forbidden is the fourth outcome: a blind
  retry or a silent loop (os-68ea0b2d).
- out-of-item: a reservation outlives its window — os-d6963652 —
  **review** (task PR against plan #175: admission gated all three
  budget verbs on `in_progress`, so a reservation whose claim window
  ended could never be settled or released while `BudgetViewAt` kept
  counting it against capacity; the gate moves to `budget.reserve`
  alone, the `budget.open` obligation drops the live-window
  restriction its reason no longer supports and is attributed
  standing-awarely to whoever can still close it, and the three budget
  affordance probes move to the conditional fence citation without
  which the dischargeability sweep could not go green at the very
  prefixes the fix is for; the shared walk now ends by suspending and
  revoking the lane holding an open reservation)

**Phase 9 exit (charter III.J as docs/next-build-plan.md's exit line
scopes it): met.** Met means the three criteria the plan's own exit
line names, the scoped-exit posture of the Phase 2 through 8 records,
each backed by a drill on `main` that a reader can find by name.
*Both modes run the full loop in CI*: `TestSmallTeamModeReachesDone`
and `TestFleetModeConvergesAndReachesDone`
(`cmd/seed/modes_e2e_test.go`), remote posture, no wake channel, every
refusal converging within one retry, with `TestBlindRetryDetector`
pinning the forbidden fourth outcome, and since #212 both modes
provisioned from the shipped role set alone rather than from
identities the fixture invented. *Injection corpus green*: the eight
corpus files under `internal/admit/testdata/injection/` with
`TestNoHostileTextWidensTheDispatcherSet`, plus the containment sweep
`TestIntentProseReachesNoDownstreamReadAutomatically`, which since
#211 plants its marker in a message payload and a message subject as
well as in an intent. *Maintenance runs unattended in the fixture*:
`TestMaintainHoldsNoPrivatePowers`,
`TestMaintainFilesDefectsAndRaisesNoEscalation` and
`TestMaintainCheckpointIsStartableByAFreshReader`
(`cmd/seed/maintain_cli_test.go`), one pass, no scheduler, no wake,
audited as an ordinary actor. Every numbered item has a merged PR:
1 across #188, #191 and #192 (review fixes os-378e44f3); 2 in #200;
3 in #205; 4 in #207; 5(a) in #171, 5(c) in #173 (post-merge defects
os-9b3f3ef3), 5(b) in #211; and the role-grant gap item 4 found, in
#212. Charter III.J as a whole is walked rather than the three the
exit line scopes, and two of its six rows are recorded UNMET rather
than glossed. Row 1 (role definitions for all six lanes as grants +
conventions, ordered fragments, validated): met by #188, with #212
making "six" enforced by name rather than counted. Row 2 (dispatcher
least standing capability; injection suite over intents, mirrors and
tool output): **UNMET, not claimed** — the row is conjunctive and the
mirror arm cannot be met, because `request.*`, the family mirror edits
and dashboard actions enter by, has zero transition rows and the build
plan named neither the family nor the word "mirror" anywhere; intents
and tool output are covered by #192 and least standing capability by
#188's allowlist. Row 3 (dispatcher re-triage rate and planner
unedited-approval rate tracked; planner receives the strongest tuples
by policy): **UNMET, not claimed** — nothing in the tree computes
either rate, and "strongest tuples" presupposes Phase 10's tuple
system. Row 4 (maintenance unattended, audited as an ordinary actor):
met by #205, with `TestMaintainHoldsNoPrivatePowers` as the
audit-posture pin. Row 5 (escalations carry packet + question +
minimal decision; waiting ones surface with age; resolution latency
tracked): met by #200 — `escalation.pending` carries the raising
`ts`, and `seed decision record` reports `resolved_after_seconds`,
derived from the chain and stored nowhere. Row 6 (small-team and fleet
modes run the full loop in CI): met by #207 with #212. Both unmet rows
are routed in the build plan's own text by this record, the move the
Phase 8 record made for III.I: row 3 to Phase 10 items 1 and 5 and
Phase 10's exit line, and row 2's mirror arm to Phase 13 item 4 and
Phase 13's exit line beside III.I. Full charter III.J conformance is
therefore claimable only once Phase 10 and Phase 13 both close. Two
things this phase learned are worth one sentence each. Its frontier
line was wrong once, claiming the phase complete while item 5(b) had
no implementation and two other paragraphs of this file said so; the
exit record re-derives the item list from the build plan rather than
reading the summary, which is how the gap was found. And three cards
in a row found hand-listed counts wrong — the exit-code table and the
refusal-site matrix (os-d03bde01), the capability-coverage gap
(os-d6a52784, where the derived drill found six ungranted verbs
against a card that named two capabilities) — so when a criterion says
"all N", N is a claim a drill derives, never a number it is told
(`memory/LEARNINGS.md`). This exit record is card os-e6cdb3d9's task
PR (an administrative card, not a Phase 9 item).

## Phase 10 — Qualification and evaluation (docs/next-build-plan.md Phase 10; deps: 9 ✓)

- 10.1 runtime tuples in grants, reported by adapters, drift as
  out-of-grant — os-8e53ffd9 — **done** (#216 against plan #215: the
  five-field tuple spelled once in `internal/tuple`; `actor.granted`
  citing an optional `tuple` at `seed/2` positions only, the keyring
  keeping the set per capability beside the string view of grants;
  `run.started` declaring a required `tuple` from `seed/2` and refusing
  one before it; the set rule at admission against the CLAIM HOLDER's
  `claim` grants (empty = unqualified, the bridge; non-empty = equal
  one member per field, else `out_of_grant` with a `Drift` detail
  naming the holder, the field and both values, exit 14 unchanged);
  `seed run start` deriving fence and reservation, filling harness and
  environment from the adapter and never inventing principal, model or
  tool policy; `Run.Tuple()` as the resolved configuration and
  `Provision` refusing `ErrTupleMismatch` with rollback when the
  adapter resolves anything else; `offer.published` scoping by `tuples`
  with `offer list` and the loop's poll filtering on it; the `seed/2`
  register entry and `version.Activated` as the named list the fold and
  keyring gate on; `next/spec/qualification.md`. Review fixes: a
  raw-pushed start's declaration re-judged in `RunStartValid`, the
  offer's `tuples` gate reading presence, a malformed scope folding to
  nothing.)
- 10.2 eval contracts through the production machinery; grants cite
  passing tuples; spot-checks; suspension on failure — os-03e47abb —
  **in review** (task PR against plan #217: `seed/3` in the register
  with `version.EvalApplies` gating the `eval` marker and the two verbs
  at `seed/3` exactly; `internal/eval` (definitions under
  `next/evals/<name>/`, the anchor read from the repository's last
  squash-merged commit touching the definition, `Check` proving the
  known verdict fixture-red-then-solution-green through the verifier's
  runner, `File`'s stable id, `Due`'s derivation of what the chain owes
  at a declared instant); the shipped `fix-the-check` eval;
  `intent.filed`'s optional `eval` marker folded to `SubjectState.Eval`
  and refused at an earlier tip; `actor.qualified` and
  `actor.disqualified` as the first non-operator actor rows, the
  keyring's admissible set and ever-cited mark, the qualification rule
  at admission (eval subject, authenticated verdict, the window's
  declaration on five fields, holder not signer, `ts` ordering, one
  verdict one consequence); the set rule exempt on eval subjects and
  the closed bridge for a once-cited empty set; mints gated on a
  recomputed receipt; tuple-wide disqualification; spot-checks aging
  from the record's own `ts` against `--as-of`; the eval terminal in
  obligations and reconciliation (D10); `seed eval list | check | file
  | status | act`, the acts performed by the lane that holds them;
  `next/spec/evals.md` and eight spec edits. Ranking waits on item 5's
  metrics; calibration is item 4.)
- 10.3 independence levels L2/L3 declared per tier and recorded in
  verdicts — not started; its table is built: **the tier vocabulary**
  verdicts — **in review** (os-99829835, task PR against plan #223:
  `seed/4` activates the ordered level vocabulary, `verdict.rendered`'s
  `independence` widening from the literal `L1` to `L1`/`L2`/`L3` and
  optionally carrying the verifier's declared `tuple`; the tier table's
  `independence` column (`trivial` and `standard` L1, `critical` L2,
  the strictest row L3) is read through `TierGates`; the level is
  computed, never asserted — L2 when the verifier's declaration differs
  from the window's admitted `run.started` in model provider or family
  (the optional `<provider>/<family>/<version>` convention) or in
  harness name, L3 when the acceptance is executable and gated with the
  receipt's reproduction at evidence grade, L1 otherwise — and the
  claimed level must equal it (a shape refusal naming both) and satisfy
  the tier (`level_short`, exit 17's refinement); `verdictBoundary`
  reapplies both along the merge chain, so a raw-pushed `critical`
  verdict at `L1` authenticates nothing; `seed verdict render` takes
  `--principal`, `--model` and `--tool-policy` (all three or none, usage
  on a pre-`seed/4` chain), fills the adapter's two fields, computes the
  level before drafting and refuses at usage when a required
  declaration is missing; `verdict check` and the contracts view render
  the level; reconcile classifies `independence_unverified` (the record
  half, and the receipt's reproduction, sealed subjects under `--key`,
  the maintenance loop reporting what its key cannot open); the
  contracts projection republishes as version 13; `EvalApplies`
  is the named list `{seed/3, seed/4}`; both modes drive a `critical`
  contract to `done`, small-team at L2 with a second model family and
  fleet at L3 on an executable gated spec); its table is built: **the
  tier vocabulary**
  (os-be12ac16, merged as #222 against plan #219) is **done** —
  `next/spec/tiers.md` declares `trivial`, `standard` and `critical`
  with a plan-required, sealed-checks-required and human-review column
  each, mirrored by `transition.Tier(name)` and pinned against the spec
  in both directions; `intent.filed` validates `tier` and `budget`
  against their tables at admission byte for byte
  (`VocabularyError`, the completeness family, naming the members); the
  three authority sites (the plan gate, the reconcile `unsealed` lint,
  `verdict render`'s `unsealed` refusal; the card said two, the tree
  had three) read the table through `TierGates`, an unknown tier taking
  the strictest row; the injection suite's characterization pin is
  replaced by the vocabulary drill, with mis-tiering (filing the valid
  `trivial`) kept pinned as tier provenance's residual. Item 3 added
  the `independence` column.
- 10.4 rubric verdicts (per-item, evidence-cited, uncertainty-marked);
  calibration harness with authority suspension on drift —
  os-2e34f66a — **in review** (task PR against plan #225: the
  `## Rubric` section of the acceptance spec read at the anchor as the
  commands are (`plan.Rubric`, a duplicate or empty id refusing
  `spec_unrunnable`); the scorecard artifact (items with score,
  cited evidence, explicit two-valued uncertainty, a bounded note)
  validated against the rubric, the receipt and the repository, its
  derivation-bearing half on `verdict.rendered` as `scorecard`
  (`seed/4`); `transition.DeriveScores` at render (`rubric_red`,
  `human_verdict` under exit 20), at admission (a verdict whose own
  items refute it never lands) and along the merge chain
  (`scoreBoundary`); `verdict.deferred` from a verdict key under L1,
  citing the receipt it computed and the items at high uncertainty,
  the whole verdict deferring on a human-review tier, creating
  `verdict.human` owed by the operator lane; the human as a key with
  a verdict grant beside operator standing, rendering over the
  deferral's receipt because sealed checks encrypt to keys without
  operator standing; the tier table's `human review` column
  consumed; `scorecard_unverified` at evidence grade; calibration
  definitions (`kind: calibration`, the gold committed to by digest
  and held outside the tree, `--gold` supplying it, the floor pinned
  to `evals.md` and raisable never lowerable), agreement at low
  uncertainty, the `verdict` qualification for the verifier's
  declared tuple, drift's tuple-wide disqualification and the
  dispatcher's idempotent defect filing, the set rule at render
  (`out_of_grant` under a drifted configuration until re-calibrated),
  spot-checks aging verdict qualifications; `seed verdict render
  --scorecard`, `seed verdict defer`, `seed eval status|act --gold`;
  drilled at the boundary (every D3 and D4 row, the chain, the
  qualification, the set rule, `seed/3` refusing each), in the fold,
  the obligation, reconcile and eval packages, at the terminal
  (scorecards, the deferral and the human's render, the situation
  read's debt, lost scorecards classifying, the calibration lifecycle
  with drift and re-qualification) and end to end in small-team mode
  (a rubric contract and a deferred one reaching done); no shipped
  calibration definition; the verifier lane's summary and fragment;
  `verdicts.md`, `evals.md`, `qualification.md`, `obligations.md`,
  `reconciliation.md`, `acceptance.md`, `envelope.md`, `tiers.md`,
  `actors.md`, `protocol.md`, `lanes.md`)
- 10.5 trajectory-prefix regression harness; dispatcher re-triage rate
  and planner unedited-approval rate (III.J row 3's metrics half,
  routed here by the Phase 9 exit record) — not started

- out of item: the client's private git dir and the verifier's clone
  arm auto-gc in production — os-711b3028 — **in review** (task PR
  against plan #224: `gitref.NewClient` writes `gc.auto=0`,
  `gc.autoDetach=false` and `receive.autoGC=false` into its git dir on
  every construction, so a state dir an older build created is
  hardened the first time a newer build opens it; `verdict.NewWorkspace`
  writes the same three keys into its clone between the clone and the
  checkout; the drills assert the repository-local property with `git
  config --local`, which the test binary's global config cannot
  satisfy, and an older-build drill proves the write happens on the
  no-init path and that a second open changes nothing).
## Phase 11 — Curation and the flywheel (docs/next-build-plan.md Phase 11; deps: 9 ✓)

- 11.1 staged curation stores (observations → hypotheses → validated
  lessons → policy) with grant-gated boundaries; workers append
  candidates only — os-f30ee0d3 — **merged** (#234 against plan
  #226: `internal/curation` with the three facts, `curation.deadend.recorded`
  on the contract inside the holder's window (the packet finding's
  shape plus the charter's failure condition and environment),
  `curation.hypothesis.proposed` on `h-<12 hex>` derived from the
  claim, citing at least two admitted observations on two distinct
  non-failed contracts re-judged from the record, refusing a
  re-proposal as a duplicate, and `curation.lesson.promoted` on the
  hypothesis subject citing an admitted proposal, a promotion citing
  none folding `unbound`; the `curate` capability, the fifth
  no-fallback row, disjoint from `claim` and `operator` at the grant in
  both directions, a root included, held by the curator manifest and
  nothing else; the raise row's sixth capability; the `knowledge`
  projection and the report's `knowledge` section, present only when a
  curation fact stands; `seed knowledge deadend | propose | promote |
  show`; the lessons store `next/knowledge/lessons/` with its
  frontmatter contract and lint; the curator's reachable set derived
  from the boundary and pinned: the proposal, the raise and the
  standing-only relay; `next/spec/curation.md` and six spec edits).
- 11.2 promotion gate (≥2-trajectory support, applies-when,
  provenance, last-validated; adversarial evaluation; contested state;
  lessons at claim time) — os-96850e5a — **merged** (#235 against
  plan #228: `applies_when` is a predicate over
  record-derivable fields (`routing`, which the fold now reads, `tier`,
  `paths`), one implementation for the boundary, the lint and the
  delivery; the support rule gains the actor arm, two distinct holders
  where the family the predicate selects has two, waived and recorded
  as `single_actor_family` where it has one; the hypothesis id derives
  from the claim and its exceptions, so an added exception is the road
  out of a contest; `curation.hypothesis.contested` from `curate`
  alone cites held-out observations on selected contracts outside the
  support set, moves the fold to `contested`, closes the promotion and
  removes the lesson from every delivery while keeping the file and
  the facts; `lesson.promoted` carries `carrier`, `adversarial`,
  `last_validated`, `expires` and `digest`, and its ledger half
  requires an uncontested admitted hypothesis whose support still
  passes and an authenticated pass on an eval filed after it with a
  marker bound to it and to this lesson anchor, for every carrier; the
  bound marker (`seed eval file --for-lesson --carrier`) and `seed
  verdict render`'s `carrier_absent`; `seed knowledge lint`, the
  gate's file half against the fact, the hypothesis and the
  repository; every refusal a `GateError` at a gate `curation.Gates()`
  registers; delivery from one derivation, verified against the
  repository (anchor ancestry, digest) on `claim take --repo`, in the
  provisioned `.seed-run/lessons.json` and in `seed situation --repo`,
  the unverified reported as such; `lesson_unverified` at evidence
  grade; the projection's `contested` stage and per-lesson
  `surfaces`).
- 11.3 poisoning drill — os-e2f1ad23 — **in review** (task PR #236
  against plan #229: `internal/admit/testdata/poisoning/`
  declares forty-seven poisons over the thirty-two registered gates and
  five residuals; `poisoning_test.go` derives coverage from
  `curation.Gates()` pinned to the spec table both ways, scripts every
  poison to an attempt to promote and asserts it fails at both ends
  (the refusal at its gate, no admitted promotion, no lesson on a
  selected claim), pins every residual by a characterization drill,
  and refuses an empty corpus, an empty table, a declared poison with
  no script and a script with no declaration; the CLI arm runs
  `worker-proposes` and `smuggled-role-lesson` through `seed knowledge`
  in the small-team fixture; `curation.md` gains "The poisoning
  drill" and `lanes.md` records III.K row 4 as met)
- 11.4 expiry, retirement (evidence kept), rollback by revert; dead
  ends un-retired on environment change; staleness flags, dedup and
  structure lint (III.K rows 6, 7 and 9) — os-0d537fbd — **in review**
  (task PR against plan #230, stacked on item 3's: expiry derived at a
  declared instant (`curation.Expired`, at-or-past) and never a fact;
  the fold keeps the latest admitted promotion per lesson path, keyed
  by path, and a re-promotion refuses at `promotion.revalidation`
  unless its `last_validated` moved forward (`LatestPromotionBefore`,
  one forward pass, never a refold); `curation.lesson.retired` from
  `observer` or `operator` with `regression` (the revert's `pr`),
  `superseded` (`superseded_by`, a later admitted promotion) or
  `expired` (neither), each field required by one reason and refused
  by the others, the cited promotion the latest of its path and
  unretired, the fold moving the path to `retired` while the file, the
  hypothesis and the observations stay, a new promotion clearing it;
  `curation.deadend.retired` and `unretired` from `curate` alone on an
  environment that differs from the one the previous act named, a
  retirement needing no standing one and an un-retirement needing one,
  applicability by string equality with the run's declared environment
  (`DeadEndFact.Applies`), the fold flagging and never deleting, the
  held-out listing excluding retired dead ends; the surfacing set's
  three new arms (latest per path, not retired, not expired at the
  read's instant: `claim take --now`, `seed situation --now`, the wall
  clock otherwise, admission reading no clock); the `knowledge`
  projection at the declared inputs' `as_of` (version 3, input
  consumption declared) flagging `stale` and listing the retirements,
  saying so when no instant is declared, the report at version 12
  counting both; `lint.structure` and `lint.duplicate` under `seed
  knowledge lint` and `make check`; the `lesson_stale` maintenance
  lint (`maintain run --stale-after`), the finding's subject
  `<path>@<promotion position>` so one stale cycle files once and a
  re-promotion that expires files new work; `seed knowledge retire`,
  `deadend retire | unretire`, `validate --environment`, `show --now`;
  eleven new gates, each with a poison in item 3's corpus; drilled at
  the boundary (every D1 to D3 row), in the fold, the projection, the
  reconcile and maintenance packages, at the terminal, and end to end
  in the modes fixture (the revert observed, the claim after carrying
  nothing, the revalidation surfacing again, the dead end retired and
  un-retired with the listing changing); `curation.md` gains "Expiry,
  retirement and applicability" and "Bloat" with III.K rows 6, 7 and 9
  in its mapping, `lanes.md` records them met, `protocol.md`,
  `actors.md`, `projections.md`, `maintenance.md`,
  `reconciliation.md` and `loop-verbs.md` follow, and the build plan
  names row 9 at Phase 11 item 4)
- 11.5 flywheel v0 — os-9075c308 — plan in review (#231)

## Frontier

Phases 0 through 5 are done and closed: every Phase 5 plan (#113–#116,
#120, #121), every implementation (5.1 #122 through 5.6 #131), the
three post-merge follow-ups (#127, #128, #129), and the engine
v0.15.1 receipt-runner pin (#130) are merged, and the Phase 5 exit
record above is card os-6e37b10e's task PR (#134, merged; card
closed). Phase 6 is done and closed: every plan (#133, #136, #138,
#140), every implementation (6.1 #135, 6.2 #137, 6.3 #139, 6.4
#141), and the exit record above (card os-600be59e's task PR) are
merged with every card closed. Phase 7 is done and closed: every
plan (#144/#146, #147/#148, #150, #153), every implementation (7.1
#145, 7.2 #149, 7.3 #151 with follow-up #152, 7.4 #154), and the
exit record above (card os-c9e24032's task PR) are merged with
every card closed. Phase 8 is done and closed: every plan (#158,
#162, #164), every implementation (8.1 #160, 8.2 #163, 8.3 #165),
the out-of-item ledger writeHead race fix (#161 against plan #159),
and the exit record above (card os-ef715d17's task PR) are merged
with every card closed. Phase 9 is done and closed: every plan (#170,
#172, #187, #189, #190, #197, #203, #204, #206, #209, #210), every
implementation (5(a) #171, 5(c) #173, 1a #188, 1c #191, 1b #192, 2
#200, 3 #205, 4 #207, 5(b) #211, the role-grant gap #212), the
out-of-item fixes (#175's task PR, os-9b3f3ef3, os-378e44f3, budget
exhaustion's exit code #208), and the exit record above (card
os-e6cdb3d9's task PR) are merged with every card closed.
Phase 10 is under way: item 1 (os-8e53ffd9) merged (#216 against plan
#215); item 2 (os-03e47abb) merged (#221 against plan #217), so
`actor.qualified` is defined; the tier vocabulary (os-be12ac16) merged
(#222 against plan #219); item 3 (os-99829835) is implemented against
plan #223 and in review, so a verdict's level is computed from the
records and held to its tier. The plans for item 4 (#225), item 5
(#227) and Phase 11 items 1 (#226) and 2 (#228) are merged; the plans
for Phase 11 items 3, 4 and 5 (#229, #230, #231) are in review. Item 3
merged (#233); item 4 (os-2e34f66a, rubric verdicts and the
calibration harness) is implemented against plan #225 and in review.
The frontier is item 5 (the trajectory-prefix regression harness).
#215); item 2 (os-03e47abb) is implemented against plan #217 and in
review, and `actor.qualified` is now defined. The frontier is item 2's
merge, then item 3 (independence levels per tier), whose tier
vocabulary is carded as os-be12ac16 with its plan in review (#219).
Phase 11 has opened in parallel, depending on Phase 9 alone: item 1
(os-f30ee0d3, the staged curation stores) is merged (#234 against
plan #226), so the curator holds its proposal grant and the curation
verbs exist; item 2 (os-96850e5a, the promotion gate, the
contested state and delivery at claim time) is implemented against
plan #228 and merged (#235); item 3 (os-e2f1ad23, the poisoning
drill) is implemented against plan #229 and in review (#236); item 4
(os-0d537fbd, expiry, retirement, dead-end applicability and bloat)
is implemented against plan #230 on top of item 3 and in review; item
5 (os-9075c308, the flywheel) is planned (#231) and next.
Phase 9 was under way: item 5's derivation and read merged (#171) and
its loop verbs merged (#173), so the lane-facing surface is whole in
shape — `seed situation` says what is true and what is owed, and the
loop verbs are how a lane acts on it without hand-assembling protocol
arguments. Two things qualify that. Part (b) is **partially met**: it
carries no unread messages, which item 5's text now names as the
remainder. And #173's post-merge review found three defects, carded
as **os-9b3f3ef3** and fixed by this card's own task PR — derived
arguments not re-derived across the optimistic retry (a rival
reservation landing mid-flight was silently closed, the exact choice
`soleOpenReservation` exists to refuse), a remote refusal stamped from
the stale view, and a JSON null packet panicking the CLI — which also
records in `next/spec/loop-verbs.md` that a derived value is
re-examined against the refreshed tip and refused rather than
replaced.
**Phase 9 item 2 is done**: escalation with packet, question and
minimal decision, planned in #197 and implemented by this card's task
PR. `escalation.raised` (`ready`, `review` -> `blocked`) and
`decision.recorded` (`blocked` -> `ready`) are the two new rows; from
`in_progress` an escalation rides `claim.parked`, because what
`lifecycle.md` pins is the set of verbs that may LEAVE that state, so
the four deliberate exits stay exactly four and their existing pinning
test is now the enforcement of that rule. `next/spec/escalation.md`
carries the design. Raising is broad because raising grants nothing;
answering is the fourth no-fallback operator row. While a question
stands, `contract.unblocked` refuses and `contract.cancelled` must cite
the escalation it answers. Waiting escalations surface as
`escalation.pending` with the raising event's `ts`, because age is
elapsed time and a position difference is event count wearing a clock's
clothes.

**Next action: Phase 10 item 1** — runtime tuples in enrollment and
grants, adapters reporting the provisioned tuple, drift as
out-of-grant. The derivation, stated rather than read off a summary:
Phase 9's items 1, 2, 3, 4, 5(a), 5(b), 5(c) and the role-grant gap
each have a merged PR (#188/#191/#192, #200, #205, #207, #171, #211,
#173, #212), the exit record above walks III.J and routes its two
unmet rows, so nothing in Phase 9 remains to claim. A correction
worth keeping, because it was made here and then unmade: an earlier revision of this line claimed the phase was complete
once items 3 and 4 merged. It was not. **Item 5(b) had no
implementation**, which the build plan and this file both said in other
paragraphs while the frontier line said otherwise — a frontier is only
useful if it is the same claim as the rest of the document. Assembling
the exit record is what surfaced it, one item too late.

Item 4 was os-6a08b166's task
PR, and with item 3 (#205) every numbered item in the phase had an
implementation except 5(b). Both modes run the full loop to `done` on the remote
posture, wakeless, with every convergence arm exercised. Three things
this card settled are worth carrying forward.

- **The terminal surface was missing, not merely local-only.**
  `merge.requested` and `merge.observed` had no CLI verb at all,
  existing only through `ledger append`, which runs no rules. All three
  terminal verbs (with `verdict render --remote`) now reach both
  postures, so the chain's last steps are drivable by a lane.
- **The mode is purely the identity plan.** Both modes run remote:
  fleet needs it for contention, and small-team could never have run
  locally, because `claim take` is refused off the remote and a claim
  is the loop's first act. Neither clause of III.J mentions transport.
- **III.J's closing row is met, and the gap it recorded is CLOSED**
  (os-d6a52784, plan #210). No shipped manifest granted `supervise`,
  `observer` or — found in review, the worst of the three since it has
  no operator fallback — `sealer`. The charter's six lanes (§II.11) are
  a closed enumeration, and both missing parts already exist in the
  charter outside it: the supervisor is §II.9 and the observer is §8's
  governed observer. So they ship as **roles**, manifests of a required
  `kind` beside the six, validated by the same code, refused by name if
  they claim to be a seventh lane; `sealer` rides the verifier, whose
  keyring the sealed bodies are already encrypted to. The mode fixtures
  now provision both roles **from their manifests** rather than staging
  them as identities the test invented, which is what makes the closure
  real rather than reported. The drill whose absence let this reach
  Phase 9 reads the capability table's own source and, run against the
  pre-fix tree, names **six** ungranted verbs across the three
  capabilities — three more than the card had found.

Item 1 is COMPLETE: 1a
merged (#188), 1c merged (#191), 1b merged (#192), with its four review
findings landing as os-378e44f3. III.J's first row is met; its second is
**two-thirds met** and the spec says so — intents and tool output are
covered, mirrors are not, because `request.*` has zero rows in the
transition table.

Item 1's four ergonomic obligations are now all real rather than
declared: the one position-stamped read each manifest names (and, since
1c, names in the posture the lane can actually claim in), the loop verbs
as the loop's acts (enforced by 1c's act gate, which refuses anything
`acts_through` does not declare), liveness riding the loop's own steps
(emitted as a side-effect of a declared act that succeeded, keyed to the
lane's actor and fence), and the one-inbox doctrine (the loop acts on
its read, never on a wake).

Item 2 merged (#200): escalation with packet, question and decision,
carrying the refusal it answers.

Item 3 is THIS card's task PR — `seed maintain run`, one unattended
pass (reap, lint, file, rebuild, checkpoint) with no scheduler and no
wake channel. Three things about it are worth carrying forward rather
than rediscovering:

- **The reap answers an unanswered request, never a timeout.**
  `observations.md` says Seed holds no lease, so there is no expiry to
  elapse, and the channel is declared lossy — a dropped stream and dead
  work look identical from outside. A reap therefore needs the
  `expired`/`wedged` classification BESIDE an admitted `run.interrupted`
  on the active fence (or a `wedge.declared`), which is exactly the
  force path `executors.md` named this loop as the consumer of.
  `no_data` carries no reap path at all, however old the claim.
- **The pass consumes the COMPLETE reconcile result.** The
  evidence-grade checks — attested heads, rewritten targets, receipt
  retrievability — moved out of `cmd/seed` into `internal/reconcile`,
  because a pass built on the record-derived half alone reports clean
  over a rewritten target: green, and omitting the very divergence the
  charter asks this loop to reconcile.
- **Checkpoints persist a snapshot a fresh reader starts from**, its
  hash and location in a versioned payload the boundary validates,
  the materialization written to the artifact store first. Shape at the
  door, contents at the read: admission reads the ledger alone, so
  retrievability is the reader's check, drilled by a round trip.

Item 4 (small-team and fleet fixtures, run wakeless and asserting
one-retry convergence) is planned in #204, which also records what it
found: `verdict render` defines no `--remote` flag, so a fleet's
verifier lane cannot act against the shared ledger. That is carded as
**os-a00b6950** rather than widened into a fixtures card, and III.J's
closing row is therefore met for small-team mode and met-to-submission
for fleet until it lands.
The III.I remainder rides the phases it was routed to: Phase 11 item
2 carries claim-time lesson surfacing, and Phase 13 carries the
machine-protocol surface and platform parity, with III.I on its exit
line — full III.I conformance closes only when both do.
**The destination is promotion (spin-out)**, now defined:
`docs/next-build-plan.md` §5 (merged #169, card os-768361cc) makes
it two steps — Seed coordinating this repository's own development,
then becoming what new users clone — with seven criteria, Phases 0
through 12 required and Phase 13 alone following, and **neither
cutover autonomously decidable**: spin-out IS the entry-point
switch, so both are escalations. That section is the authority; no
promotion criteria are restated here.
If an open task PR is red or carries review feedback, drive it green
first — nothing merges out of order.
