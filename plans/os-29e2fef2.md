# Plan (next): prefer the prior submitter's tuple when re-offering a contract returned on the forge's word (os-29e2fef2)

The follow-up `plans/os-0cd18799.md` D8 named and its decision log
filed: a contract returned by `contract.returned` citing a
`check.observed` records no lockout, so the prior submitter is the
natural next claimant, but nothing prefers it and, as this plan found,
nothing re-offers the subject at all. Supervisor ranking policy
(build plan Phase 13 item 7; `next/spec/ranking.md`), charter §II.9
(offers are the scheduling input; "wake is advisory, never the grant")
and §II.11 (the maintenance lane files what it finds, unattended).
Tier: standard. Deps: os-0cd18799 (merged, #351). Not
conformance-blocking; no Part III row moves.

**The gap.** After the maintenance pass returns a red submission
(`next/spec/observations-forge.md`), the subject stands in `ready`
with its submission, its observation and its returns on the fold, and
waits for an `offer.published` before any worker loop will poll it:
the loop's poll is `seed offer list`, and a subject with no live offer
is invisible to it. The v0 fixtures publish that offer by hand
(`modeStand.contract`). So the unattended loop os-0cd18799 built runs
red, returned, and then stalls until a supervisor notices, and when
one does the offer it publishes is unscoped or ranked by evals,
neither of which knows that one configuration already holds the
branch, the threads and the context.

## What the tree actually shows

- **The ranking is per capability, never per subject.**
  `internal/ranking.Derive` orders the qualified tuples by their eval
  evidence; `Top(r, capability, n)` fills `--strongest`; the policy
  table is drilled against `ranking.Rules`. Nothing reads a subject's
  own history.
- **The fold knows the prior submitter's configuration.** After a
  return the subject's `Submission` still names the returned
  submission; its payload's `fence` is the claim window; `RunStarts`
  carries the admitted `run.started` at that fence with the `Tuple`
  the supervisor declared (`transition.RunStartFact`); `Returns`
  carries what the latest return cited (`ReturnFact.Observation`);
  the keyring says whether the holder still stands.
- **An offer's `tuples` scope is a set.** Admission meets it by any
  member (`next/spec/offers.md`), so "prefer" has no ordering inside
  one offer: preference is a narrower offer, and widening is a later
  offer or an unscoped one. A claim consumes every offer at or before
  it, so a re-offer after the return carries current intent by
  construction.
- **The maintenance pass publishes nothing.** It reaps, observes,
  returns, lints, files defect contracts, rebuilds and checkpoints
  (`next/internal/maintain`); `offer.published` accepts `supervise`
  or `operator`, and the maintenance key holds `operator`, so the
  boundary would admit a re-offer from it.
- **`seed offer publish` takes `--tuple` and `--strongest`.** Both
  write the same scope and refuse together; `ranking_empty` (exit 4)
  is the established refusal for policy that yields nothing.

## Design decisions (binding for this task)

- **D1. `ranking.Resume` derives the prior submitter's tuple from the
  fold.** `Resume(records, fold, subject) (Resumption, bool)` reads
  the subject's latest applied return; when it cited an observation,
  the returned submission's window (the submission payload's `fence`,
  as `admit.submissionWindow` reads it), the admitted `run.started`
  at that fence, and its declared tuple; the holder that took the
  window (the `claim.taken` signer at the fence). It yields
  `{Tuple, Holder, Submission, Return}` and true, and false with a
  named reason when any link is missing: the latest return was the
  verifier's (a fail verdict routes to whoever is strongest, not to
  whoever failed), no return stands, the window carried no admitted
  `run.started` or one that declared no tuple, or the holder is
  suspended or revoked (a preference nobody active can take is an
  offer nobody can take). Record-derived at the declared instant like
  the ranking, never a clock; no admission rule reads it. Refused: a
  preference derived from anything but the chain; a preference for a
  verdict-returned subject.
- **D2. `seed offer publish --resume`.** Fills the `tuples` scope with
  the resumption's tuple, alone or beside `--strongest <n>` and
  `--tuple` (a set, the prior tuple first in the payload for the
  reader), and refuses `resume_empty` (exit 4, the `ranking_empty`
  posture) naming D1's reason when nothing derives: an unscoped offer
  stays the supervisor's explicit choice. The result names the
  resumption (`tuple`, `holder`, `return`). The payload is unchanged
  from `offers.md`; admission judges it by the existing scope rule.
  Refused: a new payload field; a new exit code.
- **D3. The maintenance pass re-offers what it returned.** A
  `reoffer` step after `return`: for every subject the pass returned
  in this pass, append `offer.published` on it, scoped by D1's tuple
  when one derives and unscoped by tuple otherwise, with the
  capabilities and tiers of the offer the returned claim consumed
  (the latest applied offer at or before the claim's position, from
  `Offers`) or, where none stands, `capabilities: [claim]` and the
  subject's filed tier. `expires` is the append's own instant plus
  `--reoffer-ttl` (default 24h, a declared duration, read at the
  effect since an offer must expire strictly after its own `ts` and
  the pass's `--as-of` may be historical). The re-offer is one per
  return, in the pass that returned it: an expired re-offer is the
  supervisor's to renew, because the pass returns work and does not
  run the queue. The report gains `reoffered`
  (`{subject, tuple?, holder?, expires}`); a refusal is reported like
  every other, and a `maintenance`-only key meets `out_of_grant`. The
  scope rule is pure in `internal/maintain` (`Reoffer(s, resumption,
  ttl)` renders the payload; the effect is `Deps.Append`), drilled
  without a ledger. Refused: a re-offer on a subject the pass did not
  return; a scope wider than the consumed offer's; a clock read
  inside the rule.
- **D4. The worker's poll shows why.** `seed offer list` already
  renders the `tuples` scope; a re-offer scoped to one tuple is
  visible as such, and a worker holding another configuration does
  not see it (the existing scope rule). Nothing is added to the read.
  The situation read is unchanged.
- **D5. Specs and fragments.** `next/spec/ranking.md` gains "Resume:
  the prior submitter's tuple" beside the policy table (the table and
  `ranking.Rules` are unchanged: the preference is per subject, not a
  ranking rule); `offers.md` the flag; `observations-forge.md` and
  `maintenance.md` the re-offer step and its one-per-return rule; the
  maintenance fragment one clause ("re-offer what you returned, to the
  configuration that held it"). `seed docs generate` regenerates the
  governed docs; the trajectory corpus re-records if the maintenance
  posture digest moves.
- **D6. Bounds.** No admission, transition, keyring or protocol
  change; no new verb; `ranking.Derive`, `Top` and the policy table
  untouched; nothing outside `next/**` but the work-product files.
  NOT `.seed/**`, NOT `scripts/**`, NOT `.github/workflows/**`.

## Steps

1. `internal/ranking`: `Resumption`, `Resume` per D1, table-driven
   drills over hand-built states and one chain.
2. `cmd/seed/offer.go`: `--resume` per D2 with its drills (alone, with
   `--strongest`, with `--tuple`, `resume_empty` for each reason).
3. `internal/maintain`: `Reoffer` and the step per D3, the report
   section, the `--reoffer-ttl` flag in `cmd/seed/maintain.go`; the
   CLI drills over the forge stand from os-0cd18799: a red return is
   followed by a re-offer the prior configuration's worker lists and
   another does not, the consumed offer's tiers ride along, no
   declaration yields an unscoped re-offer, a `maintenance`-only key
   is refused and reported, a second pass re-offers nothing.
4. Specs and the fragment per D5; `docs generate`; the corpus if it
   moved; `next/docs/progress.md`, `next/docs/decisions.md`,
   `memory/LEARNINGS.md`; `make check`; receipt; evidence; review.

## File Scope

- `next/internal/ranking/**`, `next/internal/maintain/**`
- `next/cmd/seed/offer.go`, `next/cmd/seed/maintain.go` and their
  drills
- `next/spec/ranking.md`, `next/spec/offers.md`,
  `next/spec/observations-forge.md`, `next/spec/maintenance.md`
- `next/lanes/fragments/lane/maintenance.md`,
  `next/docs/generated/**`, `next/trajectories/lanes/**`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-29e2fef2.json`

Nothing outside `next/**` except the work-product files above. NOT
`.seed/**`, NOT `scripts/**`, NOT `.github/workflows/**`, NOT
`next/internal/admit/**`, NOT `next/spec/transitions.json`.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. **The resumption derives from the chain.** `Resume` yields the
   returned window's declared tuple and holder after a return by
   observation, and refuses by name after a return by verdict, with
   no return, with no admitted `run.started` on the window, with a
   start that declared no tuple, and with a suspended or revoked
   holder.
2. **The verb scopes by resumption.** `offer publish --resume` appends
   an offer whose `tuples` scope carries the prior tuple, alone or
   beside `--strongest` and `--tuple`; the worker holding that tuple
   lists it and a worker holding another does not; `resume_empty`
   names D1's reason and appends nothing.
3. **The pass re-offers what it returned.** One `maintain run` over a
   red snapshot returns the submission and appends a re-offer scoped
   to the prior tuple with the consumed offer's capabilities and
   tiers, expiring strictly after its own `ts`; the prior worker's
   `offer list` shows it; a window with no declared tuple yields a
   re-offer unscoped by tuple; a second pass over the same subject
   re-offers nothing; a `maintenance`-only key sees the re-offer
   refused `out_of_grant` and reported.
4. **The loop closes unattended.** In the forge stand, red snapshot,
   pass, the prior worker's poll lists the re-offer, reclaim through
   `claim take`, resubmit, green snapshot, pass, verdict, merge chain
   to `done`, with no offer published by hand after the first.
5. **Mutation evidence.** Each fails a drill: a resumption derived
   after a verdict return; a re-offer wider than the consumed offer's
   scope; a re-offer on a subject the pass did not return; an expiry
   computed from the declared instant; a suspended holder's tuple
   preferred; the policy table or `ranking.Rules` edited.
6. `make check` green with coverage measured cold above the gate;
   the generated docs drift-free; no model identifiers in any
   committed artifact.

**Retention set (existing, shown unharmed):**

- `ranking.Derive`, `Top`, the policy table drill and every existing
  `offer publish` drill are unchanged; the maintenance pass's reap,
  observe, return, lint, file, rebuild and checkpoint drills are
  green; `offer.published` admits exactly as before; the modes
  fixtures reach `done` as before.

## Validation Commands

- Boundary: `cd next && go test ./internal/ranking/ ./internal/maintain/ ./cmd/seed/ -run 'Resume|Reoffer|OfferPublish|MaintainObserves|MaintainReturns' -count=1`
- Retention: `cd next && gofmt -l . && go vet ./... && go build ./... && go test ./... -count=1`
- Retention: `make check`

## Expected diff shape

New: `Resume` in `next/internal/ranking/`, `Reoffer` and the step in
`next/internal/maintain/`, their drills. Modified: `offer.go` (the
flag), `maintain.go` (the wiring and `--reoffer-ttl`), the forge
stand drills, four spec pages, the maintenance fragment and its
generated doc, the three docs files, the receipt. Roughly +700/-20
lines, all under `next/**` plus the memory files. No admission,
transition, keyring or protocol change.
