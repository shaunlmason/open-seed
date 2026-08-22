# fastcards backend

The builtin single-machine store: coordination state in a local SQLite
database instead of the `seed-state` ref. Same verbs, same evaluator, same
card format — the engine keeps the path-keyed layout (`tasks/<id>.md`,
`run-log.jsonl`, `handoff/…`) in SQLite, so card parsing, effects, and lint
code are identical. For a solo dev hammering the loop locally it removes the
R4 write ceiling entirely: every verb is one local transaction, no network.

## Switch to it

```sh
scripts/seed state export > cards.json   # from the current backend
# set [coordination] backend = "fastcards" in .seed/config.toml (reviewed PR)
scripts/seed init                        # creates the schema
scripts/seed state import cards.json --actor <operator>
scripts/seed backend verify fastcards
```

Switching away is the same procedure in reverse — `state export` /
`state import` preserve ids, states, dependency edges, rejection history,
evidence, and the run log (import refuses a non-empty target).

## What you get

- **Native atomic claims**: each verb runs in one `BEGIN IMMEDIATE`
  transaction. A contention loser blocks on the winner's live transaction
  (`busy_timeout`, bounded jittered retry — SQLITE_BUSY is never surfaced
  raw), re-reads the committed claim, and exits 2.
- **Worktree-correct**: the DB lives under the repository's *common* git
  dir (`git rev-parse --git-common-dir`), so the loop's linked worktrees —
  whose `.git` is a file — all share one claims database.
- **Native offline**: there is no remote to contact at all.

## Declared variances (never silent)

- **State is machine-local** (`state_portability = "machine"`): it does not
  travel with clones, forks, or CI. The state-ref integrity story — anchor
  tags, history-replay lint, push-access trust — is replaced by
  local-filesystem trust: `seed state anchor` refuses
  (`anchors_not_applicable`), and `state lint` runs the card lints but has
  no commit history to replay.
- **The close lane is local** (design §7.3): CI's merged-PR auto-close and
  the no-PR dispatch workflow cannot see a local database, so on this rung
  the solo human operator closes review cards through the port on their own
  machine (`seed task close <id> --no-pr --resolution … --actor <operator>`),
  with the evidence recorded on the card. fastcards is for local solo
  loops, not for repos where CI closes cards.
- **Leases and fencing** behave identically to filecards (same evaluator);
  reaping is `seed maintain reap` run locally.

## Testing

The engine's task-layer suite runs the full lifecycle against this store
(claim contention under a held transaction, fencing, plan parking,
reject-lockout, cascade, worktree sharing, export/import round-trip,
throughput smoke) — see `internal/task/fastcards_test.go` in the engine
repo. Template-side, `scripts/seed backend verify fastcards` checks the
manifest + lock.
