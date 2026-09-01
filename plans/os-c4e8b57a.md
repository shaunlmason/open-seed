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
| bare remote, mirror clone | `cmd/seed-admit/drill_test.go` | bare, pushed to; `clone --mirror` |
| three fixtures | `cmd/seed-admit/main_test.go` | two bare, one non-bare |

So the fix applies to the class, not to the one fixture that happened
to lose the race first.

## Design decisions (binding for this task)

- **D1 — WRITE the settings into the repository, before its first
  object.** Every git repository a fixture creates gets `gc.auto=0`,
  `gc.autoDetach=false` and `receive.autoGC=false` **written into its
  own config**, immediately after `init` (or `clone`) and before any
  object-producing operation.

  The first draft of this plan said to pass them as `-c` flags on the
  creating command. **That would have shipped a no-op** — review
  finding on this PR, verified here:

  ```console
  $ git -c gc.auto=0 -c receive.autoGC=false init -q --bare r.git
  $ git -C r.git config --get gc.auto        # exit 1: unset
  $ git -C r.git config --get receive.autoGC # exit 1: unset
  ```

  `git -c` scopes a value to **that one invocation**; it writes
  nothing. So the later commits, and above all the bare remote's own
  `receive-pack` — the process that actually lost the race — would
  still have run under stock auto-gc, and the flake would have
  survived a change that read like a fix. A no-op that looks like a
  fix is worse than no fix at all, because it retires the card.

  `init --bare` produces no objects, so a `git config` write on the
  next line is still before the first object: there is no window, and
  a `t.Cleanup` that waits for stragglers would be a race against a
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
- **D5 — the sweep is a property over the tree, not a list of names.**
  The table above is what the tree holds today, and it is already
  wider than the first draft recorded: `cmd/seed-admit` carries three
  more fixtures plus a `clone --mirror` (review finding on this PR —
  the stated file scope contradicted the acceptance criterion, which
  said *every* `next/**` test repository). `clonedRepo` has since
  landed with #173 and is in scope too. So the task implements
  against whatever repository-creating fixtures the tree holds when it
  lands, and the guard in step 3 checks the property rather than a
  fixed set of names — which is exactly what keeps the scope and the
  criterion from drifting apart again.

## Steps

1. Add the shared hardening helper to the `cmd/seed`,
   `cmd/seed-admit` and `internal/gitref` test packages per D3: it
   takes a repository path and writes the three settings into that
   repository's config.
2. Call it immediately after every repository creation in `next/**`
   tests — every `git init` (bare and not), every `git clone`
   (including `--mirror`) — per the table above and anything else
   present at implementation time.
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
- `next/cmd/seed-admit/main_test.go`, `next/cmd/seed-admit/drill_test.go`
- `next/internal/gitref/gitref_test.go`
- one new guard test
- `memory/*` if warranted
- `receipts/os-c4e8b57a.json`

## Acceptance Criteria

1. Every git repository created by a `next/**` test — bare or not,
   pushed to or only committed in, `cmd/seed-admit` included — has
   `gc.auto=0`, `gc.autoDetach=false` and `receive.autoGC=false`
   **written into its own config**, before its first object. A `git
   -c` flag on the creating command does NOT satisfy this and the
   guard must reject it: `git -C <repo> config --get gc.auto` returns
   the value in a fixture repository.
2. A guard test fails if a new fixture creates a repository without
   them, and it covers `cmd/seed-admit` as well as `cmd/seed` and
   `internal/gitref`.
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

A shared hardening helper in three test packages, called at roughly
nine creation sites, plus one guard test. Roughly +90/-10 lines, all
in `next/**` test files.
