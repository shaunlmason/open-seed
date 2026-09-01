# Plan: next — the coverage gate must not report a number it cannot trust (os-cafba959)

`make check`'s coverage gate fails nondeterministically, reporting a
total far below truth on a tree that is fine: 73.1%, 62.2%, 61.5%,
87.8%, on trees that read 90.9% on the next cold run with no change at
all. **Every test passes in the bad runs**; only the merged profile is
short.

It presents as *"your change dropped coverage 28 points"*, which is
exactly what a real regression looks like. This card exists because the
correct response — re-run cold once, then treat a second failure as
real — is a rule an unattended agent has to apply **against its own
instinct**, and a rule like that is better held by the tool.

This card also closes `os-4eaf8b13`, which is the same defect's
diagnostic half: `go test` caches a package's coverage contribution, so
a warm re-run **replays the lost profile at the same number**, and the
flake therefore looks deterministic. That finding is not a footnote
here: it is why the re-collection this plan adds must be cold, and a
drill pins it.

## What was established before planning

Measured on this tree, not assumed.

- **The merged profile is a concatenation of per-package fragments.**
  A full run writes 28 `_cover_.out` files totalling 80,444 body lines;
  the merged `-coverprofile` output holds 241,332 lines in 84
  monotonic runs. Fragment boundaries are countable (each fragment is
  sorted; a boundary is a descent), and a lost fragment is visible as
  a lower count.
- **`cmd/go` drops a fragment silently, by design.** `mergeCoverProfile`
  (`cmd/go/internal/test/cover.go`, go1.25) returns without a word when
  the fragment file is missing (*"Test did not create profile, which is
  OK"*) and again when it is zero-length (`n == 0`). No error, no
  message, no non-zero exit — the package still prints `ok` and its own
  `coverage:` line.
- **The Makefile's stated theory is wrong.** Its comment blames
  concurrent binaries colliding counter files "same pid and second".
  Counter files are `covcounters.<metahash>.<pid>.<nanotime>`, and a
  re-exec'd helper child cannot collide with its parent at all:
  `testing`'s `coverTearDown` gives a child with no `-test.gocoverdir`
  **its own temp directory** and deletes it. Confirmed by experiment: a
  child that covers three functions contributes nothing to the parent's
  profile whether it exits cleanly or is killed.
- **Counting contributions is not a usable gate.** Collecting into a
  pod directory instead (`-args -test.gocoverdir`) yields the identical
  90.9% and a countable artifact — but stably **27** counter files for
  28 test packages, because `internal/version`'s own binary emits none.
  An expectation with an unexplained exemption in it is a false-alarm
  generator, and a gate that cries wolf is worse than the bug.

## Design decisions (binding for this task)

- **D1 — verify the NUMBER, not the collection.** The gate re-collects
  **only when the reading is below the threshold**, and decides from
  two cold readings:

  | first | second (cold) | verdict |
  | --- | --- | --- |
  | ≥ gate | not taken | pass, output unchanged |
  | < gate | ≥ gate | pass, naming the lossy collection |
  | < gate | < gate | **fail**, printing both readings |
  | suite failed | not taken | fail on the test failure, no re-collection |

  This is the whole design, and it is chosen over every structural
  alternative for one reason: **it cannot false-alarm.** It engages
  only where the gate would already have failed, so a healthy tree
  never pays for it and never trips over it. It needs no expected
  fragment count, no exemption list, and no theory of why a
  contribution goes missing — none of which I can currently defend.

  What it gives up, stated: a loss that still leaves the total **above**
  the gate goes unnoticed. That is the right thing to give up. The
  gate's job is the threshold, and a number understated below the
  truth but above the bar costs nothing.

- **D2 — the second reading must be COLD, and that is load-bearing.**
  `go clean -testcache` runs between the two collections, because a
  warm re-run replays the cached short profile at the same number
  (`os-4eaf8b13`). A retry that omits it would confirm the bad reading
  and make the gate *more* confident of a false regression than it is
  today. A drill therefore asserts the cache clean happens between the
  readings, and deleting it must fail that drill.

- **D3 — exactly one re-collection, never a loop.** The card's own rule
  is "re-run once, then treat a second failure as real". A loop turns a
  genuine regression into a slow one and hides a systematic failure
  behind eventual success. Bound in the code and pinned by a drill that
  fails if a third collection is attempted.

- **D4 — the pass-after-loss case is LOUD.** When the first reading was
  short, the gate says so on its own line: both readings, the fact that
  a collection which loses a package's contribution reads low, and this
  card's id. The frequency therefore stays visible in CI logs and
  stays measurable, which a silent retry would destroy.

- **D5 — the happy path's output is byte-identical.** `check`'s output
  is diffed by the flavor-test core-gate-independence check, so a green
  tree must still print exactly
  `check-next: gofmt/vet/build/test ok; coverage <N>% (gate 90%)`
  and nothing else. Extra lines appear only on the paths that are
  already not green.

- **D6 — the logic is a package with drills, not a Makefile recipe.**
  `next/internal/covergate` holds the decision and the sequencing with
  the collector and the cache-cleaner **injected**;
  `next/cmd/covergate` is the thin wiring that runs the real commands,
  and the `Makefile` calls it. A retry rule living in six lines of
  `make` is a correctness claim nothing can check, which is the
  failure this repository keeps paying for.

- **D7 — the Makefile's wrong comment goes with the change.** Leaving a
  refuted theory in place next to a fix for the real behavior is how
  the next reader learns the wrong thing. It is replaced by what was
  measured, with a pointer to the decisions entry.

- **D7.5 — `flavors/core-Makefile` moves in lockstep, and it is in
  scope.** `scripts/flavor-test.sh` compares `./Makefile` byte for byte
  against `flavors/core-Makefile` in its offline mode, and
  `scripts/validate.sh` runs that mode as part of `make check`. So a
  `Makefile` edit without a refreshed mirror fails the gate with
  *"flavors/core-Makefile has drifted"*, and this plan's first draft put
  the mirror outside its own file scope — the implementation could not
  have reached acceptance criterion 9 without violating its approved
  plan (review finding on #198).

  The mirror is refreshed exactly as the check's own message says
  (`cp Makefile flavors/core-Makefile`), never hand-edited, so the two
  cannot disagree in a way `cmp` would miss.

- **D8 — scope guard.** No change to `-p 1`, to `-covermode`, to
  `-coverpkg`, or to the 90% threshold. No new coverage collection
  strategy: the pod path was evaluated and **rejected** above, and that
  rejection is recorded so it is not re-litigated from scratch. No
  production code touched.

## Steps

1. `next/internal/covergate` — the decision and the two-reading
   sequence, with `Collect func() (float64, error)` and
   `CleanCache func() error` injected; typed verdicts.
2. `next/internal/covergate/covergate_test.go` — the table above, the
   cache-clean assertion, the no-third-collection assertion, the
   suite-failure path.
3. `next/cmd/covergate` — the wiring: run `go test -p 1 ./...` with the
   existing flags, parse `go tool cover -func`'s total, print the
   gate's line.
4. `Makefile` — `check-next` calls it; the refuted comment is replaced.
   Then `cp Makefile flavors/core-Makefile`, in the same commit: the
   offline flavor check compares them byte for byte inside `make check`.
5. `next/docs/decisions.md`, `memory/LEARNINGS.md`; receipt; evidence;
   review. `os-4eaf8b13` is closed with this card.

## File Scope

- `next/internal/covergate/**` (new), `next/cmd/covergate/**` (new)
- `Makefile` (the `check-next` target: the integration point
  `docs/next-build-plan.md` §0 names)
- `flavors/core-Makefile` (its byte-identical mirror, refreshed in the
  same commit: `make check` compares them, D7.5)
- `next/docs/decisions.md`, `memory/*`
- `receipts/os-cafba959.json`

No production code under `next/internal/**` other than the new package,
and nothing else outside `next/**` beyond the two files above.

## Acceptance Criteria

1. A first reading at or above the gate produces **exactly** today's
   output line and takes **one** collection. A drill counts the
   collector's invocations, so a gate that always re-collects fails.
2. A first reading below the gate and a cold second reading at or above
   it **passes**, and the output names both readings and this card. A
   drill asserts the pass and the text.
3. Two readings below the gate **fail**, with both numbers printed. A
   drill asserts the failure and that both appear.
4. The cache is cleaned **between** the readings, not before the first
   and not after the second. The drill records the call order, and
   **deleting the clean must fail it**.
5. At most two collections ever. A drill fails if a third is attempted.
6. A failing suite fails the gate with the test output and **no**
   re-collection.
7. `make check` on a green tree prints byte-identical output to the
   current target's, verified by diffing captured output before and
   after the change.
8. **Mutation evidence, per fix.** Each must fail a drill: deleting the
   cache clean; removing the re-collection; making the re-collection
   unconditional; looping the re-collection.
9. `flavors/core-Makefile` is byte-identical to `Makefile`, asserted by
   running `sh scripts/flavor-test.sh --offline` directly as well as
   through `make check`, so a stale mirror fails before it reaches CI.
10. `make check` green, coverage measured cold, at least three readings
   above the gate.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/covergate/ -count=1
cd next && go test ./... -count=1
make check
```

## Expected diff shape

One new package with its drills, one thin `cmd` wiring, one Makefile
target rewritten with its `flavors/core-Makefile` mirror refreshed, and
the work-product files. Roughly +450/-20 lines, all under `next/**`
except the two Makefile files this plan names.

## A risk worth naming now

The honest limit of D1: **this does not fix the loss, and does not
claim to.** It makes the loss unable to produce a false verdict, and it
records each occurrence where the rate stays countable. The root cause
is still open — a contribution goes missing before or during
`mergeCoverProfile`, and I could not reproduce one in twelve controlled
cold full-suite runs while investigating, which is itself why the low-frequency
detector has to be the deliverable rather than a fix I cannot evidence.

The specific way this could be wrong: if the loss is **not** rare but
merely rare *on an idle machine*, CI (busier, different filesystem)
could hit the re-collection often enough that the gate's cost doubles
routinely. That would show up as D4's line appearing in most runs,
which is exactly why D4 makes it loud rather than silent. If it does,
the follow-up is the per-package collection this plan rejected — with
the frequency data to justify its cost.
