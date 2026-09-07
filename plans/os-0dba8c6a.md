# Plan: os-2e34f66a's dangling obligations.md citation (os-0dba8c6a)

**Retrospective plan.** This card's whole deliverable was a one-line edit to
*another* card's plan file, so it could carry no task PR of its own: a task PR
(`seed/<id>`) never touches `plans/**`, and a plan PR (`seed/<id>-plan`) must
touch exactly `plans/<its-own-id>.md`. The work therefore landed as a
plan-class PR against `plans/os-2e34f66a.md`, and no `plans/os-0dba8c6a.md`
was ever authored. It was closed with a plain `accept` and does not carry the
D7 no-PR exemption, so this file is the D3 plan-resolvability artifact the
done-consistency lint requires. It documents what was authorized and how it
was verified. It does **not** retroactively confer plan approval: the
authorization of record is the human `accept` already on the card, and this
file exists so the lint can resolve `plans/<task-id>.md`. It gates no future
work.

`plans/os-2e34f66a.md` line 130 wrote ``([`obligations.md`](obligations.md))``
inside a plan file, so the relative target resolved against `plans/`, where no
such file is. The document it meant is `next/spec/obligations.md`. The citation
checker os-5fe43832 landed (`seed docs check`) is what surfaced it.

## Steps

1. **Repoint the citation** in `plans/os-2e34f66a.md` to the path that
   resolves from the plan file's own directory.
2. **Land it as a plan-class PR** (#319), the only class whose purity rule
   admits a `plans/**` diff.

## File Scope

- `plans/os-2e34f66a.md` (one line; the cited target only).

No other file: the card is a citation repair, not a behavior change.

## Acceptance Criteria

- The citation in `plans/os-2e34f66a.md` resolves to a path the tree holds.
- `seed docs check` reports no drift and no unresolvable relative citation.
- No behavior, spec or conformance row moves: the diff is one link target.

## Validation Commands

- `git show e848631 --stat`: exactly one file changed,
  `plans/os-2e34f66a.md`, one insertion and one deletion.
- `grep -n 'obligations.md' plans/os-2e34f66a.md`: the target resolves from
  `plans/`, so the link is not the bare `obligations.md` that failed.
- `cd next && go run ./cmd/seed docs check --root ..`: exits 0 with an empty
  `drift` array, the gate that would have caught the original.
- Evidence record: card `os-0dba8c6a` review block, accepted by `shaunlmason`
  2026-09-07T06:26:03Z, evidence PR #319 merged as `e848631`.
