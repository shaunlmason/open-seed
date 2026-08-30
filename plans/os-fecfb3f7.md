# Plan: next Phase 4.2 — standard projections (os-fecfb3f7)

Implements [`docs/next-build-plan.md`](../docs/next-build-plan.md) Phase 4
item 2: the standard projections over the 4.1 engine seam. Design
authority: [`SEED-NEXT.md`](../SEED-NEXT.md) Part II §4 ("Standard
projections: the ready queue (filtered by actor eligibility), contract
detail, actor view, report, …") and conformance III.D — the
deterministic/stamped/rebuildable rows extended to every new view, and
"Staleness is visible everywhere projected state is shown; **consumers
can demand a minimum position**", which this task makes enforceable by
test for the first time. Deps: Phase 4 item 1 (plan #105; implementation
on `seed/os-4d5cacff`) — implementation stacks on that branch until it
merges and registers its views in the engine's registry.

## Scope decision (binding for this task)

Phase 4 item 2 names four views. At Phase 4 the chain's vocabulary is
governance (`system.*`, `actor.*`) plus free-form subject-bearing work
events; **readiness is not derivable until Phase 5's transition table**
(`next/spec/transitions.json`, Phase 5 item 1), and eligibility
filtering is deferred past Phase 5 by the build plan's own parenthetical.
So:

- **Ships now**: contract detail (v0), the ready queue (registered,
  schema-fixed, v0 derivation), actor view, report (skeleton), and the
  `seed project current` consumer verb (the min-position demand).
- **Deferred**: only the queue's **readiness derivation** (Phase 5
  item 1, where the transition table first defines claimable states)
  and, per the parenthetical, its eligibility filter (after Phase 5).
  The projection itself ships now — registered, stamped, drilled —
  so Phase 4 exits with the complete standard surface and Phase 5
  swaps one derivation function rather than adding registration,
  layout, spec, and drills after the fact (review finding on #106).
  Under the Phase 4 vocabulary no readiness derivation exists, so the
  v0 queue is **empty by definition and says so machine-readably**
  (`derivation: "none"` in the view, mirrored in the spec) instead of
  pretending a derivation it does not have; the handoff is recorded in
  `next/docs/decisions.md` and `next/docs/progress.md` so Phase 5
  inherits the swap explicitly.
- **Work-verb classifier (v0)**: events whose verb carries the
  `system.` or `actor.` prefix are governance vocabulary; every other
  event is work vocabulary and contributes to contract detail, keyed by
  subject. Phase 5's transition table replaces this prefix rule with
  explicit vocabulary; the rule is documented in the spec and the
  decision log.

## Steps

1. **Contract detail** (`contracts.json`): every work subject in
   first-appearance order — `{subject, first_position, last_position,
   events: [{position, verb, actor, payload}]}` with the signer
   fingerprint as `actor` and the canonical payload verbatim. No
   `state` field exists yet; the spec documents that state derivation
   arrives with Phase 5's transition table rather than reserving a
   field that would always read null. One file, not per-subject files:
   subjects are opaque strings, and mapping them to paths would trade
   the engine's path-safety refusals for an encoding scheme nothing
   consumes yet; the 4.3 cache is the lookup-throughput answer.
2. **Ready queue** (`queue.json`): the registered claimable-work
   surface, schema fixed now — `{schema_version: "1", derivation,
   ready: […]}` where entries carry at least `{subject,
   since_position}` and the field set is Phase 5's to extend. The v0
   derivation is `"none"`: the Phase 4 vocabulary defines no claimable
   states, so `ready` is empty by definition and the `derivation`
   field says so machine-readably — a consumer can refuse to treat an
   underived queue as meaning "nothing to do". Phase 5 item 1 replaces
   the derivation (and its marker) with the transition table's; the
   eligibility filter follows later, per the build plan.
3. **Actor view** (`actors.json`): the per-actor drill-down the roster
   summarizes — for every roster candidate (genesis roots + enrollment
   subjects): the roster fields, plus `standing_history` (each
   `actor.*` event on this subject: position, verb, acting signer),
   `grants` accumulated, and `signed` (position, verb, subject of every
   record this fingerprint signed — the attribution surface, which is
   how the view shows a revoked key's history surviving revocation).
4. **Report skeleton** (`report.json`): the operational summary whose
   sections later phases extend — `chain` (position, tip, active
   version), `actors` (counts by standing, root count), `halt` (halted
   flag; declaring position when halted), `checkpoints` (count, last
   position), `contracts` (subject and event counts). Sections that
   need Phase 5+ facts (claims, offers, budgets, expiry-vs-wedge,
   divergence) are named in the spec as extension points, not emitted
   empty.
5. **The consumer verb.** `seed project current --out <dir> --name
   <projection> [--min-position N]`: resolves `CURRENT`, reads the
   build's stamp, reports `{name, position, tip, path}` with the
   envelope position carrying the stamp's count verbatim (the
   count-vs-index convention is stated in the spec; #105 lesson). No
   `--ledger` flag exists: the verb is structurally a consumer and
   cannot touch authoritative state. Refusals: unknown projection or no
   published build → exit 4 `not_found`; a stamp position below
   `--min-position` → **exit 15 `stale`** (new; spec row lands in
   `next/spec/envelope.md` before the constant, per the allocation
   rule), naming current and demanded positions — the III.D demand
   made scriptable.
6. **Spec.** Extend `next/spec/projections.md`: the four views'
   schemas and derivations (work-verb classifier and the queue's
   `derivation` marker included), the queue's Phase 5 derivation
   handoff, the consumer verb and staleness semantics, and the
   conformance-mapping update. Add the exit-15 row to
   `next/spec/envelope.md`.
7. **Drills** (library + CLI): the engine's byte-identical
   rebuild/stamp/immutability drills now run over the full default
   registry (all five projections: the 4.1 roster plus these four).
   Contract detail: two interleaved work subjects with governance
   events excluded; payloads carried verbatim; empty ledger yields an
   empty array, not a missing file. Ready queue: present, stamped,
   `derivation: "none"`, `ready` empty across every Phase 4 fixture
   (populated and root-only alike) — the drill that Phase 5 must
   replace when it swaps the derivation. Actor view: across the
   Phase 3 lifecycle fixtures
   (enroll/grant/suspend/re-enroll/revoke/rotation) — standing history
   and signed attribution correct, a revoked actor present with
   history intact. Report: counts across the same fixtures; a halted
   chain reports the declaring position; root-only ledger reports
   zero contracts and the genesis roots. Consumer verb end-to-end:
   resolves after rebuild; `--min-position` at the stamp passes, one
   above refuses 15 naming both numbers; unknown name refuses 4;
   refusals write nothing.

## File Scope

- `next/internal/project/**` (the four builders + tests)
- `next/cmd/seed/**` (the `project current` subverb + tests)
- `next/spec/projections.md`, `next/spec/envelope.md` (extend)
- `next/internal/envelope/**` (the exit-15 constant)
- `next/docs/decisions.md`, `next/docs/progress.md`,
  `memory/LEARNINGS.md` (if lessons emerge)

## Acceptance Criteria

**Boundary set (new, shown working):**

- One rebuild publishes all five registered projections (the 4.1
  roster plus the four standard views), each stamped with the
  verification report's position and tip; deletion then rebuild is
  byte-identical across the whole output tree.
- Contract detail carries exactly the work-vocabulary events, grouped
  by subject in first-appearance order, payloads verbatim; governance
  events appear in no contract entry.
- The ready queue is present and stamped on every fixture with
  `derivation: "none"` and an empty `ready` array — never absent, and
  never claiming a derivation the vocabulary cannot support.
- Actor view preserves a revoked actor's full history and attribution
  while the roster shows the ended standing; histories match the
  fixture chain position-for-position.
- The report's counts equal the fixture chain's facts (actors by
  standing, halt position, checkpoint count, contract counts) on both
  populated and root-only ledgers.
- `seed project current` resolves only complete published builds;
  demanding a position above the stamp refuses with exit 15 naming
  both positions; unknown projections refuse with exit 4; neither
  refusal creates or modifies anything.

**Retention set (existing, shown unharmed):**

- All existing verb suites pass unchanged
  (`cd next && go test ./internal/... ./cmd/... -count=1`).
- The repo-wide gate stays green with the ≥90% coverage gate
  (`make check`).

## Validation Commands

- `make check-next`
- `make check`
- `cd next && go test ./internal/project/... -count=1`
- `cd next && go test ./internal/... ./cmd/... -count=1`
