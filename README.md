# open-seed (v1) — retired

**This repository is retired. The successor is
[shaunlmason/open-seed-v2](https://github.com/shaunlmason/open-seed-v2).**

open-seed was a template repository for multi-agent orchestration: a task port
with pluggable backends, coordination state on a dedicated git ref, a
plan → implement → receipt evidence chain, enforced guardrails, and a loop
runner. Its contract was *files, not an app*.

It also hosted the implementation of its successor, **Seed**, chartered in
`SEED-NEXT.md` and built under `next/`. Seed was extracted into its own
repository on 2026-09-08 and stands there independently; this repository was
retired in the same move.

## Why it was retired

Nothing used v1 but this repository itself. Once Seed stood on its own, keeping
v1 here meant maintaining a template nobody instantiated and a second copy of
the successor's code that could only diverge. The reasoning, and what the
retirement gives up, is recorded rather than summarized:

- [`docs/v1-retirement.md`](docs/v1-retirement.md) — the plan: stages, the
  deletion inventory, and what is kept permanently.
- [`decisions/0006-v1-retirement.md`](decisions/0006-v1-retirement.md) — the
  decision to retire.
- [`decisions/0009-v2-extraction-supersedes-the-cutover.md`](decisions/0009-v2-extraction-supersedes-the-cutover.md)
  — why the extraction replaced the planned in-place cutover.

## What is kept, permanently

- **The coordination history.** The `seed-state` ref, frozen at its final
  anchor, and every `seed-anchor/*` tag. These are ordinary git refs and are
  readable with stock git; nothing about reading them needs the retired shim.
- **The records.** `decisions/`, `plans/`, `receipts/` and `memory/`.
- **The research corpus.** [`docs/research/`](docs/research/), which the
  successor's charter cites as its lineage.

## What became of the parts

| part | where it went |
|---|---|
| Seed (was `next/`) | [open-seed-v2](https://github.com/shaunlmason/open-seed-v2) |
| the `seed` engine binary | `shaunlmason/open-seed-engine`, archived read-only; its releases are the provenance the frozen history was produced under |
| the template, flavors, plugin, guardrails, loop runner | retired with this repository |

Migrating v1 state into Seed is still supported: `seed import --from-open-seed`
in the successor transforms an export into a genesis chain, drilled against a
real export of this repository.
