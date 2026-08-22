# beads backend

Wraps the [`bd` CLI](https://github.com/steveyegge/beads) (Dolt-backed,
git-native issue graph) behind the seed port. Native atomic claim
(`bd update --claim`) and native close-cascade replace the file backend's
push-wins emulation — this is the documented upgrade when state-ref write
contention bites (handbook §6).

## Install & activate

1. Install `bd` (pin a version; `requires` says `>=0.57.0`) and `jq`.
2. `bd init` in the repo (embedded mode; `bd init --server` for many
   concurrent writers).
3. Set `.seed/config.toml`: `[coordination] backend = "beads"`.
4. Migrate open cards once: for each card from `seed task list` (filecards),
   re-create via the port — the state ref stays as the historical record.
5. `scripts/seed backend verify beads` confirms manifest + lock.

## State mapping

| port | bd |
|---|---|
| backlog | `deferred` |
| ready | `open` |
| in_progress | `in_progress` (assignee set) |
| review | `open` + label `seed:review` |
| blocked | `blocked` |
| done | `closed` |
| cancelled | `closed` + label `seed:cancelled` |

## Declared variances (never silent)

- **Fence is emulated**: claim mints a rotating nonce token
  (`tok:<nonce>`, persisted as a `seed:tok:<nonce>` label) verified
  against the current assignee — a pre-reclaim token dies on reclaim
  (rotation), and stale/missing/foreign tokens exit 6.
  bd's optimistic `revision` can harden this later.
- **Leases are replica-scoped** (bd semantics): a lease is enforceable only
  on the replica that granted it. `lease-renew` maps to bd's idempotent
  re-claim.
- **Event-append rides bd's audit** (Dolt history + comments). The seed
  run-log is authoritative only for the filecards backend; the maintenance
  state lint's transition replay does not apply here — external bd users can
  move issues in ways the D1 table would refuse.
- **Operator verbs are not roster-enforced by the plugin**; bd's own access
  model (push access to the Dolt remote) is the boundary.

## Testing

## Testing

Two suites share one corpus (`corpus.sh`), so they cannot drift apart:

- `sh test.sh` — offline, against `testdata/fake-bd` (a deterministic
  double of the CLI surface). Runs in `make check` via validate.sh.
- `sh live-test.sh` — the same corpus against a REAL bd install in a
  scratch repo. Self-skips (exit 0, explicit message) when `bd` is not
  on PATH, so CI needs no new binaries; validate.sh runs it after the
  offline suite.

### Validated bd pin

The adapter is validated against **bd v1.2.2**:

```sh
sudo apt install -y libicu-dev pkg-config   # CGO dependency
go install github.com/steveyegge/beads/cmd/bd@v1.2.2
```

(The module requires Go >= 1.26.2; the toolchain auto-switches.)
live-test echoes the pin at start and warns — without refusing — when
the PATH bd differs: drift discovery is the point.

### Declared v1.2.2 variances

Live-validated behaviors the adapter and fake both encode:

- `bd show <id> --json` returns an **array** of issues; a missing id is
  an error object on stdout with exit 1. The adapter normalizes to the
  single issue object.
- `bd list --json` hides closed issues; the adapter's `list` verb uses
  `--all` so the port surfaces terminal cards too.
- Comments are listed via `bd comments <id> --json` (`show` carries only
  `comment_count`).
- `bd update --claim` is the native atomic claim (assignee from
  `BD_ACTOR`, idempotent for the holder) — the adapter mints its
  rotating fence token on top of it.
- `bd note` writes a single id-less `notes` string, so evidence rides
  `bd comment` (stable per-comment ids); `comment_id`/`evidence_id` are
  the created comment's id.
- `bd label add/remove` are per-label atomic (never whole-array
  replacement); `--set-labels` exists but the adapter never uses it.
