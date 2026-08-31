# Budgets

The reservation model (SEED-NEXT.md §II.9, charter III.H rows 3–5;
plans/os-cecac5de.md): **budgets are reservations, not
observations**. After-the-fact metering cannot enforce a cap — two
workers can each observe 10 remaining and each spend 8 — so spending
requires an admitted `budget.reserve`, **checked and decremented at
admission**, the one place with a serialized view: the second of two
racing drafts admits against the tip that already carries the first,
and concurrent over-spend against one budget is structurally
impossible. Execution runs fenced against the reservation;
`budget.settle` (or `budget.release`) records actuals from adapter
metering. Where a provider cannot be stopped synchronously, the
budget is honestly a **risk limit, not a guarantee** — declared per
adapter and surfaced when Phase 7.3's adapters land (a named
extension point, like the charter's org/actor budget granularity;
v0 budgets are per-contract).

## Capacity: the class table

A contract carries its budget **class** at filing (`intent.filed`
requires the `budget` field, charter §II.6). The normative class
table, mirrored in code and pinned by test:

| class | capacity (units) |
|---|---|
| `small` | `100` |
| `medium` | `1000` |
| `large` | `10000` |

Units are abstract in v0; Phase 7.3's metering gives them adapter
meaning. A class missing from this table has **zero reservable
capacity**: reserves against it refuse — absent knowledge is never
fudged into spendable units.

## The verbs

All three are facts on the contract subject, admitted only while the
subject is `in_progress` (spend happens inside a claim window), from
the {`claim`, `operator`} capability rows ([`actors.md`](actors.md)).
Strict payloads (unknown fields refuse; the `fence` field rides the
fence rule):

- `budget.reserve` `{"amount": "<positive integer>", "fence": …}` —
  additionally, the drafting signer must be the **active claim
  holder** or the operator lane: a prior claimant reserving under
  the next holder's window would consume a budget it no longer works
  under. Refuses when `amount` exceeds remaining, naming both
  numbers.
- `budget.settle` `{"reservation": "<position>", "actuals":
  "<non-negative integer>", "fence": …}` — records **true** actuals,
  overruns included (an overrun shrinks remaining below what was
  reserved; recorded, never clamped).
- `budget.release` `{"reservation": "<position>", "fence": …}` —
  closes with zero actuals. Both closes must cite a valid, not yet
  effectively closed reservation, and the drafting signer must be
  that reservation's own reserving signer or the operator lane.

## Derivation, not mutation

The tolerant fold records reserves and close attempts as independent
fact lists, raw pushes included; **nothing is mutated and nothing is
erased**. Every consuming surface (the admission computation,
`seed budget status`, the projections) derives, position-accurately:

- a reservation is **valid** only when its signer was the operator
  lane, or held the claim capability AND was the subject's active
  claim holder, at the reservation's own position — so a raw foreign
  reserve consumes no capacity;
- a reservation is **effectively closed** only by its first **valid
  close**: an attempt whose signer is the reservation's own
  reserving signer or the operator lane at the attempt's position —
  so a raw foreign release frees no capacity and a raw foreign
  settle neither closes the reservation nor blocks the owner's later
  settle, which remains the effective closure;
- **remaining** = capacity − Σ open valid reservations − Σ settled
  actuals of valid reservations.

A close citing a position that is no reservation on the subject
folds as an anomaly, never a fact.

## Spending verbs

The charter's "spending verbs require an admitted `budget.reserve`"
lands as a data table of spending verbs. Its first entry is
**`run.started`** ([`executors.md`](executors.md)): execution spend
initiates through it, so no run provisions outside the reservation
gate. The budget rule refuses any listed verb on
a subject with no open valid reservation; the gate is additionally
drilled through test injection in isolation.
Exhaustion produces a structured refusal the worker can act on. The
parking mechanics the park invokes — the `claim.parked` exit with
its packet at a safe point — landed with 7.4, the envelope's
`{reserved, remaining}` block is Phase 8's, and the worker-lane
loop that answers exhaustion by taking that exit is Phase 9 item
1's named obligation, recorded by the Phase 7 exit
(next/docs/progress.md).

## Surfaces

- `seed budget status --ledger <dir> --subject <id>` — the derived
  view: class, capacity (when the class is known), open
  reservations, settled actuals, effective closes, remaining.
  Reserve, settle, and release append through `seed ledger append`
  and the library admission path; refusals reuse the established
  admission exits, no new exit codes.
- Projections ([`projections.md`](projections.md)): the contracts
  view serializes the derived budget object and reservations only
  beside budget facts, so budget-inactive chains keep byte-identical
  bodies; the cache's generation adds a `reservations` table and
  budget columns (`budget_class` always; capacity and remaining only
  beside facts with a known class, NULL otherwise).
