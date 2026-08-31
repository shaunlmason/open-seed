# Plan: next Phase 7.3 — executor adapter + local worktree (os-1dad487d)

Implements `docs/next-build-plan.md` Phase 7 item 3: the executor
adapter interface (provision / wake / meter / report-tuple), the
**local worktree adapter** first, metering to the observation stream,
and the `run.settled` aggregate. Design authority: SEED-NEXT.md §II.9
("Executor adapters implement provision (workspace + packet), wake
(advisory channel), and meter (observation-stream usage, settled to
the ledger at run end), and report the runtime tuple actually
provisioned. … Disposal begins only after the last confirmed,
admitted synchronization"), charter III.H rows 6–8 (the adapter
interface is public; metering flows on observation channels and
settles to the ledger at run end, no execution path unmetered;
disposability after the last admitted sync is drilled with randomized
kills, the loss window bounded by declared sync policy and stated
honestly), the observation-channel binding default (per-executor
files under `next/var/obs/`, the `observations.md` machinery Phase 5
built), and the Phase 7 exit line, whose one remaining drill this
task lands: the **disposability drill** (randomized kill after sync;
complete elsewhere) — the poll-only run landed with 7.1 and the
reservation race with 7.2. Container, cloud-session, and
enrolled-remote adapters are later phases' rows; graceful preemption
is 7.4's; real runtime tuples are Phase 10's.

## Design decisions (binding for this task)

- **D1 — the public interface is a public package.** Charter III.H
  says "the adapter interface is public": it lands as
  `next/executor`, the module's one non-internal package, so the
  interface survives spin-out importable. Two small interfaces:
  `Adapter` (Provision(spec) (Run, error); Wake(actor) error —
  advisory only, its total failure costs latency; Tuple() Tuple —
  the provisioned runtime tuple) and `Run` (Workspace() string;
  Meter(units int, step string) error — usage onto the obs stream;
  Dispose() error). Dispose's contract is documented, not enforced
  in code: the CALLER disposes only after the last confirmed,
  admitted synchronization, and the drill proves that discipline
  loses nothing admitted. `Tuple` is the honest v0 stub
  (`{"runtime": "local-worktree/v0"}`): Phase 10 gives tuples
  meaning; nothing here pretends qualification exists yet.
- **D2 — the local worktree adapter.** Provision creates a detached
  git worktree of the contract repository at the packet's base,
  writes the handoff packet to a well-known file in the workspace
  (`.seed-run/packet.json`), and ensures the per-run observation
  stream directory; Wake is a documented no-op returning nil (the
  wakeless drill already proved polling suffices — an advisory
  channel that does nothing is the honest v0); Meter appends
  observation lines carrying the new optional `units` field;
  Dispose removes the worktree. No shell interpolation anywhere: the
  worktree commands take fixed argument vectors.
- **D3 — metering rides the existing channel.** `obs.Line` gains
  `Units int` with omitempty (additive: existing streams and
  snapshot digests stay byte-identical); a metering line is an
  ordinary line with units set. `run.settled` is the aggregate at
  run end: strict payload `{"fence": "<position>", "units":
  "<non-negative integer>", "lines": "<non-negative integer>"}`,
  where `fence` names the run's claim fence and doubles as the
  fence rule's citation — one field, no duplication. Admitted only
  on subjects where the cited fence is a real applied claim.taken
  position (current or prior — a run can settle after its claim
  window closed), from {supervise, operator}
  (adapter-side summarization is the supervisor lane's act, like
  offers). One run.settled per fence: a second refuses — one run,
  one aggregate. Units sum with the 7.2 saturating arithmetic.
  Budget linkage stays honest and separate: `budget.settle` remains
  the authoritative actuals record on the reservation;
  `run.settled` is telemetry aggregate, grants nothing, gates
  nothing, and the spending-verb table stays empty (execution spend
  is not a ledger verb; the table's first fill waits for a verb
  that actually initiates spend).
- **D4 — the laundering shape.** The tolerant fold records
  well-shaped `run.settled` facts as
  `RunFact{Pos, Signer, Fence, Units, Lines}` on the subject (raw
  pushes included; a settle citing a fence that is no claim
  position on the subject folds as an anomaly, the dangling-close
  posture). The consuming surface — the contracts view's runs list
  — marks nothing as authority: run facts carry telemetry, and
  every surface that would trust one (none in v0; Phase 10
  qualification will) must apply the position-accurate boundary
  then. The one-per-fence rule is enforced at admission and derived
  tolerantly in the view (first fact per fence shown as the
  aggregate; later raw duplicates visible as facts, counted
  anomalies).
- **D5 — the disposability drill** (the Phase 7 exit's remaining
  named drill; conformance III.H). End-to-end test: file, specify,
  claim; provision a local worktree run; the worker emits metering
  and lands an admitted synchronization (submission.made); the
  adapter is killed at a randomized point AFTER the sync (seeded
  `math/rand`, seed logged, choosing among kill sites: before
  run.settled, after run.settled, mid-meter) by disposing the
  workspace without any graceful path; then the drill proves:
  every admitted fact is intact (the chain verifies), the
  contract completes elsewhere (the review flows on — verdict,
  request, observed — driven from the surviving ledger alone), and
  the only loss is observation lines after the last admitted sync
  (asserted by comparing the stream against the admitted facts,
  and stated as the loss window in the spec). Workspace removal
  loses nothing admitted, whatever the kill site.
- **D6 — surfaces and versions.** No new CLI subcommand: adapters
  are library machinery for the supervisor loop (the charter's
  CLI-completeness is about ledger verbs, and `run.settled` rides
  `seed ledger append` behind its admission rule). New
  `next/spec/executors.md` (the interface contract, the worktree
  adapter, disposal discipline, the loss window, wake advisory
  semantics); `observations.md` gains the units field;
  `lifecycle.md` fact-window sentence and `actors.md` capability
  row ({supervise, operator}); the catalog already lists `run.*`.
  Projections: contracts "11" (runs list, omitempty — run-free
  chains keep byte-identical "10" bodies), report "8" (republish
  only), cache generation 10 (a `runs` table).

## Steps

1. **Spec.** `next/spec/executors.md` per D1/D2/D5 (interface,
   worktree semantics, disposal-after-sync, the honest loss window,
   wake advisory posture, v0 tuple stub); `observations.md` units
   field; `actors.md` row; `lifecycle.md` sentence.
2. **Obs.** `Units int` omitempty on `obs.Line`; digest stability
   pinned by existing tests.
3. **Fold.** `RunFact` list on SubjectState; capture with the
   dangling-fence anomaly; saturating unit sums where summed.
4. **Admission.** A `run` rule (after budget): verb-gated on
   `run.settled`; strict payload; the cited fence must be an
   applied claim position on the subject; one per fence; units and
   lines non-negative integers. Capability rides the grant rule.
5. **The public package.** `next/executor`: `Adapter`, `Run`,
   `Tuple`, `ProvisionSpec` (repo, base, subject, actor, fence,
   packet bytes, obs dir), and the local worktree implementation
   (git worktree add/remove with fixed argv, packet file, obs
   wiring via `internal/obs`).
6. **Drills**: the **disposability drill** per D5 with its
   `// conformance: III.H` comment; adapter unit drills (provision
   writes packet + workspace, meter lines land with units and sum
   in run.settled, dispose removes, wake no-op, tuple stub);
   admission drills (lanes, shape, dangling fence, one-per-fence,
   prior-fence settles admit); fold drills (tolerant capture,
   anomaly, raw duplicate); projection coverage (runs in the view
   and cache, run-free byte-identity via the untouched goldens).
7. **Projections.** Contracts "11", report "8", cache generation 10
   with version-pin test updates.
8. **Docs.** `next/docs/progress.md` 7.3 row and frontier;
   `next/docs/decisions.md` one dated entry for D1–D6;
   `memory/LEARNINGS.md` only if implementation surfaces a durable
   insight.

## File Scope

- `next/spec/executors.md` (new), `next/spec/observations.md`,
  `next/spec/actors.md`, `next/spec/lifecycle.md`
- `next/executor/` (new public package + tests)
- `next/internal/obs/obs.go` (+ tests)
- `next/internal/keyring/keyring.go` (+ tests)
- `next/internal/transition/transition.go` (+ tests)
- `next/internal/admit/admit.go` (+ tests)
- `next/internal/project/contracts.go`, `report.go`, `cache.go`
  (+ fixtures/tests)
- `next/docs/progress.md`, `next/docs/decisions.md`,
  `memory/LEARNINGS.md` (conditional)

## Acceptance Criteria

**Boundary set (new, shown working):**

- The disposability drill: after an admitted sync, a randomized
  kill (logged seed) at any site loses no admitted fact, the
  contract completes elsewhere from the surviving ledger alone, and
  the loss is exactly the observation lines after the last sync.
- The local worktree adapter provisions a workspace with the packet
  at `.seed-run/packet.json`, meters units onto the per-fence
  observation stream, reports the v0 tuple stub, and disposes
  cleanly; wake is a nil no-op.
- `run.settled` admits from {supervise, operator} citing a real
  claim fence (current or prior), once per fence, with non-negative
  integer units and lines; refuses dangling fences, duplicates, and
  malformed payloads by name; raw-pushed facts fold tolerantly with
  dangling citations counted as anomalies.
- Contracts "11" serializes the runs list omitempty; run-free
  chains keep byte-identical "10" bodies (the untouched goldens);
  the cache's `runs` table agrees.

**Retention set (existing, shown unharmed):**

- `make check` green; coverage ≥90%; every earlier drill (offers,
  budgets, lockout, seals, verdicts, reconcile, obs) passes
  unmodified; existing observation-stream digests unchanged.
- No new exit codes; no v1 surface changes; the task PR never
  touches `plans/**`.

## Validation Commands

- Boundary: `cd next && go test ./executor/... ./... -run 'Run|Executor|Disposab|Worktree' -count=1`
- Retention: `make check`

## Expected diff shape

One new spec file, one new public package with tests, and targeted
additions to obs, keyring, transition, admit, and the three
projection files with fixtures; two docs files. No deletions, no
`.seed/**`, no `plans/**` in the task PR.
