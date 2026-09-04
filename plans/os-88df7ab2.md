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
- **A run cites one reservation, and admission already judges that
  citation.** `RunStartFact.Reservation` is the position the start
  named, and `admit.RunStartValid(records, table, subject, st)` is the
  whole question in one predicate: the payload's strict shape, the
  fence matching the active claim, the cited reservation matching the
  fold, the signer's capability and tuple, `ReservationValid` on the
  cited reservation, and `BudgetViewAt(prefix).ClosedBy` proving it
  was still open at the run's own position (admit.go:2787).
- **The derivation is not linear.** `BudgetViewAt` calls
  `ReservationValid` per reservation fact, and each call replays
  `keyring.StateAt(prefix)` and `table.StateAt(prefix, subject)` from
  the chain's start; close validation replays again, and
  `RunStartValid` pays that plus a nested `BudgetViewAt` over its own
  prefix. The honest worst case is about `O(runs x reservations x
  records)` per subject, not a single linear fold.
- **The import direction is free.** `internal/admit` does not import
  `internal/simulate`; `internal/simulate` imports `transition`
  today, and the audit's own drills already import `admit` (the
  catalog guard from os-b86dab4c).
- **The base is #306, not `main` as it stands.** On `main` today the
  bar still counts `budget.reserved`; os-b86dab4c corrects that and
  introduces the exported verb set this card's switch reads. This
  plan is written against the tree once that merges.

## Design decisions (binding for this task)

- **D1 — the bar asks the protocol about the run's own citation.**
  For each folded `RunStartFact` the audit asks
  `admit.RunStartValid(records, table, subject, st)`, and a start it
  refuses names the subject as unreserved spend. That predicate is
  the fence, the cited reservation's validity and its openness at the
  run's position, so the bar neither keeps a boolean nor re-derives a
  view: it asks the same authority admission asks. Deliberately
  **not** "some valid reservation was open at p" (this plan's first
  reading, corrected by review on #309): a start citing a closed or
  absent reservation while another is open would pass that and fail
  admission, which is the fencing the bar exists to check.
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

- **D7 — the fixtures become admission-grade, because the rule
  demands it.** Measured while implementing D1: the bar can only judge
  a start the fold recorded, and `transition` records a
  `RunStartFact` only where the payload named a fence and a
  reservation it could read. So a `run.started` the fold did not
  record cited nothing checkable and is spend with no fence, which the
  bar must name (else a malformed raw start is invisible, the hole
  this card exists to close). That makes every synthetic `{}` fixture
  in `internal/simulate/audit_test.go` and the `auditLedger` histories
  in `cmd/seed/ledger_audit_test.go` read as unreserved spend, and
  correctly so: they are not chains any admission would have taken.
  The covered arm therefore needs a real chain, and the tree already
  builds one: `internal/history.Generate` writes an admission-grade
  ledger carrying `budget.reserve`, `run.started`, `run.settled` and
  `submission.made` under enrolled lane keys, and `internal/history`
  does not import `internal/simulate`, so the drills read their
  records back through `ledger.OpenReadOnly` and
  `VerifyFromGenesis`. The violation arms are then raw appends onto
  that chain, which is what they model. This is scope the first
  reading did not carry, and it is why the file scope below gains the
  two test files' fixtures rather than a line each.
- **D6 — the cost is measured, not asserted.** The audit gains no
  cache in this card; the drill set carries one measurement instead: a
  chain of the shadow window's shape (the perf history's 40 contracts,
  each with a reservation and a run) is audited and the reading
  recorded in the PR, so the verb's cost over a real chain is a number
  in the record rather than a claim. If that reading exceeds a stated
  ceiling of ten seconds the implementation memoizes the per-position
  replays behind one pass and says so in the task PR; D1's
  correctness rule does not move either way.
## Steps

1. D1 in `internal/simulate/audit.go`: `RunStartValid` per folded
   start, the boolean removed.
2. D3's drills in `internal/simulate/audit_test.go` (including the
   cited-reservation arm: a start citing a closed reservation while
   another is open), the covered arm through the CLI in
   `cmd/seed/ledger_audit_test.go`, and D6's measurement.
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
