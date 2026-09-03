# Plan: next — the Phase 12 exit record (os-9e9ae30c)

Phase 12 (hardening, distribution, migration) is complete once #272
(item 6) merges, and this card is its exit record: the paragraph in
`next/docs/progress.md` that the Phase 5 through 11 records wrote,
confirming the charter pillars the build plan's Phase 12 exit line
names — III.B (the service posture), III.O (the compromised-actor
drill in CI), III.P (distribution and migration) — and the pillars the
Phase 10 record put on this line as owned by no other exit (III.C,
III.L, III.M, III.Q), row by row, by citation to drills on `main`.

**Why plan-first for a docs-only change.** The Phase 10 and 11 records
were planned first (#242, plan #245's sibling; os-efb2a099) because the
exit record is the document the next phase orients from, and a record
that states a phase's final shape deserves a reviewer before it becomes
the thing everyone reads instead of the items.

## What the tree actually shows

Every numbered item has an implementation on `main` or in review,
cited here so the record cites rather than asserts:

| item | what | landed |
|---|---|---|
| 1 | the compromised-actor drill and the release gate: `internal/redteam`, the code-ref rules, `make check` as the gate | #250 (plan #241); the reap arm #267 (os-32d06c65) |
| 2 | `seed-admit serve`, the proposal protocol, `internal/refusal`, the protections reconciler with the GitHub and snapshot adapters, `seed protections plan\|apply`, the doctor's probe | #252 (plan #244) |
| 3 | `checkpoints.trust`, `ledger.WithTrustedPrefix`, `project.StartFromCheckpoint` with the cross-check and `basis.json`, `internal/history`, `internal/perfgate` and `seed perf run` with budgets carrying provenance | #253 (plan #246) |
| 4 | the preseed blocks, `seed init --preseed` idempotent with `preseed_drift`, `seed preseed check` in CI, the agent ceiling, the path floor, `by_kind`, the capability audit | #254 (plan #247) |
| 5 | `seed import --from-open-seed`, anchors first (exit 29), the genesis import at `seed/5` with `system.imported`, the lossless manifest, the real fixture drilled in CI | #255 (plan #248) |
| 6 | docs generation with drift-failing CI, the operator handbook, simulation mode (`seed simulate`, credential-free, both postures), the trajectory decider seam | #272 (plan #263, os-16e55c11), in review |
| gaps | `ledger show` stamping the position its `chain_invalid` was computed at (#265, os-37fcf7c6); the cache carrying the event's `ts` (#266, os-74ce2261) | merged |

**The exit line's criteria**, each backed by a named drill on `main`:

- *III.B, the service posture*: the stateless validator as the only
  writer, the hook (#94, #99) and the service (#252:
  `TestServiceAdmitsAndRefusesLikeTheHook`,
  `TestOneDerivationAcrossServiceBoundaryAndHook`) over one rule set
  (`admit.Default()`); kill-and-replace drilled for both
  (`TestDrillKillAndReplace`, the service's rebuild drill); the three
  postures declared with the cooperative posture's consequence printed
  (`posture.Consequence`, `ErrUndeclared`); actor credentials unable to
  write the ledger ref (#250's direct-push arm, #252's sole-writer
  ruleset); the compromised-actor drill in CI (#250:
  `TestCeilingHoldsAtThePush`, `TestOneDerivationLedgerAgrees`,
  `TestCoverageBothWays`, `TestResidualsArePinned`, `TestTablesValidate`).
- *III.O, the drill in CI*: the release gate is `make check`
  (`plans/os-465e356e.md` D8; `next/spec/redteam.md`), which runs the
  drill on every push to `main` and every PR; the standing drills row is
  walked by name in the record (projection rebuild, checkpoint
  verification, packet-resume with dead-end assertions, claim race
  storms, halt including the raw-git bypass, key revocation with
  keyring rotation, verdict/merge divergence, the data-classification
  hostile corpus, budget reservation races, curator poisoning), each a
  test on `main` under `make check`; trajectory-prefix regression (#239)
  and simulation mode (#272) close row 5.
- *III.P, distribution and migration*: the repository's own
  distribution that Seed ships inside (`scripts/seed`'s pinned,
  checksum-verified engine never committed, `seed engine upgrade` with
  rollback and protocol preflight, `seed template upgrade`'s three-way
  merge, the air-gap vendor path, `checksums.txt` and provenance on
  releases); the hook and the service both stateless and rebuildable
  (#252); the preseed in one idempotent CI-verified file (#254:
  `TestInitPreseedIsIdempotentAndDriftRefuses`, `make check-next`'s
  `preseed check`); the predecessor import, two commands, drilled
  against a real fixture (#255: `TestRealFixtureImports`,
  `TestAnchorRefusalsPrecedeEveryWrite`, `TestNonEmptyLedgerRefuses`,
  `TestUnmappedVerbRefuses`, `TestImportCommandEnvelopes`); install as
  one command with no telemetry and no account (the absence of any
  network call outside git and the forge adapters, asserted by grep in
  the record). One residual named: Seed's own binary is built from
  source and is not yet a released artifact — §5 step 2's cutover,
  routed to promotion, not claimed.

## The pillars walked

The Phase 8 through 11 records' posture: the exit line's criteria
decide the exit; every row of the named pillars is walked regardless,
and an unmet row is recorded and **routed**, never glossed.

| pillar, row | status | by |
|---|---|---|
| III.B 1–5 | **met** | as above; row 6 (sharding MAY) not claimed |
| III.O 1–5 | **met** | rows 1–2 at Phase 10 (#221, #238), row 3 #250, row 4 the standing drills by name, row 5 #239 and #272 |
| III.P 1–5 | **met** | as above, with the binary-release residual routed to promotion |
| III.C 1, 2, 3, 5 | **met** | Phase 4's observation channels and classification; Phase 7's liveness, expiry and wedging |
| III.C 4 (the contention benchmark at target scale, tracked in CI) | **PARTIAL, routed** | #253's storm runs 24 writers through the hook in CI against a budget with provenance; hundreds of concurrent actors are not demonstrated per PR; routed to a scheduled run in the backlog (build plan §3), named in the record |
| III.L 1, 2, 3, 5, 6 | **met** | #254 (tiers, the protected surface, the audit), #250 (the corpus on the release gate), #252 (protections, CI identities), Phase 5's plan lint |
| III.L 4 (per-verb policy on the machine-protocol surface with attributable approvals) | **UNMET, routed** | Phase 13 item 6 (os-b55e5647, #273 in review): the machine surface exists there with the CLI's own verbs; the record says whether the per-verb policy row is met by it or stays routed |
| III.M 1 (the DAG engine) | **met, two residuals named** | the v1 engine Seed's flywheel drafts into (#240); vault-indirect secrets and which engine new users clone are §5 step 2's |
| III.Q 1–6 | **met** | `make check` green on `main`; `covergate` at 90% with the smoke and conformance suites; #272's docs governance; Appendix C and `decisions.md`; the authority order; MIT, no CLA, no open-core |
| III.Q 7 (self-hosting total once bootstrapped) | **PARTIAL, routed** | Seed coordinates this repository's `next:` cards through v1's queue today; total self-hosting is §5's shadow run and promotion |

**The routings close and open.** III.J row 2 (the mirror arm), routed
by the Phase 9 record to Phase 13 item 4, is met by #270 (the corpus
fired at `request.filed`; `lanes.md` says so). III.I's machine surface
and platform parity, routed by the Phase 8 record to Phase 13 item 6,
is #273 in review at writing time; the record states its status at
merge. III.C row 4 and III.Q row 7 stay routed as the table says.

## Design decisions (binding for this task)

- **D1 — the record follows the Phase 10 and 11 shape exactly.** One
  bold opening "Phase 12 exit (charter III.B, III.O and III.P as
  docs/next-build-plan.md's exit line scopes it, with III.C, III.L,
  III.M and III.Q walked as the Phase 10 record routed them): met"
  naming the exit line's criteria and the drills that back them; then
  the pillars walked row by row with "met by" citations, the partial
  and unmet rows in those letters with their routing; then the
  routings closed; then the closing sentence naming this card as the
  record's task PR.

- **D2 — the expectation is not the method.** The tables above were
  walked against the specs' conformance mappings and the drills by
  name; the record re-reads each drill and, where a row is overstated,
  records it UNMET or PARTIAL and routes it in
  `docs/next-build-plan.md`'s own text. No fraction in the status
  column.

- **D3 — the frontier is re-derived, not edited.** Items 1 through 6
  each with a merged PR; the two gap cards merged; therefore Phase 13
  is open and its items stand as their cards do at writing time (13.1
  merged in #269, 13.4 in #270, 13.5 landing in #279, 13.6 in review in
  #273, 13.2/13.3/13.7 planned in #274/#275/#276), and the record says
  so. The record inserts after the Phase 11 record.

- **D4 — two sessions, one record.** The other implementing session
  wrote item 6 and brings the walk of III.I and III.J row 2; this
  session brings the rest. The record is one paragraph with one voice;
  the task PR names both sessions in its body.

- **D5 — what the phase learned is one sentence pointing at
  `memory/LEARNINGS.md`.** Stacked task PRs cost a merge-forward and a
  fresh receipt at every parent merge, and a child merged into a stale
  base never reaches `main`; the record names the rule (delete or
  retarget after each parent merges) and not the cases.

## Steps

1. Write the record into `next/docs/progress.md` after the Phase 11
   record; re-derive the Frontier section.
2. Route III.C row 4 in `docs/next-build-plan.md`'s backlog text (one
   line) and confirm III.L row 4's landing under Phase 13 item 6.
3. Receipt; evidence; review.

## File Scope

- `next/docs/progress.md`
- `docs/next-build-plan.md` (the one backlog line, if the routing needs it)
- `receipts/os-9e9ae30c.json`

Nothing else. NOT `next/spec/**`, NOT `next/internal/**`, NOT `.seed/**`.

## Acceptance Criteria

1. The record names every exit-line criterion with a drill on `main`
   by test name and file, and every pillar row with its status and
   citation; PARTIAL and UNMET rows carry a routing.
2. The Frontier is re-derived from the items and cards, not edited.
3. `make check` green (docs-only; the coverage gate unchanged); no
   model identifiers in any committed artifact.

## Validation Commands

- Retention: `make check` (exit checked separately from any pipe)
- Docs: `cd next && go test ./internal/loopverb/ -count=1` (the spec-parity drills that read the docs and spec directories)

## Expected diff shape

Modified: `next/docs/progress.md` (the record and the Frontier),
possibly one line of `docs/next-build-plan.md`, the receipt. Nothing
else.
