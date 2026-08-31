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
  is advisory) — os-c61c3392 — backlog
- 7.2 budgets (budget.reserve / settle / release; admission
  decrements reservations; the reservation race drill; per-adapter
  risk limits) — os-cecac5de — backlog
- 7.3 executor adapter interface + the local worktree adapter
  (provision / wake / meter / report-tuple; metering to the
  observation stream; run.settled aggregate) — os-1dad487d — backlog
- 7.4 graceful preemption (safe-point park with packet; force reap
  packet) — os-0f718b4e — backlog

## Frontier

Phases 0 through 5 are done and closed: every Phase 5 plan (#113–#116,
#120, #121), every implementation (5.1 #122 through 5.6 #131), the
three post-merge follow-ups (#127, #128, #129), and the engine
v0.15.1 receipt-runner pin (#130) are merged, and the Phase 5 exit
record above is card os-6e37b10e's task PR (#134, merged; card
closed). Phase 6 is done and closed: every plan (#133, #136, #138,
#140), every implementation (6.1 #135, 6.2 #137, 6.3 #139, 6.4
#141), and the exit record above (card os-600be59e's task PR) are
merged with every card closed. **Next action: 7.1's plan**
(os-c61c3392, offers: offer.published eligibility-scoped and
expiring, workers pull and claim, and the wakeless poll-only drill
proving wake is advisory; plan-first on seed/os-c61c3392-plan). The
Phase 7 exit subset is charter III.H for the implemented adapter:
the poll-only run, the reservation race drill, and the
disposability drill (randomized kill after sync; complete
elsewhere).
If an open task PR is red or carries review feedback, drive it green
first — nothing merges out of order.
