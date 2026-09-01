# Plan: next — the unattended maintenance loop (os-8a5f14bb)

Phase 9 item 3 of `docs/next-build-plan.md`: *"Maintenance loop: reap
expired/wedged, reconcile divergence, rebuild projections, checkpoint
(signed), lints — runnable unattended; audited as an ordinary actor."*

Two specs already name this lane's reap as **their** recovery path, so
this card is owed to code that has shipped:

- [`loop-verbs.md`](../next/spec/loop-verbs.md): a window stranded by a
  key rotation cannot be parked by the rotated key, because the fence
  rule admits holder-signed events only from the holder. *"Recovering it
  is the maintenance lane's reap."*
- [`escalation.md`](../next/spec/escalation.md): a contract frozen by a
  persuaded lane raising a question is named as what this lane's lints
  should surface.

## What already exists, and what this card must not rebuild

Measured against the tree, not assumed:

- **`internal/reconcile`** already carries a closed finding set —
  `Finding{Subject, Class, Detail}` with `Subject`, `VerifyVerdicts`,
  `VerifySeals`, `VerifyOverrides`, `Classify`. One of its own detail
  strings forward-references this card: *"pending or diverged is an age
  judgment for maintenance, not this classifier."*
- **`internal/obs.Classify`** gives `expired` / `wedged` / `no_data`
  against `Thresholds`, as `observations.md` defines them.
- **`internal/obligation`** already emits `run.unsettled`
  **position-anchored** exactly as the Phase 7 exit requires: flagged
  only once the subject has taken a subsequent claim window or reached
  a terminal state, because post-close settlement is a valid
  intermediate state.

So the lints are largely **built**. What is missing is a lane that runs
them unattended, decides what a finding becomes, and can reap.

**A correction, since the card body implied otherwise.**
`lanes/maintenance.json` declaring `"acts_through": []` is **not a
gap**. `internal/lane`'s validation obliges only a lane holding
`CapClaim` to declare acts; a lane acting through the raw append seam
declares none, and the maintenance lane holds `maintenance` and
`operator`. Its empty declaration is the documented posture.

## Design decisions (binding for this task)

- **D1 — `seed maintain run` is a CLI verb, and the reason `seed loop
  run` was refused does not apply here.** `internal/loop` is a library
  with no verb because **Seed does not own the work**: the work step is
  the caller's, and a CLI verb would invite treating the CLI as the
  agent.

  The maintenance lane's work is Seed's own. Reaping, reconciling,
  rebuilding and checkpointing are defined acts over the ledger with no
  caller judgment inside them — there is no work step to supply. So the
  argument that refused `seed loop run` is not an argument against this
  verb; it is what distinguishes them, and the plan records that rather
  than leaving the asymmetry to look like inconsistency.

  The decision logic still lives in `internal/maintain` with its
  effects injected, so it is drillable without a ledger — the
  `internal/covergate` shape (plan `plans/os-cafba959.md`).

- **D2 — the lint set is CLOSED, and this card adds exactly one.**
  `internal/reconcile`'s classes are the set; the addition is
  **unsettled-run detection**, consumed from `internal/obligation`'s
  `run.unsettled` rather than re-derived. Re-deriving it would put the
  position-anchoring in two places, and the anchoring is the whole
  subtlety: a closed-without-settle predicate files spurious findings
  mid park or reap flow.

  Closed means a new lint lands by adding a class **with the spec that
  pairs it to its fact**, the `factDischargers` precedent in
  `obligations.md`. An open-ended lint list would make this loop a
  policy surface, which is what "audited as an ordinary actor" denies.

- **D3 — a lint finding becomes a FILED DEFECT CONTRACT, never an
  escalation.** The charter says maintenance "files defect contracts",
  and the distinction is real: an escalation **freezes** a contract and
  demands a human decision; a lint finding is work someone should do.
  Filing is `intent.filed`, which the maintenance lane can sign because
  it holds `operator`.

  Consequence, stated: the maintenance loop can therefore create work,
  which is authority. It is bounded by being attributable (its own key,
  its own lane) and by filing nothing but contracts — it cannot claim
  them, because it does not hold `CapClaim`.

- **D4 — a reap requires corroboration beyond silence, and `no_data`
  never reaps.** This is the plan's sharpest constraint and comes
  straight from `observations.md`: the channel is ephemeral and lossy
  **by declaration**, so a dropped stream and dead work look identical
  from outside.

  So a reap requires **both**:
  1. an `expired` or `wedged` classification from `obs.Classify`, and
  2. an independent position-anchored fact — the claim's own lease
     elapsed, measured against the claim event's `ts` at the read's
     `--now`, the live-read posture `offers.md` sets and
     `escalation.md` reuses.

  `no_data` carries **no reap path whatever**, however old. A drill
  asserts that directly, because it is the case where the instinct to
  reap is strongest and the evidence weakest.

  **No heartbeat predicate is added.** Non-advancing observations are
  not a heartbeat signature: a legitimate long-running step emits
  exactly that shape, and the existing expiry/wedge classification
  already distinguishes it.

- **D5 — "runnable unattended" is drilled WAKELESS.** No scheduler, no
  wake channel: the drill runs one pass against a real ledger and
  asserts what it did. Item 4's fixtures use the same posture, and a
  loop that needs a scheduler to be testable is one nobody can prove
  runs unattended.

- **D6 — audited as an ordinary actor, asserted rather than described.**
  Every act the loop takes is signed with the maintenance key and goes
  through the same admission boundary. A drill runs the loop with a key
  holding **only** `maintenance` and asserts the acts needing
  `operator` refuse `out_of_grant` — so "no private powers" is a
  property the boundary enforces, not a claim the summary makes.

- **D7 — scope guard.** No new lifecycle verb, no change to
  `obs.Classify`'s thresholds or classes, no change to the reconcile
  findings this card does not add. `claim.reaped` already exists and is
  already capability-gated; this card gives it a caller, not a new
  authority.

## Steps

1. `next/internal/maintain` — the pass: reap candidates, lint findings,
   projection rebuild, checkpoint, with the ledger effects injected.
2. `next/internal/maintain/maintain_test.go` — the drills of D4 and D6,
   including `no_data` refusing to reap and the operator-less key
   refusing.
3. `next/internal/reconcile` — the unsettled-run class, consumed from
   `internal/obligation`.
4. `next/cmd/seed` — `maintain run`, its envelope, and the affordance
   catalog entries for any act it performs.
5. `next/spec/maintenance.md` (new) — the loop, the reap's corroboration
   rule, the closed lint set, and what a finding becomes.
6. `next/lanes/maintenance.json` — `acts_through` stays empty (it acts
   on the raw seam); the summary gains what the loop actually does.
7. `next/docs/decisions.md`, `memory/LEARNINGS.md`; receipt; evidence;
   review.

## File Scope

- `next/internal/maintain/**` (new), `next/internal/reconcile/**`
- `next/cmd/seed/**`
- `next/spec/maintenance.md` (new), `next/spec/observations.md` (the
  reap's corroboration rule, cross-referenced)
- `next/lanes/maintenance.json`
- `next/docs/decisions.md`, `next/docs/progress.md`, `memory/*`
- `receipts/os-8a5f14bb.json`

Nothing outside `next/**` except the work-product files above.

## Acceptance Criteria

1. One `maintain run` pass against a real ledger reaps an expired,
   lease-elapsed claim and leaves a `claim.reaped` packet, asserted by
   reading the chain back rather than by the verb's own report.
2. **A `no_data` stream is never reaped**, however old the claim, and
   the refusal says why. Drilled directly, because this is where the
   instinct to reap is strongest and the evidence weakest.
3. An `expired` classification **without** the independent lease fact
   does not reap either — the corroboration is a conjunction, and a
   drill plants each half alone.
4. The unsettled-run lint fires only once the subject has taken a
   subsequent claim window or reached a terminal state, and **not**
   mid park or reap flow. Both directions asserted, so it cannot pass
   by never firing.
5. A lint finding files a defect contract, and the filed contract cites
   the finding's class and subject. No lint raises an escalation.
6. **The loop holds no private powers**: run with a `maintenance`-only
   key, the acts needing `operator` refuse `out_of_grant` at the
   boundary. That is D6 enforced rather than described.
7. **Mutation evidence, per fix.** Each must fail a drill: allowing
   `no_data` to reap; dropping the lease conjunct; re-deriving the
   unsettled-run anchoring instead of consuming it; turning a lint
   finding into an escalation.
8. `make check` green with coverage measured **cold**, at least three
   readings above the gate, and the suites pass unprivileged.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/maintain/ ./internal/reconcile/ -count=1
cd next && go test ./... -count=1
make check
```

## Expected diff shape

One new package with its drills, one new lint class, one CLI verb, one
new spec and two amended, plus the work-product files. Roughly
+900/-40 lines, all under `next/**`.

## A risk worth naming now

D4's corroboration rule is the decision most likely to be wrong, and it
is wrong in the direction that is hard to see: if the lease fact and the
expiry classification are **correlated** — both ultimately derived from
the same silence — then requiring both is theatre, not corroboration,
and the loop would reap on one piece of evidence while appearing to
require two.

They are not correlated as specified: the lease elapses against the
claim event's own `ts`, which is in the chain and independent of
whether any observation was ever emitted. But that independence is the
property to check, so the drill for criterion 3 plants each half alone
rather than only testing the conjunction — a test that only ever sees
both cannot tell a conjunction from a coincidence.
