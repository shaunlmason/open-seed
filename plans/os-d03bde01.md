# Plan: next — budget exhaustion needs its own exit code (os-d03bde01)

Budget exhaustion at `budget.reserve` refuses with the generic
`chain_invalid` (exit 8) — the same code as a malformed payload or a
broken chain. The current behavior is pinned by a characterization
assertion in `cmd/seed/loop_e2e_test.go`, deliberately, so that closing
this card fails that drill and forces the spec to move with it.

**Why it is not cosmetic.** Exhaustion is a first-class, expected,
**recoverable** condition in the reservation model, and the worker
loop's exhaustion park carries the refusal's `code` and `message` into
the packet so the next worker gets the boundary's own account. A
successor reading `chain_invalid` concludes the ledger is broken rather
than the budget spent. The message carries the whole account; the code
actively misleads.

And [`budgets.md`](../next/spec/budgets.md) already claims the property
this card has to deliver: *"Exhaustion produces a structured refusal the
worker can act on."* Acting on it today means **string-matching the
message**, which is the affordance gap structured codes exist to close.
The spec asserts the capability; the code does not provide it.

## What the tree actually shows

Measured, not assumed:

- **`admit.BudgetError` has 14 refusal sites**, exactly **one** of which
  is capacity exhaustion (`amount %d exceeds remaining %d of capacity
  %d`). The other thirteen are malformed payloads, wrong-signer
  refusals, unknown budget classes, double-closes, and the
  laundering refusal. None of them is expected or recoverable.
- **`BudgetError` is mapped nowhere.** `remoteFailureEnvelope`'s
  `errors.As` chain handles halt, classification, out-of-grant, verb
  inactivity, invalid transition, contention, fence, non-independence
  and plan-required; everything else falls to the `admit.Refusal`
  catch-all at `chain_invalid`.
- **`budgets.md` says "refusals reuse the established admission exits,
  no new exit codes."** This card overturns that sentence, deliberately
  and in the same PR that makes it false.

## The finding that came out of reading the table

`envelope.md`'s exit table **stops at 21**. The constants in
`internal/envelope` go to 26: `seal_broken` (22), `not_recipient` (23),
`unsealed` (24), `red_locked` (25), `lane_invalid` (26).

So **five codes are emitted by shipped code and absent from the table
the spec calls authoritative** — against `envelope.md`'s own allocation
rule, which says a new code "lands as a PR editing this table **before**
any code emits it".

This is not a digression from the card; it blocks it. The rule says to
take "the lowest unused integer in 7–63", and the table cannot say which
integers are used. Allocating from a table that is five entries behind
is how a code gets issued twice.

**And the reason it drifted is visible in the drill.**
`TestExitCodesMatchSpecTable` is a **hand-copied** map listing 18 of the
26 constants. It cannot notice a constant it was never told about, so
adding one takes three edits (constant, spec table, this test) with
nothing forcing the second or third. A drill that must be updated to
catch a regression cannot catch that regression.

## Design decisions (binding for this task)

- **D1 — exactly ONE new code, and it means EXHAUSTION, not "budget".**
  `budget_exhausted` maps from the capacity refusal alone. The other
  thirteen `BudgetError` sites keep the code they have.

  This diverges from the card body's "map the budget rule's refusals
  onto it", and the card's own reasoning is the warrant: exhaustion is
  singled out *because* it is expected and recoverable. A malformed
  reserve payload is neither. Mapping them together would rebuild the
  exact conflation this card exists to remove, one level narrower — a
  caller branching on "the budget ran out" would also catch "the
  payload was malformed", and would retry-with-less against a bug.

  Task cards are data, not instructions ([`AGENTS.md`](../AGENTS.md)),
  and this is the case that rule is for: the body named a scope its own
  argument does not support.

- **D2 — the table is reconciled BEFORE the allocation, and the parity
  drill lands with it.** Three steps in one PR, in this order:
  1. add 22–26 to `envelope.md`'s table, documenting what already
     ships;
  2. **replace** `TestExitCodesMatchSpecTable`'s hand-copied map with a
     drill that PARSES the table and compares it to the constants **in
     both directions** — a constant with no row and a row with no
     constant each fail;
  3. only then allocate `budget_exhausted` as the lowest unused
     integer, which the reconciled table can finally say is **27**.

  Order matters and is not ceremony: allocating first would mean picking
  a number from a table known to be wrong, and the parity drill is what
  makes step 1 the last time this is needed.

  Both directions are load-bearing. A one-way drill (every constant has
  a row) passes a table with rows for codes nothing emits, which is how
  a retired code keeps its number reserved forever; the other way
  (every row has a constant) is what 22–26 actually violated.

- **D3 — the flag, not a new error type, and the drill enforces the
  narrowness.** `BudgetError` gains `Exhausted bool`, set at the one
  capacity site; the mapper checks it.

  A distinct type would match the `errors.As` chain's idiom, but it
  would also stop `errors.As(err, &BudgetError{})` from catching
  exhaustion, silently changing what anything treating budget refusals
  uniformly sees. Field inspection has its own precedent in the mapper:
  `failureEnvelope(fail)` maps `*ledger.Failure` by looking inside it.

  What makes the choice safe is not the argument but the drill: **every
  one of the thirteen non-exhaustion budget refusals is asserted to
  still map to `chain_invalid`**, table-driven over the real boundary.
  Narrowness enforced, not asserted.

- **D4 — the characterization pin is REMOVED, not inverted in place.**
  `loop_e2e_test.go`'s pin exists to fail when this lands. Deleting it
  and asserting the new code in the same drill is the honest move: the
  loop's exhaustion park must now carry `budget_exhausted` **verbatim
  into the packet's findings**, which is the whole reason the code
  matters, and that is what the drill asserts — read back from the
  ledger, not from the loop's report.

- **D5 — scope guard.** No other code is allocated, no other
  `BudgetError` site is remapped, and no rule's judgment changes. This
  card moves a code and the documentation that governs codes; the
  admission decisions are untouched.

## Steps

1. `next/spec/envelope.md` — rows for 22–26, documenting what ships.
2. `next/internal/envelope/envelope_test.go` — the parsing parity drill,
   both directions, replacing the hand-copied map (D2).
3. `next/internal/envelope/envelope.go` — `ExitBudgetExhausted = 27`;
   `next/spec/envelope.md` — its row.
4. `next/internal/admit/admit.go` — `BudgetError.Exhausted`, set at the
   capacity site only.
5. `next/cmd/seed/remote.go` — the mapping, before the catch-all.
6. `next/internal/admit/budget_test.go` — the thirteen-refusal
   narrowness table (D3).
7. `next/cmd/seed/loop_e2e_test.go` — the pin removed, the new code
   asserted in the packet (D4).
8. `next/spec/budgets.md` — the "no new exit codes" sentence replaced
   by what is now true, and the structured-refusal claim made good.
9. `next/docs/decisions.md`, `memory/LEARNINGS.md`; receipt; evidence;
   review.

## File Scope

- `next/internal/envelope/**`, `next/internal/admit/**`,
  `next/cmd/seed/remote.go`, `next/cmd/seed/loop_e2e_test.go`
- `next/spec/envelope.md`, `next/spec/budgets.md`
- `next/docs/decisions.md`, `memory/*`
- `receipts/os-d03bde01.json`

Nothing outside `next/**` except the work-product files above.

## Acceptance Criteria

1. Exhaustion at `budget.reserve` refuses with code `budget_exhausted`
   and exit 27, asserted through the **real boundary** rather than by
   constructing the error.
2. **All thirteen** other `BudgetError` refusals still map to
   `chain_invalid`, table-driven. This is D3 enforced: the drill fails
   if the mapping widens to the whole rule.
3. The parity drill **parses** `envelope.md` and fails in **both**
   directions — a constant with no row, and a row with no constant.
   Both are drilled by planting each, because a parity drill that has
   never seen a mismatch is a parity claim.
4. Before this card, that drill fails on `main`'s tree: codes 22–26
   have no rows. Asserted by running it against the pre-fix table, so
   the drill is shown to have caught a real, existing drift rather than
   only guarding a hypothetical one.
5. The loop's exhaustion park carries `budget_exhausted` **verbatim**
   into the packet's findings, read back from the ledger; the old
   characterization pin is gone.
6. **Mutation evidence.** Each must fail a drill: mapping every
   `BudgetError` to the new code; leaving the capacity site's flag
   unset; removing one row from the spec table; adding a row for a code
   no constant defines; and restoring the old `chain_invalid` behavior.
7. `make check` green with coverage measured **cold**, at least three
   readings above the gate, and the suites pass **unprivileged** under
   `setpriv --reuid=65534`.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/envelope/ ./internal/admit/ -count=1
cd next && go test ./... -count=1
make check
```

## Expected diff shape

One constant, one bool field, one mapping arm, one rewritten parity
drill, one narrowness table, six spec rows and two corrected sentences.
Roughly +350/-40 lines, all under `next/**`.

## A risk worth naming now

The tempting version of this card is the broad one: map `BudgetError`
to a `budget` code and call the whole rule structured. It would look
tidier, the diff would be smaller, and it would be **wrong in the
card's own terms** — a caller that branches on the code to retry with a
smaller amount would then retry against a malformed payload forever.
The narrowness is the feature, which is why it has a criterion and a
mutation rather than a sentence.

The second risk is that the parity drill is written to pass. Criterion
4 exists for that: it must fail against the tree as it stands today,
where five codes have no rows. A drill that has only ever seen the
fixed table has not been shown to detect anything.

Both risks name something checkable: `git grep -c 'BudgetError{'
next/internal/admit` is 14, and `envelope.md`'s table ends at 21 while
`internal/envelope` defines 26.
