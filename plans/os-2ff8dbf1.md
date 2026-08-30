# Plan: next Phase 5.6 — observation streams v0, monotonic progress, expiry vs. wedge (os-2ff8dbf1)

Authority: docs/next-build-plan.md Phase 5 item 6: "Observation
streams v0 + monotonic progress counts; expiry vs. wedge detection in
the report." Charter (SEED-NEXT.md Part II §5): observations
(liveness, progress, fine metering) are **ephemeral streams** on a
non-authoritative channel, *summarized* into ledger facts only at
material transitions (`claim.taken`, `progress.milestone` — coarse,
bounded frequency — `wedge.declared`, `run.settled`, park/reap);
progress is measured by **monotonic progress counts**, never file
modification time; long-running steps carry a **declared in-step
state**; "expiry (no observations) and wedging (observations without
progress advancement) are distinct, visible conditions"; "Observation
channels are lossy by declaration: losing every observation stream
loses no coordination state." Conformance III.F rows: traffic
classification; monotonic counts + material-transitions-only;
expiry-vs-wedge distinct with distinct reap heuristics, drilled;
lossy-by-declaration. Fixed default (build plan §1): the v0 channel
is a per-executor file under `next/var/obs/` (gitignored), optional
`refs/seed/obs/<actor>` push. Catalog vocabulary used:
`progress.milestone`, `wedge.declared` (a `claim.*` catalog entry),
`run.settled` named-not-implemented (Phase 7 budgets own metering).
No other verbs are invented.

## Design decisions (binding for this task)

- **The channel is exactly the default's file, one JSONL line per
  observation.** `next/var/obs/<actor-fingerprint>.jsonl`, appended
  by the executor, gitignored. Line shape:
  `{"ts": "<RFC3339>", "subject": "<id>", "count": <int>, "step":
  "<declared in-step state>"}` — `count` is the monotonic
  completed-item counter, `step` the charter's declared in-step
  state for long-running work. Unsigned: the stream is
  non-authoritative by construction and nothing downstream trusts
  it for a decision, so signature cost buys nothing (the ledger
  facts stay signed as always). The optional per-actor ref push from
  the defaults table is **deferred, named in the spec**: the file
  channel satisfies v0 and the lossy contract makes the second
  channel additive.
- **Detection is a pure function of declared inputs, not wall
  clock.** Expiry and wedging are ages, and a projection must stay
  a deterministic function of its inputs, so the report build takes
  a **declared observation snapshot** plus a declared `as_of`
  instant, and classifies each `in_progress` subject:
  **live** (an observation within `expiry_after` and the count
  advanced within `wedge_after`), **expired** (no observation within
  `expiry_after` — the no-observations condition), **wedged**
  (observations continue within `expiry_after` but the count last
  advanced more than `wedge_after` ago — observations without
  progress advancement). Distinct conditions, distinct fields,
  distinct drill cases; the distinct **reap heuristics** are stated
  in the spec beside them (expiry reaps after grace on the lease,
  wedging reaps on operator/maintenance judgment with the packet
  capturing the wedge), while executing reaps stays the maintenance
  lane's later item — v0 makes the conditions visible where the
  build plan puts them: the report.
- **Thresholds are declared inputs with spec'd defaults.**
  `expiry_after` 900s, `wedge_after` 1800s — stated in the spec as
  v0 operational defaults, overridable at the rebuild call; both
  values ride INSIDE the report's observation section together with
  `as_of` and a sha256 digest of the snapshot bytes, so a view is
  self-describing about the inputs that shaped it and byte-identical
  for identical declared inputs. The projection stamp shape does not
  change (the cache stamp-table parity stays untouched).
- **The engine seam grows an optional declared-inputs argument; the
  report's Version bumps.** Builders today are functions of the
  record prefix; the report becomes a function of (records, declared
  observation inputs), which is exactly the charter's "deterministic
  function of a ledger prefix (+ declared observation inputs)"
  clause. The registry keeps a single builder signature by passing
  an inputs value that is empty by default — with no declared
  inputs the observation section reports `"inputs": null` and
  classifies nothing (absence of data is stated, never fabricated).
  A derivation change republishes under a new build id by version
  bump: the report's Version goes to "2", the 4.x
  version-in-identity machinery does the rest, and the cache mirrors
  the section like every report fact. Byte-identical drills run
  both ways (no inputs; fixed inputs).
- **`progress.milestone` is admitted, coarse, and monotonic in the
  ledger too.** Payload `{"count": <int>, "step": "<state>"}` with
  admission enforcing: count strictly greater than the subject's
  last admitted milestone count (the monotonic rule applied at the
  summarization boundary), and **bounded frequency** as a minimum
  spacing of 300 seconds between milestones per subject, computed
  from event timestamps (deterministic over chain content; the
  value is a spec'd v0 default). Capability row {`claim`,
  `operator`}; on a claimed subject the 5.2 citation matrix applies
  (active fence cited). No transition row: a milestone is a fact on
  an `in_progress` subject, and the 5.1 pinned invariant (four
  `in_progress` exits) stands.
- **`wedge.declared` is an operator-lane fact, not a transition.**
  It records the visible wedge condition durably (typically after
  the report surfaced it), capability {`operator`} in v0 (the
  maintenance lane inherits it later, the merge.observed posture),
  payload `{"observed": "<as_of>", "count": <int>, "since":
  "<last-advance ts>"}` — presence-enforced. The claim exit remains
  `claim.reaped` (or a deliberate exit) exactly as 5.1/5.3 bound
  them, packet included; wedge.declared changes no state.
- **Lossy by declaration is drilled literally.** Delete every file
  under `next/var/obs/`: every admission decision, fold state,
  queue, and lifecycle drill is byte-for-byte unaffected (no
  coordination path reads the channel), and a rebuild without
  declared inputs publishes a report that says so instead of
  erroring. That is the conformance row's sentence made a test.
- **Writer verb.** `seed obs emit --dir <obs-dir> --actor <fp>
  --subject <id> --count <n> --step <s>` appends one line (creating
  the per-actor file), plus `--ts` for drills; no reader daemon —
  the supervisor/tailer is the maintenance lane's item, and v0
  readers are the report build and tests.
- **Out of scope, named.** The refs-push channel; the
  supervisor/tailing loop; reap execution; `run.settled` (Phase 7
  metering); the contention benchmark row (it rides the 5.2 race
  storm lineage, tracked at the CI phase); signing of observation
  lines.

## Steps

1. **Spec** (`next/spec/observations.md`, new): the traffic
   classification, the channel and line shape, monotonic counts and
   the declared in-step state, the summarization verbs with payloads,
   frequency and monotonicity admission rules, expiry/wedge
   definitions with the distinct reap heuristics stated, thresholds
   and their defaults, lossy-by-declaration, deferred items;
   conformance mapping quoting the III.F rows. Cross-row edits:
   `next/spec/projections.md` (report section + declared-inputs
   clause + Version "2"), `next/spec/actors.md` (capability rows),
   `next/spec/lifecycle.md` (facts-not-transitions note).
2. **Observation library** (`next/internal/obs`, new): line
   encode/decode, per-actor append, snapshot load (directory →
   per-actor streams), snapshot digest; pure classification
   `Classify(claim, stream, asOf, thresholds)` returning
   live/expired/wedged with the evidencing fields.
3. **Engine seam + report v2** (`next/internal/project`): the
   declared-inputs argument (empty default), report builder consumes
   it, Version "1"→"2", observation section (inputs echo + per
   in_progress subject classification), cache mirrors the section.
4. **Verbs** (`next/cmd/seed`): `seed obs emit`; admission rules for
   `progress.milestone` (monotonic count, 300s spacing) and
   `wedge.declared` (payload presence) in `next/internal/admit`;
   capability rows in `keyring.AcceptedCapabilities`.
5. **Drills**: monotonic-count refusal (equal and lower counts);
   spacing refusal at 299s, admission at 300s; milestone under the
   citation matrix; wedge.declared operator-gating; classification
   truth table (live/expired/wedged/no-data) over fixture streams
   at a fixed as_of; byte-identical report builds with and without
   identical declared inputs; version-bump republish under the new
   id; the lossy deletion drill; spec-parsing vocabulary test rows.
6. **Docs**: progress row, decisions entry, LEARNINGS if any.

## File Scope

- `next/spec/observations.md` (new), `next/spec/projections.md`,
  `next/spec/actors.md`, `next/spec/lifecycle.md`
- `next/internal/obs/**` (new)
- `next/internal/project/**`, `next/internal/admit/**`,
  `next/internal/keyring/**`
- `next/cmd/seed/**`
- `.gitignore` (the `next/var/obs/` entry)
- `next/docs/progress.md`, `next/docs/decisions.md`,
  `memory/LEARNINGS.md`

Never: `SEED-NEXT.md`, v1 surfaces, `plans/**` in the task PR.

## Acceptance Criteria

**Boundary set (new, shown working):**

- `seed obs emit` appends well-formed lines; snapshot load and
  digest round-trip.
- The report classifies live, expired, and wedged as three distinct
  conditions from declared inputs, and reports declared-inputs
  absence honestly; identical inputs rebuild byte-identically;
  report Version "2" republishes under a new build id.
- `progress.milestone` refuses non-advancing counts and sub-300s
  spacing, admits otherwise under the claim lane with fence
  citation; `wedge.declared` is operator-gated with presence-checked
  payload.
- Deleting the whole observation directory changes no admission
  decision, no fold state, and no drill outcome outside the report's
  observation section.

**Retention set (existing, shown unharmed):**

- All 5.1–5.5 admission, fence, packet, acceptance, and plan-gate
  drills stay green.
- Projection suite: existing five-view + cache equivalence drills
  green; stamp shapes unchanged; the no-inputs report build stays
  byte-identical across rebuilds.
- `make check` green at the coverage gate.

## Validation Commands

- Boundary: `cd next && go test ./internal/obs/... ./internal/project/... ./internal/admit/... ./cmd/seed/... -count=1`
- Retention: `cd next && go test ./... -count=1` and `make check`
  (exit checked separately from any pipe).

## Expected diff shape

Two new packages (obs library, spec file) with tests; report builder
extension plus version bump; admission and keyring rows with drills;
one CLI verb; a .gitignore line; docs. No deletions of existing
tests; no stamp-shape change; no v1-surface edits; no `plans/**` in
the implementation PR.
