# Plan: next build-plan amendment — agent-ergonomics obligations for Phase 9 lanes (os-68ea0b2d)

Amends [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 9
so the lane contracts carry the ergonomic obligations as **conformance
rather than convention**. The precedent is #157 and #169: a routing
binds only when the phasing authority's own text says it, so an
obligation asserted in a frontier file, a spec aside, or a review
comment binds nothing. This task changes the build plan itself.

## Why this is not polish

Phase 9 defines six lanes and, in item 5, the surface they act
through. Parts (a) and (b) of item 5 landed in #171 (obligations
derived, `seed situation` answering *what is true for me now* with a
`--since` change report); part (c) landed in #173 (the loop verbs:
derive every derivable argument, refuse before signing, render the
boundary's error beside the caller's affordances).

What item 1 does **not** yet say is that the lanes must *use* any of
it. A worker lane written against the raw `ledger append` seam, waking
on an event stream it trusts, emitting heartbeats it remembers to
send, and retrying blind on refusals would satisfy every word of item
1 as written — and would fail promotion criterion 1 (loop-completeness)
on inspection rather than on a check. The four obligations below are
each a **new consumer of an existing table**, and each is stated so a
fixture can fail on it.

## Design decisions (binding for this task)

- **D1 — one position-stamped read on wake, named in the fragment.**
  Every lane's role fragment names the single read it orients from
  (`seed situation --key <its key> [--since <last position>]`) and the
  position it carries forward to the next wake. Not "a lane may use
  situation": the fragment declares it, and fragment validation (III.J
  row 1, "resolved and checked by validation") checks that the
  declaration is present and names a real surface. This is promotion
  criterion 1's "orienting from one position-stamped read rather than
  hand-assembling ledger payloads and hand-computing fences" made a
  property of the role file rather than of the agent writing it.
- **D2 — the loop acts through the loop verbs, never the raw seam.**
  Item 1's worker-loop text names the acts as `claim take|release|park`,
  `submission make`, `budget reserve|settle|release` — the surface item
  5 part (c) landed. The reason is not tidiness: the raw seam consults
  the admission boundary not at all, so a loop built on it cannot
  satisfy D3 below, and item 5(c) would ship a surface with no
  consumer, leaving loop-completeness unverifiable.
- **D3 — one-retry convergence, or escalate carrying the refusal.**
  A lane that meets a refusal must, on its next act, either admit or
  escalate with the refusal's `code` and `message` in the escalation
  packet. No blind retry, no silent loop, no bare "failed". This is
  checkable in item 4's fixtures and is the lane-level counterpart of
  Phase 8's affordance-drift class: there, a listed verb refused at the
  same position is a bug; here, a refusal a correct lane cannot act on
  is a bug. Success envelopes stay terse; the pedagogy already lives in
  the refusal, and this obligation is about *consuming* it.
- **D4 — liveness rides the work, and Seed has no leases.** The card
  asked that "lease renewal ride every holder-signed verb rather than
  being a separately remembered act". Seed has no lease: a claim is
  held until a deliberate exit or a reap, and liveness is classified
  from the active claim's **observation stream** against `expiry_after`
  and `wedge_after` ([`next/spec/observations.md`](../next/spec/observations.md)).
  The transferable obligation is therefore sharper than the v1
  phrasing: the observations that keep a claim from classifying
  `expired` are emitted **by the loop's own steps** (the metered run,
  the milestone at a real step, the sync), never as a bare heartbeat.
  A working lane is then a live lane by construction, and an `expired`
  classification is true information the reaper may act on rather than
  an artifact of forgotten bookkeeping. A heartbeat verb that reports
  only "still here" is forbidden by this item, not merely undesirable.
- **D5 — the one-inbox doctrine.** Push channels **wake**;
  position-stamped reads **convince**. No lane treats a wake, an
  event, or a message as a fact about the world: it is a hint to read.
  The system half is already proven (#145: the wakeless poll-only run
  through the whole loop, "wake is advisory transport, its total
  failure costs only latency"); the lane half is this obligation, and
  item 4's fixtures assert it by running **with no wake channel at
  all**, exactly as #145 did.
- **D6 — scope guard, carried from the card verbatim.** No new
  orchestration layer, no chatty success envelopes, no statechart
  creep, and no new sources of truth. Every obligation above consumes
  a table, projection or surface that already exists; none adds a
  verb, a state, or a place where truth can live.
- **D7 — no phase, dependency or exit-line changes.** The amendment is
  sentence-level extension inside Phase 9 items 1, 3 and 4. The exit
  line already carries III.J and promotion's lanes-operable and
  loop-completeness gates; these obligations tell it what to check,
  they do not add a gate.

## Steps

1. **Extend Phase 9 item 1's role-definition text** with D1, D2, D4
   and D5, each as one binding sentence in the item's own words: the
   fragment names its one position-stamped resume read and the
   position it carries forward (D1); the worker loop's acts are the
   item 5(c) loop verbs, never the raw append seam (D2); the loop's
   own steps emit the observations liveness is classified from, with
   no bare-heartbeat verb (D4); and every lane treats wakes and
   messages as hints to read rather than facts (D5).
2. **Extend Phase 9 item 2's escalation text** with D3's second half:
   an escalation raised in answer to a refusal carries that refusal's
   `code` and `message` in its packet, so the question a human is asked
   is the boundary's own account rather than a lane's paraphrase.
3. **Extend Phase 9 item 3's lint list** with D4's enforcement half:
   the maintenance lane's reaper may act on an `expired` classification
   **because** the loop's steps emit the observations, so a lint flags
   any claim window whose only observations are non-advancing — the
   heartbeat shape the item forbids — rather than reaping it silently.
4. **Extend Phase 9 item 4's fixture text** with D3 and D5's assertions:
   both mode fixtures run with **no wake channel**, and every refusal a
   lane meets in them is followed by an admitting act or by an
   escalation carrying the refusal — the checkable form of one-retry
   convergence.
5. **Correct the one v1-vocabulary leak in the spec.** In
   `next/spec/observations.md` the expired row's reap heuristic reads
   "after grace on the lease" — v1 vocabulary for a mechanism Seed does
   not have. Replace it with the grace stated in Seed's own terms (a
   grace measured from the last observation, the quantity the
   classification already computes), keeping spec and build plan
   telling one story about liveness. One sentence; no threshold, no
   default and no behavior changes.
6. `next/docs/progress.md`: record the amendment on the Phase 9 rows so
   a fresh agent reading the frontier meets the obligations where it
   meets the items. Receipt; evidence; review.

## File Scope

- `docs/next-build-plan.md` (sentence-level extensions to Phase 9
  items 1, 2, 3 and 4; no phase, dependency or exit-line changes)
- `next/spec/observations.md` (the one reap-heuristic sentence)
- `next/docs/progress.md`
- `receipts/os-68ea0b2d.json`

## Acceptance Criteria

1. Phase 9 item 1 carries D1, D2, D4 and D5 in its own words, each
   stated so a fixture or a validation check can fail on it, and none
   of them adding a verb, a state, or a source of truth.
2. Phase 9 item 2 carries D3's escalation half; item 3 carries D4's
   lint half; item 4 carries the wakeless-fixture and
   one-retry-convergence assertions.
3. The lease correction lands: `next/spec/observations.md` no longer
   assigns Seed a lease it does not have, and the build plan and the
   spec tell one story about how liveness is established.
4. No phase heading, `deps:` line or `*Exit:*` line changes anywhere
   in the file; the diff is additive sentences inside four items plus
   the one spec sentence.
5. The frontier records the amendment against the Phase 9 rows.
6. `make check` green (docs-only, so unchanged code surfaces).

## Validation Commands

```sh
make check
```

## Expected diff shape

Three files: `docs/next-build-plan.md` (additive sentences inside
Phase 9 items 1 through 4), `next/spec/observations.md` (one
corrected sentence), and `next/docs/progress.md` (the Phase 9 rows) —
plus the receipt. Roughly +45/-6 lines. No code surfaces.
