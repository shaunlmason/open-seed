# Plan: next Phase 1.5 — halt semantics in the validation rule set (os-bce3fb98)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 1
item 5: `halt.declared` / `halt.lifted` as validation rules. Design
authority: [`SEED-NEXT.md`](../SEED-NEXT.md) Part II §1 ("Genesis and
halt"): halt stops admission of every event except an operator's
`system.halt.lifted`, and what makes it real is enforcement at the
admission boundary (Phase 2); this task builds the rule as a pure library
the Phase 2 rule set (`internal/admit`) imports unchanged, the same seam
pattern as classification (1.6). Conformance: III.A halt item (the
boundary drill, including the client-bypassing writer, completes in
Phase 2 with the enforced posture).

## Steps

1. **Halt state as a projection of the chain.** `next/internal/halt`:
   `StateAt(records) State` derives the current halt state (`State`
   carries `Halted`, `By`, and the declaration `Reason`) by replaying `system.halt.declared` / `system.halt.lifted` events
   (payload schemas for both verbs: `reason` for declare, empty for lift;
   subject `system`). No stored flag anywhere: the chain is the only
   source, per the no-second-store rule.
2. **The rule.** `Check(state, proposed) error`: while halted, every verb
   refuses except `system.halt.lifted`; refusal is a typed error carrying
   the halting actor and reason so the envelope layer maps it to a distinct
   exit code (allocated in `next/spec/envelope.md` in this task: code 11,
   `halted`, per the spec's lowest-unused rule; the constant lands in
   `internal/envelope` beside the table).
3. **Operator gating placeholder, honestly scoped.** Whether the *declarer
   or lifter holds operator standing* is a grant check that lands in
   Phase 3; this task documents that boundary in the package comment and
   asserts only verb-shape rules (halt verbs carry subject `system` and
   schema-valid payloads). No fake authorization check.
4. **Tests** (conformance III.A): a chain with `halt.declared` refuses a
   following ordinary event via `Check` and accepts `halt.lifted`; lifting
   restores admission; repeated declare/lift toggles correctly; the typed
   refusal carries actor and reason; `internal/ledger` integration: a
   fixture chain containing a halt window replays green under
   `VerifyFromGenesis` (halt gates *admission of new events*, never the
   validity of admitted history); envelope exit-code table test extended
   for code 7.

## File Scope

- `next/internal/halt/**` (new package)
- `next/internal/envelope/envelope.go` (+ test) — the `halted` exit-code
  constant (7) mirroring the spec table
- `next/spec/envelope.md` — allocate code 7 per the documented rule
- `next/docs/decisions.md`, `next/docs/progress.md`

## Acceptance Criteria

**Boundary set (new, shown working):**

- `StateAt` derives halt state solely from the chain; no flag file or
  second store exists.
- While halted, `Check` refuses every verb except `system.halt.lifted`
  with a typed error carrying the halting actor and the declaration reason
  (both preserved by `StateAt`'s projected state); after a lift,
  ordinary verbs pass again; toggling is idempotent and order-driven.
- Exit code 7 (`halted`) is allocated in `next/spec/envelope.md` and
  mirrored by a tested constant in `internal/envelope`.
- A halt window inside an admitted chain does not fail
  `VerifyFromGenesis` (history stays valid; halt gates admission).

**Retention set (existing, shown unharmed):**

- Phase 1.1/1.2 suites and Phase 0 CLI/envelope suites pass unchanged
  (`cd next && go test ./internal/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the ≥90% coverage gate
  (`make check`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./internal/halt/... ./internal/envelope/... -count=1`
- `cd next && go test ./internal/... ./cmd/... -count=1`
