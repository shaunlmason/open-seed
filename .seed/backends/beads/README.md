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

- **Fence is emulated**: the claim token is the assignee identity
  (`assignee:<actor>`), verified by assignee-match — not a rotating token.
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

`sh test.sh` runs the offline contract test against `testdata/fake-bd` (a
deterministic double of the documented CLI surface). It exercises every
required verb's envelope and exit-code mapping and runs in `make check` via
validate.sh. Validation against a live beads install is tracked as follow-up
work on the seed queue.
