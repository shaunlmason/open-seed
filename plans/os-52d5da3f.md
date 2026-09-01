# Plan: next Phase 9 item 5 — obligations projection and the situation read (os-52d5da3f)

Implements the first two parts of `docs/next-build-plan.md` Phase 9
item 5, "the lane-facing surface", scheduled by the promotion
amendment (#169) and designed in `plans/os-768361cc.md` D4. Design
authority: SEED-NEXT.md §II.4 (projections are derived,
position-stamped, rebuildable, non-authoritative), §II.10
(affordances and the position stamp), and the promotion section's
loop-completeness criterion, which this surface exists to satisfy.

**The gap.** Seed represents *permission* — `admit.Affordances`
answers "what may I do", drift-tested since Phase 8 — but not
*obligation*: nothing answers "what is owed, by whom, since when,
under what clock, and which verbs discharge it". Every fact needed
is already folded (`transition.SubjectState` carries the active
`Claim` with its fence, `Submission`, `Verdict`, `Requested`,
`RunStarts`/`Runs`, `Reservations`/`BudgetCloses`, `Since`, and the
state itself), so obligation is a **projection over the fold**, not
a new authority. A lane that cannot orient or choose cannot run
unattended, which is why this is a promotion gate rather than
polish.

## Design decisions (binding for this task)

- **D1 — scope: parts (a) and (b) here, part (c) on its own card.**
  Item 5 names three parts as one deliverable; this task lands the
  **obligations projection** and the **situation read**, which are
  tightly coupled (the read is the projection's consumer), and files
  the **loop verbs** as a follow-up card under the same item. The
  reason is reviewability, not scope avoidance: loop verbs are a
  per-verb surface whose size scales with the verb list, and folding
  a dozen new CLI verbs into the same PR as a new projection would
  produce a diff no reviewer can hold. The follow-up card is filed
  in this task's PR and named in the progress entry, so item 5 stays
  one obligation with two landings.
- **D2 — obligations are derived, never declared.** A new
  `internal/obligation` package folds `[]*event.Record` (with the
  same `transition.Fold` admission uses) into rows of
  `{subject, kind, owed_by, since, due, discharged_by}`.
  `discharged_by` is read from the **transition tables**, never
  hand-listed: the table's `{verb, from, to}` entries already say
  which verbs leave a state, so the discharging set for a
  state-shaped obligation is exactly the verbs whose `from` matches
  the subject's state and whose class the owed-by actor could hold.
  Some obligations are **fact-shaped**, not state-shaped: their
  closing verb changes no lifecycle state and appears in no table
  row. That set is closed and each member cites the spec pairing it
  with its fact — `run.settled` for an unsettled run, `budget.settle`
  / `budget.release` for an open reservation, and
  **`verdict.rendered` for a pending submission**
  (`next/spec/verdicts.md`; review finding on this PR — the table's
  only exits from `review` are `merge.observed`, `contract.returned`
  and `contract.cancelled`, so a table-only derivation would have
  advertised no useful discharger for the verifier lane at all). Nothing here re-derives
  legality: an obligation whose `discharged_by` verb is refused at
  the same position is the III.I row-2 bug class one level up and
  gets the same treatment in D5.
- **D3 — the kind taxonomy is small, closed, and evidence-backed.**
  Exactly six kinds ship, each with a folded fact behind it and a
  named clock or an explicit absence: `claim.held` (an active
  `Claim`; owed by its holder; due at the lease horizon derived from
  the observation thresholds the caller declares, absent when none
  is declared), `submission.pending` (a `Submission` with no later
  `Verdict` citing it; owed by the verifier lane, not an individual,
  since independence forbids naming the claimant),
  `verdict.unmerged` (a pass `Verdict` with no `Merged`; owed by the
  operator/observer lane), `run.unsettled` (a `RunStart` whose fence
  has no matching `Run`; owed by the supervisor — the Phase 7 exit's
  metering-detection obligation, and deliberately **position-
  anchored**: flagged only once the subject has taken a subsequent
  claim window or reached a terminal state, never mid park/reap
  flow), `budget.open` (a valid open reservation **while the
  subject is in_progress**; owed by the holder), and
  `contract.blocked` (state `blocked`; owed by whoever
  the `blocked-on` names, or the operator lane when it names no
  one). Anything else waits for a card: an open-ended kind list
  would make the projection a policy surface, which it must not be.
  The `budget.open` window restriction is a finding, not a
  preference (review finding on this PR): `admit.go` gates every
  budget verb on `in_progress`, so once the holder leaves the window
  a still-open reservation has **no admissible closing act** — both
  advertised dischargers refuse. An obligation nobody can discharge
  is an anomaly, not an obligation, so this projection emits none;
  the stranded-capacity gap itself is card `os-d6963652`, which
  weighs an admission change against routing it to Phase 9 item 3's
  maintenance reap (the precedent the Phase 7 exit set for unsettled
  runs). This plan neither fixes nor hides it: it declines to
  advertise a discharge that cannot happen, and names where the fix
  is being decided.
- **D4 — the situation read is one position-stamped envelope, and
  `--since` is a delta, not a diff.** `seed situation --ledger <dir>
  --key <k> [--subject s] [--since <position>]` renders the
  standard envelope carrying: the caller's obligations (filtered to
  `owed_by` = the signing fingerprint, or its lane where the kind is
  lane-owed), the subjects where the caller holds an active window
  with the fence, unread message count, and the budget block the
  envelope already knows how to render. With `--since <position>`
  the response is a **complete change report**, not a filtered list:
  obligations that arose or changed at or after that position, **plus
  an explicit `discharged` list** naming every obligation that stood
  before the cited position and no longer does, keyed by its stable
  identity `(subject, kind)` with the position that discharged it,
  plus a count of those unchanged. Removals must be explicit (review
  finding on this PR): a delta of standing rows alone leaves a
  resuming lane holding a discharged obligation forever, and an
  unchanged *count* cannot say what disappeared. Stable identity is
  what makes the response applicable, so `(subject, kind)` is
  normative rather than incidental, and applying the response to a
  prior snapshot must reproduce the standing set exactly — the
  property the drills assert. The read
  is read-only and idempotent, opens the ledger read-only, journals
  no attempt (it is not an admission-boundary attempt), and stamps
  affordances exactly as every other keyed surface does.
- **D5 — the drift class comes with the surface.** A sweep in the
  8.2 shape: over the shared walk scenario, at every prefix
  position, every obligation's `discharged_by` verb is independently
  drafted and run through the enforcing `admit.Check` for the owed
  actor; an obligation advertising a verb that admission refuses at
  the same position fails the class by name. Lane-owed kinds probe
  with a fixture key holding the lane's capability. The sweep also
  asserts that **every emitted obligation carries a non-empty
  `discharged_by`** (review finding on this PR: a sweep over an empty
  discharger set passes vacuously, worthless exactly where a kind's
  mapping was forgotten). This is what
  keeps the projection honest without letting it become a second
  legality authority.
- **D6 — scope guard.** No new authority, no new source of truth, no
  orchestration layer, no wall-clock reads inside the build (clocks
  arrive as declared inputs, the observations pattern). Success
  envelopes stay terse; teaching text lives in refusals only. The
  projection is registered in `Default()` like every other, carries
  its own version, and the cache mirrors it only if a later card
  says so — this task adds no cache table.

## Steps

1. `internal/obligation`: the `Row` type, `Derive(records, inputs)`
   over `transition.Fold`, `discharged_by` read from the transition
   tables, and the six kinds per D3. Table-driven unit tests per
   kind, including the position-anchored `run.unsettled` condition.
2. Register the projection (`obligations.json`) in `project.Default()`
   with its stamp and version; byte-identical build drills.
3. `seed situation` per D4 in `cmd/seed`, with CLI drills: keyed and
   keyless, `--subject` filtered and whole-board, `--since` returning
   the delta and the unchanged count, and the empty case.
4. The D5 drift sweep in `internal/obligation` (or `internal/admit`
   if the fixtures live better there).
5. `next/spec/obligations.md`: the row shape, the six kinds with
   their folded evidence and clocks, the `discharged_by` derivation,
   the `--since` semantics, and the deliberate absences; referenced
   from `projections.md`.
6. File the loop-verbs follow-up card (part (c)); update
   `next/docs/progress.md` (Phase 9 section, item 5 split named) and
   `next/docs/decisions.md`; memory entries if the build teaches
   anything durable.
7. `make check`; receipt; evidence; review.

## File Scope

- `next/internal/obligation/` (new), `next/internal/project/`
  (registration + drills), `next/cmd/seed/` (the read + drills)
- `next/spec/obligations.md` (new), `next/spec/projections.md`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-52d5da3f.json`

## Acceptance Criteria

1. Obligations derive from the fold with `discharged_by` read from
   the transition tables; no hand-written legality.
2. The six kinds are covered by table-driven tests, `run.unsettled`
   position-anchored per the Phase 7 exit's routing.
3. `seed situation` returns one position-stamped envelope; `--since`
   returns arisen-or-changed obligations **and an explicit
   `discharged` list keyed by `(subject, kind)`**, such that applying
   the response to a prior snapshot reproduces the standing set
   exactly (asserted by drill); the read journals nothing and mutates
   nothing.
4. The drift sweep fails when an advertised `discharged_by` verb is
   refused at the same position, and fails when any emitted
   obligation carries an empty `discharged_by`.
5. `budget.open` is emitted only inside the live claim window, with
   the outside-window stranded-reservation gap named and routed to
   card `os-d6963652` rather than papered over.
6. `next/spec/obligations.md` documents the shape, kinds, derivation
   and absences; `projections.md` references it.
7. The loop-verbs follow-up card exists and is named in the progress
   entry, so item 5 reads as one obligation with two landings.
8. `make check` green, coverage gate ≥90% held.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/obligation/ ./internal/project/ ./cmd/seed/ -count=1
make check
```

## Expected diff shape

One new package, one new spec file, one new CLI verb with drills,
projection registration, and the docs. Roughly +900/-20 lines, all
under `next/**` plus the memory files.
