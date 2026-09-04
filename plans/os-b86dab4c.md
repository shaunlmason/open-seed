# Plan: next — the five-bar audit counts a verb the protocol does not define (os-b86dab4c)

`simulate.Audit`'s unreserved-spend bar counts a reservation under the
verb `budget.reserved`; the protocol defines and emits `budget.reserve`
(`transition.BudgetReserveVerb`). So a real chain whose `run.started`
is covered by an admitted reservation is reported as unreserved spend,
and `seed ledger audit` (os-7599c27d, #296) cannot measure charter
III.R row 5 over the shadow run, which is the one thing the verb
exists for. Found by review on #296; the two decisions were drafted as
an amendment to that card's plan (#303) and are carried here instead,
because #296 merged first and a merged card's plan is not where new
work is authorized. Tier: standard (it changes what a conformance bar
counts). Deps: none.

## What the tree actually shows

- **The name is wrong in exactly two places, and they agree with each
  other.** `next/internal/simulate/audit.go` switches on
  `"budget.reserved"`, and its own fixture
  (`internal/simulate/audit_test.go`, `happy()` and two bar drills)
  files the same string, so every drill passes. The CLI drill added
  with #296 (`cmd/seed/ledger_audit_test.go`) inherited that history.
  Nowhere else in `next/**` writes `budget.reserved`; every other
  caller, the CLI drills included, uses `budget.reserve`.
- **Nothing else exercises the bar, which is why it survived.** The
  simulation's own deployment appends `intent.filed`,
  `contract.specified`, `offer.published` and the curator and
  maintenance passes: it never files a reservation or starts a run, and
  no drill asserts the simulation's audit is `Clean`. The bar has only
  ever been read through fixtures that carry the typo.
- **The protocol already publishes what the audit is guessing at.**
  `transition` exports the verb constants (`BudgetReserveVerb`,
  `OfferPublishedVerb`, `RunStartedVerb`), `Table.Verbs()` enumerates
  every verb the table defines, and `transition.IsExit(verb)` is the
  deliberate-exit predicate the audit re-implements as its own
  `deliberateExit` map. Three copies of protocol knowledge, one of
  which has already drifted.

## Design decisions (binding for this task)

- **D1 — the bar names the protocol's constant, not a string.** The
  reservation case reads `transition.BudgetReserveVerb`, and the other
  verbs the audit switches on take their constants where the protocol
  publishes one (`OfferPublishedVerb`, `RunStartedVerb`). A rename then
  fails to compile instead of silently mis-counting, which is the
  difference between this defect and its absence.
- **D2 — the deliberate exits come from the protocol.** The audit's
  `deliberateExit` map is replaced by `transition.IsExit`, whose four
  verbs it duplicates exactly today. One authority, so the second copy
  cannot drift the way the first did.
- **D3 — a drill holds every verb the audit names to the table.** The
  audit exports the set of verbs it reads; a drill asserts each is in
  `Table.Verbs()`. That is the guard for the class: it fails on a verb
  the protocol does not define, whatever the reason, and it is what
  would have caught this. A verb the table legitimately does not carry
  is named in the drill with its reason, never skipped silently.
- **D4 — the fixtures move, and the regression is the one the old
  drills could not fail.** `audit_test.go` and the CLI drill's
  histories file `budget.reserve`; a new drill asserts both arms: a
  chain whose run is covered by an admitted `budget.reserve` audits
  clean, and a chain carrying `budget.reserved` reads as unreserved
  spend, naming the subject. Reverting D1 alone must fail that drill.
- **D5 — bounds.** No bar changes meaning, no bar is added or removed,
  and `seed ledger audit`'s envelope keeps its shape and exit codes:
  this card corrects which verb one bar counts and where the audit
  learns its vocabulary. No transition-table change, no admission
  change, no new verb, no CLI flag.

## Steps

1. D1 and D2 in `internal/simulate/audit.go`; the exported verb set D3
   reads.
2. D3's table drill and D4's two-arm regression in
   `internal/simulate/audit_test.go`; the fixtures moved to the
   protocol's verb.
3. The CLI drill's histories in `cmd/seed/ledger_audit_test.go`.
4. `next/docs/progress.md`, `next/docs/decisions.md`,
   `memory/LEARNINGS.md`; receipt; evidence; review.

## File Scope

- `next/internal/simulate/audit.go`, `next/internal/simulate/audit_test.go`
- `next/cmd/seed/ledger_audit_test.go`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-b86dab4c.json`

Nothing else. NOT `next/internal/transition/**`, NOT
`next/internal/simulate/simulate.go`, NOT `next/cmd/seed/ledger.go`,
NOT `next/spec/**`, NOT `.seed/**`.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. **A real chain audits clean.** A history whose `run.started` follows
   an admitted `budget.reserve` leaves `unreserved_spend` empty and
   `clean` true, through `simulate.Audit` and through `seed ledger
   audit`.
2. **The old name is the violation it always was.** A chain carrying
   `budget.reserved` before `run.started` reads as unreserved spend
   naming the subject; reverting D1 fails this drill by name.
3. **The audit's vocabulary is the protocol's.** A drill holds every
   verb the audit reads to `Table.Verbs()` and fails on a planted verb
   the table does not define.
4. `make check` green; no model identifiers in any committed artifact.

**Retention set (existing, shown unharmed):**

- The other four bars keep their meaning and their drills: the
  silent-abandonment, guardrail-breach, chain-violation and
  lost-update arms pass unchanged, as do `seed ledger audit`'s
  envelope drills (`TestLedgerAuditCleanAndEachBar`,
  `TestLedgerAuditVerifiesBeforeAnyBar`,
  `TestLedgerAuditRefusalOrderIsDeterministic`) with their histories
  moved to the protocol's verb.

## Validation Commands

- Boundary: `cd next && go test ./internal/simulate/ -count=1`
- Boundary: `cd next && go test ./cmd/seed/ -run 'LedgerAudit' -count=1`
- Retention: `make check` (exit checked separately from any pipe)

## Expected diff shape

Modified: `audit.go` (the reservation case, the exit predicate, the
exported verb set), `audit_test.go` (fixtures plus two drills),
`ledger_audit_test.go` (the histories), the three docs files, the
receipt. Roughly +70/-20 lines. No other file.
