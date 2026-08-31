# Executors

The executor adapter contract (SEED-NEXT.md §II.9, charter III.H
rows 6–8; plans/os-1dad487d.md): adapters implement **provision**
(workspace + packet), **wake** (advisory channel), and **meter**
(observation-stream usage, settled to the ledger at run end), and
report the **runtime tuple** actually provisioned. The interface is
public — package `next/executor`, the module's one non-internal
package — and the local worktree adapter is the first
implementation; container, cloud-session, and enrolled-remote
adapters are later phases' rows, and real tuple meaning is Phase
10's (the v0 tuple is the honest stub `local-worktree/v0`).

## The spend bracket

No execution path is unmetered, and no run provisions outside the
reservation gate:

1. `budget.reserve` — capacity checked and decremented at admission
   ([`budgets.md`](budgets.md));
2. `run.started` — the **spending-verb table's first entry**: strict
   `{"fence": "<position>", "reservation": "<position>"}`, admitted
   only while the subject is `in_progress`, citing the ACTIVE claim
   fence and an open, **valid** reservation (the budget rule's
   spending gate applies through the table, and the run rule
   revalidates the specific citation position-accurately — the
   laundering shape); once per fence, from {`supervise`,
   `operator`};
3. **Provision** — refuses without the admitted `run.started` the
   spec cites for its fence;
4. **Meter** — usage lines on the per-fence observation stream
   ([`observations.md`](observations.md), the `units` field);
5. `run.settled` — the once-per-fence aggregate: strict
   `{"fence": "<position>", "units": "<non-negative integer>",
   "lines": "<non-negative integer>"}`, admitted only citing an
   applied claim fence (current or prior — a run can settle after
   its window closed) that carries an admitted start, from
   {`supervise`, `operator`}. Telemetry, never authority;
6. `budget.settle` — the authoritative actuals record on the
   reservation ([`budgets.md`](budgets.md)).

The tolerant fold records raw-pushed run facts as independent lists;
a fact citing a fence that is no applied claim position counts an
anomaly, never a fact. Nothing trusts run facts for authority in v0;
Phase 10's qualification applies the position-accurate boundary when
it does.

## The local worktree adapter

Provision creates a **detached git worktree** of the contract
repository at the packet's base revision, writes the handoff packet
to `.seed-run/packet.json` inside the workspace, and ensures the
per-run observation directory. Git runs with fixed argument vectors;
nothing is interpolated into a shell. Wake is the documented no-op
returning nil: the wakeless poll-only drill proved polling loses
only latency, and an advisory channel that does nothing is the
honest v0. Dispose removes the worktree.

## Disposal and the loss window

**Disposal begins only after the last confirmed, admitted
synchronization** — a caller contract, drilled rather than enforced
in code. The disposability drill SIGKILLs a real worker process at a
randomized site after an admitted sync, with no graceful path, and
proves: every admitted fact survives (the chain verifies), the
contract completes elsewhere from the surviving ledger alone, and
the loss is **exactly the observation lines after the last admitted
synchronization** — the declared loss window, stated honestly. The
observation stream is lossy by design ([`observations.md`](observations.md));
anything that must survive rides the ledger.
