# Plan: next — the unreserved-spend bar counts a reservation's occurrence, not an open valid one (os-88df7ab2)

`simulate.Audit`'s unreserved-spend bar keeps one boolean per subject,
sets it on any `budget.reserve` record and clears it only on
`claim.taken`. Two consequences, both on the chains `seed ledger audit`
exists to judge:

1. **A closed reservation keeps covering spend.** Nothing clears the
   flag on `budget.settle` or `budget.release`, so reserve, settle,
   then a second `run.started` in the same claim window reads as
   covered and the bar stays empty.
2. **An inadmissible reservation counts like an admitted one.** The
   bar reads the verb's occurrence, never its validity. The audit runs
   after chain verification, which establishes signatures and
   transition legality but never admission, so a raw-pushed
   `budget.reserve` from a signer with no standing covers a later run.

Charter III.R row 5 is measured by this bar over the shadow run's real
chain, so a bar that can be satisfied by a settled or inadmissible
reservation cannot carry that measurement. Found by review on #306
(chatgpt-codex-connector); out of that card's bounds by its plan D5
(no bar changes meaning), so it was filed rather than folded in. Tier:
standard (it changes what a conformance bar counts). Deps:
**os-b86dab4c**, whose fix (#306) is the base this builds on: it
renames the verb the bar counts to the protocol's `budget.reserve`
and adds `AuditedVerbs` with the catalog guard. Implementing this
card before that one is on `main` would rewrite the same switch and
conflict, so the card waits on it as well as on this plan.

## What the tree actually shows

- **The protocol already derives this, position-accurately.**
  `transition.SubjectState.DeriveBudget(valid, closeValid)` returns a
  `BudgetView` whose `Open` holds exactly the reservations that are
  valid and not effectively closed, with `ClosedBy` mapping a
  reservation's position to the close that ended it. Its doc states
  the rules the bar is guessing at: invalid reservations consume
  nothing, their closes decide nothing, the first valid close wins,
  later attempts are inert.
- **Admission publishes the predicates.** `admit.BudgetViewAt(records,
  table, subject, state)` supplies them (`ReservationValid`,
  `BudgetCloseValid`) and is documented as "the one computation
  admission, `seed budget status`, and the projections share". The bar
  is the fourth caller that should share it.
- **The facts carry positions, so one fold answers every run.**
  `ReservationFact.Pos` and `CloseFact.Pos` are chain positions, so a
  single derivation per subject over the whole record slice answers
  "was a valid reservation open at position p" for every `run.started`
  by comparing positions. No re-fold per run, so the bar stays linear
  in the chain and the audit's cost over a week-long shadow chain does
  not become quadratic.
- **The import direction is free.** `internal/admit` does not import
  `internal/simulate`; `internal/simulate` imports `transition`
  today, and the audit's own drills already import `admit` (the
  catalog guard from os-b86dab4c).
- **The base is #306, not `main` as it stands.** On `main` today the
  bar still counts `budget.reserved`; os-b86dab4c corrects that and
  introduces the exported verb set this card's switch reads. This
  plan is written against the tree once that merges.

## Design decisions (binding for this task)

- **D1 — the bar asks the shared computation.** For each subject the
  audit derives one `BudgetView` through `admit.BudgetViewAt` over the
  records it was handed, and a `run.started` at position `p` is
  covered when some reservation is valid, opened before `p`, and
  either never closed or closed after `p`. Uncovered runs name the
  subject as they do today. The bar stops keeping its own boolean, so
  the audit no longer restates the protocol's budget model, which is
  the same correction os-b86dab4c made to its verb names.
- **D2 — a run before any reservation is still unreserved spend.**
  The existing arm keeps its meaning: a `run.started` with no
  preceding valid reservation names the subject. This card only adds
  the two cases the boolean could not see (closed, inadmissible), and
  the drill asserts the old arm unchanged so the fix is not a
  loosening.
- **D3 — the drills are the three arms, each failing on the boolean.**
  A settled reservation followed by a second run, a released
  reservation followed by a run, and a reservation admission rejects
  followed by a run: each reads as unreserved spend naming the
  subject, and each passes today (wrongly) if D1 is reverted, which
  the PR shows. A covered run still audits clean, through
  `simulate.Audit` and through `seed ledger audit`.
- **D4 — the audit stays a reader.** It refuses nothing new, adds no
  verb, and keeps its five bars and their names; `seed ledger audit`'s
  envelope, exit codes and evidence lists are unchanged. Where the
  handed records are a prefix that cannot be judged (no table), the
  bar behaves as the chain-violation arm already does and does not
  invent a violation.
- **D5 — bounds.** No change to `transition`, to `admit`, to the
  transition table, or to `cmd/seed/ledger.go`. The simulation's own
  `Audit` call site is unchanged. Nothing about III.R row 5's status
  moves: the row is met by the shadow run's evidence card, not here.

## Steps

1. D1 in `internal/simulate/audit.go`: the per-subject view, the
   position comparison, the boolean removed.
2. D3's drills in `internal/simulate/audit_test.go`, and the covered
   arm through the CLI in `cmd/seed/ledger_audit_test.go`.
3. `next/spec/simulation.md`'s sentence on what the bar counts;
   `next/docs/progress.md`, `next/docs/decisions.md`,
   `memory/LEARNINGS.md`; receipt; evidence; review.

## File Scope

- `next/internal/simulate/audit.go`, `next/internal/simulate/audit_test.go`
- `next/cmd/seed/ledger_audit_test.go`
- `next/spec/simulation.md`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-88df7ab2.json`

Nothing else. NOT `next/internal/transition/**`, NOT
`next/internal/admit/**`, NOT `next/cmd/seed/ledger.go`, NOT
`next/spec/conformance.json`, NOT `.seed/**`.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. **A closed reservation covers nothing after its close.** A chain
   with reserve, settle, then `run.started` reads as unreserved spend
   naming the subject; the same with `budget.release`.
2. **An inadmissible reservation covers nothing.** A chain whose
   `budget.reserve` admission would reject, followed by
   `run.started`, reads as unreserved spend naming the subject.
3. **A covered run still audits clean.** Reserve, run, settle in one
   claim window leaves `unreserved_spend` empty and `clean` true,
   through `simulate.Audit` and through `seed ledger audit`.
4. **The fix is what makes the difference.** Reverting D1 to the
   boolean fails the three drills of AC1 and AC2 by name (shown in
   the PR).
5. `make check` green; no model identifiers in any committed artifact.

**Retention set (existing, shown unharmed):**

- The other four bars keep their meaning and their drills, and the
  audit's own suite passes unchanged: `TestAuditCleanRun`,
  `TestAuditCatchesSilentAbandonment`, `TestAuditCatchesUnreservedSpend`,
  `TestUnreservedSpendCountsTheProtocolsReservation`,
  `TestAuditedVerbsAreTheProtocols`; and the verb's envelope drills
  `TestLedgerAuditCleanAndEachBar`, `TestLedgerAuditVerifiesBeforeAnyBar`,
  `TestLedgerAuditRefusalOrderIsDeterministic`.

## Validation Commands

- Boundary: `cd next && go test ./internal/simulate/ -count=1`
- Boundary: `cd next && go test ./cmd/seed/ -run 'LedgerAudit' -count=1`
- Retention: `make check` (exit checked separately from any pipe)

## Expected diff shape

Modified: `audit.go` (the per-subject budget view replacing the
boolean), `audit_test.go` (three drills plus the covered arm),
`ledger_audit_test.go` (one CLI arm), one paragraph in
`simulation.md`, the three docs files, the receipt. Roughly +120/-25
lines. No other file.
