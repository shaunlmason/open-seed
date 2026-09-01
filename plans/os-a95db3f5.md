# Plan: next — the preemption drills wait on the worker, not on the clock (os-a95db3f5)

`TestForcePreemptionDrill` asserts the deaf worker kept metering past
an ignored interrupt by comparing observation-line counts across a
**fixed 300ms sleep**. On a loaded runner under `-p 1` atomic-coverage
instrumentation the helper subprocess can boot slower than that
window, so the count never grows and the drill fails on a system that
is working. Suspected in the open-seed#156 `verify` flake.

## The defect is not the duration, it is what the window is measuring

Raising 300ms to 3s would make the flake rarer without fixing
anything, because the drill would still be asserting *"the worker
metered within N milliseconds"* when what it means is *"the worker
kept metering past the interrupt"*. The file already holds the right
pattern eleven lines up: the graceful path polls for the parked state
against a 15-second deadline at 30ms intervals, and asserts on the
condition rather than on the clock.

## The same window hides a second, quieter bug

Two fixed windows sit in this file, and both sample the world before
establishing that there is a worker to sample:

| Line | Window | Asserts | Failure mode |
| --- | --- | --- | --- |
| ~318 | 300ms | line count **grew** | a slow boot fails a working system (the filed flake) |
| ~221 | 250ms | a raw unprivileged interrupt parked **no one** | a slow boot **passes vacuously**: nothing parked because nothing was running |

The second is worse in kind, because a negative assertion that
succeeds for the wrong reason never fails and never gets filed. It is
the same root cause — a fixed window standing in for "the worker is
up" — found by the same reasoning, in the same file, so this task
fixes both and says so rather than leaving a known vacuous assertion
for the next card.

## Design decisions (binding for this task)

- **D1 — establish the worker is up before asserting anything about
  it.** A bounded poll on the observation stream for the first line
  keyed to the actor and fence. Every later assertion then rests on a
  live worker, which is what both windows were silently assuming.
- **D2 — the positive assertion polls for growth, not for time.**
  Sample the count *after* the interrupt lands, then poll until it
  exceeds that sample or a generous deadline expires. The deadline is
  a failure bound, never a pacing device: on a fast runner the drill
  returns in one interval.
- **D3 — the negative assertion keeps a settle window, now anchored.**
  You cannot poll for something failing to happen, so the raw-interrupt
  case keeps a bounded wait — but it runs against a worker D1 has
  already proven live, and it additionally asserts the stream **kept
  growing** across the window. "Still metering and still `in_progress`"
  is the positive form of "parked no one", and it cannot pass
  vacuously.
- **D4 — one shared helper, matching the file's existing posture.**
  A single poll-until helper used by both sites and shaped like the
  15-second/30ms loop already in the file, so a future drill copies the
  right pattern rather than the wrong one.
- **D5 — no production change, and no change to what the drills
  prove.** Same scenarios, same subprocess, same verbs, same ordering.
  The force path still kills, reaps with a packet composed from what
  is known, and completes elsewhere. Only the waits change — and D3
  makes one assertion strictly stronger than it was.

## Steps

1. Add the poll-until helper to `next/cmd/seed/preempt_cli_test.go`,
   shaped like the existing parked-state loop (bounded deadline, short
   interval, condition-driven).
2. `TestForcePreemptionDrill`: wait for the worker's first observation
   (D1), land the interrupt, sample, then poll for growth (D2).
3. The raw-unprivileged-interrupt case: wait for the worker to be up
   (D1), then assert both that the subject stayed `in_progress` and
   that the stream kept growing across the window (D3).
4. Run the package repeatedly and under load
   (`go test ./cmd/seed/ -run 'Preempt' -count=10`, and once with
   `-p 1` coverage instrumentation as CI runs it) to confirm the flake
   is gone and nothing else changed.
5. `memory/LEARNINGS.md` if the fix teaches anything durable; receipt;
   evidence; review.

## File Scope

- `next/cmd/seed/preempt_cli_test.go`
- `memory/*` if warranted
- `receipts/os-a95db3f5.json`

## Acceptance Criteria

1. No fixed sleep in `preempt_cli_test.go` stands in for "the worker
   is up" or "the worker made progress"; both are established by
   polling a condition against a bounded deadline.
2. The force drill establishes the worker is live before the
   interrupt, and asserts growth after it by polling rather than by
   elapsed time.
3. The raw-interrupt case can no longer pass vacuously: it asserts a
   live, still-metering worker alongside the unchanged state.
4. No production file changes; the scenarios, verbs and orderings the
   drills exercise are unchanged.
5. `go test ./cmd/seed/ -run 'Preempt' -count=10` green, and green
   once under the `-p 1` coverage instrumentation CI uses.
6. `make check` green, coverage gate ≥90% held.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./cmd/seed/ -run 'Preempt' -count=10
make check
```

## Expected diff shape

One helper and two rewritten waits in a single test file. Roughly
+45/-12 lines, all in `next/cmd/seed/preempt_cli_test.go`.
