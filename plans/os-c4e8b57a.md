# Plan: next — no detached git process outlives a test fixture (os-c4e8b57a)

`TestRemoteAppendExhaustsAtContention` flaked in CI on 2026-08-31
(flavor-test job, run 33376438803): the **test body passed**, and then
`t.TempDir`'s `RemoveAll` cleanup raced git's detached auto-gc in the
bare remote fixture — `unlinkat .../remote.git/objects: directory not
empty`. Green on the immediate re-run. A latent race, not a
regression.

## Why a cleanup flake is worth a card

A flake that fails *after* the assertions pass is the worst kind for
an unattended loop: the signal says "your change is broken" when the
change is fine, and the correct response — re-run once, then treat a
second failure as real — is a rule an agent has to apply against its
own instinct to go looking for a bug it did not write. Removing the
race is cheaper than paying that cost on every future red.

## The class is wider than the card names

The card says "sweep `remote_test.go` fixtures for the same pattern".
The sweep found the race is not about *bare* repositories: it is
between `t.TempDir`'s recursive removal and **any** detached git
process still writing under the directory. Every git repository a test
creates inside a `t.TempDir` is exposed, whether bare or not, whether
it is pushed to or only committed in:

| Fixture | File | Shape |
| --- | --- | --- |
| `bareRemote` | `cmd/seed/remote_test.go` | bare, pushed to |
| bare fixture | `internal/gitref/gitref_test.go` | bare, pushed to |
| `verdictRepo` | `cmd/seed/verdict_cli_test.go` | non-bare, committed in |
| `clonedRepo` | `cmd/seed/loop_cli_test.go` | bare origin **and** a clone that pushes |

So the fix applies to the class, not to the one fixture that happened
to lose the race first.

## Design decisions (binding for this task)

- **D1 — configure at creation, never clean up after.** Every git
  repository a fixture creates is created with `gc.auto=0`,
  `gc.autoDetach=false` and `receive.autoGC=false`, passed as `-c`
  flags on the creating command (and on the clone) so the setting is
  in effect from the first object write, before any hook could spawn a
  collector. A post-hoc `git config` write would leave a window; a
  `t.Cleanup` that waits for stragglers would be a race against a
  race.
- **D2 — all three settings, because they are three different
  spawners.** `gc.auto=0` disables the heuristic entirely;
  `gc.autoDetach=false` makes any gc that does run stay in the
  foreground, where the test waits for it, instead of surviving the
  command that started it; `receive.autoGC=false` covers the push
  path, which is the one that actually bit — the bare remote's
  `receive-pack`, not the client. Setting only the first would leave
  the other two paths armed on a git that changes its defaults.
- **D3 — one helper, used by every fixture.** A single
  `gitFixtureArgs()` (or equivalent shared list) so a future fixture
  inherits the hardening by construction rather than by remembering.
  Two of the four sites are in `cmd/seed`, one is in
  `internal/gitref`; the two packages get the same list, duplicated
  once across the package boundary rather than exported from a test
  package.
- **D4 — no production change, and no behavior change to the drills.**
  These are fixture-construction flags. Every assertion, timing and
  scenario stays exactly as it is; the drills must keep passing
  unchanged, which is the whole evidence that the fix is inert.
- **D5 — `clonedRepo` is covered if it exists at implementation
  time.** It arrives with `os-7e197768` (#173, unmerged at this
  writing). The task implements against whatever bare-or-not git
  fixtures the tree holds when it lands, so the sweep is stated as a
  property ("every git repository a test creates") rather than as a
  fixed list, and the drill in step 3 checks the property rather than
  four names.

## Steps

1. Add the shared fixture-config list to `cmd/seed` and
   `internal/gitref` test helpers per D3.
2. Apply it at every git repository creation in `next/**` tests: the
   `git init` calls (bare and not) and the `git clone`, per the table
   above and anything else present at implementation time.
3. Add a guard that keeps the property true: a test that walks the
   `next/**` test sources for `git init` / `git clone` invocations and
   fails if one is constructed without the hardening flags. A comment
   alone would not survive the next fixture.
4. Run the affected packages repeatedly (`-count=5`) to confirm no
   behavior changed and the drills stay green.
5. `memory/LEARNINGS.md` if the sweep teaches anything durable;
   receipt; evidence; review.

## File Scope

- `next/cmd/seed/remote_test.go`, `next/cmd/seed/verdict_cli_test.go`,
  `next/cmd/seed/loop_cli_test.go` (if present), and any further
  `next/cmd/seed` fixture that creates a repository
- `next/internal/gitref/gitref_test.go`
- one new guard test
- `memory/*` if warranted
- `receipts/os-c4e8b57a.json`

## Acceptance Criteria

1. Every git repository created by a `next/**` test — bare or not,
   pushed to or only committed in — is created with `gc.auto=0`,
   `gc.autoDetach=false` and `receive.autoGC=false`, applied on the
   creating command rather than written afterwards.
2. A guard test fails if a new fixture creates a repository without
   them.
3. No production file changes; no assertion, timing or scenario
   changes in any existing drill.
4. `go test ./cmd/seed/ ./internal/gitref/ -count=5` green.
5. `make check` green, coverage gate ≥90% held.

## Validation Commands

```sh
cd next && gofmt -l . && go vet ./... && go build ./...
cd next && go test ./cmd/seed/ ./internal/gitref/ -count=5
make check
```

## Expected diff shape

A shared flag list in two test packages, applied at four or five
creation sites, plus one guard test. Roughly +60/-10 lines, all in
`next/**` test files.
