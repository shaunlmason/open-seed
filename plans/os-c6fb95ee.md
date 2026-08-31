# Plan: next ledger writeHead race — unique temp per writer (os-c6fb95ee)

Fixes a multi-process race in `next/internal/ledger` that is flaking
CI on main. `writeHead` rewrites HEAD atomically via a temp file and
rename, but the temp path is the SHARED constant `HEAD.tmp` — and
writeHead has two callers in two different processes' hands: the
append path, and `Open`'s HEAD-repair path (which fires when the
segments are ahead of HEAD — exactly the mid-append window between
the appender's segment write and its HEAD rename). A poll-only
reader Opening at that moment repairs HEAD with its own
write-and-rename of the SAME temp path, consuming the appender's
temp; the appender's rename then fails with `rename HEAD.tmp HEAD:
no such file or directory`. Readers polling while a supervisor
appends is the architecture's designed posture (the 7.1 wakeless
drill made polling the liveness floor), so the store must tolerate
it. Observed on main: `TestGracefulPreemptionDrill` in open-seed#158's
check job (preempt_cli_test.go:220, with an orphaned `seed.test`
helper), and in hindsight the open-seed#156 verify `make check`
flake. Design authority: charter §II.2 (the chain is the source of
truth; the store is an implementation detail that must not lose or
corrupt appends) and the 7.4 preemption machinery this race trips.

## Design decisions (binding for this task)

- **D1 — per-writer unique temp names.** `writeHead` allocates its
  temp with `os.CreateTemp(s.dir, "HEAD.*.tmp")` (write, close,
  rename), so concurrent writeHeads never share a temp path and
  each rename is atomic over its own file. On any failure after
  creation the temp is removed (best-effort), so failed writes do
  not accumulate orphans. Interleaving stays safe: both racers
  write forward-consistent heads (the repair path already guards
  "disk HEAD is not behind the scan → return"), rename is atomic,
  and a momentarily stale HEAD self-heals on the next Open — the
  exact job the repair path exists for. No lockfile, no fsync
  changes, no layout change: `HEAD` keeps its name and shape, and
  temp names still start with `HEAD.` for recognizability while
  never colliding with `headFile` reads or the `segments/`
  directory scan.
- **D2 — the regression test attacks the store directly.** The
  drill only trips the window on loaded CI runners (25 local runs
  cannot), so the test hammers the mechanism: one goroutine
  appending records in a loop, one goroutine calling `Open` on the
  same directory in a loop (each Open runs the repair path), a few
  hundred iterations; pre-fix this reproduces the ENOENT rename
  within the loop, post-fix every append succeeds and a final
  `VerifyFromGenesis` walks the full chain. Same-process
  goroutines suffice: the collision is on the shared path, not on
  process identity.

## Steps

1. `writeHead`: `os.CreateTemp`-based unique temp, close before
   rename, best-effort removal on the failure paths.
2. The regression test in `next/internal/ledger`
   (`TestConcurrentAppendAndOpenRepair` or similar), verified to
   fail on the pre-fix code and pass on the fix.
3. `make check` green from the repo root with cleaned caches;
   coverage at or above the 90% gate.
4. Docs: decision-log entry; `next/docs/progress.md` gains the fix
   in the Phase 8 ledger as an out-of-item CI-health fix (the
   frontier's next action is unchanged).

## File Scope

- `next/internal/ledger/ledger.go`
- `next/internal/ledger/` (the new regression test)
- `next/docs/decisions.md`, `next/docs/progress.md`
- `receipts/os-c6fb95ee.json`

## Acceptance Criteria

1. `writeHead` uses a per-writer unique temp; no shared temp path
   remains anywhere in the store.
2. The regression test reproduces the ENOENT rename on the pre-fix
   code and passes on the fix, with the chain verifying afterward.
3. `make check` green; no behavior change for single-writer use
   (HEAD name, shape, and repair semantics unchanged).

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./internal/ledger/ -count=1
make check
```

## Expected diff shape

A few lines in `writeHead`, one new test, the two docs files, and
the receipt. Nothing else.
