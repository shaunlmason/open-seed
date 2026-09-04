# simulation.md — simulation mode and generated docs

> Status: v0, normative for `next/**`. Authority: [`SEED-NEXT.md`](../../SEED-NEXT.md)
> §II.16, III.O row 5, III.Q row 3, III.R row 7. Plan:
> [`plans/os-16e55c11.md`](../../plans/os-16e55c11.md) (Phase 12 item 6).

## Generated documents

`seed docs generate` renders four documents under
[`../docs/generated/`](../docs/generated/) from the tables the machinery
reads, never from prose: `lifecycle.md` (states and every transition from
`transitions.json`), `capabilities.md` (the affordance catalog's verbs
paired with the keyring's accepted-capability sets), `exit-codes.md` (the
`envelope` package's own `Exit*` and refinement-code constants), and
`lanes/<lane>.md` (each manifest resolved through `lane.Resolve`). The
output is committed; `seed docs check` regenerates and diffs, exit 0 or
`docs_drift` (a refinement of exit 28 `drift`), and `make check-next`
runs it. The generated table is the authority for each enumeration; the
hand-written specs keep their explanations and point at it.

The capability document reads `admit.CatalogVerbs()` — the one verb list
the affordance computation itself drafts from — so a documented verb
cannot diverge from a drafted one.

## Simulation mode

`seed simulate` builds a throwaway deployment under the declared posture,
files synthetic intents from a shipped fix-the-check catalog, and drives
every lane to `done` through the real boundary with a mock executor and
**zero credentials** — no forge, no model, no network beyond a local
bare git remote. It exists so §II.16's claim is a command an adopter runs
rather than a promise.

- **The seam is the CLI's own dispatch** (`loopVerbs`): the simulation
  runs exactly the verbs an operator would. The loop lanes (planner,
  implementer) run through `internal/loop`; the other lanes and roles act
  through their ordinary verbs (`ledger append`, `offer publish`,
  `verdict render`, `merge observe`, `knowledge propose`, `maintain run`).
- **Both postures.** Cooperative validates client-side on the push;
  enforced-self-hosted installs the `seed-admit` pre-receive hook on the
  bare remote. `claim take` is remote-only, so both run on a remote.
- **The executor is `executor.Mock`**: a throwaway workspace, synthetic
  metered usage, a tuple no forge or model backs.

## The accelerated clock

`--days d` spreads the backlog's arrivals across `d` days and advances a
declared reporting instant, threaded into offer expiry and the
maintenance pass's `--as-of`. Admission reads no clock: each event is
stamped with the real wall clock, and the simulated instant feeds only
the clock-reading surfaces. The clock is a declared input, never a fact
the boundary reads.

## The five-bar audit

At the end the run audits III.R's bar from the ledger alone — never from
its own bookkeeping. Each bar names the records that violate it; a clean
run leaves every list empty:

1. **Chain violations** — the admitted chain folds without an illegal
   transition.
2. **Lost updates** — the materialized chain is non-empty and contiguous.
3. **Silent abandonments** — every `in_progress` window ended by one of
   the four deliberate exits.
4. **Guardrail breaches** — every `claim.taken` respects the guardrail
   admission enforces on the claim path: the key that sealed a
   subject's checks never claims it. An unoffered claim is **not** a
   breach, because admission takes one: the scheduling model publishes
   offers (`SEED-NEXT.md` §II.9) but no admission rule reads them, and
   a bar must report what the boundary refuses. The scheduling concern
   (work ready with no live offer) is `internal/eval`'s read, not this
   bar's (plans/os-aaec6a3c.md D1, D3).
5. **Unreserved spend** — every `run.started` sits inside a reservation.

The same audit runs over any ledger through `seed ledger audit
--ledger <dir>` (plans/os-7599c27d.md): the chain is verified from
genesis first, then the five bars are read from the verified records,
a clean chain answering with every list empty and a violated bar
refusing `audit_violated` (exit 28) naming the bar and the records.
This is how the shadow run's real chain is measured against charter
III.R row 5 (`next/docs/promotion.md`).

Each bar's violation, planted once, is caught by name (the drills in
`internal/simulate`).

## Conformance

- §II.16 "the whole system runs end to end against synthetic intents with
  mock executors and zero credentials" — `seed simulate`, `internal/simulate`,
  `executor.Mock`, drilled to `done` under both postures with the network
  and model absent.
- III.O row 5 (second half) "a decider plugs into the trajectory-prefix
  harness" — `internal/decider` and `trajectory replay --decider scripted`
  (see [`trajectories.md`](trajectories.md)).
- III.Q row 3 "docs are governed: operator handbook, generated worker
  docs, stamped design docs" — `seed docs generate`/`check`, the committed
  `generated/`, and [`../docs/handbook.md`](../docs/handbook.md).
- III.R row 7 (adopt from the README in under an hour) — the handbook is
  written against that bar; its commands are drilled.
- Phase 12 exit "the fixture organization runs a week-long simulated
  backlog (accelerated clock) meeting III.R's zero-violation bar" —
  `seed simulate --days 7`, the five-bar audit.
