# Plan: next — a permission left open makes the doctor's complete unreachable (os-9ef9ab34)

`conformance.Assess` reports `complete: true` only when every
applicable row is `met`; `partial` and `routed` are outstanding and
there is no exemption for a permission. III.B row 6 is a permission,
not an obligation: "Admission **may** shard proposal intake without
changing semantics; ordering remains solely the admitted chain"
(`SEED-NEXT.md`). It sits `open` with the note "not claimed: MAY; the
backlog names sharded intake as a true extra". Those two facts cannot
both stand, and the contradiction is on the promotion critical path:
`plans/os-d63c7441.md` D3 closes Phase 13 only when the doctor reports
`complete: true`, and `next/docs/promotion.md` criterion 6 cites that
section. While the row stays `open`, the doctor can never report
complete, the exit record can never close, and promotion cannot
finish, whatever the shadow run and the cutovers produce. Tier:
standard (it changes a conformance row's status and the rule that
keeps such rows honest). Deps: none.

## What the tree actually shows

- **The rule admits no permissions.** `Assess` sets aside only
  `enforced-only` rows at the cooperative posture; every other row is
  judged, and any status but `met` lands in `Outstanding`, which
  forces `Complete` false (`internal/conformance/conformance.go`). The
  doc comment states the intent: the charter admits a conformance
  claim only when every criterion holds.
- **Exactly one row is permission-shaped.** Of Part III's 128 rows,
  III.B row 6 is the only one whose text says the system *may* do
  something; every other row states an obligation. So the fix is one
  row and the rule that keeps it honest, not a sweep.
- **The row's second clause is already an obligation, and it is met.**
  "ordering remains solely the admitted chain" is III.A row 3's claim,
  `met` since Phase 1 (positions derived from admitted ancestry, no
  writer asserting a sequence number, `#79`, `#85`). The first clause
  binds only a system that shards; this one does not shard, so it
  changes no semantics by construction.
- **No exit record will ever walk it.** The row is Phase 12's; the
  Phase 12 record closed without it, and `plans/os-d63c7441.md` D2
  scopes the exit record's flips to Phase 13's rows. Nothing else in
  the tree flips a row, so the row stays `open` until a card does it.

## Design decisions (binding for this task)

- **D1 — a permission is satisfied by abstention, and the table says
  so.** III.B row 6 becomes `met`, its evidence naming what actually
  holds: intake is single-path (one admission boundary, `internal/admit`
  behind `cmd/seed-admit`, no second intake), and ordering is solely
  the admitted chain (III.A row 3's drills). The note states the
  reading in one sentence: the charter's clause is conditional, a
  system that does not shard satisfies it, and a system that later
  shards must re-earn the row by showing semantics unchanged. The
  charter is not edited; its text is what the table quotes.
- **D2 — the reading is a rule, not a one-off edit.** `Assess` keeps
  its meaning (every applicable row met), and the guard is placed
  where the drift can recur: a drill refuses any row whose status is
  not `met` while its note excuses it as unclaimed or optional. That
  is the exact shape that produced this deadlock — a row parked
  `open` on the grounds that nobody claimed it — and it is what a
  future editor would repeat. The drill names the row and the note.
- **D3 — the packet and the frontier stop calling it an extra.**
  `next/docs/progress.md`'s backlog line for sharded intake
  (os-7953612b) says what it now is: a true extra whose absence
  conforms, not a row the table holds open. No other backlog line
  moves.
- **D4 — bounds.** No change to `Assess`'s completeness rule, to the
  table's schema, to `SEED-NEXT.md`, or to any other row's status.
  `next/docs/generated/conformance.md` is regenerated from the flipped
  table by `seed docs generate`, never hand-edited. This card does not
  implement sharded intake and does not touch III.C row 4, which stays
  `partial` until the weekly scale run is green.

## Steps

1. D1: the row's status, evidence and note in
   `next/spec/conformance.json`; regenerate `conformance.md`.
2. D2: the drill in `internal/conformance/conformance_test.go`.
3. D3: the frontier's backlog line; `next/docs/decisions.md`,
   `memory/LEARNINGS.md`; receipt; evidence; review.

## File Scope

- `next/spec/conformance.json`, `next/docs/generated/conformance.md`
- `next/internal/conformance/conformance_test.go`
- `next/docs/progress.md`, `next/docs/decisions.md`, `memory/*`
- `receipts/os-9ef9ab34.json`

Nothing else. NOT `SEED-NEXT.md`, NOT
`next/internal/conformance/conformance.go`, NOT `next/docs/promotion.md`,
NOT `plans/**`, NOT `.seed/**`.

## Acceptance Criteria

**Boundary set (new, shown working):**

1. **The doctor can reach complete.** With III.B row 6 `met`, the
   outstanding set at the enforced posture holds no row whose only
   reason is that its criterion was never claimed; `seed doctor
   --repo .` lists the remaining rows and each names work or a
   promotion measurement, not an unclaimed permission.
2. **The reading is pinned.** The new drill passes on the committed
   table and fails on a planted row that is `open` with a note
   excusing it as unclaimed or optional, naming that row.
3. **The rendered table follows.** `seed docs check` is clean after
   regeneration; a hand-edited `conformance.md` still fails
   `docs_drift`.
4. `make check` green; no model identifiers in any committed artifact.

**Retention set (existing, shown unharmed):**

- Every other row keeps its status, and the conformance drills pass
  unchanged: `TestTableIsTheCharterRowForRow` (the table still quotes
  the charter verbatim, which this card does not touch),
  `TestTableDriftFromTheCharterIsRefused`, `TestVocabularyHolds`,
  `TestAssessJudgesAtThePosture`, and the doctor's
  `TestDoctorReportsConformanceAtThePosture`.

## Validation Commands

- Boundary: `cd next && go test ./internal/conformance/ ./internal/docs/ -count=1`
- Boundary: `cd next && go test ./cmd/seed/ -run 'Doctor|Docs' -count=1`
- Retention: `make check` (exit checked separately from any pipe)

## Expected diff shape

Modified: one row in `conformance.json` (status, evidence, note), the
regenerated `conformance.md`, one drill in `conformance_test.go`, one
backlog line in `progress.md`, the two record files, the receipt.
Roughly +45/-10 lines. No other file.
